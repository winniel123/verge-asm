package custody

import "net/netip"

// A True or terminated cell is not False, so it is not barred and runs toward probing (ADR-0079).

type spBlock struct {
	prefix netip.Prefix
	fires  bool
}

// A non-firing row overrides its firing parent, so dropping one would be a selection (ADR-0079).

var specialPurpose = []spBlock{ // IANA registries, cell for cell, 2026-08-15 (registry note §2)
	{netip.MustParsePrefix("0.0.0.0/8"), true},
	{netip.MustParsePrefix("0.0.0.0/32"), true},
	{netip.MustParsePrefix("10.0.0.0/8"), true},
	{netip.MustParsePrefix("100.64.0.0/10"), true},
	{netip.MustParsePrefix("127.0.0.0/8"), true},
	{netip.MustParsePrefix("169.254.0.0/16"), true},
	{netip.MustParsePrefix("172.16.0.0/12"), true},
	{netip.MustParsePrefix("192.0.0.0/24"), true},
	{netip.MustParsePrefix("192.0.0.0/29"), true},
	{netip.MustParsePrefix("192.0.0.8/32"), true},
	{netip.MustParsePrefix("192.0.0.9/32"), false},
	{netip.MustParsePrefix("192.0.0.10/32"), false},
	{netip.MustParsePrefix("192.0.0.170/32"), true},
	{netip.MustParsePrefix("192.0.0.171/32"), true},
	{netip.MustParsePrefix("192.0.2.0/24"), true},
	{netip.MustParsePrefix("192.31.196.0/24"), false},
	{netip.MustParsePrefix("192.52.193.0/24"), false},
	{netip.MustParsePrefix("192.88.99.0/24"), false},
	{netip.MustParsePrefix("192.88.99.2/32"), true},
	{netip.MustParsePrefix("192.168.0.0/16"), true},
	{netip.MustParsePrefix("192.175.48.0/24"), false},
	{netip.MustParsePrefix("198.18.0.0/15"), true},
	{netip.MustParsePrefix("198.51.100.0/24"), true},
	{netip.MustParsePrefix("203.0.113.0/24"), true},
	{netip.MustParsePrefix("240.0.0.0/4"), true},
	{netip.MustParsePrefix("255.255.255.255/32"), true},

	{netip.MustParsePrefix("::1/128"), true},
	{netip.MustParsePrefix("::/128"), true},
	// Unreachable by construction: IsNonGloballyReachable Unmaps first (registry note §4.1).
	{netip.MustParsePrefix("::ffff:0:0/96"), true},
	{netip.MustParsePrefix("64:ff9b::/96"), false},
	{netip.MustParsePrefix("64:ff9b:1::/48"), true},
	{netip.MustParsePrefix("100::/64"), true},
	{netip.MustParsePrefix("100:0:0:1::/64"), true},
	{netip.MustParsePrefix("2001::/23"), true},
	{netip.MustParsePrefix("2001::/32"), false},
	{netip.MustParsePrefix("2001:1::1/128"), false},
	{netip.MustParsePrefix("2001:1::2/128"), false},
	{netip.MustParsePrefix("2001:1::3/128"), false},
	{netip.MustParsePrefix("2001:2::/48"), true},
	{netip.MustParsePrefix("2001:3::/32"), false},
	{netip.MustParsePrefix("2001:4:112::/48"), false},
	{netip.MustParsePrefix("2001:10::/28"), false},
	{netip.MustParsePrefix("2001:20::/28"), false},
	{netip.MustParsePrefix("2001:30::/28"), false},
	{netip.MustParsePrefix("2001:db8::/32"), true},
	{netip.MustParsePrefix("2002::/16"), false},
	{netip.MustParsePrefix("2620:4f:8000::/48"), false},
	{netip.MustParsePrefix("3fff::/20"), true},
	{netip.MustParsePrefix("5f00::/16"), true},
	{netip.MustParsePrefix("fc00::/7"), true},
	{netip.MustParsePrefix("fe80::/10"), true},
}

func IsNonGloballyReachable(addr netip.Addr) bool {
	addr = addr.Unmap()

	best := -1
	fires := false
	for _, b := range specialPurpose {
		if b.prefix.Contains(addr) && b.prefix.Bits() > best {
			best = b.prefix.Bits()
			fires = b.fires
		}
	}
	return fires
}

func IsGloballyReachable(addr netip.Addr) bool {
	return !IsNonGloballyReachable(addr)
}
