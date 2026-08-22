package main

import "html/template"

// Drift screen — canonical `/drift` (new; nav item 4 of 7, the product's thesis
// per ADR-0108/ADR-0110). No backing data-read exists yet, so T0 ships a
// design-system empty-state; the screen ticket (T-Drift) rewrites the body against
// examples/console/Drift.jsx (change timeline grouped by batch, diff views) and
// wires it to change history where it exists. Never fabricate data.
var _ = template.Must(tmpl.Parse(driftTemplates))

const driftTemplates = `
{{define "drift"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="microlabel">Derived · drift</div>
<h1>Drift</h1>
<p>Drift is what this product exists to watch: the change timeline over your estate,
grouped by batch, with each transition carried in its own change vocabulary
(appeared · revealed · withdrawn · descoped · returned · changed).</p>
<div class="emptystate">
<h2>No change to show yet</h2>
<p>Once two batches have folded a value for the same subject, their transitions
appear here. Declare a scope on Scope and let a scan run twice to begin measuring
change.</p>
</div>
</main>
{{template "foot" .}}{{end}}
`
