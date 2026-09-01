package custody

import "net/netip"

// AddressScopeCensusEntry is one declared address scope that holds measured shared
// edges — ADR-0129's #956 contradiction row, and the display half of the amendment
// (#989).
//
// The system now holds evidence against a declaration. An operator may declare a CIDR
// containing measured shared edges, enumeration walks it, and the estate probes a
// provider's edge WHILE HOLDING A MEASUREMENT THAT SAYS SO. ADR-0013 tolerated a false
// declaration because it held nothing to the contrary; that was a statement about
// ignorance, never a licence for silence. The evidence exists now, and evidence held
// and not shown is the fails-silently shape the model refuses.
//
// THE ROW IS DISPLAY AND NEVER A GATE. A gate here would be the veto reading the
// `Seed` limb, which EdgeFanout refuses: a measurement may narrow a Derived reach, and
// it may never overrule a Declared act. Every address counted below is still a subject,
// still derives `operator`, and is still probed. Nothing in this file is read by
// Derive, coveredByExtension or MayProbe, and nothing in it may become so.
//
// It carries NO THRESHOLD and NO VERDICT. SharedEdges is a count of the operator's own
// declared addresses, and the boundary it was compared against stays inside the
// versioned derivation (SharedEdgeThreshold, locked by the `custody/v1` corpus).
// *You may have over-asserted* is the sentence ADR-0013's nag test forbids, and it is
// forbidden here. The renderer states the two counts and names the remedy — a `Seed`
// exclusion, which ADR-0012 already extended to address scopes.
//
// THAT REMEDY IS NOT ENFORCED TODAY, and #1022 holds the gap. ADR-0012 §125 extended
// exclusions to address scopes and no gate was ever built on that limb: an address
// exclusion narrows neither ListAddressScopeCidrs, nor the enumeration, nor
// EdgeFanoutPopulation, nor Derive, nor MayProbe. So an operator who takes the row's
// remedy sees the same row at the same count on the next load. The row still names it,
// because ADR-0129's #956 amendment names it and the instrument is meant to work — the
// fix belongs on the exclusion limb and not on this display.
type AddressScopeCensusEntry struct {
	// Scope is the declared address-scope `Seed`, canonical, as the operator wrote
	// it. It is the thing they can act on: an exclusion is declared against a scope.
	Scope netip.Prefix
	// SharedEdges is how many addresses inside Scope fan-out MEASURED as shared. It
	// is never the scope's size and never a proportion, so the renderer must state
	// the denominator beside it.
	SharedEdges int
}

// AddressScopeCensus is the address-scope limb's census: one entry per declared
// address scope holding at least one measured shared edge (ADR-0129's #956 amendment,
// #989).
//
// Absence on this limb is OPEN-THEN-LABEL, which is the opposite of the extension
// limb's hold-then-open (see EdgeFanout.admits). ADR-0047 makes a declared address a
// subject from the declaration, walked every cadence whether or not anything has ever
// answered there — so there is no reach to withhold and nothing to hold. An unmeasured
// declared address is probed normally and CARRIES NO ROW; the row appears once fan-out
// has measured it. This is written down because a session will otherwise carry
// hold-then-open across by analogy and put a *pending* row on every address of every
// scope on the first day, which is noise rather than a census.
//
// A `Scan` that is not in force yields NO ENTRIES AT ALL — EdgeFanout's fourth absence
// case. The census must never name evidence the Scan does not hold.
//
// A scope with no measured shared edge yields no entry either. The register's whole
// worth is that it fires on evidence alone.
//
// It walks the MEASUREMENT rather than the scope, so its cost is the measured-address
// count and never the enumerable size of a declared range. ADR-0127 left the operator's
// address cap with no ceiling above it, so a declared scope can be an IPv4 `/8` and
// enumerating one here would allocate until it failed.
//
// It walks the measurement ONCE, and the scopes inside that walk. The measured store
// holds one row per globally-reachable address of every declared scope since #988, and
// the declared scopes are a handful, so a pass per scope would read the large side
// repeatedly to read the small side once.
//
// Two OVERLAPPING scopes both count the address they share. That is the refused
// specificity test holding on the display (see coveringAddressScope): the remedy is
// declared against one scope, so the operator has to see the edge on each scope they
// could exclude it from. The same scope declared twice is one entry — two identical
// rows would render one fact twice and read as two scopes.
//
// Entries are in declaration order, so one render matches the next.
func (e Estate) AddressScopeCensus() []AddressScopeCensusEntry {
	if !e.EdgeFanout.Enabled {
		return nil
	}
	scopes := make([]netip.Prefix, 0, len(e.AddressScopes))
	counts := make(map[netip.Prefix]int, len(e.AddressScopes))
	for _, p := range e.AddressScopes {
		if _, dup := counts[p]; dup {
			continue
		}
		counts[p] = 0
		scopes = append(scopes, p)
	}
	for addr, shared := range e.EdgeFanout.Shared {
		if !shared {
			continue
		}
		for _, p := range scopes {
			if p.Contains(addr) {
				counts[p]++
			}
		}
	}
	var out []AddressScopeCensusEntry
	for _, p := range scopes {
		if counts[p] == 0 {
			continue
		}
		out = append(out, AddressScopeCensusEntry{Scope: p, SharedEdges: counts[p]})
	}
	return out
}
