package custody

import (
	"net/netip"
	"testing"
)

// sharedFanout builds the measured store an EdgeFanout carries: first the addresses
// fan-out measured as SHARED, then the ones it measured and found not shared. An
// address named in neither is unmeasured, which is the open-then-label case.
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
