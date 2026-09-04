package main

import (
	"html/template"

	designfs "github.com/winniel123/verge-asm/design-system"
)

// The delivered email and PDF execute this same "artifactdoc" define, so an edit moves them too.

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/reportartifact.tmpl"))
