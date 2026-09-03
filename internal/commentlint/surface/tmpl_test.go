package surface

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
)

func lexTmpl(t *testing.T, src string) Result {
	t.Helper()
	res, err := Tmpl{}.Lex([]byte(src))
	if err != nil {
		t.Fatalf("Lex: %v", err)
	}
	return res
}

func TestTmplLexFindsTheCommentRange(t *testing.T) {
	cases := []struct {
		name      string
		src       string
		wantRange string
		wantLines [2]int
		trailing  bool
	}{
		{
			name:      "an own-line comment carries its delimiters",
			src:       "<p>a</p>\n{{/* the note */}}\n<p>b</p>\n",
			wantRange: "{{/* the note */}}",
			wantLines: [2]int{2, 2},
		},
		{
			name:      "a multi-line comment spans its own lines",
			src:       "{{/* the first line\n  the second line */}}\n<p>a</p>\n",
			wantRange: "{{/* the first line\n  the second line */}}",
			wantLines: [2]int{1, 2},
		},
		{
			name:      "a trim marker joins the range",
			src:       "<p>a</p>\n{{- /* the note */ -}}\n<p>b</p>\n",
			wantRange: "{{- /* the note */ -}}",
			wantLines: [2]int{2, 2},
		},
		{
			name:      "a comment sharing a markup line is trailing",
			src:       "<p>a</p> {{/* the note */}}\n<p>b</p>\n",
			wantRange: "{{/* the note */}}",
			wantLines: [2]int{1, 1},
			trailing:  true,
		},
		{
			name:      "a comment inside a define reaches the blocks",
			src:       "{{define \"x\"}}\n{{/* the note */}}\n<p>a</p>\n{{end}}\n",
			wantRange: "{{/* the note */}}",
			wantLines: [2]int{2, 2},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := lexTmpl(t, c.src)
			got := res.Blocks
			if c.trailing {
				if len(res.Blocks) != 0 {
					t.Fatalf("got %d own-line block(s), want 0", len(res.Blocks))
				}
				got = res.Trailing
			}
			if len(got) != 1 {
				t.Fatalf("got %d block(s), want 1", len(got))
			}
			b := got[0]
			if b.Lang != LangTmpl || b.Style != StyleBlock {
				t.Errorf("block is %s/%s, want tmpl/block", b.Lang, b.Style)
			}
			if c.src[b.Start:b.End] != c.wantRange {
				t.Errorf("range is %q, want %q", c.src[b.Start:b.End], c.wantRange)
			}
			if b.Text != c.wantRange {
				t.Errorf("text is %q, want %q", b.Text, c.wantRange)
			}
			if b.StartLine != c.wantLines[0] || b.EndLine != c.wantLines[1] {
				t.Errorf("lines are %d-%d, want %d-%d", b.StartLine, b.EndLine, c.wantLines[0], c.wantLines[1])
			}
		})
	}
}

func TestTmplPayloadDropsTheDelimiters(t *testing.T) {
	res := lexTmpl(t, "{{- /* the note\n   the second line */ -}}\n<p>a</p>\n")
	if len(res.Blocks) != 1 {
		t.Fatalf("got %d block(s), want 1", len(res.Blocks))
	}
	if got, want := res.Blocks[0].Payload(), "the note\nthe second line"; got != want {
		t.Errorf("payload is %q, want %q", got, want)
	}
}

func TestTmplSkeletonHoldsTheStructure(t *testing.T) {
	src := "{{define \"card\"}}<p>{{.Title}}</p>{{if .Body}}<em>{{.Body}}</em>{{else}}<em>none</em>{{end}}{{end}}\n"
	var got []string
	for _, tok := range lexTmpl(t, src).Skeleton {
		got = append(got, tok.Kind+"("+tok.Text+")")
	}
	want := []string{
		`DEFINE(card)`,
		`TEXT(<p>)`,
		`ACTION({{.Title}})`,
		`TEXT(</p>)`,
		`BRANCH(if .Body)`,
		`TEXT(<em>)`,
		`ACTION({{.Body}})`,
		`TEXT(</em>)`,
		`ELSE(else)`,
		`TEXT(<em>none</em>)`,
		`END(if)`,
		"TEXT(\n)",
	}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("skeleton is\n%v\nwant\n%v", got, want)
	}
}

