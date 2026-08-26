package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
)

// --- fake personal-token + password store ----------------------------------

func (f *fakeStore) UpdatePassword(_ context.Context, arg db.UpdatePasswordParams) error {
	acct, ok := f.accounts[arg.ID]
	if !ok {
		return pgx.ErrNoRows
	}
	acct.PasswordHash = arg.PasswordHash
	f.accounts[arg.ID] = acct
	return nil
}

func (f *fakeStore) CreatePersonalToken(_ context.Context, arg db.CreatePersonalTokenParams) (db.PersonalToken, error) {
	for _, t := range f.personalTokens {
		if t.AccountID == arg.AccountID && t.Name == arg.Name {
			return db.PersonalToken{}, &pgconn.PgError{Code: "23505", Message: "duplicate token name"}
		}
	}
	t := db.PersonalToken{
		ID: f.tokenNextID, AccountID: arg.AccountID, Name: arg.Name,
		Prefix: arg.Prefix, TokenHash: arg.TokenHash,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	f.personalTokens = append(f.personalTokens, t)
	f.tokenNextID++
	return t, nil
}

func (f *fakeStore) ListPersonalTokens(_ context.Context, accountID int64) ([]db.ListPersonalTokensRow, error) {
	rows := []db.ListPersonalTokensRow{}
	// Newest first, mirroring ORDER BY created_at DESC, id DESC.
	for i := len(f.personalTokens) - 1; i >= 0; i-- {
		t := f.personalTokens[i]
		if t.AccountID != accountID {
			continue
		}
		rows = append(rows, db.ListPersonalTokensRow{
			ID: t.ID, AccountID: t.AccountID, Name: t.Name, Prefix: t.Prefix,
			CreatedAt: t.CreatedAt, LastUsedAt: t.LastUsedAt,
		})
	}
	return rows, nil
}

func (f *fakeStore) DeletePersonalToken(_ context.Context, arg db.DeletePersonalTokenParams) error {
	kept := make([]db.PersonalToken, 0, len(f.personalTokens))
	for _, t := range f.personalTokens {
		if t.ID == arg.ID && t.AccountID == arg.AccountID {
			continue
		}
		kept = append(kept, t)
	}
	f.personalTokens = kept
	return nil
}

// GetPersonalTokenByHash mirrors the by-hash lookup the /api/v1 bearer path uses (#390,
// A2): the row whose token_hash equals the presented digest, or no row. No row is the
// same pgx.ErrNoRows the generated query returns, which the middleware renders as 401.
func (f *fakeStore) GetPersonalTokenByHash(_ context.Context, tokenHash string) (db.PersonalToken, error) {
	for _, t := range f.personalTokens {
		if t.TokenHash == tokenHash {
			return t, nil
		}
	}
	return db.PersonalToken{}, pgx.ErrNoRows
}

// UpdatePersonalTokenLastUsed mirrors the coarsened touch SQL exactly (#390, A2): stamp
// last_used_at only when it is null or older than an hour, so a busy token is not one
// write per request and the timestamp never regresses. A missing id is a no-op, matching
// the WHERE clause. It uses wall-clock time.Now() as the SQL's now() does, so a test's
// two back-to-back requests fall inside the one-hour window and the second is a no-op.
func (f *fakeStore) UpdatePersonalTokenLastUsed(_ context.Context, id int64) error {
	now := time.Now()
	for i := range f.personalTokens {
		if f.personalTokens[i].ID != id {
			continue
		}
		lu := f.personalTokens[i].LastUsedAt
		if !lu.Valid || lu.Time.Before(now.Add(-time.Hour)) {
			f.personalTokens[i].LastUsedAt = pgtype.Timestamptz{Time: now, Valid: true}
		}
		return nil
	}
	return nil
}

// --- session registry fakes (#405, ADR-0117) -------------------------------

// CreateSession opens a session row: a monotonic id, now-stamped created_at and
// last_seen_at, and the caller's token hash / user-agent / ip / expiry.
func (f *fakeStore) CreateSession(_ context.Context, arg db.CreateSessionParams) (db.Session, error) {
	if f.sessionNextID == 0 {
		f.sessionNextID = 1
	}
	now := time.Now()
	sess := db.Session{
		ID: f.sessionNextID, AccountID: arg.AccountID, TokenHash: arg.TokenHash,
		CreatedAt:  pgtype.Timestamptz{Time: now, Valid: true},
		LastSeenAt: pgtype.Timestamptz{Time: now, Valid: true},
		UserAgent:  arg.UserAgent, Ip: arg.Ip, ExpiresAt: arg.ExpiresAt,
	}
	f.sessions = append(f.sessions, sess)
	f.sessionNextID++
	return sess, nil
}

// GetSessionByTokenHash mirrors the validation query: the row whose token_hash matches
// AND is unrevoked AND is unexpired against the passed clock bound (expires_at > arg).
// A dead session simply returns no row, exactly as the SQL does.
func (f *fakeStore) GetSessionByTokenHash(_ context.Context, arg db.GetSessionByTokenHashParams) (db.Session, error) {
	for _, sess := range f.sessions {
		if sess.TokenHash == arg.TokenHash && !sess.RevokedAt.Valid && sess.ExpiresAt.Time.After(arg.ExpiresAt.Time) {
			return sess, nil
		}
	}
	return db.Session{}, pgx.ErrNoRows
}

// TouchSession refreshes last_seen_at; a missing row is a no-op, matching the SQL.
func (f *fakeStore) TouchSession(_ context.Context, arg db.TouchSessionParams) error {
	for i := range f.sessions {
		if f.sessions[i].ID == arg.ID {
			f.sessions[i].LastSeenAt = arg.LastSeenAt
			return nil
		}
	}
	return nil
}

// RevokeSession stamps revoked_at on the row scoped to its owner (id AND account_id),
// only while it is still live. Idempotent: an already-revoked or foreign row is
// untouched, mirroring the owner-scoped SQL.
func (f *fakeStore) RevokeSession(_ context.Context, arg db.RevokeSessionParams) error {
	for i := range f.sessions {
		if f.sessions[i].ID == arg.ID && f.sessions[i].AccountID == arg.AccountID && !f.sessions[i].RevokedAt.Valid {
			f.sessions[i].RevokedAt = arg.RevokedAt
			return nil
		}
	}
	return nil
}

// ListSessionsForAccount mirrors the personal-listing query: one account's live sessions
// (unrevoked, unexpired against the passed clock), newest activity first, with token_hash
// omitted from the projection so the secret never reaches the render path.
func (f *fakeStore) ListSessionsForAccount(_ context.Context, arg db.ListSessionsForAccountParams) ([]db.ListSessionsForAccountRow, error) {
	rows := []db.ListSessionsForAccountRow{}
	for _, sess := range f.sessions {
		if sess.AccountID != arg.AccountID || sess.RevokedAt.Valid || !sess.ExpiresAt.Time.After(arg.ExpiresAt.Time) {
			continue
		}
		rows = append(rows, db.ListSessionsForAccountRow{
			ID: sess.ID, AccountID: sess.AccountID, CreatedAt: sess.CreatedAt,
			LastSeenAt: sess.LastSeenAt, UserAgent: sess.UserAgent, Ip: sess.Ip,
			ExpiresAt: sess.ExpiresAt,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if !rows[i].LastSeenAt.Time.Equal(rows[j].LastSeenAt.Time) {
			return rows[i].LastSeenAt.Time.After(rows[j].LastSeenAt.Time)
		}
		return rows[i].ID > rows[j].ID
	})
	return rows, nil
}

// RevokeOtherSessionsForAccount revokes every live session for the account EXCEPT the
// current one (arg.ID) — "sign out other devices" and the password-change invalidation.
// The acting session survives, mirroring the id <> $2 predicate.
func (f *fakeStore) RevokeOtherSessionsForAccount(_ context.Context, arg db.RevokeOtherSessionsForAccountParams) error {
	for i := range f.sessions {
		if f.sessions[i].AccountID == arg.AccountID && f.sessions[i].ID != arg.ID && !f.sessions[i].RevokedAt.Valid {
			f.sessions[i].RevokedAt = arg.RevokedAt
		}
	}
	return nil
}

// RevokeAllSessionsForAccount revokes every live session for the account with no
// exception — the reset path (no current session to keep) and admin offboarding.
func (f *fakeStore) RevokeAllSessionsForAccount(_ context.Context, arg db.RevokeAllSessionsForAccountParams) error {
	for i := range f.sessions {
		if f.sessions[i].AccountID == arg.AccountID && !f.sessions[i].RevokedAt.Valid {
			f.sessions[i].RevokedAt = arg.RevokedAt
		}
	}
	return nil
}

// ListAllActiveSessions mirrors the admin query: every account's live sessions joined to
// the owning account's username and role, ordered by username then recency. token_hash is
// never projected here either.
func (f *fakeStore) ListAllActiveSessions(_ context.Context, expiresAt pgtype.Timestamptz) ([]db.ListAllActiveSessionsRow, error) {
	rows := []db.ListAllActiveSessionsRow{}
	for _, sess := range f.sessions {
		if sess.RevokedAt.Valid || !sess.ExpiresAt.Time.After(expiresAt.Time) {
			continue
		}
		acct := f.accounts[sess.AccountID]
		rows = append(rows, db.ListAllActiveSessionsRow{
			ID: sess.ID, AccountID: sess.AccountID, Username: acct.Username, Role: acct.Role,
			CreatedAt: sess.CreatedAt, LastSeenAt: sess.LastSeenAt,
			UserAgent: sess.UserAgent, Ip: sess.Ip, ExpiresAt: sess.ExpiresAt,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Username != rows[j].Username {
			return rows[i].Username < rows[j].Username
		}
		if !rows[i].LastSeenAt.Time.Equal(rows[j].LastSeenAt.Time) {
			return rows[i].LastSeenAt.Time.After(rows[j].LastSeenAt.Time)
		}
		return rows[i].ID > rows[j].ID
	})
	return rows, nil
}

// RevokeSessionByIDForAdmin revokes any one live session by id, NOT owner-scoped — the
// admin single-revoke, gated by requireAdmin at the handler. Idempotent.
func (f *fakeStore) RevokeSessionByIDForAdmin(_ context.Context, arg db.RevokeSessionByIDForAdminParams) error {
	for i := range f.sessions {
		if f.sessions[i].ID == arg.ID && !f.sessions[i].RevokedAt.Valid {
			f.sessions[i].RevokedAt = arg.RevokedAt
			return nil
		}
	}
	return nil
}

// --- tests -----------------------------------------------------------------

func profileBase(t *testing.T) (*fakeStore, string, db.Account) {
	t.Helper()
	f := newFakeStore()
	acct := seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	return f, start(t, f, ""), acct
}

var mintedRE = regexp.MustCompile(`vg_pat_[0-9a-f]{48}`)

// A Profile is personal, so it is viewer-readable — but only to a signed-in
// account. Every route redirects an anonymous caller to /login rather than
// leaking a Profile or accepting a mutation.
func TestProfileRequiresLogin(t *testing.T) {
	_, base, _ := profileBase(t)
	c := newClient(t)

	resp, err := c.Get(base + "/profile")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("GET /profile anon: status=%d loc=%q, want 303 to /login", resp.StatusCode, resp.Header.Get("Location"))
	}

	for _, path := range []string{"/profile/password", "/profile/tokens", "/profile/tokens/revoke", "/profile/session/revoke"} {
		resp := postForm(t, c, base+path, url.Values{})
		resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
			t.Fatalf("POST %s anon: status=%d loc=%q, want 303 to /login", path, resp.StatusCode, resp.Header.Get("Location"))
		}
	}
}

// The page renders the real account facts: the username identity, the 2FA status,
// the current session read from the request, and the empty tokens state.
func TestProfileRendersRealAccount(t *testing.T) {
	_, base, _ := profileBase(t)
	c := login(t, base, "ola", "hunter2hunter2")
	got := getBody(t, c, base+"/profile", http.StatusOK)

	for _, want := range []string{
		"Profile", "Who you are", `value="ola"`, // identity
		"Password &amp; two-factor", "two-factor off", "Enable two-factor", // credentials + 2FA status
		"Signed in right now", // sessions
		"Personal API tokens", "You have no personal API tokens", // tokens empty state
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("profile missing %q; body: %s", want, got)
		}
	}
}

