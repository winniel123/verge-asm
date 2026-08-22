package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func setCustody(t *testing.T, c *http.Client, base string, id int64, extend bool) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/seeds/custody", url.Values{
		"id": {strconv.FormatInt(id, 10)}, "extend": {strconv.FormatBool(extend)},
	})
}

// A name scope carries no custody extension until one is declared, and the admin
// can declare it and withdraw it again — the flag toggling both ways.
func TestDeclareAndWithdrawCustodyExtension(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	if len(f.seeds) != 1 {
		t.Fatalf("seeds after declare = %d, want 1", len(f.seeds))
	}
	id := f.seeds[0].ID

	// Off by default: stored false, and the scope reads "off" with a declare control.
	if f.seeds[0].CustodyExtension {
		t.Fatalf("custody extension on by default, want off")
	}
	page := seedsBody(t, ac, base)
	if !strings.Contains(page, "Declare extension") {
		t.Errorf("declare-extension control not offered; body: %s", page)
	}

	// Declare it.
	resp := setCustody(t, ac, base, id, true)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/scope" {
		t.Fatalf("declare custody: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	if !f.seeds[0].CustodyExtension {
		t.Fatalf("custody extension not stored on declare")
	}
	page = seedsBody(t, ac, base)
	if !strings.Contains(page, "extension on") || !strings.Contains(page, "Withdraw") {
		t.Errorf("declared extension not reflected with a withdraw control; body: %s", page)
	}

	// Withdraw it.
	resp = setCustody(t, ac, base, id, false)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("withdraw custody: status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	if f.seeds[0].CustodyExtension {
		t.Fatalf("custody extension not cleared on withdraw")
	}
}

// The census renders for a declared extension and is display-only: it states so,
// carries no denominator, and offers no per-address approve affordance — the
// leaked-affordance failure mode from #123.
func TestCustodyCensusIsDisplayOnly(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	id := f.seeds[0].ID

	// No census before the extension is declared.
	if page := seedsBody(t, ac, base); strings.Contains(page, "Covered addresses") {
		t.Errorf("census shown before extension declared; body: %s", page)
	}

	setCustody(t, ac, base, id, true).Body.Close()
	page := seedsBody(t, ac, base)
	if !strings.Contains(page, "Covered addresses") || !strings.Contains(page, "Display only") {
		t.Errorf("census not rendered display-only; body: %s", page)
	}
	if !strings.Contains(page, "No addresses measured yet") {
		t.Errorf("census not rendered as empty/unavailable; body: %s", page)
	}
	// No per-address approve control anywhere on the page.
	if strings.Contains(page, ">Approve<") || strings.Contains(page, `name="approve"`) {
		t.Errorf("census leaked a per-address approve affordance; body: %s", page)
	}
}

// A custody extension is a property of a name scope alone: an address scope is
// never offered the control, and a direct write against its id is a no-op.
func TestCustodyExtensionNameScopeOnly(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "address", "203.0.113.0/24").Body.Close()
	addrID := f.seeds[0].ID

	// The custody section names no address scope, and offers no control over one.
	if page := seedsBody(t, ac, base); strings.Contains(page, "203.0.113.0/24") && strings.Contains(page, "Declare extension") {
		t.Errorf("custody control offered over an address scope; body: %s", page)
	}

	// A direct write against the address scope's id is a no-op, matching the SQL guard.
	setCustody(t, ac, base, addrID, true).Body.Close()
	if f.seeds[0].CustodyExtension {
		t.Fatalf("custody extension set on an address scope, want no-op")
	}
}

func TestViewerCannotToggleCustodyButCanView(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	ac := login(t, base, "admin", "hunter2hunter2")
	declare(t, ac, base, "name", "example.com").Body.Close()
	id := f.seeds[0].ID
	setCustody(t, ac, base, id, true).Body.Close()

	vc := login(t, base, "viewer", "hunter2hunter2")

	// The viewer is denied the mutation.
	resp := setCustody(t, vc, base, id, false)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer set custody: status=%d, want 403", resp.StatusCode)
	}
	if !f.seeds[0].CustodyExtension {
		t.Fatalf("viewer's denied act still moved the flag")
	}

	// But the viewer sees the scope and its census, with no toggle control offered.
	page := seedsBody(t, vc, base)
	if !strings.Contains(page, "example.com") || !strings.Contains(page, "Covered addresses") {
		t.Errorf("viewer cannot see the custody scope or census; body: %s", page)
	}
	if strings.Contains(page, `action="/seeds/custody"`) {
		t.Errorf("custody toggle shown to a viewer; body: %s", page)
	}
}

func TestSetCustodyRequiresLogin(t *testing.T) {
	base := start(t, newFakeStore(), "")
	c := newClient(t)
	resp := setCustody(t, c, base, 1, true)
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon set custody: status=%d location=%q, want redirect to /login", resp.StatusCode, resp.Header.Get("Location"))
	}
}
