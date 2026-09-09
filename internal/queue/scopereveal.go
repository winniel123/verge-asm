package queue

import (
	"sort"
	"time"

	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/signalfacts"
)

// A Seed-covered address no Name cites has no root, and RootFires refuses one (ADR-0047, #1770).

func scopeRevealMessages(observedAt time.Time, changes []spanChange, in membershipInputs) []*message.Message {
	cited := foldCitedAddresses(changes)
	roots := foldMembershipRoots(changes)
	byScope := map[string][]message.CensusEntry{}
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
		scope := coveringSeedKey(subjectKindAddress, addr.String(), in)
		if scope == "" || coveredByFoldRoot(roots, c.SubjectKind, c.SubjectKey) {
			continue
		}
		id := [2]string{c.SubjectKind, c.SubjectKey}
		if seen[id] {
			continue
		}
		seen[id] = true
		byScope[scope] = append(byScope[scope], message.CensusEntry{Kind: c.SubjectKind, Key: c.SubjectKey})
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
	return msgs
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
