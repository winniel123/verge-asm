package custody

import (
	"net/netip"
	"testing"
)

func sharedFanout(shared []string, notShared []string) map[netip.Addr]bool {
	m := make(map[netip.Addr]bool, len(shared)+len(notShared))
	for _, s := range shared {
		m[netip.MustParseAddr(s).Unmap()] = true
	}
	for _, s := range notShared {
		m[netip.MustParseAddr(s).Unmap()] = false
	}
	return m
}

// A declared scope holding measured shared edges carries a row, and the row counts
// the shared addresses inside that scope alone.
func TestAddressScopeCensusCountsSharedEdgesInScope(t *testing.T) {
	e := Estate{
		AddressScopes: []netip.Prefix{cidr("93.184.216.0/24"), cidr("93.184.217.0/24")},
		edgeFanout: EdgeFanout{
			Enabled: true,
			Shared: sharedFanout(
				[]string{"93.184.216.7", "93.184.216.9", "93.184.217.4"},
				[]string{"93.184.216.8"},
			),
		},
	}
	got := e.AddressScopeCensus()
	if len(got) != 2 {
		t.Fatalf("rows = %d, want 2: %+v", len(got), got)
	}
	if got[0].Scope.String() != "93.184.216.0/24" || got[0].SharedEdges != 2 {
		t.Errorf("row 0 = %v/%d, want 93.184.216.0/24 and 2", got[0].Scope, got[0].SharedEdges)
	}
	if got[1].Scope.String() != "93.184.217.0/24" || got[1].SharedEdges != 1 {
		t.Errorf("row 1 = %v/%d, want 93.184.217.0/24 and 1", got[1].Scope, got[1].SharedEdges)
	}
}

// The acceptance criterion: a scope with no address above the threshold renders no
// row. A measured not-shared address is measured, and it is not a shared edge.
func TestAddressScopeCensusNoRowWithoutASharedEdge(t *testing.T) {
	e := Estate{
		AddressScopes: []netip.Prefix{cidr("93.184.216.0/24")},
		edgeFanout: EdgeFanout{
			Enabled: true,
			Shared:  sharedFanout(nil, []string{"93.184.216.8", "93.184.216.9"}),
		},
	}
	if got := e.AddressScopeCensus(); len(got) != 0 {
		t.Fatalf("rows = %+v, want none", got)
	}
}

// Open-then-label (ADR-0129's #956 amendment): an UNMEASURED declared address is
// probed normally and carries no row. Hold-then-open carried across by analogy would
// put a pending row on every address of every scope on the first day.
func TestAddressScopeCensusUnmeasuredCarriesNoRow(t *testing.T) {
	e := Estate{
		AddressScopes: []netip.Prefix{cidr("93.184.216.0/24")},
		edgeFanout:    EdgeFanout{Enabled: true, Shared: map[netip.Addr]bool{}},
	}
	if got := e.AddressScopeCensus(); len(got) != 0 {
		t.Fatalf("rows = %+v, want none", got)
	}
}

// A `Scan` out of force yields no row at all — EdgeFanout's fourth absence case, the
// pre-ADR-0129 behaviour. A row there would name evidence the Scan does not hold.
func TestAddressScopeCensusScanOutOfForceYieldsNoRow(t *testing.T) {
	e := Estate{
		AddressScopes: []netip.Prefix{cidr("93.184.216.0/24")},
		edgeFanout: EdgeFanout{
			Enabled: false,
			Shared:  sharedFanout([]string{"93.184.216.7"}, nil),
		},
	}
	if got := e.AddressScopeCensus(); len(got) != 0 {
		t.Fatalf("rows = %+v, want none", got)
	}
}

// A shared edge outside every declared scope belongs to the extension limb's census
// (#987) and never to a scope's own. The row counts containment, never the store.
func TestAddressScopeCensusIgnoresSharedEdgesOutsideEveryScope(t *testing.T) {
	e := Estate{
		AddressScopes: []netip.Prefix{cidr("93.184.216.0/24")},
		edgeFanout: EdgeFanout{
			Enabled: true,
			Shared:  sharedFanout([]string{"93.184.218.4"}, nil),
		},
	}
	if got := e.AddressScopeCensus(); len(got) != 0 {
		t.Fatalf("rows = %+v, want none", got)
	}
}

