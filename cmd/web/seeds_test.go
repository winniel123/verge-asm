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

	resp := declare(t, ac, base, "name", "Example.com")
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/scope") {
		t.Fatalf("declare name: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()

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
}

func TestAddressScopeOverCapRejected(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := refusalPage(t, ac, base, declare(t, ac, base, "address", "10.0.0.0/21"))
	if !strings.Contains(got, "over the 1,024-address cap") {
		t.Fatalf("over-cap scope not rejected clearly; body=%s", got)
	}
	if !strings.Contains(got, `value="10.0.0.0/21"`) {
		t.Errorf("rejected scope not retained in the form; body: %s", got)
	}
	if len(f.seeds) != 0 {
		t.Fatalf("seeds after rejected declaration = %d, want 0", len(f.seeds))
	}

	resp := declare(t, ac, base, "address", "10.0.0.0/22")
	if resp.StatusCode != http.StatusSeeOther {
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

	got := refusalPage(t, ac, base, declare(t, ac, base, "name", "www.example.com"))
	if !strings.Contains(got, "registrable domain example.com") {
		t.Fatalf("subdomain not rejected toward its registrable domain; body=%s", got)
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
	got := refusalPage(t, ac, base, declare(t, ac, base, "name", "example.com"))
	if !strings.Contains(got, "already declared") {
		t.Fatalf("duplicate not reported; body: %s", got)
	}
}

func TestViewerCannotDeclareButCanView(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	ac := login(t, base, "admin", "hunter2hunter2")
	declare(t, ac, base, "name", "example.com").Body.Close()

	vc := login(t, base, "viewer", "hunter2hunter2")

	resp := declare(t, vc, base, "name", "example.org")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer declare: status=%d, want 403", resp.StatusCode)
	}
	if len(f.seeds) != 1 {
		t.Fatalf("seeds after denied declare = %d, want 1", len(f.seeds))
	}

	page := seedsBody(t, vc, base)
	if !strings.Contains(page, "example.com") {
		t.Errorf("viewer cannot see the seeds list; body: %s", page)
	}
	if strings.Contains(page, `action="/seeds"`) {
		t.Errorf("declare form shown to a viewer; body: %s", page)
	}
}

func scopeMain(body string) string {
	i := strings.Index(body, "<main")
	j := strings.LastIndex(body, "</main>")
	if i < 0 || j < 0 || j < i {
		return body
	}
	return body[i:j]
}

func TestScopeDeclaredNameTree(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	// An internal address resolved from the internet class fires a medium-severity Name rule.
	f.addClassResolution(t, "leak.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["10.0.0.5"]}`)
	f.addClassResolution(t, "www.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["93.184.216.34"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	main := scopeMain(seedsBody(t, ac, base))

	for _, want := range []string{"Declared name tree", `class="sc-tree"`} {
		if !strings.Contains(main, want) {
			t.Errorf("scope missing name-tree marker %q", want)
		}
	}
	for _, want := range []string{
		`class="tl">example.com<`,
		`class="tc">2<`,
		`class="tl">leak<`,
		`class="tl">www<`,
	} {
		if !strings.Contains(main, want) {
			t.Errorf("name tree missing %q; body: %s", want, main)
		}
	}
	if !strings.Contains(main, "var(--sev-medium-dot)") {
		t.Errorf("name tree leaf lost its severity dot; body: %s", main)
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
