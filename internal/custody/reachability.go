package custody

import "net/netip"

// This file transcribes the IANA IPv4 and IPv6 Special-Purpose Address
// Registries' `Globally Reachable` column and reads it by the one rule both
// consumers share (docs/research/special-purpose-address-registry.md §2, §8):
//
//	Take the most specific registered block containing the address. The address
//	is non-globally-reachable if and only if that block's `Globally Reachable`
//	cell reads `False`.
//
// It authors no row and no realm taxonomy — the registry publishes exactly one
// cut and it is binary (ADR-0079). A `True` cell and an `N/A`/terminated cell are
// both *not* `False`, so neither is barred: the residue runs toward probing
// (§8.1), which is the gate-side reading of the same refusal to supply a value
// the owner declined to supply.

// spBlock is one row of the special-purpose registries. fires is true exactly
// when the `Globally Reachable` cell reads `False` — i.e. the block is
// non-globally-reachable. A `True` or `N/A`/terminated cell records fires=false,
// so a more specific non-firing block overrides a firing parent under longest
// match (e.g. 192.0.0.9/32 True inside 192.0.0.0/24 False).
type spBlock struct {
	prefix netip.Prefix
	fires  bool
}

// specialPurpose is the whole of both registries as retrieved 2026-08-15 — 50
// blocks, transcribed cell for cell. It is intentionally the complete table and
// not the firing set alone, because longest match needs the non-firing blocks to
// override their firing parents; dropping them would be a selection.
var specialPurpose = []spBlock{
	// IPv4 — 25 blocks (note §3.1).
	{netip.MustParsePrefix("0.0.0.0/8"), true},
	{netip.MustParsePrefix("0.0.0.0/32"), true},
	{netip.MustParsePrefix("10.0.0.0/8"), true},
	{netip.MustParsePrefix("100.64.0.0/10"), true},
	{netip.MustParsePrefix("127.0.0.0/8"), true}, // False `[1]`, footnote does not move the read
	{netip.MustParsePrefix("169.254.0.0/16"), true},
	{netip.MustParsePrefix("172.16.0.0/12"), true},
	{netip.MustParsePrefix("192.0.0.0/24"), true},
	{netip.MustParsePrefix("192.0.0.0/29"), true},
	{netip.MustParsePrefix("192.0.0.8/32"), true},
	{netip.MustParsePrefix("192.0.0.9/32"), false},  // Port Control Protocol Anycast — True
	{netip.MustParsePrefix("192.0.0.10/32"), false}, // TURN Anycast — True
	{netip.MustParsePrefix("192.0.0.170/32"), true},
	{netip.MustParsePrefix("192.0.0.171/32"), true},
	{netip.MustParsePrefix("192.0.2.0/24"), true},
	{netip.MustParsePrefix("192.31.196.0/24"), false}, // AS112-v4 — True
	{netip.MustParsePrefix("192.52.193.0/24"), false}, // AMT — True
	{netip.MustParsePrefix("192.88.99.0/24"), false},  // 6to4 Relay Anycast — N/A
	{netip.MustParsePrefix("192.88.99.2/32"), true},
	{netip.MustParsePrefix("192.168.0.0/16"), true},
	{netip.MustParsePrefix("192.175.48.0/24"), false}, // Direct Delegation AS112 — True
	{netip.MustParsePrefix("198.18.0.0/15"), true},
	{netip.MustParsePrefix("198.51.100.0/24"), true},
	{netip.MustParsePrefix("203.0.113.0/24"), true},
	{netip.MustParsePrefix("240.0.0.0/4"), true},
	{netip.MustParsePrefix("255.255.255.255/32"), true},

	// IPv6 — 25 blocks (note §3.2).
	{netip.MustParsePrefix("::1/128"), true},
	{netip.MustParsePrefix("::/128"), true},
	{netip.MustParsePrefix("::ffff:0:0/96"), true}, // unreachable by construction — we Unmap first (§4.1)
	{netip.MustParsePrefix("64:ff9b::/96"), false}, // NAT64 — True
	{netip.MustParsePrefix("64:ff9b:1::/48"), true},
	{netip.MustParsePrefix("100::/64"), true},
	{netip.MustParsePrefix("100:0:0:1::/64"), true},
	{netip.MustParsePrefix("2001::/23"), true},      // False `[1]`
	{netip.MustParsePrefix("2001::/32"), false},     // TEREDO — N/A
	{netip.MustParsePrefix("2001:1::1/128"), false}, // PCP Anycast — True
	{netip.MustParsePrefix("2001:1::2/128"), false}, // TURN Anycast — True
	{netip.MustParsePrefix("2001:1::3/128"), false}, // DNS-SD SRP Anycast — True
	{netip.MustParsePrefix("2001:2::/48"), true},
	{netip.MustParsePrefix("2001:3::/32"), false},     // AMT — True
	{netip.MustParsePrefix("2001:4:112::/48"), false}, // AS112-v6 — True
	{netip.MustParsePrefix("2001:10::/28"), false},    // ORCHID, terminated 2014-03
	{netip.MustParsePrefix("2001:20::/28"), false},    // ORCHIDv2 — True
	{netip.MustParsePrefix("2001:30::/28"), false},    // DETs — True
	{netip.MustParsePrefix("2001:db8::/32"), true},
	{netip.MustParsePrefix("2002::/16"), false},         // 6to4 — N/A
	{netip.MustParsePrefix("2620:4f:8000::/48"), false}, // Direct Delegation AS112 — True
	{netip.MustParsePrefix("3fff::/20"), true},
	{netip.MustParsePrefix("5f00::/16"), true},
	{netip.MustParsePrefix("fc00::/7"), true},
	{netip.MustParsePrefix("fe80::/10"), true},
}

// IsNonGloballyReachable reports whether addr's most specific registered block
// reads `Globally Reachable = False`. An address inside no registered block is
// ordinary global unicast and returns false, with no default and no inference
// (note §2). An IPv4-mapped address is read out as the IPv4 address it
// represents (ADR-0051), so it is classified in IPv4 space and never lands in
// ::ffff:0:0/96.
func IsNonGloballyReachable(addr netip.Addr) bool {
	addr = addr.Unmap()

	best := -1
	fires := false
	for _, b := range specialPurpose {
		// Contains is family-matched, so a cross-family block never matches.
		if b.prefix.Contains(addr) && b.prefix.Bits() > best {
			best = b.prefix.Bits()
			fires = b.fires
		}
	}
	return fires
}

// IsGloballyReachable is the complement, provided because both readings are
// asked by name at the call sites (a globally-reachable address is the ordinary
// case a custody extension may cover; a non-globally-reachable one is the barred
// population the denotation precondition guards).
func IsGloballyReachable(addr netip.Addr) bool {
	return !IsNonGloballyReachable(addr)
}
