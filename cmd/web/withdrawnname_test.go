package main

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// A Name whose observations aged out of the currency window keeps its closed spans (ADR-0072 §3).
func withdrawnNameFixture(t *testing.T) (*fakeStore, string) {
	t.Helper()
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "live.example.com", "dns", obsClock.Add(29*24*time.Hour), `{"outcome":"Resolved","addresses":["203.0.113.9"]}`)
	f.addResolution(t, admin.ID, "gone.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.7"]}`)
	f.addResolution(t, admin.ID, "gone.example.com", "dns", obsClock.Add(24*time.Hour), `{"outcome":"NameError"}`)
	f.withdrawSubject("name", "gone.example.com", obsClock.Add(24*time.Hour))
	return f, startAt(t, f, obsClock.Add(30*24*time.Hour))
}

func TestWithdrawnNameWithClosedSpansRendersAssetPage(t *testing.T) {
	f, base := withdrawnNameFixture(t)
	ac := login(t, base, "admin", "hunter2hunter2")

	if len(f.liveObservations(obsClock.Add(30*24*time.Hour))) != 1 {
		t.Fatalf("fixture: want only live.example.com inside the currency window")
	}

	page := getBody(t, ac, base+"/asset/gone.example.com", http.StatusOK)
	for _, want := range []string{"withdrawn", "no current member", "Drift trail", "gone.example.com"} {
		if !strings.Contains(page, want) {
			t.Errorf("withdrawn asset page missing %q; body: %s", want, page)
		}
	}
	if strings.Contains(page, "203.0.113.7") {
		t.Errorf("withdrawn asset page rendered a current value; body: %s", page)
	}
	if strings.Contains(page, "No such subject") {
		t.Errorf("withdrawn asset page rendered as missing; body: %s", page)
	}
}

func TestNameWithNoSpansAndNoObservationsIsMissing(t *testing.T) {
	_, base := withdrawnNameFixture(t)
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/asset/never.measured.example", http.StatusNotFound)
	if !strings.Contains(page, "No such subject") {
		t.Errorf("unmeasured key not reported as missing; body: %s", page)
	}
}

func TestSearchExactKeyReachesWithdrawnName(t *testing.T) {
	_, base := withdrawnNameFixture(t)
	ac := login(t, base, "admin", "hunter2hunter2")

	exact := getBody(t, ac, base+"/search?q=gone.example.com", http.StatusOK)
	for _, want := range []string{`<h3>Assets</h3>`, `1 match`, `href="/asset/gone.example.com"`} {
		if !strings.Contains(exact, want) {
			t.Errorf("exact-key search missing %q; body: %s", want, exact)
		}
	}

	substring := getBody(t, ac, base+"/search?q=gone.example", http.StatusOK)
	if strings.Contains(substring, `href="/asset/gone.example.com"`) {
		t.Errorf("substring search listed a withdrawn name; body: %s", substring)
	}
}

func TestSubjectsListingExcludesWithdrawnName(t *testing.T) {
	_, base := withdrawnNameFixture(t)
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/inventory", http.StatusOK)
	if !strings.Contains(page, "live.example.com") {
		t.Errorf("listing lost the live name; body: %s", page)
	}
	if strings.Contains(page, "gone.example.com") {
		t.Errorf("listing included the withdrawn name; body: %s", page)
	}
}
