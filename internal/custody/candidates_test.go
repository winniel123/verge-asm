package custody

import (
	"net/netip"
	"testing"
)

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
