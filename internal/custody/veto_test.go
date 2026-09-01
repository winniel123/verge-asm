package custody

import (
	"net/netip"
	"testing"
)

// The edge every fixture here measures. It is globally reachable on purpose: the
// documentation ranges are not, so an extension covers none of them and a fixture built
// from one would pass for the wrong reason.
var edge = netip.MustParseAddr("104.16.132.229")

// extended is one custody-extended zone whose in-zone name holds a direct A record on
// the edge — the exact shape the veto narrows.
func extended(f EdgeFanout) Estate {
	return Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions:   []Resolution{{Owner: "www.example.com", Address: edge}},
		edgeFanout:    f,
	}
}

// measured is an EdgeFanout holding one determination for the edge.
func measured(shared bool) EdgeFanout {
	return EdgeFanout{Enabled: true, Shared: map[netip.Addr]bool{edge: shared}}
}

// The four absence cases of ADR-0129's hold-then-open rule, read straight off the
// derivation. Nothing else in this package decides them, so this table is the rule.
func TestTheFourAbsenceCases(t *testing.T) {
	cases := []struct {
		name    string
		fanout  EdgeFanout
		want    Custody
		because string
	}{
		{
			name:    "measured shared declines the reach",
			fanout:  measured(true),
			want:    ThirdParty,
			because: "a measured shared foreign edge stays outside the estate",
		},
		{
			name:    "measured not-shared reaches the address",
			fanout:  measured(false),
			want:    Operator,
			because: "fan-out cleared the edge, so the extension covers it",
		},
		{
			name:    "enabled but not yet measured holds the reach",
			fanout:  EdgeFanout{Enabled: true},
			want:    ThirdParty,
			because: "hold-then-open admits no direct-A edge until fan-out clears it",
		},
		{
			name:    "a disabled Scan reaches the address",
			fanout:  EdgeFanout{},
			want:    Operator,
			because: "the pre-ADR-0129 behaviour, the only fall back to reach-everything",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extended(tc.fanout).Derive(edge); got != tc.want {
				t.Fatalf("Derive(%s) = %s, want %s — %s", edge, got, tc.want, tc.because)
			}
		})
	}
}

// A vetoed address is refused on every port, from every Vantage class, at every rate.
// MayProbe takes no port- or rate-shaped argument, so the class dimension is the whole
// of what a caller can vary, and the veto must close all of it (ADR-0019).
func TestMayProbeRefusesAVetoedAddressFromEveryClass(t *testing.T) {
	e := extended(measured(true))
	for _, vc := range []VantageClass{ClassInternet, ClassInternal, ClassUnverified} {
		if e.MayProbe(edge, vc) {
			t.Fatalf("MayProbe(%s, %s) = true, want false — a vetoed edge queues no probe", edge, vc)
		}
	}
	// The same estate with the edge cleared opens every class, so the refusal above is
	// the veto and not the fixture.
	cleared := extended(measured(false))
	for _, vc := range []VantageClass{ClassInternet, ClassInternal, ClassUnverified} {
		if !cleared.MayProbe(edge, vc) {
			t.Fatalf("MayProbe(%s, %s) = false on a cleared edge, want true", edge, vc)
		}
	}
}

// A measured shared edge and a CNAME-to-foreign edge reach the SAME RESTING STATE. The
// veto adds no population and no new value: it applies the existing foreign-boundary
// behaviour to one more case, which is what makes "never becomes a `Subject`, holds no
// `Custody` value, opens no `Gap`" true without a line of code saying so.
func TestAVetoedEdgeIsIndistinguishableFromACNAMEToForeign(t *testing.T) {
	vetoed := extended(measured(true))
	// The same edge cited by a FOREIGN owner — the A record after a CNAME leaves the
	// declared zone — with no fan-out measurement in play at all.
	foreign := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions:   []Resolution{{Owner: "edge.provider.net", Address: edge}},
	}
	if got, want := vetoed.Derive(edge), foreign.Derive(edge); got != want {
		t.Fatalf("vetoed Derive = %s, CNAME-to-foreign Derive = %s — the two must rest alike", got, want)
	}
	for _, vc := range []VantageClass{ClassInternet, ClassInternal, ClassUnverified} {
		if got, want := vetoed.MayProbe(edge, vc), foreign.MayProbe(edge, vc); got != want {
			t.Fatalf("at class %s: vetoed MayProbe = %v, CNAME-to-foreign MayProbe = %v", vc, got, want)
		}
	}
	if got, want := vetoed.CoversAddressScope(edge), foreign.CoversAddressScope(edge); got != want {
		t.Fatalf("vetoed CoversAddressScope = %v, CNAME-to-foreign = %v", got, want)
	}
}

