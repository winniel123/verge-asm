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
	srv := newServer(f, testKey, setupToken, fixedClock())
	srv.transcriptKey = testTranscriptKey
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)
	return ts.URL
}

// startAt is start with the server clock pinned to now — the read instant every
// derivation read of the observation corpus is gated against (#237). Two servers
// over one fakeStore at different clocks read the same corpus across the live
// boundary without any delete.
func startAt(t *testing.T, f *fakeStore, now time.Time) string {
	t.Helper()
	srv := newServer(f, testKey, "", func() time.Time { return now })
	srv.transcriptKey = testTranscriptKey
	ts := httptest.NewServer(srv.handler())
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

func TestSetupCreatesFirstAdminThenCloses(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "the-setup-token")

	// The form is served while the window is open.
	resp, err := http.Get(base + "/setup")
	if err != nil {
		t.Fatal(err)
	}
	if got := body(t, resp); !strings.Contains(got, "Create the admin account") {
		t.Fatalf("setup form missing; body: %s", got)
	}

	c := newClient(t)

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
	if got := body(t, resp); !strings.Contains(got, "at least 12") {
		t.Fatalf("short password not rejected; body: %s", got)
	}
	if n, _ := f.CountAccounts(t.Context()); n != 0 {
		t.Fatalf("accounts = %d, want 0", n)
	}
}

