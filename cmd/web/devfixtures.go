package main

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
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

// devFixtureAccounts pins fixtures.json → accounts: an admin and a viewer. One account
// per role — the mint resolves states.json's `session` role to the account here.
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

	// The fixture account whose Profile the screen-3 goldens capture (also seeded, as the
	// admin, by devFixtureAccounts).
	devProfileUsername = "ola.perez"

	// devProfileCreated is fixtures.json → profile.account.created: the date-only UTC
	// account-creation date the Profile's "Member since" row renders (isoDate(acct.CreatedAt)).
	// seedDevFixtureAccounts creates the account at wall-clock now(), so seedProfileFixtures
	// pins created_at back to this fixture date; otherwise "Member since" drifts to the seed
	// day (a "row data equals fixtures" miss). TestProfileFixtureMatchesPackage folds it back
	// through the frozen package.
	devProfileCreated = "2026-04-11"

	// The one Okta identity linked to the fixture account, and its date-only UTC link date.
	devProfileSSOProviderSlug = "okta"
	devProfileSSOProviderName = "Okta"
	devProfileSSODisplayName  = "ola.perez@acmecorp.io"
	devProfileSSOLinkedAt     = "2026-06-30"

	// Google offered as a linkable provider (enabled, no identity → appears under "Link an
	// identity"). fixtures.json → profile.sso_providers.
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
	last       string // fixtures.json display token: 2h / 14d
	lastOffset time.Duration
}

// devProfileTokens pins fixtures.json → profile.tokens, in fixture order.
var devProfileTokens = []devProfileToken{
	{name: "laptop-cli", prefix: "vg_pat_9f3k…", created: "2026-05-02", last: "2h", lastOffset: 2 * time.Hour},
	{name: "grafana-readonly", prefix: "vg_pat_x81m…", created: "2026-07-19", last: "14d", lastOffset: 14 * 24 * time.Hour},
}

// devFixtureClockTime parses the pinned clock; a VERGE_DEV server and the seeder read the
// same instant from it.
func devFixtureClockTime() (time.Time, error) {
	return time.Parse(time.RFC3339, devFixtureClock)
}

