package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/seed"
)

// Dev-only pixel-parity affordances for screen 2 (ErrorPage, package v3.5.0,
// WORK-ORDER-2-ERROR.md). Everything here is gated to a VERGE_DEV build — the /dev
// routes are registered only when s.devMode (handlers.go) and the seed helpers run
// only from the -seed-fixtures one-shot, which main.go bars outside VERGE_DEV — so no
// dev surface exists in a real deployment. It exists so the harness can drive the six
// error capture states states.json declares, deterministically.

// devFixtureIncidentID is the incident id the 500 error page shows in a VERGE_DEV build,
// so the harness's 500 golden is stable. It mirrors design-system/fixtures/fixtures.json
// → error.incident_id; TestDevFixturesMatchPackage asserts the two never drift. A real
// build keeps the crypto/rand id (newIncidentID, errors.go).
const devFixtureIncidentID = "err_9f3ka72c"

// devFixtureAccount is one design fixture login the harness signs in as. states.json
// carries a per-state `session` ("admin" | "viewer"); the dev session mint resolves that
// role to one of these accounts and opens a session before the state's route is captured.
// Dev-only — well-known passwords, seeded only under VERGE_DEV, mirroring fixtures.json →
// accounts (asserted by TestDevFixturesMatchPackage). The password is #nosec G101: not a
// real credential, only ever seeded into a throwaway dev database.
type devFixtureAccount struct {
	username string
	role     string
	password string // #nosec G101 -- dev-only fixture login, seeded only under VERGE_DEV
}

var devFixtureAccounts = []devFixtureAccount{
	{username: "ola.perez", role: roleAdmin, password: "verge-dev-1"},   // #nosec G101 -- dev-only fixture login, seeded only under VERGE_DEV
	{username: "sam.reader", role: roleViewer, password: "verge-dev-2"}, // #nosec G101 -- dev-only fixture login, seeded only under VERGE_DEV
}

// devPanic is the /dev/panic handler (VERGE_DEV only): it panics so recoverPanics
// renders the 500 error page with the deterministic incident id — the harness's 500
// capture state. The panic value is fixed for a stable host-log line.
func (s *server) devPanic(http.ResponseWriter, *http.Request) {
	panic("fixture")
}

// devSessionMint is the /dev/session/{role} handler (VERGE_DEV only): it opens a session
// as the fixture account for the requested role ("admin" | "viewer") and hands back the
// session cookie via completeLogin, so the harness can establish states.json's per-state
// `session` before it navigates the state's route. An unknown role, or a role whose
// fixture account was not seeded, is a plain 404 — the mint never invents an account.
func (s *server) devSessionMint(w http.ResponseWriter, r *http.Request) {
	role := r.PathValue("role")
	username := ""
	for _, a := range devFixtureAccounts {
		if a.role == role {
			username = a.username
			break
		}
	}
	if username == "" {
		s.notFound(w, r)
		return
	}
	acct, err := s.store.GetAccountByUsername(r.Context(), username)
	if err != nil {
		s.notFound(w, r)
		return
	}
	s.completeLogin(w, r, acct.ID)
}

