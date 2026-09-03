package surface

import (
	"strings"
	"testing"
	"time"
)

func cssSkeleton(t *testing.T, src string) string {
	t.Helper()
	res, err := CSS{}.Lex([]byte(src))
	if err != nil {
		t.Fatalf("Lex(%q): %v", src, err)
	}
	return renderSkeleton(res.Skeleton)
}

func TestCSSTokenizesTheValueForms(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "a custom property is one identifier",
			src:  "--primary-50: #effcff;",
			want: `IDENT "--primary-50" OP ":" HASH "#effcff" OP ";"`,
		},
		{
			name: "a unit joins its number",
			src:  "a { width: 20px; }",
			want: `IDENT "a" OP "{" IDENT "width" OP ":" NUMBER "20px" OP ";" OP "}"`,
		},
		{
			name: "a separated unit is a different stream",
			src:  "a { width: 20 px; }",
			want: `IDENT "a" OP "{" IDENT "width" OP ":" NUMBER "20" IDENT "px" OP ";" OP "}"`,
		},
		{
			name: "a negative percentage is one number",
			src:  "transform: translate(-50%);",
			want: `IDENT "transform" OP ":" IDENT "translate" OP "(" NUMBER "-50%" OP ")" OP ";"`,
		},
		{
			name: "a subtraction keeps its operator",
			src:  "width: calc(100% - 20px);",
			want: `IDENT "width" OP ":" IDENT "calc" OP "(" NUMBER "100%" OP "-" NUMBER "20px" OP ")" OP ";"`,
		},
		{
			name: "an at-rule keeps its keyword",
			src:  `@import "./fonts.css";`,
			want: `AT "@import" STRING "\"./fonts.css\"" OP ";"`,
		},
		{
			name: "a comment inside a string is not a comment",
			src:  `content: "/* not a comment */";`,
			want: `IDENT "content" OP ":" STRING "\"/* not a comment */\"" OP ";"`,
		},
		{
			name: "an unquoted url body is one token",
			src:  "src: url(/fonts/a.woff2);",
			want: `IDENT "src" OP ":" URL "url(/fonts/a.woff2)" OP ";"`,
		},
		{
			name: "a quoted url tokenizes as a call",
			src:  `src: url("/fonts/a.woff2");`,
			want: `IDENT "src" OP ":" IDENT "url" OP "(" STRING "\"/fonts/a.woff2\"" OP ")" OP ";"`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := cssSkeleton(t, c.src); got != c.want {
				t.Errorf("skeleton is %s, want %s", got, c.want)
			}
		})
	}
}

// §5.1: a comment can act as a token separator, so removing one is not always
// a no-op.
func TestCSSSeparatingCommentMovesTheSkeleton(t *testing.T) {
	glued := cssSkeleton(t, "a{color:re/*c*/d}")
	stripped := cssSkeleton(t, "a{color:red}")
	if glued == stripped {
		t.Errorf("both forms share the skeleton %s", glued)
	}
}

func TestCSSProtectsItsDirectives(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"a stylelint marker", "/* stylelint-disable no-descending-specificity */\na { color: red; }\n"},
		{"a postcss marker", "/* postcss-preset-env: ignore */\na { color: red; }\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := CSS{}.Lex([]byte(c.src))
			if err != nil {
				t.Fatalf("Lex: %v", err)
			}
			if len(res.Blocks) != 1 || !res.Blocks[0].Directive {
				t.Fatalf("got blocks %+v, want one directive block", res.Blocks)
			}
			with := renderSkeleton(res.Skeleton)
			without := cssSkeleton(t, "a { color: red; }\n")
			if with == without {
				t.Fatalf("deleting the directive left the skeleton at %s", with)
			}
			if !strings.HasPrefix(with, "DIRECTIVE ") {
				t.Errorf("skeleton is %s, want it to open with the directive", with)
			}
		})
	}
}

func TestCSSBlocks(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []wantBlock
	}{
		{
			name: "an own-line comment is a block",
			src:  ":root {\n  /* the azure scale */\n  --primary-50: #effcff;\n}\n",
			want: []wantBlock{
				{startLine: 2, endLine: 2, style: StyleBlock, text: "/* the azure scale */"},
			},
		},
		{
			name: "a multi-line comment is one block",
			src:  "/*\n * a header\n */\na { color: red; }\n",
			want: []wantBlock{
				{startLine: 1, endLine: 3, style: StyleBlock, text: "/*\n * a header\n */"},
			},
		},
		{
			name: "a trailing comment is never a block",
			src:  "a { color: red; } /* note */\n",
			want: nil,
		},
		{
			name: "code after a comment makes it trailing",
			src:  "/* note */ a { color: red; }\n",
			want: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := CSS{}.Lex([]byte(c.src))
			if err != nil {
				t.Fatalf("Lex: %v", err)
			}
			checkBlocks(t, c.src, res.Blocks, c.want)
		})
	}
}

func TestCSSPayloadDropsTheStarColumn(t *testing.T) {
	res, err := CSS{}.Lex([]byte("/*\n * one\n * two\n */\na { color: red; }\n"))
	if err != nil {
		t.Fatalf("Lex: %v", err)
	}
	if got := res.Blocks[0].Payload(); got != "one\ntwo" {
		t.Errorf("payload is %q, want %q", got, "one\ntwo")
	}
}

func TestCSSRejectsAnUnclosedString(t *testing.T) {
	if _, err := (CSS{}).Lex([]byte("content: \"forever\n")); err == nil {
		t.Fatal("Lex accepted a string that ends at the newline")
	}
}

func TestCSSEndsOnATrailingBackslash(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := (CSS{}).Lex([]byte("a { color: red; }\n\\")); err != nil {
			t.Errorf("Lex: %v", err)
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Lex never returned on a trailing backslash")
	}
}
