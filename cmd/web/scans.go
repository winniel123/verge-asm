package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/db"
)

// The RunDetail screen (screen 9, #565/#566) is served byte-for-byte from the frozen
// design-owned design-system/templates/rundetail.tmpl (package v3.8.0, WORKFLOW v4),
// which replaces the repo-authored templates_rundetail.go const (deleted). The tmpl
// keeps the "run" define and the .Run struct, renders inside the full app chrome
// ({{template "chrome" .}}), and styles against the design token vocabulary — so the
// render opts in with DesignTokens:true (the "head" block inlines tokens/*.css only
// then). rundetail.tmpl auto-embeds through designfs's existing templates/*.tmpl glob,
// so no designfs.go change is needed.
//
// Reconciliations SPEC-CHANGE #20 (ruled): the Outcome card is the spec's — .Completed/
// .Dead RETIRE in favour of .Transitions + .NewSignals, the read-side join of the
// derived stores (transitions folded from THIS batch's diff, signals first raised in
// it) per the 2026-08-24 binding ruling (#20a); joinRunOutcome builds it below. ADR-0041's
// corpus separation for dispatch EXECUTION stands — the comparison path is untouched; this
// only JOINS derived stores on the read path. .Degraded becomes nullable {Vantage,Detail}
// (#20). Log levels render as colored text, not level pills (#20e — encoded in the frozen
// tmpl). The "run-missing" define RETIRES (#20d): a missing run routes to error.tmpl's
// missing-run kind (renderMissingRun, landed screen 2), never a rundetail-local define.
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/rundetail.tmpl"))

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
	ID int64
	// Href is the run-detail link (/runs/{dispatch}) the Running-now scan kind and the
	// history rows carry (DF-F3): every dispatch in the recent window has a run page.
	Href         string
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
	// Status is the Dispatch's recorded disposition (DF-F4b): 'fanned-out' for a natural
	// run, or the operator-ended 'stopped' / 'terminated' the stop/terminate acts write.
	// buildRunView passes it to runStatusLabel so a stopped/terminated drill-in renders
	// its real terminal badge instead of the live-derived one.
	Status string
	Jobs   []jobView
}

// jobView is one queue job in a Dispatch's drill-down: its kind, live state, the
// attempt it is on (attempt > 1 is a retry in progress), the Vantage it runs at
// where it has one, and its Batch outcome once terminal.
type jobView struct {
	ID int64
	// Href is the per-job live-log link (/runs/{run}?job={id}, DF-F3b) each Running-now
	// job id carries. Set by fillScansSection, which knows the run (dispatch) id; empty
	// on a jobView built outside that context (the run drill-in's own job list).
	Href        string
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
func (s *server) fillScansSection(r *http.Request, acct db.Account, f settingsForms, data map[string]any) error {
	ctx := r.Context()
	// The full-range (cold) tier opt-in, relocated from /scope (#21d): every declared
	// scope with its opt-in state, and whether the tier is on (at least one scope in).
	// Best-effort — a read failure degrades the region to no scopes rather than 500ing
	// the whole tab. The coldError echo comes from a rejected opt-in POST.
	if seeds, serr := s.store.ListSeeds(ctx); serr == nil {
		if optedIn, oerr := s.store.ListColdScopeSeedIds(ctx); oerr == nil {
			data["ColdScopes"] = toColdScopeViews(toSeedViews(seeds), optedIn)
			data["ColdEnabled"] = len(optedIn) > 0
		}
	}
	data["ColdError"] = f.coldError
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
				jv := toJobView(j)
				// The per-job live-log link (DF-F3b): each Running-now job id links to the
				// run page filtered to its own rows. The run is the dispatch this job belongs
				// to, so the href is /runs/{dispatch}?job={id}.
				jv.Href = fmt.Sprintf("/runs/%d?job=%d", row.DispatchID, jv.ID)
				dv.Jobs = append(dv.Jobs, jv)
			}
			active = append(active, dv)
		} else {
			history = append(history, dv)
		}
	}
	data["Active"] = active
	data["History"] = history

	// The stop / terminate PRG dialogs (DF-F4): an admin opens one by navigating
	// ?stop={id} or ?terminate={id}. The dialog reads its counts live from the already-
	// gathered progress rows — Pending is the ready jobs, Running the running jobs — so
	// no extra read is needed. Only an in-flight dispatch owns a dialog; a ?stop for a
	// concluded or unknown id renders none (the POST guards it too). Admin-only, matching
	// the row controls that link here.
	if acct.Role == roleAdmin {
		q := r.URL.Query()
		if id, ok := parseDispatchID(q.Get("stop")); ok {
			if row, found := findDispatchRow(rows, id); found && toDispatchView(row).InFlight > 0 {
				data["StopTarget"] = map[string]any{
					"ID": row.DispatchID, "ScanKind": row.ScanKind,
					"Pending": row.Ready, "Running": row.Running,
				}
			}
		}
		if id, ok := parseDispatchID(q.Get("terminate")); ok {
			if row, found := findDispatchRow(rows, id); found && toDispatchView(row).InFlight > 0 {
				data["TerminateTarget"] = map[string]any{
					"ID": row.DispatchID, "ScanKind": row.ScanKind,
					"Running": row.Running,
				}
			}
		}
	}
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

