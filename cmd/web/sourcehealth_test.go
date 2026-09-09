package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/proposer"
)

func (f *fakeStore) ListSourceHealth(context.Context) ([]db.SourceHealth, error) {
	rows := make([]db.SourceHealth, 0, len(f.sourceHealth))
	for _, row := range f.sourceHealth {
		rows = append(rows, row)
	}
	return rows, nil
}

func (f *fakeStore) RecordSourceAttempt(_ context.Context, arg db.RecordSourceAttemptParams) (db.SourceHealth, error) {
	row := db.SourceHealth{Slug: arg.Slug, LastOutcome: arg.LastOutcome, LastAttemptAt: arg.LastAttemptAt}
	if arg.LastOutcome == outcomeError {
		row.ConsecutiveFailures = f.sourceHealth[arg.Slug].ConsecutiveFailures + 1
	}
	f.sourceHealth[arg.Slug] = row
	return row, nil
}

const arinRowName = "ARIN (entities?fn=)"

func sourceRow(t *testing.T, page, name string) string {
	// The card note names the never-attempted reading too, so a page match proves nothing.
	t.Helper()
	for _, chunk := range strings.Split(page, `<div class="st-row">`)[1:] {
		if strings.Contains(chunk, name) {
			return chunk
		}
	}
	t.Fatalf("no source row for %q; body: %s", name, page)
	return ""
}

func failingLookup(t *testing.T, f *fakeStore, base string, ac *http.Client) {
	t.Helper()
	captureLog(t)
	lookup(t, ac, base, "Example").Body.Close()
	if len(f.sourceHealth) == 0 {
		t.Fatal("the lookup wrote no health record")
	}
}

func TestANeverQueriedProposerReadsNeverAttempted(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	row := sourceRow(t, sourcesBody(t, ac, base), arinRowName)
	if !strings.Contains(row, `<span class="st-badge neutral">never attempted</span>`) {
		t.Errorf("a never-queried proposer does not read never attempted; row: %s", row)
	}
	if strings.Contains(row, "healthy") {
		t.Errorf("a never-queried proposer was called healthy (ADR-0223 §4); row: %s", row)
	}
}

func TestAnAdmittingSourceCarriesNoHealthReading(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// ADR-0223 rules the proposer surface alone, so crt.sh is left as it was (#1583).
	row := sourceRow(t, sourcesBody(t, ac, base), "crt.sh")
	if strings.Contains(row, "never attempted") || strings.Contains(row, "last attempt") {
		t.Errorf("an admitting source gained a health reading nobody ruled on; row: %s", row)
	}
}

func TestAFailingProposerStatesItFailedBesideItsToggle(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{
		err:      errors.New("arin: registry unreachable"),
		attempts: []proposer.Attempt{{SourceSlug: "arin", Err: errors.New("registry unreachable")}},
	}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")

	failingLookup(t, f, base, ac)
	row := sourceRow(t, sourcesBody(t, ac, base), arinRowName)
	if !strings.Contains(row, `<span class="st-badge danger">last attempt failed · 2026-08-15 12:00 UTC</span>`) {
		t.Fatalf("a failed lookup is not stated beside the toggle; row: %s", row)
	}

	failingLookup(t, f, base, ac)
	row = sourceRow(t, sourcesBody(t, ac, base), arinRowName)
	if !strings.Contains(row, "2 in a row") {
		t.Errorf("the consecutive-failure count is not counted; row: %s", row)
	}
}

func TestASucceededAttemptEndsTheFailureCount(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{
		err:      errors.New("arin: registry unreachable"),
		attempts: []proposer.Attempt{{SourceSlug: "arin", Err: errors.New("registry unreachable")}},
	}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")
	failingLookup(t, f, base, ac)

	fp.err = nil
	fp.attempts = []proposer.Attempt{{SourceSlug: "arin"}}
	fp.candidates = twoCandidates()
	lookup(t, ac, base, "Example").Body.Close()

	row := sourceRow(t, sourcesBody(t, ac, base), arinRowName)
	if !strings.Contains(row, `<span class="st-badge ok">last attempt succeeded · 2026-08-15 12:00 UTC</span>`) {
		t.Errorf("a succeeded attempt did not replace the failure reading; row: %s", row)
	}
	if got := f.sourceHealth["arin"].ConsecutiveFailures; got != 0 {
		t.Errorf("consecutive failures = %d after a success, want 0", got)
	}
}

func TestAnOrgCAIDAHoldsUnderNoKeyIsNotASourceFailure(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	gap := fmt.Errorf("caida search matched 71 AFRINIC records for %q and none carries an opaqueId: %w", "Telkom Kenya", proposer.ErrNoJoinKey)
	fp := &fakeProposer{
		err:      fmt.Errorf("afrinic: %w", gap),
		attempts: []proposer.Attempt{{SourceSlug: "afrinic", Err: gap}},
	}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")

	failingLookup(t, f, base, ac)
	failingLookup(t, f, base, ac)

	// CAIDA holds the org under no key, so the source did not fail (ADR-0227, #1634).
	if got := f.sourceHealth["afrinic"]; got.LastOutcome != outcomeOK || got.ConsecutiveFailures != 0 {
		t.Errorf("a join gap was recorded as a source failure: %+v", got)
	}
	row := sourceRow(t, sourcesBody(t, ac, base), "AFRINIC (CAIDA")
	if !strings.Contains(row, `<span class="st-badge ok">last attempt succeeded · 2026-08-15 12:00 UTC</span>`) {
		t.Errorf("a working source reads as failed for a gap that is not its failure; row: %s", row)
	}
}

func TestASourceOutcomeSurvivesARestart(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{
		err:      errors.New("arin: registry unreachable"),
		attempts: []proposer.Attempt{{SourceSlug: "arin", Err: errors.New("registry unreachable")}},
	}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")
	failingLookup(t, f, base, ac)

	// A second server over the same store keeps the database and loses every field on
	// the first one, which is what a restart does (ADR-0223 §4).
	restarted := start(t, f, "")
	rc := login(t, restarted, "admin", "hunter2hunter2")

	row := sourceRow(t, sourcesBody(t, rc, restarted), arinRowName)
	if !strings.Contains(row, "last attempt failed · 2026-08-15 12:00 UTC") {
		t.Errorf("the last outcome did not survive the restart; row: %s", row)
	}
}

func TestAHealthRecordNeverChangesEnablement(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{
		err:      errors.New("arin: registry unreachable"),
		attempts: []proposer.Attempt{{SourceSlug: "arin", Err: errors.New("registry unreachable")}},
	}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")

	for range 5 {
		failingLookup(t, f, base, ac)
	}

	if len(f.sourceStates) != 0 {
		t.Fatalf("a health record wrote an enablement override: %+v", f.sourceStates)
	}
	if !fp.lastEnabled["arin"] {
		t.Errorf("five failures disabled arin for the next lookup: %v", fp.lastEnabled)
	}
	row := sourceRow(t, sourcesBody(t, ac, base), arinRowName)
	if !strings.Contains(row, `aria-checked="true"`) {
		t.Errorf("five failures turned the arin toggle off; row: %s", row)
	}
}