// devProfileSessionPrepare is the /dev/profile/session handler (VERGE_DEV only): it prepares the
// Profile capture context for one state (screen 3, #542). Unlike devSessionMint — which opens a
// FRESH session row and would add a fourth, badge-stealing session to the fixture account — this:
//
//  1. re-seeds the Profile fixture (seedProfileFixtures is idempotent: it resets the account's
//     own sessions/identities/tokens), so a token minted by the previous "minted" capture state
//     never leaks into the next state's tokens table; and
//  2. hands back a signed session cookie wrapping the well-known seeded current-session token
//     (devProfileCurrentSessionToken). That token's hash is on the seeded Firefox·macOS row, so
//     currentAccount resolves the capture as ola.perez AND currentSessionID resolves to that row
//     — the one fixtures.json marks current — so it wears the "this device" badge, and no new
//     session row is created (the sessions table stays the fixture's three).
//
// The reseed recreates the current-session row with the SAME token hash, so the cookie keeps
// resolving across the per-state reseeds. Dev-only, nil-guarded on the pool.
func (s *server) devProfileSessionPrepare(w http.ResponseWriter, r *http.Request) {
	if s.pool == nil {
		s.notFound(w, r)
		return
	}
	if err := seedProfileFixtures(r.Context(), s.pool); err != nil {
		s.serverError(w, "dev: reseed profile fixture", err)
		return
	}
	acct, err := s.store.GetAccountByUsername(r.Context(), devProfileUsername)
	if err != nil {
		s.notFound(w, r)
		return
	}
	if !s.setSignedCookie(w, r, sessionCookie, auth.KindSession, acct.ID, devProfileCurrentSessionToken, s.sessionTTL) {
		return
	}
	// A 200 with a tiny body (not 204) so the harness's page.goto completes the navigation and
	// stores the Set-Cookie — a 204 makes Chromium abort the navigation (net::ERR_ABORTED).
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

// seedDevFixtureAccounts makes the fixtures.json admin + viewer exist so the dev session
// mint can sign in as either. It is idempotent — an account already present (by username)
// is left untouched — so a re-seed is a no-op, and it runs only from the -seed-fixtures
// one-shot (main.go, VERGE_DEV-gated), beside seedInventoryFixtures/seedDevOperator.
func seedDevFixtureAccounts(ctx context.Context, pool *pgxpool.Pool) error {
	q := db.New(pool)
	for _, a := range devFixtureAccounts {
		if _, err := q.GetAccountByUsername(ctx, a.username); err == nil {
			continue // already seeded
		}
		hash, err := auth.HashPassword(a.password)
		if err != nil {
			return err
		}
		if _, err := q.CreateAccount(ctx, db.CreateAccountParams{
			Username: a.username, Role: a.role, PasswordHash: hash,
		}); err != nil {
			return err
		}
	}
	return nil
}

// --- screen 3: Profile fixture (package v3.6.0, WORK-ORDER-3-PROFILE.md) ------
//
// Everything below seeds the Profile screen's frozen fixture (design-system/fixtures/
// fixtures.json → profile) into a VERGE_DEV instance so /profile renders it byte-for-byte
// for the pixel-parity harness. It runs only from the -seed-fixtures one-shot (main.go,
// VERGE_DEV-gated), beside seedDevFixtureAccounts, and touches only the fixture account's
// own rows. The pinned constants here are the single source of the seed; the drift test
// (TestProfileFixtureMatchesPackage) folds each one back through fixtures.json and fails
// the build on any divergence — the byte-exactness gate before the pixels, exactly as
// TestDevFixturesMatchPackage guards the ErrorPage slice.

const (
	// devFixtureClock is fixtures.json → clock: the injectable server clock a VERGE_DEV
	// build pins (main.go), so relative-time renders (the Profile sessions/tokens tables)
	// read a fixed instant rather than wall time and the goldens never drift. Inventory's
	// since-dates are absolute and unaffected; this only makes the relative renders stable.
	devFixtureClock = "2026-08-24T12:00:00Z"

	// devFixtureMintedToken is fixtures.json → profile.minted_token_fixture: the plaintext
	// the token-create handler reveals in a VERGE_DEV build (newPersonalToken, auth.go) so
	// the minted-dialog golden's pixel diff is stable. A real build always draws crypto/rand.
	devFixtureMintedToken = "vg_pat_cigolden0example" // #nosec G101 -- not a real credential: a fixed dev-only fixture token revealed only in a VERGE_DEV build

	devProfileUsername = "ola.perez"

	// devProfileCreated is fixtures.json → profile.account.created: the date-only UTC
	// account-creation date the Profile's "Member since" row renders (isoDate(acct.CreatedAt)).
	// seedDevFixtureAccounts creates the account at wall-clock now(), so seedProfileFixtures
	// pins created_at back to this fixture date; otherwise "Member since" drifts to the seed
	// day (a "row data equals fixtures" miss). TestProfileFixtureMatchesPackage folds it back
	// through the frozen package.
	devProfileCreated = "2026-04-11"

	devProfileSSOProviderSlug = "okta"
	devProfileSSOProviderName = "Okta"
	devProfileSSODisplayName  = "ola.perez@acmecorp.io"
	devProfileSSOLinkedAt     = "2026-06-30"

	devProfileLinkableSlug = "google"
	devProfileLinkableName = "Google"

	// devProfileCurrentSessionToken is the opaque session-cookie plaintext the capture
	// harness (#542) presents so currentSessionID resolves to the seeded Firefox·macOS row
	// and it wears the "this device" badge (fixtures.json marks that session current:true).
	// Dev-only; only its SHA-256 hash is stored on the row, like every other session token.
	devProfileCurrentSessionToken = "verge-dev-profile-session-1" // #nosec G101 -- not a real credential: a fixed dev-only session token seeded only under VERGE_DEV
)

// devProfileSession is one seeded Profile session. device is the fixture's expected DISPLAY
// (fixtures.json → profile.sessions[].device); userAgent is the stored user_agent the app's
// sessionDeviceFromUA derivation renders back into exactly that display (the drift test
// asserts the round-trip). lastOffset is subtracted from the pinned clock to stamp
// last_seen_at, so relTime renders the fixture's relative token (now / 2h / 3d).
type devProfileSession struct {
	device     string
	userAgent  string
	ip         string
	lastActive string        // fixtures.json display token: now / 2h / 3d
	lastOffset time.Duration // clock − this = last_seen_at
	current    bool
}

// devProfileSessions pins fixtures.json → profile.sessions, in fixture order (which is also
// last-active order, newest first). The user_agent strings are chosen so sessionDeviceFromUA
// derives the fixture device exactly — including the CLI session, which the derivation labels
// "CLI · <host>" from the verge-cli client string (settings.go).
var devProfileSessions = []devProfileSession{
	{device: "Firefox · macOS", userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15) Gecko/20100101 Firefox/128.0", ip: "198.51.100.7", lastActive: "now", lastOffset: 0, current: true},
	{device: "CLI · verge@build-07", userAgent: "verge-cli/1.0 (verge@build-07)", ip: "203.0.113.44", lastActive: "2h", lastOffset: 2 * time.Hour},
	{device: "Safari · iOS", userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile Safari/604.1", ip: "198.51.100.31", lastActive: "3d", lastOffset: 3 * 24 * time.Hour},
}

// devProfileToken is one seeded personal API token. created is the fixture's date-only UTC
// created date (rendered verbatim); lastOffset is subtracted from the pinned clock to stamp
// last_used_at so relTime renders the fixture's relative token (2h / 14d).
type devProfileToken struct {
	name       string
	prefix     string
	created    string // fixtures.json date-only, YYYY-MM-DD
	last       string // fixtures.json display token: 2h / 14d ("" when never used)
	lastOffset time.Duration
	lastNull   bool // #390: never-used token — last_used_at is NULL, renders "never"
}

// devProfileTokens pins fixtures.json → profile.tokens, in fixture order. The third,
// ci-export (#390, v3.18.0), has never been used: its last_used_at is NULL, so the
// Profile renders it "never".
var devProfileTokens = []devProfileToken{
	{name: "laptop-cli", prefix: "vg_pat_9f3k…", created: "2026-05-02", last: "2h", lastOffset: 2 * time.Hour},
	{name: "grafana-readonly", prefix: "vg_pat_x81m…", created: "2026-07-19", last: "14d", lastOffset: 14 * 24 * time.Hour},
	{name: "ci-export", prefix: "vg_pat_r55q…", created: "2026-08-20", last: "", lastNull: true},
}

func devFixtureClockTime() (time.Time, error) {
	return time.Parse(time.RFC3339, devFixtureClock)
}

func devFixtureDate(d string) (time.Time, error) {
	return time.Parse("2006-01-02", d)
}

// seedProfileFixtures seeds the Profile screen's fixture (fixtures.json → profile) onto the
// already-seeded ola.perez admin account: totp_enabled=true (verbatim per the frozen
// package), the three sessions with clock-relative last_seen_at, the Okta linked identity +
// Google as a linkable provider, and the two personal tokens. It is idempotent — it resets
// the account's own session/identity/token rows before reinserting — and runs only from the
// -seed-fixtures one-shot (main.go, VERGE_DEV-gated), after seedDevFixtureAccounts has
// created the account. All timestamps derive from the pinned clock so a VERGE_DEV server
// (whose clock is pinned to the same instant) renders the fixture's relative + date-only
// values exactly.
func seedProfileFixtures(ctx context.Context, pool *pgxpool.Pool) error {
	clock, err := devFixtureClockTime()
	if err != nil {
		return fmt.Errorf("profile fixture: parse clock: %w", err)
	}
	q := db.New(pool)
	acct, err := q.GetAccountByUsername(ctx, devProfileUsername)
	if err != nil {
		return fmt.Errorf("profile fixture: account %q not seeded: %w", devProfileUsername, err)
	}

	// TOTP on — the frozen fixture sets account.totp_enabled=true and states.json captures
	// the goldens on this account with no TOTP-off state, so the package is internally
	// consistent at TOTP-ON. Seed it verbatim (the "no-TOTP viewer" ticket prose is
	// superseded by the frozen package; the TOTP-off branch is proven by unit test, not a
	// golden). Only totp_enabled is set — the harness reaches this account via the dev
	// session mint, which bypasses the password/TOTP challenge, so no secret is required.
	if _, err := pool.Exec(ctx, `UPDATE account SET totp_enabled = true WHERE id = $1`, acct.ID); err != nil {
		return fmt.Errorf("profile fixture: enable totp: %w", err)
	}

	// Member since — the account was created at wall-clock now() by seedDevFixtureAccounts,
	// so pin created_at to the fixture's date-only UTC value (fixtures.json → profile.account.
	// created) or the "Member since" row (isoDate(acct.CreatedAt)) drifts to the seed day.
	created, err := devFixtureDate(devProfileCreated)
	if err != nil {
		return fmt.Errorf("profile fixture: parse account created: %w", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE account SET created_at = $2 WHERE id = $1`, acct.ID, created); err != nil {
		return fmt.Errorf("profile fixture: pin account created: %w", err)
	}

	// Sessions — reset this account's rows, then reinsert the three with last_seen_at at the
	// pinned-clock offset so relTime renders now / 2h / 3d. The first row carries the
	// deterministic current-session token hash so the harness can present its cookie and be
	// read as the current session; the others get distinct fixed hashes.
	if _, err := pool.Exec(ctx, `DELETE FROM session WHERE account_id = $1`, acct.ID); err != nil {
		return fmt.Errorf("profile fixture: reset sessions: %w", err)
	}
	const insSession = `
		INSERT INTO session (account_id, token_hash, user_agent, ip, created_at, last_seen_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	for i, ps := range devProfileSessions {
		lastSeen := clock.Add(-ps.lastOffset)
		// The current (Firefox·macOS) row carries the well-known token the capture harness
		// (#542) presents so currentSessionID resolves to it; the rest get distinct fixed ones.
		plaintext := fmt.Sprintf("verge-dev-profile-session-%d", i+1)
		if ps.current {
			plaintext = devProfileCurrentSessionToken
		}
		if _, err := pool.Exec(ctx, insSession,
			acct.ID, hashToken(plaintext), ps.userAgent, ps.ip, lastSeen, lastSeen, clock.Add(12*time.Hour),
		); err != nil {
			return fmt.Errorf("profile fixture: insert session %q: %w", ps.device, err)
		}
	}

	// SSO — Okta (the linked identity's provider) and Google (offered as linkable). Both
	// enabled and upserted by slug so a re-seed is a no-op. created_by is the fixture admin.
	oktaID, err := seedDevSSOProvider(ctx, pool, devProfileSSOProviderSlug, devProfileSSOProviderName, acct.ID, clock)
	if err != nil {
		return err
	}
	if _, err := seedDevSSOProvider(ctx, pool, devProfileLinkableSlug, devProfileLinkableName, acct.ID, clock); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM sso_identity WHERE account_id = $1`, acct.ID); err != nil {
		return fmt.Errorf("profile fixture: reset sso identities: %w", err)
	}
	linkedAt, err := devFixtureDate(devProfileSSOLinkedAt)
	if err != nil {
		return fmt.Errorf("profile fixture: parse sso linked_at: %w", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO sso_identity (provider_id, account_id, sub, display_name, created_at) VALUES ($1, $2, $3, $4, $5)`,
		oktaID, acct.ID, "okta|"+devProfileUsername, devProfileSSODisplayName, linkedAt,
	); err != nil {
		return fmt.Errorf("profile fixture: insert sso identity: %w", err)
	}

	// Personal tokens — reset, then reinsert t1/t2 with the fixture's date-only created dates
	// and clock-relative last_used_at (2h / 14d).
	if _, err := pool.Exec(ctx, `DELETE FROM personal_token WHERE account_id = $1`, acct.ID); err != nil {
		return fmt.Errorf("profile fixture: reset tokens: %w", err)
	}
	const insToken = `INSERT INTO personal_token (account_id, name, prefix, token_hash, created_at, last_used_at) VALUES ($1, $2, $3, $4, $5, $6)` // #nosec G101 -- SQL statement, not a credential: the const name and the token_hash column trip the G101 heuristic
	for _, pt := range devProfileTokens {
		created, derr := devFixtureDate(pt.created)
		if derr != nil {
			return fmt.Errorf("profile fixture: parse token created %q: %w", pt.created, derr)
		}
		// A never-used token (ci-export, #390) carries a NULL last_used_at so lastUsed
		// renders "never"; the others stamp a clock-relative last_used_at (2h / 14d).
		lastUsedAt := pgtype.Timestamptz{Time: clock.Add(-pt.lastOffset), Valid: true}
		if pt.lastNull {
			lastUsedAt = pgtype.Timestamptz{}
		}
		if _, err := pool.Exec(ctx, insToken,
			acct.ID, pt.name, pt.prefix, hashToken("verge-dev-token-"+pt.name), created, lastUsedAt,
		); err != nil {
			return fmt.Errorf("profile fixture: insert token %q: %w", pt.name, err)
		}
	}
	return nil
}

// --- screen 4: SignIn family fixture (package v3.7.0, WORK-ORDER-4-6-BATCH1.md) -------------
//
// Everything below pins the SignIn-family fixture (design-system/fixtures/fixtures.json →
// signin) so the login / totp / enroll / recovery / forgot / reset / invite states render
// byte-for-byte for the pixel harness. The pinned constants are the single source of the seed
// and the dev capture affordances (buildVersion, loginProviders, the totp code + recovery-code
// determinism, beginTOTPEnroll); TestSigninFixtureMatchesPackage folds each one back through
// fixtures.json → signin and fails the build on any divergence — the byte-exactness gate before
// the pixels, exactly as TestProfileFixtureMatchesPackage guards the Profile slice. All of it
// runs only under VERGE_DEV; a real deployment reaches none of it.

const (
	// devFixtureVersion is fixtures.json → signin.version: the build version the chrome-less
	// authfoot renders ({{.Version}}). buildVersion returns it in a VERGE_DEV build so the
	// SignIn/Setup goldens' footer is stable; a real build reads VERGE_VERSION.
	devFixtureVersion = "v0.9.2"

	// devFixtureTOTPCode is fixtures.json → signin.totp_accept_code: the code a VERGE_DEV build
	// accepts at the TOTP login step and at enrolment-confirm, so the harness can pass the
	// second factor deterministically. A real build always runs the RFC 6238 verification.
	devFixtureTOTPCode = "482913"

	// devFixtureEnrollSecret is fixtures.json → signin.enroll_secret: the secret the enrolment
	// screen shows (and encodes into its QR) in a VERGE_DEV build, so the enroll golden is
	// stable. It is the design's readable display string (dash-grouped), not a verifiable base32
	// seed — which is why dev confirm accepts the pinned code rather than verifying against it.
	devFixtureEnrollSecret = "VG7K-2Q9X-8MRD-P3TL" // #nosec G101 -- not a real credential: a fixed dev-only fixture secret shown only in a VERGE_DEV build

	// devFixtureResetToken / devFixtureInviteToken are fixtures.json → signin.reset_token /
	// invite_token: the well-known plaintext tokens the seed writes (as SHA-256 hashes) so
	// /reset?token=… and /invite?token=… resolve to a real row for the capture.
	devFixtureResetToken  = "fixture-reset-token"  // #nosec G101 -- not a real credential: a fixed dev-only reset token, seeded (hashed) only under VERGE_DEV
	devFixtureInviteToken = "fixture-invite-token" // #nosec G101 -- not a real credential: a fixed dev-only invite token, seeded (hashed) only under VERGE_DEV

	devFixtureInviteRole = roleViewer
)

// devSigninProvider is one pinned login SSO button (fixtures.json → signin.sso_providers): the
// slug, display name, and the mono Mark. loginProviders returns this set in a VERGE_DEV build
// so the login golden shows exactly the fixture's provider — even though the shared fixture DB
// also enables the Profile screen's linkable Google provider (which a real login would list too).
type devSigninProvider struct {
	slug string
	name string
	mark string
}

var devSigninProviders = []devSigninProvider{
	{slug: "okta", name: "Okta", mark: "O"},
}

// devFixtureRecoveryCodes pins fixtures.json → signin.recovery_codes: the recovery set the
// enrolment-confirm reveals in a VERGE_DEV build, so the recovery golden is stable. A real
// enrolment draws fresh high-entropy codes (newRecoveryCodes).
var devFixtureRecoveryCodes = []string{
	"k4mq-9d2x", "7hfa-t3wn", "p8rc-01zk", "vx5j-mm4d",
	"q2sl-88bh", "e6ty-r7cn", "a1zw-kk3p", "n9gd-45vu",
}

func (s *server) recoveryCodes() (plain, hashes []string, err error) {
	if s.devMode {
		plain = append(plain, devFixtureRecoveryCodes...)
		for _, code := range devFixtureRecoveryCodes {
			h, herr := auth.HashPassword(code)
			if herr != nil {
				return nil, nil, herr
			}
			hashes = append(hashes, h)
		}
		return plain, hashes, nil
	}
	return newRecoveryCodes(recoveryCodeCount)
}

// devResetTOTPEnroll clears any prior two-factor enrolment on an account (VERGE_DEV only) so the
// GET /account/totp/enroll capture always lands on the enroll screen, no matter that a previous
// recovery-state capture enabled two-factor on the same fixture account against the shared DB.
// It touches only the named account's own rows (totp columns + recovery codes) and is nil-guarded
// on the raw pool the dev build wires.
func (s *server) devResetTOTPEnroll(ctx context.Context, accountID int64) error {
	if s.pool == nil {
		return fmt.Errorf("dev: reset totp: pool not wired")
	}
	if _, err := s.pool.Exec(ctx,
		`UPDATE account SET totp_enabled = false, totp_secret = NULL, totp_last_step = NULL WHERE id = $1`,
		accountID); err != nil {
		return fmt.Errorf("dev: reset totp columns: %w", err)
	}
	if err := s.store.DeleteRecoveryCodesForAccount(ctx, accountID); err != nil {
		return fmt.Errorf("dev: reset recovery codes: %w", err)
	}
	return nil
}

// seedSigninFixtures seeds the SignIn-family fixture (fixtures.json → signin) so the harness's
// /reset and /invite capture states resolve to a real row: a single-use password-reset grant for
// the ola.perez admin under the well-known reset-token hash, and a viewer-role invite under the
// well-known invite-token hash. Both are pinned unexpired against the fixture clock. It is
// idempotent (it deletes any row under those hashes before inserting) and runs only from the
// -seed-fixtures one-shot (main.go, VERGE_DEV-gated), after seedDevFixtureAccounts. The login
// provider set and the enrol/recovery determinism need no seed — they are dev capture affordances
// resolved in the handlers. TotpEnabled on ola.perez (seeded true by seedProfileFixtures) is what
// makes the login→totp capture land on the second-factor step.
func seedSigninFixtures(ctx context.Context, pool *pgxpool.Pool) error {
	clock, err := devFixtureClockTime()
	if err != nil {
		return fmt.Errorf("signin fixture: parse clock: %w", err)
	}
	q := db.New(pool)
	acct, err := q.GetAccountByUsername(ctx, devProfileUsername)
	if err != nil {
		return fmt.Errorf("signin fixture: account %q not seeded: %w", devProfileUsername, err)
	}

	// Password reset — one unspent, unexpired grant under the well-known token hash, so
	// /reset?token=fixture-reset-token resolves. Reset any prior row under the hash first.
	resetHash := hashToken(devFixtureResetToken)
	if _, err := pool.Exec(ctx, `DELETE FROM password_reset WHERE token_hash = $1`, resetHash); err != nil {
		return fmt.Errorf("signin fixture: reset password_reset: %w", err)
	}
	if _, err := q.CreatePasswordReset(ctx, db.CreatePasswordResetParams{
		AccountID: acct.ID, TokenHash: resetHash,
		ExpiresAt: pgtype.Timestamptz{Time: clock.Add(30 * time.Minute), Valid: true},
	}); err != nil {
		return fmt.Errorf("signin fixture: create password_reset: %w", err)
	}

	// Invite — one unspent, unexpired viewer invite under the well-known token hash, so
	// /invite?token=fixture-invite-token resolves. Reset any prior row under the hash first.
	inviteHash := hashToken(devFixtureInviteToken)
	if _, err := pool.Exec(ctx, `DELETE FROM invite WHERE token_hash = $1`, inviteHash); err != nil {
		return fmt.Errorf("signin fixture: reset invite: %w", err)
	}
	if _, err := q.CreateInvite(ctx, db.CreateInviteParams{
		TokenHash: inviteHash, Role: devFixtureInviteRole,
		InvitedBy: pgtype.Int8{Int64: acct.ID, Valid: true},
		ExpiresAt: pgtype.Timestamptz{Time: clock.Add(72 * time.Hour), Valid: true},
	}); err != nil {
		return fmt.Errorf("signin fixture: create invite: %w", err)
	}
	return nil
}

// --- screen 5: Setup fixture (package v3.7.0, WORK-ORDER-4-6-BATCH1.md) ---------------------
//
// The Setup screen (#550) is the chrome-less first-run bootstrap surface, which renders ONLY
// while no accounts exist. The shared fixture DB the harness seeds always has accounts (the
// admin + viewer + the Profile/SignIn fixtures), so at boot bootstrapSetupToken returns "" and
// the setup window is shut. The dev affordance below is how the Setup capture (states.json
// setup, top-level seed:"empty") reaches the open first-run form: it empties the accounts table
// (closing every dependent row by cascade) and reopens the window under the pinned fixture
// token. There is no positive one-shot seed for Setup — its "fixture" is the empty variant plus
// the pinned token — so main.go's -seed-fixtures block is left untouched. TestSetupFixtureMatchesPackage
// is the byte-exactness gate that the token here equals the frozen fixtures.json → setup.token.

// devFixtureSetupToken is the single-use setup token the dev seed route reopens the first-run
// /setup window with, so the Setup screen's capture renders deterministically. It mirrors
// design-system/fixtures/fixtures.json → setup.token; TestSetupFixtureMatchesPackage folds it
// back through the frozen package. A real deployment draws VERGE_SETUP_TOKEN or a crypto/rand
// token (bootstrapSetupToken) — this value is never used outside a VERGE_DEV build.
const devFixtureSetupToken = "fixture-setup-token" // #nosec G101 -- dev-only fixture setup token, not a real credential

// devSetupSeedEmpty is the /dev/seed/empty handler (VERGE_DEV only): it realizes the Setup
// screen's seed:"empty" variant. Because the shared fixture DB is seeded with accounts (so the
// setup window is shut and s.setupToken is ""), this route empties the account table — cascading
// to every session / identity / token / reset / invite row — and reopens the first-run window
// under devFixtureSetupToken, so GET /setup renders the open form for the pixel capture. Setup is
// the LAST screen the run.sh candidate block captures, so emptying the shared fixture DB here
// never strands an earlier screen's capture. Dev-only, nil-guarded on the raw pool the dev build
// wires; the token assignment is serialised under the same setupMu setupSubmit takes.
func (s *server) devSetupSeedEmpty(w http.ResponseWriter, r *http.Request) {
	if s.pool == nil {
		s.notFound(w, r)
		return
	}
	if _, err := s.pool.Exec(r.Context(), `TRUNCATE account RESTART IDENTITY CASCADE`); err != nil {
		s.serverError(w, "dev: empty accounts for setup capture", err)
		return
	}
	s.setupMu.Lock()
	s.setupToken = devFixtureSetupToken
	s.setupMu.Unlock()
	// A 200 with a tiny body (not 204) so the harness's page.goto completes the navigation — a
	// 204 makes Chromium abort it (net::ERR_ABORTED), mirroring devProfileSessionPrepare.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

// --- screen 7: Exposure fixture (package v3.8.0, WORK-ORDER-7-9-BATCH2.md) ------------------
//
// The Exposure screen (#560/#561) renders inside the full app chrome. Its VIEW corpus — the
// both-legs board (six service rows with per-leg reach states and a "since"), the summary band
// (14 exposed with a +2 vs-last-batch delta, 41 firewalled, 7 not reached) and the WITHHELD
// state — is the design curated fixture, not a live-estate read: the exact rows, counts and
// signed delta cannot be reconstructed from the live derivations without fabricating domain
// data, which SPEC-CHANGE forbids. So, exactly as the SignIn/Setup/Coverage screens pin their
// dev fixture and serve it under devMode, exposurePage serves the pinned fixtures.json →
// exposure slice below when s.devMode, and TestExposureFixtureMatchesPackage folds every value
// back through the frozen package — the byte-exactness gate before the pixels. The WITHHELD
// state rides a dev ?variant=no-internet-vantage query (states.json). All of it is VERGE_DEV-only;
// a real deployment renders the honest live projection in exposure.go exposurePage instead.

const (
	devExposureExposed      = 14
	devExposureExposedDelta = 2
	devExposureFirewalled   = 41
	devExposureNotReached   = 7
	devExposureHasDeltas    = true

	// devExposureWithheldVariant is fixtures.json → exposure.withheld_variant: the ?variant token
	// (states.json exposure "withheld" state) that drives the WITHHELD render — no internet vantage,
	// so exposure is withheld rather than reported.
	devExposureWithheldVariant = "no-internet-vantage"
)

type devExposureRow struct {
	asset    string
	svc      string
	internal string
	internet string
	since    string
}

var devExposureRows = []devExposureRow{
	{asset: "edge-gw-03.acmecorp.io", svc: ":5900 vnc", internal: "exposed", internet: "exposed", since: "4m"},
	{asset: "api.acmecorp.io", svc: ":443 https", internal: "exposed", internet: "exposed", since: "69d"},
	{asset: "vpn.acmecorp.io", svc: ":1194 openvpn", internal: "exposed", internet: "exposed", since: "41d"},
	{asset: "build-07.acmecorp.io", svc: ":22 ssh", internal: "exposed", internet: "firewalled", since: "12d"},
	{asset: "grafana.acmecorp.io", svc: ":3000 http", internal: "exposed", internet: "firewalled", since: "26d"},
	{asset: "203.0.113.61", svc: ":443 https", internal: "not-reached", internet: "unverified", since: "—"},
}

// exposureFixtureData assembles the render data map exposurePage passes to the frozen
// exposure.tmpl in a VERGE_DEV build. It stamps the chrome + design-token holes, then either the
// WITHHELD state (when the ?variant matches devExposureWithheldVariant, mirroring the live
// no-internet-vantage branch) or the full pinned board — the six rows, the summary counts and the
// +2 exposed delta (rendered via the tmpl's signDelta over .ExposedDelta.Change). The delta is fed
// as a {Change} map exactly as render-goldens does, so golden and candidate agree byte-for-byte.
func (s *server) exposureFixtureData(acct db.Account, variant string) map[string]any {
	data := map[string]any{
		"Title": "Exposure", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "exposure", "DesignTokens": true,
	}
	if variant == devExposureWithheldVariant {
		data["Withheld"] = true
		return data
	}

	rows := make([]exposureRow, 0, len(devExposureRows))
	for _, r := range devExposureRows {
		rows = append(rows, exposureRow{
			Asset: r.asset, Svc: r.svc, Internal: r.internal, Internet: r.internet, Since: r.since,
		})
	}
	data["Withheld"] = false
	data["Rows"] = rows
	data["Exposed"] = devExposureExposed
	data["Firewalled"] = devExposureFirewalled
	data["NotReached"] = devExposureNotReached
	if devExposureHasDeltas {
		data["HasDeltas"] = true
		data["ExposedDelta"] = map[string]any{"Change": devExposureExposedDelta}
	}
	return data
}

// seedDevSSOProvider upserts one enabled OIDC provider by slug and returns its id. The
// issuer/client_id are dev placeholders (the Profile screen never exercises the flow — it
// only lists the provider); the client secret stays NULL. Dev-only, from the seed one-shot.
func seedDevSSOProvider(ctx context.Context, pool *pgxpool.Pool, slug, name string, createdBy int64, clock time.Time) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO sso_provider (slug, name, issuer, client_id, enabled, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, true, $5, $6, $6)
		ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, enabled = true
		RETURNING id`,
		slug, name, "https://"+slug+".example.com", slug+"-dev-client", createdBy, clock,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("profile fixture: upsert sso provider %q: %w", slug, err)
	}
	return id, nil
}

// --- screen 6: Coverage fixture (package v3.7.0, WORK-ORDER-4-6-BATCH1.md) ------------------
//
// The Coverage screen (#551/#552) renders inside the full app chrome and is DB-backed only in
// its session (a real admin, minted by the harness). Its VIEW corpus — the #19c address-scope
// counted/total meter, the two-shape aperture (address counted/total, name census), the four
// relative-time currency messages, the gaps register, the unevaluable rules and the per-zone
// stale callout — is the design curated fixture, not a live-estate read: the exact message copy,
// the 198/214 figures (with the "16 skipped" breakdown) and the when/iso pair (When is the
// last-check age, ISO the underlying event instant — the two deliberately do NOT correlate
// through the fixture clock) cannot be reconstructed from the live derivations without fabricating
// domain data, which SPEC-CHANGE forbids. So, exactly as the SignIn/Setup screens pin their dev
// fixture (login providers, recovery codes, setup token) and serve it under devMode, coveragePage
// serves the pinned fixtures.json coverage slice below when s.devMode, and
// TestCoverageFixtureMatchesPackage folds every value back through the frozen package — the
// byte-exactness gate before the pixels. All of it is VERGE_DEV-only; a real deployment renders
// the honest live census reads in cold.go coveragePage instead.

// devCoverageMeter mirrors one fixtures.json coverage.meters entry. total is a pointer so a name
// scope (no denominator) is a census (nil), matching the frozen JSON omitted "total".
type devCoverageMeter struct {
	label   string
	counted int
	total   *int
	unit    string
	detail  string
}

type devCoverageMessage struct {
	kind    string
	badge   string
	bound   string
	subject string
	text    string
	when    string
	iso     string
}

type devCoverageGap struct {
	subject  string
	gap      string
	expected string
	since    string
}

type devCoverageUnevaluable struct {
	id      string
	version int
	why     string
}

type devCoverageStaleZone struct {
	zone string
	age  string
}

// coverageTotal214 backs the address-scope meter nullable denominator (a package-level so its
// address is stable). One named var reads clearer than an intptr helper.
var coverageTotal214 = 214

var devCoverageMeters = []devCoverageMeter{
	{label: "203.0.113.0/24", counted: 198, total: &coverageTotal214, unit: "subjects", detail: "16 skipped: excluded subtree + 3 unresolvable names"},
	{label: "acmecorp.io (name scope)", counted: 62, total: nil, unit: "addresses", detail: "census state — a name scope has no denominator; custody extension reaches what resolution reveals"},
}

var devCoverageMessages = []devCoverageMessage{
	{kind: "gap", badge: "no address", subject: "old-blog.acmecorp.io", text: "Expected a resolution; none observed for 3 checks.", when: "2h", iso: "2026-08-22T12:20:04Z"},
	{kind: "stale", badge: "stale", bound: "9d", subject: "internal.acmecorp.io zone", text: "Zone aged past two re-supply intervals — the source went stale.", when: "9d", iso: "2026-08-13T04:44:19Z"},
	{kind: "silent", badge: "no reports", subject: "dc-fra-01", text: "Vantage stopped reporting mid-batch; open spans are not evaluable.", when: "41m", iso: "2026-08-22T13:41:02Z"},
	{kind: "not-evaluable", badge: "not evaluable", subject: "ap-south-1 conclusions", text: "Missed 2 of 3 checks this batch; exposure conclusions marked unverified.", when: "5h", iso: "2026-08-22T09:03:55Z"},
}

var devCoverageGaps = []devCoverageGap{
	{subject: "old-blog.acmecorp.io", gap: "no address", expected: "A record", since: "2h"},
	{subject: "203.0.113.44:22", gap: "no banner", expected: "ssh identification", since: "6h"},
	{subject: "mail.acmecorp.io:25", gap: "no exchange", expected: "smtp greeting", since: "1d"},
}

var devCoverageUnevaluables = []devCoverageUnevaluable{
	{id: "tls-weak-key", version: 3, why: "needs a completed tls-acceptance exchange; none committed this batch"},
	{id: "zone-removal", version: 1, why: "needs a fresh zone file; the upload aged into a gap"},
}

var devCoverageStaleZones = []devCoverageStaleZone{
	{zone: "internal.acmecorp.io", age: "2 re-supply intervals"},
}

// coverageFixtureData assembles the render data map coveragePage passes to the frozen
// coverage.tmpl in a VERGE_DEV build. It stamps the chrome + design-token holes, then either the
// full pinned fixture corpus or — when a preceding GET /dev/seed/empty-authed set the
// consume-once empty flag — the empty estate (every region nil, so the tmpl draws its empty
// states and no stale callout), reading-and-clearing the flag so a later "default" capture (which
// applies no seed) renders the full corpus again. The address-scope Pct is computed here with the
// same coveragePct arithmetic render-goldens replicates, so golden and candidate agree.
func (s *server) coverageFixtureData(acct db.Account) map[string]any {
	data := map[string]any{
		"Title": "Coverage", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "coverage", "DesignTokens": true,
	}

	s.coverageMu.Lock()
	empty := s.coverageEmptyOnce
	s.coverageEmptyOnce = false
	s.coverageMu.Unlock()
	if empty {
		data["Meters"] = []coverageMeterView(nil)
		data["Messages"] = []coverageMessageView(nil)
		data["Gaps"] = []coverageGapView(nil)
		data["Unevaluable"] = []unevaluableRuleView(nil)
		data["StaleZones"] = []coverageStaleZoneView(nil)
		return data
	}

	meters := make([]coverageMeterView, 0, len(devCoverageMeters))
	for _, m := range devCoverageMeters {
		mv := coverageMeterView{Label: m.label, Counted: strconv.Itoa(m.counted), Unit: m.unit, Detail: m.detail}
		if m.total != nil {
			t := strconv.Itoa(*m.total)
			mv.Total = &t
			mv.Pct = coveragePct(m.counted, *m.total)
		}
		meters = append(meters, mv)
	}
	messages := make([]coverageMessageView, 0, len(devCoverageMessages))
	for _, m := range devCoverageMessages {
		messages = append(messages, coverageMessageView{
			Kind: m.kind, Badge: m.badge, Bound: m.bound, Subject: m.subject, Text: m.text, When: m.when, ISO: m.iso,
		})
	}
	gaps := make([]coverageGapView, 0, len(devCoverageGaps))
	for _, g := range devCoverageGaps {
		gaps = append(gaps, coverageGapView{Subject: g.subject, Gap: g.gap, Expected: g.expected, Since: g.since})
	}
	unevaluable := make([]unevaluableRuleView, 0, len(devCoverageUnevaluables))
	for _, u := range devCoverageUnevaluables {
		unevaluable = append(unevaluable, unevaluableRuleView{ID: u.id, Version: strconv.Itoa(u.version), Why: u.why})
	}
	stale := make([]coverageStaleZoneView, 0, len(devCoverageStaleZones))
	for _, z := range devCoverageStaleZones {
		stale = append(stale, coverageStaleZoneView{Zone: z.zone, Age: z.age})
	}

	data["Meters"] = meters
	data["Messages"] = messages
	data["Gaps"] = gaps
	data["Unevaluable"] = unevaluable
	data["StaleZones"] = stale
	return data
}

// --- screen 9: RunDetail fixture (package v3.8.0, WORK-ORDER-7-9-BATCH2.md) -----------------
//
// The RunDetail screen (#565/#566) renders inside the full app chrome and reads inside a real
// admin session (minted by the harness). Its VIEW corpus — the header identity + batch status,
// the four-stage pipeline, the 7-line batch log (one warn, one error rendered as colored text
// per #20e), the Outcome card (7 transitions · 3 new signals, the #20a batch join), the nullable
// ap-south-1 degraded callout, the 5 run parameters and the 3-vantage health list — is the
// design curated fixture, not a live-queue read: the exact figures ("3m 12s", "7"/"3", the log
// copy, the per-vantage latencies) cannot be reconstructed from the live derivations without
// fabricating domain data, which SPEC-CHANGE forbids. So, exactly as the Exposure/Coverage
// screens pin their dev fixture and serve it under devMode, runPage serves the pinned
// fixtures.json → rundetail slice below when s.devMode, and TestRunDetailFixtureMatchesPackage
// folds every value back through the frozen package — the byte-exactness gate before the pixels.
// The MISSING id (1408) is served by renderMissingRun (error.tmpl's missing-run kind, #20d),
// proving the run-missing route. All of it is VERGE_DEV-only; a real deployment renders the honest
// live reads + batch join in scans.go runPage/joinRunOutcome instead.

const (
	devRunDetailID = "1407"

	devRunTitle       = "2026-08-22T14:00Z"
	devRunStatus      = "complete"
	devRunScope       = "all scopes"
	devRunMeta        = "standard profile · 3 vantages · 3m 12s"
	devRunActive      = false
	devRunTransitions = "7"
	devRunNewSignals  = "3"

	// The nullable degraded callout (#20): ap-south-1 missed 2 of 3 checks this batch.
	devRunDegradedVantage = "ap-south-1"
	devRunDegradedDetail  = "missed 2 of 3 checks"
)

var devRunStages = []runStage{
	{Num: 1, Title: "Resolve", Detail: "dns + zone + CT · 1,284 names", Done: true, Current: false, Last: false},
	{Num: 2, Title: "Probe", Detail: "reachability from 3 vantages", Done: true, Current: false, Last: false},
	{Num: 3, Title: "Census", Detail: "top 1,000 tcp · 62 addresses", Done: true, Current: false, Last: false},
	{Num: 4, Title: "Diff", Detail: "against 08:00Z · 7 transitions", Done: true, Current: false, Last: true},
}

// devRunLog pins fixtures.json → rundetail.log in authored order: seven batch-log lines, one
// warn and one error (rendered as colored text per LogViewer.jsx / #20e), the rest unleveled.
var devRunLog = []runLogLine{
	{Tag: "14:00:02", Level: "", Text: "batch started · 214 subjects · 3 vantages"},
	{Tag: "14:00:09", Level: "", Text: "dns sweep · acmecorp.io · 1,284 names"},
	{Tag: "14:00:41", Level: "warn", Text: "vantage ap-south-1 missed check (2/3)"},
	{Tag: "14:01:12", Level: "", Text: "tls-acceptance · vpn.acmecorp.io:443"},
	{Tag: "14:02:03", Level: "error", Text: "connect refused · 203.0.113.44:22"},
	{Tag: "14:02:31", Level: "", Text: "port census · 62 addresses · top 1,000 tcp"},
	{Tag: "14:03:14", Level: "", Text: "diff against 08:00Z · 7 transitions · 3 signals raised"},
}

var devRunParams = []runKV{
	{K: "Profile", V: "standard"},
	{K: "Cadence", V: "daily · 08:00 + 14:00"},
	{K: "Subjects", V: "214"},
	{K: "Address cap", V: "1,024"},
	{K: "Connect timeout", V: "800ms"},
}

var devRunVantages = []runVantage{
	{Name: "eu-west-1", Latency: "34ms", Status: "ok"},
	{Name: "us-east-2", Latency: "51ms", Status: "ok"},
	{Name: "ap-south-1", Latency: "—", Status: "degraded"},
}

// runDetailFixtureData assembles the render data map runPage passes to the frozen rundetail.tmpl
// in a VERGE_DEV build. It stamps the chrome + design-token holes, then the full pinned run view —
// the header, the four done stages, the seven-line log, the Outcome batch join (7 transitions · 3
// new signals, fed as the strings the join renders), the nullable degraded callout, the five params
// and the three vantages. The .Run view is the SAME runView struct the live buildRunView emits, so
// golden and candidate agree byte-for-byte. All VERGE_DEV-only.
func (s *server) runDetailFixtureData(acct db.Account) map[string]any {
	view := runView{
		ID:          1407,
		Title:       devRunTitle,
		Status:      devRunStatus,
		Scope:       devRunScope,
		Meta:        devRunMeta,
		Transitions: devRunTransitions,
		NewSignals:  devRunNewSignals,
		Active:      devRunActive,
		Stages:      devRunStages,
		Log:         devRunLog,
		Params:      devRunParams,
		Vantages:    devRunVantages,
		Degraded:    &runDegraded{Vantage: devRunDegradedVantage, Detail: devRunDegradedDetail},
	}
	return map[string]any{
		"Title": "batch " + view.Title, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive":    "drift",
		"DesignTokens": true,
		"Run":          view,
	}
}

// --- running run 1409: live scan output (DF-F3 / DF-F3b, id split ruled v3.17.0) --------------
//
// The Settings scans active dispatch is id 1409 (fixtures.json → settings.scans.active[0],
// href /runs/1409), each of whose jobs links here as /runs/1409?job={id}. This is the run-page
// side: a running dispatch whose drill-in renders the LIVE pulse + the head meta-refresh tail
// while in flight (DF-F3), and — with ?job= — the server-side per-job log filter + loghead chip
// (DF-F3b). The view is built from the same pure folds the live path uses (runStages / runLog /
// runVantages / applyJobFilter / runRefresh), so a real running dispatch and this fixture render
// byte-for-byte alike. It is VERGE_DEV-only. SPEC-CHANGE #35 (ruled v3.17.0) split the id
// collision the interim parked this behind: the Settings active dispatch moved to 1409, distinct
// from the error screen's missing-run demo (which keeps 1408), so runPage now routes /runs/1409
// here and the rundetail·running state gets its own G2 golden this round.
const devRunningRunID = "1409"

var devRunningRunJobs = []jobView{
	{ID: 912, Kind: "dns-sweep", State: "done", Attempt: 1, MaxAttempts: 3, Vantage: "eu-west-1", Batch: "1407"},
	{ID: 913, Kind: "reachability", State: "done", Attempt: 1, MaxAttempts: 3, Vantage: "eu-west-1", Batch: "1407"},
	{ID: 914, Kind: "reachability", State: "done", Attempt: 1, MaxAttempts: 3, Vantage: "us-east-2", Batch: "1407"},
	{ID: 915, Kind: "reachability", State: "ready", Attempt: 2, MaxAttempts: 3, Retrying: true, Vantage: "ap-south-1"},
	{ID: 916, Kind: "port-census", State: "running", Attempt: 1, MaxAttempts: 3, Vantage: "eu-west-1"},
	{ID: 917, Kind: "tls-acceptance", State: "done", Attempt: 1, MaxAttempts: 3, Vantage: "eu-west-1", Batch: "1407"},
}

func (s *server) runningRunFixtureData(acct db.Account, jobParam, bareHref string) map[string]any {
	jobs := devRunningRunJobs
	view := runView{
		ID:          1409,
		Title:       "2026-08-22T14:00Z",
		Status:      runStatusLabel(true, 0, ""), // in flight → running
		Scope:       "all scopes",
		Meta:        "standard profile · 3 vantages",
		Transitions: "—",
		NewSignals:  "—",
		Active:      true,
		Stages:      runStages(jobs),
		Log:         runLog(jobs),
		Vantages:    runVantages(jobs),
		Params: []runKV{
			{K: "Profile", V: "standard"},
			{K: "Cadence", V: "daily · 08:00 + 14:00"},
			{K: "Dispatched", V: "2026-08-22 14:00 UTC"},
			{K: "Jobs", V: "6"},
			{K: "Vantages", V: "3"},
		},
	}
	applyJobFilter(&view, jobParam, bareHref, jobs)
	linkRunLog(&view, bareHref)
	return map[string]any{
		"Title": "batch " + view.Title, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive":    "drift",
		"DesignTokens": true,
		"Refresh":      runRefresh(view.Status),
		"Run":          view,
	}
}

// devCoverageSeedEmpty is the GET /dev/seed/empty-authed handler (VERGE_DEV only): it realizes
// states.json coverage seed:"empty-authed" by arming the consume-once empty flag coveragePage
// reads, so the NEXT /coverage render serves the empty estate. Unlike /dev/seed/empty (Setup) it
// touches no table — the authed admin session Coverage states run under is preserved. A 200 with a
// tiny body (not 204) so the harness page.goto completes the navigation, mirroring the Setup route.
func (s *server) devCoverageSeedEmpty(w http.ResponseWriter, r *http.Request) {
	s.coverageMu.Lock()
	s.coverageEmptyOnce = true
	s.coverageMu.Unlock()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

// --- screen 8: Drift fixture (package v3.8.0, WORK-ORDER-7-9-BATCH2.md) ------------------
//
// The Drift screen (#562/#563) renders inside the full app chrome. Its VIEW corpus — the
// batch-grouped transition timeline (3 groups / 7 events, one carrying a before/after diff,
// the oldest group collapsed), the movement tally, the +2 transition delta and the range
// picker's preset vocabulary — is the design curated fixture, not a live-estate read: a
// transition is derived on read from consecutive span adjacencies (ADR-0007), and the exact
// events, subjects, diffs and counts cannot be reconstructed from the live derivations
// without fabricating domain data, which SPEC-CHANGE forbids. So, exactly as the Exposure /
// Coverage screens pin their dev fixture and serve it under devMode, driftPage serves the
// pinned fixtures.json → drift slice below when s.devMode, and TestDriftFixtureMatchesPackage
// folds every value back through the frozen package — the byte-exactness gate before the
// pixels. The .Periods and .Kinds holes are fed from driftPeriods()/driftKinds() (the same
// vocabulary the live path renders), which that test also pins to the fixture. All of it is
// VERGE_DEV-only; a real deployment renders the honest live feed in drift.go driftPage instead.

const (
	devDriftPeriod          = "7d"
	devDriftPeriodLabel     = "Last 7d"
	devDriftHasEvents       = true
	devDriftTruncated       = false
	devDriftBatchID         = "1407"
	devDriftBatchLabel      = "2026-08-22T14:00Z"
	devDriftTransitionCount = 7
	devDriftTransitionDelta = "+2"
)

var devDriftMovement = driftMovement{
	"appeared": 1, "revealed": 1, "withdrawn": 1, "descoped": 1, "returned": 1, "changed": 2,
}

type devDriftDiffLine struct {
	typ  string
	text string
}
type devDriftEvent struct {
	change  string
	family  string
	subject string
	detail  string
	time    string
	reason  string
	diff    []devDriftDiffLine
}
type devDriftGroup struct {
	label     string
	meta      string
	collapsed bool
	events    []devDriftEvent
}

var devDriftGroups = []devDriftGroup{
	{
		label: "2026-08-22T14:00Z", meta: "full scan · 3 vantages", collapsed: false,
		events: []devDriftEvent{
			{change: "changed", family: "change", subject: "api.acmecorp.io :443", detail: "service banner", time: "4m", diff: []devDriftDiffLine{
				{typ: "remove", text: "nginx/1.24.0"},
				{typ: "add", text: "nginx/1.25.0 (CVE-2026-1187)"},
			}},
			{change: "appeared", family: "gain", subject: "staging-5.acmecorp.io", detail: "name · first seen via certificate transparency", time: "8m"},
			{change: "withdrawn", family: "loss", subject: ":8080 http-alt on edge-gw-03.acmecorp.io", detail: "service", reason: "closed since last batch", time: "9m"},
		},
	},
	{
		label: "2026-08-22T08:00Z", meta: "full scan · 3 vantages", collapsed: false,
		events: []devDriftEvent{
			{change: "returned", family: "gain", subject: "mail.acmecorp.io :587", detail: "service · absent for 2 batches", time: "6h"},
			{change: "revealed", family: "gain", subject: "203.0.113.77", detail: "address · custody extension widened the aperture", time: "6h"},
		},
	},
	{
		label: "2026-08-21T14:00Z", meta: "full scan · 2 vantages", collapsed: true,
		events: []devDriftEvent{
			{change: "descoped", family: "loss", subject: "old-blog.acmecorp.io", detail: "name", reason: "operator excluded subtree", time: "1d"},
			{change: "changed", family: "change", subject: "www.acmecorp.io :443", detail: "certificate issuer", time: "1d"},
		},
	},
}

// driftFixtureData assembles the render data map driftPage passes to the frozen drift.tmpl in a
// VERGE_DEV build. It stamps the chrome + design-token holes, the range-picker vocabulary
// (driftPeriods) and change vocabulary (driftKinds), then the pinned groups (each with its
// Collapsed flag and events), the movement tally, and the trigger + tally scalars — so the
// cropped `main` is byte-identical to what render-goldens composes statically from the same
// fixture. The tmpl's own JS drives the kind-toggle / group-collapse / range-popover
// interactions client-side over this full feed.
func (s *server) driftFixtureData(acct db.Account) map[string]any {
	groups := make([]driftBatch, 0, len(devDriftGroups))
	for _, g := range devDriftGroups {
		events := make([]driftEvent, 0, len(g.events))
		for _, e := range g.events {
			var diff []driftDiffLine
			for _, d := range e.diff {
				diff = append(diff, driftDiffLine{Type: d.typ, Text: d.text})
			}
			events = append(events, driftEvent{
				Change: e.change, Family: e.family, Subject: e.subject,
				Detail: e.detail, Time: e.time, Reason: e.reason, Diff: diff,
			})
		}
		groups = append(groups, driftBatch{Label: g.label, Meta: g.meta, Collapsed: g.collapsed, Events: events})
	}

	movement := driftMovement{}
	for k, v := range devDriftMovement {
		movement[k] = v
	}

	return map[string]any{
		"Title": "Drift", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "drift", "DesignTokens": true,
		"Kinds":           driftKinds(),
		"Periods":         driftPeriods(),
		"Period":          devDriftPeriod,
		"PeriodLabel":     devDriftPeriodLabel,
		"Groups":          groups,
		"Movement":        movement,
		"HasEvents":       devDriftHasEvents,
		"Truncated":       devDriftTruncated,
		"FeedLimit":       driftFeedLimit,
		"BatchID":         devDriftBatchID,
		"BatchLabel":      devDriftBatchLabel,
		"TransitionCount": devDriftTransitionCount,
		"TransitionDelta": devDriftTransitionDelta,
	}
}

// --- screen 10: Scope fixture (package v3.9.0, WORK-ORDER-10-12-BATCH3.md) ------------------
//
// The Scope screen (#574) renders inside the full app chrome. Its VIEW corpus — the two seeds,
// the custody census (62 addresses on acmecorp.io), the one supplied zone file with its aging
// label, the seven-leaf name tree, the three coverage messages, the three proposals and two
// exclusions — is the design's curated fixture, not a live-estate read: the exact strings, the
// authored ordering, and the DERIVED figures (the census numerator, the "ages into a gap in 7d"
// countdown, the per-leaf severities) cannot be reconstructed from the live derivations without
// fabricating domain data, which SPEC-CHANGE forbids. So, exactly as the Coverage/Exposure
// screens pin their dev fixture and serve it under devMode, seedsPage serves the pinned
// fixtures.json scope slice below when s.devMode, and TestScopeFixtureMatchesPackage folds
// every value back through the frozen package — the byte-exactness gate before the pixels. The
// two non-default golden states (refusal, exclusion-preview) ride the SAME fixture with an
// overlay the POST handlers set (declareSeed over-cap sets .Refusal + .FormError; previewExclusion
// sets .ExclPreview), mirroring the states.json capture scripts. All of it is VERGE_DEV-only; a
// real deployment renders the honest live projection in seeds.go renderSeeds instead.

const (
	devScopeAddressCap = 1024

	devScopeZoneIntervalDays = "30"

	devScopeRefusalPost      = "203.0.113.0/20"
	devScopeRefusalInput     = "203.0.113.0/20"
	devScopeRefusalReason    = "Spans 4,096 addresses — the cap is 1,024 per scope."
	devScopeRefusalReachable = "203.0.113.0/22"
	devScopeRefusalFormError = "Refused — over the 1,024-address cap."

	devScopeExclPreviewKind     = "subtree"
	devScopeExclPreviewValue    = "staging-4.acmecorp.io"
	devScopeExclPreviewFires    = true
	devScopeExclPreviewHeadline = "1 name and 2 subjects would close as descoped in the next batch."
	devScopeExclPreviewLoss     = "staging-4.acmecorp.io and the service spans it anchors leave the estate; their history stays readable."

	devScopeOrgQuery  = "Acme Corporation"
	devScopeOrgNotice = "3 registries answered — 1 new proposal for \"Acme Corporation\". RIPE paths are off until you accept their terms."
)

type devScopeSeedRow struct {
	ID        string
	Anchor    string
	Scope     string
	IsAddress bool
}

var devScopeSeeds = []devScopeSeedRow{
	{ID: "s1", Anchor: "acmecorp-io", Scope: "acmecorp.io", IsAddress: false},
	{ID: "s2", Anchor: "203-0-113-0-24", Scope: "203.0.113.0/24", IsAddress: true},
}

// devScopeCustodyRow mirrors one fixtures.json scope.custody_scopes entry (#21b).
type devScopeCustodyRow struct {
	ID               string
	Scope            string
	CustodyExtension bool
	Census           int
}

var devScopeCustody = []devScopeCustodyRow{
	{ID: "s1", Scope: "acmecorp.io", CustodyExtension: true, Census: 62},
}

// devScopeZoneRow mirrors one fixtures.json scope.zone_scopes entry (#21c).
type devScopeZoneRow struct {
	ID            string
	Domain        string
	HasFile       bool
	SuppliedAt    string
	IntervalLabel string
	AgingLabel    string
}

var devScopeZones = []devScopeZoneRow{
	{ID: "s1", Domain: "acmecorp.io", HasFile: true, SuppliedAt: "2026-07-30", IntervalLabel: "monthly", AgingLabel: "ages into a gap in 7d"},
}

type devScopeTreeLeaf struct {
	Label string
	Sev   string
}

type devScopeTreeRoot struct {
	Label    string
	Count    int
	Sev      string
	Children []devScopeTreeLeaf
}

var devScopeNameTree = []devScopeTreeRoot{
	{Label: "acmecorp.io", Count: 10, Sev: "medium", Children: []devScopeTreeLeaf{
		{Label: "www", Sev: "low"},
		{Label: "api", Sev: "high"},
		{Label: "vpn", Sev: "critical"},
		{Label: "edge-gw-03", Sev: "critical"},
		{Label: "grafana", Sev: "high"},
		{Label: "mail", Sev: ""},
		{Label: "staging-4", Sev: "info"},
	}},
}

type devScopeCovMsg struct {
	Kind    string
	Badge   string
	Bound   string
	Subject string
	Text    string
	When    string
	ISO     string
}

var devScopeCoverageMsgs = []devScopeCovMsg{
	{Kind: "gap", Badge: "no address", Subject: "old-blog.acmecorp.io", Text: "Expected a resolution; none observed for 3 checks.", When: "2h", ISO: "2026-08-22T12:20:04Z"},
	{Kind: "stale", Badge: "stale", Bound: "9d", Subject: "edge-gw-03.acmecorp.io", Text: "Last full service observation is older than the scan cadence.", When: "9d", ISO: "2026-08-13T04:44:19Z"},
	{Kind: "silent", Badge: "no reports", Subject: "dc-fra-01", Text: "Vantage stopped reporting mid-batch; open spans are not evaluable.", When: "41m", ISO: "2026-08-22T13:41:02Z"},
}

type devScopeProposalRow struct {
	ID     string
	Value  string
	Kind   string
	Source string
}

var devScopeProposals = []devScopeProposalRow{
	{ID: "p1", Value: "acme-corp.net", Kind: "name", Source: "registrar match"},
	{ID: "p2", Value: "acmecorp.dev", Kind: "name", Source: "certificate SAN"},
	{ID: "p3", Value: "198.51.100.0/26", Kind: "range", Source: "announced by AS64500"},
}

type devScopeExclusionRow struct {
	ID    string
	Kind  string
	Value string
}

var devScopeExclusions = []devScopeExclusionRow{
	{ID: "x1", Kind: "subtree", Value: "old-blog.acmecorp.io"},
	{ID: "x2", Kind: "address", Value: "203.0.113.128/25"},
}

type scopeOverlay struct {
	formScope   string
	formError   string
	refusals    []refusalView
	exclKind    string
	exclValue   string
	exclPreview map[string]any
	seedConfirm map[string]any
}

// scopeFixtureData assembles the render data map seedsPage passes to the frozen scope.tmpl in a
// VERGE_DEV build. It stamps the chrome + design-token holes, then the pinned fixtures.json scope
// slice (seeds, custody, zone, name tree, coverage messages, proposals, exclusions) in authored
// order, and finally the overlay (refusal / exclusion-preview) so both non-default golden states
// ride the same corpus. render-goldens composes the identical map statically, so golden and
// candidate agree byte-for-byte.
func (s *server) scopeFixtureData(acct db.Account, ov scopeOverlay) map[string]any {
	seeds := make([]map[string]any, 0, len(devScopeSeeds))
	for _, r := range devScopeSeeds {
		seeds = append(seeds, map[string]any{"ID": r.ID, "Anchor": r.Anchor, "Scope": r.Scope, "IsAddress": r.IsAddress})
	}
	custody := make([]map[string]any, 0, len(devScopeCustody))
	for _, c := range devScopeCustody {
		custody = append(custody, map[string]any{"ID": c.ID, "Scope": c.Scope, "CustodyExtension": c.CustodyExtension, "Census": c.Census})
	}
	zones := make([]map[string]any, 0, len(devScopeZones))
	for _, z := range devScopeZones {
		zones = append(zones, map[string]any{"ID": z.ID, "Domain": z.Domain, "HasFile": z.HasFile, "SuppliedAt": z.SuppliedAt, "IntervalLabel": z.IntervalLabel, "AgingLabel": z.AgingLabel})
	}
	tree := make([]map[string]any, 0, len(devScopeNameTree))
	for _, root := range devScopeNameTree {
		kids := make([]map[string]any, 0, len(root.Children))
		for _, leaf := range root.Children {
			kids = append(kids, map[string]any{"Label": leaf.Label, "Sev": leaf.Sev})
		}
		tree = append(tree, map[string]any{"Label": root.Label, "Count": root.Count, "Sev": root.Sev, "Children": kids})
	}
	msgs := make([]map[string]any, 0, len(devScopeCoverageMsgs))
	for _, m := range devScopeCoverageMsgs {
		msgs = append(msgs, map[string]any{"Kind": m.Kind, "Badge": m.Badge, "Bound": m.Bound, "Subject": m.Subject, "Text": m.Text, "When": m.When, "ISO": m.ISO})
	}
	proposals := make([]map[string]any, 0, len(devScopeProposals))
	for _, p := range devScopeProposals {
		proposals = append(proposals, map[string]any{"ID": p.ID, "Value": p.Value, "Kind": p.Kind, "Source": p.Source})
	}
	exclusions := make([]map[string]any, 0, len(devScopeExclusions))
	for _, e := range devScopeExclusions {
		exclusions = append(exclusions, map[string]any{"ID": e.ID, "Kind": e.Kind, "Value": e.Value})
	}

	data := map[string]any{
		"Title": "Scope", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "scope", "DesignTokens": true,
		"AddressCap":       devScopeAddressCap,
		"Seeds":            seeds,
		"FormScope":        ov.formScope,
		"FormError":        ov.formError,
		"CustodyScopes":    custody,
		"ZoneScopes":       zones,
		"ZoneIntervalDays": devScopeZoneIntervalDays,
		"NameTree":         tree,
		"CoverageMsgs":     msgs,
		"Proposals":        proposals,
		"OrgQuery":         "",
		"Exclusions":       exclusions,
		"ExclKind":         ov.exclKind,
		"ExclValue":        ov.exclValue,
	}
	if len(ov.refusals) > 0 {
		data["Refusals"] = ov.refusals
	}
	if ov.exclPreview != nil {
		data["ExclPreview"] = ov.exclPreview
	}
	if ov.seedConfirm != nil {
		data["SeedConfirm"] = ov.seedConfirm
	}
	return data
}

// scopeFixtureDataRefusal is the "refusal" golden's render (states.json): the pinned corpus plus
// the RefusalCallout the over-cap seed form raises. It builds the callout from the SAME derivation
// declareSeed uses in production (refusalOverCap), so the dev render and the live refusal agree.
func (s *server) scopeFixtureDataRefusal(acct db.Account, value string) map[string]any {
	if value == "" {
		value = devScopeRefusalPost
	}
	ov := scopeOverlay{formScope: value}
	if isAddressValue(value) {
		if raw, err := netip.ParsePrefix(cidrForm(value)); err == nil && !seed.WithinCap(raw, devScopeAddressCap) {
			ref := refusalOverCap(value, raw, devScopeAddressCap)
			ov.refusals = []refusalView{ref}
			ov.formError = overCapFormError(devScopeAddressCap)
		}
	}
	return s.scopeFixtureData(acct, ov)
}

// scopeFixtureDataConfirm is the chip-remove confirm step over the pinned corpus (#1046). It
// names the FIRST fixture seed and carries NO receipt: the fixture is a curated corpus rather
// than an estate, so there is no span to count, and a fabricated headline would state a
// withdrawal nothing measured. A zero count renders the same way in production.
// It names the chip the operator actually clicked. The fixture ids are the strings the
// chips render, so the posted id is matched against them directly; an id belonging to
// no chip renders no confirm state, which is what a withdrawn Seed does in production.
func (s *server) scopeFixtureDataConfirm(acct db.Account, id string) map[string]any {
	for _, row := range devScopeSeeds {
		if row.ID == id {
			return s.scopeFixtureData(acct, scopeOverlay{
				seedConfirm: map[string]any{"ID": row.ID, "Scope": row.Scope, "Fires": false},
			})
		}
	}
	return s.scopeFixtureData(acct, scopeOverlay{})
}

func (s *server) scopeFixtureDataPreview(acct db.Account) map[string]any {
	return s.scopeFixtureData(acct, scopeOverlay{
		exclKind:  devScopeExclPreviewKind,
		exclValue: devScopeExclPreviewValue,
		exclPreview: map[string]any{
			"Fires":    devScopeExclPreviewFires,
			"Headline": devScopeExclPreviewHeadline,
			"Loss":     devScopeExclPreviewLoss,
		},
	})
}

// --- screen 12: Signals fixture (package v3.9.0, WORK-ORDER-10-12-BATCH3.md) ----------------
//
// The Signals screen (#576) renders inside the full app chrome. Its VIEW corpus — the ten open
// rows (of 47 · page 1 of 5), the three withdrawn rows, the annotations on SIG-1027/SIG-1024, the
// drift diffs on SIG-1042/SIG-1036, the per-rule refs and the drawer's rule metadata (tags, CVE,
// description prose, rule id/version, the detecting vantage) — is the design's curated fixture,
// not a live-estate read: the exact rows, the 47-of-47 open count, the authored severity order and
// the rule metadata the corpus does not carry cannot be reconstructed from the live derivations
// without fabricating domain data, which SPEC-CHANGE forbids. So, exactly as the Scope/Exposure
// screens pin their dev fixture and serve it under devMode, signalsPage serves the pinned
// fixtures.json signals slice below when s.devMode, and TestSignalsFixtureMatchesPackage folds
// every value back through the frozen package — the byte-exactness gate before the pixels. The
// drawer / descope / withdrawn / menu-open golden states ride the SAME fixture, selected by the
// tab / view / descope query string (states.json). All of it is VERGE_DEV-only; a real deployment
// renders the honest live projection in signals.go renderSignals instead.

const (
	devSignalsDetectedBy = "vantage eu-west-1"

	devSignalsOpenCount = 47
	devSignalsShown     = 10
	devSignalsPageCount = 5
	devSignalsPageInfo  = "1–10 of 47"

	devSignalsDiscovered = "2026-08-12"
)

var devSignalsSevOptions = []string{"All severities", "Critical", "High", "Medium", "Low", "Info"}

type devSignalRow struct {
	ID          string
	Severity    string
	SevLabel    string
	Title       string
	Asset       string
	IP          string
	Port        string
	Seen        string
	First       string
	Last        string
	CVE         string
	Tags        []string
	Desc        string
	RuleID      string
	RuleVersion string
}

var devSignalsOpen = []devSignalRow{
	{ID: "SIG-1042", Severity: "critical", SevLabel: "Critical", Title: "VNC exposed to internet", Asset: "edge-gw-03.acmecorp.io", IP: "203.0.113.7", Port: ":5900", Seen: "4m", First: "2026-08-22T13:58:02Z", Last: "2026-08-22T14:02:11Z", Tags: []string{"vnc", "remote-access"}, Desc: "A VNC service answered on 203.0.113.7:5900 without transport encryption. Close the port or restrict it to your VPN.", RuleID: "vnc-exposure", RuleVersion: "3"},
	{ID: "SIG-1041", Severity: "critical", SevLabel: "Critical", Title: "TLS certificate expired", Asset: "vpn.acmecorp.io", IP: "203.0.113.12", Port: ":443", Seen: "12m", First: "2026-08-21T00:00:00Z", Last: "2026-08-22T13:54:40Z", Tags: []string{"tls", "cert"}, Desc: "The certificate served on :443 expired 2026-08-21. Clients may accept downgraded or spoofed connections. Renew and redeploy.", RuleID: "tls-acceptance", RuleVersion: "2"},
	{ID: "SIG-1039", Severity: "high", SevLabel: "High", Title: "Admin panel reachable", Asset: "grafana.acmecorp.io", IP: "203.0.113.31", Port: ":3000", Seen: "26m", First: "2026-08-22T11:20:19Z", Last: "2026-08-22T13:40:08Z", Tags: []string{"grafana", "admin"}, Desc: "A Grafana login page is reachable from the internet. Put it behind SSO or your VPN.", RuleID: "dns-mail-policy", RuleVersion: "1"},
	{ID: "SIG-1036", Severity: "high", SevLabel: "High", Title: "Outdated nginx", Asset: "api.acmecorp.io", IP: "203.0.113.9", Port: ":443", Seen: "1h", First: "2026-08-22T09:03:55Z", Last: "2026-08-22T13:02:33Z", CVE: "CVE-2026-1187", Tags: []string{"nginx/1.25.0"}, Desc: "nginx/1.25.0 matches CVE-2026-1187 (request smuggling). Upgrade to 1.25.4 or later.", RuleID: "svc-exposure", RuleVersion: "3"},
	{ID: "SIG-1034", Severity: "high", SevLabel: "High", Title: "SSH password auth enabled", Asset: "build-07.acmecorp.io", IP: "203.0.113.44", Port: ":22", Seen: "2h", First: "2026-08-21T22:41:12Z", Last: "2026-08-22T12:11:47Z", Tags: []string{"ssh"}, Desc: "sshd accepts password authentication. Switch to key-only auth and disable PasswordAuthentication.", RuleID: "svc-exposure", RuleVersion: "3"},
	{ID: "SIG-1031", Severity: "medium", SevLabel: "Medium", Title: "Subdomain takeover candidate", Asset: "old-blog.acmecorp.io", IP: "—", Port: "", Seen: "3h", First: "2026-08-22T05:12:00Z", Last: "2026-08-22T11:12:29Z", Tags: []string{"dns", "cname"}, Desc: "CNAME points to an unclaimed pages.example.net project. Claim it or remove the record.", RuleID: "svc-exposure", RuleVersion: "3"},
	{ID: "SIG-1029", Severity: "medium", SevLabel: "Medium", Title: "Directory listing enabled", Asset: "assets.acmecorp.io", IP: "203.0.113.18", Port: ":443", Seen: "5h", First: "2026-08-20T18:30:00Z", Last: "2026-08-22T09:00:10Z", Tags: []string{"http"}, Desc: "Autoindex is on at /uploads/. Disable directory listing or add an index file.", RuleID: "svc-exposure", RuleVersion: "3"},
	{ID: "SIG-1027", Severity: "medium", SevLabel: "Medium", Title: "SPF record missing", Asset: "acmecorp.io", IP: "—", Port: "", Seen: "9h", First: "2026-08-19T07:22:41Z", Last: "2026-08-22T05:02:52Z", Tags: []string{"dns", "email"}, Desc: "No SPF record found. Publish one to reduce spoofed mail from your domain.", RuleID: "svc-exposure", RuleVersion: "3"},
	{ID: "SIG-1024", Severity: "low", SevLabel: "Low", Title: "Verbose server header", Asset: "www.acmecorp.io", IP: "203.0.113.4", Port: ":443", Seen: "1d", First: "2026-08-18T12:00:09Z", Last: "2026-08-21T14:10:00Z", Tags: []string{"http", "header"}, Desc: "Responses disclose exact server and OS versions. Trim the Server header.", RuleID: "svc-exposure", RuleVersion: "3"},
	{ID: "SIG-1022", Severity: "info", SevLabel: "Info", Title: "New subdomain discovered", Asset: "staging-4.acmecorp.io", IP: "203.0.113.61", Port: "", Seen: "3d", First: "2026-08-19T02:12:33Z", Last: "2026-08-19T02:12:33Z", Tags: []string{"discovery"}, Desc: "First seen via certificate transparency. Added to your inventory and scheduled for scanning.", RuleID: "svc-exposure", RuleVersion: "3"},
}

// devSignalsWithdrawn pins fixtures.json signals.withdrawn in authored order: three acceptances
// whose subject has left its rule's population — withdrawn on read, no operator act (ADR-0092).
var devSignalsWithdrawn = []devSignalRow{
	{ID: "SIG-0991", Severity: "medium", SevLabel: "Medium", Title: "Directory listing enabled", Asset: "files.acmecorp.io", IP: "203.0.113.21", Port: ":443", Seen: "6d", First: "2026-07-30T10:11:00Z", Last: "2026-08-16T09:00:00Z", Tags: []string{"http"}, Desc: "Autoindex stopped answering in a later batch. The key is in no current population — withdrawn on read, no operator act.", RuleID: "svc-exposure", RuleVersion: "3"},
	{ID: "SIG-0968", Severity: "high", SevLabel: "High", Title: "Admin panel reachable", Asset: "jenkins.acmecorp.io", IP: "203.0.113.29", Port: ":8080", Seen: "12d", First: "2026-07-18T08:40:22Z", Last: "2026-08-10T13:25:41Z", Tags: []string{"jenkins", "admin"}, Desc: "The service left the population; the world moved and the signal withdrew itself.", RuleID: "svc-exposure", RuleVersion: "3"},
	{ID: "SIG-0944", Severity: "low", SevLabel: "Low", Title: "Verbose server header", Asset: "cdn.acmecorp.io", IP: "203.0.113.66", Port: ":443", Seen: "21d", First: "2026-06-29T12:02:19Z", Last: "2026-08-01T07:12:00Z", Tags: []string{"http", "header"}, Desc: "Header trimmed upstream; key absent from the current population.", RuleID: "svc-exposure", RuleVersion: "3"},
}

type devSignalsAnnotation struct {
	ID     string
	Reason string
}

// devSignalsAnnotations pins fixtures.json signals.annotations: SIG-1027 and SIG-1024 carry an
// operator acceptance (both still open, so both surface on the Annotated tab).
var devSignalsAnnotations = map[string]devSignalsAnnotation{
	"SIG-1027": {ID: "a1", Reason: "Third-party mail provider publishes SPF on our behalf."},
	"SIG-1024": {ID: "a2", Reason: "Public banner is intentional — accepted."},
}

type devSignalsDiffLine struct {
	Type string
	Text string
}
type devSignalsDiff struct {
	Title string
	Lines []devSignalsDiffLine
}

var devSignalsDiffs = map[string]devSignalsDiff{
	"SIG-1042": {Title: "Open ports · drift", Lines: []devSignalsDiffLine{
		{Type: "same", Text: ":443 https nginx/1.25.4"},
		{Type: "add", Text: ":5900 vnc — no transport encryption"},
		{Type: "remove", Text: ":8080 http-alt"},
	}},
	"SIG-1036": {Title: "Service banner · drift", Lines: []devSignalsDiffLine{
		{Type: "remove", Text: "nginx/1.24.0"},
		{Type: "add", Text: "nginx/1.25.0 (CVE-2026-1187)"},
	}},
}

// signalsHistory derives one row's drawer history per fixtures.json signals.history_rule (span-
// derived, newest first): Still present (accent) · Drift detected (warn, mono — only when a diff
// exists) · Signal raised (danger for critical/high else neutral, mono, detail = id) · Asset
// discovered (neutral, mono, detail = asset, fixed instant). render-goldens replicates this exact
// derivation so golden and candidate agree byte-for-byte.
func signalsHistory(row devSignalRow, hasDiff bool) []map[string]any {
	raisedTone := "neutral"
	if row.Severity == "critical" || row.Severity == "high" {
		raisedTone = "danger"
	}
	hist := []map[string]any{
		{"Title": "Still present", "Detail": devSignalsDetectedBy + " re-confirmed", "Time": row.Seen, "Tone": "accent", "Mono": false},
	}
	if hasDiff {
		hist = append(hist, map[string]any{"Title": "Drift detected", "Detail": row.Asset + " changed", "Time": row.Seen, "Tone": "warn", "Mono": true})
	}
	hist = append(hist,
		map[string]any{"Title": "Signal raised", "Detail": row.ID, "Time": row.First, "Tone": raisedTone, "Mono": true},
		map[string]any{"Title": "Asset discovered", "Detail": row.Asset, "Time": devSignalsDiscovered, "Tone": "neutral", "Mono": true},
	)
	return hist
}

func signalsRowMap(row devSignalRow, closeHref string, withdrawn bool) map[string]any {
	return map[string]any{
		"Severity":    row.Severity,
		"SevLabel":    row.SevLabel,
		"Title":       row.Title,
		"Asset":       row.Asset,
		"Port":        row.Port,
		"SigID":       row.ID,
		"Seen":        row.Seen,
		"Last":        row.Last,
		"Withdrawn":   withdrawn,
		"ViewKey":     row.ID,
		"DescopeHref": closeHref + "&descope=" + row.ID,
	}
}

// signalsDrawerMap shapes one fixture row into the full spec drawer (#21j): the rule metadata
// (tags, CVE, description, rule id/version, detecting vantage), the nullable drift diff, the
// annotation state, and the span-derived history.
func signalsDrawerMap(row devSignalRow, withdrawn bool) map[string]any {
	d := map[string]any{
		"Title":       row.Title,
		"Seen":        row.Seen,
		"SigID":       row.ID,
		"Severity":    row.Severity,
		"SevLabel":    row.SevLabel,
		"Withdrawn":   withdrawn,
		"Tags":        row.Tags,
		"CVE":         row.CVE,
		"Desc":        row.Desc,
		"Asset":       row.Asset,
		"IP":          row.IP,
		"RuleID":      row.RuleID,
		"RuleVersion": row.RuleVersion,
		"Port":        row.Port,
		"DetectedBy":  devSignalsDetectedBy,
		"First":       row.First,
		"Last":        row.Last,
	}
	diff, hasDiff := devSignalsDiffs[row.ID]
	if hasDiff {
		lines := make([]map[string]any, 0, len(diff.Lines))
		for _, l := range diff.Lines {
			lines = append(lines, map[string]any{"Type": l.Type, "Text": l.Text})
		}
		d["Diff"] = map[string]any{"Title": diff.Title, "Lines": lines}
	}
	if anno, ok := devSignalsAnnotations[row.ID]; ok {
		d["Annotated"] = true
		d["AnnoID"] = anno.ID
		d["AnnoReason"] = anno.Reason
	}
	d["History"] = signalsHistory(row, hasDiff)
	return d
}

// signalsFixtureData assembles the render data map signalsPage passes to the frozen signals.tmpl in
// a VERGE_DEV build. It reads the tab / view / descope query (states.json drives the golden states
// through it), builds the tab's rows from the pinned fixture, and stamps the open-tab scalars
// (47 open · 10 shown · page 1 of 5) directly rather than deriving them over a 47-row slice — the
// count is the design's. The drawer (?view=) carries the full rule metadata + diff + history; the
// descope dialog (?descope=) resolves the row's asset for the typed-confirm gate. render-goldens
// composes the identical maps statically, so golden and candidate agree byte-for-byte.
func (s *server) signalsFixtureData(acct db.Account, r *http.Request) map[string]any {
	tab := r.URL.Query().Get("tab")
	switch tab {
	case "annotated", "withdrawn":
	default:
		tab = "open"
	}

	closeHref := "/signals?tab=" + tab + "&sort=sev&dir=asc"
	viewPrefix := closeHref + "&view="

	// The union of every fixture row (open + withdrawn), so ?view= / ?descope= resolve regardless
	// of the active tab. withdrawnSet marks which keys draw the withdrawn treatment.
	all := make([]devSignalRow, 0, len(devSignalsOpen)+len(devSignalsWithdrawn))
	all = append(all, devSignalsOpen...)
	all = append(all, devSignalsWithdrawn...)
	withdrawnSet := map[string]bool{}
	for _, w := range devSignalsWithdrawn {
		withdrawnSet[w.ID] = true
	}

	var tabRows []devSignalRow
	switch tab {
	case "withdrawn":
		tabRows = devSignalsWithdrawn
	case "annotated":
		for _, row := range devSignalsOpen {
			if _, ok := devSignalsAnnotations[row.ID]; ok {
				tabRows = append(tabRows, row)
			}
		}
	default:
		tabRows = devSignalsOpen
	}
	rows := make([]map[string]any, 0, len(tabRows))
	for _, row := range tabRows {
		rows = append(rows, signalsRowMap(row, closeHref, withdrawnSet[row.ID]))
	}

	sortHref := func(col string) string {
		nd := "asc"
		if col == "sev" {
			nd = "desc" // sev is the active asc column; its own toggle flips to desc
		}
		return "/signals?tab=" + tab + "&sort=" + col + "&dir=" + nd
	}

	data := map[string]any{
		"Title": "Signals", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "signals", "DesignTokens": true,
		"Tab":            tab,
		"OpenCount":      devSignalsOpenCount,
		"AnnotatedCount": len(devSignalsAnnotations),
		"WithdrawnCount": len(devSignalsWithdrawn),
		"Q":              "",
		"Sev":            "All severities",
		"SevOptions":     devSignalsSevOptions,
		"HasAny":         true,
		"ClearHref":      "/signals?tab=" + tab,
		"HasExport":      true,
		"ExportHref":     "/signals/export?tab=" + tab,
		"AnnoError":      "",
		"ViewPrefix":     viewPrefix,
		"CloseHref":      closeHref,
		"Rows":           rows,
		"Sort": map[string]any{
			"Key": "sev", "Dir": "asc",
			"SevHref": sortHref("sev"), "AssetHref": sortHref("asset"),
			"IDHref": sortHref("id"), "SeenHref": sortHref("seen"),
		},
	}

	// Open-tab pagination: 47 of 47 open, 10 shown on page 1 of 5. The other tabs list every row.
	if tab == "open" {
		data["Shown"] = devSignalsShown
		data["Total"] = devSignalsOpenCount
		data["ShowPagination"] = true
		data["PageInfo"] = devSignalsPageInfo
		data["PrevDisabled"] = true
		data["PrevHref"] = closeHref
		data["NextDisabled"] = false
		data["NextHref"] = closeHref + "&page=2"
		pages := make([]map[string]any, 0, devSignalsPageCount)
		for p := 1; p <= devSignalsPageCount; p++ {
			href := closeHref
			if p > 1 {
				href = closeHref + "&page=" + strconv.Itoa(p)
			}
			pages = append(pages, map[string]any{"Ellipsis": false, "Href": href, "Num": p, "Active": p == 1})
		}
		data["Pages"] = pages
	} else {
		data["Shown"] = len(tabRows)
		data["Total"] = len(tabRows)
		data["ShowPagination"] = false
		data["Pages"] = []map[string]any{}
	}

	// The row drawer (?view=<key>) and the typed-confirm descope dialog (?descope=<key>) resolve
	// against the fixture union, so both open regardless of the active tab.
	byKey := map[string]devSignalRow{}
	for _, row := range all {
		byKey[row.ID] = row
	}
	if selKey := r.URL.Query().Get("view"); selKey != "" {
		if row, ok := byKey[selKey]; ok {
			data["SelKey"] = selKey
			data["Drawer"] = signalsDrawerMap(row, withdrawnSet[selKey])
		} else {
			data["SelKey"] = ""
		}
	} else {
		data["SelKey"] = ""
	}
	if dk := r.URL.Query().Get("descope"); dk != "" {
		if row, ok := byKey[dk]; ok {
			data["Descope"] = map[string]any{"Asset": row.Asset, "CloseHref": closeHref}
		}
	}
	return data
}

// --- screen 11: Dashboard fixture (package v3.9.0, WORK-ORDER-10-12-BATCH3.md) --------------
//
// The Dashboard (`/`, #575) renders inside the full app chrome and reads inside a real admin
// session (minted by the harness). Its VIEW corpus — the header schedule (38m / 5h 22m), the
// five-cell stat band with its vs-last-scan deltas, the by-severity ramp (3/11/18/9/6), the
// two-shape coverage meters (an address scope counted/total 212/256, a name scope census 1,284),
// the silent-zone staleness callout (9d · internal.acmecorp.io), the three vantages
// (ap-south-1 unavailable · 12m) and the most-recent register (the first six signal rows) — is the
// design curated fixture, not a live-estate read: the exact figures ("47", "1,284", "216", the
// deltas and captions) cannot be reconstructed from the live derivations without fabricating domain
// data, which SPEC-CHANGE forbids. So, exactly as the sibling screens pin their dev fixture and serve
// it under devMode, home() serves the pinned fixtures.json dashboard slice below when s.devMode, and
// TestDashboardFixtureMatchesPackage folds every value back through the frozen package — the
// byte-exactness gate before the pixels. The most-recent register reuses the devSignalsOpen rows the
// Signals screen pins (fixtures.json says "first 6 of signals.rows"), so the deep-link ?view=<SIG id>
// resolves in the Signals drawer. All of it is VERGE_DEV-only; a real deployment renders the honest
// live reads in auth.go dashboardData instead.

const devDashScanningVariant = "scanning"

const devDashScanDetail = "214 subjects queued"

var devDashSchedule = map[string]any{"HasLast": true, "LastAgo": "38m", "HasNext": true, "NextIn": "5h 22m"}

var devDashUnavailable = []string{"ap-south-1"}

// devDashStat mirrors one fixtures.json dashboard.stat_band cell. liveWhenScanning is the JSON's
// live_when_scanning flag — the pulse shows only while a scan is running, so Live is set from it AND
// the active `scanning` variant, never on the resting default.
type devDashStat struct {
	label            string
	value            string
	liveWhenScanning bool
	hasDelta         bool
	change           int
	tone             string
	caption          string
}

var devDashStatBand = []devDashStat{
	{label: "Open signals", value: "47", liveWhenScanning: true, hasDelta: true, change: 3, tone: "bad", caption: "vs last scan"},
	{label: "Critical", value: "3", hasDelta: true, change: -1, tone: "good", caption: "1 withdrawn today"},
	{label: "Assets watched", value: "1,284", hasDelta: true, change: 12, tone: "neutral", caption: "8 domains · 3 ranges"},
	{label: "Exposed services", value: "216", hasDelta: true, change: 4, tone: "bad", caption: "across 62 IPs"},
	{label: "Certs expiring ≤30d", value: "9", hasDelta: true, change: -2, tone: "good", caption: "next: 2026-08-29"},
}

var devDashSevBars = []dashSevBar{
	{Sev: "critical", Pct: 17, Count: 3},
	{Sev: "high", Pct: 61, Count: 11},
	{Sev: "medium", Pct: 100, Count: 18},
	{Sev: "low", Pct: 50, Count: 9},
	{Sev: "info", Pct: 33, Count: 6},
}

// dashMeterTotal256 backs the address-scope meter's nullable denominator (package-level so its
// address is stable); the name scope leaves Total nil (a census, the striped bar).
var dashMeterTotal256 = "256"

// devDashboardMeters pins fixtures.json dashboard.coverage_meters in authored order: an ADDRESS scope
// rendering counted/total 212/256 (pct 83, #19c) and a NAME scope census of 1,284 names (no
// denominator, #21e2). Counted/Total are pre-formatted strings the tmpl prints verbatim.
var devDashboardMeters = []coverageMeterView{
	{Label: "203.0.113.0/24", Counted: "212", Total: &dashMeterTotal256, Pct: 83, Unit: "addresses"},
	{Label: "acmecorp.io names", Counted: "1,284", Unit: "names"},
}

var devDashSilentZone = &dashSilentZone{Bound: "9d", Text: "zone transfer for internal.acmecorp.io"}

var devDashVantages = []dashVantageView{
	{Name: "eu-west-1", Latency: "34ms", Avail: "available"},
	{Name: "us-east-2", Latency: "51ms", Avail: "available"},
	{Name: "ap-south-1", Latency: "12m", Avail: "unavailable"},
}

// dashRecentSignals shapes the most-recent register from the first six devSignalsOpen rows (the
// fixture's "first 6 of signals.rows"), each carrying the SevLabel + ViewKey the row + sevbadge read,
// so ?view=<SIG id> deep-links into the Signals drawer.
func dashRecentSignals() []dashRecentSignal {
	out := make([]dashRecentSignal, 0, 6)
	for _, row := range devSignalsOpen[:6] {
		out = append(out, dashRecentSignal{
			Severity: row.Severity, SevLabel: row.SevLabel, Title: row.Title,
			Asset: row.Asset, Port: row.Port, Seen: row.Seen, ViewKey: row.ID,
		})
	}
	return out
}

// dashboardFixtureData assembles the render data map home() passes to the frozen dashboard.tmpl in a
// VERGE_DEV build. It stamps the chrome + design-token holes and the pinned fixture corpus, then reads
// the `scanning` variant (states.json) — which lights .Scanning, the .ScanDetail figure and the first
// cell's live pulse — and the ?probe=dismissed query (the banner-dismissed state). render-goldens
// composes the identical maps statically, so golden and candidate agree byte-for-byte.
func (s *server) dashboardFixtureData(acct db.Account, r *http.Request) map[string]any {
	scanning := r.URL.Query().Get("variant") == devDashScanningVariant
	probeDismissed := r.URL.Query().Get("probe") == "dismissed"

	statBand := make([]dashStat, 0, len(devDashStatBand))
	for _, st := range devDashStatBand {
		statBand = append(statBand, dashStat{
			Label: st.label, Value: st.value, Live: scanning && st.liveWhenScanning,
			HasDelta: st.hasDelta, Change: st.change, Tone: st.tone, Caption: st.caption,
		})
	}

	data := map[string]any{
		"Title": "Dashboard", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "dashboard", "DesignTokens": true,
		"EmptyEstate":    false,
		"ScanSchedule":   devDashSchedule,
		"Scanning":       scanning,
		"Unavailable":    devDashUnavailable,
		"ProbeDismissed": probeDismissed,
		"StatBand":       statBand,
		"HasSignals":     true,
		"SevBars":        devDashSevBars,
		"CoverageMeters": devDashboardMeters,
		"SilentZone":     devDashSilentZone,
		"Vantages":       devDashVantages,
		"RecentSignals":  dashRecentSignals(),
	}
	if scanning {
		data["ScanDetail"] = devDashScanDetail
	}
	return data
}

// --- screen 20: FirstRun fixture (firstrun.tmpl, package v3.12.0, WORK-ORDER-19-20-BATCH6) -----
//
// The empty-estate first-run wrap of `/` (the dashboard.tmpl "home" define wrapping the frozen
// firstrun.tmpl when .EmptyEstate) rides a dev ?variant=empty-estate query (states.json). Its
// VIEW corpus — the four checklist steps, their done/undone reads, and the exact ported step copy
// — is the design's pinned fixture, not a live-estate read: the step-1 detail names a concrete
// declared seed ("acmecorp.io declared · …") the live derivation cannot reconstruct without
// fabricating domain data. So, exactly as the sibling screens pin their dev fixture and serve it
// under devMode, home() serves the pinned fixtures.json firstrun slice below when the variant is
// set. It reads the firstrun slice straight from the embedded package JSON (no hardcoded Go copy to
// drift), and render-goldens reads the SAME bytes and shapes them identically. The production path
// (auth.go firstRunChecklist) builds the same holes from real estate reads.
const devFirstRunVariant = "empty-estate"

// firstRunFixture is the pinned fixtures.json → firstrun slice: the "N of 4 complete" count and the
// four steps with their tmpl holes (Num/Done/Title/Detail and, when HasAction, one of ActionHref or
// ActionPost plus Gated/GateTitle). render-goldens reads the SAME slice and shapes it identically,
// so the cropped `main` is byte-identical to what the seeded server renders.
type firstRunFixture struct {
	FirstRunDone  int `json:"first_run_done"`
	FirstRunSteps []struct {
		Num         int    `json:"num"`
		Done        bool   `json:"done"`
		Title       string `json:"title"`
		Detail      string `json:"detail"`
		HasAction   bool   `json:"has_action"`
		ActionLabel string `json:"action_label"`
		ActionHref  string `json:"action_href"`
		ActionPost  string `json:"action_post"`
		Gated       bool   `json:"gated"`
		GateTitle   string `json:"gate_title"`
	} `json:"first_run_steps"`
}

// loadFirstRunFixture reads the pinned fixtures.json firstrun slice from the embedded design
// package (designfs). A read/parse failure degrades to the zero fixture (an empty checklist)
// rather than 500ing.
func loadFirstRunFixture() firstRunFixture {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return firstRunFixture{}
	}
	var ff struct {
		FirstRun firstRunFixture `json:"firstrun"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return firstRunFixture{}
	}
	return ff.FirstRun
}

// firstRunFixtureData shapes the pinned firstrun slice into the "home" holes for the empty-estate
// wrap: EmptyEstate lit, the done count, and the four steps. render-goldens composes the identical
// map statically (minus Account/IsAdmin, which the golden's stub chrome never reads), so golden and
// candidate agree byte-for-byte.
func (s *server) firstRunFixtureData(acct db.Account) map[string]any {
	fx := loadFirstRunFixture()
	steps := make([]firstRunStep, 0, len(fx.FirstRunSteps))
	for _, st := range fx.FirstRunSteps {
		steps = append(steps, firstRunStep{
			Num: st.Num, Done: st.Done, Title: st.Title, Detail: st.Detail,
			HasAction: st.HasAction, ActionLabel: st.ActionLabel, ActionHref: st.ActionHref,
			ActionPost: st.ActionPost, Gated: st.Gated, GateTitle: st.GateTitle,
		})
	}
	return map[string]any{
		"Title": "Dashboard", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "dashboard", "DesignTokens": true,
		"EmptyEstate":   true,
		"FirstRunDone":  fx.FirstRunDone,
		"FirstRunSteps": steps,
	}
}

// --- screen 13: AssetDetail fixture (package v3.10.0, WORK-ORDER-13-15-BATCH4.md) -------------
//
// The Asset detail (`/asset/{key}`, #581) reads inside the full app chrome and a real admin session
// (minted by the harness). Its VIEW corpus — the header identity (subdomain · Critical · exposed ·
// seen 4m · in scope since 2026-06-14), the three-port census with its joined Service strings, the
// three DNS records, the parsed TLS certificate card (valid · 47d, ECDSA-SHA256, R11, pinned
// fingerprint), the five provenance facts, the one signal-here (SIG-1042 · 4m) and the four drift
// events — is the design's curated fixture, not a live-estate read: the exact strings (the joined
// Service banners, the cert identity, the relative instants) cannot be reconstructed from the live
// derivations without fabricating domain data, which SPEC-CHANGE forbids. So, exactly as the sibling
// screens pin their dev fixture and serve it under devMode, assetPage serves the pinned fixtures.json
// asset slice below when s.devMode, and TestAssetFixtureMatchesPackage folds every value back through
// the frozen package — the byte-exactness gate before the pixels. The production path (subjects.go)
// builds the same holes from real Name-scoped reads (#22a–d); this pins the design's exact bytes.
const devAssetKey = "edge-gw-03.acmecorp.io"

const devAssetCertFingerprint = "SHA256:2b:9e:44:a1:7c:03:d8:f2:61:5b:c9:10:8e:af:72:d4" // #nosec G101 -- a public TLS certificate fingerprint fixture, not a credential

var devAssetPorts = []assetPort{
	{Port: ":443", Service: "https · nginx/1.25.0", Exposure: "exposed", Since: "2026-06-14"},
	{Port: ":5900", Service: "vnc — no transport encryption", Exposure: "exposed", Since: "2026-08-22"},
	{Port: ":22", Service: "ssh · OpenSSH 9.6", Exposure: "firewalled", Since: "2026-06-14"},
}

var devAssetDNS = []assetDNSRow{
	{Type: "A", Value: "203.0.113.7", Seen: "4m"},
	{Type: "AAAA", Value: "2001:db8::7", Seen: "4m"},
	{Type: "TXT", Value: "verge-custody=vg_7f2a91c4", Seen: "6h"},
}

var devAssetCert = &assetCert{
	Name:        "edge-gw-03.acmecorp.io",
	Issuer:      "CN=R11, O=Let's Encrypt",
	Algorithm:   "ECDSA-SHA256",
	NotAfter:    "2026-10-08",
	Label:       "valid · 47d",
	Tone:        "ok",
	Fingerprint: devAssetCertFingerprint,
}

var devAssetProvenance = []assetKV{
	{K: "Seed", V: "acmecorp.io"},
	{K: "Via", V: "CT log → dns sweep"},
	{K: "Vantage", V: "eu-west-1"},
	{K: "Custody", V: "verified · TXT record"},
	{K: "First seen", V: "2026-06-14"},
}

var devAssetSignals = []assetSignal{
	{Severity: "critical", SevLabel: "Critical", Rule: ":5900 vnc — no transport encryption", SigID: "SIG-1042", Time: "4m"},
}

var devAssetDrift = []assetDriftEvent{
	{Change: "changed", Family: "change", Subject: ":443 service banner", Detail: "nginx/1.24.0 → 1.25.0", Time: "4m"},
	{Change: "appeared", Family: "gain", Subject: ":5900 vnc", Detail: "service · new in batch 14:00Z", Time: "4m"},
	{Change: "changed", Family: "change", Subject: "certificate", Detail: "renewed · Let's Encrypt R11", Time: "12d"},
	{Change: "appeared", Family: "gain", Subject: "edge-gw-03.acmecorp.io", Detail: "name · first seen via certificate transparency", Time: "69d"},
}

func devAssetData() assetPageData {
	return assetPageData{
		Key:          devAssetKey,
		Type:         "subdomain",
		Withdrawn:    false,
		Seen:         "4m",
		InScopeSince: "2026-06-14",
		Severity:     "critical",
		SevLabel:     "Critical",
		Exposure:     "exposed",
		Ports:        devAssetPorts,
		DNS:          devAssetDNS,
		Cert:         devAssetCert,
		Provenance:   devAssetProvenance,
		Signals:      devAssetSignals,
		Drift:        devAssetDrift,
	}
}

// assetFixtureData is the render map assetPage passes to the frozen asset.tmpl in a VERGE_DEV build
// (the default golden state). render-goldens composes the identical map statically, so golden and
// candidate agree byte-for-byte.
func (s *server) assetFixtureData(acct db.Account) map[string]any {
	return map[string]any{
		"Title": devAssetKey, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "inventory", "DesignTokens": true,
		"Asset": devAssetData(),
	}
}

// --- screen 14: SubjectDetail fixture (package v3.10.0, WORK-ORDER-13-15-BATCH4.md) ----------
//
// The SubjectDetail screen (#582) serves /subjects/{service,endpoint} from the frozen
// subjectdetail.tmpl. Its VIEW corpus — the two service states (reachable + withdrawn) and the
// endpoint state, each with its citation rail, current facet, span timelines (with the closed-row
// title tooltips), rules-over-subject table, provenance and the Service's signals-here — is the
// design's curated fixture, not a live-estate read: the exact chain hops, the joined timeline
// values, the relative instants and the minted SIG id cannot be reconstructed from the live
// derivations without fabricating domain data, which SPEC-CHANGE forbids. So, exactly as the
// sibling screens pin their dev fixture and serve it under devMode, servicePage / endpointPage
// serve the pinned fixtures.json subjectdetail slices below when s.devMode, and
// TestSubjectDetailFixtureMatchesPackage folds every value back through the frozen package — the
// byte-exactness gate before the pixels. The production path (subjects.go) builds the same holes
// from real subject-scoped reads (#22 wiring); this pins the design's exact bytes. All VERGE_DEV-only.
const (
	devServiceKey          = "203.0.113.7:5900/tcp"
	devServiceWithdrawnKey = "203.0.113.29:8080/tcp"
	devEndpointKey         = "edge-gw-03.acmecorp.io · :443 https"
)

func devServiceData() servicePageData {
	return servicePageData{
		Key:          devServiceKey,
		CopyKey:      "203.0.113.7:5900 tcp",
		Withdrawn:    false,
		Exposure:     "exposed",
		Seen:         "4m",
		InScopeSince: "2026-08-22",
		Citation: []citationHop{
			{Label: "Service", Value: "203.0.113.7:5900/tcp", Detail: "an (address, port, transport) triple"},
			{Label: "Address · cited by a current resolution", Value: "203.0.113.7"},
			{Label: "Name · citing resolution", Value: "edge-gw-03.acmecorp.io", Detail: "A record, seen 4m ago"},
			{Label: "Seed · name scope", Value: "acmecorp.io", Detail: "declared 2026-06-14 · custody verified"},
		},
		CitationTerminated: true,
		Address:            "203.0.113.7",
		Port:               "5900",
		Transport:          "tcp",
		Reach:              "reached",
		Since:              "2026-08-22T14:00Z",
		Timelines: []timelineView{{
			Label:   "reachability",
			Current: &spanView{IsGap: false, Value: "reached", OpenedAt: "2026-08-22T14:00Z"},
			Breaks:  []breakView{{At: "2026-08-01T06:00Z", MovedLeaves: "transport"}},
			Closed: []spanView{
				{Value: "not-reached", OpenedAt: "2026-07-14", OpenedFull: "2026-07-14T06:00Z", ClosedAt: "2026-08-22", ClosedFull: "2026-08-22T14:00Z", Reason: "changed"},
				{IsGap: true, Value: "Gap", OpenedAt: "2026-07-02", OpenedFull: "2026-07-02T06:00Z", ClosedAt: "2026-07-14", ClosedFull: "2026-07-14T06:00Z", Reason: "stopped looking"},
			},
		}},
		Rules: []subjectRule{
			{Rule: "vnc-exposure", Version: "3", Severity: "critical", SevLabel: "Critical", Fired: true},
			{Rule: "tls-acceptance", Version: "2", Severity: "high", SevLabel: "High", Fired: false},
		},
		Provenance: []assetKV{
			{K: "Seed", V: "acmecorp.io"},
			{K: "Via", V: "dns sweep → hot scan"},
			{K: "Vantage", V: "eu-west-1"},
			{K: "First seen", V: "2026-08-22"},
		},
		Signals: []assetSignal{
			{Severity: "critical", SevLabel: "Critical", Rule: ":5900 vnc — no transport encryption", SigID: "SIG-1042", Time: "4m"},
		},
	}
}

func devServiceWithdrawnData() servicePageData {
	return servicePageData{
		Key:          devServiceWithdrawnKey,
		CopyKey:      "203.0.113.29:8080 tcp",
		Withdrawn:    true,
		Exposure:     "",
		Seen:         "12d",
		InScopeSince: "2026-07-18",
		Citation: []citationHop{
			{Label: "Service", Value: "203.0.113.29:8080/tcp", Detail: "an (address, port, transport) triple"},
			{Label: "Address · formerly cited", Value: "203.0.113.29"},
			{Label: "Name · last citing resolution", Value: "jenkins.acmecorp.io", Detail: "A record, last seen 12d ago"},
			{Label: "Seed · name scope", Value: "acmecorp.io", Detail: "declared 2026-06-14 · custody verified"},
		},
		CitationTerminated: true,
		Address:            "203.0.113.29",
		Port:               "8080",
		Transport:          "tcp",
		Reach:              "",
		Since:              "",
		Timelines: []timelineView{{
			Label:   "reachability",
			Current: nil,
			Breaks:  nil,
			Closed: []spanView{
				{Value: "reached", OpenedAt: "2026-07-18", OpenedFull: "2026-07-18T08:40Z", ClosedAt: "2026-08-10", ClosedFull: "2026-08-10T13:25Z", Reason: "withdrawn"},
			},
		}},
		Rules: []subjectRule{
			{Rule: "admin-panel-reachable", Version: "1", Severity: "high", SevLabel: "High", Fired: false},
		},
		Provenance: []assetKV{
			{K: "Seed", V: "acmecorp.io"},
			{K: "Via", V: "dns sweep → hot scan"},
			{K: "Vantage", V: "eu-west-1"},
			{K: "First seen", V: "2026-07-18"},
		},
		Signals: nil,
	}
}

func devEndpointData() endpointPageData {
	return endpointPageData{
		Key:          devEndpointKey,
		CopyKey:      "edge-gw-03.acmecorp.io 203.0.113.7:443 tcp",
		Name:         "edge-gw-03.acmecorp.io",
		Nameless:     false,
		Service:      "203.0.113.7:443/tcp",
		Withdrawn:    false,
		Seen:         "4m",
		InScopeSince: "2026-06-14",
		Citation: []citationHop{
			{Label: "Endpoint", Value: "edge-gw-03.acmecorp.io · :443 https", Detail: "a (Name, Service) pair — the only key under which HTTP identity is single-valued"},
			{Label: "Name leg", Value: "edge-gw-03.acmecorp.io"},
			{Label: "Service leg", Value: "203.0.113.7:443/tcp"},
			{Label: "Seed · name scope", Value: "acmecorp.io", Detail: "declared 2026-06-14 · custody verified"},
		},
		CitationTerminated: true,
		HasIdentity:        true,
		Status:             "200",
		Server:             "nginx/1.25.0",
		Title:              "Acme edge gateway",
		RedirectLocation:   "",
		WWWAuthenticate:    "",
		Timelines: []timelineView{{
			Label:   "http-identity",
			Current: &spanView{IsGap: false, Value: "200 · nginx/1.25.0", OpenedAt: "2026-08-12T06:00Z"},
			Breaks:  nil,
			Closed: []spanView{
				{Value: "200 · nginx/1.24.0", OpenedAt: "2026-06-14", OpenedFull: "2026-06-14T09:00Z", ClosedAt: "2026-08-12", ClosedFull: "2026-08-12T06:00Z", Reason: "changed"},
			},
		}},
		Rules: []subjectRule{
			{Rule: "admin-panel-reachable", Version: "1", Severity: "high", SevLabel: "High", Fired: false},
			{Rule: "verbose-server-header", Version: "2", Severity: "low", SevLabel: "Low", Fired: true},
		},
		Provenance: []assetKV{
			{K: "Seed", V: "acmecorp.io"},
			{K: "Via", V: "resolution × service join"},
			{K: "Vantage", V: "eu-west-1"},
			{K: "First seen", V: "2026-06-14"},
		},
	}
}

// serviceFixtureData is the render map servicePage passes to the frozen subjectdetail.tmpl in a
// VERGE_DEV build for the two pinned service keys. It returns ok=false for any other key so the
// live read still resolves. render-goldens composes the identical maps statically, so golden and
// candidate agree byte-for-byte.
func (s *server) serviceFixtureData(acct db.Account, key string) (map[string]any, bool) {
	var data servicePageData
	switch key {
	case devServiceKey:
		data = devServiceData()
	case devServiceWithdrawnKey:
		data = devServiceWithdrawnData()
	default:
		return nil, false
	}
	return map[string]any{
		"Title": key, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "inventory", "DesignTokens": true,
		"Service": data,
	}, true
}

func (s *server) endpointFixtureData(acct db.Account, key string) (map[string]any, bool) {
	if key != devEndpointKey {
		return nil, false
	}
	return map[string]any{
		"Title": key, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "inventory", "DesignTokens": true,
		"Endpoint": devEndpointData(),
	}, true
}

// --- screen 15: Graph fixture (package v3.10.0, WORK-ORDER-13-15-BATCH4.md) ------------------
//
// The Graph screen (/graph, #583) reads inside the full app chrome and a real admin session
// (minted by the harness). Its VIEW corpus — the 26-edge, 27-node topology (1 domain apex + 10
// subdomains + 9 addresses + 7 services), each node's placed canvas + minimap coordinates, its
// two halo radii (#22f), its type-derived label offset, the four detailed nodes' ports / first-seen
// / open-signals, and the domain|subdomain split (#22e) — is the design's curated fixture, not a
// live-estate read: the exact hand-classified apex, the placed coordinates and the curated per-node
// severities cannot be reconstructed from the live open-span corpus without fabricating layout data,
// which SPEC-CHANGE forbids. So, exactly as the sibling screens pin their dev fixture and serve it
// under devMode, graphPage serves the pinned fixtures.json graph slice below when s.devMode, and
// TestGraphFixtureMatchesPackage folds every value back through the frozen package — the
// byte-exactness gate before the pixels. The production path (graph.go buildGraph/joinSignals) folds
// the same holes from the real Name/Address/Service topology + the Signal engine's fired census
// (#22e/#22f wired there too); this pins the design's exact bytes for the goldens.

// devGraphData assembles the pinned graphView the goldens capture. It is the exact fixtures.json →
// graph slice in authored order (nodes then edges); TestGraphFixtureMatchesPackage folds every value
// back through the frozen package. render-goldens composes the identical holes statically, so golden
// and candidate agree byte-for-byte.
func devGraphData() graphView {
	nodes := []graphNode{
		{ID: "acmecorp.io", Label: "acmecorp.io", Type: "domain", X: 90, Y: 320, Mx: 8.3, My: 29.3, Sev: "medium", HaloA: 23, HaloB: 20.5, LabelDX: 25, Ports: ":80 :443", First: "2026-05-19T07:58:41Z", OpenSignals: []graphSignal{{Severity: "medium", SevLabel: "Medium", Rule: "dns-mail-policy", Subject: "acmecorp.io"}}},
		{ID: "www.acmecorp.io", Label: "www", Type: "subdomain", X: 400, Y: 52, Mx: 36.7, My: 4.8, Sev: "low", HaloA: 17, HaloB: 14.5, LabelDX: 19},
		{ID: "api.acmecorp.io", Label: "api", Type: "subdomain", X: 400, Y: 112, Mx: 36.7, My: 10.3, Sev: "high", HaloA: 17, HaloB: 14.5, LabelDX: 19, Ports: ":443", First: "2026-05-19T08:00:12Z", OpenSignals: []graphSignal{{Severity: "high", SevLabel: "High", Rule: "outdated-nginx", Subject: "api.acmecorp.io :443"}}},
		{ID: "vpn.acmecorp.io", Label: "vpn", Type: "subdomain", X: 400, Y: 172, Mx: 36.7, My: 15.8, Sev: "critical", HaloA: 17, HaloB: 14.5, LabelDX: 19, Ports: ":443 :1194", First: "2026-06-02T11:40:00Z", OpenSignals: []graphSignal{{Severity: "critical", SevLabel: "Critical", Rule: "tls-cert-expired", Subject: "vpn.acmecorp.io :443"}}},
		{ID: "mail.acmecorp.io", Label: "mail", Type: "subdomain", X: 400, Y: 232, Mx: 36.7, My: 21.3, Sev: "", HaloA: 17, HaloB: 14.5, LabelDX: 19},
		{ID: "grafana.acmecorp.io", Label: "grafana", Type: "subdomain", X: 400, Y: 292, Mx: 36.7, My: 26.8, Sev: "high", HaloA: 17, HaloB: 14.5, LabelDX: 19},
		{ID: "edge-gw-03.acmecorp.io", Label: "edge-gw-03", Type: "subdomain", X: 400, Y: 352, Mx: 36.7, My: 32.3, Sev: "critical", HaloA: 17, HaloB: 14.5, LabelDX: 19, Ports: ":443 :5900", First: "2026-08-12T09:14:33Z", OpenSignals: []graphSignal{{Severity: "critical", SevLabel: "Critical", Rule: "vnc-exposure", Subject: ":5900 vnc"}}},
		{ID: "build-07.acmecorp.io", Label: "build-07", Type: "subdomain", X: 400, Y: 412, Mx: 36.7, My: 37.8, Sev: "high", HaloA: 17, HaloB: 14.5, LabelDX: 19},
		{ID: "assets.acmecorp.io", Label: "assets", Type: "subdomain", X: 400, Y: 472, Mx: 36.7, My: 43.3, Sev: "medium", HaloA: 17, HaloB: 14.5, LabelDX: 19},
		{ID: "old-blog.acmecorp.io", Label: "old-blog", Type: "subdomain", X: 400, Y: 532, Mx: 36.7, My: 48.8, Sev: "medium", HaloA: 17, HaloB: 14.5, LabelDX: 19},
		{ID: "staging-4.acmecorp.io", Label: "staging-4", Type: "subdomain", X: 400, Y: 592, Mx: 36.7, My: 54.3, Sev: "info", HaloA: 17, HaloB: 14.5, LabelDX: 19},
		{ID: "203.0.113.4", Label: "203.0.113.4", Type: "ip", X: 730, Y: 80, Mx: 66.9, My: 7.3, Sev: "", HaloA: 16, HaloB: 13.5, LabelDX: 18},
		{ID: "203.0.113.9", Label: "203.0.113.9", Type: "ip", X: 730, Y: 138, Mx: 66.9, My: 12.7, Sev: "", HaloA: 16, HaloB: 13.5, LabelDX: 18},
		{ID: "203.0.113.12", Label: "203.0.113.12", Type: "ip", X: 730, Y: 196, Mx: 66.9, My: 18, Sev: "", HaloA: 16, HaloB: 13.5, LabelDX: 18},
		{ID: "203.0.113.25", Label: "203.0.113.25", Type: "ip", X: 730, Y: 254, Mx: 66.9, My: 23.3, Sev: "", HaloA: 16, HaloB: 13.5, LabelDX: 18},
		{ID: "203.0.113.31", Label: "203.0.113.31", Type: "ip", X: 730, Y: 312, Mx: 66.9, My: 28.6, Sev: "", HaloA: 16, HaloB: 13.5, LabelDX: 18},
		{ID: "203.0.113.7", Label: "203.0.113.7", Type: "ip", X: 730, Y: 370, Mx: 66.9, My: 33.9, Sev: "", HaloA: 16, HaloB: 13.5, LabelDX: 18},
		{ID: "203.0.113.44", Label: "203.0.113.44", Type: "ip", X: 730, Y: 428, Mx: 66.9, My: 39.2, Sev: "", HaloA: 16, HaloB: 13.5, LabelDX: 18},
		{ID: "203.0.113.18", Label: "203.0.113.18", Type: "ip", X: 730, Y: 486, Mx: 66.9, My: 44.5, Sev: "", HaloA: 16, HaloB: 13.5, LabelDX: 18},
		{ID: "203.0.113.61", Label: "203.0.113.61", Type: "ip", X: 730, Y: 544, Mx: 66.9, My: 49.9, Sev: "", HaloA: 16, HaloB: 13.5, LabelDX: 18},
		{ID: "svc-0", Label: ":443 https", Type: "service", X: 1030, Y: 100, Mx: 94.4, My: 9.2, Sev: "", HaloA: 13, HaloB: 10.5, LabelDX: 15},
		{ID: "svc-1", Label: ":5900 vnc", Type: "service", X: 1030, Y: 168, Mx: 94.4, My: 15.4, Sev: "critical", HaloA: 13, HaloB: 10.5, LabelDX: 15},
		{ID: "svc-2", Label: ":443 nginx", Type: "service", X: 1030, Y: 236, Mx: 94.4, My: 21.6, Sev: "high", HaloA: 13, HaloB: 10.5, LabelDX: 15},
		{ID: "svc-3", Label: ":443 tls", Type: "service", X: 1030, Y: 304, Mx: 94.4, My: 27.9, Sev: "critical", HaloA: 13, HaloB: 10.5, LabelDX: 15},
		{ID: "svc-4", Label: ":25 smtp", Type: "service", X: 1030, Y: 372, Mx: 94.4, My: 34.1, Sev: "low", HaloA: 13, HaloB: 10.5, LabelDX: 15},
		{ID: "svc-5", Label: ":3000 http", Type: "service", X: 1030, Y: 440, Mx: 94.4, My: 40.3, Sev: "high", HaloA: 13, HaloB: 10.5, LabelDX: 15},
		{ID: "svc-6", Label: ":22 ssh", Type: "service", X: 1030, Y: 508, Mx: 94.4, My: 46.6, Sev: "high", HaloA: 13, HaloB: 10.5, LabelDX: 15},
	}
	edges := []graphEdge{
		{X1: 90, Y1: 320, X2: 400, Y2: 52},
		{X1: 90, Y1: 320, X2: 400, Y2: 112},
		{X1: 90, Y1: 320, X2: 400, Y2: 172},
		{X1: 90, Y1: 320, X2: 400, Y2: 232},
		{X1: 90, Y1: 320, X2: 400, Y2: 292},
		{X1: 90, Y1: 320, X2: 400, Y2: 352},
		{X1: 90, Y1: 320, X2: 400, Y2: 412},
		{X1: 90, Y1: 320, X2: 400, Y2: 472},
		{X1: 90, Y1: 320, X2: 400, Y2: 532},
		{X1: 90, Y1: 320, X2: 400, Y2: 592},
		{X1: 400, Y1: 52, X2: 730, Y2: 80},
		{X1: 400, Y1: 112, X2: 730, Y2: 138},
		{X1: 400, Y1: 172, X2: 730, Y2: 196},
		{X1: 400, Y1: 232, X2: 730, Y2: 254},
		{X1: 400, Y1: 292, X2: 730, Y2: 312},
		{X1: 400, Y1: 352, X2: 730, Y2: 370},
		{X1: 400, Y1: 412, X2: 730, Y2: 428},
		{X1: 400, Y1: 472, X2: 730, Y2: 486},
		{X1: 400, Y1: 592, X2: 730, Y2: 544},
		{X1: 730, Y1: 80, X2: 1030, Y2: 100, ToService: true},
		{X1: 730, Y1: 370, X2: 1030, Y2: 168, ToService: true},
		{X1: 730, Y1: 138, X2: 1030, Y2: 236, ToService: true},
		{X1: 730, Y1: 196, X2: 1030, Y2: 304, ToService: true},
		{X1: 730, Y1: 254, X2: 1030, Y2: 372, ToService: true},
		{X1: 730, Y1: 312, X2: 1030, Y2: 440, ToService: true},
		{X1: 730, Y1: 428, X2: 1030, Y2: 508, ToService: true},
	}
	fitX, fitY, fitK, minK := graphFit(graphViewW, graphViewH)
	return graphView{
		Nodes: nodes, Edges: edges, Empty: false,
		ViewW: graphViewW, ViewH: graphViewH, MiniW: graphMiniW, MiniH: graphMiniH,
		// Every pinned node sits inside the viewport box, so the content box IS the
		// viewport box and the design's own pinned Mx/My stay the minimap's basis.
		ContentW: graphViewW, ContentH: graphViewH,
		FitX: fitX, FitY: fitY, FitK: fitK, MinK: minK, LabelMinK: graphLabelMinK,
	}
}

// graphFixtureData is the render map graphPage passes to the frozen graph.tmpl in a VERGE_DEV build
// (the three golden states are pure client-JS variants of this one server render). render-goldens
// composes the identical map statically, so golden and candidate agree byte-for-byte.
func (s *server) graphFixtureData(acct db.Account) map[string]any {
	return map[string]any{
		"Title": "Graph", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "graph", "DesignTokens": true,
		"Graph": devGraphData(),
	}
}

// --- screen 16: Reports fixture (package v3.11.0, WORK-ORDER-16-18-BATCH5.md) -----------------
//
// The Reports screen (`/reports`, #588) renders inside the full app chrome and a real admin session
// (minted by the harness). Its VIEW corpus — the three KPI cards with their precomputed spark/bar
// geometry and delta chips, the "Open signals over time" chart (grid, polylines, x-labels + the
// hover-readout data attributes), the by-severity bars, the 84-cell scans-per-day heat, and the
// three schedule rows — is the design's curated fixture, not a live-estate read: the exact SVG
// coordinates, the color-mix intensities and the relative instants cannot be reconstructed from the
// live derivations without fabricating domain data, which SPEC-CHANGE forbids. So, exactly as the
// sibling screens pin their dev fixture and serve it under devMode, reportsPage serves the pinned
// fixtures.json reports slice below when s.devMode. Unlike the sibling screens (which hand-pin Go
// literals and fold them back through the fixture in a drift test), the reports corpus is read
// STRAIGHT from the embedded fixtures.json — there is no second copy to drift from. render-goldens
// composes the identical maps from the SAME bytes, so golden and candidate agree; G2 is the gate.
// The production path (reports.go) builds the same holes from real reads (period picker, trend folds).
//
// The struct set below mirrors the reports slice one-to-one: numbers ride as json.Number so the
// rendered figure is the fixture's literal text (16.7 stays "16.7", never a re-formatted float), and
// the heat cell backgrounds ride as template.CSS so the color-mix()/var() values reach the style
// attribute verbatim rather than being neutralised to ZgotmplZ (the production heatCell does the
// same). render-goldens carries a byte-identical copy of these types.

type reportsFixtureDelta struct {
	Has  bool   `json:"has"`
	Text string `json:"text"`
	Dir  string `json:"dir"`
	Tone string `json:"tone"`
}

type reportsFixtureSpark struct {
	W     json.Number `json:"w"`
	H     json.Number `json:"h"`
	Area  string      `json:"area"`
	Line  string      `json:"line"`
	Color string      `json:"color"`
	DotX  json.Number `json:"dot_x"`
	DotY  json.Number `json:"dot_y"`
}

type reportsFixtureBar struct {
	HeightPct json.Number `json:"height_pct"`
	Title     string      `json:"title"`
	Last      bool        `json:"last"`
}

type reportsFixtureBars struct {
	Bars       []reportsFixtureBar `json:"bars"`
	LeftLabel  string              `json:"left_label"`
	RightLabel string              `json:"right_label"`
}

type reportsFixtureGrid struct {
	Y      json.Number `json:"y"`
	X1     json.Number `json:"x1"`
	X2     json.Number `json:"x2"`
	Stroke string      `json:"stroke"`
	LabelX json.Number `json:"label_x"`
	Label  string      `json:"label"`
}

type reportsFixtureXLabel struct {
	X    json.Number `json:"x"`
	Y    json.Number `json:"y"`
	Text string      `json:"text"`
}

type reportsFixtureSeries struct {
	W          json.Number            `json:"w"`
	H          json.Number            `json:"h"`
	N          json.Number            `json:"n"`
	Grid       []reportsFixtureGrid   `json:"grid"`
	AllOpen    string                 `json:"all_open"`
	CritHigh   string                 `json:"crit_high"`
	XLabels    []reportsFixtureXLabel `json:"x_labels"`
	LabelsAttr string                 `json:"labels_attr"`
	SeriesJSON string                 `json:"series_json"`
}

type reportsFixtureSev struct {
	Sev   string      `json:"sev"`
	Label string      `json:"label"`
	Pct   json.Number `json:"pct"`
	Count json.Number `json:"count"`
}

type reportsFixtureHeat struct {
	Title  string       `json:"title"`
	Bg     template.CSS `json:"bg"`
	Border template.CSS `json:"border"`
}

type reportsFixtureSchedule struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Cadence      string      `json:"cadence"`
	Delivery     string      `json:"delivery"`
	Format       string      `json:"format"`
	LastSent     string      `json:"last_sent"`
	LastMins     json.Number `json:"last_mins"`
	HasDelivery  bool        `json:"has_delivery"`
	DeliveryHref string      `json:"delivery_href"`
}

type reportsFixturePeriod struct {
	Token string `json:"token"`
	Label string `json:"label"`
}

type reportsFixtureWizardSection struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Checked bool   `json:"checked"`
}

type reportsFixtureWizardChannel struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Hint  string `json:"hint"`
}

