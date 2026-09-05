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

// The queue corpus is Operational, so no read on this page reaches the comparison path (ADR-0041).

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/rundetail.tmpl"))

const scansHistoryLimit = 50

type dispatchView struct {
	ID           int64
	Href         string
	ScanKind     string
	DispatchedAt string
	Live         int64
	Completed    int64
	InFlight     int64
	Done         int64
	Dead         int64
	Percent      int
	Active       bool
	Status       string
	Rollup       jobRollup
}

type jobRollup struct {
	Ready   int
	Running int
	Done    int
	Dead    int
	Total   int
}

func toJobRollup[T any](jobs []T, state func(T) string) jobRollup {
	// Run detail owns a superseded attempt: no chip, but Total counts it (scans-monitor-bounding §2).
	r := jobRollup{Total: len(jobs)}
	for _, job := range jobs {
		switch state(job) {
		case "ready":
			r.Ready++
		case "running":
			r.Running++
		case "done":
			r.Done++
		case "dead":
			r.Dead++
		}
	}
	return r
}

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

func (s *server) scansPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderSettings(w, r, acct, s.takeSettingsFlash(r, "scans"))
}

func (s *server) fillScansSection(r *http.Request, acct db.Account, f settingsForms, data map[string]any) error {
	ctx := r.Context()
	if seeds, serr := s.store.ListSeeds(ctx); serr == nil {
		if optedIn, oerr := s.store.ListColdScopeSeedIds(ctx); oerr == nil {
			data["ColdScopes"] = toColdScopeViews(toSeedViews(seeds), optedIn)
			data["ColdEnabled"] = len(optedIn) > 0
		}
	}
	data["ColdError"] = f.coldError

	if cfg, cerr := s.store.GetInstanceConfig(ctx); cerr == nil {
		if scans, serr := s.store.ListScans(ctx); serr == nil {
			if accounts, aerr := s.store.ListAccounts(ctx); aerr == nil {
				view := toAddressCapView(cfg, scans, accounts)
				data["CapControl"] = view
				capValue := strconv.FormatInt(view.Cap, 10)
				if f.section == "addresscap" {
					capValue = f.capValue
				}
				data["CapValue"] = capValue
			}
		}
	}
	data["CapError"] = f.capError

	activeRows, err := s.store.ListActiveDispatchProgress(ctx)
	if err != nil {
		return err
	}
	inFlight := activeProgressRows(activeRows)
	active := make([]dispatchView, 0, len(inFlight))
	activeKinds := make(map[string]bool)
	for _, row := range inFlight {
		dv := toDispatchView(row)
		activeKinds[row.ScanKind] = true
		jobs, err := s.store.ListJobsForDispatch(ctx, pgtype.Int8{Int64: row.DispatchID, Valid: true})
		if err != nil {
			return err
		}
		// A per-job row regrows with fan-out, so the card renders none (scans-monitor-bounding §2).
		dv.Rollup = toJobRollup(jobs, func(j db.ListJobsForDispatchRow) string { return j.State })
		active = append(active, dv)
	}
	data["Active"] = active

	// The two reads are exact complements, so a Dispatch is listed once (scans-monitor-bounding §3).
	historyRows, err := s.store.ListConcludedDispatchProgress(ctx, scansHistoryLimit+1)
	if err != nil {
		return err
	}
	truncated := len(historyRows) > scansHistoryLimit
	if truncated {
		historyRows = historyRows[:scansHistoryLimit]
	}
	history := make([]dispatchView, 0, len(historyRows))
	for _, row := range concludedProgressRows(historyRows) {
		history = append(history, toDispatchView(row))
	}
	data["History"] = history
	data["Truncated"] = truncated
	data["HistoryLimit"] = scansHistoryLimit

	if acct.Role == roleAdmin {
		q := r.URL.Query()
		if id, ok := parseDispatchID(q.Get("stop")); ok {
			if row, found := findDispatchRow(inFlight, id); found {
				data["StopTarget"] = map[string]any{
					"ID": row.DispatchID, "ScanKind": row.ScanKind,
					"Pending": row.Ready, "Running": row.Running,
				}
			}
		}
		if id, ok := parseDispatchID(q.Get("terminate")); ok {
			if row, found := findDispatchRow(inFlight, id); found {
				data["TerminateTarget"] = map[string]any{
					"ID": row.DispatchID, "ScanKind": row.ScanKind,
					"Running": row.Running,
				}
			}
		}
	}
	data["Refresh"] = len(active) > 0

	if acct.Role == roleAdmin {
		if trigger, err := s.buildTriggerPanel(r.Context(), activeKinds); err != nil {
			log.Printf("web: scans: build trigger panel: %v", err)
		} else {
			data["Trigger"] = trigger
		}
	}
	return nil
}

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

