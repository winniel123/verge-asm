package main

import "html/template"

// Drift screen — canonical `/drift` (nav item 4 of 7, the product's thesis per
// ADR-0108/ADR-0110). Ported verbatim from design-system/examples/console/Drift.jsx:
// the change-kind legend, the two-column "By batch" transitions timeline beside a
// "Movement" summary, and the per-event before/after diff affordance. Change rides
// its own palette (the `.chip.gain/.change/.loss` drift tokens shipped in T0) — never
// the severity ramp. The transition feed is estate-wide and batch-grouped (#288,
// ADR-0111): ListRecentDriftEvents returns the span open/close events for the selected
// period and buildDriftFeed classifies them into the six change kinds and groups them
// by batch, so Groups, the Movement tally, the period selector and the CSV export all
// read one feed. The timeline renders the design-system empty-state until a second
// batch has folded a value that moved. The legend is the change vocabulary itself —
// definitional, not data — so it always renders. Change events are never fabricated.
//
// T16 delta (#311): the header carries a "Batch detail" entry opening the Run detail
// screen (T2, GET /run/{id}; id is a Dispatch id) for the most recent batch. It is
// offered only when a real dispatch exists (BatchID non-zero) and omitted otherwise —
// mirroring the ported example's `onOpenRun && <Button>Batch detail</Button>`; no id
// is ever fabricated. A batch is Operational, a distinct feed from change, so the
// entry stands even while the transition timeline is still the empty-state.
var _ = template.Must(tmpl.Parse(driftTemplates))

