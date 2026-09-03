package surface

const Glue = "GLUE"

type rawComment struct {
	start     int
	end       int
	startLine int
	endLine   int
	text      string
	style     Style
	ownLine   bool
	directive bool
}

func assembleBlocks(lang Lang, src []byte, comments []rawComment) (blocks, trailing []Block) {
	open := -1
	for _, c := range comments {
		if !c.ownLine {
			open = -1
			// §3.4 holds the trailing population apart from the own-line one,
			// so no mechanical pass can reach it by walking Blocks.
			trailing = append(trailing, block(lang, c))
			continue
		}
		joins := open >= 0 && !c.directive && c.style == StyleLine && blocks[open].EndLine+1 == c.startLine
		if joins {
			blocks[open].End = c.end
			blocks[open].EndLine = c.endLine
			blocks[open].Text = string(src[blocks[open].Start:c.end])
			continue
		}
		blocks = append(blocks, block(lang, c))
		open = len(blocks) - 1
		// §6.2 gives a protected directive its own block, so it never absorbs
		// the prose beneath it.
		if c.directive || c.style == StyleBlock {
			open = -1
		}
	}
	return blocks, trailing
}

func block(lang Lang, c rawComment) Block {
	return Block{
		Lang:      lang,
		Style:     c.style,
		Start:     c.start,
		End:       c.end,
		StartLine: c.startLine,
		EndLine:   c.endLine,
		Text:      c.text,
		Directive: c.directive,
	}
}

// glued reports a comment that separates two tokens. `SELECT/*c*/a` strips to
// `SELECTa`, which is different SQL, so the separation is itself part of the
// skeleton (SPEC §5.1).
func glued(src []byte, start, end int) bool {
	if start == 0 || end >= len(src) {
		return false
	}
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