func activeProgressRows(rows []db.ListActiveDispatchProgressRow) []db.ListDispatchProgressRow {
	// sqlc emits one row type per query, so the three identical reads narrow here to one fold.
	out := make([]db.ListDispatchProgressRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, db.ListDispatchProgressRow{
			DispatchID: r.DispatchID, ScanID: r.ScanID, ScanKind: r.ScanKind,
			CreatedAt: r.CreatedAt, Status: r.Status,
			Total: r.Total, Ready: r.Ready, Running: r.Running,
			Done: r.Done, Dead: r.Dead, Retried: r.Retried,
		})
	}
	return out
}

func concludedProgressRows(rows []db.ListConcludedDispatchProgressRow) []db.ListDispatchProgressRow {
	out := make([]db.ListDispatchProgressRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, db.ListDispatchProgressRow{
			DispatchID: r.DispatchID, ScanID: r.ScanID, ScanKind: r.ScanKind,
			CreatedAt: r.CreatedAt, Status: r.Status,
			Total: r.Total, Ready: r.Ready, Running: r.Running,
			Done: r.Done, Dead: r.Dead, Retried: r.Retried,
		})
	}
	return out
}

func findDispatchRow(rows []db.ListDispatchProgressRow, id int64) (db.ListDispatchProgressRow, bool) {
	for i := range rows {
		if rows[i].DispatchID == id {
			return rows[i], true
		}
	}
	return db.ListDispatchProgressRow{}, false
}

func (s *server) concludedFlash(w http.ResponseWriter, r *http.Request, acct db.Account, detail string) {
	s.toastBackToSection(w, r, acct.ID, "scans", "danger", "Dispatch already concluded", detail)
}

func (s *server) stopScan(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, ok := parseDispatchID(r.FormValue("id"))
	if !ok {
		s.concludedFlash(w, r, acct, "There was nothing in flight to stop.")
		return
	}
	rows, err := s.store.ListActiveDispatchProgress(r.Context())
	if err != nil {
		s.serverError(w, "stop scan: list dispatches", err)
		return
	}
	row, found := findDispatchRow(activeProgressRows(rows), id)
	if !found {
		s.concludedFlash(w, r, acct, "It has already finished or been ended — nothing was stopped.")
		return
	}
	pid := pgtype.Int8{Int64: id, Valid: true}
	// ClaimJob selects state='ready' alone, so a cancelled job leaves the claimable set at once (ADR-0164 §2).
	n, err := s.store.CancelReadyJobsForDispatch(r.Context(), pid)
	if err != nil {
		s.serverError(w, "stop scan: cancel pending jobs", err)
		return
	}
	if err := s.store.SetDispatchStatus(r.Context(), db.SetDispatchStatusParams{ID: id, Status: "stopped"}); err != nil {
		s.serverError(w, "stop scan: record status", err)
		return
	}
	desc := fmt.Sprintf("%d pending %s cancelled · %d running finishing",
		n, plural(int(n), "job", "jobs"), row.Running)
	s.toastBackToSection(w, r, acct.ID, "scans", "neutral", "Dispatch stopped", desc)
}

func (s *server) terminateScan(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, ok := parseDispatchID(r.FormValue("id"))
	if !ok {
		s.concludedFlash(w, r, acct, "There was nothing in flight to terminate.")
		return
	}
	rows, err := s.store.ListActiveDispatchProgress(r.Context())
	if err != nil {
		s.serverError(w, "terminate scan: list dispatches", err)
		return
	}
	if _, found := findDispatchRow(activeProgressRows(rows), id); !found {
		s.concludedFlash(w, r, acct, "It has already finished or been ended — nothing was terminated.")
		return
	}
	pid := pgtype.Int8{Int64: id, Valid: true}
	// Nothing deletes a committed observation, so a terminate discards only staged work (ADR-0164 §3).
	n, err := s.store.CancelActiveJobsForDispatch(r.Context(), pid)
	if err != nil {
		s.serverError(w, "terminate scan: cancel jobs", err)
		return
	}
	// SetDispatchStatus guards on 'fanned-out', so a stop already recorded stands and this write no-ops (ADR-0164 §4, #1421).
	if err := s.store.SetDispatchStatus(r.Context(), db.SetDispatchStatusParams{ID: id, Status: "terminated"}); err != nil {
		s.serverError(w, "terminate scan: record status", err)
		return
	}
	desc := fmt.Sprintf("%d %s stopped", n, plural(int(n), "job", "jobs"))
	s.toastBackToSection(w, r, acct.ID, "scans", "neutral", "Scan terminated", desc)
}

