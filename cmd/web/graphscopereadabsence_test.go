package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

const (
	graphScopesDidNotResolve = "The scope list did not resolve on this load"
	graphScopePickerButton   = `id="gr-scope-btn"`
	graphDeclareASeed        = "Declare a seed on the Scope screen to bound the drawing."
	graphScopesUnbounded     = "The scope list did not resolve, so the drawing could not be bounded to a declared seed."
)

func graphScopeAbsenceStore(t *testing.T) *fakeStore {
	t.Helper()
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addClassResolution(t, "api.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.5"]}`)
	return f
}

func TestGraphPageNamesAFailedSeedsRead(t *testing.T) {
	f := graphScopeAbsenceStore(t)
	f.seedsErr = errors.New("seed read failed")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph?scope=example.com", http.StatusOK)

	if !strings.Contains(page, `id="gr-svg"`) {
		t.Fatalf("a failed seeds read took the graph away; body: %s", page)
	}
	if !strings.Contains(page, graphScopesDidNotResolve) {
		t.Errorf("the graph does not name the failed seeds read; body: %s", page)
	}
	if strings.Contains(page, graphDeclareASeed) {
		t.Errorf("a failed seeds read still claims no seed is declared; body: %s", page)
	}
	if strings.Contains(page, graphScopePickerButton) {
		t.Errorf("a failed seeds read drew a scope picker from a list it could not read; body: %s", page)
	}
}

func TestGraphPageKeepsItsScopePickerOnAResolvedSeedsRead(t *testing.T) {
	f := graphScopeAbsenceStore(t)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph?scope=example.com", http.StatusOK)

	if strings.Contains(page, graphScopesDidNotResolve) {
		t.Errorf("a resolved seeds read read as a failed one; body: %s", page)
	}
	if !strings.Contains(page, graphScopePickerButton) {
		t.Errorf("a resolved seeds read lost the scope picker its seeds feed; body: %s", page)
	}
	if !strings.Contains(page, `href="/graph?scope=example.com"`) {
		t.Errorf("a resolved seeds read lost the declared seed from the picker; body: %s", page)
	}
}

func TestGraphPageCappedByAFailedSeedsReadDoesNotAskForASeed(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	for i := 0; i < 2*graphColumnCap; i++ {
		f.addClassResolution(t, fmt.Sprintf("n%02d.example.com", i), "internet", obsClock,
			fmt.Sprintf(`{"outcome":"Resolved","addresses":["203.0.113.%d"]}`, i+1))
	}
	f.seedsErr = errors.New("seed read failed")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph?scope=example.com", http.StatusOK)

	if !strings.Contains(page, "The graph draws at most") {
		t.Fatalf("the graph is not capped, so the branch under test never renders; body: %s", page)
	}
	if !strings.Contains(page, graphScopesUnbounded) {
		t.Errorf("a capped graph does not say why it could not be bounded; body: %s", page)
	}
	if strings.Contains(page, graphDeclareASeed) {
		t.Errorf("a capped graph asks for a seed the failed read could not count; body: %s", page)
	}
}
