package main

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// toDispatchView folds the per-state job counts into progress. The seam the Scans
// monitor rests on: a retried row is superseded (a fresh job replaced it) so it is
// out of the live denominator, and a dead-lettered job is complete, not pending.
func TestToDispatchViewProgress(t *testing.T) {
	tick := time.Date(2026, 8, 16, 9, 30, 0, 0, time.UTC)
	tests := []struct {
		name                                  string
		row                                   db.ListDispatchProgressRow
		wantLive, wantCompleted, wantInFlight int64
		wantPercent                           int
		wantActive                            bool
	}{
		{
			name: "in flight, some done",
			row:  progressRow(10, "hot", tick, 3, 1, 1, 1, 0, 0),
			// total 3, ready 1, running 1, done 1
			wantLive: 3, wantCompleted: 1, wantInFlight: 2, wantPercent: 33, wantActive: true,
		},
		{
			name: "a retry supersedes its attempt, out of the denominator",
			row:  progressRow(11, "dns", tick, 4, 1, 0, 2, 0, 1),
			// total 4, ready 1 (fresh), done 2, retried 1 -> live 3, completed 2
			wantLive: 3, wantCompleted: 2, wantInFlight: 1, wantPercent: 66, wantActive: true,
		},
		{
			name:     "all done — complete, not active",
			row:      progressRow(12, "dns", tick, 2, 0, 0, 2, 0, 0),
			wantLive: 2, wantCompleted: 2, wantInFlight: 0, wantPercent: 100, wantActive: false,
		},
		{
			name:     "a dead-lettered job counts as complete",
			row:      progressRow(13, "hot", tick, 2, 0, 0, 1, 1, 0),
			wantLive: 2, wantCompleted: 2, wantInFlight: 0, wantPercent: 100, wantActive: false,
		},
		{
			name:     "no live work — percent is zero, not a divide",
			row:      progressRow(14, "cold", tick, 0, 0, 0, 0, 0, 0),
			wantLive: 0, wantCompleted: 0, wantInFlight: 0, wantPercent: 0, wantActive: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toDispatchView(tt.row)
			if got.Live != tt.wantLive || got.Completed != tt.wantCompleted || got.InFlight != tt.wantInFlight {
				t.Errorf("live/completed/inflight = %d/%d/%d, want %d/%d/%d",
					got.Live, got.Completed, got.InFlight, tt.wantLive, tt.wantCompleted, tt.wantInFlight)
			}
			if got.Percent != tt.wantPercent {
				t.Errorf("percent = %d, want %d", got.Percent, tt.wantPercent)
			}
			if got.Active != tt.wantActive {
				t.Errorf("active = %v, want %v", got.Active, tt.wantActive)
			}
			if got.DispatchedAt != "2026-08-16 09:30 UTC" {
				t.Errorf("dispatchedAt = %q, want the formatted fan-out instant", got.DispatchedAt)
			}
		})
	}
}

// toJobView reads the live state off a queue job: a 'retried' row is superseded,
// and a ready-or-running job past attempt 1 is a retry in flight.
func TestToJobView(t *testing.T) {
	superseded := toJobView(db.ListJobsForDispatchRow{ID: 1, Kind: "hot", State: "retried", Attempt: 1, MaxAttempts: 5})
	if !superseded.Superseded || superseded.Retrying {
		t.Errorf("a retried row should be superseded and not retrying, got %+v", superseded)
	}

	retry := toJobView(db.ListJobsForDispatchRow{ID: 2, Kind: "hot", State: "running", Attempt: 2, MaxAttempts: 5})
	if !retry.Retrying {
		t.Errorf("a running attempt 2 should read as retrying, got %+v", retry)
	}

	done := toJobView(db.ListJobsForDispatchRow{
		ID: 3, Kind: "dns", State: "done", Attempt: 1, MaxAttempts: 5,
		VantageName:  pgtype.Text{String: "eu-resolver", Valid: true},
		BatchOutcome: pgtype.Text{String: "completed", Valid: true},
	})
	if done.Vantage != "eu-resolver" || done.Batch != "completed" || done.Retrying || done.Superseded {
		t.Errorf("a done job read wrong: %+v", done)
	}
}

// An in-flight scan renders as in-progress with its per-job state and a self-refresh
// so it stays current as jobs complete (AC: shows in progress with job state; shows
// completed-vs-enqueued for the active dispatch).
func TestScansPageInFlight(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 16, 9, 30, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(10, "hot", tick, 3, 1, 1, 1, 0, 0), // active
	}
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{
		10: {
			{ID: 100, Kind: "hot", State: "done", Attempt: 1, MaxAttempts: 5,
				VantageName:  pgtype.Text{String: "eu-prober", Valid: true},
				BatchOutcome: pgtype.Text{String: "completed", Valid: true}},
			{ID: 101, Kind: "hot", State: "running", Attempt: 1, MaxAttempts: 5,
				VantageName: pgtype.Text{String: "us-prober", Valid: true}},
			{ID: 102, Kind: "hot", State: "ready", Attempt: 2, MaxAttempts: 5,
				VantageName: pgtype.Text{String: "ap-prober", Valid: true}},
		},
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/scans", http.StatusOK)

	// Progress: completed vs live for the active dispatch.
	if !strings.Contains(page, "1 / 3 jobs") {
		t.Errorf("progress count missing; body: %s", page)
	}
	// Per-job state renders, including the retrying attempt and the batch outcome.
	for _, want := range []string{"running", "retrying", "completed", "eu-prober", "us-prober", "ap-prober"} {
		if !strings.Contains(page, want) {
			t.Errorf("job detail missing %q; body: %s", want, page)
		}
	}
	// The in-flight view self-refreshes.
	if !strings.Contains(page, `http-equiv="refresh"`) {
		t.Errorf("an in-flight scans page should self-refresh; body: %s", page)
	}
}