type reportsFixtureWizard struct {
	Title       string                        `json:"title"`
	FormAction  string                        `json:"form_action"`
	FinishLabel string                        `json:"finish_label"`
	Steps       []string                      `json:"steps"`
	Sections    []reportsFixtureWizardSection `json:"sections"`
	Cads        []string                      `json:"cads"`
	DefaultCad  string                        `json:"default_cad"`
	Channels    []reportsFixtureWizardChannel `json:"channels"`
}

type reportsFixture struct {
	Period            string                   `json:"period"`
	PeriodLabel       string                   `json:"period_label"`
	Periods           []reportsFixturePeriod   `json:"periods"`
	RangeLabel        string                   `json:"range_label"`
	RangeWeeks        json.Number              `json:"range_weeks"`
	HasOpenSignals    bool                     `json:"has_open_signals"`
	OpenSignals       string                   `json:"open_signals"`
	OpenDelta         reportsFixtureDelta      `json:"open_delta"`
	HasOpenSpark      bool                     `json:"has_open_spark"`
	OpenSpark         reportsFixtureSpark      `json:"open_spark"`
	HasDiscovery      bool                     `json:"has_discovery"`
	DiscoveryCount    string                   `json:"discovery_count"`
	DiscoveryDelta    reportsFixtureDelta      `json:"discovery_delta"`
	DiscoveryNames    json.Number              `json:"discovery_names"`
	DiscoveryServices json.Number              `json:"discovery_services"`
	DiscoveryBars     reportsFixtureBars       `json:"discovery_bars"`
	HasMTTW           bool                     `json:"has_mttw"`
	MTTW              string                   `json:"mttw"`
	MTTWDelta         reportsFixtureDelta      `json:"mttw_delta"`
	HasMTTWSpark      bool                     `json:"has_mttw_spark"`
	MTTWSpark         reportsFixtureSpark      `json:"mttw_spark"`
	HasSignalSeries   bool                     `json:"has_signal_series"`
	SignalSeries      reportsFixtureSeries     `json:"signal_series"`
	HasSeverity       bool                     `json:"has_severity"`
	BySeverity        []reportsFixtureSev      `json:"by_severity"`
	HasHeat           bool                     `json:"has_heat"`
	Heat              []reportsFixtureHeat     `json:"heat"`
	Schedules         []reportsFixtureSchedule `json:"schedules"`
	Wizard            reportsFixtureWizard     `json:"wizard"`
}

