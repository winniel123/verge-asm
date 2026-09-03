// Package corpus is the golden corpus for the connect-outcome leaf (v1 spec §3.3,
// golden-corpus.md §4). Every row renders hermetically against an in-process
// connector, and protects this leaf alone — never pooled with a sibling corpus.
package corpus

import (
	"context"
	"net/netip"

	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

type scriptConnector struct {
	seq   map[netip.AddrPort][]co.ConnResult
	calls map[netip.AddrPort]int
}

func newScript(rules map[string][]co.ConnResult) *scriptConnector {
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
	// An unscripted target reads as silence, so a forgotten row stays legible, never a panic.
	if len(res) == 0 {
		return co.ConnTimedOut
	}
	if i < len(res) {
		return res[i]
	}
	return res[len(res)-1]
}