// devFixtureDate parses a fixtures.json date-only string to midnight UTC, which isoDate
// formats straight back to the same YYYY-MM-DD (date-only UTC, per the work order).
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
		if _, err := pool.Exec(ctx, insToken,
			acct.ID, pt.name, pt.prefix, hashToken("verge-dev-token-"+pt.name), created, clock.Add(-pt.lastOffset),
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

	// devFixtureInviteRole is fixtures.json → signin.invite_role: the role the seeded invite
	// grants (the frozen invite card renders it in its sub-line).
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

// devSigninProviders pins fixtures.json → signin.sso_providers, in fixture order.
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

// recoveryCodes returns the recovery set to reveal + store at enrolment-confirm. A VERGE_DEV
// build returns the pinned fixture set (with bcrypt hashes to store); a real build draws fresh
// high-entropy codes. Splitting here keeps the dev determinism out of the real credential path.
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
	// The summary-band figures fixtures.json → exposure pins: 14 exposed (with a +2 vs-last-batch
	// delta, has_deltas true), 41 firewalled, 7 not reached.
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

// devExposureRow mirrors one fixtures.json exposure.rows entry: a service's address, its
// ":port transport", the internal + internet reach legs (expleg display states), and its since.
type devExposureRow struct {
	asset    string
	svc      string
	internal string
	internet string
	since    string
}

// devExposureRows pins fixtures.json → exposure.rows in authored order.
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

// devCoverageMessage mirrors one fixtures.json coverage.messages entry (bound empty where the
// staleness chip carries no trailing figure).
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

// devCoverageMeters pins fixtures.json coverage.meters in authored order: an ADDRESS scope
// (203.0.113.0/24) rendering 198/214 subjects with the skip breakdown, then a NAME scope
// (acmecorp.io) as a census of 62 addresses (no denominator).
var devCoverageMeters = []devCoverageMeter{
	{label: "203.0.113.0/24", counted: 198, total: &coverageTotal214, unit: "subjects", detail: "16 skipped: excluded subtree + 3 unresolvable names"},
	{label: "acmecorp.io (name scope)", counted: 62, total: nil, unit: "addresses", detail: "census state — a name scope has no denominator; custody extension reaches what resolution reveals"},
}

// devCoverageMessages pins fixtures.json coverage.messages in authored order (gap / stale·9d /
// silent / not-evaluable), each carrying the relative When and the ISO tooltip instant.
var devCoverageMessages = []devCoverageMessage{
	{kind: "gap", badge: "no address", subject: "old-blog.acmecorp.io", text: "Expected a resolution; none observed for 3 checks.", when: "2h", iso: "2026-08-22T12:20:04Z"},
	{kind: "stale", badge: "stale", bound: "9d", subject: "internal.acmecorp.io zone", text: "Zone aged past two re-supply intervals — the source went stale.", when: "9d", iso: "2026-08-13T04:44:19Z"},
	{kind: "silent", badge: "no reports", subject: "dc-fra-01", text: "Vantage stopped reporting mid-batch; open spans are not evaluable.", when: "41m", iso: "2026-08-22T13:41:02Z"},
	{kind: "not-evaluable", badge: "not evaluable", subject: "ap-south-1 conclusions", text: "Missed 2 of 3 checks this batch; exposure conclusions marked unverified.", when: "5h", iso: "2026-08-22T09:03:55Z"},
}

// devCoverageGaps pins fixtures.json coverage.gaps in authored order.
var devCoverageGaps = []devCoverageGap{
	{subject: "old-blog.acmecorp.io", gap: "no address", expected: "A record", since: "2h"},
	{subject: "203.0.113.44:22", gap: "no banner", expected: "ssh identification", since: "6h"},
	{subject: "mail.acmecorp.io:25", gap: "no exchange", expected: "smtp greeting", since: "1d"},
}

// devCoverageUnevaluables pins fixtures.json coverage.unevaluable in authored order.
var devCoverageUnevaluables = []devCoverageUnevaluable{
	{id: "tls-weak-key", version: 3, why: "needs a completed tls-acceptance exchange; none committed this batch"},
	{id: "zone-removal", version: 1, why: "needs a fresh zone file; the upload aged into a gap"},
}

// devCoverageStaleZones pins fixtures.json coverage.stale_zones in authored order.
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
	// devRunDetailID is fixtures.json → rundetail.id: the dispatch id the run drill-in
	// serves from the fixture (the golden route /runs/1407). 1408 stays the MISSING id the
	// error goldens use — anything but 1407 routes to the missing-run ErrorPage.
	devRunDetailID = "1407"

	// The run header + Outcome figures fixtures.json → rundetail pins.
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

// devRunStages pins fixtures.json → rundetail.stages in authored order: the four-step
// pipeline, all done (the last drops its trailing connector).
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

// devRunParams pins fixtures.json → rundetail.params in authored order: the five "as
// configured" run parameters.
var devRunParams = []runKV{
	{K: "Profile", V: "standard"},
	{K: "Cadence", V: "daily · 08:00 + 14:00"},
	{K: "Subjects", V: "214"},
	{K: "Address cap", V: "1,024"},
	{K: "Connect timeout", V: "800ms"},
}

// devRunVantages pins fixtures.json → rundetail.vantages in authored order: three vantages,
// the degraded ap-south-1 carrying an em-dash latency.
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
	// The trigger + tally scalars fixtures.json → drift pins: the active preset (7d),
	// its label, the batch-detail entry (batch_id 1407 ↔ its ISO label), the transition
	// count for the period and the signed vs-previous-period delta. batch_id is carried
	// as a string (the tmpl renders it into /runs/{id} and gates the entry on non-empty).
	devDriftPeriod          = "7d"
	devDriftPeriodLabel     = "Last 7d"
	devDriftHasEvents       = true
	devDriftTruncated       = false
	devDriftBatchID         = "1407"
	devDriftBatchLabel      = "2026-08-22T14:00Z"
	devDriftTransitionCount = 7
	devDriftTransitionDelta = "+2"
)

// devDriftMovement pins fixtures.json → drift.movement: the per-change-kind tally the
// Movement card renders (keyed by change word, looked up in .Kinds vocabulary order).
var devDriftMovement = driftMovement{
	"appeared": 1, "revealed": 1, "withdrawn": 1, "descoped": 1, "returned": 1, "changed": 2,
}

