// Package corpus is the golden corpus for the http-exchange leaf (v1 spec §3.3,
// golden-corpus.md §4). Every row renders hermetically against an in-process
// exchanger, and protects this leaf alone — never pooled with a sibling corpus.
package corpus

import (
	"context"

	he "github.com/winniel123/verge-asm/internal/measure/httpexchange"
)

type scriptExchanger struct {
	byKey map[string]he.ExchangeResult
	calls map[string]int
}

func newScript(rules map[string]he.ExchangeResult) *scriptExchanger {
	return &scriptExchanger{byKey: rules, calls: map[string]int{}}
}

func (s *scriptExchanger) Exchange(_ context.Context, t he.Target) he.ExchangeResult {
	s.calls[t.EndpointKey()]++
	r, ok := s.byKey[t.EndpointKey()]
	// An unscripted target reads as a failed exchange, so a forgotten row stays legible.
	if !ok {
		return he.ExchangeResult{Failed: true, Err: "unscripted"}
	}
	return r
}
