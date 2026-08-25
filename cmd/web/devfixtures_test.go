package main

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
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
		ResetToken     string   `json:"reset_token"`
		InviteToken    string   `json:"invite_token"`
		InviteRole     string   `json:"invite_role"`
		TotpAcceptCode string   `json:"totp_accept_code"`
		EnrollSecret   string   `json:"enroll_secret"`
		RecoveryCodes  []string `json:"recovery_codes"`
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

// fixtureDriftPackage mirrors the fixtures.json drift slice the screen-8 dev fixture pins in
// devfixtures.go (the range-picker presets, the change vocabulary, the trigger + tally scalars,
// the movement map and the batch groups with their events and optional diffs).
type fixtureDriftPackage struct {
	Drift struct {
		Period          string `json:"period"`
		PeriodLabel     string `json:"period_label"`
		HasEvents       bool   `json:"has_events"`
		Truncated       bool   `json:"truncated"`
		FeedLimit       int32  `json:"feed_limit"`
		BatchID         string `json:"batch_id"`
		BatchLabel      string `json:"batch_label"`
		TransitionCount int    `json:"transition_count"`
		TransitionDelta string `json:"transition_delta"`
		Periods         []struct {
			Token string `json:"token"`
			Label string `json:"label"`
		} `json:"periods"`
		Kinds []struct {
			Change string `json:"change"`
			Family string `json:"family"`
		} `json:"kinds"`
		Movement map[string]int `json:"movement"`
		Groups   []struct {
			Label     string `json:"label"`
			Meta      string `json:"meta"`
			Collapsed bool   `json:"collapsed"`
			Events    []struct {
				Change  string `json:"change"`
				Family  string `json:"family"`
				Subject string `json:"subject"`
				Detail  string `json:"detail"`
				Time    string `json:"time"`
				Reason  string `json:"reason"`
				Diff    []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"diff"`
			} `json:"events"`
		} `json:"groups"`
	} `json:"drift"`
}

// TestDriftFixtureMatchesPackage is the byte-exactness gate for the screen-8 conversion: every
// value the dev fixture serves (devfixtures.go driftFixtureData, served by driftPage under
// devMode) equals the frozen fixtures.json drift slice, in authored order — so a drift between
// the served candidate and the golden (which composes the same fixture statically) fails here
// rather than in a screenshot diff, exactly as TestExposureFixtureMatchesPackage guards Exposure.
// It also pins the code-owned vocabulary the tmpl's .Periods/.Kinds holes are fed from
// (driftPeriods / driftKinds) to the fixture's presets and change kinds.
func TestDriftFixtureMatchesPackage(t *testing.T) {
	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures.json: %v", err)
	}
	var f fixtureDriftPackage
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixtures.json: %v", err)
	}
	d := f.Drift

	// Trigger + tally scalars.
	if d.Period != devDriftPeriod {
		t.Errorf("period drift: fixtures.json = %q, pinned = %q", d.Period, devDriftPeriod)
	}
	if d.PeriodLabel != devDriftPeriodLabel {
		t.Errorf("period_label drift: fixtures.json = %q, pinned = %q", d.PeriodLabel, devDriftPeriodLabel)
	}
	if d.HasEvents != devDriftHasEvents {
		t.Errorf("has_events drift: fixtures.json = %v, pinned = %v", d.HasEvents, devDriftHasEvents)
	}
	if d.Truncated != devDriftTruncated {
		t.Errorf("truncated drift: fixtures.json = %v, pinned = %v", d.Truncated, devDriftTruncated)
	}
	if d.FeedLimit != driftFeedLimit {
		t.Errorf("feed_limit drift: fixtures.json = %d, pinned = %d", d.FeedLimit, driftFeedLimit)
	}
	if d.BatchID != devDriftBatchID {
		t.Errorf("batch_id drift: fixtures.json = %q, pinned = %q", d.BatchID, devDriftBatchID)
	}
	if d.BatchLabel != devDriftBatchLabel {
		t.Errorf("batch_label drift: fixtures.json = %q, pinned = %q", d.BatchLabel, devDriftBatchLabel)
	}
	if d.TransitionCount != devDriftTransitionCount {
		t.Errorf("transition_count drift: fixtures.json = %d, pinned = %d", d.TransitionCount, devDriftTransitionCount)
	}
	if d.TransitionDelta != devDriftTransitionDelta {
		t.Errorf("transition_delta drift: fixtures.json = %q, pinned = %q", d.TransitionDelta, devDriftTransitionDelta)
	}

	// The transition count must equal the movement tally sum (the "This period" card total).
	sum := 0
	for _, v := range d.Movement {
		sum += v
	}
	if sum != devDriftTransitionCount {
		t.Errorf("transition_count %d != movement sum %d", devDriftTransitionCount, sum)
	}

	// Range-picker presets: driftPeriods() feeds the .Periods hole, so it must equal the
	// fixture's presets in authored order.
	periods := driftPeriods()
	if len(d.Periods) != len(periods) {
		t.Fatalf("periods length drift: fixtures.json = %d, driftPeriods = %d", len(d.Periods), len(periods))
	}
	for i, p := range d.Periods {
		if p.Token != periods[i].Token || p.Label != periods[i].Label {
			t.Errorf("period %d drift: fixtures.json = {%q %q}, driftPeriods = {%q %q}", i, p.Token, p.Label, periods[i].Token, periods[i].Label)
		}
	}

	// Change vocabulary: driftKinds() feeds the .Kinds hole (and the Movement/legend order).
	kinds := driftKinds()
	if len(d.Kinds) != len(kinds) {
		t.Fatalf("kinds length drift: fixtures.json = %d, driftKinds = %d", len(d.Kinds), len(kinds))
	}
	for i, k := range d.Kinds {
		if k.Change != kinds[i].Change || k.Family != kinds[i].Family {
			t.Errorf("kind %d drift: fixtures.json = {%q %q}, driftKinds = {%q %q}", i, k.Change, k.Family, kinds[i].Change, kinds[i].Family)
		}
	}

	// Movement map.
	if len(d.Movement) != len(devDriftMovement) {
		t.Fatalf("movement length drift: fixtures.json = %d, pinned = %d", len(d.Movement), len(devDriftMovement))
	}
	for k, v := range d.Movement {
		if devDriftMovement[k] != v {
			t.Errorf("movement[%q] drift: fixtures.json = %d, pinned = %d", k, v, devDriftMovement[k])
		}
	}

	// Batch groups, in authored order, with events and diffs.
	if len(d.Groups) != len(devDriftGroups) {
		t.Fatalf("groups length drift: fixtures.json = %d, pinned = %d", len(d.Groups), len(devDriftGroups))
	}
	for gi, g := range d.Groups {
		pg := devDriftGroups[gi]
		if g.Label != pg.label || g.Meta != pg.meta || g.Collapsed != pg.collapsed {
			t.Errorf("group %d drift: fixtures.json = {%q %q collapsed=%v}, pinned = {%q %q collapsed=%v}", gi, g.Label, g.Meta, g.Collapsed, pg.label, pg.meta, pg.collapsed)
		}
		if len(g.Events) != len(pg.events) {
			t.Fatalf("group %d events length drift: fixtures.json = %d, pinned = %d", gi, len(g.Events), len(pg.events))
		}
		for ei, e := range g.Events {
			pe := pg.events[ei]
			if e.Change != pe.change || e.Family != pe.family || e.Subject != pe.subject || e.Detail != pe.detail || e.Time != pe.time || e.Reason != pe.reason {
				t.Errorf("group %d event %d drift:\n fixtures.json = %+v\n pinned        = %+v", gi, ei, e, pe)
			}
			if len(e.Diff) != len(pe.diff) {
				t.Fatalf("group %d event %d diff length drift: fixtures.json = %d, pinned = %d", gi, ei, len(e.Diff), len(pe.diff))
			}
			for di, dl := range e.Diff {
				pd := pe.diff[di]
				if dl.Type != pd.typ || dl.Text != pd.text {
					t.Errorf("group %d event %d diff %d drift: fixtures.json = {%q %q}, pinned = {%q %q}", gi, ei, di, dl.Type, dl.Text, pd.typ, pd.text)
				}
			}
		}
	}
}

// fixtureRunDetailPackage mirrors the fixtures.json rundetail slice the screen-9 dev fixture pins
// (devfixtures.go, served by runPage under devMode): the run header + Outcome figures, the four
// stages, the seven log lines, the nullable degraded callout, the five params and the three
// vantages. Snake_case JSON → the runView PascalCase the frozen rundetail.tmpl reads.
type fixtureRunDetailPackage struct {
	RunDetail struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Status      string `json:"status"`
		Scope       string `json:"scope"`
		Meta        string `json:"meta"`
		Active      bool   `json:"active"`
		Transitions string `json:"transitions"`
		NewSignals  string `json:"new_signals"`
		Stages      []struct {
			Num     int    `json:"num"`
			Title   string `json:"title"`
			Detail  string `json:"detail"`
			Done    bool   `json:"done"`
			Current bool   `json:"current"`
			Last    bool   `json:"last"`
		} `json:"stages"`
		Log []struct {
			Tag   string `json:"tag"`
			Level string `json:"level"`
			Text  string `json:"text"`
		} `json:"log"`
		Degraded *struct {
			Vantage string `json:"vantage"`
			Detail  string `json:"detail"`
		} `json:"degraded"`
		Params []struct {
			K string `json:"k"`
			V string `json:"v"`
		} `json:"params"`
		Vantages []struct {
			Name    string `json:"name"`
			Latency string `json:"latency"`
			Status  string `json:"status"`
		} `json:"vantages"`
	} `json:"rundetail"`
}

// TestRunDetailFixtureMatchesPackage is the byte-exactness gate for the screen-9 conversion: every
// value the dev fixture pins (devfixtures.go, served by runPage under devMode) equals the frozen
// fixtures.json rundetail slice, in authored order — so a drift between the served candidate and the
// golden (which composes the same fixture statically) fails here rather than in a screenshot diff,
// exactly as TestCoverageFixtureMatchesPackage guards the Coverage slice.
func TestRunDetailFixtureMatchesPackage(t *testing.T) {
	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures.json: %v", err)
	}
	var f fixtureRunDetailPackage
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixtures.json: %v", err)
	}
	rd := f.RunDetail

	if rd.ID != devRunDetailID {
		t.Errorf("id drift: fixtures.json = %q, pinned = %q", rd.ID, devRunDetailID)
	}
	if rd.Title != devRunTitle {
		t.Errorf("title drift: fixtures.json = %q, pinned = %q", rd.Title, devRunTitle)
	}
	if rd.Status != devRunStatus {
		t.Errorf("status drift: fixtures.json = %q, pinned = %q", rd.Status, devRunStatus)
	}
	if rd.Scope != devRunScope {
		t.Errorf("scope drift: fixtures.json = %q, pinned = %q", rd.Scope, devRunScope)
	}
	if rd.Meta != devRunMeta {
		t.Errorf("meta drift: fixtures.json = %q, pinned = %q", rd.Meta, devRunMeta)
	}
	if rd.Active != devRunActive {
		t.Errorf("active drift: fixtures.json = %v, pinned = %v", rd.Active, devRunActive)
	}
	if rd.Transitions != devRunTransitions {
		t.Errorf("transitions drift: fixtures.json = %q, pinned = %q", rd.Transitions, devRunTransitions)
	}
	if rd.NewSignals != devRunNewSignals {
		t.Errorf("new_signals drift: fixtures.json = %q, pinned = %q", rd.NewSignals, devRunNewSignals)
	}

	if rd.Degraded == nil {
		t.Fatalf("degraded drift: fixtures.json has no degraded, pinned = %s/%s", devRunDegradedVantage, devRunDegradedDetail)
	}
	if rd.Degraded.Vantage != devRunDegradedVantage || rd.Degraded.Detail != devRunDegradedDetail {
		t.Errorf("degraded drift:\n fixtures.json = %+v\n pinned        = {%s %s}", *rd.Degraded, devRunDegradedVantage, devRunDegradedDetail)
	}

	if len(rd.Stages) != len(devRunStages) {
		t.Fatalf("stages length drift: fixtures.json = %d, pinned = %d", len(rd.Stages), len(devRunStages))
	}
	for i, st := range rd.Stages {
		p := devRunStages[i]
		if st.Num != p.Num || st.Title != p.Title || st.Detail != p.Detail || st.Done != p.Done || st.Current != p.Current || st.Last != p.Last {
			t.Errorf("stage %d drift:\n fixtures.json = %+v\n pinned        = %+v", i, st, p)
		}
	}

	if len(rd.Log) != len(devRunLog) {
		t.Fatalf("log length drift: fixtures.json = %d, pinned = %d", len(rd.Log), len(devRunLog))
	}
	for i, l := range rd.Log {
		p := devRunLog[i]
		if l.Tag != p.Tag || l.Level != p.Level || l.Text != p.Text {
			t.Errorf("log %d drift:\n fixtures.json = %+v\n pinned        = %+v", i, l, p)
		}
	}

	if len(rd.Params) != len(devRunParams) {
		t.Fatalf("params length drift: fixtures.json = %d, pinned = %d", len(rd.Params), len(devRunParams))
	}
	for i, pr := range rd.Params {
		p := devRunParams[i]
		if pr.K != p.K || pr.V != p.V {
			t.Errorf("param %d drift: fixtures.json = %+v, pinned = %+v", i, pr, p)
		}
	}

	if len(rd.Vantages) != len(devRunVantages) {
		t.Fatalf("vantages length drift: fixtures.json = %d, pinned = %d", len(rd.Vantages), len(devRunVantages))
	}
	for i, vt := range rd.Vantages {
		p := devRunVantages[i]
		if vt.Name != p.Name || vt.Latency != p.Latency || vt.Status != p.Status {
			t.Errorf("vantage %d drift: fixtures.json = %+v, pinned = %+v", i, vt, p)
		}
	}
}

// fixtureScopePackage mirrors the fixtures.json scope slice the screen-10 dev fixture pins
// (devfixtures.go): the cap, the seeds, the refusal + exclusion-preview + org-search fixtures,
// custody, zone, name tree, coverage messages, proposals and exclusions.
type fixtureScopePackage struct {
	Scope struct {
		AddressCap int `json:"address_cap"`
		Seeds      []struct {
			ID        string `json:"id"`
			Anchor    string `json:"anchor"`
			Scope     string `json:"scope"`
			IsAddress bool   `json:"is_address"`
		} `json:"seeds"`
		RefusalFixture struct {
			PostValue string `json:"post_value"`
			Input     string `json:"input"`
			Reason    string `json:"reason"`
			Reachable string `json:"reachable"`
			FormError string `json:"form_error"`
		} `json:"refusal_fixture"`
		CustodyScopes []struct {
			ID               string `json:"id"`
			Scope            string `json:"scope"`
			CustodyExtension bool   `json:"custody_extension"`
			Census           int    `json:"census"`
		} `json:"custody_scopes"`
		ZoneScopes []struct {
			ID            string `json:"id"`
			Domain        string `json:"domain"`
			HasFile       bool   `json:"has_file"`
			SuppliedAt    string `json:"supplied_at"`
			IntervalLabel string `json:"interval_label"`
			AgingLabel    string `json:"aging_label"`
		} `json:"zone_scopes"`
		ZoneIntervalDays int `json:"zone_interval_days"`
		NameTree         []struct {
			Label    string `json:"label"`
			Count    int    `json:"count"`
			Sev      string `json:"sev"`
			Children []struct {
				Label string `json:"label"`
				Sev   string `json:"sev"`
			} `json:"children"`
		} `json:"name_tree"`
		CoverageMsgs []struct {
			Kind    string `json:"kind"`
			Badge   string `json:"badge"`
			Bound   string `json:"bound"`
			Subject string `json:"subject"`
			Text    string `json:"text"`
			When    string `json:"when"`
			ISO     string `json:"iso"`
		} `json:"coverage_msgs"`
		Proposals []struct {
			ID     string `json:"id"`
			Value  string `json:"value"`
			Kind   string `json:"kind"`
			Source string `json:"source"`
		} `json:"proposals"`
		OrgSearchFixture struct {
			Org    string `json:"org"`
			Notice string `json:"notice"`
		} `json:"org_search_fixture"`
		Exclusions []struct {
			ID    string `json:"id"`
			Kind  string `json:"kind"`
			Value string `json:"value"`
		} `json:"exclusions"`
		ExclusionPreviewFixture struct {
			PostKind  string `json:"post_kind"`
			PostValue string `json:"post_value"`
			Fires     bool   `json:"fires"`
			Headline  string `json:"headline"`
			Loss      string `json:"loss"`
		} `json:"exclusion_preview_fixture"`
	} `json:"scope"`
}

// TestScopeFixtureMatchesPackage is the byte-exactness gate for the screen-10 conversion: every
// value the dev fixture pins (devfixtures.go, served by seedsPage/declareSeed/previewExclusion
// under devMode) equals the frozen fixtures.json scope slice, in authored order — so a drift
// between the served candidate and the golden (which composes the same fixture statically) fails
// here rather than in a screenshot diff, exactly as TestExposureFixtureMatchesPackage guards the
// Exposure slice.
func TestScopeFixtureMatchesPackage(t *testing.T) {
	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures.json: %v", err)
	}
	var f fixtureScopePackage
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixtures.json: %v", err)
	}
	sc := f.Scope

	if sc.AddressCap != devScopeAddressCap {
		t.Errorf("address_cap drift: fixtures.json = %d, pinned = %d", sc.AddressCap, devScopeAddressCap)
	}
	if got := devScopeZoneIntervalDays; got != itoa(int64(sc.ZoneIntervalDays)) {
		t.Errorf("zone_interval_days drift: fixtures.json = %d, pinned = %q", sc.ZoneIntervalDays, got)
	}

	// Seeds.
	if len(sc.Seeds) != len(devScopeSeeds) {
		t.Fatalf("seeds length drift: fixtures.json = %d, pinned = %d", len(sc.Seeds), len(devScopeSeeds))
	}
	for i, s := range sc.Seeds {
		p := devScopeSeeds[i]
		if s.ID != p.ID || s.Anchor != p.Anchor || s.Scope != p.Scope || s.IsAddress != p.IsAddress {
			t.Errorf("seed %d drift:\n fixtures.json = %+v\n pinned        = %+v", i, s, p)
		}
	}

	// Refusal fixture.
	r := sc.RefusalFixture
	if r.PostValue != devScopeRefusalPost || r.Input != devScopeRefusalInput || r.Reason != devScopeRefusalReason ||
		r.Reachable != devScopeRefusalReachable || r.FormError != devScopeRefusalFormError {
		t.Errorf("refusal_fixture drift:\n fixtures.json = %+v\n pinned        = post=%q input=%q reason=%q reachable=%q formErr=%q",
			r, devScopeRefusalPost, devScopeRefusalInput, devScopeRefusalReason, devScopeRefusalReachable, devScopeRefusalFormError)
	}

	// Custody.
	if len(sc.CustodyScopes) != len(devScopeCustody) {
		t.Fatalf("custody length drift: fixtures.json = %d, pinned = %d", len(sc.CustodyScopes), len(devScopeCustody))
	}
	for i, c := range sc.CustodyScopes {
		p := devScopeCustody[i]
		if c.ID != p.ID || c.Scope != p.Scope || c.CustodyExtension != p.CustodyExtension || c.Census != p.Census {
			t.Errorf("custody %d drift:\n fixtures.json = %+v\n pinned        = %+v", i, c, p)
		}
	}

	// Zone.
	if len(sc.ZoneScopes) != len(devScopeZones) {
		t.Fatalf("zone length drift: fixtures.json = %d, pinned = %d", len(sc.ZoneScopes), len(devScopeZones))
	}
	for i, z := range sc.ZoneScopes {
		p := devScopeZones[i]
		if z.ID != p.ID || z.Domain != p.Domain || z.HasFile != p.HasFile || z.SuppliedAt != p.SuppliedAt ||
			z.IntervalLabel != p.IntervalLabel || z.AgingLabel != p.AgingLabel {
			t.Errorf("zone %d drift:\n fixtures.json = %+v\n pinned        = %+v", i, z, p)
		}
	}

	// Name tree.
	if len(sc.NameTree) != len(devScopeNameTree) {
		t.Fatalf("name_tree length drift: fixtures.json = %d, pinned = %d", len(sc.NameTree), len(devScopeNameTree))
	}
	for i, root := range sc.NameTree {
		pr := devScopeNameTree[i]
		if root.Label != pr.Label || root.Count != pr.Count || root.Sev != pr.Sev {
			t.Errorf("name_tree root %d drift:\n fixtures.json = %+v\n pinned        = %+v", i, root, pr)
		}
		if len(root.Children) != len(pr.Children) {
			t.Fatalf("name_tree root %d children length drift: fixtures.json = %d, pinned = %d", i, len(root.Children), len(pr.Children))
		}
		for j, leaf := range root.Children {
			pl := pr.Children[j]
			if leaf.Label != pl.Label || leaf.Sev != pl.Sev {
				t.Errorf("name_tree leaf %d/%d drift:\n fixtures.json = %+v\n pinned        = %+v", i, j, leaf, pl)
			}
		}
	}

	// Coverage messages.
	if len(sc.CoverageMsgs) != len(devScopeCoverageMsgs) {
		t.Fatalf("coverage_msgs length drift: fixtures.json = %d, pinned = %d", len(sc.CoverageMsgs), len(devScopeCoverageMsgs))
	}
	for i, m := range sc.CoverageMsgs {
		p := devScopeCoverageMsgs[i]
		if m.Kind != p.Kind || m.Badge != p.Badge || m.Bound != p.Bound || m.Subject != p.Subject ||
			m.Text != p.Text || m.When != p.When || m.ISO != p.ISO {
			t.Errorf("coverage_msg %d drift:\n fixtures.json = %+v\n pinned        = %+v", i, m, p)
		}
	}

	// Proposals.
	if len(sc.Proposals) != len(devScopeProposals) {
		t.Fatalf("proposals length drift: fixtures.json = %d, pinned = %d", len(sc.Proposals), len(devScopeProposals))
	}
	for i, p := range sc.Proposals {
		pp := devScopeProposals[i]
		if p.ID != pp.ID || p.Value != pp.Value || p.Kind != pp.Kind || p.Source != pp.Source {
			t.Errorf("proposal %d drift:\n fixtures.json = %+v\n pinned        = %+v", i, p, pp)
		}
	}

	// Org-search fixture.
	if sc.OrgSearchFixture.Org != devScopeOrgQuery || sc.OrgSearchFixture.Notice != devScopeOrgNotice {
		t.Errorf("org_search_fixture drift:\n fixtures.json = %+v\n pinned        = org=%q notice=%q",
			sc.OrgSearchFixture, devScopeOrgQuery, devScopeOrgNotice)
	}

	// Exclusions.
	if len(sc.Exclusions) != len(devScopeExclusions) {
		t.Fatalf("exclusions length drift: fixtures.json = %d, pinned = %d", len(sc.Exclusions), len(devScopeExclusions))
	}
	for i, e := range sc.Exclusions {
		pe := devScopeExclusions[i]
		if e.ID != pe.ID || e.Kind != pe.Kind || e.Value != pe.Value {
			t.Errorf("exclusion %d drift:\n fixtures.json = %+v\n pinned        = %+v", i, e, pe)
		}
	}

	// Exclusion-preview fixture.
	xp := sc.ExclusionPreviewFixture
	if xp.PostKind != devScopeExclPreviewKind || xp.PostValue != devScopeExclPreviewValue || xp.Fires != devScopeExclPreviewFires ||
		xp.Headline != devScopeExclPreviewHeadline || xp.Loss != devScopeExclPreviewLoss {
		t.Errorf("exclusion_preview_fixture drift:\n fixtures.json = %+v\n pinned        = kind=%q value=%q fires=%v headline=%q loss=%q",
			xp, devScopeExclPreviewKind, devScopeExclPreviewValue, devScopeExclPreviewFires, devScopeExclPreviewHeadline, devScopeExclPreviewLoss)
	}
}

// fixtureSignalsPackage is the on-disk shape of design-system/fixtures/fixtures.json → signals,
// the frozen slice TestSignalsFixtureMatchesPackage folds the pinned dev fixture back through.
type fixtureSignalsPackage struct {
	Signals struct {
		OpenCount   int                `json:"open_count"`
		Shown       int                `json:"shown"`
		PageInfo    string             `json:"page_info"`
		PageCount   int                `json:"page_count"`
		DetectedBy  string             `json:"detected_by"`
		HistoryRule string             `json:"history_rule"`
		Rows        []fixtureSignalRow `json:"rows"`
		Withdrawn   []fixtureSignalRow `json:"withdrawn"`
		Annotations map[string]struct {
			ID     string `json:"id"`
			Reason string `json:"reason"`
		} `json:"annotations"`
		Diffs map[string]struct {
			Title string `json:"title"`
			Lines []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"lines"`
		} `json:"diffs"`
	} `json:"signals"`
}

type fixtureSignalRow struct {
	ID       string            `json:"id"`
	Severity string            `json:"severity"`
	SevLabel string            `json:"sev_label"`
	Title    string            `json:"title"`
	Asset    string            `json:"asset"`
	IP       string            `json:"ip"`
	Port     string            `json:"port"`
	Seen     string            `json:"seen"`
	First    string            `json:"first"`
	Last     string            `json:"last"`
	CVE      *string           `json:"cve"`
	Tags     []string          `json:"tags"`
	Desc     string            `json:"desc"`
	Rule     []json.RawMessage `json:"rule"`
	ViewKey  string            `json:"view_key"`
}

func (r fixtureSignalRow) ruleParts() (id, version string) {
	if len(r.Rule) >= 1 {
		_ = json.Unmarshal(r.Rule[0], &id)
	}
	if len(r.Rule) >= 2 {
		var n json.Number
		if err := json.Unmarshal(r.Rule[1], &n); err == nil {
			version = n.String()
		}
	}
	return id, version
}

func (r fixtureSignalRow) cve() string {
	if r.CVE != nil {
		return *r.CVE
	}
	return ""
}

// assertSignalRow folds one fixture row through the pinned devSignalRow and deep-asserts each field
// (the rule ref split into id + version, the nullable CVE, the tags in order).
func assertSignalRow(t *testing.T, where string, i int, f fixtureSignalRow, p devSignalRow) {
	t.Helper()
	id, ver := f.ruleParts()
	if f.ID != p.ID || f.Severity != p.Severity || f.SevLabel != p.SevLabel || f.Title != p.Title ||
		f.Asset != p.Asset || f.IP != p.IP || f.Port != p.Port || f.Seen != p.Seen ||
		f.First != p.First || f.Last != p.Last || f.cve() != p.CVE || f.Desc != p.Desc ||
		id != p.RuleID || ver != p.RuleVersion {
		t.Errorf("%s row %d drift:\n fixtures.json = %+v (rule %s@%s)\n pinned        = %+v", where, i, f, id, ver, p)
	}
	if len(f.Tags) != len(p.Tags) {
		t.Fatalf("%s row %d tags length drift: fixtures.json = %d, pinned = %d", where, i, len(f.Tags), len(p.Tags))
	}
	for j := range f.Tags {
		if f.Tags[j] != p.Tags[j] {
			t.Errorf("%s row %d tag %d drift: fixtures.json = %q, pinned = %q", where, i, j, f.Tags[j], p.Tags[j])
		}
	}
}

// TestSignalsFixtureMatchesPackage is the byte-exactness gate before the pixels: it folds the pinned
// dev Signals fixture (cmd/web/devfixtures.go) back through the frozen design package
// (design-system/fixtures/fixtures.json → signals) and fails the build on any divergence — the open
// scalars, the ten open + three withdrawn rows (with rule metadata), the annotations, the drift
// diffs, and the two span-history literals the derivation depends on. It guards the same seam
// TestScopeFixtureMatchesPackage guards for Scope.
func TestSignalsFixtureMatchesPackage(t *testing.T) {
	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures.json: %v", err)
	}
	var f fixtureSignalsPackage
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixtures.json: %v", err)
	}
	sig := f.Signals

	// Open-tab scalars.
	if sig.OpenCount != devSignalsOpenCount || sig.Shown != devSignalsShown ||
		sig.PageInfo != devSignalsPageInfo || sig.PageCount != devSignalsPageCount ||
		sig.DetectedBy != devSignalsDetectedBy {
		t.Errorf("open scalars drift:\n fixtures.json = open=%d shown=%d pageInfo=%q pageCount=%d detectedBy=%q\n pinned        = open=%d shown=%d pageInfo=%q pageCount=%d detectedBy=%q",
			sig.OpenCount, sig.Shown, sig.PageInfo, sig.PageCount, sig.DetectedBy,
			devSignalsOpenCount, devSignalsShown, devSignalsPageInfo, devSignalsPageCount, devSignalsDetectedBy)
	}

	// The span-history derivation depends on two literals embedded in history_rule: the fixed
	// discovery instant and the detecting vantage. Assert both appear there so the derivation stays
	// tied to the frozen fixture (history is a rule, not an authored array).
	if !strings.Contains(sig.HistoryRule, devSignalsDiscovered) {
		t.Errorf("history_rule missing pinned discovery instant %q: %q", devSignalsDiscovered, sig.HistoryRule)
	}
	if !strings.Contains(sig.HistoryRule, devSignalsDetectedBy) {
		t.Errorf("history_rule missing pinned detecting vantage %q: %q", devSignalsDetectedBy, sig.HistoryRule)
	}

	// Open rows.
	if len(sig.Rows) != len(devSignalsOpen) {
		t.Fatalf("open rows length drift: fixtures.json = %d, pinned = %d", len(sig.Rows), len(devSignalsOpen))
	}
	for i, row := range sig.Rows {
		assertSignalRow(t, "open", i, row, devSignalsOpen[i])
	}

	// Withdrawn rows.
	if len(sig.Withdrawn) != len(devSignalsWithdrawn) {
		t.Fatalf("withdrawn rows length drift: fixtures.json = %d, pinned = %d", len(sig.Withdrawn), len(devSignalsWithdrawn))
	}
	for i, row := range sig.Withdrawn {
		assertSignalRow(t, "withdrawn", i, row, devSignalsWithdrawn[i])
	}

	// Annotations.
	if len(sig.Annotations) != len(devSignalsAnnotations) {
		t.Fatalf("annotations length drift: fixtures.json = %d, pinned = %d", len(sig.Annotations), len(devSignalsAnnotations))
	}
	for key, a := range sig.Annotations {
		p, ok := devSignalsAnnotations[key]
		if !ok {
			t.Errorf("annotation %q missing from pinned set", key)
			continue
		}
		if a.ID != p.ID || a.Reason != p.Reason {
			t.Errorf("annotation %q drift:\n fixtures.json = %+v\n pinned        = %+v", key, a, p)
		}
	}

	// Drift diffs.
	if len(sig.Diffs) != len(devSignalsDiffs) {
		t.Fatalf("diffs length drift: fixtures.json = %d, pinned = %d", len(sig.Diffs), len(devSignalsDiffs))
	}
	for key, d := range sig.Diffs {
		p, ok := devSignalsDiffs[key]
		if !ok {
			t.Errorf("diff %q missing from pinned set", key)
			continue
		}
		if d.Title != p.Title || len(d.Lines) != len(p.Lines) {
			t.Fatalf("diff %q drift:\n fixtures.json = %+v\n pinned        = %+v", key, d, p)
		}
		for j, l := range d.Lines {
			if l.Type != p.Lines[j].Type || l.Text != p.Lines[j].Text {
				t.Errorf("diff %q line %d drift:\n fixtures.json = %+v\n pinned        = %+v", key, j, l, p.Lines[j])
			}
		}
	}
}

