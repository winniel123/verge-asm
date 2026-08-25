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

// fixtureSigninPackage mirrors the design-owned fixtures.json → signin slice the screen-4 dev
// affordances pin in devfixtures.go: the build version, provider set (slug/name/mark), the
// well-known reset/invite tokens + invite role, the accepted TOTP code, the enroll secret, and
// the recovery-code set.
type fixtureSigninPackage struct {
	Signin struct {
		Version      string `json:"version"`
		SSOProviders []struct {
			Slug string `json:"slug"`
			Name string `json:"name"`
			Mark string `json:"mark"`
		} `json:"sso_providers"`
		ResetToken    string   `json:"reset_token"`
		InviteToken   string   `json:"invite_token"`
		InviteRole    string   `json:"invite_role"`
		TotpAcceptCode string  `json:"totp_accept_code"`
		EnrollSecret  string   `json:"enroll_secret"`
		RecoveryCodes []string `json:"recovery_codes"`
	} `json:"signin"`
}

// TestSigninFixtureMatchesPackage is the byte-exactness gate for the screen-4 conversion: every
// value devfixtures.go pins is folded back through the frozen fixtures.json → signin slice, so a
// drift between the seed/dev affordances and the frozen package fails here rather than in a
// screenshot diff — exactly as TestProfileFixtureMatchesPackage guards the Profile slice. It
// also asserts the repo's Mark derivation reproduces the fixture's mark, and that the recovery
// count matches the enrolment count.
func TestSigninFixtureMatchesPackage(t *testing.T) {
	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures.json: %v", err)
	}
	var f fixtureSigninPackage
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixtures.json: %v", err)
	}
	s := f.Signin

	if s.Version != devFixtureVersion {
		t.Errorf("version drift: fixtures.json = %q, devFixtureVersion = %q", s.Version, devFixtureVersion)
	}
	if s.ResetToken != devFixtureResetToken {
		t.Errorf("reset token drift: fixtures.json = %q, devFixtureResetToken = %q", s.ResetToken, devFixtureResetToken)
	}
	if s.InviteToken != devFixtureInviteToken {
		t.Errorf("invite token drift: fixtures.json = %q, devFixtureInviteToken = %q", s.InviteToken, devFixtureInviteToken)
	}
	if s.InviteRole != devFixtureInviteRole {
		t.Errorf("invite role drift: fixtures.json = %q, devFixtureInviteRole = %q", s.InviteRole, devFixtureInviteRole)
	}
	if s.TotpAcceptCode != devFixtureTOTPCode {
		t.Errorf("totp code drift: fixtures.json = %q, devFixtureTOTPCode = %q", s.TotpAcceptCode, devFixtureTOTPCode)
	}
	if s.EnrollSecret != devFixtureEnrollSecret {
		t.Errorf("enroll secret drift: fixtures.json = %q, devFixtureEnrollSecret = %q", s.EnrollSecret, devFixtureEnrollSecret)
	}

	// Providers — slug, name, and the repo-derived Mark all equal the fixture, in order.
	if len(s.SSOProviders) != len(devSigninProviders) {
		t.Fatalf("provider count drift: fixtures.json = %d, devSigninProviders = %d", len(s.SSOProviders), len(devSigninProviders))
	}
	for i, want := range s.SSOProviders {
		got := devSigninProviders[i]
		if got.slug != want.Slug || got.name != want.Name || got.mark != want.Mark {
			t.Errorf("provider[%d] drift: fixtures.json = {%q,%q,%q}, seeder = {%q,%q,%q}",
				i, want.Slug, want.Name, want.Mark, got.slug, got.name, got.mark)
		}
		if m := ssoMark(want.Name); m != want.Mark {
			t.Errorf("provider[%d] Mark derivation drift: ssoMark(%q) = %q, fixtures.json = %q", i, want.Name, m, want.Mark)
		}
	}

	// Recovery codes — exact set + order, and the count matches the enrolment count.
	if len(s.RecoveryCodes) != len(devFixtureRecoveryCodes) {
		t.Fatalf("recovery count drift: fixtures.json = %d, devFixtureRecoveryCodes = %d", len(s.RecoveryCodes), len(devFixtureRecoveryCodes))
	}
	if len(s.RecoveryCodes) != recoveryCodeCount {
		t.Errorf("recovery count %d != enrolment count %d", len(s.RecoveryCodes), recoveryCodeCount)
	}
	for i, want := range s.RecoveryCodes {
		if devFixtureRecoveryCodes[i] != want {
			t.Errorf("recovery[%d] drift: fixtures.json = %q, seeder = %q", i, want, devFixtureRecoveryCodes[i])
		}
	}
}

// fixtureSetupPackage mirrors the design-owned fixtures.json → setup slice the screen-5 dev
// affordance pins in devfixtures.go: the empty-seed variant and the single-use setup token the
// dev seed route (/dev/seed/empty) reopens the first-run window with.
type fixtureSetupPackage struct {
	Setup struct {
		Variant string `json:"variant"`
		Token   string `json:"token"`
	} `json:"setup"`
}

