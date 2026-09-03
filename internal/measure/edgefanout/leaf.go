// Package edgefanout is the `edge-fanout` leaf: one no-SNI TLS handshake per candidate
// edge address, reporting the certificate that address serves to a client naming nothing.
// It is not a facet and opens no timeline; the fan-out reduction lives in Custody (ADR-0129 §6).
package edgefanout

import (
	"context"
	"net/netip"
	"time"

	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

const Kind = "edge-fanout"

const (
	Port      uint16 = 443
	Transport        = "tcp"
)

const NoServerName = ""

type Outcome string

const (
	Presented   Outcome = "presented"
	TLSRefused  Outcome = "tls-refused"
	NoTLS       Outcome = "no-tls"
	Unreachable Outcome = "unreachable"
)

func (o Outcome) Valid() bool {
	// The store's CHECK holds this same membership, so neither side is the only guard.
	switch o {
	case Presented, TLSRefused, NoTLS, Unreachable:
		return true
	default:
		return false
	}
}

type Result struct {
	Outcome     Outcome
	Fingerprint string
	LeafDER     []byte
}

type Handshaker interface {
	Handshake(ctx context.Context, target netip.AddrPort) Result
}

func Fold(res co.HandshakeResult) Result {
	if res.Outcome == co.TLSPresented {
		// connectoutcome renders the chain leaf-first, so chain[0] is the leaf's fingerprint.
		if len(res.Chain) == 0 {
			// An empty chain is a TLS peer turning us down, never a plaintext port.
			return Result{Outcome: TLSRefused}
		}
		return Result{Outcome: Presented, Fingerprint: res.Chain[0], LeafDER: res.LeafDER}
	}
	// Read before the TLS negatives: those assert something was there, this that nothing was.
	if res.Unreachable {
		return Result{Outcome: Unreachable}
	}
	// An SNI-required listener is the modal case, the false negative ADR-0129 §2 accepts.
	if res.Outcome == co.TLSRefused {
		return Result{Outcome: TLSRefused}
	}
	return Result{Outcome: NoTLS}
}

type NetHandshaker struct {
	Timeout time.Duration
	inner   co.Handshaker
}

func (n NetHandshaker) Handshake(ctx context.Context, target netip.AddrPort) Result {
	// Reusing connectoutcome's dial whole is what brings the egress guard along (#743).
	h := n.inner
	if h == nil {
		h = co.NetHandshaker{Timeout: n.Timeout}
	}
	// A server name here would silently measure a tenant's certificate, not the edge's (ADR-0129 §6).
	return Fold(h.Handshake(ctx, target, NoServerName))
}
