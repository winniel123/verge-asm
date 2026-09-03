package custody

import (
	"net/netip"
	"testing"
)

func TestGloballyReachableReading(t *testing.T) {
	cases := []struct {
		addr   string
		barred bool
		why    string
	}{
		{"52.1.2.3", false, "public IPv4"},
		{"2606:4700:4700::1111", false, "public IPv6"},

		{"10.0.0.5", true, "RFC 1918 10/8"},
		{"172.16.0.1", true, "RFC 1918 172.16/12"},
		{"192.168.1.1", true, "RFC 1918 192.168/16"},
		{"100.64.0.1", true, "shared address space (CGNAT)"},
		{"127.0.0.1", true, "loopback — False despite footnote"},
		{"169.254.169.254", true, "link-local (cloud metadata)"},
		{"::1", true, "IPv6 loopback"},
		{"fc00::1", true, "unique-local"},
		{"fe80::1", true, "IPv6 link-local"},
		{"2001:db8::1", true, "documentation"},
		{"203.0.113.7", true, "TEST-NET-3"},

		{"192.0.0.9", false, "PCP Anycast True inside 192.0.0.0/24 False"},
		{"192.0.0.8", true, "IPv4 dummy False, sibling of the True /32"},
		{"2001:1::1", false, "PCP Anycast True inside 2001::/23 False"},
		{"2001:5::1", true, "2001::/23 IETF Protocol Assignments, no more-specific block"},

		{"2002::1", false, "6to4 N/A — residue toward probing"},
		{"2001:0:1:2:3:4:5:6", false, "Teredo 2001::/32 N/A overrides 2001::/23"},
		{"192.88.99.10", false, "6to4 Relay Anycast N/A"},
		{"192.88.99.2", true, "6a44-relay False /32 inside the N/A /24"},

		{"64:ff9b::1", false, "64:ff9b::/96 True"},
		{"64:ff9b:1::1", true, "64:ff9b:1::/48 False"},
	}
	for _, c := range cases {
		addr := netip.MustParseAddr(c.addr)
		if got := IsNonGloballyReachable(addr); got != c.barred {
			t.Errorf("IsNonGloballyReachable(%s) = %v, want %v (%s)", c.addr, got, c.barred, c.why)
		}
		if IsGloballyReachable(addr) == c.barred {
			t.Errorf("IsGloballyReachable(%s) disagrees with its complement (%s)", c.addr, c.why)
		}
	}
}

func TestIPv4MappedReadOutAsIPv4(t *testing.T) {
	if !IsNonGloballyReachable(netip.MustParseAddr("::ffff:10.0.0.1")) {
		t.Error("::ffff:10.0.0.1 should be barred as 10.0.0.1")
	}
	if IsNonGloballyReachable(netip.MustParseAddr("::ffff:52.1.2.3")) {
		t.Error("::ffff:52.1.2.3 should be globally reachable as 52.1.2.3")
	}
}
