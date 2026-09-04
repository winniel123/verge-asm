package main

import (
	"html/template"
	"io/fs"
	"sort"
	"strings"

	designfs "github.com/winniel123/verge-asm/design-system"
)

// Inventory screen — canonical `/inventory`. Under WORKFLOW v4 the repo no longer
// authors this template: the design package ships the frozen, design-owned
// inventory.tmpl (embedded read-only via the designfs package), and the app parses
// it straight into the shared `tmpl` set rather than restyling a repo copy. The tmpl
// carries all three definitions the old repo-authored const string did — `inventory`,
// the shared `recordrows` partial, and the `subject-missing` page — so the parse
// replaces templates_inventory.go's former `inventoryTemplates` const wholesale. The
// tmpl's `{{template "head"/"chrome"/"foot" .}}` calls resolve against the same shared
// shell blocks defined in templates_shell.go. The data holes match the
// inventorySubject/inventoryFacet structs emitted by buildInventory in inventory.go.
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/inventory.tmpl"))

// The concatenation is deliberately simple and deterministic — sorted filename order,
// joined by "\n", with no transformation — so a separate golden-render tool can
// reproduce the exact same string byte-for-byte. fs.Glob already returns sorted
// results; the explicit sort is belt-and-suspenders against a future FS whose Glob
// does not.
var designTokensCSS = loadDesignTokens()

func loadDesignTokens() string {
	names, err := fs.Glob(designfs.FS, "tokens/*.css")
	if err != nil {
		panic("web: glob design tokens: " + err.Error())
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		b, err := fs.ReadFile(designfs.FS, name)
		if err != nil {
			panic("web: read design token " + name + ": " + err.Error())
		}
		parts = append(parts, string(b))
	}
	return strings.Join(parts, "\n")
}
