package main

import "html/template"

// Onboarding wizard markup (#307, T12) — ported from
// design-system/examples/console/Onboarding.jsx: the Wizard dialog (title,
// description, numbered step progress, Back/Next footer with the per-step valid
// gate) over the four steps' content — TagInput (Seeds), RadioCards + CadenceSelect
// (Cadence), Input (Channel), and KeyValueList (Review). The app is server-rendered
// with no client runtime, so the controlled React state becomes a post-back form:
// the accumulated values ride hidden fields, Back/Next re-render, and completion
// posts to /onboarding/finish (the existing admin-only trigger). The example's
// components are translated to template-local CSS within the existing token
// vocabulary (restyling, not authoring — ADR-0109); no design-system component is
// authored here. See onboarding.go for the controlled flow and the real scan
// enqueue. The reference to tmpl orders its initialisation before this one.
var _ = template.Must(tmpl.Parse(onboardingTemplates))

const onboardingTemplates = `
{{define "onboarding"}}{{template "head" .}}
{{template "chrome" .}}
<style>
.ob-wrap{display:flex;justify-content:center}
.ob-panel{background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-xl);box-shadow:var(--shadow-lg);padding:var(--space-6);width:560px;max-width:100%}
.ob-title{font-size:18px;margin:0 0 4px;color:var(--ink);letter-spacing:-0.015em}
.ob-desc{margin:0 0 var(--space-5);color:var(--muted);font-size:13px}
.ob-steps{display:flex;align-items:center;gap:8px;margin-bottom:20px}
.ob-conn{flex:1;min-width:12px;height:1px;background:var(--border-strong)}
.ob-conn.lit{background:var(--accent);opacity:0.4}
.ob-step{display:inline-flex;align-items:center;gap:8px}
.ob-num{width:22px;height:22px;border-radius:var(--r-full);flex:none;display:inline-flex;align-items:center;justify-content:center;font:600 11px var(--mono);border:1px solid var(--border-strong);color:var(--muted);background:transparent}
.ob-num svg{width:12px;height:12px}
.ob-num.cur{border:1.5px solid var(--accent);color:var(--accent)}
.ob-num.done{background:var(--accent-soft);border:1px solid transparent;color:var(--accent)}
.ob-step .lbl{font:400 12.5px var(--sans);color:var(--muted);white-space:nowrap}
.ob-step.cur .lbl{font-weight:600;color:var(--ink)}
.ob-body{display:flex;flex-direction:column;gap:10px}
.ob-flabel{font:500 12.5px var(--sans);color:var(--body);display:block;margin-bottom:4px}
.ob-hint{font:400 12px/1.6 var(--sans);color:var(--muted)}
.ob-tags{display:flex;align-items:center;flex-wrap:wrap;gap:6px;min-height:36px;padding:5px 10px;background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-md)}
.ob-tags:focus-within{border-color:var(--focus);box-shadow:0 0 0 2px var(--surface),0 0 0 4px var(--focus)}
.ob-tag{display:inline-flex;align-items:center;gap:4px;font:400 12px var(--mono);background:var(--sunken);border:1px solid var(--hairline);border-radius:var(--r-sm);padding:2px 4px 2px 8px;color:var(--body)}
.ob-tag button{border:none;background:transparent;color:var(--muted);cursor:pointer;padding:0 2px;font:600 12px var(--mono);line-height:1}
.ob-tag button:hover{color:var(--danger)}
.ob-tags input{flex:1;min-width:80px;border:none;outline:none;background:transparent;font:400 12.5px var(--mono);color:var(--ink);padding:0}
.ob-tags input:focus{box-shadow:none;border:none}
.ob-cards{display:grid;grid-template-columns:1fr;gap:10px}
.ob-card{position:relative;display:flex;flex-direction:column;gap:6px;padding:12px 14px;background:var(--surface);border:1.5px solid var(--hairline);border-radius:var(--r-lg);cursor:pointer}
.ob-card.on{background:var(--accent-soft);border-color:var(--accent)}
.ob-card input{position:absolute;opacity:0;width:0;height:0}
.ob-card .top{display:flex;align-items:center;gap:8px}
.ob-card .radio{width:14px;height:14px;border-radius:var(--r-full);flex:none;border:1.5px solid var(--border-strong);background:var(--surface);box-sizing:border-box}
.ob-card.on .radio{border:4.5px solid var(--accent)}
.ob-card .ct{font:600 13px var(--sans);color:var(--ink)}
.ob-card .meta{margin-left:auto;font:400 10.5px var(--mono);color:var(--muted)}
.ob-card .cd{font:400 12px/1.5 var(--sans);color:var(--body)}
.ob-cad{display:flex;flex-direction:column;gap:8px;margin-top:8px}
.ob-cad label{margin-bottom:0}
.ob-cron{width:100%;height:34px;padding:0 10px;font:400 12px var(--mono)}
.ob-kv{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px 20px;padding:16px;background:var(--sunken);border-radius:var(--r-md)}
.ob-kv-item{display:flex;flex-direction:column;gap:3px;min-width:0}
.ob-kv-k{font:500 11px var(--mono);letter-spacing:0.07em;text-transform:uppercase;color:var(--muted)}
.ob-kv-v{font:400 12.5px var(--mono);color:var(--body);overflow-wrap:anywhere}
.ob-foot{display:flex;align-items:center;gap:var(--space-3);margin-top:var(--space-5)}
.ob-count{margin-right:auto;font:500 11px var(--mono);letter-spacing:0.06em;color:var(--muted)}
</style>
<main class="ob-wrap">
<section class="ob-panel">
<h1 class="ob-title">Set up this workspace</h1>
<p class="ob-desc">Three calls and the first scan runs.</p>

<div class="ob-steps">
{{range $i, $s := .Steps}}
{{if $i}}<span class="ob-conn{{if or $s.Done $s.Current}} lit{{end}}"></span>{{end}}
<span class="ob-step{{if $s.Current}} cur{{end}}">
<span class="ob-num{{if $s.Done}} done{{else if $s.Current}} cur{{end}}">{{if $s.Done}}<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6 9 17l-5-5"></path></svg>{{else}}{{$s.Num}}{{end}}</span>
<span class="lbl">{{$s.Title}}</span>
</span>
{{end}}
</div>

<form method="post" action="/onboarding">
<input type="hidden" name="step" value="{{.Step}}">
<input type="hidden" name="seeds" value="{{.SeedsField}}">
{{if ne .Step 1}}<input type="hidden" name="profile" value="{{.Profile}}">
<input type="hidden" name="cad" value="{{.Cad}}">
<input type="hidden" name="cron" value="{{.Cron}}">{{end}}
{{if ne .Step 2}}<input type="hidden" name="channel" value="{{.Channel}}">{{end}}

<div class="ob-body">
{{if eq .Step 0}}
<span class="microlabel">What you own</span>
<span class="ob-tags">
{{range .Seeds}}<span class="ob-tag">{{.}}<button type="submit" name="rm" value="{{.}}" aria-label="Remove {{.}}">&#215;</button></span>{{end}}
<input type="text" name="seedsadd" placeholder="acmecorp.io, 203.0.113.0/24" spellcheck="false" autocomplete="off" aria-label="Add a seed">
</span>
<span class="ob-hint">Domains or CIDR ranges. Discovery expands each seed into subjects &mdash; you never enumerate hosts by hand.</span>
{{end}}

{{if eq .Step 1}}
<div class="ob-cards" role="radiogroup" aria-label="Scan profile">
<label class="ob-card{{if eq .Profile "standard"}} on{{end}}">
<input type="radio" name="profile" value="standard"{{if eq .Profile "standard"}} checked{{end}}>
<span class="top"><span class="radio" aria-hidden="true"></span><span class="ct">Standard</span><span class="meta">default</span></span>
<span class="cd">Top 1,000 TCP ports, plus any port previously seen.</span>
</label>
<label class="ob-card{{if eq .Profile "passive"}} on{{end}}">
<input type="radio" name="profile" value="passive"{{if eq .Profile "passive"}} checked{{end}}>
<span class="top"><span class="radio" aria-hidden="true"></span><span class="ct">Passive only</span></span>
<span class="cd">Public datasets only &mdash; no active probing.</span>
</label>
</div>
<div class="ob-cad">
<label><span class="ob-flabel">Cadence</span>
<select name="cad">{{range .Cads}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Value}}</option>{{end}}</select>
</label>
{{if .Custom}}<input class="ob-cron" type="text" name="cron" value="{{.Cron}}" placeholder="0 8 * * 1" spellcheck="false" aria-label="Cron expression">{{end}}
</div>
{{end}}

{{if eq .Step 2}}
<label><span class="ob-flabel">Delivery URL (optional)</span>
<input type="text" name="channel" value="{{.Channel}}" placeholder="https://ops.example/hook" spellcheck="false" autocomplete="off">
</label>
<span class="ob-hint">Signal and drift summaries post here. Add more channels later in Settings.</span>
{{end}}

{{if eq .Step 3}}
<div class="ob-kv">
{{range .Review}}<div class="ob-kv-item"><span class="ob-kv-k">{{.K}}</span><span class="ob-kv-v">{{.V}}</span></div>{{end}}
</div>
{{end}}
</div>

<div class="ob-foot">
<span class="ob-count">{{.StepNum}} / {{.StepTotal}}</span>
{{if gt .Step 0}}<button type="submit" class="secondary" name="action" value="back">Back</button>{{end}}
{{if .Last}}<input type="hidden" name="kind" value="{{.Kind}}"><button type="submit" formaction="/onboarding/finish">Start first scan</button>{{else}}<button type="submit" name="action" value="next">Next</button>{{end}}
</div>
</form>
</section>
</main>
{{template "foot" .}}{{end}}
`
