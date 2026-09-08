package queue

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

func TestFacetMoveOpeningARuleAtFiredFiresOneDriftMessage(t *testing.T) {
	changes := []spanChange{
		{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Value: []byte(certExpired), Previous: []byte(certNoTLS)},
	}
	store := &fakeMessageStore{}
	var log []routed
	if err := produceMessages(context.Background(), store, 22, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 1 {
		t.Fatalf("one cause, one message; got %d: %+v", len(store.inserted), store.inserted)
	}
	m := store.inserted[0]
	if m.Cause != string(message.CauseDrift) || m.Class != string(message.ClassDrift) {
		t.Errorf("the world moved, so this is drift class; got cause=%q class=%q", m.Cause, m.Class)
	}
	if m.SubjectKind != "endpoint" || m.FiredAt != sensitiveEp {
		t.Errorf("fires at the move, got %s %s", m.SubjectKind, m.FiredAt)
	}
	rules, _ := ruleKeys(t, m.Census)
	if strings.Join(rules, ",") != "certificate-expired,certificate-self-signed" {
		t.Errorf("census = %v, want the two rules the move opened at fired", rules)
	}
	if !strings.Contains(m.Headline, "certificate moved") || !strings.Contains(m.Headline, "2 rules opened at fired") {
		t.Errorf("headline names the move and the rules: %q", m.Headline)
	}
	if len(log) != 1 || log[0].class != message.ClassDrift {
		t.Errorf("routed once as drift, got %+v", log)
	}
}

func TestFacetMoveInsideAFlagshipFiresNoSeparateMessage(t *testing.T) {
	changes := []spanChange{
		{SubjectKind: "service", SubjectKey: sensitiveSvc, Facet: "reachability", Value: reachValue("reached"), Previous: reachValue("not-reached")},
		{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Value: []byte(certExpired), Previous: []byte(certNoTLS)},
	}
	store := flagshipStore(sensitiveSvc)
	var log []routed
	if err := produceMessages(context.Background(), store, 25, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	// The residue clause: the flagship's census already carries the opening (ADR-0033 §3).
	if len(store.inserted) != 1 || store.inserted[0].SubjectKind != "service" {
		t.Fatalf("want the flagship alone, got %+v", store.inserted)
	}
	rules, _ := ruleKeys(t, store.inserted[0].Census)
	if !strings.Contains(strings.Join(rules, ","), "certificate-expired") {
		t.Errorf("the flagship census carries the rule the move opened, got %v", rules)
	}
}

func TestFacetMoveReadsTheUnchangedFacetsFromTheStore(t *testing.T) {
	changes := []spanChange{
		{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "http-identity", Value: []byte(http200), Previous: []byte(httpNone)},
	}
	store := &fakeMessageStore{open: map[string][]db.ListOpenSpansForSubjectRow{
		"endpoint|" + sensitiveEp: {
			{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Value: []byte(certNoTLS), OpenedAt: pgtype.Timestamptz{Time: produceT0.Add(-24 * time.Hour), Valid: true}},
			{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "http-identity", Value: []byte(http200), OpenedAt: pgtype.Timestamptz{Time: produceT0, Valid: true}},
		},
	}}
	var log []routed
	if err := produceMessages(context.Background(), store, 23, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 1 {
		t.Fatalf("want one message, got %d", len(store.inserted))
	}
	rules, _ := ruleKeys(t, store.inserted[0].Census)
	// plaintext-http-no-https reads the unchanged certificate facet the fold did not carry.
	if strings.Join(rules, ",") != "plaintext-http-no-https,unauthenticated-request-answered" {
		t.Errorf("census = %v, want the rules whose domain the HTTP move entered", rules)
	}
}

func TestFacetMoveOpeningNoRuleIsSilent(t *testing.T) {
	cases := map[string][]spanChange{
		"no-tls -> tls-refused stays outside every domain": {
			{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Value: []byte(certRefused), Previous: []byte(certNoTLS)},
		},
		"a within-domain not-fired -> fired edge is not an opening": {
			{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Value: []byte(certExpired), Previous: []byte(certValid)},
		},
		"an opening emits no Transition": {
			{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Opened: true, Value: []byte(certExpired)},
		},
		"a name facet move evaluates no rule at the cause": {
			{SubjectKind: "name", SubjectKey: "admin.example.com", Facet: "dns-record", Value: []byte(`{}`), Previous: []byte(`{}`)},
		},
	}
	for name, changes := range cases {
		t.Run(name, func(t *testing.T) {
			store := &fakeMessageStore{}
			var log []routed
			if err := produceMessages(context.Background(), store, 24, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
				t.Fatalf("produce: %v", err)
			}
			if len(store.inserted) != 0 {
				t.Errorf("want silence, got %+v", store.inserted)
			}
		})
	}
}