func TestTmplTextRunsJoinAcrossAComment(t *testing.T) {
	// A comment between two text runs splits one TextNode into two, and the
	// delete merges them back, so an unjoined run reports a false difference.
	src := "<p>a</p>\n{{/* the note */}}\n<p>b</p>\n"
	res := lexTmpl(t, src)
	cut := TmplCut([]byte(src), res.Blocks)
	if string(cut) != "<p>a</p>\n\n<p>b</p>\n" {
		t.Fatalf("the cut is %q", string(cut))
	}
	after, err := Tmpl{}.Lex(cut)
	if err != nil {
		t.Fatalf("Lex after the cut: %v", err)
	}
	if len(res.Skeleton) != 1 || res.Skeleton[0].Kind != TmplText {
		t.Fatalf("the base skeleton is %v, want one joined text token", res.Skeleton)
	}
	if !sameSkeleton(res.Skeleton, after.Skeleton) {
		t.Errorf("the byte-range delete moved the skeleton: %v vs %v", res.Skeleton, after.Skeleton)
	}
}

func TestTmplByteRangeDeleteKeepsTheRender(t *testing.T) {
	// §5.4: deleting the comment's line takes the line's own newline with it,
	// and that newline is a byte the browser receives.
	src := "<p>{{.Title}}</p>\n" +
		"{{/* the header names the shot this template came from */}}\n" +
		"<p>{{.Body}}</p>\n"
	res := lexTmpl(t, src)
	if len(res.Blocks) != 1 {
		t.Fatalf("got %d block(s), want 1", len(res.Blocks))
	}
	data := struct{ Title, Body string }{"a", "b"}

	base := render(t, src, data)
	if got := render(t, string(TmplCut([]byte(src), res.Blocks)), data); got != base {
		t.Errorf("the byte-range delete rendered %q, want %q", got, base)
	}
	if got := render(t, cutWholeLines(src, res.Blocks), data); got == base {
		t.Errorf("the whole-line delete rendered %q, and §5.4 says it cannot match", got)
	}
}

func TestTmplLexFailsOnABrokenTemplate(t *testing.T) {
	if _, err := (Tmpl{}).Lex([]byte("{{if .A}}<p>a</p>\n")); err == nil {
		t.Error("an unclosed if lexed clean, and §6.7 fails closed on it")
	}
}

func TestTmplDeleteRuleTable(t *testing.T) {
	// §6.5 holds one delete rule per surface, and a wrong row is how a sweep
	// agent deletes a `.tmpl` line the browser needs.
	cases := map[Lang]string{
		LangGo:   "remove the block's own lines, then gofmt (SPEC §3.8)",
		LangTmpl: "delete the comment's byte range, leave its line (SPEC §5.4)",
		LangSQL:  DeleteRuleUnmeasured,
		LangCSS:  DeleteRuleUnmeasured,
		LangJS:   DeleteRuleUnmeasured,
	}
	for lang, want := range cases {
		if got := lang.DeleteRule(); got != want {
			t.Errorf("the %s row is %q, want %q", lang, got, want)
		}
	}
}

func TestLangOf(t *testing.T) {
	cases := map[string]Lang{
		"internal/p/a.go":                LangGo,
		"db/queries/scan.sql":            LangSQL,
		"design-system/tokens/base.css":  LangCSS,
		"docs-site/scripts/doclint.mjs":  LangJS,
		"docs-site/src/ds/Icon.jsx":      LangJS,
		"design-system/templates/a.tmpl": LangTmpl,
	}
	for name, want := range cases {
		got, ok := LangOf(name)
		if !ok || got != want {
			t.Errorf("%s is %s/%t, want %s/true", name, got, ok, want)
		}
	}
	if _, ok := LangOf("README.md"); ok {
		t.Error("LangOf named a surface for a markdown file")
	}
}

func render(t *testing.T, src string, data any) string {
	t.Helper()
	tpl, err := template.New("t").Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out bytes.Buffer
	if err := tpl.Execute(&out, data); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return out.String()
}

func cutWholeLines(src string, blocks []Block) string {
	dead := map[int]bool{}
	for _, b := range blocks {
		for n := b.StartLine; n <= b.EndLine; n++ {
			dead[n] = true
		}
	}
	var out []string
	for i, line := range strings.SplitAfter(src, "\n") {
		if !dead[i+1] {
			out = append(out, line)
		}
	}
	return strings.Join(out, "")
}

func sameSkeleton(base, head []Token) bool {
	if len(base) != len(head) {
		return false
	}
	for i := range base {
		if !base[i].Equal(head[i]) {
			return false
		}
	}
	return true
}