// devDriftDiffLine / devDriftEvent / devDriftGroup mirror one fixtures.json → drift.groups
// entry, in authored order: a batch group (its ISO label, scope meta, and collapsed flag)
// carrying its transition events (change kind + drift family, subject, detail, relative time,
// an optional closure reason, and an optional before/after diff).
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

// devDriftGroups pins fixtures.json → drift.groups in authored order: the newest batch first
// (a value move with a nginx-banner diff, an appearance, a withdrawal), the prior batch (a
// return, a reveal), and the collapsed 2026-08-21 batch (a descope, a certificate change).
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
	// devScopeAddressCap is fixtures.json scope.address_cap.
	devScopeAddressCap = 1024

	// devScopeZoneIntervalDays is fixtures.json scope.zone_interval_days, the re-supply
	// dial value the interval-form input echoes.
	devScopeZoneIntervalDays = "30"

	// The refusal golden (states.json scope "refusal") posts devScopeRefusalPost through the
	// seed form; the handler refuses it over the cap and renders the RefusalCallout + FormError.
	// These mirror fixtures.json scope.refusal_fixture.
	devScopeRefusalPost      = "203.0.113.0/20"
	devScopeRefusalInput     = "203.0.113.0/20"
	devScopeRefusalReason    = "Spans 4,096 addresses — the cap is 1,024 per scope."
	devScopeRefusalReachable = "203.0.113.0/22"
	devScopeRefusalFormError = "Refused — over the 1,024-address cap."

	// The exclusion-preview golden (states.json scope "exclusion-preview") types
	// devScopeExclPreviewValue and clicks Preview; the handler renders the firing receipt.
	// These mirror fixtures.json scope.exclusion_preview_fixture.
	devScopeExclPreviewKind     = "subtree"
	devScopeExclPreviewValue    = "staging-4.acmecorp.io"
	devScopeExclPreviewFires    = true
	devScopeExclPreviewHeadline = "1 name and 2 subjects would close as descoped in the next batch."
	devScopeExclPreviewLoss     = "staging-4.acmecorp.io and the service spans it anchors leave the estate; their history stays readable."

	// devScopeOrgQuery / devScopeOrgNotice are fixtures.json scope.org_search_fixture: the
	// org-name search echo and the partial-answer notice. They are asserted by the drift test
	// but do not drive a golden state.
	devScopeOrgQuery  = "Acme Corporation"
	devScopeOrgNotice = "3 registries answered — 1 new proposal for \"Acme Corporation\". RIPE paths are off until you accept their terms."
)

// devScopeSeedRow mirrors one fixtures.json scope.seeds entry.
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

// devScopeTreeLeaf / devScopeTreeRoot mirror fixtures.json scope.name_tree.
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

// devScopeCovMsg mirrors one fixtures.json scope.coverage_msgs entry.
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

// devScopeProposalRow mirrors one fixtures.json scope.proposals entry.
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

// devScopeExclusionRow mirrors one fixtures.json scope.exclusions entry.
type devScopeExclusionRow struct {
	ID    string
	Kind  string
	Value string
}

var devScopeExclusions = []devScopeExclusionRow{
	{ID: "x1", Kind: "subtree", Value: "old-blog.acmecorp.io"},
	{ID: "x2", Kind: "address", Value: "203.0.113.128/25"},
}

// scopeOverlay carries the transient state a scope golden's POST handler overlays on the
// pinned fixture: a RefusalCallout (the over-cap seed form), or the exclusion Preview receipt.
// The zero value is the default state.
type scopeOverlay struct {
	formScope   string
	formError   string
	refusal     *refusalView
	exclKind    string
	exclValue   string
	exclPreview map[string]any
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
	if ov.refusal != nil {
		data["Refusal"] = ov.refusal
	}
	if ov.exclPreview != nil {
		data["ExclPreview"] = ov.exclPreview
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
			ov.refusal = &ref
			ov.formError = overCapFormError(devScopeAddressCap)
		}
	}
	return s.scopeFixtureData(acct, ov)
}

