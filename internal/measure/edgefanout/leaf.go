// Package edgefanout is the `edge-fanout` leaf: one no-SNI TLS handshake against a
// candidate edge address, reporting the certificate that address serves to a client
// that names nothing.
//
// It is NOT a facet. It carries none of ADR-0011's six parts, opens no timeline, and so
// has no differ and no discriminator. It is the `wildcard-discrimination` shape — a
// measurement the binary makes to DECIDE MEMBERSHIP, recorded on its `Batch` by content
// and composed into the `Custody` derivation. A facet could not carry it: the vetoed
// edge is a non-member with no subject, so a timeline has nothing to hang on in the
// exact case the measurement exists to decide (ADR-0129 §6, as sharpened by the #954
// amendment).
//
// The leaf measures. It does not interpret: the fan-out reduction (eTLD+1 through the
// Public Suffix List) and the `shared-edge` threshold are versioned parameters of the
// `Custody` derivation, and they live there, not here (#984). What this package
// Observes is the certificate the edge presented — held, as every certificate is, by
// its `sha256:<hex>` fingerprint.
package edgefanout

import (
	"context"
	"net/netip"
	"time"

	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

// Kind is the JobSpec.Kind that dispatches to this leaf, and the `Scan` that fires it —
// the seventh (CONTEXT.md `Scan`).
const Kind = "edge-fanout"

// Port and Transport are the leaf's fixed measurement point. An edge serves its default
// certificate on 443/tcp, and there is no port list: this `Scan` is one handshake per
// candidate address, never a port tier (ADR-0028).
const (
	Port      uint16 = 443
	Transport        = "tcp"
)

// NoServerName is the server name the handshake sends: none. crypto/tls omits the
// server_name extension entirely for an empty ServerName, so the ClientHello carries no
// SNI at all and the edge answers with its DEFAULT certificate — the whole point of the
// measurement. Naming the empty string keeps the intent legible at the call site, since
// an accidental server name here would silently measure a tenant's certificate instead
// of the edge's (ADR-0129 §6).
const NoServerName = ""

// Outcome is the closed union this leaf reports for one candidate edge. Every value
// space is a closed union, never a record with optional fields, and an absence is never
// a value: a candidate the `Scan` did not measure carries no row at all, which is the
// state the `Custody` derivation reads as *measurement pending* (CONTEXT.md `Scan`).
type Outcome string

const (
	// Presented: the edge completed a no-SNI handshake and served a certificate. This
	// is the only value that carries a fingerprint.
	Presented Outcome = "presented"
	// TLSRefused: the edge spoke TLS and turned the nameless handshake down — an
	// SNI-required listener is the modal case, and it is exactly the false negative
	// ADR-0129 §2 accepts. A value, and distinct from NoTLS.
	TLSRefused Outcome = "tls-refused"
	// NoTLS: something answered on 443 and no TLS came back — a plaintext listener, or
	// a handshake that stalled after the connect completed. A value, and distinct from
	// a refusal: collapsing the two files a plaintext listener under *refused*.
	NoTLS Outcome = "no-tls"
	// Unreachable: the connect never reached a peer — refused, reset, timed out, or
	// refused at the socket by the egress guard. A value, and distinct from the two TLS
	// negatives: those say something was there, this says nothing was.
	Unreachable Outcome = "unreachable"
)

// Valid reports whether o is a member of the closed union. It is what the recording
// side tests a prober's self-reported outcome against before it persists one: the union
// is closed here, in the leaf that defines it, and the store's own CHECK holds the same
// membership, so neither is the only guard.
func (o Outcome) Valid() bool {
	switch o {
	case Presented, TLSRefused, NoTLS, Unreachable:
		return true
	default:
		return false
	}
}

// Result is one candidate edge's measurement. The fingerprint and the DER are carried
// iff the outcome is Presented; every negative carries neither, because a negative is a
// value in its own right and never a certificate we half-read.
//
// LeafDER is the raw material, not the value: the observation records the FINGERPRINT
// and the DER lands in the `certificate_material` side store keyed by it (ADR-0027,
// spec §5.3). The SAN set the fan-out reduction counts is derived from that DER at
// read, so the leaf stores it and interprets nothing.
type Result struct {
	Outcome     Outcome
	Fingerprint string
	LeafDER     []byte
}

// Handshaker performs one no-SNI TLS handshake against a candidate edge and reports the
// result. The production adapter dials real TLS through connectoutcome; a test scripts
// an in-process Handshaker, so the fold runs hermetically with no network.
//
// It takes no server name. That is the measurement's shape, not a caller's choice.
type Handshaker interface {
	Handshake(ctx context.Context, target netip.AddrPort) Result
}

// Fold reduces one connectoutcome handshake result to this leaf's closed union. It is
// the pure heart of the leaf: the network adapter and a scripted handshaker both pass
// through it, so one classification serves both.
//
// The four cases are ordered by what they assert. A presented chain is the positive.
// An unreachable dial asserts nothing was there, so it is read before the two TLS
// negatives, which both assert something was. A TLS refusal and a plaintext peer stay
// apart. Nothing else can arrive: connectoutcome's outcome space is closed.
func Fold(res co.HandshakeResult) Result {
	if res.Outcome == co.TLSPresented {
		// chain[0] IS the leaf's `sha256:<hex>` fingerprint — connectoutcome renders the
		// chain leaf-first — so a scripted handshaker that carries a chain and no DER
		// still names the certificate it presented.
		if len(res.Chain) == 0 {
			// A completed handshake that offered nothing we can hold is a TLS peer
			// turning us down, not a plaintext port — connectoutcome's own rule.
			return Result{Outcome: TLSRefused}
		}
		return Result{Outcome: Presented, Fingerprint: res.Chain[0], LeafDER: res.LeafDER}
	}
	if res.Unreachable {
		return Result{Outcome: Unreachable}
	}
	if res.Outcome == co.TLSRefused {
		return Result{Outcome: TLSRefused}
	}
	return Result{Outcome: NoTLS}
}

// NetHandshaker is the production Handshaker: connectoutcome's TLS dial run with NO
// server name. It reuses that dial whole — the egress guard on the Control hook, the
// record-not-verify stance, the absent ALPN and the leaf-DER capture all come with it —
// so this leaf obeys the same socket-level backstop every other connecting leaf obeys
// (#743) and forks none of it.
type NetHandshaker struct {
	// Timeout bounds the handshake dial. Zero uses connectoutcome's 3 s default.
	Timeout time.Duration
	// inner is the connectoutcome handshaker the dial goes through. Nil is the
	// production adapter. It exists as a seam so a test can pin the ONE argument this
	// leaf controls — the server name, which must always be NoServerName — without a
	// socket.
	inner co.Handshaker
}

// Handshake implements Handshaker against the network. It is one connect per address:
// the TLS dial's own TCP connect is the only connect the leaf makes, so a candidate
// edge sees exactly one socket per measurement.
func (n NetHandshaker) Handshake(ctx context.Context, target netip.AddrPort) Result {
	h := n.inner
	if h == nil {
		h = co.NetHandshaker{Timeout: n.Timeout}
	}
	return Fold(h.Handshake(ctx, target, NoServerName))
}
