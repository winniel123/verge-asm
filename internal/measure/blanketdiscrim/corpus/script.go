// Package corpus is the golden corpus for the blanket-discrimination leaf (v1
// spec §3.3, ADR-0104, golden-corpus.md §4). It is never pooled with the
// connect-outcome or certificate corpora, so a break moves only this digest.
package corpus

import (
	"context"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

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
	// An unscripted target reads as silence, so a forgotten target renders rather than panicking.
	if len(res) == 0 {
		return connectoutcome.ConnTimedOut
	}
	if i < len(res) {
		return res[i]
	}
	return res[len(res)-1]
}

type scriptHandshaker struct {
	byEndpoint map[string]connectoutcome.HandshakeResult
}

func (h scriptHandshaker) Handshake(_ context.Context, t netip.AddrPort, serverName string) connectoutcome.HandshakeResult {
	// A blanket or gapped Service never reaches here, which is itself a fact the corpus pins.
	return h.byEndpoint[connectoutcome.EndpointKey(serverName, t, "tcp")]
}
