package main

import "html/template"

// Reports screen — canonical `/reports`. Restored to design-system parity against
// examples/console/Reports.jsx (shots 09→10) under ADR-0116 (design normative for
// look AND functionality; PARITY-CHART.md P2.4): three trend KPI cards with
// vs-last-batch deltas, the "Open signals over time" chart, a by-severity bar
// region, the scans-per-day heatmap, and the recurring-reports table with its row
// menu. Every data region is painted from a real derivation (P0.1/P0.2/P0.3) and
// degrades to a design-system empty/skeleton pattern only where a read is
// unavailable — never a fabricated figure. These classes are template-local CSS
// translated from design-system/components/* within the token vocabulary; this is
// restyling, not authoring (ADR-0109).
var _ = template.Must(tmpl.Parse(reportsTemplates))

// The New/Edit report-schedule wizard (#290, P0.6/T4) — the "schedulewizard"
// template, ported from design-system/examples/console/Reports.jsx's Wizard
// (Scope: name + Sections checkbox group; Cadence: CadenceSelect; Review:
// KeyValueList). The app is server-rendered with no client runtime, so the
// controlled React state becomes a post-back form exactly as the onboarding wizard
// does: the accumulated values ride hidden fields, Back/Next re-render the step, and
// the finishing submit posts to the create/edit route. The panel, step progress,
// checkbox group, select and key-value list are template-local CSS in the token
// vocabulary (restyling, not authoring — ADR-0109). See reports_schedule.go for the
// controlled flow and the real insert/update.
var _ = template.Must(tmpl.Parse(scheduleWizardTemplate))

