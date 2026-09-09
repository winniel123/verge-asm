package main

// Throwaway prototype for #1791. Lands nothing.
//
// #1788 §1 rules that the recorder call sits AFTER the mutation, and bars recorder-before outright.
// This measures whether an AST rule can hold that ordering.

import (
	"go/ast"
	"go/token"
	"sort"
	"strings"
	"testing"
)

type posCall struct {
	name     string
	pos      token.Pos
	deferred bool
}

// orderedCallsIn returns the mutating store calls and the Record calls in one body, by position.
func orderedCallsIn(fn *ast.FuncDecl, mut map[string]string) (writes, records []posCall) {
	deferred := map[ast.Node]bool{}
	ast.Inspect(fn, func(n ast.Node) bool {
		if d, ok := n.(*ast.DeferStmt); ok {
			deferred[d.Call] = true
		}
		return true
	})
	inspectLive(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if sel.Sel.Name == "Record" {
			records = append(records, posCall{"Record", call.Pos(), deferred[call]})
			return true
		}
		inner, ok := sel.X.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := inner.X.(*ast.Ident); !ok || id.Name != "s" {
			return true
		}
		if _, isMut := mut[sel.Sel.Name]; isMut {
			writes = append(writes, posCall{sel.Sel.Name, call.Pos(), deferred[call]})
		}
		return true
	})
	return writes, records
}

func TestProtoRecorderSitsAfterTheMutation(t *testing.T) {
	c := parseWebPackage(t)
	mut := mutatingQueries(t)
	aud := auditableRoutes(t)

	var checked, sameFunc, split, violations int
	var notes []string
	for pattern, handler := range aud {
		var hasRecord, writeElsewhere bool
		var localBad []string
		c.reach("s."+handler, actStop(), func(name string, fn *ast.FuncDecl) {
			writes, records := orderedCallsIn(fn, mut)
			if len(records) > 0 {
				hasRecord = true
			}
			if len(writes) > 0 && len(records) == 0 {
				writeElsewhere = true
			}
			if len(records) == 0 || len(writes) == 0 {
				return
			}
			sameFunc++
			for _, rec := range records {
				if rec.deferred {
					continue
				}
				for _, w := range writes {
					if rec.pos < w.pos {
						localBad = append(localBad, name+": Record before "+w.name)
					}
				}
			}
		})
		if !hasRecord {
			continue
		}
		checked++
		if writeElsewhere {
			split++
			notes = append(notes, "SPLIT "+pattern+" ("+handler+"): the mutation and the Record sit in different functions")
		}
		if len(localBad) > 0 {
			violations++
			notes = append(notes, "ORDER "+pattern+" ("+handler+"): "+strings.Join(uniqSorted(localBad), "; "))
		}
	}
	sort.Strings(notes)
	for _, n := range notes {
		t.Logf("%s", n)
	}
	t.Logf("")
	t.Logf("auditable routes carrying a Record: %d", checked)
	t.Logf("  bodies where a mutation and a Record share one function: %d", sameFunc)
	t.Logf("  routes where they sit in DIFFERENT functions (a lexical rule is blind): %d", split)
	t.Logf("  routes with a lexical Record-before-mutation: %d", violations)
}
