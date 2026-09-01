package custody

import "net/netip"

// EdgeFanout is the `edge-fanout` Scan's measured result, in the form the custody
// extension's reach reads it (ADR-0129 §4 as amended by #944 and #954). It is the
// SECOND input to that reach. The first is ADR-0013's label-suffix test, which says
// which in-zone-cited addresses the extension would pull in. This one says which of
// those it declines.
//
// It only ever narrows. The law, stated here because a session reasoning about
// `Custody` will look for it here and nowhere else:
//
//	A measurement may narrow a Derived reach. It may never overrule a Declared act.
//
// So this record reaches coveredByExtension and NOTHING ELSE. An address a literal
// address-scope `Seed` covers satisfies the OTHER limb of subject membership
// (ADR-0047 makes it a subject from the declaration), and Derive returns from that
// limb before the extension is asked. Such an address derives `operator` at any
// fan-out count. There is no precedence rule to write, because the two mechanisms
// are disjoint rather than ranked (ADR-0129's #956 amendment).
//
// The Scan MEASURES that limb as well (#988, EdgeFanoutPopulation), so Shared below
// carries keys for declared addresses too. Those keys reach no gate: admits is called
// from coveredByExtension alone, and Derive never gets there for a `Seed`-covered
// address. On the `Seed` limb fan-out is labelling only, and its display is #989's.
//
// A SPECIFICITY TEST IS REFUSED. The declaration does not win at a `/32` naming the
// address and lose inside a `/13` that swallows it. Every version of that is an
// invented threshold inside the safety path — the shape #27 and ADR-0013 §3 already
// refused — and it would make the boundary depend on a second product-chosen number
// beside the fan-out threshold.
//
// ADR-0013 §6 is the same law one level up (`Vantage class` reads literal address
// scopes alone, never the extension), and ADR-0079 is it pointing the other way (a
// custody extension may not open the gate over a non-globally-reachable address).
type EdgeFanout struct {
	// Enabled reports whether the `edge-fanout` Scan is IN FORCE. That is the
	// Scan's disposition and its health together: it is false where the Scan is
	// disabled, where its row is absent, and where it runs and measures nothing at
	// all. ADR-0129's fourth absence case names those last two as *errored*, and
	// the assembler decides which of them holds.
	//
	// False is the pre-ADR-0129 behaviour — the extension reaches every direct-A
	// target it holds — and it is the ZERO VALUE on purpose. An Estate assembled
	// without this input probes what it probed before the measurement existed,
	// which is the loud, wasteful direction ADR-0129 §2 accepts. The opposite zero
	// value would silently withhold every extension-covered address on an install
	// that never populated the field, and a silently missing estate is the failure
	// this whole derivation exists to avoid.
	Enabled bool

	// Shared is the measured determination per address: true where the edge
	// presented identities for at least SharedEdgeThreshold distinct registrable
	// domains. A key is present ONLY for an address the Scan measured.
	//
	// An absence is never a value. A missing key is *measurement pending*, and it
	// is a CURRENCY state rather than a count band (ADR-0129's #954 amendment), so
	// it is read by the absence rule below and never by SharedEdge.
	//
	// Keys are Unmap'ed addresses, matching the family-agnostic form every
	// comparison in this package runs over.
	Shared map[netip.Addr]bool
}

// admits reports whether the fan-out measurement lets the extension reach addr.
// This is ADR-0129's absence rule, and it is HOLD-THEN-OPEN. Four cases and no
// fifth:
//
//  1. Measured SHARED — the extension declines the reach.
//  2. Measured NOT-SHARED — the extension reaches the address.
//  3. Enabled but NOT YET MEASURED — the extension HOLDS the reach, neither
//     reaching nor declining, bounded by the Scan's daily cadence.
//  4. The Scan DISABLED or ERRORED — the extension reaches the address, the
//     behaviour before ADR-0129. Both arrive here as Enabled false; see that
//     field for what counts as errored.
//
// Case 3 is what makes the rule hold-then-open rather than open-then-narrow: no
// direct-A edge is admitted until fan-out clears it, so the modal install that
// fronts everything behind a CDN never shows appear-then-withdraw churn on its
// first two ticks. Case 4 is the only fall back to reach-everything, and it fires
// only where the feature is off.
//
// Neither hold nor decline is a new membership state. A held address and a
// declined address are both simply *not reached*, with a reason the §7 census
// renders (#987).
//
// addr is expected already Unmap'ed, as every comparison in this package is.
func (f EdgeFanout) admits(addr netip.Addr) bool {
	if !f.Enabled {
		return true // case 4
	}
	shared, measured := f.Shared[addr]
	if !measured {
		return false // case 3 — held, neither reached nor declined
	}
	return !shared // cases 1 and 2
}
