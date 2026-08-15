package seed

import (
	"net/netip"
	"testing"
)

func TestNormalizeDomain(t *testing.T) {
	ok := map[string]string{
		"example.com":     "example.com",
		"  Example.COM  ": "example.com",
		"example.com.":    "example.com",
		"example.co.uk":   "example.co.uk", // eTLD+1 over a multi-label suffix
	}
	for in, want := range ok {
		got, err := NormalizeDomain(in)
		if err != nil {
			t.Errorf("NormalizeDomain(%q) errored: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("NormalizeDomain(%q) = %q, want %q", in, got, want)
		}
	}

	bad := []string{
		"",
		"www.example.com", // subdomain, not registrable
		"co.uk",           // bare public suffix
		"example",         // no suffix
		"*.example.com",   // wildcard
		"http://example.com",
		"example.com/path",
	}
	for _, in := range bad {
		if got, err := NormalizeDomain(in); err == nil {
			t.Errorf("NormalizeDomain(%q) = %q, want error", in, got)
		}
	}
}

func TestParseCIDRCanonicalises(t *testing.T) {
	p, err := ParseCIDR("10.0.0.5/24")
	if err != nil {
		t.Fatal(err)
	}
	if p.String() != "10.0.0.0/24" {
		t.Fatalf("ParseCIDR canonical form = %q, want 10.0.0.0/24", p.String())
	}
	if _, err := ParseCIDR("not-a-cidr"); err == nil {
		t.Fatal("expected error for invalid CIDR")
	}
	if _, err := ParseCIDR("10.0.0.0"); err == nil {
		t.Fatal("expected error for a bare address with no prefix")
	}
}

func TestAddressCountAndCap(t *testing.T) {
	cases := []struct {
		cidr  string
		count int64
	}{
		{"203.0.113.0/24", 256},
		{"10.0.0.0/22", 1024},    // the IPv4 boundary case
		{"2001:db8::/118", 1024}, // the equivalently-sized IPv6 block
		{"2001:db8::/128", 1},
	}
	for _, c := range cases {
		p := netip.MustParsePrefix(c.cidr)
		if got := AddressCount(p).Int64(); got != c.count {
			t.Errorf("AddressCount(%s) = %d, want %d", c.cidr, got, c.count)
		}
	}

	// The cap counts addresses regardless of family: /22 and /118 both sit
	// exactly on a 1024 cap, while one bit larger blows it.
	within := []string{"10.0.0.0/22", "2001:db8::/118", "203.0.113.0/24"}
	for _, s := range within {
		if !WithinCap(netip.MustParsePrefix(s), DefaultAddressCap) {
			t.Errorf("%s should be within cap %d", s, DefaultAddressCap)
		}
	}
	over := []string{"10.0.0.0/21", "2001:db8::/117", "0.0.0.0/0", "::/0"}
	for _, s := range over {
		if WithinCap(netip.MustParsePrefix(s), DefaultAddressCap) {
			t.Errorf("%s should exceed cap %d", s, DefaultAddressCap)
		}
	}
}
