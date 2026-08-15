package connectoutcome

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/netip"
	"strings"
	"time"
)

// CertVersion is the tls-handshake leaf's Derivation version. The handshake that
// feeds the `certificate` facet is a STEP inside the `reachability` exchange, not
// a new dispatch path (ADR-0028) — it shares the `connect-outcome` Kind — but it
// is a distinct leaf with its own version, so a change to how the chain is read
// Breaks `certificate` timelines without touching `reachability` ones. It is gated
// by its own golden corpus (certcorpus), separately from the connect verdict.
const CertVersion = "tls-handshake/v1"

// TLSOutcome is the closed union the `certificate` facet's value space is built
// on (CONTEXT.md `Certificate`). Every value space is a closed union, never a
// record with optional fields, because a facet's measured negatives are values
// and must not collapse into "not measured".
type TLSOutcome string

const (
	// TLSPresented: the peer completed a TLS handshake and presented a
	// certificate chain. The value carries the chain, ordered leaf-first.
	TLSPresented TLSOutcome = "presented"
	// TLSRefused: the peer spoke TLS but accepted no candidate we offered — an
	// SSLv3-only or SNI-required listener that would otherwise be misfiled under
	// *not a TLS server*. A value, and distinct from NoTLS.
	TLSRefused TLSOutcome = "tls-refused"
	// NoTLS: nothing on the port spoke TLS at all. A value, and distinct from a
	// refusal — collapsing the two files a plain listener under *TLS server*.
	NoTLS TLSOutcome = "no-tls"
)

// HandshakeParams is the tls-handshake leaf's declared-parameter set: the fixed
// shape of the handshake that feeds the `certificate` facet. None is
// operator-configurable; they are recorded so that a change to any of them —
// adding an ALPN extension, sending SNI for the nameless endpoint, verifying the
// chain instead of recording it, or changing the fingerprint hash or chain order
// — is a declared-parameter change that moves the leaf's params digest and forces
// a CertVersion bump, gated by certcorpus's lock (golden-corpus.md §9).
type HandshakeParams struct {
	// SNIEqualsEndpointName: the handshake sends SNI equal to the Endpoint's
	// name, and none for the nameless endpoint (CONTEXT.md `Certificate`).
	SNIEqualsEndpointName bool `json:"sni_equals_endpoint_name"`
	// ALPN is empty — NO ALPN extension at all, so a listener refusing our
	// application protocols cannot cost us a chain we could otherwise read.
	ALPN string `json:"alpn"`
	// RecordNotVerify: verification is disabled — we record WHAT was presented,
	// not whether it validated, because a chain that fails to verify is still a
	// measured presentation.
	RecordNotVerify bool `json:"record_not_verify"`
	// FingerprintHash names the digest a chain is held by — sha-256 over DER.
	FingerprintHash string `json:"fingerprint_hash"`
	// ChainOrder records that the chain is the fingerprints leaf-first, since
	// order is on the wire.
	ChainOrder string `json:"chain_order"`
}

// DefaultHandshakeParams is the v1 shipped handshake shape — the CONTEXT.md
// `Certificate` table exactly.
func DefaultHandshakeParams() HandshakeParams {
	return HandshakeParams{
		SNIEqualsEndpointName: true,
		ALPN:                  "", // no ALPN extension
		RecordNotVerify:       true,
		FingerprintHash:       "sha-256",
		ChainOrder:            "leaf-first",
	}
}

