package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func getBody(t *testing.T, c *http.Client, url string, wantStatus int) string {
	t.Helper()
	resp, err := c.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != wantStatus {
		t.Fatalf("GET %s status = %d, want %d (body: %s)", url, resp.StatusCode, wantStatus, got)
	}
	return got
}

func addNameSeed(t *testing.T, f *fakeStore, createdBy int64, domain string) {
	t.Helper()
	if _, err := f.CreateNameSeed(t.Context(), db.CreateNameSeedParams{
		NameDomain: pgtype.Text{String: domain, Valid: true}, CreatedBy: createdBy,
	}); err != nil {
		t.Fatal(err)
	}
}

var obsClock = time.Date(2026, 8, 10, 9, 0, 0, 0, time.UTC)

func TestSubjectsListsCurrentNamesWithSearchAndNoDenominator(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.5"]}`)
	f.addResolution(t, admin.ID, "www.example.net", "dns", obsClock, `{"outcome":"NoData"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/subjects", http.StatusOK)
	for _, name := range []string{"api.example.com", "www.example.net"} {
		if !strings.Contains(page, name) {
			t.Errorf("listing missing %q; body: %s", name, page)
		}
		if !strings.Contains(page, `href="/subjects/`+name+`"`) {
			t.Errorf("listing missing drill-down link for %q; body: %s", name, page)
		}
	}
	// No denominator: the screen states it holds no total, and renders no count.
	if !strings.Contains(page, "There is no total") {
		t.Errorf("listing does not refuse a denominator in copy; body: %s", page)
	}
	// No rendered count of the listing anywhere on the screen.
	for _, denom := range []string{"2 names", "2 subjects", "Showing 2", "of 2"} {
		if strings.Contains(page, denom) {
			t.Errorf("listing rendered a count/denominator %q; body: %s", denom, page)
		}
	}

	// Search narrows to the matching Name.
	only := getBody(t, ac, base+"/subjects?q=example.com", http.StatusOK)
	if !strings.Contains(only, "api.example.com") || strings.Contains(only, "www.example.net") {
		t.Errorf("search did not narrow to example.com; body: %s", only)
	}
}

func TestWithdrawnNameNotListedButReachableByKey(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// A live Name and a withdrawn Name (latest resolution is a Name Error).
	f.addResolution(t, admin.ID, "live.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.9"]}`)
	f.addResolution(t, admin.ID, "gone.example.com", "dns", obsClock, `{"outcome":"Resolved"}`)
	f.addResolution(t, admin.ID, "gone.example.com", "dns", obsClock.Add(24*time.Hour), `{"outcome":"NameError"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/subjects", http.StatusOK)
	if !strings.Contains(page, "live.example.com") {
		t.Errorf("live name not listed; body: %s", page)
	}
	if strings.Contains(page, "gone.example.com") {
		t.Errorf("withdrawn name appears in listing; body: %s", page)
	}
	// Searching for it does not surface it in the listing either.
	searched := getBody(t, ac, base+"/subjects?q=gone.example.com", http.StatusOK)
	if strings.Contains(searched, `href="/subjects/gone.example.com"`) {
		t.Errorf("withdrawn name surfaced by search listing; body: %s", searched)
	}

	// It is still reachable by its own key, marked as a population of no member.
	drill := getBody(t, ac, base+"/subjects/gone.example.com", http.StatusOK)
	if !strings.Contains(drill, "withdrawn") || !strings.Contains(drill, "no current member") {
		t.Errorf("withdrawn drill-down not marked; body: %s", drill)
	}
}

func TestSubjectDrilldownRendersCitationChainAndPlaceholders(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	// An earlier and a later observation; the chain cites the earliest.
	f.addResolution(t, admin.ID, "example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.1"]}`)
	f.addResolution(t, admin.ID, "example.com", "dns", obsClock.Add(48*time.Hour), `{"outcome":"Resolved","addresses":["203.0.113.1","203.0.113.2"]}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/example.com", http.StatusOK)

	// The "why is this here" Citation chain: subject → introducing observation →
	// terminating Seed.
	for _, want := range []string{
		"Citation chain", "resolution-walk", "dns Scan",
		"name scope example.com", "declared by admin",
	} {
		if !strings.Contains(drill, want) {
			t.Errorf("citation chain missing %q; body: %s", want, drill)
		}
	}
	// Current resolution value renders.
	if !strings.Contains(drill, "Resolved") || !strings.Contains(drill, "203.0.113.2") {
		t.Errorf("current resolution not rendered; body: %s", drill)
	}
	// The Timelines section is now wired (#190): the two Resolved answers fold to a
	// closed span and a current one, on the resolution timeline. The Rules section
	// remains a placeholder for ticket 22.
	if !strings.Contains(drill, "Current and closed timelines") {
		t.Errorf("timelines section missing; body: %s", drill)
	}
	if !strings.Contains(drill, "ticket 22") {
		t.Errorf("rules placeholder missing; body: %s", drill)
	}
}

func TestSubjectDrilldownRendersCurrentAndClosedTimelines(t *testing.T) {
	// AC6: re-running the dns Scan with a changed answer produces a new Span and
	// closes the old; the drill-down renders current + closed timelines.
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.1"]}`)
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock.Add(24*time.Hour), `{"outcome":"Resolved","addresses":["203.0.113.2"]}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/api.example.com", http.StatusOK)

	// A current span (the later answer) and a closed-history row (the earlier one).
	for _, want := range []string{"Current and closed timelines", "Current", "Opened", "Closed"} {
		if !strings.Contains(drill, want) {
			t.Errorf("timeline drill-down missing %q; body: %s", want, drill)
		}
	}
	// The resolution facet timeline is labelled and its closed span is present.
	if !strings.Contains(drill, "resolution") {
		t.Errorf("resolution timeline not labelled; body: %s", drill)
	}
}

func TestSubjectMissingReturns404(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := getBody(t, ac, base+"/subjects/never.measured.example", http.StatusNotFound)
	if !strings.Contains(got, "No such subject") {
		t.Errorf("missing subject not reported as 404 page; body: %s", got)
	}
}

func TestSubjectsRequiresLogin(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	c := newClient(t)

	resp, err := c.Get(base + "/subjects")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("unauthenticated /subjects: status=%d location=%q, want redirect to /login",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}
