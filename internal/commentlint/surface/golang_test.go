package surface

import (
	"strings"
	"testing"
)

func TestGoBlocks(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []wantBlock
	}{
		{
			name: "an own-line run is one block in declaration position",
			src: "package p\n" +
				"\n" +
				"// one\n" +
				"// two\n" +
				"func F() {}\n",
			want: []wantBlock{
				{startLine: 3, endLine: 4, style: StyleLine, declaration: true, text: "// one\n// two"},
			},
		},
		{
			name: "a blank line splits the run",
			src: "package p\n" +
				"\n" +
				"// one\n" +
				"\n" +
				"// two\n" +
				"func F() {}\n",
			want: []wantBlock{
				{startLine: 3, endLine: 3, style: StyleLine, text: "// one"},
				{startLine: 5, endLine: 5, style: StyleLine, declaration: true, text: "// two"},
			},
		},
		{
			name: "a style change splits the run",
			src: "package p\n" +
				"\n" +
				"// one\n" +
				"/* two */\n" +
				"func F() {}\n",
			want: []wantBlock{
				{startLine: 3, endLine: 3, style: StyleLine, declaration: true, text: "// one"},
				{startLine: 4, endLine: 4, style: StyleBlock, declaration: true, text: "/* two */"},
			},
		},
		{
			name: "a multi-line general comment is one block",
			src: "package p\n" +
				"\n" +
				"/* one\n" +
				"two */\n" +
				"func F() {}\n",
			want: []wantBlock{
				{startLine: 3, endLine: 4, style: StyleBlock, declaration: true, text: "/* one\ntwo */"},
			},
		},
		{
			name: "a trailing comment is never a block",
			src: "package p\n" +
				"\n" +
				"func F() { // opener\n" +
				"\t_ = 1 // note\n" +
				"}\n",
			want: nil,
		},
		{
			name: "code after a general comment makes it trailing",
			src: "package p\n" +
				"\n" +
				"func F() {\n" +
				"\t/* c */ _ = 1\n" +
				"}\n",
			want: nil,
		},
		{
			name: "a directive line is its own block and absorbs no prose",
			src: "// prose above\n" +
				"//go:build linux\n" +
				"// prose below\n" +
				"\n" +
				"package p\n",
			want: []wantBlock{
				{startLine: 1, endLine: 1, style: StyleLine, text: "// prose above"},
				{startLine: 2, endLine: 2, style: StyleLine, directive: true, text: "//go:build linux"},
				{startLine: 3, endLine: 3, style: StyleLine, text: "// prose below"},
			},
		},
		{
			name: "every protected pattern marks a directive",
			src: "package p\n" +
				"\n" +
				"// +build linux\n" +
				"\n" +
				"//nolint:gosec\n" +
				"\n" +
				"//lint:ignore SA1000 reason\n" +
				"\n" +
				"//revive:disable\n" +
				"\n" +
				"func F() {}\n",
			want: []wantBlock{
				{startLine: 3, endLine: 3, style: StyleLine, directive: true, text: "// +build linux"},
				{startLine: 5, endLine: 5, style: StyleLine, directive: true, text: "//nolint:gosec"},
				{startLine: 7, endLine: 7, style: StyleLine, directive: true, text: "//lint:ignore SA1000 reason"},
				{startLine: 9, endLine: 9, style: StyleLine, directive: true, text: "//revive:disable"},
			},
		},
		{
			name: "a package clause is declaration position",
			src: "// package prose\n" +
				"package p\n",
			want: []wantBlock{
				{startLine: 1, endLine: 1, style: StyleLine, declaration: true, text: "// package prose"},
			},
		},
		{
			name: "a struct field and a const member are declaration position",
			src: "package p\n" +
				"\n" +
				"const (\n" +
				"\t// member prose\n" +
				"\tA = 1\n" +
				")\n" +
				"\n" +
				"type T struct {\n" +
				"\t// field prose\n" +
				"\tF int\n" +
				"}\n",
			want: []wantBlock{
				{startLine: 4, endLine: 4, style: StyleLine, declaration: true, text: "// member prose"},
				{startLine: 9, endLine: 9, style: StyleLine, declaration: true, text: "// field prose"},
			},
		},
		{
			name: "a carriage-return file keeps the whole byte range",
			src: "package p\r\n" +
				"\r\n" +
				"/* one\r\n" +
				"two */\r\n" +
				"func F() {}\r\n",
			want: []wantBlock{
				{startLine: 3, endLine: 4, style: StyleBlock, declaration: true, text: "/* one\r\ntwo */"},
			},
		},
		{
			name: "a carriage-return run joins on the whole byte range",
			src: "package p\r\n" +
				"\r\n" +
				"// one\r\n" +
				"// two\r\n" +
				"func F() {}\r\n",
			want: []wantBlock{
				{startLine: 3, endLine: 4, style: StyleLine, declaration: true, text: "// one\r\n// two"},
			},
		},
		{
			name: "a block inside a body is not declaration position",
			src: "package p\n" +
				"\n" +
				"func F() {\n" +
				"\t// body prose\n" +
				"\t_ = 1\n" +
				"}\n",
			want: []wantBlock{
				{startLine: 4, endLine: 4, style: StyleLine, text: "// body prose"},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Go{}.Lex([]byte(c.src))
			if err != nil {
				t.Fatalf("Lex: %v", err)
			}
			checkBlocks(t, c.src, got.Blocks, c.want)
		})
	}
}

