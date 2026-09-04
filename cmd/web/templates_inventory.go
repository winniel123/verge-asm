package main

import (
	"html/template"
	"io/fs"
	"sort"
	"strings"

	designfs "github.com/winniel123/verge-asm/design-system"
)

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/inventory.tmpl"))

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
