package main

import (
	"bytes"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/auth"
)

// enableTOTP arms a real TOTP secret on a seeded account directly in the fake and
// marks it enabled, so the throttle and replay tests can drive /login/totp without
// walking the whole enrollment flow. It returns the secret so a test can render the
// live code for it.
func enableTOTP(t *testing.T, f *fakeStore, id int64) string {
	t.Helper()
	secret, err := auth.NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	// Store the secret exactly as the enable handler does (#337): encrypted at rest
	// with the AEAD sub-key derived from the same test session key, so the login path's
	// decrypt round-trips. The cleartext base32 is returned for rendering live codes.
	totpKey, err := auth.DeriveTOTPKey(testKey)
	if err != nil {
		t.Fatal(err)
	}
	enc, err := auth.EncryptTOTPSecret(totpKey, secret)
	if err != nil {
		t.Fatal(err)
	}
	f.acctMu.Lock()
	acct := f.accounts[id]
	acct.TotpSecret = pgtype.Text{String: enc, Valid: true}
	acct.TotpEnabled = true
	f.accounts[id] = acct
	f.acctMu.Unlock()
	return secret
}

// --- #322: rate limit / lockout on the credential endpoints -----------------

// TestPasswordBruteForceLocksOut is the password half of #322: repeated wrong
// passwords lock the account, and once locked even the correct password is refused
// and grants no session.
func TestPasswordBruteForceLocksOut(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := newClient(t)
	for i := 0; i < 5; i++ {
		postForm(t, c, base+"/login", url.Values{"username": {"admin"}, "password": {"wrong"}}).Body.Close()
	}

	// The (N+1)th attempt is refused regardless of correctness: the CORRECT password
	// now yields the lockout, not a session.
	resp := postForm(t, c, base+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}})
	got := body(t, resp)
	if !strings.Contains(got, "Too many attempts") {
		t.Fatalf("correct password after 5 wrong attempts was not refused; body: %s", got)
	}
	if hasCookie(c, base, sessionCookie) {
		t.Fatal("session granted while the account was locked out")
	}
}

// TestTOTPBruteForceLocksOut is the TOTP half of #322: a 6-digit code is otherwise
// unboundedly guessable, so repeated wrong codes must lock the account and, at the
// threshold, invalidate the pending cookie so the attacker cannot keep hammering
// the same grant.
func TestTOTPBruteForceLocksOut(t *testing.T) {
	f := newFakeStore()
	acct := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	enableTOTP(t, f, acct.ID)
	base := start(t, f, "")

	c := newClient(t)
	// Password step mints the pending cookie.
	postForm(t, c, base+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}}).Body.Close()
	if !hasCookie(c, base, pendingCookie) {
		t.Fatal("no pending cookie after the password step")
	}

	var last string
	for i := 0; i < 5; i++ {
		last = body(t, postForm(t, c, base+"/login/totp", url.Values{"code": {"000000"}}))
	}
	if !strings.Contains(last, "Too many attempts") {
		t.Fatalf("TOTP not locked after 5 wrong codes; body: %s", last)
	}
	// The pending cookie is cleared at the threshold, so the half-finished flow
	// cannot be re-presented.
	if hasCookie(c, base, pendingCookie) {
		t.Fatal("pending cookie survived the TOTP lockout")
	}

	// The account is now locked: a fresh, fully-correct sign-in is refused at the
	// password step and grants nothing.
	c2 := newClient(t)
	got := body(t, postForm(t, c2, base+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}}))
	if !strings.Contains(got, "Too many attempts") {
		t.Fatalf("locked account still accepted the password step; body: %s", got)
	}
	if hasCookie(c2, base, sessionCookie) {
		t.Fatal("session granted while the account was locked out")
	}
}

// --- #323: TOTP codes are single-use within their validity window -----------

// TestTOTPCodeNotReplayable is the #323 guarantee: a valid code completes login
// once, advances the stored step, and the same code at the same clock is refused on
// a second, independent sign-in — no replay within the ~90s window.
func TestTOTPCodeNotReplayable(t *testing.T) {
	f := newFakeStore()
	acct := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	secret := enableTOTP(t, f, acct.ID)
	base := start(t, f, "")
	now := fixedClock()()

	code, err := auth.TOTPCode(secret, now)
	if err != nil {
		t.Fatal(err)
	}

	// First sign-in: the code completes login.
	cA := newClient(t)
	postForm(t, cA, base+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}}).Body.Close()
	respA := postForm(t, cA, base+"/login/totp", url.Values{"code": {code}})
	respA.Body.Close()
	if respA.StatusCode != http.StatusSeeOther || !hasCookie(cA, base, sessionCookie) {
		t.Fatalf("first use of a valid code did not complete login: status=%d", respA.StatusCode)
	}
	// The replay watermark advanced.
	if got := f.accounts[acct.ID]; !got.TotpLastStep.Valid || got.TotpLastStep.Int64 == 0 {
		t.Fatalf("stored totp_last_step did not advance: %+v", got.TotpLastStep)
	}

	// Second, independent sign-in reusing the SAME code at the SAME clock: refused.
	cB := newClient(t)
	postForm(t, cB, base+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}}).Body.Close()
	got := body(t, postForm(t, cB, base+"/login/totp", url.Values{"code": {code}}))
	if hasCookie(cB, base, sessionCookie) {
		t.Fatal("a replayed TOTP code granted a session")
	}
	if !strings.Contains(got, "Incorrect code") {
		t.Fatalf("replayed code was not refused as incorrect; body: %s", got)
	}
}

