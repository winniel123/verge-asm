package main

import (
	"html/template"

	designfs "github.com/winniel123/verge-asm/design-system"
)

// Settings screen — canonical `/settings`. The view layer is the design-owned,
// frozen settings.tmpl (design-system/templates/settings.tmpl, package v3.13.0),
// embedded read-only via the designfs package and parsed into the shared template
// set here. It defines "settings" + "forbidden" + one "settings-<tab>" per section
// (scans · vantages · sso · team · sessions · audit · sources · aperture · instance
// · channels · integrations · messages · delivery). The repo authors no markup, CSS
// or view-JS for /settings; the handlers wire real data into the tmpl's declared
// holes (settings.go and the per-section files).
//
// Two repo-owned defines the settings tmpl still calls stay repo-authored until
// their own map items: "scantrigger" (scantrigger.go — the admin on-demand trigger
// panel on the Scans tab) and the "integrationsEnabled" funcmap gate
// (templates_shell.go). This file only parses the frozen tmpl; it holds no template
// text of its own (WORKFLOW v4 — the old repo-authored consts are deleted).
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/settings.tmpl"))
