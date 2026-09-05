package surface

import "strings"

const Glue = "GLUE"

type byteScan struct {
	src      []byte
	i        int
	line     int
	comments []rawComment
	skeleton []Token
	lastCode int
}

func (s *byteScan) peek(n int) byte {
	if s.i+n >= len(s.src) {
		return 0
	}
	return s.src[s.i+n]
}

func (s *byteScan) space(c byte) bool {
	if c == '\n' {
		s.line++
		s.i++
		return true
	}
	if isSpaceByte(c) {
		s.i++
		return true
	}
	return false
}

func (s *byteScan) emit(kind, text string, line int) {
	// A comment sharing a code line is trailing, and §3.4 holds that population apart.
	for k := len(s.comments) - 1; k >= 0 && s.comments[k].endLine == line; k-- {
		s.comments[k].ownLine = false
	}
	s.skeleton = append(s.skeleton, Token{Kind: kind, Text: text, Line: line})
	s.lastCode = s.line
}

func (s *byteScan) exponent() {
	if s.i >= len(s.src) || (s.src[s.i] != 'e' && s.src[s.i] != 'E') {
		return
	}
	j := s.i + 1
	if j < len(s.src) && (s.src[j] == '+' || s.src[j] == '-') {
		j++
	}
	if j >= len(s.src) || !isDigit(s.src[j]) {
		return
	}
	s.i = j
	for s.i < len(s.src) && isDigit(s.src[s.i]) {
		s.i++
	}
}

type rawComment struct {
	start     int
	end       int
	startLine int
	endLine   int
	text      string
	style     Style
	ownLine   bool
	directive bool
	waiver    bool
	field     bool
}

func assembleBlocks(lang Lang, src []byte, comments []rawComment) (blocks, trailing []Block) {
	cols := lineColumns(src)
	open, waiverEnd := -1, -1
	for _, c := range comments {
		if !c.ownLine {
			open = -1
			// §3.4 holds trailing apart, so no mechanical pass reaches it by walking Blocks.
			trailing = append(trailing, block(lang, cols, c))
			continue
		}
		joins := open >= 0 && !c.directive && c.style == StyleLine && blocks[open].EndLine+1 == c.startLine
		if joins {
			blocks[open].End = c.end
			blocks[open].EndLine = c.endLine
			blocks[open].Text = string(src[blocks[open].Start:c.end])
			blocks[open].Columns = max(blocks[open].Columns, spanColumns(cols, c.startLine, c.endLine))
			continue
		}
		b := block(lang, cols, c)
		// A gosec waiver's justification wraps onto the line below it (SPEC §2.3, #1274).
		b.WaiverTail = !c.directive && c.style == StyleLine && waiverEnd+1 == c.startLine
		blocks = append(blocks, b)
		open = len(blocks) - 1
		// §6.2 gives a protected directive its own block, so it never absorbs the prose beneath it.
		if c.directive || c.style == StyleBlock {
			open = -1
		}
		if c.waiver {
			waiverEnd = c.endLine
		}
	}
	return blocks, trailing
}

func block(lang Lang, cols []int, c rawComment) Block {
	return Block{
		Lang:      lang,
		Style:     c.style,
		Start:     c.start,
		End:       c.end,
		StartLine: c.startLine,
		EndLine:   c.endLine,
		Text:      c.text,
		Directive: c.directive,
		DTSField:  c.field,
		Columns:   spanColumns(cols, c.startLine, c.endLine),
	}
}

func lineColumns(src []byte) []int {
	lines := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	out := make([]int, len(lines))
	for i, line := range lines {
		out[i] = lineWidth(line)
	}
	return out
}

func lineWidth(line string) int {
	n := 0
	for _, r := range line {
		if r == '\t' {
			// 84 of `cmd/web`'s breaches are invisible in bytes, so §4.4 resolves at tab = 4 alone.
			n += TabColumns
			continue
		}
		n++
	}
	return n
}

func spanColumns(cols []int, startLine, endLine int) int {
	widest := 0
	for line := startLine; line <= endLine; line++ {
		if line < 1 || line > len(cols) {
			continue
		}
		widest = max(widest, cols[line-1])
	}
	return widest
}

func glued(src []byte, start, end int) bool {
	if start == 0 || end >= len(src) {
		return false
	}
	// `SELECT/*c*/a` strips to `SELECTa`, which is different SQL (SPEC §5.1).
	return !isSpaceByte(src[start-1]) && !isSpaceByte(src[end])
}

func isSpaceByte(c byte) bool {
	switch c {
	case ' ', '\t', '\r', '\n', '\f', '\v':
		return true
	}
	return false
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func hasAnyPrefix(text string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(text, p) {
			return true
		}
	}
	return false
}
