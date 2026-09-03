package connectoutcome

import (
	"context"
	"crypto/dsa" //nolint:staticcheck // SA1019: DSA keys are legacy but a measured cert may still present one; we read its params to fire certificate-weak-key-or-signature.
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
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

	"github.com/winniel123/verge-asm/internal/custody"
)

// A step inside the reachability exchange, sharing its Kind, but with its own timelines (ADR-0028).

const CertVersion = "tls-handshake/v3"

// A closed union, never optional fields: a measured negative is a value, not "not measured".

type TLSOutcome string

// Collapsing the two negatives files an SNI-required listener as no TLS server (CONTEXT.md).

const (
	TLSPresented TLSOutcome = "presented"
	TLSRefused   TLSOutcome = "tls-refused"
	NoTLS        TLSOutcome = "no-tls"
)

// None is operator-configurable; changing one moves the params digest and forces a version bump.

type HandshakeParams struct {
	SNIEqualsEndpointName bool   `json:"sni_equals_endpoint_name"`
	ALPN                  string `json:"alpn"`
	RecordNotVerify       bool   `json:"record_not_verify"`
	FingerprintHash       string `json:"fingerprint_hash"`
	ChainOrder            string `json:"chain_order"`
}

func DefaultHandshakeParams() HandshakeParams {
	return HandshakeParams{
		SNIEqualsEndpointName: true,
		ALPN:                  "",
		RecordNotVerify:       true,
		FingerprintHash:       "sha-256",
		ChainOrder:            "leaf-first",
	}
}

