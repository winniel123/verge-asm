package main

import "html/template"

// First-run checklist — the home's empty-estate state (#302, design-system V3 map
// #294). Composed after design-system/examples/console/FirstRun.jsx (19-console.jpg):
// a four-step checklist that stands in for the Dashboard until the estate holds
// observed subjects. It is a STATE of `/`, not a nav item — auth.go's dashboardData
// picks it when the estate is empty and renders the Dashboard otherwise.
//
// The four steps are the real setup funnel, each reflecting a real read (ADR-0110,
// verbatim port; only the JSX's sample data is swapped for real figures):
//
//   1. Declare your domain   — a scope/seed is declared (ListSeeds)
//   2. Upload a zone file    — a name scope holds a supplied zone file (ListZoneFileStatus)
//   3. Add an internet vantage — a prober classed `internet` exists (ListVantages)
//   4. Run the first batch   — a batch has been dispatched (ListDispatchProgress)
//
// Step 4 is GATED on step 3: without an internet vantage its action is disabled and
// names the gate ("Needs an internet vantage first"), and the footer states why —
// probing your own address from inside is a hairpinning trap that never traverses
// the inbound policy (the withheld/gating pattern, same signal exposure.go uses).
// No step is ever a fabricated "done"; each is the honest read or the invite to act.
//
// The JSX's design-system tokens are mapped to the shell's token vocabulary
// (--font-ui→--sans, --text-ink→--ink, --text-muted→--muted, --text-secondary→
// --body, --surface-sunken→--sunken, --border-default/--row-sep→--hairline). This
// is restyling within the existing token set, not authoring (ADR-0109).
var _ = template.Must(tmpl.Parse(firstRunTemplates))

const firstRunTemplates = `
{{define "firstrun"}}
<main style="max-width:760px;margin:0 auto;padding:56px 32px;display:flex;flex-direction:column;gap:20px">
  <header style="display:flex;flex-direction:column;gap:6px">
    <h1 style="margin:0;font:600 24px var(--sans);letter-spacing:-0.015em;color:var(--ink)">Welcome to Verge</h1>
    <span style="font:400 13px/1.6 var(--sans);color:var(--muted)">Each step unlocks a capability. Until they complete, the console stays honest about what it cannot conclude &#8212; exposure claims are withheld, never guessed.</span>
    <span style="font:500 11px var(--mono);letter-spacing:0.06em;color:var(--body);margin-top:6px">{{.FirstRunDone}} of 4 complete</span>
  </header>
  <div class="card" style="padding:10px">
    <div style="display:flex;flex-direction:column">
      {{range $i, $s := .FirstRunSteps}}
      <div style="display:flex;align-items:center;gap:14px;padding:16px 12px;{{if $i}}border-top:1px solid var(--hairline){{end}}">
        <span style="display:inline-flex;align-items:center;justify-content:center;width:28px;height:28px;flex:none;border-radius:999px;background:{{if $s.Done}}var(--ok-soft){{else}}var(--sunken){{end}};border:1px solid {{if $s.Done}}var(--ok-border){{else}}var(--hairline){{end}};color:{{if $s.Done}}var(--ok){{else}}var(--muted){{end}};font:600 12px var(--mono)">
          {{if $s.Done}}<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>{{else}}{{$s.N}}{{end}}
        </span>
        <span style="display:flex;flex-direction:column;gap:2px;min-width:0;flex:1">
          <span style="font:500 13.5px var(--sans);color:var(--ink)">{{$s.Title}}</span>
          <span style="font:400 12px/1.55 var(--sans);color:var(--muted)">{{$s.Detail}}</span>
        </span>
        {{if and $s.ActionLabel (not $s.Done)}}
          {{if $s.Gated}}
          <button class="btn secondary" disabled title="Needs an internet vantage first" style="opacity:0.55;cursor:not-allowed">{{$s.ActionLabel}}</button>
          {{else}}
          <a class="btn" href="{{$s.ActionHref}}">{{$s.ActionLabel}}</a>
          {{end}}
        {{end}}
      </div>
      {{end}}
    </div>
  </div>
  <span style="font:400 11.5px/1.6 var(--sans);color:var(--muted)">Step 4 stays gated until an internet vantage exists &#8212; probing your own address from inside is a hairpinning trap that never traverses the inbound policy.</span>
</main>
{{end}}
`
