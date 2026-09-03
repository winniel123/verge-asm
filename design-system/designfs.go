// Package designfs embeds the UI artifacts (templates, tokens, and fixtures)
// that the web app renders and exposes them as a read-only fs.FS.
//
// This package exists because //go:embed forbids parent (`..`) paths and the
// module has no root-level Go package, so a consumer two levels down such as
// cmd/web cannot embed design-system/templates/inventory.tmpl directly. The
// embed directive must live beside the files, so this file sits at the
// design-system root — beside, never inside, the artifact subdirectories.
//
// design-system/ is the shared UI asset home: these templates/tokens/fixtures
// are the source of truth for the served UI (this package embeds them) and the
// docs-site consumes tokens/ and components/ via its @ds alias. They may be
// edited in-repo (the design-system handoff workflow and its byte-compare
// gates were retired 2026-08-28). This .go file is sibling glue that only READS
// the artifacts.
//
// The globs match only the data artifacts by extension.
package designfs

import (
	"embed"
	"io/fs"
)

//go:embed templates/*.tmpl tokens/*.css fixtures/*.json
var files embed.FS

var FS fs.FS = files
