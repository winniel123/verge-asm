package surface

import (
	"bytes"
	"fmt"
)

type SQL struct{}

var sqlDirectivePrefixes = []string{"-- name:", "-- +goose"}

const (
	SQLIdent     = "IDENT"
	SQLQuotedID  = "QIDENT"
	SQLString    = "STRING"
	SQLDollar    = "DOLLAR"
	SQLNumber    = "NUMBER"
	SQLParam     = "PARAM"
	SQLOp        = "OP"
	SQLDirective = "DIRECTIVE"
)

type sqlScan struct {
	byteScan
}

func (SQL) Lex(src []byte) (Result, error) {
	s := &sqlScan{byteScan{src: src, line: 1}}
	if err := s.run(); err != nil {
		return Result{}, err
	}
	blocks, trailing := assembleBlocks(LangSQL, src, s.comments)
	return Result{Blocks: blocks, Trailing: trailing, Skeleton: s.skeleton}, nil
}

func (s *sqlScan) run() error {
	for s.i < len(s.src) {
		c := s.src[s.i]
		switch {
		case s.space(c):
		case c == '-' && s.peek(1) == '-':
			s.lineComment()
		case c == '/' && s.peek(1) == '*':
			if err := s.blockComment(); err != nil {
				return err
			}
		case c == '\'':
			if err := s.quoted(s.i, '\'', SQLString, false); err != nil {
				return err
			}
		case c == '"':
			if err := s.quoted(s.i, '"', SQLQuotedID, false); err != nil {
				return err
			}
		case s.stringPrefix() > 0:
			// The introducer is part of the literal, so `E'a'` and `'a'` are
			// different tokens.
			start := s.i
			backslash := c == 'E' || c == 'e'
			s.i += s.stringPrefix()
			if err := s.quoted(start, '\'', SQLString, backslash); err != nil {
				return err
			}
		case c == '$':
			if err := s.dollar(); err != nil {
				return err
			}
		case isDigit(c) || (c == '.' && isDigit(s.peek(1))):
			s.number()
		case sqlIdentStart(c):
			s.ident()
		default:
			start := s.line
			s.i++
			s.emit(SQLOp, string(c), start)
		}
	}
	return nil
}

func (s *sqlScan) lineComment() {
	start := s.i
	for s.i < len(s.src) && s.src[s.i] != '\n' {
		s.i++
	}
	end := s.i
	if end > start && s.src[end-1] == '\r' {
		end--
	}
	text := string(s.src[start:end])
	directive := hasAnyPrefix(text, sqlDirectivePrefixes)
	s.comments = append(s.comments, rawComment{
		start: start, end: end, startLine: s.line, endLine: s.line,
		text: text, style: StyleLine,
		ownLine:   s.lastCode != s.line,
		directive: directive,
	})
	// §5.1 keeps a protected directive in the skeleton, which turns ruling 10
	// into an enforced property.
	if directive {
		s.skeleton = append(s.skeleton, Token{Kind: SQLDirective, Text: text, Line: s.line})
	}
}

func (s *sqlScan) blockComment() error {
	start, startLine := s.i, s.line
	depth := 0
	for s.i < len(s.src) {
		switch {
		// PostgreSQL nests a block comment, so the first `*/` need not close
		// the outermost one.
		case s.src[s.i] == '/' && s.peek(1) == '*':
			depth++
			s.i += 2
		case s.src[s.i] == '*' && s.peek(1) == '/':
			depth--
			s.i += 2
			if depth == 0 {
				s.finishBlockComment(start, startLine)
				return nil
			}
		default:
			if s.src[s.i] == '\n' {
				s.line++
			}
			s.i++
		}
	}
	return fmt.Errorf("scan: %d: a block comment opens and never closes", startLine)
}

func (s *sqlScan) finishBlockComment(start, startLine int) {
	s.comments = append(s.comments, rawComment{
		start: start, end: s.i, startLine: startLine, endLine: s.line,
		text: string(s.src[start:s.i]), style: StyleBlock,
		ownLine: s.lastCode != startLine,
	})
	if glued(s.src, start, s.i) {
		s.skeleton = append(s.skeleton, Token{Kind: Glue, Line: startLine})
	}
}

