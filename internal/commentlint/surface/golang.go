package surface

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"strings"
)

type Go struct{}

var goDirectivePrefixes = []string{"//go:", "// +build", "//nolint", "//lint:", "//revive:"}

var goConstraintPrefixes = []string{"//go:build", "// +build"}

const BlankLine = "BLANKLINE"

func (Go) Lex(src []byte) (Result, error) {
	docs, err := goDocSpans(src)
	if err != nil {
		return Result{}, err
	}
	comments, skeleton, err := goScan(src)
	if err != nil {
		return Result{}, err
	}
	blocks, trailing := assembleBlocks(LangGo, src, comments)
	for i, b := range blocks {
		if s, ok := spansHold(docs, b.Start); ok {
			blocks[i].Declaration = true
			blocks[i].PackageDoc = s.packageDoc
			blocks[i].DeclGroup = s.group
			blocks[i].DeclName = s.name
		}
	}
	return Result{Blocks: blocks, Trailing: trailing, Skeleton: skeleton}, nil
}

func goScan(src []byte) ([]rawComment, []Token, error) {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var failure error
	var s scanner.Scanner
	s.Init(file, src, func(pos token.Position, msg string) {
		if failure == nil {
			failure = fmt.Errorf("scan: %d:%d: %s", pos.Line, pos.Column, msg)
		}
	}, scanner.ScanComments)

	var comments []rawComment
	var skeleton []Token
	lastCodeLine := 0
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		// A //line directive remaps the reported line, so every position
		// this tool reports asks for the physical one (SPEC §2.3, #1133).
		line := file.PositionFor(pos, false).Line
		if tok == token.COMMENT {
			c := rawComment{
				start:     file.Offset(pos),
				startLine: line,
				style:     StyleLine,
				directive: goDirective(lit),
			}
			// go/scanner strips every carriage return from a comment literal,
			// so the literal's length is not the source range (#1133).
			c.end = goCommentEnd(src, c.start)
			c.text = string(src[c.start:c.end])
			c.endLine = line + strings.Count(c.text, "\n")
			if strings.HasPrefix(lit, "/*") {
				c.style = StyleBlock
			}
			// go/scanner emits the automatic semicolon at the comment's own
			// position, so a trailing comment shares a line with that token.
			c.ownLine = lastCodeLine != line
			comments = append(comments, c)
			if c.directive {
				skeleton = append(skeleton, Token{Kind: tok.String(), Text: lit, Line: line})
			}
			// A build constraint applies only when a blank line separates it
			// from the package clause, so that blank line is part of the
			// directive rather than layout (SPEC §5.1, #1133).
			if goConstraint(lit) && blankLineFollows(src, c.end) {
				skeleton = append(skeleton, Token{Kind: BlankLine, Line: c.endLine + 1})
			}
			continue
		}
		for i := len(comments) - 1; i >= 0 && comments[i].endLine == line; i-- {
			comments[i].ownLine = false
		}
		skeleton = append(skeleton, Token{Kind: tok.String(), Text: lit, Line: line})
		lastCodeLine = line
		if tok == token.STRING {
			lastCodeLine += strings.Count(lit, "\n")
		}
	}
	if failure != nil {
		return nil, nil, failure
	}
	return comments, skeleton, nil
}

func goDirective(text string) bool {
	return hasAnyPrefix(text, goDirectivePrefixes)
}

func goConstraint(text string) bool {
	return hasAnyPrefix(text, goConstraintPrefixes)
}

func hasAnyPrefix(text string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(text, p) {
			return true
		}
	}
	return false
}

func goCommentEnd(src []byte, start int) int {
	if start+1 < len(src) && src[start+1] == '*' {
		if i := bytes.Index(src[start:], []byte("*/")); i >= 0 {
			return start + i + 2
		}
		return len(src)
	}
	end := start
	for end < len(src) && src[end] != '\n' {
		end++
	}
	if end > start && src[end-1] == '\r' {
		end--
	}
	return end
}

func blankLineFollows(src []byte, end int) bool {
	i := end
	for i < len(src) && src[i] != '\n' {
		i++
	}
	if i == len(src) {
		return false
	}
	i++
	j := i
	for j < len(src) && src[j] != '\n' {
		j++
	}
	return strings.TrimSpace(string(src[i:j])) == ""
}

type span struct {
	start      int
	end        int
	name       string
	packageDoc bool
	group      bool
}

func goDocSpans(src []byte) ([]span, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	var spans []span
	add := func(g *ast.CommentGroup, name string, packageDoc, group bool) {
		if g == nil {
			return
		}
		spans = append(spans, span{
			start:      fset.PositionFor(g.Pos(), false).Offset,
			end:        fset.PositionFor(g.End(), false).Offset,
			name:       name,
			packageDoc: packageDoc,
			group:      group,
		})
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch d := n.(type) {
		case *ast.File:
			add(d.Doc, "", true, false)
		case *ast.GenDecl:
			add(d.Doc, genDeclName(d), false, d.Lparen.IsValid())
		case *ast.FuncDecl:
			add(d.Doc, d.Name.Name, false, false)
		case *ast.TypeSpec:
			add(d.Doc, d.Name.Name, false, false)
		case *ast.ValueSpec:
			add(d.Doc, firstName(d.Names), false, false)
		case *ast.ImportSpec:
			add(d.Doc, "", false, true)
		case *ast.Field:
			add(d.Doc, firstName(d.Names), false, false)
		}
		return true
	})
	return spans, nil
}

func genDeclName(d *ast.GenDecl) string {
	// go/ast hangs a one-spec decl's doc on the GenDecl, not on the spec, so
	// the name a §2.1 rule 7 comparison needs sits one node down.
	if d.Lparen.IsValid() || len(d.Specs) != 1 {
		return ""
	}
	switch s := d.Specs[0].(type) {
	case *ast.TypeSpec:
		return s.Name.Name
	case *ast.ValueSpec:
		return firstName(s.Names)
	}
	return ""
}

func firstName(names []*ast.Ident) string {
	if len(names) == 0 {
		return ""
	}
	return names[0].Name
}

func spansHold(spans []span, start int) (span, bool) {
	for _, s := range spans {
		// go/ast reports a group's end from the carriage-return-stripped
		// literal, so only the start offset is comparable (#1133).
		if start >= s.start && start < s.end {
			return s, true
		}
	}
	return span{}, false
}
