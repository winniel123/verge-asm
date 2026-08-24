package main

import "html/template"

// Signals screen — canonical `/signals`. Ported from
// design-system/examples/console/Signals.jsx (+ SignalData.jsx) as a FLAT
// PER-INSTANCE table (PARITY-CHART P2.2, #448; ADR-0116 — the design is normative
// for look AND functionality). One row per currently-fired (rule, subject) pair,
// carrying the SeverityBadge, the signal, its asset, port, minted SIG-#### id and
// last-seen, with a severity filter, a text filter, sortable columns, pagination,
// Open/Annotated/Withdrawn tabs with counts, a row Drawer, the AnnotationControl
// operator dial, and the typed-name descope ConfirmDialog.
//
// The old per-rule census grouping has left the screen: severity is a real datum
// now (P0.1, #442), so the SeverityBadge / `.sev` ramp paints, and the single
// sortable per-instance table the census once rejected is what the spec renders.
// All state — tab, filters, sort, page, the Drawer and the dialog — is threaded
// through the query string, since the shell ships no client table/drawer/dialog
// machinery (T0 seam); every class is the shared pageCSS, and this file adds no
// second stylesheet.
//
// Tab data mapping (all real): Open is every currently-fired instance. Annotated
// is every operator acceptance whose subject is still a current member of its
// rule's population; Withdrawn is every acceptance whose subject has left it —
// orphan on read, withdrawn by the world, never resolved by an operator (ADR-0092).
// The descope confirm posts to the existing `POST /exclusions` (kind=subtree) — the
// real "remove a name and its subjects from scope, recorded on Scope" act — so no
// route is added here.
var _ = template.Must(tmpl.Parse(signalsTemplates))

