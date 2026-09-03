package verify

import (
	"fmt"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/commentlint/surface"
)

func source(files map[string]string) ReadFunc {
	return func(name string) ([]byte, error) {
		src, ok := files[name]
		if !ok {
			return nil, fmt.Errorf("path %s is absent", name)
		}
		return []byte(src), nil
	}
}

func TestRun(t *testing.T) {
	cases := []struct {
		name        string
		path        string
		base        string
		head        string
		inScopeOnly bool
		want        Status
		wantExit    int
	}{
		{
			name: "a comment-only deletion is clean",
			path: "internal/p/a.go",
			base: "package p\n" +
				"\n" +
				"// This prose says what the function returns.\n" +
				"func F() int {\n" +
				"\t// the counter starts at one\n" +
				"\treturn 1\n" +
				"}\n",
			head: "package p\n" +
				"\n" +
				"func F() int {\n" +
				"\treturn 1\n" +
				"}\n",
			want:     Clean,
			wantExit: 0,
		},
		{
			name: "an indentation change is clean",
			path: "internal/p/a.go",
			base: "package p\n\nfunc F() int {\nreturn 1\n}\n",
			head: "package p\n\nfunc F() int {\n\treturn 1\n}\n",
			want: Clean,
		},
		{
			name: "a moved literal is a violation",
			path: "internal/p/a.go",
			base: "package p\n\n// prose\nfunc F() int { return 1 }\n",
			head: "package p\n\nfunc F() int { return 2 }\n",
			want: Changed, wantExit: 1,
		},
		{
			name: "a renamed identifier is a violation",
			path: "internal/p/a.go",
			base: "package p\n\nfunc F() int { return 1 }\n",
			head: "package p\n\nfunc G() int { return 1 }\n",
			want: Changed, wantExit: 1,
		},
		{
			name: "deleting a go:build line is a violation",
			path: "internal/p/a.go",
			base: "//go:build linux\n\npackage p\n\nfunc F() {}\n",
			head: "package p\n\nfunc F() {}\n",
			want: Changed, wantExit: 1,
		},
		{
			name: "deleting a nolint directive is a violation",
			path: "internal/p/a.go",
			base: "package p\n\nfunc F() { //nolint:gosec\n}\n",
			head: "package p\n\nfunc F() {\n}\n",
			want: Changed, wantExit: 1,
		},
		{
			name: "deleting the blank line under a build constraint is a violation",
			path: "internal/p/a.go",
			base: "//go:build linux\n\npackage p\n\nfunc F() {}\n",
			head: "//go:build linux\npackage p\n\nfunc F() {}\n",
			want: Changed, wantExit: 1,
		},
		{
			name: "a file absent from the base ref is a violation",
			path: "internal/p/absent.go",
			head: "package p\n",
			want: Changed, wantExit: 1,
		},
		{
			name: "a file absent from the working tree is a violation",
			path: "internal/p/deleted.go",
			base: "package p\n",
			want: Changed, wantExit: 1,
		},
		{
			name: "unparseable head content is a lex failure",
			path: "internal/p/a.go",
			base: "package p\n\nfunc F() {}\n",
			head: "package p\n\nfunc F() {\n",
			want: LexFailed, wantExit: 2,
		},
		{
			name: "a surface with no lexer is a lex failure",
			path: "design-system/templates/shell.tmpl",
			base: "<p>{{ .Title }}</p>\n",
			head: "<p>{{ .Name }}</p>\n",
			want: LexFailed, wantExit: 2,
		},
		{
			name: "a .sql comment delete keeps the skeleton",
			path: "db/queries/scans.sql",
			base: "-- name: One :one\n-- why this exists\nSELECT 1;\n",
			head: "-- name: One :one\nSELECT 1;\n",
			want: Clean,
		},
		{
			name: "a deleted -- name: directive moves the skeleton",
			path: "db/queries/scans.sql",
			base: "-- name: One :one\nSELECT 1;\n",
			head: "SELECT 1;\n",
			want: Changed, wantExit: 1,
		},
		{
			name: "a .css comment delete keeps the skeleton",
			path: "design-system/tokens/colors.css",
			base: ":root {\n  /* the azure scale */\n  --primary-50: #effcff;\n}\n",
			head: ":root {\n  --primary-50: #effcff;\n}\n",
			want: Clean,
		},
		{
			name: "a deleted stylelint directive moves the skeleton",
			path: "design-system/tokens/colors.css",
			base: "/* stylelint-disable no-descending-specificity */\na { color: red; }\n",
			head: "a { color: red; }\n",
			want: Changed, wantExit: 1,
		},
		{
			name: "a changed .html file is refused",
			path: "design-system/preview/index.html",
			base: "<p>a</p>\n",
			head: "<p>b</p>\n",
			want: Refused, wantExit: 2,
		},
		{
			name: "a changed .astro file is refused",
			path: "docs-site/src/pages/index.astro",
			base: "<p>a</p>\n",
			head: "<p>b</p>\n",
			want: Refused, wantExit: 2,
		},
		{
			name:        "--in-scope-only drops a refused file",
			path:        "design-system/preview/index.html",
			base:        "<p>a</p>\n",
			head:        "<p>b</p>\n",
			inScopeOnly: true,
			want:        Skipped,
		},
		{
			name: "an out-of-scope path is skipped",
			path: "docs/spec/comment-policy.md",
			base: "# a\n",
			head: "# b\n",
			want: Skipped,
		},
		{
			name: "a generated sqlc file is skipped",
			path: "internal/db/scans.sql.go",
			base: "package db\n\nfunc F() int { return 1 }\n",
			head: "package db\n\nfunc F() int { return 2 }\n",
			want: Skipped,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base := map[string]string{}
			head := map[string]string{}
			if c.base != "" {
				base[c.path] = c.base
			}
			if c.head != "" {
				head[c.path] = c.head
			}
			report := Run([]string{c.path}, c.inScopeOnly, source(base), source(head))
			if len(report.Findings) != 1 {
				t.Fatalf("got %d findings, want 1", len(report.Findings))
			}
			if got := report.Findings[0].Status; got != c.want {
				t.Fatalf("status is %s (%s), want %s", got, report.Findings[0].Detail, c.want)
			}
			if got := report.Exit(); got != c.wantExit {
				t.Errorf("exit is %d, want %d", got, c.wantExit)
			}
		})
	}
}