// loadReportsFixture reads the pinned fixtures.json reports slice from the embedded design
// package (designfs). Both reportsPage (candidate) and render-goldens (golden) read the SAME
// bytes and shape them identically, so the two agree. A read/parse failure degrades to the
// zero fixture (the page's empty states) rather than 500ing.
func loadReportsFixture() reportsFixture {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return reportsFixture{}
	}
	var ff struct {
		Reports reportsFixture `json:"reports"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return reportsFixture{}
	}
	return ff.Reports
}

// reportsFixtureData is the render map reportsPage passes to the frozen reports.tmpl in a
// VERGE_DEV build. It stamps the chrome + design-token holes and the pinned fixture corpus,
// passing each sub-struct straight through to the tmpl's holes. render-goldens composes the
// identical map statically from the same fixtures.json, so golden and candidate agree.
func (s *server) reportsFixtureData(acct db.Account) map[string]any {
	fx := loadReportsFixture()
	return map[string]any{
		"Title": "Reports", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "reports", "DesignTokens": true,

		"RangeLabel":  fx.RangeLabel,
		"RangeWeeks":  fx.RangeWeeks,
		"Periods":     fx.Periods,
		"Period":      fx.Period,
		"PeriodLabel": fx.PeriodLabel,

		"HasOpenSignals": fx.HasOpenSignals,
		"OpenSignals":    fx.OpenSignals,
		"OpenDelta":      fx.OpenDelta,
		"HasOpenSpark":   fx.HasOpenSpark,
		"OpenSpark":      fx.OpenSpark,

		"HasDiscovery":      fx.HasDiscovery,
		"DiscoveryCount":    fx.DiscoveryCount,
		"DiscoveryDelta":    fx.DiscoveryDelta,
		"DiscoveryNames":    fx.DiscoveryNames,
		"DiscoveryServices": fx.DiscoveryServices,
		"DiscoveryBars":     fx.DiscoveryBars,

		"HasMTTW":      fx.HasMTTW,
		"MTTW":         fx.MTTW,
		"MTTWDelta":    fx.MTTWDelta,
		"HasMTTWSpark": fx.HasMTTWSpark,
		"MTTWSpark":    fx.MTTWSpark,

		"HasSignalSeries": fx.HasSignalSeries,
		"SignalSeries":    fx.SignalSeries,

		"HasSeverity": fx.HasSeverity,
		"BySeverity":  fx.BySeverity,

		"HasHeat": fx.HasHeat,
		"Heat":    fx.Heat,

		"Schedules": fx.Schedules,
	}
}

// reportartifactFixture is the pinned fixtures.json → reportartifact slice: the page holes
// (.Heading .Period .ScheduleID) and the delivered document (.Doc, a message.ArtifactDoc whose
// JSON tags mirror the fixture keys verbatim), plus the never-delivered variant (schedule s2,
// .Doc.Empty) the states.json ?variant=never-delivered golden pins.
type reportartifactFixture struct {
	Heading        string                       `json:"heading"`
	Period         string                       `json:"period"`
	ScheduleID     string                       `json:"schedule_id"`
	Doc            message.ArtifactDoc          `json:"doc"`
	NeverDelivered reportartifactFixtureVariant `json:"never_delivered_variant"`
}

// reportartifactFixtureVariant is the never-delivered overlay: a schedule that exists (s2) but
// has not delivered, so .Doc.Empty stands and .Period reads "no delivery yet". Its heading is not
// pinned in the slice — it is the schedule's own honest name, resolved from the reports schedules
// fixture by id (reportartifactVariantHeading), the same datum the Reports screen shows for s2.
type reportartifactFixtureVariant struct {
	Period     string              `json:"period"`
	ScheduleID string              `json:"schedule_id"`
	Doc        message.ArtifactDoc `json:"doc"`
}

// loadReportartifactFixture reads the pinned fixtures.json reportartifact slice from the embedded
// design package. Both reportDeliveryPage (candidate) and render-goldens (golden) read the SAME
// bytes and shape them identically, so the two agree. A read/parse failure degrades to the zero
// fixture (the empty-state document) rather than 500ing.
func loadReportartifactFixture() reportartifactFixture {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return reportartifactFixture{}
	}
	var ff struct {
		ReportArtifact reportartifactFixture `json:"reportartifact"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return reportartifactFixture{}
	}
	return ff.ReportArtifact
}