// fixtureDashboardPackage mirrors the fixtures.json dashboard slice the screen-11 dev fixture pins in
// devfixtures.go, plus the signals.rows the most-recent register reuses. counted is a RawMessage
// because the JSON is mixed (a number for the address scope, a pre-formatted string for the name
// scope); total is a pointer so the name-scope census (no denominator) round-trips as nil.
type fixtureDashboardPackage struct {
	Dashboard struct {
		ScanSchedule struct {
			HasLast bool   `json:"has_last"`
			LastAgo string `json:"last_ago"`
			HasNext bool   `json:"has_next"`
			NextIn  string `json:"next_in"`
		} `json:"scan_schedule"`
		ScanningVariant string   `json:"scanning_variant"`
		ScanDetail      string   `json:"scan_detail"`
		Unavailable     []string `json:"unavailable"`
		StatBand        []struct {
			Label            string `json:"label"`
			Value            string `json:"value"`
			LiveWhenScanning bool   `json:"live_when_scanning"`
			HasDelta         bool   `json:"has_delta"`
			Change           int    `json:"change"`
			Tone             string `json:"tone"`
			Caption          string `json:"caption"`
		} `json:"stat_band"`
		SevBars []struct {
			Sev   string `json:"sev"`
			Pct   int    `json:"pct"`
			Count int    `json:"count"`
		} `json:"sev_bars"`
		CoverageMeters []struct {
			Label   string          `json:"label"`
			Counted json.RawMessage `json:"counted"`
			Total   *int            `json:"total"`
			Pct     int             `json:"pct"`
			Unit    string          `json:"unit"`
		} `json:"coverage_meters"`
		SilentZone struct {
			Bound string `json:"bound"`
			Text  string `json:"text"`
		} `json:"silent_zone"`
		Vantages []struct {
			Name    string `json:"name"`
			Latency string `json:"latency"`
			Avail   string `json:"avail"`
		} `json:"vantages"`
	} `json:"dashboard"`
	Signals struct {
		Rows []struct {
			ID       string `json:"id"`
			Severity string `json:"severity"`
			SevLabel string `json:"sev_label"`
			Title    string `json:"title"`
			Asset    string `json:"asset"`
			Port     string `json:"port"`
			Seen     string `json:"seen"`
			ViewKey  string `json:"view_key"`
		} `json:"rows"`
	} `json:"signals"`
}

