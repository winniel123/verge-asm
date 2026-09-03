package designfs

import (
	"io/fs"
	"os"
	"strings"
	"testing"
)

// The console group (#1085). A log or code panel reads as a terminal, and a
// terminal is dark in EVERY theme. --surface-inverted is not that: it means the
// opposite of the page ground, so it flips, and a console riding it turned
// off-white in dark mode. These tests pin both halves of the repair — the group
// does not flip, and the console surfaces read the group rather than the
// inverted token.
var consoleTokens = []string{
	"--console-surface",
	"--console-text",
	"--console-warn",
	"--console-danger",
	"--console-pill-warn-bg",
	"--console-pill-warn-border",
	"--console-pill-warn-solid",
	"--console-pill-warn-fg",
}

// Anchored per declaration rather than per file. A file-wide ban would fail the
// day settings.tmpl grows a tooltip or a bulk-actions bar, both of which are
// legitimate --surface-inverted callers.
var consoleDecls = []struct {
	file   string
	anchor string
	want   string
}{
	{"templates/rundetail.tmpl", ".rd-log {", "--console-surface"},
	{"templates/rundetail.tmpl", ".rd-line .x {", "--console-text"},
	{"templates/rundetail.tmpl", `<span class="msg" style="color:`, "--console-text"},
	{"templates/rundetail-raw.tmpl", ".rd-log {", "--console-surface"},
	{"templates/rundetail-raw.tmpl", ".rawline {", "--console-text"},
	{"templates/settings.tmpl", ".st-code {", "--console-surface"},
	{"templates/settings.tmpl", ".st-code code {", "--console-text"},
	{"templates/settings.tmpl", `<pre style="margin:0;padding:12px 14px;`, "--console-surface"},
	{"components/display/LogViewer.jsx", `<div style={{ background:`, "--console-surface"},
	{"components/display/CodeBlock.jsx", "style={{ position: \"relative\", background:", "--console-surface"},
	{"components/display/CodeBlock.jsx", "<code style=", "--console-text"},
	{"examples/console/Settings.jsx", `<pre style={{ margin: 0, padding: "12px 14px"`, "--console-surface"},
}

// Asserted from the other side so a future sweep cannot flatten the distinction
// by rewriting every caller at once.
var invertedKeepers = []string{
	"components/feedback/Tooltip.jsx",
	"components/feedback/BulkActionsBar.jsx",
}

// Every block form counts, not just the two the file happens to use today. A
// media-query dark form (as internal/message/render.go already ships) must not
// be able to reintroduce the flip behind the guard's back.
func TestConsoleTokensHoldOneValueEverywhereTheyAreDeclared(t *testing.T) {
	b, err := fs.ReadFile(FS, "tokens/colors.css")
	if err != nil {
		t.Fatalf("read tokens/colors.css: %v", err)
	}
	css := stripComments(string(b))

	for _, name := range consoleTokens {
		values := declaredValues(css, name)
		switch {
		case len(values) == 0:
			t.Errorf("colors.css declares no %s", name)
		case len(values) < 2:
			t.Errorf("%s is declared once; restate it in the dark block so the pin is visible", name)
		}
		for _, v := range values[1:] {
			if v != values[0] {
				t.Errorf("%s flips with the theme: %s then %s", name, values[0], v)
				break
			}
		}
	}
}

func TestConsoleDeclarationsReadTheConsoleGroup(t *testing.T) {
	for _, d := range consoleDecls {
		decl, ok := lineContaining(readArtifact(t, d.file), d.anchor)
		if !ok {
			t.Errorf("%s has no declaration anchored at %q", d.file, d.anchor)
			continue
		}
		if !strings.Contains(decl, d.want) {
			t.Errorf("%s %q does not read %s", d.file, d.anchor, d.want)
		}
		for _, tok := range []string{"--surface-inverted", "--text-on-inverted"} {
			if strings.Contains(decl, tok) {
				t.Errorf("%s %q reads %s; a console surface takes the console group", d.file, d.anchor, tok)
			}
		}
	}
}

// The warn pill sits ON the pinned ground, so it may not read a token that
// flips. --warn-soft is #2d2413 in dark, which is 1.07:1 against the ground.
func TestConsolePillReadsNoFlippingToken(t *testing.T) {
	decl, ok := lineContaining(readArtifact(t, "templates/rundetail.tmpl"), ".rd-rawlink {")
	if !ok {
		t.Fatal("rundetail.tmpl has no .rd-rawlink declaration")
	}
	for _, tok := range []string{"var(--warn)", "var(--warn-soft)", "var(--warn-border)", "var(--warn-solid)"} {
		if strings.Contains(decl, tok) {
			t.Errorf(".rd-rawlink reads %s, which flips against the pinned console ground", tok)
		}
	}
}

func TestFeedbackSurfacesKeepTheInvertedToken(t *testing.T) {
	for _, name := range invertedKeepers {
		src := readArtifact(t, name)
		if !strings.Contains(src, "--surface-inverted") {
			t.Errorf("%s no longer reads --surface-inverted; it must flip with the theme", name)
		}
		if strings.Contains(src, "--console-surface") {
			t.Errorf("%s reads --console-surface; it is not a console surface", name)
		}
	}
}

// The embedded FS carries only the data artifacts (*.tmpl, *.css, *.json), so
// the .jsx sources are read from disk beside this file.
func readArtifact(t *testing.T, name string) string {
	t.Helper()
	if b, err := fs.ReadFile(FS, name); err == nil {
		return string(b)
	}
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

func lineContaining(src, anchor string) (string, bool) {
	for _, line := range strings.Split(src, "\n") {
		if strings.Contains(line, anchor) {
			return line, true
		}
	}
	return "", false
}

func declaredValues(css, name string) []string {
	var out []string
	for _, decl := range strings.Split(blockBreaker.Replace(css), ";") {
		decl = strings.TrimSpace(decl)
		if !strings.HasPrefix(decl, name+":") {
			continue
		}
		out = append(out, strings.TrimSpace(strings.TrimPrefix(decl, name+":")))
	}
	return out
}

// A declaration is found by splitting on ";", so a selector and its brace glue
// themselves to the front of the first declaration inside the block. Break on
// the braces too, and the split stays block-form agnostic.
var blockBreaker = strings.NewReplacer("{", ";", "}", ";")

func stripComments(css string) string {
	var b strings.Builder
	for {
		i := strings.Index(css, "/*")
		if i < 0 {
			b.WriteString(css)
			return b.String()
		}
		b.WriteString(css[:i])
		j := strings.Index(css[i:], "*/")
		if j < 0 {
			return b.String()
		}
		css = css[i+j+2:]
	}
}
