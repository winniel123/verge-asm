package signal

import "net/netip"

func anyNonGloballyReachable(addrs []string) bool {
	// Classified over the address key, never a spelling, so an unparseable one is ignored (ADR-0051).
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

func globallyReachable(addr netip.Addr) bool {
	if !addr.IsValid() {
		return false
	}
	if !addr.IsGlobalUnicast() {
		return false
	}
	if addr.IsPrivate() {
		// RFC 1918 and ULA fc00::/7 are global unicast in form and non-routable in fact.
		return false
	}
	if addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() {
		return false
	}
	return true
}
