package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
)

// searchHighlight finds its match in the lowered text, so the wrapped span's byte
// length must come from the lowered query, never len(q): strings.ToLower can change
// a value's byte length (U+212A KELVIN SIGN → "k"; U+023A → the longer U+2C65), and
// mixing the two offset spaces once sliced the original out of range and panicked
// the whole /search page (#340). These cases must not panic and must stay escaped,
// well-formed HTML; a plain ASCII match must still wrap exactly the matched run.
func TestSearchHighlight(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		q    string
		want string // full expected output
	}{
		{
			name: "ascii wraps the matched run",
			text: "foobar",
			q:    "bar",
			want: `foo<span style="color:var(--link);font-weight:600">bar</span>`,
		},
		{
			name: "ascii highlight preserves original case",
			text: "FooBar",
			q:    "bar",
			want: `Foo<span style="color:var(--link);font-weight:600">Bar</span>`,
		},
		{
			// U+212A ToLower→"k": the lowered query is 1 byte where q is 3, so the
			// span length must track the lowered form to slice "k" in range.
			name: "kelvin-sign query does not panic",
			text: "k",
			q:    "K",
			want: `<span style="color:var(--link);font-weight:600">k</span>`,
		},
		{
			// U+023A ToLower→U+2C65 grows the lowered text past the original; the
			// clamp falls back to the plain escaped original rather than panicking.
			name: "growing-lowercase text does not panic",
			text: "Ⱥk",
			q:    "k",
			want: "Ⱥk",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			func() {
				defer func() {
					if p := recover(); p != nil {
						t.Fatalf("searchHighlight(%q, %q) panicked: %v", tc.text, tc.q, p)
					}
				}()
				got = string(searchHighlight(tc.text, tc.q))
			}()
			if got != tc.want {
				t.Errorf("searchHighlight(%q, %q) = %q, want %q", tc.text, tc.q, got, tc.want)
			}
		})
	}
}

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

// The example's fourth group, Documentation, was dropped (#316): there is no docs
// content store or /docs route to search over and none is planned, so the group is
// removed rather than left as permanently-empty markup. Even with results in every
// other kind, no Documentation card renders and the placeholder no longer advertises
// docs.
func TestSearchHasNoDocumentationGroup(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved"}`)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(7, "hot", obsClock, 2, 0, 0, 2, 0, 0),
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/search", http.StatusOK)

	for _, absent := range []string{
		`<h3>Documentation</h3>`,          // the removed group card title
		`Assets, signals, batches, docs`,  // the old input placeholder advertising docs
	} {
		if strings.Contains(page, absent) {
			t.Errorf("search page still carries removed docs surface %q; body: %s", absent, page)
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

// The ⌘K command palette (templates_shell.go's "chrome" block) hands off to this
// screen (#315): every chrome page carries a persistent "Search everything" item
// marked data-cmdk-search, matching design-system/examples/console/ConsoleApp.jsx's
// CommandPalette entry of the same label. Its static href (used when JS hasn't run,
// or as the empty-query default) points at bare /search — the handler already
// browses everything unfiltered on an empty q — and the shell's filter() script
// rewrites the href to /search?q=<query> as the operator types, so it always stays
// visible instead of being filtered out like the quick-nav items.
func TestSearchCommandPaletteHandoff(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/reports", http.StatusOK)

	for _, want := range []string{
		`data-cmdk-search`,
		`class="cmdk-item" href="/search" data-cmdk-search`,
		`>Search everything<`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("chrome missing palette search-handoff %q; body: %s", want, page)
		}
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