// dashCountedStr renders a fixtures.json coverage-meter counted RawMessage as the pinned string:
// a JSON string is unquoted ("1,284"), a JSON number is used verbatim ("212").
func dashCountedStr(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if len(s) >= 1 && s[0] == '"' {
		var out string
		if err := json.Unmarshal(raw, &out); err == nil {
			return out
		}
	}
	return s
}

// TestDashboardFixtureMatchesPackage is the byte-exactness gate for the screen-11 conversion: every
// value the dev fixture pins (devfixtures.go, served by home() under devMode) equals the frozen
// fixtures.json dashboard slice, in authored order — so a drift between the served candidate and the
// golden (which composes the same fixture statically) fails here rather than in a screenshot diff. It
// also folds the most-recent register back through the fixtures.json signals.rows the note points at
// (first six), confirming the deep-link ViewKey resolves the Signals drawer.
func TestDashboardFixtureMatchesPackage(t *testing.T) {
	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures.json: %v", err)
	}
	var f fixtureDashboardPackage
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixtures.json: %v", err)
	}
	d := f.Dashboard

	if d.ScanningVariant != devDashScanningVariant {
		t.Errorf("scanning variant drift: fixtures.json = %q, pinned = %q", d.ScanningVariant, devDashScanningVariant)
	}
	if d.ScanDetail != devDashScanDetail {
		t.Errorf("scan_detail drift: fixtures.json = %q, pinned = %q", d.ScanDetail, devDashScanDetail)
	}
	if d.ScanSchedule.HasLast != devDashSchedule["HasLast"] || d.ScanSchedule.LastAgo != devDashSchedule["LastAgo"] ||
		d.ScanSchedule.HasNext != devDashSchedule["HasNext"] || d.ScanSchedule.NextIn != devDashSchedule["NextIn"] {
		t.Errorf("scan_schedule drift: fixtures.json = %+v, pinned = %+v", d.ScanSchedule, devDashSchedule)
	}
	if len(d.Unavailable) != len(devDashUnavailable) {
		t.Fatalf("unavailable length drift: fixtures.json = %d, pinned = %d", len(d.Unavailable), len(devDashUnavailable))
	}
	for i := range d.Unavailable {
		if d.Unavailable[i] != devDashUnavailable[i] {
			t.Errorf("unavailable[%d] drift: fixtures.json = %q, pinned = %q", i, d.Unavailable[i], devDashUnavailable[i])
		}
	}

	if len(d.StatBand) != len(devDashStatBand) {
		t.Fatalf("stat_band length drift: fixtures.json = %d, pinned = %d", len(d.StatBand), len(devDashStatBand))
	}
	for i, st := range d.StatBand {
		p := devDashStatBand[i]
		if st.Label != p.label || st.Value != p.value || st.LiveWhenScanning != p.liveWhenScanning ||
			st.HasDelta != p.hasDelta || st.Change != p.change || st.Tone != p.tone || st.Caption != p.caption {
			t.Errorf("stat_band %d drift:\n fixtures.json = %+v\n pinned        = %+v", i, st, p)
		}
	}

	if len(d.SevBars) != len(devDashSevBars) {
		t.Fatalf("sev_bars length drift: fixtures.json = %d, pinned = %d", len(d.SevBars), len(devDashSevBars))
	}
	for i, b := range d.SevBars {
		p := devDashSevBars[i]
		if b.Sev != p.Sev || b.Pct != p.Pct || b.Count != p.Count {
			t.Errorf("sev_bars %d drift: fixtures.json = %+v, pinned = %+v", i, b, p)
		}
	}

	if len(d.CoverageMeters) != len(devDashboardMeters) {
		t.Fatalf("coverage_meters length drift: fixtures.json = %d, pinned = %d", len(d.CoverageMeters), len(devDashboardMeters))
	}
	for i, m := range d.CoverageMeters {
		p := devDashboardMeters[i]
		if m.Label != p.Label || dashCountedStr(m.Counted) != p.Counted || m.Unit != p.Unit {
			t.Errorf("coverage_meters %d drift: fixtures.json = {%q,%q,%q}, pinned = {%q,%q,%q}",
				i, m.Label, dashCountedStr(m.Counted), m.Unit, p.Label, p.Counted, p.Unit)
		}
		switch {
		case m.Total == nil && p.Total != nil, m.Total != nil && p.Total == nil:
			t.Errorf("coverage_meters %d total presence drift: fixtures.json nil=%v, pinned nil=%v", i, m.Total == nil, p.Total == nil)
		case m.Total != nil && p.Total != nil:
			if strconv.Itoa(*m.Total) != *p.Total {
				t.Errorf("coverage_meters %d total drift: fixtures.json = %d, pinned = %q", i, *m.Total, *p.Total)
			}
			if m.Pct != p.Pct {
				t.Errorf("coverage_meters %d pct drift: fixtures.json = %d, pinned = %d", i, m.Pct, p.Pct)
			}
		}
	}

	if d.SilentZone.Bound != devDashSilentZone.Bound || d.SilentZone.Text != devDashSilentZone.Text {
		t.Errorf("silent_zone drift: fixtures.json = %+v, pinned = %+v", d.SilentZone, *devDashSilentZone)
	}

	if len(d.Vantages) != len(devDashVantages) {
		t.Fatalf("vantages length drift: fixtures.json = %d, pinned = %d", len(d.Vantages), len(devDashVantages))
	}
	for i, v := range d.Vantages {
		p := devDashVantages[i]
		if v.Name != p.Name || v.Latency != p.Latency || v.Avail != p.Avail {
			t.Errorf("vantages %d drift: fixtures.json = %+v, pinned = {%q,%q,%q}", i, v, p.Name, p.Latency, p.Avail)
		}
	}

	// Most-recent register: the pinned dashRecentSignals() equals the first six fixtures.json
	// signals.rows, each carrying the deep-link ViewKey the Signals drawer resolves.
	if len(f.Signals.Rows) < 6 {
		t.Fatalf("signals.rows has %d rows, dashboard register needs >= 6", len(f.Signals.Rows))
	}
	recent := dashRecentSignals()
	if len(recent) != 6 {
		t.Fatalf("dashRecentSignals length = %d, want 6", len(recent))
	}
	for i, got := range recent {
		want := f.Signals.Rows[i]
		if got.Severity != want.Severity || got.SevLabel != want.SevLabel || got.Title != want.Title ||
			got.Asset != want.Asset || got.Port != want.Port || got.Seen != want.Seen || got.ViewKey != want.ViewKey {
			t.Errorf("recent[%d] drift:\n fixtures.json = %+v\n pinned        = %+v", i, want, got)
		}
	}
}

