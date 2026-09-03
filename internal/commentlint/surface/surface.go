package surface

import (
	"fmt"
	"path"
	"strings"
)

type Lang int

const (
	LangGo Lang = iota
	LangSQL
	LangCSS
	LangJS
	LangTmpl
)

func (l Lang) String() string {
	switch l {
	case LangSQL:
		return "sql"
	case LangCSS:
		return "css"
	case LangJS:
		return "js"
	case LangTmpl:
		return "tmpl"
	}
	return "go"
}

const DeleteRuleUnmeasured = "measured when the surface's sweep is scheduled (SPEC §6.5)"

func (l Lang) DeleteRule() string {
	switch l {
	case LangGo:
		return "remove the block's own lines, then gofmt (SPEC §3.8)"
	case LangTmpl:
		return "delete the comment's byte range, leave its line (SPEC §5.4)"
	}
	return DeleteRuleUnmeasured
}

func (l Lang) lineMarker() string {
	if l == LangSQL {
		return "--"
	}
	return "//"
}

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
	Lang        Lang
	Style       Style
	Start       int
	End         int
	StartLine   int
	EndLine     int
	Text        string
	Directive   bool
	Declaration bool
	PackageDoc  bool
	DeclGroup   bool
	DeclName    string
}

func (b Block) Lines() int {
	return b.EndLine - b.StartLine + 1
}

func (b Block) PayloadLines() []string {
	text := b.Text
	if b.Lang == LangTmpl {
		// A template action carries an optional trim marker, so the payload is found, not trimmed.
		if i := strings.Index(text, "/*"); i >= 0 {
			text = text[i:]
		}
		if i := strings.LastIndex(text, "*/"); i >= 0 {
			text = text[:i+2]
		}
	}
	if b.Style == StyleBlock {
		text = strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/")
	}
	raw := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		switch {
		case b.Style == StyleLine:
			line = strings.TrimPrefix(line, b.Lang.lineMarker())
		case line == "*":
			line = ""
		default:
			// Only a decorative star carries the trailing space, so `*p = 1` stays whole.
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

func LangOf(name string) (Lang, bool) {
	switch strings.ToLower(path.Ext(name)) {
	case ".go":
		return LangGo, true
	case ".sql":
		return LangSQL, true
	case ".css":
		return LangCSS, true
	case ".mjs", ".ts", ".jsx":
		return LangJS, true
	case ".tmpl":
		return LangTmpl, true
	}
	return LangGo, false
}

func For(name string) (Lexer, error) {
	ext := strings.ToLower(path.Ext(name))
	// TypeScript generics look like JSX, so the extension decides the lexer (SPEC §5.3).
	switch ext {
	case ".go":
		return Go{}, nil
	case ".sql":
		return SQL{}, nil
	case ".css":
		return CSS{}, nil
	case ".mjs", ".ts":
		// esbuild erases a `.d.ts` file, and 109 of the tree's 116 `.ts` files are `.d.ts` (§5.3).
		return JS{}, nil
	case ".jsx":
		return JSX{Path: name}, nil
	case ".tmpl":
		return Tmpl{}, nil
	}
	if ext == "" {
		ext = path.Base(name)
	}
	return nil, &UnsupportedError{Surface: ext}
}
