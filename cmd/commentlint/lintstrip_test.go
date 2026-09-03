package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

func TestLintHumanOutputNamesFileLineAndRule(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "p.go", "package p\n\n// F reports the estate.\nfunc F() {}\n")

	var stdout, stderr bytes.Buffer
	got := runWith([]string{"lint", "p.go"}, &stdout, &stderr, stubGit(nil, nil))
	if got != 1 {
		t.Errorf("exit is %d, want 1 (stderr %q)", got, stderr.String())
	}
	if !strings.Contains(stdout.String(), "p.go:3 -> docstring-exported-conventional") {
		t.Errorf("stdout is %q, want a file:line -> rule line", stdout.String())
	}
}

func TestLintExitsZeroOnACleanFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "p.go", "package p\n\nfunc F() {}\n")

	var stdout, stderr bytes.Buffer
	if got := runWith([]string{"lint", "p.go"}, &stdout, &stderr, stubGit(nil, nil)); got != 0 {
		t.Errorf("exit is %d, want 0 (stdout %q, stderr %q)", got, stdout.String(), stderr.String())
	}
}

func TestLintWalksTheTreeWithNoPath(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "internal/p/a.go", "package p\n\n// F reports the estate.\nfunc F() {}\n")
	writeFile(t, dir, "internal/p/b.go", "package p\n\n// G reports the estate.\nfunc G() {}\n")

	var stdout, stderr bytes.Buffer
	runWith([]string{"lint"}, &stdout, &stderr, stubGit(nil, nil))
	for _, want := range []string{"internal/p/a.go:3", "internal/p/b.go:3"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("stdout is %q, want it to name %s", stdout.String(), want)
		}
	}
}

func TestLintInScopeOnlyNeverWalksTheTree(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "internal/p/a.go", "package p\n\n// F reports the estate.\nfunc F() {}\n")

	var stdout, stderr bytes.Buffer
	if got := runWith([]string{"lint", "--in-scope-only"}, &stdout, &stderr, stubGit(nil, nil)); got != 0 {
		t.Errorf("exit is %d, want 0", got)
	}
	if strings.Contains(stdout.String(), "internal/p/a.go") {
		t.Errorf("an empty changed set linted the tree: %q", stdout.String())
	}
}

func TestLintSkipsASurfaceWithNoLexerYet(t *testing.T) {
	// Every in-scope surface lexes now, so the skip belongs to a surface the
	// sweep never reads (#1142).
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "docs/spec/comment-policy.md", "# The comment policy\n")

	var stdout, stderr bytes.Buffer
	if got := runWith([]string{"lint", "docs/spec/comment-policy.md"}, &stdout, &stderr, stubGit(nil, nil)); got != 0 {
		t.Errorf("exit is %d, want 0 (stderr %q)", got, stderr.String())
	}
	if !strings.Contains(stdout.String(), "0 lex failure(s)") {
		t.Errorf("stdout is %q, want no lex failure for an unlexed surface", stdout.String())
	}
}

func TestLintReadsTheSQLAndCSSSurfaces(t *testing.T) {
	// §6.4 gives `lint` every lexable surface, so a divider in `db/queries` or
	// a token file reports like a Go one (#1140).
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "db/queries/scan.sql", "-- name: X :one\n-- ----------------\nSELECT 1;\n")
	writeFile(t, dir, "design-system/tokens/colors.css", "/* ---------------- */\na { color: red; }\n")

	var stdout, stderr bytes.Buffer
	got := runWith([]string{"lint", "db/queries/scan.sql", "design-system/tokens/colors.css"}, &stdout, &stderr, stubGit(nil, nil))
	if got != 1 {
		t.Fatalf("exit is %d, want 1 (stderr %q)", got, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"db/queries/scan.sql:2 -> section-divider", "design-system/tokens/colors.css:1 -> section-divider", "0 lex failure(s)"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout is %q, want it to hold %q", out, want)
		}
	}
}

func TestLintReadsTheTmplSurface(t *testing.T) {
	// §6.4 gives `lint` every lexable surface, and `.tmpl` is the last one to
	// arrive (#1142).
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "design-system/templates/shell.tmpl", "{{/* ---------------- */}}\n<p>{{.Title}}</p>\n")

	var stdout, stderr bytes.Buffer
	got := runWith([]string{"lint", "design-system/templates/shell.tmpl"}, &stdout, &stderr, stubGit(nil, nil))
	if got != 1 {
		t.Fatalf("exit is %d, want 1 (stderr %q)", got, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"design-system/templates/shell.tmpl:1 -> section-divider", "0 lex failure(s)"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout is %q, want it to hold %q", out, want)
		}
	}
}

