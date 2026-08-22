package main

import "html/template"

// Graph screen — canonical `/graph` (new). No backing data-read exists yet, so T0
// ships a design-system empty-state; the screen ticket (T-Graph) rewrites the body
// against examples/console/GraphView.jsx (pan/zoom/minimap, node drawer) and wires
// it where a graph read exists. Never fabricate data.
var _ = template.Must(tmpl.Parse(graphTemplates))

const graphTemplates = `
{{define "graph"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="microlabel">Derived · graph</div>
<h1>Graph</h1>
<p>The estate as a graph — names, addresses, services and endpoints and the
citations that bind them, pannable and zoomable, with a drawer for any node.</p>
<div class="emptystate">
<h2>Nothing to plot yet</h2>
<p>The graph draws from the same subjects the inventory holds. Declare a scope on
Scope and let a scan measure a subject into the estate to populate it.</p>
</div>
</main>
{{template "foot" .}}{{end}}
`
