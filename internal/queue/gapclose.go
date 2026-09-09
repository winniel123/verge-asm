package queue

import (
	"context"
	"encoding/json"
	"net/netip"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/retention"
)

// A Gap opening is Coverage inventory and never a message, so only the closing edge fires (#882).

func gapCloseMessages(ctx context.Context, store messageStore, observedAt time.Time, changes []spanChange, in membershipInputs) ([]*message.Message, error) {
	var closed []spanChange
	for _, c := range changes {
		if !c.Opened && c.PrevIsGap && !c.IsGap {
			closed = append(closed, c)
		}
	}
	if len(closed) == 0 {
		return nil, nil
	}
	scopes, err := coveringScopes(ctx, store, observedAt, closed, in)
	if err != nil {
		return nil, err
	}
	rules, err := rulesOpenedByGapClose(ctx, store, observedAt, changes, closed)
	if err != nil {
		return nil, err
	}
	type firing struct {
		kind     string
		closures []message.GapClosure
		rules    []message.CensusEntry
	}
	byScope := map[string]*firing{}
	for _, c := range closed {
		scope, kind := scopes[subjectID(c)], "seed"
		if scope == "" {
			// An uncovered subject fires at itself, so the closing is never lost (ADR-0014).
			scope, kind = c.SubjectKey, c.SubjectKind
		}
		f := byScope[scope]
		if f == nil {
			f = &firing{kind: kind}
			byScope[scope] = f
		}
		f.closures = append(f.closures, message.GapClosure{
			Kind:   c.SubjectKind,
			Key:    c.SubjectKey,
			Facet:  c.Facet,
			Before: gapValueLabel(c.Facet, c.BeforeGap),
			After:  gapValueLabel(c.Facet, c.Value),
			Broke:  c.BrokeAcrossGap,
		})
		f.rules = append(f.rules, rules[c.SubjectKey]...)
	}
	keys := make([]string, 0, len(byScope))
	for scope := range byScope {
		keys = append(keys, scope)
	}
	sort.Strings(keys)
	var msgs []*message.Message
	for _, scope := range keys {
		f := byScope[scope]
		if m := message.GapClosed(scope, f.kind, f.closures, f.rules, observedAt); m != nil {
			msgs = append(msgs, m)
		}
	}
	return msgs, nil
}

func rulesOpenedByGapClose(ctx context.Context, store messageStore, observedAt time.Time, changes, closed []spanChange) (map[string][]message.CensusEntry, error) {
	var services []string
	seen := map[string]bool{}
	for _, c := range closed {
		if c.SubjectKind == subjectKindService && c.Facet == connectoutcome.FacetReachability && !seen[c.SubjectKey] {
			seen[c.SubjectKey] = true
			services = append(services, c.SubjectKey)
		}
	}
	if len(services) == 0 {
		return nil, nil
	}
	// A rule reads the composed internet leg, never one vantage's own reading (ADR-0080).
	covered, err := coveredAddressScope(ctx, store)
	if err != nil {
		return nil, err
	}
	current, err := store.ListServiceReachabilitySpansByClassForServices(ctx, services)
	if err != nil {
		return nil, err
	}
	legs := legsFromCurrent(current, covered)
	out := map[string][]message.CensusEntry{}
	for _, svc := range services {
		after, ok := composeInternetLeg(legs, svc)
		// The Gap held no leg, so every rule that reads reach opened here (ADR-0026 §5).
		reach := reachAtCause{after: reachLeg{has: ok, outcome: string(after)}}
		census, err := censusWithRules(ctx, store, observedAt, changes, svc, message.Census{}, reach)
		if err != nil {
			return nil, err
		}
		out[svc] = census.Rules()
	}
	return out, nil
}

func subjectID(c spanChange) [2]string { return [2]string{c.SubjectKind, c.SubjectKey} }

func coveringScopes(ctx context.Context, store messageStore, observedAt time.Time, closed []spanChange, in membershipInputs) (map[[2]string]string, error) {
	out := map[[2]string]string{}
	uncovered := map[[2]string]netip.Addr{}
	for _, c := range closed {
		id := subjectID(c)
		if _, seen := out[id]; seen {
			continue
		}
		out[id] = coveringSeedKey(c.SubjectKind, c.SubjectKey, in)
		if out[id] != "" {
			continue
		}
		if addr, ok := subjectAddress(c.SubjectKind, c.SubjectKey); ok {
			out[id] = coveringSeedKey(subjectKindAddress, addr.String(), in)
			if out[id] == "" {
				uncovered[id] = addr
			}
		}
	}
	if len(uncovered) == 0 {
		return out, nil
	}
	// An Address under a custody extension is covered through its citing Name (ADR-0013).
	rows, err := store.ListNameCitationSpansWithinCurrency(ctx, db.ListNameCitationSpansWithinCurrencyParams{
		At:            tstz(observedAt),
		FloorCadences: retention.FloorCadences,
	})
	if err != nil {
		return nil, err
	}
	citers := map[string][]string{}
	for _, r := range parseCitationSpans(rows).resolutions {
		if r.row.ClosedAt.Valid {
			continue
		}
		for _, a := range r.addrs {
			citers[a] = append(citers[a], r.row.SubjectKey)
		}
	}
	for id, addr := range uncovered {
		names := citers[addr.String()]
		sort.Strings(names)
		for _, name := range names {
			if scope := coveringSeedKey(subjectKindName, name, in); scope != "" {
				out[id] = scope
				break
			}
		}
	}
	return out, nil
}

func lastValueBeforeGap(rows []db.ListSpansForSubjectRow, key drift.TimelineKey, vantageID pgtype.Int8, vector drift.Vector) (value []byte, broke bool) {
	var latest *db.ListSpansForSubjectRow
	for i := range rows {
		r := &rows[i]
		if r.IsGap || !r.ClosedAt.Valid || r.Facet != key.Facet || r.Discriminator != key.Discriminator ||
			r.Source != key.Source || r.VantageID != vantageID {
			continue
		}
		if latest == nil || r.ClosedAt.Time.After(latest.ClosedAt.Time) ||
			(r.ClosedAt.Time.Equal(latest.ClosedAt.Time) && r.ID > latest.ID) {
			latest = r
		}
	}
	if latest == nil {
		return nil, false
	}
	// One derivation licenses a pair, so a Break beneath the Gap states none (ADR-0014).
	if !vectorFromJSON(latest.Derivation).Equal(vector) {
		return nil, true
	}
	return canonicalJSON(latest.Value), false
}

func gapValueLabel(facet string, value []byte) string {
	if len(value) == 0 {
		return ""
	}
	var v struct {
		Outcome   string   `json:"outcome"`
		Addresses []string `json:"addresses"`
	}
	if err := json.Unmarshal(value, &v); err != nil || v.Outcome == "" {
		return strings.TrimSpace(string(value))
	}
	if facet != resolutionwalk.FacetResolution || len(v.Addresses) == 0 {
		return v.Outcome
	}
	addrs := append([]string(nil), v.Addresses...)
	sort.Slice(addrs, func(i, j int) bool { return addrLess(addrs[i], addrs[j]) })
	return v.Outcome + " " + strings.Join(addrs, ", ")
}

func addrLess(a, b string) bool {
	pa, ea := netip.ParseAddr(a)
	pb, eb := netip.ParseAddr(b)
	if ea != nil || eb != nil {
		return a < b
	}
	return pa.Less(pb)
}
