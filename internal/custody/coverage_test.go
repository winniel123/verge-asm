package custody

import (
	"net/netip"
	"testing"
)

func TestCoversAddressScopeFamilyMatched(t *testing.T) {
	e := Estate{AddressScopes: []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("2001:db8::/32"),
	}}
	cases := []struct {
		addr string
		want bool
	}{
		{"10.1.2.3", true},
		{"11.0.0.1", false},
		{"2001:db8::1", true},
		{"2001:dead::1", false},
		{"::ffff:10.1.2.3", true},
	}
	for _, c := range cases {
		if got := e.CoversAddressScope(netip.MustParseAddr(c.addr)); got != c.want {
			t.Errorf("CoversAddressScope(%s) = %v, want %v", c.addr, got, c.want)
		}
	}
}

func TestCoversAddressScopeRefusesExtension(t *testing.T) {
	globallyReachable := netip.MustParseAddr("93.184.216.34")
	e := Estate{
		AddressScopes: nil,
		ExtendedZones: []string{"example.com"},
		Resolutions:   []Resolution{{Owner: "api.example.com", Address: globallyReachable}},
	}

	if e.Derive(globallyReachable) != Operator {
		t.Fatal("precondition: the extension should derive Operator for probing")
	}
	if e.CoversAddressScope(globallyReachable) {
		t.Error("CoversAddressScope admitted an extension-covered address; it must use address scopes ALONE")
	}
}
