package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// --- screen 3 Profile fixtures-mode determinism (#541) ----------------------
//
// These exercise the SEEDER's derivations and the fixtures-mode handler affordances through
// the real render path (relTime / sessionDeviceFromUA / newPersonalToken), against the pinned
// fixture clock. They are the gate before the pixel harness: a drift between the seed and the
// frozen render fails here, not in a screenshot. The byte-exactness of the pinned values vs
// fixtures.json is asserted separately by TestProfileFixtureMatchesPackage.

// pinnedFixtureClock is the frozen fixture instant a VERGE_DEV server pins (fixtures.json →
// clock). Sessions/tokens are stamped as offsets from it so relTime renders the fixture's
// relative tokens.
func pinnedFixtureClock(t *testing.T) time.Time {
	t.Helper()
	c, err := devFixtureClockTime()
	if err != nil {
		t.Fatalf("parse devFixtureClock: %v", err)
	}
	return c
}

// seedProfileFixtureIntoFake mirrors seedProfileFixtures against the in-memory fake: it
// appends the three sessions, the Okta identity + Google linkable provider, and the two
// tokens, all with the same clock-relative / date-only timestamps the pg seeder writes. It
// leaves TOTP alone so a password-only login still resolves (the pg path reaches the account
// via the dev session mint, which bypasses the challenge); the TOTP-on render is covered by
// TestProfile2FAStatusEnabled and TestProfileFixtureMatchesPackage.
func seedProfileFixtureIntoFake(t *testing.T, f *fakeStore, accountID int64, clock time.Time) {
	t.Helper()

	// Sessions — IDs from the fake's current cursor so a later login mints a distinct one.
	for _, ps := range devProfileSessions {
		last := clock.Add(-ps.lastOffset)
		id := f.sessionNextID
		f.sessions = append(f.sessions, db.Session{
			ID: id, AccountID: accountID, TokenHash: "seed-session-" + strconv.FormatInt(id, 10),
			CreatedAt:  pgtype.Timestamptz{Time: last, Valid: true},
			LastSeenAt: pgtype.Timestamptz{Time: last, Valid: true},
			UserAgent:  ps.userAgent, Ip: ps.ip,
			ExpiresAt: pgtype.Timestamptz{Time: clock.Add(12 * time.Hour), Valid: true},
		})
		f.sessionNextID++
	}

	// SSO — Okta linked (excluded from the linkable list) + Google linkable.
	f.ssoProviders = append(f.ssoProviders,
		fakeSSOProvider{id: 1, slug: devProfileSSOProviderSlug, name: devProfileSSOProviderName, enabled: true},
		fakeSSOProvider{id: 2, slug: devProfileLinkableSlug, name: devProfileLinkableName, enabled: true},
	)
	f.ssoNextID = 2
	linkedAt, err := devFixtureDate(devProfileSSOLinkedAt)
	if err != nil {
		t.Fatalf("parse sso linked_at: %v", err)
	}
	f.ssoIdentities = append(f.ssoIdentities, fakeSSOIdentity{
		id: 1, providerID: 1, accountID: accountID, sub: "okta|ola",
		displayName: devProfileSSODisplayName, createdAt: linkedAt,
	})
	f.ssoIdentNextID = 1

	// Tokens.
	for i, pt := range devProfileTokens {
		created, derr := devFixtureDate(pt.created)
		if derr != nil {
			t.Fatalf("parse token created: %v", derr)
		}
		lastUsedAt := pgtype.Timestamptz{Time: clock.Add(-pt.lastOffset), Valid: true}
		if pt.lastNull {
			lastUsedAt = pgtype.Timestamptz{}
		}
		f.personalTokens = append(f.personalTokens, db.PersonalToken{
			ID: int64(i + 1), AccountID: accountID, Name: pt.name, Prefix: pt.prefix,
			TokenHash:  "seed-token-" + strconv.Itoa(i),
			CreatedAt:  pgtype.Timestamptz{Time: created, Valid: true},
			LastUsedAt: lastUsedAt,
		})
	}
	f.tokenNextID = int64(len(devProfileTokens) + 1)
}

