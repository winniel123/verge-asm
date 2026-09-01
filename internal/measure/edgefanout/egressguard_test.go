package edgefanout

import (
	"context"
	"net/netip"
	"testing"
	"time"
)

// nonGlobalTargets are literal addresses in non-globally-reachable ranges an active
// probe must never dial: cloud metadata (link-local), loopback and RFC1918. They are
// the same set connect-outcome's guard test pins, because this leaf must be blocked
// exactly as that one is.
var nonGlobalTargets = []netip.AddrPort{
	netip.MustParseAddrPort("169.254.169.254:443"),
	netip.MustParseAddrPort("127.0.0.1:443"),
	netip.MustParseAddrPort("10.0.0.5:443"),
	netip.MustParseAddrPort("192.168.1.1:443"),
}

// TestNetHandshakerGuardRefusesNonGlobal pins this leaf's socket-level backstop. The
// dial reuses connectoutcome's, so custody.EgressGuard refuses the socket before any
// TLS bytes: the outcome is `unreachable` — nothing was reached, because nothing was
// dialled — never a presented certificate, and never the `no-tls` value that would
// claim something answered (#743, ADR-0129 §6).
//
// Were the guard absent these would block until the dial timeout; the guard refuses
// instantly, which the elapsed-time assertion pins.
func TestNetHandshakerGuardRefusesNonGlobal(t *testing.T) {
	h := NetHandshaker{Timeout: 2 * time.Second}
	for _, target := range nonGlobalTargets {
		start := time.Now()
		got := h.Handshake(context.Background(), target)
		if got.Outcome != Unreachable {
			t.Errorf("Handshake(%s) = %q, want %q (guard refusal)", target, got.Outcome, Unreachable)
		}
		if got.Fingerprint != "" {
			t.Errorf("Handshake(%s) carries fingerprint %q, want none", target, got.Fingerprint)
		}
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Errorf("Handshake(%s) took %v — the guard did not refuse before the dial", target, elapsed)
		}
	}
}

// TestNetHandshakerRejectsInvalidTarget pins the entry assertion: a zero AddrPort never
// reaches a dial, so it is `unreachable` and not a measured negative.
func TestNetHandshakerRejectsInvalidTarget(t *testing.T) {
	h := NetHandshaker{Timeout: 2 * time.Second}
	if got := h.Handshake(context.Background(), netip.AddrPort{}); got.Outcome != Unreachable {
		t.Errorf("Handshake(zero) = %q, want %q", got.Outcome, Unreachable)
	}
}
