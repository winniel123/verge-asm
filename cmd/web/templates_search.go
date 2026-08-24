package main

import "html/template"

// Search results markup — ported from
// design-system/examples/console/SearchResults.jsx (20-console.jpg). The layout is
// the example's: a centred 900px column, a query header (heading + mono input +
// mono result count), the design-system empty-state when nothing matches, then one
// Card per kind — Assets, Signals, Batches, Documentation — each a micro-label +
// title header over rows with the matched term highlighted. Signal rows lead with a
// SeverityBadge (the shared .sev pill; severity is P0.1's per-rule datum). The
// Documentation card is indexed over the embedded operator guides (docs/guides/);
// its rows are non-navigating, as the spec's doc rows carry no onClick. The
// example's Card / Tag / SeverityBadge / BatchStatus / EmptyState / Input / Icon are
// translated to template-local CSS within the existing token vocabulary (restyling,
// not authoring — ADR-0109); no design-system component is authored here. See
// search.go for the real-data reads and the P2.5 parity restorations.
var _ = template.Must(tmpl.Parse(searchTemplates))

const searchTemplates = `
{{define "search"}}{{template "head" .}}
{{template "chrome" .}}
<style>
.searchhead{display:flex;flex-direction:column;gap:12px}
.searchhead h1{margin:0;font:600 21px var(--sans);letter-spacing:-0.015em;color:var(--ink)}
.searchq{display:flex;align-items:center;gap:8px;height:36px;padding:0 12px;background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-md)}
.searchq:focus-within{border-color:var(--focus);box-shadow:0 0 0 2px var(--surface),0 0 0 4px var(--focus)}
.searchq input{flex:1;min-width:0;border:none;outline:none;background:transparent;color:var(--ink);font-family:var(--mono);font-size:12.5px;padding:0}
.searchq input:focus{box-shadow:none}
.searchcount{font:400 12px var(--mono);color:var(--muted)}
.searchform{margin:0}
/* Card (translated from Card.jsx, pad=10) */
.scard{background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);display:flex;flex-direction:column;overflow:visible}
.scard>header{display:flex;align-items:center;gap:12px;padding:10px 10px 0}
.scard>header .microlabel{margin:0}
.scard>header h3{margin:0;font:600 15px var(--sans);letter-spacing:-0.015em;color:var(--ink)}
.scard>.scard-body{padding:10px}
/* Result row — an anchor to the item's existing route */
.srow{display:flex;align-items:center;gap:10px;width:100%;padding:10px 8px;border-radius:var(--r-sm);text-decoration:none;color:inherit}
.srow:hover{background:var(--sunken);text-decoration:none}
.srow .sic{color:var(--muted);flex:none}
.srow .sname{font:500 12.5px var(--mono);color:var(--ink)}
.srow .stext{font:500 12.5px var(--sans);color:var(--ink);min-width:0}
.srow .sasset{margin-left:auto;font:400 11.5px var(--mono);color:var(--muted)}
.srow .schev{margin-left:auto;color:var(--muted);flex:none}
/* Documentation row — a two-line title + snippet column beside the file glyph */
.srow.sdocrow{cursor:default}
.srow .sdoc{display:flex;flex-direction:column;gap:2px;min-width:0}
.srow .sdoctitle{font:500 12.5px var(--sans);color:var(--ink)}
.srow .sdocsnip{font:400 11.5px var(--sans);color:var(--muted)}
/* Tag (translated from Tag.jsx) */
.stag{display:inline-flex;align-items:center;height:22px;padding:0 8px;border-radius:var(--r-sm);background:var(--sunken);border:1px solid var(--hairline);color:var(--muted);font:400 11.5px var(--mono);white-space:nowrap}
/* BatchStatus chip (translated from BatchStatus.jsx) */
.sbatch{display:inline-flex;align-items:center;gap:6px;height:20px;padding:0 8px;border-radius:var(--r-sm);font-family:var(--mono);font-size:10.5px;font-weight:600;letter-spacing:0.04em;white-space:nowrap;border:1px solid transparent;line-height:1}
.sbatch .bdot{width:5px;height:5px;border-radius:var(--r-full);flex:none;background:currentColor}
.sbatch.complete{background:var(--ok-soft);border-color:var(--ok-border);color:var(--ok)}
.sbatch.running{background:var(--accent-soft);color:var(--link)}
.sbatch.running .bdot{background:var(--accent);animation:verge-pulse 1.6s infinite}
.sbatch.failed{background:var(--danger-soft);border-color:var(--danger-border);color:var(--danger)}
.sbatch .bscope{font-weight:400;opacity:0.8}
/* EmptyState (translated from EmptyState.jsx) */
.sempty{display:flex;flex-direction:column;align-items:center;text-align:center;gap:6px;padding:48px 24px;font-family:var(--sans)}
.sempty .eic{display:inline-flex;align-items:center;justify-content:center;width:48px;height:48px;border-radius:var(--r-full);background:var(--accent-soft);border:1px solid var(--accent-soft);color:var(--accent);margin-bottom:8px}
.sempty .emsg{font:600 14px var(--sans);color:var(--ink)}
.sempty .edet{font:400 13px/1.5 var(--sans);color:var(--muted);max-width:380px}
</style>
<main style="max-width:900px;margin:0 auto;display:flex;flex-direction:column;gap:20px">
<header class="searchhead">
<h1>Search</h1>
<form class="searchform" method="get" action="/search"><span class="searchq"><svg class="sic" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="11" cy="11" r="8"></circle><path d="m21 21-4.3-4.3"></path></svg><input type="text" name="q" value="{{.Query}}" placeholder="Assets, signals, batches, docs" autofocus spellcheck="false" aria-label="Search"></span></form>
<span class="searchcount">{{.Total}} results{{if .Query}} for &ldquo;{{.Query}}&rdquo;{{end}}</span>
</header>

{{if eq .Total 0}}
<div class="sempty">
<span class="eic"><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="11" cy="11" r="8"></circle><path d="m21 21-4.3-4.3"></path></svg></span>
<span class="emsg">Nothing matches.</span>
<span class="edet">Try a hostname fragment, a signal phrase, or a batch timestamp.</span>
</div>
{{end}}

{{if .Assets}}
<section class="scard">
<header><div class="microlabel">{{len .Assets}} match{{if ne (len .Assets) 1}}es{{end}}</div><h3>Assets</h3></header>
<div class="scard-body">
{{range .Assets}}<a class="srow" href="{{.Href}}">
<svg class="sic" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect width="20" height="8" x="2" y="2" rx="2" ry="2"></rect><rect width="20" height="8" x="2" y="14" rx="2" ry="2"></rect><line x1="6" x2="6.01" y1="6" y2="6"></line><line x1="6" x2="6.01" y1="18" y2="18"></line></svg>
<span class="sname">{{.Name}}</span>
<span class="stag">{{.Type}}</span>
<svg class="schev" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m9 18 6-6-6-6"></path></svg>
</a>{{end}}
</div>
</section>
{{end}}

{{if .Signals}}
<section class="scard">
<header><div class="microlabel">{{len .Signals}} open</div><h3>Signals</h3></header>
<div class="scard-body">
{{range .Signals}}<a class="srow" href="{{.Href}}">
<span class="sev sev-{{.Severity}}">{{if ne .Severity "critical"}}<span class="sev-dot"></span>{{end}}{{.Severity}}</span>
<span class="stext">{{.Rule}}</span>
<span class="sasset">{{.Subject}}</span>
</a>{{end}}
</div>
</section>
{{end}}

{{if .Batches}}
<section class="scard">
<header><div class="microlabel">{{len .Batches}} recent</div><h3>Batches</h3></header>
<div class="scard-body">
{{range .Batches}}<a class="srow" href="{{.Href}}">
<span class="sbatch {{.Status}}"><span class="bdot"></span>{{.Status}}<span class="bscope">· {{.Label}}</span></span>
<svg class="schev" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m9 18 6-6-6-6"></path></svg>
</a>{{end}}
</div>
</section>
{{end}}

{{if .Docs}}
<section class="scard">
<header><div class="microlabel">Docs</div><h3>Documentation</h3></header>
<div class="scard-body">
{{range .Docs}}<div class="srow sdocrow">
<svg class="sic" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"></path><path d="M14 2v4a2 2 0 0 0 2 2h4"></path><path d="M10 9H8"></path><path d="M16 13H8"></path><path d="M16 17H8"></path></svg>
<span class="sdoc"><span class="sdoctitle">{{.Title}}</span><span class="sdocsnip">{{.Snip}}</span></span>
</div>{{end}}
</div>
</section>
{{end}}

</main>
{{template "foot" .}}{{end}}
`
