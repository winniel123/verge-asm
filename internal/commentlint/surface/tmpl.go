package surface

import (
	"fmt"
	"sort"
	"strings"
	"text/template/parse"
	"unicode/utf8"
)

type Tmpl struct{}

const (
	TmplDefine = "DEFINE"
	TmplText   = "TEXT"
	TmplBranch = "BRANCH"
	TmplElse   = "ELSE"
	TmplEnd    = "END"
	TmplAction = "ACTION"

	tmplTextChunk = 200

	tmplRootName = "commentlint.root"
)

type tmplToken struct {
	Token
	pos int
}

func (Tmpl) Lex(src []byte) (Result, error) {
	// The root name has to miss every `{{define}}` in the tree, because a
	// collision reaches the caller as "multiple definition of template".
	root := parse.New(tmplRootName)
	// SkipFuncCheck builds the tree without the application's function map, and
	// ParseComments keeps a `{{/* */}}` node so its byte range is exact (§5.2).
	root.Mode = parse.ParseComments | parse.SkipFuncCheck
	trees := map[string]*parse.Tree{}
	if _, err := root.Parse(string(src), "", "", trees); err != nil {
		return Result{}, fmt.Errorf("parse: %v", err)
	}

	w := &tmplWalk{src: src, newlines: newlineOffsets(src)}
	for name, tree := range trees {
		if tree.Root == nil {
			continue
		}
		w.last = 0
		if tree != root {
			w.emit(TmplDefine, name, int(tree.Root.Pos))
		}
		w.node(tree.Root)
		w.flush()
	}
	if w.err != nil {
		return Result{}, w.err
	}

	// A `{{define}}` tree's span sits inside the root tree's, so walking the
	// set tree by tree reaches the comments out of source order.
	sort.SliceStable(w.tokens, func(a, b int) bool { return w.tokens[a].pos < w.tokens[b].pos })
	sort.SliceStable(w.comments, func(a, b int) bool { return w.comments[a].start < w.comments[b].start })

	skeleton := make([]Token, 0, len(w.tokens))
	for _, t := range w.tokens {
		skeleton = append(skeleton, t.Token)
	}
	blocks, trailing := assembleBlocks(LangTmpl, src, w.comments)
	return Result{Blocks: blocks, Trailing: trailing, Skeleton: skeleton}, nil
}

type tmplWalk struct {
	src      []byte
	newlines []int
	tokens   []tmplToken
	comments []rawComment
	err      error
	last     int

	pending     []string
	pendingAt   int
	pendingOpen bool
}

func (w *tmplWalk) node(n parse.Node) {
	switch v := n.(type) {
	case *parse.ListNode:
		if v == nil {
			return
		}
		for _, child := range v.Nodes {
			w.node(child)
		}
	case *parse.TextNode:
		if !w.pendingOpen {
			w.pendingOpen, w.pendingAt = true, int(v.Pos)
		}
		w.pending = append(w.pending, string(v.Text))
	case *parse.CommentNode:
		w.comment(v)
	case *parse.IfNode:
		w.branch("if", v.Pos, v.Pipe, v.List, v.ElseList)
	case *parse.RangeNode:
		w.branch("range", v.Pos, v.Pipe, v.List, v.ElseList)
	case *parse.WithNode:
		w.branch("with", v.Pos, v.Pipe, v.List, v.ElseList)
	default:
		// Every remaining node holds a pipeline and no list, so its String()
		// can carry no comment into the skeleton.
		w.emit(TmplAction, n.String(), int(n.Position()))
	}
}

func (w *tmplWalk) branch(kind string, pos parse.Pos, pipe *parse.PipeNode, list, elseList *parse.ListNode) {
	w.emit(TmplBranch, kind+" "+pipe.String(), int(pos))
	w.node(list)
	if elseList != nil {
		w.emit(TmplElse, "else", int(elseList.Pos))
		w.node(elseList)
	}
	w.flush()
	// A `BranchNode` carries no position for its `{{end}}`, and the token has
	// to sort behind every child the branch holds.
	w.emit(TmplEnd, kind, w.last+1)
}

