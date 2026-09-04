package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

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

func countLiveSessions(f *fakeStore, accountID int64) int {
	n := 0
	for _, sess := range f.sessions {
		if sess.AccountID == accountID && !sess.RevokedAt.Valid {
			n++
		}
	}
	return n
}

func TestAdminSessionsListsEveryAccount(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "boyd", roleViewer, "hunter2hunter2")
	seedAccount(t, f, "cara", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	login(t, base, "boyd", "hunter2hunter2")
	login(t, base, "cara", "hunter2hunter2")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := getBody(t, ac, base+"/settings?tab=sessions", http.StatusOK)
	for _, want := range []string{"Active sessions", "admin", "boyd", "cara"} {
		if !strings.Contains(got, want) {
			t.Fatalf("sessions surface missing %q; body: %s", want, got)
		}
	}
	if !strings.Contains(got, "(you)") {
		t.Fatalf("admin's own current session not marked \"(you)\"; body: %s", got)
	}
	if got, want := strings.Count(got, `href="/settings?tab=sessions&revoke=`), 2; got != want {
		t.Fatalf("per-session revoke controls = %d, want %d", got, want)
	}
}

func TestAdminRevokeSingleSession(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	boyd := seedAccount(t, f, "boyd", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	bc := login(t, base, "boyd", "hunter2hunter2")
	ac := login(t, base, "admin", "hunter2hunter2")

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

	resp, err := bc.Get(base + "/profile")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("revoked session next request: status=%d loc=%q, want 303 to /login", resp.StatusCode, resp.Header.Get("Location"))
	}

	if !strings.Contains(getBody(t, ac, base+"/settings?tab=sessions", http.StatusOK), "admin") {
		t.Fatalf("admin session should survive another account's revoke")
	}
}

func TestAdminRevokeAllForAccount(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	boyd := seedAccount(t, f, "boyd", roleViewer, "hunter2hunter2")
	cara := seedAccount(t, f, "cara", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	login(t, base, "boyd", "hunter2hunter2")
	login(t, base, "boyd", "hunter2hunter2")
	login(t, base, "cara", "hunter2hunter2")
	ac := login(t, base, "admin", "hunter2hunter2")

	if countLiveSessions(f, boyd.ID) != 2 {
		t.Fatalf("boyd live sessions = %d, want 2", countLiveSessions(f, boyd.ID))
	}

	const sessionsTab = "/settings?tab=sessions"
	resp := postForm(t, ac, base+"/settings/sessions/revoke-account", url.Values{
		"account_id": {itoa(boyd.ID)}, "confirm_name": {"nope"},
		"return": {sessionsTab + "&revoke-account=" + itoa(boyd.ID)},
	})
	if loc := submitLoc(t, resp); loc != sessionsTab {
		t.Fatalf("refused revoke-all landed at %q, want %q", loc, sessionsTab)
	}
	got := getBody(t, ac, base+sessionsTab, http.StatusOK)
	if !strings.Contains(got, "That did not match") {
		t.Fatalf("mismatch typed-name missing re-open error; body: %s", got)
	}
	if countLiveSessions(f, boyd.ID) != 2 {
		t.Fatalf("boyd sessions changed on a rejected confirm: %d, want 2", countLiveSessions(f, boyd.ID))
	}

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
	if countLiveSessions(f, cara.ID) != 1 {
		t.Fatalf("cara live sessions = %d, want 1 (must survive Boyd's offboard)", countLiveSessions(f, cara.ID))
	}
}

func TestSessionsAdminOnly(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	boyd := seedAccount(t, f, "boyd", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	vc := login(t, base, "boyd", "hunter2hunter2")

	resp, err := vc.Get(base + "/settings?tab=sessions")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer GET /settings?tab=sessions: status=%d, want 403", resp.StatusCode)
	}

	sid := liveSessionID(t, f, boyd.ID)
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
	if countLiveSessions(f, boyd.ID) != 1 {
		t.Fatalf("viewer session count = %d after refused mutations, want 1", countLiveSessions(f, boyd.ID))
	}
}
