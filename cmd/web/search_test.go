package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
)

// The Search screen (#303, T8) groups results by kind and highlights the matched
// term in each row, linking every hit to its existing route. A query narrows the
// groups; an empty query browses everything (where the palette's "see everything"
// lands).
func TestSearchGroupsAndHighlights(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/search?q=api", http.StatusOK)

	// The Assets group renders (grouped by kind) with its micro-label count and the
	// asset row links to the asset's existing route.
	for _, want := range []string{
		"Search",                          // the screen heading
		`<h3>Assets</h3>`,                 // the group card title
		`1 match`,                         // the group's micro-label count
		`href="/asset/api.example.com"`,   // the row links to the existing route
		`1 results for`,                   // the count line, verbatim from the example
	} {
		if !strings.Contains(page, want) {
			t.Errorf("search page missing %q; body: %s", want, page)
		}
	}

	// The matched term is highlighted on the accent — an escaped span wrapping just
	// the matched run.
	if !strings.Contains(page, `<span style="color:var(--link);font-weight:600">api</span>`) {
		t.Errorf("search page did not highlight the matched term; body: %s", page)
	}

	// Assets carry no severity (ADR-0024): the row leads with the subject glyph, never
	// a fabricated severity pill.
	for _, pill := range []string{`class="sev sev-critical"`, `class="sev sev-high"`, `class="sev sev-medium"`} {
		if strings.Contains(page, pill) {
			t.Errorf("search page rendered a severity pill %q — assets/signals are not a ramp", pill)
		}
	}
}

// An empty query browses every kind: with a Name subject and a recent Dispatch
// seeded, both the Assets and Batches groups render — grouping across kinds.
func TestSearchBrowsesAcrossKinds(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved"}`)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(7, "hot", obsClock, 2, 0, 0, 2, 0, 0), // a complete batch
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/search", http.StatusOK)

	for _, want := range []string{
		`<h3>Assets</h3>`,      // the Assets group
		`<h3>Batches</h3>`,     // the Batches group
		`href="/run/7"`,        // the batch links to its run detail
		`sbatch complete`,      // the BatchStatus chip state
	} {
		if !strings.Contains(page, want) {
			t.Errorf("search browse missing %q; body: %s", want, page)
		}
	}
}

// With no data (and a query that matches nothing), the design-system empty-state
// renders — fact plus next action — and no group card appears.
func TestSearchNoResults(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/search?q=zzznomatchzzz", http.StatusOK)

	for _, want := range []string{
		"Nothing matches.",                                      // the empty-state fact
		"Try a hostname fragment, a signal phrase, or a batch timestamp.", // the next action
		"0 results for",                                         // the count line
	} {
		if !strings.Contains(page, want) {
			t.Errorf("search no-results missing %q; body: %s", want, page)
		}
	}
	if strings.Contains(page, "<h3>Assets</h3>") {
		t.Errorf("search no-results rendered a group card; body: %s", page)
	}
}

// The Search route is behind requireLogin: an unauthenticated request redirects to
// the login form rather than rendering the screen.
func TestSearchRequiresLogin(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	c := newClient(t)

	resp, err := c.Get(base + "/search")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("unauthenticated /search: status=%d location=%q, want redirect to /login",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}