const reportsTemplates = `
{{define "deltachip"}}<span style="display:inline-flex;align-items:center;gap:4px;height:18px;padding:0 7px;border-radius:var(--r-full);font:600 11px var(--mono);line-height:1;white-space:nowrap;transform:translateY(-2px);{{if eq .Tone "good"}}background:var(--ok-soft);border:1px solid var(--ok-border);color:var(--ok){{else if eq .Tone "bad"}}background:var(--danger-soft);border:1px solid var(--danger-border);color:var(--danger){{else}}background:var(--sunken);border:1px solid var(--hairline);color:var(--body){{end}}">{{if .Dir}}<svg viewBox="0 0 10 10" width="8" height="8" aria-hidden="true"{{if eq .Dir "down"}} style="transform:rotate(180deg)"{{end}}><path d="M5 8.5V1.5M1.8 4.7L5 1.5l3.2 3.2" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"></path></svg>{{end}}{{.Text}}</span>{{end}}

{{define "spark"}}<svg width="100%" height="{{.H}}" viewBox="0 0 {{.W}} {{.H}}" preserveAspectRatio="none" style="display:block;width:100%;overflow:visible" aria-hidden="true"><path d="{{.Area}}" fill="{{.Color}}" opacity="0.1"></path><polyline points="{{.Line}}" fill="none" stroke="{{.Color}}" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"></polyline><circle cx="{{.DotX}}" cy="{{.DotY}}" r="2.5" fill="{{.Color}}"></circle></svg>{{end}}

{{define "barchart"}}<div style="display:flex;flex-direction:column;gap:6px" aria-hidden="true"><div style="display:flex;align-items:flex-end;gap:4px;height:44px;border-bottom:1px solid var(--chart-grid);padding-bottom:1px">{{range .Bars}}<span title="{{.Title}}" style="flex:1;min-width:3px;height:{{.HeightPct}}%;min-height:2px;border-radius:3px 3px 0 0;background:var(--chart-1);opacity:{{if .Last}}1{{else}}0.45{{end}}"></span>{{end}}</div><div style="display:flex;justify-content:space-between;gap:8px"><span class="mono muted" style="font-size:9.5px">{{.LeftLabel}}</span><span class="mono muted" style="font-size:9.5px">{{.RightLabel}}</span></div></div>{{end}}

{{define "reports"}}{{template "head" .}}
{{template "chrome" .}}
<main style="display:flex;flex-direction:column;gap:var(--space-5)">

<header style="display:flex;align-items:center;gap:var(--space-4)">
  <div style="display:flex;flex-direction:column;gap:2px">
    <h1 style="margin:0;font-size:21px">Reports</h1>
    <span class="muted" style="font-size:12.5px">Trends and scheduled exports for the selected period.</span>
  </div>
  <div style="margin-left:auto;display:flex;gap:var(--space-2);align-items:center">
    <form method="get" action="/reports" style="margin:0">
      <label class="microlabel" for="reports-range" style="position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0)">Range</label>
      <select id="reports-range" name="weeks" onchange="this.form.submit()" aria-label="Reporting period" style="margin:0">
        {{range .RangeOptions}}<option value="{{.Weeks}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
      </select>
      <noscript><button type="submit" class="secondary" style="margin-left:var(--space-2)">Apply</button></noscript>
    </form>
    <details style="position:relative">
      <summary class="btn secondary" style="list-style:none;cursor:pointer;display:inline-flex;align-items:center;gap:6px">Export
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true"><path d="m6 9 6 6 6-6"></path></svg></summary>
      <div role="menu" style="position:absolute;right:0;top:calc(100% + 6px);min-width:170px;background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-md);box-shadow:var(--shadow-md);padding:6px;z-index:35;display:flex;flex-direction:column;gap:2px">
        <a role="menuitem" href="/reports/export?format=csv&amp;weeks={{.RangeWeeks}}" style="display:flex;align-items:center;gap:8px;padding:7px 10px;border-radius:var(--r-sm);font:500 13px var(--sans);color:var(--body);text-decoration:none"><svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z"></path><path d="M14 2v6h6"></path></svg>Export CSV</a>
        <a role="menuitem" href="/reports/export?format=json&amp;weeks={{.RangeWeeks}}" style="display:flex;align-items:center;gap:8px;padding:7px 10px;border-radius:var(--r-sm);font:500 13px var(--sans);color:var(--body);text-decoration:none"><svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z"></path><path d="M14 2v6h6"></path></svg>Export JSON</a>
      </div>
    </details>
  </div>
</header>

<div style="display:grid;grid-template-columns:repeat(3,1fr);gap:var(--space-5)">

  <section style="background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);padding:var(--space-5);display:flex;flex-direction:column;gap:14px;min-width:0">
    <div style="display:flex;align-items:baseline;gap:10px">
      <span class="microlabel">Open signals</span>
      <span class="mono muted" style="margin-left:auto;font-size:11px">{{.RangeLabel}}</span>
    </div>
    <div style="display:flex;flex-direction:column;gap:2px">
      <span style="display:flex;align-items:baseline;gap:8px">
        <span class="mono" style="font-size:28px;font-weight:600;color:var(--ink);line-height:1.1">{{if .HasOpenSignals}}{{.OpenSignals}}{{else}}&#8212;{{end}}</span>
        {{if .OpenDelta.Has}}{{template "deltachip" .OpenDelta}}{{end}}
      </span>
      <span class="muted" style="font-size:11.5px">vs previous batch</span>
    </div>
    {{if .HasOpenSpark}}<div style="margin-top:auto">{{template "spark" .OpenSpark}}</div>{{end}}
  </section>

  <section style="background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);padding:var(--space-5);display:flex;flex-direction:column;gap:14px;min-width:0">
    <div style="display:flex;align-items:baseline;gap:10px">
      <span class="microlabel">New assets discovered</span>
      <span class="mono muted" style="margin-left:auto;font-size:11px">{{.RangeLabel}}</span>
    </div>
    <div style="display:flex;flex-direction:column;gap:2px">
      <span style="display:flex;align-items:baseline;gap:8px">
        <span class="mono" style="font-size:28px;font-weight:600;color:var(--ink);line-height:1.1">{{if .HasDiscovery}}{{.DiscoveryCount}}{{else}}&#8212;{{end}}</span>
        {{if .DiscoveryDelta.Has}}{{template "deltachip" .DiscoveryDelta}}{{end}}
      </span>
      <span class="muted" style="font-size:11.5px">{{if .HasDiscovery}}{{.DiscoveryNames}} names &#183; {{.DiscoveryServices}} services{{else}}first appearances over the period{{end}}</span>
    </div>
    {{if .HasDiscovery}}<div style="margin-top:auto">{{template "barchart" .DiscoveryBars}}</div>{{end}}
  </section>

  <section style="background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);padding:var(--space-5);display:flex;flex-direction:column;gap:14px;min-width:0">
    <div style="display:flex;align-items:baseline;gap:10px">
      <span class="microlabel">Mean time to withdrawal</span>
      <span class="mono muted" style="margin-left:auto;font-size:11px">{{.RangeLabel}}</span>
    </div>
    <div style="display:flex;flex-direction:column;gap:2px">
      <span style="display:flex;align-items:baseline;gap:8px">
        <span class="mono" style="font-size:28px;font-weight:600;color:var(--ink);line-height:1.1">{{if .HasMTTW}}{{.MTTW}}{{else}}&#8212;{{end}}</span>
        {{if .MTTWDelta.Has}}{{template "deltachip" .MTTWDelta}}{{end}}
      </span>
      <span class="muted" style="font-size:11.5px">from appearance to withdrawal</span>
    </div>
    {{if .HasMTTWSpark}}<div style="margin-top:auto">{{template "spark" .MTTWSpark}}</div>{{end}}
  </section>

</div>

<section style="background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);padding:var(--space-5)">
  <div style="display:flex;flex-direction:column;gap:3px;margin-bottom:var(--space-4)">
    <span class="microlabel">Trend</span>
    <h2 style="margin:0;font-size:15px">Open signals over time</h2>
  </div>
  {{if .HasSignalSeries}}
  {{with .SignalSeries}}
  <div style="width:100%">
    <svg width="100%" height="{{.H}}" viewBox="0 0 {{.W}} {{.H}}" role="img" aria-label="Open signals over time" style="display:block;overflow:visible;max-width:100%">
      {{range .Grid}}<line x1="{{.X1}}" x2="{{.X2}}" y1="{{.Y}}" y2="{{.Y}}" stroke="{{.Stroke}}" stroke-width="1"></line><text x="{{.LabelX}}" y="{{.Y}}" dy="3" text-anchor="end" style="font:400 10.5px var(--mono);fill:var(--muted)">{{.Label}}</text>{{end}}
      <polyline points="{{.AllOpen}}" fill="none" stroke="var(--chart-1)" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"></polyline>
      <polyline points="{{.CritHigh}}" fill="none" stroke="var(--chart-2)" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"></polyline>
      {{range .XLabels}}<text x="{{.X}}" y="{{.Y}}" text-anchor="middle" style="font:400 10.5px var(--mono);fill:var(--muted)">{{.Text}}</text>{{end}}
    </svg>
    <div style="display:flex;gap:16px;flex-wrap:wrap;padding-left:40px;margin-top:8px">
      <span style="display:inline-flex;align-items:center;gap:6px"><span style="width:8px;height:8px;border-radius:2px;background:var(--chart-1)"></span><span style="font-size:12px;color:var(--body)">All open</span></span>
      <span style="display:inline-flex;align-items:center;gap:6px"><span style="width:8px;height:8px;border-radius:2px;background:var(--chart-2)"></span><span style="font-size:12px;color:var(--body)">Critical + high</span></span>
    </div>
  </div>
  {{end}}
  {{else}}
  <div class="emptystate">
    <div class="microlabel">No history yet</div>
    <h2>No signal history</h2>
    <p style="max-width:60ch;margin:var(--space-3) auto">Open-signal history builds here as signals are first raised over the period. Declare a scope on Scope and run a scan to start the series.</p>
    <a class="btn ghost" href="/signals">Go to Signals</a>
  </div>
  {{end}}
</section>

<div style="display:grid;grid-template-columns:380px 1fr;gap:var(--space-5);align-items:start">
  <div style="display:flex;flex-direction:column;gap:var(--space-5)">

    <section style="background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);padding:var(--space-5)">
      <div style="display:flex;flex-direction:column;gap:3px;margin-bottom:var(--space-4)">
        <span class="microlabel">Open signals</span>
        <h2 style="margin:0;font-size:15px">By severity</h2>
      </div>
      {{if .HasSeverity}}
      <div style="display:flex;flex-direction:column;gap:12px">
        {{range .BySeverity}}
        <div style="display:flex;align-items:center;gap:12px">
          <span class="mono" style="width:72px;font-size:11px;letter-spacing:0.06em;text-transform:uppercase;color:var(--muted)">{{.Label}}</span>
          <span style="flex:1;height:8px;border-radius:999px;background:var(--sunken);overflow:hidden">
            <span style="display:block;height:100%;width:{{.Pct}}%;border-radius:999px;background:{{if eq .Sev "critical"}}var(--sev-critical-dot){{else if eq .Sev "high"}}var(--sev-high-dot){{else if eq .Sev "medium"}}var(--sev-medium-dot){{else if eq .Sev "low"}}var(--sev-low-dot){{else}}var(--sev-info-dot){{end}}"></span>
          </span>
          <span class="mono" style="width:26px;text-align:right;font-size:12.5px;color:var(--body)">{{.Count}}</span>
        </div>
        {{end}}
      </div>
      {{else}}
      <div class="emptystate">
        <div class="microlabel">All quiet</div>
        <h2>No signals firing</h2>
        <p style="max-width:60ch;margin:var(--space-3) auto">Nothing is firing across the rule set right now. When a rule raises a signal it appears here, ranked by severity.</p>
        <a class="btn ghost" href="/signals">Go to Signals</a>
      </div>
      {{end}}
    </section>

    <section style="background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);padding:var(--space-5)">
      <div style="display:flex;flex-direction:column;gap:3px;margin-bottom:var(--space-4)">
        <span class="microlabel">Activity</span>
        <h2 style="margin:0;font-size:15px">Scans per day</h2>
      </div>
      {{if .HasHeat}}
      <div style="display:inline-flex;flex-direction:column;gap:var(--space-2)">
        <div role="img" aria-label="Scans per day, {{.RangeLabel}}" style="display:grid;grid-template-rows:repeat(7,12px);grid-auto-flow:column;grid-auto-columns:12px;gap:3px">
          {{range .Heat}}<span title="{{.Title}}" style="width:12px;height:12px;border-radius:3px;box-sizing:border-box;background:{{.Bg}};border:1px solid {{.Border}}"></span>{{end}}
        </div>
        <div style="display:flex;align-items:center;gap:6px">
          <span class="mono muted" style="font-size:10.5px">{{.RangeWeeks}} weeks ago</span>
          <span class="mono muted" style="margin-left:auto;font-size:10.5px">less</span>
          <span style="width:10px;height:10px;border-radius:3px;box-sizing:border-box;background:var(--sunken);border:1px solid var(--hairline)"></span>
          <span style="width:10px;height:10px;border-radius:3px;box-sizing:border-box;background:color-mix(in srgb, var(--chart-1) 28%, var(--surface))"></span>
          <span style="width:10px;height:10px;border-radius:3px;box-sizing:border-box;background:color-mix(in srgb, var(--chart-1) 48%, var(--surface))"></span>
          <span style="width:10px;height:10px;border-radius:3px;box-sizing:border-box;background:color-mix(in srgb, var(--chart-1) 72%, var(--surface))"></span>
          <span style="width:10px;height:10px;border-radius:3px;box-sizing:border-box;background:color-mix(in srgb, var(--chart-1) 100%, var(--surface))"></span>
          <span class="mono muted" style="font-size:10.5px">more</span>
          <span class="mono muted" style="font-size:10.5px;margin-left:8px">today</span>
        </div>
      </div>
      {{else}}
      <div class="emptystate">
        <div class="microlabel">Nothing dispatched</div>
        <h2>No scans in the {{.RangeLabel}}</h2>
        <p style="max-width:60ch;margin:var(--space-3) auto">Scan activity shows here once dispatches land. Declare a scope on Scope and trigger a scan, or wait for the next cadence.</p>
        <a class="btn ghost" href="/scans">Go to Scans</a>
      </div>
      {{end}}
    </section>

  </div>

  <section style="background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);padding:var(--space-5)">
    <div style="display:flex;align-items:center;gap:var(--space-3);margin-bottom:var(--space-4)">
      <div style="display:flex;flex-direction:column;gap:3px">
        <span class="microlabel">Scheduled</span>
        <h2 style="margin:0;font-size:15px">Recurring reports</h2>
      </div>
      <a class="btn ghost" href="/reports/schedule/new" style="margin-left:auto;display:inline-flex;align-items:center;gap:6px;text-decoration:none"><svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true"><path d="M12 5v14M5 12h14"></path></svg>New schedule</a>
    </div>
    {{if .Schedules}}
    <table class="vg-table">
      <thead><tr>
        <th>Report</th>
        <th style="width:170px">Cadence</th>
        <th style="width:90px">Format</th>
        <th style="width:90px;text-align:right">Last sent</th>
        <th style="width:58px" aria-label="Actions"></th>
      </tr></thead>
      <tbody>
      {{range .Schedules}}
        <tr>
          <td><span style="font:500 13px var(--sans);color:var(--ink)">{{.Name}}</span></td>
          <td class="mono">{{.Cadence}}</td>
          <td><span class="badge">{{.Format}}</span></td>
          <td class="mono" style="text-align:right">{{.LastSent}}</td>
          <td style="text-align:right;overflow:visible">
            <details style="position:relative;display:inline-block">
              <summary class="iconbtn" aria-label="Actions" style="list-style:none;cursor:pointer"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><circle cx="12" cy="12" r="1"></circle><circle cx="19" cy="12" r="1"></circle><circle cx="5" cy="12" r="1"></circle></svg></summary>
              <div role="menu" style="position:absolute;right:0;top:calc(100% + 6px);min-width:190px;background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-md);box-shadow:var(--shadow-md);padding:6px;z-index:35;display:flex;flex-direction:column;gap:2px">
                {{if .HasDelivery}}
                <a role="menuitem" href="{{.DeliveryHref}}" style="display:flex;align-items:center;gap:8px;padding:7px 10px;border-radius:var(--r-sm);font:500 13px var(--sans);color:var(--body);text-decoration:none"><svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"></path><circle cx="12" cy="12" r="3"></circle></svg>View last delivery</a>
                {{else}}
                <span role="menuitem" aria-disabled="true" title="No delivery yet" style="display:flex;align-items:center;gap:8px;padding:7px 10px;border-radius:var(--r-sm);font:500 13px var(--sans);color:var(--body);opacity:0.5;cursor:default"><svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"></path><circle cx="12" cy="12" r="3"></circle></svg>View last delivery</span>
                {{end}}
                <form method="post" action="/reports/schedule/run" style="margin:0"><input type="hidden" name="id" value="{{.ID}}"><button type="submit" role="menuitem" style="width:100%;text-align:left;border:none;background:transparent;cursor:pointer;display:flex;align-items:center;gap:8px;padding:7px 10px;border-radius:var(--r-sm);font:500 13px var(--sans);color:var(--body)"><svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.75"><polygon points="6 3 20 12 6 21 6 3"></polygon></svg>Run now</button></form>
                <a role="menuitem" href="/reports/schedule/{{.ID}}/edit" style="display:flex;align-items:center;gap:8px;padding:7px 10px;border-radius:var(--r-sm);font:500 13px var(--sans);color:var(--body);text-decoration:none"><svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M12 20h9"></path><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"></path></svg>Edit schedule</a>
                <span style="height:1px;background:var(--hairline);margin:4px 6px"></span>
                <form method="post" action="/reports/schedule/delete" style="margin:0"><input type="hidden" name="id" value="{{.ID}}"><button type="submit" role="menuitem" style="width:100%;text-align:left;border:none;background:transparent;cursor:pointer;display:flex;align-items:center;gap:8px;padding:7px 10px;border-radius:var(--r-sm);font:500 13px var(--sans);color:var(--danger)"><svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M3 6h18"></path><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path><line x1="10" x2="10" y1="11" y2="17"></line><line x1="14" x2="14" y1="11" y2="17"></line></svg>Delete schedule</button></form>
              </div>
            </details>
          </td>
        </tr>
      {{end}}
      </tbody>
    </table>
    {{else}}
    <div class="emptystate">
      <div class="microlabel">None yet</div>
      <h2>No recurring reports</h2>
      <p style="max-width:60ch;margin:var(--space-3) auto">Report scheduling is not yet available. When it lands, recurring exports declared here run on their cadence and their last delivery shows in this table.</p>
    </div>
    {{end}}
  </section>
</div>

</main>
{{template "foot" .}}{{end}}

`

