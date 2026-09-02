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
}

type Result struct {
	Blocks   []Block
	Skeleton []Token
}

type Lexer interface {
	Lex(src []byte) (Result, error)
}

func For(name string) (Lexer, error) {
	ext := strings.ToLower(path.Ext(name))
	if ext == ".go" {
		return Go{}, nil
	}
	if ext == "" {
		ext = path.Base(name)
	}
	return nil, fmt.Errorf("commentlint does not read the %s surface yet", ext)
}
