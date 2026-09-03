package main

import (
	"html/template"

	designfs "github.com/winniel123/verge-asm/design-system"
)

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/settings.tmpl"))
