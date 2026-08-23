package main

import "html/template"

// Inventory screen — canonical `/inventory`. Folds today's inventory + subjects
// list + the Name/Service/Endpoint detail views (and the shared `recordrows`
// partial and the `subject-missing` page). The screen ticket (T-Inventory)
// rewrites the body against examples/console/Inventory.jsx (saved views, column
// picker, density, hover peeks). Ported verbatim for T0.
var _ = template.Must(tmpl.Parse(inventoryTemplates))

const inventoryTemplates = `
{{define "inventory"}}{{template "head" .}}
{{template "chrome" .}}
<style>
/* Inventory delta (#310) — navigable rows + saved-views/columns/density controls,
   translated from design-system/examples/console/Inventory.jsx within the existing
   token vocabulary (restyling, not authoring — ADR-0109). */
.invrow { cursor: pointer; }
.invrow:hover { background: var(--sunken); }
.invrow:focus, .invrow:focus-visible { outline: none; background: var(--accent-soft); }
.invrow:focus > td:first-child, .invrow:focus-visible > td:first-child { box-shadow: inset 3px 0 0 var(--accent); }
.invtables[data-density="compact"] .vg-table td,
.invtables[data-density="compact"] .vg-table th { padding-top: 8px; padding-bottom: 8px; }
.invtables[data-density="comfortable"] .vg-table td,
.invtables[data-density="comfortable"] .vg-table th { padding-top: var(--space-4); padding-bottom: var(--space-4); }
.invtables.hide-type [data-col="type"],
.invtables.hide-holds [data-col="holds"],
.invtables.hide-since [data-col="since"] { display: none; }
.seg { display: inline-flex; border: 1px solid var(--hairline); border-radius: var(--r-full); overflow: hidden; }
.seg button { border: none; border-radius: 0; background: transparent; color: var(--muted);
  font-family: var(--mono); font-size: 11px; padding: 4px 10px; cursor: pointer; }
.seg button[aria-pressed="true"] { background: var(--accent-soft); color: var(--link); }
</style>
<main>
<div style="display:flex;align-items:flex-start;gap:16px;margin-bottom:var(--space-4)">
<div>
<div class="microlabel">Observed · inventory</div>
<h1 style="margin-bottom:4px">Inventory</h1>
<span class="muted">Everything you expose, watched for drift — the actual values behind the verdicts.</span>
</div>
<div style="margin-left:auto;display:flex;gap:8px;align-items:center">
<button class="btn secondary" disabled title="Nothing to export until a value is folded">Export CSV</button>
<a class="btn" href="/scope" style="text-decoration:none">Add seed</a>
</div>
</div>

<p class="muted" style="max-width:82ch">What your estate holds right now — the addresses a name
resolves to, the records it carries, the certificate a service presents, the identity an
endpoint returns. Each row is a subject; open it for the asset's full record, or expand a value
to its individual records. A withdrawn subject holds no current span and so is not here. As on
the change views there is no total: your estate's completeness is yours alone to state.</p>

<div style="display:flex;gap:12px;align-items:center;flex-wrap:wrap;margin-bottom:var(--space-4)">
<div style="display:inline-flex;gap:6px;align-items:center;flex-wrap:wrap">
<span class="chip" style="background:var(--accent-soft);border-color:var(--accent-soft);color:var(--link)">All subjects</span>
{{range .Groups}}<a class="chip" href="#{{.Kind}}" style="text-decoration:none">{{.Label}}</a>{{end}}
</div>
<span style="margin-left:auto;display:inline-flex;gap:12px;align-items:center">
<span class="microlabel" data-invnav-hint>j/k or arrows to move · enter opens</span>
<span style="display:inline-flex;gap:6px;align-items:center">
<span class="microlabel">Density</span>
<span class="seg" role="group" aria-label="Row density">
<button type="button" data-density="compact" aria-pressed="true">Compact</button>
<button type="button" data-density="comfortable" aria-pressed="false">Comfortable</button>
</span>
</span>
<details style="position:relative">
<summary class="iconbtn" style="list-style:none;cursor:pointer">Columns</summary>
<div style="position:absolute;right:0;top:calc(100% + 6px);min-width:190px;background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-md);box-shadow:var(--shadow-md);padding:6px;z-index:35;display:flex;flex-direction:column;gap:2px">
<label class="check" style="padding:6px 8px;margin:0"><input type="checkbox" checked disabled><span>Subject</span></label>
<label class="check" style="padding:6px 8px;margin:0"><input type="checkbox" checked data-col-toggle="type"><span>Type</span></label>
<label class="check" style="padding:6px 8px;margin:0"><input type="checkbox" checked data-col-toggle="holds"><span>Holds</span></label>
<label class="check" style="padding:6px 8px;margin:0"><input type="checkbox" checked data-col-toggle="since"><span>Since</span></label>
</div>
</details>
</span>
</div>

<div class="invtables" data-density="compact">
{{if .Groups}}
{{range .Groups}}
<div class="section" id="{{.Kind}}">
<div class="microlabel" style="margin-bottom:var(--space-3)">{{.Label}}</div>
<table class="vg-table">
<thead><tr>
<th>Subject</th>
<th data-col="type" style="width:110px">Type</th>
<th data-col="holds">Holds</th>
<th data-col="since" style="width:1%;white-space:nowrap;text-align:right">Since</th>
</tr></thead>
<tbody>
{{range .Subjects}}<tr{{if .Link}} class="invrow" tabindex="0" data-href="{{.Link}}" role="link" aria-label="Open {{.Key}}"{{end}}>
<td>{{if .Link}}<a class="mono" href="{{.Link}}" title="{{range .Facets}}{{.Label}}: {{.Summary}}&#10;{{end}}">{{.Key}}</a>{{else}}<span class="mono">{{.Key}}</span>{{end}}</td>
<td data-col="type"><span class="badge">{{.Type}}</span></td>
<td data-col="holds">
{{range .Facets}}<div style="display:flex;gap:8px;align-items:baseline;padding:2px 0">
<span class="microlabel" style="min-width:150px;white-space:nowrap">{{.Label}}</span>
<span>{{if .IsGap}}<span class="chip loss">Gap</span>{{else if .Details}}<details class="spanrecords" style="display:inline"><summary><span class="badge">{{.Summary}}</span></summary>{{template "recordrows" .Details}}</details>{{else}}<span class="badge">{{.Summary}}</span>{{end}}</span>
</div>{{end}}
</td>
<td data-col="since" class="mono muted" style="text-align:right;white-space:nowrap;vertical-align:top">{{with index .Facets 0}}{{.Since}}{{end}}</td>
</tr>{{end}}
</tbody>
</table>
</div>
{{end}}
{{else}}
<div class="emptystate">
<h2>No population measured yet</h2>
<p>No subject holds an open span yet. Declare a scope on Scope and let a Scan measure a value;
the inventory fills as facets are folded.</p>
</div>
{{end}}
</div>
<script>
/* Inventory delta (#310): roving keyboard focus over the rows (j/k or arrows to
   move, Enter opens the Asset detail), whole-row click, plus the density toggle and
   column picker. Vanilla, guarded so the command palette owns keys when it is open
   and typing in a field is never intercepted. */
(function () {
  var container = document.querySelector(".invtables");
  if (!container) return;
  var rows = Array.prototype.slice.call(container.querySelectorAll("tr.invrow"));
  function go(tr) { if (tr && tr.getAttribute("data-href")) window.location.assign(tr.getAttribute("data-href")); }
  rows.forEach(function (tr) {
    tr.addEventListener("click", function (e) {
      // Let an expander (the span-records details) toggle without navigating.
      if (e.target.closest("summary, details")) return;
      go(tr);
    });
    tr.addEventListener("keydown", function (e) {
      if (e.key === "Enter") { e.preventDefault(); go(tr); }
    });
  });
  document.addEventListener("keydown", function (e) {
    var pal = document.getElementById("cmdk");
    if (pal && !pal.hasAttribute("hidden")) return;
    var t = e.target;
    if (t && (t.tagName === "INPUT" || t.tagName === "TEXTAREA" || t.isContentEditable)) return;
    if (!rows.length) return;
    var i = rows.indexOf(document.activeElement);
    if (e.key === "j" || e.key === "ArrowDown") { e.preventDefault(); (rows[i < 0 ? 0 : Math.min(i + 1, rows.length - 1)]).focus(); }
    else if (e.key === "k" || e.key === "ArrowUp") { e.preventDefault(); (rows[i < 0 ? rows.length - 1 : Math.max(i - 1, 0)]).focus(); }
  });
  document.querySelectorAll("[data-density]").forEach(function (btn) {
    if (btn.tagName !== "BUTTON") return;
    btn.addEventListener("click", function () {
      var d = btn.getAttribute("data-density");
      container.setAttribute("data-density", d);
      document.querySelectorAll(".seg [data-density]").forEach(function (b) {
        b.setAttribute("aria-pressed", b === btn ? "true" : "false");
      });
    });
  });
  document.querySelectorAll("[data-col-toggle]").forEach(function (cb) {
    cb.addEventListener("change", function () {
      container.classList.toggle("hide-" + cb.getAttribute("data-col-toggle"), !cb.checked);
    });
  });
})();
</script>
</main>
{{template "foot" .}}{{end}}

{{define "recordrows"}}<table class="records"><tbody>
{{range .}}<tr>{{if .Type}}<td class="rrtype"><span class="badge">{{.Type}}</span></td>{{else}}<td class="rrtype"></td>{{end}}<td class="mono">{{.Data}}</td></tr>{{end}}
</tbody></table>{{end}}

{{define "subject"}}{{template "head" .}}
{{template "chrome" .}}
<main>
{{with .Subject}}
<div style="display:flex;align-items:flex-start;gap:16px;margin-bottom:var(--space-5)">
<div>
<div class="microlabel">Observed · Name</div>
<h1 class="mono" style="margin-bottom:0">{{.Name}}</h1>
</div>
<div style="margin-left:auto;display:flex;gap:8px;align-items:center">
<span class="badge">Name</span>
<a class="btn secondary" href="/inventory" style="text-decoration:none">Back to inventory</a>
</div>
</div>
{{if .Withdrawn}}
<div class="notice">This name is withdrawn — it names a population of no current member. Its timelines are closed. It is reached by its own key and never appears in the listing.</div>
{{end}}

<div class="section">
<div class="microlabel">Why is this here</div>
<h2>Citation chain</h2>
<p>Following a subject's citations backwards always terminates at a Seed you declared — that is what makes "why is this here" answerable for everything in the estate.</p>
<ol class="chain">
{{range .Citation}}<li>
<div class="microlabel">{{.Label}}</div>
<div class="mono chainval">{{.Value}}</div>
{{if .Detail}}<div class="muted">{{.Detail}}</div>{{end}}
</li>{{end}}
</ol>
{{if not .CitationTerminated}}<p class="muted">The chain does not reach a declared Seed. That is an integrity gap, not a normal state — every subject in the estate should trace back to a scope you declared.</p>{{end}}
</div>

<div class="section">
<div class="microlabel">Current · resolution</div>
<h2>Resolution</h2>
{{if .Resolution}}
<div class="kv"><div class="k">Outcome</div><div><span class="badge">{{.Resolution}}</span></div></div>
{{if .Addresses}}<div class="kv"><div class="k">Addresses</div><div class="mono">{{range .Addresses}}{{.}}<br>{{end}}</div></div>{{end}}
{{else}}<p class="muted">No resolution value recorded.</p>{{end}}
</div>

<div class="section">
<div class="microlabel">Timelines</div>
<h2>Current and closed timelines</h2>
{{if .Timelines}}
<p class="muted">Each timeline is one period a value was held. A Break marks two spans the drift engine may not compare, naming the leaf that moved; it is derived on read and never stored.</p>
{{range .Timelines}}
<div class="timeline">
<div class="microlabel">{{.Label}}</div>
{{if .Current}}
<div class="kv"><div class="k">Current</div><div>{{if .Current.IsGap}}<span class="badge">Gap</span>{{else if .Current.Details}}<details class="spanrecords"><summary><span class="badge">{{.Current.Value}}</span></summary>{{template "recordrows" .Current.Details}}</details>{{else}}<span class="badge">{{.Current.Value}}</span>{{end}} <span class="muted mono">since {{.Current.OpenedAt}}</span></div></div>
{{else}}
<div class="kv"><div class="k">Current</div><div class="muted">Closed — this timeline holds no current value.</div></div>
{{end}}
{{if .Breaks}}{{range .Breaks}}
<div class="notice">Break at {{.At}} — not comparable across it. Leaf that moved: <span class="mono">{{.MovedLeaves}}</span></div>
{{end}}{{end}}
{{if .Closed}}
<table class="closedspans">
<thead><tr><th>Value</th><th>Opened</th><th>Closed</th><th>Ground</th></tr></thead>
<tbody>
{{range .Closed}}<tr>
<td>{{if .IsGap}}<span class="muted">Gap</span>{{else if .Details}}<details class="spanrecords"><summary><span class="mono">{{.Value}}</span></summary>{{template "recordrows" .Details}}</details>{{else}}<span class="mono">{{.Value}}</span>{{end}}</td>
<td class="mono">{{.OpenedAt}}</td>
<td class="mono">{{.ClosedAt}}</td>
<td>{{if .Reason}}<span class="badge">{{.Reason}}</span>{{else}}<span class="muted">—</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
{{end}}
</div>
{{end}}
{{else}}
<p class="muted">No timeline has been folded yet. A Span opens when the dns Scan first measures a value for this name; re-running it with a changed answer closes the open span and opens the next.</p>
{{end}}
</div>

<div class="section">
<div class="microlabel">Rules</div>
<h2>Rules over this subject</h2>
<p class="muted">Every rule whose predicate domain includes this subject renders here, each carrying its own versioned verdict. Wired up by ticket 22.</p>
</div>
{{end}}
</main>
{{template "foot" .}}{{end}}

{{define "service"}}{{template "head" .}}
{{template "chrome" .}}
<main>
{{with .Service}}
<div style="display:flex;align-items:flex-start;gap:16px;margin-bottom:var(--space-5)">
<div>
<div class="microlabel">Observed · Service</div>
<h1 class="mono" style="margin-bottom:0">{{.Key}}</h1>
</div>
<div style="margin-left:auto;display:flex;gap:8px;align-items:center">
<span class="badge">Service</span>
<a class="btn secondary" href="/inventory" style="text-decoration:none">Back to inventory</a>
</div>
</div>
{{if .Withdrawn}}
<div class="notice">This service's address has left the estate — no current resolution cites it and no Seed covers it. It names a population of no current member; its timelines are closed and it is reached by its own key.</div>
{{end}}

<div class="section">
<div class="microlabel">Why is this here</div>
<h2>Citation chain</h2>
<p>A Service is an (address, port, transport) triple. Its membership is its address's membership restated — an address is in the estate exactly while a current resolution cites it or a Seed covers it — so the chain runs from the Service down through its address to the Seed you declared.</p>
<ol class="chain">
{{range .Citation}}<li>
<div class="microlabel">{{.Label}}</div>
<div class="mono chainval">{{.Value}}</div>
{{if .Detail}}<div class="muted">{{.Detail}}</div>{{end}}
</li>{{end}}
</ol>
{{if not .CitationTerminated}}<p class="muted">The chain does not reach a declared Seed. For a service whose address a resolution cites, that is the address's name-scope Seed, one hop past the citing name; for one only a Seed covers, it is the address scope directly.</p>{{end}}
</div>

<div class="section">
<div class="microlabel">Current · reachability</div>
<h2>Reachability</h2>
<div class="kv"><div class="k">Address</div><div class="mono">{{.Address}}</div></div>
<div class="kv"><div class="k">Port</div><div class="mono">{{.Port}}/{{.Transport}}</div></div>
{{if .ReachGap}}
<div class="kv"><div class="k">Verdict</div><div><span class="badge">Gap</span></div></div>
<div class="notice">{{.ReachGapReason}}. From this vantage we cannot tell a real origin service behind the edge from the edge answering for it, so the reach is undiscriminated — a Gap, not <span class="mono">reached</span>. Declare your origin IPs as an address scope to measure the real surface.</div>
{{else if .Reach}}
<div class="kv"><div class="k">Verdict</div><div><span class="badge">{{.Reach}}</span></div></div>
{{else}}<p class="muted">No reachability value recorded.</p>{{end}}
</div>

<div class="section">
<div class="microlabel">Timelines</div>
<h2>Current and closed timelines</h2>
{{if .Timelines}}
<p class="muted">Each timeline is one period a value was held. A Break marks two spans the drift engine may not compare, naming the leaf that moved; it is derived on read and never stored.</p>
{{range .Timelines}}
<div class="timeline">
<div class="microlabel">{{.Label}}</div>
{{if .Current}}
<div class="kv"><div class="k">Current</div><div>{{if .Current.IsGap}}<span class="badge">Gap</span>{{else if .Current.Details}}<details class="spanrecords"><summary><span class="badge">{{.Current.Value}}</span></summary>{{template "recordrows" .Current.Details}}</details>{{else}}<span class="badge">{{.Current.Value}}</span>{{end}} <span class="muted mono">since {{.Current.OpenedAt}}</span></div></div>
{{else}}
<div class="kv"><div class="k">Current</div><div class="muted">Closed — this timeline holds no current value.</div></div>
{{end}}
{{if .Breaks}}{{range .Breaks}}
<div class="notice">Break at {{.At}} — not comparable across it. Leaf that moved: <span class="mono">{{.MovedLeaves}}</span></div>
{{end}}{{end}}
{{if .Closed}}
<table class="closedspans">
<thead><tr><th>Value</th><th>Opened</th><th>Closed</th><th>Ground</th></tr></thead>
<tbody>
{{range .Closed}}<tr>
<td>{{if .IsGap}}<span class="muted">Gap</span>{{else if .Details}}<details class="spanrecords"><summary><span class="mono">{{.Value}}</span></summary>{{template "recordrows" .Details}}</details>{{else}}<span class="mono">{{.Value}}</span>{{end}}</td>
<td class="mono">{{.OpenedAt}}</td>
<td class="mono">{{.ClosedAt}}</td>
<td>{{if .Reason}}<span class="badge">{{.Reason}}</span>{{else}}<span class="muted">—</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
{{end}}
</div>
{{end}}
{{else}}
<p class="muted">No timeline has been folded yet. A Span opens when the hot Scan first reaches for this port; re-running it with the port opening or closing closes the open span and opens the next.</p>
{{end}}
</div>

<div class="section">
<div class="microlabel">Rules</div>
<h2>Rules over this subject</h2>
<p class="muted">Every rule whose predicate domain includes this subject renders here, each carrying its own versioned verdict. Wired up by ticket 22.</p>
</div>
{{end}}
</main>
{{template "foot" .}}{{end}}

{{define "endpoint"}}{{template "head" .}}
{{template "chrome" .}}
<main>
{{with .Endpoint}}
<div style="display:flex;align-items:flex-start;gap:16px;margin-bottom:var(--space-5)">
<div>
<div class="microlabel">Observed · Endpoint</div>
<h1 class="mono" style="margin-bottom:0">{{if .Nameless}}<span class="muted">(nameless)</span> {{end}}{{.Key}}</h1>
</div>
<div style="margin-left:auto;display:flex;gap:8px;align-items:center">
<span class="badge">Endpoint</span>
<a class="btn secondary" href="/inventory" style="text-decoration:none">Back to inventory</a>
</div>
</div>
{{if .Withdrawn}}
<div class="notice">This endpoint's service has left the estate — no current resolution cites its address and no Seed covers it. It names a population of no current member; its timelines are closed and it is reached by its own key. An endpoint closes when either leg — its Name or its Service — withdraws.</div>
{{end}}

<div class="section">
<div class="microlabel">Why is this here</div>
<h2>Citation chain</h2>
<p>An Endpoint is a (Name, Service) pair — the only key under which HTTP identity is single-valued. Its membership is its Service's, restated: a Service is in the estate exactly while a current resolution cites its address or a Seed covers it, so the chain runs from the Endpoint through its Name and Service legs down to the Seed you declared.</p>
<ol class="chain">
{{range .Citation}}<li>
<div class="microlabel">{{.Label}}</div>
<div class="mono chainval">{{.Value}}</div>
{{if .Detail}}<div class="muted">{{.Detail}}</div>{{end}}
</li>{{end}}
</ol>
{{if not .CitationTerminated}}<p class="muted">The chain does not reach a declared Seed. For an endpoint whose service address a resolution cites, that is the address's name-scope Seed, one hop past the citing name; for one only a Seed covers, it is the address scope directly.</p>{{end}}
</div>

<div class="section">
<div class="microlabel">Current · http-identity</div>
<h2>HTTP identity</h2>
<div class="kv"><div class="k">Name</div><div class="mono">{{if .Nameless}}<span class="muted">nameless endpoint</span>{{else}}{{.Name}}{{end}}</div></div>
<div class="kv"><div class="k">Service</div><div class="mono">{{.Service}}</div></div>
{{if .HasIdentity}}
<div class="kv"><div class="k">Status</div><div><span class="badge">{{.Status}}</span></div></div>
{{if .Server}}<div class="kv"><div class="k">Server</div><div class="mono">{{.Server}}</div></div>{{end}}
{{if .Title}}<div class="kv"><div class="k">Title</div><div class="mono">{{.Title}}</div></div>{{end}}
{{if .WWWAuthenticate}}<div class="kv"><div class="k">WWW-Authenticate</div><div class="mono">{{.WWWAuthenticate}}</div></div>{{end}}
{{if .RedirectLocation}}<div class="kv"><div class="k">Redirect</div><div class="mono">{{.RedirectLocation}} <span class="muted">(recorded, not followed)</span></div></div>{{end}}
{{else}}<p class="muted">No HTTP identity value recorded.</p>{{end}}
</div>

<div class="section">
<div class="microlabel">Timelines</div>
<h2>Current and closed timelines</h2>
{{if .Timelines}}
<p class="muted">Each timeline is one period a value was held. A Break marks two spans the drift engine may not compare, naming the leaf that moved; it is derived on read and never stored.</p>
{{range .Timelines}}
<div class="timeline">
<div class="microlabel">{{.Label}}</div>
{{if .Current}}
<div class="kv"><div class="k">Current</div><div>{{if .Current.IsGap}}<span class="badge">Gap</span>{{else if .Current.Details}}<details class="spanrecords"><summary><span class="badge">{{.Current.Value}}</span></summary>{{template "recordrows" .Current.Details}}</details>{{else}}<span class="badge">{{.Current.Value}}</span>{{end}} <span class="muted mono">since {{.Current.OpenedAt}}</span></div></div>
{{else}}
<div class="kv"><div class="k">Current</div><div class="muted">Closed — this timeline holds no current value.</div></div>
{{end}}
{{if .Breaks}}{{range .Breaks}}
<div class="notice">Break at {{.At}} — not comparable across it. Leaf that moved: <span class="mono">{{.MovedLeaves}}</span></div>
{{end}}{{end}}
{{if .Closed}}
<table class="closedspans">
<thead><tr><th>Value</th><th>Opened</th><th>Closed</th><th>Ground</th></tr></thead>
<tbody>
{{range .Closed}}<tr>
<td>{{if .IsGap}}<span class="muted">Gap</span>{{else if .Details}}<details class="spanrecords"><summary><span class="mono">{{.Value}}</span></summary>{{template "recordrows" .Details}}</details>{{else}}<span class="mono">{{.Value}}</span>{{end}}</td>
<td class="mono">{{.OpenedAt}}</td>
<td class="mono">{{.ClosedAt}}</td>
<td>{{if .Reason}}<span class="badge">{{.Reason}}</span>{{else}}<span class="muted">—</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
{{end}}
</div>
{{end}}
{{else}}
<p class="muted">No timeline has been folded yet. A Span opens when the hot Scan first exchanges with this endpoint; re-running it with a changed identity closes the open span and opens the next.</p>
{{end}}
</div>

<div class="section">
<div class="microlabel">Rules</div>
<h2>Rules over this subject</h2>
<p class="muted">Every rule whose predicate domain includes this subject renders here, each carrying its own versioned verdict. Wired up by ticket 22.</p>
</div>
{{end}}
</main>
{{template "foot" .}}{{end}}

{{define "subject-missing"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="microlabel">No such subject</div>
<h1 class="mono">{{.Name}}</h1>
<div class="section">
<p>No subject is keyed under that name. Nothing has ever measured it into the
estate — this is not a withdrawn subject, which would still be reachable here by
its own key.</p>
<p><a href="/inventory">Back to inventory</a></p>
</div>
</main>
{{template "foot" .}}{{end}}

`