func TestGoSkeletonDropsCommentsAndKeepsDirectives(t *testing.T) {
	src := "//go:build linux\n" +
		"\n" +
		"package p\n" +
		"\n" +
		"// prose\n" +
		"func F() { //nolint:gosec\n" +
		"\t_ = 1 // note\n" +
		"}\n"

	got, err := Go{}.Lex([]byte(src))
	if err != nil {
		t.Fatalf("Lex: %v", err)
	}
	var comments []string
	for _, tok := range got.Skeleton {
		if tok.Kind == "COMMENT" {
			comments = append(comments, tok.Text)
		}
	}
	want := []string{"//go:build linux", "//nolint:gosec"}
	if len(comments) != len(want) {
		t.Fatalf("skeleton holds comments %q, want %q", comments, want)
	}
	for i := range want {
		if comments[i] != want[i] {
			t.Errorf("skeleton comment %d is %q, want %q", i, comments[i], want[i])
		}
	}
}

func TestGoSkeletonHoldsTheBuildConstraintSeparator(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want bool
	}{
		{"a blank line separates the constraint", "//go:build linux\n\npackage p\n", true},
		{"no blank line separates the constraint", "//go:build linux\npackage p\n", false},
		{"the old-style constraint counts too", "// +build linux\n\npackage p\n", true},
		{"a prose line does not separate it", "//go:build linux\n// prose\n\npackage p\n", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Go{}.Lex([]byte(c.src))
			if err != nil {
				t.Fatalf("Lex: %v", err)
			}
			found := false
			for _, tok := range got.Skeleton {
				if tok.Kind == BlankLine {
					found = true
				}
			}
			if found != c.want {
				t.Errorf("the skeleton holds a %s token %t, want %t: %v", BlankLine, found, c.want, got.Skeleton)
			}
		})
	}
}

func TestGoLineDirectiveDoesNotMoveTheReportedLine(t *testing.T) {
	src := "package p\n" +
		"\n" +
		"//line fake.go:100\n" +
		"\n" +
		"// prose\n" +
		"func F() {}\n"

	got, err := Go{}.Lex([]byte(src))
	if err != nil {
		t.Fatalf("Lex: %v", err)
	}
	last := got.Blocks[len(got.Blocks)-1]
	if last.StartLine != 5 {
		t.Errorf("the prose block reports line %d, want the physical line 5", last.StartLine)
	}
}

func TestGoLexRejectsUnparseableSource(t *testing.T) {
	_, err := Go{}.Lex([]byte("package p\n\nfunc F() {\n"))
	if err == nil {
		t.Fatal("Lex accepted unparseable source")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error is %q, want a parse failure", err)
	}
}

func TestForRejectsAnUnsupportedSurface(t *testing.T) {
	for _, name := range []string{"design-system/examples/console/Coverage.html", "docs-site/src/pages/index.astro"} {
		if _, err := For(name); err == nil {
			t.Errorf("For accepted %s, which §6.4 excludes from every subcommand", name)
		}
	}
	for _, name := range []string{"cmd/web/main.go", "design-system/templates/shell.tmpl"} {
		if _, err := For(name); err != nil {
			t.Errorf("For rejected %s: %v", name, err)
		}
	}
}

func TestForDecidesByExtensionAndNotByContent(t *testing.T) {
	// §5.3: TypeScript generics look like JSX to a lexer, so the extension
	// decides and the content never does.
	cases := []struct {
		name string
		want Lexer
	}{
		{"cmd/web/main.go", Go{}},
		{"db/queries/scan.sql", SQL{}},
		{"db/migrations/00100_init.sql", SQL{}},
		{"design-system/tokens/colors.css", CSS{}},
		{"DESIGN-SYSTEM/TOKENS/COLORS.CSS", CSS{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := For(c.name)
			if err != nil {
				t.Fatalf("For: %v", err)
			}
			if got != c.want {
				t.Errorf("For chose %T, want %T", got, c.want)
			}
		})
	}
}
