package surface

import (
	"strings"
	"testing"
)

func jsLex(t *testing.T, src string) Result {
	t.Helper()
	res, err := JS{}.Lex([]byte(src))
	if err != nil {
		t.Fatalf("Lex(%q): %v", src, err)
	}
	return res
}

func jsSkeleton(t *testing.T, src string) string {
	t.Helper()
	return renderSkeleton(jsLex(t, src).Skeleton)
}

func TestJSTokenizesTheLiteralForms(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "a division is not a regular expression",
			src:  "const half = total / 2;",
			want: `IDENT "const" IDENT "half" OP "=" IDENT "total" OP "/" NUMBER "2" OP ";"`,
		},
		{
			name: "a regular expression after a keyword is one token",
			src:  "return /a\\/b/g;",
			want: `IDENT "return" REGEX "/a\\/b/g" OP ";"`,
		},
		{
			name: "a character class holds a slash",
			src:  "const re = /[/]/;",
			want: `IDENT "const" IDENT "re" OP "=" REGEX "/[/]/" OP ";"`,
		},
		{
			name: "a template literal is one token",
			src:  "const s = `a${b + 1}c`;",
			want: "IDENT \"const\" IDENT \"s\" OP \"=\" TEMPLATE \"`a${b + 1}c`\" OP \";\"",
		},
		{
			name: "a nested template stays inside its outer token",
			src:  "const s = `a${`b${c}`}d`;",
			want: "IDENT \"const\" IDENT \"s\" OP \"=\" TEMPLATE \"`a${`b${c}`}d`\" OP \";\"",
		},
		{
			name: "a bigint keeps its suffix",
			src:  "const n = 10n;",
			want: `IDENT "const" IDENT "n" OP "=" NUMBER "10n" OP ";"`,
		},
		{
			name: "a hex literal is one number",
			src:  "const n = 0xFF_00;",
			want: `IDENT "const" IDENT "n" OP "=" NUMBER "0xFF_00" OP ";"`,
		},
		{
			name: "an exponent keeps its sign",
			src:  "const n = 1.5e-3;",
			want: `IDENT "const" IDENT "n" OP "=" NUMBER "1.5e-3" OP ";"`,
		},
		{
			name: "a private field keeps its hash",
			src:  "this.#count = 1;",
			want: `IDENT "this" OP "." IDENT "#count" OP "=" NUMBER "1" OP ";"`,
		},
		{
			name: "a typescript annotation tokenizes as operators",
			src:  "let a: Map<string, number>;",
			want: `IDENT "let" IDENT "a" OP ":" IDENT "Map" OP "<" IDENT "string" OP "," IDENT "number" OP ">" OP ";"`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := jsSkeleton(t, c.src); got != c.want {
				t.Errorf("skeleton is %s, want %s", got, c.want)
			}
		})
	}
}

func TestJSFindsNoCommentInsideALiteral(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"a string holds a line marker", `const url = "https://example.com";` + "\n"},
		{"a string holds a block opener", `const s = "/* not a comment */";` + "\n"},
		{"a template holds a line marker", "const s = `see https://example.com`;\n"},
		{"a template substitution holds a block comment", "const s = `${x /* keep */}`;\n"},
		{"a regular expression holds a line marker", "const re = /https:\\/\\//;\n"},
		{"a hashbang is not a comment", "#!/usr/bin/env node\nconst a = 1;\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := jsLex(t, c.src)
			if len(res.Blocks) != 0 || len(res.Trailing) != 0 {
				t.Errorf("got %d own-line and %d trailing block(s), want none", len(res.Blocks), len(res.Trailing))
			}
		})
	}
}

func TestJSHashbangStaysInTheSkeleton(t *testing.T) {
	// node reads the interpreter from the hashbang, so deleting it changes how
	// the file runs.
	before := jsSkeleton(t, "#!/usr/bin/env node\nconst a = 1;\n")
	after := jsSkeleton(t, "const a = 1;\n")
	if before == after {
		t.Errorf("the skeleton is %s with and without the hashbang", before)
	}
}

func TestJSGroupsAnOwnLineRun(t *testing.T) {
	src := "// the first line\n// the second line\nconst a = 1; // the trailing one\n"
	res := jsLex(t, src)
	if len(res.Blocks) != 1 {
		t.Fatalf("got %d own-line block(s), want 1", len(res.Blocks))
	}
	if got := res.Blocks[0].Payload(); got != "the first line\nthe second line" {
		t.Errorf("payload is %q", got)
	}
	if len(res.Trailing) != 1 {
		t.Fatalf("got %d trailing block(s), want 1", len(res.Trailing))
	}
	if got := res.Trailing[0].Payload(); got != "the trailing one" {
		t.Errorf("trailing payload is %q", got)
	}
}

func TestJSDirectiveFormsItsOwnBlock(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"an eslint disable", "// eslint-disable-next-line no-console\n"},
		{"a ts check", "// @ts-check\n"},
		{"a ts expect error", "// @ts-expect-error the type ships later\n"},
		{"a prettier ignore", "// prettier-ignore\n"},
		{"a coverage ignore", "/* c8 ignore next */\n"},
		{"a jsx pragma", "/** @jsx h */\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := c.src + "// the prose beneath it\nconst a = 1;\n"
			res := jsLex(t, src)
			if len(res.Blocks) != 2 {
				t.Fatalf("got %d block(s), want the directive and the prose apart", len(res.Blocks))
			}
			if !res.Blocks[0].Directive {
				t.Errorf("block 0 is not a directive")
			}
			if res.Blocks[1].Directive {
				t.Errorf("block 1 is a directive, want the prose")
			}
		})
	}
}

