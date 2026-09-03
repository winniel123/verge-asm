package edgefanout

import (
	"context"
	"net/netip"
	"testing"
	"time"
)

var nonGlobalTargets = []netip.AddrPort{
	netip.MustParseAddrPort("169.254.169.254:443"),
	netip.MustParseAddrPort("127.0.0.1:443"),
	netip.MustParseAddrPort("10.0.0.5:443"),
	netip.MustParseAddrPort("192.168.1.1:443"),
}

func TestNetHandshakerGuardRefusesNonGlobal(t *testing.T) {
	// The same ranges connect-outcome's guard test pins: metadata, loopback and RFC1918.
	h := NetHandshaker{Timeout: 2 * time.Second}
	// A guard refusal is unreachable, never the no-tls value that would claim an answer (#743).
	for _, target := range nonGlobalTargets {
		// Were the guard absent the dial would block to the timeout, so a fast refusal is the proof.
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

func TestNetHandshakerRejectsInvalidTarget(t *testing.T) {
	h := NetHandshaker{Timeout: 2 * time.Second}
	if got := h.Handshake(context.Background(), netip.AddrPort{}); got.Outcome != Unreachable {
		t.Errorf("Handshake(zero) = %q, want %q", got.Outcome, Unreachable)
	}
}
