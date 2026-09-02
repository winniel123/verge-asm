package surface

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"strings"
)

type Go struct{}

var goDirectivePrefixes = []string{"//go:", "// +build", "//nolint", "//lint:", "//revive:"}

func (Go) Lex(src []byte) (Result, error) {
	docs, err := goDocSpans(src)
	if err != nil {
		return Result{}, err
	}
	comments, skeleton, err := goScan(src)
	if err != nil {
		return Result{}, err
	}
	blocks := goBlocks(src, comments)
	for i, b := range blocks {
		blocks[i].Declaration = spansCover(docs, b.Start, b.End)
	}
	return Result{Blocks: blocks, Skeleton: skeleton}, nil
}

type goComment struct {
	start     int
	end       int
	startLine int
	endLine   int
	text      string
	style     Style
	ownLine   bool
	directive bool
}

func goScan(src []byte) ([]goComment, []Token, error) {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var failure error
	var s scanner.Scanner
	s.Init(file, src, func(pos token.Position, msg string) {
		if failure == nil {
			failure = fmt.Errorf("scan: %d:%d: %s", pos.Line, pos.Column, msg)
		}
	}, scanner.ScanComments)

	var comments []goComment
	var skeleton []Token
	lastCodeLine := 0
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		line := file.Position(pos).Line
		if tok == token.COMMENT {
			c := goComment{
				start:     file.Offset(pos),
				startLine: line,
				endLine:   line + strings.Count(lit, "\n"),
				text:      lit,
				style:     StyleLine,
				directive: goDirective(lit),
			}
			c.end = c.start + len(lit)
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
	for _, p := range goDirectivePrefixes {
		if strings.HasPrefix(text, p) {
			return true
		}
	}
	return false
}

func goBlocks(src []byte, comments []goComment) []Block {
	var blocks []Block
	open := -1
	for _, c := range comments {
		if !c.ownLine {
			open = -1
			continue
		}
		joins := open >= 0 && !c.directive && c.style == StyleLine && blocks[open].EndLine+1 == c.startLine
		if joins {
			blocks[open].End = c.end
			blocks[open].EndLine = c.endLine
			blocks[open].Text = string(src[blocks[open].Start:c.end])
			continue
		}
		blocks = append(blocks, Block{
			Style:     c.style,
			Start:     c.start,
			End:       c.end,
			StartLine: c.startLine,
			EndLine:   c.endLine,
			Text:      c.text,
			Directive: c.directive,
		})
		open = len(blocks) - 1
		if c.directive || c.style == StyleBlock {
			open = -1
		}
	}
	return blocks
}

type span struct {
	start int
	end   int
}

func goDocSpans(src []byte) ([]span, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	var spans []span
	add := func(g *ast.CommentGroup) {
		if g == nil {
			return
		}
		spans = append(spans, span{fset.Position(g.Pos()).Offset, fset.Position(g.End()).Offset})
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch d := n.(type) {
		case *ast.File:
			add(d.Doc)
		case *ast.GenDecl:
			add(d.Doc)
		case *ast.FuncDecl:
			add(d.Doc)
		case *ast.TypeSpec:
			add(d.Doc)
		case *ast.ValueSpec:
			add(d.Doc)
		case *ast.ImportSpec:
			add(d.Doc)
		case *ast.Field:
			add(d.Doc)
		}
		return true
	})
	return spans, nil
}

func spansCover(spans []span, start, end int) bool {
	for _, s := range spans {
		if start >= s.start && end <= s.end {
			return true
		}
	}
	return false
}
