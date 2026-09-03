package queue

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"math/big"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/cryptobyte"

	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/wire"
)

// routeFetcher is a CTFetcher fake: each route matches a URL substring in order and returns a
// canned status and body. The first match wins, so more-specific routes come first.
type routeFetcher struct {
	routes []route
}

type route struct {
	sub    string
	status int
	body   []byte
}

func (f routeFetcher) Fetch(_ context.Context, url string) (int, []byte, error) {
	for _, r := range f.routes {
		if strings.Contains(url, r.sub) {
			return r.status, r.body, nil
		}
	}
	return 404, nil, nil
}

func testWorker(f CTFetcher) *Worker {
	return &Worker{log: log.New(io.Discard, "", 0), ctVerifyFetcher: f}
}

func serializeSCT(logID [32]byte, ts uint64, ext []byte) []byte {
	var b cryptobyte.Builder
	b.AddUint8(0)
	b.AddBytes(logID[:])
	b.AddUint64(ts)
	b.AddUint16LengthPrefixed(func(c *cryptobyte.Builder) { c.AddBytes(ext) })
	b.AddUint8(4)
	b.AddUint8(3)
	b.AddUint16LengthPrefixed(func(c *cryptobyte.Builder) { c.AddBytes([]byte{0x01, 0x02}) })
	return b.BytesOrPanic()
}

func leafIndexExtension(idx int64) []byte {
	var b cryptobyte.Builder
	b.AddUint8(0)
	b.AddUint16LengthPrefixed(func(c *cryptobyte.Builder) {
		var five [5]byte
		v := uint64(idx)
		for i := 4; i >= 0; i-- {
			five[i] = byte(v)
			v >>= 8
		}
		c.AddBytes(five[:])
	})
	return b.BytesOrPanic()
}

func selfSignedLeaf(t *testing.T) (der []byte, spki []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "verify.example"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     []string{"verify.example"},
	}
	der, err = x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return der, cert.RawSubjectPublicKeyInfo
}

