package designfs

import (
	"io/fs"
	"os"
	"strings"
	"testing"
)

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

var invertedKeepers = []string{
	"components/feedback/Tooltip.jsx",
	"components/feedback/BulkActionsBar.jsx",
}

func TestConsoleTokensHoldOneValueEverywhereTheyAreDeclared(t *testing.T) {
	// A console reads as a terminal and is dark in every theme, so its group must not flip (#1085).
	b, err := fs.ReadFile(FS, "tokens/colors.css")
	if err != nil {
		t.Fatalf("read tokens/colors.css: %v", err)
	}
	css := stripComments(string(b))

	for _, name := range consoleTokens {
		// Every block form counts, so a media-query dark form cannot reintroduce the flip.
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
	// A file-wide ban would fail the day settings.tmpl grows a legitimate --surface-inverted caller.
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

func TestConsolePillReadsNoFlippingToken(t *testing.T) {
	// The pill sits on the pinned ground, and --warn-soft is 1.07:1 against it in dark.
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
	// Asserted from the other side, so a sweep cannot rewrite every caller and flatten it.
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

func readArtifact(t *testing.T, name string) string {
	t.Helper()
	if b, err := fs.ReadFile(FS, name); err == nil {
		return string(b)
	}
	// The embed globs take only the data artifacts, so a .jsx source is read from disk.
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
