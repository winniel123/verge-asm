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
	// Enabled reports whether the `edge-fanout` Scan is IN FORCE — its DISPOSITION
	// alone. It is false where the Scan is disabled and where its row is absent.
	// The Scan's HEALTH is ExtensionErrored below, and the two together are
	// ADR-0129's fourth absence case (see inForce).
	//
	// False is the pre-ADR-0129 behaviour — the extension reaches every direct-A
	// target it holds — and it is the ZERO VALUE on purpose. An Estate assembled
	// without this input probes what it probed before the measurement existed,
	// which is the loud, wasteful direction ADR-0129 §2 accepts. The opposite zero
	// value would silently withhold every extension-covered address on an install
	// that never populated the field, and a silently missing estate is the failure
	// this whole derivation exists to avoid.
	Enabled bool

	// BatchCompleted reports whether a Batch of the Scan has ever COMPLETED on this
	// install. It is what tells the Scan's two unmeasured states apart, and it is an
	// INPUT to the floor below rather than something the gate reads: a Scan that has
	// not run yet is *measurement pending* and HOLDS, and a Scan that has run and
	// measured no candidate is *errored* and REACHES.
	BatchCompleted bool

	// ExtensionErrored reports whether the Scan RAN and measured NO EXTENSION
	// CANDIDATE — the Scan's health on the limb the veto gates. It is the ERRORED
	// half of ADR-0129's fourth absence case, and it opens the reach exactly as a
	// disabled Scan does.
	//
	// It is PER LIMB, and that is the whole of #1018. Since #988 the two limbs share
	// one store, so *the store is empty* and *the extension limb measured nothing*
	// are no longer the same sentence: on an install holding an address scope AND a
	// custody extension, a declaration-limb row alone would lift a whole-store floor
	// while every extension candidate stayed unmeasured and HELD — silently, and for
	// as long as the condition lasted. The floor therefore asks *did the Scan measure
	// any extension candidate?* and never *did it record any row?*.
	//
	// It is resolved by Estate.WithEdgeFanout, which holds the candidate set this
	// question needs. Nothing else may set it: the read path (queue.ReadEdgeFanout)
	// sees rows and no candidates, and admits sees one address and no candidates.
	ExtensionErrored bool

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
	//
	// IT MAY COVER ONE LIMB ALONE. See Partial.
	Shared map[netip.Addr]bool

	// Partial reports that Shared covers a BOUND SET of addresses rather than every
	// address the Scan measured (#1036). The `/scope` render binds its read to the
	// extension candidates, so the declaration limb's rows never arrive.
	//
	// It exists because the two ways of reading Shared do not survive a bound
	// equally. A reader that LOOKS UP a key it holds a candidate for is safe: the
	// bound names that candidate, so a missing key still means what it always meant.
	// A reader that WALKS the map wholesale is not: every row the bound left behind
	// reads as an address the Scan never measured, and the answer is quietly short.
	//
	// AddressScopeCensus is that second reader, and it is the only one. It refuses a
	// partial record rather than reporting over it — see there for why no entries is
	// the honest answer on that surface where an undercount is not.
	//
	// FALSE IS THE WHOLE POPULATION, and it is the zero value. An Estate assembled
	// without this field carries a complete measurement, which is what every reader
	// assumed before the bound existed. Only the reader that narrows its own query
	// sets it, and queue.ReadEdgeFanout is that reader.
	//
	// It moves NO GATE and NO VERDICT. admits, inForce and overExtension never read
	// it: the veto's question is per address, over a candidate the bound named, so
	// the answer is the same either way.
	Partial bool
}