func TestExitPrefersTheLexFailure(t *testing.T) {
	base := map[string]string{
		"a.go": "package p\n\nfunc F() int { return 1 }\n",
		"b.go": "package p\n\nfunc G() {}\n",
	}
	head := map[string]string{
		"a.go": "package p\n\nfunc F() int { return 2 }\n",
		"b.go": "package p\n\nfunc G() {\n",
	}
	report := Run([]string{"a.go", "b.go"}, false, source(base), source(head))
	if got := report.Count(Changed); got != 1 {
		t.Errorf("got %d violations, want 1", got)
	}
	if got := report.Count(LexFailed); got != 1 {
		t.Errorf("got %d lex failures, want 1", got)
	}
	if got := report.Exit(); got != 2 {
		t.Errorf("exit is %d, want 2", got)
	}
}

func TestRunReadsTheJSSurfaces(t *testing.T) {
	// §6.4 gives `verify` all nine lexable surfaces, so ruling 15 has no hole
	// where the sweep carries the most risk (#1141).
	cases := []struct {
		name string
		path string
		base string
		head string
		want Status
	}{
		{
			name: "an mjs comment deletion is clean",
			path: "docs-site/scripts/doclint.mjs",
			base: "// the note\nexport const a = 1;\n",
			head: "export const a = 1;\n",
			want: Clean,
		},
		{
			name: "an mjs code change is a violation",
			path: "docs-site/scripts/doclint.mjs",
			base: "export const a = 1;\n",
			head: "export const a = 2;\n",
			want: Changed,
		},
		{
			name: "a declaration file comment deletion is clean",
			path: "design-system/components/display/Card.d.ts",
			base: "export interface Props {\n  /** The visible label. */\n  label: string;\n}\n",
			head: "export interface Props {\n  label: string;\n}\n",
			want: Clean,
		},
		{
			// esbuild erases a `.d.ts` file to the empty string, so 109 of 109
			// code mutations are invisible to it (SPEC §5.3).
			name: "a declaration file code change is a violation",
			path: "design-system/components/display/Card.d.ts",
			base: "export interface Props {\n  label: string;\n}\n",
			head: "export interface Props {\n  label: number;\n}\n",
			want: Changed,
		},
		{
			name: "a ts comment deletion is clean",
			path: "docs-site/src/pipeline/nav-build.ts",
			base: "// the note\nexport const a: number = 1;\n",
			head: "export const a: number = 1;\n",
			want: Clean,
		},
		{
			name: "a ts type change is a violation",
			path: "docs-site/src/pipeline/nav-build.ts",
			base: "export const a: number = 1;\n",
			head: "export const a: string = 1;\n",
			want: Changed,
		},
		{
			name: "a deleted eslint directive is a violation",
			path: "docs-site/scripts/doclint.mjs",
			base: "// eslint-disable-next-line no-console\nconsole.log(1);\n",
			head: "console.log(1);\n",
			want: Changed,
		},
		{
			name: "an unreadable mjs file fails closed",
			path: "docs-site/scripts/doclint.mjs",
			base: "export const a = 1;\n",
			head: "export const a = `1;\n",
			want: LexFailed,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			report := Run([]string{c.path}, false,
				source(map[string]string{c.path: c.base}),
				source(map[string]string{c.path: c.head}))
			if got := report.Findings[0].Status; got != c.want {
				t.Fatalf("status is %s (%s), want %s", got, report.Findings[0].Detail, c.want)
			}
		})
	}
}