type runStage struct {
	Num     int
	Title   string
	Detail  string
	Done    bool
	Current bool
	Last    bool
}

type runLogLine struct {
	JobID int64
	Tag   string
	Level string
	Text  string
	Href  string
}

type runJobFilter struct {
	ID        int64
	Kind      string
	Vantage   string
	ClearHref string
	RawHref   string
}

type runKV struct {
	K, V string
}

type runVantage struct {
	Name    string
	Latency string
	Status  string
}

type runDegraded struct {
	Vantage string
	Detail  string // rundetail.tmpl wraps this in its own sentence, so it is the middle clause only
}

type runView struct {
	ID          int64
	Title       string
	Status      string
	Scope       string
	Meta        string
	Transitions string
	NewSignals  string
	Active      bool
	Stages      []runStage
	Log         []runLogLine
	Params      []runKV
	Vantages    []runVantage
	Degraded    *runDegraded
	JobFilter   *runJobFilter
	StreamHref  string
}

func (s *server) runPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	raw := r.PathValue("id")

	if s.devMode {
		if raw == devRunDetailID {
			s.render(w, r, "run", s.runDetailFixtureData(acct))
			return
		}
		if raw == devRunningRunID {
			s.render(w, r, "run", s.runningRunFixtureData(acct, r.URL.Query().Get("job"), r.URL.Path))
			return
		}
		s.renderMissingRun(w, r, acct, raw)
		return
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		s.renderMissingRun(w, r, acct, raw)
		return
	}

	// Run detail resolves off the monitor's own two reads, so every listed row has a run page (#962).
	activeRows, err := s.store.ListActiveDispatchProgress(r.Context())
	if err != nil {
		s.serverError(w, "run detail: list dispatches", err)
		return
	}
	found, ok := findDispatchRow(activeProgressRows(activeRows), id)
	if !ok {
		historyRows, herr := s.store.ListConcludedDispatchProgress(r.Context(), scansHistoryLimit)
		if herr != nil {
			s.serverError(w, "run detail: list dispatches", herr)
			return
		}
		found, ok = findDispatchRow(concludedProgressRows(historyRows), id)
	}
	if !ok {
		s.renderMissingRun(w, r, acct, raw)
		return
	}

	dv := toDispatchView(found)
	jobRows, err := s.store.ListJobsForDispatch(r.Context(), pgtype.Int8{Int64: id, Valid: true})
	if err != nil {
		s.serverError(w, "run detail: list jobs", err)
		return
	}

	view := s.buildRunView(r, dv, jobRows)
	s.render(w, r, "run", map[string]any{
		"Title": "batch " + view.Title, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "drift",
		"Refresh":   runRefresh(view.Status),
		// rundetail.tmpl also reads StreamHref at root scope, so the attribute and script emit together.
		"StreamHref": view.StreamHref,
		"Run":        view,
	})
}

func (s *server) buildRunView(r *http.Request, dv dispatchView, jobRows []db.ListJobsForDispatchRow) runView {
	v := runView{
		ID:     dv.ID,
		Title:  dv.DispatchedAt,
		Active: dv.Active,
		Scope:  "all scopes",
	}
	v.Status = runStatusLabel(dv.Active, dv.Dead, dispatchOutcome(dv.Status))

	jobs := make([]jobView, 0, len(jobRows))
	for _, j := range jobRows {
		jobs = append(jobs, toJobView(j))
	}

	v.Vantages = runVantages(jobs)
	v.Stages = runStages(jobs)
	v.Log = runLog(jobs)

	batchIDs := dispatchBatchIDs(jobRows)
	out := s.joinRunOutcome(r.Context(), batchIDs)
	if out.Concluded {
		v.Transitions = strconv.Itoa(out.Transitions)
		v.NewSignals = strconv.Itoa(out.NewSignals)
	} else {
		v.Transitions = "—"
		v.NewSignals = "—"
	}

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

	applyJobFilter(&v, r.URL.Query().Get("job"), r.URL.Path, jobs)
	linkRunLog(&v, r.URL.Path)

	v.StreamHref = runStreamHref(r.URL.Path, v.JobFilter, dv.Active, jobs)
	return v
}

