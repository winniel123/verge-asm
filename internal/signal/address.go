package signal

import "net/netip"

func anyNonGloballyReachable(addrs []string) bool {
	// Classified on the address key, never a spelling: an unparseable one is ignored (ADR-0051).
	for _, a := range addrs {
		addr, err := netip.ParseAddr(a)
		if err != nil {
			continue
		}
		if specialPurposeFires(addr.Unmap()) {
			return true
		}
	}
	return false
}

type specialPurposeBlock struct {
	prefix netip.Prefix
	fires  bool
}

func block(cidr string, fires bool) specialPurposeBlock {
	return specialPurposeBlock{prefix: netip.MustParsePrefix(cidr), fires: fires}
}

// Both IANA registries in full, never a selection; a non-False row never fires (ADR-0071 §2).

var specialPurposeTable = []specialPurposeBlock{
	block("0.0.0.0/8", true),
	block("0.0.0.0/32", true),
	block("10.0.0.0/8", true),
	block("100.64.0.0/10", true),
	block("127.0.0.0/8", true),
	block("169.254.0.0/16", true),
	block("172.16.0.0/12", true),
	block("192.0.0.0/24", true),
	block("192.0.0.0/29", true),
	block("192.0.0.8/32", true),
	block("192.0.0.9/32", false),
	block("192.0.0.10/32", false),
	block("192.0.0.170/32", true),
	block("192.0.0.171/32", true),
	block("192.0.2.0/24", true),
	block("192.31.196.0/24", false),
	block("192.52.193.0/24", false),
	block("192.88.99.0/24", false),
	block("192.88.99.2/32", true),
	block("192.168.0.0/16", true),
	block("192.175.48.0/24", false),
	block("198.18.0.0/15", true),
	block("198.51.100.0/24", true),
	block("203.0.113.0/24", true),
	block("240.0.0.0/4", true),
	block("255.255.255.255/32", true),

	block("::1/128", true),
	block("::/128", true),
	block("::ffff:0:0/96", true),
	block("64:ff9b::/96", false),
	block("64:ff9b:1::/48", true),
	block("100::/64", true),
	block("100:0:0:1::/64", true),
	block("2001::/23", true),
	block("2001::/32", false),
	block("2001:1::1/128", false),
	block("2001:1::2/128", false),
	block("2001:1::3/128", false),
	block("2001:2::/48", true),
	block("2001:3::/32", false),
	block("2001:4:112::/48", false),
	block("2001:10::/28", false),
	block("2001:20::/28", false),
	block("2001:30::/28", false),
	block("2001:db8::/32", true),
	block("2002::/16", false),
	block("2620:4f:8000::/48", false),
	block("3fff::/20", true),
	block("5f00::/16", true),
	block("fc00::/7", true),
	block("fe80::/10", true),
}

func specialPurposeFires(addr netip.Addr) bool {
	if !addr.IsValid() {
		return false
	}
	// Longest match is the registry's own footnote: "unless allowed by a more specific allocation".
	best, found := specialPurposeBlock{}, false
	for _, b := range specialPurposeTable {
		if !b.prefix.Contains(addr) {
			continue
		}
		if !found || b.prefix.Bits() > best.prefix.Bits() {
			best, found = b, true
		}
	}
	return found && best.fires
}