// reportartifactVariantHeading resolves the never-delivered variant's heading — the honest name
// of the schedule behind it — from the reports schedules fixture by id (s2 → "Monthly asset
// inventory"). The default state pins its heading directly (s1's name); the variant carries only
// the schedule id, so its title comes from the same package datum rather than being fabricated. A
// missing id falls back to the generic delivery heading.
func reportartifactVariantHeading(scheduleID string) string {
	for _, sc := range loadReportsFixture().Schedules {
		if sc.ID == scheduleID {
			return sc.Name
		}
	}
	return "Report delivery"
}

// reportartifactFixtureData is the render map reportDeliveryPage passes to the frozen
// reportartifact.tmpl in a VERGE_DEV build. The default state serves the pinned delivery document;
// the never-delivered variant (?variant=never-delivered, states.json) overlays schedule s2's
// empty-state document, its "no delivery yet" period, and s2's resolved heading. render-goldens
// composes the identical map from the same fixtures.json, so golden and candidate agree.
func (s *server) reportartifactFixtureData(acct db.Account, variant string) map[string]any {
	fx := loadReportartifactFixture()

	heading, period, scheduleID, doc := fx.Heading, fx.Period, fx.ScheduleID, fx.Doc
	if variant == "never-delivered" {
		heading = reportartifactVariantHeading(fx.NeverDelivered.ScheduleID)
		period = fx.NeverDelivered.Period
		scheduleID = fx.NeverDelivered.ScheduleID
		doc = fx.NeverDelivered.Doc
	}

	var scheduleHole any
	if scheduleID != "" {
		scheduleHole = scheduleID
	}

	return map[string]any{
		"Title": "Report delivery", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "reports", "DesignTokens": true,
		"Heading":    heading,
		"Period":     period,
		"ScheduleID": scheduleHole,
		"Doc":        doc,
	}
}

