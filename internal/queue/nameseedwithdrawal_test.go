package queue

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func nameTombstone(id int64, domain string) db.ListPendingNameSeedWithdrawalsRow {
	return db.ListPendingNameSeedWithdrawalsRow{
		ID:         id,
		NameDomain: pgtype.Text{String: domain, Valid: true},
	}
}

func nameCandidate(id int64, key string) db.ListNameSeedWithdrawalCandidatesRow {
	return db.ListNameSeedWithdrawalCandidatesRow{ID: id, SubjectKey: key}
}

// A withdrawal covers the apex and everything beneath it, and nothing else. A
// sibling domain that merely ends in the same letters is not beneath it.
func TestCoveringNameSeedWithdrawal(t *testing.T) {
	pending := []db.ListPendingNameSeedWithdrawalsRow{
		nameTombstone(1, "example.com"),
		nameTombstone(2, "corp.example.net"),
	}
	tests := []struct {
		name string
		want string
	}{
		{"example.com", "example.com"},
		{"www.example.com", "example.com"},
		{"a.b.example.com", "example.com"},
		{"EXAMPLE.COM.", "example.com"},
		{"notexample.com", ""},
		{"example.net", ""},
		{"corp.example.net", "corp.example.net"},
		{"www.corp.example.net", "corp.example.net"},
	}
	for _, tt := range tests {
		if got := coveringNameSeedWithdrawal(tt.name, pending); got != tt.want {
			t.Errorf("coveringNameSeedWithdrawal(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
	if got := coveringNameSeedWithdrawal("example.com", nil); got != "" {
		t.Errorf("no tombstone covers nothing, got %q", got)
	}
	invalid := []db.ListPendingNameSeedWithdrawalsRow{{ID: 1}}
	if got := coveringNameSeedWithdrawal("example.com", invalid); got != "" {
		t.Errorf("a tombstone with no domain covers nothing, got %q", got)
	}
}

// The withdrawal states its two counts with their factors, never as a product: a
// Name holding two timelines is ONE subject withdrawn and TWO timelines removed
// (message.NarrowingReceipt). The message fires at the withdrawn DOMAIN, because a
// name Seed's display scope IS its domain.
func TestComposeNameSeedWithdrawalsCountsSubjectsAndTimelines(t *testing.T) {
	pending := []db.ListPendingNameSeedWithdrawalsRow{nameTombstone(1, "example.com")}
	rows := []db.ListNameSeedWithdrawalCandidatesRow{
		nameCandidate(1, "www.example.com"),
		nameCandidate(2, "www.example.com"),
		nameCandidate(3, "api.example.com"),
	}

	spanIDs, receipts := composeNameSeedWithdrawals(rows, pending, membershipInputs{}, nil)

	if len(spanIDs) != 3 {
		t.Fatalf("every candidate timeline closes, got %v", spanIDs)
	}
	if len(receipts) != 1 {
		t.Fatalf("one act states itself once, got %d receipts", len(receipts))
	}
	r := receipts[0]
	if r.Scope != "example.com" || r.Removed != "example.com" {
		t.Errorf("the receipt fires at the withdrawn domain, got scope=%q removed=%q", r.Scope, r.Removed)
	}
	if r.SubjectsWithdrawn != 2 || r.TimelinesRemoved != 3 {
		t.Errorf("counts = (%d subjects, %d timelines), want (2, 3)", r.SubjectsWithdrawn, r.TimelinesRemoved)
	}
	if !r.Fires {
		t.Error("an inhabited withdrawal fires")
	}
	if want := "example.com withdrawn · 2 subjects withdrawn · 3 timelines taken out of the estate"; r.Headline != want {
		t.Errorf("headline = %q, want %q", r.Headline, want)
	}
}

// ADR-0135 §1: a name Seed withdrawal removes many Names in ONE act, so it writes
// one aggregate receipt rather than a row per Name — the count IS the payload, and
// a row per subject would be the census the receipt exists to replace (ADR-0074).
// This pins the count for a multi-Name withdrawal.
func TestComposeNameSeedWithdrawalsStatesOneActPerScope(t *testing.T) {
	pending := []db.ListPendingNameSeedWithdrawalsRow{
		nameTombstone(1, "example.com"),
		nameTombstone(2, "example.net"),
	}
	var rows []db.ListNameSeedWithdrawalCandidatesRow
	for i, n := range []string{"a.example.com", "b.example.com", "c.example.com", "d.example.com"} {
		rows = append(rows, nameCandidate(int64(i+1), n))
	}
	rows = append(rows, nameCandidate(5, "a.example.net"))

	_, receipts := composeNameSeedWithdrawals(rows, pending, membershipInputs{}, nil)

	if len(receipts) != 2 {
		t.Fatalf("one receipt per withdrawn scope, got %d: %+v", len(receipts), receipts)
	}
	if receipts[0].Scope != "example.com" || receipts[0].SubjectsWithdrawn != 4 {
		t.Errorf("four Names under one act state it once, got %+v", receipts[0])
	}
	if receipts[1].Scope != "example.net" || receipts[1].SubjectsWithdrawn != 1 {
		t.Errorf("the second scope states its own act, got %+v", receipts[1])
	}
}

// ADR-0135 §3, survivor one: a LIVE name Seed still declaring the Name keeps it,
// read from the Seed corpus and never from the tombstone. This is what settles a
// second covering Seed and a re-declared domain.
func TestComposeNameSeedWithdrawalsLiveSeedSurvives(t *testing.T) {
	pending := []db.ListPendingNameSeedWithdrawalsRow{nameTombstone(1, "example.com")}
	rows := []db.ListNameSeedWithdrawalCandidatesRow{
		nameCandidate(1, "www.example.com"),
		nameCandidate(2, "api.example.com"),
	}
	in := membershipInputs{seeds: []db.ListSeedsRow{nameSeed("api.example.com")}}

	spanIDs, receipts := composeNameSeedWithdrawals(rows, pending, in, nil)

	if len(spanIDs) != 1 || spanIDs[0] != 1 {
		t.Fatalf("only the Name no live Seed declares leaves, got %v", spanIDs)
	}
	if len(receipts) != 1 || receipts[0].SubjectsWithdrawn != 1 {
		t.Errorf("the receipt counts the survivor out, got %+v", receipts)
	}
}

// A re-declared domain closes nothing at all: the stale tombstone still names it,
// and reading the live Seed corpus is what stops it withdrawing ground that is
// declared again (ADR-0135 §3).
func TestComposeNameSeedWithdrawalsRedeclaredScopeClosesNothing(t *testing.T) {
	pending := []db.ListPendingNameSeedWithdrawalsRow{nameTombstone(1, "example.com")}
	rows := []db.ListNameSeedWithdrawalCandidatesRow{nameCandidate(1, "www.example.com")}
	in := membershipInputs{seeds: []db.ListSeedsRow{nameSeed("example.com")}}

	spanIDs, receipts := composeNameSeedWithdrawals(rows, pending, in, nil)

	if len(spanIDs) != 0 {
		t.Errorf("a re-declared scope withdraws nothing, got %v", spanIDs)
	}
	if len(receipts) != 0 {
		t.Errorf("and states no act, got %+v", receipts)
	}
}

// ADR-0135 §3, survivor two: a Name a SURVIVING Seed still admits keeps its
// timelines. The `admitted_name` rows carry their own seed_id, so the cascade takes
// only the withdrawn Seed's admissions; the Name stays in the resolution set and is
// still walked every batch. Closing it would be a `descoped` departure over a Name
// the estate never stopped measuring, reopened next batch and closed the batch
// after, for ever.
//
// The survivor is keyed by resolutionNameKey — the same key the resolution set uses
// — so a trailing dot or a case difference cannot make the two disagree.
func TestComposeNameSeedWithdrawalsAdmittedNameSurvives(t *testing.T) {
	pending := []db.ListPendingNameSeedWithdrawalsRow{nameTombstone(1, "example.com")}
	rows := []db.ListNameSeedWithdrawalCandidatesRow{
		nameCandidate(1, "www.example.com"),
		nameCandidate(2, "api.example.com"),
		nameCandidate(3, "mail.example.com"),
	}
	admitted := []string{"API.example.com.", "mail.example.com"}

	spanIDs, receipts := composeNameSeedWithdrawals(rows, pending, membershipInputs{}, admitted)

	if len(spanIDs) != 1 || spanIDs[0] != 1 {
		t.Fatalf("only the unadmitted Name leaves, got %v", spanIDs)
	}
	if len(receipts) != 1 || receipts[0].SubjectsWithdrawn != 1 {
		t.Errorf("the receipt counts the survivors out, got %+v", receipts)
	}
}

// A row no tombstone covers is DROPPED, not closed. A closure with no mover to name
// is a withdrawal the operator cannot trace back to their own act.
func TestComposeNameSeedWithdrawalsDropsUnattributableRows(t *testing.T) {
	pending := []db.ListPendingNameSeedWithdrawalsRow{nameTombstone(1, "example.com")}
	rows := []db.ListNameSeedWithdrawalCandidatesRow{
		nameCandidate(1, "www.example.com"),
		nameCandidate(2, "www.elsewhere.test"),
	}

	spanIDs, receipts := composeNameSeedWithdrawals(rows, pending, membershipInputs{}, nil)

	if len(spanIDs) != 1 || spanIDs[0] != 1 {
		t.Fatalf("only the attributable row closes, got %v", spanIDs)
	}
	if len(receipts) != 1 || receipts[0].TimelinesRemoved != 1 {
		t.Errorf("the unattributable row is in no count, got %+v", receipts)
	}
}

// The act is idempotent. The first fold closes the timelines, so a second fold over
// the same withdrawal reads no open candidate, closes nothing more and writes no
// second message.
func TestComposeNameSeedWithdrawalsIsIdempotent(t *testing.T) {
	pending := []db.ListPendingNameSeedWithdrawalsRow{nameTombstone(1, "example.com")}

	spanIDs, receipts := composeNameSeedWithdrawals(nil, pending, membershipInputs{}, nil)

	if len(spanIDs) != 0 {
		t.Errorf("a spent withdrawal closes nothing more, got %v", spanIDs)
	}
	if len(receipts) != 0 {
		t.Errorf("and states no second act, got %+v", receipts)
	}
}
