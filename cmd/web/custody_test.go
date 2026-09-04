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

	if f.seeds[0].CustodyExtension {
		t.Fatalf("custody extension on by default, want off")
	}
	page := seedsBody(t, ac, base)
	if !strings.Contains(page, `aria-label="Extend custody — example.com"`) {
		t.Errorf("custody toggle not offered; body: %s", page)
	}
	if strings.Contains(page, `aria-checked="true"`) {
		t.Errorf("custody shown on before it was declared; body: %s", page)
	}

	resp := setCustody(t, ac, base, id, true)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/scope" {
		t.Fatalf("declare custody: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	if !f.seeds[0].CustodyExtension {
		t.Fatalf("custody extension not stored on declare")
	}
	page = seedsBody(t, ac, base)
	if !strings.Contains(page, `aria-checked="true"`) || !strings.Contains(page, "recomputed each batch") {
		t.Errorf("declared extension not reflected with the on switch + census; body: %s", page)
	}

	resp = setCustody(t, ac, base, id, false)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("withdraw custody: status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	if f.seeds[0].CustodyExtension {
		t.Fatalf("custody extension not cleared on withdraw")
	}
}

func TestCustodyCensusIsDisplayOnly(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	id := f.seeds[0].ID

	if page := seedsBody(t, ac, base); strings.Contains(page, "recomputed each batch") {
		t.Errorf("census shown before extension declared; body: %s", page)
	}

	setCustody(t, ac, base, id, true).Body.Close()
	page := seedsBody(t, ac, base)
	if !strings.Contains(page, "census ·") || !strings.Contains(page, "recomputed each batch — read-only, never per-address approval") {
		t.Errorf("census not rendered display-only; body: %s", page)
	}
	if strings.Contains(page, ">Approve<") || strings.Contains(page, `name="approve"`) {
		t.Errorf("census leaked a per-address approve affordance; body: %s", page)
	}
}

func TestCustodyExtensionNameScopeOnly(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "address", "203.0.113.0/24").Body.Close()
	addrID := f.seeds[0].ID

	if page := seedsBody(t, ac, base); strings.Contains(page, "203.0.113.0/24") && strings.Contains(page, "Declare extension") {
		t.Errorf("custody control offered over an address scope; body: %s", page)
	}

	// The fake repeats custody.sql's kind='name' guard; the handler itself checks no kind.
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

	resp := setCustody(t, vc, base, id, false)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer set custody: status=%d, want 403", resp.StatusCode)
	}
	if !f.seeds[0].CustodyExtension {
		t.Fatalf("viewer's denied act still moved the flag")
	}

	page := seedsBody(t, vc, base)
	if !strings.Contains(page, "example.com") || !strings.Contains(page, "recomputed each batch") {
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
