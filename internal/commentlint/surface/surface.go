package surface

import (
	"fmt"
	"path"
	"strings"
)

type Style int

const (
	StyleLine Style = iota
	StyleBlock
)

func (s Style) String() string {
	if s == StyleBlock {
		return "block"
	}
	return "line"
}

type Token struct {
	Kind string
	Text string
	Line int
}

func (t Token) Equal(other Token) bool {
	return t.Kind == other.Kind && t.Text == other.Text
}

func (t Token) String() string {
	if t.Text == "" || t.Text == t.Kind {
		return t.Kind
	}
	return fmt.Sprintf("%s %q", t.Kind, t.Text)
}

type Block struct {
	Style       Style
	Start       int
	End         int
	StartLine   int
	EndLine     int
	Text        string
	Directive   bool
	Declaration bool
	PackageDoc  bool
	DeclName    string
}

func (b Block) Lines() int {
	return b.EndLine - b.StartLine + 1
}

func (b Block) PayloadLines() []string {
	text := b.Text
	if b.Style == StyleBlock {
		text = strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/")
	}
	raw := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		switch {
		case b.Style == StyleLine:
			line = strings.TrimPrefix(line, "//")
		case line == "*":
			line = ""
		default:
			// The trailing space keeps a dereference such as `*p = 1` whole,
			// because only a decorative star carries one.
			if rest, ok := strings.CutPrefix(line, "* "); ok {
				line = rest
			}
		}
		out = append(out, strings.TrimSpace(line))
	}
	return out
}

func (b Block) Payload() string {
	return strings.TrimSpace(strings.Join(b.PayloadLines(), "\n"))
}

type Result struct {
	Blocks   []Block
	Trailing []Block
	Skeleton []Token
}

type Lexer interface {
	Lex(src []byte) (Result, error)
}

type UnsupportedError struct {
	Surface string
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("commentlint does not read the %s surface yet", e.Surface)
}

func For(name string) (Lexer, error) {
	ext := strings.ToLower(path.Ext(name))
	if ext == ".go" {
		return Go{}, nil
	}
	if ext == "" {
		ext = path.Base(name)
	}
	return nil, &UnsupportedError{Surface: ext}
}
