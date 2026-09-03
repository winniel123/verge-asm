package custody

import (
	"net/netip"
	"testing"
)

func censusEstate(shared map[netip.Addr]bool, scopes ...netip.Prefix) Estate {
	// Real global unicast: a documentation range is barred before the census is asked (ADR-0079).
	return Estate{
		AddressScopes: scopes,
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "shop.example.com", Address: netip.MustParseAddr("93.184.216.10")},
			{Owner: "api.example.com", Address: netip.MustParseAddr("93.184.216.20")},
			{Owner: "www.example.com", Address: netip.MustParseAddr("93.184.216.30")},
		},
		edgeFanout: EdgeFanout{Enabled: true, Shared: shared},
	}
}

func TestExtensionCensusStates(t *testing.T) {
	e := censusEstate(map[netip.Addr]bool{
		netip.MustParseAddr("93.184.216.10"): true,
		netip.MustParseAddr("93.184.216.30"): false,
	})
	got := e.ExtensionCensus()
	if len(got) != 2 {
		t.Fatalf("census = %d rows, want 2 (one declined, one pending): %+v", len(got), got)
	}
	if got[0].Name != "shop.example.com" || got[0].State != ExtensionDeclined {
		t.Errorf("row 0 = %+v, want shop.example.com declined", got[0])
	}
	if got[1].Name != "api.example.com" || got[1].State != ExtensionPending {
		t.Errorf("row 1 = %+v, want api.example.com pending", got[1])
	}
	for _, r := range got {
		if r.Scope.IsValid() {
			t.Errorf("row %+v names an address scope, but none is declared", r)
		}
	}
}

func TestExtensionCensusDualLimbRow(t *testing.T) {
	scope := netip.MustParsePrefix("93.184.216.0/24")
	e := censusEstate(map[netip.Addr]bool{
		netip.MustParseAddr("93.184.216.10"): true,
		netip.MustParseAddr("93.184.216.20"): false,
		netip.MustParseAddr("93.184.216.30"): false,
	}, scope)
	got := e.ExtensionCensus()
	if len(got) != 1 {
		t.Fatalf("census = %d rows, want 1: %+v", len(got), got)
	}
	if got[0].State != ExtensionDeclined || got[0].Scope != scope {
		t.Fatalf("row = %+v, want declined and covered by %s", got[0], scope)
	}
	if got := e.Derive(netip.MustParseAddr("93.184.216.10")); got != Operator {
		t.Errorf("Derive on a dual-limb address = %q, want %q — the veto overruled a Declared act", got, Operator)
	}
}

func TestExtensionCensusEmptyWhereScanNotInForce(t *testing.T) {
	e := censusEstate(nil).WithEdgeFanout(EdgeFanout{})
	if got := e.ExtensionCensus(); len(got) != 0 {
		t.Errorf("census = %+v on a Scan out of force, want no rows", got)
	}
}

func TestExtensionCensusHoldsTheReachConditions(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "internal.example.com", Address: netip.MustParseAddr("10.0.0.5")},
			{Owner: "edge.foreign.test", Address: netip.MustParseAddr("93.184.216.40")},
		},
		edgeFanout: EdgeFanout{Enabled: true, Shared: map[netip.Addr]bool{
			netip.MustParseAddr("10.0.0.5"):      true,
			netip.MustParseAddr("93.184.216.40"): true,
		}},
	}
	if got := e.ExtensionCensus(); len(got) != 0 {
		t.Errorf("census = %+v, want no rows: neither address is one the extension reaches for", got)
	}
}

func TestExtensionCensusRowPerCitingName(t *testing.T) {
	edge := netip.MustParseAddr("93.184.216.10")
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "shop.example.com", Address: edge},
			{Owner: "www.example.com", Address: edge},
			{Owner: "shop.example.com", Address: edge},
		},
		edgeFanout: EdgeFanout{Enabled: true, Shared: map[netip.Addr]bool{edge: true}},
	}
	got := e.ExtensionCensus()
	if len(got) != 2 {
		t.Fatalf("census = %d rows, want 2 citing names: %+v", len(got), got)
	}
	if got[0].Name != "shop.example.com" || got[1].Name != "www.example.com" {
		t.Errorf("census = %+v, want the citing names in resolution order", got)
	}
}

func TestExtensionCensusAgreesWithCoveredByExtension(t *testing.T) {
	e := censusEstate(map[netip.Addr]bool{
		netip.MustParseAddr("93.184.216.10"): true,
		netip.MustParseAddr("93.184.216.30"): false,
	}, netip.MustParsePrefix("93.184.216.30/32"))
	listed := map[netip.Addr]bool{}
	for _, r := range e.ExtensionCensus() {
		listed[r.Address] = true
	}
	// Drift misinforms the operator: a decline that did not happen, or the silence #987 closes.
	for _, addr := range e.ExtensionCandidates() {
		reached := e.coveredByExtension(addr)
		if reached == listed[addr] {
			t.Errorf("candidate %s: coveredByExtension=%v, listed in census=%v — the two must be opposites",
				addr, reached, listed[addr])
		}
	}
}
