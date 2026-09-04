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

type scanTrigger interface {
	Trigger(ctx context.Context, kind string) (int, error)
}

type triggerScanView struct {
	Kind    string
	Enabled bool
	Active  bool
	Cadence string
	IsCold  bool
}

type triggerPanel struct {
	Scans []triggerScanView
}

type triggerOutcome struct {
	Tone        string
	Title       string
	Description string
}

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

	// Triggering a disabled scan is the one-off ADR-0044 forbids, so the flag is read to name it.
	if !sc.Enabled {
		detail := "A disabled scan cannot be triggered."
		if kind == scan.ColdKind {
			detail = "It runs the full-range sweep. Enable it by opting an address scope " +
				"into the full-range tier on Scope."
		}
		return triggerOutcome{"danger", "The " + kind + " scan is disabled", detail}, true
	}

	// The (scan, tick) key collapses same-second presses alone, so a long run needs this guard.
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
		s.serverError(w, "trigger scan: dispatch", err)
		return triggerOutcome{}, false
	}
	if n == 0 {
		// A cadence-lag skip also answers (0, nil), so the receipt names every cause (#1114).
		return triggerOutcome{"warn", "The " + kind + " scan enqueued no jobs",
			"Nothing covers it yet — no scope or vantage — or its current tick was already dispatched, or an earlier dispatch has not finished."}, true
	}
	return triggerOutcome{"neutral", kind + " scan dispatched",
		strconv.Itoa(n) + " " + plural(n, "job", "jobs") + " fanned out"}, true
}

func (s *server) triggerScan(w http.ResponseWriter, r *http.Request, acct db.Account) {
	out, ok := s.runTrigger(w, r)
	if !ok {
		return
	}
	s.flash.set(acct.ID, toastVM{Tone: out.Tone, Title: out.Title, Description: out.Description})
	// A trigger answers no confirm, so unlike stop and terminate it must not strip the dialog params.
	s.redirectBack(w, r, "/settings?tab=scans")
}

func (s *server) finishOnboarding(w http.ResponseWriter, r *http.Request, acct db.Account) {
	out, ok := s.runTrigger(w, r)
	if !ok {
		return
	}
	// Leaving the wizard is the point, so classEExempt exempts this route (ADR-0130).
	s.flashRedirect(w, r, acct.ID, "/scans", out.Tone, out.Title, out.Description)
}

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

func (s *server) chromeScanRunning(ctx context.Context) bool {
	active, err := s.activeDispatchKinds(ctx)
	// A dark pill beats a fabricated one, and restore.go's scanInFlight fails the other way.
	if err != nil {
		log.Printf("web: chrome: scan in-flight read: %v", err)
		return false
	}
	return len(active) > 0
}

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

var _ = template.Must(tmpl.Parse(triggerTemplates))

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
