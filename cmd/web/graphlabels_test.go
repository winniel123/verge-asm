package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/winniel123/verge-asm/internal/db"
)

var viewportScale = regexp.MustCompile(`id="gr-viewport" transform="translate\([^"]*\) scale\(([0-9.]+)\)"`)

func labelColumn(n int) []graphNode {
	// Mirrors buildGraph's own column placement, so a threshold assertion reads real geometry.
	nodes := make([]graphNode, 0, n)
	for i := 0; i < n; i++ {
		nodes = append(nodes, graphNode{
			Type: "subdomain", X: graphColName, Y: graphRowTop + i*graphRowStep,
		})
	}
	return nodes
}

func labelColumnFitScale(t *testing.T, n int) float64 {
	t.Helper()
	w, h := graphContentBounds(labelColumn(n))
	_, _, k, _ := graphFit(w, h)
	return k
}

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

func TestGraphPageAtTheCapOpensAboveTheLabelThreshold(t *testing.T) {
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
	m := viewportScale.FindStringSubmatch(page)
	if m == nil {
		t.Fatalf("graph page has no viewport transform to read a scale from; body: %s", page)
	}
	k, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		t.Fatalf("viewport scale %q does not parse: %v", m[1], err)
	}
	if k < graphLabelMinK {
		t.Errorf("a capped drawing opens at scale %v, under the %v the view hides labels below: the cap drew a column its own threshold hushes",
			k, graphLabelMinK)
	}
}
