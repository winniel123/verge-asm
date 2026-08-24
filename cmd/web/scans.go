package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// The Scans monitor (#245, v1 spec §4.1): a read-only window onto the queue so an
// operator can see a scan is in flight and how far along it is, without querying
// Postgres directly. Dispatch, queue_job and batch are Operational — they record
// what the system did, never what is true of the estate — so this page reads them
// freely and the drift engine never sees this read (CONTEXT.md, ADR-0041).
//
// The monitor read is a viewer act. The on-demand trigger (#252, ask 3 of #245)
// is the one mutation this page hosts: an admin may dispatch an enabled scan now,
// and it appears in flight here. The trigger's handler, guardrails and panel live
// in scantrigger.go; scansPage only assembles the panel's read-side data.

// scansHistoryLimit bounds the Dispatch read. A Dispatch is one fan-out of one
// Scan, so recent history is short; 50 covers every enabled Scan's last several
// cadences without paging.
const scansHistoryLimit = 50

// dispatchView is one Dispatch shaped for the monitor: its Scan kind and tick, the
// per-state job counts folded into a completed / in-flight / total progress, and —
// for an in-flight Dispatch — the per-job detail.
type dispatchView struct {
	ID           int64
	ScanKind     string
	DispatchedAt string
	// Live is the count of jobs a retry has not superseded (total − retried).
	// Completed and InFlight partition it; Percent is completed / live.
	Live      int64
	Completed int64
	InFlight  int64
	Done      int64
	Dead      int64
	Percent   int
	// Active is true while any job is ready or running — the Dispatch is in flight.
	Active bool
	Jobs   []jobView
}

// jobView is one queue job in a Dispatch's drill-down: its kind, live state, the
// attempt it is on (attempt > 1 is a retry in progress), the Vantage it runs at
// where it has one, and its Batch outcome once terminal.
type jobView struct {
	ID          int64
	Kind        string
	State       string
	Attempt     int32
	MaxAttempts int32
	Retrying    bool
	Superseded  bool
	Vantage     string
	Batch       string
}

// scansPage renders the queue monitor as the Settings scans sub-tab (#281): the
// Scans currently in flight with their per-job progress, and recent completed
// Dispatches beneath as history. With nothing dispatched it shows an idle state —
// a fact and the next action — never an error or a blank. A viewer reads it.
func (s *server) scansPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderSettings(w, r, acct, settingsForms{tab: "scans"})
}

// fillScansSection assembles the Settings scans sub-tab's read-side data: the
// in-flight Dispatches with their per-job detail, the recent history, the
// self-refresh flag while a scan is running, and — for an admin — the on-demand
// trigger panel (#252), whose "in flight" markers reuse the active kinds computed
// here. A failed panel build degrades to an absent panel rather than 500ing the
// read-only monitor a viewer depends on.
func (s *server) fillScansSection(r *http.Request, acct db.Account, data map[string]any) error {
	ctx := r.Context()
	rows, err := s.store.ListDispatchProgress(ctx, scansHistoryLimit)
	if err != nil {
		return err
	}
	var active, history []dispatchView
	activeKinds := make(map[string]bool)
	for _, row := range rows {
		dv := toDispatchView(row)
		if dv.Active {
			activeKinds[row.ScanKind] = true
			jobs, err := s.store.ListJobsForDispatch(ctx, pgtype.Int8{Int64: row.DispatchID, Valid: true})
			if err != nil {
				return err
			}
			for _, j := range jobs {
				dv.Jobs = append(dv.Jobs, toJobView(j))
			}
			active = append(active, dv)
		} else {
			history = append(history, dv)
		}
	}
	data["Active"] = active
	data["History"] = history
	// A meta refresh keeps the in-flight view current as jobs complete, since the
	// page is server-rendered with no client runtime; it runs only while a scan is
	// in flight, so the idle page does not spin.
	data["Refresh"] = len(active) > 0

	if acct.Role == roleAdmin {
		if trigger, err := s.buildTriggerPanel(r, activeKinds); err != nil {
			log.Printf("web: scans: build trigger panel: %v", err)
		} else {
			data["Trigger"] = trigger
		}
	}
	return nil
}

