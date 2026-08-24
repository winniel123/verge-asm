package main

import "html/template"

// Exposure — canonical `/exposure` (#300, T5). Ported from
// design-system/examples/console/Exposure.jsx: a header, then either the
// both-legs table (summary stat band, the internal/internet per-leg table, and the
// "one leg never concludes" callout) or — when the install has no internet vantage
// — the WITHHELD state, a first-class render that names its cause.
//
// The example's components are translated to template-local CSS within the existing
// token vocabulary (restyling, not authoring — ADR-0109); no design-system
// component is authored here. The leg chip (.exp) mirrors the asset-detail chip so
// a reached leg reads `exposed`, a not-reached leg `not reached`, and an unmeasured
// or Gap leg `unverified` — never a firewalled/exposed claim for a leg not
// concluded. Both light and dark render through the shared tokens.
//
// The stat band renders the spec's vs-last-batch delta (P0.2 #443, P2.6 #452,
// ADR-0116) as the DeltaChip.jsx pill on the exposed tile only — the one tile the
// spec chips (Exposure.jsx) — signed by the movement's meaning: a rise in exposure
// is `bad` (danger), a fall `good` (ok), no movement `neutral`. The chip renders
// only when HasDeltas holds (a previous batch exists); with none, the tile shows no
// chip — its honest no-delta state, never a fabricated +0. Firewalled and Not
// reached carry no delta, matching the spec's band.
var _ = template.Must(tmpl.Parse(exposureTemplates))