// parseDispatchID reads a dispatch id from a query or form value, refusing the empty,
// unparseable, or non-positive so a hand-crafted ?stop=/?terminate= is a clean no-op.
func parseDispatchID(raw string) (int64, bool) {
	if raw == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// findDispatchRow locates one dispatch's progress row in the recent-history read, the
// same read the monitor and the run drill-in use (no new store method). A dispatch that
// has aged past the window is not found — treated as already concluded, never fabricated.
func findDispatchRow(rows []db.ListDispatchProgressRow, id int64) (db.ListDispatchProgressRow, bool) {
	for i := range rows {
		if rows[i].DispatchID == id {
			return rows[i], true
		}
	}
	return db.ListDispatchProgressRow{}, false
}

// concludedFlash stashes the shared "already concluded" danger toast for a stop/terminate
// aimed at an unknown or already-terminal dispatch, and redirects back to the tab. It is
// the honest refusal: nothing was ended because there was nothing in flight to end.
func (s *server) concludedFlash(w http.ResponseWriter, r *http.Request, acct db.Account, detail string) {
	s.flashRedirect(w, r, acct.ID, "/settings?tab=scans", "danger", "Dispatch already concluded", detail)
}

// stopScan gracefully ends a Dispatch in flight (DF-F4). Its pending (ready) jobs are
// cancelled — they leave the claimable set at once, since ClaimJob selects state='ready'
// alone — while its running jobs are left to finish and commit their batches (nothing
// already observed is discarded). The dispatch is recorded 'stopped'. Admin-gated
// (requireAdmin) exactly as the trigger is; a non-admin POST is 403 before this runs.
func (s *server) stopScan(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, ok := parseDispatchID(r.FormValue("id"))
	if !ok {
		s.concludedFlash(w, r, acct, "There was nothing in flight to stop.")
		return
	}
	rows, err := s.store.ListDispatchProgress(r.Context(), scansHistoryLimit)
	if err != nil {
		s.serverError(w, "stop scan: list dispatches", err)
		return
	}
	row, found := findDispatchRow(rows, id)
	if !found || toDispatchView(row).InFlight == 0 {
		// Unknown id, or a dispatch that already finished or was already ended: the disabled
		// cold tier is never in flight, so this also refuses a stop aimed at it.
		s.concludedFlash(w, r, acct, "It has already finished or been ended — nothing was stopped.")
		return
	}
	pid := pgtype.Int8{Int64: id, Valid: true}
	n, err := s.store.CancelReadyJobsForDispatch(r.Context(), pid)
	if err != nil {
		s.serverError(w, "stop scan: cancel pending jobs", err)
		return
	}
	if err := s.store.SetDispatchStatus(r.Context(), db.SetDispatchStatusParams{ID: id, Status: "stopped"}); err != nil {
		s.serverError(w, "stop scan: record status", err)
		return
	}
	// Running jobs are left to finish (the stop contract); the row's running count at
	// action time is how many are finishing.
	desc := fmt.Sprintf("%d pending %s cancelled · %d running finishing",
		n, plural(int(n), "job", "jobs"), row.Running)
	s.flashRedirect(w, r, acct.ID, "/settings?tab=scans", "neutral", "Dispatch stopped", desc)
}

// terminateScan hard-kills a Dispatch in flight (DF-F4). Both its pending AND running
// jobs are cancelled; a running job's guarded terminal write then affects no row and the
// worker rolls its transaction back, so its uncommitted batch and observations are
// discarded (internal/queue/worker.go). Batches already committed stand — observations are
// append-only. The dispatch is recorded 'terminated'. Admin-gated like the trigger.
func (s *server) terminateScan(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, ok := parseDispatchID(r.FormValue("id"))
	if !ok {
		s.concludedFlash(w, r, acct, "There was nothing in flight to terminate.")
		return
	}
	rows, err := s.store.ListDispatchProgress(r.Context(), scansHistoryLimit)
	if err != nil {
		s.serverError(w, "terminate scan: list dispatches", err)
		return
	}
	row, found := findDispatchRow(rows, id)
	if !found || toDispatchView(row).InFlight == 0 {
		s.concludedFlash(w, r, acct, "It has already finished or been ended — nothing was terminated.")
		return
	}
	pid := pgtype.Int8{Int64: id, Valid: true}
	n, err := s.store.CancelActiveJobsForDispatch(r.Context(), pid)
	if err != nil {
		s.serverError(w, "terminate scan: cancel jobs", err)
		return
	}
	if err := s.store.SetDispatchStatus(r.Context(), db.SetDispatchStatusParams{ID: id, Status: "terminated"}); err != nil {
		s.serverError(w, "terminate scan: record status", err)
		return
	}
	desc := fmt.Sprintf("%d %s stopped", n, plural(int(n), "job", "jobs"))
	s.flashRedirect(w, r, acct.ID, "/settings?tab=scans", "neutral", "Scan terminated", desc)
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
// and the terse text (kind · state · vantage · batch). JobID carries the source
// queue-job id so the per-job filter (DF-F3b, ?job={id}) can narrow the log
// server-side; the frozen tmpl reads only Tag/Level/Text, never JobID.
type runLogLine struct {
	JobID int64
	Tag   string
	Level string // "" | "warn" | "error"
	Text  string
}

// runJobFilter is the loghead per-job filter chip (DF-F3b): the queue-job the log is
// narrowed to (its id, kind and — where it has one — vantage) and the bare run route
// the × clears back to. Nil on runView renders no chip ({{with .JobFilter}}); the
// filter itself is applied server-side (applyJobFilter), never in the tmpl.
type runJobFilter struct {
	ID        int64
	Kind      string
	Vantage   string
	ClearHref string
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

// runDegraded is the nullable Outcome callout datum (#20): the vantage that fell
// short in this batch and the terse reason ("missed 2 of 3 checks"). Nil renders
// nothing — the frozen tmpl's {{with .Degraded}} guards it. The tmpl supplies the
// surrounding sentence, so Detail carries only the middle clause.
type runDegraded struct {
	Vantage string
	Detail  string
}

// runView is one Dispatch shaped for the Run detail drill-in: the header identity
// and batch status, the four sections' data, the batch-joined Outcome (transitions +
// new signals, #20a), and the nullable degraded-vantage callout.
type runView struct {
	ID     int64
	Title  string // the dispatched instant — the h1 and breadcrumb id
	Status string // "running" | "complete" | "failed" (BatchStatus)
	Scope  string
	Meta   string
	// Transitions / NewSignals are the Outcome card's batch join (#20a, ruled):
	// transitions folded from THIS batch's diff stage and signals first raised in it,
	// rendered as strings. Both read "—" until the run's diff stage has concluded.
	Transitions string
	NewSignals  string
	Active      bool
	Stages      []runStage
	Log         []runLogLine
	Params      []runKV
	Vantages    []runVantage
	// Degraded is nullable (#20): a *runDegraded, nil where no vantage fell short.
	Degraded *runDegraded
	// JobFilter is nullable (DF-F3b): set when the request carries ?job={id}; .Log has
	// already been narrowed to that job's rows server-side by the time it renders.
	JobFilter *runJobFilter
	// StreamHref is the per-job stdout long-poll endpoint the frozen rundetail.tmpl
	// tails while a job is running (R4-D7 #761). It is nullable — empty when no job is
	// streaming — and left "" here: with it empty the tmpl's {{if .StreamHref}} blocks
	// skip and the static log renders. The actual long-poll endpoint is a separate
	// follow-up for #761; this field only satisfies the frozen template's hole.
	StreamHref string
}

// runPage renders the per-run drill-in. The run id is a Dispatch id; the dispatch
// is found in the same recent-history read the monitor uses (no new store method),
// so a run that has aged out of history 404s to the run-missing page rather than
// fabricating a record. A viewer reads it — like the Scans monitor, it is a
// read-only window onto the Operational queue.
func (s *server) runPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	raw := r.PathValue("id")

	// VERGE_DEV pixel-parity path (#565/#566). The frozen rundetail.tmpl renders the run
	// 1407 drill-in — the four done stages, the 7-line log (one warn, one error), the
	// Outcome card (7 transitions · 3 new signals), the nullable ap-south-1 degraded
	// callout, the 5 params and the 3 vantages — a curated corpus whose exact figures are
	// the design's, not a live-queue read. Reproducing them from the live derivations would
	// mean fabricating domain data, which SPEC-CHANGE forbids — so, exactly as the
	// Exposure/Coverage screens pin their dev fixture and serve it under devMode with a
	// drift test (TestRunDetailFixtureMatchesPackage), runPage serves the pinned
	// fixtures.json → rundetail slice for the completed fixture id (1407) here, and the
	// running-run demo for 1409 (#35). The pinned MISSING id (1408) and any other id route
	// to the missing-run ErrorPage — the #20d wiring — so the "run-missing" state is proven
	// end-to-end. A real deployment (devMode == false) falls through to the honest live
	// reads + batch join below.
	if s.devMode {
		if raw == devRunDetailID {
			s.render(w, r, "run", s.runDetailFixtureData(acct))
			return
		}
		// The running run (DF-F3/F3b) drill-in: SPEC-CHANGE #35 gave the Settings active
		// dispatch its own id, 1409 (fixtures.json), distinct from the error screen's
		// missing-run demo, which keeps 1408. So /runs/1409 now routes to the live-tail
		// running fixture (re-enabled with package v3.17.0) and gets its own G2 golden
		// (rundetail·running), while bare /runs/1408 stays the missing-run route the frozen
		// error golden pins. The ?job= per-job filter rides through to runningRunFixtureData.
		if raw == devRunningRunID {
			s.render(w, r, "run", s.runningRunFixtureData(acct, r.URL.Query().Get("job"), r.URL.Path))
			return
		}
		// Any other id (1408, the pinned MISSING id) routes to the missing-run ErrorPage.
		s.renderMissingRun(w, r, acct, raw)
		return
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		s.renderMissingRun(w, r, acct, raw)
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
		s.renderMissingRun(w, r, acct, raw)
		return
	}

	dv := toDispatchView(*found)
	jobRows, err := s.store.ListJobsForDispatch(r.Context(), pgtype.Int8{Int64: id, Valid: true})
	if err != nil {
		s.serverError(w, "run detail: list jobs", err)
		return
	}

	view := s.buildRunView(r, dv, jobRows)
	s.render(w, r, "run", map[string]any{
		"Title": "batch " + view.Title, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "drift",
		// rundetail.tmpl styles against the design token vocabulary; the "head" block
		// inlines tokens/*.css only when this datum is set (as Coverage/Exposure do).
		"DesignTokens": true,
		// While the run is in flight the head's meta-refresh hole tails the log on a
		// cadence (DF-F3); on conclusion it returns 0 and the terminal state stands. The
		// filter lives in the URL, so a refresh preserves it.
		"Refresh": runRefresh(view.Status),
		// StreamHref is set at BOTH scopes the frozen rundetail.tmpl reads: .Run.StreamHref
		// drives the data-stream attribute inside {{with .Run}} (line ~151), while the
		// root-scope {{if .StreamHref}} that emits the long-poll <script> (line ~216, outside
		// the with) reads this top-level datum. Same value, so the attribute and the script
		// that consumes it are always emitted together (or both skipped when "").
		"StreamHref": view.StreamHref,
		"Run":        view,
	})
}

// buildRunView shapes one Dispatch and its jobs into the Run detail view. Every
// value is real: the batch status and job counts off the folded progress, the
// stages / log / vantage health off the jobs, and the parameters off the dispatch
// and its Scan. A superseded (retried) attempt is out of the stage and vantage
// reads — a fresh job replaced it — but stays in the log, marked, as a real event.
func (s *server) buildRunView(r *http.Request, dv dispatchView, jobRows []db.ListJobsForDispatchRow) runView {
	v := runView{
		ID:     dv.ID,
		Title:  dv.DispatchedAt,
		Active: dv.Active,
		Scope:  "all scopes",
	}
	// A dispatch stopped/terminated via DF-F4 carries a recorded outcome the run page
	// renders as its terminal status (runStatusLabel — the literal word "stopped" /
	// "terminated" in the .Status hole; the .rd-batch stopped/terminated badge treatment
	// landed in rundetail.tmpl with package v3.17.0, #34/DF-F4b). That recording is the
	// Scans ticket's (#633) write side; it now reaches this read through the dispatch
	// status the progress query surfaces (ListDispatchProgress → dv.Status), so a stopped
	// or terminated drill-in renders its real disposition. A natural run's 'fanned-out'
	// is not a terminal outcome, so runStatusLabel falls through to the live
	// running/failed/complete derivation — unchanged for every non-ended dispatch.
	v.Status = runStatusLabel(dv.Active, dv.Dead, dispatchOutcome(dv.Status))

	jobs := make([]jobView, 0, len(jobRows))
	for _, j := range jobRows {
		jobs = append(jobs, toJobView(j))
	}

	v.Vantages = runVantages(jobs)
	v.Stages = runStages(jobs)
	v.Log = runLog(jobs)

	// The Outcome card's batch join (#20a, ruled): .Transitions and .NewSignals join the
	// derived stores on the READ path (the comparison path is untouched — ADR-0041 stands
	// for dispatch execution). The run's batch id set is the batch(es) its jobs committed
	// under; joinRunOutcome counts the drift transitions folded from those batches and the
	// signals first raised in them. Both read "—" until the run's diff stage has concluded.
	batchIDs := dispatchBatchIDs(jobRows)
	out := s.joinRunOutcome(r.Context(), batchIDs)
	if out.Concluded {
		v.Transitions = strconv.Itoa(out.Transitions)
		v.NewSignals = strconv.Itoa(out.NewSignals)
	} else {
		v.Transitions = "—"
		v.NewSignals = "—"
	}

	// The degraded callout is nullable (#20): raised only where a vantage fell short in
	// this batch, with the terse reason its jobs record. A healthy run leaves it nil and
	// the frozen tmpl's {{with .Degraded}} renders nothing.
	v.Degraded = runDegradedFrom(jobs)

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

	// The per-job filter (DF-F3b): with ?job={id} the log is narrowed to that job's rows
	// server-side and the loghead chip is set — the tmpl renders whatever .Log it is
	// handed, never filtering client-side. The filter lives in the URL so it survives the
	// live meta-refresh; the × clears back to the bare run route.
	applyJobFilter(&v, r.URL.Query().Get("job"), r.URL.Path, jobs)

	// StreamHref (R4-D7 #761, collision #40 a-scoped): the per-job live-progress long-poll
	// the frozen tmpl tails while work is in flight. It is set only while the in-scope job
	// is non-terminal — a ?job filter scopes it to that job, the bare run to any running
	// job — and left "" once terminal, so the static state-derived .Log stands (nothing new
	// is persisted). The base is the run's own path + /stream; the client appends ?after=.
	v.StreamHref = runStreamHref(r.URL.Path, v.JobFilter, dv.Active, jobs)
	return v
}

// --- per-job live progress stream (R4-D7 #761, SPEC-CHANGE collision #40 a-scoped) -------
//
// The frozen rundetail.tmpl tails a running job by long-polling a per-job endpoint and
// APPENDING the JSON lines it returns after a cursor. The ruling (#40) scopes this to the
// LIVE TRANSPORT only: the endpoint RE-DERIVES the run's state log on each poll from
// queue_job — the exact ListJobsForDispatch → toJobView → runLog path the static .Log uses
// — and returns the lines after the client's cursor. It PERSISTS NOTHING NEW: there is no
// raw-stdout store (ADR-0041 corpus retention + instance-privacy posture are untouched), so
// on a job's conclusion the persisted state-derived .Log stands exactly as today. The
// "events" are the incremental state-derived lines as jobs advance ready→running→terminal —
// including a retry's fresh queue_job row and its superseded (retried) attempt, which runLog
// renders as real events. The redaction/format is runLog's verbatim; no new format is
// invented. Liveness is a bounded poll+timeout long-poll (the queue also carries
// LISTEN/NOTIFY on notifyChannel "queue_job", pg_notify in internal/queue/queue.go; polling
// is the ruling's accepted alternative and keeps the endpoint on the same read seam the page
// uses, testable without a live Postgres): it holds briefly for new lines, then returns an
// empty {done:false} nudge so the client re-polls — it never blocks a connection forever.

// runStreamHold caps one long-poll hold; runStreamPoll is how often it re-derives the state
// log while holding. Past the hold the endpoint returns an empty nudge so the client re-polls.
const (
	runStreamHold = 25 * time.Second
	runStreamPoll = time.Second
)

// jobActive reports whether a queue job is still in flight — ready or running. Every other
// state (done, dead, retried, cancelled) is terminal: the stream is done for that job.
func jobActive(state string) bool {
	return state == "ready" || state == "running"
}

// runStreamHref is the per-job live-progress endpoint the frozen rundetail.tmpl tails, or
// "" when nothing is streaming (the static .Log stands). A ?job filter scopes it to that
// job and streams only while the job is non-terminal (an unknown/aged id → "", static log);
// the bare run streams while any job is still in flight. The base is the run's own request
// path + /stream so it rides whichever alias (/run or /runs) the viewer is on; the client
// appends ?after={cursor} (and the frozen JS handles the ?/& join for a job-scoped base).
func runStreamHref(runPath string, filter *runJobFilter, runActive bool, jobs []jobView) string {
	base := runPath + "/stream"
	if filter != nil {
		for _, j := range jobs {
			if j.ID == filter.ID {
				if jobActive(j.State) {
					return base + "?job=" + strconv.FormatInt(filter.ID, 10)
				}
				return "" // the viewed job is terminal — the static filtered .Log stands
			}
		}
		return "" // unknown/aged job id — nothing to stream
	}
	if runActive {
		return base
	}
	return ""
}

// runStreamLine is one live-progress line on the wire — the SAME tag/level/text the static
// .Log renders (runLogLine), with the internal JobID dropped: the stream redacts EXACTLY as
// .Log does (collision #40) and invents no new format.
type runStreamLine struct {
	Tag   string `json:"tag"`
	Level string `json:"level"`
	Text  string `json:"text"`
}

// runStreamResp is the long-poll body the frozen client expects: the state-derived lines
// after the client's cursor, the new cursor (next), and whether the in-scope work concluded.
type runStreamResp struct {
	Lines []runStreamLine `json:"lines"`
	Next  int             `json:"next"`
	Done  bool            `json:"done"`
}

// deriveRunStream re-derives the run's state log (ListJobsForDispatch → toJobView → runLog)
// and, with a numeric ?job, narrows it to that job exactly as applyJobFilter narrows the
// page's .Log. done is true once the in-scope work is terminal: for a ?job filter, that one
// job (an unknown id counts as done — nothing will stream); for the bare run, no job still
// ready or running. It reuses the same read seam and pure folds the page uses, so a live run
// and the stream never disagree, and it persists nothing.
func (s *server) deriveRunStream(ctx context.Context, dispatchID int64, jobParam string) ([]runLogLine, bool, error) {
	jobRows, err := s.store.ListJobsForDispatch(ctx, pgtype.Int8{Int64: dispatchID, Valid: true})
	if err != nil {
		return nil, false, err
	}
	jobs := make([]jobView, 0, len(jobRows))
	for _, j := range jobRows {
		jobs = append(jobs, toJobView(j))
	}
	log := runLog(jobs)

	if jobParam != "" {
		if jobID, perr := strconv.ParseInt(jobParam, 10, 64); perr == nil {
			filtered := make([]runLogLine, 0, len(log))
			for _, ln := range log {
				if ln.JobID == jobID {
					filtered = append(filtered, ln)
				}
			}
			done := true
			for _, j := range jobs {
				if j.ID == jobID {
					done = !jobActive(j.State)
					break
				}
			}
			return filtered, done, nil
		}
	}

	done := true
	for _, j := range jobs {
		if jobActive(j.State) {
			done = false
			break
		}
	}
	return log, done, nil
}

// runStream serves the per-job live progress long-poll (#761, collision #40). It re-derives
// the run's state log (deriveRunStream) and returns the lines after ?after={cursor} as JSON
// {lines:[{tag,level,text}], next, done}, where next is the new cursor (the line count) and
// done marks the in-scope work terminal. A ?job={id} narrows the log to that job. It holds up
// to runStreamHold for new lines (re-deriving every runStreamPoll), then returns an empty
// {done:false} nudge so the client re-polls — never blocking forever. Login-gated like
// runPage; it mutates nothing and persists nothing. A malformed id or after is a clean read
// (id → an immediate done; after → treated as 0).
func (s *server) runStream(w http.ResponseWriter, r *http.Request, _ db.Account) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeAPIJSON(w, runStreamResp{Lines: []runStreamLine{}, Next: 0, Done: true})
		return
	}
	after := 0
	if n, perr := strconv.Atoi(r.URL.Query().Get("after")); perr == nil && n > 0 {
		after = n
	}
	jobParam := r.URL.Query().Get("job")
	ctx := r.Context()

	timeout := time.NewTimer(runStreamHold)
	defer timeout.Stop()
	ticker := time.NewTicker(runStreamPoll)
	defer ticker.Stop()

	for {
		lines, done, derr := s.deriveRunStream(ctx, id, jobParam)
		if derr != nil {
			apiReadError(w, "run detail: stream", derr)
			return
		}
		if len(lines) > after || done {
			from := min(after, len(lines))
			out := make([]runStreamLine, 0, len(lines)-from)
			for _, ln := range lines[from:] {
				out = append(out, runStreamLine{Tag: ln.Tag, Level: ln.Level, Text: ln.Text})
			}
			writeAPIJSON(w, runStreamResp{Lines: out, Next: len(lines), Done: done})
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-timeout.C:
			writeAPIJSON(w, runStreamResp{Lines: []runStreamLine{}, Next: len(lines), Done: false})
			return
		case <-ticker.C:
		}
	}
}