// Run detail (#297, T2) — the per-run drill-in ported from
// design-system/examples/console/RunDetail.jsx. A "run" is one Dispatch (a fan-out
// of one Scan); the screen reads the same Operational queue corpus the Scans
// monitor does (ADR-0041 bars it from the comparison path, so it never reports
// drift or signal counts). It is the destination of a Drift "Batch detail" entry —
// the route `/run/{id}` is stable so T16 can link straight to it.

// runStage is one step of the run's pipeline, folded from the dispatch's jobs
// grouped by kind. Done renders a filled check, Current an accent ring; Num is the
// 1-based position (shown only while not done), Last drops the trailing connector.
type runStage struct {
	Num     int
	Title   string
	Detail  string
	Done    bool
	Current bool
	Last    bool
}

// runLogLine is one line of the batch log — one queue job's event: its id tag, an
// optional level (a dead job is an error, a superseded or retrying attempt a warn),
// and the terse text (kind · state · vantage · batch).
type runLogLine struct {
	Tag   string
	Level string // "" | "warn" | "error"
	Text  string
}

// runKV is one row of the run's "as configured" parameters.
type runKV struct {
	K, V string
}

// runVantage is one vantage's health in this run: the vantage that looked, a
// latency (not stored, so "—"), and a status folded from its jobs (degraded if any
// dead-lettered, else ok). It is a vantage, never a probe/scanner/agent.
type runVantage struct {
	Name    string
	Latency string
	Status  string // "ok" | "degraded"
}

// runView is one Dispatch shaped for the Run detail drill-in: the header identity
// and batch status, the four sections' data, and the degraded-vantage name that
// raises the outcome callout when one did not finish.
type runView struct {
	ID           int64
	Title        string // the dispatched instant — the h1 and breadcrumb id
	Status       string // "running" | "complete" | "failed" (BatchStatus)
	Scope        string
	Meta         string
	Completed    int64
	Dead         int64
	Active       bool
	Stages       []runStage
	Log          []runLogLine
	Params       []runKV
	Vantages     []runVantage
	Degraded     string
}

// runPage renders the per-run drill-in. The run id is a Dispatch id; the dispatch
// is found in the same recent-history read the monitor uses (no new store method),
// so a run that has aged out of history 404s to the run-missing page rather than
// fabricating a record. A viewer reads it — like the Scans monitor, it is a
// read-only window onto the Operational queue.
func (s *server) runPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	raw := r.PathValue("id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		s.renderMissingRun(w, acct, raw)
		return
	}

	rows, err := s.store.ListDispatchProgress(r.Context(), scansHistoryLimit)
	if err != nil {
		s.serverError(w, "run detail: list dispatches", err)
		return
	}
	var found *db.ListDispatchProgressRow
	for i := range rows {
		if rows[i].DispatchID == id {
			found = &rows[i]
			break
		}
	}
	if found == nil {
		s.renderMissingRun(w, acct, raw)
		return
	}

	dv := toDispatchView(*found)
	jobRows, err := s.store.ListJobsForDispatch(r.Context(), pgtype.Int8{Int64: id, Valid: true})
	if err != nil {
		s.serverError(w, "run detail: list jobs", err)
		return
	}

	view := s.buildRunView(r, dv, jobRows)
	s.render(w, "run", map[string]any{
		"Title": "batch " + view.Title, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "drift",
		"Run":       view,
	})
}

