package main

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
)

// start spins up the handler over httptest and returns the base URL. Clients
// do not auto-follow redirects, so tests can assert the 303 Location and the
// cookies set alongside it.
func start(t *testing.T, f *fakeStore, setupToken string) string {
	t.Helper()
	ts := httptest.NewServer(newServer(f, testKey, setupToken, fixedClock()).handler())
	t.Cleanup(ts.Close)
	return ts.URL
}

// startAt is start with the server clock pinned to now — the read instant every
// derivation read of the observation corpus is gated against (#237). Two servers
// over one fakeStore at different clocks read the same corpus across the live
// boundary without any delete.
func startAt(t *testing.T, f *fakeStore, now time.Time) string {
	t.Helper()
	ts := httptest.NewServer(newServer(f, testKey, "", func() time.Time { return now }).handler())
	t.Cleanup(ts.Close)
	return ts.URL
}

func newClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func postForm(t *testing.T, c *http.Client, url string, form url.Values) *http.Response {
	t.Helper()
	resp, err := c.PostForm(url, form)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func body(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// seedAccount inserts an account with a real bcrypt hash directly into the fake.
func seedAccount(t *testing.T, f *fakeStore, username, role, password string) db.Account {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	acct, err := f.CreateAccount(t.Context(), db.CreateAccountParams{Username: username, Role: role, PasswordHash: hash})
	if err != nil {
		t.Fatal(err)
	}
	return acct
}

// login runs a password-only login and returns the authenticated client.
func login(t *testing.T, base, username, password string) *http.Client {
	t.Helper()
	c := newClient(t)
	resp := postForm(t, c, base+"/login", url.Values{"username": {username}, "password": {password}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("login: status = %d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	return c
}

func hasCookie(c *http.Client, base, name string) bool {
	u, _ := url.Parse(base)
	for _, ck := range c.Jar.Cookies(u) {
		if ck.Name == name {
			return true
		}
	}
	return false
}

// --- setup / bootstrap -----------------------------------------------------

func TestSetupCreatesFirstAdminThenCloses(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "the-setup-token")

	// The form is served while the window is open.
	resp, err := http.Get(base + "/setup")
	if err != nil {
		t.Fatal(err)
	}
	if got := body(t, resp); !strings.Contains(got, "Create the first admin") {
		t.Fatalf("setup form missing; body: %s", got)
	}

	c := newClient(t)

	// A wrong token creates nothing.
	resp = postForm(t, c, base+"/setup", url.Values{"token": {"wrong"}, "username": {"admin"}, "password": {"hunter2hunter2"}})
	if got := body(t, resp); !strings.Contains(got, "Invalid setup token") {
		t.Fatalf("wrong token not rejected; body: %s", got)
	}
	if n, _ := f.CountAccounts(t.Context()); n != 0 {
		t.Fatalf("accounts after wrong token = %d, want 0", n)
	}

	// The correct token creates the first admin.
	resp = postForm(t, c, base+"/setup", url.Values{"token": {"the-setup-token"}, "username": {"admin"}, "password": {"hunter2hunter2"}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("setup success: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	n, _ := f.CountAccounts(t.Context())
	if n != 1 {
		t.Fatalf("accounts after setup = %d, want 1", n)
	}
	first, _ := f.GetAccountByUsername(t.Context(), "admin")
	if first.Role != roleAdmin {
		t.Fatalf("first account role = %q, want admin", first.Role)
	}

	// The token is single-use: presented again it does nothing, because an
	// account now exists.
	resp = postForm(t, c, base+"/setup", url.Values{"token": {"the-setup-token"}, "username": {"admin2"}, "password": {"hunter2hunter2"}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("reused token: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	if n, _ := f.CountAccounts(t.Context()); n != 1 {
		t.Fatalf("accounts after token reuse = %d, want 1 (single-use)", n)
	}

	// And the form itself is now closed.
	resp, err = newClient(t).Get(base + "/setup")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("setup form after bootstrap: status=%d, want 303", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestSetupRejectsShortPassword(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "tok")
	c := newClient(t)
	resp := postForm(t, c, base+"/setup", url.Values{"token": {"tok"}, "username": {"admin"}, "password": {"short"}})
	if got := body(t, resp); !strings.Contains(got, "at least 8") {
		t.Fatalf("short password not rejected; body: %s", got)
	}
	if n, _ := f.CountAccounts(t.Context()); n != 0 {
		t.Fatalf("accounts = %d, want 0", n)
	}
}

// --- login / session -------------------------------------------------------

func TestLoginAndSession(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := login(t, base, "admin", "hunter2hunter2")
	if !hasCookie(c, base, sessionCookie) {
		t.Fatal("no session cookie after login")
	}

	resp, err := c.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != http.StatusOK || !strings.Contains(got, "admin") || !strings.Contains(got, "Invite an account") {
		t.Fatalf("home not shown for admin; status=%d body=%s", resp.StatusCode, got)
	}
}

func TestHomeRequiresLogin(t *testing.T) {
	base := start(t, newFakeStore(), "")
	c := newClient(t)
	resp, err := c.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("home without login: status=%d location=%q, want redirect to /login", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := newClient(t)
	resp := postForm(t, c, base+"/login", url.Values{"username": {"admin"}, "password": {"wrong"}})
	if got := body(t, resp); !strings.Contains(got, "Invalid username or password") {
		t.Fatalf("bad password not rejected; body: %s", got)
	}
	if hasCookie(c, base, sessionCookie) {
		t.Fatal("session cookie set on failed login")
	}

	// An unknown username gives the same message (no user enumeration).
	resp = postForm(t, c, base+"/login", url.Values{"username": {"ghost"}, "password": {"whatever!"}})
	if got := body(t, resp); !strings.Contains(got, "Invalid username or password") {
		t.Fatalf("unknown user not rejected uniformly; body: %s", got)
	}
}

// --- permission check on the mutating endpoint -----------------------------

func TestViewerDeniedMutation(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	// A viewer is denied the account-creation mutation.
	vc := login(t, base, "viewer", "hunter2hunter2")
	resp := postForm(t, vc, base+"/accounts", url.Values{"username": {"eve"}, "password": {"hunter2hunter2"}, "role": {"admin"}})
	ct := resp.Header.Get("Content-Type")
	page403 := body(t, resp)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer mutation: status=%d, want 403", resp.StatusCode)
	}
	// The error page must carry a Content-Type, or the browser renders the
	// markup as plain text — and it must actually tell the viewer why.
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("403 Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(page403, "admin role") {
		t.Fatalf("403 page does not explain the denial: %s", page403)
	}
	if n, _ := f.CountAccounts(t.Context()); n != 2 {
		t.Fatalf("accounts after denied mutation = %d, want 2", n)
	}

	// An unauthenticated caller is redirected to login, never trusted.
	uc := newClient(t)
	resp = postForm(t, uc, base+"/accounts", url.Values{"username": {"eve"}, "password": {"hunter2hunter2"}, "role": {"admin"}})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("anon mutation: status=%d, want 303 to login", resp.StatusCode)
	}

	// An admin may perform it.
	ac := login(t, base, "admin", "hunter2hunter2")
	resp = postForm(t, ac, base+"/accounts", url.Values{"username": {"eve"}, "password": {"hunter2hunter2"}, "role": {"viewer"}})
	got := body(t, resp)
	if resp.StatusCode != http.StatusOK || !strings.Contains(got, "created") {
		t.Fatalf("admin mutation: status=%d body=%s", resp.StatusCode, got)
	}
	if n, _ := f.CountAccounts(t.Context()); n != 3 {
		t.Fatalf("accounts after admin create = %d, want 3", n)
	}
}

func TestCreateAccountDuplicateUsername(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/accounts", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}, "role": {"viewer"}})
	if got := body(t, resp); !strings.Contains(got, "already taken") {
		t.Fatalf("duplicate username not reported; body: %s", got)
	}
}

// --- TOTP ------------------------------------------------------------------

var secretRE = regexp.MustCompile(`<div class="secret">([A-Z2-7]+)</div>`)

func TestTOTPEnableThenRequiredAtLogin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	now := fixedClock()()

	ac := login(t, base, "admin", "hunter2hunter2")

	// Begin enrolment: the page shows a secret and TOTP is not yet enabled.
	resp := postForm(t, ac, base+"/account/totp/enable", nil)
	page := body(t, resp)
	m := secretRE.FindStringSubmatch(page)
	if m == nil {
		t.Fatalf("no secret shown on enrol page; body: %s", page)
	}
	secret := m[1]
	if acct, _ := f.GetAccountByUsername(t.Context(), "admin"); acct.TotpEnabled {
		t.Fatal("TOTP enabled before confirmation")
	}

	// Confirm with the current code: TOTP becomes enabled.
	code, err := auth.TOTPCode(secret, now)
	if err != nil {
		t.Fatal(err)
	}
	resp = postForm(t, ac, base+"/account/totp/confirm", url.Values{"code": {code}})
	if got := body(t, resp); !strings.Contains(got, "enabled") {
		t.Fatalf("confirm did not report enabled; body: %s", got)
	}
	if acct, _ := f.GetAccountByUsername(t.Context(), "admin"); !acct.TotpEnabled {
		t.Fatal("TOTP not enabled after confirmation")
	}

	// A fresh login now requires the code: password alone yields the TOTP
	// step and no session cookie.
	c := newClient(t)
	resp = postForm(t, c, base+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}})
	if got := body(t, resp); !strings.Contains(got, "Enter your code") {
		t.Fatalf("password login did not demand TOTP; body: %s", got)
	}
	if hasCookie(c, base, sessionCookie) {
		t.Fatal("session granted before TOTP step")
	}

	// A wrong code is refused.
	resp = postForm(t, c, base+"/login/totp", url.Values{"code": {"000000"}})
	if got := body(t, resp); !strings.Contains(got, "Incorrect code") {
		t.Fatalf("wrong TOTP code accepted; body: %s", got)
	}
	if hasCookie(c, base, sessionCookie) {
		t.Fatal("session granted on wrong TOTP code")
	}

	// The correct code completes the login.
	code, _ = auth.TOTPCode(secret, now)
	resp = postForm(t, c, base+"/login/totp", url.Values{"code": {code}})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || !hasCookie(c, base, sessionCookie) {
		t.Fatalf("correct TOTP did not complete login: status=%d", resp.StatusCode)
	}
}

func TestTOTPReEnableBlockedWhenEnabled(t *testing.T) {
	f := newFakeStore()
	acct := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	now := fixedClock()()

	// Authenticate (no TOTP yet), then enrol and confirm.
	ac := login(t, base, "admin", "hunter2hunter2")
	page := body(t, postForm(t, ac, base+"/account/totp/enable", nil))
	secret := secretRE.FindStringSubmatch(page)[1]
	code, _ := auth.TOTPCode(secret, now)
	postForm(t, ac, base+"/account/totp/confirm", url.Values{"code": {code}}).Body.Close()

	// A second enable must not re-roll the secret or drop the enabled flag.
	resp := postForm(t, ac, base+"/account/totp/enable", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("re-enable on enrolled account: status=%d, want 303 (no re-roll)", resp.StatusCode)
	}
	got, _ := f.GetAccountByID(t.Context(), acct.ID)
	if !got.TotpEnabled || got.TotpSecret.String != secret {
		t.Fatalf("TOTP was downgraded: enabled=%v secretChanged=%v", got.TotpEnabled, got.TotpSecret.String != secret)
	}
}

func TestPasswordTooLongRejected(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	long := strings.Repeat("x", 73)
	resp := postForm(t, ac, base+"/accounts", url.Values{"username": {"eve"}, "password": {long}, "role": {"viewer"}})
	if got := body(t, resp); !strings.Contains(got, "72 characters or fewer") {
		t.Fatalf("over-long password not rejected clearly; body: %s", got)
	}
	if n, _ := f.CountAccounts(t.Context()); n != 1 {
		t.Fatalf("accounts = %d, want 1 (no account created)", n)
	}
}

// --- no forward-auth -------------------------------------------------------

func TestNoForwardAuthHeaderTrusted(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	req, _ := http.NewRequest(http.MethodGet, base+"/", nil)
	// Headers a trusting reverse proxy would set. None must authenticate.
	req.Header.Set("X-Forwarded-User", "admin")
	req.Header.Set("X-Remote-User", "admin")
	req.Header.Set("X-Forwarded-Email", "admin@example.com")

	c := newClient(t)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("forward-auth header was trusted: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
}
