package main

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
)

// --- fake pre-auth token stores (#314, T19) ---------------------------------

func (f *fakeStore) CreatePasswordReset(_ context.Context, arg db.CreatePasswordResetParams) (db.PasswordReset, error) {
	pr := db.PasswordReset{
		ID: f.resetNextID, AccountID: arg.AccountID, TokenHash: arg.TokenHash,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true}, ExpiresAt: arg.ExpiresAt,
	}
	f.passwordResets = append(f.passwordResets, pr)
	f.resetNextID++
	return pr, nil
}

func (f *fakeStore) GetPasswordResetByHash(_ context.Context, tokenHash string) (db.PasswordReset, error) {
	for _, pr := range f.passwordResets {
		if pr.TokenHash == tokenHash {
			return pr, nil
		}
	}
	return db.PasswordReset{}, pgx.ErrNoRows
}

func (f *fakeStore) ConsumePasswordReset(_ context.Context, arg db.ConsumePasswordResetParams) error {
	for i := range f.passwordResets {
		if f.passwordResets[i].ID == arg.ID {
			f.passwordResets[i].ConsumedAt = arg.ConsumedAt
			return nil
		}
	}
	return pgx.ErrNoRows
}

func (f *fakeStore) CreateRecoveryCode(_ context.Context, arg db.CreateRecoveryCodeParams) error {
	for _, rc := range f.recoveryCodes {
		if rc.AccountID == arg.AccountID && rc.CodeHash == arg.CodeHash {
			return &pgconn.PgError{Code: "23505", Message: "duplicate recovery code"}
		}
	}
	f.recoveryCodes = append(f.recoveryCodes, db.RecoveryCode{
		ID: f.recoveryNextID, AccountID: arg.AccountID, CodeHash: arg.CodeHash,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	f.recoveryNextID++
	return nil
}

func (f *fakeStore) DeleteRecoveryCodesForAccount(_ context.Context, accountID int64) error {
	kept := f.recoveryCodes[:0:0]
	for _, rc := range f.recoveryCodes {
		if rc.AccountID != accountID {
			kept = append(kept, rc)
		}
	}
	f.recoveryCodes = kept
	return nil
}

func (f *fakeStore) ListUnusedRecoveryCodeHashes(_ context.Context, accountID int64) ([]db.ListUnusedRecoveryCodeHashesRow, error) {
	rows := []db.ListUnusedRecoveryCodeHashesRow{}
	for _, rc := range f.recoveryCodes {
		if rc.AccountID == accountID && !rc.UsedAt.Valid {
			rows = append(rows, db.ListUnusedRecoveryCodeHashesRow{ID: rc.ID, CodeHash: rc.CodeHash})
		}
	}
	return rows, nil
}

func (f *fakeStore) ConsumeRecoveryCode(_ context.Context, arg db.ConsumeRecoveryCodeParams) error {
	for i := range f.recoveryCodes {
		if f.recoveryCodes[i].ID == arg.ID {
			f.recoveryCodes[i].UsedAt = arg.UsedAt
			return nil
		}
	}
	return pgx.ErrNoRows
}

func (f *fakeStore) GetInviteByTokenHash(_ context.Context, tokenHash string) (db.Invite, error) {
	for _, inv := range f.invites {
		if inv.TokenHash == tokenHash {
			return inv, nil
		}
	}
	return db.Invite{}, pgx.ErrNoRows
}

func (f *fakeStore) ConsumeInvite(_ context.Context, arg db.ConsumeInviteParams) error {
	for i := range f.invites {
		if f.invites[i].ID == arg.ID {
			f.invites[i].ConsumedAt = arg.ConsumedAt
			f.invites[i].AcceptedAccountID = arg.AcceptedAccountID
			return nil
		}
	}
	return pgx.ErrNoRows
}

// --- test seeding helpers ---------------------------------------------------

// serverClock is the instant start()'s fixed clock is pinned to; expiry seeding is
// relative to it so a test can mint a live or a deliberately stale grant.
var serverClock = time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

// addReset seeds a password-reset row whose token hashes from a known plaintext, so
// a test can drive /reset without scraping the delivered link out of the logs.
func addReset(t *testing.T, f *fakeStore, acctID int64, plaintext string, expires time.Time) {
	t.Helper()
	if _, err := f.CreatePasswordReset(t.Context(), db.CreatePasswordResetParams{
		AccountID: acctID, TokenHash: hashToken(plaintext),
		ExpiresAt: pgtype.Timestamptz{Time: expires, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
}

// addInvite seeds an invite row at a role whose token hashes from a known plaintext.
// The creation side lands in T18; the acceptance tests seed the store directly.
func addInvite(t *testing.T, f *fakeStore, role, plaintext string, expires time.Time) {
	t.Helper()
	f.invites = append(f.invites, db.Invite{
		ID: f.inviteNextID, TokenHash: hashToken(plaintext), Role: role,
		CreatedAt: pgtype.Timestamptz{Time: serverClock, Valid: true},
		ExpiresAt: pgtype.Timestamptz{Time: expires, Valid: true},
	})
	f.inviteNextID++
}

// --- forgot / reset ---------------------------------------------------------

// The sign-in card links to the forgot flow, and the SSO affordance stays the
// design-system not-configured state — no provider is fabricated (#293 is absent).
func TestSignInLinksForgotAndSSONotConfigured(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	got := getAnon(t, base+"/login", http.StatusOK)
	for _, want := range []string{`href="/forgot"`, "Forgot password?", "Single sign-on not configured"} {
		if !strings.Contains(got, want) {
			t.Fatalf("login page missing %q; body: %s", want, got)
		}
	}
	for _, forbidden := range []string{"Okta", "Continue with"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("login page fabricated an IdP (%q)", forbidden)
		}
	}
}

// Forgot is non-enumerating: a known and an unknown username get the identical done
// state, and only the known one mints a (hashed) reset grant.
func TestForgotIsNonEnumerating(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	c := newClient(t)

	known := body(t, postForm(t, c, base+"/forgot", url.Values{"username": {"ola"}}))
	unknown := body(t, postForm(t, c, base+"/forgot", url.Values{"username": {"ghost"}}))
	if known != unknown {
		t.Fatalf("forgot response differs by account existence (enumeration leak)\nknown:   %s\nunknown: %s", known, unknown)
	}
	if !strings.Contains(known, "Check for your link") {
		t.Fatalf("forgot done state missing; body: %s", known)
	}
	if len(f.passwordResets) != 1 {
		t.Fatalf("reset grants minted = %d, want 1 (only the real account)", len(f.passwordResets))
	}
	// Only the hash is kept — never the plaintext.
	if f.passwordResets[0].TokenHash == "" {
		t.Fatal("reset grant stored no hash")
	}
}

// A valid link sets the password and is single-use; a stale link is refused.
func TestResetFlowSetsPasswordOnceAndExpires(t *testing.T) {
	f := newFakeStore()
	acct := seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	addReset(t, f, acct.ID, "live-token", serverClock.Add(time.Hour))

	// The form renders for a valid token.
	if got := getAnon(t, base+"/reset?token=live-token", http.StatusOK); !strings.Contains(got, "Set a new password") {
		t.Fatalf("reset form missing; body: %s", got)
	}

	c := newClient(t)
	resp := postForm(t, c, base+"/reset", url.Values{
		"token": {"live-token"}, "password": {"brand-new-pass"}, "confirm": {"brand-new-pass"},
	})
	if got := body(t, resp); !strings.Contains(got, "Password updated") {
		t.Fatalf("reset did not report success; body: %s", got)
	}
	// The new password now authenticates and the old one does not.
	login(t, base, "ola", "brand-new-pass")
	resp = postForm(t, newClient(t), base+"/login", url.Values{"username": {"ola"}, "password": {"hunter2hunter2"}})
	if got := body(t, resp); !strings.Contains(got, "Invalid username or password") {
		t.Fatalf("old password still works after reset; body: %s", got)
	}

	// Single-use: the same token is now refused.
	if got := getAnon(t, base+"/reset?token=live-token", http.StatusOK); !strings.Contains(got, "expired or already used") {
		t.Fatalf("spent reset token not refused; body: %s", got)
	}

	// A stale token is refused too.
	addReset(t, f, acct.ID, "stale-token", serverClock.Add(-time.Hour))
	if got := getAnon(t, base+"/reset?token=stale-token", http.StatusOK); !strings.Contains(got, "expired or already used") {
		t.Fatalf("expired reset token not refused; body: %s", got)
	}
}

// reset-done does not claim a global sign-out — this build's sessions are stateless
// cookies with no registry, so it says a session elsewhere lapses when it expires.
func TestResetDoneDoesNotClaimGlobalSignOut(t *testing.T) {
	f := newFakeStore()
	acct := seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	addReset(t, f, acct.ID, "tok", serverClock.Add(time.Hour))
	got := body(t, postForm(t, newClient(t), base+"/reset", url.Values{
		"token": {"tok"}, "password": {"brand-new-pass"}, "confirm": {"brand-new-pass"},
	}))
	if strings.Contains(strings.ToLower(got), "every other session") || strings.Contains(strings.ToLower(got), "signed out") {
		t.Fatalf("reset-done fabricates a global sign-out this build cannot do; body: %s", got)
	}
}

// --- TOTP enrollment + recovery codes ---------------------------------------

// recoveryCodeRE matches a recovery code only inside its reveal span, so it counts
// the shown codes rather than any incidental hex or class text elsewhere on the page.
var recoveryCodeRE = regexp.MustCompile(`class="mono cvcode"[^>]*>([a-z2-9]{4}-[a-z2-9]{4})<`)

// Confirming TOTP enrollment reveals the recovery codes once, stores only their
// hashes, and one of them redeems the login two-factor step exactly once.
func TestTOTPEnrollmentRevealsRecoveryCodesOnce(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	now := serverClock

	ac := login(t, base, "admin", "hunter2hunter2")
	page := body(t, postForm(t, ac, base+"/account/totp/enable", nil))
	secret := secretRE.FindStringSubmatch(page)[1]
	code, _ := auth.TOTPCode(secret, now)

	confirmPage := body(t, postForm(t, ac, base+"/account/totp/confirm", url.Values{"code": {code}}))
	if !strings.Contains(confirmPage, "Two-factor enabled") || !strings.Contains(confirmPage, "Shown once") {
		t.Fatalf("recovery-codes screen missing; body: %s", confirmPage)
	}
	var codes []string
	for _, m := range recoveryCodeRE.FindAllStringSubmatch(confirmPage, -1) {
		codes = append(codes, m[1])
	}
	if len(codes) != recoveryCodeCount {
		t.Fatalf("revealed %d recovery codes, want %d; body: %s", len(codes), recoveryCodeCount, confirmPage)
	}
	if len(f.recoveryCodes) != recoveryCodeCount {
		t.Fatalf("stored %d recovery codes, want %d", len(f.recoveryCodes), recoveryCodeCount)
	}
	// Reveal-once: only hashes are kept — no stored row equals a shown plaintext.
	for _, rc := range f.recoveryCodes {
		for _, shown := range codes {
			if rc.CodeHash == shown {
				t.Fatal("recovery code stored in plaintext (reveal-once violated)")
			}
		}
		if len(rc.CodeHash) != 64 {
			t.Fatalf("recovery code hash is not a sha-256 hex digest: %q", rc.CodeHash)
		}
	}

	// A recovery code redeems the login two-factor step once.
	c := newClient(t)
	if got := body(t, postForm(t, c, base+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}})); !strings.Contains(got, "Two-factor check") {
		t.Fatalf("password login did not demand a second factor; body: %s", got)
	}
	resp := postForm(t, c, base+"/login/totp", url.Values{"code": {codes[0]}})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || !hasCookie(c, base, sessionCookie) {
		t.Fatalf("recovery code did not complete login: status=%d", resp.StatusCode)
	}

	// Single-use: the same code is now refused.
	c2 := newClient(t)
	postForm(t, c2, base+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}}).Body.Close()
	if got := body(t, postForm(t, c2, base+"/login/totp", url.Values{"code": {codes[0]}})); !strings.Contains(got, "Incorrect code") {
		t.Fatalf("spent recovery code was accepted a second time; body: %s", got)
	}
}