const exposureTemplates = `
{{define "expleg"}}<span class="exp {{if eq . "exposed"}}exposed{{else if eq . "firewalled"}}firewalled{{else if eq . "not-reached"}}notreached{{else}}unverified{{end}}">{{if eq . "not-reached"}}not reached{{else}}{{.}}{{end}}</span>{{end}}

{{define "exposure"}}{{template "head" .}}
{{template "chrome" .}}
<style>
.exp{display:inline-flex;align-items:center;height:20px;padding:0 9px;border-radius:var(--r-sm);font-family:var(--mono);font-size:10.5px;font-weight:600;letter-spacing:0.04em;white-space:nowrap;border:1px solid transparent}
.exp.exposed{background:var(--danger-soft);border-color:var(--danger-border);color:var(--danger)}
.exp.firewalled{background:var(--ok-soft);border-color:var(--ok-border);color:var(--ok)}
.exp.notreached{background:var(--sunken);border-color:var(--hairline);color:var(--muted)}
.exp.unverified{background:transparent;border:1px dashed var(--border-strong);color:var(--muted)}
.excard{background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm)}
.exstat{display:flex;flex-direction:column;gap:6px;padding:var(--space-5)}
.exstat .num{font-family:var(--mono);font-size:28px;font-weight:600;color:var(--ink);line-height:1.1}
.exnum{display:flex;align-items:baseline;gap:8px}
.exdelta{display:inline-flex;align-items:center;gap:4px;height:18px;padding:0 7px;border-radius:999px;font-family:var(--mono);font-size:11px;font-weight:600;line-height:1;white-space:nowrap;border:1px solid transparent;transform:translateY(-2px)}
.exdelta.bad{background:var(--danger-soft);border-color:var(--danger-border);color:var(--danger)}
.exdelta.good{background:var(--ok-soft);border-color:var(--ok-border);color:var(--ok)}
.exdelta.neutral{background:var(--sunken);border-color:var(--hairline);color:var(--muted)}
.exdelta svg.down{transform:rotate(180deg)}
.excard-head{display:flex;flex-direction:column;gap:3px;padding:var(--space-5) var(--space-5) 0}
.extable{padding:var(--space-4) var(--space-5) var(--space-2)}
.excallout{border:1px solid var(--hairline);background:var(--surface);box-shadow:var(--shadow-xs);border-radius:var(--r-lg);padding:var(--space-4) var(--space-5)}
.excallout .title{font-weight:600;color:var(--ink)}
.excallout p{margin:6px 0 0;color:var(--body);max-width:80ch}
.exwithheld{display:flex;flex-direction:column;align-items:center;gap:var(--space-4);text-align:center;padding:56px 24px}
.exwithheld svg{color:var(--muted)}
.exwithheld p{margin:0;max-width:60ch;color:var(--muted)}
.exwithheld h2{margin:0;color:var(--ink)}
</style>
<main style="display:flex;flex-direction:column;gap:20px">

<header style="display:flex;align-items:center;gap:16px">
<div style="display:flex;flex-direction:column;gap:2px">
<h1 style="margin:0;font-size:21px;letter-spacing:-0.015em;color:var(--ink)">Exposure</h1>
<span class="muted" style="font-size:12.5px;white-space:nowrap">Composed from two reach legs &#8212; internal and internet. States, never a score.</span>
</div>
</header>

{{if .Withheld}}
<section class="excard">
<div class="exwithheld">
<svg viewBox="0 0 24 24" width="28" height="28" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M9.88 9.88a3 3 0 1 0 4.24 4.24"/><path d="M10.73 5.08A10.4 10.4 0 0 1 12 5c7 0 10 7 10 7a13.2 13.2 0 0 1-1.67 2.68"/><path d="M6.61 6.61A13.5 13.5 0 0 0 2 12s3 7 10 7a9.7 9.7 0 0 0 5.39-1.61"/><line x1="2" y1="2" x2="22" y2="22"/></svg>
<h2>Exposure withheld.</h2>
<p>No internet vantage exists. Internal reachability is complete, but exposure claims need the outside leg &#8212; Verge degrades to internal-only rather than report firewalled or exposed for something it did not look at.</p>
<a class="btn" href="/scope" style="text-decoration:none">Provision a prober</a>
</div>
</section>
{{else}}

<div style="display:grid;grid-template-columns:repeat(3,1fr);gap:24px">
<section class="excard exstat"><span class="microlabel">Exposed to internet</span><span class="exnum"><span class="num">{{.Exposed}}</span>{{if .HasDeltas}}{{$c := .ExposedDelta.Change}}<span class="exdelta {{if gt $c 0}}bad{{else if lt $c 0}}good{{else}}neutral{{end}}">{{if ne $c 0}}<svg class="{{if lt $c 0}}down{{end}}" viewBox="0 0 10 10" width="8" height="8" aria-hidden="true"><path d="M5 8.5V1.5M1.8 4.7L5 1.5l3.2 3.2" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/></svg>{{end}}{{signDelta $c}}</span>{{end}}</span><span class="muted" style="font-size:11.5px">services &#183; both legs concluded</span></section>
<section class="excard exstat"><span class="microlabel">Firewalled</span><span class="num">{{.Firewalled}}</span><span class="muted" style="font-size:11.5px">reachable inside, filtered outside</span></section>
<section class="excard exstat"><span class="microlabel">Not reached</span><span class="num">{{.NotReached}}</span><span class="muted" style="font-size:11.5px">no leg concluded this batch</span></section>
</div>

<section class="excard">
<div class="excard-head"><span class="microlabel">Both legs</span><h2 style="margin:0;font-size:15px">Service exposure</h2></div>
<div class="extable">
{{if .Rows}}
<table>
<thead><tr><th>Asset</th><th style="width:150px">Service</th><th style="width:140px">Internal leg</th><th style="width:140px">Internet leg</th><th style="text-align:right;width:70px">Since</th></tr></thead>
<tbody>
{{range .Rows}}<tr>
<td class="mono">{{.Asset}}</td>
<td class="mono">{{.Svc}}</td>
<td>{{template "expleg" .Internal}}</td>
<td>{{template "expleg" .Internet}}</td>
<td class="mono muted" style="text-align:right">{{.Since}}</td>
</tr>{{end}}
</tbody>
</table>
{{else}}
<div class="emptystate"><h2>No service exposure measured yet</h2><p>No Service holds a current reachability span from either vantage class. Trigger a scan on Scope; each Service a vantage reaches appears here with both its legs.</p></div>
{{end}}
</div>
</section>

<div class="excallout"><span class="title">One leg never concludes</span><p>A single vantage can say reached or not reached from where it stands. Exposed and firewalled exist only where the internal and internet legs are both constructible in the same derivation.</p></div>

{{end}}
</main>
{{template "foot" .}}{{end}}
`
