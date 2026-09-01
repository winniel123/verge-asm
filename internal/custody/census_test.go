package custody

import (
	"net/netip"
	"testing"
)

// censusEstate is a custody-extended estate over one zone, with the fan-out Scan in
// force and the given measurements. Every address is ordinary global unicast, because
// a documentation range is non-globally-reachable and ADR-0079 shuts the gate on it
// before the census is ever asked.
func censusEstate(shared map[netip.Addr]bool, scopes ...netip.Prefix) Estate {
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

// The census carries the two states the extension's non-reach splits into, and NOTHING
// for an edge it reaches. A reached edge is an ordinary covered address that holds
// `Custody` and queues probes; naming it here would make the section a coverage list
// rather than the decline register ADR-0129 §5 asks for.
func TestExtensionCensusStates(t *testing.T) {
	e := censusEstate(map[netip.Addr]bool{
		netip.MustParseAddr("93.184.216.10"): true,  // measured shared — declined
		netip.MustParseAddr("93.184.216.30"): false, // measured clear — reached, no row
		// 93.184.216.20 carries no key at all — measurement pending.
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

// The dual-limb row (ADR-0129's #956 amendment): an address the extension declined and
// an address-scope `Seed` covers at once. The row must state BOTH limbs. A bare
// *declined* is true about the extension and reads as a contradiction to the person the
// census exists for, and dropping the row would hide a decline they need if they later
// withdraw the `Seed`.
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
	// The two mechanisms are disjoint rather than ranked: the declaration still wins
	// the derivation outright, at any fan-out count. The row states the decline; it
	// does not withdraw the coverage.
	if got := e.Derive(netip.MustParseAddr("93.184.216.10")); got != Operator {
		t.Errorf("Derive on a dual-limb address = %q, want %q — the veto overruled a Declared act", got, Operator)
	}
}

// A Scan that is not IN FORCE yields no row at all. Nothing is declined and nothing is
// held where the measurement does not narrow — that is EdgeFanout's fourth absence
// case, the pre-ADR-0129 reach-everything behaviour — so a row here would name a
// decline that did not happen. The census must never fabricate one.
func TestExtensionCensusEmptyWhereScanNotInForce(t *testing.T) {
	e := censusEstate(nil).WithEdgeFanout(EdgeFanout{})
	if got := e.ExtensionCensus(); len(got) != 0 {
		t.Errorf("census = %+v on a Scan out of force, want no rows", got)
	}
}

// The census reaches exactly what the extension REACHES FOR, so it holds the same two
// stopping conditions extensionReaches holds: a non-globally-reachable address extends
// nothing (ADR-0079), and transitivity stops where the chain leaves the declared zone.
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

// Two in-zone names citing ONE shared edge are two rows: each names its own citing
// name, which is the one thing the operator can act on. The same name citing the same
// edge twice is one row.
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

// The census and the reach must not drift apart. A row naming an address the extension
// DOES reach would tell the operator a probe was withheld that was not, and a
// candidate the extension does not reach with no row is the silence #987 exists to
// close. So over a mixed estate, the census is exactly the set of candidate addresses
// coveredByExtension refuses.
func TestExtensionCensusAgreesWithCoveredByExtension(t *testing.T) {
	e := censusEstate(map[netip.Addr]bool{
		netip.MustParseAddr("93.184.216.10"): true,
		netip.MustParseAddr("93.184.216.30"): false,
	}, netip.MustParsePrefix("93.184.216.30/32"))
	listed := map[netip.Addr]bool{}
	for _, r := range e.ExtensionCensus() {
		listed[r.Address] = true
	}
	for _, addr := range e.ExtensionCandidates() {
		reached := e.coveredByExtension(addr)
		if reached == listed[addr] {
			t.Errorf("candidate %s: coveredByExtension=%v, listed in census=%v — the two must be opposites",
				addr, reached, listed[addr])
		}
	}
}
