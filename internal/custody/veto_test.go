package custody

import (
	"net/netip"
	"testing"
)

// A documentation range is not globally reachable, so a fixture built from one would pass wrongly.

var edge = netip.MustParseAddr("104.16.132.229")

func extended(f EdgeFanout) Estate {
	return Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions:   []Resolution{{Owner: "www.example.com", Address: edge}},
		edgeFanout:    f,
	}
}

func measured(shared bool) EdgeFanout {
	return EdgeFanout{Enabled: true, Shared: map[netip.Addr]bool{edge: shared}}
}

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

func TestMayProbeRefusesAVetoedAddressFromEveryClass(t *testing.T) {
	// The gate takes no port or rate argument, so class is all a caller can vary (ADR-0019).
	e := extended(measured(true))
	for _, vc := range []VantageClass{ClassInternet, ClassInternal, ClassUnverified} {
		if e.MayProbe(edge, vc) {
			t.Fatalf("MayProbe(%s, %s) = true, want false — a vetoed edge queues no probe", edge, vc)
		}
	}
	cleared := extended(measured(false))
	for _, vc := range []VantageClass{ClassInternet, ClassInternal, ClassUnverified} {
		if !cleared.MayProbe(edge, vc) {
			t.Fatalf("MayProbe(%s, %s) = false on a cleared edge, want true", edge, vc)
		}
	}
}

func TestAVetoedEdgeIsIndistinguishableFromACNAMEToForeign(t *testing.T) {
	vetoed := extended(measured(true))
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

func TestASeedCoveredAddressIsOperatorAtAnyFanOutCount(t *testing.T) {
	e := extended(measured(true))
	e.AddressScopes = []netip.Prefix{netip.MustParsePrefix("104.16.132.0/24")}

	if got := e.Derive(edge); got != Operator {
		t.Fatalf("Derive(%s) = %s, want %s — a declared address scope covers it", edge, got, Operator)
	}
	if !e.MayProbe(edge, ClassInternet) {
		t.Fatal("MayProbe = false on a declared address — the veto reached the wrong limb")
	}
	wide := e
	wide.AddressScopes = []netip.Prefix{netip.MustParsePrefix("104.16.0.0/13")}
	if got := wide.Derive(edge); got != Operator {
		t.Fatalf("Derive(%s) inside a /13 = %s, want %s — the veto must not read prefix length", edge, got, Operator)
	}
	declaredOnly := Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("104.16.0.0/13")},
		edgeFanout:    measured(true),
	}
	if got := declaredOnly.Derive(edge); got != Operator {
		t.Fatalf("Derive(%s) on the Seed limb alone = %s, want %s", edge, got, Operator)
	}
}

func TestCoversAddressScopeIsUnchangedByTheVeto(t *testing.T) {
	// The Vantage-class derivation binds this predicate, so no extension may move it (ADR-0013 §6).
	declared := netip.MustParseAddr("104.16.132.10")
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
	if e.CoversAddressScope(cited) {
		t.Fatal("an extension-cited address entered the address-scope coverage")
	}
	cleared := e
	cleared.edgeFanout = EdgeFanout{Enabled: true, Shared: map[netip.Addr]bool{cited: false, declared: false}}
	if cleared.CoversAddressScope(cited) {
		t.Fatal("clearing the fan-out admitted an extension-cited address to the address-scope coverage")
	}
}

func TestAVetoedEdgeStaysACandidate(t *testing.T) {
	// A post-veto population would freeze the first shared result forever (#985).
	e := extended(measured(true))
	got := e.ExtensionCandidates()
	if len(got) != 1 || got[0] != edge {
		t.Fatalf("candidates = %v, want [%s] — a veto must not narrow the population it is measured from", got, edge)
	}
	if e.Derive(edge) != ThirdParty {
		t.Fatal("the fixture did not actually veto the edge")
	}
}

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

func TestAClearedMeasurementDoesNotOpenANonGloballyReachableAddress(t *testing.T) {
	// A measurement may narrow a reach and never widen one, so a clear does not lift ADR-0079's stop.
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

// An out-of-zone owner cites it, so it is a declaration-limb address and never a candidate.

var declaredEdge = netip.MustParseAddr("23.20.0.20")

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

func TestADeclarationLimbRowAloneDoesNotLiftTheFloor(t *testing.T) {
	// A whole-store floor would hold every extension candidate silently, and for good (#1018).
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
	if got := e.ExtensionCensus(); len(got) != 0 {
		t.Fatalf("ExtensionCensus = %+v on an errored limb, want none", got)
	}
}

func TestAScanWithNoCompletedBatchStillHoldsItsCandidates(t *testing.T) {
	// Holding here keeps the modal all-CDN install from showing appear-then-withdraw churn (ADR-0129).
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

func TestOneMeasuredCandidateLeavesTheRestHeld(t *testing.T) {
	// A lag is bounded by the daily cadence, so a partial failure is case 3, never the floor (#1018).
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

func TestWithEdgeFanoutResolvesTheFloorAndTheZeroValueReaches(t *testing.T) {
	errored := bothLimbs(EdgeFanout{Enabled: true, BatchCompleted: true, Shared: map[netip.Addr]bool{declaredEdge: true}})
	if !errored.edgeFanout.ExtensionErrored {
		t.Fatal("WithEdgeFanout left the floor unresolved on an errored extension limb")
	}
	if (Estate{}).WithEdgeFanout(EdgeFanout{}).Derive(edge) != ThirdParty {
		t.Fatal("the zero estate covered an address")
	}
	if got := bothLimbs(EdgeFanout{}).Derive(edge); got != Operator {
		t.Fatalf("Derive(%s) on the zero measurement = %s, want %s — the pre-ADR-0129 behaviour", edge, got, Operator)
	}
}

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