func TestStripNamesTheTmplDeleteRule(t *testing.T) {
	// §6.5 records the `.tmpl` rule now, because the D3 sweep agent deletes by
	// hand and `strip` is the only place the rule reaches them (#1142).
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "design-system/templates/shell.tmpl", "{{/* the note */}}\n<p>x</p>\n")

	var stdout, stderr bytes.Buffer
	if got := runWith([]string{"strip", "design-system/templates/shell.tmpl"}, &stdout, &stderr, stubGit(nil, nil)); got != 2 {
		t.Fatalf("exit is %d, want 2", got)
	}
	if !strings.Contains(stderr.String(), "delete the comment's byte range, leave its line") {
		t.Errorf("stderr is %q, want it to name the §5.4 rule", stderr.String())
	}
}

func TestLintReadsTheJSAndTSSurfaces(t *testing.T) {
	// §6.4 gives `lint` every lexable surface. `.jsx` needs esbuild, so this
	// case stays on the two the hand lexer reads (#1141).
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "docs-site/scripts/doclint.mjs", "// ----------------\nexport const a = 1;\n")
	writeFile(t, dir, "design-system/components/display/Card.d.ts", "// ----------------\nexport type A = 1;\n")

	var stdout, stderr bytes.Buffer
	paths := []string{"lint", "docs-site/scripts/doclint.mjs", "design-system/components/display/Card.d.ts"}
	if got := runWith(paths, &stdout, &stderr, stubGit(nil, nil)); got != 1 {
		t.Fatalf("exit is %d, want 1 (stderr %q)", got, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"docs-site/scripts/doclint.mjs:1 -> section-divider",
		"design-system/components/display/Card.d.ts:1 -> section-divider",
		"0 lex failure(s)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout is %q, want it to hold %q", out, want)
		}
	}
}

func TestLintFailsClosedOnAJSXFileWithoutEsbuild(t *testing.T) {
	// §6.7: a lex failure means the tool cannot judge the file, and a missing
	// esbuild is exactly that case (#1141).
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "docs-site/src/ds/Icon.jsx", "const a = <p>x</p>;\n")

	var stdout, stderr bytes.Buffer
	if got := runWith([]string{"lint", "docs-site/src/ds/Icon.jsx"}, &stdout, &stderr, stubGit(nil, nil)); got != 2 {
		t.Errorf("exit is %d, want 2 (stderr %q)", got, stderr.String())
	}
	if !strings.Contains(stderr.String(), "esbuild") {
		t.Errorf("stderr is %q, want it to name the missing install", stderr.String())
	}
}

func TestGithubSummaryReportsLexFailuresOnTheirOwnLine(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "broken.go", "package p\n\nfunc F() {\n")

	var stdout, stderr bytes.Buffer
	if got := runWith([]string{"lint", "--github", "broken.go"}, &stdout, &stderr, stubGit(nil, nil)); got != 2 {
		t.Errorf("exit is %d, want 2 on a lex failure", got)
	}
	if !strings.Contains(stdout.String(), "**Lex failures: 1.**") {
		t.Errorf("summary is %q, want a lex-failure line of its own", stdout.String())
	}
	if strings.Contains(stdout.String(), "1 violation(s)") {
		t.Error("the summary merged the lex failure into the violation count")
	}
}

func TestGithubModeAnnotatesAndExitsZeroOnAViolation(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "p.go", "package p\n\n// F reports the estate.\nfunc F() {}\n")

	var stdout, stderr bytes.Buffer
	if got := runWith([]string{"lint", "--github", "p.go"}, &stdout, &stderr, stubGit(nil, nil)); got != 0 {
		t.Errorf("exit is %d, want 0 in --github mode", got)
	}
	if !strings.Contains(stdout.String(), "::warning file=p.go,line=3,title=commentlint (docstring-exported-conventional)::") {
		t.Errorf("stdout is %q, want one annotation per violation", stdout.String())
	}
}

