package surface

import (
	"fmt"
	"strings"
)

type JS struct{}

var jsDirectiveMarkers = []string{
	"eslint", "@ts-check", "@ts-expect-error", "prettier-ignore", "c8 ignore", "@jsx",
}

const (
	JSIdent     = "IDENT"
	JSNumber    = "NUMBER"
	JSString    = "STRING"
	JSTemplate  = "TEMPLATE"
	JSRegex     = "REGEX"
	JSOp        = "OP"
	JSDirective = "DIRECTIVE"
	JSHashbang  = "HASHBANG"
)

// A regex literal and a division share the `/` byte, so the token before it
// decides. These keywords end no expression, so a `/` after one opens a regex.
var jsRegexKeywords = map[string]bool{
	"await": true, "case": true, "delete": true, "do": true, "else": true,
	"in": true, "instanceof": true, "new": true, "of": true, "return": true,
	"throw": true, "typeof": true, "void": true, "yield": true,
}

type jsScan struct {
	byteScan
}

func (JS) Lex(src []byte) (Result, error) {
	s := &jsScan{byteScan{src: src, line: 1}}
	if err := s.run(); err != nil {
		return Result{}, err
	}
	blocks, trailing := assembleBlocks(LangJS, src, s.comments)
	return Result{Blocks: blocks, Trailing: trailing, Skeleton: s.skeleton}, nil
}

func (s *jsScan) run() error {
	s.hashbang()
	for s.i < len(s.src) {
		c := s.src[s.i]
		switch {
		case s.space(c):
		case c == '/' && s.peek(1) == '/':
			s.lineComment()
		case c == '/' && s.peek(1) == '*':
			s.blockComment()
		case c == '/':
			if s.regexAllowed() {
				if err := s.regex(); err != nil {
					return err
				}
				continue
			}
			s.op()
		case c == '\'' || c == '"':
			start, startLine := s.i, s.line
			if err := s.skipString(c); err != nil {
				return err
			}
			s.emit(JSString, string(s.src[start:s.i]), startLine)
		case c == '`':
			start, startLine := s.i, s.line
			if err := s.skipTemplate(); err != nil {
				return err
			}
			s.emit(JSTemplate, string(s.src[start:s.i]), startLine)
		case isDigit(c) || (c == '.' && isDigit(s.peek(1))):
			s.number()
		case jsIdentStart(c), c == '#' && jsIdentStart(s.peek(1)):
			start, startLine := s.i, s.line
			s.i++
			s.identRest()
			s.emit(JSIdent, string(s.src[start:s.i]), startLine)
		default:
			s.op()
		}
	}
	return nil
}

func (s *jsScan) op() {
	start := s.line
	c := s.src[s.i]
	s.i++
	// One byte per operator token. The stream is compared for identity, so a
	// split `=>` is as decisive as a joined one.
	s.emit(JSOp, string(c), start)
}

func (s *jsScan) hashbang() {
	if len(s.src) < 2 || s.src[0] != '#' || s.src[1] != '!' {
		return
	}
	for s.i < len(s.src) && s.src[s.i] != '\n' {
		s.i++
	}
	// A hashbang is executable configuration, not a comment: node reads the
	// interpreter from it, so deleting it changes how the file runs.
	s.emit(JSHashbang, string(s.src[:s.i]), 1)
}

func (s *jsScan) lineComment() {
	start := s.i
	for s.i < len(s.src) && s.src[s.i] != '\n' {
		s.i++
	}
	end := s.i
	if end > start && s.src[end-1] == '\r' {
		end--
	}
	s.addComment(start, end, s.line, s.line, StyleLine)
}

func (s *jsScan) blockComment() {
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
	s.addComment(start, s.i, startLine, s.line, StyleBlock)
}

func (s *jsScan) addComment(start, end, startLine, endLine int, style Style) {
	text := string(s.src[start:end])
	directive := jsDirective(text)
	s.comments = append(s.comments, rawComment{
		start: start, end: end, startLine: startLine, endLine: endLine,
		text: text, style: style,
		ownLine:   s.lastCode != startLine,
		directive: directive,
	})
	// §5.1 keeps a protected directive in the skeleton, which turns ruling 10
	// into an enforced property.
	if directive {
		s.skeleton = append(s.skeleton, Token{Kind: JSDirective, Text: text, Line: startLine})
	}
	if glued(s.src, start, end) {
		s.skeleton = append(s.skeleton, Token{Kind: Glue, Line: startLine})
	}
}

func (s *jsScan) regexAllowed() bool {
	if len(s.skeleton) == 0 {
		return true
	}
	last := s.skeleton[len(s.skeleton)-1]
	switch last.Kind {
	case JSIdent:
		return jsRegexKeywords[last.Text]
	case JSNumber, JSString, JSTemplate, JSRegex, JSHashbang:
		return false
	case JSOp:
		return last.Text != ")" && last.Text != "]" && last.Text != "}"
	}
	return true
}

