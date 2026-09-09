package queue

import (
	"context"
	"time"

	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/signal"
	"github.com/winniel123/verge-asm/internal/signalfacts"
)

// A clock crosses with no observation, so the edge is read across two folds (ADR-0064 §2, #1728).

// These job kinds insert a completed batch and never reach produceMessages (worker.go).

var unfoldedBatchKinds = []string{scan.ZoneKind, scan.CTKind, scan.CTTailKind}

type clockEdge struct {
	key   string
	rule  string
	moved bool
	clock signal.CertClock
}

func certificateLifetimeMessages(ctx context.Context, store messageStore, observedAt time.Time, changes []spanChange, prior []*message.Message) ([]*message.Message, error) {
	window, err := store.FoldedBatchWindow(ctx, unfoldedBatchKinds)
	if err != nil {
		return nil, err
	}
	// A first fold has no decided before, as the flagship has none (ADR-0029).
	if !window.PrevAt.Valid || !window.LatestAt.Valid {
		return nil, nil
	}
	prev, latest := window.PrevAt.Time, window.LatestAt.Time

	var edges []clockEdge
	touched := map[string]bool{}
	for _, c := range changes {
		if c.SubjectKind != "endpoint" || c.Facet != connectoutcome.FacetCertificate || touched[c.SubjectKey] {
			continue
		}
		touched[c.SubjectKey] = true
		// An opening is a rule opened at fired, which the move's census carries (ADR-0033 §3).
		if c.Opened || len(c.Previous) == 0 {
			continue
		}
		before := readClockRules(c.SubjectKey, c.Previous, prev, prev)
		after := readClockRules(c.SubjectKey, c.Value, observedAt, latest)
		edges = append(edges, clockCrossings(c.SubjectKey, before, after, true, prior)...)
	}

	rows, err := store.ListOpenEndpointCertificateSpans(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		// The first vantage's span stands in for the rest (ADR-0080).
		if touched[r.SubjectKey] {
			continue
		}
		touched[r.SubjectKey] = true
		before := readClockRules(r.SubjectKey, r.Value, r.ObservedAt.Time, prev)
		after := readClockRules(r.SubjectKey, r.Value, r.ObservedAt.Time, latest)
		edges = append(edges, clockCrossings(r.SubjectKey, before, after, false, prior)...)
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
		if muted[annotatedPair{subject: e.key, rule: e.rule}] {
			continue
		}
		if e.moved {
			msgs = append(msgs, message.SignalEdge("endpoint", e.key, e.rule, []string{connectoutcome.FacetCertificate}, observedAt))
			continue
		}
		msgs = append(msgs, message.ClockEdge("endpoint", e.key, e.rule, e.clock, observedAt))
	}
	return msgs, nil
}

type clockReading struct {
	outcome map[string]signal.Outcome
	clock   signal.CertClock
}

func readClockRules(key string, value []byte, observed, at time.Time) clockReading {
	facts := signalfacts.EndpointFactsFrom(key, signalfacts.EndpointEvidence{HasCertificate: true, Certificate: value, CertObservedAt: observed}, at, nil)
	r := clockReading{outcome: map[string]signal.Outcome{}}
	if facts.CertDetails != nil && facts.CertDetails.Clock != nil {
		r.clock = *facts.CertDetails.Clock
	}
	for _, rule := range clockRuleSet() {
		r.outcome[rule.Name()] = rule.Eval(facts)
	}
	return r
}

func clockRuleSet() []signal.EndpointRule {
	var out []signal.EndpointRule
	for _, rule := range signal.AllEndpointRules() {
		if clockRules[rule.Name()] {
			out = append(out, rule)
		}
	}
	return out
}

func clockCrossings(key string, before, after clockReading, moved bool, prior []*message.Message) []clockEdge {
	var out []clockEdge
	subject := subjectAtCause{kind: "endpoint", key: key}
	for _, rule := range clockRuleSet() {
		name := rule.Name()
		if before.outcome[name] != signal.NotFired || after.outcome[name] != signal.Fired {
			continue
		}
		if coveredByPrior(prior, subject, name) {
			continue
		}
		out = append(out, clockEdge{key: key, rule: name, moved: moved, clock: after.clock})
	}
	return out
}
