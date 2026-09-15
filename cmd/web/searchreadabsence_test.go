package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
)

const (
	searchSignalsFailed  = "Signals did not resolve"
	searchTotalWithheld  = "Results not totalled"
	searchNothingMatches = "Nothing matches."
	// A rendered severity badge is identified by its pill geometry, because every page inlines the --sev-* tokens.
	sevBadgeGeometry = "height:18px;padding:0 8px;border-radius:999px"
)

func searchCountLine(t *testing.T, page string) string {
	t.Helper()
	const open = `<span class="se-count">`
	from := strings.Index(page, open)
	if from < 0 {
		t.Fatalf("search page carries no result-count line; body: %s", page)
	}
	line := page[from+len(open):]
	end := strings.Index(line, "</span>")
	if end < 0 {
		t.Fatalf("search page's result-count line is unterminated; body: %s", page)
	}
	return line[:end]
}

func searchAssetsGroup(t *testing.T, page string) string {
	t.Helper()
	from := strings.Index(page, "<h3>Assets</h3>")
	if from < 0 {
		t.Fatalf("search page carries no Assets group; body: %s", page)
	}
	group := page[from:]
	if end := strings.Index(group, "</section>"); end >= 0 {
		group = group[:end]
	}
	return group
}

func searchAbsenceStore(t *testing.T) *fakeStore {
	t.Helper()
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "lame.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	lameName(t, f, "lame.example.com")
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(7, "hot", obsClock, 2, 0, 0, 2, 0, 0),
	}
	return f
}

func TestSearchNamesAFailedSignalRead(t *testing.T) {
	f := searchAbsenceStore(t)
	corpusReadFails(f)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/search", http.StatusOK)

	for _, want := range []string{searchSignalsFailed, signalsDidNotResolve, searchTotalWithheld} {
		if !strings.Contains(page, want) {
			t.Errorf("search page does not name the failed signal read: missing %q; body: %s", want, page)
		}
	}
	if count := searchCountLine(t, page); strings.Contains(count, " results") {
		t.Errorf("a failed signal read still states a result total: %q", count)
	}
	if strings.Contains(page, searchNothingMatches) {
		t.Errorf("a failed signal read still claims nothing matches; body: %s", page)
	}
	if strings.Contains(page, sevBadgeGeometry) {
		t.Errorf("a failed signal read rendered a severity badge anyway; body: %s", page)
	}
	if strings.Contains(page, "lame-delegation") {
		t.Errorf("a failed corpus read listed a signal anyway; body: %s", page)
	}
	// The nav badge is the fourth thing the corpus read feeds, and it counts open signals.
	if strings.Contains(page, `Signals<span class="ct">`) {
		t.Errorf("a failed signal read raised the chrome signal badge; body: %s", page)
	}
	for _, want := range []string{"<h3>Assets</h3>", `href="/asset/lame.example.com"`, "<h3>Batches</h3>", `href="/run/7"`, "<h3>Documentation</h3>"} {
		if !strings.Contains(page, want) {
			t.Errorf("a failed signal read took away %q from another result group; body: %s", want, page)
		}
	}
}

func TestSearchStatesItsTotalOnAResolvedRead(t *testing.T) {
	f := searchAbsenceStore(t)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/search", http.StatusOK)

	if count := searchCountLine(t, page); !strings.Contains(count, " results") {
		t.Errorf("a resolved read lost its result total: %q", count)
	}
	for _, banned := range []string{searchSignalsFailed, signalsDidNotResolve, searchTotalWithheld} {
		if strings.Contains(page, banned) {
			t.Errorf("a resolved read read as a failed one: found %q; body: %s", banned, page)
		}
	}
	if group := searchAssetsGroup(t, page); !strings.Contains(group, sevBadgeGeometry) {
		t.Errorf("a resolved read lost the severity a firing rule gives an asset row; group: %s", group)
	}
}

func TestSearchKeepsItsEmptyStateWhenNothingMatches(t *testing.T) {
	f := searchAbsenceStore(t)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/search?q=nosuchsubjectanywhere", http.StatusOK)

	if !strings.Contains(page, searchNothingMatches) {
		t.Errorf("a query matching nothing lost its original empty state; body: %s", page)
	}
	if count := searchCountLine(t, page); !strings.HasPrefix(count, "0 results") {
		t.Errorf("a query matching nothing lost its correct total: %q", count)
	}
	for _, banned := range []string{searchSignalsFailed, signalsDidNotResolve, searchTotalWithheld} {
		if strings.Contains(page, banned) {
			t.Errorf("a resolved read matching no signal read as a failed one: found %q; body: %s", banned, page)
		}
	}
}
