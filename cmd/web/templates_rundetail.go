package main

import "html/template"

// Run detail — the per-run drill-in reached from a Drift "Batch detail" entry (the
// stable `/run/{id}` route T16 links to). Ported verbatim from
// design-system/examples/console/RunDetail.jsx (11-console.jpg): a breadcrumb +
// header carrying the batch status and a "Drift from this batch" button, then the
// four sections the spec names — the Stages pipeline (Stepper), the batch log
// (LogViewer, inverted ink), the Outcome ("what it produced") beside the run
// Parameters ("as configured") and the vantage-health list ("who looked"). The
// example's components are translated to template-local CSS within the existing
// token vocabulary (restyling, not authoring — ADR-0109); no design-system
// component is authored here.
//
// A "run" is one Dispatch — a fan-out of one Scan (scans.go, ADR-0041). Real
// data is wired off the Operational queue corpus the Scans monitor already reads
// (ListDispatchProgress + ListJobsForDispatch): the batch status and job counts,
// the stages folded from the dispatch's job kinds, the per-job log, the run
// parameters, and the vantage health folded from the jobs' vantages. Where a
// dispatch carries no jobs the section renders the design-system empty-state
// rather than fabricate one.
//
// Two holds against the mock, on the same footing as the Drift screen: the
// Outcome stats are the run's OWN job outcomes (completed / dead-lettered), never
// "transitions" or "new signals" — the dispatch corpus is barred from the
// comparison path by construction (ADR-0041), so it cannot honestly report drift
// or signal counts. And the vocabulary is held: it is a VANTAGE that looked, never
// a probe/scanner/agent. No value is invented.
var _ = template.Must(tmpl.Parse(runDetailTemplates))

