package main

import "html/template"

// Signals screen — canonical `/signals`. Ported from
// design-system/examples/console/Signals.jsx (+ SignalData.jsx) re-skinned around
// the real Derived corpus (ADR-0110: port the example's composition verbatim,
// swapping only sample data for real data of the same shape). The JSX's visual
// language and affordances are adopted — the Open/Annotated/Withdrawn tabs, the
// operator dial (AnnotationControl), the row-detail Drawer, the WithdrawnMark, and
// the typed-name descope ConfirmDialog — rendered server-side via query params
// (T0 ships no client drawer/tab/dialog machinery, and this file must not touch
// the shell or add a second stylesheet). All classes are T0's shared pageCSS.
//
// Three divergences from Signals.jsx are forced by the domain, which owns the
// vocabulary and IA where they collide with a visual convention (docs/agents/
// design-system.md: "the domain term wins and the visual convention gets
// re-skinned around it"):
//
//  1. NO SEVERITY. CONTEXT.md is explicit — "A signal carries no severity: it is a
//     named fact, and urgency belongs to the transition that surfaces it", and it
//     lists `severity` among the rejected words; signal.go states the same. So the
//     JSX's severity column, severity sort and severity filter are dropped, and the
//     SeverityBadge / `.sev` ramp is unused on this screen.
//  2. THE CENSUS IS NOT A FLAT RANKED TABLE. A Signal is the current-state census of
//     one rule — three registers (fired / did not fire / not-evaluable) over one
//     population — and ADR-0024/ADR-0102 forbid ranking, sampling or truncating it
//     into one paginated table. The census keeps the per-rule members (each drillable
//     to its subject, no per-row control, header count locked to list.length); the
//     JSX's single sortable table + pagination + per-row search are the per-instance
//     IA the census rejects.
//  3. NO FABRICATED PER-SIGNAL FIELDS. The JSX rows carry SIG-ids, CVEs, per-signal
//     timestamps — none exist in the domain, so none are invented.
//
// Tab data mapping (all real): the Open tab is the working surface — the rule
// censuses (the open signals) plus the full acceptance ledger, so the census and
// its annotations are read in one place (this is the screen the annotations flow
// is pinned against). Annotated narrows the ledger to accepted risks still in
// population; Withdrawn narrows it to orphans — a subject that has left its rule's
// population, withdrawn by the world, never resolved by an operator (ADR-0092). The
// descope confirm posts to the existing `POST /exclusions` (kind=subtree) — the real
// "remove a name and its subjects from scope, recorded on Scope" act — so no route
// is added here.
var _ = template.Must(tmpl.Parse(signalsTemplates))