// A literal address-scope `Seed` is DISJOINT from the veto, never ranked against it. An
// address the operator declared derives `operator` at any fan-out count, because the
// veto is scoped to the resolution limb and the declaration satisfies the other one.
//
// A measurement may narrow a Derived reach. It may never overrule a Declared act.
func TestASeedCoveredAddressIsOperatorAtAnyFanOutCount(t *testing.T) {
	e := extended(measured(true))
	e.AddressScopes = []netip.Prefix{netip.MustParsePrefix("104.16.132.0/24")}

	if got := e.Derive(edge); got != Operator {
		t.Fatalf("Derive(%s) = %s, want %s — a declared address scope covers it", edge, got, Operator)
	}
	if !e.MayProbe(edge, ClassInternet) {
		t.Fatal("MayProbe = false on a declared address — the veto reached the wrong limb")
	}
	// The declaration wins at a /24 here. It wins identically inside a /13: a
	// SPECIFICITY TEST IS REFUSED, so no prefix length changes the answer.
	wide := e
	wide.AddressScopes = []netip.Prefix{netip.MustParsePrefix("104.16.0.0/13")}
	if got := wide.Derive(edge); got != Operator {
		t.Fatalf("Derive(%s) inside a /13 = %s, want %s — the veto must not read prefix length", edge, got, Operator)
	}
	// And with no custody extension in play at all, so the address reaches the
	// declaration limb alone.
	declaredOnly := Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("104.16.0.0/13")},
		edgeFanout:    measured(true),
	}
	if got := declaredOnly.Derive(edge); got != Operator {
		t.Fatalf("Derive(%s) on the Seed limb alone = %s, want %s", edge, got, Operator)
	}
}

// CoversAddressScope is the `covered` predicate the Vantage-class derivation binds. It
// reads declared address scopes ALONE, so the veto must not move it in either
// direction: a vantage's side of the boundary is decided by the declaration and never
// by an extension, measured or not (CONTEXT.md `Vantage class`, ADR-0013 §6).
func TestCoversAddressScopeIsUnchangedByTheVeto(t *testing.T) {
	declared := netip.MustParseAddr("104.16.132.10")
	// Cited by an in-zone name and OUTSIDE the declared scope, so the extension is the
	// only thing that could ever admit it.
	cited := netip.MustParseAddr("93.184.216.34")
	e := Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("104.16.132.0/24")},
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "www.example.com", Address: cited},
			{Owner: "api.example.com", Address: declared},
		},
		edgeFanout: EdgeFanout{Enabled: true, Shared: map[netip.Addr]bool{cited: true, declared: true}},
	}
	if !e.CoversAddressScope(declared) {
		t.Fatal("a declared address measured as shared left the address-scope coverage")
	}
	// An extension never admits an address to this predicate — vetoed or cleared.
	if e.CoversAddressScope(cited) {
		t.Fatal("an extension-cited address entered the address-scope coverage")
	}
	cleared := e
	cleared.edgeFanout = EdgeFanout{Enabled: true, Shared: map[netip.Addr]bool{cited: false, declared: false}}
	if cleared.CoversAddressScope(cited) {
		t.Fatal("clearing the fan-out admitted an extension-cited address to the address-scope coverage")
	}
}

// A vetoed edge stays a candidate of the `edge-fanout` Scan. The population is the
// PRE-veto reach, so the edge is handshaked again on the next tick and a later
// measurement can lift the veto. Reading the post-veto reach into the population would
// freeze the first shared result forever.
func TestAVetoedEdgeStaysACandidate(t *testing.T) {
	e := extended(measured(true))
	got := e.ExtensionCandidates()
	if len(got) != 1 || got[0] != edge {
		t.Fatalf("candidates = %v, want [%s] — a veto must not narrow the population it is measured from", got, edge)
	}
	if e.Derive(edge) != ThirdParty {
		t.Fatal("the fixture did not actually veto the edge")
	}
}

// The determination is keyed on the address, never on a spelling. An IPv4-mapped IPv6
// resolution of a vetoed edge is the same edge and is vetoed too.
func TestTheVetoFoldsMappedSpellings(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions:   []Resolution{{Owner: "www.example.com", Address: netip.MustParseAddr("::ffff:104.16.132.229")}},
		edgeFanout:    measured(true),
	}
	if got := e.Derive(netip.MustParseAddr("::ffff:104.16.132.229")); got != ThirdParty {
		t.Fatalf("Derive(mapped) = %s, want %s — the lookup turned on a rendering", got, ThirdParty)
	}
}

