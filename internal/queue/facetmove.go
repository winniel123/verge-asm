package queue

import (
	"context"
	"time"

	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/signal"
)

// The fifth census producer: a move that opens a rule at fired, once per cause (ADR-0033 §3).

func facetMoveMessages(ctx context.Context, store messageStore, observedAt time.Time, changes []spanChange, prior []*message.Message) ([]*message.Message, error) {
	subjects, err := movedSubjectsAtCause(ctx, store, observedAt, changes)
	if err != nil {
		return nil, err
	}
	var msgs []*message.Message
	for _, s := range subjects {
		var entries []message.CensusEntry
		for _, o := range rulesAtCause(s, reachAtCause{}, observedAt) {
			// A not-fired -> fired edge is ADR-0026 §5's message, never this one (ADR-0033 §2).
			if o.before != signal.OutsideDomain || o.after != signal.Fired {
				continue
			}
			if coveredByPrior(prior, s, o.name) {
				continue
			}
			entries = append(entries, ruleEntry(o.name, s.key))
		}
		if m := message.FacetMove(s.kind, s.key, s.moved, message.NewCensus(entries...), observedAt); m != nil {
			msgs = append(msgs, m)
		}
	}
	return msgs, nil
}

func coveredByPrior(prior []*message.Message, s subjectAtCause, rule string) bool {
	// A census above the move that carries the opening silences it (ADR-0033 §3).
	for _, m := range prior {
		if m == nil || m.Census == nil {
			continue
		}
		if m.FiredAt != s.key && !keyNestsService(m.FiredAt, s.key) {
			continue
		}
		for _, e := range m.Census.Entries {
			if e.Kind == message.KindRule && e.Key == rule {
				return true
			}
		}
	}
	return false
}
