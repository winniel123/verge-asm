package httpexchange

import (
	"context"
	"strings"
	"syscall"
	"testing"
	"time"
)

// allowAllControl is a net.Dialer Control hook that permits every socket. It lets a
// wiring test reach a loopback httptest server, which the production egress guard
// (custody.EgressGuard) refuses (#743).
func allowAllControl(_, _ string, _ syscall.RawConn) error { return nil }

// TestExchangeRejectsHostnameAddress pins the entry assertion: an Address that is
// not a literal IP is refused before any dial, so a hostname can never be
// re-resolved by the HTTP client at connect time with no rebinding backstop (#743).
func TestExchangeRejectsHostnameAddress(t *testing.T) {
	ex := NetExchanger{Params: DefaultParams()}
	res := ex.Exchange(context.Background(), Target{
		Address: "metadata.internal", // a hostname, not a literal
		Port:    80,
		Scheme:  "http",
	})
	if !res.Failed {
		t.Fatalf("hostname Address was not refused: %+v", res)
	}
	if !strings.Contains(res.Err, "refusing non-literal address") {
		t.Errorf("err = %q, want a non-literal refusal", res.Err)
	}
}

// TestExchangeGuardRefusesNonGlobalLiteral pins the socket-level backstop: a
// literal target in a non-globally-reachable range (link-local metadata, RFC1918,
// loopback) is refused at the dialer's Control hook, so no packet is sent. The
// production NetExchanger installs custody.EgressGuard by default (nil control).
func TestExchangeGuardRefusesNonGlobalLiteral(t *testing.T) {
	// A tight timeout: were the guard absent, dialing these would block until the
	// deadline; the guard refuses instantly, so a fast Failed proves it fired.
	params := DefaultParams()
	params.TimeoutMillis = 2000
	ex := NetExchanger{Params: params}
	for _, addr := range []string{"169.254.169.254", "10.0.0.5", "127.0.0.1", "192.168.1.1"} {
		start := time.Now()
		res := ex.Exchange(context.Background(), Target{Address: addr, Port: 80, Scheme: "http"})
		if !res.Failed {
			t.Errorf("Exchange(%s) not refused: %+v", addr, res)
		}
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Errorf("Exchange(%s) took %v — guard did not refuse before the dial", addr, elapsed)
		}
	}
}
