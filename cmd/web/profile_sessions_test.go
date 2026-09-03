package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// --- personal sessions surface (#406, ADR-0117) ----------------------------
//
// The Profile lists an account's OWN live sessions read from the registry, marks the
// current one, and offers a per-row revoke plus "sign out others". These exercise the
// real store fakes (ListSessionsForAccount / RevokeSession / RevokeOtherSessionsForAccount)
// through the handlers — every figure is a real read, owner-scoped in the query.

// loginUA runs a password-only login carrying a chosen User-Agent, so the session row it
// opens stores that agent and the sessions table renders a distinct device label. Each
// call returns a fresh authenticated client (its own cookie jar = its own session).
func loginUA(t *testing.T, base, username, password, ua string) *http.Client {
	t.Helper()
	c := newClient(t)
	form := url.Values{"username": {username}, "password": {password}}
	req, err := http.NewRequest(http.MethodPost, base+"/login", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", ua)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("login %s: %v", username, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("login %s: status = %d, want 303", username, resp.StatusCode)
	}
	return c
}

func sessionIDByUA(f *fakeStore, accountID int64, uaSubstr string) int64 {
	for _, s := range f.sessions {
		if s.AccountID == accountID && !s.RevokedAt.Valid && strings.Contains(s.UserAgent, uaSubstr) {
			return s.ID
		}
	}
	return 0
}

// The sessions card lists the account's real live sessions and marks exactly the one
// making the request with the "this device" badge; every other device is a plain row.
func TestProfileListsOwnSessions(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	c1 := loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15) Firefox/120.0")
	loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (Windows NT 10.0; Win64) Chrome/120.0")

	got := getBody(t, c1, base+"/profile", http.StatusOK)
	for _, want := range []string{
		"Firefox · macOS",  // the requesting device (rendered via sessionDeviceFromUA)
		"Chrome · Windows", // the other live session
		"this device",      // the current-session badge
		"Sign out others",  // the card action appears once there is more than one session
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("profile sessions missing %q; body: %s", want, got)
		}
	}
	if n := strings.Count(got, "this device"); n != 1 {
		t.Fatalf("current-session badge count = %d, want exactly 1", n)
	}
}