func TestJSDeletingADirectiveMovesTheSkeleton(t *testing.T) {
	// §5.1 keeps a protected directive in the skeleton, which turns ruling 10
	// into an enforced property.
	cases := []struct {
		name  string
		with  string
		clean string
	}{
		{
			name:  "an eslint disable",
			with:  "// eslint-disable-next-line no-console\nconsole.log(1);\n",
			clean: "console.log(1);\n",
		},
		{
			name:  "a ts expect error",
			with:  "// @ts-expect-error the type ships later\nwiden(a);\n",
			clean: "widen(a);\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if jsSkeleton(t, c.with) == jsSkeleton(t, c.clean) {
				t.Errorf("the skeleton survived the directive's deletion")
			}
		})
	}
}

func TestJSDirectiveNeedsAWordBoundary(t *testing.T) {
	// A marker that runs on into a word is prose that names a linter, and
	// protecting it would keep a line the sweep is meant to reach (SPEC §2.3).
	cases := []struct {
		src  string
		want bool
	}{
		{"// eslint-disable-next-line no-console\n", true},
		{"// eslint\n", true},
		{"// c8 ignore next\n", true},
		{"// eslint.org explains the rule\n", false},
		{"// eslinting the tree is a later ticket\n", false},
		{"// see https://eslint.org for the rule\n", false},
	}
	for _, c := range cases {
		t.Run(c.src, func(t *testing.T) {
			if got := jsLex(t, c.src+"const a = 1;\n").Blocks[0].Directive; got != c.want {
				t.Errorf("Directive is %v, want %v", got, c.want)
			}
		})
	}
}

func TestJSNoJSDocTagIsADirective(t *testing.T) {
	// No JSDoc tag is consumed by any tool in this repo, so protecting one
	// would keep 13,533 lines the sweep is meant to reach (SPEC §2.3).
	src := "/**\n * @param {string} name the caller's name\n * @returns {number}\n */\nfunction f(name) { return 1; }\n"
	res := jsLex(t, src)
	if len(res.Blocks) != 1 {
		t.Fatalf("got %d block(s), want 1", len(res.Blocks))
	}
	if res.Blocks[0].Directive {
		t.Errorf("a JSDoc block is a protected directive")
	}
}

func TestJSDeclarationFileKeepsItsFieldProse(t *testing.T) {
	// §4.3 keeps `.d.ts` field prose for the reader. The lexer still reports
	// it, because the carve-out is sweep-scoped and not a lint exemption.
	src := "export interface Props {\n  /** The visible label. */\n  label: string;\n}\n"
	res := jsLex(t, src)
	if len(res.Blocks) != 1 {
		t.Fatalf("got %d block(s), want the field comment", len(res.Blocks))
	}
	if got := res.Blocks[0].Payload(); got != "The visible label." {
		t.Errorf("payload is %q", got)
	}
}

func TestJSDeclarationFileMutationMovesTheSkeleton(t *testing.T) {
	// esbuild erases a `.d.ts` file to the empty string, so 109 of 109 code
	// mutations are invisible to it. The hand lexer sees this one (SPEC §5.3).
	base := "export interface Props {\n  /** The visible label. */\n  label: string;\n}\n"
	head := "export interface Props {\n  /** The visible label. */\n  label: number;\n}\n"
	if jsSkeleton(t, base) == jsSkeleton(t, head) {
		t.Errorf("the skeleton is blind to a changed field type")
	}
}

func TestJSSeparatingCommentMovesTheSkeleton(t *testing.T) {
	// §5.1: a comment can act as a token separator, so removing one is not a
	// no-op on every source.
	if jsSkeleton(t, "a/**/b") == jsSkeleton(t, "ab") {
		t.Errorf("a glued comment left no trace in the skeleton")
	}
}

func TestJSReportsAnUnreadableFile(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"a string ends at the newline", "const s = 'a\nb';\n", "a string ends at the newline"},
		{"a string never closes", "const s = 'abc", "opens and never closes"},
		{"a template never closes", "const s = `abc", "a template literal opens and never closes"},
		{"a substitution never closes", "const s = `a${b", "a template substitution opens and never closes"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := JS{}.Lex([]byte(c.src))
			if err == nil {
				t.Fatalf("Lex(%q) returned no error", c.src)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error is %q, want it to name %q", err, c.want)
			}
		})
	}
}

func TestForChoosesTheLexerByExtension(t *testing.T) {
	// §5.3: TypeScript generics look like JSX to a lexer, so the extension
	// decides and the content never does.
	cases := []struct {
		name string
		want Lexer
	}{
		{"docs-site/scripts/doclint.mjs", JS{}},
		{"docs-site/src/pipeline/toc.ts", JS{}},
		{"design-system/components/display/Card.d.ts", JS{}},
		{"docs-site/src/ds/Icon.jsx", JSX{Path: "docs-site/src/ds/Icon.jsx"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := For(c.name)
			if err != nil {
				t.Fatalf("For(%q): %v", c.name, err)
			}
			if got != c.want {
				t.Errorf("For(%q) chose %#v, want %#v", c.name, got, c.want)
			}
		})
	}
}

func TestForRefusesATSXFile(t *testing.T) {
	// A `.tsx` file meets the `.d.ts` erasure problem under esbuild, so it
	// needs a third tool. The repo has none today (SPEC §5.3).
	if _, err := For("docs-site/src/App.tsx"); err == nil {
		t.Fatalf("For(.tsx) returned a lexer")
	}
}
