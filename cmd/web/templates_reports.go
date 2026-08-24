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

const reportsTemplates = `
{{define "deltachip"}}<span style="display:inline-flex;align-items:center;gap:4px;height:18px;padding:0 7px;border-radius:var(--r-full);font:600 11px var(--mono);line-height:1;white-space:nowrap;transform:translateY(-2px);{{if eq .Tone "good"}}background:var(--ok-soft);border:1px solid var(--ok-border);color:var(--ok){{else if eq .Tone "bad"}}background:var(--danger-soft);border:1px solid var(--danger-border);color:var(--danger){{else}}background:var(--sunken);border:1px solid var(--hairline);color:var(--body){{end}}">{{if .Dir}}<svg viewBox="0 0 10 10" width="8" height="8" aria-hidden="true"{{if eq .Dir "down"}} style="transform:rotate(180deg)"{{end}}><path d="M5 8.5V1.5M1.8 4.7L5 1.5l3.2 3.2" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"></path></svg>{{end}}{{.Text}}</span>{{end}}

{{define "spark"}}<svg width="100%" height="{{.H}}" viewBox="0 0 {{.W}} {{.H}}" preserveAspectRatio="none" style="display:block;width:100%;overflow:visible" aria-hidden="true"><path d="{{.Area}}" fill="{{.Color}}" opacity="0.1"></path><polyline points="{{.Line}}" fill="none" stroke="{{.Color}}" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"></polyline><circle cx="{{.DotX}}" cy="{{.DotY}}" r="2.5" fill="{{.Color}}"></circle></svg>{{end}}

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
      <span class="microlabel">Assets watched</span>
      <span class="mono muted" style="margin-left:auto;font-size:11px">{{.RangeLabel}}</span>
    </div>
    <div style="display:flex;flex-direction:column;gap:2px">
      <span style="display:flex;align-items:baseline;gap:8px">
        <span class="mono" style="font-size:28px;font-weight:600;color:var(--ink);line-height:1.1">{{if .HasAssets}}{{.AssetsCount}}{{else}}&#8212;{{end}}</span>
        {{if .AssetsDelta.Has}}{{template "deltachip" .AssetsDelta}}{{end}}
      </span>
      <span class="muted" style="font-size:11.5px">distinct names and services</span>
    </div>
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
            <span style="display:block;height:100%;width:{{.Pct}}%;border-radius:999px;background:{{.DotVar}}"></span>
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
      <button type="button" class="btn ghost" disabled aria-disabled="true" title="Report scheduling is not available yet" style="margin-left:auto;opacity:0.5;cursor:default;display:inline-flex;align-items:center;gap:6px">New schedule</button>
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
                <span role="menuitem" aria-disabled="true" title="Report scheduling is not wired yet" style="display:flex;align-items:center;gap:8px;padding:7px 10px;border-radius:var(--r-sm);font:500 13px var(--sans);color:var(--body);opacity:0.5;cursor:default"><svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.75"><polygon points="6 3 20 12 6 21 6 3"></polygon></svg>Run now</span>
                <span role="menuitem" aria-disabled="true" title="Report scheduling is not wired yet" style="display:flex;align-items:center;gap:8px;padding:7px 10px;border-radius:var(--r-sm);font:500 13px var(--sans);color:var(--body);opacity:0.5;cursor:default"><svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M12 20h9"></path><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"></path></svg>Edit schedule</span>
                <span style="height:1px;background:var(--hairline);margin:4px 6px"></span>
                <span role="menuitem" aria-disabled="true" title="Report scheduling is not wired yet" style="display:flex;align-items:center;gap:8px;padding:7px 10px;border-radius:var(--r-sm);font:500 13px var(--sans);color:var(--danger);opacity:0.5;cursor:default"><svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M3 6h18"></path><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path><line x1="10" x2="10" y1="11" y2="17"></line><line x1="14" x2="14" y1="11" y2="17"></line></svg>Delete schedule</span>
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
