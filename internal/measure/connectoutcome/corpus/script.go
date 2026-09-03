// Package corpus is the golden corpus for the connect-outcome leaf (v1 spec
// §3.3, golden-corpus.md §4). It holds a checked-in enumeration of the cells the
// leaf owes a pinning row, each row being a (scope, scripted connector, expected
// NDJSON) triple run hermetically against an in-process connector — no network,
// no containers. A row protects the connect-outcome leaf and nothing else, so it
// is a sibling of the resolution/wildcard corpora and never pooled with them.
package corpus

import (
	"context"
	"net/netip"

	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

// scriptConnector answers each `(Address, port)` from a fixed per-target
// sequence, so a row can script a transient timeout followed by a decided
// answer. An unscripted target reads as a timeout — silence, which on TCP
// decides not-reached after the retries — so a row that forgets a target still
// renders a legible value rather than panicking.
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
	if len(res) == 0 {
		return co.ConnTimedOut
	}
	if i < len(res) {
		return res[i]
	}
	return res[len(res)-1]
}
