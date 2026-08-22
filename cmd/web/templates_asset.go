package main

import "html/template"

// Asset detail — the per-asset drill-in reached from an Inventory row (the stable
// `/asset/{key}` route T15 links to). Ported verbatim from
// design-system/examples/console/AssetDetail.jsx (12-console.jpg): a breadcrumb +
// header, then a two-column grid — ports census, DNS records and the TLS
// certificate down the left; provenance, signals-here, the drift trail and a copy
// affordance down the right. The example's components are translated to
// template-local CSS within the existing token vocabulary (restyling, not
// authoring — ADR-0109); no design-system component is authored here.
//
// The "asset" is a Name subject. Real data is wired where a Name-scoped read
// exists (subjects.go): DNS records (resolution + dns-record), provenance (the
// citation chain), signals-here (the fired census filtered to this subject), the
// drift trail (the subject's Span timelines), and the ports census (the open
// Service reachability spans on the Name's addresses). Where no honest source
// exists — the TLS certificate's parsed identity (issuer/algorithm/validity) is
// not stored (signals.go) — the section renders the design-system empty-state
// rather than fabricate a value.
//
// Two domain holds against the mock: the census carries NO technology
// fingerprint (no product/version — a guardrail on drift-integrity grounds), and
// signals carry NO severity (the census "is deliberately not a severity ramp",
// signals.go / ADR-0024), so signals-here lists the honest firing rules without a
// fabricated SeverityBadge level. Change rides its own palette (the drift chips
// and the shared `changeglyph`), never the severity ramp; signals are withdrawn
// by the world, never "resolved".
var _ = template.Must(tmpl.Parse(assetTemplates))

