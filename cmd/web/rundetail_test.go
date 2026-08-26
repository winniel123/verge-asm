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
	if !strings.Contains(page, `class="sh-pill on" href="/drift"`) {
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

// DF-F3: while a run is in flight the head's meta-refresh hole tails the log and the
// loghead shows the LIVE pulse; on conclusion the refresh returns 0 and the terminal
// state settles. The refresh is a truthy toggle in the frozen head — running drives it on.
func TestRunDetailRefreshWhileRunning(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(50, "standard", tick, 3, 1, 1, 1, 0, 0), // in flight: 1 ready, 1 running, 1 done
		progressRow(51, "standard", tick, 2, 0, 0, 2, 0, 0), // concluded: 2 of 2 done
	}
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{
		50: {
			{ID: 900, Kind: "dns-sweep", State: "done", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "eu-west-1", Valid: true}},
			{ID: 901, Kind: "reachability", State: "running", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "us-east-2", Valid: true}},
			{ID: 902, Kind: "port-census", State: "ready", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "ap-south-1", Valid: true}},
		},
		51: {
			{ID: 910, Kind: "dns-sweep", State: "done", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "eu-west-1", Valid: true}},
		},
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	running := getBody(t, ac, base+"/run/50", http.StatusOK)
	if !strings.Contains(running, `http-equiv="refresh"`) {
		t.Errorf("a running run should tail via the head meta-refresh; body: %s", running)
	}
	if !strings.Contains(running, "streaming") {
		t.Errorf("a running run should show the LIVE streaming pulse; body: %s", running)
	}
	if !strings.Contains(running, "rd-batch running") {
		t.Errorf("a running run should carry the running batch badge; body: %s", running)
	}

	// A concluded run stops the tail — no meta-refresh, no pulse.
	done := getBody(t, ac, base+"/run/51", http.StatusOK)
	if strings.Contains(done, `http-equiv="refresh"`) {
		t.Errorf("a concluded run must not keep meta-refreshing; body: %s", done)
	}
	if strings.Contains(done, "streaming") {
		t.Errorf("a concluded run must not show the LIVE pulse; body: %s", done)
	}
}

// DF-F3b: with ?job={id} the handler filters the log to that job's rows SERVER-SIDE and
// renders the loghead chip (× clears to the bare run route). The template is handed an
// already-narrowed .Log — nothing filters client-side. An unknown id renders the honest
// empty log plus the chip; the filter rides the URL so the live refresh survives it.
func TestRunDetailJobFilterServerSide(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(52, "standard", tick, 3, 1, 1, 1, 0, 0),
	}
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{
		52: {
			{ID: 900, Kind: "dns-sweep", State: "done", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "eu-west-1", Valid: true}},
			{ID: 901, Kind: "reachability", State: "running", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "us-east-2", Valid: true}},
			{ID: 902, Kind: "port-census", State: "ready", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "ap-south-1", Valid: true}},
		},
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Filter to job 901: only its line survives, the chip names it, and it clears to /run/52.
	filtered := getBody(t, ac, base+"/run/52?job=901", http.StatusOK)
	if !strings.Contains(filtered, "#901") {
		t.Errorf("filtered log should keep the target job's line; body: %s", filtered)
	}
	for _, gone := range []string{"#900", "#902"} {
		if strings.Contains(filtered, gone) {
			t.Errorf("filtered log leaked a non-target job line %q; body: %s", gone, filtered)
		}
	}
	if !strings.Contains(filtered, "job #901") || !strings.Contains(filtered, "Clear job filter") {
		t.Errorf("loghead job-filter chip missing; body: %s", filtered)
	}
	if !strings.Contains(filtered, `href="/run/52"`) {
		t.Errorf("chip should clear to the bare run route; body: %s", filtered)
	}
	// The filter survives the live tail — a running run still meta-refreshes.
	if !strings.Contains(filtered, `http-equiv="refresh"`) {
		t.Errorf("a filtered running run should still tail; body: %s", filtered)
	}

	// Unknown job id: the log is filtered to zero rows, so the run page renders the honest
	// "No log to show" empty state. The handler still SETS .JobFilter (see TestApplyJobFilter),
	// but the frozen rundetail.tmpl gates the loghead — chip included — inside {{if .Log}}, so
	// the chip does not render over an empty log: the DF-F3b edge ("empty .Log + chip still
	// renders") is unsatisfiable with the frozen template and is flagged for design (a patch
	// would move the loghead outside the {{if .Log}} gate). The handler is forward-compatible:
	// the chip appears with zero handler change once that patch lands.
	unknown := getBody(t, ac, base+"/run/52?job=99999", http.StatusOK)
	if !strings.Contains(unknown, "No log to show") {
		t.Errorf("unknown job filter should render the honest empty log; body: %s", unknown)
	}
	for _, gone := range []string{"#900", "#901", "#902"} {
		if strings.Contains(unknown, gone) {
			t.Errorf("unknown job filter leaked a job line %q; body: %s", gone, unknown)
		}
	}
}