// The seeded Profile renders every fixture value against the pinned clock: the three session
// devices (including the CLI session derived from its verge-cli user-agent), their IPs and
// bare relative last-active tokens (now / 2h / 3d), the Okta linked identity with a date-only
// UTC linked date, Google as the only linkable provider, and the two tokens with date-only
// created dates and bare relative last-used tokens (2h / 14d).
func TestProfileFixtureRendersAgainstPinnedClock(t *testing.T) {
	f := newFakeStore()
	acct := seedAccount(t, f, devProfileUsername, roleAdmin, "hunter2hunter2")
	clock := pinnedFixtureClock(t)
	seedProfileFixtureIntoFake(t, f, acct.ID, clock)

	base := startAt(t, f, clock)
	c := login(t, base, devProfileUsername, "hunter2hunter2")
	got := getBody(t, c, base+"/profile", http.StatusOK)

	for _, want := range []string{
		// Sessions: device (UA-derived, incl. CLI), IP, bare relative last-active.
		"Firefox · macOS", "CLI · verge@build-07", "Safari · iOS",
		"198.51.100.7", "203.0.113.44", "198.51.100.31",
		"now</td>", "2h</td>", "3d</td>",
		// SSO: Okta linked identity, date-only linked date, Google linkable.
		"Okta", "ola.perez@acmecorp.io", "2026-06-30",
		"/profile/sso/google/link", "Google",
		// Tokens: names, non-secret prefixes, date-only created, bare relative last-used.
		"laptop-cli", "grafana-readonly", "vg_pat_9f3k…", "vg_pat_x81m…",
		"2026-05-02", "2026-07-19", "14d</td>",
		// ci-export (#390): a never-used token — NULL last_used_at renders "never".
		"ci-export", "vg_pat_r55q…", ">never</span>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("profile fixture render missing %q; body: %s", want, got)
		}
	}

	// Okta is already linked, so it is NOT offered again as a linkable provider.
	if strings.Contains(got, "/profile/sso/okta/link") {
		t.Fatalf("linked Okta should not appear in the linkable list; body: %s", got)
	}
}

// In a VERGE_DEV build the token-create handler reveals the fixture-deterministic plaintext
// (vg_pat_cigolden0example) so the minted-dialog golden is pixel-stable; a real build keeps
// the crypto/rand draw (asserted by TestProfileTokenRevealOnce).
func TestProfileDeterministicMintInDevMode(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, devProfileUsername, roleAdmin, "hunter2hunter2")
	srv := newServer(f, testKey, "", fixedClock())
	srv.devMode = true
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)

	c := login(t, ts.URL, devProfileUsername, "hunter2hunter2")
	page := body(t, postForm(t, c, ts.URL+"/profile/tokens", url.Values{"name": {"ci-golden"}}))

	if !strings.Contains(page, devFixtureMintedToken) {
		t.Fatalf("dev mint did not reveal the fixture token %q; body: %s", devFixtureMintedToken, page)
	}
	if got := mintedRE.FindString(page); got != "" {
		t.Fatalf("dev mint revealed a crypto/rand token %q, want the deterministic fixture value", got)
	}
	if len(f.personalTokens) != 1 {
		t.Fatalf("tokens stored = %d, want 1", len(f.personalTokens))
	}
	if f.personalTokens[0].TokenHash == devFixtureMintedToken {
		t.Fatalf("plaintext token was stored instead of its hash")
	}
}

// The TOTP-off code branch (the "two-factor off" badge + Enable-two-factor form) is reachable
// and correct for an account with TotpEnabled=false. The frozen fixture account is seeded
// TOTP-ON (SPEC-CHANGE #18 reconciliation: the package governs over the ticket's "no-TOTP
// viewer" prose), so this branch is proven by this unit test rather than a golden.
func TestProfileTOTPOffBranchRenders(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "noteotp", roleViewer, "hunter2hunter2") // TotpEnabled defaults false
	base := start(t, f, "")
	c := login(t, base, "noteotp", "hunter2hunter2")

	got := getBody(t, c, base+"/profile", http.StatusOK)
	for _, want := range []string{
		"two-factor off",
		`action="/account/totp/enable"`,
		"Enable two-factor",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("TOTP-off branch missing %q; body: %s", want, got)
		}
	}
	if strings.Contains(got, "two-factor enabled") {
		t.Fatalf("TOTP-on badge rendered for a no-TOTP account; body: %s", got)
	}
}
