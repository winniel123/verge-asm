package custody

import (
	"strings"
	"testing"
)

// TestEgressGuardRefusesNonGlobal pins the shared socket-level backstop the
// active-probe leaves, the delivery runner and resolutionwalk all install: given
// the ACTUAL resolved socket address the kernel is about to connect to, the Control
// hook refuses every non-globally-reachable range and passes ordinary global
// unicast (#743). The label names the refusing subsystem in the error.
func TestEgressGuardRefusesNonGlobal(t *testing.T) {
	guard := EgressGuard("testleaf")

	refused := []string{
		"169.254.169.254:80",       // cloud metadata (link-local)
		"127.0.0.1:80",             // loopback
		"10.0.0.5:80",              // RFC1918
		"192.168.1.1:80",           // RFC1918
		"172.16.0.1:80",            // RFC1918
		"[fd00::1]:80",             // ULA
		"[::1]:80",                 // IPv6 loopback
		"[::ffff:169.254.0.1]:80",  // IPv4-mapped link-local
	}
	for _, addr := range refused {
		err := guard("tcp", addr, nil)
		if err == nil {
			t.Errorf("EgressGuard(%q) = nil, want refusal (non-globally-reachable)", addr)
			continue
		}
		if !strings.Contains(err.Error(), "testleaf") {
			t.Errorf("EgressGuard(%q) err = %q, want the label in the message", addr, err)
		}
	}

	allowed := []string{"8.8.8.8:80", "1.1.1.1:443", "[2001:4860:4860::8888]:443"}
	for _, addr := range allowed {
		if err := guard("tcp", addr, nil); err != nil {
			t.Errorf("EgressGuard(%q) = %v, want nil (globally reachable)", addr, err)
		}
	}

	// A non-literal host (a hostname that slipped past validation) fails closed too:
	// ParseAddr rejects it, so the socket is never opened.
	if err := guard("tcp", "metadata.internal:80", nil); err == nil {
		t.Error("EgressGuard(hostname) = nil, want refusal (non-literal host)")
	}
}
