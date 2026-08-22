package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func exclude(t *testing.T, c *http.Client, base, kind, value string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/exclusions", url.Values{"kind": {kind}, "value": {value}})
}

func unexclude(t *testing.T, c *http.Client, base string, id int64) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/exclusions/delete", url.Values{"id": {strconv.FormatInt(id, 10)}})
}

func TestDeclareExclusions(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	for _, tc := range []struct{ kind, value, want string }{
		{"name", "API.Example.com", "api.example.com"}, // an exact name, case-folded
		{"subtree", "internal.example.com", "internal.example.com"},
		{"address", "203.0.113.0/24", "203.0.113.0/24"}, // a CIDR
		{"address", "198.51.100.7", "198.51.100.7/32"},  // a bare address carves one host
	} {
		resp := exclude(t, ac, base, tc.kind, tc.value)
		if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/scope" {
			t.Fatalf("exclude %s %q: status=%d location=%q", tc.kind, tc.value, resp.StatusCode, resp.Header.Get("Location"))
		}
		resp.Body.Close()
	}

	page := seedsBody(t, ac, base)
	for _, want := range []string{"api.example.com", "internal.example.com", "203.0.113.0/24", "198.51.100.7/32"} {
		if !strings.Contains(page, want) {
			t.Errorf("exclusion %q not listed; body: %s", want, page)
		}
	}
	// The three kinds are distinguished in the listing.
	for _, badge := range []string{">name<", ">subtree<", ">address<"} {
		if !strings.Contains(page, badge) {
			t.Errorf("exclusion kind %q not shown in listing; body: %s", badge, page)
		}
	}
}

func TestUnexclude(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	exclude(t, ac, base, "name", "gone.example.com").Body.Close()
	if len(f.exclusions) != 1 {
		t.Fatalf("exclusions after declare = %d, want 1", len(f.exclusions))
	}
	id := f.exclusions[0].ID

	resp := unexclude(t, ac, base, id)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("un-exclude: status=%d, want 303", resp.StatusCode)
	}
	resp.Body.Close()
	if len(f.exclusions) != 0 {
		t.Fatalf("exclusions after un-exclude = %d, want 0", len(f.exclusions))
	}
	if page := seedsBody(t, ac, base); strings.Contains(page, "gone.example.com") {
		t.Errorf("un-excluded value still listed; body: %s", page)
	}
}

func TestInvalidExclusionsRejected(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// A wildcard is not a name.
	resp := exclude(t, ac, base, "name", "*.example.com")
	if got := body(t, resp); resp.StatusCode != http.StatusBadRequest || !strings.Contains(got, "bare name") {
		t.Fatalf("wildcard not rejected clearly: status=%d body=%s", resp.StatusCode, got)
	}

	// A malformed address is rejected, and the typed value is preserved.
	resp = exclude(t, ac, base, "address", "not-an-address")
	got := body(t, resp)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(got, "not an address or CIDR block") {
		t.Fatalf("bad address not rejected clearly: status=%d body=%s", resp.StatusCode, got)
	}
	if !strings.Contains(got, `value="not-an-address"`) {
		t.Errorf("rejected value not retained in the form; body: %s", got)
	}
	if len(f.exclusions) != 0 {
		t.Fatalf("exclusions after rejected declarations = %d, want 0", len(f.exclusions))
	}
}

func TestDuplicateExclusionRejected(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	exclude(t, ac, base, "name", "dup.example.com").Body.Close()
	resp := exclude(t, ac, base, "name", "dup.example.com")
	if got := body(t, resp); !strings.Contains(got, "already excluded") {
		t.Fatalf("duplicate not reported; body: %s", got)
	}
	// A subtree of the same string is a different claim and is accepted.
	if resp = exclude(t, ac, base, "subtree", "dup.example.com"); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("subtree of the same name should be accepted: status=%d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestViewerCannotExcludeButCanView(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	ac := login(t, base, "admin", "hunter2hunter2")
	exclude(t, ac, base, "name", "seen.example.com").Body.Close()
	id := f.exclusions[0].ID

	vc := login(t, base, "viewer", "hunter2hunter2")

	// The viewer is denied both mutations.
	resp := exclude(t, vc, base, "name", "nope.example.com")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer exclude: status=%d, want 403", resp.StatusCode)
	}
	resp = unexclude(t, vc, base, id)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer un-exclude: status=%d, want 403", resp.StatusCode)
	}
	if len(f.exclusions) != 1 {
		t.Fatalf("exclusions after denied viewer acts = %d, want 1", len(f.exclusions))
	}

	// But the viewer can read the list, and neither exclusion control is offered.
	page := seedsBody(t, vc, base)
	if !strings.Contains(page, "seen.example.com") {
		t.Errorf("viewer cannot see the exclusions list; body: %s", page)
	}
	if strings.Contains(page, `action="/exclusions"`) || strings.Contains(page, `action="/exclusions/delete"`) {
		t.Errorf("an exclusion control was shown to a viewer; body: %s", page)
	}
}

func TestExcludeRequiresLogin(t *testing.T) {
	base := start(t, newFakeStore(), "")
	c := newClient(t)
	resp := exclude(t, c, base, "name", "example.com")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon exclude: status=%d location=%q, want redirect to /login", resp.StatusCode, resp.Header.Get("Location"))
	}
}
