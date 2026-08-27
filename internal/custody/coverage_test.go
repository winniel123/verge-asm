package custody

import (
	"net/netip"
	"testing"
)

// CoversAddressScope is the family-matched prefix containment the Vantage-class
// derivation binds (#711): an address inside a declared address scope is covered, one
// outside is not, and containment never crosses address families.
func TestCoversAddressScopeFamilyMatched(t *testing.T) {
	e := Estate{AddressScopes: []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("2001:db8::/32"),
	}}
	cases := []struct {
		addr string
		want bool
	}{
		{"10.1.2.3", true},                 // inside the v4 scope
		{"11.0.0.1", false},                // outside every scope
		{"2001:db8::1", true},              // inside the v6 scope
		{"2001:dead::1", false},            // v6 outside the scope
		{"::ffff:10.1.2.3", true},          // v4-mapped v6 unmaps and matches the v4 scope
	}
	for _, c := range cases {
		if got := e.CoversAddressScope(netip.MustParseAddr(c.addr)); got != c.want {
			t.Errorf("CoversAddressScope(%s) = %v, want %v", c.addr, got, c.want)
		}
	}
}

// GUARDRAIL (#711): CoversAddressScope routes through address scopes ALONE. An address
// covered only by a custody EXTENSION (a resolution inside a declared, extended zone)
// is Operator under Derive/MayProbe, but must NOT read as covered here — admitting it
// would let an extension decide a vantage's side of the boundary and corrupt the class.
func TestCoversAddressScopeRefusesExtension(t *testing.T) {
	globallyReachable := netip.MustParseAddr("93.184.216.34")
	e := Estate{
		// No address scope covers the address...
		AddressScopes: nil,
		// ...but a custody extension does, via a resolution inside the extended zone.
		ExtendedZones: []string{"example.com"},
		Resolutions:   []Resolution{{Owner: "api.example.com", Address: globallyReachable}},
	}

	// Sanity: the extension really does make the address Operator for probing.
	if e.Derive(globallyReachable) != Operator {
		t.Fatal("precondition: the extension should derive Operator for probing")
	}
	// But the class coverage predicate must refuse it — address scopes only.
	if e.CoversAddressScope(globallyReachable) {
		t.Error("CoversAddressScope admitted an extension-covered address; it must use address scopes ALONE")
	}
}