// admits reports whether the fan-out measurement lets the extension reach addr.
// This is ADR-0129's absence rule, and it is HOLD-THEN-OPEN. Four cases and no
// fifth:
//
//  1. Measured SHARED — the extension declines the reach.
//  2. Measured NOT-SHARED — the extension reaches the address.
//  3. Enabled but NOT YET MEASURED — the extension HOLDS the reach, neither
//     reaching nor declining, bounded by the Scan's daily cadence.
//  4. The Scan DISABLED or ERRORED on this limb — the extension reaches the
//     address, the behaviour before ADR-0129. Both arrive here through inForce;
//     see ExtensionErrored for what counts as errored.
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
	if !f.inForce() {
		return true // case 4
	}
	shared, measured := f.Shared[addr]
	if !measured {
		return false // case 3 — held, neither reached nor declined
	}
	return !shared // cases 1 and 2
}

// inForce reports whether the measurement narrows the EXTENSION limb at all. It is
// ADR-0129's fourth absence case as ONE predicate — disposition and health together —
// so the gate (admits) and the census (ExtensionCensus) cannot read the case
// differently. False is the pre-ADR-0129 behaviour: the extension reaches every
// direct-A target it holds, and the census names no decline and no hold.
func (f EdgeFanout) inForce() bool {
	return f.Enabled && !f.ExtensionErrored
}

// overExtension resolves the errored floor over the extension limb and returns the
// record carrying the verdict. candidates is the limb the veto gates
// (Estate.ExtensionCandidates), and it is the input the floor cannot be read without.
//
// The floor fires on ONE question: the Scan is enabled, it has completed a Batch, it
// holds candidates, and it measured NONE of them. That is not a lag — it is the
// measurement failing on the limb that gates, and it repeats every tick. Holding
// instead would withhold every extension-covered address, silently and for as long as
// the condition lasts, and a silently missing estate is the direction ADR-0129 §2
// refuses. So the errored limb REACHES, which is the pre-ADR-0129 behaviour the fourth
// case names.
//
// ONE MEASURED CANDIDATE LIFTS IT, and the three negative outcomes each record one. So
// an errored limb means the Scan measured NOTHING there — never that the network was
// bad — and the floor cannot be reached by a run of failed handshakes. A PARTIAL
// failure that spares some candidates leaves the rest HELD, which is case 3 doing its
// job rather than the floor failing.
//
// An install with no extension candidates is NOT errored. There is no limb to measure,
// so there is no reach to open and nothing the verdict could change.
//
// The Shared map keeps every key the Scan recorded, the declaration limb's included.
// The floor decides the EXTENSION limb's reach and moves no label: on the declaration
// limb the fan-out result gates nothing at any floor state (#988).
//
// WHAT IT STILL CANNOT TELL APART, stated rather than smoothed. BatchCompleted is
// install-wide and the candidates are the CURRENT set, so a candidate set that is
// wholly NEW reads the same as one the measurement failed on. An install that has run
// the Scan over address scopes for a while and then declares its FIRST custody
// extension, and an install that repoints its zone onto fresh addresses, both arrive
// here with a completed Batch and no measured candidate — so their candidates REACH for
// one cadence where #985 would have held them, and a shared edge among them appears and
// then withdraws. Telling a lag from a failure here needs a per-candidate signal — a
// Batch completed since the candidate set last moved, or a per-address dispatch count —
// which no input this function reads carries. The residue is in the direction ADR-0129
// §2 accepts: probing an edge it could have skipped is the loud failure, and holding
// every candidate silently and for good is the one the floor exists to stop.
//
// It is TOTAL and IDEMPOTENT: it resolves the verdict from the inputs alone and clears
// an inbound one, so re-resolving a record over a second estate cannot carry a stale
// errored reading into it.
func (f EdgeFanout) overExtension(candidates []netip.Addr) EdgeFanout {
	f.ExtensionErrored = false
	if !f.Enabled || !f.BatchCompleted || len(candidates) == 0 {
		return f
	}
	for _, addr := range candidates {
		if _, measured := f.Shared[addr]; measured {
			return f
		}
	}
	f.ExtensionErrored = true
	return f
}