func firstLog(t *testing.T, tiled bool) (scan.CTLog, [32]byte) {
	t.Helper()
	logs, err := scan.AllLogs()
	if err != nil {
		t.Fatal(err)
	}
	for _, lg := range logs {
		if lg.Tiled != tiled {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(lg.LogID)
		if err != nil || len(raw) != 32 {
			continue
		}
		var id [32]byte
		copy(id[:], raw)
		return lg, id
	}
	t.Skipf("embedded log list has no tiled=%v log", tiled)
	return scan.CTLog{}, [32]byte{}
}

func sthBody(treeSize int64, root []byte) []byte {
	b, _ := json.Marshal(map[string]any{"tree_size": treeSize, "sha256_root_hash": base64.StdEncoding.EncodeToString(root)})
	return b
}

func proofBody(index int64, path [][]byte) []byte {
	enc := make([]string, len(path))
	for i, p := range path {
		enc[i] = base64.StdEncoding.EncodeToString(p)
	}
	b, _ := json.Marshal(map[string]any{"leaf_index": index, "audit_path": enc})
	return b
}

func TestVerifyMaterialRFCLogged(t *testing.T) {
	lg, logID := firstLog(t, false)
	der, _ := selfSignedLeaf(t)
	ts := uint64(1700000000000)
	sct := serializeSCT(logID, ts, nil)

	// A tree of size 1: the leaf hash is the root and the audit path is empty.
	leafHash := scan.LeafHashX509(der, nil, ts)
	fetch := routeFetcher{routes: []route{
		{sub: "get-sth", status: 200, body: sthBody(1, leafHash)},
		{sub: "get-proof-by-hash", status: 200, body: proofBody(0, nil)},
	}}
	blob := wire.EncodeSCTCapture(wire.SCTCapture{TLSExt: [][]byte{sct}})
	logs, _ := scan.AllLogs()
	res := testWorker(fetch).verifyMaterial(context.Background(), logs, der, blob, nil)
	if res.Outcome != VerifyLogged {
		t.Fatalf("outcome = %v (%s), want logged; log=%s", res.Outcome, res.Reason, lg.Description)
	}
}

func TestVerifyMaterialRFCNotLogged(t *testing.T) {
	_, logID := firstLog(t, false)
	der, _ := selfSignedLeaf(t)
	ts := uint64(1700000000000)
	sct := serializeSCT(logID, ts, nil)
	fetch := routeFetcher{routes: []route{
		{sub: "get-sth", status: 200, body: sthBody(5, make([]byte, 32))},
		{sub: "get-proof-by-hash", status: 404, body: nil},
	}}
	blob := wire.EncodeSCTCapture(wire.SCTCapture{TLSExt: [][]byte{sct}})
	logs, _ := scan.AllLogs()
	res := testWorker(fetch).verifyMaterial(context.Background(), logs, der, blob, nil)
	if res.Outcome != VerifyNotLogged {
		t.Fatalf("outcome = %v (%s), want not-logged", res.Outcome, res.Reason)
	}
}

func TestVerifyMaterialUnreachableIsUnverifiable(t *testing.T) {
	_, logID := firstLog(t, false)
	der, _ := selfSignedLeaf(t)
	ts := uint64(1700000000000)
	sct := serializeSCT(logID, ts, nil)
	fetch := routeFetcher{routes: []route{
		{sub: "get-sth", status: 503, body: nil}, // the log is unreachable
	}}
	blob := wire.EncodeSCTCapture(wire.SCTCapture{TLSExt: [][]byte{sct}})
	logs, _ := scan.AllLogs()
	res := testWorker(fetch).verifyMaterial(context.Background(), logs, der, blob, nil)
	if res.Outcome != VerifyUnverifiable {
		t.Fatalf("outcome = %v (%s), want unverifiable — an unreachable log is never not-logged", res.Outcome, res.Reason)
	}
}

func TestVerifyMaterialNoSCTs(t *testing.T) {
	der, _ := selfSignedLeaf(t)
	logs, _ := scan.AllLogs()
	res := testWorker(routeFetcher{}).verifyMaterial(context.Background(), logs, der, nil, nil)
	if res.Outcome != VerifyUnverifiable {
		t.Fatalf("outcome = %v (%s), want unverifiable for a cert with no SCTs", res.Outcome, res.Reason)
	}
}

func TestVerifyMaterialTiledLogged(t *testing.T) {
	_, logID := firstLog(t, true)
	der, _ := selfSignedLeaf(t)
	ts := uint64(1700000000000)
	ext := leafIndexExtension(0)
	sct := serializeSCT(logID, ts, ext)

	leafHash := scan.LeafHashX509(der, ext, ts)
	// A checkpoint of a size-1 tree, and a partial level-0 hash tile holding our leaf hash.
	checkpoint := []byte("example.log\n1\n" + base64.StdEncoding.EncodeToString(make([]byte, 32)) + "\n\n— example sig\n")
	fetch := routeFetcher{routes: []route{
		{sub: "checkpoint", status: 200, body: checkpoint},
		{sub: "tile/0/", status: 200, body: leafHash},
	}}
	blob := wire.EncodeSCTCapture(wire.SCTCapture{TLSExt: [][]byte{sct}})
	logs, _ := scan.AllLogs()
	res := testWorker(fetch).verifyMaterial(context.Background(), logs, der, blob, nil)
	if res.Outcome != VerifyLogged {
		t.Fatalf("outcome = %v (%s), want logged", res.Outcome, res.Reason)
	}
}

func TestVerifyMaterialEmbeddedPrecertLogged(t *testing.T) {
	_, logID := firstLog(t, false)
	ts := uint64(1700000000000)
	sct := serializeSCT(logID, ts, nil)

	var list cryptobyte.Builder
	list.AddUint16LengthPrefixed(func(outer *cryptobyte.Builder) {
		outer.AddUint16LengthPrefixed(func(one *cryptobyte.Builder) { one.AddBytes(sct) })
	})
	octet, err := asn1.Marshal(list.BytesOrPanic())
	if err != nil {
		t.Fatal(err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	oidSCT := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 4, 2}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(7),
		Subject:               pkix.Name{CommonName: "embed.example"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		BasicConstraintsValid: true,
		ExtraExtensions:       []pkix.Extension{{Id: oidSCT, Value: octet}},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	spki := cert.RawSubjectPublicKeyInfo

	tbs, err := scan.PrecertTBS(der)
	if err != nil {
		t.Fatal(err)
	}
	leafHash := scan.LeafHashPrecert(scan.IssuerKeyHash(spki), tbs, nil, ts)
	fetch := routeFetcher{routes: []route{
		{sub: "get-sth", status: 200, body: sthBody(1, leafHash)},
		{sub: "get-proof-by-hash", status: 200, body: proofBody(0, nil)},
	}}
	logs, _ := scan.AllLogs()
	// No out-of-cert SCTs — only the embedded one — so the precert branch is the only path.
	res := testWorker(fetch).verifyMaterial(context.Background(), logs, der, nil, spki)
	if res.Outcome != VerifyLogged {
		t.Fatalf("outcome = %v (%s), want logged via embedded precert SCT", res.Outcome, res.Reason)
	}

	// Without the issuer key the embedded SCT cannot be verified: unverifiable, not not-logged.
	res = testWorker(fetch).verifyMaterial(context.Background(), logs, der, nil, nil)
	if res.Outcome != VerifyUnverifiable {
		t.Fatalf("outcome = %v (%s), want unverifiable without the issuer key", res.Outcome, res.Reason)
	}
}

func TestVerifyByFingerprintNotConfigured(t *testing.T) {
	w := &Worker{log: log.New(io.Discard, "", 0)} // no verify fetcher
	if _, err := w.VerifyByFingerprint(context.Background(), "sha256:abc"); err == nil {
		t.Fatal("expected an error when verification is not configured")
	}
}
