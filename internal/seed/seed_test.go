package seed

import (
	"net/netip"
	"slices"
	"testing"
)

func TestNormalizeDomain(t *testing.T) {
	ok := map[string]string{
		"example.com":     "example.com",
		"  Example.COM  ": "example.com",
		"example.com.":    "example.com",
		"example.co.uk":   "example.co.uk",
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
		"www.example.com",
		"co.uk",
		"example",
		"*.example.com",
		"http://example.com",
		"example.com/path",
		// The normalized domain reaches the crt.sh query URL unencoded (#774).
		"example.com&output=text",
		"example.com#frag",
		"a;b.com",
		"example.com'",
		"example.com%2e",
		"exa mple.com",
		"exa\tmple.com",
		"exa\nmple.com",
		"example_com",
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
		{"10.0.0.0/22", 1024},
		{"2001:db8::/118", 1024},
		{"2001:db8::/128", 1},
	}
	for _, c := range cases {
		p := netip.MustParsePrefix(c.cidr)
		if got := AddressCount(p).Int64(); got != c.count {
			t.Errorf("AddressCount(%s) = %d, want %d", c.cidr, got, c.count)
		}
	}

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

func TestLargestPrefixLen(t *testing.T) {
	cases := []struct {
		cap        int
		familyBits int
		want       int
	}{
		{1024, 32, 22},
		{1024, 128, 118},
		{262144, 32, 14},
		{262144, 128, 110},
		{1000, 32, 23},
		{1023, 32, 23},
		{1, 32, 32},
		{1, 128, 128},
		{0, 32, 32},
		{1 << 33, 32, 0},
	}
	for _, c := range cases {
		if got := LargestPrefixLen(c.cap, c.familyBits); got != c.want {
			t.Errorf("LargestPrefixLen(%d, %d) = %d, want %d", c.cap, c.familyBits, got, c.want)
		}
	}
}

func TestEnumerateAddresses(t *testing.T) {
	got := slices.Collect(EnumerateAddresses(netip.MustParsePrefix("192.0.2.8/30")))
	want := []string{"192.0.2.8", "192.0.2.9", "192.0.2.10", "192.0.2.11"}
	if len(got) != len(want) {
		t.Fatalf("a /30 enumerates %d addresses, got %d: %v", len(want), len(got), got)
	}
	for i, w := range want {
		if got[i].String() != w {
			t.Errorf("address %d = %s, want %s", i, got[i], w)
		}
	}

	if g := slices.Collect(EnumerateAddresses(netip.MustParsePrefix("203.0.113.5/32"))); len(g) != 1 || g[0].String() != "203.0.113.5" {
		t.Errorf("a /32 enumerates its one address, got %v", g)
	}

	// Network and broadcast are never exempted (ADR-0047).
	if g := slices.Collect(EnumerateAddresses(netip.MustParsePrefix("198.51.100.0/31"))); len(g) != 2 {
		t.Errorf("a /31 enumerates both addresses, got %v", g)
	}

	if g := slices.Collect(EnumerateAddresses(netip.MustParsePrefix("10.0.0.5/30"))); len(g) != 4 || g[0].String() != "10.0.0.4" {
		t.Errorf("a host-bits-set prefix must be masked before enumeration, got %v", g)
	}

	// Enumeration is family-agnostic (ADR-0049).
	if g := slices.Collect(EnumerateAddresses(netip.MustParsePrefix("2001:db8::/126"))); len(g) != 4 || g[0].String() != "2001:db8::" {
		t.Errorf("a /126 enumerates 4 IPv6 addresses, got %v", g)
	}
}

func TestEnumerateAddressesStreamsLazily(t *testing.T) {
	var got []string
	// No materializing implementation passes this: 2^31 addresses, broken early (ADR-0127).
	for a := range EnumerateAddresses(netip.MustParsePrefix("0.0.0.0/1")) {
		got = append(got, a.String())
		if len(got) == 5 {
			break
		}
	}
	want := []string{"0.0.0.0", "0.0.0.1", "0.0.0.2", "0.0.0.3", "0.0.0.4"}
	if !slices.Equal(got, want) {
		t.Fatalf("a lazy enumeration yields the first five of a huge scope, got %v", got)
	}
}

func BenchmarkEnumerateAddressesLargeScope(b *testing.B) {
	p := netip.MustParsePrefix("10.0.0.0/16")
	b.ReportAllocs()
	for b.Loop() {
		n := 0
		for range EnumerateAddresses(p) {
			n++
		}
		if n != 65536 {
			b.Fatalf("walked %d addresses, want 65536", n)
		}
	}
}

func TestEnumCapHint(t *testing.T) {
	if got := EnumCapHint(netip.MustParsePrefix("203.0.113.0/24")); got != 256 {
		t.Errorf("EnumCapHint(/24) = %d, want 256", got)
	}
	if got := EnumCapHint(netip.MustParsePrefix("203.0.113.5/32")); got != 1 {
		t.Errorf("EnumCapHint(/32) = %d, want 1", got)
	}
	if got := EnumCapHint(netip.MustParsePrefix("10.0.0.0/8")); got != maxEnumCapHint {
		t.Errorf("EnumCapHint(/8) = %d, want the %d ceiling", got, maxEnumCapHint)
	}
	if got := EnumCapHint(netip.MustParsePrefix("::/0")); got != 0 {
		t.Errorf("EnumCapHint(::/0) = %d, want 0 (count exceeds int)", got)
	}
}

func TestEnumerateAddressesTerminatesAtTopOfSpace(t *testing.T) {
	got := slices.Collect(EnumerateAddresses(netip.MustParsePrefix("255.255.255.254/31")))
	if len(got) != 2 || got[0].String() != "255.255.255.254" || got[1].String() != "255.255.255.255" {
		t.Fatalf("enumeration at the top of the space must terminate cleanly, got %v", got)
	}
}
