package custody

import (
	"fmt"
	"net"
	"net/netip"
	"syscall"
)

// EgressGuard returns a net.Dialer Control hook that inspects the ACTUAL resolved
// socket address of every connection and refuses to open the socket when it lands
// in a non-globally-reachable range (IsNonGloballyReachable). Control runs after
// DNS resolution, on the very address the kernel is about to connect to, so the
// address vetted is the address dialed — the rebinding-proof, fail-closed backstop
// that stands even when an upstream literal-IP invariant is broken (#743).
//
// It is the single source of the guard the delivery runner (NewHTTPDoer, #325) and
// resolutionwalk (custodyDialer, #335) install inline, factored here so the active-
// probe leaves (connect-outcome, tls-acceptance, http-exchange) reuse the exact
// same logic rather than minting their own. label names the refusing subsystem in
// the error so a refusal is legible at its call site.
func EgressGuard(label string) func(network, address string, c syscall.RawConn) error {
	return func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}
		ip, err := netip.ParseAddr(host)
		if err != nil {
			return err
		}
		if IsNonGloballyReachable(ip.Unmap()) {
			return fmt.Errorf("%s: refusing to dial non-globally-reachable address %s", label, host)
		}
		return nil
	}
}
