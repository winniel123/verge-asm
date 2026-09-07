package custody

import (
	"net/netip"
	"testing"
)

func TestProbeRealmReturnsTheDeclaredScopeThatAdmitted(t *testing.T) {
	e := Estate{AddressScopes: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/24")}}
	addr := netip.MustParseAddr("10.0.0.5")

	for _, vc := range []VantageClass{ClassUnverified, ClassInternal} {
		scope, ok := e.ProbeRealm(addr, vc)
		if !ok {
			t.Fatalf("ProbeRealm(10.0.0.5, %s) = _, false, want true (ADR-0079 worked table)", vc)
		}
		if scope != netip.MustParsePrefix("10.0.0.0/24") {
			t.Errorf("ProbeRealm(10.0.0.5, %s) scope = %v, want 10.0.0.0/24", vc, scope)
		}
		if !guardAdmits(t, Realm{}.With(scope), "10.0.0.5:80") {
			t.Errorf("the guard refused an address ProbeRealm admitted under %s", vc)
		}
	}
}

func TestProbeRealmYieldsNoRealmForAnInternetClassVantage(t *testing.T) {
	e := Estate{AddressScopes: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/24")}}
	scope, ok := e.ProbeRealm(netip.MustParseAddr("10.0.0.5"), ClassInternet)
	if ok {
		t.Fatal("ProbeRealm(10.0.0.5, internet) = _, true, want false (ADR-0079)")
	}
	// A refused address must yield no scope, so no realm can be built out of one (ADR-0225 §1).
	if scope.IsValid() {
		t.Errorf("ProbeRealm returned scope %v beside a refusal, want the zero Prefix", scope)
	}
}

func TestProbeRealmYieldsNoRealmForAGloballyReachableAddress(t *testing.T) {
	e := Estate{AddressScopes: []netip.Prefix{netip.MustParsePrefix("93.184.216.0/24")}}
	scope, ok := e.ProbeRealm(netip.MustParseAddr("8.8.8.8"), ClassUnverified)
	if ok {
		t.Fatal("ProbeRealm(8.8.8.8, unverified) = _, true, want false (third-party)")
	}
	if scope.IsValid() {
		t.Errorf("ProbeRealm returned scope %v beside a refusal, want the zero Prefix", scope)
	}
}

// The test that stops a later session widening this into an SSRF (#1610, ADR-0225 §2).

func TestADiscoveredPrivateAddressGetsNoRealmAndStaysRefused(t *testing.T) {
	// A hostile in-scope zone publishes an A record into private space the operator never declared.
	discovered := netip.MustParseAddr("192.168.1.7")
	metadata := netip.MustParseAddr("169.254.169.254")
	loopback := netip.MustParseAddr("127.0.0.1")

	e := Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/24")},
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "api.example.com", Address: discovered},
			{Owner: "meta.example.com", Address: metadata},
			{Owner: "self.example.com", Address: loopback},
		},
	}

	for _, addr := range []netip.Addr{discovered, metadata, loopback} {
		if got := e.Derive(addr); got != ThirdParty {
			t.Errorf("Derive(%s) = %s, want third-party (an extension declares no realm)", addr, got)
		}
		for _, vc := range []VantageClass{ClassUnverified, ClassInternal, ClassInternet} {
			scope, ok := e.ProbeRealm(addr, vc)
			if ok {
				t.Errorf("ProbeRealm(%s, %s) = _, true, want false", addr, vc)
			}
			if scope.IsValid() {
				t.Errorf("ProbeRealm(%s, %s) yielded scope %v, want the zero Prefix", addr, vc, scope)
			}
		}
	}

	// The realm a legitimate job carries never reaches an address the operator did not declare.
	realm := Realm{}.With(netip.MustParsePrefix("10.0.0.0/24"))
	for _, target := range []string{"192.168.1.7:80", "169.254.169.254:80", "127.0.0.1:80"} {
		if guardAdmits(t, realm, target) {
			t.Errorf("EgressGuard admitted discovered target %s, want refusal", target)
		}
	}
}

func guardAdmits(t *testing.T, realm Realm, target string) bool {
	t.Helper()
	return EgressGuard("testleaf", realm)("tcp", target, nil) == nil
}