// A non-globally-reachable address is stopped by the reach itself, before the fan-out
// is consulted. Clearing it in the measurement must not open it: an extension declares
// no realm, and ADR-0079's stopping condition is not the veto's to lift.
func TestAClearedMeasurementDoesNotOpenANonGloballyReachableAddress(t *testing.T) {
	private := netip.MustParseAddr("10.0.0.5")
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions:   []Resolution{{Owner: "api.example.com", Address: private}},
		edgeFanout:    EdgeFanout{Enabled: true, Shared: map[netip.Addr]bool{private: false}},
	}
	if got := e.Derive(private); got != ThirdParty {
		t.Fatalf("Derive(%s) = %s, want %s — a measurement may narrow a reach, never widen one", private, got, ThirdParty)
	}
}

// A measurement for an address no extension reaches changes nothing. The fan-out result
// is a second input to the reach, not a reach of its own, so a stray row cannot pull a
// foreign address into the estate.
func TestAClearedMeasurementDoesNotWidenTheReach(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions:   []Resolution{{Owner: "edge.provider.net", Address: edge}},
		edgeFanout:    measured(false),
	}
	if got := e.Derive(edge); got != ThirdParty {
		t.Fatalf("Derive(%s) = %s, want %s — a cleared measurement widened the reach", edge, got, ThirdParty)
	}
}

// declaredEdge is an address a declared address scope covers, cited by an OUT-of-zone
// owner. It is a DECLARATION-limb address and never an extension candidate, which is
// what lets a fixture record a row on one limb alone.
var declaredEdge = netip.MustParseAddr("23.20.0.20")

// bothLimbs is the estate the per-limb floor exists for: one custody extension whose
// in-zone name cites `edge`, and one declared address scope covering declaredEdge. The
// caller supplies the measurement, which WithEdgeFanout takes with its floor resolved.
func bothLimbs(f EdgeFanout) Estate {
	return Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("23.20.0.0/24")},
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "www.example.com", Address: edge},
			{Owner: "edge.provider.net", Address: declaredEdge},
		},
	}.WithEdgeFanout(f)
}

// A DECLARATION-LIMB ROW ALONE DOES NOT LIFT THE FLOOR (#1018). The Scan is enabled, it
// completed a Batch, and it measured one declared address and no extension candidate.
// That is the measurement failing on the limb the veto gates, and it repeats every
// tick — so the extension limb is errored and REACHES, which is case 4.
//
// Before the floor was per limb this estate read as in force: the store held a row, so
// `edge` stayed unmeasured and was HELD by case 3, silently and for as long as the
// condition lasted.
func TestADeclarationLimbRowAloneDoesNotLiftTheFloor(t *testing.T) {
	e := bothLimbs(EdgeFanout{
		Enabled:        true,
		BatchCompleted: true,
		Shared:         map[netip.Addr]bool{declaredEdge: true},
	})
	if got := e.Derive(edge); got != Operator {
		t.Fatalf("Derive(%s) = %s, want %s — the extension limb measured nothing and must reach", edge, got, Operator)
	}
	if !e.MayProbe(edge, ClassInternet) {
		t.Fatal("MayProbe = false on an errored extension limb — the reach did not open")
	}
	// The census reads the same floor, so it names no decline and no hold. A *pending*
	// row beside a reached address would state a hold that is not happening.
	if got := e.ExtensionCensus(); len(got) != 0 {
		t.Fatalf("ExtensionCensus = %+v on an errored limb, want none", got)
	}
}

// A Scan that has completed NO BATCH still HOLDS its extension candidates. It is the
// same estate one bit earlier — the fresh install and the newly-declared extension —
// and holding here is what keeps the modal all-CDN install from showing
// appear-then-withdraw churn. The floor must not swallow this case.
func TestAScanWithNoCompletedBatchStillHoldsItsCandidates(t *testing.T) {
	e := bothLimbs(EdgeFanout{
		Enabled: true,
		Shared:  map[netip.Addr]bool{declaredEdge: true},
	})
	if got := e.Derive(edge); got != ThirdParty {
		t.Fatalf("Derive(%s) = %s, want %s — a Scan that has not run holds, it does not reach", edge, got, ThirdParty)
	}
	entries := e.ExtensionCensus()
	if len(entries) != 1 || entries[0].State != ExtensionPending {
		t.Fatalf("ExtensionCensus = %+v, want one pending row for the held candidate", entries)
	}
}