// fixtureAssetPackage is the fixtures.json → asset slice, snake_case as stored.
type fixtureAssetPackage struct {
	Asset struct {
		Key          string `json:"key"`
		Type         string `json:"type"`
		Severity     string `json:"severity"`
		SevLabel     string `json:"sev_label"`
		Exposure     string `json:"exposure"`
		Seen         string `json:"seen"`
		InScopeSince string `json:"in_scope_since"`
		Withdrawn    bool   `json:"withdrawn"`
		Ports        []struct {
			Port     string `json:"port"`
			Service  string `json:"service"`
			Exposure string `json:"exposure"`
			Since    string `json:"since"`
		} `json:"ports"`
		DNS []struct {
			Type  string `json:"type"`
			Value string `json:"value"`
			Seen  string `json:"seen"`
		} `json:"dns"`
		Cert struct {
			Name        string `json:"name"`
			Issuer      string `json:"issuer"`
			Algorithm   string `json:"algorithm"`
			NotAfter    string `json:"not_after"`
			Tone        string `json:"tone"`
			Label       string `json:"label"`
			Fingerprint string `json:"fingerprint"`
		} `json:"cert"`
		Provenance []struct {
			K string `json:"k"`
			V string `json:"v"`
		} `json:"provenance"`
		Signals []struct {
			Severity string `json:"severity"`
			SevLabel string `json:"sev_label"`
			Rule     string `json:"rule"`
			SigID    string `json:"sig_id"`
			Time     string `json:"time"`
		} `json:"signals"`
		Drift []struct {
			Change  string `json:"change"`
			Family  string `json:"family"`
			Subject string `json:"subject"`
			Detail  string `json:"detail"`
			Time    string `json:"time"`
		} `json:"drift"`
	} `json:"asset"`
}