const scheduleWizardTemplate = `
{{define "schedulewizard"}}{{template "head" .}}
{{template "chrome" .}}
<style>
.sw-wrap{display:flex;justify-content:center}
.sw-panel{background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-xl);box-shadow:var(--shadow-lg);padding:var(--space-6);width:560px;max-width:100%}
.sw-title{font-size:18px;margin:0 0 4px;color:var(--ink);letter-spacing:-0.015em}
.sw-desc{margin:0 0 var(--space-5);color:var(--muted);font-size:13px}
.sw-steps{display:flex;align-items:center;gap:8px;margin-bottom:20px}
.sw-conn{flex:1;min-width:12px;height:1px;background:var(--border-strong)}
.sw-conn.lit{background:var(--accent);opacity:0.4}
.sw-step{display:inline-flex;align-items:center;gap:8px}
.sw-num{width:22px;height:22px;border-radius:var(--r-full);flex:none;display:inline-flex;align-items:center;justify-content:center;font:600 11px var(--mono);border:1px solid var(--border-strong);color:var(--muted);background:transparent}
.sw-num svg{width:12px;height:12px}
.sw-num.cur{border:1.5px solid var(--accent);color:var(--accent)}
.sw-num.done{background:var(--accent-soft);border:1px solid transparent;color:var(--accent)}
.sw-step .lbl{font:400 12.5px var(--sans);color:var(--muted);white-space:nowrap}
.sw-step.cur .lbl{font-weight:600;color:var(--ink)}
.sw-body{display:flex;flex-direction:column;gap:16px}
.sw-flabel{font:500 12.5px var(--sans);color:var(--body);display:block;margin-bottom:4px}
.sw-secs{display:flex;flex-direction:column;gap:10px}
.sw-check{display:flex;align-items:center;gap:8px;font:400 13px var(--sans);color:var(--body);cursor:pointer}
.sw-check input{width:15px;height:15px;accent-color:var(--accent);cursor:pointer}
.sw-cad{display:flex;flex-direction:column;gap:8px}
.sw-cron{width:100%;height:34px;padding:0 10px;font:400 12px var(--mono)}
.sw-kv{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px 20px;padding:16px;background:var(--sunken);border-radius:var(--r-md)}
.sw-kv-item{display:flex;flex-direction:column;gap:3px;min-width:0}
.sw-kv-k{font:500 11px var(--mono);letter-spacing:0.07em;text-transform:uppercase;color:var(--muted)}
.sw-kv-v{font:400 12.5px var(--mono);color:var(--body);overflow-wrap:anywhere}
.sw-foot{display:flex;align-items:center;gap:var(--space-3);margin-top:var(--space-5)}
.sw-count{margin-right:auto;font:500 11px var(--mono);letter-spacing:0.06em;color:var(--muted)}
</style>
<main class="sw-wrap">
<section class="sw-panel">
<h1 class="sw-title">{{.WizardTitle}}</h1>
<p class="sw-desc">A recurring export, delivered on cadence.</p>

<div class="sw-steps">
{{range $i, $s := .Steps}}
{{if $i}}<span class="sw-conn{{if or $s.Done $s.Current}} lit{{end}}"></span>{{end}}
<span class="sw-step{{if $s.Current}} cur{{end}}">
<span class="sw-num{{if $s.Done}} done{{else if $s.Current}} cur{{end}}">{{if $s.Done}}<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6 9 17l-5-5"></path></svg>{{else}}{{$s.Num}}{{end}}</span>
<span class="lbl">{{$s.Title}}</span>
</span>
{{end}}
</div>

<form method="post" action="{{.FormAction}}">
<input type="hidden" name="step" value="{{.Step}}">
{{if .EditMode}}<input type="hidden" name="id" value="{{.ID}}">{{end}}
{{if ne .Step 0}}<input type="hidden" name="name" value="{{.Name}}">
{{range .SectionsKeys}}<input type="hidden" name="sections" value="{{.}}">{{end}}{{end}}
{{if ne .Step 1}}<input type="hidden" name="cad" value="{{.Cad}}">
<input type="hidden" name="cron" value="{{.Cron}}">{{end}}

<div class="sw-body">
{{if eq .Step 0}}
<label><span class="sw-flabel">Report name</span>
<input type="text" name="name" value="{{.Name}}" placeholder="Weekly exposure summary" spellcheck="false" autocomplete="off">
</label>
<div class="sw-secs">
<span class="microlabel">Sections</span>
{{range .Sections}}<label class="sw-check"><input type="checkbox" name="sections" value="{{.Key}}"{{if .Checked}} checked{{end}}>{{.Label}}</label>{{end}}
</div>
{{end}}

{{if eq .Step 1}}
<div class="sw-cad">
<label><span class="sw-flabel">Cadence</span>
<select name="cad">{{range .Cads}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Value}}</option>{{end}}</select>
</label>
{{if .Custom}}<input class="sw-cron" type="text" name="cron" value="{{.Cron}}" placeholder="0 8 * * 1" spellcheck="false" aria-label="Cron expression">{{end}}
</div>
{{end}}

{{if eq .Step 2}}
<div class="sw-kv">
{{range .Review}}<div class="sw-kv-item"><span class="sw-kv-k">{{.K}}</span><span class="sw-kv-v">{{.V}}</span></div>{{end}}
</div>
{{end}}
</div>

<div class="sw-foot">
<span class="sw-count">{{.StepNum}} / {{.StepTotal}}</span>
<a class="btn secondary" href="/reports" style="text-decoration:none">Cancel</a>
{{if gt .Step 0}}<button type="submit" class="secondary" name="action" value="back">Back</button>{{end}}
{{if .Last}}<button type="submit" name="action" value="finish">{{.FinishLabel}}</button>{{else}}<button type="submit" name="action" value="next">Next</button>{{end}}
</div>
</form>
</section>
</main>
{{template "foot" .}}{{end}}
`
