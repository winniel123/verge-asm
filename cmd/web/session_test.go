package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
)

// authed reports whether the client still holds a live session: GET / renders the
// authed home (200) for a valid session, and redirects to /login (303) once the
// session is gone. The home renders regardless of estate contents, so no fixtures are
// needed to distinguish the two.
func authed(t *testing.T, c *http.Client, base string) bool {
	t.Helper()
	resp, err := c.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return true
	case http.StatusSeeOther:
		if loc := resp.Header.Get("Location"); loc != "/login" {
			t.Fatalf("GET / redirected to %q, want /login", loc)
		}
		return false
	default:
		t.Fatalf("GET /: status = %d, want 200 or 303", resp.StatusCode)
		return false
	}
}

// A login opens exactly one live session row, and it is the row the request validates
// against.
func TestLoginOpensOneSessionRow(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := login(t, base, "admin", "hunter2hunter2")
	if len(f.sessions) != 1 {
		t.Fatalf("session rows after login = %d, want 1", len(f.sessions))
	}
	if f.sessions[0].RevokedAt.Valid {
		t.Fatal("new session row is already revoked")
	}
	if !authed(t, c, base) {
		t.Fatal("authed page not reachable right after login")
	}
}

// Revoking the current session's row takes effect on the very next request: the same,
// still-signed, still-unexpired cookie no longer authenticates. Revoking directly on the
// registry (keeping the cookie in the jar) isolates the per-request row validation from
// the cookie-clearing the sign-out/end-session handlers also do.
func TestRevokeTakesEffectNextRequest(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := login(t, base, "admin", "hunter2hunter2")
	if !authed(t, c, base) {
		t.Fatal("authed page unreachable before revoke")
	}

	// Revoke the row itself, leaving the client's cookie in place.
	if err := f.RevokeSession(t.Context(), db.RevokeSessionParams{
		ID:        f.sessions[0].ID,
		AccountID: f.sessions[0].AccountID,
		RevokedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	if !f.sessions[0].RevokedAt.Valid {
		t.Fatal("RevokeSession did not stamp revoked_at")
	}

	if authed(t, c, base) {
		t.Fatal("session still authenticates after its row was revoked")
	}
}

// TestEndSessionHandlerRevokesRow proves the end-session ConfirmDialog handler itself
// stamps revoked_at (and redirects to /login), not merely clears the cookie.
func TestEndSessionHandlerRevokesRow(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := login(t, base, "admin", "hunter2hunter2")
	resp := postForm(t, c, base+"/profile/session/revoke", url.Values{})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	// End-session carries the "Session ended" toast (#18) on the /login redirect so it
	// fires on the sign-in page, so the destination is /login?toast=… not bare /login.
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, "/login") {
		t.Fatalf("end-session: status=%d loc=%q, want 303 -> /login", resp.StatusCode, loc)
	}
	if len(f.sessions) != 1 || !f.sessions[0].RevokedAt.Valid {
		t.Fatalf("end-session did not stamp revoked_at on the session row: rows=%d", len(f.sessions))
	}
	// The cookie was cleared too, so the session is doubly gone.
	if authed(t, c, base) {
		t.Fatal("session still authenticates after end-session")
	}
}

// Sign-out revokes the server-side session row, not just the cookie.
func TestLogoutRevokesSessionRow(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := login(t, base, "admin", "hunter2hunter2")
	resp := postForm(t, c, base+"/logout", url.Values{})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("logout: status=%d loc=%q, want 303 -> /login", resp.StatusCode, resp.Header.Get("Location"))
	}
	if len(f.sessions) != 1 || !f.sessions[0].RevokedAt.Valid {
		t.Fatalf("logout did not revoke the session row: rows=%d", len(f.sessions))
	}
}

// Two logins for the same account open two independent rows: revoking one leaves the
// other working, so a second device is not signed out when the first ends its session.
func TestSecondDeviceIndependent(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c1 := login(t, base, "admin", "hunter2hunter2")
	c2 := login(t, base, "admin", "hunter2hunter2")
	if len(f.sessions) != 2 {
		t.Fatalf("session rows after two logins = %d, want 2", len(f.sessions))
	}
	if !authed(t, c1, base) || !authed(t, c2, base) {
		t.Fatal("both devices should be authed before any revoke")
	}

	// Device 1 ends its own session.
	postForm(t, c1, base+"/profile/session/revoke", url.Values{}).Body.Close()

	if authed(t, c1, base) {
		t.Fatal("device 1 still authed after ending its session")
	}
	if !authed(t, c2, base) {
		t.Fatal("device 2 was signed out when device 1 ended its session")
	}
}

// An expired session row is rejected even though the signed cookie is still within its
// own validity window — the row's expiry is an independent, server-side gate.
func TestExpiredSessionRowRejected(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := login(t, base, "admin", "hunter2hunter2")
	if !authed(t, c, base) {
		t.Fatal("authed page unreachable before expiry")
	}

	// Expire the row in the past relative to the server clock (fixedClock, 2026-08-15).
	// The cookie is untouched and still verifies, but the row is no longer live.
	f.sessions[0].ExpiresAt = pgtype.Timestamptz{Time: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	if authed(t, c, base) {
		t.Fatal("expired session row still authenticated")
	}
}

// A cookie with no session token (an old pre-registry cookie) resolves no row and is
// treated as absent, so the request is bounced to /login rather than trusted.
func TestPreRegistryCookieRejected(t *testing.T) {
	f := newFakeStore()
	acct := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := newClient(t)
	// Sign a KindSession cookie the OLD way: a valid, unexpired session claim carrying
	// no opaque token — exactly what a cookie minted before #405 looks like. It is
	// signed with the same test key and dated past the server's fixedClock, so it passes
	// VerifySession; only the missing registry row must stop it.
	old, err := auth.SignSession(testKey, auth.Session{
		AccountID: acct.ID, Kind: auth.KindSession,
		ExpiresAt: time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(base)
	c.Jar.SetCookies(u, []*http.Cookie{{Name: sessionCookie, Value: old}})

	if authed(t, c, base) {
		t.Fatal("a pre-registry cookie with no session row authenticated")
	}
}
