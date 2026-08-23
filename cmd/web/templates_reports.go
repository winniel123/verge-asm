package main

import "html/template"

// Reports screen — canonical `/reports` (new). Folds today's exposure board; the
// screen ticket (T-Reports) rewrites the body against examples/console/Reports.jsx
// (time-series chart, heatmap calendar, schedule wizard) plus scans analytics,
// shipping design-system empty-states where no data exists yet. Ported verbatim
// for T0.
var _ = template.Must(tmpl.Parse(reportsTemplates))

const reportsTemplates = `
{{define "reports"}}{{template "head" .}}
{{template "chrome" .}}
<main style="display:flex;flex-direction:column;gap:var(--space-5)">

<header style="display:flex;align-items:center;gap:var(--space-4)">
  <div style="display:flex;flex-direction:column;gap:2px">
    <h1 style="margin:0;font-size:21px">Reports</h1>
    <span class="muted" style="font-size:12.5px">Operational activity and scheduled exports for the selected period.</span>
  </div>
  <div style="margin-left:auto;display:flex;gap:var(--space-2);align-items:center">
    <span class="badge" title="Range selection is not wired yet">Last 12 weeks</span>
    <button type="button" class="secondary" disabled title="Export is not wired yet" style="opacity:0.6;cursor:default">Export CSV</button>
  </div>
</header>

<div style="display:grid;grid-template-columns:repeat(3,1fr);gap:var(--space-5)">
  <section style="background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);padding:var(--space-5);display:flex;flex-direction:column;gap:var(--space-3);min-width:0">
    <div style="display:flex;align-items:baseline;gap:10px">
      <span class="microlabel">Open signals</span>
      <span class="mono muted" style="margin-left:auto;font-size:11px">now</span>
    </div>
    <span class="mono" style="font-size:28px;font-weight:600;color:var(--ink);line-height:1.1">{{if .HasOpenSignals}}{{.OpenSignals}}{{else}}&#8212;{{end}}</span>
    <span class="muted" style="font-size:11.5px">signals firing right now, across every rule</span>
  </section>
  <section style="background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);padding:var(--space-5);display:flex;flex-direction:column;gap:var(--space-3);min-width:0">
    <div style="display:flex;align-items:baseline;gap:10px">
      <span class="microlabel">Scans run</span>
      <span class="mono muted" style="margin-left:auto;font-size:11px">last 12 weeks</span>
    </div>
    <span class="mono" style="font-size:28px;font-weight:600;color:var(--ink);line-height:1.1">{{.ScansWindow}}</span>
    <span class="muted" style="font-size:11.5px">scans dispatched in the period</span>
  </section>
  <section style="background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);padding:var(--space-5);display:flex;flex-direction:column;gap:var(--space-3);min-width:0">
    <div style="display:flex;align-items:baseline;gap:10px">
      <span class="microlabel">In flight</span>
      <span class="mono muted" style="margin-left:auto;font-size:11px">now</span>
    </div>
    <span class="mono" style="font-size:28px;font-weight:600;color:var(--ink);line-height:1.1">{{.ActiveScans}}</span>
    <span class="muted" style="font-size:11.5px">scans still running</span>
  </section>
</div>

<section style="background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);padding:var(--space-5)">
  <div style="display:flex;flex-direction:column;gap:3px;margin-bottom:var(--space-4)">
    <span class="microlabel">Trend</span>
    <h2 style="margin:0;font-size:15px">Open signals over time</h2>
  </div>
  <div class="emptystate">
    <div class="microlabel">No series</div>
    <h2>Signals are a current-state census</h2>
    <p style="max-width:70ch;margin:var(--space-3) auto">A signal census is never a delta, trend or series — subtracting two censuses conflates a moved population with a moved rule. The current count is on the band above; open Signals for the live census.</p>
    <a class="btn ghost" href="/signals">Go to Signals</a>
  </div>
</section>

<div style="display:grid;grid-template-columns:380px 1fr;gap:var(--space-5);align-items:start">
  <div style="display:flex;flex-direction:column;gap:var(--space-5)">

    <section style="background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);padding:var(--space-5)">
      <div style="display:flex;flex-direction:column;gap:3px;margin-bottom:var(--space-4)">
        <span class="microlabel">Open signals</span>
        <h2 style="margin:0;font-size:15px">By severity</h2>
      </div>
      <div class="emptystate">
        <div class="microlabel">No severity ramp</div>
        <h2>Signals carry no severity</h2>
        <p style="max-width:60ch;margin:var(--space-3) auto">A signal is a census member, not a scored one — the register set is deliberately not a severity ramp, so there is nothing to rank here. See each rule's fired, did-not-fire and not-evaluable members on Signals.</p>
        <a class="btn ghost" href="/signals">Go to Signals</a>
      </div>
    </section>

    <section style="background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);padding:var(--space-5)">
      <div style="display:flex;flex-direction:column;gap:3px;margin-bottom:var(--space-4)">
        <span class="microlabel">Activity</span>
        <h2 style="margin:0;font-size:15px">Scans per day</h2>
      </div>
      {{if .HasHeat}}
      <div style="display:inline-flex;flex-direction:column;gap:var(--space-2)">
        <div role="img" aria-label="Scans per day, last 12 weeks" style="display:grid;grid-template-rows:repeat(7,12px);grid-auto-flow:column;grid-auto-columns:12px;gap:3px">
          {{range .Heat}}<span title="{{.Title}}" style="width:12px;height:12px;border-radius:3px;box-sizing:border-box;background:{{.Bg}};border:1px solid {{.Border}}"></span>{{end}}
        </div>
        <div style="display:flex;align-items:center;gap:6px">
          <span class="mono muted" style="font-size:10.5px">12 weeks ago</span>
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
        <h2>No scans in the last 12 weeks</h2>
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
      <details style="margin-left:auto;position:relative">
        <summary class="btn ghost" style="list-style:none;cursor:pointer;display:inline-flex;align-items:center;gap:6px">New schedule</summary>
        <div class="scrim"></div>
        <div style="position:fixed;inset:0;z-index:41;display:flex;align-items:center;justify-content:center;pointer-events:none;padding:var(--space-5)">
          <div class="dialog-panel" style="width:560px;max-width:calc(100vw - 32px);pointer-events:auto;max-height:calc(100vh - 48px);overflow-y:auto">
            <div class="microlabel">Scheduled export</div>
            <h2 style="margin:4px 0 6px;font-size:16px">New report schedule</h2>
            <p class="muted" style="margin-bottom:var(--space-4)">A recurring export, delivered on cadence. Report scheduling is not yet wired; this preview shows the fields a schedule will capture.</p>

            <div style="display:flex;flex-direction:column;gap:var(--space-2)">
              <span class="microlabel">Step 1 &#183; Scope</span>
              <label style="margin-bottom:0"><span>Report name</span>
                <input type="text" placeholder="Weekly exposure summary" disabled></label>
              <span class="microlabel">Sections</span>
              <label class="check"><input type="checkbox" checked disabled><span>Summary KPIs</span></label>
              <label class="check"><input type="checkbox" checked disabled><span>New assets</span></label>
              <label class="check"><input type="checkbox" checked disabled><span>Signal changes</span></label>
              <label class="check"><input type="checkbox" disabled><span>Coverage gaps</span></label>
            </div>

            <div style="display:flex;flex-direction:column;gap:var(--space-2);margin-top:var(--space-4)">
              <span class="microlabel">Step 2 &#183; Cadence</span>
              <label style="margin-bottom:0"><span>Cadence</span>
                <select disabled>
                  <option>Every 6h</option>
                  <option>Daily &#183; 08:00</option>
                  <option selected>Weekly &#183; mon 09:00</option>
                  <option>Monthly &#183; 1st</option>
                  <option>Custom&#8230;</option>
                </select></label>
            </div>

            <div style="display:flex;flex-direction:column;gap:var(--space-2);margin-top:var(--space-4)">
              <span class="microlabel">Step 3 &#183; Review</span>
              <div class="census" style="margin-top:0">
                <div class="kv"><span class="k">Sections</span><span class="mono">Summary KPIs, New assets, Signal changes</span></div>
                <div class="kv"><span class="k">Cadence</span><span class="mono">weekly &#183; mon 09:00</span></div>
                <div class="kv"><span class="k">Format</span><span class="mono">pdf</span></div>
              </div>
            </div>

            <div class="dialog-actions">
              <span class="muted" style="font-size:12px">Close with the New schedule toggle — creating a schedule needs a backend (see #285).</span>
            </div>
          </div>
        </div>
      </details>
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