const signalsTemplates = `
{{define "withdrawnmark"}}<span class="mono" style="display:inline-flex;align-items:center;gap:4px;height:18px;padding:0 6px;border-radius:var(--r-sm);border:1px dashed var(--drift-loss-border);color:var(--drift-loss-fg);font-size:10px;font-weight:600;letter-spacing:0.04em;white-space:nowrap"><svg viewBox="0 0 10 10" width="9" height="9" style="flex:none" aria-hidden="true"><path d="M1.6 5h6.8" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"></path></svg>withdrawn</span>{{end}}

{{define "signals"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div style="display:flex;align-items:flex-start;gap:16px;margin-bottom:var(--space-5)">
<div>
<div class="microlabel">Derived · signals</div>
<h1 style="margin-bottom:4px">Signals</h1>
<span class="muted">Raised when your attack surface drifts. Each rule is a named fact read over your estate as it stands now — current state, never a trend and never ranked.</span>
</div>
<div style="margin-left:auto;display:flex;gap:8px;align-items:center">
{{if .IsAdmin}}<a class="btn secondary" href="/signals?tab={{.Tab}}&amp;descope=1">Descope seed</a>{{end}}
<button class="btn secondary" disabled title="Signal export lands with the reporting pipeline">Export CSV</button>
</div>
</div>

<div class="tabs">
<a class="tab{{if eq .Tab "open"}} active{{end}}" href="/signals?tab=open">Open<span class="mono" style="margin-left:6px;font-size:10.5px;color:var(--muted)">{{.OpenCount}}</span></a>
<a class="tab{{if eq .Tab "annotated"}} active{{end}}" href="/signals?tab=annotated">Annotated<span class="mono" style="margin-left:6px;font-size:10.5px;color:var(--muted)">{{.AnnotatedCount}}</span></a>
<a class="tab{{if eq .Tab "withdrawn"}} active{{end}}" href="/signals?tab=withdrawn">Withdrawn<span class="mono" style="margin-left:6px;font-size:10.5px;color:var(--muted)">{{.WithdrawnCount}}</span></a>
</div>

{{if eq .Tab "open"}}
<p class="muted" style="max-width:82ch;margin-bottom:var(--space-5)">A signal's census is three members over one population — fired, did not fire, and
not-evaluable — current state only, never a delta or a trend. A signal is a named fact, not
a ranked one: urgency belongs to the transition that surfaces it, never to the rule. All
seventeen v1 rules render — Name, Service and Endpoint — each over the population its own
name could be true of.</p>

{{range .Censuses}}
<div class="section">
<div class="rulehead">
<h2 class="mono">{{.Rule}}</h2>
<span class="microlabel ver">version {{.Version}}</span>
</div>
{{if .Empty}}
<div class="census">
<div class="microlabel">No population</div>
<p>No subject in your estate is one this rule's fact could be true of, so there
is nothing to census. This is a legible state, not a clean bill of health.</p>
</div>
{{else}}
<div class="members">
{{range .Groups}}
<div class="mgroup">
{{if .Prose}}
<div class="fullmute">
<div class="microlabel">Fired · every subject accepted</div>
<p>{{.Prose}}</p>
</div>
{{else}}
<div class="mgroup-head"><span class="microlabel">{{.Label}}</span><span class="count">{{len .Members}}</span></div>
{{if .Members}}
<ul class="mgroup-list">
{{range .Members}}<li><a class="mono" href="{{.Href}}">{{.Subject}}</a></li>{{end}}
</ul>
{{else}}
<div class="mgroup-empty">None.</div>
{{end}}
{{end}}
</div>
{{end}}
</div>
{{end}}
</div>
{{end}}

<div class="section">
<div class="microlabel">Operator dial · annotations</div>
<h2>Annotations</h2>
<p class="muted" style="max-width:82ch">An annotation accepts one rule's firing on one subject as a known risk on a thing you are
still measuring. It moves no number — the subject is still measured, still counted under
fired — and changes only the message: an annotated pair's next firing reaches no one. It
carries your reason and the instant you declared it, and nothing else: no status, no expiry
and no author. Declaring and withdrawing are neither of them a message.</p>
{{if .IsAdmin}}
{{if .AnnoError}}<div class="error">{{.AnnoError}}</div>{{end}}
<form method="post" action="/annotations" class="annoform">
<label class="grow"><span>Subject</span><input name="subject" value="{{.AnnoSubject}}" placeholder="host.example.com" autocomplete="off" required></label>
<label><span>Signal</span><select name="signal" required>
<option value="">Choose a signal…</option>
{{range .RuleNames}}<option value="{{.}}"{{if eq . $.AnnoSignal}} selected{{end}}>{{.}}</option>{{end}}
</select></label>
<label class="grow"><span>Reason</span><input name="reason" value="{{.AnnoReason}}" placeholder="Why this risk is accepted" autocomplete="off" required></label>
<button type="submit">Accept this risk</button>
</form>
{{end}}
{{if .Annotations}}
<table class="annos vg-table">
<tr><th>Subject</th><th>Signal</th><th>Reason</th><th>Declared</th>{{if .IsAdmin}}<th></th>{{end}}<th></th></tr>
{{range .Annotations}}
<tr{{if and $.Sel (eq $.Sel .ID)}} class="row-selected"{{end}}>
<td><a class="mono" href="{{.Href}}">{{.Subject}}</a>{{if .Orphan}} <span class="orphan">names no current member</span>{{end}}</td>
<td class="mono">{{.Signal}}</td>
<td>{{.Reason}}</td>
<td class="mono">{{.At}}</td>
{{if $.IsAdmin}}<td><form method="post" action="/annotations/withdraw"><input type="hidden" name="id" value="{{.ID}}"><button class="secondary" type="submit">Withdraw</button></form></td>{{end}}
<td><a href="/signals?tab={{$.Tab}}&amp;view={{.ID}}">Detail</a></td>
</tr>
{{end}}
</table>
{{else}}
<p class="muted">No annotation is declared. A fired signal you accept as a known
risk is declared here, keyed on its subject.</p>
{{end}}
</div>
{{end}}

{{if eq .Tab "annotated"}}
<div class="section">
<div class="microlabel">Operator dial · annotations</div>
<h2>Annotated</h2>
<p class="muted" style="max-width:82ch">Accepted risks still in their rule's population. An annotation moves no number — the
subject is still measured and still counted under fired — and changes only the message: an
annotated pair's next firing reaches no one. It carries your reason and the instant you
declared it, and nothing else.</p>
{{if .IsAdmin}}
{{if .AnnoError}}<div class="error">{{.AnnoError}}</div>{{end}}
<form method="post" action="/annotations" class="annoform">
<label class="grow"><span>Subject</span><input name="subject" value="{{.AnnoSubject}}" placeholder="host.example.com" autocomplete="off" required></label>
<label><span>Signal</span><select name="signal" required>
<option value="">Choose a signal…</option>
{{range .RuleNames}}<option value="{{.}}"{{if eq . $.AnnoSignal}} selected{{end}}>{{.}}</option>{{end}}
</select></label>
<label class="grow"><span>Reason</span><input name="reason" value="{{.AnnoReason}}" placeholder="Why this risk is accepted" autocomplete="off" required></label>
<button type="submit">Accept this risk</button>
</form>
{{end}}
{{if .Annotated}}
<table class="annos vg-table">
<tr><th>Subject</th><th>Signal</th><th>Reason</th><th>Declared</th>{{if .IsAdmin}}<th></th>{{end}}<th></th></tr>
{{range .Annotated}}
<tr{{if and $.Sel (eq $.Sel .ID)}} class="row-selected"{{end}}>
<td><a class="mono" href="{{.Href}}">{{.Subject}}</a>{{if .Orphan}} <span class="orphan">names no current member</span>{{end}}</td>
<td class="mono">{{.Signal}}</td>
<td>{{.Reason}}</td>
<td class="mono">{{.At}}</td>
{{if $.IsAdmin}}<td><form method="post" action="/annotations/withdraw"><input type="hidden" name="id" value="{{.ID}}"><button class="secondary" type="submit">Withdraw</button></form></td>{{end}}
<td><a href="/signals?tab={{$.Tab}}&amp;view={{.ID}}">Detail</a></td>
</tr>
{{end}}
</table>
{{else}}
<div class="emptystate" style="margin-top:var(--space-4)">
<h2>No annotation is declared</h2>
<p>A fired signal you accept as a known risk is declared here, keyed on its subject. The
reason you record is the reviewable artefact — it is all an annotation carries.</p>
</div>
{{end}}
</div>
{{end}}

{{if eq .Tab "withdrawn"}}
<div class="section">
<div class="microlabel">Withdrawn · by the world</div>
<h2>Withdrawn</h2>
<p class="muted" style="max-width:82ch">A signal you accepted whose subject has since left its rule's population. The world moved
and the key is in no current census — withdrawn by the world, not resolved by you. The
acceptance stays recorded here until you withdraw it.</p>
{{if .Withdrawn}}
<table class="annos vg-table">
<tr><th>Subject</th><th>Signal</th><th>Reason</th><th>Declared</th>{{if .IsAdmin}}<th></th>{{end}}<th></th></tr>
{{range .Withdrawn}}
<tr{{if and $.Sel (eq $.Sel .ID)}} class="row-selected"{{end}}>
<td><a class="mono" href="{{.Href}}">{{.Subject}}</a> {{template "withdrawnmark"}}{{if .Orphan}} <span class="orphan">names no current member</span>{{end}}</td>
<td class="mono">{{.Signal}}</td>
<td>{{.Reason}}</td>
<td class="mono">{{.At}}</td>
{{if $.IsAdmin}}<td><form method="post" action="/annotations/withdraw"><input type="hidden" name="id" value="{{.ID}}"><button class="secondary" type="submit">Withdraw</button></form></td>{{end}}
<td><a href="/signals?tab={{$.Tab}}&amp;view={{.ID}}">Detail</a></td>
</tr>
{{end}}
</table>
{{else}}
<div class="emptystate" style="margin-top:var(--space-4)">
<h2>No signal has withdrawn</h2>
<p>When a subject you annotated leaves its rule's population, its acceptance surfaces here
as withdrawn — derived on read, no operator act. Nothing has withdrawn right now.</p>
</div>
{{end}}
</div>
{{end}}
</main>

{{if .ViewAnno}}
<a class="scrim" href="/signals?tab={{.Tab}}" aria-label="Close detail"></a>
<aside class="drawer-panel" role="dialog" aria-label="Signal detail">
<div class="microlabel">{{if .ViewAnno.Orphan}}Withdrawn · signal detail{{else}}Annotated · signal detail{{end}}</div>
<h1 class="mono" style="margin:6px 0 2px;font-size:16px">{{.ViewAnno.Subject}}</h1>
<div class="muted mono" style="font-size:12px;margin-bottom:var(--space-5)">{{.ViewAnno.Signal}}</div>

<div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap;margin-bottom:var(--space-4)">
<span class="mono" style="display:inline-flex;align-items:center;gap:6px;height:20px;padding:0 8px;border-radius:var(--r-sm);background:var(--surface);border:1px solid var(--hairline);color:var(--muted);font-size:10.5px;font-weight:600;letter-spacing:0.05em"><span style="width:6px;height:6px;border-radius:999px;background:var(--muted)"></span>accepted risk</span>
{{if .ViewAnno.Orphan}}{{template "withdrawnmark"}}{{end}}
</div>

<p style="margin:0 0 var(--space-5);color:var(--body);line-height:1.55">{{.ViewAnno.Reason}}</p>

<table class="invfacets" style="margin-bottom:var(--space-5)">
<tr><td class="invfacet muted">Subject</td><td><a class="mono" href="{{.ViewAnno.Href}}">{{.ViewAnno.Subject}}</a></td></tr>
<tr><td class="invfacet muted">Signal</td><td class="mono">{{.ViewAnno.Signal}}</td></tr>
<tr><td class="invfacet muted">Declared</td><td class="mono">{{.ViewAnno.At}}</td></tr>
<tr><td class="invfacet muted">Population</td><td>{{if .ViewAnno.Orphan}}names no current member — withdrawn{{else}}current member of this rule{{end}}</td></tr>
</table>

<div class="drawer-actions">
{{if .IsAdmin}}<form method="post" action="/annotations/withdraw"><input type="hidden" name="id" value="{{.ViewAnno.ID}}"><button class="btn secondary" type="submit">Withdraw annotation</button></form>{{end}}
<a class="btn secondary" href="/signals?tab={{.Tab}}">Close</a>
</div>
</aside>
{{end}}

{{if and .Descope .IsAdmin}}
<a class="scrim" href="/signals?tab={{.Tab}}" aria-label="Cancel"></a>
<div style="position:fixed;inset:0;z-index:41;display:flex;align-items:center;justify-content:center;padding:16px;pointer-events:none">
<div class="dialog-panel" style="pointer-events:auto" role="dialog" aria-modal="true" aria-label="Descope seed">
<h1 style="margin:0 0 8px;font-size:16px">Descope seed</h1>
<p class="muted" style="margin:0 0 var(--space-5);line-height:1.55">Removes the name and its subjects from scope. Spans close as descoped in the next batch,
and the exclusion is recorded on Scope. This does not resolve any signal — it narrows what
you measure.</p>
<form method="post" action="/exclusions">
<input type="hidden" name="kind" value="subtree">
<label><span>Type the exact name to descope</span><input name="value" placeholder="host.example.com" autocomplete="off" spellcheck="false" required></label>
<div class="dialog-actions">
<a class="btn secondary" href="/signals?tab={{.Tab}}">Cancel</a>
<button class="btn danger" type="submit">Descope seed</button>
</div>
</form>
</div>
</div>
{{end}}
{{template "foot" .}}{{end}}
`