// runStatusLabel is the run page's batch-status word (the frozen .Status hole, which is
// both the rd-batch CSS class and the visible label). A dispatch stopped or terminated via
// DF-F4 renders its recorded outcome verbatim — the literal "stopped" / "terminated", which
// rundetail.tmpl styles with the .rd-batch stopped (warn) / terminated (danger-outline)
// treatments landed in package v3.17.0 (#34/DF-F4b). Absent an outcome it is the live
// derivation: in-flight → running, any dead-lettered job → failed, else complete.
func runStatusLabel(active bool, dead int64, outcome string) string {
	switch outcome {
	case "stopped", "terminated":
		return outcome
	}
	switch {
	case active:
		return "running"
	case dead > 0:
		return "failed"
	default:
		return "complete"
	}
}

// dispatchOutcome maps a Dispatch's recorded status to the terminal outcome runStatusLabel
// honors. Only the operator-ended dispositions ('stopped' / 'terminated', migration 22901)
// are terminal outcomes; a natural run's 'fanned-out' (and any absent status) yields "", so
// runStatusLabel falls through to the live running/failed/complete derivation.
func dispatchOutcome(status string) string {
	switch status {
	case "stopped", "terminated":
		return status
	default:
		return ""
	}
}

// runRefresh drives the head's meta-refresh hole from run status (DF-F3): 5 while the run
// is running so the log tails, 0 once it concludes so the terminal page settles. The frozen
// head renders the hole as a truthy toggle ({{if .Refresh}}<meta http-equiv="refresh" …>),
// so 5 turns the tail on and 0 turns it off; the literal cadence the meta tag carries is the
// design's, fixed in the frozen shell head, not the handler's to set.
func runRefresh(status string) int {
	if status == "running" {
		return 5
	}
	return 0
}

