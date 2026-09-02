package verify

import (
	"fmt"

	"github.com/winniel123/verge-asm/internal/commentlint/scope"
	"github.com/winniel123/verge-asm/internal/commentlint/surface"
)

type Status int

const (
	Clean Status = iota
	Changed
	LexFailed
	Refused
	Skipped
)

func (s Status) String() string {
	switch s {
	case Clean:
		return "clean"
	case Changed:
		return "changed"
	case LexFailed:
		return "lex failed"
	case Refused:
		return "refused"
	}
	return "skipped"
}

type Finding struct {
	Path   string
	Status Status
	Detail string
}

type Report struct {
	Findings []Finding
}

func (r Report) Count(s Status) int {
	n := 0
	for _, f := range r.Findings {
		if f.Status == s {
			n++
		}
	}
	return n
}

func (r Report) Exit() int {
	if r.Count(LexFailed) > 0 || r.Count(Refused) > 0 {
		return 2
	}
	if r.Count(Changed) > 0 {
		return 1
	}
	return 0
}

type ReadFunc func(name string) ([]byte, error)

func Run(paths []string, inScopeOnly bool, base, head ReadFunc) Report {
	var r Report
	for _, p := range paths {
		r.Findings = append(r.Findings, check(p, inScopeOnly, base, head))
	}
	return r
}

func check(p string, inScopeOnly bool, base, head ReadFunc) Finding {
	switch scope.Classify(p) {
	case scope.Refused:
		if inScopeOnly {
			return Finding{Path: p, Status: Skipped, Detail: "outside the sweep's surfaces"}
		}
		return Finding{Path: p, Status: Refused, Detail: "a sweep PR changes no .html or .astro file"}
	case scope.OutOfScope:
		return Finding{Path: p, Status: Skipped, Detail: "outside the sweep's surfaces"}
	}

	lexer, err := surface.For(p)
	if err != nil {
		return Finding{Path: p, Status: LexFailed, Detail: err.Error()}
	}
	baseSrc, err := base(p)
	if err != nil {
		return Finding{Path: p, Status: Changed, Detail: fmt.Sprintf("no content at the base ref: %v", err)}
	}
	headSrc, err := head(p)
	if err != nil {
		return Finding{Path: p, Status: Changed, Detail: fmt.Sprintf("no content in the working tree: %v", err)}
	}
	baseResult, err := lexer.Lex(baseSrc)
	if err != nil {
		return Finding{Path: p, Status: LexFailed, Detail: "base: " + err.Error()}
	}
	headResult, err := lexer.Lex(headSrc)
	if err != nil {
		return Finding{Path: p, Status: LexFailed, Detail: "head: " + err.Error()}
	}
	if detail := compare(baseResult.Skeleton, headResult.Skeleton); detail != "" {
		return Finding{Path: p, Status: Changed, Detail: detail}
	}
	return Finding{Path: p, Status: Clean}
}

func compare(base, head []surface.Token) string {
	for i := range base {
		if i >= len(head) {
			return fmt.Sprintf("the skeleton lost %s (base line %d)", base[i], base[i].Line)
		}
		if !base[i].Equal(head[i]) {
			return fmt.Sprintf("line %d: the skeleton holds %s where the base holds %s", head[i].Line, head[i], base[i])
		}
	}
	if len(head) > len(base) {
		next := head[len(base)]
		return fmt.Sprintf("line %d: the skeleton gained %s", next.Line, next)
	}
	return ""
}
