package custody

import (
	"iter"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/seed"
)

// Pre-veto on purpose: a vetoed edge stays a candidate, so a later measurement can lift it (#985).

func (e Estate) ExtensionCandidates() []netip.Addr {
	var out []netip.Addr
	// The extension limb alone, because it is the limb the veto reads; #988 measures the other one.
	admitted := make(map[netip.Addr]struct{}, len(e.Resolutions))
	// One linear pass over the resolutions, because this runs under the per-scan advisory lock.
	for _, r := range e.Resolutions {
		// A provider-flattened ALIAS or ANAME on a zone apex is a direct A record and arrives here.
		addr := r.Address.Unmap()
		if _, dup := admitted[addr]; dup {
			continue
		}
		// Mirrors extensionReaches: a candidate outside it probes what no extension claims.
		if IsNonGloballyReachable(addr) || !e.withinExtendedZone(r.Owner) {
			continue
		}
		// A rejected address is not recorded: a later resolution may hold it on an in-zone owner.
		admitted[addr] = struct{}{}
		// The append order is the order the job chunking reads, so one tick matches the next (ADR-0188 §3).
		out = append(out, addr)
	}
	return out
}

// Two purposes, one population: the extension limb decides membership, the declaration limb labels.

func (e Estate) EdgeFanoutPopulation() iter.Seq[netip.Addr] {
	// No consent dial: the gate is total over an address, so a handshake adds no authority (#983).
	return func(yield func(netip.Addr) bool) {
		candidates := e.ExtensionCandidates()
		// Only the candidates are held: a declared scope can be a /8, so no map holds one (ADR-0127).
		seen := make(map[netip.Addr]struct{}, len(candidates))
		for _, a := range candidates {
			seen[a] = struct{}{}
			if !yield(a) {
				return
			}
		}
		// Open-then-label here: holding would put a pending row on every declared address on day one.
		for _, p := range e.AddressScopes {
			for a := range seed.EnumerateAddresses(p) {
				a = a.Unmap()
				// Overlapping declared scopes are handshaked twice, the accepted cost of not holding a scope (ADR-0216 §2).
				if _, dup := seen[a]; dup {
					continue
				}
				// The Scan has no vantage dimension, so it cannot satisfy ADR-0079's declared-realm condition.
				if IsNonGloballyReachable(a) {
					continue
				}
				// A cost skip: an excluded /16 inside a declared /8 is 65,536 addresses per tick (ADR-0133 §3).
				if e.AddressExcluded(a) {
					// Never prefix arithmetic: subtraction is easy to get wrong at the family boundary.
					continue
				}
				if !yield(a) {
					return
				}
			}
		}
	}
}
