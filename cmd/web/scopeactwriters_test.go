package main

import (
	"go/ast"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/act"
)

// What this gate proves, stated honestly (#2205).
//
// TestEveryScopeShapedActClassIsInThePanel reads the act payload, so it catches a new class that
// carries a SeedScope or an ExclusionRef. It cannot see the write. This gate reads the other end:
// every route that reaches a query which moves addressScopeCovered must also construct an act
// class the Exposure panel reads. A handler that writes an address Seed or an address exclusion
// and records some other class fails here, and passes the payload guard.
//
// It does not prove the Record runs, runs once, runs after the write, or carries that write's own
// scope. It proves the class is on the same live call graph as the write.
//
// Three coverage gaps, named rather than smoothed.
//
//  1. A raw SQL write reaches the predicate through no query in this map, and the tree holds two.
//     restore.applied replaces the whole corpus, so it moves addressScopeCovered wholesale, and it
//     carries a RestoreRef rather than a scope. ADR-2171 §6 reserves whether the panel owes it a
//     row, so this gate does not claim it. seedFixtureAddressScope inserts an address Seed from
//     the dev-only --seed-fixtures flag, off every route, and a reseed is configuration rather
//     than an operator act, so it owes no Act and no row.
//  2. It keys on the bare selector name, the way the Record gate does. A second query of the same
//     name in this package would widen the gate silently.
//  3. The walk is static and shares reach's limits: dynamic dispatch is invisible, an
//     `if s.devMode` body is pruned, and actStop's refusal helpers are not descended into. The
//     writer-sits-on-a-route check below is what turns a lost tree into a failure rather than a
//     silent pass.

// Each query moves addressScopeCovered, through ListAddressScopeCidrs or through ADR-0133 §4's
// exclusion subtraction. The gate keys on the name, so a rename that left this map behind would
// read every handler as writing nothing, which the liveness check below refuses.

var addressScopeWriters = map[string]string{
	"CreateAddressSeed":               "an address Seed enters ListAddressScopeCidrs",
	"WithdrawSeed":                    "a withdrawn address Seed leaves ListAddressScopeCidrs",
	"CreateAddressExclusion":          "an address exclusion subtracts from covered (ADR-0133 §4)",
	"CreateDeclinedProposalExclusion": "a decline writes an address exclusion (ADR-2171 §2)",
	"DeleteExclusion":                 "lifting an address exclusion returns the range",
	"DeleteUnclaimedAddressExclusion": "un-declining removes the decline's exclusion (ADR-2171 §2)",
}

func actTypeClasses() map[string]string {
	out := make(map[string]string, len(act.Classes))
	for _, a := range act.Classes {
		out[reflect.TypeOf(a).Name()] = a.Class()
	}
	return out
}

func scopeWritersIn(fn *ast.FuncDecl) []string {
	var out []string
	inspectLive(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if _, isWriter := addressScopeWriters[sel.Sel.Name]; isWriter {
			out = append(out, sel.Sel.Name)
		}
		return true
	})
	return out
}

func actLiteralsIn(fn *ast.FuncDecl) []string {
	var out []string
	inspectLive(fn, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		sel, ok := lit.Type.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == "act" {
			out = append(out, sel.Sel.Name)
		}
		return true
	})
	return out
}

type scopeWriteSite struct {
	writers []string
	classes []string
}

func (c *contractPkg) scopeWriteSites() map[string]scopeWriteSite {
	out := map[string]scopeWriteSite{}
	for pattern, handler := range c.routes {
		var site scopeWriteSite
		c.reach("s."+handler, actStop(), func(_ string, fn *ast.FuncDecl) {
			site.writers = append(site.writers, scopeWritersIn(fn)...)
			site.classes = append(site.classes, actLiteralsIn(fn)...)
		})
		if len(site.writers) > 0 {
			out[pattern] = site
		}
	}
	return out
}

func TestEveryAddressScopeWriteRecordsAClassThePanelReads(t *testing.T) {
	c := parseWebPackage(t)
	classOf := actTypeClasses()

	var bad []string
	for pattern, site := range c.scopeWriteSites() {
		var rendered bool
		for _, lit := range site.classes {
			if _, reads := addressScopeActVerbs[classOf[lit]]; reads {
				rendered = true
				break
			}
		}
		if !rendered {
			bad = append(bad, pattern+" writes "+strings.Join(uniqSorted(site.writers), ", "))
		}
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		t.Errorf("these routes move addressScopeCovered and record no class the Exposure panel "+
			"reads (ADR-2114, ADR-2171 §6, #2205):\n  %s\n"+
			"An operator would see the figure move beside a panel reporting no address-scope edit.\n"+
			"Record one of the classes in addressScopeActVerbs, or add the new class there and give\n"+
			"addressScopeOf a case for it.",
			strings.Join(bad, "\n  "))
	}
}

// A zero count everywhere reads as compliance, so the gate proves it still finds the writes.

func TestTheAddressScopeWriterSetIsLive(t *testing.T) {
	c := parseWebPackage(t)

	called := map[string]bool{}
	onARoute := map[string]bool{}
	for _, handler := range c.routes {
		c.reach("s."+handler, actStop(), func(name string, fn *ast.FuncDecl) {
			for _, w := range scopeWritersIn(fn) {
				called[w] = true
				onARoute[name] = true
			}
		})
	}
	for writer, why := range addressScopeWriters {
		if !called[writer] {
			t.Errorf("no route reaches %s (%s).\n"+
				"Either the query is gone and the entry must go, or it moved off the route graph\n"+
				"and this gate now proves nothing about it.", writer, why)
		}
	}

	var offRoute []string
	visit := func(owner string, fn *ast.FuncDecl) {
		if len(scopeWritersIn(fn)) > 0 && !onARoute[owner] {
			offRoute = append(offRoute, owner)
		}
	}
	for name, fn := range c.methods {
		visit(name, fn)
	}
	for name, fn := range c.funcs {
		visit(name, fn)
	}
	if len(offRoute) > 0 {
		t.Errorf("these move addressScopeCovered off every route's call graph, so the gate above\n"+
			"cannot tie them to a panel class (#2205):\n  %s\n"+
			"Root the write in a handler, or widen the gate deliberately and say what it now reads.",
			strings.Join(uniqSorted(offRoute), "\n  "))
	}
}
