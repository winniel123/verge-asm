package custody

import (
	"fmt"
	"net"
	"net/netip"
	"syscall"
)

// The zero Realm exempts nothing, so a caller that supplies none keeps the whole guard (ADR-0225 §2).

func EgressGuard(label string, realm Realm) func(network, address string, c syscall.RawConn) error {
	return func(_, address string, _ syscall.RawConn) error {
		// Control runs after resolution, so the address vetted is the address dialed.
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}
		ip, err := netip.ParseAddr(host)
		if err != nil {
			return err
		}
		// A deliberate second line of defence: it holds where an upstream literal-IP check fails.
		if IsNonGloballyReachable(ip.Unmap()) && !realm.Contains(ip) {
			return fmt.Errorf("%s: refusing to dial non-globally-reachable address %s", label, host)
		}
		return nil
	}
}
