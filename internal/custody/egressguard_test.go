package custody

import (
	"net/netip"
	"strings"
	"testing"
)

func TestEgressGuardRefusesNonGlobal(t *testing.T) {
	guard := EgressGuard("testleaf", Realm{})

	// 169.254.169.254 is the cloud metadata endpoint the guard exists to refuse (ADR-0079).
	refused := []string{
		"169.254.169.254:80",
		"127.0.0.1:80",
		"10.0.0.5:80",
		"192.168.1.1:80",
		"172.16.0.1:80",
		"[fd00::1]:80",
		"[::1]:80",
		"[::ffff:169.254.0.1]:80",
	}
	for _, addr := range refused {
		err := guard("tcp", addr, nil)
		if err == nil {
			t.Errorf("EgressGuard(%q) = nil, want refusal (non-globally-reachable)", addr)
			continue
		}
		if !strings.Contains(err.Error(), "testleaf") {
			t.Errorf("EgressGuard(%q) err = %q, want the label in the message", addr, err)
		}
	}

	allowed := []string{"8.8.8.8:80", "1.1.1.1:443", "[2001:4860:4860::8888]:443"}
	for _, addr := range allowed {
		if err := guard("tcp", addr, nil); err != nil {
			t.Errorf("EgressGuard(%q) = %v, want nil (globally reachable)", addr, err)
		}
	}

	if err := guard("tcp", "metadata.internal:80", nil); err == nil {
		t.Error("EgressGuard(hostname) = nil, want refusal (non-literal host)")
	}
}

func TestEgressGuardAdmitsOnlyTheDeclaredRealm(t *testing.T) {
	realm := Realm{}.With(netip.MustParsePrefix("10.0.0.0/24"))
	guard := EgressGuard("testleaf", realm)

	if err := guard("tcp", "10.0.0.5:80", nil); err != nil {
		t.Errorf("EgressGuard(10.0.0.5:80) = %v, want nil (inside the declared scope)", err)
	}

	// One declared scope exempts that scope alone, never private space at large (ADR-0225 §2).
	outside := []string{
		"10.0.1.5:80",
		"127.0.0.1:80",
		"169.254.169.254:80",
		"192.168.1.1:80",
		"[fd00::1]:80",
	}
	for _, addr := range outside {
		if err := guard("tcp", addr, nil); err == nil {
			t.Errorf("EgressGuard(%q) = nil, want refusal (outside the declared scope)", addr)
		}
	}
}

func TestRealmRoundTripsThroughTheWireRendering(t *testing.T) {
	realm := Realm{}.With(netip.MustParsePrefix("10.0.0.7/24")).With(netip.MustParsePrefix("192.168.4.0/22"))
	cidrs := realm.CIDRs()
	if len(cidrs) != 2 {
		t.Fatalf("CIDRs() = %v, want two entries", cidrs)
	}
	back := ParseRealm(cidrs)
	for _, s := range []string{"10.0.0.7", "10.0.0.255", "192.168.4.1", "192.168.7.9"} {
		if !back.Contains(netip.MustParseAddr(s)) {
			t.Errorf("ParseRealm(%v).Contains(%s) = false, want true", cidrs, s)
		}
	}
	for _, s := range []string{"10.0.1.1", "192.168.8.1", "127.0.0.1"} {
		if back.Contains(netip.MustParseAddr(s)) {
			t.Errorf("ParseRealm(%v).Contains(%s) = true, want false", cidrs, s)
		}
	}
}

func TestParseRealmDropsAMalformedEntry(t *testing.T) {
	realm := ParseRealm([]string{"not-a-cidr", "10.0.0.0/24", "10.0.0.0/99", ""})
	if !realm.Contains(netip.MustParseAddr("10.0.0.5")) {
		t.Error("ParseRealm dropped the well-formed entry beside a malformed one")
	}
	if got := len(realm.CIDRs()); got != 1 {
		t.Errorf("ParseRealm kept %d entries, want 1", got)
	}
}

func TestZeroRealmContainsNothing(t *testing.T) {
	var realm Realm
	for _, s := range []string{"10.0.0.5", "127.0.0.1", "8.8.8.8"} {
		if realm.Contains(netip.MustParseAddr(s)) {
			t.Errorf("Realm{}.Contains(%s) = true, want false", s)
		}
	}
	if realm.CIDRs() != nil {
		t.Errorf("Realm{}.CIDRs() = %v, want nil", realm.CIDRs())
	}
}
