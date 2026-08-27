package main

import (
	"html/template"
	"strconv"

	designfs "github.com/winniel123/verge-asm/design-system"
)

// tmpl is the single template the whole console renders through. Under WORKFLOW v4
// the shell itself is now design-owned: the frozen design-system/templates/shell.tmpl
// (package v3.14.0, map #22, embedded read-only via designfs) carries the "head",
// "chrome", "foot" and "cmdkicon" definitions every screen renders through. Every
// per-screen file appends its own blocks to this same tmpl via
// `var _ = template.Must(tmpl.ParseFS(...))` (see templates_*.go).
//
// This is P4.4: the shell swap is ATOMIC. templates_shell.go's former inline "head"/
// "chrome"/"foot" defines AND its `pageCSS` const (the shell glue, the .DesignTokens
// opt-in gate, the data-design-shell bleed-isolation resets, and every legacy page
// class) are deleted wholesale in the same commit that wires the shell.tmpl ParseFS
// below — the old const defines and the new design shell cannot coexist serving the
// live shell (shell.tmpl redefines the exact "head"/"chrome"/"foot" names). The
// funcmap (integrationsEnabled, designTokens, signDelta) survives; designTokens() now
// serves EVERY page unconditionally (the head inlines it for all routes, not only the
// pages that once opted in via .DesignTokens). The repo-authored "scantrigger" define
// (scantrigger.go) stays repo-owned, restyled inline within the token vocabulary.
var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	// integrationsEnabled exposes the compile-time #388 flag (integrations.go) to
	// templates, so the shell's command palette and the Settings tab bar can gate
	// the hidden Integrations surface without threading a data field through every
	// handler that renders the shell.
	"integrationsEnabled": func() bool { return integrationsEnabled },
	// designTokens returns the design-owned CSS-token vocabulary (design-system/
	// tokens/*.css, loaded by loadDesignTokens in templates_inventory.go) as trusted
	// CSS. The shell.tmpl "head" block inlines it in a <style data-design-tokens>
	// block on EVERY page — the whole console is design-owned now, so there is no
	// pageCSS below it and no .DesignTokens gate. It is design-authored CSS from the
	// embedded package, no user input, so template.CSS keeps html/template from
	// escaping it.
	"designTokens": func() template.CSS { return template.CSS(designTokensCSS) }, // #nosec G203 -- design-owned CSS from the embedded design package (designfs), no user input
	// signDelta formats a vs-last-batch stat delta (drift.Delta.Change, P0.2 #443) as
	// the design's signed chip label: "+N" for a rise, "−N" (a true minus, the
	// voice's signed-delta rule — design-system README) for a fall, and "0" for no
	// movement. The output is digits and a sign only — safe to mark trusted so
	// html/template keeps the literal "+" rather than escaping it to an entity.
	"signDelta": func(n int) template.HTML {
		var s string
		switch {
		case n > 0:
			s = "+" + strconv.Itoa(n)
		case n < 0:
			s = "−" + strconv.Itoa(-n)
		default:
			s = "0"
		}
		return template.HTML(s) // #nosec G203 -- a sign and digits only, from an int; no user input
	},
}).Parse(``))

// The frozen design-owned shell — head / chrome / foot / cmdkicon — parsed into the
// shared set. This ParseFS IS the atomic swap (P4.4): it must land in the same commit
// that deleted the old inline defines + pageCSS above, because shell.tmpl redefines
// the exact "head"/"chrome"/"foot" names the app serves live. shell.tmpl auto-embeds
// through designfs's existing templates/*.tmpl glob, so no designfs.go change is needed.
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/shell.tmpl"))
