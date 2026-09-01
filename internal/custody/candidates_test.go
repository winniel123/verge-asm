package custody

import (
	"net/netip"
	"slices"
	"testing"
)

// population collects the streamed EdgeFanoutPopulation for a whole-sequence assertion.
// Only a test that pins the ORDER AND CONTENTS of a small estate collects it; the
// sequence is lazy on purpose (ADR-0127) and the streaming test below never does.
func population(e Estate) []netip.Addr { return slices.Collect(e.EdgeFanoutPopulation()) }

// Every address here is globally reachable on purpose. The documentation ranges
// (192.0.2.0/24, 198.51.100.0/24, 203.0.113.0/24) are non-globally-reachable, so an
// extension covers none of them and a fixture built from one would pass for the wrong
// reason.
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

// The candidate set is the addresses the custody extension would reach — the direct-A
// targets of names that pass the label-suffix test. A resolution whose owner is outside
// every extended zone contributes nothing, exactly as it extends nothing.
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

// The apex ALIAS/ANAME flattened to A is a direct A record on the apex name, so it is a
// candidate on the same limb as any other in-zone A record — the zone apex passes the
// label-suffix test against itself.
func TestExtensionCandidatesIncludesTheZoneApex(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "example.com", Address: netip.MustParseAddr("104.16.132.229")},
		},
	}
	addrs(t, e.ExtensionCandidates(), "104.16.132.229")
}

// One handshake per address: two in-zone names flattening to the same edge is the modal
// case, and measuring that edge twice would tell us nothing the first handshake did not.
// The order is first-seen, so a fan-out is deterministic across ticks.
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

// A non-globally-reachable address is not a candidate: the extension does not cover one
// (ADR-0079), so there is no reach to narrow and nothing to measure. This is the same
// stopping condition extensionReaches applies, read from one place.
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

// Every candidate the set yields is one the extension actually covers, and every
// extension-covered resolution address is in the set. The two must not drift: the Scan
// exists to narrow that exact reach, and a candidate outside it would be a probe of an
// address no extension claims.
func TestExtensionCandidatesAgreeWithExtensionReaches(t *testing.T) {
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

// An instance with no custody extension has an empty candidate set. It is a legible
// state, not an error: the dispatcher enqueues no job and the Scan records an empty
// scope. A declared address scope is the OTHER limb of membership and contributes no
// candidate on this one (#988 widens the Scan onto it, where it labels only).
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

// An address a foreign owner holds is still a candidate when some other resolution
// holds it on an in-zone owner. The walk takes the first owner that extends and passes
// over the ones that do not — a shared edge cited by both an in-zone name and a foreign
// one is the very case this Scan exists to measure.
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

// An IPv4-mapped IPv6 spelling of an address already seen is the same address, so it
// yields no second candidate and no second handshake.
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

// #988: the Scan's population is BOTH limbs. The extension candidates come first, then
// every address a declared address scope covers, so one tick's fan-out matches the next.
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

// #983's *empty until a custody extension is declared* legibility is gone. An install
// holding address scopes and no extension dispatches a non-empty scope, which is the
// stated cost of widening the measurement onto the declaration limb.
func TestEdgeFanoutPopulationNonEmptyWithNoExtension(t *testing.T) {
	e := Estate{AddressScopes: []netip.Prefix{netip.MustParsePrefix("104.16.132.0/31")}}
	addrs(t, population(e), "104.16.132.0", "104.16.132.1")
	if got := e.ExtensionCandidates(); got != nil {
		t.Fatalf("extension candidates = %v, want none — the declaration limb must not feed the veto's population", got)
	}
	// An install with NEITHER limb still measures nothing, and that stays legible.
	if got := population(Estate{}); got != nil {
		t.Fatalf("empty estate population = %v, want none", got)
	}
}

// A dual-limb address — cited by an in-zone name and covered by a declared scope at once
// (#987) — is ONE handshake per tick, not two. It is emitted on the extension limb, which
// is where its measurement decides something.
func TestEdgeFanoutPopulationDedupsTheDualLimbAddress(t *testing.T) {
	dual := "104.16.132.1"
	e := Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("104.16.132.0/31")},
		ExtendedZones: []string{"example.com"},
		Resolutions:   []Resolution{{Owner: "www.example.com", Address: netip.MustParseAddr(dual)}},
	}
	addrs(t, population(e), dual, "104.16.132.0")
}

// Two declared scopes that OVERLAP are not deduped against each other, so the overlap is
// handshaked twice. That is ADR-0127's accepted cost, and the same one queue.candidateAddrs
// takes: deduping across scopes needs a map holding the whole scope, which is the memory
// ceiling the streamed fan-out exists to avoid. Each handshake is its own idempotent
// Batch, and the read takes the newest row per address.
func TestEdgeFanoutPopulationDoesNotDedupOverlappingScopes(t *testing.T) {
	e := Estate{AddressScopes: []netip.Prefix{
		netip.MustParsePrefix("104.16.132.0/31"),
		netip.MustParsePrefix("104.16.132.0/30"),
	}}
	addrs(t, population(e),
		"104.16.132.0", "104.16.132.1",
		"104.16.132.0", "104.16.132.1", "104.16.132.2", "104.16.132.3")
}

// A non-globally-reachable declared address is not in the population. The Scan carries no
// Vantage, so it can never satisfy ADR-0079's *from a Vantage that is not internet-class*,
// and the shared egress guard refuses the dial. Dispatching one would spend a job to
// record `unreachable` and label nothing.
func TestEdgeFanoutPopulationSkipsNonGloballyReachableDeclared(t *testing.T) {
	e := Estate{AddressScopes: []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/30"),
		netip.MustParsePrefix("104.16.132.0/31"),
	}}
	addrs(t, population(e), "104.16.132.0", "104.16.132.1")
}

// NOTHING BOUNDS THE POPULATION, so it must stream. ADR-0127 removed the ceiling above
// the operator's address cap and ADR-0047 refuses a scan-time aperture, so a declared
// scope can be an IPv4 `/8` — 16.7M addresses. A consumer takes what it needs and stops
// the walk; no record holds the whole scope.
//
// A regression to a materialized slice fails here by enumerating all 16.7M addresses
// before yielding the first one, which is the memory ceiling the streaming exists to
// avoid.
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

// The veto reads the extension candidates ALONE. A declared address measured as shared is
// reached at any fan-out count: Derive returns from the address-scope limb before the
// extension is asked, and the probing gate stays open. This is the #956 amendment, and
// getting it backwards would let a measurement overrule a Declared act.
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

// Open-then-label: an UNMEASURED declared address is probed normally and carries no row.
// Hold-then-open is the extension limb's rule and must not be carried across by analogy —
// it would hold a declared subject out of its own scope's census, and it would put a
// pending row on every address of every scope on the first day.
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
