package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func (f *fakeStore) addFoldedBatch(t *testing.T, at time.Time, prefix string, n int) int64 {
	t.Helper()
	b := f.freshBatch("hot", "resolution-walk")
	for i := 0; i < n; i++ {
		f.observations = append(f.observations, db.Observation{
			ID: f.obsNextID, BatchID: b, Facet: "resolution", SubjectKind: "name",
			SubjectKey: fmt.Sprintf("%s%03d.example.com", prefix, i), Source: "resolver",
			Value:      []byte(`{"outcome":"Resolved"}`),
			ObservedAt: pgtype.Timestamptz{Time: at, Valid: true},
		})
		f.obsNextID++
	}
	return b
}

func foldRows(batchID int64, at time.Time, prefix string, n int) []db.ListRecentDriftEventsRow {
	rows := make([]db.ListRecentDriftEventsRow, 0, n)
	for i := range n {
		rows = append(rows, driftOpenedRow(batchID, at, fmt.Sprintf("%s%03d.example.com", prefix, i),
			`{"outcome":"Resolved"}`, ""))
	}
	return rows
}

func TestBuildDriftFeedBoundsEachBatchAndMarksTheOverflow(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	rows := foldRows(2, now.Add(-time.Hour), "f", int(driftBatchLimit)+1)
	rows = append(rows, foldRows(1, now.Add(-2*time.Hour), "s", 2)...)

	groups, movement := buildDriftFeed(rows, now)
	if len(groups) != 2 {
		t.Fatalf("groups = %d, want 2: a bounded fold must not evict the batch behind it", len(groups))
	}
	if got := len(groups[0].Events); got != int(driftBatchLimit) {
		t.Errorf("the fold listed %d events, want %d", got, driftBatchLimit)
	}
	if !groups[0].Truncated {
		t.Error("a fold one row past the bound reported no per-batch truncation")
	}
	if len(groups[1].Events) != 2 || groups[1].Truncated {
		t.Errorf("the small batch listed %d events (truncated=%v), want 2 and false",
			len(groups[1].Events), groups[1].Truncated)
	}
	if got, want := movement["appeared"], int(driftBatchLimit)+2; got != want {
		t.Errorf("movement counted %d appearances, want %d: it counts what the page lists", got, want)
	}
}

func TestDriftExportCSVStatesAPerBatchTruncationApartFromTheWindow(t *testing.T) {
	at := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	s := &server{now: func() time.Time { return at.Add(time.Hour) }}

	rows := foldRows(2, at, "f", int(driftBatchLimit)+1)
	rows = append(rows, driftOpenedRow(1, at.Add(-time.Hour), "a.example.com", `{"outcome":"Resolved"}`, ""))

	rec := httptest.NewRecorder()
	s.writeDriftExportCSV(rec, "7d", rows, false)
	out := rec.Body.String()

	note := driftBatchCapNote(driftBatchLabel(rows[0], s.now()))
	if !strings.Contains(out, note) {
		t.Errorf("the export omitted %q, so a bounded batch reads as a whole fold; body:\n%s", note, out)
	}
	if !strings.Contains(out, "a.example.com") {
		t.Errorf("the export dropped the batch behind the fold; body:\n%s", out)
	}
	if strings.Contains(out, "feed capped at") {
		t.Errorf("a per-batch bound was reported as the window cap, which is a different fact; body:\n%s", out)
	}
	if strings.Contains(out, fmt.Sprintf("f%03d.example.com", driftBatchLimit)) {
		t.Errorf("the export listed the probe row the read takes past the bound; body:\n%s", out)
	}
}

func TestDriftPageStatesAPerBatchTruncationApartFromTheWindow(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")

	now := time.Now().UTC()
	f.addFoldedBatch(t, now.Add(-3*time.Hour), "small", 2)
	f.addFoldedBatch(t, now.Add(-time.Hour), "fold", int(driftBatchLimit)+1)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/drift?period=7d", http.StatusOK)

	want := fmt.Sprintf("It reads at most %d changes a batch.", driftBatchLimit)
	if !strings.Contains(page, want) {
		t.Errorf("a fold past the per-batch bound stated no truncation of its own; body: %s", page)
	}
	if !strings.Contains(page, "small000.example.com") {
		t.Errorf("a large fold evicted the batch behind it from the window; body: %s", page)
	}
	if strings.Contains(page, "Showing the most recent") {
		t.Errorf("a per-batch bound was stated as the window cap, which is a different fact; body: %s", page)
	}
}

func TestAPIv1DriftReportsBothTruncationsApart(t *testing.T) {
	f := newFakeStore()
	seedAPIToken(t, f, roleViewer)

	now := fixedClock()()
	f.addFoldedBatch(t, now.Add(-3*time.Hour), "small", 2)
	f.addFoldedBatch(t, now.Add(-time.Hour), "fold", int(driftBatchLimit)+1)

	rec := serveAPI(t, f, http.MethodGet, "/api/v1/drift", "Bearer "+apiTokenPlaintext)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var out apiDriftResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("body is not valid JSON: %v (%q)", err, rec.Body.String())
	}

	if out.Truncated {
		t.Errorf("a window well under %d events reported the window cap", out.FeedLimit)
	}
	if out.BatchLimit != driftBatchLimit {
		t.Errorf("batch_limit = %d, want %d", out.BatchLimit, driftBatchLimit)
	}
	if len(out.Batches) != 2 {
		t.Fatalf("batches = %d, want 2: a large fold evicted the batch behind it", len(out.Batches))
	}
	if !out.Batches[0].Truncated {
		t.Error("the fold reported no per-batch truncation")
	}
	if out.Batches[1].Truncated {
		t.Error("a two-event batch reported a per-batch truncation")
	}
	if got := len(out.Batches[0].Events); got != int(driftBatchLimit) {
		t.Errorf("the fold listed %d events, want %d", got, driftBatchLimit)
	}
}

func TestAPIv1DriftAtExactlyTheCapClaimsNoTruncation(t *testing.T) {
	read := func(t *testing.T, batches int) apiDriftResponse {
		t.Helper()
		f := newFakeStore()
		seedAPIToken(t, f, roleViewer)
		now := fixedClock()()
		for b := range batches {
			f.addFoldedBatch(t, now.Add(-time.Duration(b+1)*time.Hour), fmt.Sprintf("b%02d-", b), int(driftBatchLimit))
		}
		rec := serveAPI(t, f, http.MethodGet, "/api/v1/drift", "Bearer "+apiTokenPlaintext)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
		}
		var out apiDriftResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("body is not valid JSON: %v (%q)", err, rec.Body.String())
		}
		return out
	}

	full := int(driftFeedLimit / driftBatchLimit)

	exact := read(t, full)
	if exact.TransitionCount != int(driftFeedLimit) {
		t.Fatalf("transition_count = %d, want %d: the window must hold exactly the cap", exact.TransitionCount, driftFeedLimit)
	}
	if exact.Truncated {
		t.Errorf("a window holding exactly %d events reported the window cap", driftFeedLimit)
	}

	over := read(t, full+1)
	if !over.Truncated {
		t.Errorf("a window past %d events reported no window cap", driftFeedLimit)
	}
	if over.TransitionCount != int(driftFeedLimit) {
		t.Errorf("transition_count = %d, want %d: the response renders at most the cap", over.TransitionCount, driftFeedLimit)
	}
}
