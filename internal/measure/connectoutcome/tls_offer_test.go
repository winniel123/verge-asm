package connectoutcome

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/netip"
	"reflect"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/measure/tlsoffer"
)

func TestDefaultHandshakeParamsCarryTheOneCandidateSet(t *testing.T) {
	p := DefaultHandshakeParams()
	if p.MinVersion != tlsoffer.TLS10 || p.MaxVersion != tlsoffer.TLS13 {
		t.Errorf("declared versions %s..%s, want 1.0..1.3 (measurement-offers §1.2)", p.MinVersion, p.MaxVersion)
	}
	if !reflect.DeepEqual(p.CipherSuites, tlsoffer.Ciphers()) {
		t.Error("the certificate handshake must offer tls-acceptance's list, not its own (ADR-0030 §3)")
	}
	if p.ALPN != "" {
		t.Errorf("ALPN = %q, want none: the certificate handshake sends no ALPN extension (ADR-0030 §6)", p.ALPN)
	}
	if DefaultHandshakeParams().Digest() != p.Digest() {
		t.Error("handshake params digest is not stable")
	}
}

func tlsListener(t *testing.T, cfg *tls.Config) netip.AddrPort {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "legacy.test"},
		DNSNames:     []string{"legacy.test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Certificates = []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	if err != nil {
		t.Skipf("no loopback listener available: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				if tc, ok := c.(*tls.Conn); ok {
					_ = tc.Handshake()
				}
			}(c)
		}
	}()
	ap, err := netip.ParseAddrPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	return ap
}

func loopbackHandshaker() NetHandshaker {
	return NetHandshaker{Timeout: 2 * time.Second, realm: custody.ParseRealm([]string{"127.0.0.0/22"})}
}

func TestHandshakeReadsATLS10OnlyListener(t *testing.T) {
	// ADR-0025's forcing example: Go's default MinVersion 1.2 would file this as TLSRefused.
	target := tlsListener(t, &tls.Config{MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS10}) // #nosec G402 (test listener: the legacy box the wide offer exists to detect)
	res := loopbackHandshaker().Handshake(context.Background(), target, "legacy.test")
	if res.Outcome != TLSPresented {
		t.Fatalf("a TLS-1.0-only listener read %q, want presented", res.Outcome)
	}
	if len(res.Chain) != 1 {
		t.Fatalf("chain = %v, want the one self-signed leaf", res.Chain)
	}
}

func TestHandshakeOfferGovernsTheWire(t *testing.T) {
	// The declared parameter, not the library, decides what the legacy box reads as.
	target := tlsListener(t, &tls.Config{MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS10}) // #nosec G402 (test listener: the legacy box the wide offer exists to detect)
	h := loopbackHandshaker()
	h.Params = DefaultHandshakeParams()
	h.Params.MinVersion = tlsoffer.TLS12
	res := h.Handshake(context.Background(), target, "legacy.test")
	if res.Outcome != TLSRefused {
		t.Fatalf("a 1.2-floor offer against a 1.0-only listener read %q, want tls-refused", res.Outcome)
	}
}

func TestHandshakeReadsATLS13OnlyListener(t *testing.T) {
	target := tlsListener(t, &tls.Config{MinVersion: tls.VersionTLS13})
	res := loopbackHandshaker().Handshake(context.Background(), target, "legacy.test")
	if res.Outcome != TLSPresented {
		t.Fatalf("a TLS-1.3-only listener read %q, want presented", res.Outcome)
	}
}
