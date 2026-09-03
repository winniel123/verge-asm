package custody

import (
	"fmt"
	"net"
	"net/netip"
	"syscall"
)

func EgressGuard(label string) func(network, address string, c syscall.RawConn) error {
	return func(_, address string, _ syscall.RawConn) error {
		// Control runs after resolution, so the address vetted is the address dialed (#743).
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}
		ip, err := netip.ParseAddr(host)
		if err != nil {
			return err
		}
		// A deliberate second line of defence: it holds where an upstream literal-IP check does not.
		if IsNonGloballyReachable(ip.Unmap()) {
			return fmt.Errorf("%s: refusing to dial non-globally-reachable address %s", label, host)
		}
		return nil
	}
}
