package main

import "html/template"

// Dashboard screen — canonical `/` (#277, V2 console map #275). Composed after
// design-system/examples/console/Dashboard.jsx (01-console.jpg): a KPI band, a
// running-scan banner, a by-severity card, a coverage card, a vantage-health card
// and the open-signal register.
//
// The example is a mock over a severity-scored, per-instance signal model this
// product does not have. Two of its regions are domain-incompatible and are
// re-skinned to honest current-state facts + design-system empty-states rather
// than fabricated data (CONTEXT.md; signals.go; the same stance Reports took):
//
//   - Signals carry NO severity — the census is "deliberately not a severity
//     ramp". So the "By severity" bars have no real series (empty-stated), and the
//     "most recent signals" table carries no severity pill and no per-signal
//     recency feed: it is reshaped to the current per-rule firing census, the one
//     honest signal read.
//   - Coverage denominators live on their own screen; the landing points to them
//     rather than restating a partial figure.
//
// Everything else is wired from real reads: the KPI counts (open-signal census,
// current Name/Service subjects, declared scopes, in-flight scans), the vantage
// health (ListVantages), the unreachable-vantage banner (ListUnavailableVantages),
// and the running-scan state (the #245 active-dispatch source). Where a read does
// not resolve, the figure degrades to an em dash, never a fabricated zero.
var _ = template.Must(tmpl.Parse(dashboardTemplates))