const signalsTemplates = `
{{define "withdrawnmark"}}<span class="mono" style="display:inline-flex;align-items:center;gap:4px;height:18px;padding:0 6px;border-radius:var(--r-sm);border:1px dashed var(--drift-loss-border);color:var(--drift-loss-fg);font-size:10px;font-weight:600;letter-spacing:0.04em;white-space:nowrap"><svg viewBox="0 0 10 10" width="9" height="9" style="flex:none" aria-hidden="true"><path d="M1.6 5h6.8" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"></path></svg>withdrawn</span>{{end}}

{{define "sevbadge"}}<span class="sev sev-{{.Severity}}">{{if ne .Severity "critical"}}<span class="sev-dot"></span>{{end}}{{.SevLabel}}</span>{{end}}

{{define "signals"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div style="display:flex;align-items:flex-start;gap:16px;margin-bottom:var(--space-5)">
<div>
<h1 style="margin-bottom:4px">Signals</h1>
<span class="muted">Raised when your attack surface drifts. Severity is Critical &#8594; Info.</span>
</div>
<div style="margin-left:auto;display:flex;gap:8px;align-items:center">
{{if .IsAdmin}}<a class="btn secondary" href="{{.DescopeHref}}">Descope seed</a>{{end}}
{{if .HasExport}}<a class="btn secondary" href="{{.ExportHref}}" style="text-decoration:none" title="Export the current tab's filtered rows as CSV"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="vertical-align:-2px;margin-right:6px" aria-hidden="true"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>Export CSV</a>
{{else}}<button class="btn secondary" disabled title="Nothing to export from this tab">Export CSV</button>{{end}}
</div>
</div>

{{if .AnnoError}}<div class="error" style="margin-bottom:var(--space-4)">{{.AnnoError}}</div>{{end}}

<div class="tabs">
<a class="tab{{if eq .Tab "open"}} active{{end}}" href="/signals?tab=open">Open<span class="mono" style="margin-left:6px;font-size:10.5px;color:var(--muted)">{{.OpenCount}}</span></a>
<a class="tab{{if eq .Tab "annotated"}} active{{end}}" href="/signals?tab=annotated">Annotated<span class="mono" style="margin-left:6px;font-size:10.5px;color:var(--muted)">{{.AnnotatedCount}}</span></a>
<a class="tab{{if eq .Tab "withdrawn"}} active{{end}}" href="/signals?tab=withdrawn">Withdrawn<span class="mono" style="margin-left:6px;font-size:10.5px;color:var(--muted)">{{.WithdrawnCount}}</span></a>
</div>

<form method="get" action="/signals" style="display:flex;gap:12px;align-items:flex-end;margin-bottom:var(--space-5)">
<input type="hidden" name="tab" value="{{.Tab}}">
<input type="hidden" name="sort" value="{{.Sort.Key}}">
<input type="hidden" name="dir" value="{{.Sort.Dir}}">
<label style="position:relative;display:inline-block;margin:0">
<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="position:absolute;left:10px;top:50%;transform:translateY(-50%);opacity:0.5;pointer-events:none" aria-hidden="true"><circle cx="11" cy="11" r="8"></circle><path d="m21 21-4.3-4.3"></path></svg>
<input class="mono" type="search" name="q" value="{{.Q}}" placeholder="Search signals, assets, ids" aria-label="Search signals" style="width:320px;padding-left:30px">
</label>
<select name="sev" onchange="this.form.submit()" aria-label="Severity filter" style="width:170px">
{{range .SevOptions}}<option{{if eq . $.Sev}} selected{{end}}>{{.}}</option>{{end}}
</select>
<span class="muted mono" style="margin-left:auto;font-size:12px">{{.Shown}} of {{.Total}} shown</span>
</form>

{{if .Rows}}
<table class="vg-table">
<thead><tr>
<th style="width:104px"><a href="{{.Sort.SevHref}}" style="color:inherit;text-decoration:none;display:inline-flex;align-items:center">Severity{{.Sort.SevArrow}}</a></th>
<th>Signal</th>
<th><a href="{{.Sort.AssetHref}}" style="color:inherit;text-decoration:none;display:inline-flex;align-items:center">Asset{{.Sort.AssetArrow}}</a></th>
<th style="width:76px">Port</th>
<th style="width:112px"><a href="{{.Sort.IDHref}}" style="color:inherit;text-decoration:none;display:inline-flex;align-items:center">Id{{.Sort.IDArrow}}</a></th>
<th style="width:70px;text-align:right"><a href="{{.Sort.SeenHref}}" style="color:inherit;text-decoration:none;display:inline-flex;align-items:center;justify-content:flex-end">Seen{{.Sort.SeenArrow}}</a></th>
<th style="width:44px"></th>
</tr></thead>
<tbody>
{{range .Rows}}
<tr{{if and $.SelKey (eq .ViewKey $.SelKey)}} class="row-selected"{{end}}>
<td>{{template "sevbadge" .}}</td>
<td><a href="{{$.ViewPrefix}}{{.ViewKey}}" style="color:var(--ink);font-weight:500;text-decoration:none">{{.Title}}</a>{{if .Withdrawn}} {{template "withdrawnmark"}}{{end}}</td>
<td class="mono">{{.Asset}}</td>
<td class="mono">{{if .Port}}{{.Port}}{{else}}<span class="muted">&#8212;</span>{{end}}</td>
<td class="mono">{{.SigID}}</td>
<td class="mono" style="text-align:right"><span title="{{.Last}}">{{if .Seen}}{{.Seen}}{{else}}<span class="muted">&#8212;</span>{{end}}</span></td>
<td style="text-align:right"><a href="{{$.ViewPrefix}}{{.ViewKey}}" title="Open signal" aria-label="Open signal" style="color:var(--muted);display:inline-flex"><svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><circle cx="5" cy="12" r="1.6"></circle><circle cx="12" cy="12" r="1.6"></circle><circle cx="19" cy="12" r="1.6"></circle></svg></a></td>
</tr>
{{end}}
</tbody>
</table>
{{if .ShowPagination}}
<div style="display:flex;justify-content:flex-end;align-items:center;gap:4px;margin-top:var(--space-4)">
<span class="muted mono" style="margin-right:10px;font-size:11.5px">{{.PageInfo}}</span>
{{if .PrevDisabled}}<span style="display:inline-flex;align-items:center;justify-content:center;min-width:28px;height:28px;border-radius:8px;color:var(--muted);opacity:0.45"><svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true"><path d="M10 3.5L5.5 8 10 12.5" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"></path></svg></span>{{else}}<a href="{{.PrevHref}}" aria-label="Previous page" style="display:inline-flex;align-items:center;justify-content:center;min-width:28px;height:28px;border-radius:8px;color:var(--muted);text-decoration:none"><svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true"><path d="M10 3.5L5.5 8 10 12.5" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"></path></svg></a>{{end}}
{{range .Pages}}{{if .Ellipsis}}<span class="mono" style="display:inline-flex;align-items:center;justify-content:center;min-width:28px;height:28px;color:var(--muted);font-size:12px">&#8230;</span>{{else}}<a href="{{.Href}}" aria-label="Page {{.Num}}" class="mono" style="display:inline-flex;align-items:center;justify-content:center;min-width:28px;height:28px;padding:0 6px;border-radius:8px;font-size:12px;text-decoration:none;{{if .Active}}border:1px solid var(--accent);background:var(--accent-soft);color:var(--link);font-weight:600{{else}}border:1px solid transparent;color:var(--muted){{end}}">{{.Num}}</a>{{end}}{{end}}
{{if .NextDisabled}}<span style="display:inline-flex;align-items:center;justify-content:center;min-width:28px;height:28px;border-radius:8px;color:var(--muted);opacity:0.45"><svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true"><path d="M6 3.5L10.5 8 6 12.5" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"></path></svg></span>{{else}}<a href="{{.NextHref}}" aria-label="Next page" style="display:inline-flex;align-items:center;justify-content:center;min-width:28px;height:28px;border-radius:8px;color:var(--muted);text-decoration:none"><svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true"><path d="M6 3.5L10.5 8 6 12.5" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"></path></svg></a>{{end}}
</div>
{{end}}
{{else if .HasAny}}
<div class="emptystate">
<h2>No signals match your filters</h2>
<p>Clear the search or severity filter to see {{if eq .Tab "annotated"}}annotated{{else if eq .Tab "withdrawn"}}withdrawn{{else}}open{{end}} signals.</p>
<p style="margin-top:var(--space-4)"><a class="btn secondary" href="{{.ClearHref}}">Clear filters</a></p>
</div>
{{else if eq .Tab "annotated"}}
<div class="emptystate">
<h2>No annotation is declared</h2>
<p>A fired signal you accept as a known risk is declared here, keyed on its subject. Open a signal and record the reason &#8212; it is all an annotation carries.</p>
</div>
{{else if eq .Tab "withdrawn"}}
<div class="emptystate">
<h2>No signal has withdrawn</h2>
<p>When a subject you annotated leaves its rule's population, its acceptance surfaces here as withdrawn &#8212; derived on read, no operator act. Nothing has withdrawn right now.</p>
</div>
{{else}}
<div class="emptystate">
<h2>No open signals</h2>
<p>Nothing in your estate is firing a rule right now. Signals appear here the moment your attack surface drifts.</p>
</div>
{{end}}
</main>

{{if .Drawer}}
<a class="scrim" href="{{.CloseHref}}" aria-label="Close detail"></a>
<aside class="drawer-panel" role="dialog" aria-label="Signal detail">
<div class="microlabel">{{if .Drawer.Withdrawn}}Withdrawn{{else if .Drawer.Annotated}}Annotated{{else}}Open{{end}} &#183; signal detail</div>
<h1 style="margin:6px 0 2px;font-size:16px">{{.Drawer.Title}}</h1>
<div class="muted mono" style="font-size:12px;margin-bottom:var(--space-4)">{{.Drawer.SigID}}{{if .Drawer.Seen}} &#183; seen {{.Drawer.Seen}}{{end}}</div>

<div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap;margin-bottom:var(--space-4)">
{{template "sevbadge" .Drawer}}
{{if .Drawer.Withdrawn}}{{template "withdrawnmark"}}{{end}}
</div>

<table class="invfacets" style="margin-bottom:var(--space-5)">
<tr><td class="invfacet muted">Asset</td><td class="mono">{{.Drawer.Asset}}</td></tr>
{{if .Drawer.IP}}<tr><td class="invfacet muted">IP</td><td class="mono">{{.Drawer.IP}}</td></tr>{{end}}
<tr><td class="invfacet muted">Rule</td><td class="mono">{{.Drawer.Signal}}</td></tr>
{{if .Drawer.Port}}<tr><td class="invfacet muted">Port</td><td class="mono">{{.Drawer.Port}}</td></tr>{{end}}
{{if .Drawer.First}}<tr><td class="invfacet muted">First seen</td><td class="mono">{{.Drawer.First}}</td></tr>{{end}}
{{if .Drawer.Last}}<tr><td class="invfacet muted">Last seen</td><td class="mono">{{.Drawer.Last}}</td></tr>{{end}}
</table>

<div style="margin-bottom:var(--space-5)">
{{if .Drawer.Annotated}}
<div style="display:flex;flex-direction:column;gap:8px;padding:14px;background:var(--sunken);border-radius:var(--r-md)">
<div style="display:flex;align-items:center;gap:10px">
<span class="mono" style="display:inline-flex;align-items:center;gap:6px;height:20px;padding:0 8px;border-radius:var(--r-sm);background:var(--surface);border:1px solid var(--hairline);color:var(--muted);font-size:10.5px;font-weight:600;letter-spacing:0.05em"><span style="width:6px;height:6px;border-radius:999px;background:var(--muted)"></span>accepted risk</span>
{{if .IsAdmin}}<form method="post" action="/annotations/withdraw" style="margin:0 0 0 auto"><input type="hidden" name="id" value="{{.Drawer.AnnoID}}"><button class="btn ghost" type="submit">Remove annotation</button></form>{{end}}
</div>
<p style="margin:0;line-height:1.55;color:var(--body)">{{.Drawer.AnnoReason}}</p>
</div>
{{else if .IsAdmin}}
<form method="post" action="/annotations" style="display:flex;flex-direction:column;gap:8px">
<span class="microlabel">Annotation</span>
<input type="hidden" name="subject" value="{{.Drawer.Asset}}">
<input type="hidden" name="signal" value="{{.Drawer.Signal}}">
<textarea name="reason" rows="3" placeholder="Why this risk is accepted" required style="width:100%;font-family:var(--sans);font-size:13px;padding:8px 10px;border:1px solid var(--hairline);border-radius:var(--r-md);background:var(--surface);color:var(--ink);resize:vertical"></textarea>
<div style="display:flex;align-items:center;gap:10px"><button type="submit">Accept this risk</button><span class="muted" style="font-size:11.5px">Applies to this subject&#8211;signal pair. No expiry, no status, no author.</span></div>
</form>
{{else}}
<p class="muted">No annotation. An admin can accept this signal as a known risk.</p>
{{end}}
</div>

{{if or .Drawer.First .Drawer.Last}}
<div style="margin-bottom:var(--space-5)">
<div class="microlabel" style="margin-bottom:10px">History</div>
{{if .Drawer.Last}}<div class="timeline"><div style="font-weight:500;color:var(--ink)">{{if .Drawer.Withdrawn}}Last seen firing{{else}}Still present{{end}}</div><div class="muted mono" style="font-size:12px">{{.Drawer.Last}}</div></div>{{end}}
{{if .Drawer.First}}<div class="timeline"><div style="font-weight:500;color:var(--ink)">Signal raised</div><div class="muted mono" style="font-size:12px">{{.Drawer.First}}{{if .Drawer.SigID}} &#183; {{.Drawer.SigID}}{{end}}</div></div>{{end}}
{{if .Drawer.Withdrawn}}<div class="timeline"><div style="font-weight:500;color:var(--ink)">Withdrawn by the world</div><div class="muted">The subject left this rule's population; the acceptance stays recorded until you withdraw it.</div></div>{{end}}
</div>
{{end}}

<div class="drawer-actions">
{{if .IsAdmin}}<a class="btn secondary" href="{{.DescopeHref}}">Descope seed</a>{{end}}
<a class="btn secondary" href="{{.CloseHref}}">Close</a>
</div>
</aside>
{{end}}

{{if and .Descope .IsAdmin}}
<a class="scrim" href="{{.CloseHref}}" aria-label="Cancel"></a>
<div style="position:fixed;inset:0;z-index:41;display:flex;align-items:center;justify-content:center;padding:16px;pointer-events:none">
<div class="dialog-panel" style="pointer-events:auto" role="dialog" aria-modal="true" aria-label="Descope seed">
<h1 style="margin:0 0 8px;font-size:16px">Descope seed</h1>
<p class="muted" style="margin:0 0 var(--space-5);line-height:1.55">Removes the name and its subjects from scope. Spans close as descoped in the next batch,
and the exclusion is recorded on Scope. This does not resolve any signal &#8212; it narrows what
you measure.</p>
<form method="post" action="/exclusions">
<input type="hidden" name="kind" value="subtree">
<label><span>Type the exact name to descope</span><input name="value" placeholder="host.example.com" autocomplete="off" spellcheck="false" required></label>
<div class="dialog-actions">
<a class="btn secondary" href="{{.CloseHref}}">Cancel</a>
<button class="btn danger" type="submit">Descope seed</button>
</div>
</form>
</div>
</div>
{{end}}
{{template "foot" .}}{{end}}
`
