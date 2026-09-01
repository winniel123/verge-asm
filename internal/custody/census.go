package custody

import "net/netip"

// ExtensionState is the resting state of one candidate edge on the custody
// extension's limb — the two states the §7 census renders, and nothing else
// (ADR-0129 §5 as amended by #944 and #954; #987).
//
// There is no `reached` state. An edge the extension reaches is an ordinary
// covered address that carries `Custody` and queues probes, and it is the census
// with a denominator that counts it. This census is the OTHER register: it names
// what the extension does NOT reach, and why.
type ExtensionState string

const (
	// ExtensionDeclined: fan-out measured the edge as shared, so the extension
	// does not reach it. The remedy is the operator's — declare the origin
	// addresses as an address scope, and the true origin is monitored directly.
	ExtensionDeclined ExtensionState = "declined"
	// ExtensionPending: the `edge-fanout` Scan is in force and has not measured
	// the candidate yet, so the extension HOLDS the reach. It is a currency state
	// bounded by the Scan's daily cadence, never a count band, and it is the one
	// state that resolves on its own.
	ExtensionPending ExtensionState = "pending"
)

// ExtensionCensusEntry is one row of the custody-extension census: an in-zone Name
// whose direct A record cites an edge the extension does not reach, and the reason.
//
// It carries NO count and NO threshold. The fan-out figure and the boundary it is
// compared against are parameters of this derivation (SharedEdgeThreshold, locked
// by the `custody/v2` corpus), and a row that rendered either would put a
// product-chosen number in front of the operator as if it were their business
// (ADR-0129 §5, #987).
type ExtensionCensusEntry struct {
	// Name is the in-zone Name holding the A record — the CITING name, which is
	// what the operator recognises. It is never the edge's own reverse name.
	Name string
	// Address is the edge the record points at, Unmap'ed as every address in this
	// package is.
	Address netip.Addr
	// State is why the extension does not reach it.
	State ExtensionState
	// Scope is the declared address scope that ALSO covers the edge, and the zero
	// Prefix where none does. A valid Prefix here is ADR-0129's dual-limb row: the
	// address is declined by the extension and covered by a `Seed` at once, and the
	// row must state both limbs.
	//
	// The two facts are disjoint rather than ranked (the #956 amendment), so this
	// is not a precedence marker. A bare *declined* would be true about the
	// extension and read as a contradiction to the person the census exists for,
	// and dropping the row would hide a decline they need if they later withdraw
	// the `Seed`.
	Scope netip.Prefix
}

// ExtensionCensus is the custody-extension census: one entry per (in-zone Name,
// cited edge) pair the extension REACHES FOR but does not reach (ADR-0129 §5, #987).
//
// It reads the same two stopping conditions extensionReaches reads — the address
// must be globally reachable, and the name owning the A record must itself be within
// a custody-extended zone — and then splits the pre-veto reach by the measurement:
//
//   - Measured SHARED — declined.
//   - Enabled but NOT YET MEASURED — pending.
//   - Measured NOT-SHARED — NO ENTRY. The extension reaches it, and a reached
//     address is not this census's business.
//
// It walks the RESOLUTIONS, so an address a declared address scope covers and no
// in-zone name cites gets no entry — measured or not. That is #988's open-then-label
// absence rule holding: on the declaration limb an unmeasured address is probed
// normally and carries no row, and a *pending* row on every address of every scope on
// the first day would be noise rather than a census.
//
// A Scan that is not in force ON THIS LIMB yields NO ENTRIES AT ALL — disabled, its
// row absent, or errored over the extension candidates (EdgeFanout.inForce). Nothing
// is declined and nothing is held where the measurement does not narrow — that is
// EdgeFanout's fourth absence case, the pre-ADR-0129 behaviour — so a row here would
// name a decline that did not happen. The census must never fabricate a row.
//
// It reads inForce and never Enabled, so the census and the gate cannot disagree about
// the errored case. An errored extension limb REACHES every candidate, and a *pending*
// row beside a reached address would name a hold that is not happening (#1018).
//
// Entries are one per (Name, Address) pair, in resolution order, so the render is
// deterministic. Two in-zone names citing the same shared edge are TWO entries:
// each names its own citing name, which is the one thing the operator can act on.
// The same name citing the same edge twice is one.
func (e Estate) ExtensionCensus() []ExtensionCensusEntry {
	if !e.edgeFanout.inForce() {
		return nil
	}
	type pair struct {
		name string
		addr netip.Addr
	}
	var out []ExtensionCensusEntry
	seen := make(map[pair]struct{}, len(e.Resolutions))
	for _, r := range e.Resolutions {
		addr := r.Address.Unmap()
		if IsNonGloballyReachable(addr) || !e.withinExtendedZone(r.Owner) {
			continue
		}
		shared, measured := e.edgeFanout.Shared[addr]
		if measured && !shared {
			continue
		}
		key := pair{name: r.Owner, addr: addr}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		state := ExtensionPending
		if measured {
			state = ExtensionDeclined
		}
		scope, _ := e.coveringAddressScope(addr)
		out = append(out, ExtensionCensusEntry{
			Name: r.Owner, Address: addr, State: state, Scope: scope,
		})
	}
	return out
}
