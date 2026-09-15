package queue

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

// What this gate proves, stated honestly (ADR-1945 §3).
//
// It proves that each named comparison below reaches an address-scope binding producer at exactly
// one site on its static call graph. It does not prove that both legs read the same value. A
// comparison that binds once, then rebinds the name before its second leg, passes here. ADR-1895
// §10's two per-site tests hold the value. This gate holds the shape, and neither replaces the
// other.
//
// Three coverage gaps, named rather than smoothed (ADR-1945 §3).
//
//  1. The root set is a list, not a derivation. A wholly new comparison escapes this gate,
//     because nobody adds its function to queueComparisonRoots. What the gate does change is the
//     failure mode of a stale entry: a root naming a function this package no longer declares
//     fails, rather than passing in silence.
//  2. It counts acquisition sites, never executions. The walk visits each function once, so a
//     helper that binds once and is called from two places reads as one site. A loop around one
//     acquisition reads as one site too.
//  3. Dynamic dispatch is invisible. A binding reached through a function value, a struct field
//     or an interface method is not a call on the static graph. Every site in this package calls
//     the producer directly today, and the gate reads that tree.
//
// The walk reads package-level functions alone. Every name in the three tables below is one, and
// the liveness test refuses an entry that is not.

// The one producer in this package. The gate keys on the name, so a rename that left this map
// behind would read every comparison as binding zero times, which the count check below refuses.

var queueBindingProducers = map[string]string{
	"coveredAddressScope": "reads the address-scope Seeds and the live exclusion corpus (ADR-0133 §4)",
}

// Each entry compares across time, so ADR-1945's rule governs its whole call graph. Each must
// bind exactly once: a count of zero means the gate lost the tree rather than that the rule holds.

var queueComparisonRoots = map[string]string{
	"readBatchLegs": "builds the fold's current and previous leg sets, bounded to its candidates",
	"buildMessages": "the fold, which passes one batchLegs to the flagship and to the widening",
}

// The walk stops at each entry, so an acquisition beneath one is not counted against the root
// that reached it. An entry that binds nothing has no work here, and the liveness test says so.

var queueOutsideTheSeam = map[string]string{
	"rulesOpenedByGapClose": "reads present state through legsFromCurrent and starts no comparison (ADR-1945 §2)",
}

type queuePkg struct {
	funcs map[string]*ast.FuncDecl
}

func parseQueuePackage(t *testing.T) *queuePkg {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read internal/queue: %v", err)
	}
	fset := token.NewFileSet()
	c := &queuePkg{funcs: map[string]*ast.FuncDecl{}}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, perr := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if perr != nil {
			t.Fatalf("parse %s: %v", name, perr)
		}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil {
				c.funcs[fn.Name.Name] = fn
			}
		}
	}
	if len(c.funcs) == 0 {
		t.Fatal("found no package-level function; the reader is broken, not the tree")
	}
	return c
}

func (c *queuePkg) calleesOf(fn *ast.FuncDecl) []string {
	var out []string
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if id, ok := call.Fun.(*ast.Ident); ok {
			if _, known := c.funcs[id.Name]; known {
				out = append(out, id.Name)
			}
		}
		return true
	})
	return out
}

func (c *queuePkg) bindingSites(root string, producers, stop map[string]string) []string {
	var sites []string
	seen := map[string]bool{}
	var walk func(string)
	walk = func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		fn, ok := c.funcs[name]
		if !ok {
			return
		}
		for _, callee := range c.calleesOf(fn) {
			if _, isProducer := producers[callee]; isProducer {
				sites = append(sites, name+" → "+callee)
			}
			if _, exempt := stop[callee]; exempt {
				continue
			}
			walk(callee)
		}
	}
	walk(root)
	sort.Strings(sites)
	return sites
}

func TestEveryQueueComparisonBindsTheAddressScopeOnce(t *testing.T) {
	c := parseQueuePackage(t)
	for root, why := range queueComparisonRoots {
		sites := c.bindingSites(root, queueBindingProducers, queueOutsideTheSeam)
		switch {
		case len(sites) == 1:
		case len(sites) == 0:
			t.Errorf("%s (%s) reaches no address-scope binding (ADR-1945 §2).\n"+
				"Either the comparison no longer classifies, and the entry must go, or the\n"+
				"producer moved out of queueBindingProducers and this gate now proves nothing.",
				root, why)
		default:
			t.Errorf("%s (%s) acquires %d address-scope bindings (ADR-1945 §2):\n  %s\n"+
				"Every leg of one comparison classifies under one binding. Build it once and\n"+
				"carry it on batchLegs, the way readBatchLegs does.",
				root, why, len(sites), strings.Join(sites, "\n  "))
		}
	}
}

// An entry naming a function the package no longer declares would sit here reading as compliance,
// so the gate asserts all three tables against the live tree. An exemption is asserted twice: the
// function must exist, and it must still acquire a binding, or the walk stops at it for nothing.

func TestQueueBindingGateNamesAreLive(t *testing.T) {
	c := parseQueuePackage(t)
	for name := range queueBindingProducers {
		if _, ok := c.funcs[name]; !ok {
			t.Errorf("queueBindingProducers names %q, which this package no longer declares as a "+
				"package-level function; re-point the entry at the producer's new name", name)
		}
	}
	for name := range queueComparisonRoots {
		if _, ok := c.funcs[name]; !ok {
			t.Errorf("queueComparisonRoots names %q, which this package no longer declares as a "+
				"package-level function; delete the entry, or re-point it at the comparison's new name", name)
		}
	}
	for name, why := range queueOutsideTheSeam {
		if _, ok := c.funcs[name]; !ok {
			t.Errorf("queueOutsideTheSeam names %q, which this package no longer declares as a "+
				"package-level function; delete the entry", name)
			continue
		}
		// A direct-callee scan would fire the day the acquisition moved one hop down, and
		// obeying it would fail the count test at buildMessages. The two must not disagree.
		if len(c.bindingSites(name, queueBindingProducers, map[string]string{})) == 0 {
			t.Errorf("queueOutsideTheSeam names %q (%s), which reaches no address-scope binding; "+
				"the walk stops there for nothing, so delete the entry", name, why)
		}
	}
}