func TestLoginAndSession(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// A non-empty estate — one observed Name — so `/` renders the Dashboard rather
	// than the empty-estate first-run checklist (#302).
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	base := start(t, f, "")

	c := login(t, base, "admin", "hunter2hunter2")
	if !hasCookie(c, base, sessionCookie) {
		t.Fatal("no session cookie after login")
	}

	// `/` is now the Dashboard (V2 console map #275, #277): the KPI band and the
	// open-signal register, with the Dashboard nav pill lit.
	resp, err := c.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	dash := body(t, resp)
	if resp.StatusCode != http.StatusOK ||
		!strings.Contains(dash, "Open signals") ||
		!strings.Contains(dash, `class="sh-pill on" href="/"`) {
		t.Fatalf("dashboard not shown at /; status=%d body=%s", resp.StatusCode, dash)
	}

	// The account surface folded into Settings → Team (#281, retargeted #313): /account
	// now redirects there, and member management lives on the Settings Team sub-tab.
	resp, err = c.Get(base + "/account")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/settings?tab=team" {
		t.Fatalf("/account should redirect to Settings team; status=%d loc=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	got := getBody(t, c, base+"/settings?tab=team", http.StatusOK)
	if !strings.Contains(got, "admin") || !strings.Contains(got, "Who can sign in") {
		t.Fatalf("settings team tab missing the member surface; body=%s", got)
	}
}

// The sign-in page renders the ported composition: the credentials card and the
// SSO affordance. This build configures no identity provider, so the affordance
// is the design-system disabled/empty state — never a fabricated provider.
func TestSignInPageComposition(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	resp, err := http.Get(base + "/login")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	for _, want := range []string{"Sign in", "Single sign-on not configured"} {
		if !strings.Contains(got, want) {
			t.Fatalf("login page missing %q; body: %s", want, got)
		}
	}
	// No fabricated identity provider leaks onto the page.
	for _, forbidden := range []string{"Okta", "Continue with"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("login page fabricated an IdP (%q); body: %s", forbidden, got)
		}
	}
	// The "locked out? reset on the host" CLI line moved off login onto the forgot card
	// (SPEC-CHANGE #19): it is no longer on /login, and /forgot carries it.
	if strings.Contains(got, "verge users reset-password") {
		t.Fatalf("login page still carries the host-reset CLI line (it moved to /forgot per #19); body: %s", got)
	}
	if forgot := getAnon(t, base+"/forgot", http.StatusOK); !strings.Contains(forgot, "verge users reset-password") {
		t.Fatalf("forgot card missing the host-reset CLI line; body: %s", forgot)
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
	// markup as plain text — and it must actually tell the viewer why (the ported
	// ErrorPage 403 frame: "Access denied", with how an admin widens it).
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("403 Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(page403, "Access denied") || !strings.Contains(page403, "widen") {
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

	ac := login(t, base, "admin", "hunter2hunter2")
	// The create is a post-redirect-get (ADR-0130 §3, #974): the confirmation line rides
	// the session flash to the landing GET, so the 303 itself carries no body.
	resp = postForm(t, ac, base+"/accounts", url.Values{"username": {"eve"}, "password": {"hunter2hunter2"}, "role": {"viewer"}})
	if got := prgLanding(t, ac, base, resp); !strings.Contains(got, "created") {
		t.Fatalf("admin mutation: landing body=%s", got)
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
	if got := refusalPage(t, ac, base, resp); !strings.Contains(got, "already taken") {
		t.Fatalf("duplicate username not reported; landing body: %s", got)
	}
}

var secretRE = regexp.MustCompile(`<span class="val">([A-Z2-7]+)</span>`)

// TestTOTPEnrollShowsQR covers #317: the enrollment page renders a scannable QR
// of the otpauth:// URI (generated in-process, so the secret never leaves the
// origin) while keeping the secret text as the manual-entry fallback, and the QR
// reappears on the incorrect-code re-render.
func TestTOTPEnrollShowsQR(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := body(t, postForm(t, ac, base+"/account/totp/enable", nil))
	if !strings.Contains(page, "<svg") || !strings.Contains(page, `aria-label="Two-factor enrollment QR code`) {
		t.Fatalf("enroll page has no QR image; body: %s", page)
	}
	if secretRE.FindStringSubmatch(page) == nil {
		t.Fatalf("enroll page dropped the secret text fallback; body: %s", page)
	}
	// The QR must be self-contained: no reference to any external host (an
	// external QR service would leak the secret — ADR-0053).
	if strings.Contains(page, "//api.qrserver") || strings.Contains(page, "chart.googleapis") {
		t.Fatalf("enroll page references an external QR service; body: %s", page)
	}

	// The incorrect-code re-render keeps the QR and reports the error.
	reRender := body(t, postForm(t, ac, base+"/account/totp/confirm", url.Values{"code": {"000000"}}))
	if !strings.Contains(reRender, "<svg") || !strings.Contains(reRender, "Incorrect code") {
		t.Fatalf("incorrect-code re-render missing QR or error; body: %s", reRender)
	}
}

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
	if got := body(t, resp); !strings.Contains(got, "Recovery codes") {
		t.Fatalf("confirm did not reach the recovery-codes screen; body: %s", got)
	}
	if acct, _ := f.GetAccountByUsername(t.Context(), "admin"); !acct.TotpEnabled {
		t.Fatal("TOTP not enabled after confirmation")
	}

	// A fresh login now requires the code: password alone yields the TOTP
	// step and no session cookie.
	c := newClient(t)
	resp = postForm(t, c, base+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}})
	if got := body(t, resp); !strings.Contains(got, "Two-factor check") {
		t.Fatalf("password login did not demand TOTP; body: %s", got)
	}
	if hasCookie(c, base, sessionCookie) {
		t.Fatal("session granted before TOTP step")
	}

	resp = postForm(t, c, base+"/login/totp", url.Values{"code": {"000000"}})
	if got := body(t, resp); !strings.Contains(got, "Incorrect code") {
		t.Fatalf("wrong TOTP code accepted; body: %s", got)
	}
	if hasCookie(c, base, sessionCookie) {
		t.Fatal("session granted on wrong TOTP code")
	}

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

	// Capture the stored (encrypted, #337) secret right after enrolment; a re-enable
	// must leave this exact ciphertext untouched — comparing the stored value to the
	// cleartext base32 would always differ now, so compare stored-to-stored instead.
	before, _ := f.GetAccountByID(t.Context(), acct.ID)

	// A second enable must not re-roll the secret or drop the enabled flag.
	resp := postForm(t, ac, base+"/account/totp/enable", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("re-enable on enrolled account: status=%d, want 303 (no re-roll)", resp.StatusCode)
	}
	got, _ := f.GetAccountByID(t.Context(), acct.ID)
	if !got.TotpEnabled || got.TotpSecret.String != before.TotpSecret.String {
		t.Fatalf("TOTP was downgraded: enabled=%v secretChanged=%v", got.TotpEnabled, got.TotpSecret.String != before.TotpSecret.String)
	}
}

func TestPasswordTooLongRejected(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	long := strings.Repeat("x", 73)
	resp := postForm(t, ac, base+"/accounts", url.Values{"username": {"eve"}, "password": {long}, "role": {"viewer"}})
	if got := refusalPage(t, ac, base, resp); !strings.Contains(got, "72 characters or fewer") {
		t.Fatalf("over-long password not rejected clearly; landing body: %s", got)
	}
	if n, _ := f.CountAccounts(t.Context()); n != 1 {
		t.Fatalf("accounts = %d, want 1 (no account created)", n)
	}
}

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
