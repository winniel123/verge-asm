package queue

import (
	"context"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/signal"
	"github.com/winniel123/verge-asm/internal/signalfacts"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

// The rule is named here because internal/signal exports no name constant for it (#1723).

const sensitivePortRule = "sensitive-port-reached-from-internet"

// Its domain reads the estate's name set, which the fold does not hold at the cause (ADR-0033 §3).

const estateBoundRule = "redirect-to-host-outside-estate"

type facetValue struct {
	value      []byte
	observedAt time.Time
}

type subjectAtCause struct {
	kind   string
	key    string
	moved  []string
	before map[string]facetValue
	after  map[string]facetValue
}

type reachLeg struct {
	has     bool
	outcome string
}

type reachAtCause struct {
	before reachLeg
	after  reachLeg
}

type ruleOutcome struct {
	name   string
	before signal.Outcome
	after  signal.Outcome
}

func ruleFacet(facet string) bool {
	switch facet {
	case "", connectoutcome.FacetReachability, resolutionwalk.FacetResolution, resolutionwalk.FacetDNSRecord:
		return false
	default:
		return true
	}
}

func ruleSubjectKind(kind string) bool {
	// Name rules read a composed, multi-class resolution the fold does not hold (ADR-0080).
	return kind == "service" || kind == "endpoint"
}

func censusWithRules(ctx context.Context, store messageStore, observedAt time.Time, changes []spanChange, service string, base message.Census, reach reachAtCause) (message.Census, error) {
	beneath := func(kind, key string) bool { return ruleSubjectKind(kind) && keyNestsService(service, key) }
	subjects, err := subjectsAtCause(ctx, store, observedAt, changes, beneath)
	if err != nil {
		return message.Census{}, err
	}
	if !hasSubject(subjects, "service", service) {
		s, err := readSubjectAtCause(ctx, store, "service", service)
		if err != nil {
			return message.Census{}, err
		}
		subjects = append(subjects, s)
	}
	entries := append([]message.CensusEntry(nil), base.Entries...)
	seen := map[string]bool{}
	for _, s := range subjects {
		var legs reachAtCause
		if s.kind == "service" {
			legs = reach
		}
		for _, o := range rulesAtCause(s, legs, observedAt) {
			// The message carries the firing edge, so not-fired reads as opened.
			if o.after != signal.Fired || o.before == signal.Fired || seen[o.name] {
				continue
			}
			seen[o.name] = true
			entries = append(entries, ruleEntry(o.name, service))
		}
	}
	return message.NewCensus(entries...), nil
}

func ruleEntry(rule, subjectKey string) message.CensusEntry {
	e := message.CensusEntry{Kind: message.KindRule, Key: rule}
	if rule == sensitivePortRule {
		_, service := signalfacts.SplitEndpointName(subjectKey)
		if pair, _, ok := signalfacts.ParseServicePair(service); ok {
			e.Detail = pair.String()
		}
	}
	return e
}

func rulesAtCause(s subjectAtCause, reach reachAtCause, now time.Time) []ruleOutcome {
	var out []ruleOutcome
	switch s.kind {
	case "service":
		list := vergecore.Default()
		before := signalfacts.ServiceFactsFrom(s.key, serviceEvidence(s.before, reach.before), list)
		after := signalfacts.ServiceFactsFrom(s.key, serviceEvidence(s.after, reach.after), list)
		for _, r := range signal.AllServiceRules() {
			out = append(out, ruleOutcome{name: r.Name(), before: r.Eval(before), after: r.Eval(after)})
		}
	case "endpoint":
		before := signalfacts.EndpointFactsFrom(s.key, endpointEvidence(s.before), now, nil)
		after := signalfacts.EndpointFactsFrom(s.key, endpointEvidence(s.after), now, nil)
		for _, r := range signal.AllEndpointRules() {
			if r.Name() == estateBoundRule {
				continue
			}
			out = append(out, ruleOutcome{name: r.Name(), before: r.Eval(before), after: r.Eval(after)})
		}
	}
	return out
}

func serviceEvidence(facets map[string]facetValue, reach reachLeg) signalfacts.ServiceEvidence {
	ev := signalfacts.ServiceEvidence{HasInternetReach: reach.has, InternetReach: reach.outcome}
	if v, ok := facets[tlsacceptance.Facet]; ok {
		ev.HasTLSAcceptance = true
		ev.TLSAcceptance = v.value
	}
	return ev
}

func endpointEvidence(facets map[string]facetValue) signalfacts.EndpointEvidence {
	var ev signalfacts.EndpointEvidence
	if v, ok := facets[connectoutcome.FacetCertificate]; ok {
		ev.HasCertificate = true
		ev.Certificate = v.value
		ev.CertObservedAt = v.observedAt
	}
	if v, ok := facets[httpexchange.FacetHTTPIdentity]; ok {
		ev.HasHTTPIdentity = true
		ev.HTTPIdentity = v.value
	}
	return ev
}

func hasSubject(subjects []subjectAtCause, kind, key string) bool {
	for _, s := range subjects {
		if s.kind == kind && s.key == key {
			return true
		}
	}
	return false
}

func subjectsAtCause(ctx context.Context, store messageStore, observedAt time.Time, changes []spanChange, want func(kind, key string) bool) ([]subjectAtCause, error) {
	var subjects []subjectAtCause
	index := map[[2]string]int{}
	movedFacet := map[[2]string]map[string]bool{}
	for _, c := range changes {
		if !ruleFacet(c.Facet) || !want(c.SubjectKind, c.SubjectKey) {
			continue
		}
		id := [2]string{c.SubjectKind, c.SubjectKey}
		i, ok := index[id]
		if !ok {
			s, err := readSubjectAtCause(ctx, store, c.SubjectKind, c.SubjectKey)
			if err != nil {
				return nil, err
			}
			i = len(subjects)
			index[id] = i
			subjects = append(subjects, s)
			movedFacet[id] = map[string]bool{}
		}
		s := &subjects[i]
		s.after[c.Facet] = facetValue{value: c.Value, observedAt: observedAt}
		if movedFacet[id][c.Facet] {
			continue
		}
		if c.Opened {
			delete(s.before, c.Facet)
			continue
		}
		movedFacet[id][c.Facet] = true
		s.moved = append(s.moved, c.Facet)
		s.before[c.Facet] = facetValue{value: c.Previous, observedAt: observedAt}
	}
	return subjects, nil
}

func readSubjectAtCause(ctx context.Context, store messageStore, kind, key string) (subjectAtCause, error) {
	rows, err := store.ListOpenSpansForSubject(ctx, db.ListOpenSpansForSubjectParams{SubjectKind: kind, SubjectKey: key})
	if err != nil {
		return subjectAtCause{}, err
	}
	s := subjectAtCause{kind: kind, key: key, before: map[string]facetValue{}, after: map[string]facetValue{}}
	for _, r := range rows {
		// A Gap is no value, and the first vantage's value stands in for the rest (ADR-0080).
		if r.IsGap || !ruleFacet(r.Facet) {
			continue
		}
		if _, ok := s.after[r.Facet]; ok {
			continue
		}
		v := facetValue{value: r.Value, observedAt: r.OpenedAt.Time}
		s.after[r.Facet] = v
		s.before[r.Facet] = v
	}
	return s, nil
}
