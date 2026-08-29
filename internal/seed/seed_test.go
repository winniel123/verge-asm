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
		// Query-injection characters must not survive the validator (#774): the
		// normalized domain is interpolated into the crt.sh query URL, and
		// publicsuffix's wildcard rule would otherwise pass these through.
		"example.com&output=text", // injects a competing query param
		"example.com#frag",        // fragment drops &output=json
		"a;b.com",                 // statement/param separator
		"example.com'",            // quote
		"example.com%2e",          // percent-encoding
		"exa mple.com",            // internal whitespace (trailing is trimmed)
		"exa\tmple.com",           // internal tab
		"exa\nmple.com",           // internal newline
		"example_com",             // underscore (not LDH)
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

func TestEnumerateAddresses(t *testing.T) {
	// A /30 enumerates all four addresses in ascending order.
	got := EnumerateAddresses(netip.MustParsePrefix("192.0.2.8/30"))
	want := []string{"192.0.2.8", "192.0.2.9", "192.0.2.10", "192.0.2.11"}
	if len(got) != len(want) {
		t.Fatalf("a /30 enumerates %d addresses, got %d: %v", len(want), len(got), got)
	}
	for i, w := range want {
		if got[i].String() != w {
			t.Errorf("address %d = %s, want %s", i, got[i], w)
		}
	}

	// A /32 enumerates exactly its one address.
	if g := EnumerateAddresses(netip.MustParsePrefix("203.0.113.5/32")); len(g) != 1 || g[0].String() != "203.0.113.5" {
		t.Errorf("a /32 enumerates its one address, got %v", g)
	}

	// Network and broadcast are NOT exempted (ADR-0047): a /31 yields both ends.
	if g := EnumerateAddresses(netip.MustParsePrefix("198.51.100.0/31")); len(g) != 2 {
		t.Errorf("a /31 enumerates both addresses, got %v", g)
	}

	// The prefix is masked first, so host bits in the input do not move the block.
	if g := EnumerateAddresses(netip.MustParsePrefix("10.0.0.5/30")); len(g) != 4 || g[0].String() != "10.0.0.4" {
		t.Errorf("a host-bits-set prefix must be masked before enumeration, got %v", g)
	}

	// Enumeration is family-agnostic (ADR-0049): a /126 yields four IPv6 addresses.
	if g := EnumerateAddresses(netip.MustParsePrefix("2001:db8::/126")); len(g) != 4 || g[0].String() != "2001:db8::" {
		t.Errorf("a /126 enumerates 4 IPv6 addresses, got %v", g)
	}
}

func TestEnumCapHint(t *testing.T) {
	// A within-cap scope hints its exact size.
	if got := EnumCapHint(netip.MustParsePrefix("203.0.113.0/24")); got != 256 {
		t.Errorf("EnumCapHint(/24) = %d, want 256", got)
	}
	if got := EnumCapHint(netip.MustParsePrefix("203.0.113.5/32")); got != 1 {
		t.Errorf("EnumCapHint(/32) = %d, want 1", got)
	}
	// An oversized scope whose count still fits an int is bounded to the ceiling,
	// never allocated whole; the walk (EnumerateAddresses) stays unbounded and only
	// the pre-size hint is capped.
	if got := EnumCapHint(netip.MustParsePrefix("10.0.0.0/8")); got != maxEnumCapHint {
		t.Errorf("EnumCapHint(/8) = %d, want the %d ceiling", got, maxEnumCapHint)
	}
	// A count that does not fit an int (2^128) yields hint 0 — the buffer simply
	// grows from empty, never a pre-allocation from an unbounded number.
	if got := EnumCapHint(netip.MustParsePrefix("::/0")); got != 0 {
		t.Errorf("EnumCapHint(::/0) = %d, want 0 (count exceeds int)", got)
	}
}

// A scope at the very top of the address space must terminate: Next overflows to
// the invalid zero address, and the walk stops rather than looping.
func TestEnumerateAddressesTerminatesAtTopOfSpace(t *testing.T) {
	got := EnumerateAddresses(netip.MustParsePrefix("255.255.255.254/31"))
	if len(got) != 2 || got[0].String() != "255.255.255.254" || got[1].String() != "255.255.255.255" {
		t.Fatalf("enumeration at the top of the space must terminate cleanly, got %v", got)
	}
}
