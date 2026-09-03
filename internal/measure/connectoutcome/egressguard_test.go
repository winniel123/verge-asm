package connectoutcome

import (
	"context"
	"net/netip"
	"testing"
	"time"
)

// nonGlobalTargets are literal addresses in non-globally-reachable ranges the
// active-probe leaves must never dial: cloud metadata (link-local), loopback and
// RFC1918. The socket-level egress guard refuses each at the dialer's Control hook.
var nonGlobalTargets = []netip.AddrPort{
	netip.MustParseAddrPort("169.254.169.254:80"),
	netip.MustParseAddrPort("127.0.0.1:80"),
	netip.MustParseAddrPort("10.0.0.5:80"),
	netip.MustParseAddrPort("192.168.1.1:80"),
}

// TestNetConnectorGuardRefusesNonGlobal pins the connect leaf's socket-level
// backstop: a connect to a non-globally-reachable literal is refused at the
// Control hook (a local error, never a real handshake or a timeout), so no SYN is
// sent regardless of upstream validation (#743). Were the guard absent these
// would block until the connect timeout; the guard refuses instantly.
func TestNetConnectorGuardRefusesNonGlobal(t *testing.T) {
	c := NetConnector{Timeout: 2 * time.Second}
	for _, target := range nonGlobalTargets {
		start := time.Now()
		got := c.Connect(context.Background(), target)
		if got != ConnError {
			t.Errorf("Connect(%s) = %q, want %q (guard refusal)", target, got, ConnError)
		}
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Errorf("Connect(%s) took %v — guard did not refuse before the dial", target, elapsed)
		}
	}
}

func TestNetConnectorRejectsInvalidTarget(t *testing.T) {
	c := NetConnector{Timeout: 2 * time.Second}
	if got := c.Connect(context.Background(), netip.AddrPort{}); got != ConnError {
		t.Errorf("Connect(zero) = %q, want %q", got, ConnError)
	}
}

// TestNetHandshakerGuardRefusesNonGlobal pins the tls-handshake leaf's backstop: a
// handshake against a non-globally-reachable literal is refused at the Control hook
// before any TLS bytes, so the outcome is a negative, never a presented chain (#743).
func TestNetHandshakerGuardRefusesNonGlobal(t *testing.T) {
	h := NetHandshaker{Timeout: 2 * time.Second}
	for _, target := range nonGlobalTargets {
		start := time.Now()
		res := h.Handshake(context.Background(), target, "")
		if res.Outcome == TLSPresented {
			t.Errorf("Handshake(%s) = presented, want a refusal negative", target)
		}
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Errorf("Handshake(%s) took %v — guard did not refuse before the dial", target, elapsed)
		}
	}
}
