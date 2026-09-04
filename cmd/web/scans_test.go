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
			name:     "in flight, some done",
			row:      progressRow(10, "hot", tick, 3, 1, 1, 1, 0, 0),
			wantLive: 3, wantCompleted: 1, wantInFlight: 2, wantPercent: 33, wantActive: true,
		},
		{
			name:     "a retry supersedes its attempt, out of the denominator",
			row:      progressRow(11, "dns", tick, 4, 1, 0, 2, 0, 1),
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

func TestToJobRollup(t *testing.T) {
	self := func(s string) string { return s }

	got := toJobRollup([]string{"done", "done", "running", "ready", "dead", "retried"}, self)
	want := jobRollup{Ready: 1, Running: 1, Done: 2, Dead: 1, Total: 6}
	if got != want {
		t.Errorf("rollup = %+v, want %+v", got, want)
	}

	if empty := toJobRollup(nil, self); empty != (jobRollup{}) {
		t.Errorf("a dispatch with no jobs should roll up to zero, got %+v", empty)
	}
}

func TestFillFixtureRollups(t *testing.T) {
	active := []sfActive{{
		ID: 1409,
		Jobs: []sfJob{
			{ID: 912, State: "done"}, {ID: 913, State: "done"},
			{ID: 914, State: "ready"}, {ID: 915, State: "running"},
			{ID: 916, State: "retried"},
		},
	}}
	fillFixtureRollups(active)

	want := jobRollup{Ready: 1, Running: 1, Done: 2, Total: 5}
	if active[0].Rollup != want {
		t.Errorf("fixture rollup = %+v, want %+v", active[0].Rollup, want)
	}
}

func TestScansPageInFlight(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 16, 9, 30, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(10, "hot", tick, 3, 1, 1, 1, 0, 0),
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

	if !strings.Contains(page, "1 / 3 jobs") {
		t.Errorf("progress count missing; body: %s", page)
	}
	for _, want := range []string{
		`<span class="n">1</span>running`,
		`<span class="n">1</span>ready`,
		`<span class="n">1</span>done`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("rollup chip missing %q; body: %s", want, page)
		}
	}
	if !strings.Contains(page, `href="/runs/10">View all 3 jobs</a>`) {
		t.Errorf("drill button missing; body: %s", page)
	}
	// The history table is this tab's only st-table, and this fixture seeds no history.
	if strings.Contains(page, `class="st-table"`) {
		t.Errorf("the active card must render no table; body: %s", page)
	}
	for _, gone := range []string{
		"eu-prober", "us-prober", "ap-prober", "retrying", `</span>dead`,
		">Vantage</th>", ">Outcome</th>",
	} {
		if strings.Contains(page, gone) {
			t.Errorf("the card should not carry %q; body: %s", gone, page)
		}
	}
	if !strings.Contains(page, `http-equiv="refresh"`) {
		t.Errorf("an in-flight scans page should self-refresh; body: %s", page)
	}
}

