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

`
