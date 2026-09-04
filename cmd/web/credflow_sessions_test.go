package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func decodeToast(t *testing.T, loc string) map[string]string {
	t.Helper()
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse redirect %q: %v", loc, err)
	}
	raw := u.Query().Get("toast")
	if raw == "" {
		t.Fatalf("redirect %q carries no toast payload", loc)
	}
	blob, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode toast %q: %v", raw, err)
	}
	var m map[string]string
	if err := json.Unmarshal(blob, &m); err != nil {
		t.Fatalf("unmarshal toast %q: %v", string(blob), err)
	}
	return m
}

func bounced(t *testing.T, c *http.Client, base string) bool {
	t.Helper()
	resp, err := c.Get(base + "/profile")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusSeeOther && resp.Header.Get("Location") == "/login"
}

func TestChangePasswordSignsOutOtherSessions(t *testing.T) {
	f := newFakeStore()
	ola := seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	c1 := loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (Macintosh) Firefox/120.0")
	c2 := loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0")

	if got := countLiveSessions(f, ola.ID); got != 2 {
		t.Fatalf("live sessions before change = %d, want 2", got)
	}

	resp := postForm(t, c1, base+"/profile/password", url.Values{
		"current_password": {"hunter2hunter2"}, "new_password": {"brandnewpass99"},
	})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, "/profile?toast=") {
		t.Fatalf("change password: status=%d loc=%q, want 303 to /profile?toast=…", resp.StatusCode, loc)
	}
	toast := decodeToast(t, loc)
	if toast["tone"] != "ok" || toast["title"] != "Password changed" ||
		toast["description"] != "Other sessions keep working until they expire." {
		t.Fatalf("change-password toast = %+v, want ok/Password changed/Other sessions keep working until they expire.", toast)
	}

	if got := countLiveSessions(f, ola.ID); got != 1 {
		t.Fatalf("live sessions after change = %d, want 1", got)
	}

	getBody(t, c1, base+"/profile", http.StatusOK)

	if !bounced(t, c2, base) {
		t.Fatal("second session survived a password change — other sessions not revoked")
	}
}

func TestChangePasswordKeepsActingSession(t *testing.T) {
	f := newFakeStore()
	ola := seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	c1 := loginUA(t, base, "ola", "hunter2hunter2", "Mozilla/5.0 (Macintosh) Firefox/120.0")

	resp := postForm(t, c1, base+"/profile/password", url.Values{
		"current_password": {"hunter2hunter2"}, "new_password": {"brandnewpass99"},
	})
	resp.Body.Close()

	if got := countLiveSessions(f, ola.ID); got != 1 {
		t.Fatalf("live sessions after solo change = %d, want 1 (acting session kept)", got)
	}
	if bounced(t, c1, base) {
		t.Fatal("acting session was signed out of its own password change")
	}
}

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

	if got := countLiveSessions(f, ola.ID); got != 0 {
		t.Fatalf("live sessions after reset = %d, want 0", got)
	}
	if !bounced(t, c1, base) {
		t.Fatal("session c1 survived a password reset — all sessions should be revoked")
	}
	if !bounced(t, c2, base) {
		t.Fatal("session c2 survived a password reset — all sessions should be revoked")
	}
}
