package edgefanout

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"io"
	"net"
	"net/netip"
	"testing"
	"time"

	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

// extServerName is the server_name extension's type code (RFC 6066 §3). Its ABSENCE
// from the ClientHello is what makes the edge answer with its DEFAULT certificate,
// which is the whole measurement.
const extServerName uint16 = 0

// recordingHandshaker records the server name it was handed and answers a fixed result.
type recordingHandshaker struct {
	serverNames []string
}

func (h *recordingHandshaker) Handshake(_ context.Context, _ netip.AddrPort, serverName string) co.HandshakeResult {
	h.serverNames = append(h.serverNames, serverName)
	return co.HandshakeResult{Outcome: co.NoTLS}
}

// TestNetHandshakerSendsNoServerName pins the one handshake argument this leaf
// controls: the production path always hands connectoutcome's dial an EMPTY server
// name. An accidental name here would measure a tenant's certificate rather than the
// edge's default one, and the fan-out reduction would count the wrong SAN set.
func TestNetHandshakerSendsNoServerName(t *testing.T) {
	rec := &recordingHandshaker{}
	got := NetHandshaker{inner: rec}.Handshake(context.Background(), netip.MustParseAddrPort("198.51.100.7:443"))

	if len(rec.serverNames) != 1 {
		t.Fatalf("handshakes = %d, want exactly one connect per address", len(rec.serverNames))
	}
	if rec.serverNames[0] != NoServerName {
		t.Errorf("server name = %q, want %q (no SNI)", rec.serverNames[0], NoServerName)
	}
	if got.Outcome != NoTLS {
		t.Errorf("outcome = %q, want %q — the inner result must fold, not be re-decided", got.Outcome, NoTLS)
	}
}

// TestClientHelloCarriesNoSNIExtension is the byte-level proof behind the leaf's
// no-SNI claim: crypto/tls handed an empty ServerName emits a ClientHello with no
// server_name extension AT ALL — not an empty one. The named control proves the parser
// finds the extension when it is really there, so the empty case is a real absence and
// not a broken assertion.
func TestClientHelloCarriesNoSNIExtension(t *testing.T) {
	t.Run("no server name emits no extension", func(t *testing.T) {
		exts := clientHelloExtensions(t, captureClientHello(t, NoServerName))
		if hasExtension(exts, extServerName) {
			t.Errorf("ClientHello extensions = %v, want no server_name (0x%04x)", exts, extServerName)
		}
	})
	t.Run("a server name emits the extension", func(t *testing.T) {
		exts := clientHelloExtensions(t, captureClientHello(t, "edge.example.com"))
		if !hasExtension(exts, extServerName) {
			t.Errorf("ClientHello extensions = %v, want server_name (0x%04x) present", exts, extServerName)
		}
	})
}

func hasExtension(exts []uint16, want uint16) bool {
	for _, e := range exts {
		if e == want {
			return true
		}
	}
	return false
}

// captureClientHello drives one crypto/tls client handshake over an in-memory pipe and
// returns the ClientHello handshake message it wrote. The peer never answers, so the
// handshake fails after the first flight — which is all this test reads. No socket is
// opened, so the egress guard is not in play here.
func captureClientHello(t *testing.T, serverName string) []byte {
	t.Helper()
	clientConn, serverConn := net.Pipe()

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = clientConn.Close() }()
		// #nosec G402 -- this is the measurement stance under test: a client with no
		// server name must set InsecureSkipVerify or crypto/tls refuses to build the
		// ClientHello at all, and nothing here validates a peer.
		cfg := &tls.Config{ServerName: serverName, InsecureSkipVerify: true}
		_ = tls.Client(clientConn, cfg).Handshake()
	}()
	// Defers run last-in-first-out, so the close must be REGISTERED after the wait to
	// RUN before it. The client blocks reading a reply this test never sends; closing
	// the pipe first is what unblocks it, and waiting first would deadlock.
	defer func() { <-done }()
	defer func() { _ = serverConn.Close() }()

	if err := serverConn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	// One TLS record: content type (1), legacy version (2), length (2), then the
	// handshake message.
	header := make([]byte, 5)
	if _, err := io.ReadFull(serverConn, header); err != nil {
		t.Fatalf("read record header: %v", err)
	}
	if header[0] != 0x16 {
		t.Fatalf("record content type = 0x%02x, want 0x16 (handshake)", header[0])
	}
	record := make([]byte, binary.BigEndian.Uint16(header[3:5]))
	if _, err := io.ReadFull(serverConn, record); err != nil {
		t.Fatalf("read record body: %v", err)
	}
	if len(record) < 4 || record[0] != 0x01 {
		t.Fatalf("handshake message type = 0x%02x, want 0x01 (ClientHello)", record[0])
	}
	// Skip the handshake message header: type (1) and a 24-bit length (3).
	return record[4:]
}

// clientHelloExtensions parses a ClientHello body down to its extension type codes. It
// walks the fixed prefix — legacy version, random, session id, cipher suites,
// compression methods — then reads the extension list (RFC 8446 §4.1.2).
func clientHelloExtensions(t *testing.T, hello []byte) []uint16 {
	t.Helper()
	r := &byteReader{t: t, b: hello}
	r.skip(2)           // legacy_version
	r.skip(32)          // random
	r.skipVector(1)     // legacy_session_id
	r.skipVector(2)     // cipher_suites
	r.skipVector(1)     // legacy_compression_methods
	body := r.vector(2) // extensions

	var types []uint16
	ext := &byteReader{t: t, b: body}
	for len(ext.b) > 0 {
		types = append(types, binary.BigEndian.Uint16(ext.take(2)))
		ext.skipVector(2)
	}
	return types
}

// byteReader walks a ClientHello, failing the test on a short read rather than
// panicking, so a parse bug reads as a test failure and never as a false absence.
type byteReader struct {
	t *testing.T
	b []byte
}

func (r *byteReader) take(n int) []byte {
	r.t.Helper()
	if len(r.b) < n {
		r.t.Fatalf("ClientHello truncated: want %d bytes, have %d", n, len(r.b))
	}
	out := r.b[:n]
	r.b = r.b[n:]
	return out
}

func (r *byteReader) skip(n int) { r.take(n) }

// vector reads a length-prefixed vector whose length field is lenBytes wide (1 or 2)
// and returns its contents.
func (r *byteReader) vector(lenBytes int) []byte {
	r.t.Helper()
	prefix := r.take(lenBytes)
	n := 0
	for _, b := range prefix {
		n = n<<8 | int(b)
	}
	return r.take(n)
}

func (r *byteReader) skipVector(lenBytes int) { r.vector(lenBytes) }
