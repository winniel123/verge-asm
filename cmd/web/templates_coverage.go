package main

import "html/template"

// Coverage screen — the V3 redesign of /coverage (#301, design-system map #294,
// ADR-0110). Composed after design-system/examples/console/Coverage.jsx
// (07-console.jpg): a two-column grid whose left column carries the aperture
// meters ("what the last batch walked") and the coverage messages, and whose
// right column carries the gaps register ("expected, not observed") and the
// unevaluable-rules list.
//
// The example is a mock over sample data; this port keeps its layout, hierarchy,
// spacing and copy verbatim and swaps the sample data for real reads of the same
// shape (coldTemplates' coveragePage does the shaping). Where a region has no
// honest read yet it ships the design-system empty-state, never fabricated data:
//
//   - Aperture meters render in the CoverageMeter *census* state only — a census
//     claims no denominator and no percentage, which is the aperture screen's
//     standing rule (ADR-0095: state what the instrument looks at, never a
//     proportion of the estate). An address scope's census counts the addresses it
//     enumerates; a name scope's counts the owner names its zone declares, its
//     own addresses arriving by resolution.
//   - Coverage messages and the gaps register are wired from the Gap'd-Service
//     register (ListBlanketedReachServices, #254) and the unavailable-vantage
//     register (ListUnavailableVantages, ADR-0108).
//   - Unevaluable rules are the rules whose current census carries not-evaluable
//     members this batch (signal.EvaluateCorpus).
//   - The zone-gone-stale callout is gated on a real stale-zone read; that read
//     lands later, so the block ports the structure without ever rendering
//     fabricated staleness.
//
// This is template-local CSS translated from design-system/components/* within the
// existing token vocabulary (CoverageMeter, CoverageMessageList, GapBadge,
// SignalRuleRef, Callout, Card) — restyling, not authoring (ADR-0109). No
// design-system component is authored here.
var _ = template.Must(tmpl.Parse(coverageTemplates))