// Two overlapping scopes both cover the address, and both rows count it. That is
// coveringAddressScope's refused specificity test holding on the display: the remedy
// is per scope, so the operator must see the edge on each scope they could exclude
// it from.
func TestAddressScopeCensusCountsAnOverlapOnBothScopes(t *testing.T) {
	e := Estate{
		AddressScopes: []netip.Prefix{cidr("93.184.216.0/24"), cidr("93.184.216.0/25")},
		edgeFanout: EdgeFanout{
			Enabled: true,
			Shared:  sharedFanout([]string{"93.184.216.7"}, nil),
		},
	}
	got := e.AddressScopeCensus()
	if len(got) != 2 {
		t.Fatalf("rows = %d, want 2: %+v", len(got), got)
	}
	for i, row := range got {
		if row.SharedEdges != 1 {
			t.Errorf("row %d shared = %d, want 1", i, row.SharedEdges)
		}
	}
}

// The same scope declared twice is ONE row. Two identical rows would render the same
// fact twice and read as two scopes.
func TestAddressScopeCensusCollapsesADuplicateScope(t *testing.T) {
	p := cidr("93.184.216.0/24")
	e := Estate{
		AddressScopes: []netip.Prefix{p, p},
		edgeFanout: EdgeFanout{
			Enabled: true,
			Shared:  sharedFanout([]string{"93.184.216.7"}, nil),
		},
	}
	if got := e.AddressScopeCensus(); len(got) != 1 {
		t.Fatalf("rows = %+v, want one", got)
	}
}

// The census is DISPLAY and never a gate. A `Seed`-covered address whose measurement
// says shared is still reached, still derives operator, and is still probed — the
// #956 disjointness, asserted beside the row that reports it so a session repairing
// the apparent inconsistency fails here first.
func TestAddressScopeCensusRowGatesNothing(t *testing.T) {
	a := netip.MustParseAddr("93.184.216.7")
	e := Estate{
		AddressScopes: []netip.Prefix{cidr("93.184.216.0/24")},
		edgeFanout: EdgeFanout{
			Enabled: true,
			Shared:  sharedFanout([]string{"93.184.216.7"}, nil),
		},
	}
	if len(e.AddressScopeCensus()) != 1 {
		t.Fatal("want a row for the declared shared edge")
	}
	if got := e.Derive(a); got != Operator {
		t.Errorf("Derive = %q, want %q", got, Operator)
	}
	if !e.MayProbe(a, ClassInternet) {
		t.Error("MayProbe = false, want true: the row labels and never gates")
	}
}

// The extension limb's census is untouched by a declaration-limb row. The two
// registers stay on separate surfaces (the #944 amendment), so an address-scope
// address must not appear as an extension member.
func TestAddressScopeCensusLeavesTheExtensionCensusAlone(t *testing.T) {
	e := Estate{
		AddressScopes: []netip.Prefix{cidr("93.184.216.0/24")},
		edgeFanout: EdgeFanout{
			Enabled: true,
			Shared:  sharedFanout([]string{"93.184.216.7"}, nil),
		},
	}
	if got := e.ExtensionCensus(); len(got) != 0 {
		t.Fatalf("extension census = %+v, want none: no in-zone name cites the edge", got)
	}
	if len(e.AddressScopeCensus()) != 1 {
		t.Fatal("want the declaration-limb row")
	}
}