func TestScansCardRollupChipRules(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 16, 9, 30, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(11, "hot", tick, 7, 0, 2, 3, 1, 1),
	}
	jobs := []db.ListJobsForDispatchRow{
		{ID: 200, Kind: "hot", State: "running", Attempt: 1, MaxAttempts: 5},
		{ID: 201, Kind: "hot", State: "running", Attempt: 1, MaxAttempts: 5},
		{ID: 202, Kind: "hot", State: "done", Attempt: 1, MaxAttempts: 5},
		{ID: 203, Kind: "hot", State: "done", Attempt: 1, MaxAttempts: 5},
		{ID: 204, Kind: "hot", State: "done", Attempt: 2, MaxAttempts: 5},
		{ID: 205, Kind: "hot", State: "dead", Attempt: 5, MaxAttempts: 5},
		{ID: 206, Kind: "hot", State: "retried", Attempt: 1, MaxAttempts: 5},
	}
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{11: jobs}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/scans", http.StatusOK)

	for _, want := range []string{
		`<span class="n">2</span>running`,
		`<span class="n">3</span>done`,
		`<span class="n">1</span>dead`,
		`href="/runs/11">View all 7 jobs</a>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("rollup missing %q; body: %s", want, page)
		}
	}
	for _, gone := range []string{`</span>ready`, "superseded"} {
		if strings.Contains(page, gone) {
			t.Errorf("the rollup should not carry %q; body: %s", gone, page)
		}
	}
}

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

func TestScansPageHistory(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(9, "dns", tick, 2, 0, 0, 2, 0, 0),
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

func TestScansPageViewerReads(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")

	base := start(t, f, "")
	ac := login(t, base, "viewer", "hunter2hunter2")
	page := getBody(t, ac, base+"/scans", http.StatusOK)

	if !strings.Contains(page, ">Scans</h2>") || !strings.Contains(page, "No scan running") {
		t.Errorf("scans monitor did not render for a viewer; body: %s", page)
	}
}

func TestScanScheduleInstants(t *testing.T) {
	// The fake seeds dns and hot daily-enabled and cold monthly-disabled, so midnight is next.
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

func openedEvent(batchID int64, batchAt, openedAt time.Time) db.ListRecentDriftEventsRow {
	return db.ListRecentDriftEventsRow{
		Role:       "opened",
		BatchID:    batchID,
		BatchAt:    pgtype.Timestamptz{Time: batchAt, Valid: true},
		SubjectKey: "acmecorp.io",
		Facet:      "resolution",
		OpenedAt:   pgtype.Timestamptz{Time: openedAt, Valid: true},
		PrevValue:  nil,
	}
}

func sigAt(name string, first time.Time) db.SignalInstance {
	return db.SignalInstance{SignalName: name, SubjectKey: "acmecorp.io", FirstSeen: pgtype.Timestamptz{Time: first, Valid: true}}
}

func TestCountRunOutcome(t *testing.T) {
	now := time.Date(2026, 8, 22, 15, 0, 0, 0, time.UTC)
	batchAt := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	nextAt := time.Date(2026, 8, 22, 14, 30, 0, 0, time.UTC)
	prevAt := time.Date(2026, 8, 22, 8, 0, 0, 0, time.UTC)

	driftRows := []db.ListRecentDriftEventsRow{
		openedEvent(1500, nextAt, nextAt),
		openedEvent(1407, batchAt, batchAt),
		openedEvent(1407, batchAt, batchAt),
		openedEvent(1407, batchAt, batchAt),
		openedEvent(1300, prevAt, prevAt),
	}
	signals := []db.SignalInstance{
		sigAt("tls-1.0-accepted", batchAt),
		sigAt("sensitive-port", batchAt.Add(10*time.Minute)),
		sigAt("later-signal", nextAt),
		sigAt("earlier-signal", prevAt),
	}

	got := countRunOutcome(map[int64]bool{1407: true}, driftRows, signals, now)
	if !got.Concluded {
		t.Fatalf("expected concluded (run committed a batch)")
	}
	if got.Transitions != 3 {
		t.Errorf("transitions: got %d, want 3", got.Transitions)
	}
	if got.NewSignals != 2 {
		t.Errorf("new signals: got %d, want 2", got.NewSignals)
	}

	empty := countRunOutcome(map[int64]bool{}, driftRows, signals, now)
	if empty.Concluded {
		t.Errorf("expected not-concluded for a run with no batch")
	}
}

func TestDispatchBatchIDs(t *testing.T) {
	rows := []db.ListJobsForDispatchRow{
		{ID: 1, BatchID: pgtype.Int8{Int64: 1407, Valid: true}},
		{ID: 2, BatchID: pgtype.Int8{Int64: 1407, Valid: true}},
		{ID: 3, BatchID: pgtype.Int8{}},
	}
	set := dispatchBatchIDs(rows)
	if len(set) != 1 || !set[1407] {
		t.Errorf("dispatchBatchIDs: got %v, want {1407}", set)
	}
}

func TestRunDegradedFrom(t *testing.T) {
	healthy := []jobView{
		{Vantage: "eu-west-1", State: "done"},
		{Vantage: "us-east-2", State: "done"},
	}
	if d := runDegradedFrom(healthy); d != nil {
		t.Errorf("healthy run: got %+v, want nil", d)
	}

	degraded := []jobView{
		{Vantage: "eu-west-1", State: "done"},
		{Vantage: "ap-south-1", State: "done"},
		{Vantage: "ap-south-1", State: "dead"},
		{Vantage: "ap-south-1", State: "done"},
	}
	d := runDegradedFrom(degraded)
	if d == nil {
		t.Fatalf("degraded run: got nil, want a callout")
	}
	if d.Vantage != "ap-south-1" {
		t.Errorf("degraded vantage: got %q, want ap-south-1", d.Vantage)
	}
	if d.Detail != "missed 1 of 3 checks" {
		t.Errorf("degraded detail: got %q, want %q", d.Detail, "missed 1 of 3 checks")
	}
}