// buildRunView shapes one Dispatch and its jobs into the Run detail view. Every
// value is real: the batch status and job counts off the folded progress, the
// stages / log / vantage health off the jobs, and the parameters off the dispatch
// and its Scan. A superseded (retried) attempt is out of the stage and vantage
// reads — a fresh job replaced it — but stays in the log, marked, as a real event.
func (s *server) buildRunView(r *http.Request, dv dispatchView, jobRows []db.ListJobsForDispatchRow) runView {
	v := runView{
		ID:        dv.ID,
		Title:     dv.DispatchedAt,
		Completed: dv.Completed,
		Dead:      dv.Dead,
		Active:    dv.Active,
		Scope:     "all scopes",
	}
	switch {
	case dv.Active:
		v.Status = "running"
	case dv.Dead > 0:
		v.Status = "failed"
	default:
		v.Status = "complete"
	}

	jobs := make([]jobView, 0, len(jobRows))
	for _, j := range jobRows {
		jobs = append(jobs, toJobView(j))
	}

	v.Vantages = runVantages(jobs)
	for _, vt := range v.Vantages {
		if vt.Status == "degraded" {
			v.Degraded = vt.Name
			break
		}
	}
	v.Stages = runStages(jobs)
	v.Log = runLog(jobs)

	nv := len(v.Vantages)
	v.Meta = dv.ScanKind + " profile"
	if nv > 0 {
		v.Meta += fmt.Sprintf(" · %d %s", nv, plural(nv, "vantage", "vantages"))
	}

	v.Params = []runKV{{K: "Profile", V: dv.ScanKind}}
	if sc, err := s.store.GetScanByKind(r.Context(), dv.ScanKind); err == nil {
		v.Params = append(v.Params, runKV{K: "Cadence", V: cadenceLabel(sc.CadenceSeconds)})
	}
	if dv.DispatchedAt != "" {
		v.Params = append(v.Params, runKV{K: "Dispatched", V: dv.DispatchedAt})
	}
	v.Params = append(v.Params, runKV{K: "Jobs", V: strconv.FormatInt(dv.Live, 10)})
	if nv > 0 {
		v.Params = append(v.Params, runKV{K: "Vantages", V: strconv.Itoa(nv)})
	}
	return v
}

// runStages folds the dispatch's jobs into pipeline steps, grouped by job kind in
// first-seen order. A stage with no in-flight job is done (a filled check); one
// still running or ready is current (an accent ring). Superseded attempts are
// excluded — the fresh job that replaced them carries the count.
func runStages(jobs []jobView) []runStage {
	var order []string
	idx := map[string]int{}
	type agg struct{ total, done, dead, inflight int }
	var aggs []agg
	for _, j := range jobs {
		if j.Superseded {
			continue
		}
		i, ok := idx[j.Kind]
		if !ok {
			i = len(order)
			idx[j.Kind] = i
			order = append(order, j.Kind)
			aggs = append(aggs, agg{})
		}
		aggs[i].total++
		switch j.State {
		case "done":
			aggs[i].done++
		case "dead":
			aggs[i].dead++
		case "ready", "running":
			aggs[i].inflight++
		}
	}
	stages := make([]runStage, 0, len(order))
	for i, k := range order {
		a := aggs[i]
		detail := fmt.Sprintf("%d of %d done", a.done, a.total)
		if a.dead > 0 {
			detail += fmt.Sprintf(" · %d dead-lettered", a.dead)
		}
		stages = append(stages, runStage{
			Num:     i + 1,
			Title:   k,
			Detail:  detail,
			Done:    a.inflight == 0,
			Current: a.inflight > 0,
			Last:    i == len(order)-1,
		})
	}
	return stages
}

// runLog turns the dispatch's jobs into the batch log — one line per job, the id
// as its tag, a level from its state (a dead job errors, a superseded or retrying
// attempt warns), and the terse kind · state · vantage · batch text. Every line is
// a real queue event; nothing is invented.
func runLog(jobs []jobView) []runLogLine {
	out := make([]runLogLine, 0, len(jobs))
	for _, j := range jobs {
		level := ""
		switch {
		case j.State == "dead":
			level = "error"
		case j.Superseded || j.Retrying:
			level = "warn"
		}
		text := j.Kind + " · " + j.State
		if j.Vantage != "" {
			text += " · " + j.Vantage
		}
		if j.Batch != "" {
			text += " · " + j.Batch
		}
		out = append(out, runLogLine{Tag: "#" + strconv.FormatInt(j.ID, 10), Level: level, Text: text})
	}
	return out
}

