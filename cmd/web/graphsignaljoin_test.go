package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

const graphSignalsDidNotResolve = "The signal read did not resolve on this load."

func TestGraphPageNamesAFailedSignalJoin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addClassResolution(t, "leak.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["10.0.0.5"]}`)
	corpusReadFails(f)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph", http.StatusOK)

	if !strings.Contains(page, `id="gr-svg"`) {
		t.Fatalf("a failed signal read took the graph away; body: %s", page)
	}
	if !strings.Contains(page, graphSignalsDidNotResolve) {
		t.Errorf("the graph does not name the failed signal read; body: %s", page)
	}
	if strings.Contains(page, `class="gnode-halo"`) {
		t.Errorf("a failed read lit a node; body: %s", page)
	}
	if strings.Contains(page, "non-globally-reachable-address-resolved-from-internet") {
		t.Errorf("a failed corpus read listed a rule anyway; body: %s", page)
	}
	if strings.Contains(page, "No open signals on this node.") {
		t.Errorf("a failed read still asserts that no signal is open on a node; body: %s", page)
	}
	if strings.Contains(page, "All severities") {
		t.Errorf("a failed read left the severity filter, which sorts nodes by a verdict nothing read; body: %s", page)
	}
	for _, sev := range []string{"critical", "high", "medium", "low", "info"} {
		if canvas := graphCanvasOf(t, page); strings.Contains(canvas, `data-sev="`+sev+`"`) {
			t.Errorf("a failed read gave a node the severity %q; canvas: %s", sev, canvas)
		}
	}
}

func graphCanvasOf(t *testing.T, page string) string {
	t.Helper()
	// The severity filter's options carry data-sev too, so a node's verdict is only readable inside the plot.
	from := strings.Index(page, `<svg id="gr-svg"`)
	if from < 0 {
		t.Fatalf("graph page carries no canvas, so its nodes cannot be isolated; body: %s", page)
	}
	canvas := page[from:]
	if end := strings.Index(canvas, "</svg>"); end >= 0 {
		canvas = canvas[:end]
	}
	return canvas
}

func TestGraphPageKeepsItsSilentRenderWhenNoRuleFires(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.5"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph", http.StatusOK)

	if strings.Contains(page, graphSignalsDidNotResolve) {
		t.Errorf("a resolved read read as a failed one; body: %s", page)
	}
	if strings.Contains(page, `class="gnode-halo"`) {
		t.Errorf("a signal-free estate lit a node; body: %s", page)
	}
	if !strings.Contains(page, "No open signals on this node.") {
		t.Errorf("a signal-free estate lost the drawer's empty state; body: %s", page)
	}
	if !strings.Contains(page, "All severities") {
		t.Errorf("a signal-free estate lost the severity filter; body: %s", page)
	}
}

func TestGraphPageFailsLoudlyWhenTheSpansReadFails(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.openSpansErr = errors.New("list all open spans failed")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	getBody(t, ac, base+"/graph", http.StatusInternalServerError)
}

func TestGraphPageEmptyEstateNamesNoFailedJoin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	corpusReadFails(f)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph", http.StatusOK)

	if !strings.Contains(page, "Nothing to plot yet") {
		t.Errorf("empty graph lost its empty state; body: %s", page)
	}
	if strings.Contains(page, graphSignalsDidNotResolve) {
		t.Errorf("an empty graph skips the join, so it must name no failed read; body: %s", page)
	}
}
