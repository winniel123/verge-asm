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

type vantageClassStore interface {
	queue.AddressExclusionStore

	ListAddressScopeCidrs(ctx context.Context) ([]*netip.Prefix, error)
}

// One binding serves batch gating and every render, so a second predicate is refused (#711).

func (s *server) addressScopeCovered(ctx context.Context) (func(netip.Addr) bool, error) {
	scopes, err := s.vantageClassStore.ListAddressScopeCidrs(ctx)
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
	excluded, err := queue.ReadAddressExclusions(ctx, s.vantageClassStore)
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

func listedVantageClasses(rows []db.ListVantagesRow, covered func(netip.Addr) bool) []custody.VantageClass {
	out := make([]custody.VantageClass, 0, len(rows))
	for _, v := range rows {
		out = append(out, vantageFactsClass(v.DialledAddr, v.Egress, covered))
	}
	return out
}

func runningVantageClasses(vantages []db.ListVantagesForDispatchRow, covered func(netip.Addr) bool) []string {
	// An unavailable vantage still names its class; the gap reads not-evaluable (ADR-0080).
	seen := map[string]struct{}{}
	for _, v := range vantages {
		seen[string(vantageFactsClass(v.DialledAddr, v.Egress, covered))] = struct{}{}
	}
	return sortedKeys(seen)
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

func reachRowsForServices(rows []db.ListServiceReachabilitySpansByClassForServicesRow) []reachLegRow {
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
	grouped := map[string]map[string][]reachLegRow{}
	for _, row := range rows {
		class := string(vantageclass.Derive(row.dialled, row.egress, covered))
		m := grouped[row.subject]
		if m == nil {
			m = map[string][]reachLegRow{}
			grouped[row.subject] = m
		}
		m[class] = append(m[class], row)
	}
	out := make(map[string]map[string]legInfo, len(grouped))
	for subj, byClass := range grouped {
		cm := make(map[string]legInfo, len(byClass))
		for class, group := range byClass {
			cm[class] = legFromClassGroup(group)
		}
		out[subj] = cm
	}
	return out
}

func legFromClassGroup(group []reachLegRow) legInfo {
	cur := group[0]
	for _, row := range group[1:] {
		if moreRecent(row.openedAt, row.id, cur) {
			cur = row
		}
	}
	rv := decodeReachability(cur.value)
	info := legInfo{outcome: rv.Outcome, reason: rv.Reason, cause: rv.Cause, since: cur.openedAt, isGap: cur.isGap, present: true}
	held := legFrom(info)
	for _, row := range group {
		other := decodeReachability(row.value)
		// The leg holds one value, and a span of another value never dates it (#2017).
		if legFrom(legInfo{outcome: other.Outcome, isGap: row.isGap, present: true}) != held {
			continue
		}
		if row.openedAt.Before(info.since) {
			info.since = row.openedAt
		}
	}
	return info
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
