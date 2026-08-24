package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// liveSessionID returns the id of an account's single live (unrevoked) session in the
// fake registry, failing the test if none is found. A signed-in account holds exactly
// one session per login client.
func liveSessionID(t *testing.T, f *fakeStore, accountID int64) int64 {
	t.Helper()
	for _, sess := range f.sessions {
		if sess.AccountID == accountID && !sess.RevokedAt.Valid {
			return sess.ID
		}
	}
	t.Fatalf("no live session for account %d", accountID)
	return 0
}

// countLiveSessions returns how many live (unrevoked) sessions an account holds in the
// fake registry.
func countLiveSessions(f *fakeStore, accountID int64) int {
	n := 0
	for _, sess := range f.sessions {
		if sess.AccountID == accountID && !sess.RevokedAt.Valid {
			n++
		}
	}
	return n
}

// The admin-wide Sessions surface lists every account's live session — not just the
// admin's own — joined to the owning account's username and role (#407).
func TestAdminSessionsListsEveryAccount(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "boyd", roleViewer, "hunter2hunter2")
	seedAccount(t, f, "cara", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	// Sign each account in so the registry holds a live session per account.
	login(t, base, "boyd", "hunter2hunter2")
	login(t, base, "cara", "hunter2hunter2")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := getBody(t, ac, base+"/settings?tab=sessions", http.StatusOK)
	for _, want := range []string{"Active sessions", "admin", "boyd", "cara"} {
		if !strings.Contains(got, want) {
			t.Fatalf("sessions surface missing %q; body: %s", want, got)
		}
	}
	// Every live account's session is present — three signed-in accounts, three rows.
	if got, want := strings.Count(got, `href="/settings?tab=sessions&amp;revoke=`), 3; got != want {
		t.Fatalf("per-session revoke controls = %d, want %d", got, want)
	}
}

// An admin revokes any single session by id; that session is dead at once — bounced to
// /login on its next request — and gone from the surface, while other sessions live on.
func TestAdminRevokeSingleSession(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	boyd := seedAccount(t, f, "boyd", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	bc := login(t, base, "boyd", "hunter2hunter2")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Boyd's session is live: a protected read succeeds (302-free 200 on /profile).
	getBody(t, bc, base+"/profile", http.StatusOK)

	sid := liveSessionID(t, f, boyd.ID)
	resp := postForm(t, ac, base+"/settings/sessions/revoke", url.Values{"id": {itoa(sid)}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, "/settings?tab=sessions") {
		t.Fatalf("admin revoke: status=%d loc=%q, want 303 to /settings?tab=sessions", resp.StatusCode, loc)
	}
	if countLiveSessions(f, boyd.ID) != 0 {
		t.Fatalf("boyd still has a live session after revoke")
	}

	// Boyd's very next request resolves no live session and is bounced to /login.
	resp, err := bc.Get(base + "/profile")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("revoked session next request: status=%d loc=%q, want 303 to /login", resp.StatusCode, resp.Header.Get("Location"))
	}

	// The admin's own session is untouched — the surface still renders for them.
	if !strings.Contains(getBody(t, ac, base+"/settings?tab=sessions", http.StatusOK), "admin") {
		t.Fatalf("admin session should survive another account's revoke")
	}
}

// Revoke-all-for-account (offboarding) kills every session of one account through a
// typed-name confirm, and leaves another account's sessions intact.
func TestAdminRevokeAllForAccount(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	boyd := seedAccount(t, f, "boyd", roleViewer, "hunter2hunter2")
	cara := seedAccount(t, f, "cara", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	// Boyd signs in on two devices; Cara on one.
	login(t, base, "boyd", "hunter2hunter2")
	login(t, base, "boyd", "hunter2hunter2")
	login(t, base, "cara", "hunter2hunter2")
	ac := login(t, base, "admin", "hunter2hunter2")

	if countLiveSessions(f, boyd.ID) != 2 {
		t.Fatalf("boyd live sessions = %d, want 2", countLiveSessions(f, boyd.ID))
	}

	// A wrong typed name changes nothing and re-opens the dialog with an error.
	resp := postForm(t, ac, base+"/settings/sessions/revoke-account", url.Values{
		"account_id": {itoa(boyd.ID)}, "confirm_name": {"nope"},
	})
	got := body(t, resp)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("mismatch typed-name: status=%d, want 400", resp.StatusCode)
	}
	if !strings.Contains(got, "That did not match") {
		t.Fatalf("mismatch typed-name missing re-open error; body: %s", got)
	}
	if countLiveSessions(f, boyd.ID) != 2 {
		t.Fatalf("boyd sessions changed on a rejected confirm: %d, want 2", countLiveSessions(f, boyd.ID))
	}

	// The exact username offboards Boyd: every session of his is revoked.
	resp = postForm(t, ac, base+"/settings/sessions/revoke-account", url.Values{
		"account_id": {itoa(boyd.ID)}, "confirm_name": {"boyd"},
	})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, "/settings?tab=sessions") {
		t.Fatalf("revoke-all: status=%d loc=%q, want 303 to /settings?tab=sessions", resp.StatusCode, loc)
	}
	if countLiveSessions(f, boyd.ID) != 0 {
		t.Fatalf("boyd live sessions after offboard = %d, want 0", countLiveSessions(f, boyd.ID))
	}
	// Cara's session is untouched.
	if countLiveSessions(f, cara.ID) != 1 {
		t.Fatalf("cara live sessions = %d, want 1 (must survive Boyd's offboard)", countLiveSessions(f, cara.ID))
	}
}

// A viewer (non-admin) is refused the whole Sessions surface and every mutation it
// hosts — requireAdmin gates the read and both writes.
func TestSessionsAdminOnly(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	boyd := seedAccount(t, f, "boyd", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	vc := login(t, base, "boyd", "hunter2hunter2")

	// The surface itself is admin-only (GET /settings is wholesale admin-gated).
	resp, err := vc.Get(base + "/settings?tab=sessions")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer GET /settings?tab=sessions: status=%d, want 403", resp.StatusCode)
	}

	sid := liveSessionID(t, f, boyd.ID)
	// Both mutation routes are refused for a viewer, and neither takes effect.
	for _, tc := range []struct {
		path string
		form url.Values
	}{
		{"/settings/sessions/revoke", url.Values{"id": {itoa(sid)}}},
		{"/settings/sessions/revoke-account", url.Values{"account_id": {itoa(boyd.ID)}, "confirm_name": {"boyd"}}},
	} {
		resp := postForm(t, vc, base+tc.path, tc.form)
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("viewer POST %s: status=%d, want 403", tc.path, resp.StatusCode)
		}
	}
	// The viewer's own session survived — no mutation ran.
	if countLiveSessions(f, boyd.ID) != 1 {
		t.Fatalf("viewer session count = %d after refused mutations, want 1", countLiveSessions(f, boyd.ID))
	}
}
