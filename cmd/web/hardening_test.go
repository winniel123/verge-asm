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

func enableTOTP(t *testing.T, f *fakeStore, id int64) string {
	t.Helper()
	secret, err := auth.NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	// The enable handler encrypts before storage, so the login path's decrypt round-trips (#337).
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

func TestPasswordBruteForceLocksOut(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := newClient(t)
	for i := 0; i < 5; i++ {
		postForm(t, c, base+"/login", url.Values{"username": {"admin"}, "password": {"wrong"}}).Body.Close()
	}

	resp := postForm(t, c, base+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}})
	got := body(t, resp)
	if !strings.Contains(got, "Too many attempts") {
		t.Fatalf("correct password after 5 wrong attempts was not refused; body: %s", got)
	}
	if hasCookie(c, base, sessionCookie) {
		t.Fatal("session granted while the account was locked out")
	}
}

func TestTOTPBruteForceLocksOut(t *testing.T) {
	f := newFakeStore()
	acct := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	enableTOTP(t, f, acct.ID)
	base := start(t, f, "")

	c := newClient(t)
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
	if hasCookie(c, base, pendingCookie) {
		t.Fatal("pending cookie survived the TOTP lockout")
	}

	c2 := newClient(t)
	got := body(t, postForm(t, c2, base+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}}))
	if !strings.Contains(got, "Too many attempts") {
		t.Fatalf("locked account still accepted the password step; body: %s", got)
	}
	if hasCookie(c2, base, sessionCookie) {
		t.Fatal("session granted while the account was locked out")
	}
}

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

	cA := newClient(t)
	postForm(t, cA, base+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}}).Body.Close()
	respA := postForm(t, cA, base+"/login/totp", url.Values{"code": {code}})
	respA.Body.Close()
	if respA.StatusCode != http.StatusSeeOther || !hasCookie(cA, base, sessionCookie) {
		t.Fatalf("first use of a valid code did not complete login: status=%d", respA.StatusCode)
	}
	if got := f.accounts[acct.ID]; !got.TotpLastStep.Valid || got.TotpLastStep.Int64 == 0 {
		t.Fatalf("stored totp_last_step did not advance: %+v", got.TotpLastStep)
	}

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
	totpKey, err := auth.DeriveTOTPKey(testKey)
	if err != nil {
		t.Fatal(err)
	}
	if dec, derr := auth.DecryptTOTPSecret(totpKey, stored); derr != nil || dec != secret {
		t.Fatalf("stored ciphertext did not decrypt to the secret: dec=%q err=%v", dec, derr)
	}

	code, _ := auth.TOTPCode(secret, now)
	if got := body(t, postForm(t, ac, base+"/account/totp/confirm", url.Values{"code": {code}})); !strings.Contains(got, "Recovery codes") {
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

func TestTOTPLegacyCleartextSecretFailsLoud(t *testing.T) {
	f := newFakeStore()
	acct := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	// A pre-#337 install stored the base32 directly; this is that legacy row.
	secret, err := auth.NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	f.acctMu.Lock()
	a := f.accounts[acct.ID]
	a.TotpSecret = pgtype.Text{String: secret, Valid: true}
	a.TotpEnabled = true
	f.accounts[acct.ID] = a
	f.acctMu.Unlock()

	base := start(t, f, "")

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	c := newClient(t)
	postForm(t, c, base+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}}).Body.Close()

	resp := postForm(t, c, base+"/login/totp", url.Values{"code": {"000000"}})
	got := body(t, resp)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("legacy cleartext secret was tolerated: status=%d, want 500; body: %s", resp.StatusCode, got)
	}
	if strings.Contains(got, "Incorrect code") {
		t.Fatalf("decrypt failure was masked as a verification miss; body: %s", got)
	}
	if hasCookie(c, base, sessionCookie) {
		t.Fatal("session granted despite an undecryptable secret")
	}
	if strings.Contains(buf.String(), secret) {
		t.Fatalf("cleartext secret leaked into the log: %s", buf.String())
	}
}

func TestForgotDoesNotLogPlaintextToken(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	postForm(t, newClient(t), base+"/forgot", url.Values{"username": {"admin"}}).Body.Close()

	out := buf.String()
	if strings.Contains(out, "token=") {
		t.Fatalf("plaintext reset token was logged by default; log: %s", out)
	}
	if !strings.Contains(out, "password reset requested for") || !strings.Contains(out, "admin") {
		t.Fatalf("reset request was not recorded by username/id; log: %s", out)
	}
}