// runStatusLabel maps a dispatch to the run page's .Status word. A stopped/terminated
// dispatch (DF-F4, via the shared seam) renders its literal outcome per the ruled interim
// edge; otherwise it is the live running/failed/complete derivation.
func TestRunStatusLabel(t *testing.T) {
	cases := []struct {
		name    string
		active  bool
		dead    int64
		outcome string
		want    string
	}{
		{"running", true, 0, "", "running"},
		{"failed", false, 2, "", "failed"},
		{"complete", false, 0, "", "complete"},
		{"stopped literal", false, 1, "stopped", "stopped"},
		{"terminated literal", false, 0, "terminated", "terminated"},
		{"outcome wins over active", true, 0, "terminated", "terminated"},
	}
	for _, c := range cases {
		if got := runStatusLabel(c.active, c.dead, c.outcome); got != c.want {
			t.Errorf("%s: runStatusLabel(%v,%d,%q)=%q, want %q", c.name, c.active, c.dead, c.outcome, got, c.want)
		}
	}
}

// runRefresh turns the head's meta-refresh tail on only while running.
func TestRunRefresh(t *testing.T) {
	if got := runRefresh("running"); got != 5 {
		t.Errorf("runRefresh(running)=%d, want 5", got)
	}
	for _, terminal := range []string{"complete", "failed", "stopped", "terminated", ""} {
		if got := runRefresh(terminal); got != 0 {
			t.Errorf("runRefresh(%q)=%d, want 0", terminal, got)
		}
	}
}

// applyJobFilter narrows the log to one job's rows and sets the chip — the DF-F3b pure core.
func TestApplyJobFilter(t *testing.T) {
	jobs := []jobView{
		{ID: 900, Kind: "dns-sweep", Vantage: "eu-west-1"},
		{ID: 901, Kind: "reachability", Vantage: "us-east-2"},
	}
	fullLog := func() []runLogLine {
		return []runLogLine{
			{JobID: 900, Tag: "#900", Text: "dns-sweep · done"},
			{JobID: 901, Tag: "#901", Text: "reachability · running"},
		}
	}

	// Blank param: no filter, no chip, log untouched.
	v := runView{Log: fullLog()}
	applyJobFilter(&v, "", "/run/52", jobs)
	if v.JobFilter != nil || len(v.Log) != 2 {
		t.Errorf("blank job param should not filter; JobFilter=%v log=%d", v.JobFilter, len(v.Log))
	}

	// Non-numeric param: also a no-op (not a filter).
	v = runView{Log: fullLog()}
	applyJobFilter(&v, "abc", "/run/52", jobs)
	if v.JobFilter != nil || len(v.Log) != 2 {
		t.Errorf("non-numeric job param should not filter; JobFilter=%v log=%d", v.JobFilter, len(v.Log))
	}

	// Known id: one line, chip carries kind + vantage + clear href.
	v = runView{Log: fullLog()}
	applyJobFilter(&v, "901", "/run/52", jobs)
	if v.JobFilter == nil || v.JobFilter.ID != 901 || v.JobFilter.Kind != "reachability" ||
		v.JobFilter.Vantage != "us-east-2" || v.JobFilter.ClearHref != "/run/52" {
		t.Errorf("chip not set from the matching job: %+v", v.JobFilter)
	}
	if len(v.Log) != 1 || v.Log[0].JobID != 901 {
		t.Errorf("log not narrowed to job 901: %+v", v.Log)
	}

	// Unknown id: empty log, chip still renders (id only, no kind/vantage).
	v = runView{Log: fullLog()}
	applyJobFilter(&v, "77", "/run/52", jobs)
	if v.JobFilter == nil || v.JobFilter.ID != 77 || v.JobFilter.Kind != "" {
		t.Errorf("unknown id should still render an id-only chip: %+v", v.JobFilter)
	}
	if len(v.Log) != 0 {
		t.Errorf("unknown id should empty the log, got %d lines", len(v.Log))
	}
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