// TestSetupFixtureMatchesPackage is the byte-exactness gate for the screen-5 conversion: the
// setup token devfixtures.go pins (devFixtureSetupToken) equals the frozen fixtures.json →
// setup.token, and the fixture's seed variant is the "empty" one the capture harness + dev route
// realize — so a drift between the dev affordance and the frozen package fails here rather than
// in a screenshot diff, exactly as TestSigninFixtureMatchesPackage guards the SignIn slice.
func TestSetupFixtureMatchesPackage(t *testing.T) {
	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures.json: %v", err)
	}
	var f fixtureSetupPackage
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixtures.json: %v", err)
	}
	if f.Setup.Token != devFixtureSetupToken {
		t.Errorf("setup token drift: fixtures.json = %q, devFixtureSetupToken = %q", f.Setup.Token, devFixtureSetupToken)
	}
	if f.Setup.Variant != "empty" {
		t.Errorf("setup variant drift: fixtures.json = %q, want %q (the dev seed route empties accounts)", f.Setup.Variant, "empty")
	}
}

// fixtureCoveragePackage mirrors the fixtures.json coverage slice the screen-6 dev fixture pins
// in devfixtures.go (devCoverageMeters/Messages/Gaps/Unevaluables/StaleZones). total and bound are
// pointers/optional so an absent JSON key (a name-scope census meter; a message with no staleness
// figure) round-trips as nil/"".
type fixtureCoveragePackage struct {
	Coverage struct {
		Meters []struct {
			Label   string `json:"label"`
			Counted int    `json:"counted"`
			Total   *int   `json:"total"`
			Unit    string `json:"unit"`
			Detail  string `json:"detail"`
		} `json:"meters"`
		Messages []struct {
			Kind    string `json:"kind"`
			Badge   string `json:"badge"`
			Bound   string `json:"bound"`
			Subject string `json:"subject"`
			Text    string `json:"text"`
			When    string `json:"when"`
			ISO     string `json:"iso"`
		} `json:"messages"`
		Gaps []struct {
			Subject  string `json:"subject"`
			Gap      string `json:"gap"`
			Expected string `json:"expected"`
			Since    string `json:"since"`
		} `json:"gaps"`
		Unevaluable []struct {
			ID      string `json:"id"`
			Version int    `json:"version"`
			Why     string `json:"why"`
		} `json:"unevaluable"`
		StaleZones []struct {
			Zone string `json:"zone"`
			Age  string `json:"age"`
		} `json:"stale_zones"`
	} `json:"coverage"`
}

// TestCoverageFixtureMatchesPackage is the byte-exactness gate for the screen-6 conversion: every
// value the dev fixture pins (devfixtures.go, served by coveragePage under devMode) equals the
// frozen fixtures.json coverage slice, in authored order — so a drift between the served candidate
// and the golden (which composes the same fixture statically) fails here rather than in a
// screenshot diff, exactly as TestSigninFixtureMatchesPackage guards the SignIn slice. It also
// asserts the *int total round-trips (address scope set, name scope nil), the anchor of #19c.
func TestCoverageFixtureMatchesPackage(t *testing.T) {
	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures.json: %v", err)
	}
	var f fixtureCoveragePackage
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixtures.json: %v", err)
	}

	if len(f.Coverage.Meters) != len(devCoverageMeters) {
		t.Fatalf("meters length drift: fixtures.json = %d, pinned = %d", len(f.Coverage.Meters), len(devCoverageMeters))
	}
	for i, m := range f.Coverage.Meters {
		p := devCoverageMeters[i]
		if m.Label != p.label || m.Counted != p.counted || m.Unit != p.unit || m.Detail != p.detail {
			t.Errorf("meter %d drift: fixtures.json = %+v, pinned = %+v", i, m, p)
		}
		switch {
		case m.Total == nil && p.total != nil, m.Total != nil && p.total == nil:
			t.Errorf("meter %d total presence drift: fixtures.json nil=%v, pinned nil=%v", i, m.Total == nil, p.total == nil)
		case m.Total != nil && p.total != nil && *m.Total != *p.total:
			t.Errorf("meter %d total drift: fixtures.json = %d, pinned = %d", i, *m.Total, *p.total)
		}
	}

	if len(f.Coverage.Messages) != len(devCoverageMessages) {
		t.Fatalf("messages length drift: fixtures.json = %d, pinned = %d", len(f.Coverage.Messages), len(devCoverageMessages))
	}
	for i, m := range f.Coverage.Messages {
		p := devCoverageMessages[i]
		if m.Kind != p.kind || m.Badge != p.badge || m.Bound != p.bound || m.Subject != p.subject || m.Text != p.text || m.When != p.when || m.ISO != p.iso {
			t.Errorf("message %d drift:\n fixtures.json = %+v\n pinned        = %+v", i, m, p)
		}
	}

	if len(f.Coverage.Gaps) != len(devCoverageGaps) {
		t.Fatalf("gaps length drift: fixtures.json = %d, pinned = %d", len(f.Coverage.Gaps), len(devCoverageGaps))
	}
	for i, g := range f.Coverage.Gaps {
		p := devCoverageGaps[i]
		if g.Subject != p.subject || g.Gap != p.gap || g.Expected != p.expected || g.Since != p.since {
			t.Errorf("gap %d drift: fixtures.json = %+v, pinned = %+v", i, g, p)
		}
	}

	if len(f.Coverage.Unevaluable) != len(devCoverageUnevaluables) {
		t.Fatalf("unevaluable length drift: fixtures.json = %d, pinned = %d", len(f.Coverage.Unevaluable), len(devCoverageUnevaluables))
	}
	for i, u := range f.Coverage.Unevaluable {
		p := devCoverageUnevaluables[i]
		if u.ID != p.id || u.Version != p.version || u.Why != p.why {
			t.Errorf("unevaluable %d drift: fixtures.json = %+v, pinned = %+v", i, u, p)
		}
	}

	if len(f.Coverage.StaleZones) != len(devCoverageStaleZones) {
		t.Fatalf("stale_zones length drift: fixtures.json = %d, pinned = %d", len(f.Coverage.StaleZones), len(devCoverageStaleZones))
	}
	for i, z := range f.Coverage.StaleZones {
		p := devCoverageStaleZones[i]
		if z.Zone != p.zone || z.Age != p.age {
			t.Errorf("stale_zone %d drift: fixtures.json = %+v, pinned = %+v", i, z, p)
		}
	}
}

