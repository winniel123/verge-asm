package main

import "html/template"

// Dashboard screen — canonical `/` (#277, V2 console map #275). Ported to full parity
// with design-system/examples/console/Dashboard.jsx (01-console.jpg) under ADR-0116
// (P2.1, PARITY-CHART.md): the design is normative for look AND functionality, so the
// two re-skinned holds this screen used to carry — the "signals carry no severity"
// by-severity empty state and the "coverage is on its own screen" pointer — are
// deleted, and every region renders its real datum:
//
//   - Header sub-line: "last full scan Xm ago · next in Yh Zm" (P0.4, ScanSchedule).
//   - One framed stat band of five cells (open signals · critical · assets watched ·
//     exposed services · certs expiring ≤30d), each with its vs-last-batch delta chip
//     (P0.2, #443), replacing the five loose .kpi tiles.
//   - A running-scan Progress row with detail; a dismissible warn Banner with Retry.
//   - By-severity bars over the real five-level ramp (#442); a Coverage card with
//     census CoverageMeters (+ a StalenessBadge when a vantage is silent); a Vantages
//     card; and the most-recent Signals register as a flat per-instance table (#442).
//
// Per-vantage latency is now a real datum (P0.5, #485, resolving SPEC-CHANGE.md
// collision #7): the worker measures the round-trip of the prober connect that pins
// the host key and stores it nullable on the vantage, and this card renders the
// spec's mono "34ms" reading where a measurement exists, or the spec's pending em
// dash where the prober has not yet been reached — never a fabricated number and
// never a dropped card. Every other read is best-effort: a failed read degrades to
// an em dash or an empty region, never a fabricated value.
//
// The classes below are template-local CSS translated from the design-system
// components (Stat/DeltaChip, Progress, Banner, CoverageMeter, StalenessBadge,
// AvailabilityBadge, SeverityBadge) within the existing token vocabulary — restyling,
// not authoring (ADR-0109). No design-system component is authored here.
var _ = template.Must(tmpl.Parse(dashboardTemplates))

