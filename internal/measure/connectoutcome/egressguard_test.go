package connectoutcome

import (
	"context"
	"net/netip"
	"testing"
	"time"
)

// 169.254.169.254 is the cloud metadata endpoint, the SSRF target the guard refuses (#743).

var nonGlobalTargets = []netip.AddrPort{
	netip.MustParseAddrPort("169.254.169.254:80"),
	netip.MustParseAddrPort("127.0.0.1:80"),
	netip.MustParseAddrPort("10.0.0.5:80"),
	netip.MustParseAddrPort("192.168.1.1:80"),
}

func TestNetConnectorGuardRefusesNonGlobal(t *testing.T) {
	c := NetConnector{Timeout: 2 * time.Second}
	// A socket-level backstop: no SYN leaves the host even where upstream validation fails (#743).
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
