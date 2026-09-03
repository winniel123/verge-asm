package httpexchange

import (
	"context"
	"strings"
	"syscall"
	"testing"
	"time"
)

func allowAllControl(_, _ string, _ syscall.RawConn) error { return nil }

func TestExchangeRejectsHostnameAddress(t *testing.T) {
	// #743: a re-resolved hostname would carry no rebinding backstop at connect time.
	ex := NetExchanger{Params: DefaultParams()}
	res := ex.Exchange(context.Background(), Target{
		Address: "metadata.internal",
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

func TestExchangeGuardRefusesNonGlobalLiteral(t *testing.T) {
	// Were the guard absent the dial would block to the deadline, so a fast refusal is the proof.
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