const dashboardTemplates = `
{{define "home"}}{{template "head" .}}
{{template "chrome" .}}
{{if .EmptyEstate}}{{template "firstrun" .}}{{else}}{{template "dashboard" .}}{{end}}
{{template "foot" .}}{{end}}

{{define "dashboard"}}
<style>
.dhead{display:flex;align-items:center;gap:var(--space-4)}
.dsub{font-size:12.5px;color:var(--muted);white-space:nowrap}
.dsub .dsub-v{font-family:var(--mono);font-size:12px;color:var(--body)}
.dbtn-ico{display:inline-flex;align-items:center;gap:6px}
.dbtn-ico svg{width:14px;height:14px}
/* Progress (indeterminate scan sweep) */
.dprog{display:flex;flex-direction:column;gap:6px}
.dprog-head{display:flex;align-items:baseline;gap:8px}
.dprog-detail{margin-left:auto;font-family:var(--mono);font-size:11.5px;color:var(--body);white-space:nowrap}
.dprog-track{height:6px;border-radius:999px;background:var(--sunken);overflow:hidden;position:relative}
.dprog-track > span{position:absolute;top:0;bottom:0;width:34%;border-radius:999px;background:var(--accent);animation:dash-sweep 1.4s ease-in-out infinite}
@keyframes dash-sweep{0%{left:-34%}100%{left:100%}}
/* Banner action + dismiss */
.dbanner{align-items:center}
.dbanner .banner-body{flex:1;min-width:0}
.dbanner .banner-act{flex:none;margin-left:8px}
.dbanner .banner-x{flex:none;width:24px;height:24px;margin:-2px -4px 0 0;display:inline-flex;align-items:center;justify-content:center;border-radius:var(--r-sm);color:var(--warn);text-decoration:none;font-size:15px;line-height:1}
.dbanner .banner-x:hover{background:var(--warn-border);text-decoration:none}
/* Framed stat band */
.statband{background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm)}
.statband-grid{display:grid;grid-template-columns:repeat(5,1fr)}
.statcell{padding:20px 24px;min-width:0}
.statcell + .statcell{border-left:1px solid var(--hairline)}
.stat-label{display:flex;align-items:center;gap:8px;font-family:var(--mono);font-size:11px;font-weight:500;letter-spacing:0.07em;text-transform:uppercase;color:var(--muted)}
.stat-live{width:7px;height:7px;border-radius:999px;background:var(--accent);animation:verge-pulse 1.6s infinite}
.stat-row{display:flex;align-items:baseline;gap:8px;margin-top:4px}
.stat-num{font-family:var(--mono);font-size:28px;font-weight:600;color:var(--ink);line-height:1.1}
.stat-cap{display:block;margin-top:4px;font-size:11.5px;color:var(--muted)}
.dchip{display:inline-flex;align-items:center;gap:4px;height:18px;padding:0 7px;border-radius:999px;font-family:var(--mono);font-size:11px;font-weight:600;line-height:1;white-space:nowrap;border:1px solid transparent;transform:translateY(-2px)}
.dchip.bad{background:var(--danger-soft);border-color:var(--danger-border);color:var(--danger)}
.dchip.good{background:var(--ok-soft);border-color:var(--ok-border);color:var(--ok)}
.dchip.neutral{background:var(--sunken);border-color:var(--hairline);color:var(--muted)}
.dchip svg.down{transform:rotate(180deg)}
/* By-severity bars */
.sevbar{display:flex;align-items:center;gap:12px}
.sevbar .sb-label{width:72px;font-family:var(--mono);font-size:11px;font-weight:500;letter-spacing:0.06em;text-transform:uppercase;color:var(--muted)}
.sevbar .sb-track{flex:1;height:8px;border-radius:999px;background:var(--sunken);overflow:hidden}
.sevbar .sb-fill{display:block;height:100%;border-radius:999px}
.sevbar .sb-count{width:26px;text-align:right;font-family:var(--mono);font-size:12.5px;color:var(--body)}
/* Coverage census meters */
.covmeter{display:flex;flex-direction:column;gap:6px}
.covmeter-head{display:flex;align-items:baseline;gap:8px}
.covmeter-head .cm-count{margin-left:auto;font-family:var(--mono);font-size:11.5px;color:var(--body);white-space:nowrap}
.covmeter-bar{height:4px;border-radius:999px;background:repeating-linear-gradient(45deg,var(--accent-soft) 0 5px,var(--sunken) 5px 10px)}
.covmeter-detail{font-size:11px;color:var(--muted);white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.cov-stale{display:flex;align-items:center;gap:8px}
.stale-badge{display:inline-flex;align-items:center;gap:4px;height:18px;padding:0 6px;border-radius:var(--r-sm);background:var(--stale-bg);border:1px solid var(--stale-border);color:var(--stale-fg);font-family:var(--mono);font-size:10px;font-weight:600;letter-spacing:0.04em;white-space:nowrap;line-height:1}
.stale-badge svg{width:9px;height:9px;flex:none}
/* Vantage rows + latency reading + availability badge */
.vrow{display:flex;align-items:center;gap:10px}
.vrow .vname{font-family:var(--mono);font-size:12.5px;color:var(--body)}
.vlat{font-family:var(--mono);font-size:12px;color:var(--muted)}
.avbadge{display:inline-flex;align-items:center;gap:4px;height:18px;padding:0 6px;border-radius:var(--r-sm);font-family:var(--mono);font-size:10px;font-weight:600;letter-spacing:0.04em;white-space:nowrap;line-height:1;border:1px solid transparent}
.avbadge .av-dot{width:5px;height:5px;border-radius:999px;flex:none}
.avbadge.available{background:var(--ok-soft);border-color:var(--ok-border);color:var(--ok)}
.avbadge.available .av-dot{background:var(--ok)}
.avbadge.unavailable{background:var(--danger-soft);border-color:var(--danger-border);color:var(--danger)}
.avbadge.unavailable .av-dot{background:var(--danger)}
.avbadge.unverified{background:transparent;border:1px dashed var(--border-strong);color:var(--muted)}
.avbadge.unverified .av-dot{background:var(--border-strong)}
/* Most-recent Signals register (flat per-instance, whole-row deep-link) */
.dsig-head,.dsig-row{display:grid;grid-template-columns:110px 1.4fr 1fr 70px 64px;align-items:center;gap:12px}
.dsig-head{padding:0 0 var(--space-3);border-bottom:1px solid var(--border-strong)}
.dsig-head span{font-family:var(--mono);font-size:10px;font-weight:600;text-transform:uppercase;letter-spacing:0.06em;color:var(--muted)}
.dsig-head .r,.dsig-row .r{text-align:right}
.dsig-row{padding:10px 0;border-bottom:1px solid var(--hairline);text-decoration:none;color:var(--body)}
.dsig-row:last-child{border-bottom:none}
.dsig-row:hover{background:var(--sunken);text-decoration:none}
.dsig-row .dsig-title{font-size:13px;font-weight:500;color:var(--ink);white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.dsig-row .mono{font-family:var(--mono);font-size:12px}
</style>
<main style="display:flex;flex-direction:column;gap:var(--space-5)">

<header class="dhead">
  <div style="display:flex;flex-direction:column;gap:2px">
    <h1 style="margin:0;font-size:21px">Dashboard</h1>
    <span class="dsub">{{with .ScanSchedule}}{{if .HasLast}}Last full scan <span class="dsub-v">{{.LastAgo}}</span> ago{{else}}No full scan yet{{end}}{{if .HasNext}} &#183; next in <span class="dsub-v">{{.NextIn}}</span>{{end}}{{end}}</span>
  </div>
  <div style="margin-left:auto;display:flex;gap:var(--space-2)">
    <a class="btn secondary dbtn-ico" href="/scope"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"><path d="M12 5v14M5 12h14"/></svg>Add seed</a>
    {{if .Scanning}}<button class="btn dbtn-ico" disabled><svg viewBox="0 0 24 24" fill="currentColor" stroke="none"><path d="M8 5v14l11-7z"/></svg>Scan running</button>{{else}}<a class="btn dbtn-ico" href="/scans"><svg viewBox="0 0 24 24" fill="currentColor" stroke="none"><path d="M8 5v14l11-7z"/></svg>Run scan</a>{{end}}
  </div>
</header>

{{if .Scanning}}
<div class="dprog">
  <div class="dprog-head"><span class="microlabel">Scan running</span><span class="dprog-detail">{{.ActiveScans}} scan{{if ne .ActiveScans 1}}s{{end}} in flight</span></div>
  <div class="dprog-track"><span></span></div>
</div>
{{end}}

{{if and .Unavailable (not .ProbeDismissed)}}
<div class="banner warn dbanner">
  <div class="banner-body">
    <div style="font-weight:600;color:var(--ink)">Vantage unreachable</div>
    <div style="font-size:12.5px;color:var(--body)">{{range $i, $n := .Unavailable}}{{if $i}}, {{end}}<span class="mono">{{$n}}</span>{{end}} could not be reached. Scans continue from your other vantages.</div>
  </div>
  <span class="banner-act"><a class="btn secondary" href="/scans" style="padding:5px 12px;font-size:12px;text-decoration:none">Retry now</a></span>
  <a class="banner-x" href="/?probe=dismissed" aria-label="Dismiss">&#215;</a>
</div>
{{end}}

<div class="statband">
  <div class="statband-grid">
    {{range .StatBand}}
    <div class="statcell">
      <span class="stat-label">{{.Label}}{{if .Live}}<span class="stat-live"></span>{{end}}</span>
      <span class="stat-row">
        <span class="stat-num">{{.Value}}</span>
        {{if .HasDelta}}<span class="dchip {{.Tone}}">{{if ne .Change 0}}<svg class="{{if lt .Change 0}}down{{end}}" viewBox="0 0 10 10" width="8" height="8" aria-hidden="true"><path d="M5 8.5V1.5M1.8 4.7L5 1.5l3.2 3.2" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/></svg>{{end}}{{signDelta .Change}}</span>{{end}}
      </span>
      <span class="stat-cap">{{.Caption}}</span>
    </div>
    {{end}}
  </div>
</div>

<div style="display:grid;grid-template-columns:380px 1fr;gap:var(--space-5);align-items:start">
  <div style="display:flex;flex-direction:column;gap:var(--space-5)">

    <section class="section" style="margin-bottom:0">
      <div style="display:flex;flex-direction:column;gap:3px;margin-bottom:var(--space-4)">
        <span class="microlabel">Open signals</span>
        <h2 style="margin:0;font-size:15px">By severity</h2>
      </div>
      {{if .HasSignals}}
      <div style="display:flex;flex-direction:column;gap:12px">
        {{range .SevBars}}
        <div class="sevbar">
          <span class="sb-label">{{.Sev}}</span>
          <span class="sb-track"><span class="sb-fill" style="width:{{.Pct}}%;background:var(--sev-{{.Sev}}-dot)"></span></span>
          <span class="sb-count">{{.Count}}</span>
        </div>
        {{end}}
      </div>
      {{else}}
      <div class="emptystate">
        <div class="microlabel">Unavailable</div>
        <h2>Severity could not be read</h2>
        <p style="max-width:60ch;margin:var(--space-3) auto">The signal census did not resolve on this load. Open Signals for the live ramp.</p>
        <a class="btn ghost" href="/signals">Go to Signals</a>
      </div>
      {{end}}
    </section>

    <section class="section" style="margin-bottom:0">
      <div style="display:flex;flex-direction:column;gap:3px;margin-bottom:var(--space-4)">
        <span class="microlabel">Coverage</span>
        <h2 style="margin:0;font-size:15px">Did we look, how completely</h2>
      </div>
      {{if .CoverageMeters}}
      <div style="display:flex;flex-direction:column;gap:16px">
        {{range .CoverageMeters}}
        <div class="covmeter">
          <div class="covmeter-head"><span class="microlabel">{{.Label}}</span><span class="cm-count">census &#183; {{.Counted}}{{if .Unit}} {{.Unit}}{{end}}</span></div>
          <div class="covmeter-bar" aria-label="Census — no denominator"></div>
          {{if .Detail}}<div class="covmeter-detail">{{.Detail}}</div>{{end}}
        </div>
        {{end}}
        {{if .SilentVantage}}
        <div class="cov-stale">
          <span class="stale-badge"><svg viewBox="0 0 10 10" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="5" cy="5" r="3.6"/><path d="M5 3.2V5l1.4 1" stroke-linecap="round"/></svg>no reports</span>
          <span class="muted" style="font-size:11.5px">position <span class="mono">{{.SilentVantage}}</span> went silent</span>
        </div>
        {{end}}
      </div>
      {{else}}
      <div class="emptystate">
        <div class="microlabel">No scope declared</div>
        <h2>Nothing to census yet</h2>
        <p style="max-width:60ch;margin:var(--space-3) auto">A census counts what each declared scope looks at. Declare a seed on Scope, and its aperture appears here.</p>
        <a class="btn ghost" href="/scope">Go to Scope</a>
      </div>
      {{end}}
    </section>

    <section class="section" style="margin-bottom:0">
      <div style="display:flex;flex-direction:column;gap:3px;margin-bottom:var(--space-4)">
        <span class="microlabel">Vantages</span>
        <h2 style="margin:0;font-size:15px">Scan infrastructure</h2>
      </div>
      {{if .Vantages}}
      <div style="display:flex;flex-direction:column;gap:var(--space-3)">
        {{range .Vantages}}
        <div class="vrow">
          <span class="vname">{{.Name}}</span>
          <span class="vlat">{{if .Latency}}{{.Latency}}{{else}}&#8212;{{end}}</span>
          <span style="margin-left:auto">{{if eq .Avail "available"}}<span class="avbadge available"><span class="av-dot"></span>available</span>{{else if eq .Avail "unavailable"}}<span class="avbadge unavailable"><span class="av-dot"></span>unavailable</span>{{else}}<span class="avbadge unverified"><span class="av-dot"></span>{{if .Avail}}{{.Avail}}{{else}}unverified{{end}}</span>{{end}}</span>
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
        <span class="microlabel">Most recent</span>
        <h2 style="margin:0;font-size:15px">Signals</h2>
      </div>
      <a class="btn ghost" href="/signals" style="margin-left:auto">View all</a>
    </div>
    {{if .RecentSignals}}
    <div class="dsig-head"><span>Severity</span><span>Signal</span><span>Asset</span><span>Port</span><span class="r">Seen</span></div>
    {{range .RecentSignals}}
    <a class="dsig-row" href="/signals">
      <span><span class="sev sev-{{.Severity}}">{{if ne .Severity "critical"}}<span class="sev-dot"></span>{{end}}{{.Severity}}</span></span>
      <span class="dsig-title">{{.Title}}</span>
      <span class="mono">{{.Asset}}</span>
      <span class="mono">{{if .Port}}{{.Port}}{{else}}&#8212;{{end}}</span>
      <span class="mono r">{{.Seen}}</span>
    </a>
    {{end}}
    {{else if .HasSignals}}
    <div class="emptystate">
      <div class="microlabel">All quiet</div>
      <h2>No signals firing</h2>
      <p style="max-width:60ch;margin:var(--space-3) auto">No rule is firing on any subject right now. A signal appears here the moment the world moves your estate into a rule's population, and is withdrawn just as quietly when the world moves back.</p>
      <a class="btn ghost" href="/signals">Go to Signals</a>
    </div>
    {{else}}
    <div class="emptystate">
      <div class="microlabel">Unavailable</div>
      <h2>Signal register could not be read</h2>
      <p style="max-width:60ch;margin:var(--space-3) auto">The signal register did not resolve on this load. Open Signals for the live census.</p>
      <a class="btn ghost" href="/signals">Go to Signals</a>
    </div>
    {{end}}
  </section>
</div>

</main>
{{end}}
`
