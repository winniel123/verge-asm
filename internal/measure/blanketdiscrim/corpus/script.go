// Package corpus is the golden corpus for the blanket-discrimination leaf (v1
// spec §3.3, ADR-0104, golden-corpus.md §4). It holds a checked-in enumeration of
// the cells the leaf owes a pinning row, each row a (scope, scripted connector,
// scripted handshaker, fixed control-port set, expected NDJSON) tuple run
// hermetically through the composed reachability exchange — no network, no
// containers. A row protects the blanket-discrimination decision (control-port
// probe + verdict + reach `Gap` emission) and nothing else, so it is a sibling of
// the connect-outcome and certificate corpora and never pooled with them: a break
// in the blanket verdict moves THIS corpus's digest and bumps
// `blanket-discrimination`'s version, never `connect-outcome`'s.
package corpus

import (
	"context"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

// scriptConnector answers each `(Address, port)` from a fixed per-target sequence,
// so a row can script a blanket responder (every control port answers), an origin
// (a control port refuses), or a filtered host (the control ports time out). It
// serves both the control-port probe and the service-port connects. An unscripted
// target reads as a timeout — silence, which the control probe reads as an
// incomplete reading and the service connect reads as not-reached after its
// retries — so a row that forgets a target still renders a legible value rather
// than panicking.
type scriptConnector struct {
	seq   map[netip.AddrPort][]connectoutcome.ConnResult
	calls map[netip.AddrPort]int
}

func newScript(rules map[string][]connectoutcome.ConnResult) *scriptConnector {
	seq := make(map[netip.AddrPort][]connectoutcome.ConnResult, len(rules))
	for k, v := range rules {
		seq[netip.MustParseAddrPort(k)] = v
	}
	return &scriptConnector{seq: seq, calls: map[netip.AddrPort]int{}}
}

func (s *scriptConnector) Connect(_ context.Context, t netip.AddrPort) connectoutcome.ConnResult {
	i := s.calls[t]
	s.calls[t]++
	res := s.seq[t]
	if len(res) == 0 {
		return connectoutcome.ConnTimedOut
	}
	if i < len(res) {
		return res[i]
	}
	return res[len(res)-1]
}

// scriptHandshaker answers a fixed certificate outcome per Endpoint, so a
// not-blanket reached Service renders its certificate line deterministically. A
// blanket or gapped Service never reaches the handshaker — no reached Service is
// handed to it — which is itself a fact the corpus pins.
type scriptHandshaker struct {
	byEndpoint map[string]connectoutcome.HandshakeResult
}

func (h scriptHandshaker) Handshake(_ context.Context, t netip.AddrPort, serverName string) connectoutcome.HandshakeResult {
	return h.byEndpoint[connectoutcome.EndpointKey(serverName, t, "tcp")]
}
