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
	"github.com/winniel123/verge-asm/internal/measure/tlsoffer"
)

// A step inside the reachability exchange, sharing its Kind, but with its own timelines (ADR-0028).

const CertVersion = "tls-handshake/v5"

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
	SNIEqualsEndpointName bool     `json:"sni_equals_endpoint_name"`
	ALPN                  string   `json:"alpn"`
	RecordNotVerify       bool     `json:"record_not_verify"`
	FingerprintHash       string   `json:"fingerprint_hash"`
	ChainOrder            string   `json:"chain_order"`
	MinVersion            string   `json:"min_version"`
	MaxVersion            string   `json:"max_version"`
	CipherSuites          []string `json:"cipher_suites"`
}

func DefaultHandshakeParams() HandshakeParams {
	return HandshakeParams{
		SNIEqualsEndpointName: true,
		ALPN:                  "",
		RecordNotVerify:       true,
		FingerprintHash:       "sha-256",
		ChainOrder:            "leaf-first",
		// Wide on purpose: Go's default MinVersion 1.2 hides a TLS-1.0-only listener (ADR-0025).
		MinVersion: tlsoffer.TLS10,
		MaxVersion: tlsoffer.TLS13,
		// One list with tls-acceptance, so a widening Breaks both facets at once (ADR-0030 §3).
		CipherSuites: tlsoffer.Ciphers(),
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
	Unreachable bool // Plumbing no facet renders; only edge-fanout reads it (ADR-0151 §3).
	TimedOut    bool // Plumbing no facet renders; only the pacer reads it (ADR-0151 §2, #1710).
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
	// One chain serves many endpoints, so the fingerprint is its key (CONTEXT.md Certificate).
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
	Params  HandshakeParams
	realm   custody.Realm
}

func (n NetHandshaker) Handshake(ctx context.Context, target netip.AddrPort, serverName string) HandshakeResult {
	timeout := n.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	p := n.Params
	minVersion, maxVersion, ok := offeredVersions(p)
	if !ok {
		// An undeclared version would hand the offer back to the library default (ADR-0025).
		p = DefaultHandshakeParams()
		minVersion, maxVersion, _ = offeredVersions(p)
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
		// The declared set goes on the wire, never a library default (ADR-0025, #1680).
		MinVersion:   minVersion,
		MaxVersion:   maxVersion,
		CipherSuites: tlsoffer.CipherIDs(p.CipherSuites),
	}
	d := tls.Dialer{
		NetDialer: &net.Dialer{Control: custody.EgressGuard("connectoutcome", n.realm)},
		Config:    cfg,
	}
	conn, err := d.DialContext(dialCtx, "tcp", target.String())
	if err != nil {
		outcome, unreachable, timedOut := classifyDialError(err)
		return HandshakeResult{Outcome: outcome, Unreachable: unreachable, TimedOut: timedOut}
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
		chainCerts = append(chainCerts, ParseChainCert(c))
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

func offeredVersions(p HandshakeParams) (minVersion, maxVersion uint16, ok bool) {
	minVersion, okMin := tlsoffer.VersionID(p.MinVersion)
	maxVersion, okMax := tlsoffer.VersionID(p.MaxVersion)
	return minVersion, maxVersion, okMin && okMax
}

func issuerSPKI(chain []*x509.Certificate) []byte {
	// An embedded SCT's precert hash needs SHA-256 of the issuer SPKI (RFC 6962 §3.2, #878).
	if len(chain) < 2 {
		return nil
	}
	return chain[1].RawSubjectPublicKeyInfo
}

func ParseChainCert(c *x509.Certificate) ChainCert {
	// Store raw and derive at read: the four dark certificate rules run at read, not here (#712).

	// CheckSignatureFrom also refuses SHA-1 and applies CA policy, voiding the carve-out (#1426).
	selfSig := c.CheckSignature(c.SignatureAlgorithm, c.RawTBSCertificate, c.Signature) == nil
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

// The golden rows pin the fold, never this live split, so a change here is uncovered (ADR-0152 §3).

func classifyDialError(err error) (outcome TLSOutcome, unreachable, timedOut bool) {
	timedOut = dialTimedOut(err)
	var opErr *net.OpError
	// A connect-phase failure carries Op "dial", so the phase is read off the error, never guessed.
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return NoTLS, true, timedOut
	}
	var recordErr tls.RecordHeaderError
	if errors.As(err, &recordErr) {
		return NoTLS, false, timedOut
	}
	var alertErr *tls.CertificateVerificationError
	if errors.As(err, &alertErr) {
		return TLSRefused, false, timedOut
	}
	if errors.Is(err, io.EOF) {
		return NoTLS, false, timedOut
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "first record does not look like a tls handshake"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "eof"):
		return NoTLS, false, timedOut
	case strings.Contains(msg, "tls:"),
		strings.Contains(msg, "handshake failure"),
		strings.Contains(msg, "protocol version"),
		strings.Contains(msg, "no cipher suite"):
		return TLSRefused, false, timedOut
	}
	// The unclassifiable case asserts no refusal we did not observe.
	return NoTLS, false, timedOut
}

func dialTimedOut(err error) bool {
	// crypto/tls hands back the bare context error when its deadline interrupts the handshake.
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	// A poll deadline surfaces os.ErrDeadlineExceeded, which carries no Is chain to the context.
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}
