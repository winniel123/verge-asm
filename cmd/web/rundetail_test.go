package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func TestRunDetailRendersSections(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 16, 9, 30, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(42, "hot", tick, 2, 0, 0, 2, 0, 0),
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

	for _, want := range []string{
		"Stages",
		"What it produced",
		"As configured",
		"Who looked",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("run detail missing section %q; body: %s", want, page)
		}
	}

	for _, want := range []string{
		"complete",
		"batch 2026-08-16",
		"2 of 2 done",
		"eu-west-1", "us-east-2",
		"hot profile",
		"2 vantages",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("run detail missing wired value %q; body: %s", want, page)
		}
	}

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

	if !strings.Contains(page, `href="/drift"`) {
		t.Errorf("run detail breadcrumb missing Drift root; body: %s", page)
	}
	if !strings.Contains(page, `class="sh-pill on" href="/drift"`) {
		t.Errorf("run detail nav pill not marked active; body: %s", page)
	}

	for _, banned := range []string{"probe", "scanner", "agent"} {
		if strings.Contains(page, banned) {
			t.Errorf("run detail leaked a wire noun %q for vantage; body: %s", banned, page)
		}
	}
}

func TestRunDetailDegradedVantage(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 16, 14, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(43, "dns", tick, 3, 0, 0, 2, 1, 0),
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

func TestRunDetailEmptyState(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(44, "hot", tick, 0, 0, 0, 0, 0, 0),
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
	if !strings.Contains(page, "As configured") || !strings.Contains(page, "hot") {
		t.Errorf("parameters section should render without jobs; body: %s", page)
	}
}

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

func TestRunDetailRefreshWhileRunning(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(50, "standard", tick, 3, 1, 1, 1, 0, 0),
		progressRow(51, "standard", tick, 2, 0, 0, 2, 0, 0),
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

	done := getBody(t, ac, base+"/run/51", http.StatusOK)
	if strings.Contains(done, `http-equiv="refresh"`) {
		t.Errorf("a concluded run must not keep meta-refreshing; body: %s", done)
	}
	if strings.Contains(done, "streaming") {
		t.Errorf("a concluded run must not show the LIVE pulse; body: %s", done)
	}
}

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
	if !strings.Contains(filtered, `http-equiv="refresh"`) {
		t.Errorf("a filtered running run should still tail; body: %s", filtered)
	}

	unknown := getBody(t, ac, base+"/run/52?job=99999", http.StatusOK)
	if !strings.Contains(unknown, "No log to show") {
		t.Errorf("unknown job filter should render the honest empty log; body: %s", unknown)
	}
	if !strings.Contains(unknown, "job #99999") || !strings.Contains(unknown, "Clear job filter") {
		t.Errorf("the loghead chip should render over the empty log (#34); body: %s", unknown)
	}
	for _, gone := range []string{"#900", "#901", "#902"} {
		if strings.Contains(unknown, gone) {
			t.Errorf("unknown job filter leaked a job line %q; body: %s", gone, unknown)
		}
	}
}

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

