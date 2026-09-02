package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"

	designfs "github.com/winniel123/verge-asm/design-system"
)

// #1088: reports.go named --sunken and --hairline, which only the self-contained
// artifact stylesheet in internal/message/render.go defines. An undefined var()
// fails at computed-value time with no console error, so the heatmap's zero cells
// lost both fill and border and the trend gridlines vanished, silently, in both
// themes. This guard makes that class loud at test time.
var (
	cssComment     = regexp.MustCompile(`(?s)/\*.*?\*/`)
	cssVarDeclared = regexp.MustCompile(`(--[a-z0-9-]+)[ \t]*:`)
	cssVarUsed     = regexp.MustCompile(`var\((--[a-z0-9-]+)\)`)
)

// Scoped to what composes a served console page: the repo-authored Go in this
// package and the design templates it renders, both read through the same embed
// the server ships. Only a fully literal var() reference is decidable — a name
// assembled at render time (var(--sev-{{.Sev}}-dot)) or one carrying its own
// fallback (var(--x, y)) is deliberately out of scope.
func TestConsolePageStylesOnlyNameDefinedTokens(t *testing.T) {
	defined := map[string]bool{}
	tokenFiles, err := fs.Glob(designfs.FS, "tokens/*.css")
	if err != nil {
		t.Fatalf("glob tokens: %v", err)
	}
	if len(tokenFiles) == 0 {
		t.Fatal("designfs embeds no token files")
	}
	for _, f := range tokenFiles {
		b, err := fs.ReadFile(designfs.FS, f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for _, name := range declaredCSSVars(b) {
			defined[name] = true
		}
	}
	if !defined["--surface-sunken"] || !defined["--row-sep"] {
		t.Fatalf("token scrape found %d names but not the reports card's own; the extractor is wrong", len(defined))
	}

	tmplFiles, err := fs.Glob(designfs.FS, "templates/*.tmpl")
	if err != nil {
		t.Fatalf("glob templates: %v", err)
	}
	if len(tmplFiles) == 0 {
		t.Fatal("designfs embeds no templates")
	}
	for _, f := range tmplFiles {
		b, err := fs.ReadFile(designfs.FS, f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		checkStyleStrings(t, f, b, defined)
	}

	goFiles, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob *.go: %v", err)
	}
	if len(goFiles) == 0 {
		t.Fatal("no package sources to scan")
	}
	for _, f := range goFiles {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		checkStyleStrings(t, f, b, defined)
	}
}

// A commented-out declaration defines nothing, so strip before scraping.
func declaredCSSVars(b []byte) []string {
	src := cssComment.ReplaceAllString(string(b), " ")
	var names []string
	for _, m := range cssVarDeclared.FindAllStringSubmatch(src, -1) {
		names = append(names, m[1])
	}
	return names
}

func checkStyleStrings(t *testing.T, path string, b []byte, defined map[string]bool) {
	t.Helper()
	// A file carrying its own self-contained stylesheet resolves its own names.
	local := map[string]bool{}
	for _, name := range declaredCSSVars(b) {
		local[name] = true
	}
	var undefined []string
	seen := map[string]bool{}
	for _, m := range cssVarUsed.FindAllStringSubmatch(string(b), -1) {
		name := m[1]
		if defined[name] || local[name] || seen[name] {
			continue
		}
		seen[name] = true
		undefined = append(undefined, name)
	}
	sort.Strings(undefined)
	for _, name := range undefined {
		t.Errorf("%s: var(%s) names no token in design-system/tokens/ and the file does not define it", path, name)
	}
}