// --- #339: TOTP single-use is atomic under concurrency ----------------------

// TestTOTPCodeSingleUseUnderConcurrency is the #339 guarantee: the single-use spend
// is an atomic conditional UPDATE, not a read-then-write, so two independent logins
// presenting the SAME valid code at the SAME instant cannot both complete — at most
// one receives a session cookie. It complements the sequential TestTOTPCodeNotReplayable.
func TestTOTPCodeSingleUseUnderConcurrency(t *testing.T) {
	f := newFakeStore()
	acct := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	secret := enableTOTP(t, f, acct.ID)
	base := start(t, f, "")
	now := fixedClock()()

	code, err := auth.TOTPCode(secret, now)
	if err != nil {
		t.Fatal(err)
	}

	// Each client completes the password step first to hold its own pending cookie.
	const n = 2
	clients := make([]*http.Client, n)
	for i := range clients {
		c := newClient(t)
		postForm(t, c, base+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}}).Body.Close()
		if !hasCookie(c, base, pendingCookie) {
			t.Fatalf("client %d: no pending cookie after the password step", i)
		}
		clients[i] = c
	}

	// Fire both TOTP steps with the same code, released together.
	var wg sync.WaitGroup
	var mu sync.Mutex
	var reqErrs []error
	gate := make(chan struct{})
	for _, c := range clients {
		wg.Add(1)
		go func(c *http.Client) {
			defer wg.Done()
			<-gate
			resp, err := c.PostForm(base+"/login/totp", url.Values{"code": {code}})
			if err != nil {
				mu.Lock()
				reqErrs = append(reqErrs, err)
				mu.Unlock()
				return
			}
			resp.Body.Close()
		}(c)
	}
	close(gate)
	wg.Wait()
	if len(reqErrs) > 0 {
		t.Fatalf("concurrent /login/totp request error: %v", reqErrs[0])
	}

	granted := 0
	for _, c := range clients {
		if hasCookie(c, base, sessionCookie) {
			granted++
		}
	}
	if granted != 1 {
		t.Fatalf("same valid code granted %d sessions concurrently, want exactly 1", granted)
	}
}

// --- #337: totp_secret is encrypted at rest ---------------------------------

// TestTOTPSecretEncryptedAtRest is the #337 guarantee: the enrollment secret is
// encrypted before it reaches storage (ADR-0053, no cleartext secrets in Postgres),
// so the raw stored column neither equals nor contains the cleartext base32 — yet
// verification still succeeds through the encrypt/decrypt round-trip on both the
// confirm and the login read paths.
func TestTOTPSecretEncryptedAtRest(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	now := fixedClock()()

	ac := login(t, base, "admin", "hunter2hunter2")
	page := body(t, postForm(t, ac, base+"/account/totp/enable", nil))
	m := secretRE.FindStringSubmatch(page)
	if m == nil {
		t.Fatalf("no secret shown on enrol page; body: %s", page)
	}
	secret := m[1]

	// The raw stored column is ciphertext: it must not equal or contain the base32.
	rec, err := f.GetAccountByID(t.Context(), admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored := rec.TotpSecret.String
	if stored == "" {
		t.Fatal("no totp secret was stored")
	}
	if stored == secret || strings.Contains(stored, secret) {
		t.Fatalf("totp secret stored in cleartext: stored=%q contains/equals secret=%q", stored, secret)
	}
	// And it decrypts back to the exact secret under the file-backed AEAD sub-key.
	totpKey, err := auth.DeriveTOTPKey(testKey)
	if err != nil {
		t.Fatal(err)
	}
	if dec, derr := auth.DecryptTOTPSecret(totpKey, stored); derr != nil || dec != secret {
		t.Fatalf("stored ciphertext did not decrypt to the secret: dec=%q err=%v", dec, derr)
	}

	// Verification round-trips: confirm enables 2FA, and a fresh login accepts the code
	// via the decrypt read path.
	code, _ := auth.TOTPCode(secret, now)
	if got := body(t, postForm(t, ac, base+"/account/totp/confirm", url.Values{"code": {code}})); !strings.Contains(got, "enabled") {
		t.Fatalf("confirm did not enable via the encrypt/decrypt round-trip; body: %s", got)
	}
	c := newClient(t)
	postForm(t, c, base+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}}).Body.Close()
	code2, _ := auth.TOTPCode(secret, now)
	resp := postForm(t, c, base+"/login/totp", url.Values{"code": {code2}})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || !hasCookie(c, base, sessionCookie) {
		t.Fatalf("login via decrypted secret did not complete: status=%d", resp.StatusCode)
	}
}

// --- #328: password-reset plaintext token is not logged by default ----------

// TestForgotDoesNotLogPlaintextToken covers #328 (CWE-532): a reset request logs
// the account and the reset-record id, but never the bearer token, by default.
func TestForgotDoesNotLogPlaintextToken(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	postForm(t, newClient(t), base+"/forgot", url.Values{"username": {"admin"}}).Body.Close()

	out := buf.String()
	// The bearer token — the whole point of the leak — must not appear.
	if strings.Contains(out, "token=") {
		t.Fatalf("plaintext reset token was logged by default; log: %s", out)
	}
	// But the request is still recorded, by username and reset id, so an operator can
	// still act on it out of band.
	if !strings.Contains(out, "password reset requested for") || !strings.Contains(out, "admin") {
		t.Fatalf("reset request was not recorded by username/id; log: %s", out)
	}
}
