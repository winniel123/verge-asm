package queue

import (
	"encoding/json"
	"net/netip"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/estate"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/message"
)

// An Address root is read from a resolution move, never from a span of its own (ADR-0006).

type rePoint struct {
	name          string
	discriminator string
	vantageID     pgtype.Int8
	source        string
	before        map[string]bool
	after         map[string]bool
}

// One citing timeline, from the fold's own read or from the poll's at-instant one (#1818).

type citerRef struct {
	addr          string
	subjectKey    string
	discriminator string
	vantageID     pgtype.Int8
	source        string
}

func foldCiters(rows []db.ListResolutionCitersForAddressesRow) []citerRef {
	out := make([]citerRef, 0, len(rows))
	for _, r := range rows {
		out = append(out, citerRef{addr: r.Addr, subjectKey: r.SubjectKey, discriminator: r.Discriminator, vantageID: r.VantageID, source: r.Source})
	}
	return out
}

func citersAtInstant(rows []db.ListResolutionCitersForAddressesAtRow) []citerRef {
	out := make([]citerRef, 0, len(rows))
	for _, r := range rows {
		out = append(out, citerRef{addr: r.Addr, subjectKey: r.SubjectKey, discriminator: r.Discriminator, vantageID: r.VantageID, source: r.Source})
	}
	return out
}

// ADR-0026 §2's residue is read in the poll, so the fold writes the Address root alone (#1818).

func rePointMessages(batchID int64, observedAt time.Time, changes []spanChange, moves []rePoint, in membershipInputs, citers []db.ListResolutionCitersForAddressesRow, holdCensus bool) []*message.Message {
	if len(moves) == 0 {
		return nil
	}
	var msgs []*message.Message
	for _, addr := range addressesNewToEstate(moves, in, foldCiters(citers)) {
		root := spanChange{SubjectKind: subjectKindAddress, SubjectKey: addr}
		if m := membershipRoot(batchID, observedAt, message.EntryAppeared, root, "", changes, holdCensus); m != nil {
			msgs = append(msgs, m)
		}
	}
	return msgs
}

func rePoints(changes []spanChange) []rePoint {
	var out []rePoint
	for _, c := range changes {
		// An opening roots on the Name itself, so only a move can root on an Address (ADR-0031).
		if c.Opened || c.IsGap || c.Facet != resolutionwalk.FacetResolution || c.SubjectKind != subjectKindName {
			continue
		}
		// A Gap-closing edge is coverage by construction, so gapclose alone carries it (ADR-0014).
		if c.PrevIsGap {
			continue
		}
		out = append(out, rePoint{
			name:          c.SubjectKey,
			discriminator: c.Discriminator,
			vantageID:     c.VantageID,
			source:        c.Source,
			before:        citedIn(c.Previous),
			after:         citedIn(c.Value),
		})
	}
	return out
}

func (mv rePoint) sameTimeline(r citerRef) bool {
	return r.subjectKey == mv.name && r.discriminator == mv.discriminator &&
		r.vantageID == mv.vantageID && r.source == mv.source
}

func rePointCandidateAddresses(moves []rePoint) []string {
	before := map[string]bool{}
	for _, mv := range moves {
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
	keys := make([]string, 0, len(candidates))
	for a := range candidates {
		keys = append(keys, a)
	}
	sort.Strings(keys)
	return keys
}

func addressesNewToEstate(moves []rePoint, in membershipInputs, citers []citerRef) []string {
	keys := rePointCandidateAddresses(moves)
	if len(keys) == 0 {
		return nil
	}
	want := make(map[string]bool, len(keys))
	for _, a := range keys {
		want[a] = true
	}
	citedElsewhere := map[string]bool{}
	for _, r := range citers {
		// The read is shared, so it carries rows the other producer asked for (#1784).
		if !want[r.addr] {
			continue
		}
		// A timeline that moved here is open, so it is no prior citer of its address (#1730).
		if movedTimeline(moves, r) {
			continue
		}
		citedElsewhere[r.addr] = true
	}
	var fresh []string
	for _, a := range keys {
		addr, err := netip.ParseAddr(a)
		if err != nil {
			continue
		}
		if estate.AddressPresent(citedElsewhere[a], addressSeedCovered(addr, in.seeds)) {
			continue
		}
		// A declared exclusion is refused ground, so it is never announced as new (ADR-0047).
		if coveringAddressExclusion(addr, in.exclusions) != nil {
			continue
		}
		fresh = append(fresh, a)
	}
	return fresh
}

func movedTimeline(moves []rePoint, r citerRef) bool {
	for _, mv := range moves {
		if mv.sameTimeline(r) {
			return true
		}
	}
	return false
}

func rePointResidue(mv rePoint, subjects []subjectRef, fresh map[string]bool, roots []spanChange) message.Census {
	seen := map[string]bool{}
	var entries []message.CensusEntry
	for _, s := range subjects {
		if s.kind != subjectKindEndpoint || seen[s.key] {
			continue
		}
		addr, ok := subjectAddress(s.kind, s.key)
		if !ok {
			continue
		}
		// Only an Endpoint beneath a newly cited address is the move's consequence (ADR-0026 §2).
		key := addr.String()
		if !mv.after[key] || mv.before[key] || fresh[key] {
			continue
		}
		// fresh covers an Address root, so a Name root needs its own test (ADR-0026 §2).
		if coveredByFoldRoot(roots, s.kind, s.key) {
			continue
		}
		seen[s.key] = true
		entries = append(entries, message.CensusEntry{Kind: s.kind, Key: s.key})
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
