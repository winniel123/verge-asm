package main

import (
	"html/template"

	designfs "github.com/winniel123/verge-asm/design-system"
)

// Report artifact — the delivered report rendered, reached from Reports' "view last
// delivery" at the stable `/reports/delivery` route. As of batch 5 · screen 17 this
// screen is byte-served from the frozen design-owned template
// design-system/templates/reportartifact.tmpl (package v3.11.0, WORKFLOW v4): the old
// repo-authored markup const is gone and the tmpl is parsed into the shared `tmpl` set
// here. Its page define "reportartifact" wires the holes .Heading .Period .ScheduleID
// (nullable — a gone schedule renders the disabled Edit-schedule treatment, #23h) and
// .Doc, and delegates the delivered document to the "artifactdoc" define. That define
// calls "deltachip" (reports.tmpl), "sevbadge" (signals.tmpl) and "changeglyph"
// (drift.tmpl) — all already parsed into this set — so it resolves at execute time.
//
// The document body itself now lives in "artifactdoc" (SPEC-CHANGE #23g), which
// internal/message.RenderArtifact also executes, so the on-screen page, the delivered
// email and the PDF form render one markup. The handler (reportDeliveryPage,
// reports.go) shapes .Doc via message.BuildArtifactDoc under the live backend, or serves
// the pinned fixtures.json slice under VERGE_DEV (reportartifactFixtureData).
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/reportartifact.tmpl"))
