package tlsacceptance

import (
	"context"
	"crypto/tls"
	"net"
	"net/netip"
	"time"

	"github.com/winniel123/verge-asm/internal/custody"
)

// AcceptanceOutcome is the closed union the `tls-acceptance` value space is built
// on. Every value space is a closed union, never a record with optional fields,
// because a facet's measured negatives are values and must not collapse into "not
// measured" (CONTEXT.md, ADR-0011).
type AcceptanceOutcome string

const (
	// Enumerated: the listener spoke TLS and accepted at least one candidate. The
	// value carries the accepted versions and, for TLS 1.0–1.2, the accepted suites.
	Enumerated AcceptanceOutcome = "enumerated"
	// TLSRefused: the peer spoke TLS but accepted NO candidate we offered — an
	// SSLv3-only, RC4-only or 3DES-refusing-everything-else listener. `TLSRefused`
	// read together with the batch's recorded candidate set IS the finding — *the
	// peer spoke TLS and refused all of this* (measurement-offers §1.2). A value,
	// and distinct from NoTLS.
	TLSRefused AcceptanceOutcome = "tls-refused"
	// NoTLS: nothing on the port spoke TLS at all. A value, and distinct from a
	// refusal — collapsing the two files a plaintext listener under *TLS server*.
	NoTLS AcceptanceOutcome = "no-tls"
)

// VersionAcceptance is one accepted protocol version and, for TLS 1.0–1.2, the
// suites the listener accepted under it in the order it selected them (its own
// preference, a measured fact). For TLS 1.3 the suites list is empty by
// construction: the three TLS 1.3 suites are the library's choice, not ours, so
// the leaf records only that the version was accepted (measurement-offers §1.3).
type VersionAcceptance struct {
	Version string   `json:"version"`
	Ciphers []string `json:"ciphers,omitempty"`
}

// Attempt is the raw outcome of one enumeration handshake against a Service, at a
// pinned version, offering a set of suites. It is what the Enumerator reports; the
// pure fold turns a sequence of them into the acceptance value. `Spoke` records
// that the peer spoke TLS at all — the single bit that separates `TLSRefused` (all
// refused, but it spoke) from `NoTLS` (nothing there spoke TLS).
type Attempt struct {
	// Spoke is true where the peer spoke TLS at this handshake, whether or not it
	// accepted a candidate.
	Spoke bool
	// Accepted is true where the handshake completed under the pinned version with
	// one of the offered suites (or, for TLS 1.3, under a library-chosen suite).
	Accepted bool
	// SelectedCipher is the Go constant name of the suite the listener selected,
	// for a TLS 1.0–1.2 accept. Empty for a TLS 1.3 accept and for a refusal.
	SelectedCipher string
}

// Enumerator performs one enumeration handshake against a Service, pinning the
// protocol version and offering a suite set, and reports the raw Attempt. The
// production adapter dials real TLS (NetEnumerator); the golden corpus scripts an
// in-process Enumerator so the accept-fold runs hermetically with no network. An
// Enumerator never probes a Service it was not handed — the Scan's scope is the
// open `Service` population, and the leaf enumerates exactly it.
type Enumerator interface {
	Handshake(ctx context.Context, target netip.AddrPort, version string, offeredCiphers []string) Attempt
}