// reportsWizardWindow shapes the "schedulewizard" holes from the pinned fixtures.json wizard
// slice and the request query (#23f). The wizard states hit the PRG GET URLs directly
// (states.json), so the step, name, sections, cad and channel are read from the query — a
// fresh GET (no ?step) opens step 0 with the fixture's checked defaults. render-goldens
// composes the identical map for wizard-1..4, so golden and candidate agree.
func (s *server) reportsWizardFixtureData(r *http.Request, acct db.Account) map[string]any {
	fx := loadReportsFixture().Wizard
	return reportsWizardMap(fx, r.URL.Query(), acct)
}

// reportsWizardMap is the shared shaping the dev handler and render-goldens both apply, so the
// two never diverge. It reconstructs the wizard's controlled state from the query using the
// fixture vocabulary (section keys, cadence presets, channel tokens) and stamps every hole the
// frozen "schedulewizard" tmpl reads.
func reportsWizardMap(fx reportsFixtureWizard, q map[string][]string, acct db.Account) map[string]any {
	get := func(k string) string {
		if v, ok := q[k]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}
	step := 0
	if n, err := strconv.Atoi(get("step")); err == nil {
		step = n
	}
	if step < 0 {
		step = 0
	}
	if step > len(fx.Steps)-1 {
		step = len(fx.Steps) - 1
	}

	// Sections: an explicit query set (the PRG steps carry them) else the fixture's checked
	// defaults (the fresh first step). Canonicalised to the fixture order, unknowns dropped.
	selected := map[string]bool{}
	if raw, ok := q["sections"]; ok {
		for _, k := range raw {
			selected[k] = true
		}
	} else {
		for _, sec := range fx.Sections {
			if sec.Checked {
				selected[sec.Key] = true
			}
		}
	}
	sectionOpts := make([]map[string]any, 0, len(fx.Sections))
	orderedKeys := make([]string, 0, len(fx.Sections))
	labels := make([]string, 0, len(fx.Sections))
	for _, sec := range fx.Sections {
		on := selected[sec.Key]
		sectionOpts = append(sectionOpts, map[string]any{"Key": sec.Key, "Label": sec.Label, "Checked": on})
		if on {
			orderedKeys = append(orderedKeys, sec.Key)
			labels = append(labels, sec.Label)
		}
	}

	cad := get("cad")
	if cad == "" {
		cad = fx.DefaultCad
	}
	cron := get("cron")
	cads := make([]map[string]any, 0, len(fx.Cads))
	for _, c := range fx.Cads {
		cads = append(cads, map[string]any{"Value": c, "Selected": c == cad})
	}

	channel := get("channel")
	if channel == "" && len(fx.Channels) > 0 {
		channel = fx.Channels[0].Value
	}
	channelLabel := ""
	channels := make([]map[string]any, 0, len(fx.Channels))
	for _, c := range fx.Channels {
		sel := c.Value == channel
		if sel {
			channelLabel = c.Label
		}
		channels = append(channels, map[string]any{"Value": c.Value, "Label": c.Label, "Hint": c.Hint, "Selected": sel})
	}

	steps := make([]map[string]any, 0, len(fx.Steps))
	for i, title := range fx.Steps {
		steps = append(steps, map[string]any{"Num": i + 1, "Title": title, "Done": i < step, "Current": i == step})
	}

	nameSummary := reportsWizardName(get("name"))
	sectionsSummary := "—"
	if len(labels) > 0 {
		sectionsSummary = reportsWizardJoin(labels)
	}
	review := []map[string]any{
		{"K": "Report", "V": nameSummary},
		{"K": "Sections", "V": sectionsSummary},
		{"K": "Cadence", "V": reportCadLabel(cad, cron)},
		{"K": "Format", "V": reportScheduleFormat},
		{"K": "Delivery", "V": channelLabel},
	}

	last := step == len(fx.Steps)-1
	return map[string]any{
		"Title": fx.Title, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "reports", "DesignTokens": true,

		"WizardTitle": fx.Title,
		"FormAction":  fx.FormAction,
		"FinishLabel": fx.FinishLabel,
		"EditMode":    false,
		"ID":          int64(0),

		"Step":      step,
		"StepNum":   step + 1,
		"StepTotal": len(fx.Steps),
		"Last":      last,
		"Steps":     steps,

		"Name":         get("name"),
		"Sections":     sectionOpts,
		"SectionsKeys": orderedKeys,
		"Cads":         cads,
		"Cad":          cad,
		"Cron":         cron,
		"Custom":       cad == reportCustomCad,
		"Channels":     channels,
		"ChannelID":    channel,
		"ChannelLabel": channelLabel,

		"Review": review,
	}
}

