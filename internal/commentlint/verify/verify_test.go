package verify

import (
	"fmt"
	"strings"
	"testing"
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
			path: "db/queries/scans.sql",
			base: "-- name: One :one\nSELECT 1;\n",
			head: "SELECT 1;\n",
			want: LexFailed, wantExit: 2,
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

func TestChangedDetailNamesTheHeadLine(t *testing.T) {
	base := map[string]string{"a.go": "package p\n\nfunc F() int {\n\treturn 1\n}\n"}
	head := map[string]string{"a.go": "package p\n\nfunc F() int {\n\treturn 2\n}\n"}
	report := Run([]string{"a.go"}, false, source(base), source(head))
	detail := report.Findings[0].Detail
	if !strings.HasPrefix(detail, "line 4:") {
		t.Errorf("detail is %q, want it to open with the head line", detail)
	}
}