// The row names an exclusion as its remedy, and since #1022 (ADR-0133 §7) taking that
// remedy CLEARS the row. The measurement is filtered on READ: the stored observation
// is untouched, so the count returns the moment the exclusion is withdrawn.
func TestAddressScopeCensusClearsOnAnAddressExclusion(t *testing.T) {
	fanout := EdgeFanout{
		Enabled: true,
		Shared:  sharedFanout([]string{"93.184.216.7"}, nil),
	}
	base := Estate{
		AddressScopes: []netip.Prefix{cidr("93.184.216.0/24")},
		edgeFanout:    fanout,
	}
	if len(base.AddressScopeCensus()) != 1 {
		t.Fatal("want the declaration-limb row before the remedy is taken")
	}

	excluded := base.WithAddressExclusions([]netip.Prefix{cidr("93.184.216.0/29")})
	if got := excluded.AddressScopeCensus(); len(got) != 0 {
		t.Errorf("census = %+v, want none: the row's own remedy must clear the row", got)
	}
	if got := len(excluded.edgeFanout.Shared); got != len(fanout.Shared) {
		t.Errorf("the stored measurement holds %d rows, want %d: the census filtered on READ and deletes nothing", got, len(fanout.Shared))
	}

	// The exclusion narrows the count and nothing else. An address the exclusion
	// does NOT cover keeps its row on the same scope.
	outside := base.WithAddressExclusions([]netip.Prefix{cidr("93.184.216.128/25")})
	if got := outside.AddressScopeCensus(); len(got) != 1 {
		t.Errorf("census = %+v, want the row: an exclusion elsewhere in the scope must not clear it", got)
	}
}

// The census REFUSES a partial measurement (#1036). It is the one reader that walks
// Shared wholesale, so a record bound to the extension limb would show it every
// declaration-limb row missing and it would count short with nothing to say so.
//
// The two estates below differ in the Partial flag and in nothing else, so the refusal
// cannot be read off the fixture: the same map counts a row when the record covers the
// whole population.
func TestAddressScopeCensusRefusesAPartialMeasurement(t *testing.T) {
	measured := sharedFanout([]string{"93.184.216.7", "93.184.216.9"}, nil)
	scopes := []netip.Prefix{cidr("93.184.216.0/24")}

	whole := Estate{
		AddressScopes: scopes,
		edgeFanout:    EdgeFanout{Enabled: true, Shared: measured},
	}
	if got := whole.AddressScopeCensus(); len(got) != 1 || got[0].SharedEdges != 2 {
		t.Fatalf("census = %+v over the whole population, want one row of 2: the fixture proves nothing", got)
	}

	partial := Estate{
		AddressScopes: scopes,
		edgeFanout:    EdgeFanout{Enabled: true, Partial: true, Shared: measured},
	}
	if got := partial.AddressScopeCensus(); got != nil {
		t.Errorf("census = %+v over a partial measurement, want no entry — a short count "+
			"would state a number this install did not measure (#1036)", got)
	}
}

// Partial moves NO GATE and NO VERDICT. The veto asks its question per address, over a
// candidate the bound named, so the answer cannot turn on how the map was read.
func TestPartialMovesNoVerdict(t *testing.T) {
	shared := netip.MustParseAddr("104.16.132.10")
	dedicated := netip.MustParseAddr("93.184.216.34")
	measured := map[netip.Addr]bool{shared: true, dedicated: false}

	for _, partial := range []bool{false, true} {
		e := Estate{
			ExtendedZones: []string{"example.com"},
			Resolutions: []Resolution{
				{Owner: "cdn.example.com", Address: shared},
				{Owner: "www.example.com", Address: dedicated},
			},
		}.WithEdgeFanout(EdgeFanout{Enabled: true, BatchCompleted: true, Partial: partial, Shared: measured})

		if got := e.Derive(shared); got != ThirdParty {
			t.Errorf("Partial=%v: Derive(%s) = %s, want %s", partial, shared, got, ThirdParty)
		}
		if got := e.Derive(dedicated); got != Operator {
			t.Errorf("Partial=%v: Derive(%s) = %s, want %s", partial, dedicated, got, Operator)
		}
		if len(e.ExtensionCensus()) != 1 {
			t.Errorf("Partial=%v: the extension census named %d entries, want the one decline",
				partial, len(e.ExtensionCensus()))
		}
	}
}
