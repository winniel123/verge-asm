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

const extServerName uint16 = 0 // the server_name extension's type code (RFC 6066 §3)

type recordingHandshaker struct {
	serverNames []string
}

func (h *recordingHandshaker) Handshake(_ context.Context, _ netip.AddrPort, serverName string) co.HandshakeResult {
	h.serverNames = append(h.serverNames, serverName)
	return co.HandshakeResult{Outcome: co.NoTLS}
}

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

func TestClientHelloCarriesNoSNIExtension(t *testing.T) {
	// crypto/tls omits server_name entirely for an empty name, never sending an empty one.
	t.Run("no server name emits no extension", func(t *testing.T) {
		exts := clientHelloExtensions(t, captureClientHello(t, NoServerName))
		if hasExtension(exts, extServerName) {
			t.Errorf("ClientHello extensions = %v, want no server_name (0x%04x)", exts, extServerName)
		}
	})
	// The named control proves the parser finds the extension, so the empty case is a real absence.
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

func captureClientHello(t *testing.T, serverName string) []byte {
	t.Helper()
	// net.Pipe opens no socket, so the egress guard is not in play here.
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
	// Defers run LIFO, so the close must register after the wait or this deadlocks.
	defer func() { <-done }()
	defer func() { _ = serverConn.Close() }()

	if err := serverConn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	// A TLS record header: content type (1), legacy version (2), length (2) — RFC 8446 §5.1.
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
	// Skips the handshake header: type (1) and a 24-bit length (3) — RFC 8446 §4.
	return record[4:]
}

func clientHelloExtensions(t *testing.T, hello []byte) []uint16 {
	t.Helper()
	// The ClientHello prefix and its extension list are fixed by RFC 8446 §4.1.2.
	r := &byteReader{t: t, b: hello}
	r.skip(2)
	r.skip(32)
	r.skipVector(1)
	r.skipVector(2)
	r.skipVector(1)
	body := r.vector(2)

	var types []uint16
	ext := &byteReader{t: t, b: body}
	for len(ext.b) > 0 {
		types = append(types, binary.BigEndian.Uint16(ext.take(2)))
		ext.skipVector(2)
	}
	return types
}

type byteReader struct {
	t *testing.T
	b []byte
}

func (r *byteReader) take(n int) []byte {
	r.t.Helper()
	// A short read fails the test, so a parse bug never reads as a false absence.
	if len(r.b) < n {
		r.t.Fatalf("ClientHello truncated: want %d bytes, have %d", n, len(r.b))
	}
	out := r.b[:n]
	r.b = r.b[n:]
	return out
}

func (r *byteReader) skip(n int) { r.take(n) }

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
