package surface

import (
	"strings"
	"testing"
)

func sqlSkeleton(t *testing.T, src string) string {
	t.Helper()
	res, err := SQL{}.Lex([]byte(src))
	if err != nil {
		t.Fatalf("Lex(%q): %v", src, err)
	}
	return renderSkeleton(res.Skeleton)
}

func renderSkeleton(tokens []Token) string {
	parts := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		parts = append(parts, tok.String())
	}
	return strings.Join(parts, " ")
}

func TestSQLTokenizesTheLiteralForms(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "a doubled quote stays inside the string",
			src:  "SELECT 'it''s -- not a comment';",
			want: `IDENT "SELECT" STRING "'it''s -- not a comment'" OP ";"`,
		},
		{
			name: "an escape string reads a backslash quote",
			src:  `SELECT E'a\'b';`,
			want: `IDENT "SELECT" STRING "E'a\\'b'" OP ";"`,
		},
		{
			name: "a dollar quote holds a quote and a comment",
			src:  "SELECT $$ it's /* not a comment */ $$;",
			want: `IDENT "SELECT" DOLLAR "$$ it's /* not a comment */ $$" OP ";"`,
		},
		{
			name: "a tagged dollar quote ends only on its own tag",
			src:  "SELECT $tag$ a $$ b $tag$;",
			want: `IDENT "SELECT" DOLLAR "$tag$ a $$ b $tag$" OP ";"`,
		},
		{
			name: "a placeholder is not a dollar quote",
			src:  "WHERE id = $1;",
			want: `IDENT "WHERE" IDENT "id" OP "=" PARAM "$1" OP ";"`,
		},
		{
			name: "a quoted identifier keeps its case",
			src:  `SELECT "Odd Name" FROM t;`,
			want: `IDENT "SELECT" QIDENT "\"Odd Name\"" IDENT "FROM" IDENT "t" OP ";"`,
		},
		{
			name: "a dollar never opens a quote inside an identifier",
			src:  "SELECT a$$b$$;",
			want: `IDENT "SELECT" IDENT "a$$b$$" OP ";"`,
		},
		{
			name: "a number carries its exponent",
			src:  "SELECT 1.5e-3;",
			want: `IDENT "SELECT" NUMBER "1.5e-3" OP ";"`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sqlSkeleton(t, c.src); got != c.want {
				t.Errorf("skeleton is %s, want %s", got, c.want)
			}
		})
	}
}

func TestSQLNestsABlockComment(t *testing.T) {
	src := "SELECT 1 /* outer /* inner */ still the comment */ , 2;"
	res, err := SQL{}.Lex([]byte(src))
	if err != nil {
		t.Fatalf("Lex: %v", err)
	}
	if len(res.Trailing) != 1 {
		t.Fatalf("got %d trailing comments, want 1", len(res.Trailing))
	}
	want := "/* outer /* inner */ still the comment */"
	if got := res.Trailing[0].Text; got != want {
		t.Errorf("the comment is %q, want %q", got, want)
	}
	if got := renderSkeleton(res.Skeleton); got != `IDENT "SELECT" NUMBER "1" OP "," NUMBER "2" OP ";"` {
		t.Errorf("skeleton is %s", got)
	}
}

func TestSQLRejectsAnUnclosedConstruct(t *testing.T) {
	cases := map[string]string{
		"a block comment": "SELECT 1 /* forever\n",
		"a string":        "SELECT 'forever\n",
		"a dollar quote":  "SELECT $tag$ forever\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := (SQL{}).Lex([]byte(src)); err == nil {
				t.Fatal("Lex accepted an unclosed construct")
			}
		})
	}
}

func TestSQLSeparatingCommentMovesTheSkeleton(t *testing.T) {
	// §5.1: `SELECT/*c*/a` strips to `SELECTa`, which is different SQL.
	glued := sqlSkeleton(t, "SELECT/*c*/a")
	spaced := sqlSkeleton(t, "SELECT a")
	stripped := sqlSkeleton(t, "SELECTa")
	if glued == spaced {
		t.Errorf("SELECT/*c*/a and SELECT a share the skeleton %s", glued)
	}
	if glued == stripped {
		t.Errorf("SELECT/*c*/a and SELECTa share the skeleton %s", glued)
	}
}

func TestSQLProtectsItsDirectives(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"a sqlc opener", "-- name: GetOne :one\nSELECT 1;\n"},
		{"a goose marker", "-- +goose Up\nSELECT 1;\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			with := sqlSkeleton(t, c.src)
			without := sqlSkeleton(t, "SELECT 1;\n")
			if with == without {
				t.Fatalf("deleting the directive left the skeleton at %s", with)
			}
			if !strings.HasPrefix(with, "DIRECTIVE ") {
				t.Errorf("skeleton is %s, want it to open with the directive", with)
			}
		})
	}
}

func TestSQLBlocks(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []wantBlock
	}{
		{
			name: "an own-line run is one block",
			src:  "-- one\n-- two\nSELECT 1;\n",
			want: []wantBlock{
				{startLine: 1, endLine: 2, style: StyleLine, text: "-- one\n-- two"},
			},
		},
		{
			name: "a directive never absorbs the prose beneath it",
			src:  "-- name: GetOne :one\n-- why this query exists\nSELECT 1;\n",
			want: []wantBlock{
				{startLine: 1, endLine: 1, style: StyleLine, directive: true, text: "-- name: GetOne :one"},
				{startLine: 2, endLine: 2, style: StyleLine, text: "-- why this query exists"},
			},
		},
		{
			name: "a blank line splits the run",
			src:  "-- one\n\n-- two\nSELECT 1;\n",
			want: []wantBlock{
				{startLine: 1, endLine: 1, style: StyleLine, text: "-- one"},
				{startLine: 3, endLine: 3, style: StyleLine, text: "-- two"},
			},
		},
		{
			name: "a trailing comment is never a block",
			src:  "SELECT 1; -- note\n",
			want: nil,
		},
		{
			name: "code after a block comment makes it trailing",
			src:  "/* note */ SELECT 1;\n",
			want: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := SQL{}.Lex([]byte(c.src))
			if err != nil {
				t.Fatalf("Lex: %v", err)
			}
			checkBlocks(t, c.src, res.Blocks, c.want)
		})
	}
}

func TestSQLPayloadDropsTheDashMarker(t *testing.T) {
	res, err := SQL{}.Lex([]byte("-- one\n-- two\nSELECT 1;\n"))
	if err != nil {
		t.Fatalf("Lex: %v", err)
	}
	if got := res.Blocks[0].Payload(); got != "one\ntwo" {
		t.Errorf("payload is %q, want %q", got, "one\ntwo")
	}
}
