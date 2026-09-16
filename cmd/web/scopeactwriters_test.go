package main

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
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
// It is per route, not per write. It asks whether some panel class sits on the same live call
// graph as some writer, so a handler holding two writes and one Record passes, and so does one
// whose Record sits on a branch the write never reaches. It does not prove the Record runs, runs
// once, runs after the write, or carries that write's own scope. Ordering, cardinality and
// content are held by review and by the unit tests in scopeacts_test.go.
//
// Four coverage gaps, named rather than smoothed.
//
//  1. A raw SQL write reaches the predicate through no query in this map, and the tree holds two.
//     restore.applied replaces the whole corpus, so it moves addressScopeCovered wholesale, and it
//     carries a RestoreRef rather than a scope. ADR-2171 §6 rules the principle and not the next
//     class found, so this gate does not claim it. seedFixtureAddressScope inserts an address Seed
//     from the dev-only --seed-fixtures flag, off every route, and a reseed is configuration
//     rather than an operator act, so it owes no Act and no row.
//  2. It keys on the bare selector name, the way the Record gate does. The hazard is the liveness
//     check rather than the width: a stray call of the same name would hold called[writer] true
//     after the real query left the route graph. The token-freeness test below is what keeps that
//     premise, as TestTheRecordTokenStaysFreeInThisPackage keeps the Record gate's.
//  3. The route walk is static and shares reach's limits: dynamic dispatch is invisible, actStop's
//     refusal helpers are not descended into, and calleesOf follows only an s.X call. The sweep
//     below reads every function declaration in the package, on any receiver and without reach,
//     so each of those three reddens naming the declaration rather than passing in silence.
//  4. A writer inside an `if s.devMode` body is pruned from the route walk by inspectLive, so it
//     reaches no route. The sweep does not prune, which is what turns that into a failure too,
//     except inside a declaration that already holds a live writer: the sweep keys per
//     declaration rather than per call site, so that one declaration is already on a route.
//     A writer inside a package-level func value is in neither the sweep nor c.funcs, and stays
//     invisible to both.

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

func actTypeClasses(t *testing.T) map[string]string {
	t.Helper()
	out := make(map[string]string, len(act.Classes))
	for _, a := range act.Classes {
		name := reflect.TypeOf(a).Name()
		if name == "" {
			// An unnamed entry keys the map at "", which is what every unknown literal reads
			// back as, so one pointer in act.Classes would pass every route unconditionally.
			t.Fatalf("act.Classes holds an unnamed type for %q; every class lookup here would "+
				"resolve through it", a.Class())
		}
		out[name] = a.Class()
	}
	return out
}

