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
	"--surface-console",
	"--text-on-console",
	"--warn-on-console",
	"--danger-on-console",
}

// The two callers that use --surface-inverted for its intended meaning. They are
// asserted from the other side so a future sweep cannot flatten the distinction
// by rewriting every caller at once.
var invertedKeepers = []string{
	"components/feedback/Tooltip.jsx",
	"components/feedback/BulkActionsBar.jsx",
}

var consoleSurfaces = []string{
	"templates/rundetail.tmpl",
	"templates/rundetail-raw.tmpl",
	"templates/settings.tmpl",
	"components/display/LogViewer.jsx",
	"components/display/CodeBlock.jsx",
	"examples/console/Settings.jsx",
}

func TestConsoleTokensHoldOneValueAcrossThemes(t *testing.T) {
	b, err := fs.ReadFile(FS, "tokens/colors.css")
	if err != nil {
		t.Fatalf("read tokens/colors.css: %v", err)
	}
	css := string(b)
	light := themeBlock(t, css, ":root")
	dark := themeBlock(t, css, `[data-theme="dark"]`)

	for _, name := range consoleTokens {
		lv, ok := declaredValue(light, name)
		if !ok {
			t.Errorf(":root declares no %s", name)
			continue
		}
		dv, ok := declaredValue(dark, name)
		if !ok {
			t.Errorf(`[data-theme="dark"] declares no %s`, name)
			continue
		}
		if lv != dv {
			t.Errorf("%s flips with the theme: light %s, dark %s", name, lv, dv)
		}
	}
}

func TestConsoleSurfacesDoNotReadTheInvertedToken(t *testing.T) {
	for _, name := range consoleSurfaces {
		src := readArtifact(t, name)
		for _, tok := range []string{"--surface-inverted", "--text-on-inverted"} {
			if strings.Contains(src, tok) {
				t.Errorf("%s reads %s; a console surface takes the console group", name, tok)
			}
		}
		if !strings.Contains(src, "--surface-console") {
			t.Errorf("%s declares no --surface-console ground", name)
		}
	}
}

func TestFeedbackSurfacesKeepTheInvertedToken(t *testing.T) {
	for _, name := range invertedKeepers {
		src := readArtifact(t, name)
		if !strings.Contains(src, "--surface-inverted") {
			t.Errorf("%s no longer reads --surface-inverted; it must flip with the theme", name)
		}
		if strings.Contains(src, "--surface-console") {
			t.Errorf("%s reads --surface-console; it is not a console surface", name)
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

func themeBlock(t *testing.T, css, selector string) string {
	t.Helper()
	open := "\n" + selector + " {"
	i := strings.Index(css, open)
	if i < 0 {
		t.Fatalf("colors.css has no %s block", selector)
	}
	rest := css[i+len(open):]
	j := strings.Index(rest, "\n}")
	if j < 0 {
		t.Fatalf("the %s block is unterminated", selector)
	}
	return stripComments(rest[:j])
}

// A declaration is found by splitting on ";", and a /* ... */ note glues itself
// to the front of the declaration that follows it. Drop the notes first.
func stripComments(block string) string {
	var b strings.Builder
	for {
		i := strings.Index(block, "/*")
		if i < 0 {
			b.WriteString(block)
			return b.String()
		}
		b.WriteString(block[:i])
		j := strings.Index(block[i:], "*/")
		if j < 0 {
			return b.String()
		}
		block = block[i+j+2:]
	}
}

func declaredValue(block, name string) (string, bool) {
	for _, decl := range strings.Split(block, ";") {
		decl = strings.TrimSpace(decl)
		if !strings.HasPrefix(decl, name+":") {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(decl, name+":")), true
	}
	return "", false
}