func TestRunDetailTerminalStatusBadge(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	stopped := progressRow(60, "standard", tick, 2, 0, 0, 2, 0, 0)
	stopped.Status = "stopped"
	terminated := progressRow(61, "standard", tick, 2, 0, 0, 1, 1, 0)
	terminated.Status = "terminated"
	failed := progressRow(62, "standard", tick, 1, 0, 0, 0, 1, 0)
	failed.Status = "fanned-out"
	f.dispatchProgress = []db.ListDispatchProgressRow{stopped, terminated, failed}
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{
		60: {{ID: 900, Kind: "dns-sweep", State: "done", Attempt: 1, MaxAttempts: 3,
			VantageName: pgtype.Text{String: "eu-west-1", Valid: true}}},
		61: {{ID: 910, Kind: "dns-sweep", State: "done", Attempt: 1, MaxAttempts: 3,
			VantageName: pgtype.Text{String: "eu-west-1", Valid: true}}},
		62: {{ID: 920, Kind: "dns-sweep", State: "dead", Attempt: 3, MaxAttempts: 3,
			VantageName: pgtype.Text{String: "eu-west-1", Valid: true}}},
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	for _, c := range []struct{ id, want string }{
		{"60", "rd-batch stopped"},
		{"61", "rd-batch terminated"},
		{"62", "rd-batch failed"},
	} {
		body := getBody(t, ac, base+"/run/"+c.id, http.StatusOK)
		if !strings.Contains(body, c.want) {
			t.Errorf("/run/%s should render %q; body: %s", c.id, c.want, body)
		}
	}
}

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

	v := runView{Log: fullLog()}
	applyJobFilter(&v, "", "/run/52", jobs)
	if v.JobFilter != nil || len(v.Log) != 2 {
		t.Errorf("blank job param should not filter; JobFilter=%v log=%d", v.JobFilter, len(v.Log))
	}

	v = runView{Log: fullLog()}
	applyJobFilter(&v, "abc", "/run/52", jobs)
	if v.JobFilter != nil || len(v.Log) != 2 {
		t.Errorf("non-numeric job param should not filter; JobFilter=%v log=%d", v.JobFilter, len(v.Log))
	}

	v = runView{Log: fullLog()}
	applyJobFilter(&v, "901", "/run/52", jobs)
	if v.JobFilter == nil || v.JobFilter.ID != 901 || v.JobFilter.Kind != "reachability" ||
		v.JobFilter.Vantage != "us-east-2" || v.JobFilter.ClearHref != "/run/52" {
		t.Errorf("chip not set from the matching job: %+v", v.JobFilter)
	}
	if len(v.Log) != 1 || v.Log[0].JobID != 901 {
		t.Errorf("log not narrowed to job 901: %+v", v.Log)
	}

	v = runView{Log: fullLog()}
	applyJobFilter(&v, "77", "/run/52", jobs)
	if v.JobFilter == nil || v.JobFilter.ID != 77 || v.JobFilter.Kind != "" {
		t.Errorf("unknown id should still render an id-only chip: %+v", v.JobFilter)
	}
	if len(v.Log) != 0 {
		t.Errorf("unknown id should empty the log, got %d lines", len(v.Log))
	}
}

type streamLine struct {
	Tag   string `json:"tag"`
	Level string `json:"level"`
	Text  string `json:"text"`
}
type streamResp struct {
	Lines []streamLine `json:"lines"`
	Next  int          `json:"next"`
	Done  bool         `json:"done"`
}

func getStream(t *testing.T, c *http.Client, url string) streamResp {
	t.Helper()
	raw := getBody(t, c, url, http.StatusOK)
	var got streamResp
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("stream %s: decode %q: %v", url, raw, err)
	}
	return got
}

func TestRunStreamAdvancesAndConcludes(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(70, "standard", tick, 2, 0, 1, 1, 0, 0),
	}
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{
		70: {
			{ID: 700, Kind: "dns-sweep", State: "done", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "eu-west-1", Valid: true}},
			{ID: 701, Kind: "reachability", State: "running", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "us-east-2", Valid: true}},
		},
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// The poll persists nothing at rest (raw-job-output.md §6.2).
	got := getStream(t, ac, base+"/run/70/stream?after=0")
	if len(got.Lines) != 2 || got.Next != 2 || got.Done {
		t.Fatalf("after=0: got %+v, want 2 lines, next 2, done false", got)
	}
	if got.Lines[1].Tag != "#701" || got.Lines[1].Text != "reachability · running · us-east-2" {
		t.Errorf("stream line does not match runLog derivation: %+v", got.Lines[1])
	}

	f.jobsByDispatch[70] = append(f.jobsByDispatch[70],
		db.ListJobsForDispatchRow{ID: 702, Kind: "reachability", State: "ready", Attempt: 2, MaxAttempts: 3,
			VantageName: pgtype.Text{String: "ap-south-1", Valid: true}})
	got = getStream(t, ac, base+"/run/70/stream?after=2")
	if len(got.Lines) != 1 || got.Next != 3 || got.Done {
		t.Fatalf("after=2: got %+v, want 1 new line, next 3, done false", got)
	}
	if got.Lines[0].Tag != "#702" || got.Lines[0].Level != "warn" {
		t.Errorf("retry-in-flight line should carry the warn level: %+v", got.Lines[0])
	}

	rows := f.jobsByDispatch[70]
	for i := range rows {
		rows[i].State = "done"
	}
	rows[len(rows)-1].State = "retried"
	got = getStream(t, ac, base+"/run/70/stream?after=3")
	if len(got.Lines) != 0 || got.Next != 3 || !got.Done {
		t.Fatalf("terminal: got %+v, want 0 lines, next 3, done true", got)
	}
}