// runVantages folds the jobs' vantages into per-vantage health, in first-seen
// order. A vantage is degraded if any of its non-superseded jobs dead-lettered,
// else ok. Latency is not stored, so it reads "—" (as the example's does). It is a
// vantage that looked, never a probe/scanner/agent.
func runVantages(jobs []jobView) []runVantage {
	var order []string
	seen := map[string]bool{}
	dead := map[string]bool{}
	for _, j := range jobs {
		if j.Vantage == "" || j.Superseded {
			continue
		}
		if !seen[j.Vantage] {
			seen[j.Vantage] = true
			order = append(order, j.Vantage)
		}
		if j.State == "dead" {
			dead[j.Vantage] = true
		}
	}
	out := make([]runVantage, 0, len(order))
	for _, n := range order {
		status := "ok"
		if dead[n] {
			status = "degraded"
		}
		out = append(out, runVantage{Name: n, Latency: "—", Status: status})
	}
	return out
}

// plural picks the singular or plural noun for a count.
func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// toDispatchView folds one Dispatch's per-state job counts into progress. Live work
// excludes retried rows — a retry is a fresh job, so each retried row has a
// successor and counting it would double the denominator. Completed is the terminal
// outcomes (done + dead); a dead-lettered job is finished, not pending, so it counts
// toward progress. Percent is 0 while there is no live work rather than a divide.
func toDispatchView(row db.ListDispatchProgressRow) dispatchView {
	live := row.Total - row.Retried
	completed := row.Done + row.Dead
	inFlight := row.Ready + row.Running

	percent := 0
	if live > 0 {
		percent = int(completed * 100 / live)
	}

	dv := dispatchView{
		ID:        row.DispatchID,
		ScanKind:  row.ScanKind,
		Live:      live,
		Completed: completed,
		InFlight:  inFlight,
		Done:      row.Done,
		Dead:      row.Dead,
		Percent:   percent,
		Active:    inFlight > 0,
	}
	if row.CreatedAt.Valid {
		dv.DispatchedAt = row.CreatedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
	}
	return dv
}

// --- scan schedule instants (P0.4, #445) ----------------------------------
//
// The Dashboard header sub-line renders two instants — "last full scan Xm ago ·
// next in Yh Zm" (Dashboard.jsx, PARITY-CHART P0.4/P2.1). Both are real reads over
// the scheduler's own corpora, assembled here so the home handler (auth.go
// dashboardData) exposes them; the markup that renders them is P2.1's.
//
// "Last" is the instant of the most recent Dispatch across every Scan kind — the
// last time any measurement actually fanned out (dispatch.created_at, the same
// "when did this scan start" instant the monitor reads, #245). "Next" is the
// soonest upcoming cadence boundary among the ENABLED Scans, floored exactly the
// way the dispatcher floors a tick (internal/queue.scheduledTick) so the figure
// matches when the worker will really fire. Dispatch and Scan are Operational, so
// this read never touches the comparison path (ADR-0041).

// scanScheduleView is the header sub-line's two instants and their humanized forms.
// Has* is false where the datum is genuinely absent — no Dispatch has ever fanned
// out, or no enabled Scan carries a cadence — so the surface renders the honest
// "never scanned" state rather than a fabricated instant.
type scanScheduleView struct {
	HasLast    bool
	LastScanAt time.Time     // the most recent Dispatch's fan-out instant (UTC)
	SinceLast  time.Duration // now − LastScanAt, floored at zero
	LastAgo    string        // humanized SinceLast, e.g. "38m"

	HasNext    bool
	NextScanAt time.Time     // the soonest enabled-Scan cadence boundary after now (UTC)
	UntilNext  time.Duration // NextScanAt − now, floored at zero
	NextIn     string        // humanized UntilNext, e.g. "5h 22m"
}

