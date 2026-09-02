package main

import (
	"context"
	"errors"
	"html/template"
	"log"
	"net/http"
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
// trigger rows.
//
// It carries no receipt of its own. Ticket #1087 moved every trigger outcome onto
// the single-consume toast flash, because the `notice` / `kind` / `jobs` query the
// panel used to read made the landing URL differ from the submitting one, so the
// scroll key ticket #970 set missed by construction (ADR-0130 §2, failure class E).
type triggerPanel struct {
	Scans []triggerScanView
}

// triggerOutcome is the receipt one guarded dispatch produced, shaped for the toast
// carrier: the tone the console styles it with, the headline, and the sentence under
// it. Every outcome — the dispatch and all four refusals — is one of these, so the
// two routes that share the dispatch differ only in where they land it.
type triggerOutcome struct {
	Tone        string
	Title       string
	Description string
}

// runTrigger dispatches one Scan on demand and returns the receipt to carry. It is
// reached only through requireAdmin, so a viewer never triggers a scan. The
// guardrails run in order: an unknown kind is refused at the door, a disabled scan
// is refused before the Dispatcher (the reason named to the operator), an in-flight
// kind is not dispatched again, and only then does the fan-out run.
//
// It reports false when it has already answered with a 500, so a caller returns
// without writing a second answer.
func (s *server) runTrigger(w http.ResponseWriter, r *http.Request) (triggerOutcome, bool) {
	ctx := r.Context()
	kind := r.FormValue("kind")

	sc, err := s.store.GetScanByKind(ctx, kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return triggerOutcome{"danger", "That scan is not one this deployment runs",
			"Nothing was dispatched."}, true
	}
	if err != nil {
		s.serverError(w, "trigger scan: get scan", err)
		return triggerOutcome{}, false
	}

	// A disabled scan is the one-off ADR-0044 forbids. The Dispatcher would refuse
	// it too; checking the live flag here names the reason rather than surfacing a
	// bare 500, and keeps a disabled scan from ever reaching the fan-out.
	if !sc.Enabled {
		detail := "A disabled scan cannot be triggered."
		if kind == scan.ColdKind {
			detail = "It runs the full-range sweep. Enable it by opting an address scope " +
				"into the full-range tier on Scope."
		}
		return triggerOutcome{"danger", "The " + kind + " scan is disabled", detail}, true
	}

	// Overlap guard: a kind already in flight is not dispatched again. The unique
	// (scan, tick) key already collapses two presses within the same second, but a
	// scan that runs for minutes could otherwise be re-triggered a second later;
	// this refuses the redundant fan-out against the live Dispatch corpus.
	active, err := s.activeDispatchKinds(ctx)
	if err != nil {
		s.serverError(w, "trigger scan: active dispatches", err)
		return triggerOutcome{}, false
	}
	if active[kind] {
		return triggerOutcome{"warn", "A " + kind + " scan is already in flight",
			"It was not dispatched again. Watch its progress in Running now."}, true
	}

	n, err := s.dispatcher.Trigger(ctx, kind)
	if err != nil {
		// The expected refusals — unknown kind, disabled scan, in-flight overlap —
		// were all decided above, so a residual error here is a genuine fan-out
		// failure (a transaction, advisory-lock or enqueue error), not a normal
		// outcome. Surface it as a 500 rather than mislabel it a benign refusal.
		s.serverError(w, "trigger scan: dispatch", err)
		return triggerOutcome{}, false
	}
	if n == 0 {
		// A zero-job dispatch is ambiguous from here: either the fan-out found
		// nothing to enqueue (no scope or vantage covers this scan yet) or the tick
		// was already owned by an overlapping dispatch. Trigger returns (0, nil) for
		// both, so the receipt names both rather than guess — never a false "already
		// dispatched" over a scan that just found nothing to look at.
		return triggerOutcome{"warn", "The " + kind + " scan enqueued no jobs",
			"Nothing covers it yet — no scope or vantage — or its current tick was already dispatched."}, true
	}
	// Copy per the dogfood note: "<kind> scan dispatched" / "N jobs fanned out".
	return triggerOutcome{"neutral", kind + " scan dispatched",
		strconv.Itoa(n) + " " + plural(n, "job", "jobs") + " fanned out"}, true
}

// triggerScan is POST /scans/trigger: the Run now button beside a scan kind. It
// lands the operator back on the URL they pressed the button from — /scans or
// /settings?tab=scans, query and all — at the offset they were at (ADR-0130 §3,
// ticket #1087).
//
// It used to spell its receipt on the destination instead (`/scans?notice=…&kind=…`),
// which made the landing URL differ from the submitting one by construction, so the
// scroll key ticket #970 set could never hit. The receipt is a single-consume toast
// now: `toast` is the one parameter both stripToastParam and the shell's keyFor drop,
// so it cannot move the key. It rides the per-account flash store rather than the URL
// because the Scans surface meta-refreshes itself while a dispatch is in flight, and a
// toast spelled on the URL fires again on every one of those reloads.
//
// This is redirectBack and NOT toastBackToSection, which is the one place the trigger
// parts from its stop and terminate siblings. toastBackToSection strips the ?stop= and
// ?terminate= confirm openers off the destination (settings.go dialogParams). That is
// right for those two acts, because each ANSWERS the confirm it strips. A trigger ends
// no dispatch, so an unanswered confirm is part of the place the operator is returning
// to, and dropping it would move the scroll key for nothing.
func (s *server) triggerScan(w http.ResponseWriter, r *http.Request, acct db.Account) {
	out, ok := s.runTrigger(w, r)
	if !ok {
		return
	}
	s.flash.set(acct.ID, toastVM{Tone: out.Tone, Title: out.Title, Description: out.Description})
	s.redirectBack(w, r, "/settings?tab=scans")
}

// finishOnboarding is POST /onboarding/finish: the wizard's "Start first scan". It
// runs the identical guarded dispatch and then LEAVES the wizard for the monitor,
// which is a deliberate page move rather than a return to the submitting URL (the
// class-E exemption names it). The receipt rides the same single-consume flash.
func (s *server) finishOnboarding(w http.ResponseWriter, r *http.Request, acct db.Account) {
	out, ok := s.runTrigger(w, r)
	if !ok {
		return
	}
	s.flashRedirect(w, r, acct.ID, "/scans", out.Tone, out.Title, out.Description)
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

// chromeScanRunning reports whether ANY scan kind has a Dispatch in flight right now —
// the single in-flight flag the design-owned TopNav "Scan running" pill reads on
// every view (R4-D3, #758). It rests on the same activeDispatchKinds seam the
// dashboard's own scanning indicator uses (a scan is in flight when any kind is
// active), so the shell pill and the dashboard's Run/Running control never disagree.
// Best-effort: a failed read reports not-running rather than fabricating a pulse or
// failing the page — the pill simply stays dark until the next render. (Distinct from
// restore.go's scanInFlight, which fails toward in-flight to guard a destructive act.)
func (s *server) chromeScanRunning(ctx context.Context) bool {
	active, err := s.activeDispatchKinds(ctx)
	if err != nil {
		log.Printf("web: chrome: scan in-flight read: %v", err)
		return false
	}
	return len(active) > 0
}

// buildTriggerPanel assembles the admin control block: one row per scan with its
// enabled and in-flight state. Every scan is listed — the disabled cold tier
// included — so the panel states honestly which scans can be triggered and which
// cannot. The receipt from the last press is a toast, not a panel row, so nothing
// here reads the request's query.
func (s *server) buildTriggerPanel(ctx context.Context, active map[string]bool) (triggerPanel, error) {
	scans, err := s.store.ListScans(ctx)
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
	return triggerPanel{Scans: views}, nil
}

// triggerTemplates adds the admin trigger panel to the shared template set. It is
// parsed here — beside #252's handler — rather than inlined in templates.go, so
// this ticket's markup lives in this ticket's file (the same split sources.go
// uses). The scans page invokes {{template "scantrigger" .}} where an admin sees
// it. The reference to tmpl orders its initialisation before this one.
var _ = template.Must(tmpl.Parse(triggerTemplates))

// The scantrigger markup is repo-owned glue (ADR-0109: no design component is
// authored). Under P4.4 pageCSS is deleted, so its few controls are restyled INLINE
// within the design token vocabulary (the same token colors + literal metrics the
// frozen shell.tmpl uses), rather than leaning on the retired legacy classes. It
// renders inside the design-owned settings.tmpl "scans" sub-tab, styled to sit beside
// its .st-card blocks.
const triggerTemplates = `
{{define "scantrigger"}}
{{if .IsAdmin}}
{{with .Trigger}}
<section style="background:var(--surface-raised);border:1px solid var(--border-default);border-radius:16px;box-shadow:var(--shadow-sm);padding:20px;margin-bottom:20px;font-family:var(--font-ui)">
<span style="display:block;font:500 10.5px var(--font-mono);letter-spacing:0.07em;text-transform:uppercase;color:var(--text-muted)">Admin · on-demand</span>
<h3 style="margin:6px 0 8px;font:600 15px var(--font-ui);letter-spacing:var(--heading-tracking,-0.01em);color:var(--text-ink)">Trigger a scan</h3>
<p style="margin:0 0 14px;font:400 12.5px/1.5 var(--font-ui);color:var(--text-secondary);max-width:78ch">Dispatch an enabled scan now, without waiting for its cadence. It enqueues the same fan-out the worker runs on cadence — a scan already in flight is not dispatched again, and the disabled cold tier cannot be triggered at all. Pressing this runs an active measurement; the result appears in flight above.</p>
<table style="width:100%;border-collapse:collapse">
<thead><tr>
<th style="text-align:left;font:600 10px var(--font-mono);text-transform:uppercase;letter-spacing:0.06em;color:var(--text-muted);padding:0 16px 8px 0;border-bottom:1px solid var(--border-strong)">Scan</th>
<th style="text-align:left;font:600 10px var(--font-mono);text-transform:uppercase;letter-spacing:0.06em;color:var(--text-muted);padding:0 16px 8px 0;border-bottom:1px solid var(--border-strong)">Cadence</th>
<th style="text-align:left;font:600 10px var(--font-mono);text-transform:uppercase;letter-spacing:0.06em;color:var(--text-muted);padding:0 16px 8px 0;border-bottom:1px solid var(--border-strong)">State</th>
<th style="border-bottom:1px solid var(--border-strong)"></th>
</tr></thead>
<tbody>
{{range .Scans}}<tr>
<td style="font:400 12px var(--font-mono);color:var(--text-body);padding:10px 16px 10px 0;border-bottom:1px solid var(--row-sep)">{{.Kind}}</td>
<td style="font:400 12px var(--font-mono);color:var(--text-body);padding:10px 16px 10px 0;border-bottom:1px solid var(--row-sep)">{{.Cadence}}</td>
<td style="padding:10px 16px 10px 0;border-bottom:1px solid var(--row-sep)">{{if .Active}}<span style="display:inline-block;width:8px;height:8px;border-radius:999px;background:var(--accent);margin-right:6px;vertical-align:middle"></span><span style="display:inline-flex;align-items:center;height:20px;padding:0 8px;border-radius:999px;border:1px solid var(--border-default);font:600 10px var(--font-mono);text-transform:uppercase;letter-spacing:0.06em;color:var(--text-secondary)">in flight</span>{{else if .Enabled}}<span style="display:inline-flex;align-items:center;height:20px;padding:0 8px;border-radius:999px;border:1px solid var(--border-default);font:600 10px var(--font-mono);text-transform:uppercase;letter-spacing:0.06em;color:var(--text-secondary)">enabled</span>{{else}}<span style="display:inline-flex;align-items:center;height:20px;padding:0 8px;border-radius:999px;border:1px solid var(--border-default);font:600 10px var(--font-mono);text-transform:uppercase;letter-spacing:0.06em;color:var(--text-muted)">disabled</span>{{end}}</td>
<td style="padding:10px 0;border-bottom:1px solid var(--row-sep)">
{{if .Active}}<span style="color:var(--text-muted)">running</span>
{{else if .Enabled}}<form method="post" action="/scans/trigger" style="display:inline;margin:0"><input type="hidden" name="kind" value="{{.Kind}}">{{template "backfield" $}}<button type="submit" style="font:500 12px var(--font-ui);padding:6px 14px;border:1px solid var(--accent);background:var(--accent);color:var(--on-accent);border-radius:10px;cursor:pointer">Run now</button></form>
{{else if .IsCold}}<span style="color:var(--text-muted)">disabled — opt a scope in on <a href="/scope" style="color:var(--link)">Scope</a></span>
{{else}}<span style="color:var(--text-muted)">disabled</span>{{end}}
</td>
</tr>{{end}}
</tbody>
</table>
</section>
{{end}}
{{end}}
{{end}}
`