// reportsWizardName / reportsWizardJoin are the tiny review helpers the wizard shaping shares
// with render-goldens (an em dash for an empty name, a comma-joined section list), so both
// produce the identical Review rows.
func reportsWizardName(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func reportsWizardJoin(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}

// inboxFixture is the pinned fixtures.json → inbox slice: the list holes (.Unread .Filter
// .AllHref .UnreadHref .Messages) and the selected-message detail. Per SPEC-CHANGE #24 (ruled)
// the detail carries NO prose body — the form is the census (kind + linked mono key) plus the
// delivery receipts, a failed one flagged undelivered with its host + drill-down reason. Message
// ids are the fixture's own string keys (m1–m5); ?id selects a detail, ?filter=unread trims the
// list. The golden (render-goldens) reads the SAME slice and shapes it identically, so the two
// agree byte-for-byte.
type inboxFixture struct {
	Unread     int    `json:"unread"`
	Filter     string `json:"filter"`
	AllHref    string `json:"all_href"`
	UnreadHref string `json:"unread_href"`
	Messages   []struct {
		ID       string `json:"id"`
		Read     bool   `json:"read"`
		Cls      string `json:"cls"`
		Instant  string `json:"instant"`
		Rel      string `json:"rel"`
		Headline string `json:"headline"`
	} `json:"messages"`
	Selected struct {
		ID       string `json:"id"`
		Cls      string `json:"cls"`
		Headline string `json:"headline"`
		Rel      string `json:"rel"`
		Instant  string `json:"instant"`
		Census   []struct {
			Kind string `json:"kind"`
			Key  string `json:"key"`
			Href string `json:"href"`
		} `json:"census"`
		Deliveries []struct {
			State       string `json:"state"`
			ChannelHost string `json:"channel_host"`
			Failed      bool   `json:"failed"`
			LastError   string `json:"last_error"`
		} `json:"deliveries"`
		Href      string `json:"href"`
		JumpLabel string `json:"jump_label"`
	} `json:"selected_fixture"`
}

// loadInboxFixture reads the pinned fixtures.json inbox slice from the embedded design package
// (designfs). Both inboxPage (candidate) and render-goldens (golden) read the SAME bytes and shape
// them identically, so the two agree. A read/parse failure degrades to the zero fixture (the
// inbox-zero empty states) rather than 500ing.
func loadInboxFixture() inboxFixture {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return inboxFixture{}
	}
	var ff struct {
		Inbox inboxFixture `json:"inbox"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return inboxFixture{}
	}
	return ff.Inbox
}

// inboxFixtureData is the render map inboxPage passes to the frozen inbox.tmpl in a VERGE_DEV
// build. It reads the selection (?id) and the all/unread filter (?filter) straight off the query —
// the fixture is READ-ONLY, so a dev capture never mutates read-state (the additive, deterministic
// posture the sibling fixtures paths keep): the id only marks a row .Selected and picks the detail
// from the pinned selected_fixture, and the filter trims the list to the unread rows. render-goldens
// composes the identical map statically from the same fixtures.json, so golden and candidate agree.
func (s *server) inboxFixtureData(acct db.Account, r *http.Request) map[string]any {
	fx := loadInboxFixture()

	selID := r.URL.Query().Get("id")
	filter := "all"
	if r.URL.Query().Get("filter") == "unread" {
		filter = "unread"
	}

	messages := make([]map[string]any, 0, len(fx.Messages))
	for _, m := range fx.Messages {
		if filter == "unread" && m.Read {
			continue
		}
		messages = append(messages, map[string]any{
			"ID":       m.ID,
			"Read":     m.Read,
			"Selected": selID != "" && m.ID == selID,
			"Class":    m.Cls,
			"Instant":  m.Instant,
			"Rel":      m.Rel,
			"Headline": m.Headline,
		})
	}

	// The detail card renders only when the selected id resolves to the pinned message (m1). No
	// prose body (SPEC-CHANGE #24): the form is the census + the delivery receipts.
	var selected map[string]any
	if selID != "" && selID == fx.Selected.ID {
		census := make([]map[string]any, 0, len(fx.Selected.Census))
		for _, c := range fx.Selected.Census {
			census = append(census, map[string]any{"Kind": c.Kind, "Key": c.Key, "Href": c.Href})
		}
		deliveries := make([]map[string]any, 0, len(fx.Selected.Deliveries))
		for _, d := range fx.Selected.Deliveries {
			deliveries = append(deliveries, map[string]any{
				"State": d.State, "ChannelHost": d.ChannelHost, "Failed": d.Failed, "LastError": d.LastError,
			})
		}
		selected = map[string]any{
			"ID":         fx.Selected.ID,
			"Class":      fx.Selected.Cls,
			"Headline":   fx.Selected.Headline,
			"Rel":        fx.Selected.Rel,
			"Instant":    fx.Selected.Instant,
			"Census":     census,
			"Deliveries": deliveries,
			"Href":       fx.Selected.Href,
			"JumpLabel":  fx.Selected.JumpLabel,
		}
	}

	// The filter toggle preserves the open message, so its links carry the id (mirrors the live
	// inboxPage). Without a selection the fixture's own hrefs stand.
	allHref, unreadHref := fx.AllHref, fx.UnreadHref
	if selID != "" {
		allHref = "/inbox?id=" + selID
		unreadHref = "/inbox?filter=unread&id=" + selID
	}

	return map[string]any{
		"Title": "Inbox", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "inbox", "DesignTokens": true,
		"Messages":   messages,
		"Selected":   selected,
		"Unread":     fx.Unread,
		"Filter":     filter,
		"AllHref":    allHref,
		"UnreadHref": unreadHref,
	}
}

// searchSeg is one fixtures.json search segment: the literal run of a matched field
// and whether it is the highlighted (query-matched) run. Both the render-goldens
// golden and this candidate read the SAME segments, and reproduce them by folding the
// reconstructed field text through searchSegs (the live handler's builder), so the two
// agree byte-for-byte and TestBuildSearchMatchesDesignFixture proves the first-match
// rule reproduces the authored segmentation.
type searchSeg struct {
	Text string `json:"text"`
	Hit  bool   `json:"hit"`
}

// searchFixture is the design-system/fixtures/fixtures.json → search slice: the query,
// the total, the four groups (assets carry a nullable severity; #25b), and the empty
// variant (q "zzz-none", total 0). Fields are pre-segmented in the package.
type searchFixture struct {
	Query  string `json:"query"`
	Total  int    `json:"total"`
	Assets []struct {
		Href     string      `json:"href"`
		NameSegs []searchSeg `json:"name_segs"`
		Type     string      `json:"type"`
		Severity string      `json:"severity"`
		SevLabel string      `json:"sev_label"`
	} `json:"assets"`
	Signals []struct {
		Href        string      `json:"href"`
		Severity    string      `json:"severity"`
		SevLabel    string      `json:"sev_label"`
		RuleSegs    []searchSeg `json:"rule_segs"`
		SubjectSegs []searchSeg `json:"subject_segs"`
	} `json:"signals"`
	Batches []struct {
		Href      string      `json:"href"`
		Status    string      `json:"status"`
		LabelSegs []searchSeg `json:"label_segs"`
	} `json:"batches"`
	Docs []struct {
		TitleSegs []searchSeg `json:"title_segs"`
		SnipSegs  []searchSeg `json:"snip_segs"`
	} `json:"docs"`
	EmptyVariant struct {
		Query string `json:"query"`
		Total int    `json:"total"`
	} `json:"empty_variant"`
}

// loadSearchFixture reads the pinned fixtures.json search slice from the embedded
// design package (designfs). Both searchPage (candidate, under devMode) and
// render-goldens (golden) read the SAME bytes and shape them identically, so the two
// agree. A read/parse failure degrades to the zero fixture (the empty state) rather
// than 500ing.
func loadSearchFixture() searchFixture {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return searchFixture{}
	}
	var ff struct {
		Search searchFixture `json:"search"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return searchFixture{}
	}
	return ff.Search
}