// Enumerate is the pure heart of the leaf and the thing the golden corpus pins: it
// drives an Enumerator over the candidate set and folds the attempts to the
// acceptance value. It is the enumeration strategy of measurement-offers §1.5 —
// one handshake per version, and per version 1.0–1.2 an ITERATIVE NARROWING that
// costs *accepted + 1* handshakes rather than one per candidate:
//
//   - Versions: one handshake per version with the version pinned (§1.5).
//   - Suites, per version 1.0–1.2: offer every remaining candidate, record the
//     selected suite, remove it, repeat until the listener refuses. The final
//     refusing round is what licenses the per-candidate negatives — the remaining
//     suites were offered TOGETHER and all refused, so each is a per-candidate
//     negative honestly obtained (§1.5).
//   - TLS 1.3: one handshake, no suites offered, recording only version acceptance.
//
// The verb is *accepted*, never *supported* — a listener may refuse an ECDSA suite
// because its certificate is RSA, and the leaf records the measured accept, not the
// capability claim it cannot carry (CONTEXT.md).
func Enumerate(ctx context.Context, e Enumerator, set CandidateSet, target netip.AddrPort) acceptanceValue {
	spoke := false
	var accepted []VersionAcceptance

	for _, ver := range set.Versions {
		if ver == TLS13 {
			att := e.Handshake(ctx, target, ver, nil)
			spoke = spoke || att.Spoke
			if att.Accepted {
				accepted = append(accepted, VersionAcceptance{Version: ver})
			}
			continue
		}
		// TLS 1.0–1.2: iterative narrowing over the suite candidates.
		remaining := append([]string(nil), set.Ciphers...)
		var got []string
		for len(remaining) > 0 {
			att := e.Handshake(ctx, target, ver, remaining)
			spoke = spoke || att.Spoke
			if !att.Accepted {
				break
			}
			next := removeString(remaining, att.SelectedCipher)
			if len(next) == len(remaining) {
				// The listener selected a suite we did not offer — a misbehaving
				// peer. Record the accept and stop rather than looping forever; the
				// enumeration never manufactures a negative it did not observe.
				got = append(got, att.SelectedCipher)
				break
			}
			got = append(got, att.SelectedCipher)
			remaining = next
		}
		if len(got) > 0 {
			accepted = append(accepted, VersionAcceptance{Version: ver, Ciphers: got})
		}
	}

	switch {
	case len(accepted) > 0:
		return acceptanceValue{Outcome: Enumerated, Versions: accepted}
	case spoke:
		// It spoke TLS and accepted nothing we offered — the legacy-listener finding.
		return acceptanceValue{Outcome: TLSRefused}
	default:
		return acceptanceValue{Outcome: NoTLS}
	}
}

// removeString returns s without the first occurrence of v. When v is absent the
// slice is returned unchanged (same length), which the caller reads as "the peer
// selected a suite outside our offer".
func removeString(s []string, v string) []string {
	for i, x := range s {
		if x == v {
			out := make([]string, 0, len(s)-1)
			out = append(out, s[:i]...)
			out = append(out, s[i+1:]...)
			return out
		}
	}
	return s
}

// NetEnumerator is the production Enumerator: a real TLS handshake with the version
// pinned (MinVersion == MaxVersion) and `Config.CipherSuites` set to the offered
// candidates mapped to their library IDs, SNI omitted (SNI is the subject of the
// `certificate` handshake, never a candidate here — measurement-offers §1.6) and
// verification disabled (acceptance is the measurement, not chain validity). It is
// NOT exercised by the hermetic golden corpus, which pins the accept-fold against a
// scripted Enumerator instead; its error handling is best-effort.
type NetEnumerator struct {
	// Timeout bounds one handshake dial. Zero uses a 3 s default.
	Timeout time.Duration
}

// Handshake implements Enumerator against the network. A completed handshake reads
// as Accepted with the negotiated suite's constant name (empty under TLS 1.3); a
// TLS-level rejection as spoke-but-refused; a peer that never spoke TLS as not-spoke.
func (n NetEnumerator) Handshake(ctx context.Context, target netip.AddrPort, version string, offeredCiphers []string) Attempt {
	timeout := n.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	v, ok := versionID(version)
	if !ok {
		return Attempt{} // an undeclared version is never put on the wire
	}
	// A target must be a valid literal IP: the leaf handshakes only pre-validated
	// addresses. Reject an invalid target at entry, backed by the socket-level
	// egress guard on the dialer below so a non-globally-reachable literal fails
	// closed even if that invariant is ever broken upstream (#743).
	if !target.Addr().IsValid() {
		return Attempt{}
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cfg := &tls.Config{
		InsecureSkipVerify: true, // #nosec G402 (accepted: TLS measurement probe — enumerates untrusted listeners; verifying the chain would drop the acceptance measurement. Not a trusted-service client call.)
		MinVersion:         v,
		MaxVersion:         v,
	}
	if version != TLS13 {
		cfg.CipherSuites = cipherIDs(offeredCiphers)
	}
	// custody.EgressGuard is the shared Control-hook backstop (delivery, resolutionwalk):
	// it refuses the socket when the resolved address is non-globally-reachable (#743).
	d := tls.Dialer{
		NetDialer: &net.Dialer{Control: custody.EgressGuard("tlsacceptance")},
		Config:    cfg,
	}
	conn, err := d.DialContext(dialCtx, "tcp", target.String())
	if err != nil {
		return Attempt{Spoke: spokeTLS(err)}
	}
	defer func() { _ = conn.Close() }()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return Attempt{}
	}
	state := tlsConn.ConnectionState()
	return Attempt{Spoke: true, Accepted: true, SelectedCipher: cipherName(state.CipherSuite, version)}
}
