package main

import "html/template"

// Inbox screen (#299, T4) — the V3 primary message surface at `/inbox`, the
// destination the shell bell deep-links to. Ported verbatim from
// design-system/examples/console/Inbox.jsx: the read/unread list (dot + class Tag +
// relative instant + headline) beside the per-message detail, the all/unread
// SegmentedControl filter, the mark-all-read control, and the per-mover jump link.
// Sample data is swapped for real Message rows of the same shape — the class
// vocabulary is the store's own (drift / coverage / clock). Where there are no
// messages the design-system inbox-zero empty-state renders; nothing is fabricated.
//
// The classes below are TEMPLATE-LOCAL CSS translated from the example's components
// (Card, MessageList, EmptyState, SegmentedControl, Tag, Button) within the existing
// token vocabulary — restyling, not authoring (ADR-0109). No design-system component
// is authored here. The React example's client state maps to server state: opening a
// message is an ?id link (marks read, like the port's open()/initialId), the filter
// is a query param, and mark-all-read is a POST returning to /inbox.
var _ = template.Must(tmpl.Parse(inboxTemplates))

const inboxTemplates = `
{{define "inbox"}}{{template "head" .}}
{{template "chrome" .}}
<style>
.ibx-main{display:flex;flex-direction:column;gap:20px}
.ibx-head{display:flex;align-items:center;gap:16px}
.ibx-title{margin:0;font:600 21px var(--sans);letter-spacing:-0.015em;color:var(--ink)}
.ibx-sub{font:400 12.5px var(--sans);color:var(--muted)}
.ibx-sub .n{font-family:var(--mono);font-size:12px}
.ibx-toolbar{margin-left:auto;display:inline-flex;gap:8px;align-items:center}
.ibx-ghost{display:inline-flex;align-items:center;height:30px;padding:0 12px;border:1px solid transparent;border-radius:var(--r-md);font:600 12px var(--sans);background:transparent;color:var(--muted);cursor:pointer}
.ibx-ghost:hover{background:var(--sunken);color:var(--body)}
.ibx-ghost:disabled{opacity:.45;cursor:default}
.ibx-ghost:disabled:hover{background:transparent;color:var(--muted)}
.ibx-seg{display:inline-flex;gap:2px;padding:2px;background:var(--sunken);border:1px solid var(--hairline);border-radius:10px}
.ibx-seg a{display:inline-flex;align-items:center;height:22px;padding:0 10px;border-radius:var(--r-sm);font:500 12px var(--sans);color:var(--muted);text-decoration:none;white-space:nowrap}
.ibx-seg a:hover{text-decoration:none;color:var(--body)}
.ibx-seg a.on{background:var(--surface);color:var(--ink);font-weight:600;box-shadow:var(--shadow-xs)}
.ibx-grid{display:grid;grid-template-columns:380px minmax(0,1fr);gap:24px;align-items:start}
.ibx-card{background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm)}
.ibx-list{display:flex;flex-direction:column;padding:10px}
.ibx-row{display:flex;align-items:flex-start;gap:10px;width:100%;padding:10px 8px;border-radius:var(--r-sm);text-decoration:none;background:transparent}
.ibx-row + .ibx-row{border-top:1px solid var(--hairline)}
.ibx-row:hover{background:var(--sunken);text-decoration:none}
.ibx-row.sel{background:var(--accent-soft)}
.ibx-dot{width:7px;height:7px;border-radius:999px;margin-top:5px;flex:none;background:transparent}
.ibx-dot.unread{background:var(--accent)}
.ibx-rowbody{display:flex;flex-direction:column;gap:3px;min-width:0;flex:1}
.ibx-rowtop{display:flex;align-items:center;gap:8px}
.ibx-tag{display:inline-flex;align-items:center;gap:6px;height:22px;padding:0 8px;border-radius:var(--r-sm);background:var(--sunken);border:1px solid var(--hairline);color:var(--muted);font-family:var(--mono);font-size:11.5px;white-space:nowrap}
.ibx-time{margin-left:auto;font:400 11px var(--mono);color:var(--muted);white-space:nowrap}
.ibx-text{font:400 12.5px/1.5 var(--sans);color:var(--body)}
.ibx-row.unread .ibx-text{font-weight:500}
.ibx-cardhead{display:flex;align-items:center;gap:12px;padding:20px 20px 0}
.ibx-cardtitle{margin:0;font:600 15px var(--sans);letter-spacing:-0.015em;color:var(--ink)}
.ibx-cardbody{padding:20px}
.ibx-detail{display:flex;flex-direction:column;gap:16px}
.ibx-meta{font:400 11.5px var(--mono);color:var(--muted)}
.ibx-actions{display:flex;gap:8px}
.ibx-btn{display:inline-flex;align-items:center;gap:6px;height:36px;padding:0 16px;border:1px solid transparent;border-radius:var(--r-md);font:600 13px var(--sans);background:var(--accent);color:var(--on-accent);text-decoration:none}
.ibx-btn:hover{background:var(--accent-hover);text-decoration:none}
.ibx-empty{display:flex;flex-direction:column;align-items:center;text-align:center;gap:6px;font-family:var(--sans)}
.ibx-emptyicon{display:inline-flex;align-items:center;justify-content:center;width:48px;height:48px;border-radius:999px;background:var(--accent-soft);border:1px solid var(--accent-soft);color:var(--accent);margin-bottom:8px}
.ibx-emptymsg{font:600 14px var(--sans);color:var(--ink)}
.ibx-emptydetail{font:400 13px/1.5 var(--sans);color:var(--muted);max-width:380px}
</style>
<main class="ibx-main">

<header class="ibx-head">
<div style="display:flex;flex-direction:column;gap:2px">
<h1 class="ibx-title">Inbox</h1>
<span class="ibx-sub">Everything Verge told you, by class. <span class="n">{{.Unread}}</span> unread.</span>
</div>
<span class="ibx-toolbar">
<form method="post" action="/messages/read-all" style="margin:0"><input type="hidden" name="return" value="/inbox">
<button type="submit" class="ibx-ghost"{{if not .Unread}} disabled{{end}}>Mark all read</button></form>
<span class="ibx-seg" role="radiogroup" aria-label="Filter">
<a class="{{if eq .Filter "all"}}on{{end}}" role="radio" aria-checked="{{if eq .Filter "all"}}true{{else}}false{{end}}" href="{{.AllHref}}">All</a>
<a class="{{if eq .Filter "unread"}}on{{end}}" role="radio" aria-checked="{{if eq .Filter "unread"}}true{{else}}false{{end}}" href="{{.UnreadHref}}">Unread</a>
</span>
</span>
</header>

<div class="ibx-grid">

<div class="ibx-card">
{{if .Messages}}
<div class="ibx-list">
{{range .Messages}}
<a class="ibx-row{{if not .Read}} unread{{end}}{{if .Selected}} sel{{end}}" href="/inbox?id={{.ID}}{{if eq $.Filter "unread"}}&filter=unread{{end}}">
<span class="ibx-dot{{if not .Read}} unread{{end}}"></span>
<span class="ibx-rowbody">
<span class="ibx-rowtop">
<span class="ibx-tag">{{.Class}}</span>
<span class="ibx-time" title="{{.Instant}}">{{.Rel}}</span>
</span>
<span class="ibx-text">{{.Headline}}</span>
</span>
</a>
{{end}}
</div>
{{else}}
<div class="ibx-cardbody"><div class="ibx-empty" style="padding:28px 16px">
<span class="ibx-emptyicon"><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M22 12h-6l-2 3h-4l-2-3H2"/><path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/></svg></span>
<span class="ibx-emptymsg">Nothing unread.</span>
<span class="ibx-emptydetail">New messages land here as batches conclude.</span>
</div></div>
{{end}}
</div>

{{with .Selected}}
<div class="ibx-card">
<header class="ibx-cardhead">
<div style="display:flex;flex-direction:column;gap:3px;min-width:0">
<span class="microlabel">{{.Class}}</span>
<h3 class="ibx-cardtitle">{{.Headline}}</h3>
</div>
<div style="margin-left:auto;display:flex;gap:8px;flex:none"><span class="ibx-tag">{{.Class}}</span></div>
</header>
<div class="ibx-cardbody">
<div class="ibx-detail">
<span class="ibx-meta">{{if .Rel}}{{.Rel}} ago{{end}}{{if .Instant}} · {{.Instant}}{{end}}</span>
{{if .Census}}
<ul class="msgcensus">
{{range .Census}}<li><span class="k">{{.Kind}}</span>{{if .Href}}<a href="{{.Href}}">{{.Key}}</a>{{else}}{{.Key}}{{end}}</li>{{end}}
</ul>
{{end}}
{{if .Deliveries}}
<ul class="msgdelivery">
{{range .Deliveries}}<li class="{{if .Failed}}delivery-failed{{end}}">
{{if .Failed}}<span class="badge off">undelivered</span> to <span class="mono">{{.ChannelHost}}</span> — this message could not be delivered, not that nothing fired{{if .LastError}} <span class="muted why" title="{{.LastError}}">(reason)</span>{{end}}
{{else}}<span class="muted">{{.State}} to <span class="mono">{{.ChannelHost}}</span></span>{{end}}
</li>{{end}}
</ul>
{{end}}
<div class="ibx-actions">
<a class="ibx-btn" href="{{.Href}}"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14M13 6l6 6-6 6"/></svg>{{.JumpLabel}}</a>
</div>
</div>
</div>
</div>
{{else}}
<div class="ibx-card">
<div class="ibx-cardbody"><div class="ibx-empty" style="padding:40px 16px">
<span class="ibx-emptyicon"><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 8v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8"/><path d="m3 8 2.5-5h13L21 8"/><path d="m3 8 9 6 9-6"/></svg></span>
<span class="ibx-emptymsg">No message selected.</span>
<span class="ibx-emptydetail">Pick a message on the left to read it here.</span>
</div></div>
</div>
{{end}}

</div>
</main>
{{template "foot" .}}{{end}}
`