// Digest is a stable content hash of the handshake params, used by certcorpus's
// lock to bind a declared-parameter change to a CertVersion bump.
func (p HandshakeParams) Digest() string {
	b, err := json.Marshal(p)
	if err != nil {
		panic("connectoutcome: marshal handshake params: " + err.Error())
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// HandshakeResult is the raw outcome of one TLS handshake step against a reached
// Service, for one Endpoint's server name. It is what the Handshaker reports; the
// leaf folds it to a certificate observation. A Chain is carried iff the outcome
// is TLSPresented, and it is the ordered list of fingerprints, leaf first, since
// order is on the wire.
type HandshakeResult struct {
	Outcome TLSOutcome
	Chain   []string
}

// Handshaker performs one TLS handshake against a Service that the connect
// already reached and reports the raw result. The handshake is deliberately
// stripped: it sends SNI equal to the Endpoint's name (serverName; empty for the
// nameless endpoint, which sends no SNI) and NO ALPN extension at all, so that a
// listener refusing our application protocols cannot cost us a chain we could
// otherwise read — ALPN belongs to `http-exchange`, not here (CONTEXT.md
// `Certificate`). The production adapter dials real TLS; the golden corpus scripts
// an in-process Handshaker so the fold runs hermetically with no network.
type Handshaker interface {
	Handshake(ctx context.Context, target netip.AddrPort, serverName string) HandshakeResult
}

// Fingerprint renders one certificate's identity: the lowercase hex SHA-256 of
// its DER bytes, prefixed `sha256:`. It is how a chain is held — by fingerprint,
// shared across every endpoint presenting the same certificate (CONTEXT.md
// `Certificate`).
func Fingerprint(der []byte) string {
	sum := sha256.Sum256(der)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// chainFingerprints renders a presented chain to fingerprints, leaf first — the
// order crypto/tls already reports PeerCertificates in.
func chainFingerprints(certs []*x509.Certificate) []string {
	if len(certs) == 0 {
		return nil
	}
	out := make([]string, 0, len(certs))
	for _, c := range certs {
		out = append(out, Fingerprint(c.Raw))
	}
	return out
}

// NetHandshaker is the production Handshaker: a real TLS handshake with SNI equal
// to the server name (omitted when empty), no ALPN, and verification disabled —
// we record WHAT was presented, not whether it validated, because a chain that
// fails to verify is still a measured presentation (CONTEXT.md `Certificate`). It
// is not exercised by the hermetic golden corpus, which pins the fold logic
// against a scripted Handshaker instead; its error classification is best-effort.
type NetHandshaker struct {
	// Timeout bounds the handshake dial. Zero uses a 3 s default.
	Timeout time.Duration
}

// Handshake implements Handshaker against the network. A completed handshake
// reads as TLSPresented with the peer's chain; a TLS-level rejection (an alert,
// or a protocol/version negotiation failure) as TLSRefused; a peer that never
// spoke TLS — a plaintext listener that resets or drops mid-handshake — as NoTLS.
func (n NetHandshaker) Handshake(ctx context.Context, target netip.AddrPort, serverName string) HandshakeResult {
	timeout := n.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cfg := &tls.Config{
		InsecureSkipVerify: true, // record what was presented, not whether it verified
		ServerName:         serverName,
		NextProtos:         nil, // no ALPN — CONTEXT.md `Certificate`
	}
	d := tls.Dialer{Config: cfg}
	conn, err := d.DialContext(dialCtx, "tcp", target.String())
	if err != nil {
		return HandshakeResult{Outcome: classifyTLSError(err)}
	}
	defer func() { _ = conn.Close() }()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return HandshakeResult{Outcome: NoTLS}
	}
	state := tlsConn.ConnectionState()
	chain := chainFingerprints(state.PeerCertificates)
	if len(chain) == 0 {
		// A completed handshake with no certificate is a TLS peer that offered
		// nothing we can hold — a refusal, not a plaintext port.
		return HandshakeResult{Outcome: TLSRefused}
	}
	return HandshakeResult{Outcome: TLSPresented, Chain: chain}
}

// classifyTLSError splits a failed handshake dial into its two negatives. A TLS
// record- or alert-level rejection is TLSRefused (the peer spoke TLS and turned us
// down); a reset, EOF or plaintext-looking failure is NoTLS (nothing there spoke
// TLS). The split is best-effort and unversioned by the corpus — the golden rows
// pin the fold, not this live classification.
func classifyTLSError(err error) TLSOutcome {
	var recordErr tls.RecordHeaderError
	if errors.As(err, &recordErr) {
		// A malformed TLS record header means the peer answered with something
		// that is not TLS at all.
		return NoTLS
	}
	var alertErr *tls.CertificateVerificationError
	if errors.As(err, &alertErr) {
		return TLSRefused
	}
	if errors.Is(err, io.EOF) {
		return NoTLS
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "first record does not look like a tls handshake"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "eof"):
		return NoTLS
	case strings.Contains(msg, "tls:"),
		strings.Contains(msg, "handshake failure"),
		strings.Contains(msg, "protocol version"),
		strings.Contains(msg, "no cipher suite"):
		return TLSRefused
	}
	// An unclassifiable transport failure is treated as no TLS spoken — the
	// conservative side, since it asserts no refusal we did not observe.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return NoTLS
	}
	return NoTLS
}