// joinFixtureSegs reconstructs a matched field's raw text from its authored segments
// (the concatenation of every segment's Text) so the field can be re-segmented through
// searchSegs — the live handler's #25a builder — rather than read straight off the
// fixture. This exercises the builder in the golden path and lets the drift test assert
// the first-match rule reproduces the authored segmentation exactly.
func joinFixtureSegs(segs []searchSeg) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(s.Text)
	}
	return b.String()
}

// searchFixtureData is the render map searchPage passes to the frozen search.tmpl in a
// VERGE_DEV build. It reads ?q= straight off the query (the fixture is READ-ONLY): the
// canonical query ("acme") renders the authored slice, folding every field through
// searchSegs; any other query — including the empty variant "zzz-none" — renders the
// zero-result empty state. render-goldens composes the identical map from the same
// fixtures.json, so golden and candidate agree.
func (s *server) searchFixtureData(acct db.Account, r *http.Request) map[string]any {
	fx := loadSearchFixture()
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q != fx.Query {
		return searchRenderMap(acct, q, 0, nil, nil, nil, nil)
	}

	assets := make([]searchAsset, 0, len(fx.Assets))
	for _, a := range fx.Assets {
		assets = append(assets, searchAsset{
			NameSegs: searchSegs(joinFixtureSegs(a.NameSegs), q),
			Type:     a.Type,
			Severity: a.Severity,
			SevLabel: a.SevLabel,
			Href:     a.Href,
		})
	}
	signals := make([]searchSignal, 0, len(fx.Signals))
	for _, sg := range fx.Signals {
		signals = append(signals, searchSignal{
			RuleSegs:    searchSegs(joinFixtureSegs(sg.RuleSegs), q),
			SubjectSegs: searchSegs(joinFixtureSegs(sg.SubjectSegs), q),
			Severity:    sg.Severity,
			SevLabel:    sg.SevLabel,
			Href:        sg.Href,
		})
	}
	batches := make([]searchBatch, 0, len(fx.Batches))
	for _, b := range fx.Batches {
		batches = append(batches, searchBatch{
			Status:    b.Status,
			LabelSegs: searchSegs(joinFixtureSegs(b.LabelSegs), q),
			Href:      b.Href,
		})
	}
	docs := make([]searchDoc, 0, len(fx.Docs))
	for _, d := range fx.Docs {
		docs = append(docs, searchDoc{
			TitleSegs: searchSegs(joinFixtureSegs(d.TitleSegs), q),
			SnipSegs:  searchSegs(joinFixtureSegs(d.SnipSegs), q),
		})
	}

	total := len(assets) + len(signals) + len(batches) + len(docs)
	return searchRenderMap(acct, q, total, assets, signals, batches, docs)
}
