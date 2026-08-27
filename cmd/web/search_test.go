package main

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
)

// searchSegs splits a matched field on the FIRST case-insensitive occurrence of the
// query into the [{Text,Hit}] list "hisegs" renders (#25a): the un-hit text before,
// the hit run (original case preserved), the un-hit text after — empty edges omitted,
// a non-match a single un-hit seg. The match is located in the lowered text, so the
// run length must come from the lowered query, never len(q): strings.ToLower can
// change a value's byte length (U+212A KELVIN SIGN then "k"; U+023A then the longer
// U+2C65), and mixing the two offset spaces once sliced the original out of range and
// panicked the whole /search page (#340). These cases must not panic.
func TestSearchSegs(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		q    string
		want []hiSeg
	}{
		{
			name: "ascii splits before, hit, after",
			text: "foobar",
			q:    "oob",
			want: []hiSeg{{Text: "f"}, {Text: "oob", Hit: true}, {Text: "ar"}},
		},
		{
			name: "leading hit omits the empty before seg",
			text: "acmecorp.io",
			q:    "acme",
			want: []hiSeg{{Text: "acme", Hit: true}, {Text: "corp.io"}},
		},
		{
			name: "hit preserves original case",
			text: "FooBar",
			q:    "bar",
			want: []hiSeg{{Text: "Foo"}, {Text: "Bar", Hit: true}},
		},
		{
			name: "non-match is a single un-hit seg",
			text: "no match here",
			q:    "acme",
			want: []hiSeg{{Text: "no match here"}},
		},
		{
			name: "empty query is a single un-hit seg",
			text: "anything",
			q:    "",
			want: []hiSeg{{Text: "anything"}},
		},
		{
			// U+212A ToLower is "k": the lowered query is 1 byte where q is 3, so the
			// span length must track the lowered form to slice "k" in range.
			name: "kelvin-sign query does not panic",
			text: "k",
			q:    "K",
			want: []hiSeg{{Text: "k", Hit: true}},
		},
		{
			// U+023A ToLower is U+2C65, which grows the lowered text past the original;
			// the clamp falls back to a single un-hit seg rather than panicking.
			name: "growing-lowercase text does not panic",
			text: "Ⱥk",
			q:    "k",
			want: []hiSeg{{Text: "Ⱥk"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []hiSeg
			func() {
				defer func() {
					if p := recover(); p != nil {
						t.Fatalf("searchSegs(%q, %q) panicked: %v", tc.text, tc.q, p)
					}
				}()
				got = searchSegs(tc.text, tc.q)
			}()
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("searchSegs(%q, %q) = %#v, want %#v", tc.text, tc.q, got, tc.want)
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
	// A query specific enough to hit only the asset (no guide, signal, or batch
	// mentions "api.example"), so the total-count assertion stays deterministic now
	// that the Documentation group indexes docs/guides/.
	page := getBody(t, ac, base+"/search?q=api.example", http.StatusOK)

	// The Assets group renders (grouped by kind) with its micro-label count and the
	// asset row links to the asset's existing route.
	for _, want := range []string{
		"Search",                        // the screen heading
		`<h3>Assets</h3>`,               // the group card title
		`1 match`,                       // the group's micro-label count
		`href="/asset/api.example.com"`, // the row links to the existing route
		`1 results for`,                 // the count line, verbatim from the spec
	} {
		if !strings.Contains(page, want) {
			t.Errorf("search page missing %q; body: %s", want, page)
		}
	}

	// The matched term is highlighted on the accent — the "hisegs" span wrapping just
	// the matched run.
	if !strings.Contains(page, `<span style="color:var(--link);font-weight:600">api.example</span>`) {
		t.Errorf("search page did not highlight the matched term; body: %s", page)
	}

	// Assets carry a nullable severity (#25b): it is the rollup across the signals
	// firing on the subject, so a seed that fires none renders no sevbadge at all —
	// the landed sevbadge's inline-style signature is absent. (Signals, which DO carry
	// the rule's severity, are covered by TestSearchSignalsCarrySeverity.) The
	// substring is the sevbadge define's unique pill geometry, not the pageCSS .sev-*
	// classes (which name --sev-* tokens on every page).
	if strings.Contains(page, "height:18px;padding:0 8px;border-radius:999px") {
		t.Errorf("search page rendered a severity badge on an assets-only result that fires no signal; body: %s", page)
	}
}

// A fired signal carries its rule's severity (P0.1): the Signals group's row leads
// with the SeverityBadge for the rule, the same five-level ramp the Signals screen
// shows — read from internal/signal, never fabricated. A lame delegation fires
// lame-delegation, a medium rule, so its row renders the landed medium sevbadge.
func TestSearchSignalsCarrySeverity(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedZone(t, f, admin, "example.com", "$ORIGIN example.com.\n@ IN SOA ns1 admin 1 2 3 4 5\n")

	// An all-refusing delegation composes to Lame → lame-delegation FIRES (medium).
	f.addClassResolution(t, "lame.example.com", "internet", obsClock, `{"outcome":"Gap"}`)
	f.addDNSRecord(t, "lame.example.com", "NS", obsClock, `{"rrs":[],"delegation":{"lame":true}}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/search?q=lame", http.StatusOK)

	for _, want := range []string{
		`<h3>Signals</h3>`, // the Signals group renders
		// the row leads with the landed medium sevbadge (its unique inline signature,
		// not the pageCSS .sev-medium class):
		`background:var(--sev-medium-bg);border:1px solid var(--sev-medium-border);color:var(--sev-medium-fg)`,
		`>Medium</span>`,          // the sevbadge label
		`>lame</span>-delegation`, // the firing rule, its query match highlighted
	} {
		if !strings.Contains(page, want) {
			t.Errorf("search signal row missing %q; body: %s", want, page)
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
		`<h3>Assets</h3>`,   // the Assets group
		`<h3>Batches</h3>`,  // the Batches group
		`href="/run/7"`,     // the batch links to its run detail
		`se-batch complete`, // the BatchStatus chip state
	} {
		if !strings.Contains(page, want) {
			t.Errorf("search browse missing %q; body: %s", want, page)
		}
	}
}

// The spec's fourth group, Documentation, is restored (P2.5, #451): the operator
// guides under docs/guides/ ARE the content store its original drop (#316) said was
// missing. The group indexes each guide's front-matter title + description, so a
// query matching a guide's title lights it up, and the input placeholder advertises
// docs again.
func TestSearchHasDocumentationGroup(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	// "signals" matches the Signals reference guide's title/description.
	page := getBody(t, ac, base+"/search?q=signals", http.StatusOK)

	for _, want := range []string{
		`<h3>Documentation</h3>`,         // the restored group card title
		`Assets, signals, batches, docs`, // the input placeholder advertises docs again
		`>Signals</span> reference`,      // a guide's front-matter title, indexed + highlighted
	} {
		if !strings.Contains(page, want) {
			t.Errorf("search page missing restored docs surface %q; body: %s", want, page)
		}
	}
}

// An empty query browses every guide (the "see everything" landing): with the guides
// embedded, the Documentation group renders unfiltered — the group is real data now,
// not permanently-empty markup.
func TestSearchBrowsesDocumentation(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/search", http.StatusOK)

	if !strings.Contains(page, `<h3>Documentation</h3>`) {
		t.Errorf("empty-query browse missing the Documentation group; body: %s", page)
	}
	if len(guideIndex) == 0 {
		t.Fatalf("guide index is empty — docs/guides/ did not embed")
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
		"Nothing matches.", // the empty-state fact
		"Try a hostname fragment, a signal phrase, or a batch timestamp.", // the next action
		"0 results for", // the count line
	} {
		if !strings.Contains(page, want) {
			t.Errorf("search no-results missing %q; body: %s", want, page)
		}
	}
	if strings.Contains(page, "<h3>Assets</h3>") {
		t.Errorf("search no-results rendered a group card; body: %s", page)
	}
}

// The command-K palette (templates_shell.go's "chrome" block) hands off to this
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
