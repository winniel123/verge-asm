package main

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
)

// The on-demand scan trigger (#252, ask 3 of #245). It reverses the deliberate
// cadence-only stance — "there is no button to press" (using.md) — for one seat:
// an admin may dispatch an enabled Scan now, without waiting for its cadence
// window. It is paired with the #245 running indicator on the same page, so the
// operator sees the result of pressing it.
//
// Four decided guardrails carry the reversal, since this exposes an *active*
// scan (hot runs an outbound port scan for minutes) to the web:
//
//  1. Admin-only. The POST is gated behind requireAdmin, the same gate /sources
//     toggling uses; a viewer never sees the control and cannot reach the handler.
//  2. Disabled scans are refused. The shipped-off cold tier is a full-range sweep
//     with no cadence and so no currency bound — the ad-hoc one-off ADR-0044
//     forbids. The Dispatcher is the authority on this refusal; the handler reads
//     the live enabled flag only to name the reason to the operator.
//  3. Overlap protection. A kind already in flight is not dispatched again — the
//     guard against an accidental double-click piling up redundant fan-outs. It
//     rests on the same Dispatch corpus the monitor reads.
//  4. It changes nothing about *what* runs. The trigger enqueues the identical
//     fan-out the CLI `-trigger` path uses (queue.Dispatcher.Trigger), honouring
//     the same disabled-scan refusal and the same (scan, tick) idempotency key.

// scanTrigger is the seam the on-demand trigger sits behind. Production wires the
// queue Dispatcher over the pool (main.go); a fake is injected under test so a
// trigger asserts the enqueue without a live Postgres or a running worker. Its
// one method is queue.Dispatcher.Trigger: it fans a Scan out at the current
// instant and returns the job count, or the disabled-scan refusal.
type scanTrigger interface {
	Trigger(ctx context.Context, kind string) (int, error)
}

// Trigger notices — the operator-facing outcome of pressing the button, carried
// across the post-redirect-get so a browser refresh does not re-file the trigger.
// The redirect names the outcome and the kind; the panel renders one sentence.
const (
	noticeTriggered = "triggered" // dispatched, N jobs enqueued
	noticeRunning   = "running"   // a scan of this kind is already in flight
	noticeDisabled  = "disabled"  // the scan is disabled — refused (cold, ADR-0044)
	noticeNoJobs    = "nojobs"    // dispatched but nothing enqueued — empty scope, or the tick was already owned
	noticeUnknown   = "unknown"   // no such scan kind — a hand-crafted POST
)

// triggerScanView is one scan shaped for the trigger panel: its kind, whether it
// is enabled (only an enabled scan carries a submit control), whether it is in
// flight right now (an active kind is not re-triggerable), and its cadence for
// context. A disabled scan is shown as not-triggerable rather than hidden, so the
// panel states honestly why the cold tier has no button.
type triggerScanView struct {
	Kind    string
	Enabled bool
	Active  bool
	Cadence string
	IsCold  bool // the cold tier carries the "opt a scope in to enable" pointer
}

// triggerPanel is the admin-only control block on the Scans page: the per-scan
// trigger rows and the receipt from the last press. Notice is the rendered
// sentence ("" when there is nothing to report); NoticeOK styles a success.
type triggerPanel struct {
	Scans    []triggerScanView
	Notice   string
	NoticeOK bool
}