const (
	runStreamHold = 25 * time.Second
	runStreamPoll = time.Second
)

func jobActive(state string) bool {
	return state == "ready" || state == "running"
}

func runStreamHref(runPath string, filter *runJobFilter, runActive bool, jobs []jobView) string {
	// The base is the request's own path, so it rides whichever of /run and /runs the viewer is on.
	base := runPath + "/stream"
	if filter != nil {
		for _, j := range jobs {
			if j.ID == filter.ID {
				if jobActive(j.State) {
					return base + "?job=" + strconv.FormatInt(filter.ID, 10)
				}
				return ""
			}
		}
		return ""
	}
	if runActive {
		return base
	}
	return ""
}

// The wire shape is .Log's own, so the stream redacts exactly as the page does and adds no format.

type runStreamLine struct {
	Tag   string `json:"tag"`
	Level string `json:"level"`
	Text  string `json:"text"`
}

type runStreamResp struct {
	Lines []runStreamLine `json:"lines"`
	Next  int             `json:"next"`
	Done  bool            `json:"done"`
}

func (s *server) deriveRunStream(ctx context.Context, dispatchID int64, jobParam string) (state, events []runStreamLine, done bool, err error) {
	jobRows, err := s.store.ListJobsForDispatch(ctx, pgtype.Int8{Int64: dispatchID, Valid: true})
	if err != nil {
		return nil, nil, false, err
	}
	jobs := make([]jobView, 0, len(jobRows))
	for _, j := range jobRows {
		jobs = append(jobs, toJobView(j))
	}

	var jobFilter int64
	filtered := false
	if jobParam != "" {
		if jobID, perr := strconv.ParseInt(jobParam, 10, 64); perr == nil {
			jobFilter, filtered = jobID, true
		}
	}

	stateLog := runLog(jobs)
	state = make([]runStreamLine, 0, len(stateLog))
	for _, ln := range stateLog {
		if filtered && ln.JobID != jobFilter {
			continue
		}
		state = append(state, runStreamLine{Tag: ln.Tag, Level: ln.Level, Text: ln.Text})
	}

	if s.progress != nil {
		events = eventStreamLines(s.progress.ForDispatch(dispatchID), jobFilter, filtered)
	}

	if filtered {
		done = true
		for _, j := range jobs {
			if j.ID == jobFilter {
				done = !jobActive(j.State)
				break
			}
		}
		return state, events, done, nil
	}
	done = true
	for _, j := range jobs {
		if jobActive(j.State) {
			done = false
			break
		}
	}
	return state, events, done, nil
}

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
	// The client's initial cursor is its rendered state-line count, so state must stay the low part.
	eventCur, stateCur := decodeStreamCursor(after)
	jobParam := r.URL.Query().Get("job")
	ctx := r.Context()

	timeout := time.NewTimer(runStreamHold)
	defer timeout.Stop()
	ticker := time.NewTicker(runStreamPoll)
	defer ticker.Stop()

	for {
		state, events, done, derr := s.deriveRunStream(ctx, id, jobParam)
		if derr != nil {
			apiReadError(w, "run detail: stream", derr)
			return
		}
		next := encodeStreamCursor(len(events), len(state))
		if len(state) > stateCur || len(events) > eventCur || done {
			out := make([]runStreamLine, 0, len(state)+len(events))
			if from := min(stateCur, len(state)); from < len(state) {
				out = append(out, state[from:]...)
			}
			if from := min(eventCur, len(events)); from < len(events) {
				out = append(out, events[from:]...)
			}
			writeAPIJSON(w, runStreamResp{Lines: out, Next: next, Done: done})
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-timeout.C:
			writeAPIJSON(w, runStreamResp{Lines: []runStreamLine{}, Next: next, Done: false})
			return
		case <-ticker.C:
		}
	}
}