const coverageTemplates = `
{{define "coverage"}}{{template "head" .}}
{{template "chrome" .}}
<main data-screen-label="Coverage" style="max-width:1440px;margin:0 auto;padding:var(--space-6);display:flex;flex-direction:column;gap:var(--space-5)">

<header style="display:flex;flex-direction:column;gap:2px">
  <h1 style="margin:0;font-size:21px">Coverage</h1>
  <span class="muted" style="font-size:12.5px">Where "we cannot construct this claim" lives &#8212; a feature, not an error.</span>
</header>

<div style="display:grid;grid-template-columns:minmax(0, 1fr) minmax(0, 1fr);gap:var(--space-5);align-items:start">

  <div style="display:flex;flex-direction:column;gap:var(--space-5)">

    <section class="section" style="margin:0">
      <div style="display:flex;flex-direction:column;gap:3px;margin-bottom:var(--space-4)">
        <span class="microlabel">Aperture</span>
        <h2 style="margin:0;font-size:15px">What the last batch walked</h2>
      </div>
      {{if .Meters}}
      <div style="display:flex;flex-direction:column;gap:18px">
        {{range .Meters}}
        <div style="display:flex;flex-direction:column;gap:6px">
          <div style="display:flex;align-items:baseline;gap:8px">
            <span style="font:500 11px var(--mono);letter-spacing:0.07em;text-transform:uppercase;color:var(--muted)">{{.Label}}</span>
            <span style="margin-left:auto;font:400 11.5px var(--mono);color:var(--body);white-space:nowrap">census &#183; {{.Count}}{{if .Unit}} {{.Unit}}{{end}}</span>
          </div>
          <div aria-label="Census &#8212; no denominator" style="height:6px;border-radius:999px;overflow:hidden;background:repeating-linear-gradient(45deg, var(--accent-soft) 0 5px, var(--sunken) 5px 10px)"></div>
          {{if .Detail}}<span style="font:400 11px var(--sans);color:var(--muted)">{{.Detail}}</span>{{end}}
        </div>
        {{end}}
      </div>
      {{else}}
      <div class="emptystate">
        <div class="microlabel">Nothing declared</div>
        <h2>No scope to walk yet</h2>
        <p style="max-width:60ch;margin:var(--space-3) auto">No scope is declared, so the last batch walked nothing. Declare a name or address scope and the aperture fills as the batch reports at its cadence.</p>
        <a class="btn ghost" href="/scope">Go to Scope</a>
      </div>
      {{end}}
    </section>

    <section class="section" style="margin:0">
      <div style="display:flex;flex-direction:column;gap:3px;margin-bottom:var(--space-4)">
        <span class="microlabel">Currency</span>
        <h2 style="margin:0;font-size:15px">Coverage messages</h2>
      </div>
      {{if .Messages}}
      <div style="display:flex;flex-direction:column">
        {{range $i, $m := .Messages}}
        <div style="display:grid;grid-template-columns:auto 1fr;gap:12px;align-items:start;padding:11px 0;{{if $i}}border-top:1px solid var(--hairline){{end}}">
          <span style="padding-top:1px">
            {{if eq $m.Kind "gap"}}<span style="display:inline-flex;align-items:center;gap:4px;height:18px;padding:0 6px;border-radius:var(--r-sm);border:1px dotted var(--border-strong);color:var(--body);font-family:var(--mono);font-size:10px;font-weight:600;letter-spacing:0.04em;white-space:nowrap">{{$m.Badge}}</span>{{else}}<span class="chip stale" style="font-size:10px">{{$m.Badge}}</span>{{end}}
          </span>
          <span style="display:flex;flex-direction:column;gap:2px;min-width:0">
            <span style="font:600 12px var(--mono);color:var(--ink);overflow:hidden;text-overflow:ellipsis;white-space:nowrap">{{$m.Subject}}</span>
            <span style="font:400 12.5px/1.5 var(--sans);color:var(--body)">{{$m.Text}}</span>
          </span>
        </div>
        {{end}}
      </div>
      {{else}}
      <div class="emptystate">
        <div class="microlabel">All current</div>
        <h2>No coverage messages</h2>
        <p style="max-width:60ch;margin:var(--space-3) auto">Nothing has aged past its currency and no position has gone silent. A gap, a stale source, or a vantage we could not look from would surface here.</p>
      </div>
      {{end}}
    </section>

  </div>

  <div style="display:flex;flex-direction:column;gap:var(--space-5)">

    <section class="section" style="margin:0">
      <div style="display:flex;flex-direction:column;gap:3px;margin-bottom:var(--space-4)">
        <span class="microlabel">Gaps</span>
        <h2 style="margin:0;font-size:15px">Expected, not observed</h2>
      </div>
      {{if .Gaps}}
      <table class="vg-table">
        <thead><tr><th>Subject</th><th>Gap</th><th>Expected</th><th style="text-align:right">Since</th></tr></thead>
        <tbody>
        {{range .Gaps}}<tr>
          <td class="mono">{{.Subject}}</td>
          <td><span style="display:inline-flex;align-items:center;gap:4px;height:18px;padding:0 6px;border-radius:var(--r-sm);border:1px dotted var(--border-strong);color:var(--body);font-family:var(--mono);font-size:10px;font-weight:600;letter-spacing:0.04em;white-space:nowrap">{{.Gap}}</span></td>
          <td class="muted">{{.Expected}}</td>
          <td class="mono" style="text-align:right">{{.Since}}</td>
        </tr>{{end}}
        </tbody>
      </table>
      {{else}}
      <div class="emptystate">
        <div class="microlabel">Nothing expected</div>
        <h2>No gaps this batch</h2>
        <p style="max-width:60ch;margin:var(--space-3) auto">Every expected observation arrived. A subject we expected to observe but did not &#8212; a name with no address, a service with no origin behind its edge &#8212; would appear here.</p>
      </div>
      {{end}}
    </section>

    <section class="section" style="margin:0">
      <div style="display:flex;flex-direction:column;gap:3px;margin-bottom:var(--space-4)">
        <span class="microlabel">Rules</span>
        <h2 style="margin:0;font-size:15px">Unevaluable this batch</h2>
      </div>
      {{if .Unevaluable}}
      <div style="display:flex;flex-direction:column;gap:12px">
        {{range .Unevaluable}}
        <div style="display:flex;align-items:baseline;gap:10px;flex-wrap:wrap">
          <span style="display:inline-flex;align-items:center;gap:6px;height:22px;padding:0 8px;border-radius:var(--r-sm);background:var(--sunken);border:1px solid var(--hairline);color:var(--body);font:500 11.5px var(--mono)">{{.ID}}@{{.Version}}</span>
          <span style="font:400 12px/1.6 var(--sans);color:var(--muted)">{{.Why}}</span>
        </div>
        {{end}}
      </div>
      {{else}}
      <div class="emptystate">
        <div class="microlabel">All evaluated</div>
        <h2>Every rule could evaluate</h2>
        <p style="max-width:60ch;margin:var(--space-3) auto">No rule was blocked from evaluating this batch. A rule that needs a reading the batch did not commit &#8212; a handshake never completed, a zone that aged into a gap &#8212; would be listed here with what it was waiting on.</p>
      </div>
      {{end}}
    </section>

    {{if .StaleZones}}
    <div class="banner warn">
      <div><strong>Zone gone stale</strong> &#8212; a supplied zone file has aged past two re-supply intervals, so removal detection is suspended for that scope until a fresh upload.
        <div style="margin-top:10px"><a class="btn secondary" href="/scope">Upload zone</a></div>
      </div>
    </div>
    {{end}}

  </div>

</div>
</main>
{{template "foot" .}}{{end}}
`