const dashboardTemplates = `
{{define "home"}}{{template "head" .}}
{{template "chrome" .}}
<main style="display:flex;flex-direction:column;gap:var(--space-5)">

<header style="display:flex;align-items:center;gap:var(--space-4)">
  <div style="display:flex;flex-direction:column;gap:2px">
    <h1 style="margin:0;font-size:21px">Dashboard</h1>
    <span class="muted" style="font-size:12.5px">Signals, coverage and scan activity across your estate.</span>
  </div>
  <div style="margin-left:auto;display:flex;gap:var(--space-2)">
    <a class="btn secondary" href="/scope">Add seed</a>
    <a class="btn" href="/scans">Run scan</a>
  </div>
</header>

{{if .Scanning}}
<div class="banner info">
  <span class="dot live"></span>
  <div><strong>Scan running</strong> &#8212; {{.ActiveScans}} scan{{if ne .ActiveScans 1}}s{{end}} in flight. Figures update as the worker reports.</div>
</div>
{{end}}

{{if .Unavailable}}
<div class="banner warn">
  <div><strong>Vantage unreachable</strong> &#8212; {{range $i, $n := .Unavailable}}{{if $i}}, {{end}}<span class="mono">{{$n}}</span>{{end}}. Scans continue from your other vantages.</div>
</div>
{{end}}

<div class="kpis" style="margin-bottom:0">
  <div class="kpi"><div class="kpi-label">Open signals</div><div class="kpi-num">{{if .HasOpenSignals}}{{.OpenSignals}}{{else}}&#8212;{{end}}</div><div class="kpi-delta">firing right now</div></div>
  <div class="kpi"><div class="kpi-label">Names watched</div><div class="kpi-num">{{if .HasNames}}{{.Names}}{{else}}&#8212;{{end}}</div><div class="kpi-delta">in your estate</div></div>
  <div class="kpi"><div class="kpi-label">Services seen</div><div class="kpi-num">{{if .HasServices}}{{.Services}}{{else}}&#8212;{{end}}</div><div class="kpi-delta">current reachability</div></div>
  <div class="kpi"><div class="kpi-label">Scopes declared</div><div class="kpi-num">{{if .HasScopes}}{{.Scopes}}{{else}}&#8212;{{end}}</div><div class="kpi-delta">{{if .HasScopes}}{{.NameScopes}} name &#183; {{.AddrScopes}} address{{else}}unavailable{{end}}</div></div>
  <div class="kpi"><div class="kpi-label">Scans in flight</div><div class="kpi-num">{{.ActiveScans}}</div><div class="kpi-delta">dispatched now</div></div>
</div>

<div style="display:grid;grid-template-columns:380px 1fr;gap:var(--space-5);align-items:start">
  <div style="display:flex;flex-direction:column;gap:var(--space-5)">

    <section class="section" style="margin-bottom:0">
      <div style="display:flex;flex-direction:column;gap:3px;margin-bottom:var(--space-4)">
        <span class="microlabel">Open signals</span>
        <h2 style="margin:0;font-size:15px">By severity</h2>
      </div>
      <div class="emptystate">
        <div class="microlabel">No severity ramp</div>
        <h2>Signals carry no severity</h2>
        <p style="max-width:60ch;margin:var(--space-3) auto">A signal is a census member, not a scored one &#8212; the register set is deliberately not a severity ramp, so there is nothing to rank here. See each rule's fired, did-not-fire and not-evaluable members on Signals.</p>
        <a class="btn ghost" href="/signals">Go to Signals</a>
      </div>
    </section>

    <section class="section" style="margin-bottom:0">
      <div style="display:flex;flex-direction:column;gap:3px;margin-bottom:var(--space-4)">
        <span class="microlabel">Coverage</span>
        <h2 style="margin:0;font-size:15px">Did we look, how completely</h2>
      </div>
      <div class="emptystate">
        <div class="microlabel">Lives on Coverage</div>
        <h2>Coverage detail is on its own screen</h2>
        <p style="max-width:60ch;margin:var(--space-3) auto">The denominator reads &#8212; how much of each declared range and zone you have looked at, and how current each is &#8212; live on Coverage. This landing does not restate a partial figure.</p>
        <a class="btn ghost" href="/coverage">Go to Coverage</a>
      </div>
    </section>

    <section class="section" style="margin-bottom:0">
      <div style="display:flex;flex-direction:column;gap:3px;margin-bottom:var(--space-4)">
        <span class="microlabel">Vantages</span>
        <h2 style="margin:0;font-size:15px">Scan infrastructure</h2>
      </div>
      {{if .Vantages}}
      <div style="display:flex;flex-direction:column;gap:var(--space-3)">
        {{range .Vantages}}
        <div style="display:flex;align-items:center;gap:10px">
          <span class="mono" style="font-size:12.5px;color:var(--body)">{{.Name}}</span>
          <span class="mono muted" style="font-size:12px">{{.Class}}</span>
          <span style="margin-left:auto">{{if eq .Avail "available"}}<span class="badge">available</span>{{else if eq .Avail "unavailable"}}<span class="badge off">unavailable</span>{{else}}<span class="badge off">{{if .Avail}}{{.Avail}}{{else}}unknown{{end}}</span>{{end}}</span>
        </div>
        {{end}}
      </div>
      {{else}}
      <div class="emptystate">
        <div class="microlabel">None provisioned</div>
        <h2>No vantage yet</h2>
        <p style="max-width:60ch;margin:var(--space-3) auto">A vantage is a position you scan from. None is provisioned, so scans resolve from the shipped resolver only. Provision a prober on Scope to measure reachability from the internet.</p>
        <a class="btn ghost" href="/scope">Go to Scope</a>
      </div>
      {{end}}
    </section>

  </div>

  <section class="section" style="margin-bottom:0">
    <div style="display:flex;align-items:center;gap:var(--space-3);margin-bottom:var(--space-4)">
      <div style="display:flex;flex-direction:column;gap:3px">
        <span class="microlabel">Open signals</span>
        <h2 style="margin:0;font-size:15px">Firing now, by rule</h2>
      </div>
      <a class="btn ghost" href="/signals" style="margin-left:auto">View all</a>
    </div>
    {{if .Firing}}
    <table class="vg-table">
      <thead><tr><th>Rule</th><th>Subject</th><th style="text-align:right">Fired</th></tr></thead>
      <tbody>
      {{range .Firing}}<tr>
        <td class="mono">{{.Rule}}</td>
        <td><span class="badge">{{.Kind}}</span></td>
        <td class="mono" style="text-align:right">{{.Fired}}</td>
      </tr>{{end}}
      </tbody>
    </table>
    {{else if .HasOpenSignals}}
    <div class="emptystate">
      <div class="microlabel">All quiet</div>
      <h2>No signals firing</h2>
      <p style="max-width:60ch;margin:var(--space-3) auto">No rule is firing on any subject right now. A signal appears here the moment the world moves your estate into a rule's population, and is withdrawn just as quietly when the world moves back.</p>
      <a class="btn ghost" href="/signals">Go to Signals</a>
    </div>
    {{else}}
    <div class="emptystate">
      <div class="microlabel">Unavailable</div>
      <h2>Signal census could not be read</h2>
      <p style="max-width:60ch;margin:var(--space-3) auto">The signal register did not resolve on this load. Open Signals for the live census.</p>
      <a class="btn ghost" href="/signals">Go to Signals</a>
    </div>
    {{end}}
  </section>
</div>

</main>
{{template "foot" .}}{{end}}
`