func TestRunStreamJobScoped(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(71, "standard", tick, 3, 1, 1, 1, 0, 0),
	}
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{
		71: {
			{ID: 710, Kind: "dns-sweep", State: "done", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "eu-west-1", Valid: true}},
			{ID: 711, Kind: "reachability", State: "running", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "us-east-2", Valid: true}},
			{ID: 712, Kind: "port-census", State: "ready", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "ap-south-1", Valid: true}},
		},
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := getStream(t, ac, base+"/run/71/stream?job=711&after=0")
	if len(got.Lines) != 1 || got.Next != 1 || got.Done {
		t.Fatalf("job 711 after=0: got %+v, want 1 line, next 1, done false", got)
	}
	if got.Lines[0].Tag != "#711" {
		t.Errorf("job filter leaked a non-target line: %+v", got.Lines)
	}

	f.jobsByDispatch[71][1].State = "done"
	got = getStream(t, ac, base+"/run/71/stream?job=711&after=1")
	if len(got.Lines) != 0 || !got.Done {
		t.Fatalf("job 711 terminal: got %+v, want 0 lines, done true", got)
	}

	got = getStream(t, ac, base+"/run/71/stream?job=99999&after=0")
	if len(got.Lines) != 0 || !got.Done {
		t.Fatalf("unknown job: got %+v, want 0 lines, done true", got)
	}
}

func TestRunDetailStreamHref(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(80, "standard", tick, 3, 1, 1, 1, 0, 0),
		progressRow(81, "standard", tick, 2, 0, 0, 2, 0, 0),
	}
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{
		80: {
			{ID: 800, Kind: "dns-sweep", State: "done", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "eu-west-1", Valid: true}},
			{ID: 801, Kind: "reachability", State: "running", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "us-east-2", Valid: true}},
		},
		81: {
			{ID: 810, Kind: "dns-sweep", State: "done", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "eu-west-1", Valid: true}},
		},
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	live := getBody(t, ac, base+"/run/80", http.StatusOK)
	if !strings.Contains(live, `data-stream="/run/80/stream"`) {
		t.Errorf("a live run should carry the bare stream endpoint; body: %s", live)
	}
	if !strings.Contains(live, "data-stream") || !strings.Contains(live, ".rd-logbody[data-stream]") {
		t.Errorf("the long-poll script should be emitted for a live run; body: %s", live)
	}

	filtered := getBody(t, ac, base+"/run/80?job=801", http.StatusOK)
	if !strings.Contains(filtered, `data-stream="/run/80/stream?job=801"`) {
		t.Errorf("a filtered live job should carry the job-scoped stream endpoint; body: %s", filtered)
	}

	term := getBody(t, ac, base+"/run/80?job=800", http.StatusOK)
	if strings.Contains(term, "data-stream") {
		t.Errorf("a terminal job must not stream; body: %s", term)
	}

	done := getBody(t, ac, base+"/run/81", http.StatusOK)
	if strings.Contains(done, "data-stream") {
		t.Errorf("a concluded run must not stream; body: %s", done)
	}
}

