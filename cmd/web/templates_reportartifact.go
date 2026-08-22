package main

import "html/template"

// Report artifact — the delivered report rendered, reached from Reports' "view
// last delivery" at the stable `/reports/delivery` route (T17 links here). Ported
// from design-system/examples/console/ReportArtifact.jsx (10-console.jpg): a
// breadcrumb, a header (title, delivery window, download / edit controls), then
// the delivered document itself.
//
// The document body is NOT built here — it is rendered by internal/message's
// RenderArtifact, the one canonical rendered form that also serves as the
// PDF / email spec (the same markup is the delivered document). This page wraps
// that document in the console chrome. The header controls are inert: there is no
// report-scheduling or export backend yet (#285, #290/#291), so they render
// disabled rather than fabricating an action, and a schedule that has never
// delivered renders the design-system empty-state inside the document — never
// fabricated data (ADR-0110).
var _ = template.Must(tmpl.Parse(reportArtifactTemplates))

const reportArtifactTemplates = `
{{define "reportartifact"}}{{template "head" .}}
{{template "chrome" .}}
<main style="max-width:1440px;margin:0 auto;display:flex;flex-direction:column;gap:20px">

<nav aria-label="Breadcrumb" class="microlabel" style="display:flex;align-items:center;gap:8px">
<a href="/reports" style="color:var(--muted);text-decoration:none">Reports</a>
<span aria-hidden="true" style="color:var(--muted)">/</span>
<span aria-current="page" style="color:var(--body)">{{.Heading}}</span>
</nav>

<header style="display:flex;align-items:flex-start;gap:16px;flex-wrap:wrap">
<div style="display:flex;flex-direction:column;gap:6px">
<h1 style="margin:0;font-size:21px;letter-spacing:-0.01em;color:var(--ink)">{{.Heading}}</h1>
{{if .Period}}<span class="mono muted" style="font-size:12px">{{.Period}}</span>{{end}}
</div>
<div style="margin-left:auto;display:flex;gap:8px">
<button type="button" class="secondary" disabled title="Export is not wired yet" style="opacity:0.6;cursor:default">Download PDF</button>
<button type="button" class="btn ghost" disabled title="Report scheduling is not wired yet" style="opacity:0.6;cursor:default">Edit schedule</button>
</div>
</header>

{{.Body}}

</main>
{{template "foot" .}}{{end}}
`
