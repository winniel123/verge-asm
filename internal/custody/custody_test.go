package custody

import (
	"net/netip"
	"testing"
)

func addr(s string) netip.Addr { return netip.MustParseAddr(s) }
func cidr(s string) netip.Prefix {
	return netip.MustParsePrefix(s).Masked()
}

func TestDeriveFromAddressScope(t *testing.T) {
	e := Estate{AddressScopes: []netip.Prefix{cidr("52.1.2.0/24"), cidr("2001:db8:1::/48")}}

	operator := []string{"52.1.2.1", "52.1.2.254", "2001:db8:1::5"}
	for _, a := range operator {
		if got := e.Derive(addr(a)); got != Operator {
			t.Errorf("Derive(%s) = %q, want operator", a, got)
		}
	}
	thirdParty := []string{"52.1.3.1", "203.0.113.9", "2001:db8:2::1"}
	for _, a := range thirdParty {
		if got := e.Derive(addr(a)); got != ThirdParty {
			t.Errorf("Derive(%s) = %q, want third-party", a, got)
		}
	}
}

func TestDeriveFromExtensionInZone(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "api.example.com", Address: addr("52.1.2.3")},
			// A CNAME puts the A record on the foreign name, inside no extended zone (ADR-0013 §3).
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

func TestExtensionRequiresTheExtensionFlag(t *testing.T) {
	e := Estate{
		Resolutions: []Resolution{{Owner: "api.example.com", Address: addr("52.1.2.3")}},
	}
	if got := e.Derive(addr("52.1.2.3")); got != ThirdParty {
		t.Errorf("Derive(52.1.2.3) = %q, want third-party (extension off)", got)
	}
}

func TestExtensionStopsAtNonGloballyReachableHop(t *testing.T) {
	ext := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions:   []Resolution{{Owner: "api.example.com", Address: addr("10.0.0.5")}},
	}
	if got := ext.Derive(addr("10.0.0.5")); got != ThirdParty {
		t.Errorf("Derive(10.0.0.5) via extension = %q, want third-party (NGR hop)", got)
	}

	scoped := Estate{AddressScopes: []netip.Prefix{cidr("10.0.0.0/24")}}
	if got := scoped.Derive(addr("10.0.0.5")); got != Operator {
		t.Errorf("Derive(10.0.0.5) via address scope = %q, want operator", got)
	}
}

func TestLabelSuffixIsNotStringSuffix(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "example.com", Address: addr("52.0.0.1")},
			{Owner: "deep.sub.example.com", Address: addr("52.0.0.2")},
			{Owner: "evilexample.com", Address: addr("52.0.0.3")},
			{Owner: "example.com.attacker.net", Address: addr("52.0.0.4")},
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

func TestCaseAndTrailingDotFold(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"Example.COM"},
		Resolutions:   []Resolution{{Owner: "API.Example.Com.", Address: addr("52.1.2.3")}},
	}
	if got := e.Derive(addr("52.1.2.3")); got != Operator {
		t.Errorf("Derive with mixed-case owner/zone = %q, want operator", got)
	}
}

func TestWithinAnyZone(t *testing.T) {
	// Four packages route their in-zone question here, so the fold is locked here (ADR-0055).
	zones := []string{"Example.COM", "corp.test"}
	cases := map[string]bool{
		"example.com":          true,
		"api.example.com.":     true,
		"corp.test":            true,
		"evilexample.com":      false,
		"example.com.evil.net": false,
		"test":                 false,
		"":                     false,
	}
	for name, want := range cases {
		if got := WithinAnyZone(name, zones); got != want {
			t.Errorf("WithinAnyZone(%q) = %v, want %v", name, got, want)
		}
	}
	if WithinAnyZone("example.com", nil) {
		t.Error("no zones must contain nothing")
	}
}

func TestNoRegistryExpansionInputReachesTheDerivation(t *testing.T) {
	// A proposal is no Estate field, so the 76M-address AWS range reaches nothing (ADR-0002, #27).
	unconfirmed := Estate{}
	if got := unconfirmed.Derive(addr("52.1.2.3")); got != ThirdParty {
		t.Errorf("Derive(52.1.2.3) with only a proposal = %q, want third-party", got)
	}

	confirmed := Estate{AddressScopes: []netip.Prefix{cidr("52.1.2.3/32")}}
	if got := confirmed.Derive(addr("52.1.2.3")); got != Operator {
		t.Errorf("Derive(52.1.2.3) after confirmation = %q, want operator", got)
	}
}