func TestRunStreamHref(t *testing.T) {
	jobs := []jobView{
		{ID: 900, Kind: "dns-sweep", State: "done"},
		{ID: 901, Kind: "reachability", State: "running"},
	}
	if got := runStreamHref("/run/52", nil, true, jobs); got != "/run/52/stream" {
		t.Errorf("bare active run: got %q, want /run/52/stream", got)
	}
	if got := runStreamHref("/run/52", nil, false, jobs); got != "" {
		t.Errorf("bare inactive run: got %q, want empty", got)
	}
	f := &runJobFilter{ID: 901}
	if got := runStreamHref("/run/52", f, true, jobs); got != "/run/52/stream?job=901" {
		t.Errorf("filtered running job: got %q, want /run/52/stream?job=901", got)
	}
	f = &runJobFilter{ID: 900}
	if got := runStreamHref("/run/52", f, true, jobs); got != "" {
		t.Errorf("filtered terminal job: got %q, want empty", got)
	}
	f = &runJobFilter{ID: 99999}
	if got := runStreamHref("/run/52", f, true, jobs); got != "" {
		t.Errorf("filtered unknown job: got %q, want empty", got)
	}
}

func TestRunStreamRequiresLogin(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	c := newClient(t)

	resp, err := c.Get(base + "/run/42/stream?after=0")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("unauthenticated /run/{id}/stream: status=%d location=%q, want redirect to /login",
			resp.StatusCode, resp.Header.Get("Location"))
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

func TestLinkRunLog(t *testing.T) {
	v := runView{Log: []runLogLine{
		{JobID: 900, Tag: "#900", Text: "dns-sweep · done"},
		{JobID: 901, Tag: "#901", Text: "reachability · running"},
	}}
	linkRunLog(&v, "/runs/52")
	if v.Log[0].Href != "/runs/52?job=900" || v.Log[1].Href != "/runs/52?job=901" {
		t.Errorf("unfiltered log should link each job: %+v", v.Log)
	}

	v = runView{Log: []runLogLine{{Tag: "14:00:02", Text: "batch started"}}}
	linkRunLog(&v, "/runs/52")
	if v.Log[0].Href != "" {
		t.Errorf("a line with no job id should not link, got %q", v.Log[0].Href)
	}

	v = runView{
		Log:       []runLogLine{{JobID: 901, Tag: "#901", Text: "reachability · running"}},
		JobFilter: &runJobFilter{ID: 901},
	}
	linkRunLog(&v, "/runs/52")
	if v.Log[0].Href != "" {
		t.Errorf("a filtered log should not link its tags, got %q", v.Log[0].Href)
	}
}

func TestRunDetailLogTagsLinkToJob(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(52, "standard", tick, 2, 1, 0, 1, 0, 0),
	}
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{
		52: {
			{ID: 900, Kind: "dns-sweep", State: "done", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "eu-west-1", Valid: true}},
			{ID: 901, Kind: "reachability", State: "running", Attempt: 1, MaxAttempts: 3,
				VantageName: pgtype.Text{String: "us-east-2", Valid: true}},
		},
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	bare := getBody(t, ac, base+"/run/52", http.StatusOK)
	for _, want := range []string{`href="/run/52?job=900"`, `href="/run/52?job=901"`} {
		if !strings.Contains(bare, want) {
			t.Errorf("run detail should link each log tag to its job (%s); body: %s", want, bare)
		}
	}
	if strings.Contains(bare, "Raw output (admin)") {
		t.Errorf("the bare run should not render the raw-output link; body: %s", bare)
	}

	filtered := getBody(t, ac, base+"/run/52?job=901", http.StatusOK)
	if !strings.Contains(filtered, "Raw output (admin)") ||
		!strings.Contains(filtered, `href="/run/52/raw?job=901"`) {
		t.Errorf("following a job tag should reveal the admin raw-output link; body: %s", filtered)
	}
	if strings.Contains(filtered, `href="/run/52?job=901"`) {
		t.Errorf("a filtered log should not re-link its own job tag; body: %s", filtered)
	}
}