// triggerScan dispatches one Scan on demand. It is reached only through
// requireAdmin, so a viewer never triggers a scan. The guardrails run in order:
// an unknown kind is refused at the door, a disabled scan is refused before the
// Dispatcher (the reason named to the operator), an in-flight kind is not
// dispatched again, and only then does the fan-out run.
func (s *server) triggerScan(w http.ResponseWriter, r *http.Request, _ db.Account) {
	ctx := r.Context()
	kind := r.FormValue("kind")

	sc, err := s.store.GetScanByKind(ctx, kind)
	if errors.Is(err, pgx.ErrNoRows) {
		s.redirectTrigger(w, r, noticeUnknown, kind, 0)
		return
	}
	if err != nil {
		s.serverError(w, "trigger scan: get scan", err)
		return
	}

	// A disabled scan is the one-off ADR-0044 forbids. The Dispatcher would refuse
	// it too; checking the live flag here names the reason rather than surfacing a
	// bare 500, and keeps a disabled scan from ever reaching the fan-out.
	if !sc.Enabled {
		s.redirectTrigger(w, r, noticeDisabled, kind, 0)
		return
	}

	// Overlap guard: a kind already in flight is not dispatched again. The unique
	// (scan, tick) key already collapses two presses within the same second, but a
	// scan that runs for minutes could otherwise be re-triggered a second later;
	// this refuses the redundant fan-out against the live Dispatch corpus.
	active, err := s.activeDispatchKinds(ctx)
	if err != nil {
		s.serverError(w, "trigger scan: active dispatches", err)
		return
	}
	if active[kind] {
		s.redirectTrigger(w, r, noticeRunning, kind, 0)
		return
	}

	n, err := s.dispatcher.Trigger(ctx, kind)
	if err != nil {
		// The expected refusals — unknown kind, disabled scan, in-flight overlap —
		// were all decided above, so a residual error here is a genuine fan-out
		// failure (a transaction, advisory-lock or enqueue error), not a normal
		// outcome. Surface it as a 500 rather than mislabel it a benign refusal.
		s.serverError(w, "trigger scan: dispatch", err)
		return
	}
	if n == 0 {
		// A zero-job dispatch is ambiguous from here: either the fan-out found
		// nothing to enqueue (no scope or vantage covers this scan yet) or the tick
		// was already owned by an overlapping dispatch. Trigger returns (0, nil) for
		// both, so the notice names both rather than guess — never a false "already
		// dispatched" over a scan that just found nothing to look at.
		s.redirectTrigger(w, r, noticeNoJobs, kind, 0)
		return
	}
	// The act fires a toast across the post-redirect-get (PARITY-CHART P1.7, the
	// "Scan started" toast ConsoleApp fires on runScan); the /scans monitor also
	// renders the fuller receipt sentence the notice query carries.
	jobs := "1 job"
	if n != 1 {
		jobs = strconv.Itoa(n) + " jobs"
	}
	s.flashToast(w, "neutral", "Scan started", "A "+kind+" scan dispatched — "+jobs+" enqueued.")
	s.redirectTrigger(w, r, noticeTriggered, kind, n)
}

// redirectTrigger sends the post-redirect-get back to the monitor, naming the
// outcome so a browser refresh re-reads the receipt instead of re-firing the
// trigger (the same pattern /seeds uses for its partial-proposal notice, #251).
func (s *server) redirectTrigger(w http.ResponseWriter, r *http.Request, notice, kind string, jobs int) {
	q := url.Values{"notice": {notice}, "kind": {kind}}
	if jobs > 0 {
		q.Set("jobs", strconv.Itoa(jobs))
	}
	http.Redirect(w, r, "/scans?"+q.Encode(), http.StatusSeeOther)
}

// activeDispatchKinds reads which scan kinds have a Dispatch in flight right now,
// off the same Operational corpus the monitor reads (#245). It is the seam the
// overlap guard and the panel's "running" marker both rest on: a kind is active
// when any of its recent Dispatches still has ready-or-running jobs.
func (s *server) activeDispatchKinds(ctx context.Context) (map[string]bool, error) {
	rows, err := s.store.ListDispatchProgress(ctx, scansHistoryLimit)
	if err != nil {
		return nil, err
	}
	active := make(map[string]bool)
	for _, row := range rows {
		if toDispatchView(row).Active {
			active[row.ScanKind] = true
		}
	}
	return active, nil
}

// buildTriggerPanel assembles the admin control block: one row per scan with its
// enabled and in-flight state, and the receipt from the last trigger read off the
// request's notice query. Every scan is listed — the disabled cold tier included —
// so the panel states honestly which scans can be triggered and which cannot.
func (s *server) buildTriggerPanel(r *http.Request, active map[string]bool) (triggerPanel, error) {
	scans, err := s.store.ListScans(r.Context())
	if err != nil {
		return triggerPanel{}, err
	}
	views := make([]triggerScanView, 0, len(scans))
	for _, sc := range scans {
		views = append(views, triggerScanView{
			Kind:    sc.Kind,
			Enabled: sc.Enabled,
			Active:  active[sc.Kind],
			Cadence: cadenceLabel(sc.CadenceSeconds),
			IsCold:  sc.Kind == scan.ColdKind,
		})
	}
	notice, ok := triggerNotice(r.URL.Query())
	return triggerPanel{Scans: views, Notice: notice, NoticeOK: ok}, nil
}

