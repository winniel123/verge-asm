package queue

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// applyAvailability takes a *db.Queries, so no fake reaches it and no Postgres runs here.
// What can be held is the order of the two writes inside the batch transaction, and that
// order is the whole of the hazard, so this gate reads the tree.

func firstCallPos(t *testing.T, fn *ast.FuncDecl, name string) token.Pos {
	t.Helper()
	found := token.NoPos
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		id, ok := call.Fun.(*ast.Ident)
		if !ok || id.Name != name {
			return true
		}
		if !found.IsValid() || id.Pos() < found {
			found = id.Pos()
		}
		return true
	})
	if !found.IsValid() {
		t.Fatalf("Worker.complete calls no %s; the gate lost the tree, not the rule", name)
	}
	return found
}

func completeDecl(t *testing.T) *ast.FuncDecl {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "worker.go", nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse worker.go: %v", err)
	}
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if ok && fn.Recv != nil && fn.Name.Name == "complete" {
			return fn
		}
	}
	t.Fatal("worker.go declares no complete method")
	return nil
}

func TestRecoveryIsAppliedAfterTheBatchsOwnFold(t *testing.T) {
	// MarkVantageAvailable closes every open vantage-unavailable Gap. Run before the fold, it
	// would close the Gap this batch's own reading must close, and a Gap closing is the sole
	// carrier of the pair across it and of the rules that opened (ADR-2087, ADR-0014, #882).
	fn := completeDecl(t)
	fold := firstCallPos(t, fn, "foldObservationsIntoSpans")
	apply := firstCallPos(t, fn, "applyAvailability")

	if apply < fold {
		t.Errorf("Worker.complete applies availability before it folds the observations, so a "+
			"recovering batch closes its own Gaps silently and no GapClosed message fires "+
			"(ADR-2087, #882); applyAvailability at %d, foldObservationsIntoSpans at %d",
			apply, fold)
	}
}
