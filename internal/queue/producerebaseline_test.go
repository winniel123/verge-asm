package queue

import (
	"context"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/message"
)

func resolutionVector(walk, wildcard string) drift.Vector {
	return drift.NewVector(
		drift.Component{Leaf: "resolution-walk", Version: walk},
		drift.Component{Leaf: "wildcard-discrimination", Version: wildcard},
	)
}

func breakChange(kind, key, facet string, before, after drift.Vector) spanChange {
	return spanChange{
		SubjectKind: kind, SubjectKey: key, Facet: facet,
		Opened: false, Value: []byte(`{"outcome":"Resolved"}`),
		Vector: after, PrevVector: before,
	}
}

func TestRebaselineFiresOncePerAlertingDerivationPerRelease(t *testing.T) {
	before, after := resolutionVector("1", "1"), resolutionVector("2", "1")
	changes := []spanChange{
		breakChange("name", "a.example.com", "resolution", before, after),
		breakChange("name", "b.example.com", "resolution", before, after),
		breakChange("name", "a.example.com", "dns-record", before, after),
	}
	store := &fakeMessageStore{}
	var log []routed
	if err := produceMessages(context.Background(), store, 20, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false, true); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 1 {
		t.Fatalf("one re-baseline per alerting derivation per release, got %d: %+v", len(store.inserted), store.inserted)
	}
	m := store.inserted[0]
	if m.Cause != string(message.CauseAperture) || m.Class != string(message.ClassCoverage) {
		t.Errorf("a re-baseline is a coverage-class firing, got cause=%q class=%q", m.Cause, m.Class)
	}
	if m.SubjectKind != "derivation" || m.FiredAt != "resolution" {
		t.Errorf("a re-baseline fires at the derivation, got %q/%q", m.SubjectKind, m.FiredAt)
	}
	if !strings.Contains(m.Headline, "resolution-walk") || strings.Contains(m.Headline, "wildcard-discrimination") {
		t.Errorf("the headline names exactly the moved leaves, got %q", m.Headline)
	}
	if !strings.Contains(m.Headline, "appeared") {
		t.Errorf("a membership leaf move states the returned loss, got %q", m.Headline)
	}
	if c, _ := message.ParseCensus(m.Census); c.Len() != 1 || c.Entries[0].Key != "resolution-walk" {
		t.Errorf("the census carries the difference set, got %+v", c)
	}
	if len(log) != 1 || log[0].class != message.ClassCoverage {
		t.Errorf("the message is routed on the coverage class, got %+v", log)
	}
}

func TestRebaselineEmptyDifferenceSetFiresNothing(t *testing.T) {
	same := resolutionVector("1", "1")
	changes := []spanChange{
		breakChange("name", "a.example.com", "resolution", same, same),
		{SubjectKind: "name", SubjectKey: "new.example.com", Facet: "resolution", Opened: true, Vector: same},
	}
	if got := rebaselineMessages(produceT0, changes); len(got) != 0 {
		t.Errorf("a value move under one vector and a first opening carry no difference set, got %+v", got)
	}
}

func TestRebaselineNonAlertingDerivationFiresNothing(t *testing.T) {
	before := drift.NewVector(drift.Component{Leaf: "tls-handshake", Version: "1"})
	after := drift.NewVector(drift.Component{Leaf: "tls-handshake", Version: "2"})
	changes := []spanChange{
		breakChange("endpoint", "example.com|198.51.100.1:443/tcp", "certificate", before, after),
	}
	if got := rebaselineMessages(produceT0, changes); len(got) != 0 {
		t.Errorf("no message reads the certificate derivation, so its move fires nothing, got %+v", got)
	}
}

func TestRebaselineReachabilityMoveNamesNoReturnedLoss(t *testing.T) {
	before := drift.NewVector(
		drift.Component{Leaf: "connect-outcome", Version: "1"},
		drift.Component{Leaf: "blanket-discrimination", Version: "1"},
	)
	after := drift.NewVector(
		drift.Component{Leaf: "connect-outcome", Version: "1"},
		drift.Component{Leaf: "blanket-discrimination", Version: "2"},
	)
	changes := []spanChange{
		breakChange("service", "198.51.100.1:443/tcp", "reachability", before, after),
	}
	got := rebaselineMessages(produceT0, changes)
	if len(got) != 1 {
		t.Fatalf("the flagship reads reachability, so its move fires one re-baseline, got %d", len(got))
	}
	if got[0].FiredAt != "reachability" || strings.Contains(got[0].Headline, "appeared") {
		t.Errorf("a reachability move fires at its derivation and says nothing about returns, got %+v", got[0])
	}
}