func (s *jsScan) regex() error {
	start, startLine := s.i, s.line
	s.i++
	class := false
	for s.i < len(s.src) {
		c := s.src[s.i]
		switch {
		case c == '\\' && s.i+1 < len(s.src):
			s.i += 2
			continue
		case c == '\n':
			// A regex literal holds no raw newline, so the guess was wrong and
			// §6.7 prefers "cannot judge this file" over a silent misread.
			return fmt.Errorf("scan: %d: a regular expression opens and never closes", startLine)
		case c == '[':
			class = true
		case c == ']':
			class = false
		case c == '/' && !class:
			s.i++
			s.identRest()
			s.emit(JSRegex, string(s.src[start:s.i]), startLine)
			return nil
		}
		s.i++
	}
	return fmt.Errorf("scan: %d: a regular expression opens and never closes", startLine)
}

func (s *jsScan) skipString(quote byte) error {
	startLine := s.line
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
			return nil
		case c == '\n':
			return fmt.Errorf("scan: %d: a string ends at the newline", startLine)
		default:
			s.i++
		}
	}
	return fmt.Errorf("scan: %d: a %c-quoted string opens and never closes", startLine, quote)
}

// A template literal is one token, holding its substitutions whole. A comment
// inside `${}` therefore stays with the agent rather than reaching the sweep.
func (s *jsScan) skipTemplate() error {
	startLine := s.line
	s.i++
	for s.i < len(s.src) {
		c := s.src[s.i]
		switch {
		case c == '\\' && s.i+1 < len(s.src):
			if s.src[s.i+1] == '\n' {
				s.line++
			}
			s.i += 2
		case c == '`':
			s.i++
			return nil
		case c == '$' && s.peek(1) == '{':
			s.i += 2
			if err := s.skipTemplateExpr(); err != nil {
				return err
			}
		case c == '\n':
			s.line++
			s.i++
		default:
			s.i++
		}
	}
	return fmt.Errorf("scan: %d: a template literal opens and never closes", startLine)
}

func (s *jsScan) skipTemplateExpr() error {
	startLine := s.line
	depth := 1
	for s.i < len(s.src) {
		c := s.src[s.i]
		switch {
		case c == '\n':
			s.line++
			s.i++
		case c == '{':
			depth++
			s.i++
		case c == '}':
			depth--
			s.i++
			if depth == 0 {
				return nil
			}
		case c == '\'' || c == '"':
			if err := s.skipString(c); err != nil {
				return err
			}
		case c == '`':
			if err := s.skipTemplate(); err != nil {
				return err
			}
		case c == '/' && s.peek(1) == '/':
			for s.i < len(s.src) && s.src[s.i] != '\n' {
				s.i++
			}
		case c == '/' && s.peek(1) == '*':
			s.i += 2
			for s.i < len(s.src) && !(s.src[s.i] == '*' && s.peek(1) == '/') {
				if s.src[s.i] == '\n' {
					s.line++
				}
				s.i++
			}
			s.i += 2
		default:
			s.i++
		}
	}
	return fmt.Errorf("scan: %d: a template substitution opens and never closes", startLine)
}

func (s *jsScan) number() {
	start, startLine := s.i, s.line
	if s.src[s.i] == '0' && jsRadixMark(s.peek(1)) {
		s.i += 2
		for s.i < len(s.src) && (isHexDigit(s.src[s.i]) || s.src[s.i] == '_') {
			s.i++
		}
	} else {
		for s.i < len(s.src) && (isDigit(s.src[s.i]) || s.src[s.i] == '.' || s.src[s.i] == '_') {
			s.i++
		}
		s.exponent()
	}
	// A BigInt suffix is part of the literal, so `1n` and `1` are different
	// tokens.
	if s.i < len(s.src) && s.src[s.i] == 'n' {
		s.i++
	}
	s.emit(JSNumber, string(s.src[start:s.i]), startLine)
}

func (s *jsScan) identRest() {
	for s.i < len(s.src) && jsIdentPart(s.src[s.i]) {
		s.i++
	}
}

func jsIdentStart(c byte) bool {
	return isLetter(c) || c == '_' || c == '$' || c >= 0x80
}

func jsIdentPart(c byte) bool {
	return jsIdentStart(c) || isDigit(c)
}

func jsRadixMark(c byte) bool {
	switch c {
	case 'x', 'X', 'b', 'B', 'o', 'O':
		return true
	}
	return false
}

func isHexDigit(c byte) bool {
	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func jsDirective(text string) bool {
	// A marker anywhere in the text would protect prose that merely names a
	// linter, so the directive has to open the comment (SPEC §2.3).
	return hasAnyPrefix(jsInner(text), jsDirectiveMarkers)
}

func jsInner(text string) string {
	if strings.HasPrefix(text, "//") {
		return strings.TrimSpace(strings.TrimPrefix(text, "//"))
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/")
	return strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(inner), "*"))
}
