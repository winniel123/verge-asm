package surface

import (
	"errors"
	"strings"
	"testing"
)

func esbuildOrSkip(t *testing.T) {
	t.Helper()
	// The pinned CI container holds no node, so this test proves the mechanism
	// where docs-site is installed and stands aside where it is not (§6.1).
	if _, err := findEsbuild("jsx_test.go"); err != nil {
		t.Skip("docs-site holds no esbuild: run `npm ci` in docs-site")
	}
}

func jsxSkeleton(t *testing.T, src string) string {
	t.Helper()
	res, err := JSX{Path: "jsx_test.jsx"}.Lex([]byte(src))
	if err != nil {
		t.Fatalf("Lex(%q): %v", src, err)
	}
	if len(res.Blocks) != 0 || len(res.Trailing) != 0 {
		t.Errorf("got %d own-line and %d trailing block(s), want none", len(res.Blocks), len(res.Trailing))
	}
	return renderSkeleton(res.Skeleton)
}

func TestJSXReadsTextHoldingALineMarker(t *testing.T) {
	// JSX text makes `//` literal, which is why no hand lexer reads this
	// surface (SPEC §5.3).
	esbuildOrSkip(t)
	got := jsxSkeleton(t, "const a = <p>see https://example.com now</p>;\n")
	if !strings.Contains(got, "https://example.com now") {
		t.Errorf("the skeleton is %s, want it to hold the JSX text", got)
	}
}

func TestJSXTextChangeMovesTheSkeleton(t *testing.T) {
	esbuildOrSkip(t)
	base := jsxSkeleton(t, "const a = <p>see https://example.com now</p>;\n")
	head := jsxSkeleton(t, "const a = <p>see https://example.org now</p>;\n")
	if base == head {
		t.Errorf("the skeleton is blind to a changed JSX text run")
	}
}

func TestJSXCommentDeletionLeavesTheSkeleton(t *testing.T) {
	cases := []struct {
		name  string
		with  string
		clean string
	}{
		{
			name:  "an own-line comment",
			with:  "// the note\nconst a = <p>x</p>;\n",
			clean: "const a = <p>x</p>;\n",
		},
		{
			name:  "a trailing comment",
			with:  "const a = <p>x</p>; // the note\n",
			clean: "const a = <p>x</p>;\n",
		},
		{
			name:  "a block comment",
			with:  "/* the note */\nconst a = <p>x</p>;\n",
			clean: "const a = <p>x</p>;\n",
		},
		{
			// esbuild's plain transform keeps a comment that opens an object
			// property, so the canonical form needs --minify-whitespace.
			name:  "a comment opening an object property",
			with:  "const a = {\n  // the note\n  b: 1,\n};\n",
			clean: "const a = {\n  b: 1,\n};\n",
		},
		{
			name:  "a comment opening an array item",
			with:  "const a = [\n  // the note\n  1,\n];\n",
			clean: "const a = [\n  1,\n];\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			esbuildOrSkip(t)
			if jsxSkeleton(t, c.with) != jsxSkeleton(t, c.clean) {
				t.Errorf("deleting a comment moved the skeleton")
			}
		})
	}
}

func TestJSXCodeMutationMovesTheSkeleton(t *testing.T) {
	esbuildOrSkip(t)
	base := jsxSkeleton(t, "const a = <p className=\"lead\">x</p>;\n")
	head := jsxSkeleton(t, "const a = <p className=\"body\">x</p>;\n")
	if base == head {
		t.Errorf("the skeleton is blind to a changed attribute")
	}
}

func TestJSXDeletingADirectiveMovesTheSkeleton(t *testing.T) {
	// esbuild strips every comment, so the canonical form alone would report a
	// deleted `eslint` line clean. §2.3 protects those 2 tree lines.
	esbuildOrSkip(t)
	with := "// eslint-disable-next-line no-console\nconsole.log(<p>x</p>);\n"
	clean := "console.log(<p>x</p>);\n"
	if jsxSkeleton(t, with) == jsxSkeleton(t, clean) {
		t.Errorf("the skeleton survived the directive's deletion")
	}
}

func TestJSXDirectiveScanReadsATrailingMarker(t *testing.T) {
	src := "const url = \"https://example.com\"; // eslint-disable-line no-undef\n"
	got := jsxDirectiveTokens([]byte(src))
	if len(got) != 1 {
		t.Fatalf("got %d directive token(s), want 1", len(got))
	}
	if got[0].Line != 1 || !strings.Contains(got[0].Text, "eslint-disable-line") {
		t.Errorf("the token is %v", got[0])
	}
}

func TestJSXDirectiveScanIgnoresAURL(t *testing.T) {
	// The scan tests every `//` in the line, so a URL reaches the marker test
	// and must fail it.
	if got := jsxDirectiveTokens([]byte("const url = \"https://example.com\";\n")); len(got) != 0 {
		t.Errorf("got %d directive token(s), want none", len(got))
	}
}

func TestJSXReportsAnUnreadableFile(t *testing.T) {
	esbuildOrSkip(t)
	if _, err := (JSX{Path: "jsx_test.jsx"}).Lex([]byte("const a = <p>x</q>;\n")); err == nil {
		t.Fatalf("Lex accepted unparseable JSX")
	}
}

func TestFindEsbuildNamesTheMissingInstall(t *testing.T) {
	// §6.7 fails `verify` closed here rather than reporting the file clean.
	if _, err := findEsbuild(t.TempDir() + "/a.jsx"); !errors.Is(err, ErrNoEsbuild) {
		t.Fatalf("findEsbuild returned %v, want ErrNoEsbuild", err)
	}
}