func (w *tmplWalk) comment(c *parse.CommentNode) {
	start, end, err := tmplRange(w.src, int(c.Pos), len(c.Text))
	if err != nil {
		if w.err == nil {
			w.err = fmt.Errorf("scan: %d: %v", w.line(int(c.Pos)), err)
		}
		return
	}
	// A comment between two text runs splits one TextNode into two, and the
	// delete merges them back, so the run stays open across it.
	w.comments = append(w.comments, rawComment{
		start: start, end: end,
		startLine: w.line(start), endLine: w.line(end - 1),
		text:    string(w.src[start:end]),
		style:   StyleBlock,
		ownLine: tmplOwnLine(w.src, start),
	})
}

func (w *tmplWalk) emit(kind, text string, pos int) {
	w.flush()
	w.tokens = append(w.tokens, tmplToken{Token: Token{Kind: kind, Text: text, Line: w.line(pos)}, pos: pos})
	w.mark(pos)
}

func (w *tmplWalk) flush() {
	if !w.pendingOpen {
		return
	}
	joined := strings.Join(w.pending, "")
	at, line := w.pendingAt, w.line(w.pendingAt)
	w.pending, w.pendingOpen = w.pending[:0], false
	for _, t := range tmplTextTokens(joined, line) {
		w.tokens = append(w.tokens, tmplToken{Token: t, pos: at})
	}
	w.mark(at)
}

func (w *tmplWalk) mark(pos int) {
	if pos > w.last {
		w.last = pos
	}
}

func (w *tmplWalk) line(pos int) int {
	return sort.SearchInts(w.newlines, pos) + 1
}

func tmplTextTokens(text string, line int) []Token {
	var out []Token
	for text != "" {
		n := len(text)
		// A text node is output the browser receives, so the token carries its
		// bytes, and the chunk bounds the run a `verify` failure prints.
		if n > tmplTextChunk {
			n = tmplTextChunk
			for n > 0 && !utf8.RuneStart(text[n]) {
				n--
			}
			if n == 0 {
				n = tmplTextChunk
			}
		}
		out = append(out, Token{Kind: TmplText, Text: text[:n], Line: line})
		line += strings.Count(text[:n], "\n")
		text = text[n:]
	}
	return out
}

func tmplRange(src []byte, pos, length int) (start, end int, err error) {
	// The parser reports the `/* */` span, and §5.4 deletes the whole action,
	// so the range widens to the delimiters and any trim marker.
	start = pos
	for start > 0 && (src[start-1] == '-' || isSpaceByte(src[start-1])) {
		start--
	}
	if start < 2 || src[start-2] != '{' || src[start-1] != '{' {
		return 0, 0, fmt.Errorf("a comment opens with no left delimiter")
	}
	end = pos + length
	for end < len(src) && (src[end] == '-' || isSpaceByte(src[end])) {
		end++
	}
	if end+2 > len(src) || src[end] != '}' || src[end+1] != '}' {
		return 0, 0, fmt.Errorf("a comment closes with no right delimiter")
	}
	return start - 2, end + 2, nil
}

func tmplOwnLine(src []byte, start int) bool {
	for i := start - 1; i >= 0 && src[i] != '\n'; i-- {
		if !isSpaceByte(src[i]) {
			return false
		}
	}
	return true
}

func TmplCut(src []byte, blocks []Block) []byte {
	// §5.4: the byte range goes and the line stays. Removing the line fails
	// byte-exact comparison in 24 of 24 files, because its newline is output.
	ordered := append([]Block(nil), blocks...)
	sort.SliceStable(ordered, func(a, b int) bool { return ordered[a].Start < ordered[b].Start })
	var out []byte
	at := 0
	for _, b := range ordered {
		if b.Start < at || b.End > len(src) {
			continue
		}
		out = append(out, src[at:b.Start]...)
		at = b.End
	}
	return append(out, src[at:]...)
}

func newlineOffsets(src []byte) []int {
	var out []int
	for i, c := range src {
		if c == '\n' {
			out = append(out, i)
		}
	}
	return out
}
