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
