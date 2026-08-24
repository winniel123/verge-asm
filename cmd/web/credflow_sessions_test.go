package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// --- credential changes revoke other sessions (#408, ADR-0118) --------------
//
// Once sessions are a server-side registry, a credential change can actually sign the
// old credential out everywhere. These drive the two credential flows through the real
// handlers and prove the revocation lands: a password change keeps the acting session
// and kills the rest; a reset kills all of them (the user re-logs in).

// (countLiveSessions lives in settings_sessions_test.go — reused here to prove a revoke
// landed in the registry.)

// bounced reports whether a client's next authed request is redirected to /login —
// the observable signature of a session whose registry row was revoked.
func bounced(t *testing.T, c *http.Client, base string) bool {
	t.Helper()
	resp, err := c.Get(base + "/profile")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusSeeOther && resp.Header.Get("Location") == "/login"
}

// A self-service password change signs out every OTHER session for the account while
// keeping the tab that made the change alive: the acting client still loads its Profile,
// a second client is bounced to /login on its next request, and the registry holds
// exactly one live session afterward.
func TestChangePasswordSignsOutOtherSessions(t *testing.T) {
	f := newFakeStore()
	ola := seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	c1 := loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (Macintosh) Firefox/120.0") // makes the change
	c2 := loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0")

	if got := countLiveSessions(f, ola.ID); got != 2 {
		t.Fatalf("live sessions before change = %d, want 2", got)
	}

	resp := postForm(t, c1, base+"/profile/password", url.Values{
		"current_password": {"hunter2hunter2"}, "new_password": {"brandnewpass99"},
	})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || loc != "/profile?saved=1" {
		t.Fatalf("change password: status=%d loc=%q, want 303 to /profile?saved=1", resp.StatusCode, loc)
	}

	// Exactly one live session remains — the others were revoked.
	if got := countLiveSessions(f, ola.ID); got != 1 {
		t.Fatalf("live sessions after change = %d, want 1", got)
	}

	// The acting session keeps working and, following the redirect, sees the honest notice.
	got := getBody(t, c1, base+"/profile?saved=1", http.StatusOK)
	if !strings.Contains(got, "Every other signed-in session has been signed out") {
		t.Fatalf("change-password notice missing the global sign-out copy; body: %s", got)
	}

	// The other client is bounced to /login on its next request.
	if !bounced(t, c2, base) {
		t.Fatal("second session survived a password change — other sessions not revoked")
	}
}

// If the current session id cannot be resolved on the change, the handler skips the
// revoke rather than sweep with no exception — it must never sign the caller out of the
// tab they just changed the password from. (Belt-and-braces on the ok=false branch.)
func TestChangePasswordKeepsActingSession(t *testing.T) {
	f := newFakeStore()
	ola := seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	c1 := loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (Macintosh) Firefox/120.0")

	resp := postForm(t, c1, base+"/profile/password", url.Values{
		"current_password": {"hunter2hunter2"}, "new_password": {"brandnewpass99"},
	})
	resp.Body.Close()

	// The one session that made the change is still live and still usable.
	if got := countLiveSessions(f, ola.ID); got != 1 {
		t.Fatalf("live sessions after solo change = %d, want 1 (acting session kept)", got)
	}
	if bounced(t, c1, base) {
		t.Fatal("acting session was signed out of its own password change")
	}
}

// A password reset signs out ALL of the account's sessions with no exception — there is
// no acting session to keep, the owner re-logs in. Both pre-existing clients are bounced
// to /login and the registry holds no live session for the account.
func TestResetSignsOutAllSessions(t *testing.T) {
	f := newFakeStore()
	ola := seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	c1 := loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (Macintosh) Firefox/120.0")
	c2 := loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0")
	if got := countLiveSessions(f, ola.ID); got != 2 {
		t.Fatalf("live sessions before reset = %d, want 2", got)
	}

	addReset(t, f, ola.ID, "tok", serverClock.Add(time.Hour))
	resp := postForm(t, newClient(t), base+"/reset", url.Values{
		"token": {"tok"}, "password": {"brandnewpass99"}, "confirm": {"brandnewpass99"},
	})
	resp.Body.Close()

	// No live session survives a reset.
	if got := countLiveSessions(f, ola.ID); got != 0 {
		t.Fatalf("live sessions after reset = %d, want 0", got)
	}
	// Both pre-existing clients are bounced to /login on their next request.
	if !bounced(t, c1, base) {
		t.Fatal("session c1 survived a password reset — all sessions should be revoked")
	}
	if !bounced(t, c2, base) {
		t.Fatal("session c2 survived a password reset — all sessions should be revoked")
	}
}