// An enrolled account shows the enabled 2FA status rather than the enable control.
func TestProfile2FAStatusEnabled(t *testing.T) {
	f, base, acct := profileBase(t)
	// Sign in first (password-only), then enrol: enabling TOTP does not invalidate
	// the stateless session already held, so the enabled status renders on reload.
	c := login(t, base, "ola", "hunter2hunter2")
	a := f.accounts[acct.ID]
	a.TotpSecret = pgtype.Text{String: "SECRET", Valid: true}
	a.TotpEnabled = true
	f.accounts[acct.ID] = a

	got := getBody(t, c, base+"/profile", http.StatusOK)
	if !strings.Contains(got, "two-factor enabled") {
		t.Fatalf("enabled 2FA status not shown; body: %s", got)
	}
	if strings.Contains(got, "Enable two-factor") {
		t.Fatalf("enable control shown while already enrolled; body: %s", got)
	}
}

// Minting a token reveals the plaintext exactly once and keeps only its hash — a
// reload never shows the secret again, and the stored value is the digest.
func TestProfileTokenRevealOnce(t *testing.T) {
	f, base, _ := profileBase(t)
	c := login(t, base, "ola", "hunter2hunter2")

	resp := postForm(t, c, base+"/profile/tokens", url.Values{"name": {"laptop-cli"}})
	page := body(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create token: status=%d", resp.StatusCode)
	}
	m := mintedRE.FindString(page)
	if m == "" {
		t.Fatalf("minted token not revealed; body: %s", page)
	}
	if !strings.Contains(page, "Shown once") {
		t.Fatalf("reveal-once warning missing; body: %s", page)
	}

	// Stored material is the hash, never the plaintext.
	if len(f.personalTokens) != 1 {
		t.Fatalf("tokens stored = %d, want 1", len(f.personalTokens))
	}
	sum := sha256.Sum256([]byte(m))
	if f.personalTokens[0].TokenHash != hex.EncodeToString(sum[:]) {
		t.Fatalf("stored hash does not match sha256(plaintext)")
	}
	if strings.Contains(f.personalTokens[0].TokenHash, m) || f.personalTokens[0].TokenHash == m {
		t.Fatalf("plaintext token was stored")
	}

	// A reload shows the prefix in the list but never the plaintext again.
	got := getBody(t, c, base+"/profile", http.StatusOK)
	if strings.Contains(got, m) {
		t.Fatalf("plaintext token shown again on reload")
	}
	if !strings.Contains(got, "laptop-cli") || !strings.Contains(got, f.personalTokens[0].Prefix) {
		t.Fatalf("token row (name + prefix) not listed; body: %s", got)
	}
}