func TestRunReadsTheJSXSurface(t *testing.T) {
	const path = "docs-site/src/ds/Icon.jsx"
	// `.jsx` lexes through the node esbuild docs-site installs, so the case
	// stands aside where that install is absent (SPEC §6.1, #1141).
	if _, err := (surface.JSX{Path: path}).Lex([]byte("const a = 1;\n")); err != nil {
		t.Skip("docs-site holds no esbuild: run `npm ci` in docs-site")
	}
	cases := []struct {
		name string
		base string
		head string
		want Status
	}{
		{
			name: "a comment deletion is clean",
			base: "// the note\nconst a = <p>x</p>;\n",
			head: "const a = <p>x</p>;\n",
			want: Clean,
		},
		{
			name: "a changed attribute is a violation",
			base: "const a = <p className=\"lead\">x</p>;\n",
			head: "const a = <p className=\"body\">x</p>;\n",
			want: Changed,
		},
		{
			// JSX text makes `//` literal, so a hand lexer would read this
			// line as a comment and report the change clean (SPEC §5.3).
			name: "a changed url in JSX text is a violation",
			base: "const a = <p>see https://example.com</p>;\n",
			head: "const a = <p>see https://example.org</p>;\n",
			want: Changed,
		},
		{
			name: "unreadable JSX fails closed",
			base: "const a = <p>x</p>;\n",
			head: "const a = <p>x</q>;\n",
			want: LexFailed,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			report := Run([]string{path}, false,
				source(map[string]string{path: c.base}),
				source(map[string]string{path: c.head}))
			if got := report.Findings[0].Status; got != c.want {
				t.Fatalf("status is %s (%s), want %s", got, report.Findings[0].Detail, c.want)
			}
		})
	}
}

func TestChangedDetailNamesTheHeadLine(t *testing.T) {
	base := map[string]string{"a.go": "package p\n\nfunc F() int {\n\treturn 1\n}\n"}
	head := map[string]string{"a.go": "package p\n\nfunc F() int {\n\treturn 2\n}\n"}
	report := Run([]string{"a.go"}, false, source(base), source(head))
	detail := report.Findings[0].Detail
	if !strings.HasPrefix(detail, "line 4:") {
		t.Errorf("detail is %q, want it to open with the head line", detail)
	}
}