// TestAssetFixtureMatchesPackage folds every pinned AssetDetail dev-fixture value back
// through design-system/fixtures/fixtures.json → asset, so the VERGE_DEV render the
// pixel goldens capture can never drift from the frozen package (#581).
func TestAssetFixtureMatchesPackage(t *testing.T) {
	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures.json: %v", err)
	}
	var f fixtureAssetPackage
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixtures.json: %v", err)
	}
	a := f.Asset
	d := devAssetData()

	if a.Key != d.Key || a.Key != devAssetKey {
		t.Errorf("key drift: fixtures.json = %q, pinned = %q/%q", a.Key, d.Key, devAssetKey)
	}
	if a.Type != d.Type || a.Severity != d.Severity || a.SevLabel != d.SevLabel ||
		a.Exposure != d.Exposure || a.Seen != d.Seen || a.InScopeSince != d.InScopeSince || a.Withdrawn != d.Withdrawn {
		t.Errorf("header drift:\n fixtures.json = %+v\n pinned        = %+v", a, d)
	}

	if len(a.Ports) != len(d.Ports) {
		t.Fatalf("ports length drift: fixtures.json = %d, pinned = %d", len(a.Ports), len(d.Ports))
	}
	for i, p := range a.Ports {
		q := d.Ports[i]
		if p.Port != q.Port || p.Service != q.Service || p.Exposure != q.Exposure || p.Since != q.Since {
			t.Errorf("ports[%d] drift: fixtures.json = %+v, pinned = %+v", i, p, q)
		}
	}

	if len(a.DNS) != len(d.DNS) {
		t.Fatalf("dns length drift: fixtures.json = %d, pinned = %d", len(a.DNS), len(d.DNS))
	}
	for i, r := range a.DNS {
		q := d.DNS[i]
		if r.Type != q.Type || r.Value != q.Value || r.Seen != q.Seen {
			t.Errorf("dns[%d] drift: fixtures.json = %+v, pinned = %+v", i, r, q)
		}
	}

	if d.Cert == nil {
		t.Fatalf("pinned cert is nil; fixtures.json carries one")
	}
	if a.Cert.Name != d.Cert.Name || a.Cert.Issuer != d.Cert.Issuer || a.Cert.Algorithm != d.Cert.Algorithm ||
		a.Cert.NotAfter != d.Cert.NotAfter || a.Cert.Tone != d.Cert.Tone || a.Cert.Label != d.Cert.Label ||
		a.Cert.Fingerprint != d.Cert.Fingerprint {
		t.Errorf("cert drift:\n fixtures.json = %+v\n pinned        = %+v", a.Cert, *d.Cert)
	}

	if len(a.Provenance) != len(d.Provenance) {
		t.Fatalf("provenance length drift: fixtures.json = %d, pinned = %d", len(a.Provenance), len(d.Provenance))
	}
	for i, p := range a.Provenance {
		q := d.Provenance[i]
		if p.K != q.K || p.V != q.V {
			t.Errorf("provenance[%d] drift: fixtures.json = %+v, pinned = %+v", i, p, q)
		}
	}

	if len(a.Signals) != len(d.Signals) {
		t.Fatalf("signals length drift: fixtures.json = %d, pinned = %d", len(a.Signals), len(d.Signals))
	}
	for i, s := range a.Signals {
		q := d.Signals[i]
		if s.Severity != q.Severity || s.SevLabel != q.SevLabel || s.Rule != q.Rule || s.SigID != q.SigID || s.Time != q.Time {
			t.Errorf("signals[%d] drift: fixtures.json = %+v, pinned = %+v", i, s, q)
		}
	}

	if len(a.Drift) != len(d.Drift) {
		t.Fatalf("drift length drift: fixtures.json = %d, pinned = %d", len(a.Drift), len(d.Drift))
	}
	for i, e := range a.Drift {
		q := d.Drift[i]
		if e.Change != q.Change || e.Family != q.Family || e.Subject != q.Subject || e.Detail != q.Detail || e.Time != q.Time {
			t.Errorf("drift[%d] drift: fixtures.json = %+v, pinned = %+v", i, e, q)
		}
	}
}

