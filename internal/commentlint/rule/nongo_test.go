package rule

import (
	"testing"

	"github.com/winniel123/verge-asm/internal/commentlint/surface"
)

func lexOne(t *testing.T, lexer surface.Lexer, src string) surface.Block {
	t.Helper()
	res, err := lexer.Lex([]byte(src))
	if err != nil {
		t.Fatalf("Lex: %v", err)
	}
	if len(res.Blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(res.Blocks))
	}
	return res.Blocks[0]
}

func TestClassifyReadsTheNonGoSurfaces(t *testing.T) {
	// §6.6: off Go there is no parser, so `commented-out-code` is a shape test.
	cases := []struct {
		name  string
		lexer surface.Lexer
		src   string
		want  Class
	}{
		{
			name:  "a commented-out sql statement",
			lexer: surface.SQL{},
			src:   "-- SELECT id FROM account;\nSELECT 1;\n",
			want:  CommentedOutCode,
		},
		{
			name:  "sql prose that ends in a terminator is not code",
			lexer: surface.SQL{},
			src:   "-- The account list omits the password hash column, and it omits the totp secret;\nSELECT 1;\n",
			want:  ProseOther,
		},
		{
			name:  "sql prose that opens with a keyword is not code",
			lexer: surface.SQL{},
			src:   "-- Select the newest run, then read its receipt\nSELECT 1;\n",
			want:  ProseOther,
		},
		{
			name:  "a commented-out css declaration",
			lexer: surface.CSS{},
			src:   "/* color: red; */\na { color: blue; }\n",
			want:  CommentedOutCode,
		},
		{
			name:  "css prose is not code",
			lexer: surface.CSS{},
			src:   "/* the azure scale */\na { color: blue; }\n",
			want:  ShortLabel,
		},
		{
			name:  "a commented-out js statement",
			lexer: surface.JS{},
			src:   "// const total = count + 1;\nexport const a = 1;\n",
			want:  CommentedOutCode,
		},
		{
			name:  "js prose that ends in a terminator is not code",
			lexer: surface.JS{},
			src:   "// The loader reads the guides one level up, outside this Astro project;\nexport const a = 1;\n",
			want:  ProseOther,
		},
		{
			// JavaScript is case-sensitive, so `If` opens no statement.
			name:  "js prose that opens with a capitalised keyword is not code",
			lexer: surface.JS{},
			src:   "// If the value changes;\nexport const a = 1;\n",
			want:  ShortLabel,
		},
		{
			name:  "a commented-out tmpl action",
			lexer: surface.Tmpl{},
			src:   "{{/* {{if .Total}}<p>a</p>{{end}} */}}\n<p>b</p>\n",
			want:  CommentedOutCode,
		},
		{
			name:  "commented-out tmpl markup",
			lexer: surface.Tmpl{},
			src:   "{{/* <p class=\"lead\">the counted total</p> */}}\n<p>b</p>\n",
			want:  CommentedOutCode,
		},
		{
			name:  "tmpl prose that names a tag is not code",
			lexer: surface.Tmpl{},
			src:   "{{/* The lead paragraph carries the counted total */}}\n<p>b</p>\n",
			want:  ProseOther,
		},
		{
			name:  "a sql divider",
			lexer: surface.SQL{},
			src:   "-- ------------------------\nSELECT 1;\n",
			want:  SectionDivider,
		},
		{
			name:  "a sql citation",
			lexer: surface.SQL{},
			src:   "-- The instance holds one row (ADR-0042).\nSELECT 1;\n",
			want:  Citation,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Classify(lexOne(t, c.lexer, c.src)); got != c.want {
				t.Errorf("class is %s, want %s", got, c.want)
			}
		})
	}
}

func TestDeletableHoldsOffGo(t *testing.T) {
	// §3.6 reads `agent` in every non-Go cell, so no non-Go block is deletable.
	cases := []struct {
		name  string
		lexer surface.Lexer
		src   string
	}{
		{"a sql divider", surface.SQL{}, "-- ------------------------\nSELECT 1;\n"},
		{"a sql short label", surface.SQL{}, "-- the account list\nSELECT 1;\n"},
		{"a css short label", surface.CSS{}, "/* the azure scale */\na { color: blue; }\n"},
		{"a js divider", surface.JS{}, "// ------------------------\nexport const a = 1;\n"},
		{"a js short label", surface.JS{}, "// the guide loader\nexport const a = 1;\n"},
		{"a tmpl divider", surface.Tmpl{}, "{{/* ------------------------ */}}\n<p>b</p>\n"},
		{"a tmpl short label", surface.Tmpl{}, "{{/* the shell chrome */}}\n<p>b</p>\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := lexOne(t, c.lexer, c.src)
			class, _, deletable := Deletable(b)
			if !InDeleteSet(class) {
				t.Fatalf("class is %s, want a delete-set class so the guard is what refuses", class)
			}
			if deletable {
				t.Errorf("Deletable said yes on the %s surface", b.Lang)
			}
		})
	}
}

func TestLintFlagsOffGo(t *testing.T) {
	// §3.6 binds the delete pass and not the flag set, so `lint` still reports.
	res, err := surface.SQL{}.Lex([]byte("-- ------------------------\nSELECT 1;\n"))
	if err != nil {
		t.Fatalf("Lex: %v", err)
	}
	found := Lint(res, false)
	if len(found) != 1 || found[0].Rule != string(SectionDivider) {
		t.Fatalf("got %+v, want one section-divider finding", found)
	}
}
