package surface

import (
	"fmt"
	"strings"
)

type CSS struct{}

var cssDirectiveMarkers = []string{"stylelint-", "postcss-"}

const (
	CSSIdent     = "IDENT"
	CSSAt        = "AT"
	CSSHash      = "HASH"
	CSSString    = "STRING"
	CSSNumber    = "NUMBER"
	CSSURL       = "URL"
	CSSOp        = "OP"
	CSSDirective = "DIRECTIVE"
)

type cssScan struct {
	src      []byte
	i        int
	line     int
	comments []rawComment
	skeleton []Token
	lastCode int
}

func (CSS) Lex(src []byte) (Result, error) {
	s := &cssScan{src: src, line: 1}
	if err := s.run(); err != nil {
		return Result{}, err
	}
	blocks, trailing := assembleBlocks(LangCSS, src, s.comments)
	return Result{Blocks: blocks, Trailing: trailing, Skeleton: s.skeleton}, nil
}

func (s *cssScan) run() error {
	for s.i < len(s.src) {
		c := s.src[s.i]
		switch {
		case c == '\n':
			s.line++
			s.i++
		case isSpaceByte(c):
			s.i++
		case c == '/' && s.peek(1) == '*':
			s.comment()
		case c == '\'' || c == '"':
			if err := s.str(c); err != nil {
				return err
			}
		case c == '@' && cssIdentStart(s.src, s.i+1):
			start := s.line
			s.i++
			s.emit(CSSAt, "@"+s.identText(), start)
		case c == '#':
			start := s.line
			s.i++
			s.emit(CSSHash, "#"+s.identText(), start)
		case cssNumberStart(s.src, s.i):
			s.number()
		case cssIdentStart(s.src, s.i):
			if err := s.identOrURL(); err != nil {
				return err
			}
		default:
			start := s.line
			s.i++
			s.emit(CSSOp, string(c), start)
		}
	}
	return nil
}

func (s *cssScan) peek(n int) byte {
	if s.i+n >= len(s.src) {
		return 0
	}
	return s.src[s.i+n]
}

func (s *cssScan) emit(kind, text string, line int) {
	for k := len(s.comments) - 1; k >= 0 && s.comments[k].endLine == line; k-- {
		s.comments[k].ownLine = false
	}
	s.skeleton = append(s.skeleton, Token{Kind: kind, Text: text, Line: line})
	s.lastCode = s.line
}

func (s *cssScan) comment() {
	start, startLine := s.i, s.line
	s.i += 2
	for s.i < len(s.src) {
		if s.src[s.i] == '*' && s.peek(1) == '/' {
			s.i += 2
			break
		}
		if s.src[s.i] == '\n' {
			s.line++
		}
		s.i++
	}
	// CSS Syntax §4.3.2 runs an unterminated comment to end-of-file rather
	// than raising a parse error, so a browser and this lexer agree.
	text := string(s.src[start:s.i])
	directive := containsAny(text, cssDirectiveMarkers)
	s.comments = append(s.comments, rawComment{
		start: start, end: s.i, startLine: startLine, endLine: s.line,
		text: text, style: StyleBlock,
		ownLine:   s.lastCode != startLine,
		directive: directive,
	})
	// §5.1 keeps a protected directive in the skeleton, which turns ruling 10
	// into an enforced property.
	if directive {
		s.skeleton = append(s.skeleton, Token{Kind: CSSDirective, Text: text, Line: startLine})
	}
	if glued(s.src, start, s.i) {
		s.skeleton = append(s.skeleton, Token{Kind: Glue, Line: startLine})
	}
}

func (s *cssScan) str(quote byte) error {
	start, startLine := s.i, s.line
	s.i++
	for s.i < len(s.src) {
		c := s.src[s.i]
		switch {
		case c == '\\' && s.i+1 < len(s.src):
			if s.src[s.i+1] == '\n' {
				s.line++
			}
			s.i += 2
		case c == quote:
			s.i++
			s.emit(CSSString, string(s.src[start:s.i]), startLine)
			return nil
		case c == '\n':
			return fmt.Errorf("scan: %d: a string ends at the newline", startLine)
		default:
			s.i++
		}
	}
	return fmt.Errorf("scan: %d: a %c-quoted string opens and never closes", startLine, quote)
}

