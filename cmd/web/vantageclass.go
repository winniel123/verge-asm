package main

import (
	"context"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/vantageclass"
)

// Vantage class is DERIVED per read from each vantage's persisted presented-address
// facts, never from the vestigial `class` column (#709 keystone (b)). The web layer
// has no Estate assembler of its own — only internal/queue/hot.go:hotEstate builds one
// at batch time — so the render/compose path assembles the same address-scope corpus
// here, from the same ListAddressScopeCidrs query, and offers the identical family-
// matched containment predicate to the class derivation (#711). One binding, used
// identically by batch gating and every render.

// addressScopeCovered reads the declared address-scope Seed CIDRs and returns the
// `covered` predicate the Vantage-class derivation binds — custody.Estate's family-
// matched CoversAddressScope over ADDRESS SCOPES ALONE (never the extension or
// MayProbe, #711). It mirrors internal/queue/hot.go:hotEstate's scope load exactly
// (drop nil rows into custody.Estate{AddressScopes}). A read failure returns the error;
// callers degrade the affected screen rather than 500ing.
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
	estate := custody.Estate{AddressScopes: prefixes}
	return estate.CoversAddressScope, nil
}

// vantageFactsClass derives one vantage's class per read from its two persisted
// presented-address facts (#709/#710), for the render/compose sites that hold a row
// carrying dialled_addr + egress. Every call site routes through this so class is
// computed one way everywhere.
func vantageFactsClass(dialled, egress pgtype.Text, covered func(netip.Addr) bool) custody.VantageClass {
	return vantageclass.Derive(dialled.String, egress.String, covered)
}

// presentedAddrs is this ticket's contract (#710): the 0/1/2 addresses an outside
// observer saw of a vantage — the dialled peer and the SSH_CLIENT egress — parsed from
// its persisted facts, unmapped, never fabricated.
func presentedAddrs(v db.Vantage) []netip.Addr {
	return vantageclass.PresentedAddrs(v.DialledAddr.String, v.Egress.String)
}

// deriveVantageClasses derives each vantage's class per read (#709), keyed by vantage
// id, so a pass can memoize it once and reuse it across every class-keyed read. A
// vantage with no presented facts derives `unverified`, exactly as the vestigial column
// reads today (the seeded `local` fixture keeps rendering `unverified`; no golden moves).
func deriveVantageClasses(vantages []db.Vantage, covered func(netip.Addr) bool) map[int64]custody.VantageClass {
	out := make(map[int64]custody.VantageClass, len(vantages))
	for _, v := range vantages {
		out[v.ID] = vantageFactsClass(v.DialledAddr, v.Egress, covered)
	}
	return out
}

// reachLegRow is one per-vantage reachability leg normalized across the current and
// as-of by-class reads, which return distinct row types with identical fields. It
// carries the vantage's presented-address facts (dialled + egress) so the fold derives
// each leg's class, and (opened_at, id) so it can re-collapse to the most-recent leg
// per (subject, derived class) — the collapse the retired SQL DISTINCT ON did.
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

// moreRecent reports whether a per-vantage leg row should replace the one currently
// chosen for its (subject, derived class) bucket — the opened_at DESC, id DESC tiebreak
// the retired SQL DISTINCT ON encoded (a resolution read passes observed_at as openedAt).
func moreRecent(openedAt time.Time, id int64, cur reachLegRow) bool {
	if openedAt.After(cur.openedAt) {
		return true
	}
	return openedAt.Equal(cur.openedAt) && id > cur.id
}

// collapseReachLegs derives each per-vantage leg's class and collapses to the most
// recent leg per (subject, DERIVED class) — reproducing the retired SQL
// DISTINCT ON (subject_key, v.class) ORDER BY opened_at DESC, id DESC, but with the
// class derived in Go (#709). Returns subject -> class -> legInfo, the shape foldExposure
// and the stat folds already consume.
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

// collapseNameResolutions derives each per-vantage resolution's class and collapses to
// the most recent value per (name, DERIVED class) — the resolution twin of
// collapseReachLegs (observed_at DESC, id DESC), replacing the retired SQL
// DISTINCT ON (subject_key, v.class) (#709). Returns name -> class -> resolutionValue,
// the shape buildNameFacts already consumes.
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
