package queue

import (
	"context"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/message"
)

func TestProduceMeasuredAbsentDepartureWritesAnUnroutedWithdrawal(t *testing.T) {
	departures := []departure{
		{SubjectKind: "name", SubjectKey: "gone.example.com", Reason: string(drift.ReasonMeasuredAbsent), SourceKey: "", Timelines: 2},
	}
	store := &fakeMessageStore{}
	var log []routed

	if err := produceMessages(context.Background(), store, 10, produceT0, nil, departures, nil, membershipInputs{}, fakeEnqueuer(1, &log), false, true); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 1 {
		t.Fatalf("a world withdrawal writes one membership-withdrawal message, got %d", len(store.inserted))
	}
	m := store.inserted[0]
	if m.Cause != string(message.CauseDrift) || m.Class != string(message.ClassDrift) {
		t.Errorf("a withdrawal is the world's act: cause=%q class=%q, want drift/drift", m.Cause, m.Class)
	}
	if m.SubjectKind != "name" || m.FiredAt != "gone.example.com" {
		t.Errorf("the withdrawal fires at the withdrawn subject, got %q/%q", m.SubjectKind, m.FiredAt)
	}
	if !strings.Contains(m.Headline, "gone.example.com") || !strings.Contains(m.Headline, "2 timelines") {
		t.Errorf("the headline states the subject and its timeline count, got %q", m.Headline)
	}
	if len(log) != 0 {
		t.Errorf("a withdrawal is written and never routed (ADR-0087), got %d routings", len(log))
	}
}

func TestProduceUncitedDepartureStaysSilent(t *testing.T) {
	departures := []departure{
		{SubjectKind: "address", SubjectKey: "198.51.100.7", Reason: string(drift.ReasonUncited), Timelines: 3},
	}
	store := &fakeMessageStore{}
	var log []routed
	if err := produceMessages(context.Background(), store, 15, produceT0, nil, departures, nil, membershipInputs{}, fakeEnqueuer(1, &log), false, true); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 0 || len(log) != 0 {
		t.Errorf("an uncited cascade closure writes and routes nothing, got %d messages / %d routings", len(store.inserted), len(log))
	}
}

func TestProduceMeasuredAbsentDepartureIsNoOpUnderDevMode(t *testing.T) {
	departures := []departure{
		{SubjectKind: "name", SubjectKey: "gone.example.com", Reason: string(drift.ReasonMeasuredAbsent), Timelines: 2},
	}
	store := &fakeMessageStore{}
	var log []routed
	if err := produceMessages(context.Background(), store, 16, produceT0, nil, departures, nil, membershipInputs{}, fakeEnqueuer(1, &log), true, true); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 0 || len(log) != 0 {
		t.Errorf("a dev install writes and routes nothing, got %d messages / %d routings", len(store.inserted), len(log))
	}
}
