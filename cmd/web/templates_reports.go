package main

import (
	"html/template"

	designfs "github.com/winniel123/verge-asm/design-system"
)

// Reports + the schedule wizard — byte-served from the frozen
// design-system/templates/reports.tmpl (package v3.11.0, WORKFLOW v4, #588). The
// repo-authored markup/CSS/JS that used to live here is deleted; the handler
// (reports.go / reports_schedule.go) shapes its data to the tmpl's declared holes.
//
// reports.tmpl defines "reports" + "schedulewizard" + "deltachip" + "spark" +
// "barchart". "deltachip" is now design-owned HERE and is consumed by
// reportartifact.tmpl (screen 17 parses against this same set), so this parse must
// land before that screen's — it does, both being embedded into the one shared
// cmd/web template set `tmpl` (see auth.go).
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/reports.tmpl"))
