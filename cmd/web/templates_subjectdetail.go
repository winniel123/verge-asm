package main

import "html/template"

// Subject detail — the Service and Endpoint drill-ins reached from Inventory rows
// (U1, #478), ported from design-system/examples/console/SubjectDetail.jsx
// (screenshots 22 service, 23 endpoint, 24 withdrawn). A Name subject opens the
// Asset detail instead (subjectPage delegates to assetPage); this screen covers the
// other two kinds. The example's components are translated to template-local CSS
// within the existing token vocabulary (restyling, not authoring — ADR-0109); no
// design-system component is authored here.
//
// Shared skeleton (both kinds): breadcrumb → header (key, kind tag,
// ExposureBadge/WithdrawnMark, seen/scope-since) with Rescan + a more-actions menu
// → withdrawn banner (when withdrawn) → a two-column grid: citation chain card,
// current-facet card (Reachability for a Service, HTTP identity for an Endpoint),
// timelines card (current + closed, with a Break banner where the drift engine
// derived one), and the rules-over-subject table down the left; provenance, the
// Service's signals-here, and a copy-key affordance down the rail.
//
// Every value is a real read of the subject's own shape (subjects.go): the
// reachability verdict / HTTP identity, the citation chain back to a Seed, the Span
// timelines the hot Scan folds, the rules whose predicate domain includes this
// subject (each with its versioned verdict and its real per-rule SeverityBadge,
// internal/signal SeverityFor), and the fired census filtered to this key. No datum
// is fabricated: a provenance fact with no honest source is omitted rather than
// invented, and nothing fingerprints the technology a Service or Endpoint runs.
var _ = template.Must(tmpl.Parse(subjectDetailTemplates))

