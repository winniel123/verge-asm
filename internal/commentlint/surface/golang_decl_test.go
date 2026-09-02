package surface

import (
	"errors"
	"slices"
	"testing"
)

func TestGoTrailingCommentsAreTheirOwnList(t *testing.T) {
	src := "package p\n" +
		"\n" +
		"// own line\n" +
		"func F() int { // opener\n" +
		"\treturn 1 // ports census\n" +
		"}\n"

	got, err := Go{}.Lex([]byte(src))
	if err != nil {
		t.Fatalf("Lex: %v", err)
	}
	if len(got.Blocks) != 1 || got.Blocks[0].Text != "// own line" {
		t.Fatalf("Blocks is %+v, want the own-line block alone", got.Blocks)
	}
	var texts []string
	for _, b := range got.Trailing {
		texts = append(texts, b.Text)
	}
	want := []string{"// opener", "// ports census"}
	if !slices.Equal(texts, want) {
		t.Errorf("Trailing holds %q, want %q", texts, want)
	}
	if got.Trailing[1].StartLine != 5 {
		t.Errorf("the second trailing block reports line %d, want 5", got.Trailing[1].StartLine)
	}
}

func TestGoDeclarationNames(t *testing.T) {
	cases := []struct {
		name        string
		src         string
		wantName    string
		wantIsPkg   bool
		wantIsDecl  bool
		wantIsGroup bool
	}{
		{"a func", "package p\n\n// c\nfunc F() {}\n", "F", false, true, false},
		{"a type", "package p\n\n// c\ntype T struct{}\n", "T", false, true, false},
		{"a single var", "package p\n\n// c\nvar x = 1\n", "x", false, true, false},
		{"a group member", "package p\n\nconst (\n\t// c\n\tA = 1\n)\n", "A", false, true, false},
		{"a struct field", "package p\n\ntype T struct {\n\t// c\n\tN int\n}\n", "N", false, true, false},
		{"an embedded field", "package p\n\ntype T struct {\n\t// c\n\tio.Reader\n}\n", "", false, true, false},
		{"a package clause", "// c\npackage p\n", "", true, true, false},
		{"a group opener declares no identifier", "package p\n\n// c\nconst (\n\tA = 1\n)\n", "", false, true, true},
		{"an import spec declares no identifier", "package p\n\nimport (\n\t// c\n\t\"os\"\n)\n", "", false, true, true},
		{"a body block declares nothing", "package p\n\nfunc F() {\n\t// c\n\t_ = 1\n}\n", "", false, false, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Go{}.Lex([]byte(c.src))
			if err != nil {
				t.Fatalf("Lex: %v", err)
			}
			b := got.Blocks[0]
			if b.DeclName != c.wantName {
				t.Errorf("DeclName is %q, want %q", b.DeclName, c.wantName)
			}
			if b.PackageDoc != c.wantIsPkg {
				t.Errorf("PackageDoc is %t, want %t", b.PackageDoc, c.wantIsPkg)
			}
			if b.Declaration != c.wantIsDecl {
				t.Errorf("Declaration is %t, want %t", b.Declaration, c.wantIsDecl)
			}
			if b.DeclGroup != c.wantIsGroup {
				t.Errorf("DeclGroup is %t, want %t", b.DeclGroup, c.wantIsGroup)
			}
		})
	}
}

func TestBlockPayload(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"a line run", "package p\n\n// one\n// two\nfunc F() {}\n", "one\ntwo"},
		{"a general comment", "package p\n\n/* one\n * two\n */\nfunc F() {}\n", "one\ntwo"},
		{"a dereference keeps its star", "package p\n\n/* *p = 1 */\nfunc F() {}\n", "*p = 1"},
		{"a carriage-return run", "package p\r\n\r\n// one\r\n// two\r\nfunc F() {}\r\n", "one\ntwo"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Go{}.Lex([]byte(c.src))
			if err != nil {
				t.Fatalf("Lex: %v", err)
			}
			if p := got.Blocks[0].Payload(); p != c.want {
				t.Errorf("Payload is %q, want %q", p, c.want)
			}
		})
	}
}

func TestForNamesTheUnsupportedSurface(t *testing.T) {
	_, err := For("db/queries/scan.sql")
	var unsupported *UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("For returned %v, want an UnsupportedError", err)
	}
	if unsupported.Surface != ".sql" {
		t.Errorf("the error names %q, want .sql", unsupported.Surface)
	}
}
