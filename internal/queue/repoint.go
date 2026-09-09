package queue

import (
	"context"
	"encoding/json"
	"net/netip"
	"sort"
	"time"

	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/signalfacts"
)

// An Address root is read from a resolution move, never from a span of its own (ADR-0006).

type rePoint struct {
	name   string
	before map[string]bool
	after  map[string]bool
}

func rePointMessages(ctx context.Context, store messageStore, observedAt time.Time, changes []spanChange, in membershipInputs) ([]*message.Message, error) {
	moves := rePoints(changes)
	if len(moves) == 0 {
		return nil, nil
	}
	fresh, err := addressesNewToEstate(ctx, store, moves, in)
	if err != nil {
		return nil, err
	}
	var msgs []*message.Message
	for _, addr := range fresh {
		root := spanChange{SubjectKind: subjectKindAddress, SubjectKey: addr}
		if m := message.Membership(message.EntryAppeared, subjectKindAddress, addr, "", membershipCensus(changes, root), observedAt); m != nil {
			msgs = append(msgs, m)
		}
	}
	isFresh := make(map[string]bool, len(fresh))
	for _, a := range fresh {
		isFresh[a] = true
	}
	for _, mv := range moves {
		if m := message.RePoint(mv.name, rePointResidue(changes, mv.name, isFresh), observedAt); m != nil {
			msgs = append(msgs, m)
		}
	}
	return msgs, nil
}

func rePoints(changes []spanChange) []rePoint {
	var out []rePoint
	for _, c := range changes {
		// An opening roots on the Name itself, so only a move can root on an Address (ADR-0031).
		if c.Opened || c.IsGap || c.Facet != resolutionwalk.FacetResolution || c.SubjectKind != subjectKindName {
			continue
		}
		prev := c.Previous
		if c.PrevIsGap {
			prev = c.BeforeGap
		}
		out = append(out, rePoint{name: c.SubjectKey, before: citedIn(prev), after: citedIn(c.Value)})
	}
	return out
}

func addressesNewToEstate(ctx context.Context, store messageStore, moves []rePoint, in membershipInputs) ([]string, error) {
	moved := make(map[string]bool, len(moves))
	before := map[string]bool{}
	for _, mv := range moves {
		moved[mv.name] = true
		for a := range mv.before {
			before[a] = true
		}
	}
	candidates := map[string]bool{}
	for _, mv := range moves {
		for a := range mv.after {
			if !before[a] {
				candidates[a] = true
			}
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(candidates))
	for a := range candidates {
		keys = append(keys, a)
	}
	sort.Strings(keys)
	rows, err := store.ListResolutionCitersForAddresses(ctx, keys)
	if err != nil {
		return nil, err
	}
	citedElsewhere := map[string]bool{}
	for _, r := range rows {
		for _, n := range r.Citers {
			// A Name that moved here is open, so it is no prior citer of its own address (#1730).
			if !moved[n] {
				citedElsewhere[r.Addr] = true
				break
			}
		}
	}
	var fresh []string
	for _, a := range keys {
		addr, err := netip.ParseAddr(a)
		if err != nil || citedElsewhere[a] {
			continue
		}
		// A declared scope never appears, and a declared exclusion is refused ground (ADR-0047).
		if addressSeedCovered(addr, in.seeds) || coveringAddressExclusion(addr, in.exclusions) != nil {
			continue
		}
		fresh = append(fresh, a)
	}
	return fresh, nil
}

func rePointResidue(changes []spanChange, name string, fresh map[string]bool) message.Census {
	seen := map[string]bool{}
	var entries []message.CensusEntry
	for _, c := range changes {
		if !c.Opened || c.SubjectKind != subjectKindEndpoint || seen[c.SubjectKey] {
			continue
		}
		if owner, _ := signalfacts.SplitEndpointName(c.SubjectKey); owner != name {
			continue
		}
		addr, ok := subjectAddress(c.SubjectKind, c.SubjectKey)
		if !ok || fresh[addr.String()] {
			continue
		}
		seen[c.SubjectKey] = true
		entries = append(entries, message.CensusEntry{Kind: c.SubjectKind, Key: c.SubjectKey})
	}
	return message.NewCensus(entries...)
}

func citedIn(value []byte) map[string]bool {
	if len(value) == 0 {
		return nil
	}
	var v struct {
		Addresses []string `json:"addresses"`
	}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil
	}
	out := make(map[string]bool, len(v.Addresses))
	for _, a := range v.Addresses {
		if addr, err := netip.ParseAddr(a); err == nil {
			out[addr.String()] = true
		}
	}
	return out
}
