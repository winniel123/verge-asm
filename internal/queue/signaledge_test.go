package queue

import (
	"context"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

const (
	tlsSvc = "198.51.100.1:443/tcp"

	tlsModern = `{"outcome":"enumerated","versions":[{"version":"1.2","ciphers":["a"]},{"version":"1.3","ciphers":["b"]}]}`
	tlsLegacy = `{"outcome":"enumerated","versions":[{"version":"1.0","ciphers":["a"]},{"version":"1.2","ciphers":["b"]}]}`
)

func signalEdgeMessagesOf(store *fakeMessageStore) []db.InsertMessageParams {
	var out []db.InsertMessageParams
	for _, m := range store.inserted {
		if strings.Contains(m.Headline, "now fired") {
			out = append(out, m)
		}
	}
	return out
}

func TestSignalEdgeOnACertificateMoveFiresOneDriftMessagePerRule(t *testing.T) {
	// certValid -> certExpired flips certificate-self-signed inside its domain; certificate-expired
	// flips too but reads a clock, and its firing is #1728's.
	changes := []spanChange{
		{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Value: []byte(certExpired), Previous: []byte(certValid)},
	}
	store := &fakeMessageStore{}
	var log []routed
	if err := produceMessages(context.Background(), store, 30, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 1 {
		t.Fatalf("one edge, one message; got %d: %+v", len(store.inserted), store.inserted)
	}
	m := store.inserted[0]
	if m.Cause != string(message.CauseDrift) || m.Class != string(message.ClassDrift) {
		t.Errorf("the certificate span moved, so the firing is drift class; got cause=%q class=%q", m.Cause, m.Class)
	}
	if m.SubjectKind != "endpoint" || m.FiredAt != sensitiveEp {
		t.Errorf("fires at the endpoint the rule read, got %s %s", m.SubjectKind, m.FiredAt)
	}
	if !strings.Contains(m.Headline, "certificate-self-signed") || !strings.Contains(m.Headline, "certificate moved") {
		t.Errorf("headline names the rule and the moved facet: %q", m.Headline)
	}
	if strings.Contains(m.Headline, "certificate-expired") {
		t.Errorf("a clock-reading rule's firing is not this producer's: %q", m.Headline)
	}
	if strings.Contains(m.Headline, "opened at fired") {
		t.Errorf("a within-domain edge is no opening, so no facet-move message fires: %q", m.Headline)
	}
	if len(log) != 1 || log[0].class != message.ClassDrift {
		t.Errorf("routed once as drift, got %+v", log)
	}
}

func TestSignalEdgeOnATLSAcceptanceMoveFiresAtTheService(t *testing.T) {
	changes := []spanChange{
		{SubjectKind: "service", SubjectKey: tlsSvc, Facet: "tls-acceptance", Value: []byte(tlsLegacy), Previous: []byte(tlsModern)},
	}
	store := &fakeMessageStore{}
	var log []routed
	if err := produceMessages(context.Background(), store, 31, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 1 {
		t.Fatalf("want one message, got %+v", store.inserted)
	}
	m := store.inserted[0]
	if m.SubjectKind != "service" || m.FiredAt != tlsSvc {
		t.Errorf("fires at the Service the rule read, got %s %s", m.SubjectKind, m.FiredAt)
	}
	if !strings.Contains(m.Headline, "tls-1.0-accepted") || !strings.Contains(m.Headline, "tls-acceptance moved") {
		t.Errorf("headline = %q", m.Headline)
	}
}

func TestSignalEdgeBeneathAFlagshipRidesItsCensus(t *testing.T) {
	changes := []spanChange{
		{SubjectKind: "service", SubjectKey: sensitiveSvc, Facet: "reachability", Value: reachValue("reached"), Previous: reachValue("not-reached")},
		{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Value: []byte(certExpired), Previous: []byte(certValid)},
	}
	store := flagshipStore(sensitiveSvc)
	var log []routed
	if err := produceMessages(context.Background(), store, 32, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	// One cause, one message: the flagship names the edge in its payload (ADR-0026 §6).
	if len(store.inserted) != 1 || store.inserted[0].SubjectKind != "service" {
		t.Fatalf("want the flagship alone, got %+v", store.inserted)
	}
	rules, _ := ruleKeys(t, store.inserted[0].Census)
	if !strings.Contains(strings.Join(rules, ","), "certificate-self-signed") {
		t.Errorf("the flagship census carries the edge, got %v", rules)
	}
}

func TestSignalEdgeOnAnAnnotatedPairIsRecordedAndNotAMessage(t *testing.T) {
	changes := []spanChange{
		{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Value: []byte(certExpired), Previous: []byte(certValid)},
	}
	cases := map[string]struct {
		annotations []db.Annotation
		want        int
	}{
		"the exact pair mutes": {
			annotations: []db.Annotation{{SubjectKey: sensitiveEp, SignalName: "certificate-self-signed"}},
			want:        0,
		},
		"another rule on the subject does not": {
			annotations: []db.Annotation{{SubjectKey: sensitiveEp, SignalName: "certificate-expired"}},
			want:        1,
		},
		"the same rule on another subject does not": {
			annotations: []db.Annotation{{SubjectKey: "other.example.com@" + sensitiveSvc, SignalName: "certificate-self-signed"}},
			want:        1,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			store := &fakeMessageStore{annotations: tc.annotations}
			var log []routed
			if err := produceMessages(context.Background(), store, 33, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
				t.Fatalf("produce: %v", err)
			}
			// The mute is on the pair's own edge and moves no number (ADR-0016).
			if got := len(signalEdgeMessagesOf(store)); got != tc.want {
				t.Errorf("want %d edge messages, got %+v", tc.want, store.inserted)
			}
		})
	}
}

func TestSignalEdgeIsSilentWhereNoRuleCrossedIt(t *testing.T) {
	cases := map[string][]spanChange{
		"a rule that stays fired has no edge": {
			{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Value: []byte(certExpired), Previous: []byte(certExpired)},
		},
		"fired -> not-fired is silent on a certificate rule": {
			{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Value: []byte(certValid), Previous: []byte(certExpired)},
		},
		"an opening is ADR-0033's message, not this one": {
			{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Value: []byte(certExpired), Previous: []byte(certNoTLS)},
		},
		"a reachability move on a sensitive port rides the flagship": {
			{SubjectKind: "service", SubjectKey: sensitiveSvc, Facet: "reachability", Value: reachValue("reached"), Previous: reachValue("not-reached")},
		},
		"a name resolution move evaluates no rule at the cause": {
			{SubjectKind: "name", SubjectKey: "admin.example.com", Facet: "resolution", Value: []byte(`{"outcome":"NameError"}`), Previous: []byte(`{"outcome":"Resolved"}`)},
		},
	}
	for name, changes := range cases {
		t.Run(name, func(t *testing.T) {
			store := &fakeMessageStore{}
			var log []routed
			if err := produceMessages(context.Background(), store, 34, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
				t.Fatalf("produce: %v", err)
			}
			if got := signalEdgeMessagesOf(store); len(got) != 0 {
				t.Errorf("want no edge message, got %+v", got)
			}
		})
	}
}

func TestSignalEdgeReadsAnnotationsOnlyWhenAnEdgeExists(t *testing.T) {
	changes := []spanChange{
		{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Value: []byte(certExpired), Previous: []byte(certExpired)},
	}
	store := &fakeMessageStore{}
	if err := produceMessages(context.Background(), store, 35, produceT0, changes, nil, nil, membershipInputs{}, nil, false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if store.annotationReads != 0 {
		t.Errorf("no edge, so the dial is never read; got %d reads", store.annotationReads)
	}
}
