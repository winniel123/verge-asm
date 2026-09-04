package main

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
)

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
			name: "kelvin-sign query does not panic",
			text: "k",
			q:    "K",
			want: []hiSeg{{Text: "k", Hit: true}},
		},
		{
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

func TestSearchGroupsAndHighlights(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	// The Documentation group indexes docs/guides/, so a broader query would move the total count.
	page := getBody(t, ac, base+"/search?q=api.example", http.StatusOK)

	for _, want := range []string{
		"Search",
		`<h3>Assets</h3>`,
		`1 match`,
		`href="/asset/api.example.com"`,
		`1 results for`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("search page missing %q; body: %s", want, page)
		}
	}

	if !strings.Contains(page, `<span style="color:var(--link);font-weight:600">api.example</span>`) {
		t.Errorf("search page did not highlight the matched term; body: %s", page)
	}

	// Every page inlines the --sev-* tokens, so only the pill geometry identifies a rendered badge.
	if strings.Contains(page, "height:18px;padding:0 8px;border-radius:999px") {
		t.Errorf("search page rendered a severity badge on an assets-only result that fires no signal; body: %s", page)
	}
}

func TestSearchSignalsCarrySeverity(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedZone(t, f, admin, "example.com", "$ORIGIN example.com.\n@ IN SOA ns1 admin 1 2 3 4 5\n")

	f.addClassResolution(t, "lame.example.com", "internet", obsClock, `{"outcome":"Gap"}`)
	f.addDNSRecord(t, "lame.example.com", "NS", obsClock, `{"rrs":[],"delegation":{"lame":true}}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/search?q=lame", http.StatusOK)

	for _, want := range []string{
		`<h3>Signals</h3>`,
		`background:var(--sev-medium-bg);border:1px solid var(--sev-medium-border);color:var(--sev-medium-fg)`,
		`>Medium</span>`,
		`>lame</span>-delegation`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("search signal row missing %q; body: %s", want, page)
		}
	}
}

func TestSearchBrowsesAcrossKinds(t *testing.T) {
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

	for _, want := range []string{
		`<h3>Assets</h3>`,
		`<h3>Batches</h3>`,
		`href="/run/7"`,
		`se-batch complete`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("search browse missing %q; body: %s", want, page)
		}
	}
}

func TestSearchHasDocumentationGroup(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/search?q=signals", http.StatusOK)

	for _, want := range []string{
		`<h3>Documentation</h3>`,
		`Assets, signals, batches, docs`,
		`>Signals</span> reference`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("search page missing restored docs surface %q; body: %s", want, page)
		}
	}
}

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

func TestSearchNoResults(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/search?q=zzznomatchzzz", http.StatusOK)

	for _, want := range []string{
		"Nothing matches.",
		"Try a hostname fragment, a signal phrase, or a batch timestamp.",
		"0 results for",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("search no-results missing %q; body: %s", want, page)
		}
	}
	if strings.Contains(page, "<h3>Assets</h3>") {
		t.Errorf("search no-results rendered a group card; body: %s", page)
	}
}

func TestSearchCommandPaletteHandoff(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/reports", http.StatusOK)

	// The label must match ConsoleApp.jsx's CommandPalette entry (design-system/examples/console/).
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
