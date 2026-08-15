// Package corpus is the golden corpus for the http-exchange leaf (v1 spec §3.3,
// golden-corpus.md §4). It holds a checked-in enumeration of the cells the leaf
// owes a pinning row, each row being a (scope, scripted exchanger, expected NDJSON)
// triple run hermetically against an in-process exchanger — no network, no
// containers. A row protects the http-exchange leaf and nothing else, so it is a
// sibling of the connect-outcome / resolution / wildcard corpora and never pooled
// with them.
package corpus

import (
	"context"

	he "github.com/winniel123/verge-asm/internal/measure/httpexchange"
)

// scriptExchanger answers each Endpoint key from a fixed result and counts the
// calls, so a row can assert that a 3xx is never followed with a second request.
// An unscripted target reads as a failed exchange — a not-completed result, which
// creates no Endpoint — so a row that forgets a target renders a legible absence
// rather than panicking.
type scriptExchanger struct {
	byKey map[string]he.ExchangeResult
	calls map[string]int
}

func newScript(rules map[string]he.ExchangeResult) *scriptExchanger {
	return &scriptExchanger{byKey: rules, calls: map[string]int{}}
}

// Exchange implements httpexchange.Exchanger.
func (s *scriptExchanger) Exchange(_ context.Context, t he.Target) he.ExchangeResult {
	s.calls[t.EndpointKey()]++
	r, ok := s.byKey[t.EndpointKey()]
	if !ok {
		return he.ExchangeResult{Failed: true, Err: "unscripted"}
	}
	return r
}
