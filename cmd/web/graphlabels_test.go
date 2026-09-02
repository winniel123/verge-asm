package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/winniel123/verge-asm/internal/db"
)

// column places n subdomain nodes down the name column exactly as buildGraph does, so
// a threshold assertion reads the real geometry rather than a restatement of it.
func labelColumn(n int) []graphNode {
	nodes := make([]graphNode, 0, n)
	for i := 0; i < n; i++ {
		nodes = append(nodes, graphNode{
			Type: "subdomain", X: graphColName, Y: graphRowTop + i*graphRowStep,
		})
	}
	return nodes
}

// fitScale is the scale the view opens a column of n at.
func labelColumnFitScale(t *testing.T, n int) float64 {
	t.Helper()
	w, h := graphContentBounds(labelColumn(n))
	_, _, k, _ := graphFit(w, h)
	return k
}

// #1104 (ADR-0136 §5): the suppression threshold and the column cap are one constraint,
// so the cap must be the largest column the threshold still admits. A cap that drew a
// column whose labels the same view then hides would be the drift the ADR forbids.
func TestGraphColumnCapIsTheLabelThresholdSolvedForN(t *testing.T) {
	if graphColumnCap != 20 {
		t.Fatalf("graphColumnCap = %d, want 20 (ADR-0136 §4)", graphColumnCap)
	}
	if got := float64(graphLabelPx) * labelColumnFitScale(t, graphColumnCap); got < graphLabelMinPx {
		t.Errorf("a full column renders its labels at %.2fpx, under the %dpx threshold", got, graphLabelMinPx)
	}
	if got := float64(graphLabelPx) * labelColumnFitScale(t, graphColumnCap+1); got >= graphLabelMinPx {
		t.Errorf("a column of %d renders its labels at %.2fpx, so the cap is not the largest one that clears %dpx",
			graphColumnCap+1, got, graphLabelMinPx)
	}
}

// The view hides a label below graphLabelMinK. The threshold rounds UP, so it never
// leaves a label drawn under graphLabelMinPx.
func TestGraphLabelMinKHidesNothingStillLegible(t *testing.T) {
	if got := float64(graphLabelPx) * graphLabelMinK; got < graphLabelMinPx {
		t.Errorf("at the threshold a label renders at %.4fpx, under the %dpx it promises", got, graphLabelMinPx)
	}
	if got := float64(graphLabelPx) * (graphLabelMinK - 1e-4); got >= graphLabelMinPx {
		t.Errorf("just under the threshold a label still renders at %.4fpx, so the threshold is too high", got)
	}
	if graphLabelMinK >= 1 {
		t.Errorf("graphLabelMinK = %v, want a scale under 1 so scale 1 draws every label", graphLabelMinK)
	}
}

// A drawing that fits carries the threshold and opens above it, so every label renders
// exactly as it did before this change.
func TestGraphPageDrawsEveryLabelWhenItFits(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.5"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph", http.StatusOK)

	for _, want := range []string{
		`#gr-svg.gr-hushed .gnode:not([data-type="domain"]) text { display: none; }`,
		"var LMINK =  0.6364 ;",
		`svg.classList.toggle("gr-hushed", view.k < LMINK);`,
		`if (hushed) svg.classList.add("gr-hushed");`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("graph page missing the label suppression %q; body: %s", want, page)
		}
	}
	if !strings.Contains(page, `transform="translate(0,0) scale(1)"`) {
		t.Errorf("graph page did not open at scale 1, so it hides labels a fitting drawing should draw; body: %s", page)
	}
}

// A capped drawing opens fit to its content, which is below the threshold, so the view
// hushes at load. The class is client-side, so the server still emits every label: the
// assertion is that the drawing opens under LMINK and the rule that hides them is present.
func TestGraphPageOpensUnderTheLabelThresholdWhenCapped(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	if _, err := f.CreateNameSeed(context.Background(), db.CreateNameSeedParams{
		NameDomain: pgtype.Text{String: "example.com", Valid: true}, CreatedBy: admin.ID,
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 25; i++ {
		f.addResolution(t, admin.ID, fmt.Sprintf("n%02d.example.com", i), "dns", obsClock,
			`{"outcome":"Resolved","addresses":["203.0.113.5"]}`)
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph", http.StatusOK)

	if !strings.Contains(page, "var LMINK =  0.6364 ;") {
		t.Fatalf("graph page carries no label threshold; body: %s", page)
	}
	if !strings.Contains(page, "n19.example.com") {
		t.Errorf("graph page dropped a label the cap kept; body: %s", page)
	}
}
