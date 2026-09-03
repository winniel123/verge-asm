package custody

import (
	"net/netip"
	"slices"
	"testing"
)

// Collecting is safe on a small estate only: the sequence is lazy against an IPv4 /8 (ADR-0127).

func population(e Estate) []netip.Addr { return slices.Collect(e.EdgeFanoutPopulation()) }

func addrs(t *testing.T, got []netip.Addr, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("candidates = %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i].String() != w {
			t.Fatalf("candidates[%d] = %s, want %s (whole: %v)", i, got[i], w, got)
		}
	}
}

func TestExtensionCandidatesReadsTheExtendedZonesAlone(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "www.example.com", Address: netip.MustParseAddr("93.184.216.34")},
			{Owner: "edge.provider.net", Address: netip.MustParseAddr("104.16.132.229")},
			{Owner: "evilexample.com", Address: netip.MustParseAddr("23.20.0.10")},
		},
	}
	addrs(t, e.ExtensionCandidates(), "93.184.216.34")
}

func TestExtensionCandidatesIncludesTheZoneApex(t *testing.T) {
	// A provider flattens an apex ALIAS or ANAME into an A record, so the apex arrives on this limb.
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "example.com", Address: netip.MustParseAddr("104.16.132.229")},
		},
	}
	addrs(t, e.ExtensionCandidates(), "104.16.132.229")
}

func TestExtensionCandidatesAreDistinctInFirstSeenOrder(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "www.example.com", Address: netip.MustParseAddr("104.16.132.229")},
			{Owner: "api.example.com", Address: netip.MustParseAddr("104.16.132.230")},
			{Owner: "shop.example.com", Address: netip.MustParseAddr("104.16.132.229")},
		},
	}
	addrs(t, e.ExtensionCandidates(), "104.16.132.229", "104.16.132.230")
}

func TestExtensionCandidatesSkipsNonGloballyReachable(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "api.example.com", Address: netip.MustParseAddr("10.0.0.5")},
			{Owner: "vpn.example.com", Address: netip.MustParseAddr("127.0.0.1")},
			{Owner: "docs.example.com", Address: netip.MustParseAddr("203.0.113.10")},
			{Owner: "www.example.com", Address: netip.MustParseAddr("93.184.216.34")},
		},
	}
	addrs(t, e.ExtensionCandidates(), "93.184.216.34")
}

func TestExtensionCandidatesAgreeWithExtensionReaches(t *testing.T) {
	// A candidate outside the reach would be a probe of an address no extension claims (ADR-0129 §6).
	e := Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("104.16.132.0/24")},
		ExtendedZones: []string{"example.com", "example.org"},
		Resolutions: []Resolution{
			{Owner: "www.example.com", Address: netip.MustParseAddr("93.184.216.34")},
			{Owner: "api.example.org", Address: netip.MustParseAddr("104.16.132.10")},
			{Owner: "edge.provider.net", Address: netip.MustParseAddr("23.20.0.10")},
			{Owner: "int.example.com", Address: netip.MustParseAddr("10.0.0.5")},
		},
	}
	got := map[netip.Addr]bool{}
	for _, a := range e.ExtensionCandidates() {
		if !e.extensionReaches(a) {
			t.Fatalf("candidate %s is not covered by the extension", a)
		}
		got[a] = true
	}
	if len(got) == 0 {
		t.Fatal("no candidates at all — the fixture proves nothing")
	}
	for _, r := range e.Resolutions {
		if e.extensionReaches(r.Address) && !got[r.Address] {
			t.Fatalf("extension-covered address %s is not a candidate", r.Address)
		}
	}
}

func TestExtensionCensusAsksOnlyAboutExtensionCandidates(t *testing.T) {
	// The census and the candidates are written apart, so a session widening one is told here (#1036).
	e := Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("23.20.0.0/24")},
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "www.example.com", Address: netip.MustParseAddr("93.184.216.34")},
			{Owner: "cdn.example.com", Address: netip.MustParseAddr("104.16.132.10")},
			{Owner: "edge.provider.net", Address: netip.MustParseAddr("23.20.0.10")},
			{Owner: "int.example.com", Address: netip.MustParseAddr("10.0.0.5")},
		},
	}
	candidates := map[netip.Addr]bool{}
	for _, a := range e.ExtensionCandidates() {
		candidates[a] = true
	}
	if len(candidates) == 0 {
		t.Fatal("no candidates at all — the fixture proves nothing")
	}

	// A completed Batch would error the limb and the assertion below would pass over an empty set.
	e = e.WithEdgeFanout(EdgeFanout{Enabled: true})
	asked := map[netip.Addr]bool{}
	for _, entry := range e.ExtensionCensus() {
		if !candidates[entry.Address] {
			t.Fatalf("the census asked about %s, which is not an extension candidate: "+
				"a read bound to the candidate set would hold it (#1036)", entry.Address)
		}
		asked[entry.Address] = true
	}
	if len(asked) != len(candidates) {
		t.Fatalf("the census asked about %d addresses over %d candidates", len(asked), len(candidates))
	}
}

func TestExtensionCandidatesEmptyWithNoExtension(t *testing.T) {
	e := Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("104.16.132.0/24")},
		Resolutions: []Resolution{
			{Owner: "www.example.com", Address: netip.MustParseAddr("104.16.132.10")},
		},
	}
	if got := e.ExtensionCandidates(); got != nil {
		t.Fatalf("candidates = %v, want none", got)
	}
	if got := (Estate{}).ExtensionCandidates(); got != nil {
		t.Fatalf("empty estate candidates = %v, want none", got)
	}
}