const driftTemplates = `
{{define "changeglyph"}}<svg viewBox="0 0 10 10" width="10" height="10" style="flex:none" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">` +
	`{{if eq . "appeared"}}<path d="M5 1.6v6.8M1.6 5h6.8"></path>` +
	`{{else if eq . "revealed"}}<circle cx="5" cy="5" r="3.4" stroke-dasharray="2.1 1.7"></circle>` +
	`{{else if eq . "returned"}}<path d="M8.2 5.8A3.3 3.3 0 1 1 7.4 2.8"></path><path d="M7 1l0.6 2L5.6 3.4"></path>` +
	`{{else if eq . "withdrawn"}}<path d="M1.6 5h6.8"></path>` +
	`{{else if eq . "descoped"}}<path d="M1.6 6.4h6.8"></path><circle cx="5" cy="2.6" r="1" fill="currentColor" stroke="none"></circle>` +
	`{{else if eq . "changed"}}<path d="M5 1.8L8.4 8H1.6Z"></path>{{end}}</svg>{{end}}

{{define "drift"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div style="display:flex;align-items:flex-start;gap:16px;margin-bottom:var(--space-5)">
<div>
<div class="microlabel">Derived · drift</div>
<h1 style="margin-bottom:4px">Drift</h1>
<span class="muted">What moved since last time, grouped by batch. Change is its own language — not severity.</span>
</div>
<div style="margin-left:auto;display:flex;gap:8px;align-items:center">
<div style="display:flex;gap:4px;align-items:center">
{{range .Periods}}<a href="/drift?period={{.Token}}" class="badge" style="text-decoration:none{{if eq .Token $.Period}};border-color:var(--accent);color:var(--link){{end}}" title="Show transitions from {{.Label}}">{{.Label}}</a>{{end}}
</div>
{{if .HasEvents}}<a class="btn secondary" href="/drift/export?period={{.Period}}" style="text-decoration:none" title="Download the transition feed for this period as CSV">Export CSV</a>
{{else}}<button class="btn secondary" disabled title="Nothing to export until a transition is folded">Export CSV</button>{{end}}
{{if .BatchID}}<a class="btn secondary" href="/run/{{.BatchID}}" style="display:inline-flex;align-items:center;gap:6px;text-decoration:none" title="Open the most recent batch{{if .BatchLabel}} · {{.BatchLabel}}{{end}} in Run detail"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M14 2v4a2 2 0 0 0 2 2h4"></path><path d="M4 7V4a2 2 0 0 1 2-2h9l5 5v11a2 2 0 0 1-2 2h-3.5"></path><circle cx="7.5" cy="15.5" r="2.5"></circle><path d="m11 19-1.7-1.7"></path></svg>Batch detail</a>{{end}}
</div>
</div>

<div style="display:grid;grid-template-columns:1fr 320px;gap:24px;align-items:start">

<div class="section" style="margin-bottom:0">
<div class="microlabel">Transitions</div>
<h2>By batch</h2>
<div style="display:flex;gap:6px;flex-wrap:wrap;padding-bottom:var(--space-4);margin-bottom:var(--space-2);border-bottom:1px solid var(--hairline)">
{{range .Kinds}}<span class="chip {{.Family}}">{{template "changeglyph" .Change}}{{.Change}}</span>{{end}}
</div>
{{if .Truncated}}<div class="banner warn" style="margin-bottom:var(--space-3)">Showing the most recent {{.FeedLimit}} transitions for this period. Narrow the period to see older change.</div>{{end}}
{{if .Groups}}
{{range .Groups}}
<div class="timeline">
<div class="microlabel">{{.Label}}{{if .Meta}} · {{.Meta}}{{end}}</div>
{{range .Events}}
<div style="display:flex;flex-direction:column;gap:6px;padding:var(--space-3) 0;border-top:1px solid var(--hairline)">
<div style="display:flex;align-items:center;gap:8px;flex-wrap:wrap">
<span class="chip {{.Family}}">{{template "changeglyph" .Change}}{{.Change}}</span>
<span class="mono" style="font-size:12.5px;color:var(--ink)">{{.Subject}}</span>
<span class="mono muted" style="margin-left:auto">{{.Time}}</span>
</div>
<div class="muted">{{.Detail}}{{if .Reason}} · {{.Reason}}{{end}}</div>
{{if .Diff}}
<div class="mono" style="background:var(--sunken);border:1px solid var(--hairline);border-radius:var(--r-md);padding:var(--space-3);font-size:12px;display:flex;flex-direction:column;gap:2px">
{{range .Diff}}
{{if eq .Type "remove"}}<div style="color:var(--danger)">− {{.Text}}</div>
{{else if eq .Type "add"}}<div style="color:var(--ok)">+ {{.Text}}</div>
{{else}}<div class="muted">&nbsp;&nbsp;{{.Text}}</div>{{end}}
{{end}}
</div>
{{end}}
</div>
{{end}}
</div>
{{end}}
{{else}}
<div class="emptystate" style="margin-top:var(--space-4)">
<h2>No change to show yet</h2>
<p>Once two batches have folded a value for the same subject, their transitions appear
here — grouped by batch, each carried in its own change chip, with a before/after diff
where a held value moved. Declare a scope on Scope and let a scan run twice to begin
measuring change.</p>
</div>
{{end}}
</div>

<div class="section" style="margin-bottom:0">
<div class="microlabel">This period</div>
<h2>Movement</h2>
{{if .Groups}}
<p class="muted">Transitions folded in the selected period, by change kind.</p>
{{else}}
<p class="muted">No transition has been folded yet, so there is nothing to count. Each
change kind will tally here once a second batch measures a value that moved.</p>
{{end}}
<div style="display:flex;flex-direction:column;gap:8px;margin-top:var(--space-4)">
{{range .Kinds}}
<div style="display:flex;align-items:center;gap:10px">
<span class="chip {{.Family}}">{{template "changeglyph" .Change}}{{.Change}}</span>
<span class="mono muted" style="margin-left:auto">{{$c := index $.Movement .Change}}{{if $c}}{{$c}}{{else}}&#8212;{{end}}</span>
</div>
{{end}}
</div>
</div>

</div>
</main>
{{template "foot" .}}{{end}}
`
