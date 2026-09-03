package tlsacceptance

import (
	"context"
	"net/netip"
	"testing"
	"time"
)

func TestNetEnumeratorGuardRefusesNonGlobal(t *testing.T) {
	// Without the guard these would block to the timeout, so the elapsed check is the proof (#743).
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