func (s *cssScan) identOrURL() error {
	startLine := s.line
	text := s.identText()
	// An unquoted url() body is not a token stream: a `/*` inside it is part
	// of the URL, so the whole body is one token.
	if strings.EqualFold(text, "url") && s.i < len(s.src) && s.src[s.i] == '(' {
		j := s.i + 1
		for j < len(s.src) && isSpaceByte(s.src[j]) && s.src[j] != '\n' {
			j++
		}
		if j < len(s.src) && s.src[j] != '\'' && s.src[j] != '"' && s.src[j] != ')' {
			start := s.i
			for s.i < len(s.src) && s.src[s.i] != ')' {
				if s.src[s.i] == '\n' {
					s.line++
				}
				s.i++
			}
			if s.i >= len(s.src) {
				return fmt.Errorf("scan: %d: an unquoted url() opens and never closes", startLine)
			}
			s.i++
			s.emit(CSSURL, text+string(s.src[start:s.i]), startLine)
			return nil
		}
	}
	s.emit(CSSIdent, text, startLine)
	return nil
}

func (s *cssScan) identText() string {
	start := s.i
	// A trailing backslash is an identifier byte that consumes nothing, and a
	// scanner that consumes nothing never ends.
	if s.i < len(s.src) && s.src[s.i] == '\\' && s.i+1 >= len(s.src) {
		s.i++
		return string(s.src[start:s.i])
	}
	for s.i < len(s.src) {
		c := s.src[s.i]
		if c == '\\' && s.i+1 < len(s.src) {
			s.i += 2
			continue
		}
		if !cssIdentPart(c) {
			break
		}
		s.i++
	}
	return string(s.src[start:s.i])
}

func (s *cssScan) number() {
	start, startLine := s.i, s.line
	if s.src[s.i] == '+' || s.src[s.i] == '-' {
		s.i++
	}
	for s.i < len(s.src) && (isDigit(s.src[s.i]) || s.src[s.i] == '.') {
		s.i++
	}
	if s.i < len(s.src) && (s.src[s.i] == 'e' || s.src[s.i] == 'E') {
		j := s.i + 1
		if j < len(s.src) && (s.src[j] == '+' || s.src[j] == '-') {
			j++
		}
		if j < len(s.src) && isDigit(s.src[j]) {
			s.i = j
			for s.i < len(s.src) && isDigit(s.src[s.i]) {
				s.i++
			}
		}
	}
	// The unit joins the number, so `20px` and `20 px` are different streams.
	if s.i < len(s.src) && s.src[s.i] == '%' {
		s.i++
	} else if cssIdentStart(s.src, s.i) {
		s.identText()
	}
	s.emit(CSSNumber, string(s.src[start:s.i]), startLine)
}

func cssIdentStart(src []byte, i int) bool {
	if i >= len(src) {
		return false
	}
	c := src[i]
	if isLetter(c) || c == '_' || c >= 0x80 || c == '\\' {
		return true
	}
	// A custom property opens `--`, and a vendor prefix opens `-webkit-`, so a
	// leading dash starts an identifier only ahead of an identifier byte.
	if c == '-' && i+1 < len(src) {
		n := src[i+1]
		return isLetter(n) || n == '_' || n == '-' || n >= 0x80 || n == '\\'
	}
	return false
}

func cssIdentPart(c byte) bool {
	return isLetter(c) || isDigit(c) || c == '_' || c == '-' || c >= 0x80
}

func cssNumberStart(src []byte, i int) bool {
	c := src[i]
	if isDigit(c) {
		return true
	}
	if c == '.' && i+1 < len(src) && isDigit(src[i+1]) {
		return true
	}
	if c != '+' && c != '-' {
		return false
	}
	if i+1 < len(src) && isDigit(src[i+1]) {
		return true
	}
	return i+2 < len(src) && src[i+1] == '.' && isDigit(src[i+2])
}

func containsAny(text string, markers []string) bool {
	for _, m := range markers {
		if strings.Contains(text, m) {
			return true
		}
	}
	return false
}
