// Package certcorpus is the golden corpus for the tls-handshake step of the
// reachability exchange — the step that feeds the `certificate` facet (#197, v1
// spec §3.4, golden-corpus.md §4). It is a sibling of the connect-outcome corpus,
// not pooled with it: a row here protects the handshake fold and the certificate
// value space, and the connect verdict it rides is pinned by its own corpus.
//
// Each row is a (scope, scripted connector, scripted handshaker, expected NDJSON)
// tuple run hermetically through RunExchange against in-process fakes — no
// network, no TLS stack, no container. The rendered NDJSON carries BOTH the
// reachability line the connect produced and the certificate line the handshake
// produced, so a row shows the handshake riding the exchange rather than
// dispatching on its own (AC #197).
package certcorpus

import (
	"context"
	"net/netip"

	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

// scriptConnector answers each `(Address, port)` from a fixed per-target
// sequence. An unscripted target reads as a timeout — silence, which on TCP
// decides not-reached — so a row that means "not reached" simply leaves the
// target unscripted or scripts a refusal.
type scriptConnector struct {
	seq   map[netip.AddrPort][]co.ConnResult
	calls map[netip.AddrPort]int
}

func newConn(rules map[string][]co.ConnResult) *scriptConnector {
	seq := make(map[netip.AddrPort][]co.ConnResult, len(rules))
	for k, v := range rules {
		seq[netip.MustParseAddrPort(k)] = v
	}
	return &scriptConnector{seq: seq, calls: map[netip.AddrPort]int{}}
}

func (s *scriptConnector) Connect(_ context.Context, t netip.AddrPort) co.ConnResult {
	i := s.calls[t]
	s.calls[t]++
	res := s.seq[t]
	if len(res) == 0 {
		return co.ConnTimedOut
	}
	if i < len(res) {
		return res[i]
	}
	return res[len(res)-1]
}

// scriptHandshaker answers a fixed HandshakeResult per Endpoint key. An Endpoint
// with no scripted result reads as NoTLS — the conservative negative — so a row
// that forgets an Endpoint still renders a legible value rather than panicking.
type scriptHandshaker struct {
	byEndpoint map[string]co.HandshakeResult
}

func newHandshaker(rules map[string]co.HandshakeResult) *scriptHandshaker {
	return &scriptHandshaker{byEndpoint: rules}
}

func (s *scriptHandshaker) Handshake(_ context.Context, t netip.AddrPort, serverName string) co.HandshakeResult {
	key := co.EndpointKey(serverName, t, "tcp")
	if r, ok := s.byEndpoint[key]; ok {
		return r
	}
	return co.HandshakeResult{Outcome: co.NoTLS}
}