func TestStripRefusesANonGoPath(t *testing.T) {
	// §6.4 keeps `strip` on Go alone, even now that every surface lexes
	// (#1140, #1141).
	cases := map[string]string{
		"db/queries/scan.sql":                        "-- name: X :one\nSELECT 1;\n",
		"design-system/tokens/colors.css":            "/* the azure scale */\na { color: red; }\n",
		"docs-site/scripts/doclint.mjs":              "// the note\nexport const a = 1;\n",
		"design-system/components/display/Card.d.ts": "// the note\nexport type A = 1;\n",
		"docs-site/src/ds/Icon.jsx":                  "// the note\nconst a = <p>x</p>;\n",
		"design-system/templates/shell.tmpl":         "{{/* the note */}}\n<p>x</p>\n",
	}
	for path, src := range cases {
		t.Run(path, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeFile(t, dir, path, src)

			var stdout, stderr bytes.Buffer
			if got := runWith([]string{"strip", path}, &stdout, &stderr, stubGit(nil, nil)); got != 2 {
				t.Errorf("exit is %d, want 2 on a non-Go path", got)
			}
			if !strings.Contains(stderr.String(), "Go surface only") {
				t.Errorf("stderr is %q, want it to name the surface rule", stderr.String())
			}
		})
	}
}

func TestStripChangesNoFileWithoutWrite(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	src := "package p\n\n// F reports the estate.\nfunc F() {}\n"
	p := writeFile(t, dir, "p.go", src)

	var stdout, stderr bytes.Buffer
	if got := runWith([]string{"strip", "p.go"}, &stdout, &stderr, stubGit(nil, nil)); got != 0 {
		t.Errorf("exit is %d, want 0 (stderr %q)", got, stderr.String())
	}
	after, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(after) != src {
		t.Errorf("the file changed without --write: %q", after)
	}
	if _, err := os.Stat(filepath.Join(dir, ".commentlint")); !os.IsNotExist(err) {
		t.Error("a dry run wrote the manifest directory")
	}
}

func TestStripWriteDeletesAndSavesTheManifest(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	p := writeFile(t, dir, "p.go", "// Package p serves the estate.\n"+
		"package p\n"+
		"\n"+
		"// F reports the estate.\n"+
		"func F() {}\n"+
		"\n"+
		"// G reports the estate (ADR-0127).\n"+
		"func G() {}\n")

	var stdout, stderr bytes.Buffer
	if got := runWith([]string{"strip", "--write", "p.go"}, &stdout, &stderr, stubGit(nil, nil)); got != 0 {
		t.Fatalf("exit is %d, want 0 (stderr %q)", got, stderr.String())
	}

	after, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if strings.Contains(string(after), "// F reports") {
		t.Error("the marker-free docstring survived --write")
	}
	if !strings.Contains(string(after), "// G reports the estate (ADR-0127).") {
		t.Error("the cited docstring was deleted")
	}
	if !strings.Contains(string(after), "// Package p serves the estate.") {
		t.Error("the package doc was deleted")
	}

	manifest, err := os.ReadFile(filepath.Join(dir, ".commentlint", "residue.jsonl"))
	if err != nil {
		t.Fatalf("read the manifest: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(manifest)), "\n")
	if len(lines) != 2 {
		t.Fatalf("the manifest holds %d line(s), want 2: %q", len(lines), manifest)
	}
	if !strings.Contains(lines[0], `"class":"package-doc"`) {
		t.Errorf("manifest line 1 is %q, want the package doc", lines[0])
	}
	if !strings.Contains(lines[1], `"signal":"citation"`) {
		t.Errorf("manifest line 2 is %q, want the citation signal", lines[1])
	}
}

func TestStripRefusesAManifestPathWithoutWrite(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "p.go", "package p\n\n// F reports the estate.\nfunc F() {}\n")

	var stdout, stderr bytes.Buffer
	got := runWith([]string{"strip", "--manifest", "residue.jsonl", "p.go"}, &stdout, &stderr, stubGit(nil, nil))
	if got != 2 {
		t.Errorf("exit is %d, want 2 on --manifest without --write", got)
	}
	if !strings.Contains(stderr.String(), "--manifest needs --write") {
		t.Errorf("stderr is %q, want it to name the missing flag", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "residue.jsonl")); !os.IsNotExist(err) {
		t.Error("the refused run wrote the named manifest")
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout is %q, want nothing on a usage error", stdout.String())
	}
}

func TestStripNeedsAPath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := runWith([]string{"strip"}, &stdout, &stderr, stubGit(nil, nil)); got != 2 {
		t.Errorf("exit is %d, want 2", got)
	}
}