// Revoke-one ends a chosen OTHER session: it drops from the next render and the revoked
// client is bounced to /login on its next request, while the acting session stays live.
func TestProfileRevokeOneSession(t *testing.T) {
	f := newFakeStore()
	ola := seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	c1 := loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (Macintosh) Firefox/120.0") // current
	c2 := loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0")

	other := sessionIDByUA(f, ola.ID, "Chrome")
	if other == 0 {
		t.Fatal("could not find the second session")
	}

	resp := postForm(t, c1, base+"/profile/sessions/revoke", url.Values{"id": {strconv.FormatInt(other, 10)}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, "/profile") {
		t.Fatalf("revoke other: status=%d loc=%q, want 303 to /profile", resp.StatusCode, loc)
	}
	// The act result rides the shell toast (#18) naming the revoked device (Profile.jsx:100).
	toast := decodeToast(t, loc)
	if toast["tone"] != "neutral" || toast["title"] != "Session revoked" ||
		!strings.HasSuffix(toast["description"], " signs out on its next request.") {
		t.Fatalf("session-revoke toast = %+v, want neutral/Session revoked/<device> signs out on its next request.", toast)
	}

	// The revoked row is gone from the acting session's next render, and the badge still
	// marks exactly the surviving current session.
	got := getBody(t, c1, base+"/profile", http.StatusOK)
	if strings.Contains(got, "Chrome") {
		t.Fatalf("revoked session still listed; body: %s", got)
	}

	// The revoked client resolves no live session and is bounced to /login next request.
	resp2, err := c2.Get(base + "/profile")
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusSeeOther || resp2.Header.Get("Location") != "/login" {
		t.Fatalf("revoked client next request: status=%d loc=%q, want 303 to /login", resp2.StatusCode, resp2.Header.Get("Location"))
	}
}

// Owner-scoping: a posted id belonging to ANOTHER account cannot revoke that account's
// session — the account_id predicate in the query makes it a no-op, so the foreign
// session survives and its client stays signed in.
func TestProfileRevokeOneSessionForeignIsNoOp(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	kai := seedAccount(t, f, "kai", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	c1 := loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (Macintosh) Firefox/120.0")
	cKai := loginUA(t, base, "kai", "hunter2hunter2", "Mozilla/5.0 (iPhone) Safari/605.1")

	kaiSess := sessionIDByUA(f, kai.ID, "Safari")
	if kaiSess == 0 {
		t.Fatal("could not find kai's session")
	}

	// ola tries to revoke kai's session by id — owner-scoped, so nothing happens.
	resp := postForm(t, c1, base+"/profile/sessions/revoke", url.Values{"id": {strconv.FormatInt(kaiSess, 10)}})
	resp.Body.Close()

	if id := sessionIDByUA(f, kai.ID, "Safari"); id == 0 {
		t.Fatal("kai's session was revoked by another account — owner-scoping breached")
	}
	// kai is still signed in and can load their own Profile.
	getBody(t, cKai, base+"/profile", http.StatusOK)
}

// Revoking the CURRENT session from the sessions card ends the acting session: the
// response redirects to /login (the caller just signed itself out here).
func TestProfileRevokeCurrentSessionSignsOut(t *testing.T) {
	f := newFakeStore()
	ola := seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	c1 := loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (Macintosh) Firefox/120.0")
	cur := sessionIDByUA(f, ola.ID, "Firefox")

	resp := postForm(t, c1, base+"/profile/sessions/revoke", url.Values{"id": {strconv.FormatInt(cur, 10)}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || loc != "/login" {
		t.Fatalf("revoke current: status=%d loc=%q, want 303 to /login", resp.StatusCode, loc)
	}
}

// Sign-out-others revokes every OTHER live session and keeps the current one: the acting
// client still loads its Profile, while the others are bounced to /login.
func TestProfileSignOutOthers(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	c1 := loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (Macintosh) Firefox/120.0") // current
	c2 := loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0")
	c3 := loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (iPhone) Safari/605.1")

	resp := postForm(t, c1, base+"/profile/sessions/revoke-others", url.Values{})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, "/profile") {
		t.Fatalf("sign out others: status=%d loc=%q, want 303 to /profile", resp.StatusCode, loc)
	}
	// Two other sessions were signed out, so the act toast (#18) reads "2 sessions ended."
	// (Profile.jsx:158).
	toast := decodeToast(t, loc)
	if toast["tone"] != "neutral" || toast["title"] != "Other sessions signed out" ||
		toast["description"] != "2 sessions ended." {
		t.Fatalf("sign-out-others toast = %+v, want neutral/Other sessions signed out/2 sessions ended.", toast)
	}

	// The acting session keeps working and no longer lists the others.
	got := getBody(t, c1, base+"/profile", http.StatusOK)
	if strings.Contains(got, "Chrome") || strings.Contains(got, "iOS") {
		t.Fatalf("other sessions still listed after sign-out-others; body: %s", got)
	}

	// The other two are bounced to /login on their next request.
	for name, c := range map[string]*http.Client{"c2": c2, "c3": c3} {
		resp, err := c.Get(base + "/profile")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
			t.Fatalf("%s after sign-out-others: status=%d loc=%q, want 303 to /login", name, resp.StatusCode, resp.Header.Get("Location"))
		}
	}
}

// The two new mutation routes reject an anonymous caller, redirecting to /login rather
// than acting — a Profile mutation is never reachable unauthenticated.
func TestProfileSessionRoutesRequireLogin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	c := newClient(t)

	for _, path := range []string{"/profile/sessions/revoke", "/profile/sessions/revoke-others"} {
		resp := postForm(t, c, base+path, url.Values{})
		resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
			t.Fatalf("POST %s anon: status=%d loc=%q, want 303 to /login", path, resp.StatusCode, resp.Header.Get("Location"))
		}
	}
}
