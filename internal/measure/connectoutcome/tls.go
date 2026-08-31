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

// CertVersion is the tls-handshake leaf's Derivation version. The handshake that
// feeds the `certificate` facet is a STEP inside the `reachability` exchange, not
// a new dispatch path (ADR-0028) — it shares the `connect-outcome` Kind — but it
// is a distinct leaf with its own version, so a change to how the chain is read
// Breaks `certificate` timelines without touching `reachability` ones. It is gated
// by its own golden corpus (certcorpus), separately from the connect verdict.
//
// v2 adds the leaf certificate's `not_after` (RFC3339) to the presented value — a
// value-shape change that Breaks the old `certificate` timelines by design
// (ADR-0082): the datum the Dashboard "Certs expiring ≤30d" stat reads was never
// captured under v1 (SPEC-CHANGE.md collision #8, #464).
//
// v3 adds the leaf's `not_before` + leaf SANs (`san_dns`/`san_ip`) and per-link
// parsed chain facts (`chain_certs`: key params, sig digest, self-sig-verifies,
// subject/issuer DNs) so the four dark certificate rules — certificate-not-yet-valid,
// certificate-hostname-san-mismatch, certificate-weak-key-or-signature and
// certificate-self-signed — derive AT READ instead of rendering not-evaluable (store-
// raw / derive-at-read, P0.10b, #704). It is a value-shape change that Breaks the old
// `certificate` timelines by design. HandshakeParams is UNCHANGED — nothing new is
// sent, negotiated or verified (RecordNotVerify stays true) — so the params digest is
// unmoved; only the read of the already-presented chain widened.
const CertVersion = "tls-handshake/v3"

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
//
// NotAfter is the leaf certificate's (chain[0]) expiry, carried iff the outcome is
// TLSPresented — the two negatives leave it zero. The leaf folds it to `not_after`
// (RFC3339) on the presented value; a zero NotAfter renders no key (CONTEXT.md
// `Certificate`, SPEC-CHANGE.md collision #8).
type HandshakeResult struct {
	Outcome  TLSOutcome
	Chain    []string
	NotAfter time.Time
	// Issuer and Algorithm are the leaf's parsed identity, read off chain[0] on a
	// presented handshake — the issuer distinguished name and the signature
	// algorithm the leaf certificate carries. Both are carried only on TLSPresented
	// (the negatives leave them empty); the fold drops an empty via omitempty, so a
	// negative outcome and a pre-parse span carry neither key (SPEC-CHANGE.md
	// collision #22c, #581). They feed the Asset detail's TLS-certificate card.
	Issuer    string
	Algorithm string
	// NotBefore is the leaf's (chain[0]) validity floor, carried iff TLSPresented; the
	// fold renders it `not_before` (RFC3339), a zero value rendering no key. It lights
	// certificate-not-yet-valid at read (P0.10b, #704).
	NotBefore time.Time
	// SANDNS / SANIP are the leaf's Subject Alternative Names — the dNSName entries
	// verbatim (wildcards NOT expanded) and the iPAddress entries in canonical string
	// form — carried iff TLSPresented. certificate-hostname-san-mismatch reads san_dns
	// at read (P0.10b, #704).
	SANDNS []string
	SANIP  []string
	// ChainCerts is the per-link parsed facts, leaf-first and index-aligned with Chain,
	// carried iff TLSPresented. Each link's key params, signature digest, self-signature
	// verification and subject/issuer DNs feed certificate-weak-key-or-signature and
	// certificate-self-signed at read (P0.10b, #704).
	ChainCerts []ChainCert
	// LeafDER is the leaf certificate's DER bytes (chain[0].Raw), carried iff TLSPresented.
	// It is the raw material the certificate_material side store holds — its sha-256 is the
	// leaf fingerprint — and it carries any SCTs embedded in the cert. It NEVER feeds the
	// facet value (ADR-0027, spec §5.3); it rides the observation as CertMaterial for a
	// side-store write only. Reading it is a read-only widening of the presented chain, so
	// it moves no params digest and needs no CertVersion bump (see the v3 note above).
	LeafDER []byte
	// SCTsTLSExt is the SCTs the peer delivered in the TLS handshake extension
	// (ConnectionState.SignedCertificateTimestamps), each a serialized SCT, carried iff
	// TLSPresented. Out-of-cert SCT material captured for verification (#878); never fed to
	// the facet value.
	SCTsTLSExt [][]byte
	// OCSPStaple is the raw stapled OCSP response (ConnectionState.OCSPResponse), carried
	// iff TLSPresented and non-empty. It may carry SCTs in an extension; captured verbatim
	// for verification (#878), never fed to the facet value.
	OCSPStaple []byte
	// IssuerSPKI is the issuer certificate's SubjectPublicKeyInfo DER (chain[1]), carried iff
	// TLSPresented and the peer presented an issuer above the leaf. Verification of an EMBEDDED
	// SCT hashes the precertificate, whose leaf hash carries issuer_key_hash = SHA-256(issuer
	// SPKI) (RFC 6962 §3.2); the leaf alone does not carry the issuer's key, so it is captured
	// here beside the leaf for the certificate_material side store (#878). Never fed to the
	// facet value; nil for a lone self-signed leaf.
	IssuerSPKI []byte
}

