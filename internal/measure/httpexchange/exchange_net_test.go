package httpexchange

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync/atomic"
	"testing"

	"github.com/winniel123/verge-asm/internal/measure"
)

func TestNetExchangerSendsOneGetRootWithProbeUA(t *testing.T) {
	var calls int32
	var gotMethod, gotPath, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Server", "test-server")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// httptest binds an ephemeral loopback host:port; split it into the Target the
	// NetExchanger builds its URL from.
	ap, err := netip.ParseAddrPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("parse listener addr: %v", err)
	}

	// httptest binds loopback, which the production egress guard refuses; this
	// wiring test proves the request shape, so it installs an allow-all control.
	ex := NetExchanger{Params: DefaultParams(), control: allowAllControl}
	res := ex.Exchange(context.Background(), Target{
		Address: ap.Addr().String(),
		Port:    ap.Port(),
		Scheme:  "http",
	})

	if res.Failed {
		t.Fatalf("exchange failed: %s", res.Err)
	}
	if n := atomic.LoadInt32(&calls); n != 1 {
		t.Fatalf("expected exactly one request, got %d", n)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("expected GET, got %s", gotMethod)
	}
	if gotPath != "/" {
		t.Fatalf("expected path /, got %q", gotPath)
	}
	if gotUA != measure.ProbeUserAgent {
		t.Fatalf("expected User-Agent %q, got %q", measure.ProbeUserAgent, gotUA)
	}
	if res.Status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Status)
	}
}
