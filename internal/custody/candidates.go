package custody

import (
	"iter"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/seed"
)

// ExtensionCandidates is the EXTENSION LIMB of the address population the `edge-fanout`
// Scan measures: the addresses a custody extension would reach (ADR-0129 §6, CONTEXT.md
// `Scan`). Those are the direct A/AAAA targets of in-zone names that pass the
// label-suffix test — and, on a zone apex, the `ALIAS`/`ANAME` a provider has flattened
// into an A record, which is a direct A record on the apex name and so arrives on this
// same limb.
//
// It is the SET form of extensionReaches's per-address question, and it holds that
// predicate's two stopping conditions: the address must be globally reachable, and the
// name owning the A record must itself be within a custody-extended zone. The set and
// the reach must not drift apart — a candidate outside the reach would be a probe of an
// address no extension claims, and a reached address outside the candidates would be an
// edge the veto could never narrow — so a test pins the two against each other over a
// mixed estate (TestExtensionCandidatesAgreeWithExtensionReaches).
//
// It reads the PRE-VETO reach on purpose (#985). A vetoed edge stays a candidate and is
// handshaked again on every tick, so a shared edge that later serves a dedicated
// certificate is measured again and the veto lifts. Reading the post-veto reach here
// would freeze the last measurement in place and no later one could ever contradict it.
//
// The conditions are applied to the resolution in hand rather than by asking
// extensionReaches per address, which would rescan every resolution for each one. This
// runs inside the dispatch transaction, under the per-scan advisory lock, over an estate
// that grows with the address-scope cap; one linear pass keeps that lock held for as
// long as the read takes and no longer.
//
// Addresses are distinct and in first-seen order. The dedup is what makes the Scan one
// handshake per address: two in-zone names flattening to the same edge is the modal
// case, and a second handshake against that edge would report what the first already
// did. The order is deterministic so one tick's fan-out matches the next.
//
// It reads the declared address scopes NOT AT ALL, and since #988 it is no longer the
// whole Scan population. Those scopes are the OTHER limb of subject membership, and the
// Scan measures them too — see EdgeFanoutPopulation, which unions the two. This function
// stays the extension limb ALONE, because it is the limb the VETO reads. An address a
// literal address-scope `Seed` covers is a probed subject at any fan-out count, so
// admitting one here would let a measurement overrule a Declared act (see EdgeFanout).
func (e Estate) ExtensionCandidates() []netip.Addr {
	var out []netip.Addr
	// admitted holds every address already emitted, so one edge cited by many in-zone
	// names yields one candidate. An address a resolution failed to admit is NOT
	// recorded: a later resolution may hold the same address on an in-zone owner, and
	// that one does admit it — exactly as extensionReaches takes the first owner that
	// extends and ignores the ones that do not.
	admitted := make(map[netip.Addr]struct{}, len(e.Resolutions))
	for _, r := range e.Resolutions {
		addr := r.Address.Unmap()
		if _, dup := admitted[addr]; dup {
			continue
		}
		if IsNonGloballyReachable(addr) || !e.withinExtendedZone(r.Owner) {
			continue
		}
		admitted[addr] = struct{}{}
		out = append(out, addr)
	}
	return out
}