// ChainCert is one presented link's parsed facts, read off the DER at handshake time
// (the exported mirror of the fold's chainCert). self_sig_verifies is the ONE datum
// computed in-leaf — a raw crypto fact needing the parsed key bytes, unavailable at
// read (T-leaf #712 §0). The rest are raw reads the four dark rules derive verdicts
// from at read.
type ChainCert struct {
	Subject               string // this cert's subject DN
	Issuer                string // this cert's issuer DN
	SelfSignatureVerifies *bool  // signature validates against THIS cert's own key
	KeyAlg                string // "RSA"|"ECDSA"|"DSA"|"Ed25519"|"Ed448"|raw OID
	KeyBits               int    // RSA nlen | ECDSA len(n) | DSA L
	KeyParamN             int    // DSA subgroup N (bits); 0 for non-DSA
	SigDigest             string // digest name: "MD5"|"SHA-1"|"SHA-256"|…
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
	// A target must be a valid literal IP: the leaf handshakes only pre-validated
	// addresses. Reject an invalid target at entry, backed by the socket-level
	// egress guard on the dialer below so a non-globally-reachable literal fails
	// closed even if that invariant is ever broken upstream (#743).
	if !target.Addr().IsValid() {
		return HandshakeResult{Outcome: NoTLS}
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cfg := &tls.Config{
		InsecureSkipVerify: true, // #nosec G402 (accepted: certificate-measurement probe — records the presented chain incl. self-signed/expired; verification is a declared-param OFF by design, digest-locked to CertVersion. See HandshakeParams.RecordNotVerify.)
		ServerName:         serverName,
		NextProtos:         nil, // no ALPN — CONTEXT.md `Certificate`
	}
	// custody.EgressGuard is the shared Control-hook backstop (delivery, resolutionwalk):
	// it refuses the socket when the resolved address is non-globally-reachable (#743).
	d := tls.Dialer{
		NetDialer: &net.Dialer{Control: custody.EgressGuard("connectoutcome")},
		Config:    cfg,
	}
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
	// The leaf (chain[0]) is already in hand for fingerprinting; read its NotAfter
	// here so the presented value can carry the expiry the certs-expiring stat reads
	// (SPEC-CHANGE.md collision #8). InsecureSkipVerify does not withhold the chain.
	// The same leaf carries its parsed identity — the issuer DN and the signature
	// algorithm — which the presented value carries for the Asset detail's cert card
	// (#22c, #581); a leaf that carries neither renders the empty string, which the
	// fold drops via omitempty.
	//
	// v3 also reads the leaf's validity floor and SANs, and the per-link parsed facts
	// off the whole presented chain, so the four dark certificate rules derive at read
	// (P0.10b, #704). InsecureSkipVerify withholds none of this — the chain is fully
	// parsed regardless.
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
		Outcome:    TLSPresented,
		Chain:      chain,
		NotAfter:   leaf.NotAfter,
		Issuer:     leaf.Issuer.String(),
		Algorithm:  leaf.SignatureAlgorithm.String(),
		NotBefore:  leaf.NotBefore,
		SANDNS:     leaf.DNSNames,
		SANIP:      sanIP,
		ChainCerts: chainCerts,
		// Capture the raw CT inputs for the certificate_material side store (spec §5.3):
		// the leaf DER (embedded SCTs ride inside it), the TLS-extension SCTs, the stapled
		// OCSP response, and the issuer's SPKI (chain[1], for the embedded-SCT precert leaf
		// hash) — all already in `state`, previously discarded.
		LeafDER:    leaf.Raw,
		SCTsTLSExt: state.SignedCertificateTimestamps,
		OCSPStaple: state.OCSPResponse,
		IssuerSPKI: issuerSPKI(state.PeerCertificates),
	}
}