// fixtureSubjectSpan / …Timeline / …Service / …Endpoint mirror the
// design-system/fixtures/fixtures.json → subjectdetail shapes for the drift test below.
type fixtureSubjectSpan struct {
	IsGap      bool   `json:"is_gap"`
	Value      string `json:"value"`
	OpenedAt   string `json:"opened_at"`
	OpenedFull string `json:"opened_full"`
	ClosedAt   string `json:"closed_at"`
	ClosedFull string `json:"closed_full"`
	Reason     string `json:"reason"`
}

type fixtureSubjectTimeline struct {
	Label   string `json:"label"`
	Current *struct {
		IsGap    bool   `json:"is_gap"`
		Value    string `json:"value"`
		OpenedAt string `json:"opened_at"`
	} `json:"current"`
	Breaks []struct {
		At          string `json:"at"`
		MovedLeaves string `json:"moved_leaves"`
	} `json:"breaks"`
	Closed []fixtureSubjectSpan `json:"closed"`
}

type fixtureSubjectService struct {
	Key          string `json:"key"`
	CopyKey      string `json:"copy_key"`
	Withdrawn    bool   `json:"withdrawn"`
	Exposure     string `json:"exposure"`
	Seen         string `json:"seen"`
	InScopeSince string `json:"in_scope_since"`
	Citation     []struct {
		Label  string `json:"label"`
		Value  string `json:"value"`
		Detail string `json:"detail"`
	} `json:"citation"`
	CitationTerminated bool                     `json:"citation_terminated"`
	Address            string                   `json:"address"`
	Port               string                   `json:"port"`
	Transport          string                   `json:"transport"`
	Reach              string                   `json:"reach"`
	Since              string                   `json:"since"`
	Timelines          []fixtureSubjectTimeline `json:"timelines"`
	Rules              []struct {
		Rule     string `json:"rule"`
		Version  int    `json:"version"`
		Severity string `json:"severity"`
		SevLabel string `json:"sev_label"`
		Fired    bool   `json:"fired"`
	} `json:"rules"`
	Provenance []struct {
		K string `json:"k"`
		V string `json:"v"`
	} `json:"provenance"`
	Signals []struct {
		Severity string `json:"severity"`
		SevLabel string `json:"sev_label"`
		Rule     string `json:"rule"`
		SigID    string `json:"sig_id"`
		Time     string `json:"time"`
	} `json:"signals"`
}