func (p HandshakeParams) Digest() string {
	b, err := json.Marshal(p)
	if err != nil {
		panic("connectoutcome: marshal handshake params: " + err.Error())
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Chain order is the wire order, leaf first, so it is a fact and never a sort we chose.

type HandshakeResult struct {
	Outcome     TLSOutcome
	Chain       []string
	NotAfter    time.Time
	Issuer      string
	Algorithm   string
	NotBefore   time.Time
	SANDNS      []string
	SANIP       []string
	ChainCerts  []ChainCert
	LeafDER     []byte
	SCTsTLSExt  [][]byte
	OCSPStaple  []byte
	IssuerSPKI  []byte
	Unreachable bool // Only edge-fanout reads it; certificate folds an unreachable dial into no-tls.
}

// A self-signature check needs parsed key bytes, so it is the one datum computed in-leaf (#712).

type ChainCert struct {
	Subject               string
	Issuer                string
	SelfSignatureVerifies *bool
	KeyAlg                string
	KeyBits               int
	KeyParamN             int
	SigDigest             string
}

type Handshaker interface {
	Handshake(ctx context.Context, target netip.AddrPort, serverName string) HandshakeResult
}

func Fingerprint(der []byte) string {
	// A chain is held by fingerprint, shared by every endpoint presenting it (CONTEXT.md Certificate).
	sum := sha256.Sum256(der)
	return "sha256:" + hex.EncodeToString(sum[:])
}

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

type NetHandshaker struct {
	Timeout time.Duration
}

func (n NetHandshaker) Handshake(ctx context.Context, target netip.AddrPort, serverName string) HandshakeResult {
	timeout := n.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	if !target.Addr().IsValid() {
		return HandshakeResult{Outcome: NoTLS, Unreachable: true}
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cfg := &tls.Config{
		InsecureSkipVerify: true, // #nosec G402 (accepted: certificate-measurement probe — records the presented chain incl. self-signed/expired; verification is a declared-param OFF by design, digest-locked to CertVersion. See HandshakeParams.RecordNotVerify.)
		ServerName:         serverName,
		// No ALPN at all, so a listener refusing our protocols cannot cost us a readable chain.
		NextProtos: nil,
	}
	d := tls.Dialer{
		NetDialer: &net.Dialer{Control: custody.EgressGuard("connectoutcome")},
		Config:    cfg,
	}
	conn, err := d.DialContext(dialCtx, "tcp", target.String())
	if err != nil {
		outcome, unreachable := classifyDialError(err)
		return HandshakeResult{Outcome: outcome, Unreachable: unreachable}
	}
	defer func() { _ = conn.Close() }()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return HandshakeResult{Outcome: NoTLS}
	}
	state := tlsConn.ConnectionState()
	chain := chainFingerprints(state.PeerCertificates)
	// A completed handshake with no certificate is a refusal, never a plaintext port.
	if len(chain) == 0 {
		return HandshakeResult{Outcome: TLSRefused}
	}
	// Reading more of the presented chain sends nothing new, so it moves no params digest (#704).
	leaf := state.PeerCertificates[0]
	sanIP := make([]string, 0, len(leaf.IPAddresses))
	for _, ip := range leaf.IPAddresses {
		sanIP = append(sanIP, ip.String())
	}
	chainCerts := make([]ChainCert, 0, len(state.PeerCertificates))
	for _, c := range state.PeerCertificates {
		chainCerts = append(chainCerts, parseChainCert(c))
	}
	return HandshakeResult{
		Outcome:   TLSPresented,
		Chain:     chain,
		NotAfter:  leaf.NotAfter,
		Issuer:    leaf.Issuer.String(),
		Algorithm: leaf.SignatureAlgorithm.String(),
		NotBefore: leaf.NotBefore,
		// dNSName SANs ride verbatim; wildcards are never expanded for the read-time rule (#704).
		SANDNS:     leaf.DNSNames,
		SANIP:      sanIP,
		ChainCerts: chainCerts,
		LeafDER:    leaf.Raw,
		SCTsTLSExt: state.SignedCertificateTimestamps,
		OCSPStaple: state.OCSPResponse,
		IssuerSPKI: issuerSPKI(state.PeerCertificates),
	}
}

func issuerSPKI(chain []*x509.Certificate) []byte {
	// An embedded SCT's precert hash needs SHA-256 of the issuer SPKI (RFC 6962 §3.2, #878).
	if len(chain) < 2 {
		return nil
	}
	return chain[1].RawSubjectPublicKeyInfo
}

func parseChainCert(c *x509.Certificate) ChainCert {
	// Store raw and derive at read: the four dark certificate rules run at read, not here (#712).
	selfSig := c.CheckSignatureFrom(c) == nil
	cc := ChainCert{
		Subject:               c.Subject.String(),
		Issuer:                c.Issuer.String(),
		SelfSignatureVerifies: &selfSig,
		SigDigest:             sigDigestName(c.SignatureAlgorithm),
	}
	// It never fails: an unknown algorithm reads as not-weak rather than firing a rule (#712).
	switch pk := c.PublicKey.(type) {
	case *rsa.PublicKey:
		cc.KeyAlg = "RSA"
		cc.KeyBits = pk.N.BitLen()
	case *ecdsa.PublicKey:
		cc.KeyAlg = "ECDSA"
		cc.KeyBits = pk.Curve.Params().BitSize
	case *dsa.PublicKey:
		cc.KeyAlg = "DSA"
		cc.KeyBits = pk.P.BitLen()
		cc.KeyParamN = pk.Q.BitLen()
	case ed25519.PublicKey:
		cc.KeyAlg = "Ed25519"
	default:
		cc.KeyAlg = c.PublicKeyAlgorithm.String()
	}
	return cc
}

func sigDigestName(a x509.SignatureAlgorithm) string {
	// The read-time deny-list is {MD5, SHA-1}, so the datum is the digest and not the OID (#712).
	switch a {
	case x509.MD5WithRSA:
		return "MD5"
	case x509.SHA1WithRSA, x509.DSAWithSHA1, x509.ECDSAWithSHA1:
		return "SHA-1"
	case x509.SHA256WithRSA, x509.DSAWithSHA256, x509.ECDSAWithSHA256, x509.SHA256WithRSAPSS:
		return "SHA-256"
	case x509.SHA384WithRSA, x509.ECDSAWithSHA384, x509.SHA384WithRSAPSS:
		return "SHA-384"
	case x509.SHA512WithRSA, x509.ECDSAWithSHA512, x509.SHA512WithRSAPSS:
		return "SHA-512"
	case x509.PureEd25519:
		return "Ed25519"
	default:
		return ""
	}
}

// Best-effort and unversioned: the golden rows pin the fold, never this live classification.

func classifyDialError(err error) (outcome TLSOutcome, unreachable bool) {
	var opErr *net.OpError
	// A connect-phase failure carries Op "dial", so the phase is read off the error, never guessed.
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return NoTLS, true
	}
	var recordErr tls.RecordHeaderError
	if errors.As(err, &recordErr) {
		return NoTLS, false
	}
	var alertErr *tls.CertificateVerificationError
	if errors.As(err, &alertErr) {
		return TLSRefused, false
	}
	if errors.Is(err, io.EOF) {
		return NoTLS, false
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "first record does not look like a tls handshake"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "eof"):
		return NoTLS, false
	case strings.Contains(msg, "tls:"),
		strings.Contains(msg, "handshake failure"),
		strings.Contains(msg, "protocol version"),
		strings.Contains(msg, "no cipher suite"):
		return TLSRefused, false
	}
	// The unclassifiable case takes the conservative side: it asserts no refusal we did not observe.
	return NoTLS, false
}
