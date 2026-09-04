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

func pinnedFixtureClock(t *testing.T) time.Time {
	t.Helper()
	c, err := devFixtureClockTime()
	if err != nil {
		t.Fatalf("parse devFixtureClock: %v", err)
	}
	return c
}

func seedProfileFixtureIntoFake(t *testing.T, f *fakeStore, accountID int64, clock time.Time) {
	t.Helper()

	// Seeded ids come off the fake's own cursor, so a later login still mints a distinct session.
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

func TestProfileFixtureRendersAgainstPinnedClock(t *testing.T) {
	f := newFakeStore()
	acct := seedAccount(t, f, devProfileUsername, roleAdmin, "hunter2hunter2")
	clock := pinnedFixtureClock(t)
	seedProfileFixtureIntoFake(t, f, acct.ID, clock)

	base := startAt(t, f, clock)
	c := login(t, base, devProfileUsername, "hunter2hunter2")
	got := getBody(t, c, base+"/profile", http.StatusOK)

	for _, want := range []string{
		"Firefox · macOS", "CLI · verge@build-07", "Safari · iOS",
		"198.51.100.7", "203.0.113.44", "198.51.100.31",
		"now</td>", "2h</td>", "3d</td>",
		"Okta", "ola.perez@acmecorp.io", "2026-06-30",
		"/profile/sso/google/link", "Google",
		"laptop-cli", "grafana-readonly", "vg_pat_9f3k…", "vg_pat_x81m…",
		"2026-05-02", "2026-07-19", "14d</td>",
		"ci-export", "vg_pat_r55q…", ">never</span>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("profile fixture render missing %q; body: %s", want, got)
		}
	}

	if strings.Contains(got, "/profile/sso/okta/link") {
		t.Fatalf("linked Okta should not appear in the linkable list; body: %s", got)
	}
}

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

func TestProfileTOTPOffBranchRenders(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "noteotp", roleViewer, "hunter2hunter2")
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
