package queue

import (
	"context"
	"sort"
	"time"

	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/signalfacts"
)

// A Seed-covered address no Name cites has no root, and RootFires refuses one (ADR-0047, #1770).

type scopeCandidate struct {
	scope string
	kind  string
	key   string
	addr  string
}

func scopeRevealMessages(ctx context.Context, store messageStore, observedAt time.Time, changes []spanChange, in membershipInputs) ([]*message.Message, error) {
	candidates := darkScopeOpenings(changes, in)
	if len(candidates) == 0 {
		return nil, nil
	}
	cited, err := openCitedAddresses(ctx, store, candidates)
	if err != nil {
		return nil, err
	}
	byScope := map[string][]message.CensusEntry{}
	for _, c := range candidates {
		if cited[c.addr] {
			continue
		}
		byScope[c.scope] = append(byScope[c.scope], message.CensusEntry{Kind: c.kind, Key: c.key})
	}
	scopes := make([]string, 0, len(byScope))
	for s := range byScope {
		scopes = append(scopes, s)
	}
	sort.Strings(scopes)
	// The unit is the declaration, so a scope fires once however wide it is (ADR-0047, ADR-0052).
	msgs := make([]*message.Message, 0, len(scopes))
	for _, s := range scopes {
		if m := message.ScopeRevealed(s, message.NewCensus(byScope[s]...), observedAt); m != nil {
			msgs = append(msgs, m)
		}
	}
	return msgs, nil
}

func darkScopeOpenings(changes []spanChange, in membershipInputs) []scopeCandidate {
	cited := foldCitedAddresses(changes)
	roots := foldMembershipRoots(changes)
	var out []scopeCandidate
	seen := map[[2]string]bool{}
	for _, c := range changes {
		if !c.Opened || (c.SubjectKind != subjectKindService && c.SubjectKind != subjectKindEndpoint) {
			continue
		}
		// An Endpoint key carries the Name that cited the address, so its opening is not dark.
		if owner, _ := signalfacts.SplitEndpointName(c.SubjectKey); owner != "" {
			continue
		}
		addr, ok := subjectAddress(c.SubjectKind, c.SubjectKey)
		if !ok || cited[addr.String()] {
			continue
		}
		// An exclusion cuts the Seed limb, so narrowed ground is no aperture (ADR-0133 §3).
		if coveringAddressExclusion(addr, in.exclusions) != nil {
			continue
		}
		scope := coveringSeedKey(subjectKindAddress, addr.String(), in)
		if scope == "" || coveredByFoldRoot(roots, c.SubjectKind, c.SubjectKey) {
			continue
		}
		id := [2]string{c.SubjectKind, c.SubjectKey}
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, scopeCandidate{scope: scope, kind: c.SubjectKind, key: c.SubjectKey, addr: addr.String()})
	}
	return out
}

func openCitedAddresses(ctx context.Context, store messageStore, candidates []scopeCandidate) (map[string]bool, error) {
	seen := map[string]bool{}
	keys := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if seen[c.addr] {
			continue
		}
		seen[c.addr] = true
		keys = append(keys, c.addr)
	}
	sort.Strings(keys)
	// An unchanged open span still cites its address, and no change in this fold names it (#1779).
	rows, err := store.ListResolutionCitersForAddresses(ctx, keys)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[r.Addr] = true
	}
	return out, nil
}

func foldCitedAddresses(changes []spanChange) map[string]bool {
	out := map[string]bool{}
	for _, c := range changes {
		if c.SubjectKind != subjectKindName || c.Facet != resolutionwalk.FacetResolution {
			continue
		}
		for a := range citedIn(c.Value) {
			out[a] = true
		}
	}
	return out
}

func foldMembershipRoots(changes []spanChange) []spanChange {
	var out []spanChange
	// The same predicate membershipMessages roots on, so the two partition the fold (ADR-0031).
	for _, c := range changes {
		if c.Opened && c.Facet == resolutionwalk.FacetResolution && message.RootFires(c.SubjectKind) {
			out = append(out, c)
		}
	}
	return out
}

func coveredByFoldRoot(roots []spanChange, kind, key string) bool {
	for _, root := range roots {
		if subjectBeneathRoot(root, citedAddresses(root), kind, key) {
			return true
		}
	}
	return false
}
