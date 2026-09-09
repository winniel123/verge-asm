package httpexchange

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/winniel123/verge-asm/internal/measure"
	"github.com/winniel123/verge-asm/internal/wire"
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

	ap, err := netip.ParseAddrPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("parse listener addr: %v", err)
	}

	// httptest binds loopback, which the production egress guard refuses, so this allows it.
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

func TestNetExchangerReadsAnHTTPSListenerWhoseCertificateNamesNoIP(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "tls-test-server")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ap, err := netip.ParseAddrPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("parse listener addr: %v", err)
	}

	ex := NetExchanger{Params: DefaultParams(), control: allowAllControl}
	res := ex.Exchange(context.Background(), Target{
		Address: ap.Addr().String(),
		Port:    ap.Port(),
		Scheme:  "https",
	})

	if res.Failed {
		t.Fatalf("an https listener with an unverifiable certificate must still be read, got: %s", res.Err)
	}
	if res.Status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Status)
	}
	if res.Server != "tls-test-server" {
		t.Fatalf("expected Server header, got %q", res.Server)
	}
}

func TestNetExchangerOffersTheDeclaredALPNList(t *testing.T) {
	var gotProto string
	var gotMajor int
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotProto = r.TLS.NegotiatedProtocol
		gotMajor = r.ProtoMajor
		w.WriteHeader(http.StatusOK)
	}))
	srv.EnableHTTP2 = true
	srv.StartTLS()
	defer srv.Close()

	ap, err := netip.ParseAddrPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("parse listener addr: %v", err)
	}
	ex := NetExchanger{Params: DefaultParams(), control: allowAllControl}
	res := ex.Exchange(context.Background(), Target{Address: ap.Addr().String(), Port: ap.Port(), Scheme: "https"})
	if res.Failed {
		t.Fatalf("exchange failed: %s", res.Err)
	}
	// An h2-only-over-TLS listener answers only when h2 is offered (measurement-offers §3.1).
	if gotProto != "h2" || gotMajor != 2 {
		t.Fatalf("negotiated %q HTTP/%d, want h2 HTTP/2 from the declared ALPN list", gotProto, gotMajor)
	}
}

func TestNetExchangerReleasesTheConnectionAfterOneExchange(t *testing.T) {
	var gotClose bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClose = r.Close
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ap, err := netip.ParseAddrPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("parse listener addr: %v", err)
	}
	ex := NetExchanger{Params: DefaultParams(), control: allowAllControl}
	res := ex.Exchange(context.Background(), Target{Address: ap.Addr().String(), Port: ap.Port(), Scheme: "http"})
	if res.Failed {
		t.Fatalf("exchange failed: %s", res.Err)
	}
	// A pooled connection in a dropped per-target transport holds a socket 90 s (#1661).
	if !gotClose {
		t.Fatal("the one GET did not ask the listener to close the connection")
	}
}

func TestNetExchangerRefusesAResponseHeaderAboveTheCap(t *testing.T) {
	huge := strings.Repeat("x", 2<<20)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", huge)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ap, err := netip.ParseAddrPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("parse listener addr: %v", err)
	}
	target := Target{Name: "big.example.com", Address: ap.Addr().String(), Port: ap.Port(), Scheme: "http"}
	ex := NetExchanger{Params: DefaultParams(), control: allowAllControl}
	res := ex.Exchange(context.Background(), target)
	if !res.Failed {
		t.Fatal("a 2 MiB Server header was read as a response")
	}
	var buf bytes.Buffer
	obs := EmitEndpoint("b1", "v1", target, Identity(res, DefaultParams().BodyCapBytes))
	if err := wire.EncodeObservation(&buf, obs); err != nil {
		t.Fatal(err)
	}
	if buf.Len() > wire.MaxObservationLine {
		t.Fatalf("observation line is %d bytes, above wire.MaxObservationLine %d", buf.Len(), wire.MaxObservationLine)
	}
}

func TestNetExchangerReadsAResponseHeaderBelowTheCap(t *testing.T) {
	long := strings.Repeat("y", 32<<10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", long)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ap, err := netip.ParseAddrPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("parse listener addr: %v", err)
	}
	ex := NetExchanger{Params: DefaultParams(), control: allowAllControl}
	res := ex.Exchange(context.Background(), Target{Address: ap.Addr().String(), Port: ap.Port(), Scheme: "http"})
	if res.Failed {
		t.Fatalf("a 32 KiB Server header must still be read: %s", res.Err)
	}
	if res.Server != long {
		t.Fatalf("Server header not carried verbatim (%d bytes)", len(res.Server))
	}
}
