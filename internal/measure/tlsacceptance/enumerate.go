package tlsacceptance

import (
	"context"
	"crypto/tls"
	"net"
	"net/netip"
	"time"

	"github.com/winniel123/verge-asm/internal/custody"
)

// A measured negative is a value, so this is a closed union, not optional fields (ADR-0011).

type AcceptanceOutcome string

const (
	Enumerated AcceptanceOutcome = "enumerated"
	TLSRefused AcceptanceOutcome = "tls-refused"
	NoTLS      AcceptanceOutcome = "no-tls"
)

// The suite order is the listener's own selection preference, itself a measured fact (§1.5).

type VersionAcceptance struct {
	Version string   `json:"version"`
	Ciphers []string `json:"ciphers,omitempty"`
}

type Attempt struct {
	Spoke          bool
	Accepted       bool
	SelectedCipher string
}

type Enumerator interface {
	Handshake(ctx context.Context, target netip.AddrPort, version string, offeredCiphers []string) Attempt
}

func Enumerate(ctx context.Context, e Enumerator, set CandidateSet, target netip.AddrPort) acceptanceValue {
	// The verb is accepted, never supported: an RSA certificate refuses an ECDSA suite (CONTEXT.md).
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
		// Narrowing costs accepted + 1 handshakes per version, not one per candidate (§1.5).
		remaining := append([]string(nil), set.Ciphers...)
		var got []string
		// The refusing round is what licenses the negatives: the rest were offered together.
		for len(remaining) > 0 {
			att := e.Handshake(ctx, target, ver, remaining)
			spoke = spoke || att.Spoke
			if !att.Accepted {
				break
			}
			next := removeString(remaining, att.SelectedCipher)
			if len(next) == len(remaining) {
				// A peer selecting an unoffered suite would loop forever; we manufacture no negative.
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
		return acceptanceValue{Outcome: TLSRefused}
	default:
		return acceptanceValue{Outcome: NoTLS}
	}
}

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

// The hermetic golden corpus pins the accept-fold, not this path, so its errors are best-effort.

type NetEnumerator struct {
	Timeout time.Duration
}

func (n NetEnumerator) Handshake(ctx context.Context, target netip.AddrPort, version string, offeredCiphers []string) Attempt {
	timeout := n.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	v, ok := versionID(version)
	if !ok {
		return Attempt{}
	}
	if !target.Addr().IsValid() {
		return Attempt{}
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// SNI is the certificate exchange's subject, so ServerName is deliberately unset (§1.6).
	cfg := &tls.Config{
		InsecureSkipVerify: true, // #nosec G402 (accepted: TLS measurement probe — enumerates untrusted listeners; verifying the chain would drop the acceptance measurement. Not a trusted-service client call.)
		MinVersion:         v,
		MaxVersion:         v,
	}
	if version != TLS13 {
		cfg.CipherSuites = cipherIDs(offeredCiphers)
	}
	// The dialer's guard is the backstop: a non-globally-reachable literal fails closed here (#743).
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
