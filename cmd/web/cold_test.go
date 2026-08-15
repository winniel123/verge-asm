package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func setColdScope(t *testing.T, c *http.Client, base string, id int64, optIn bool) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/seeds/cold", url.Values{
		"id": {strconv.FormatInt(id, 10)}, "opt_in": {strconv.FormatBool(optIn)},
	})
}

func coldScanEnabled(f *fakeStore) bool {
	for _, sc := range f.scans {
		if sc.Kind == "cold" {
			return sc.Enabled
		}
	}
	return false
}

// The cold tier ships disabled with an empty scope list, and enabling is
// per-Seed: opting a scope in enables the tier for that scope, opting the last
// scope out returns it to off. The enabled flag is derived from the scope set,
// never a global toggle (ADR-0044).
func TestColdScanOptInEnablesPerScope(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	id := f.seeds[0].ID

	// Shipped state: disabled, no scope opted in, the page says so.
	if coldScanEnabled(f) {
		t.Fatalf("cold Scan enabled on a fresh install, want disabled")
	}
	page := seedsBody(t, ac, base)
	if !strings.Contains(page, "tier off") || !strings.Contains(page, "Opt in") {
		t.Errorf("cold tier not shown off with an opt-in control; body: %s", page)
	}

	// Opt the scope in: the tier enables, and no scan is fired by the save.
	resp := setColdScope(t, ac, base, id, true)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/seeds" {
		t.Fatalf("opt in cold scope: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	if !f.coldScopes[id] {
		t.Fatalf("scope not recorded as opted into the cold tier")
	}
	if !coldScanEnabled(f) {
		t.Fatalf("cold Scan not enabled after a scope opted in")
	}
	// Never scan on config save: opting in queues nothing and fires nothing — no
	// Batch, no Observation is produced by the opt-in act itself (ADR-0044).
	if len(f.batches) != 0 || len(f.observations) != 0 {
		t.Fatalf("opting a scope in fired a scan on config save: %d batches, %d observations",
			len(f.batches), len(f.observations))
	}

	page = seedsBody(t, ac, base)
	if !strings.Contains(page, "tier on") || !strings.Contains(page, "opted in") || !strings.Contains(page, "Opt out") {
		t.Errorf("opted-in tier not reflected with an opt-out control; body: %s", page)
	}

	// Opt the last scope out: the tier returns to disabled.
	setColdScope(t, ac, base, id, false).Body.Close()
	if f.coldScopes[id] {
		t.Fatalf("scope not removed from the cold tier on opt-out")
	}
	if coldScanEnabled(f) {
		t.Fatalf("cold Scan still enabled after the last scope opted out")
	}
}

// The full-range opt-in accepts an address scope too — the tier is per-Seed
// scope, name or address alike.
func TestColdScanOptInAddressScope(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "address", "203.0.113.0/24").Body.Close()
	id := f.seeds[0].ID

	setColdScope(t, ac, base, id, true).Body.Close()
	if !f.coldScopes[id] || !coldScanEnabled(f) {
		t.Fatalf("address scope did not opt into the cold tier")
	}
}

// A viewer can read the cold-tier state but never move it.
func TestViewerCannotOptIntoColdTier(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	ac := login(t, base, "admin", "hunter2hunter2")
	declare(t, ac, base, "name", "example.com").Body.Close()
	id := f.seeds[0].ID

	vc := login(t, base, "viewer", "hunter2hunter2")
	resp := setColdScope(t, vc, base, id, true)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer opt in cold scope: status=%d, want 403", resp.StatusCode)
	}
	if f.coldScopes[id] || coldScanEnabled(f) {
		t.Fatalf("viewer's denied act still enabled the cold tier")
	}

	page := seedsBody(t, vc, base)
	if !strings.Contains(page, "full-range scan") && !strings.Contains(page, "Full-range") {
		t.Errorf("viewer cannot see the cold-tier section; body: %s", page)
	}
	if strings.Contains(page, `action="/seeds/cold"`) {
		t.Errorf("cold opt-in control shown to a viewer; body: %s", page)
	}
}

func TestSetColdScopeRequiresLogin(t *testing.T) {
	base := start(t, newFakeStore(), "")
	c := newClient(t)
	resp := setColdScope(t, c, base, 1, true)
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon opt in: status=%d location=%q, want redirect to /login", resp.StatusCode, resp.Header.Get("Location"))
	}
}