// EdgeFanoutPopulation is the whole address population the `edge-fanout` Scan measures,
// as a LAZY SEQUENCE: the extension candidates, followed by every address a declared
// address scope covers (ADR-0129's #956 amendment, ticket #988).
//
// THE SCAN SERVES TWO PURPOSES OVER ONE POPULATION, and they must never be merged:
//
//   - On the EXTENSION limb (ExtensionCandidates) the result decides MEMBERSHIP. A
//     shared edge is vetoed: it never becomes a `Subject`, holds no `Custody`, opens no
//     `Gap` and queues no probe (EdgeFanout.admits).
//   - On the DECLARATION limb (the addresses the scopes enumerate) the result LABELS and
//     decides NOTHING. A declared address is a subject from the declaration (ADR-0047),
//     Derive returns from the address-scope limb before the extension is asked, and the
//     fan-out count moves no gate at any value.
//
// THE TWO ABSENCE RULES ARE OPPOSITE, and that is the point of the split:
//
//   - Extension limb: HOLD-THEN-OPEN. An unmeasured candidate is held, neither reached
//     nor declined, bounded by the Scan's daily cadence (EdgeFanout.admits case 3).
//   - Declaration limb: OPEN-THEN-LABEL. An unmeasured declared address is probed
//     normally and carries no row at all. The label appears once fan-out has measured
//     it. Carrying hold-then-open across by analogy would hold a declared subject out of
//     its own scope's census, and would put a *pending* row on every address of every
//     scope on the first day — noise rather than a census.
//
// The Scan still adds NO CONSENT DIAL. #983 rested that on two clauses, and the second
// — *it reduces total probing* — DOES NOT HOLD HERE: on a declared address the handshake
// narrows nothing and adds a connect. The first clause carries the authority alone: the
// probing gate is total over an `Address` (MayProbe), so an address a `Seed` covers is
// already connected to on every port, and one further handshake asks for no authority
// the declaration did not already give.
//
// A non-globally-reachable declared address is NOT in the population. The Scan has no
// vantage dimension, so it cannot satisfy ADR-0079's *from a Vantage that is not
// `internet`-class*, and the shared egress guard (EgressGuard) refuses the dial anyway.
// Dispatching one would spend a job to record `unreachable` and label nothing.
//
// IT IS STREAMED, AND THE DEDUP IS AGAINST THE SMALL SET ONLY — the same shape
// queue.candidateAddrs holds for the hot and cold tiers, and for the same reason
// (ADR-0127). The per-scope address cap does NOT bound this: ADR-0127 removed the
// ceiling above the operator's value, so a declared scope can be an IPv4 `/8`, and
// ADR-0047 refuses a scan-time aperture that would truncate one. So no record holds the
// whole scope. `seen` holds the EXTENSION CANDIDATES alone, which is what keeps a
// dual-limb address — declined by the extension and covered by a `Seed` at once (#987) —
// to one handshake rather than two. Two declared scopes that OVERLAP are not deduped
// against each other, so the overlap is handshaked twice; that is the accepted cost of
// not holding a scope in a map, each handshake is its own idempotent Batch, and the
// newest row per address is what the read takes.
//
// Order is deterministic — extension limb first, then the scopes as declared — so one
// tick's fan-out matches the next. A consumer that stops early stops the walk.
func (e Estate) EdgeFanoutPopulation() iter.Seq[netip.Addr] {
	return func(yield func(netip.Addr) bool) {
		candidates := e.ExtensionCandidates()
		seen := make(map[netip.Addr]struct{}, len(candidates))
		for _, a := range candidates {
			seen[a] = struct{}{}
			if !yield(a) {
				return
			}
		}
		for _, p := range e.AddressScopes {
			for a := range seed.EnumerateAddresses(p) {
				a = a.Unmap()
				if _, dup := seen[a]; dup {
					continue // a dual-limb address — measured once, on the limb that gates
				}
				if IsNonGloballyReachable(a) {
					continue
				}
				if e.AddressExcluded(a) {
					// A declared `address` exclusion cuts the Seed limb, so the address
					// is out of the DECLARATION limb of this population (ADR-0133 §3).
					// The gate already refuses it, so this skip is for COST: an excluded
					// /16 inside a declared /8 is 65,536 addresses walked per tick and
					// refused one at a time. The skip is a `continue` and never prefix
					// arithmetic — subtracting an excluded /25 from a scope produces a
					// set of covering prefixes and is easy to get wrong at the family
					// boundary.
					//
					// The EXTENSION limb above is NOT narrowed. An excluded address a
					// custody extension reaches still derives operator and is still
					// probed, so it stays a candidate and stays measured.
					continue
				}
				if !yield(a) {
					return
				}
			}
		}
	}
}
