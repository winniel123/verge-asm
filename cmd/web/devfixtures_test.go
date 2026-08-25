package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// fixtureDevPackage mirrors the slices of design-system/fixtures/fixtures.json the
// dev harness affordances pin in code: the deterministic incident id and the login
// accounts. This test is the byte-exactness gate BEFORE the pixel harness — a drift
// between the frozen fixture and the constants devfixtures.go pins (which drive the
// 500 golden and the per-state session mint) fails here rather than in a screenshot
// diff, exactly as inventory_fixture_test.go guards the inventory corpus.
type fixtureDevPackage struct {
	Accounts []struct {
		Username    string `json:"username"`
		Role        string `json:"role"`
		DevPassword string `json:"dev_password"`
	} `json:"accounts"`
	Error struct {
		IncidentID string `json:"incident_id"`
	} `json:"error"`
}

func loadFixtureDevPackage(t *testing.T) fixtureDevPackage {
	t.Helper()
	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures.json: %v", err)
	}
	var f fixtureDevPackage
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixtures.json: %v", err)
	}
	return f
}

// TestDevFixturesMatchPackage asserts the dev constants never drift from the frozen
// package: the deterministic incident id and every seeded account (username, role,
// password) equal fixtures.json's, in order.
func TestDevFixturesMatchPackage(t *testing.T) {
	f := loadFixtureDevPackage(t)

	if f.Error.IncidentID != devFixtureIncidentID {
		t.Errorf("incident id drift: fixtures.json = %q, devFixtureIncidentID = %q",
			f.Error.IncidentID, devFixtureIncidentID)
	}

	if len(f.Accounts) != len(devFixtureAccounts) {
		t.Fatalf("account count drift: fixtures.json = %d, devFixtureAccounts = %d",
			len(f.Accounts), len(devFixtureAccounts))
	}
	for i, want := range f.Accounts {
		got := devFixtureAccounts[i]
		if got.username != want.Username || got.role != want.Role || got.password != want.DevPassword {
			t.Errorf("account[%d] drift: fixtures.json = {%q,%q,%q}, devFixtureAccounts = {%q,%q,%q}",
				i, want.Username, want.Role, want.DevPassword, got.username, got.role, got.password)
		}
	}
}

// fixtureProfilePackage mirrors the design-owned fixtures.json → profile slice (plus the
// top-level clock) the screen-3 seeder pins in devfixtures.go.
type fixtureProfilePackage struct {
	Clock   string `json:"clock"`
	Profile struct {
		Account struct {
			Username    string `json:"username"`
			Role        string `json:"role"`
			Created     string `json:"created"`
			TotpEnabled bool   `json:"totp_enabled"`
			Initials    string `json:"initials"`
		} `json:"account"`
		Sessions []struct {
			Device     string `json:"device"`
			IP         string `json:"ip"`
			LastActive string `json:"last_active"`
			Current    bool   `json:"current"`
		} `json:"sessions"`
		SSOIdentities []struct {
			Provider    string `json:"provider"`
			DisplayName string `json:"display_name"`
			LinkedAt    string `json:"linked_at"`
		} `json:"sso_identities"`
		SSOProviders []struct {
			Slug string `json:"slug"`
			Name string `json:"name"`
		} `json:"sso_providers"`
		Tokens []struct {
			Name    string `json:"name"`
			Prefix  string `json:"prefix"`
			Created string `json:"created"`
			Last    string `json:"last"`
		} `json:"tokens"`
		MintedTokenFixture string `json:"minted_token_fixture"`
	} `json:"profile"`
}

func loadFixtureProfilePackage(t *testing.T) fixtureProfilePackage {
	t.Helper()
	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures.json: %v", err)
	}
	var f fixtureProfilePackage
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixtures.json: %v", err)
	}
	return f
}