// --- invite acceptance ------------------------------------------------------

// A valid invite → set-credentials screen creates the account at the invited role,
// is single-use, and grants no session (the acceptor signs in afterwards).
func TestInviteAcceptanceSetsCredentials(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	addInvite(t, f, roleViewer, "invite-token", serverClock.Add(24*time.Hour))

	// The set-credentials screen names the role.
	if got := getAnon(t, base+"/invite?token=invite-token", http.StatusOK); !strings.Contains(got, "Accept your invitation") || !strings.Contains(got, "viewer") {
		t.Fatalf("invite screen missing role/heading; body: %s", got)
	}

	c := newClient(t)
	resp := postForm(t, c, base+"/invite", url.Values{
		"token": {"invite-token"}, "username": {"newbie"}, "password": {"hunter2hunter2"},
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login?invited=1" {
		t.Fatalf("invite accept: status=%d loc=%q, want 303 to /login?invited=1", resp.StatusCode, resp.Header.Get("Location"))
	}
	if hasCookie(c, base, sessionCookie) {
		t.Fatal("invite acceptance granted a session (should require sign-in)")
	}
	// The account exists at the invited role and can sign in with the set credentials.
	acct, err := f.GetAccountByUsername(t.Context(), "newbie")
	if err != nil || acct.Role != roleViewer {
		t.Fatalf("invited account not created at viewer role: %+v err=%v", acct, err)
	}
	login(t, base, "newbie", "hunter2hunter2")

	// Single-use: the token no longer accepts.
	if got := getAnon(t, base+"/invite?token=invite-token", http.StatusOK); !strings.Contains(got, "expired or already used") {
		t.Fatalf("spent invite token not refused; body: %s", got)
	}
}

// An expired or unknown invite token renders the honest invalid state, never a form.
func TestInviteInvalidToken(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	addInvite(t, f, roleViewer, "stale", serverClock.Add(-time.Hour))

	if got := getAnon(t, base+"/invite?token=stale", http.StatusOK); !strings.Contains(got, "expired or already used") {
		t.Fatalf("expired invite not refused; body: %s", got)
	}
	if got := getAnon(t, base+"/invite?token=nope", http.StatusOK); !strings.Contains(got, "expired or already used") {
		t.Fatalf("unknown invite not refused; body: %s", got)
	}
	// A stale token cannot create an account.
	postForm(t, newClient(t), base+"/invite", url.Values{
		"token": {"stale"}, "username": {"eve"}, "password": {"hunter2hunter2"},
	}).Body.Close()
	if n, _ := f.CountAccounts(t.Context()); n != 0 {
		t.Fatalf("accounts after invalid invite = %d, want 0", n)
	}
}

// getAnon GETs a URL with no session and asserts the status, returning the body.
func getAnon(t *testing.T, url string, want int) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != want {
		t.Fatalf("GET %s: status=%d, want %d", url, resp.StatusCode, want)
	}
	return body(t, resp)
}
