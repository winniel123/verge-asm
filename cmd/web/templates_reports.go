package main

import "html/template"

// Reports screen — canonical `/reports` (new). Folds today's exposure board; the
// screen ticket (T-Reports) rewrites the body against examples/console/Reports.jsx
// (time-series chart, heatmap calendar, schedule wizard) plus scans analytics,
// shipping design-system empty-states where no data exists yet. Ported verbatim
// for T0.
var _ = template.Must(tmpl.Parse(reportsTemplates))

const reportsTemplates = `
{{define "exposure"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="microlabel">Derived · exposure</div>
<h1>Exposure</h1>
<p>The exposure board — the reachability of every service seen from both sides of your
boundary at once, never the raw inventory. Each service is placed by two measurements:
whether an internet vantage reached it, and whether an internal one did. The board is a
census, not an alert — the one move worth waking for, a service becoming reachable from
the internet, is called out under "what moved".</p>

<div class="avail">
<div><span class="k">Internet vantage</span>{{if .InternetPresent}}<span class="badge">present</span>{{else}}<span class="badge off">none</span>{{end}}</div>
<div><span class="k">Internal vantage</span>{{if .InternalPresent}}<span class="badge">present</span>{{else}}<span class="badge off">none</span>{{end}}</div>
</div>

{{if .NoServices}}
<div class="precond">
<div class="microlabel">Precondition · nothing to place</div>
<h2>No service in your estate yet</h2>
<p>A service is a port on an address your estate reaches for. None has been measured yet,
so there is nothing to place on the board. Declare a scope on Seeds and run the hot scan
once a resolution has cited an address.</p>
</div>
{{else}}

{{if not .Constructible}}
<div class="precond">
<div class="microlabel">Precondition · no exposure constructible</div>
<h2>Exposure needs both sides, and only one is looking</h2>
<p>Exposure is composed only from services measured by at least two vantage classes. Fewer
than two hold a current value here, so no exposure verdict is constructed — you see each
service's raw reach on the one side that looked, below, never a stand-in reading of the
side that did not. {{if not .InternetPresent}}There is no internet vantage: provision a
prober on Seeds to measure the side that matters most.{{else}}There is no internal vantage:
the internet reach renders on its own until one is configured.{{end}}</p>
</div>
{{end}}

{{if .WhatMoved}}
<div class="moved">
<div class="microlabel">What moved · flagship</div>
<h2 style="margin:4px 0 8px;font-size:14px">A service became reachable from the internet</h2>
<p class="muted" style="margin-bottom:8px">The internet reach of these services crossed not-reached to reached — the move the product
exists to catch. It fires on this leg alone, whether or not the internal side exists.</p>
<ul class="movedlist">
{{range .WhatMoved}}<li><a href="/subjects/service?key={{.}}">{{.}}</a></li>{{end}}
</ul>
</div>
{{end}}

{{if .HasBoard}}
<div class="microlabel">Populated board · {{.BoardTotal}} services measured from both sides</div>
<div class="board">
<div class="corner"><span class="microlabel">internet ↓</span><span class="microlabel">internal →</span></div>
<div class="colhead"><span class="microlabel">internal reached</span></div>
<div class="colhead"><span class="microlabel">internal not-reached</span></div>

<div class="rowhead"><span class="microlabel">internet reached</span></div>
<div class="cell hot">
<div class="microlabel">exposed</div>
<div class="count">{{len .Exposed}}</div>
{{if .Exposed}}<ul>{{range .Exposed}}<li><a href="/subjects/service?key={{.}}">{{.}}</a></li>{{end}}</ul>{{else}}<div class="none">—</div>{{end}}
</div>
<div class="cell">
<div class="microlabel">edge-only</div>
<div class="count">{{len .EdgeOnly}}</div>
{{if .EdgeOnly}}<ul>{{range .EdgeOnly}}<li><a href="/subjects/service?key={{.}}">{{.}}</a></li>{{end}}</ul>{{else}}<div class="none">—</div>{{end}}
</div>

<div class="rowhead"><span class="microlabel">internet not-reached</span></div>
<div class="cell">
<div class="microlabel">firewalled</div>
<div class="count">{{len .Firewalled}}</div>
{{if .Firewalled}}<ul>{{range .Firewalled}}<li><a href="/subjects/service?key={{.}}">{{.}}</a></li>{{end}}</ul>{{else}}<div class="none">—</div>{{end}}
</div>
<div class="cell">
<div class="microlabel">unreachable</div>
<div class="count">{{len .Unreachable}}</div>
{{if .Unreachable}}<ul>{{range .Unreachable}}<li><a href="/subjects/service?key={{.}}">{{.}}</a></li>{{end}}</ul>{{else}}<div class="none">—</div>{{end}}
</div>
</div>
{{end}}

{{if .OneLegged}}
<div class="section">
<div class="microlabel">One-legged · the surviving side's raw reach</div>
<h2>We only looked from one side</h2>
<p>These services were measured from a single vantage class, so no exposure verdict exists
for them — only the raw reach of the side that looked. This is never a fifth exposure value;
it is one measurement, honestly labelled with the side we did not see.</p>
<table>
<thead><tr><th>Service</th><th>Side looked</th><th>Reach</th><th>The other side</th></tr></thead>
<tbody>
{{range .OneLegged}}<tr>
<td><a class="mono" href="/subjects/service?key={{.Service}}">{{.Service}}</a></td>
<td><span class="badge">{{.Class}}</span></td>
<td class="mono">{{.Value}}</td>
<td class="muted">{{.Statement}}</td>
</tr>{{end}}
</tbody>
</table>
</div>
{{end}}

{{if .Broken}}
<div class="precond">
<div class="microlabel">Precondition · your rules changed</div>
<h2>Nothing to compare yet for these services</h2>
<p>The derivation that composes exposure moved for these services, so their two spans are
not comparable and no verdict is drawn across the break. This is your rules changing, not
your exposure — a new value ships as a break, never as rewritten history. The rest of the
board is unaffected.</p>
<ul class="movedlist">
{{range .Broken}}<li><a href="/subjects/service?key={{.}}">{{.}}</a></li>{{end}}
</ul>
</div>
{{end}}
{{end}}
</main>
{{template "foot" .}}{{end}}

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
        <p style="max-width:60ch;margin:var(--space-3) auto">A signal is a census member, not a scored finding — the register set is deliberately not a severity ramp, so there is nothing to rank here. See each rule's fired, did-not-fire and not-evaluable members on Signals.</p>
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
    <div class="emptystate">
      <div class="microlabel">None yet</div>
      <h2>No recurring reports</h2>
      <p style="max-width:60ch;margin:var(--space-3) auto">Report scheduling is not yet available. When it lands, recurring exports declared here run on their cadence and their last delivery shows in this table.</p>
    </div>
  </section>
</div>

</main>
{{template "foot" .}}{{end}}

`