func runStatusLabel(active bool, dead int64, outcome string) string {
	// This word is rundetail.tmpl's rd-batch CSS class as well as the visible label (ADR-0165 §1).
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

func dispatchOutcome(status string) string {
	switch status {
	case "stopped", "terminated":
		return status
	default:
		return ""
	}
}

func runRefresh(status string) int {
	// The shell head fixes the cadence and reads this as a toggle, so 5 only means on (ADR-0165 §4).
	if status == "running" {
		return 5
	}
	return 0
}

func applyJobFilter(v *runView, jobParam, bareHref string, jobs []jobView) {
	if jobParam == "" {
		return
	}
	jobID, err := strconv.ParseInt(jobParam, 10, 64)
	if err != nil {
		return
	}
	jf := &runJobFilter{ID: jobID, ClearHref: bareHref, RawHref: bareHref + "/raw?job=" + jobParam}
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

func linkRunLog(v *runView, bareHref string) {
	if v.JobFilter != nil {
		return
	}
	// The raw-output link sits inside the job-filter chip, so this tag is its only way in (#1083).
	for i, ln := range v.Log {
		if ln.JobID == 0 {
			continue
		}
		v.Log[i].Href = bareHref + "?job=" + strconv.FormatInt(ln.JobID, 10)
	}
}

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

func dispatchBatchIDs(jobRows []db.ListJobsForDispatchRow) map[int64]bool {
	set := map[int64]bool{}
	for _, j := range jobRows {
		if j.BatchID.Valid {
			set[j.BatchID.Int64] = true
		}
	}
	return set
}

type runOutcome struct {
	Transitions int
	NewSignals  int
	Concluded   bool
}

func (s *server) joinRunOutcome(ctx context.Context, batchIDs map[int64]bool) runOutcome {
	if len(batchIDs) == 0 {
		return runOutcome{Concluded: false}
	}
	driftRows, err := s.store.ListRecentDriftEvents(ctx, db.ListRecentDriftEventsParams{
		// A run's batch can be older than any fixed period, so the zero instant excludes none by age.
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

func countRunOutcome(batchIDs map[int64]bool, driftRows []db.ListRecentDriftEventsRow, signals []db.SignalInstance, now time.Time) runOutcome {
	if len(batchIDs) == 0 {
		return runOutcome{Concluded: false}
	}
	out := runOutcome{Concluded: true}

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

	for _, row := range driftRows {
		if !batchIDs[row.BatchID] {
			continue
		}
		if _, ok := classifyDriftEvent(row, now); ok {
			out.Transitions++
		}
	}

	for id := range batchIDs {
		start, ok := instantOf[id]
		if !ok {
			continue // this batch raised no transition, so we cannot bound its window
		}
		// A signal's first_seen is minted at fold, so it lands in the window of the fold that raised it.
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

func nextInstantAfter(asc []time.Time, t time.Time) time.Time {
	for _, x := range asc {
		if x.After(t) {
			return x
		}
	}
	return time.Time{}
}

func sortTimesAsc(ts []time.Time) {
	for i := 1; i < len(ts); i++ {
		for j := i; j > 0 && ts[j].Before(ts[j-1]); j-- {
			ts[j], ts[j-1] = ts[j-1], ts[j]
		}
	}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func toDispatchView(row db.ListDispatchProgressRow) dispatchView {
	// A retry enqueues a fresh job, so counting the retired row too would double the denominator.
	live := row.Total - row.Retried
	completed := row.Done + row.Dead
	inFlight := row.Ready + row.Running

	percent := 0
	if live > 0 {
		percent = int(completed * 100 / live)
	}

	dv := dispatchView{
		ID:        row.DispatchID,
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

type scanScheduleView struct {
	HasLast    bool
	LastScanAt time.Time
	SinceLast  time.Duration
	LastAgo    string

	HasNext    bool
	NextScanAt time.Time
	UntilNext  time.Duration
	NextIn     string
}

func (s *server) scanSchedule(ctx context.Context) scanScheduleView {
	now := s.now().UTC()
	var v scanScheduleView

	// The read is newest-first, so the first row with a real created_at is the latest fan-out.
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

func nextCadenceBoundary(now time.Time, cadenceSeconds int64) time.Time {
	// A missed tick is never caught up, so this mirrors the dispatcher's flooring (v1-spec §4.1).
	secs := cadenceSeconds
	if secs <= 0 {
		secs = 1
	}
	floored := (now.UTC().Unix() / secs) * secs
	return time.Unix(floored+secs, 0).UTC()
}

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
