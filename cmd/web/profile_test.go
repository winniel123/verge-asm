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

func (f *fakeStore) GetPersonalTokenByHash(_ context.Context, tokenHash string) (db.PersonalToken, error) {
	for _, t := range f.personalTokens {
		if t.TokenHash == tokenHash {
			return t, nil
		}
	}
	return db.PersonalToken{}, pgx.ErrNoRows
}

func (f *fakeStore) UpdatePersonalTokenLastUsed(_ context.Context, id int64) error {
	// Both this fake and the SQL read wall-clock now, so a test's two touches share one hour.
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

func (f *fakeStore) GetSessionByTokenHash(_ context.Context, arg db.GetSessionByTokenHashParams) (db.Session, error) {
	for _, sess := range f.sessions {
		if sess.TokenHash == arg.TokenHash && !sess.RevokedAt.Valid && sess.ExpiresAt.Time.After(arg.ExpiresAt.Time) {
			return sess, nil
		}
	}
	return db.Session{}, pgx.ErrNoRows
}

func (f *fakeStore) TouchSession(_ context.Context, arg db.TouchSessionParams) error {
	for i := range f.sessions {
		if f.sessions[i].ID == arg.ID {
			f.sessions[i].LastSeenAt = arg.LastSeenAt
			return nil
		}
	}
	return nil
}

func (f *fakeStore) RevokeSession(_ context.Context, arg db.RevokeSessionParams) error {
	for i := range f.sessions {
		if f.sessions[i].ID == arg.ID && f.sessions[i].AccountID == arg.AccountID && !f.sessions[i].RevokedAt.Valid {
			f.sessions[i].RevokedAt = arg.RevokedAt
			return nil
		}
	}
	return nil
}

func (f *fakeStore) ListSessionsForAccount(_ context.Context, arg db.ListSessionsForAccountParams) ([]db.ListSessionsForAccountRow, error) {
	// The projection drops token_hash, so secret material stays out of the render path.
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

func (f *fakeStore) RevokeOtherSessionsForAccount(_ context.Context, arg db.RevokeOtherSessionsForAccountParams) error {
	for i := range f.sessions {
		if f.sessions[i].AccountID == arg.AccountID && f.sessions[i].ID != arg.ID && !f.sessions[i].RevokedAt.Valid {
			f.sessions[i].RevokedAt = arg.RevokedAt
		}
	}
	return nil
}

func (f *fakeStore) RevokeAllSessionsForAccount(_ context.Context, arg db.RevokeAllSessionsForAccountParams) error {
	for i := range f.sessions {
		if f.sessions[i].AccountID == arg.AccountID && !f.sessions[i].RevokedAt.Valid {
			f.sessions[i].RevokedAt = arg.RevokedAt
		}
	}
	return nil
}

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

func (f *fakeStore) RevokeSessionByIDForAdmin(_ context.Context, arg db.RevokeSessionByIDForAdminParams) error {
	for i := range f.sessions {
		if f.sessions[i].ID == arg.ID && !f.sessions[i].RevokedAt.Valid {
			f.sessions[i].RevokedAt = arg.RevokedAt
			return nil
		}
	}
	return nil
}

func profileBase(t *testing.T) (*fakeStore, string, db.Account) {
	t.Helper()
	f := newFakeStore()
	acct := seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	return f, start(t, f, ""), acct
}

var mintedRE = regexp.MustCompile(`vg_pat_[0-9a-f]{48}`)

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

func TestProfileRendersRealAccount(t *testing.T) {
	_, base, _ := profileBase(t)
	c := login(t, base, "ola", "hunter2hunter2")
	got := getBody(t, c, base+"/profile", http.StatusOK)

	for _, want := range []string{
		"Profile", "Who you are", `value="ola"`,
		"Password &amp; two-factor", "two-factor off", "Enable two-factor",
		"Signed in right now",
		"Personal API tokens", "You have no personal API tokens",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("profile missing %q; body: %s", want, got)
		}
	}
}

func TestProfile2FAStatusEnabled(t *testing.T) {
	f, base, acct := profileBase(t)
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

	got := getBody(t, c, base+"/profile", http.StatusOK)
	if strings.Contains(got, m) {
		t.Fatalf("plaintext token shown again on reload")
	}
	if !strings.Contains(got, "laptop-cli") || !strings.Contains(got, f.personalTokens[0].Prefix) {
		t.Fatalf("token row (name + prefix) not listed; body: %s", got)
	}
}

func TestProfileTokenDuplicateName(t *testing.T) {
	f, base, _ := profileBase(t)
	c := login(t, base, "ola", "hunter2hunter2")

	postForm(t, c, base+"/profile/tokens", url.Values{"name": {"ci"}}).Body.Close()
	if loc := submitLoc(t, postForm(t, c, base+"/profile/tokens", url.Values{"name": {"ci"}})); loc != "/profile" {
		t.Fatalf("refused token create landed at %q, want /profile", loc)
	}
	got := getBody(t, c, base+"/profile", http.StatusOK)
	if !strings.Contains(got, "already have a token named that") {
		t.Fatalf("duplicate token name not reported on the landing page; body: %s", got)
	}
	if !strings.Contains(got, `value="ci"`) {
		t.Fatalf("the typed token name was not echoed back; body: %s", got)
	}
	if again := getBody(t, c, base+"/profile", http.StatusOK); strings.Contains(again, "already have a token named that") {
		t.Fatalf("the callout survived a reload; body: %s", again)
	}
	if len(f.personalTokens) != 1 {
		t.Fatalf("tokens after duplicate = %d, want 1", len(f.personalTokens))
	}
}

func TestProfileTokenRevokePlainConfirm(t *testing.T) {
	f, base, acct := profileBase(t)
	c := login(t, base, "ola", "hunter2hunter2")
	tok, _ := f.CreatePersonalToken(t.Context(), db.CreatePersonalTokenParams{
		AccountID: acct.ID, Name: "grafana", Prefix: "vg_pat_x81m…", TokenHash: "h",
	})

	dialog := getBody(t, c, base+"/profile?revoke="+strconv.FormatInt(tok.ID, 10), http.StatusOK)
	if !strings.Contains(dialog, "Revoke grafana") {
		t.Fatalf("revoke ConfirmDialog not shown; body: %s", dialog)
	}
	if strings.Contains(dialog, "confirm_name") {
		t.Fatalf("typed-name gate should be dropped from the token-revoke dialog (#18); body: %s", dialog)
	}

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

func TestProfileChangePassword(t *testing.T) {
	f, base, acct := profileBase(t)
	c := login(t, base, "ola", "hunter2hunter2")

	before := f.accounts[acct.ID].PasswordHash
	resp := postForm(t, c, base+"/profile/password", url.Values{
		"current_password": {"nope"}, "new_password": {"brandnewpass99"},
	})
	if loc := submitLoc(t, resp); loc != "/profile" {
		t.Fatalf("refused password change landed at %q, want /profile", loc)
	}
	if got := getBody(t, c, base+"/profile", http.StatusOK); !strings.Contains(got, "Current password is incorrect") {
		t.Fatalf("wrong current password not reported on the landing page; body: %s", got)
	}
	if f.accounts[acct.ID].PasswordHash != before {
		t.Fatalf("password changed despite wrong current password")
	}

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
	login(t, base, "ola", "brandnewpass99")
}
