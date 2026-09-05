package surface

import "testing"

func lexColumns(t *testing.T, name, src string) Result {
	t.Helper()
	lexer, err := For(name)
	if err != nil {
		t.Fatalf("For(%q): %v", name, err)
	}
	res, err := lexer.Lex([]byte(src))
	if err != nil {
		t.Fatalf("Lex(%q): %v", src, err)
	}
	return res
}

func onlyBlock(t *testing.T, blocks []Block) Block {
	t.Helper()
	if len(blocks) != 1 {
		t.Fatalf("got %d block(s), want 1: %+v", len(blocks), blocks)
	}
	return blocks[0]
}

func TestBlockColumnsCoversEverySurfaceWithALexer(t *testing.T) {
	cases := []struct {
		name string
		file string
		src  string
		want int
	}{
		{"go", "a.go", "package p\n\nfunc f() {\n\t// abc\n}\n", 10},
		{"sql", "a.sql", "-- abcd\nSELECT 1;\n", 7},
		{"css", "a.css", "/* abc */\na { color: red; }\n", 9},
		{"js", "a.mjs", "// abcde\nconst a = 1;\n", 8},
		{"ts", "a.d.ts", "export interface A {\n\t// abc\n\treadonly b: string;\n}\n", 10},
		{"tmpl", "a.tmpl", "{{/* abc */}}\n<p>hi</p>\n", 13},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := onlyBlock(t, lexColumns(t, c.file, c.src).Blocks)
			if b.Columns != c.want {
				t.Errorf("got %d columns, want %d", b.Columns, c.want)
			}
		})
	}
}

func TestBlockColumnsCountsATabAsFour(t *testing.T) {
	src := "package p\n\nfunc f() {\n\t// abc\n}\n"
	b := onlyBlock(t, lexColumns(t, "a.go", src).Blocks)
	if b.Columns != 10 {
		t.Errorf("got %d columns, want 10 for a 7-byte line holding one tab", b.Columns)
	}
}

func TestBlockColumnsCountsRunesNotBytes(t *testing.T) {
	src := "package p\n\n// café ✓\nfunc f() {}\n"
	b := onlyBlock(t, lexColumns(t, "a.go", src).Blocks)
	if b.Columns != 9 {
		t.Errorf("got %d columns, want 9 for a 12-byte line of 9 runes", b.Columns)
	}
}

func TestBlockColumnsHoldsTheWholeTrailingLine(t *testing.T) {
	res := lexColumns(t, "a.go", "package p\n\nvar x = 1 // note\n")
	if len(res.Blocks) != 0 {
		t.Fatalf("got %d own-line block(s), want none: %+v", len(res.Blocks), res.Blocks)
	}
	b := onlyBlock(t, res.Trailing)
	if b.Columns != 17 {
		t.Errorf("got %d columns, want 17 including the code prefix", b.Columns)
	}
}

func TestBlockColumnsTakesTheWidestLineOfAMultiLineBlock(t *testing.T) {
	src := "package p\n\n// short\n// a much longer second line\nfunc f() {}\n"
	b := onlyBlock(t, lexColumns(t, "a.go", src).Blocks)
	if b.StartLine != 3 || b.EndLine != 4 {
		t.Fatalf("got lines %d-%d, want 3-4", b.StartLine, b.EndLine)
	}
	if b.Columns != 28 {
		t.Errorf("got %d columns, want 28 from the block's widest line", b.Columns)
	}
}

func TestBlockColumnsCoversABlockCommentSpanningLines(t *testing.T) {
	src := "package p\n\n/*\nabc\na much longer second line\n*/\nfunc f() {}\n"
	b := onlyBlock(t, lexColumns(t, "a.go", src).Blocks)
	if b.Columns != 25 {
		t.Errorf("got %d columns, want 25 from the block's widest line", b.Columns)
	}
}

func TestBlockColumnsIsPopulatedOnADirectiveTheRuleLayerSkips(t *testing.T) {
	src := "package p\n\n" +
		"//go:generate stringer -type=Kind -output=kind_string.go -linecomment -trimprefix=Kind -tags=none,more\n" +
		"type Kind int\n"
	b := onlyBlock(t, lexColumns(t, "a.go", src).Blocks)
	if !b.Directive {
		t.Fatalf("got Directive %t, want true", b.Directive)
	}
	if b.Columns != 102 {
		t.Errorf("got %d columns, want 102", b.Columns)
	}
}

func TestJSXPopulatesNoColumnsBecauseItPopulatesNoBlocks(t *testing.T) {
	esbuildOrSkip(t)
	res, err := JSX{Path: "columns_test.jsx"}.Lex([]byte("// a comment\nconst a = <p>hi</p>;\n"))
	if err != nil {
		t.Fatalf("Lex: %v", err)
	}
	if len(res.Blocks) != 0 || len(res.Trailing) != 0 {
		t.Errorf("got %d own-line and %d trailing block(s), want none", len(res.Blocks), len(res.Trailing))
	}
}