// TestProfileFixtureMatchesPackage is the byte-exactness gate for the screen-3 seed: every
// value devfixtures.go pins is folded back through the frozen fixtures.json → profile slice
// (and the pinned clock), and the clock-relative / UA-derived renders are reproduced with
// the real formatters (relTime, sessionDeviceFromUA) — so any drift between the seeder and
// the frozen package fails here rather than in a screenshot, exactly as
// TestDevFixturesMatchPackage guards the ErrorPage slice.
func TestProfileFixtureMatchesPackage(t *testing.T) {
	f := loadFixtureProfilePackage(t)
	p := f.Profile

	if f.Clock != devFixtureClock {
		t.Errorf("clock drift: fixtures.json = %q, devFixtureClock = %q", f.Clock, devFixtureClock)
	}
	clock, err := devFixtureClockTime()
	if err != nil {
		t.Fatalf("parse devFixtureClock: %v", err)
	}

	// Account.
	if p.Account.Username != devProfileUsername {
		t.Errorf("account username drift: %q vs %q", p.Account.Username, devProfileUsername)
	}
	if p.Account.Role != roleAdmin {
		t.Errorf("account role drift: fixtures.json = %q, seeder seeds admin", p.Account.Role)
	}
	if !p.Account.TotpEnabled {
		t.Errorf("account totp_enabled drift: fixtures.json = %v, seeder seeds true", p.Account.TotpEnabled)
	}
	if p.Account.Created != devProfileCreated {
		t.Errorf("account created drift: fixtures.json = %q, devProfileCreated = %q", p.Account.Created, devProfileCreated)
	}
	if _, err := time.Parse("2006-01-02", devProfileCreated); err != nil {
		t.Errorf("account created not a date-only value: %q", devProfileCreated)
	}
	if got := initials(p.Account.Username); got != p.Account.Initials {
		t.Errorf("initials drift: initials(%q) = %q, fixtures.json = %q", p.Account.Username, got, p.Account.Initials)
	}

	if p.MintedTokenFixture != devFixtureMintedToken {
		t.Errorf("minted token drift: fixtures.json = %q, devFixtureMintedToken = %q", p.MintedTokenFixture, devFixtureMintedToken)
	}

	// Sessions — device (derived from the seeded UA), ip, relative last-active, current.
	if len(p.Sessions) != len(devProfileSessions) {
		t.Fatalf("session count drift: fixtures.json = %d, seeder = %d", len(p.Sessions), len(devProfileSessions))
	}
	for i, want := range p.Sessions {
		got := devProfileSessions[i]
		if got.device != want.Device || got.ip != want.IP || got.lastActive != want.LastActive || got.current != want.Current {
			t.Errorf("session[%d] drift: fixtures.json = {%q,%q,%q,%v}, seeder = {%q,%q,%q,%v}",
				i, want.Device, want.IP, want.LastActive, want.Current, got.device, got.ip, got.lastActive, got.current)
		}
		if dev := sessionDeviceFromUA(got.userAgent); dev != want.Device {
			t.Errorf("session[%d] UA→device drift: sessionDeviceFromUA(%q) = %q, want %q", i, got.userAgent, dev, want.Device)
		}
		if rel := profileRelTime(clock.Add(-got.lastOffset), clock); rel != want.LastActive {
			t.Errorf("session[%d] last-active drift: profileRelTime(clock-%s) = %q, want %q", i, got.lastOffset, rel, want.LastActive)
		}
	}

	// SSO — one Okta identity, Google linkable.
	if len(p.SSOIdentities) != 1 || p.SSOIdentities[0].Provider != devProfileSSOProviderName ||
		p.SSOIdentities[0].DisplayName != devProfileSSODisplayName || p.SSOIdentities[0].LinkedAt != devProfileSSOLinkedAt {
		t.Errorf("sso identity drift: fixtures.json = %+v, seeder = {%q,%q,%q}",
			p.SSOIdentities, devProfileSSOProviderName, devProfileSSODisplayName, devProfileSSOLinkedAt)
	}
	if len(p.SSOProviders) != 1 || p.SSOProviders[0].Slug != devProfileLinkableSlug || p.SSOProviders[0].Name != devProfileLinkableName {
		t.Errorf("sso linkable-provider drift: fixtures.json = %+v, seeder = {%q,%q}",
			p.SSOProviders, devProfileLinkableSlug, devProfileLinkableName)
	}

	// Tokens — name, prefix, date-only created, relative last-used.
	if len(p.Tokens) != len(devProfileTokens) {
		t.Fatalf("token count drift: fixtures.json = %d, seeder = %d", len(p.Tokens), len(devProfileTokens))
	}
	for i, want := range p.Tokens {
		got := devProfileTokens[i]
		if got.name != want.Name || got.prefix != want.Prefix || got.created != want.Created || got.last != want.Last {
			t.Errorf("token[%d] drift: fixtures.json = {%q,%q,%q,%q}, seeder = {%q,%q,%q,%q}",
				i, want.Name, want.Prefix, want.Created, want.Last, got.name, got.prefix, got.created, got.last)
		}
		if rel := profileRelTime(clock.Add(-got.lastOffset), clock); rel != want.Last {
			t.Errorf("token[%d] last-used drift: profileRelTime(clock-%s) = %q, want %q", i, got.lastOffset, rel, want.Last)
		}
		if _, err := time.Parse("2006-01-02", got.created); err != nil {
			t.Errorf("token[%d] created not a date-only value: %q", i, got.created)
		}
	}
}
