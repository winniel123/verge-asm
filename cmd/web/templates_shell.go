package main

import (
	"html/template"
	"strconv"

	designfs "github.com/winniel123/verge-asm/design-system"
)

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"integrationsEnabled": func() bool { return integrationsEnabled },
	"designTokens":        func() template.CSS { return template.CSS(designTokensCSS) }, // #nosec G203 -- design-owned CSS from the embedded design package (designfs), no user input
	"signDelta": func(n int) template.HTML {
		var s string
		switch {
		case n > 0:
			s = "+" + strconv.Itoa(n)
		case n < 0:
			// A true minus, not a hyphen, is the voice's signed-delta rule (design-system/README.md).
			s = "−" + strconv.Itoa(-n)
		default:
			s = "0"
		}
		return template.HTML(s) // #nosec G203 -- a sign and digits only, from an int; no user input
	},
}).Parse(``))

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/shell.tmpl"))