type fixtureSubjectEndpoint struct {
	Key          string `json:"key"`
	CopyKey      string `json:"copy_key"`
	Nameless     bool   `json:"nameless"`
	Withdrawn    bool   `json:"withdrawn"`
	Seen         string `json:"seen"`
	InScopeSince string `json:"in_scope_since"`
	Citation     []struct {
		Label  string `json:"label"`
		Value  string `json:"value"`
		Detail string `json:"detail"`
	} `json:"citation"`
	CitationTerminated bool                     `json:"citation_terminated"`
	Name               string                   `json:"name"`
	Service            string                   `json:"service"`
	HasIdentity        bool                     `json:"has_identity"`
	Status             string                   `json:"status"`
	Server             string                   `json:"server"`
	Title              string                   `json:"title"`
	RedirectLocation   string                   `json:"redirect_location"`
	WWWAuthenticate    string                   `json:"www_authenticate"`
	Timelines          []fixtureSubjectTimeline `json:"timelines"`
	Rules              []struct {
		Rule     string `json:"rule"`
		Version  int    `json:"version"`
		Severity string `json:"severity"`
		SevLabel string `json:"sev_label"`
		Fired    bool   `json:"fired"`
	} `json:"rules"`
	Provenance []struct {
		K string `json:"k"`
		V string `json:"v"`
	} `json:"provenance"`
}

type fixtureSubjectPackage struct {
	SubjectDetail struct {
		Service          fixtureSubjectService  `json:"service"`
		ServiceWithdrawn fixtureSubjectService  `json:"service_withdrawn"`
		Endpoint         fixtureSubjectEndpoint `json:"endpoint"`
	} `json:"subjectdetail"`
}

