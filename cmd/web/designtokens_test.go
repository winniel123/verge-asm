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

// An undefined var() is invalid at computed-value time and logs nothing (#1088).

var (
	cssComment     = regexp.MustCompile(`(?s)/\*.*?\*/`)
	cssVarDeclared = regexp.MustCompile(`(--[a-z0-9-]+)[ \t]*:`)
	cssVarUsed     = regexp.MustCompile(`var\((--[a-z0-9-]+)\)`)
)

func TestConsolePageStylesOnlyNameDefinedTokens(t *testing.T) {
	// Only a literal var() is decidable; a render-time or own-fallback name is out of scope.
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
