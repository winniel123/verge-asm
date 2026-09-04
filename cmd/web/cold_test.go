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
	return postForm(t, c, base+"/settings/cold", url.Values{
		"id": {strconv.FormatInt(id, 10)}, "opt_in": {strconv.FormatBool(optIn)},
	})
}

func coldBody(t *testing.T, c *http.Client, base string) string {
	t.Helper()
	return getBody(t, c, base+"/settings?tab=scans", http.StatusOK)
}

func coldScanEnabled(f *fakeStore) bool {
	for _, sc := range f.scans {
		if sc.Kind == "cold" {
			return sc.Enabled
		}
	}
	return false
}

func TestColdScanOptInEnablesPerScope(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	id := f.seeds[0].ID

	if coldScanEnabled(f) {
		t.Fatalf("cold Scan enabled on a fresh install, want disabled")
	}
	page := coldBody(t, ac, base)
	if !strings.Contains(page, "tier off") || !strings.Contains(page, "Opt in") {
		t.Errorf("cold tier not shown off with an opt-in control; body: %s", page)
	}

	resp := setColdScope(t, ac, base, id, true)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/settings?tab=scans" {
		t.Fatalf("opt in cold scope: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	if !f.coldScopes[id] {
		t.Fatalf("scope not recorded as opted into the cold tier")
	}
	if !coldScanEnabled(f) {
		t.Fatalf("cold Scan not enabled after a scope opted in")
	}
	if len(f.batches) != 0 || len(f.observations) != 0 {
		t.Fatalf("opting a scope in fired a scan on config save: %d batches, %d observations",
			len(f.batches), len(f.observations))
	}

	page = coldBody(t, ac, base)
	if !strings.Contains(page, "tier on") || !strings.Contains(page, "opted in") || !strings.Contains(page, "Opt out") {
		t.Errorf("opted-in tier not reflected with an opt-out control; body: %s", page)
	}

	setColdScope(t, ac, base, id, false).Body.Close()
	if f.coldScopes[id] {
		t.Fatalf("scope not removed from the cold tier on opt-out")
	}
	if coldScanEnabled(f) {
		t.Fatalf("cold Scan still enabled after the last scope opted out")
	}
}

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

	resp = get(t, vc, base+"/settings?tab=scans")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("viewer reached the admin-only cold-tier surface: status=%d, want 403", resp.StatusCode)
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
