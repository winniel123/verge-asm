package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func declare(t *testing.T, c *http.Client, base, kind, scope string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/seeds", url.Values{"kind": {kind}, "scope": {scope}})
}

func seedsBody(t *testing.T, c *http.Client, base string) string {
	t.Helper()
	// The scope presentation is canonical (#286); GET /seeds now permanently
	// redirects here, so the listing is read at /scope (identical seedsPage render).
	resp, err := c.Get(base + "/scope")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /scope status = %d, want 200", resp.StatusCode)
	}
	return body(t, resp)
}

func TestDeclareNameAndAddressSeeds(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// A name scope.
	resp := declare(t, ac, base, "name", "Example.com")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/scope" {
		t.Fatalf("declare name: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()

	// An address scope.
	resp = declare(t, ac, base, "address", "203.0.113.0/24")
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("declare address: status=%d", resp.StatusCode)
	}
	resp.Body.Close()

	page := seedsBody(t, ac, base)
	if !strings.Contains(page, "example.com") {
		t.Errorf("name scope not listed; body: %s", page)
	}
	if !strings.Contains(page, "203.0.113.0/24") {
		t.Errorf("address scope not listed; body: %s", page)
	}
	// The two kinds are distinguished in the listing.
	if !strings.Contains(page, ">name<") || !strings.Contains(page, ">address<") {
		t.Errorf("name/address not distinguished in listing; body: %s", page)
	}
}

func TestAddressScopeOverCapRejected(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// /21 is 2048 addresses, over the 1024 cap.
	resp := declare(t, ac, base, "address", "10.0.0.0/21")
	got := body(t, resp)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(got, "over the cap of 1024") {
		t.Fatalf("over-cap scope not rejected clearly: status=%d body=%s", resp.StatusCode, got)
	}
	// The rejected value is preserved so the admin need not retype it.
	if !strings.Contains(got, `value="10.0.0.0/21"`) {
		t.Errorf("rejected scope not retained in the form; body: %s", got)
	}
	if len(f.seeds) != 0 {
		t.Fatalf("seeds after rejected declaration = %d, want 0", len(f.seeds))
	}

	// The IPv4 boundary /22 (exactly 1024) is accepted, and so is an
	// equivalently-sized IPv6 block — the cap is family-agnostic.
	if resp = declare(t, ac, base, "address", "10.0.0.0/22"); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("/22 at the cap should be accepted: status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	if resp = declare(t, ac, base, "address", "2001:db8::/118"); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("IPv6 /118 at the cap should be accepted: status=%d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestNameScopeMustBeRegistrable(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := declare(t, ac, base, "name", "www.example.com")
	if got := body(t, resp); resp.StatusCode != http.StatusBadRequest || !strings.Contains(got, "registrable domain example.com") {
		t.Fatalf("subdomain not rejected toward its registrable domain: status=%d body=%s", resp.StatusCode, got)
	}
	if len(f.seeds) != 0 {
		t.Fatalf("seeds = %d, want 0", len(f.seeds))
	}
}

func TestDuplicateSeedRejected(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	resp := declare(t, ac, base, "name", "example.com")
	if got := body(t, resp); !strings.Contains(got, "already declared") {
		t.Fatalf("duplicate not reported; body: %s", got)
	}
}

func TestViewerCannotDeclareButCanView(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	// Admin declares one so the list is non-empty.
	ac := login(t, base, "admin", "hunter2hunter2")
	declare(t, ac, base, "name", "example.com").Body.Close()

	vc := login(t, base, "viewer", "hunter2hunter2")

	// The viewer is denied the declaration mutation.
	resp := declare(t, vc, base, "name", "example.org")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer declare: status=%d, want 403", resp.StatusCode)
	}
	if len(f.seeds) != 1 {
		t.Fatalf("seeds after denied declare = %d, want 1", len(f.seeds))
	}

	// But the viewer can read the list, and the declare form is not offered.
	page := seedsBody(t, vc, base)
	if !strings.Contains(page, "example.com") {
		t.Errorf("viewer cannot see the seeds list; body: %s", page)
	}
	if strings.Contains(page, `action="/seeds"`) {
		t.Errorf("declare form shown to a viewer; body: %s", page)
	}
}

func TestDeclareRequiresLogin(t *testing.T) {
	base := start(t, newFakeStore(), "")
	c := newClient(t)
	resp := declare(t, c, base, "name", "example.com")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon declare: status=%d location=%q, want redirect to /login", resp.StatusCode, resp.Header.Get("Location"))
	}
}
