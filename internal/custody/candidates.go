package custody

import (
	"iter"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/seed"
)

// Pre-veto on purpose: a vetoed edge stays a candidate, so a later measurement can lift it (#985).

func (e Estate) ExtensionCitations() []Resolution {
	var out []Resolution
	// The extension limb alone, because it is the limb the veto reads; #988 measures the other one.
	seen := make(map[Resolution]struct{}, len(e.Resolutions))
	for _, r := range e.Resolutions {
		// A provider-flattened ALIAS or ANAME on a zone apex is a direct A record and arrives here.
		r.Address = r.Address.Unmap()
		if _, dup := seen[r]; dup {
			continue
		}
		// Mirrors extensionReaches: a candidate outside it probes what no extension claims.
		if IsNonGloballyReachable(r.Address) || !e.withinExtendedZone(r.Owner) {
			continue
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	return out
}

// The gain message reads the reach through this too, so one predicate serves both (ADR-0013 #55).

func (e Estate) ExtensionCandidates() []netip.Addr {
	var out []netip.Addr
	admitted := make(map[netip.Addr]struct{}, len(e.Resolutions))
	for _, r := range e.ExtensionCitations() {
		if _, dup := admitted[r.Address]; dup {
			continue
		}
		admitted[r.Address] = struct{}{}
		// Append order is the job chunking read order, so one tick matches the next (ADR-0188 §3).
		out = append(out, r.Address)
	}
	return out
}

// Two purposes, one population: the extension limb decides membership, the declaration limb labels.

func (e Estate) EdgeFanoutPopulation() iter.Seq[netip.Addr] {
	// No consent dial: the gate is total over an address, so a handshake adds no authority (#983).
	return func(yield func(netip.Addr) bool) {
		candidates := e.ExtensionCandidates()
		// Only candidates are held: a declared scope can be a /8, so no map holds one (ADR-0127).
		seen := make(map[netip.Addr]struct{}, len(candidates))
		for _, a := range candidates {
			seen[a] = struct{}{}
			if !yield(a) {
				return
			}
		}
		// Open-then-label: holding would put a pending row on every declared address on day one.
		for _, p := range e.AddressScopes {
			for a := range seed.EnumerateAddresses(p) {
				a = a.Unmap()
				// Overlapping scopes handshake twice: the cost of not holding one (ADR-0216 §2).
				if _, dup := seen[a]; dup {
					continue
				}
				// No vantage dimension on the Scan, so ADR-0079's declared-realm condition fails.
				if IsNonGloballyReachable(a) {
					continue
				}
				// An excluded /16 in a declared /8 is 65,536 addresses a tick (ADR-0133 §3).
				if e.AddressExcluded(a) {
					// Never prefix arithmetic: subtraction is easy to get wrong across families.
					continue
				}
				if !yield(a) {
					return
				}
			}
		}
	}
}