const subjectDetailTemplates = `
{{define "subjectstyle"}}<style>
.scard{background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);display:flex;flex-direction:column;overflow:hidden}
.scard>header{padding:var(--space-5) var(--space-5) 0}
.scard>.scard-body{padding:var(--space-5)}
.scard>.scard-body.tight{padding:12px}
.scard h2{font-size:15px;margin:4px 0 0}
.exp{display:inline-flex;align-items:center;height:20px;padding:0 9px;border-radius:var(--r-sm);font-family:var(--mono);font-size:10.5px;font-weight:600;letter-spacing:0.04em;white-space:nowrap;border:1px solid transparent}
.exp.exposed{background:var(--danger-soft);border-color:var(--danger-border);color:var(--danger)}
.exp.firewalled{background:var(--ok-soft);border-color:var(--ok-border);color:var(--ok)}
.exp.notreached{background:var(--sunken);border-color:var(--hairline);color:var(--muted)}
.exp.unverified{background:transparent;border:1px dashed var(--border-strong);color:var(--muted)}
.skv{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px 20px;padding:var(--space-4);background:var(--sunken);border-radius:var(--r-md)}
.skv .k{font-family:var(--mono);font-size:11px;font-weight:500;text-transform:uppercase;letter-spacing:0.07em;color:var(--muted)}
.skv .v{font-family:var(--mono);font-size:12.5px;color:var(--body);overflow-wrap:anywhere}
.sighere{display:flex;align-items:center;gap:10px;padding:10px 8px;border-radius:var(--r-sm);text-decoration:none}
.sighere:hover{background:var(--sunken);text-decoration:none}
.sighere .txt{flex:1;min-width:0;display:flex;flex-direction:column;gap:2px}
.actions{position:relative}
.actions>summary{list-style:none;cursor:pointer}
.actions>summary::-webkit-details-marker{display:none}
.actionsmenu{position:absolute;right:0;top:calc(100% + 6px);min-width:180px;background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-md);box-shadow:var(--shadow-md);padding:6px;z-index:35;display:flex;flex-direction:column;gap:2px}
.actionsmenu a{display:block;padding:6px 8px;border-radius:var(--r-sm);color:var(--body);text-decoration:none;font-size:12.5px}
.actionsmenu a:hover{background:var(--sunken);text-decoration:none}
.actionsmenu a.danger{color:var(--danger)}
</style>{{end}}

{{define "subjectbreadcrumb"}}<nav aria-label="Breadcrumb" class="microlabel" style="display:flex;align-items:center;gap:8px">
<a href="/inventory" style="color:var(--muted);text-decoration:none">Inventory</a>
<span aria-hidden="true" style="color:var(--muted)">/</span>
<span aria-current="page" style="color:var(--body)">{{.}}</span>
</nav>{{end}}

{{define "subjecttimelines"}}<section class="scard">
<header><div class="microlabel">Timelines</div><h2>Current and closed timelines</h2></header>
<div class="scard-body">
{{if .}}
<p class="muted">Each timeline is one period a value was held. A Break marks two spans the drift engine may not compare, naming the leaf that moved; it is derived on read and never stored.</p>
{{range .}}
<div class="timeline">
<div class="microlabel">{{.Label}}</div>
{{if .Current}}
<div class="kv"><div class="k">Current</div><div>{{if .Current.IsGap}}<span class="badge">Gap</span>{{else if .Current.Details}}<details class="spanrecords"><summary><span class="badge">{{.Current.Value}}</span></summary>{{template "recordrows" .Current.Details}}</details>{{else}}<span class="badge">{{.Current.Value}}</span>{{end}} <span class="muted mono">since {{.Current.OpenedAt}}</span></div></div>
{{else}}
<div class="kv"><div class="k">Current</div><div class="muted">Closed — this timeline holds no current value.</div></div>
{{end}}
{{if .Breaks}}{{range .Breaks}}
<div class="notice">Break at {{.At}} — not comparable across it. Leaf that moved: <span class="mono">{{.MovedLeaves}}</span>. Derived on read, never stored.</div>
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
<p class="muted">No timeline has been folded yet. A Span opens when the hot Scan first measures a value; re-running it with a changed value closes the open span and opens the next.</p>
{{end}}
</div>
</section>{{end}}

{{define "subjectrules"}}<section class="scard">
<header><div class="microlabel">Rules</div><h2>Rules over this subject</h2></header>
<div class="scard-body">
{{if .}}
<table class="vg-table">
<thead><tr><th>Rule</th><th style="width:90px">Version</th><th style="width:130px">Verdict</th></tr></thead>
<tbody>
{{range .}}<tr>
<td><span style="display:inline-flex;align-items:center;gap:8px"><span class="sev sev-{{.Severity}}">{{if ne .Severity "critical"}}<span class="sev-dot"></span>{{end}}{{.Severity}}</span><span class="mono">{{.Rule}}</span></span></td>
<td class="mono">{{.Version}}</td>
<td>{{if .Fired}}<span class="badge">fired</span>{{else}}<span class="muted">did not fire</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
{{else}}
<p class="muted">No rule's predicate domain includes this subject yet. A rule renders here — with its own versioned verdict and severity — once the estate holds evidence of its kind for it to read.</p>
{{end}}
</div>
</section>{{end}}

{{define "subjectprovenance"}}<section class="scard">
<header><div class="microlabel">Provenance</div><h2>How it got here</h2></header>
<div class="scard-body">
{{if .}}
<div class="skv">{{range .}}<div style="display:flex;flex-direction:column;gap:3px;min-width:0"><span class="k">{{.K}}</span><span class="v">{{.V}}</span></div>{{end}}</div>
{{else}}
<p class="muted">No provenance to trace yet. Its "why is this here" chain fills in as a Scan measures it against a scope you declared.</p>
{{end}}
</div>
</section>{{end}}

{{define "service"}}{{template "head" .}}
{{template "chrome" .}}
{{template "subjectstyle" .}}
<main style="display:flex;flex-direction:column;gap:20px">
{{with .Service}}
{{template "subjectbreadcrumb" .Key}}

<header style="display:flex;align-items:flex-start;gap:16px;flex-wrap:wrap">
<div style="display:flex;flex-direction:column;gap:8px">
<h1 class="mono" style="margin:0;font-size:21px;letter-spacing:-0.01em;color:var(--ink)">{{.Key}}</h1>
<div style="display:flex;align-items:center;gap:8px;flex-wrap:wrap">
<span class="chip">service</span>
{{if .Withdrawn}}<span class="chip loss">withdrawn</span>{{else if .Exposure}}{{template "assetexposure" .Exposure}}{{end}}
{{if or .Seen .InScopeSince}}<span class="mono muted" style="font-size:12px">{{if .Seen}}{{if .Withdrawn}}last seen{{else}}seen{{end}} {{.Seen}}{{end}}{{if and .Seen .InScopeSince}} · {{end}}{{if .InScopeSince}}in scope since {{.InScopeSince}}{{end}}</span>{{end}}
</div>
</div>
<div style="margin-left:auto;display:flex;gap:8px;align-items:center">
{{if .Withdrawn}}<button class="btn secondary" disabled title="A withdrawn service names no current member">Rescan service</button>{{else}}<a class="btn secondary" href="/scans" style="text-decoration:none">Rescan service</a>{{end}}
<details class="actions"><summary class="btn secondary" role="button" aria-haspopup="menu" aria-label="More actions">More</summary>
<div class="actionsmenu" role="menu">
<a role="menuitem" href="/signals">Annotate</a>
<a role="menuitem" class="danger" href="/scope">Descope address</a>
</div></details>
</div>
</header>

{{if .Withdrawn}}
<div class="notice">This service's address has left the estate — no current resolution cites it and no Seed covers it. It names a population of no current member; its timelines are closed and it is reached by its own key.</div>
{{end}}

<div style="display:grid;grid-template-columns:minmax(0,1fr) 340px;gap:24px;align-items:start">

<div style="display:flex;flex-direction:column;gap:24px">

<section class="scard">
<header><div class="microlabel">Why is this here</div><h2>Citation chain</h2></header>
<div class="scard-body">
<p class="muted" style="max-width:62ch">A Service is an (address, port, transport) triple. Its membership is its address's membership restated — an address is in the estate exactly while a current resolution cites it or a Seed covers it — so following citations backwards always terminates at a Seed you declared.</p>
<ol class="chain">
{{range .Citation}}<li><div class="microlabel">{{.Label}}</div><div class="mono chainval">{{.Value}}</div>{{if .Detail}}<div class="muted">{{.Detail}}</div>{{end}}</li>{{end}}
</ol>
{{if not .CitationTerminated}}<p class="muted">The chain does not reach a declared Seed. For a service whose address a resolution cites, that is the address's name-scope Seed, one hop past the citing name; for one only a Seed covers, it is the address scope directly.</p>{{end}}
</div>
</section>

<section class="scard">
<header><div class="microlabel">Current · reachability</div><h2>Reachability</h2></header>
<div class="scard-body">
<div class="kv"><div class="k">Address</div><div class="mono">{{.Address}}</div></div>
<div class="kv"><div class="k">Port</div><div class="mono">{{.Port}}/{{.Transport}}</div></div>
{{if .ReachGap}}
<div class="kv"><div class="k">Verdict</div><div><span class="badge">Gap</span></div></div>
<div class="notice">{{.ReachGapReason}}. From this vantage we cannot tell a real origin service behind the edge from the edge answering for it, so the reach is undiscriminated — a Gap, not <span class="mono">reached</span>. Declare your origin IPs as an address scope to measure the real surface.</div>
{{else if .Withdrawn}}
<div class="kv"><div class="k">Verdict</div><div class="muted">—</div></div>
{{else if .Reach}}
<div class="kv"><div class="k">Verdict</div><div><span class="badge">{{.Reach}}</span></div></div>
{{if .Since}}<div class="kv"><div class="k">Since</div><div class="mono muted">{{.Since}}</div></div>{{end}}
{{else}}<p class="muted">No reachability value recorded.</p>{{end}}
</div>
</section>

{{template "subjecttimelines" .Timelines}}

{{template "subjectrules" .Rules}}

</div>

<div style="display:flex;flex-direction:column;gap:24px">

{{template "subjectprovenance" .Provenance}}

{{if and (not .Withdrawn) .Signals}}
<section class="scard">
<header><div class="microlabel">Open</div><h2>Signals here</h2></header>
<div class="scard-body tight">
{{range .Signals}}<a class="sighere" href="/signals">
<span class="sev sev-{{.Severity}}">{{if ne .Severity "critical"}}<span class="sev-dot"></span>{{end}}{{.Severity}}</span>
<span class="txt"><span style="font-weight:500;font-size:12.5px;color:var(--ink)">{{.Rule}}</span><span class="mono muted" style="font-size:11px">{{.Subject}} · raised</span></span>
<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.75" style="color:var(--muted);flex:none"><path d="M9 18l6-6-6-6"/></svg>
</a>{{end}}
</div>
</section>
{{end}}

<section class="scard">
<header><div class="microlabel">Service key</div><h2>Copy</h2></header>
<div class="scard-body"><span class="mono" style="overflow-wrap:anywhere">{{.Key}}</span></div>
</section>

</div>

</div>
{{end}}
</main>
{{template "foot" .}}{{end}}

{{define "endpoint"}}{{template "head" .}}
{{template "chrome" .}}
{{template "subjectstyle" .}}
<main style="display:flex;flex-direction:column;gap:20px">
{{with .Endpoint}}
{{template "subjectbreadcrumb" .Key}}

<header style="display:flex;align-items:flex-start;gap:16px;flex-wrap:wrap">
<div style="display:flex;flex-direction:column;gap:8px">
<h1 class="mono" style="margin:0;font-size:21px;letter-spacing:-0.01em;color:var(--ink)">{{if .Nameless}}<span class="muted">(nameless)</span> {{end}}{{.Key}}</h1>
<div style="display:flex;align-items:center;gap:8px;flex-wrap:wrap">
<span class="chip">endpoint</span>
{{if .Withdrawn}}<span class="chip loss">withdrawn</span>{{end}}
{{if or .Seen .InScopeSince}}<span class="mono muted" style="font-size:12px">{{if .Seen}}{{if .Withdrawn}}last seen{{else}}seen{{end}} {{.Seen}}{{end}}{{if and .Seen .InScopeSince}} · {{end}}{{if .InScopeSince}}in scope since {{.InScopeSince}}{{end}}</span>{{end}}
</div>
</div>
<div style="margin-left:auto;display:flex;gap:8px;align-items:center">
{{if .Withdrawn}}<button class="btn secondary" disabled title="A withdrawn endpoint names no current member">Rescan endpoint</button>{{else}}<a class="btn secondary" href="/scans" style="text-decoration:none">Rescan endpoint</a>{{end}}
<details class="actions"><summary class="btn secondary" role="button" aria-haspopup="menu" aria-label="More actions">More</summary>
<div class="actionsmenu" role="menu">
<a role="menuitem" href="/signals">Annotate</a>
<a role="menuitem" class="danger" href="/scope">Descope name</a>
</div></details>
</div>
</header>

{{if .Withdrawn}}
<div class="notice">This endpoint's service has left the estate — no current resolution cites its address and no Seed covers it. It names a population of no current member; its timelines are closed and it is reached by its own key. An endpoint closes when either leg — its Name or its Service — withdraws.</div>
{{end}}

<div style="display:grid;grid-template-columns:minmax(0,1fr) 340px;gap:24px;align-items:start">

<div style="display:flex;flex-direction:column;gap:24px">

<section class="scard">
<header><div class="microlabel">Why is this here</div><h2>Citation chain</h2></header>
<div class="scard-body">
<p class="muted" style="max-width:62ch">An Endpoint is a (Name, Service) pair — the only key under which HTTP identity is single-valued. Its membership is its Service's, restated, so following citations backwards runs from the Endpoint through its Name and Service legs down to the Seed you declared.</p>
<ol class="chain">
{{range .Citation}}<li><div class="microlabel">{{.Label}}</div><div class="mono chainval">{{.Value}}</div>{{if .Detail}}<div class="muted">{{.Detail}}</div>{{end}}</li>{{end}}
</ol>
{{if not .CitationTerminated}}<p class="muted">The chain does not reach a declared Seed. For an endpoint whose service address a resolution cites, that is the address's name-scope Seed, one hop past the citing name; for one only a Seed covers, it is the address scope directly.</p>{{end}}
</div>
</section>

<section class="scard">
<header><div class="microlabel">Current · http-identity</div><h2>HTTP identity</h2></header>
<div class="scard-body">
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
</section>

{{template "subjecttimelines" .Timelines}}

{{template "subjectrules" .Rules}}

</div>

<div style="display:flex;flex-direction:column;gap:24px">

{{template "subjectprovenance" .Provenance}}

<section class="scard">
<header><div class="microlabel">Endpoint key</div><h2>Copy</h2></header>
<div class="scard-body"><span class="mono" style="overflow-wrap:anywhere">{{.Key}}</span></div>
</section>

</div>

</div>
{{end}}
</main>
{{template "foot" .}}{{end}}
`