func (s *sqlScan) quoted(start int, quote byte, kind string, backslash bool) error {
	startLine := s.line
	s.i++
	for s.i < len(s.src) {
		c := s.src[s.i]
		switch {
		case backslash && c == '\\' && s.i+1 < len(s.src):
			if s.src[s.i+1] == '\n' {
				s.line++
			}
			s.i += 2
		case c == quote && s.peek(1) == quote:
			// A doubled quote is the literal quote, not the end of the string.
			s.i += 2
		case c == quote:
			s.i++
			s.emit(kind, string(s.src[start:s.i]), startLine)
			return nil
		default:
			if c == '\n' {
				s.line++
			}
			s.i++
		}
	}
	return fmt.Errorf("scan: %d: a %c-quoted literal opens and never closes", startLine, quote)
}

var sqlStringPrefixes = []string{"E'", "e'", "N'", "n'", "B'", "b'", "X'", "x'", "U&'", "u&'"}

func (s *sqlScan) stringPrefix() int {
	// PostgreSQL reads `abc'x'` as an identifier and then a string, so an
	// introducer never follows an identifier byte.
	if s.i > 0 && sqlIdentPart(s.src[s.i-1]) {
		return 0
	}
	rest := s.src[s.i:]
	for _, p := range sqlStringPrefixes {
		if len(rest) >= len(p) && string(rest[:len(p)]) == p {
			return len(p) - 1
		}
	}
	return 0
}

func (s *sqlScan) dollar() error {
	start, startLine := s.i, s.line
	if isDigit(s.peek(1)) {
		s.i++
		for s.i < len(s.src) && isDigit(s.src[s.i]) {
			s.i++
		}
		s.emit(SQLParam, string(s.src[start:s.i]), startLine)
		return nil
	}
	tag, ok := sqlDollarTag(s.src, s.i)
	if !ok {
		s.i++
		s.emit(SQLOp, "$", startLine)
		return nil
	}
	s.i += len(tag)
	for s.i < len(s.src) {
		if s.src[s.i] == '$' && bytes.HasPrefix(s.src[s.i:], []byte(tag)) {
			s.i += len(tag)
			s.emit(SQLDollar, string(s.src[start:s.i]), startLine)
			return nil
		}
		if s.src[s.i] == '\n' {
			s.line++
		}
		s.i++
	}
	return fmt.Errorf("scan: %d: the dollar quote %s opens and never closes", startLine, tag)
}

func sqlDollarTag(src []byte, i int) (string, bool) {
	j := i + 1
	if j < len(src) && src[j] == '$' {
		return "$$", true
	}
	if j >= len(src) || !(isLetter(src[j]) || src[j] == '_') {
		return "", false
	}
	for j < len(src) && (isLetter(src[j]) || isDigit(src[j]) || src[j] == '_') {
		j++
	}
	if j >= len(src) || src[j] != '$' {
		return "", false
	}
	return string(src[i : j+1]), true
}

func (s *sqlScan) number() {
	start, startLine := s.i, s.line
	if s.src[s.i] == '0' && (s.peek(1) == 'x' || s.peek(1) == 'X') {
		s.i += 2
		for s.i < len(s.src) && (isDigit(s.src[s.i]) || isLetter(s.src[s.i])) {
			s.i++
		}
		s.emit(SQLNumber, string(s.src[start:s.i]), startLine)
		return
	}
	for s.i < len(s.src) && (isDigit(s.src[s.i]) || s.src[s.i] == '.') {
		s.i++
	}
	s.exponent()
	s.emit(SQLNumber, string(s.src[start:s.i]), startLine)
}

func (s *sqlScan) ident() {
	start, startLine := s.i, s.line
	for s.i < len(s.src) && sqlIdentPart(s.src[s.i]) {
		s.i++
	}
	s.emit(SQLIdent, string(s.src[start:s.i]), startLine)
}

func sqlIdentStart(c byte) bool {
	return isLetter(c) || c == '_' || c >= 0x80
}

func sqlIdentPart(c byte) bool {
	// PostgreSQL allows `$` inside an identifier, so a dollar quote never
	// opens straight after one.
	return sqlIdentStart(c) || isDigit(c) || c == '$'
}