// ONE MEASURED CANDIDATE LIFTS THE FLOOR, and the rest stay HELD. A partial failure is
// case 3 doing its job, not the floor failing: the Scan demonstrably measured this
// limb, so an unmeasured candidate on it is a lag and is bounded by the daily cadence.
func TestOneMeasuredCandidateLeavesTheRestHeld(t *testing.T) {
	second := netip.MustParseAddr("93.184.216.34")
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "www.example.com", Address: edge},
			{Owner: "api.example.com", Address: second},
		},
	}.WithEdgeFanout(EdgeFanout{
		Enabled:        true,
		BatchCompleted: true,
		Shared:         map[netip.Addr]bool{edge: false},
	})
	if got := e.Derive(edge); got != Operator {
		t.Fatalf("Derive(%s) = %s, want %s — the measured candidate cleared", edge, got, Operator)
	}
	if got := e.Derive(second); got != ThirdParty {
		t.Fatalf("Derive(%s) = %s, want %s — one measured candidate must not open the unmeasured ones", second, got, ThirdParty)
	}
}

// THE DECLARATION LIMB IS UNAFFECTED at every floor state. It gates nothing — a
// declared address is a subject from the declaration and returns from the first limb
// before the extension is asked — and its LABEL survives too: the errored floor decides
// one limb's reach and takes no row off the address-scope census.
func TestTheErroredLimbLeavesTheDeclarationLimbAlone(t *testing.T) {
	e := bothLimbs(EdgeFanout{
		Enabled:        true,
		BatchCompleted: true,
		Shared:         map[netip.Addr]bool{declaredEdge: true},
	})
	if got := e.Derive(declaredEdge); got != Operator {
		t.Fatalf("Derive(%s) = %s, want %s — a declared address is operator at any fan-out count", declaredEdge, got, Operator)
	}
	if !e.MayProbe(declaredEdge, ClassInternet) {
		t.Fatal("MayProbe = false on a declared address — the floor reached the wrong limb")
	}
	entries := e.AddressScopeCensus()
	if len(entries) != 1 || entries[0].SharedEdges != 1 {
		t.Fatalf("AddressScopeCensus = %+v, want one scope holding one shared edge — the floor took the row", entries)
	}
}

// An estate holding NO EXTENSION CANDIDATES is not errored. There is no limb to
// measure, so there is no reach to open and nothing the verdict could change — and the
// declaration limb's census must still render, which reads the Scan's disposition.
func TestAnEstateWithNoExtensionCandidatesIsNotErrored(t *testing.T) {
	e := Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("23.20.0.0/24")},
	}.WithEdgeFanout(EdgeFanout{
		Enabled:        true,
		BatchCompleted: true,
		Shared:         map[netip.Addr]bool{declaredEdge: true},
	})
	if e.edgeFanout.ExtensionErrored {
		t.Fatal("an estate with no extension candidates read as errored")
	}
	if got := e.AddressScopeCensus(); len(got) != 1 {
		t.Fatalf("AddressScopeCensus = %+v, want the one scope's row", got)
	}
}

// WithEdgeFanout is the ONE way a measurement enters an Estate, and it resolves the
// floor against the estate's own candidates. The zero Estate takes the zero
// measurement, which reaches — the pre-ADR-0129 behaviour an Estate assembled without
// this input must keep.
func TestWithEdgeFanoutResolvesTheFloorAndTheZeroValueReaches(t *testing.T) {
	errored := bothLimbs(EdgeFanout{Enabled: true, BatchCompleted: true, Shared: map[netip.Addr]bool{declaredEdge: true}})
	if !errored.edgeFanout.ExtensionErrored {
		t.Fatal("WithEdgeFanout left the floor unresolved on an errored extension limb")
	}
	// The same record carried in with the candidate set EMPTY — the shape a caller
	// producing an estate without its resolutions would get — is not errored, which is
	// why the field is unexported and this is the only setter.
	if (Estate{}).WithEdgeFanout(EdgeFanout{}).Derive(edge) != ThirdParty {
		t.Fatal("the zero estate covered an address")
	}
	if got := bothLimbs(EdgeFanout{}).Derive(edge); got != Operator {
		t.Fatalf("Derive(%s) on the zero measurement = %s, want %s — the pre-ADR-0129 behaviour", edge, got, Operator)
	}
}

// The resolution is TOTAL: it CLEARS an inbound verdict rather than only ever setting
// one. A record already resolved over one estate, carried into a second whose candidate
// IS measured, must read that second estate's answer — otherwise a stale errored
// reading opens a whole extension limb that nothing errored.
func TestWithEdgeFanoutClearsAnInboundErroredReading(t *testing.T) {
	stale := EdgeFanout{
		Enabled:          true,
		BatchCompleted:   true,
		ExtensionErrored: true,
		Shared:           map[netip.Addr]bool{edge: true},
	}
	e := bothLimbs(stale)
	if e.edgeFanout.ExtensionErrored {
		t.Fatal("a stale errored reading survived a resolution over a measured candidate")
	}
	if got := e.Derive(edge); got != ThirdParty {
		t.Fatalf("Derive(%s) = %s, want %s — the stale reading opened a measured shared edge", edge, got, ThirdParty)
	}
}