// scanSchedule assembles the last/next scan instants. Each half is best-effort: a
// failed read logs and leaves its Has* false rather than 500ing the landing page a
// viewer depends on, matching the rest of dashboardData's degradation discipline.
func (s *server) scanSchedule(ctx context.Context) scanScheduleView {
	now := s.now().UTC()
	var v scanScheduleView

	// Last: ListDispatchProgress is newest-first (ORDER BY d.id DESC), so the first
	// row carrying a real created_at is the most recent fan-out.
	if rows, err := s.store.ListDispatchProgress(ctx, scansHistoryLimit); err != nil {
		log.Printf("web: dashboard: scan schedule: list dispatches: %v", err)
	} else {
		for _, r := range rows {
			if r.CreatedAt.Valid {
				v.LastScanAt = r.CreatedAt.Time.UTC()
				if d := now.Sub(v.LastScanAt); d > 0 {
					v.SinceLast = d
				}
				v.LastAgo = humanizeDuration(v.SinceLast)
				v.HasLast = true
				break
			}
		}
	}

	// Next: the soonest cadence boundary among the enabled Scans. A disabled Scan is
	// off the dispatcher's cadence, so it never contributes a next tick.
	if scans, err := s.store.ListScans(ctx); err != nil {
		log.Printf("web: dashboard: scan schedule: list scans: %v", err)
	} else {
		var best time.Time
		for _, sc := range scans {
			if !sc.Enabled || sc.CadenceSeconds <= 0 {
				continue
			}
			next := nextCadenceBoundary(now, sc.CadenceSeconds)
			if best.IsZero() || next.Before(best) {
				best = next
			}
		}
		if !best.IsZero() {
			v.NextScanAt = best
			if d := best.Sub(now); d > 0 {
				v.UntilNext = d
			}
			v.NextIn = humanizeCountdown(v.UntilNext)
			v.HasNext = true
		}
	}

	return v
}

// nextCadenceBoundary is the next dispatch tick strictly after now for a Scan of the
// given cadence: the same flooring internal/queue.scheduledTick uses, plus one
// cadence. Missed ticks are not caught up (dispatch is idempotent on the tick), so
// the next fan-out is always the next boundary from now, never a stale past one.
func nextCadenceBoundary(now time.Time, cadenceSeconds int64) time.Time {
	secs := cadenceSeconds
	if secs <= 0 {
		secs = 1
	}
	floored := (now.UTC().Unix() / secs) * secs
	return time.Unix(floored+secs, 0).UTC()
}

// humanizeCountdown renders a countdown as the spec's two-unit figure — "2d 3h",
// "5h 22m", or "47m" (Dashboard.jsx "next in 5h 22m"). Under a minute reads "<1m"
// so an imminent tick never renders a bare 0. It is the countdown twin of
// humanizeDuration's single-unit "ago" figure.
func humanizeCountdown(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "<1m"
	case d >= 24*time.Hour:
		days := int(d / (24 * time.Hour))
		hrs := int((d % (24 * time.Hour)) / time.Hour)
		return fmt.Sprintf("%dd %dh", days, hrs)
	case d >= time.Hour:
		hrs := int(d / time.Hour)
		mins := int((d % time.Hour) / time.Minute)
		return fmt.Sprintf("%dh %dm", hrs, mins)
	default:
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
}

// toJobView shapes one queue job for the drill-down. A 'retried' row is a
// superseded attempt the fresh job replaced; a ready-or-running job past its first
// attempt is a retry currently in flight.
func toJobView(j db.ListJobsForDispatchRow) jobView {
	v := jobView{
		ID:          j.ID,
		Kind:        j.Kind,
		State:       j.State,
		Attempt:     j.Attempt,
		MaxAttempts: j.MaxAttempts,
		Superseded:  j.State == "retried",
		Retrying:    j.Attempt > 1 && (j.State == "ready" || j.State == "running"),
	}
	if j.VantageName.Valid {
		v.Vantage = j.VantageName.String
	}
	if j.BatchOutcome.Valid {
		v.Batch = j.BatchOutcome.String
	}
	return v
}
