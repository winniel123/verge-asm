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

func authed(t *testing.T, c *http.Client, base string) bool {
	// The home renders whatever the estate holds, so no fixture separates the two answers.
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

func TestRevokeTakesEffectNextRequest(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := login(t, base, "admin", "hunter2hunter2")
	if !authed(t, c, base) {
		t.Fatal("authed page unreachable before revoke")
	}

	// Leaving the cookie in the jar isolates row validation from the handlers' clearing.
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

func TestEndSessionHandlerRevokesRow(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := login(t, base, "admin", "hunter2hunter2")
	resp := postForm(t, c, base+"/profile/session/revoke", url.Values{})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	// The redirect carries a toast so it fires on the sign-in page, hence the prefix test.
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, "/login") {
		t.Fatalf("end-session: status=%d loc=%q, want 303 -> /login", resp.StatusCode, loc)
	}
	if len(f.sessions) != 1 || !f.sessions[0].RevokedAt.Valid {
		t.Fatalf("end-session did not stamp revoked_at on the session row: rows=%d", len(f.sessions))
	}
	if authed(t, c, base) {
		t.Fatal("session still authenticates after end-session")
	}
}

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

	postForm(t, c1, base+"/profile/session/revoke", url.Values{}).Body.Close()

	if authed(t, c1, base) {
		t.Fatal("device 1 still authed after ending its session")
	}
	if !authed(t, c2, base) {
		t.Fatal("device 2 was signed out when device 1 ended its session")
	}
}

func TestExpiredSessionRowRejected(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := login(t, base, "admin", "hunter2hunter2")
	if !authed(t, c, base) {
		t.Fatal("authed page unreachable before expiry")
	}

	f.sessions[0].ExpiresAt = pgtype.Timestamptz{Time: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	if authed(t, c, base) {
		t.Fatal("expired session row still authenticated")
	}
}

func TestPreRegistryCookieRejected(t *testing.T) {
	f := newFakeStore()
	acct := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := newClient(t)
	// Signed with the test key and dated past fixedClock, so VerifySession passes and only
	// the missing registry row can stop it (#405).
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
