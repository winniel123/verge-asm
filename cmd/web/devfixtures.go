package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
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
	{username: "ola.perez", role: roleAdmin, password: "verge-dev-1"},  // #nosec G101 -- dev-only fixture login, seeded only under VERGE_DEV
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
	const insToken = `
		INSERT INTO personal_token (account_id, name, prefix, token_hash, created_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
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
