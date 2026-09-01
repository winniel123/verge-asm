package custody

import "net/netip"

// ExtensionCandidates is the address population the `edge-fanout` Scan measures: the
// addresses a custody extension would reach (ADR-0129 §6, CONTEXT.md `Scan`). Those are
// the direct A/AAAA targets of in-zone names that pass the label-suffix test — and, on
// a zone apex, the `ALIAS`/`ANAME` a provider has flattened into an A record, which is
// a direct A record on the apex name and so arrives on this same limb.
//
// It is the SET form of coveredByExtension's per-address question, and it holds that
// predicate's two stopping conditions: the address must be globally reachable, and the
// name owning the A record must itself be within a custody-extended zone. The set and
// the reach must not drift apart — a candidate outside the reach would be a probe of an
// address no extension claims, and a reached address outside the candidates would be an
// edge the veto could never narrow — so a test pins the two against each other over a
// mixed estate (TestExtensionCandidatesAgreeWithCoveredByExtension).
//
// The conditions are applied to the resolution in hand rather than by asking
// coveredByExtension per address, which would rescan every resolution for each one. This
// runs inside the dispatch transaction, under the per-scan advisory lock, over an estate
// that grows with the address-scope cap; one linear pass keeps that lock held for as
// long as the read takes and no longer.
//
// Addresses are distinct and in first-seen order. The dedup is what makes the Scan one
// handshake per address: two in-zone names flattening to the same edge is the modal
// case, and a second handshake against that edge would report what the first already
// did. The order is deterministic so one tick's fan-out matches the next.
//
// It reads the declared address scopes NOT AT ALL. Those are the *other* limb of subject
// membership: an address a `Seed` declares literally is a probed subject at any fan-out
// count, and widening this Scan onto that population — where the result labels and
// decides no membership — is #988's, not this limb's.
func (e Estate) ExtensionCandidates() []netip.Addr {
	var out []netip.Addr
	// admitted holds every address already emitted, so one edge cited by many in-zone
	// names yields one candidate. An address a resolution failed to admit is NOT
	// recorded: a later resolution may hold the same address on an in-zone owner, and
	// that one does admit it — exactly as coveredByExtension takes the first owner that
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