// applyJobFilter narrows a run view's log to a single queue job and sets the loghead chip,
// server-side (DF-F3b). A blank or non-numeric ?job param is not a filter and leaves the
// view untouched. A numeric id always renders the chip — even an unknown or superseded id:
// the log is filtered to the rows tagged with that job id (empty for an unknown id, the
// honest "No log to show" the tmpl renders), and the chip's kind/vantage come from the
// matching job where the run has one. ClearHref is the bare run route the × navigates to.
func applyJobFilter(v *runView, jobParam, bareHref string, jobs []jobView) {
	if jobParam == "" {
		return
	}
	jobID, err := strconv.ParseInt(jobParam, 10, 64)
	if err != nil {
		return
	}
	jf := &runJobFilter{ID: jobID, ClearHref: bareHref}
	for _, j := range jobs {
		if j.ID == jobID {
			jf.Kind = j.Kind
			jf.Vantage = j.Vantage
			break
		}
	}
	filtered := make([]runLogLine, 0, len(v.Log))
	for _, ln := range v.Log {
		if ln.JobID == jobID {
			filtered = append(filtered, ln)
		}
	}
	v.Log = filtered
	v.JobFilter = jf
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
		out = append(out, runLogLine{JobID: j.ID, Tag: "#" + strconv.FormatInt(j.ID, 10), Level: level, Text: text})
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

// runDegradedFrom folds the jobs into the nullable Outcome callout (#20): the first
// vantage that fell short in this batch (any non-superseded job dead-lettered) with the
// terse "missed N of M checks" reason its own jobs record. A run where every vantage
// finished returns nil, so the frozen tmpl's {{with .Degraded}} renders nothing. It is a
// vantage that looked, never a probe/scanner/agent.
func runDegradedFrom(jobs []jobView) *runDegraded {
	var order []string
	seen := map[string]bool{}
	total := map[string]int{}
	dead := map[string]int{}
	for _, j := range jobs {
		if j.Vantage == "" || j.Superseded {
			continue
		}
		if !seen[j.Vantage] {
			seen[j.Vantage] = true
			order = append(order, j.Vantage)
		}
		total[j.Vantage]++
		if j.State == "dead" {
			dead[j.Vantage]++
		}
	}
	for _, n := range order {
		if dead[n] > 0 {
			return &runDegraded{
				Vantage: n,
				Detail:  fmt.Sprintf("missed %d of %d checks", dead[n], total[n]),
			}
		}
	}
	return nil
}

// dispatchBatchIDs is the set of Batch ids a Dispatch's jobs committed under (queue_job.
// batch_id, carried by ListJobsForDispatch). It is the key the read-side Outcome join
// (#20a) uses to pull THIS run's transitions and new signals out of the estate-wide
// derived stores — the batch is where a run's diff and signals are keyed (ADR-0111,
// ADR-0041), so the run drill-in joins on it.
func dispatchBatchIDs(jobRows []db.ListJobsForDispatchRow) map[int64]bool {
	set := map[int64]bool{}
	for _, j := range jobRows {
		if j.BatchID.Valid {
			set[j.BatchID.Int64] = true
		}
	}
	return set
}

// runOutcome is the Outcome card's batch join result (#20a): the count of transitions
// folded from this run's batch diff and the count of signals first raised in it.
// Concluded is false where the run committed no batch yet — its diff stage has not
// concluded — in which case both figures render "—".
type runOutcome struct {
	Transitions int
	NewSignals  int
	Concluded   bool
}

// joinRunOutcome reads the estate-wide derived stores and joins them to a run's batches
// (#20a, ruled). It reads the drift-event corpus (ListRecentDriftEvents, the same read
// the /drift feed folds) and the signal-instance corpus (ListSignalInstances), then
// hands both to countRunOutcome to key them by batch. This is the READ joining derived
// stores; it never touches the comparison path (ADR-0041 stands for dispatch execution).
// A read failure degrades to a not-concluded outcome ("—") rather than 500ing the page.
func (s *server) joinRunOutcome(ctx context.Context, batchIDs map[int64]bool) runOutcome {
	if len(batchIDs) == 0 {
		return runOutcome{Concluded: false}
	}
	driftRows, err := s.store.ListRecentDriftEvents(ctx, db.ListRecentDriftEventsParams{
		// The zero (all) window reads from the zero instant, so no batch is excluded by
		// age — a run's batch may be older than any fixed period.
		Since: pgtype.Timestamptz{Time: time.Time{}, Valid: true}, MaxEvents: driftFeedLimit,
	})
	if err != nil {
		log.Printf("web: run detail: outcome join: list drift events: %v", err)
		return runOutcome{Concluded: false}
	}
	signals, err := s.store.ListSignalInstances(ctx)
	if err != nil {
		log.Printf("web: run detail: outcome join: list signal instances: %v", err)
		return runOutcome{Concluded: false}
	}
	return countRunOutcome(batchIDs, driftRows, signals, s.now())
}

// countRunOutcome is the pure join over the derived-store reads (unit-tested). Transitions
// are the narratable drift events (the SAME classifyDriftEvent the /drift feed applies)
// whose Batch is one of the run's — transitions folded from this batch's diff. New signals
// are the signal instances first raised inside a run batch's fold window: [batchAt,
// nextBatchAt), where nextBatchAt is the next batch fold instant after it across the
// corpus (a signal's first_seen is minted at fold, so it lands in the fold that raised it).
// A run that committed a batch is concluded even where it moved nothing (0 · 0), distinct
// from a still-running run ("—").
func countRunOutcome(batchIDs map[int64]bool, driftRows []db.ListRecentDriftEventsRow, signals []db.SignalInstance, now time.Time) runOutcome {
	// No committed batch means the run's diff stage has not concluded — render "—".
	if len(batchIDs) == 0 {
		return runOutcome{Concluded: false}
	}
	out := runOutcome{Concluded: true}

	// Batch fold instants across the whole corpus, ascending — the boundaries that bound
	// each batch's signal-raise window.
	instantOf := map[int64]time.Time{}
	for _, row := range driftRows {
		if row.BatchAt.Valid {
			instantOf[row.BatchID] = row.BatchAt.Time.UTC()
		}
	}
	allInstants := make([]time.Time, 0, len(instantOf))
	for _, t := range instantOf {
		allInstants = append(allInstants, t)
	}
	sortTimesAsc(allInstants)

	// Transitions: narratable drift events keyed to one of the run's batches.
	for _, row := range driftRows {
		if !batchIDs[row.BatchID] {
			continue
		}
		if _, ok := classifyDriftEvent(row, now); ok {
			out.Transitions++
		}
	}

	// New signals: signal instances whose first_seen falls in a run batch's fold window.
	for id := range batchIDs {
		start, ok := instantOf[id]
		if !ok {
			continue // this batch raised no transition, so we cannot bound its window
		}
		end := nextInstantAfter(allInstants, start)
		for _, sig := range signals {
			if !sig.FirstSeen.Valid {
				continue
			}
			fs := sig.FirstSeen.Time.UTC()
			if !fs.Before(start) && (end.IsZero() || fs.Before(end)) {
				out.NewSignals++
			}
		}
	}
	return out
}

// nextInstantAfter returns the smallest instant in the ascending slice strictly greater
// than t, or the zero time when t is the latest (an open-ended final window).
func nextInstantAfter(asc []time.Time, t time.Time) time.Time {
	for _, x := range asc {
		if x.After(t) {
			return x
		}
	}
	return time.Time{}
}

// sortTimesAsc sorts instants ascending in place (insertion sort — the batch-instant
// list is short, at most driftFeedLimit distinct batches, usually a handful).
func sortTimesAsc(ts []time.Time) {
	for i := 1; i < len(ts); i++ {
		for j := i; j > 0 && ts[j].Before(ts[j-1]); j-- {
			ts[j], ts[j-1] = ts[j-1], ts[j]
		}
	}
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
		ID: row.DispatchID,
		// Every dispatch in the recent window has a run page (runPage serves /runs/{id}
		// off this same read), so the Running-now kind and each history row link to it
		// (DF-F3). A history row whose dispatch aged past the window would 404, but the
		// same read bounds both, so a listed row always resolves.
		Href:      "/runs/" + strconv.FormatInt(row.DispatchID, 10),
		ScanKind:  row.ScanKind,
		Live:      live,
		Completed: completed,
		InFlight:  inFlight,
		Done:      row.Done,
		Dead:      row.Dead,
		Percent:   percent,
		Active:    inFlight > 0,
		Status:    row.Status,
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
