package queue

import (
	"context"
	"time"

	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/signal"
)

// Their class is read per firing, and the unchanged-span firing needs the clock sweep (#1728).

var clockRules = map[string]bool{
	"certificate-expired":       true,
	"certificate-not-yet-valid": true,
	"certificate-expiring":      true,
}

func signalEdgeRule(name string) bool {
	// The flagship names the sensitive-port edge, so it never fires alone (ADR-0026 §6).
	return !clockRules[name] && name != sensitivePortRule
}

func signalEdgeMessages(ctx context.Context, store messageStore, observedAt time.Time, changes []spanChange, prior []*message.Message) ([]*message.Message, error) {
	moved := map[[2]string]bool{}
	for _, c := range changes {
		if !c.Opened && ruleFacet(c.Facet) && ruleSubjectKind(c.SubjectKind) {
			moved[[2]string{c.SubjectKind, c.SubjectKey}] = true
		}
	}
	if len(moved) == 0 {
		return nil, nil
	}
	subjects, err := subjectsAtCause(ctx, store, observedAt, changes, func(kind, key string) bool {
		return moved[[2]string{kind, key}]
	})
	if err != nil {
		return nil, err
	}
	type edge struct {
		subject subjectAtCause
		rule    string
	}
	var edges []edge
	for _, s := range subjects {
		for _, o := range rulesAtCause(s, reachAtCause{}, observedAt) {
			if o.before != signal.NotFired || o.after != signal.Fired || !signalEdgeRule(o.name) {
				continue
			}
			if coveredByPrior(prior, s, o.name) {
				continue
			}
			edges = append(edges, edge{subject: s, rule: o.name})
		}
	}
	if len(edges) == 0 {
		return nil, nil
	}
	muted, err := annotatedPairs(ctx, store)
	if err != nil {
		return nil, err
	}
	var msgs []*message.Message
	for _, e := range edges {
		// An annotated pair's firing edge is recorded and is not a message (ADR-0016).
		if muted[[2]string{e.subject.key, e.rule}] {
			continue
		}
		msgs = append(msgs, message.SignalEdge(e.subject.kind, e.subject.key, e.rule, e.subject.moved, observedAt))
	}
	return msgs, nil
}

func annotatedPairs(ctx context.Context, store messageStore) (map[[2]string]bool, error) {
	rows, err := store.ListAnnotations(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[[2]string]bool, len(rows))
	for _, a := range rows {
		out[[2]string{a.SubjectKey, a.SignalName}] = true
	}
	return out, nil
}