// With nothing dispatched the page shows an idle state — a fact and the next
// action — and does NOT spin (AC: no scan running -> idle state, not error/blank).
func TestScansPageIdle(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/scans", http.StatusOK)

	if !strings.Contains(page, "No scan running") {
		t.Errorf("idle state missing; body: %s", page)
	}
	if strings.Contains(page, `http-equiv="refresh"`) {
		t.Errorf("an idle scans page must not self-refresh; body: %s", page)
	}
}

// A completed dispatch appears under recent history, not in flight, and the page
// does not refresh once nothing is running.
func TestScansPageHistory(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(9, "dns", tick, 2, 0, 0, 2, 0, 0), // complete
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/scans", http.StatusOK)

	if !strings.Contains(page, "Recent dispatches") || !strings.Contains(page, "dns") {
		t.Errorf("history table missing the completed dispatch; body: %s", page)
	}
	if !strings.Contains(page, "No scan running") {
		t.Errorf("with only history, the in-flight section should be idle; body: %s", page)
	}
	if strings.Contains(page, `http-equiv="refresh"`) {
		t.Errorf("no in-flight scan means no refresh; body: %s", page)
	}
}

// The monitor is read-only: a viewer reads it. (Post-T0 the scans monitor folds
// under Settings, so it is no longer a top-level nav pill; the surface still
// resolves and renders for a viewer.)
func TestScansPageViewerReads(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")

	base := start(t, f, "")
	ac := login(t, base, "viewer", "hunter2hunter2")
	page := getBody(t, ac, base+"/scans", http.StatusOK)

	// Post-#281 the scans monitor is the Settings scans sub-tab; the surface still
	// resolves and renders its idle state for a viewer.
	if !strings.Contains(page, "<h2>Scans</h2>") || !strings.Contains(page, "No scan running") {
		t.Errorf("scans monitor did not render for a viewer; body: %s", page)
	}
}

// scanSchedule (P0.4, #445): the header sub-line's "last full scan Xm ago · next in
// Yh Zm" instants. Last is the most recent Dispatch's fan-out; next is the soonest
// enabled-Scan cadence boundary, floored the way the dispatcher floors a tick.
func TestScanScheduleInstants(t *testing.T) {
	// fixedClock() is 2026-08-15 12:00:00 UTC; the seeded dns/hot Scans are daily
	// (enabled) and cold is monthly (disabled). The next daily boundary is midnight,
	// 12h out; the cold Scan is off the cadence, so it never contributes a tick.
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(9, "dns", now.Add(-40*time.Minute), 4, 0, 0, 4, 0, 0),
		progressRow(8, "hot", now.Add(-6*time.Hour), 4, 0, 0, 4, 0, 0),
	}
	srv := newServer(f, testKey, "", fixedClock())

	v := srv.scanSchedule(context.Background())

	if !v.HasLast {
		t.Fatal("HasLast = false, want a most-recent dispatch instant")
	}
	if want := now.Add(-40 * time.Minute); !v.LastScanAt.Equal(want) {
		t.Errorf("LastScanAt = %s, want %s (newest dispatch)", v.LastScanAt, want)
	}
	if v.LastAgo != "40m" {
		t.Errorf("LastAgo = %q, want %q", v.LastAgo, "40m")
	}
	if !v.HasNext {
		t.Fatal("HasNext = false, want a next cadence boundary")
	}
	if want := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC); !v.NextScanAt.Equal(want) {
		t.Errorf("NextScanAt = %s, want %s (next daily midnight)", v.NextScanAt, want)
	}
	if v.NextIn != "12h 0m" {
		t.Errorf("NextIn = %q, want %q", v.NextIn, "12h 0m")
	}
}

// With no Dispatch ever fanned out and every Scan disabled, both halves report the
// honest absence rather than a fabricated instant — the "never scanned" state.
func TestScanScheduleAbsent(t *testing.T) {
	f := newFakeStore()
	f.dispatchProgress = nil
	for i := range f.scans {
		f.scans[i].Enabled = false
	}
	srv := newServer(f, testKey, "", fixedClock())

	v := srv.scanSchedule(context.Background())
	if v.HasLast {
		t.Errorf("HasLast = true, want false with no dispatches")
	}
	if v.HasNext {
		t.Errorf("HasNext = true, want false with every Scan disabled")
	}
}

// humanizeCountdown renders the spec's two-unit "next in" figure across the ranges.
func TestHumanizeCountdown(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{5*time.Hour + 22*time.Minute, "5h 22m"},
		{47 * time.Minute, "47m"},
		{2*24*time.Hour + 3*time.Hour, "2d 3h"},
		{30 * time.Second, "<1m"},
		{time.Hour, "1h 0m"},
	}
	for _, tt := range tests {
		if got := humanizeCountdown(tt.d); got != tt.want {
			t.Errorf("humanizeCountdown(%s) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

// progressRow builds a ListDispatchProgressRow with the given per-state counts.
func progressRow(id int64, kind string, tick time.Time, total, ready, running, done, dead, retried int64) db.ListDispatchProgressRow {
	return db.ListDispatchProgressRow{
		DispatchID: id,
		ScanID:     id,
		ScanKind:   kind,
		CreatedAt:  pgtype.Timestamptz{Time: tick, Valid: true},
		Total:      total,
		Ready:      ready,
		Running:    running,
		Done:       done,
		Dead:       dead,
		Retried:    retried,
	}
}