const assetTemplates = `
{{define "assetexposure"}}<span class="exp {{if eq . "exposed"}}exposed{{else if eq . "firewalled"}}firewalled{{else if eq . "not-reached"}}notreached{{else}}unverified{{end}}">{{if eq . "not-reached"}}not reached{{else}}{{.}}{{end}}</span>{{end}}

{{define "asset"}}{{template "head" .}}
{{template "chrome" .}}
<style>
.acard{background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);display:flex;flex-direction:column;overflow:hidden}
.acard>header{padding:var(--space-5) var(--space-5) 0}
.acard>.acard-body{padding:var(--space-5)}
.acard h2{font-size:15px;margin:4px 0 0}
.exp{display:inline-flex;align-items:center;height:20px;padding:0 9px;border-radius:var(--r-sm);font-family:var(--mono);font-size:10.5px;font-weight:600;letter-spacing:0.04em;white-space:nowrap;border:1px solid transparent}
.exp.exposed{background:var(--danger-soft);border-color:var(--danger-border);color:var(--danger)}
.exp.firewalled{background:var(--ok-soft);border-color:var(--ok-border);color:var(--ok)}
.exp.notreached{background:var(--sunken);border-color:var(--hairline);color:var(--muted)}
.exp.unverified{background:transparent;border:1px dashed var(--border-strong);color:var(--muted)}
.assetkv{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px 20px;padding:var(--space-4);background:var(--sunken);border-radius:var(--r-md)}
.assetkv .k{font-family:var(--mono);font-size:11px;font-weight:500;text-transform:uppercase;letter-spacing:0.07em;color:var(--muted)}
.assetkv .v{font-family:var(--mono);font-size:12.5px;color:var(--body);overflow-wrap:anywhere}
.sighere{display:flex;align-items:center;gap:10px;padding:10px 8px;border-radius:var(--r-sm);text-decoration:none}
.sighere:hover{background:var(--sunken);text-decoration:none}
.sighere .txt{flex:1;min-width:0;display:flex;flex-direction:column;gap:2px}
.driftev{display:flex;flex-direction:column;gap:6px;padding:var(--space-3) 0;border-top:1px solid var(--hairline)}
.driftev:first-child{border-top:0}
</style>
<main style="display:flex;flex-direction:column;gap:20px">
{{with .Asset}}
<nav aria-label="Breadcrumb" class="microlabel" style="display:flex;align-items:center;gap:8px">
<a href="/inventory" style="color:var(--muted);text-decoration:none">Inventory</a>
<span aria-hidden="true" style="color:var(--muted)">/</span>
<span aria-current="page" style="color:var(--body)">{{.Key}}</span>
</nav>

<header style="display:flex;align-items:flex-start;gap:16px;flex-wrap:wrap">
<div style="display:flex;flex-direction:column;gap:8px">
<h1 class="mono" style="margin:0;font-size:21px;letter-spacing:-0.01em;color:var(--ink)">{{.Key}}</h1>
<div style="display:flex;align-items:center;gap:8px;flex-wrap:wrap">
<span class="chip">{{.Type}}</span>
{{if or .Seen .InScopeSince}}<span class="mono muted" style="font-size:12px">{{if .Seen}}seen {{.Seen}}{{end}}{{if and .Seen .InScopeSince}} · {{end}}{{if .InScopeSince}}in scope since {{.InScopeSince}}{{end}}</span>{{end}}
</div>
</div>
<div style="margin-left:auto;display:flex;gap:8px">
<a class="btn secondary" href="/scans" style="text-decoration:none">Rescan asset</a>
</div>
</header>

{{if .Withdrawn}}<div class="notice">This name is withdrawn — it names a population of no current member. Its timelines are closed. It is reached by its own key and never appears in the listing.</div>{{end}}

<div style="display:grid;grid-template-columns:minmax(0,1fr) 340px;gap:24px;align-items:start">

<div style="display:flex;flex-direction:column;gap:24px">

<section class="acard">
<header><div class="microlabel">Census</div><h2>Open ports</h2></header>
<div class="acard-body">
{{if .Ports}}
<table>
<thead><tr><th style="width:90px">Port</th><th>Service</th><th style="width:150px">Exposure</th><th style="text-align:right;width:110px">First seen</th></tr></thead>
<tbody>
{{range .Ports}}<tr>
<td class="mono">{{.Port}}</td>
<td class="mono">{{.Transport}}</td>
<td>{{template "assetexposure" .Exposure}}</td>
<td class="mono muted" style="text-align:right">{{.Since}}</td>
</tr>{{end}}
</tbody>
</table>
{{else}}
<div class="emptystate"><h2>No open port measured</h2><p>No Service holds an open reachability span on this asset's addresses yet. Let a scan reach for its ports; each one that answers appears here with its exposure.</p></div>
{{end}}
</div>
</section>

<section class="acard">
<header><div class="microlabel">Resolution</div><h2>DNS records</h2></header>
<div class="acard-body">
{{if .DNS}}
<table>
<thead><tr><th style="width:90px">Type</th><th>Value</th><th style="text-align:right;width:80px">Seen</th></tr></thead>
<tbody>
{{range .DNS}}<tr>
<td class="mono">{{.Type}}</td>
<td class="mono">{{.Value}}</td>
<td class="mono muted" style="text-align:right">{{if .Seen}}{{.Seen}}{{else}}—{{end}}</td>
</tr>{{end}}
</tbody>
</table>
{{else}}
<div class="emptystate"><h2>No records resolved</h2><p>This name holds no current resolution or dns-record value. A record appears here once the dns scan folds one for it.</p></div>
{{end}}
</div>
</section>

<section class="acard">
<header><div class="microlabel">Transport</div><h2>TLS certificate</h2></header>
<div class="acard-body">
<div class="emptystate"><h2>No certificate detail to show</h2><p>The certificate a service presents is watched for drift, but its parsed identity — issuer, algorithm, validity — is not stored yet, so no certificate detail can be shown here. It appears once the certificate leaf folds a parsed value.</p></div>
</div>
</section>

</div>

<div style="display:flex;flex-direction:column;gap:24px">

<section class="acard">
<header><div class="microlabel">Provenance</div><h2>How it got here</h2></header>
<div class="acard-body">
{{if .Provenance}}
<div class="assetkv">
{{range .Provenance}}<div style="display:flex;flex-direction:column;gap:3px;min-width:0">
<span class="k">{{.K}}</span>
<span class="v">{{.V}}</span>
</div>{{end}}
</div>
{{else}}
<div class="emptystate"><h2>No provenance to trace</h2><p>Nothing has cited this name into the estate yet. Its "why is this here" chain fills in as a scan measures it against a scope you declared.</p></div>
{{end}}
</div>
</section>

<section class="acard">
<header><div class="microlabel">Open</div><h2>Signals here</h2></header>
<div class="acard-body" style="padding:12px">
{{if .Signals}}
<div style="display:flex;flex-direction:column">
{{range .Signals}}<a class="sighere" href="/signals">
<span class="txt">
<span style="font-weight:500;font-size:12.5px;color:var(--ink)">{{.Rule}}</span>
<span class="mono muted" style="font-size:11px">{{.Subject}} · raised</span>
</span>
<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.75" style="color:var(--muted);flex:none"><path d="M9 18l6-6-6-6"/></svg>
</a>{{end}}
</div>
{{else}}
<div class="emptystate"><h2>No signal raised here</h2><p>No rule is firing on this asset right now. A signal is raised when the world drifts into a rule's predicate, and withdrawn when the world moves back — never resolved by hand.</p></div>
{{end}}
</div>
</section>

<section class="acard">
<header><div class="microlabel">History</div><h2>Drift trail</h2></header>
<div class="acard-body">
{{if .Drift}}
<div style="display:flex;flex-direction:column">
{{range .Drift}}
<div class="driftev">
<div style="display:flex;align-items:center;gap:8px;flex-wrap:wrap">
<span class="chip {{.Family}}">{{template "changeglyph" .Change}}{{.Change}}</span>
<span class="mono" style="font-size:12.5px;color:var(--ink)">{{.Subject}}</span>
<span class="mono muted" style="margin-left:auto">{{.Time}}</span>
</div>
<div class="muted">{{.Detail}}</div>
</div>
{{end}}
</div>
{{else}}
<div class="emptystate"><h2>No change to show yet</h2><p>This asset holds no folded span yet, so there is no transition to trace. Its drift trail fills in as a scan measures a value and a re-run moves it.</p></div>
{{end}}
</div>
</section>

<section class="acard">
<header><div class="microlabel">Address</div><h2>Copy</h2></header>
<div class="acard-body"><span class="mono" style="overflow-wrap:anywhere">{{.Key}}</span></div>
</section>

</div>

</div>
{{end}}
</main>
{{template "foot" .}}{{end}}
`
