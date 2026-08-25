package main

import (
	"html/template"

	designfs "github.com/winniel123/verge-asm/design-system"
)

// The error page — 404 / 403 / 500 plus the three contextual states (U3/U4,
// #480/#481: missing-subject · missing-run · settings-forbidden). Under WORKFLOW v4
// (P4.0), the served markup/CSS/JS is the DESIGN-OWNED, frozen
// design-system/templates/error.tmpl (package v3.5.0), embedded read-only via the
// designfs package and parsed into the shared template set here. The repo authors no
// error markup: it only wires data into the holes the frozen tmpl declares
// (.Kind/.Code/.Subject/.IncidentID/.ActionLabel/.ActionHref/.Chrome — errors.go) and
// never edits the tmpl (CI gate G1 byte-compares it to the package). A needed change
// goes through SPEC-CHANGE and returns in the next package version.
//
// error.tmpl auto-embeds through designfs's existing `templates/*.tmpl` glob, so no
// designfs.go change is needed. Its "error-page" definition supersedes the repo-authored
// const that used to live here (deleted in E2, #533): the design tmpl is now the single
// source of the served error page. The handlers that feed it stay in errors.go.
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/error.tmpl"))
