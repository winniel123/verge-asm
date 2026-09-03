package signal

import "net/netip"

// anyNonGloballyReachable reports whether any address in the set is not globally
// reachable — the predicate of
// `non-globally-reachable-address-resolved-from-internet`. It classifies over the
// address key (family and octets), never a spelling, so an unparseable spelling
// is ignored rather than guessed at (CONTEXT.md `Address`; ADR-0051).
func anyNonGloballyReachable(addrs []string) bool {
	for _, a := range addrs {
		addr, err := netip.ParseAddr(a)
		if err != nil {
			continue
		}
		if !globallyReachable(addr.Unmap()) {
			return true
		}
	}
	return false
}

// globallyReachable reports whether an address is a public unicast address an
// outside host could route to. The complement is the fact the rule reads: an
// RFC 1918 / ULA private address, loopback, link-local, the unspecified address,
// or any non-global-unicast (multicast, and IPv6 special-purpose space) resolved
// into a public DNS answer is an internal address leaking to the internet.
func globallyReachable(addr netip.Addr) bool {
	if !addr.IsValid() {
		return false
	}
	if !addr.IsGlobalUnicast() {
		return false
	}
	if addr.IsPrivate() {
		// RFC 1918 (IPv4) and ULA fc00::/7 (IPv6): global unicast in form,
		// non-routable on the public internet in fact.
		return false
	}
	if addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() {
		return false
	}
	return true
}
