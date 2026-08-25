package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// The Run detail screen (#297, T2) — the per-run drill-in ported from
// design-system/examples/console/RunDetail.jsx. A run is one Dispatch; the screen
// renders all four sections (stages, log, outcome, vantage health) off the same
// Operational queue corpus the Scans monitor reads, and empty-states where a
// dispatch carries no jobs. The route `/run/{id}` is stable for T16 to link to.
func TestRunDetailRendersSections(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 16, 9, 30, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(42, "hot", tick, 2, 0, 0, 2, 0, 0), // complete: 2 of 2 done
	}
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{
		42: {
			{ID: 100, Kind: "hot", State: "done", Attempt: 1, MaxAttempts: 5,
				VantageName:  pgtype.Text{String: "eu-west-1", Valid: true},
				BatchOutcome: pgtype.Text{String: "completed", Valid: true}},
			{ID: 101, Kind: "hot", State: "done", Attempt: 1, MaxAttempts: 5,
				VantageName:  pgtype.Text{String: "us-east-2", Valid: true},
				BatchOutcome: pgtype.Text{String: "completed", Valid: true}},
		},
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/run/42", http.StatusOK)

	// All four sections render, in the example's vocabulary.
	for _, want := range []string{
		"Stages",           // pipeline / stepper
		"What it produced", // outcome
		"As configured",    // parameters
		"Who looked",       // vantage health
	} {
		if !strings.Contains(page, want) {
			t.Errorf("run detail missing section %q; body: %s", want, page)
		}
	}

	// Real data wired: the batch status, the dispatched instant as the log title, the
	// stage folded from the job kind with its count, both vantages with an ok badge,
	// and the header meta.
	for _, want := range []string{
		"complete",               // batch status
		"batch 2026-08-16",       // log title carries the dispatched instant
		"2 of 2 done",            // stage count folded from the jobs
		"eu-west-1", "us-east-2", // the vantages that looked
		"hot profile",            // header meta: the scan kind as profile
		"2 vantages",             // vantage count
	} {
		if !strings.Contains(page, want) {
			t.Errorf("run detail missing wired value %q; body: %s", want, page)
		}
	}

	// The Outcome card is now the spec's — Transitions + New signals, the read-side join
	// of the derived stores keyed by the run's batch (#20a, ruled; the .Completed/.Dead
	// pair retired). This fake dispatch's jobs carry no committed batch, so the diff has
	// not concluded and both figures render the honest em dash — never a fabricated count.
	for _, want := range []string{"Transitions", "New signals"} {
		if !strings.Contains(page, want) {
			t.Errorf("outcome card missing the batch-join stat %q; body: %s", want, page)
		}
	}
	for _, retired := range []string{"Completed", "Dead-lettered"} {
		if strings.Contains(page, retired) {
			t.Errorf("run detail still renders the retired outcome stat %q; body: %s", retired, page)
		}
	}

	// The breadcrumb roots at Drift (the batch-detail origin) and the Drift nav pill
	// is the active one — keyed on NavActive.
	if !strings.Contains(page, `href="/drift"`) {
		t.Errorf("run detail breadcrumb missing Drift root; body: %s", page)
	}
	if !strings.Contains(page, `class="navpill active" href="/drift"`) {
		t.Errorf("run detail nav pill not marked active; body: %s", page)
	}

	// Vantage vocabulary is held — a vantage looked, never a probe/scanner/agent.
	for _, banned := range []string{"probe", "scanner", "agent"} {
		if strings.Contains(page, banned) {
			t.Errorf("run detail leaked a wire noun %q for vantage; body: %s", banned, page)
		}
	}
}

// A run with a dead-lettered job reads as failed, marks the vantage that missed as
// degraded, and raises the outcome callout naming it.
func TestRunDetailDegradedVantage(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 16, 14, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(43, "dns", tick, 3, 0, 0, 2, 1, 0), // 2 done, 1 dead
	}
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{
		43: {
			{ID: 200, Kind: "dns", State: "done", Attempt: 1, MaxAttempts: 5,
				VantageName: pgtype.Text{String: "eu-west-1", Valid: true}},
			{ID: 201, Kind: "dns", State: "done", Attempt: 1, MaxAttempts: 5,
				VantageName: pgtype.Text{String: "us-east-2", Valid: true}},
			{ID: 202, Kind: "dns", State: "dead", Attempt: 5, MaxAttempts: 5,
				VantageName: pgtype.Text{String: "ap-south-1", Valid: true}},
		},
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/run/43", http.StatusOK)

	if !strings.Contains(page, "failed") {
		t.Errorf("a run with a dead-lettered job should read as failed; body: %s", page)
	}
	if !strings.Contains(page, "One vantage degraded") || !strings.Contains(page, "ap-south-1") {
		t.Errorf("degraded vantage callout missing; body: %s", page)
	}
	if !strings.Contains(page, "degraded") {
		t.Errorf("degraded vantage badge missing; body: %s", page)
	}
}

// A dispatch that enqueued no job renders the design-system empty-state in each
// section that reads the jobs, rather than fabricating a stage, log line or vantage.
func TestRunDetailEmptyState(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(44, "hot", tick, 0, 0, 0, 0, 0, 0), // no jobs
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/run/44", http.StatusOK)

	for _, want := range []string{
		"No stage to show",
		"No log to show",
		"No vantage recorded",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("run detail missing empty-state %q; body: %s", want, page)
		}
	}
	// The parameters section still renders from the dispatch — it needs no jobs.
	if !strings.Contains(page, "As configured") || !strings.Contains(page, "hot") {
		t.Errorf("parameters section should render without jobs; body: %s", page)
	}
}

// A run id that is not in recent history — aged out or never dispatched — 404s to
// the missing-run ErrorPage (U3, #480) rather than manufacturing a record: the id
// big-mono as `run #<id>`, the "no dispatch is keyed under that id" copy, and the way
// back to Drift. A non-numeric id 404s the same way.
func TestRunDetailUnknownRun(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/run/999", http.StatusNotFound)
	for _, want := range []string{"No such run", "No dispatch is keyed under that id", "run #999", "Back to drift"} {
		if !strings.Contains(page, want) {
			t.Errorf("missing-run page missing %q; body: %s", want, page)
		}
	}
	getBody(t, ac, base+"/run/not-a-number", http.StatusNotFound)
}

func TestRunDetailRequiresLogin(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	c := newClient(t)

	resp, err := c.Get(base + "/run/42")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("unauthenticated /run: status=%d location=%q, want redirect to /login",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}