// assertServiceFixture folds one fixtures.json service slice against its pinned servicePageData.
func assertServiceFixture(t *testing.T, name string, a fixtureSubjectService, d servicePageData) {
	t.Helper()
	if a.Key != d.Key || a.CopyKey != d.CopyKey || a.Withdrawn != d.Withdrawn || a.Exposure != d.Exposure ||
		a.Seen != d.Seen || a.InScopeSince != d.InScopeSince || a.CitationTerminated != d.CitationTerminated ||
		a.Address != d.Address || a.Port != d.Port || a.Transport != d.Transport || a.Reach != d.Reach || a.Since != d.Since {
		t.Errorf("%s: service header drift:\n fixtures.json = %+v\n pinned        = %+v", name, a, d)
	}
	if len(a.Citation) != len(d.Citation) {
		t.Fatalf("%s: citation length drift: %d vs %d", name, len(a.Citation), len(d.Citation))
	}
	for i, c := range a.Citation {
		q := d.Citation[i]
		if c.Label != q.Label || c.Value != q.Value || c.Detail != q.Detail {
			t.Errorf("%s: citation[%d] drift: %+v vs %+v", name, i, c, q)
		}
	}
	assertSubjectTimelines(t, name, a.Timelines, d.Timelines)
	if len(a.Rules) != len(d.Rules) {
		t.Fatalf("%s: rules length drift: %d vs %d", name, len(a.Rules), len(d.Rules))
	}
	for i, r := range a.Rules {
		q := d.Rules[i]
		if r.Rule != q.Rule || strconv.Itoa(r.Version) != q.Version || r.Severity != q.Severity || r.SevLabel != q.SevLabel || r.Fired != q.Fired {
			t.Errorf("%s: rules[%d] drift: %+v vs %+v", name, i, r, q)
		}
	}
	if len(a.Provenance) != len(d.Provenance) {
		t.Fatalf("%s: provenance length drift: %d vs %d", name, len(a.Provenance), len(d.Provenance))
	}
	for i, p := range a.Provenance {
		q := d.Provenance[i]
		if p.K != q.K || p.V != q.V {
			t.Errorf("%s: provenance[%d] drift: %+v vs %+v", name, i, p, q)
		}
	}
	if len(a.Signals) != len(d.Signals) {
		t.Fatalf("%s: signals length drift: %d vs %d", name, len(a.Signals), len(d.Signals))
	}
	for i, s := range a.Signals {
		q := d.Signals[i]
		if s.Severity != q.Severity || s.SevLabel != q.SevLabel || s.Rule != q.Rule || s.SigID != q.SigID || s.Time != q.Time {
			t.Errorf("%s: signals[%d] drift: %+v vs %+v", name, i, s, q)
		}
	}
}

// assertSubjectTimelines folds the fixtures.json timeline slice against the pinned []timelineView.
func assertSubjectTimelines(t *testing.T, name string, a []fixtureSubjectTimeline, d []timelineView) {
	t.Helper()
	if len(a) != len(d) {
		t.Fatalf("%s: timelines length drift: %d vs %d", name, len(a), len(d))
	}
	for i, tl := range a {
		q := d[i]
		if tl.Label != q.Label {
			t.Errorf("%s: timelines[%d] label drift: %q vs %q", name, i, tl.Label, q.Label)
		}
		if (tl.Current == nil) != (q.Current == nil) {
			t.Errorf("%s: timelines[%d] current presence drift: %v vs %v", name, i, tl.Current != nil, q.Current != nil)
		} else if tl.Current != nil {
			if tl.Current.IsGap != q.Current.IsGap || tl.Current.Value != q.Current.Value || tl.Current.OpenedAt != q.Current.OpenedAt {
				t.Errorf("%s: timelines[%d] current drift: %+v vs %+v", name, i, *tl.Current, *q.Current)
			}
		}
		if len(tl.Breaks) != len(q.Breaks) {
			t.Fatalf("%s: timelines[%d] breaks length drift: %d vs %d", name, i, len(tl.Breaks), len(q.Breaks))
		}
		for j, b := range tl.Breaks {
			if b.At != q.Breaks[j].At || b.MovedLeaves != q.Breaks[j].MovedLeaves {
				t.Errorf("%s: timelines[%d] breaks[%d] drift: %+v vs %+v", name, i, j, b, q.Breaks[j])
			}
		}
		if len(tl.Closed) != len(q.Closed) {
			t.Fatalf("%s: timelines[%d] closed length drift: %d vs %d", name, i, len(tl.Closed), len(q.Closed))
		}
		for j, c := range tl.Closed {
			s := q.Closed[j]
			if c.IsGap != s.IsGap || c.Value != s.Value || c.OpenedAt != s.OpenedAt || c.OpenedFull != s.OpenedFull ||
				c.ClosedAt != s.ClosedAt || c.ClosedFull != s.ClosedFull || c.Reason != s.Reason {
				t.Errorf("%s: timelines[%d] closed[%d] drift: %+v vs %+v", name, i, j, c, s)
			}
		}
	}
}

// TestSubjectDetailFixtureMatchesPackage folds every pinned SubjectDetail dev-fixture value back
// through design-system/fixtures/fixtures.json → subjectdetail, so the VERGE_DEV render the pixel
// goldens capture can never drift from the frozen package (#582).
func TestSubjectDetailFixtureMatchesPackage(t *testing.T) {
	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures.json: %v", err)
	}
	var f fixtureSubjectPackage
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixtures.json: %v", err)
	}
	if f.SubjectDetail.Service.Key != devServiceKey {
		t.Errorf("service key drift: fixtures.json = %q, pinned = %q", f.SubjectDetail.Service.Key, devServiceKey)
	}
	if f.SubjectDetail.ServiceWithdrawn.Key != devServiceWithdrawnKey {
		t.Errorf("withdrawn key drift: fixtures.json = %q, pinned = %q", f.SubjectDetail.ServiceWithdrawn.Key, devServiceWithdrawnKey)
	}
	if f.SubjectDetail.Endpoint.Key != devEndpointKey {
		t.Errorf("endpoint key drift: fixtures.json = %q, pinned = %q", f.SubjectDetail.Endpoint.Key, devEndpointKey)
	}
	assertServiceFixture(t, "service", f.SubjectDetail.Service, devServiceData())
	assertServiceFixture(t, "service_withdrawn", f.SubjectDetail.ServiceWithdrawn, devServiceWithdrawnData())

	// Endpoint.
	a, d := f.SubjectDetail.Endpoint, devEndpointData()
	if a.Key != d.Key || a.CopyKey != d.CopyKey || a.Nameless != d.Nameless || a.Withdrawn != d.Withdrawn ||
		a.Seen != d.Seen || a.InScopeSince != d.InScopeSince || a.CitationTerminated != d.CitationTerminated ||
		a.Name != d.Name || a.Service != d.Service || a.HasIdentity != d.HasIdentity || a.Status != d.Status ||
		a.Server != d.Server || a.Title != d.Title || a.RedirectLocation != d.RedirectLocation || a.WWWAuthenticate != d.WWWAuthenticate {
		t.Errorf("endpoint header drift:\n fixtures.json = %+v\n pinned        = %+v", a, d)
	}
	if len(a.Citation) != len(d.Citation) {
		t.Fatalf("endpoint citation length drift: %d vs %d", len(a.Citation), len(d.Citation))
	}
	for i, c := range a.Citation {
		q := d.Citation[i]
		if c.Label != q.Label || c.Value != q.Value || c.Detail != q.Detail {
			t.Errorf("endpoint citation[%d] drift: %+v vs %+v", i, c, q)
		}
	}
	assertSubjectTimelines(t, "endpoint", a.Timelines, d.Timelines)
	if len(a.Rules) != len(d.Rules) {
		t.Fatalf("endpoint rules length drift: %d vs %d", len(a.Rules), len(d.Rules))
	}
	for i, r := range a.Rules {
		q := d.Rules[i]
		if r.Rule != q.Rule || strconv.Itoa(r.Version) != q.Version || r.Severity != q.Severity || r.SevLabel != q.SevLabel || r.Fired != q.Fired {
			t.Errorf("endpoint rules[%d] drift: %+v vs %+v", i, r, q)
		}
	}
	if len(a.Provenance) != len(d.Provenance) {
		t.Fatalf("endpoint provenance length drift: %d vs %d", len(a.Provenance), len(d.Provenance))
	}
	for i, p := range a.Provenance {
		q := d.Provenance[i]
		if p.K != q.K || p.V != q.V {
			t.Errorf("endpoint provenance[%d] drift: %+v vs %+v", i, p, q)
		}
	}
}