func TestExtensionCandidatesTakeTheFirstOwnerThatExtends(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "edge.provider.net", Address: netip.MustParseAddr("104.16.132.229")},
			{Owner: "www.example.com", Address: netip.MustParseAddr("104.16.132.229")},
		},
	}
	addrs(t, e.ExtensionCandidates(), "104.16.132.229")
}

func TestExtensionCandidatesFoldMappedSpellings(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "www.example.com", Address: netip.MustParseAddr("93.184.216.34")},
			{Owner: "api.example.com", Address: netip.MustParseAddr("::ffff:93.184.216.34")},
		},
	}
	addrs(t, e.ExtensionCandidates(), "93.184.216.34")
}

func TestEdgeFanoutPopulationCarriesBothLimbs(t *testing.T) {
	e := Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("104.16.132.0/30")},
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "www.example.com", Address: netip.MustParseAddr("93.184.216.34")},
			{Owner: "edge.provider.net", Address: netip.MustParseAddr("23.20.0.10")},
		},
	}
	addrs(t, population(e),
		"93.184.216.34",
		"104.16.132.0", "104.16.132.1", "104.16.132.2", "104.16.132.3")
}

func TestEdgeFanoutPopulationNonEmptyWithNoExtension(t *testing.T) {
	e := Estate{AddressScopes: []netip.Prefix{netip.MustParsePrefix("104.16.132.0/31")}}
	addrs(t, population(e), "104.16.132.0", "104.16.132.1")
	if got := e.ExtensionCandidates(); got != nil {
		t.Fatalf("extension candidates = %v, want none — the declaration limb must not feed the veto's population", got)
	}
	if got := population(Estate{}); got != nil {
		t.Fatalf("empty estate population = %v, want none", got)
	}
}

func TestEdgeFanoutPopulationDedupsTheDualLimbAddress(t *testing.T) {
	dual := "104.16.132.1"
	e := Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("104.16.132.0/31")},
		ExtendedZones: []string{"example.com"},
		Resolutions:   []Resolution{{Owner: "www.example.com", Address: netip.MustParseAddr(dual)}},
	}
	addrs(t, population(e), dual, "104.16.132.0")
}

func TestEdgeFanoutPopulationDoesNotDedupOverlappingScopes(t *testing.T) {
	e := Estate{AddressScopes: []netip.Prefix{
		netip.MustParsePrefix("104.16.132.0/31"),
		netip.MustParsePrefix("104.16.132.0/30"),
	}}
	addrs(t, population(e),
		"104.16.132.0", "104.16.132.1",
		"104.16.132.0", "104.16.132.1", "104.16.132.2", "104.16.132.3")
}

func TestEdgeFanoutPopulationSkipsNonGloballyReachableDeclared(t *testing.T) {
	// The Scan carries no Vantage, so it can never satisfy ADR-0079's declared-realm clause.
	e := Estate{AddressScopes: []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/30"),
		netip.MustParsePrefix("104.16.132.0/31"),
	}}
	addrs(t, population(e), "104.16.132.0", "104.16.132.1")
}

func TestEdgeFanoutPopulationStreamsAndHoldsNoScope(t *testing.T) {
	e := Estate{AddressScopes: []netip.Prefix{netip.MustParsePrefix("104.0.0.0/8")}}
	var got []netip.Addr
	for a := range e.EdgeFanoutPopulation() {
		got = append(got, a)
		if len(got) == 3 {
			break
		}
	}
	addrs(t, got, "104.0.0.0", "104.0.0.1", "104.0.0.2")
}

func TestEdgeFanoutPopulationLabelsTheDeclarationLimbAndGatesNothing(t *testing.T) {
	declared := netip.MustParseAddr("104.16.132.1")
	e := Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("104.16.132.0/31")},
		edgeFanout:    EdgeFanout{Enabled: true, Shared: map[netip.Addr]bool{declared: true}},
	}
	if got := e.Derive(declared); got != Operator {
		t.Fatalf("Derive(%s) = %s, want %s — a measurement overruled a Declared act", declared, got, Operator)
	}
	if !e.MayProbe(declared, ClassInternet) {
		t.Fatalf("MayProbe(%s) = false — a shared measurement withheld a declared subject", declared)
	}
	found := false
	for a := range e.EdgeFanoutPopulation() {
		if a == declared {
			found = true
		}
	}
	if !found {
		t.Fatalf("%s is not in the population — the declaration limb was not measured", declared)
	}
	if got := e.ExtensionCensus(); got != nil {
		t.Fatalf("census = %v, want none — a declared-only address carries no row", got)
	}
}

func TestAnUnmeasuredDeclaredAddressIsProbedAndCarriesNoRow(t *testing.T) {
	declared := netip.MustParseAddr("104.16.132.1")
	e := Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("104.16.132.0/31")},
		edgeFanout:    EdgeFanout{Enabled: true, Shared: map[netip.Addr]bool{}},
	}
	if got := e.Derive(declared); got != Operator {
		t.Fatalf("Derive(%s) = %s, want %s — an unmeasured declared address was held", declared, got, Operator)
	}
	if !e.MayProbe(declared, ClassInternet) {
		t.Fatalf("MayProbe(%s) = false — an unmeasured declared address was held out of its own scope", declared)
	}
	if got := e.ExtensionCensus(); got != nil {
		t.Fatalf("census = %v, want none — a pending row on every declared address is noise, not a census", got)
	}
}