// fixtureExposurePackage mirrors the fixtures.json exposure slice the screen-7 dev fixture pins
// in devfixtures.go (the summary band, the +2 exposed delta, the withheld variant and the six
// board rows).
type fixtureExposurePackage struct {
	Exposure struct {
		Exposed         int    `json:"exposed"`
		HasDeltas       bool   `json:"has_deltas"`
		ExposedDelta    int    `json:"exposed_delta"`
		Firewalled      int    `json:"firewalled"`
		NotReached      int    `json:"not_reached"`
		WithheldVariant string `json:"withheld_variant"`
		Rows            []struct {
			Asset    string `json:"asset"`
			Svc      string `json:"svc"`
			Internal string `json:"internal"`
			Internet string `json:"internet"`
			Since    string `json:"since"`
		} `json:"rows"`
	} `json:"exposure"`
}

// TestExposureFixtureMatchesPackage is the byte-exactness gate for the screen-7 conversion: every
// value the dev fixture pins (devfixtures.go, served by exposurePage under devMode) equals the
// frozen fixtures.json exposure slice, in authored order — so a drift between the served candidate
// and the golden (which composes the same fixture statically) fails here rather than in a
// screenshot diff, exactly as TestCoverageFixtureMatchesPackage guards the Coverage slice.
func TestExposureFixtureMatchesPackage(t *testing.T) {
	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures.json: %v", err)
	}
	var f fixtureExposurePackage
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixtures.json: %v", err)
	}
	e := f.Exposure

	if e.Exposed != devExposureExposed {
		t.Errorf("exposed drift: fixtures.json = %d, pinned = %d", e.Exposed, devExposureExposed)
	}
	if e.HasDeltas != devExposureHasDeltas {
		t.Errorf("has_deltas drift: fixtures.json = %v, pinned = %v", e.HasDeltas, devExposureHasDeltas)
	}
	if e.ExposedDelta != devExposureExposedDelta {
		t.Errorf("exposed_delta drift: fixtures.json = %d, pinned = %d", e.ExposedDelta, devExposureExposedDelta)
	}
	if e.Firewalled != devExposureFirewalled {
		t.Errorf("firewalled drift: fixtures.json = %d, pinned = %d", e.Firewalled, devExposureFirewalled)
	}
	if e.NotReached != devExposureNotReached {
		t.Errorf("not_reached drift: fixtures.json = %d, pinned = %d", e.NotReached, devExposureNotReached)
	}
	if e.WithheldVariant != devExposureWithheldVariant {
		t.Errorf("withheld_variant drift: fixtures.json = %q, pinned = %q", e.WithheldVariant, devExposureWithheldVariant)
	}

	if len(e.Rows) != len(devExposureRows) {
		t.Fatalf("rows length drift: fixtures.json = %d, pinned = %d", len(e.Rows), len(devExposureRows))
	}
	for i, r := range e.Rows {
		p := devExposureRows[i]
		if r.Asset != p.asset || r.Svc != p.svc || r.Internal != p.internal || r.Internet != p.internet || r.Since != p.since {
			t.Errorf("row %d drift:\n fixtures.json = %+v\n pinned        = %+v", i, r, p)
		}
	}
}
