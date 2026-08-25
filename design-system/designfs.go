// Package designfs embeds the design-owned artifacts (templates, tokens,
// fixtures, and verify configs) that ship in the design package and exposes
// them as a read-only fs.FS.
//
// This package exists because //go:embed forbids parent (`..`) paths and the
// module has no root-level Go package, so a consumer two levels down such as
// cmd/web cannot embed design-system/templates/inventory.tmpl directly. The
// embed directive must live beside the files, so this file sits at the
// design-system root — beside, never inside, the design-owned subdirectories.
//
// These artifacts are DESIGN-OWNED and frozen (CLAUDE.md §Design-owned view
// layer, CI gate G1). This package only READS them; nothing here may author,
// edit, or reformat a templates/, tokens/, fixtures/, or verify/ file. This
// .go file is sibling glue, not a design-owned artifact, so G1 (which
// byte-compares templates/tokens/fixtures/verify/goldens against the package)
// does not cover it.
//
// The globs match only the data artifacts by extension so that Go source added
// later under verify/ (the render-goldens / capture harness) is never swept in.
package designfs

import (
	"embed"
	"io/fs"
)

//go:embed templates/*.tmpl tokens/*.css fixtures/*.json verify/*.json
var files embed.FS

// FS is the read-only tree of design-owned artifacts, rooted at the
// design-system directory. Paths are package-relative, e.g.
// "templates/inventory.tmpl" and "tokens/colors.css".
var FS fs.FS = files