// triggerNotice renders the receipt sentence from the redirect's query. It
// returns "" when there is nothing to report, and a bool marking a success so the
// panel can style it apart from a refusal.
func triggerNotice(q url.Values) (string, bool) {
	kind := q.Get("kind")
	switch q.Get("notice") {
	case noticeTriggered:
		// The success path always carries a positive job count. If the count is
		// missing or unparseable (only reachable by a hand-crafted URL), fall back
		// to the countless sentence rather than render "  job s enqueued".
		if jobs, err := strconv.Atoi(q.Get("jobs")); err == nil && jobs > 0 {
			plural := "s"
			if jobs == 1 {
				plural = ""
			}
			return "Dispatched a " + kind + " scan — " + strconv.Itoa(jobs) + " job" + plural +
				" enqueued. It appears in flight below as the worker runs it.", true
		}
		return "Dispatched a " + kind + " scan. It appears in flight below as the worker runs it.", true
	case noticeRunning:
		return "A " + kind + " scan is already in flight — it was not dispatched again. Watch its progress below.", false
	case noticeDisabled:
		if kind == scan.ColdKind {
			return "The cold scan is disabled, so it cannot be triggered. It runs the full-range " +
				"sweep; enable it by opting an address scope into the full-range tier on Seeds.", false
		}
		return "The " + kind + " scan is disabled, so it cannot be triggered.", false
	case noticeNoJobs:
		return "The " + kind + " scan was dispatched but enqueued no jobs — it has nothing to look at yet " +
			"(no scope or vantage covers it), or its current tick was already dispatched.", false
	case noticeUnknown:
		return "That scan is not one this deployment runs — nothing was dispatched.", false
	default:
		return "", false
	}
}

// triggerTemplates adds the admin trigger panel to the shared template set. It is
// parsed here — beside #252's handler — rather than inlined in templates.go, so
// this ticket's markup lives in this ticket's file (the same split sources.go
// uses). The scans page invokes {{template "scantrigger" .}} where an admin sees
// it. The reference to tmpl orders its initialisation before this one.
var _ = template.Must(tmpl.Parse(triggerTemplates))

const triggerTemplates = `
{{define "scantrigger"}}
{{if .IsAdmin}}
{{with .Trigger}}
<div class="section">
<div class="microlabel">Admin · on-demand</div>
<h2>Trigger a scan</h2>
<p>Dispatch an enabled scan now, without waiting for its cadence. It enqueues the same
fan-out the worker runs on cadence — a scan already in flight is not dispatched again, and
the disabled cold tier cannot be triggered at all. Pressing this runs an active measurement;
the result appears in flight above.</p>
{{if .Notice}}<div class="{{if .NoticeOK}}notice{{else}}error{{end}}">{{.Notice}}</div>{{end}}
<table>
<thead><tr><th>Scan</th><th>Cadence</th><th>State</th><th></th></tr></thead>
<tbody>
{{range .Scans}}<tr>
<td class="mono">{{.Kind}}</td>
<td class="mono">{{.Cadence}}</td>
<td>{{if .Active}}<span class="dot live"></span><span class="badge">in flight</span>{{else if .Enabled}}<span class="badge">enabled</span>{{else}}<span class="badge off">disabled</span>{{end}}</td>
<td>
{{if .Active}}<span class="muted">running</span>
{{else if .Enabled}}<form method="post" action="/scans/trigger" style="display:inline"><input type="hidden" name="kind" value="{{.Kind}}"><button type="submit">Run now</button></form>
{{else if .IsCold}}<span class="muted">disabled — opt a scope in on <a href="/scope">Scope</a></span>
{{else}}<span class="muted">disabled</span>{{end}}
</td>
</tr>{{end}}
</tbody>
</table>
</div>
{{end}}
{{end}}
{{end}}
`