// issuerSPKI returns the DER SubjectPublicKeyInfo of the certificate that issued the leaf —
// chain position 1 in the presented chain — or nil when the peer presented only the leaf. The
// leaf's own key is at position 0; the issuer's key hash (SHA-256 of this SPKI) is what an
// embedded SCT's precert leaf hash needs (RFC 6962 §3.2, #878).
func issuerSPKI(chain []*x509.Certificate) []byte {
	if len(chain) < 2 {
		return nil
	}
	return chain[1].RawSubjectPublicKeyInfo
}

// parseChainCert reads one presented certificate's raw facts into a ChainCert: its
// subject/issuer DNs, whether its own signature verifies under its bound key (the ONE
// in-leaf crypto datum — CheckSignatureFrom against itself), its public-key algorithm
// and size parameters, and its signature DIGEST name. It never fails: an unrecognised
// key algorithm carries the algorithm name and no size (not weak at read), and an
// unrecognised signature algorithm carries an empty digest (never MD5/SHA-1, so never
// fires the signature limb). Store-raw, derive-at-read (T-leaf #712).
func parseChainCert(c *x509.Certificate) ChainCert {
	selfSig := c.CheckSignatureFrom(c) == nil
	cc := ChainCert{
		Subject:               c.Subject.String(),
		Issuer:                c.Issuer.String(),
		SelfSignatureVerifies: &selfSig,
		SigDigest:             sigDigestName(c.SignatureAlgorithm),
	}
	switch pk := c.PublicKey.(type) {
	case *rsa.PublicKey:
		cc.KeyAlg = "RSA"
		cc.KeyBits = pk.N.BitLen()
	case *ecdsa.PublicKey:
		cc.KeyAlg = "ECDSA"
		cc.KeyBits = pk.Curve.Params().BitSize
	case *dsa.PublicKey:
		cc.KeyAlg = "DSA"
		cc.KeyBits = pk.P.BitLen()   // L
		cc.KeyParamN = pk.Q.BitLen() // N
	case ed25519.PublicKey:
		cc.KeyAlg = "Ed25519"
	default:
		// Unrecognised key algorithm — carry the algorithm name and no size. A key
		// with no size row is not weak at read (deny-list §4.2).
		cc.KeyAlg = c.PublicKeyAlgorithm.String()
	}
	return cc
}

// sigDigestName maps a certificate's signature algorithm to its DIGEST name — the
// datum certificate-weak-key-or-signature's deny-list {MD5, SHA-1} reads (T-leaf #712
// §3.1). It is the digest, NOT the signature OID: SHA1WithRSA, DSAWithSHA1 and
// ECDSAWithSHA1 all map to "SHA-1". An unknown/unset algorithm maps to "" — never
// MD5/SHA-1, so it never fires the signature limb.
func sigDigestName(a x509.SignatureAlgorithm) string {
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
		// Ed25519 uses SHA-512 internally; it has no key row and never fires the sig
		// limb when self-signed. Only the {MD5,SHA-1} deny-list consults this, so any
		// non-MD5/SHA-1 name is equivalent — name it honestly.
		return "Ed25519"
	default:
		return ""
	}
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
