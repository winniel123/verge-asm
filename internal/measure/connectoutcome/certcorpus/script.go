// Package certcorpus is the golden corpus for the tls-handshake step of the
// reachability exchange (#197, v1 spec §3.4, golden-corpus.md §4). Rows run
// hermetically against in-process fakes — no network, no TLS stack, no container.
package certcorpus

import (
	"context"
	"net/netip"

	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

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
	// Silence on TCP decides not-reached, so an unscripted target is how a row means that.
	if len(res) == 0 {
		return co.ConnTimedOut
	}
	if i < len(res) {
		return res[i]
	}
	return res[len(res)-1]
}

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
	// The conservative negative, so a forgotten Endpoint renders legibly instead of panicking.
	return co.HandshakeResult{Outcome: co.NoTLS}
}
