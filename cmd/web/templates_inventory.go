package main

import "html/template"

// Inventory screen — canonical `/inventory`. Folds today's inventory + subjects
// list (and the shared `recordrows` partial and the `subject-missing` page). The
// Name/Service/Endpoint detail views live elsewhere: a Name row opens the Asset
// detail (`/asset/{key}`, T1), and the Service/Endpoint drill-ins are the
// SubjectDetail templates in templates_subjectdetail.go (U1, #478). The body is
// the re-specced Inventory (SPEC-CHANGE #13 / U6, package v3.2.4): the open-span
// read grouped by subject, with client-side Kind / Gaps-only / subject scoping,
// column picker, and density — the exposure-era saved views / tag filters /
// bulk-actions bar were retired by that ruling as bound to nothing in the
// subject/facet/span model.
var _ = template.Must(tmpl.Parse(inventoryTemplates))

const inventoryTemplates = `
{{define "inventory"}}{{template "head" .}}
{{template "chrome" .}}
<style>
/* Inventory delta (#310, re-specced U6/#517) — navigable rows + client-side
   Kind/Gaps/subject scope, columns, and density controls, translated from
   design-system/examples/console/Inventory.jsx within the existing token
   vocabulary (restyling, not authoring — ADR-0109). */
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
/* Client-side scope controls (SPEC-CHANGE #13): a Gaps-only toggle and a
   subject filter, both scoping the already-rendered corpus — no server search. */
.invswitch { display: inline-flex; align-items: center; gap: 8px; cursor: pointer; }
.invswitch input { position: absolute; opacity: 0; width: 0; height: 0; }
.invswitch .switch { display: inline-block; width: 34px; height: 18px; border-radius: var(--r-full);
  position: relative; background: var(--sunken); border: 1px solid var(--hairline); transition: background 120ms; }
.invswitch input:checked + .switch { background: var(--accent); border-color: var(--accent); }
.invswitch input:focus-visible + .switch { outline: 2px solid var(--accent); outline-offset: 2px; }
.invswitch .knob { position: absolute; top: 1px; left: 1px; width: 14px; height: 14px; border-radius: var(--r-full);
  background: var(--surface); transition: left 120ms; }
.invswitch input:checked + .switch .knob { left: 17px; background: var(--on-accent); }
.invfilter { font-family: var(--mono); font-size: 12px; padding: 5px 10px; border: 1px solid var(--hairline);
  border-radius: var(--r-md); background: var(--surface); color: var(--ink); width: 210px; }
.invfilter::placeholder { color: var(--muted); }
tr[data-invsub][hidden], .section[data-kind][hidden] { display: none; }
</style>
<main>
<div style="display:flex;align-items:flex-start;gap:16px;margin-bottom:var(--space-4)">
<div>
<div class="microlabel">Observed · inventory</div>
<h1 style="margin-bottom:4px">Inventory</h1>
<span class="muted">Everything you expose, watched for drift — the actual values behind the verdicts.</span>
</div>
<div style="margin-left:auto;display:flex;gap:8px;align-items:center">
{{if .HasData}}<a class="btn secondary" href="/inventory/export" style="text-decoration:none" title="Download the folded inventory as CSV">Export CSV</a>
{{else}}<button class="btn secondary" disabled title="Nothing to export until a value is folded">Export CSV</button>{{end}}
<a class="btn" href="/scope" style="text-decoration:none">Add seed</a>
</div>
</div>

<p class="muted" style="max-width:82ch">What your estate holds right now — the addresses a name
resolves to, the records it carries, the certificate a service presents, the identity an
endpoint returns. Each row is a subject; open it for the asset's full record, or expand a value
to its individual records. A withdrawn subject holds no current span and so is not here. As on
the change views there is no total: your estate's completeness is yours alone to state.</p>

<div style="display:flex;gap:14px;align-items:center;flex-wrap:wrap;margin-bottom:var(--space-4)">
<span class="seg" role="group" aria-label="Kind" data-invkind>
<button type="button" data-kind="all" aria-pressed="true">All subjects</button>
{{range .Groups}}<button type="button" data-kind="{{.Kind}}" aria-pressed="false">{{.Label}}</button>{{end}}
</span>
<label class="invswitch"><input type="checkbox" data-invgaps aria-label="Gaps only"><span class="switch"><span class="knob"></span></span><span class="microlabel">Gaps only</span></label>
<input class="invfilter" type="search" data-invfilter placeholder="Filter subjects&#8230;" aria-label="Filter subjects">
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
<div class="section" id="{{.Kind}}" data-kind="{{.Kind}}">
<div class="microlabel" style="margin-bottom:var(--space-3)">{{.Label}}</div>
<table class="vg-table">
<thead><tr>
<th>Subject</th>
<th data-col="type" style="width:110px">Type</th>
<th data-col="holds">Holds</th>
<th data-col="since" style="width:1%;white-space:nowrap;text-align:right">Since</th>
</tr></thead>
<tbody>
{{range .Subjects}}<tr data-invsub data-key="{{.Key}}" data-has-gap="{{if .HasGap}}1{{else}}0{{end}}"{{if .Link}} class="invrow" tabindex="0" data-href="{{.Link}}" role="link" aria-label="Open {{.Key}}"{{end}}>
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
<div class="emptystate" data-invnomatch hidden>
<h2>No subjects match this scope</h2>
<p>The corpus is unaffected — only this view is scoped. <button type="button" class="btn secondary" data-invclear>Clear filters</button></p>
</div>
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
  /* Client-side scope (SPEC-CHANGE #13, package v3.2.4): the Kind segmented
     control, the Gaps-only switch, and the subject filter narrow the
     already-rendered corpus — nothing is fetched. Hidden rows and sections stay
     in the DOM, so clearing the scope restores the full read instantly, and the
     "no subjects match" state stands in when the scope empties the view. */
  var kindBtns = Array.prototype.slice.call(container.querySelectorAll("[data-invkind] button"));
  var gapsBox = container.querySelector("[data-invgaps]");
  var filterInput = container.querySelector("[data-invfilter]");
  var noMatch = container.querySelector("[data-invnomatch]");
  var sections = Array.prototype.slice.call(container.querySelectorAll(".section[data-kind]"));
  var scope = { kind: "all", gaps: false, q: "" };
  function applyScope() {
    var anyVisible = false;
    sections.forEach(function (sec) {
      var kindOk = scope.kind === "all" || sec.getAttribute("data-kind") === scope.kind;
      var shown = 0;
      Array.prototype.slice.call(sec.querySelectorAll("tr[data-invsub]")).forEach(function (tr) {
        var ok = kindOk &&
          (!scope.gaps || tr.getAttribute("data-has-gap") === "1") &&
          (!scope.q || (tr.getAttribute("data-key") || "").toLowerCase().indexOf(scope.q) !== -1);
        if (ok) { tr.removeAttribute("hidden"); shown++; } else { tr.setAttribute("hidden", ""); }
      });
      if (shown > 0) { sec.removeAttribute("hidden"); anyVisible = true; } else { sec.setAttribute("hidden", ""); }
    });
    if (noMatch) { if (anyVisible) noMatch.setAttribute("hidden", ""); else noMatch.removeAttribute("hidden"); }
  }
  kindBtns.forEach(function (btn) {
    btn.addEventListener("click", function () {
      scope.kind = btn.getAttribute("data-kind");
      kindBtns.forEach(function (b) { b.setAttribute("aria-pressed", b === btn ? "true" : "false"); });
      applyScope();
    });
  });
  if (gapsBox) gapsBox.addEventListener("change", function () { scope.gaps = gapsBox.checked; applyScope(); });
  if (filterInput) filterInput.addEventListener("input", function () { scope.q = filterInput.value.trim().toLowerCase(); applyScope(); });
  var clearBtn = container.querySelector("[data-invclear]");
  if (clearBtn) clearBtn.addEventListener("click", function () {
    scope = { kind: "all", gaps: false, q: "" };
    kindBtns.forEach(function (b) { b.setAttribute("aria-pressed", b.getAttribute("data-kind") === "all" ? "true" : "false"); });
    if (gapsBox) gapsBox.checked = false;
    if (filterInput) filterInput.value = "";
    applyScope();
  });
})();
</script>
</main>
{{template "foot" .}}{{end}}

{{define "recordrows"}}<table class="records"><tbody>
{{range .}}<tr>{{if .Type}}<td class="rrtype"><span class="badge">{{.Type}}</span></td>{{else}}<td class="rrtype"></td>{{end}}<td class="mono">{{.Data}}</td></tr>{{end}}
</tbody></table>{{end}}

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
