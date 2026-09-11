package queue

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/message"
)

var reentryT0 = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

func atDay(d int) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: reentryT0.Add(time.Duration(d) * 24 * time.Hour), Valid: true}
}

type spanFixture struct {
	id      int64
	facet   string
	vantage int64
	vector  drift.Vector
	opened  int
	closed  int
	reason  string
}

func subjectRows(fixtures ...spanFixture) []db.ListSpansForSubjectRow {
	rows := make([]db.ListSpansForSubjectRow, 0, len(fixtures))
	for _, f := range fixtures {
		row := db.ListSpansForSubjectRow{
			ID: f.id, SubjectKind: "name", SubjectKey: "back.example.com",
			Facet: f.facet, VantageID: pgtype.Int8{Int64: f.vantage, Valid: true},
			Value: []byte(`{"outcome":"Resolved"}`), Derivation: mustVectorJSON(f.vector),
			OpenedAt: atDay(f.opened),
		}
		if f.closed > 0 {
			row.ClosedAt = atDay(f.closed)
		}
		if f.reason != "" {
			row.ClosureReason = pgtype.Text{String: f.reason, Valid: true}
		}
		rows = append(rows, row)
	}
	return rows
}

func TestReEntryInputsSingleVantageCleanReturn(t *testing.T) {
	v1 := resolutionVector("1", "1")
	prior, broke := reEntryInputs(subjectRows(
		spanFixture{id: 1, facet: "resolution", vantage: 1, vector: v1, opened: 1, closed: 3, reason: "measured-absent"},
		spanFixture{id: 2, facet: "resolution", vantage: 1, vector: v1, opened: 5},
	))
	if prior == nil || prior.Reason != drift.ReasonMeasuredAbsent {
		t.Fatalf("the prior closure is the withdrawal, got %+v", prior)
	}
	if broke {
		t.Error("one witness under one vector is Break-free")
	}
}

func TestReEntryInputsSingleVantageBumpBreaks(t *testing.T) {
	prior, broke := reEntryInputs(subjectRows(
		spanFixture{id: 1, facet: "resolution", vantage: 1, vector: resolutionVector("1", "1"), opened: 1, closed: 3, reason: "measured-absent"},
		spanFixture{id: 2, facet: "resolution", vantage: 1, vector: resolutionVector("1", "2"), opened: 5},
	))
	if prior == nil {
		t.Fatal("the prior closure exists")
	}
	if !broke {
		t.Error("a wildcard-discrimination bump between withdrawal and return is a Break on the witness")
	}
}

func TestReEntryInputsNoPriorClosure(t *testing.T) {
	prior, broke := reEntryInputs(subjectRows(
		spanFixture{id: 1, facet: "resolution", vantage: 1, vector: resolutionVector("1", "1"), opened: 5},
	))
	if prior != nil || broke {
		t.Errorf("a first-ever opening has no prior closure and no Break, got %+v/%v", prior, broke)
	}
}

func TestReEntryInputsValueMoveClosureIsNoWithdrawal(t *testing.T) {
	v1 := resolutionVector("1", "1")
	prior, _ := reEntryInputs(subjectRows(
		spanFixture{id: 1, facet: "resolution", vantage: 1, vector: v1, opened: 1, closed: 3, reason: "measured-absent"},
		spanFixture{id: 2, facet: "resolution", vantage: 2, vector: v1, opened: 4, closed: 6},
		spanFixture{id: 3, facet: "resolution", vantage: 2, vector: v1, opened: 6},
		spanFixture{id: 4, facet: "resolution", vantage: 1, vector: v1, opened: 7},
	))
	if prior != nil {
		t.Errorf("the latest closure is a value move, so the subject never left, got %+v", prior)
	}
}

func TestReEntryInputsTwoClassesOneWitnessMoved(t *testing.T) {
	old, bumped := resolutionVector("1", "1"), resolutionVector("1", "2")
	prior, broke := reEntryInputs(subjectRows(
		spanFixture{id: 1, facet: "resolution", vantage: 1, vector: old, opened: 1, closed: 3, reason: "measured-absent"},
		spanFixture{id: 2, facet: "resolution", vantage: 2, vector: old, opened: 1, closed: 3, reason: "measured-absent"},
		spanFixture{id: 3, facet: "resolution", vantage: 2, vector: bumped, opened: 5},
		spanFixture{id: 4, facet: "resolution", vantage: 1, vector: old, opened: 6},
	))
	if prior == nil || prior.Reason != drift.ReasonMeasuredAbsent {
		t.Fatalf("prior = %+v, want the measured-absent withdrawal", prior)
	}
	if !broke {
		t.Error("one Break among several witnesses voids returned for the subject (ADR-0097)")
	}
}

