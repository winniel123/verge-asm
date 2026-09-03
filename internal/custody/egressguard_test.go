package custody

import (
	"strings"
	"testing"
)

func TestEgressGuardRefusesNonGlobal(t *testing.T) {
	guard := EgressGuard("testleaf")

	// 169.254.169.254 is the cloud metadata endpoint the guard exists to refuse (#743).
	refused := []string{
		"169.254.169.254:80",
		"127.0.0.1:80",
		"10.0.0.5:80",
		"192.168.1.1:80",
		"172.16.0.1:80",
		"[fd00::1]:80",
		"[::1]:80",
		"[::ffff:169.254.0.1]:80",
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

	if err := guard("tcp", "metadata.internal:80", nil); err == nil {
		t.Error("EgressGuard(hostname) = nil, want refusal (non-literal host)")
	}
}