const runDetailTemplates = `
{{define "run"}}{{template "head" .}}
{{template "chrome" .}}
<style>
.rcard{background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);display:flex;flex-direction:column;overflow:visible}
.rcard>header{display:flex;align-items:center;gap:12px;padding:var(--space-5) var(--space-5) 0}
.rcard>header .microlabel{margin:0}
.rcard>header h2{margin:4px 0 0;font-size:15px}
.rcard>.rcard-body{padding:var(--space-5)}
/* BatchStatus chip */
.rbatch{display:inline-flex;align-items:center;gap:6px;height:20px;padding:0 8px;border-radius:var(--r-sm);font-family:var(--mono);font-size:10.5px;font-weight:600;letter-spacing:0.04em;white-space:nowrap;border:1px solid transparent;line-height:1}
.rbatch .bdot{width:5px;height:5px;border-radius:var(--r-full);flex:none;background:currentColor}
.rbatch.complete{background:var(--ok-soft);border-color:var(--ok-border);color:var(--ok)}
.rbatch.running{background:var(--accent-soft);color:var(--link)}
.rbatch.running .bdot{background:var(--accent);animation:verge-pulse 1.6s infinite}
.rbatch.failed{background:var(--danger-soft);border-color:var(--danger-border);color:var(--danger)}
.rbatch .bscope{font-weight:400;opacity:0.8}
/* Stepper */
.rstep{display:flex;gap:14px}
.rstep .rail{display:flex;flex-direction:column;align-items:center;flex:none}
.rstep .num{display:inline-flex;align-items:center;justify-content:center;width:26px;height:26px;border-radius:var(--r-full);flex:none;background:var(--surface);border:1px solid var(--border-strong);font:600 11.5px var(--mono);color:var(--muted)}
.rstep .num.done{background:var(--accent);border-color:var(--accent);color:var(--on-accent)}
.rstep .num.current{border:2px solid var(--accent);color:var(--link)}
.rstep .conn{width:1.5px;flex:1;min-height:18px;background:var(--hairline);margin:4px 0}
.rstep .conn.done{background:var(--accent)}
.rstep .stbody{display:flex;flex-direction:column;gap:2px;flex:1;padding-top:3px;min-width:0}
.rstep.notlast .stbody{padding-bottom:20px}
.rstep .sttitle{font:500 13.5px var(--sans);color:var(--body)}
.rstep .sttitle.current{font-weight:600;color:var(--ink)}
.rstep .stdetail{font:400 12.5px/1.5 var(--sans);color:var(--muted)}
/* LogViewer — batch output on inverted ink */
.rlog{background:var(--inverted);border-radius:var(--r-md);overflow:hidden}
.rlog-head{display:flex;align-items:center;gap:8px;padding:8px 14px 0}
.rlog-head .lbl{font:500 10.5px var(--mono);letter-spacing:0.07em;text-transform:uppercase;color:var(--muted)}
.rlog-head .live{margin-left:auto;display:inline-flex;align-items:center;gap:6px;font:500 10px var(--mono);letter-spacing:0.07em;text-transform:uppercase;color:var(--accent)}
.rlog-head .live .ldot{width:6px;height:6px;border-radius:var(--r-full);background:var(--accent);animation:verge-pulse 1.6s infinite}
.rlog-body{height:300px;overflow-y:auto;overflow-x:hidden;padding:10px 14px 12px}
.rlog-line{display:flex;align-items:baseline;gap:10px;font:400 11.5px/1.7 var(--mono)}
.rlog-line .t{color:var(--muted);flex:none}
.rlog-line .x{color:var(--on-inverted);overflow-wrap:anywhere}
/* level pills carry their own AA-tuned bg+fg pair, so they read on the inverted panel in both themes */
.rlog-line .lvl{font:600 9.5px var(--mono);text-transform:uppercase;letter-spacing:0.06em;padding:0 5px;border-radius:var(--r-sm);flex:none;align-self:center}
.rlog-line .lvl.warn{background:var(--warn-soft);color:var(--warn);border:1px solid var(--warn-border)}
.rlog-line .lvl.error{background:var(--danger-soft);color:var(--danger);border:1px solid var(--danger-border)}
/* Stat pair */
.rstats{display:grid;grid-template-columns:1fr 1fr;gap:16px}
.rstat{display:flex;flex-direction:column;gap:4px;min-width:0}
.rstat .rstat-l{font:500 11px var(--mono);letter-spacing:0.07em;text-transform:uppercase;color:var(--muted)}
.rstat .rstat-v{font:600 28px var(--mono);color:var(--ink);line-height:1.1}
/* Callout (warn) */
.rcallout{display:flex;gap:10px;align-items:flex-start;padding:12px 14px;border-radius:var(--r-md);border:1px solid transparent}
.rcallout.warn{background:var(--warn-soft);border-color:var(--warn-border)}
.rcallout .ic{display:inline-flex;margin-top:1px;flex:none;color:var(--warn)}
.rcallout .ct{display:flex;flex-direction:column;gap:2px;min-width:0}
.rcallout .ct-title{font:600 13px var(--sans);color:var(--ink)}
.rcallout .ct-body{font:400 13px/1.55 var(--sans);color:var(--body)}
/* KeyValueList */
.rkv{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px 20px;padding:var(--space-4);background:var(--sunken);border-radius:var(--r-md)}
.rkv .k{font:500 11px var(--mono);letter-spacing:0.07em;text-transform:uppercase;color:var(--muted)}
.rkv .v{font:400 12.5px var(--mono);color:var(--body);overflow-wrap:anywhere}
/* Vantage health row */
.rvant{display:flex;align-items:center;gap:10px}
.rvant .rv-name{font:500 12.5px var(--mono);color:var(--ink)}
.rvant .rv-right{margin-left:auto;display:inline-flex;align-items:center;gap:8px}
.rvant .rv-lat{font:400 11.5px var(--mono);color:var(--muted)}
.rbadge{display:inline-flex;align-items:center;gap:6px;height:22px;padding:0 10px;border-radius:var(--r-full);font:500 12px var(--sans);white-space:nowrap;border:1px solid transparent}
.rbadge .rb-dot{width:6px;height:6px;border-radius:var(--r-full);background:currentColor}
.rbadge.ok{background:var(--ok-soft);border-color:var(--ok-border);color:var(--ok)}
.rbadge.degraded{background:var(--warn-soft);border-color:var(--warn-border);color:var(--warn)}
</style>
<main style="display:flex;flex-direction:column;gap:20px">
{{with .Run}}
<nav aria-label="Breadcrumb" class="microlabel" style="display:flex;align-items:center;gap:8px">
<a href="/drift" style="color:var(--muted);text-decoration:none">Drift</a>
<span aria-hidden="true" style="color:var(--muted)">/</span>
<span aria-current="page" style="color:var(--body)">batch {{.Title}}</span>
</nav>

<header style="display:flex;align-items:flex-start;gap:16px;flex-wrap:wrap">
<div style="display:flex;flex-direction:column;gap:8px">
<h1 class="mono" style="margin:0;font-size:21px;letter-spacing:-0.01em;color:var(--ink)">{{.Title}}</h1>
<div style="display:flex;align-items:center;gap:10px;flex-wrap:wrap">
<span class="rbatch {{.Status}}"><span class="bdot"></span>{{.Status}}{{if .Scope}}<span class="bscope">· {{.Scope}}</span>{{end}}</span>
<span class="mono muted" style="font-size:12px">{{.Meta}}</span>
</div>
</div>
<div style="margin-left:auto">
<a class="btn secondary" href="/drift" style="display:inline-flex;align-items:center;gap:6px;text-decoration:none"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><line x1="6" y1="3" x2="6" y2="15"></line><circle cx="18" cy="6" r="3"></circle><circle cx="6" cy="18" r="3"></circle><path d="M18 9a9 9 0 0 1-9 9"></path></svg>Drift from this batch</a>
</div>
</header>

<section class="rcard">
<header><div><div class="microlabel">Pipeline</div><h2>Stages</h2></div></header>
<div class="rcard-body">
{{if .Stages}}
<div>
{{range .Stages}}
<div class="rstep{{if not .Last}} notlast{{end}}">
<div class="rail">
<span class="num{{if .Done}} done{{else if .Current}} current{{end}}">{{if .Done}}<svg viewBox="0 0 18 18" width="12" height="12"><path d="M3.5 9.5l3.5 3.5 7.5-8" fill="none" stroke="var(--on-accent)" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"></path></svg>{{else}}{{.Num}}{{end}}</span>
{{if not .Last}}<span class="conn{{if .Done}} done{{end}}"></span>{{end}}
</div>
<div class="stbody">
<span class="sttitle{{if .Current}} current{{end}}">{{.Title}}</span>
{{if .Detail}}<span class="stdetail">{{.Detail}}</span>{{end}}
</div>
</div>
{{end}}
</div>
{{else}}
<div class="emptystate"><h2>No stage to show</h2><p>This run enqueued no job, so its pipeline has no stage to trace. A stage appears here for each kind of work the fan-out enqueued.</p></div>
{{end}}
</div>
</section>

<div style="display:grid;grid-template-columns:minmax(0,1fr) 340px;gap:24px;align-items:start">

{{if .Log}}
<div class="rlog">
<div class="rlog-head">
<span class="lbl">batch {{.Title}}</span>
{{if .Active}}<span class="live"><span class="ldot"></span>streaming</span>{{end}}
</div>
<div class="rlog-body">
{{range .Log}}
<div class="rlog-line"><span class="t">{{.Tag}}</span>{{if .Level}}<span class="lvl {{.Level}}">{{.Level}}</span>{{end}}<span class="x">{{.Text}}</span></div>
{{end}}
</div>
</div>
{{else}}
<div class="emptystate"><h2>No log to show</h2><p>This run recorded no per-job event yet. Each job the fan-out enqueued appears here with its kind, live state and the vantage it ran at as the worker moves it.</p></div>
{{end}}

<div style="display:flex;flex-direction:column;gap:24px">

<section class="rcard">
<header><div><div class="microlabel">Outcome</div><h2>What it produced</h2></div></header>
<div class="rcard-body">
<div style="display:flex;flex-direction:column;gap:16px">
<div class="rstats">
<div class="rstat"><span class="rstat-l">Completed</span><span class="rstat-v">{{.Completed}}</span></div>
<div class="rstat"><span class="rstat-l">Dead-lettered</span><span class="rstat-v">{{.Dead}}</span></div>
</div>
{{if .Degraded}}
<div class="rcallout warn">
<span class="ic"><svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path><line x1="12" y1="9" x2="12" y2="13"></line><line x1="12" y1="17" x2="12.01" y2="17"></line></svg></span>
<span class="ct"><span class="ct-title">One vantage degraded</span><span class="ct-body">{{.Degraded}} did not complete every job in this run — exposure conclusions from it are marked unverified for this batch.</span></span>
</div>
{{end}}
</div>
</div>
</section>

<section class="rcard">
<header><div><div class="microlabel">Parameters</div><h2>As configured</h2></div></header>
<div class="rcard-body">
<div class="rkv">
{{range .Params}}<div style="display:flex;flex-direction:column;gap:3px;min-width:0"><span class="k">{{.K}}</span><span class="v">{{.V}}</span></div>{{end}}
</div>
</div>
</section>

<section class="rcard">
<header><div><div class="microlabel">Vantages</div><h2>Who looked</h2></div></header>
<div class="rcard-body">
{{if .Vantages}}
<div style="display:flex;flex-direction:column;gap:10px">
{{range .Vantages}}
<div class="rvant">
<span class="rv-name">{{.Name}}</span>
<span class="rv-right">
<span class="rv-lat">{{.Latency}}</span>
<span class="rbadge {{.Status}}"><span class="rb-dot"></span>{{.Status}}</span>
</span>
</div>
{{end}}
</div>
{{else}}
<div class="emptystate"><h2>No vantage recorded</h2><p>No job in this run carried a vantage, so there is no per-vantage health to show. A vantage appears here once the fan-out runs a check from one.</p></div>
{{end}}
</div>
</section>

</div>

</div>
{{end}}
</main>
{{template "foot" .}}{{end}}

{{define "run-missing"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="microlabel">No such run</div>
<h1 class="mono">{{.Run}}</h1>
<div class="section">
<p>No dispatch is keyed under that id, or it has aged out of recent history. A run
detail is reached from a batch on Drift.</p>
<p><a href="/drift">Back to Drift</a></p>
</div>
</main>
{{template "foot" .}}{{end}}
`