func TestReEntryInputsTwoClassesBothClean(t *testing.T) {
	v1 := resolutionVector("1", "1")
	_, broke := reEntryInputs(subjectRows(
		spanFixture{id: 1, facet: "resolution", vantage: 1, vector: v1, opened: 1, closed: 3, reason: "measured-absent"},
		spanFixture{id: 2, facet: "resolution", vantage: 2, vector: v1, opened: 1, closed: 3, reason: "measured-absent"},
		spanFixture{id: 3, facet: "resolution", vantage: 2, vector: v1, opened: 5},
		spanFixture{id: 4, facet: "resolution", vantage: 1, vector: v1, opened: 6},
		spanFixture{id: 5, facet: "dns-record", vantage: 1, vector: resolutionVector("9", "9"), opened: 6},
	))
	if broke {
		t.Error("two clean resolution witnesses read returned; a non-resolution timeline is no witness")
	}
}

func TestReEntryInputsWitnessWithoutPriorSpanIsNoBreak(t *testing.T) {
	old := resolutionVector("1", "1")
	_, broke := reEntryInputs(subjectRows(
		spanFixture{id: 1, facet: "resolution", vantage: 1, vector: old, opened: 1, closed: 3, reason: "measured-absent"},
		spanFixture{id: 2, facet: "resolution", vantage: 1, vector: old, opened: 5},
		spanFixture{id: 3, facet: "resolution", vantage: 2, vector: resolutionVector("2", "2"), opened: 5},
	))
	if broke {
		t.Error("a witness with no prior closed span contributes no Break")
	}
}

func reEntryChange(prior *drift.Span, broke, aperture bool) spanChange {
	return spanChange{
		SubjectKind: "name", SubjectKey: "back.example.com", Facet: "resolution",
		Opened: true, OpenedAperture: aperture,
		Value:        []byte(`{"outcome":"Resolved","addresses":["198.51.100.9"]}`),
		Vector:       resolutionVector("1", "1"),
		PriorClosure: prior, WitnessBroke: broke,
	}
}

func headlineFor(t *testing.T, change spanChange) string {
	t.Helper()
	store := &fakeMessageStore{}
	var log []routed
	if err := produceMessages(context.Background(), store, 30, produceT0, []spanChange{change}, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false, true); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 1 {
		t.Fatalf("want one membership message, got %d", len(store.inserted))
	}
	return store.inserted[0].Headline
}

func TestMembershipEntryReturnedOnCleanReEntry(t *testing.T) {
	prior := &drift.Span{Reason: drift.ReasonMeasuredAbsent}
	if h := headlineFor(t, reEntryChange(prior, false, false)); !strings.Contains(h, "returned to the estate") {
		t.Errorf("a clean re-entry across a measured-absent closure reads returned, got %q", h)
	}
	if h := headlineFor(t, reEntryChange(prior, false, true)); !strings.Contains(h, "returned to the estate") {
		t.Errorf("the aperture marker does not void returned across a measured closure, got %q", h)
	}
}

func TestMembershipEntryAppearedWhenAWitnessBroke(t *testing.T) {
	prior := &drift.Span{Reason: drift.ReasonMeasuredAbsent}
	if h := headlineFor(t, reEntryChange(prior, true, false)); !strings.Contains(h, "entered the estate") {
		t.Errorf("a Break on a witness re-enters as appeared, got %q", h)
	}
}

func TestMembershipEntryDescopedClosureNeverReturns(t *testing.T) {
	prior := &drift.Span{Reason: drift.ReasonDescoped}
	if h := headlineFor(t, reEntryChange(prior, false, false)); !strings.Contains(h, "entered the estate") {
		t.Errorf("an unmarked re-entry across a descoped closure reads appeared (ADR-0087), got %q", h)
	}
	if h := headlineFor(t, reEntryChange(prior, false, true)); !strings.Contains(h, "came into view") {
		t.Errorf("a marked re-entry across a descoped closure stays revealed (#1039), got %q", h)
	}
}

func TestMembershipEntryFirstOpeningAppearsOrIsRevealed(t *testing.T) {
	if h := headlineFor(t, reEntryChange(nil, false, false)); !strings.Contains(h, "entered the estate") {
		t.Errorf("a first-ever opening reads appeared, got %q", h)
	}
	if h := headlineFor(t, reEntryChange(nil, false, true)); !strings.Contains(h, "came into view") {
		t.Errorf("a first-ever opening under a Seed reads revealed, got %q", h)
	}
}

func TestMembershipEntryReturnedIsADriftFiringAtTheName(t *testing.T) {
	store := &fakeMessageStore{}
	var log []routed
	change := reEntryChange(&drift.Span{Reason: drift.ReasonMeasuredAbsent}, false, false)
	if err := produceMessages(context.Background(), store, 31, produceT0, []spanChange{change}, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false, true); err != nil {
		t.Fatalf("produce: %v", err)
	}
	m := store.inserted[0]
	if m.Cause != string(message.CauseDrift) || m.SubjectKind != "name" || m.FiredAt != "back.example.com" {
		t.Errorf("returned fires drift at the Name, got %+v", m)
	}
	if m.Class != string(message.ClassDrift) {
		t.Errorf("returned carries the drift class its release routes on, got %q", m.Class)
	}
	// The census is not real yet, so the release poll enqueues the delivery (ADR-1806 §2).
	if len(log) != 0 {
		t.Errorf("a held row routes nothing at the cause, got %+v", log)
	}
}
