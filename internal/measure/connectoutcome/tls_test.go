package connectoutcome

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/netip"
	"testing"

	"github.com/winniel123/verge-asm/internal/measure/blanketdiscrim"
	"github.com/winniel123/verge-asm/internal/wire"
)

type scriptHandshaker struct {
	byEndpoint map[string]HandshakeResult
	seen       map[string]string
}

func (s *scriptHandshaker) Handshake(_ context.Context, t netip.AddrPort, serverName string) HandshakeResult {
	if s.seen == nil {
		s.seen = map[string]string{}
	}
	key := EndpointKey(serverName, t, "tcp")
	s.seen[key] = serverName
	return s.byEndpoint[key]
}

func ndjsonLines(t *testing.T, b []byte) []wire.Observation {
	t.Helper()
	var out []wire.Observation
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		var o wire.Observation
		if err := json.Unmarshal(sc.Bytes(), &o); err != nil {
			t.Fatalf("decode observation line: %v", err)
		}
		out = append(out, o)
	}
	return out
}

func TestEndpointKeyNamelessAndNamed(t *testing.T) {
	target := ap("198.51.100.1:443")
	if got := EndpointKey("", target, "tcp"); got != "@198.51.100.1:443/tcp" {
		t.Errorf("nameless endpoint key = %q, want @198.51.100.1:443/tcp", got)
	}
	if got := EndpointKey("api.example.com", target, "tcp"); got != "api.example.com@198.51.100.1:443/tcp" {
		t.Errorf("named endpoint key = %q", got)
	}
	if EndpointKey("", target, "tcp") == EndpointKey("api.example.com", target, "tcp") {
		t.Error("a nameless and a named endpoint on one Service must not collide")
	}
}

func TestEmitCertificateValueSpace(t *testing.T) {
	target := ap("198.51.100.1:443")
	cases := []struct {
		name string
		res  HandshakeResult
		want string
	}{
		{"presented", HandshakeResult{Outcome: TLSPresented, Chain: []string{"sha256:leaf", "sha256:ca"}},
			`{"outcome":"presented","chain":["sha256:leaf","sha256:ca"]}`},
		{"tls-refused", HandshakeResult{Outcome: TLSRefused}, `{"outcome":"tls-refused"}`},
		{"no-tls", HandshakeResult{Outcome: NoTLS}, `{"outcome":"no-tls"}`},
	}
	for _, c := range cases {
		obs := EmitCertificate("b1", "v1", target, "api.example.com", c.res)
		if obs.Facet != FacetCertificate {
			t.Errorf("%s: facet = %q, want certificate", c.name, obs.Facet)
		}
		if obs.Subject != "api.example.com@198.51.100.1:443/tcp" {
			t.Errorf("%s: subject = %q", c.name, obs.Subject)
		}
		if got := string(obs.Data); got != c.want {
			t.Errorf("%s: value = %s, want %s", c.name, got, c.want)
		}
	}
}

func TestEmitCertificateMaterial(t *testing.T) {
	target := ap("198.51.100.1:443")
	leafDER := []byte("\x30\x82fake-leaf-der")
	sct1, sct2 := []byte("sct-one"), []byte("sct-two")
	ocsp := []byte("ocsp-staple")

	presented := HandshakeResult{
		Outcome:    TLSPresented,
		Chain:      []string{Fingerprint(leafDER), "sha256:ca"},
		LeafDER:    leafDER,
		SCTsTLSExt: [][]byte{sct1, sct2},
		OCSPStaple: ocsp,
	}
	obs := EmitCertificate("b1", "v1", target, "api.example.com", presented)

	// ADR-0027's fence: no CT input may feed the facet value, so capture moves no byte.
	bare := EmitCertificate("b1", "v1", target, "api.example.com",
		HandshakeResult{Outcome: TLSPresented, Chain: presented.Chain})
	if string(obs.Data) != string(bare.Data) {
		t.Fatalf("capture changed the facet value:\n got  %s\n want %s", obs.Data, bare.Data)
	}

	if obs.CertMaterial == nil {
		t.Fatal("presented handshake with leaf DER carried no CertMaterial")
	}
	if obs.CertMaterial.Fingerprint != Fingerprint(leafDER) {
		t.Errorf("material fingerprint = %q, want %q", obs.CertMaterial.Fingerprint, Fingerprint(leafDER))
	}
	if obs.CertMaterial.Fingerprint != presented.Chain[0] {
		t.Errorf("material fingerprint %q != chain leaf %q", obs.CertMaterial.Fingerprint, presented.Chain[0])
	}
	if !bytes.Equal(obs.CertMaterial.DER, leafDER) {
		t.Errorf("material DER = %q, want %q", obs.CertMaterial.DER, leafDER)
	}
	capt, err := wire.DecodeSCTCapture(obs.CertMaterial.SCTs)
	if err != nil {
		t.Fatalf("DecodeSCTCapture: %v", err)
	}
	if len(capt.TLSExt) != 2 || !bytes.Equal(capt.TLSExt[0], sct1) || !bytes.Equal(capt.TLSExt[1], sct2) {
		t.Errorf("captured TLS-ext SCTs = %v, want [%q %q]", capt.TLSExt, sct1, sct2)
	}
	if !bytes.Equal(capt.OCSP, ocsp) {
		t.Errorf("captured OCSP staple = %q, want %q", capt.OCSP, ocsp)
	}

	for _, c := range []struct {
		name string
		res  HandshakeResult
	}{
		{"tls-refused", HandshakeResult{Outcome: TLSRefused}},
		{"no-tls", HandshakeResult{Outcome: NoTLS}},
		{"presented-no-der", HandshakeResult{Outcome: TLSPresented, Chain: []string{"sha256:leaf"}}},
	} {
		if m := EmitCertificate("b1", "v1", target, "api.example.com", c.res).CertMaterial; m != nil {
			t.Errorf("%s: carried CertMaterial %+v, want nil", c.name, m)
		}
	}
}

