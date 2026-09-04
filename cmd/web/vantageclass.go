package main

import (
	"context"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/queue"
	"github.com/winniel123/verge-asm/internal/vantageclass"
)

// One binding serves batch gating and every render, so a second predicate is refused (#711).

func (s *server) addressScopeCovered(ctx context.Context) (func(netip.Addr) bool, error) {
	scopes, err := s.store.ListAddressScopeCidrs(ctx)
	if err != nil {
		return nil, err
	}
	var prefixes []netip.Prefix
	for _, p := range scopes {
		if p != nil {
			prefixes = append(prefixes, *p)
		}
	}
	// An excluded range is not the operator's, so a prober inside it may reclassify (ADR-0133 §4).
	excluded, err := queue.ReadAddressExclusions(ctx, s.store)
	if err != nil {
		return nil, err
	}
	estate := custody.Estate{AddressScopes: prefixes}.WithAddressExclusions(excluded)
	return estate.CoversAddressScope, nil
}

func vantageFactsClass(dialled, egress pgtype.Text, covered func(netip.Addr) bool) custody.VantageClass {
	return vantageclass.Derive(dialled.String, egress.String, covered)
}

func presentedAddrs(v db.Vantage) []netip.Addr {
	return vantageclass.PresentedAddrs(v.DialledAddr.String, v.Egress.String)
}

func deriveVantageClasses(vantages []db.Vantage, covered func(netip.Addr) bool) map[int64]custody.VantageClass {
	out := make(map[int64]custody.VantageClass, len(vantages))
	for _, v := range vantages {
		out[v.ID] = vantageFactsClass(v.DialledAddr, v.Egress, covered)
	}
	return out
}

type reachLegRow struct {
	subject  string
	dialled  string
	egress   string
	value    []byte
	isGap    bool
	openedAt time.Time
	id       int64
}

func reachRowsFromCurrent(rows []db.ListServiceReachabilitySpansByClassRow) []reachLegRow {
	out := make([]reachLegRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, reachLegRow{
			subject: r.SubjectKey, dialled: r.DialledAddr.String, egress: r.Egress.String,
			value: r.Value, isGap: r.IsGap, openedAt: r.OpenedAt.Time, id: r.ID,
		})
	}
	return out
}

func reachRowsFromAt(rows []db.ListServiceReachabilitySpansByClassAtRow) []reachLegRow {
	out := make([]reachLegRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, reachLegRow{
			subject: r.SubjectKey, dialled: r.DialledAddr.String, egress: r.Egress.String,
			value: r.Value, isGap: r.IsGap, openedAt: r.OpenedAt.Time, id: r.ID,
		})
	}
	return out
}

func moreRecent(openedAt time.Time, id int64, cur reachLegRow) bool {
	if openedAt.After(cur.openedAt) {
		return true
	}
	return openedAt.Equal(cur.openedAt) && id > cur.id
}

func collapseReachLegs(rows []reachLegRow, covered func(netip.Addr) bool) map[string]map[string]legInfo {
	best := map[string]map[string]reachLegRow{}
	for _, row := range rows {
		class := string(vantageclass.Derive(row.dialled, row.egress, covered))
		m := best[row.subject]
		if m == nil {
			m = map[string]reachLegRow{}
			best[row.subject] = m
		}
		if cur, ok := m[class]; !ok || moreRecent(row.openedAt, row.id, cur) {
			m[class] = row
		}
	}
	out := make(map[string]map[string]legInfo, len(best))
	for subj, byClass := range best {
		cm := make(map[string]legInfo, len(byClass))
		for class, row := range byClass {
			cm[class] = legInfo{outcome: decodeReachability(row.value).Outcome, isGap: row.isGap, present: true}
		}
		out[subj] = cm
	}
	return out
}

func collapseNameResolutions(rows []db.ListNameResolutionsByClassRow, covered func(netip.Addr) bool) map[string]map[string]resolutionValue {
	type chosen struct {
		value      []byte
		observedAt time.Time
		id         int64
	}
	best := map[string]map[string]chosen{}
	for _, r := range rows {
		class := string(vantageclass.Derive(r.DialledAddr.String, r.Egress.String, covered))
		m := best[r.SubjectKey]
		if m == nil {
			m = map[string]chosen{}
			best[r.SubjectKey] = m
		}
		cur, ok := m[class]
		if !ok || r.ObservedAt.Time.After(cur.observedAt) ||
			(r.ObservedAt.Time.Equal(cur.observedAt) && r.ID > cur.id) {
			m[class] = chosen{value: r.Value, observedAt: r.ObservedAt.Time, id: r.ID}
		}
	}
	out := make(map[string]map[string]resolutionValue, len(best))
	for name, byClass := range best {
		cm := make(map[string]resolutionValue, len(byClass))
		for class, c := range byClass {
			cm[class] = decodeResolution(c.value)
		}
		out[name] = cm
	}
	return out
}
