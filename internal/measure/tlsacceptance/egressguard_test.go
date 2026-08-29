package tlsacceptance

import (
	"context"
	"net/netip"
	"testing"
	"time"
)

// TestNetEnumeratorGuardRefusesNonGlobal pins the tls-acceptance leaf's socket-level
// backstop: a handshake against a non-globally-reachable literal (cloud metadata,
// loopback, RFC1918) is refused at the dialer's Control hook before any TLS bytes,
// so the attempt neither spoke TLS nor accepted, regardless of upstream validation
// (#743). Were the guard absent these would block until the timeout; the guard
// refuses instantly.
func TestNetEnumeratorGuardRefusesNonGlobal(t *testing.T) {
	e := NetEnumerator{Timeout: 2 * time.Second}
	targets := []netip.AddrPort{
		netip.MustParseAddrPort("169.254.169.254:443"),
		netip.MustParseAddrPort("127.0.0.1:443"),
		netip.MustParseAddrPort("10.0.0.5:443"),
		netip.MustParseAddrPort("192.168.1.1:443"),
	}
	for _, target := range targets {
		start := time.Now()
		got := e.Handshake(context.Background(), target, TLS12, []string{"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA"})
		if got.Accepted || got.Spoke {
			t.Errorf("Handshake(%s) = %+v, want refusal (not spoke, not accepted)", target, got)
		}
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Errorf("Handshake(%s) took %v — guard did not refuse before the dial", target, elapsed)
		}
	}
}
