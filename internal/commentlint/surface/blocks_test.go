package surface

import "testing"

type wantBlock struct {
	startLine   int
	endLine     int
	style       Style
	directive   bool
	declaration bool
	text        string
}

func checkBlocks(t *testing.T, src string, got []Block, want []wantBlock) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d blocks, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		b := got[i]
		if b.StartLine != w.startLine || b.EndLine != w.endLine {
			t.Errorf("block %d: got lines %d-%d, want %d-%d", i, b.StartLine, b.EndLine, w.startLine, w.endLine)
		}
		if b.Style != w.style {
			t.Errorf("block %d: got style %s, want %s", i, b.Style, w.style)
		}
		if b.Directive != w.directive {
			t.Errorf("block %d: got directive %t, want %t", i, b.Directive, w.directive)
		}
		if b.Declaration != w.declaration {
			t.Errorf("block %d: got declaration %t, want %t", i, b.Declaration, w.declaration)
		}
		if b.Text != w.text {
			t.Errorf("block %d: got text %q, want %q", i, b.Text, w.text)
		}
		if slice := src[b.Start:b.End]; slice != b.Text {
			t.Errorf("block %d: byte range holds %q, text holds %q", i, slice, b.Text)
		}
	}
}
