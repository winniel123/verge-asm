package custody

import (
	"net/netip"
	"testing"
)

func addr(s string) netip.Addr { return netip.MustParseAddr(s) }
func cidr(s string) netip.Prefix {
	return netip.MustParsePrefix(s).Masked()
}

// TestDeriveFromAddressScope: every address inside a declared address scope is
// operator directly, from the declaration, family-matched.
func TestDeriveFromAddressScope(t *testing.T) {
	e := Estate{AddressScopes: []netip.Prefix{cidr("52.1.2.0/24"), cidr("2001:db8:1::/48")}}

	operator := []string{"52.1.2.1", "52.1.2.254", "2001:db8:1::5"}
	for _, a := range operator {
		if got := e.Derive(addr(a)); got != Operator {
			t.Errorf("Derive(%s) = %q, want operator", a, got)
		}
	}
	// Outside the scope, and the other family, are third-party — nothing else
	// covers them.
	thirdParty := []string{"52.1.3.1", "203.0.113.9", "2001:db8:2::1"}
	for _, a := range thirdParty {
		if got := e.Derive(addr(a)); got != ThirdParty {
			t.Errorf("Derive(%s) = %q, want third-party", a, got)
		}
	}
}

// TestDeriveFromExtensionInZone: a direct A record on a name within a
// custody-extended zone extends custody to a globally-reachable address; a CNAME
// to a foreign name does not, because the A record's owner is then outside every
// extended scope (ADR-0013 §3).
func TestDeriveFromExtensionInZone(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			// api.example.com A 52.1.2.3 — direct A inside the declared zone.
			{Owner: "api.example.com", Address: addr("52.1.2.3")},
			// The CNAME target of shop.example.com is a foreign name that holds
			// the A record — outside every name scope, so it does not extend.
			{Owner: "d1x2y3.cloudfront.net", Address: addr("13.32.1.1")},
		},
	}

	if got := e.Derive(addr("52.1.2.3")); got != Operator {
		t.Errorf("Derive(52.1.2.3) = %q, want operator (direct A in extended zone)", got)
	}
	if got := e.Derive(addr("13.32.1.1")); got != ThirdParty {
		t.Errorf("Derive(13.32.1.1) = %q, want third-party (CNAME left the zone)", got)
	}
}

// TestExtensionRequiresTheExtensionFlag: a name scope without the custody
// extension confers nothing — its resolved addresses stay third-party. The
// extension is off by default and is the only thing that opens the name-scope
// route.
func TestExtensionRequiresTheExtensionFlag(t *testing.T) {
	e := Estate{
		// example.com is a declared name scope but NOT in ExtendedZones.
		Resolutions: []Resolution{{Owner: "api.example.com", Address: addr("52.1.2.3")}},
	}
	if got := e.Derive(addr("52.1.2.3")); got != ThirdParty {
		t.Errorf("Derive(52.1.2.3) = %q, want third-party (extension off)", got)
	}
}

// TestExtensionStopsAtNonGloballyReachableHop: an extension does not cover a
// non-globally-reachable address even by a direct A record inside the zone
// (ADR-0079's amendment to ADR-0013 §3). The same address covered by a declared
// address scope IS operator — the address-scope route is untouched.
func TestExtensionStopsAtNonGloballyReachableHop(t *testing.T) {
	ext := Estate{
		ExtendedZones: []string{"example.com"},
		// api.example.com A 10.0.0.5 — direct A, inside the zone, but private.
		Resolutions: []Resolution{{Owner: "api.example.com", Address: addr("10.0.0.5")}},
	}
	if got := ext.Derive(addr("10.0.0.5")); got != ThirdParty {
		t.Errorf("Derive(10.0.0.5) via extension = %q, want third-party (NGR hop)", got)
	}

	// Route 2: a declared address scope over the same private space is operator.
	scoped := Estate{AddressScopes: []netip.Prefix{cidr("10.0.0.0/24")}}
	if got := scoped.Derive(addr("10.0.0.5")); got != Operator {
		t.Errorf("Derive(10.0.0.5) via address scope = %q, want operator", got)
	}
}

// TestLabelSuffixIsNotStringSuffix: containment is label-wise, so evilexample.com
// does not read as inside example.com, and the zone covers its own apex.
func TestLabelSuffixIsNotStringSuffix(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "example.com", Address: addr("52.0.0.1")},           // apex — covered
			{Owner: "deep.sub.example.com", Address: addr("52.0.0.2")},  // subtree — covered
			{Owner: "evilexample.com", Address: addr("52.0.0.3")},       // string suffix, not label suffix
			{Owner: "example.com.attacker.net", Address: addr("52.0.0.4")}, // prefix, not suffix
		},
	}
	covered := map[string]Custody{
		"52.0.0.1": Operator,
		"52.0.0.2": Operator,
		"52.0.0.3": ThirdParty,
		"52.0.0.4": ThirdParty,
	}
	for a, want := range covered {
		if got := e.Derive(addr(a)); got != want {
			t.Errorf("Derive(%s) = %q, want %q", a, got, want)
		}
	}
}

// TestCaseAndTrailingDotFold: the owner-name test folds ASCII case and a trailing
// dot, matching the Name key, so a mixed-case A-record owner still extends.
func TestCaseAndTrailingDotFold(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"Example.COM"},
		Resolutions:   []Resolution{{Owner: "API.Example.Com.", Address: addr("52.1.2.3")}},
	}
	if got := e.Derive(addr("52.1.2.3")); got != Operator {
		t.Errorf("Derive with mixed-case owner/zone = %q, want operator", got)
	}
}

// TestNoRegistryExpansionInputReachesTheDerivation is the structural proof of
// AC-1: the Estate accepts only confirmed Seeds. A registry-proposed address
// scope the operator has not confirmed is simply not an input, so an address
// inside it derives third-party. Compare with the same range confirmed into an
// address-scope Seed, which derives operator — the only difference is the
// operator's confirmation act.
func TestNoRegistryExpansionInputReachesTheDerivation(t *testing.T) {
	// A proposed range covering 76M AWS addresses (ADR-0002's #27 measurement)
	// is NOT confirmed, so it is absent from the Estate.
	unconfirmed := Estate{}
	if got := unconfirmed.Derive(addr("52.1.2.3")); got != ThirdParty {
		t.Errorf("Derive(52.1.2.3) with only a proposal = %q, want third-party", got)
	}

	// The same address once the operator confirms a /32 Seed over it.
	confirmed := Estate{AddressScopes: []netip.Prefix{cidr("52.1.2.3/32")}}
	if got := confirmed.Derive(addr("52.1.2.3")); got != Operator {
		t.Errorf("Derive(52.1.2.3) after confirmation = %q, want operator", got)
	}
}