func TestRunExchangeRidesReachability(t *testing.T) {
	scope := Scope{
		Vantage:   "v1",
		Addresses: []string{"198.51.100.10", "198.51.100.11"},
		TCPPorts:  []uint16{443},
		Names:     []string{"api.example.com"},
		Profile:   DefaultProfile(),
	}
	conn := &scriptConnector{seq: map[netip.AddrPort][]ConnResult{
		ap("198.51.100.10:443"): {ConnOpen},
		ap("198.51.100.11:443"): {ConnRefused},
	}}
	hs := &scriptHandshaker{byEndpoint: map[string]HandshakeResult{
		"api.example.com@198.51.100.10:443/tcp": {Outcome: TLSPresented, Chain: []string{"sha256:leaf"}},
	}}

	var buf bytes.Buffer
	if err := RunExchange(context.Background(), conn, hs, blanketdiscrim.FixedPorts{}, "b1", scope, &buf); err != nil {
		t.Fatal(err)
	}
	var certs, reach int
	for _, o := range ndjsonLines(t, buf.Bytes()) {
		switch o.Facet {
		case FacetReachability:
			reach++
		case FacetCertificate:
			certs++
			if o.Subject != "api.example.com@198.51.100.10:443/tcp" {
				t.Errorf("certificate emitted for the wrong endpoint: %q", o.Subject)
			}
		}
	}
	if reach != 2 {
		t.Errorf("want 2 reachability observations (one per Service), got %d", reach)
	}
	if certs != 1 {
		t.Errorf("want 1 certificate observation (only the reached Service), got %d", certs)
	}
	if hs.seen["api.example.com@198.51.100.10:443/tcp"] != "api.example.com" {
		t.Error("the handshake must send SNI equal to the Endpoint name")
	}
	if _, probed := hs.seen["api.example.com@198.51.100.11:443/tcp"]; probed {
		t.Error("a not-reached Service must not be handshaked")
	}
}

func TestRunExchangeNamelessEndpoint(t *testing.T) {
	scope := Scope{
		Vantage:   "v1",
		Addresses: []string{"198.51.100.10"},
		TCPPorts:  []uint16{443},
		Profile:   DefaultProfile(),
	}
	conn := &scriptConnector{seq: map[netip.AddrPort][]ConnResult{
		ap("198.51.100.10:443"): {ConnOpen},
	}}
	hs := &scriptHandshaker{byEndpoint: map[string]HandshakeResult{
		"@198.51.100.10:443/tcp": {Outcome: NoTLS},
	}}
	var buf bytes.Buffer
	if err := RunExchange(context.Background(), conn, hs, blanketdiscrim.FixedPorts{}, "b1", scope, &buf); err != nil {
		t.Fatal(err)
	}
	if got, ok := hs.seen["@198.51.100.10:443/tcp"]; !ok || got != "" {
		t.Errorf("nameless endpoint must be handshaked with empty SNI, seen=%v", hs.seen)
	}
}

func TestFingerprintStable(t *testing.T) {
	sum := sha256.Sum256([]byte("der-bytes"))
	want := "sha256:" + hex.EncodeToString(sum[:])
	if got := Fingerprint([]byte("der-bytes")); got != want {
		t.Errorf("Fingerprint = %q, want %q", got, want)
	}
}