func writerCalls(fn *ast.FuncDecl, walk func(*ast.FuncDecl, func(ast.Node) bool)) []string {
	var out []string
	walk(fn, func(n ast.Node) bool {
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

// The route walk prunes what reach prunes, so both halves count the same tree.

func scopeWritersIn(fn *ast.FuncDecl) []string { return writerCalls(fn, inspectLive) }

// The sweep prunes nothing, so a dev-mode write reddens rather than reading as compliance.

func scopeWritersAnywhereIn(fn *ast.FuncDecl) []string {
	return writerCalls(fn, func(d *ast.FuncDecl, f func(ast.Node) bool) { ast.Inspect(d, f) })
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
	stop := actStop()
	for pattern, handler := range c.routes {
		var site scopeWriteSite
		c.reach("s."+handler, stop, func(_ string, fn *ast.FuncDecl) {
			site.writers = append(site.writers, scopeWritersIn(fn)...)
			site.classes = append(site.classes, actLiteralsIn(fn)...)
		})
		if len(site.writers) > 0 {
			out[pattern] = site
		}
	}
	return out
}

func (c *contractPkg) routesMovingScopeWithNoPanelClass(classOf map[string]string) []string {
	var bad []string
	for pattern, site := range c.scopeWriteSites() {
		var rendered bool
		for _, lit := range site.classes {
			class, known := classOf[lit]
			if !known {
				continue
			}
			if _, reads := addressScopeActVerbs[class]; reads {
				rendered = true
				break
			}
		}
		if !rendered {
			bad = append(bad, pattern+" writes "+strings.Join(uniqSorted(site.writers), ", "))
		}
	}
	sort.Strings(bad)
	return bad
}

func TestEveryAddressScopeWriteRecordsAClassThePanelReads(t *testing.T) {
	c := parseWebPackage(t)
	if bad := c.routesMovingScopeWithNoPanelClass(actTypeClasses(t)); len(bad) > 0 {
		t.Errorf("these routes move addressScopeCovered and record no class the Exposure panel "+
			"reads (ADR-2114, ADR-2171 §2, #2205):\n  %s\n"+
			"An operator would see the figure move beside a panel reporting no address-scope edit.\n"+
			"Record one of the classes in addressScopeActVerbs, or add the new class there and give\n"+
			"addressScopeOf a case for it.",
			strings.Join(bad, "\n  "))
	}
}

// Removing the Record is the defect the payload guard cannot see, so the gate proves it reddens.

func TestTheScopeWriteGateCatchesARemovedAndAMisdirectedRecord(t *testing.T) {
	classOf := actTypeClasses(t)
	const wrote = "POST /wired writes CreateAddressExclusion"

	wired := fixturePkg(t, scopeWriteFixture(scopeActRecordCall)).routesMovingScopeWithNoPanelClass(classOf)
	if len(wired) != 0 {
		t.Errorf("the wired fixture reports %v, want none: the gate reddens on a compliant route", wired)
	}

	removed := fixturePkg(t, scopeWriteFixture("")).routesMovingScopeWithNoPanelClass(classOf)
	if !contains(removed, wrote) {
		t.Errorf("removing the Record left POST /wired passing; the gate proves nothing: %v", removed)
	}

	// A Record alone is not the rule. The class has to be one the panel reads (ADR-2114).
	offPanel := fixturePkg(t, scopeWriteFixture(
		"s.recorder().Record(r.Context(), actingAccount(acct), act.ScanTriggered{})")).
		routesMovingScopeWithNoPanelClass(classOf)
	if !contains(offPanel, wrote) {
		t.Errorf("recording a class outside addressScopeActVerbs passed; the gate reads any act: %v",
			offPanel)
	}
}

const scopeActRecordCall = `s.recorder().Record(r.Context(), actingAccount(acct), ` +
	`act.ExclusionDeclared{ExclusionRef: act.ExclusionRef{Kind: "address", Scope: "192.0.2.0/24"}})`

func scopeWriteFixture(record string) string {
	return `package main

func (s *server) mount(mux *http.ServeMux) {
	mux.HandleFunc("POST /wired", s.requireAdmin(s.wired))
	mux.HandleFunc("POST /reads", s.requireAdmin(s.reads))
}

func (s *server) wired(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.write(r)
	` + record + `
}

func (s *server) write(r *http.Request) {
	s.exclusionsStore.CreateAddressExclusion(r.Context(), db.CreateAddressExclusionParams{})
}

func (s *server) reads(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.exclusionsStore.ListExclusions(r.Context())
}
`
}

// A zero count everywhere reads as compliance, so the gate proves it still finds the writes.

func TestTheAddressScopeWriterSetIsLive(t *testing.T) {
	c := parseWebPackage(t)

	called := map[string]bool{}
	onARoute := map[string]bool{}
	stop := actStop()
	for _, handler := range c.routes {
		c.reach("s."+handler, stop, func(_ string, fn *ast.FuncDecl) {
			for _, w := range scopeWritersIn(fn) {
				called[w] = true
				// parseWebPackage and webFuncDecls parse the same files under the same bare
				// names, so a position is the one key that survives the second parse.
				onARoute[c.fset.Position(fn.Pos()).String()] = true
			}
		})
	}
	var dead []string
	for writer, why := range addressScopeWriters {
		if !called[writer] {
			dead = append(dead, writer+" ("+why+")")
		}
	}
	if len(dead) > 0 {
		t.Errorf("no route reaches these:\n  %s\n"+
			"Either the query is gone and the entry must go, or it moved off the route graph\n"+
			"and this gate now proves nothing about it.", strings.Join(uniqSorted(dead), "\n  "))
	}

	// A write on a second receiver passes both gates, so this sweep re-parses the package.

	var offRoute []string
	fset, decls := webFuncDecls(t)
	for _, fn := range decls {
		at := fset.Position(fn.Pos())
		if len(scopeWritersAnywhereIn(fn)) > 0 && !onARoute[at.String()] {
			offRoute = append(offRoute, declLabel(fset, fn)+" at "+at.String())
		}
	}
	if len(offRoute) > 0 {
		t.Errorf("these move addressScopeCovered off every route's call graph, so the gate above\n"+
			"cannot tie them to a panel class (#2205):\n  %s\n"+
			"Root the write in a handler, or widen the gate deliberately and say what it now reads.",
			strings.Join(uniqSorted(offRoute), "\n  "))
	}
}

// The gate reads one selector name, so a free token is what makes it sound. A stray call of the
// same name would hold the liveness check green after the real query left the route graph.

func TestTheAddressScopeWriterTokensStayFreeInThisPackage(t *testing.T) {
	fset, decls := webFuncDecls(t)
	var foreign []string
	for _, fn := range decls {
		ast.Inspect(fn, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if _, isWriter := addressScopeWriters[sel.Sel.Name]; !isWriter || fromServerStore(sel.X) {
				return true
			}
			foreign = append(foreign, declLabel(fset, fn)+" calls "+sel.Sel.Name+
				" at "+fset.Position(call.Pos()).String())
			return true
		})
	}
	if len(foreign) > 0 {
		t.Errorf("these hold an address-scope writer token on no server store (#2205):\n  %s\n"+
			"The gate keys on the bare method name. Rename the other call, or widen\n"+
			"addressScopeWriters deliberately and say in the PR what the gate now reads.",
			strings.Join(uniqSorted(foreign), "\n  "))
	}
}

func fromServerStore(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == "s"
}

// parseWebPackage keeps only *server methods and package-level funcs, so the sweep reads the
// files again rather than its maps.

func webFuncDecls(t *testing.T) (*token.FileSet, []*ast.FuncDecl) {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read cmd/web: %v", err)
	}
	fset := token.NewFileSet()
	var out []*ast.FuncDecl
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
			if fn, ok := d.(*ast.FuncDecl); ok {
				out = append(out, fn)
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("found no function declarations; the sweep is broken, not the tree")
	}
	return fset, out
}

func declLabel(fset *token.FileSet, fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) != 1 {
		return fn.Name.Name
	}
	var recv strings.Builder
	if err := printer.Fprint(&recv, fset, fn.Recv.List[0].Type); err != nil {
		return fn.Name.Name
	}
	return "(" + recv.String() + ")." + fn.Name.Name
}