// A duplicate token name is refused rather than minting a second row.
func TestProfileTokenDuplicateName(t *testing.T) {
	f, base, _ := profileBase(t)
	c := login(t, base, "ola", "hunter2hunter2")

	postForm(t, c, base+"/profile/tokens", url.Values{"name": {"ci"}}).Body.Close()
	resp := postForm(t, c, base+"/profile/tokens", url.Values{"name": {"ci"}})
	if got := body(t, resp); !strings.Contains(got, "already have a token named that") {
		t.Fatalf("duplicate token name not reported; body: %s", got)
	}
	if len(f.personalTokens) != 1 {
		t.Fatalf("tokens after duplicate = %d, want 1", len(f.personalTokens))
	}
}

// Revoke is a plain danger ConfirmDialog (SPEC-CHANGE #18): the dialog names the token
// and confirms with a single danger action — no typed-name gate — and a confirm POST
// deletes it, carrying the "Token revoked" toast (Profile.jsx:150) on the redirect.
func TestProfileTokenRevokePlainConfirm(t *testing.T) {
	f, base, acct := profileBase(t)
	c := login(t, base, "ola", "hunter2hunter2")
	tok, _ := f.CreatePersonalToken(t.Context(), db.CreatePersonalTokenParams{
		AccountID: acct.ID, Name: "grafana", Prefix: "vg_pat_x81m…", TokenHash: "h",
	})

	// The revoke control is a link to the confirm dialog, never a direct POST. The dialog
	// names its target but no longer collects a typed confirmation.
	dialog := getBody(t, c, base+"/profile?revoke="+strconv.FormatInt(tok.ID, 10), http.StatusOK)
	if !strings.Contains(dialog, "Revoke grafana") {
		t.Fatalf("revoke ConfirmDialog not shown; body: %s", dialog)
	}
	if strings.Contains(dialog, "confirm_name") {
		t.Fatalf("typed-name gate should be dropped from the token-revoke dialog (#18); body: %s", dialog)
	}

	// Confirming the plain dialog (id only) revokes the token and redirects with the toast.
	resp := postForm(t, c, base+"/profile/tokens/revoke", url.Values{"id": {strconv.FormatInt(tok.ID, 10)}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, "/profile?toast=") {
		t.Fatalf("revoke: status=%d loc=%q, want 303 to /profile?toast=…", resp.StatusCode, loc)
	}
	toast := decodeToast(t, loc)
	if toast["tone"] != "neutral" || toast["title"] != "Token revoked" || toast["description"] != "grafana" {
		t.Fatalf("token-revoke toast = %+v, want neutral/Token revoked/grafana", toast)
	}
	if len(f.personalTokens) != 0 {
		t.Fatalf("token not revoked; count=%d", len(f.personalTokens))
	}
}

// Changing the password verifies the current one, updates the hash, and leaves the
// account able to sign in with the new password.
func TestProfileChangePassword(t *testing.T) {
	f, base, acct := profileBase(t)
	c := login(t, base, "ola", "hunter2hunter2")

	// A wrong current password is refused and changes nothing.
	before := f.accounts[acct.ID].PasswordHash
	resp := postForm(t, c, base+"/profile/password", url.Values{
		"current_password": {"nope"}, "new_password": {"brandnewpass99"},
	})
	if got := body(t, resp); !strings.Contains(got, "Current password is incorrect") {
		t.Fatalf("wrong current password not reported; body: %s", got)
	}
	if f.accounts[acct.ID].PasswordHash != before {
		t.Fatalf("password changed despite wrong current password")
	}

	// The correct current password changes it.
	resp = postForm(t, c, base+"/profile/password", url.Values{
		"current_password": {"hunter2hunter2"}, "new_password": {"brandnewpass99"},
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("change password: status=%d, want 303", resp.StatusCode)
	}
	if !auth.CheckPassword(f.accounts[acct.ID].PasswordHash, "brandnewpass99") {
		t.Fatalf("password hash not updated to the new password")
	}
	// The new password now signs in.
	login(t, base, "ola", "brandnewpass99")
}
