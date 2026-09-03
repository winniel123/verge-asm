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

	// A name scope. The save lands back on Scope, now carrying a post-redirect-get
	// toast in the query (PARITY-CHART P1.7), so match the path, not the whole URL.
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
	// The frozen scope.tmpl renders each declared scope as a chip carrying the scope
	// itself (#574); the kind is inferred from the value's shape and no longer shown as a
	// per-chip badge.
}

func TestAddressScopeOverCapRejected(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// /21 is 2048 addresses, over the 1024 cap. The refusal is a post-redirect-get
	// (ADR-0130 §1), so the callout is read off the landing GET.
	got := refusalPage(t, ac, base, declare(t, ac, base, "address", "10.0.0.0/21"))
	// #21a: over-cap declarations are REFUSED, not auto-corrected — the RefusalCallout
	// names the span against the cap.
	if !strings.Contains(got, "over the 1,024-address cap") {
		t.Fatalf("over-cap scope not rejected clearly; body=%s", got)
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

// scopeMain returns the Scope screen's <main> region, excluding shell chrome (the
// command palette lists current Names per P1.5, so a page-wide match could find a
// name outside the tree).
func scopeMain(body string) string {
	i := strings.Index(body, "<main")
	j := strings.LastIndex(body, "</main>")
	if i < 0 || j < 0 || j < i {
		return body
	}
	return body[i:j]
}

// The Scope screen renders the declared name tree (SPEC-CHANGE collision #12,
// ADR-0116): each declared name scope is a registrable-domain root, every in-estate
// name under it is a leaf, and each leaf carries its own max-of-firing-signals
// severity — degrading to no dot where a name raises no signal.
func TestScopeDeclaredNameTree(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	// A name resolving an internal address from the internet class fires
	// non-globally-reachable-address-resolved-from-internet (severity medium) on the
	// Name, so its leaf carries a medium severity dot.
	f.addClassResolution(t, "leak.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["10.0.0.5"]}`)
	// A name resolving a public address raises no such signal, so its leaf carries no
	// severity dot — the spec's per-leaf empty pattern.
	f.addClassResolution(t, "www.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["93.184.216.34"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	main := scopeMain(seedsBody(t, ac, base))

	// The card renders with its heading and the tree container (frozen scope.tmpl, #574).
	for _, want := range []string{"Declared name tree", `class="sc-tree"`} {
		if !strings.Contains(main, want) {
			t.Errorf("scope missing name-tree marker %q", want)
		}
	}
	// The registrable domain is the root, with its two leaf names under it, labelled
	// relative to the domain.
	for _, want := range []string{
		`class="tl">example.com<`, // registrable-domain root
		`class="tc">2<`,           // two leaves
		`class="tl">leak<`,        // leaf name, relative to the domain
		`class="tl">www<`,
	} {
		if !strings.Contains(main, want) {
			t.Errorf("name tree missing %q; body: %s", want, main)
		}
	}
	// The signalling leaf carries its rule's real severity dot (medium) — a built
	// datum, never fabricated.
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
