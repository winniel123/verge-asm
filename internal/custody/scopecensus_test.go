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

func TestAddressScopeCensusUnmeasuredCarriesNoRow(t *testing.T) {
	e := Estate{
		AddressScopes: []netip.Prefix{cidr("93.184.216.0/24")},
		edgeFanout:    EdgeFanout{Enabled: true, Shared: map[netip.Addr]bool{}},
	}
	if got := e.AddressScopeCensus(); len(got) != 0 {
		t.Fatalf("rows = %+v, want none", got)
	}
}

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

func TestAddressScopeCensusRowGatesNothing(t *testing.T) {
	// A session repairing the apparent inconsistency fails here first (ADR-0129's #956 amendment).
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

	outside := base.WithAddressExclusions([]netip.Prefix{cidr("93.184.216.128/25")})
	if got := outside.AddressScopeCensus(); len(got) != 1 {
		t.Errorf("census = %+v, want the row: an exclusion elsewhere in the scope must not clear it", got)
	}
}

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

func TestPartialMovesNoVerdict(t *testing.T) {
	// The veto asks per address over a named candidate, so a bound cannot move the answer (#1036).
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