// scopeFixtureDataPreview is the "exclusion-preview" golden's render (states.json): the pinned
// corpus plus the firing Preview receipt for the staging-4.acmecorp.io subtree exclusion.
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
	// devSignalsDetectedBy is fixtures.json signals.detected_by: the vantage every row's drawer
	// names as the detector.
	devSignalsDetectedBy = "vantage eu-west-1"

	// The open-tab scalars fixtures.json signals pins: 47 open (the tab badge + Total), 10 shown
	// on page 1 of 5, with the authored page-info label. Only ten rows are authored — the count
	// is the design's, not a derivation over a 47-row slice.
	devSignalsOpenCount = 47
	devSignalsShown     = 10
	devSignalsPageCount = 5
	devSignalsPageInfo  = "1–10 of 47"

	// devSignalsDiscovered is the fixed discovery instant the span-derived "Asset discovered"
	// history entry renders (fixtures.json signals.history_rule).
	devSignalsDiscovered = "2026-08-12"
)

// devSignalsSevOptions is the severity listbox vocabulary (fixtures.json signals — the filter
// options), in authored order. "All severities" is the unfiltered default.
var devSignalsSevOptions = []string{"All severities", "Critical", "High", "Medium", "Low", "Info"}

// devSignalRow mirrors one fixtures.json signals row (open or withdrawn): the SIG id + view key,
// the severity token + its capitalised label, the human title, the asset/ip/port, the relative
// seen + the two instants, the nullable CVE, the rule tags, the description prose, and the rule
// ref (name + version) split from the fixture's [name, version] pair.
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

// devSignalsOpen pins fixtures.json signals.rows in authored order (default sort sev·asc): the ten
// open rows shown on page 1.
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

// devSignalsAnnotation is one fixtures.json signals.annotations entry: the operator's accepted-risk
// reason, keyed on the row's view key (SIG id).
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

// devSignalsDiffLine / devSignalsDiff mirror one fixtures.json signals.diffs entry: the drift join
// the drawer shows for a subject (a titled block of typed before/after lines).
type devSignalsDiffLine struct {
	Type string
	Text string
}
type devSignalsDiff struct {
	Title string
	Lines []devSignalsDiffLine
}

// devSignalsDiffs pins fixtures.json signals.diffs: SIG-1042 (open ports) and SIG-1036 (service
// banner) carry a drift transition.
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

// signalsRowMap shapes one fixture row for the table (the holes the frozen tmpl's row reads),
// carrying the per-row descope link (filters preserved) and the withdrawn flag (the withdrawn tab
// draws the mark).
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

	// The active tab's rows, in authored order (default sort sev·asc is the authored order).
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

// devDashScanningVariant is the states.json dashboard `scanning` variant token: it pins .Scanning
// (the running-scan Progress row + live pulse) and the .ScanDetail figure.
const devDashScanningVariant = "scanning"

// devDashScanDetail is fixtures.json dashboard.scan_detail — the running-scan Progress detail the
// `scanning` variant renders.
const devDashScanDetail = "214 subjects queued"

// devDashSchedule is fixtures.json dashboard.scan_schedule: the header sub-line's two instants.
var devDashSchedule = map[string]any{"HasLast": true, "LastAgo": "38m", "HasNext": true, "NextIn": "5h 22m"}

// devDashUnavailable is fixtures.json dashboard.unavailable: the missed-check vantage the banner names.
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

// devDashStatBand pins fixtures.json dashboard.stat_band in authored order (five cells).
var devDashStatBand = []devDashStat{
	{label: "Open signals", value: "47", liveWhenScanning: true, hasDelta: true, change: 3, tone: "bad", caption: "vs last scan"},
	{label: "Critical", value: "3", hasDelta: true, change: -1, tone: "good", caption: "1 withdrawn today"},
	{label: "Assets watched", value: "1,284", hasDelta: true, change: 12, tone: "neutral", caption: "8 domains · 3 ranges"},
	{label: "Exposed services", value: "216", hasDelta: true, change: 4, tone: "bad", caption: "across 62 IPs"},
	{label: "Certs expiring ≤30d", value: "9", hasDelta: true, change: -2, tone: "good", caption: "next: 2026-08-29"},
}

// devDashSevBars pins fixtures.json dashboard.sev_bars in ramp order (critical→info).
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

// devDashSilentZone pins fixtures.json dashboard.silent_zone: the Coverage card's staleness callout.
var devDashSilentZone = &dashSilentZone{Bound: "9d", Text: "zone transfer for internal.acmecorp.io"}

// devDashVantages pins fixtures.json dashboard.vantages in authored order (ap-south-1 unavailable).
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
