package main

// PROTOTYPE — throwaway. Answers #1790 for map #1786. Not production code.

import "reflect"

// Entry is one act class with a real sample, so every claim below renders.
type Entry struct {
	Limb  string
	Route string
	Actor Actor
	Act   Act
	Note  string
}

var acct = Account{AccountID: 7, UsernameSnapshot: "alice"}

// catalogue holds one entry per variant. It is the single source for the label
// table, the decoder table and the rendered walkthrough, so a variant that is
// added without a sample fails the count test rather than rendering blank.
var catalogue = []Entry{
	{"2", "POST /setup", GrantHolder{SetupToken{}}, SetupCompleted{AccountAtRole{1, "root", "admin"}}, "the first admin; a human holds the token (#1795)"},
	{"2", "POST /reset", GrantHolder{PasswordReset{}}, PasswordResetApplied{AccountRef{7, "alice"}}, "not Account: the link rides the instance's own web log (#1795)"},
	{"2", "POST /invite", GrantHolder{Invite{ID: 12}}, InviteAccepted{AccountAtRole{9, "bob", "viewer"}}, "the id correlates with invite.minted (#1795)"},
	{"3", "POST /onboarding/finish", acct, OnboardingFinished{ScanKindRef{"hot"}}, ""},

	{"1", "POST /seeds", acct, SeedDeclared{SeedScope{"10.0.0.0/8"}}, ""},
	{"1", "POST /seeds/delete", acct, SeedWithdrawn{SeedScope{"10.0.0.0/8"}}, "hazard 1: the Seed is gone and the row still reads"},
	{"1", "POST /seeds/custody", acct, SeedCustodyMoved{ScopeMove{"example.com", "custody extended"}}, ""},
	{"1", "POST /seeds/zone", acct, ZoneDeclared{ZoneName{"example.com"}}, ""},
	{"1", "POST /seeds/zone/interval", acct, ZoneCadenceSet{DialMove{"zone scan cadence", "24h"}}, ""},
	{"1", "POST /seeds/dns/interval", acct, DNSCadenceSet{DialMove{"dns scan cadence", "6h"}}, ""},
	{"1", "POST /exclusions", acct, ExclusionDeclared{ExclusionTerm{"address", "10.1.2.0/24"}}, ""},
	{"1", "POST /exclusions/delete", acct, ExclusionLifted{ExclusionTerm{"address", "10.1.2.0/24"}}, ""},
	{"1", "POST /settings/cold", acct, ColdMoved{ScopeMove{"10.0.0.0/8", "cold opt-in"}}, ""},
	{"1", "POST /settings/probers", acct, VantageDeclared{VantageRef{"probe@edge-01.example.com:22"}}, ""},
	{"1", "POST /settings/vantages/resolver", acct, VantageResolverSet{VantageRef{"local"}}, ""},

	{"1", "POST /reports/schedule/new", acct, ScheduleDeclared{ScheduleRef{"weekly exposure"}}, ""},
	{"1", "POST /reports/schedule/{id}/edit", acct, ScheduleEdited{ScheduleRef{"weekly exposure"}}, ""},
	{"1", "POST /reports/schedule/delete", acct, ScheduleWithdrawn{ScheduleRef{"weekly exposure"}}, ""},

	{"1", "POST /annotations", acct, AnnotationDeclared{AnnotationRef{"10.0.4.9:443", "expired certificate"}}, "the Act carries the actor; the Annotation carries none (ADR-0073)"},
	{"1", "POST /annotations/withdraw", acct, AnnotationWithdrawn{AnnotationRef{"10.0.4.9:443", "expired certificate"}}, ""},

	{"3", "POST /proposals", acct, ProposalQueried{ProposalQuery{"Example Holdings Ltd"}}, "one class, two routes (+ /proposals/search)"},
	{"1", "POST /proposals/confirm", acct, ProposalConfirmed{SeedScope{"198.51.100.0/24"}}, ""},
	{"1", "POST /proposals/decline", acct, ProposalDeclined{ExclusionTerm{"address", "203.0.113.0/24"}}, "one row per subject: a 200-item decline writes 200 rows (#1788)"},
	{"1", "POST /proposals/undo-decline", acct, ProposalDeclineUndone{ExclusionTerm{"address", "203.0.113.0/24"}}, ""},

	{"3", "POST /scans/trigger", acct, ScanTriggered{ScanKindRef{"hot"}}, "#11's own second example"},
	{"3", "POST /scans/stop", acct, ScanStopped{DispatchRef{"dispatch 418", "hot"}}, ""},
	{"3", "POST /scans/terminate", acct, ScanTerminated{DispatchRef{"dispatch 418", "hot"}}, ""},

	{"1", "POST /verge-core/frequency", acct, FrequencyMoved{PortMove{"8443", "add"}}, ""},
	{"1", "POST /sources/toggle", acct, SourceMoved{SourceMove{"crtsh", "enabled"}}, "one class, two routes (+ /settings/sources)"},

	{"2", "POST /profile/password", acct, PasswordChanged{AccountRef{7, "alice"}}, ""},
	{"2", "POST /profile/tokens", acct, TokenMinted{TokenRef{"ci reader", "vga_7f21"}}, "hazard 2: no field for the token itself"},
	{"2", "POST /profile/tokens/revoke", acct, TokenRevoked{TokenRef{"ci reader", "vga_7f21"}}, ""},
	{"2", "POST /profile/sso/unlink", acct, SSOUnlinked{ProviderRef{"okta"}}, "ADR-0113 self-unlink"},

	{"2", "POST /accounts", acct, AccountCreated{AccountAtRole{9, "bob", "viewer"}}, ""},
	{"2", "POST /account/totp/confirm", acct, TOTPEnrolled{AccountRef{7, "alice"}}, "the confirm is the act, never the enable (#1789)"},

	{"2", "POST /settings/accounts", acct, InviteMinted{RoleRef{"viewer"}}, "no account exists yet; the subject is the role"},
	{"2", "POST /settings/accounts/role", acct, AccountRoleMoved{AccountAtRole{9, "bob", "admin"}}, ""},
	{"2", "POST /settings/accounts/reenroll", acct, TOTPStripped{AccountRef{9, "bob"}}, ""},
	{"2", "POST /settings/accounts/remove", acct, AccountRemoved{AccountRef{9, "bob"}}, "hazard 1: the account is gone and the row still reads"},

	{"1", "POST /settings/channels", acct, ChannelDeclared{ChannelRef{"hooks.example.com/services/T0/B0"}}, "ADR-0053: the set is auditable, the secret is not"},
	{"1", "POST /settings/channels/update", acct, ChannelUpdated{ChannelRef{"hooks.example.com/services/T0/B0"}}, ""},
	{"1", "POST /settings/channels/delete", acct, ChannelWithdrawn{ChannelRef{"hooks.example.com/services/T0/B0"}}, ""},
	{"3", "POST /settings/channels/test", acct, ChannelTested{ChannelRef{"hooks.example.com/services/T0/B0"}}, "a real signed POST, third party on the other end"},

	{"1", "POST /settings/retention", acct, TranscriptCurrencySet{DialMove{"transcript currency", "14 days"}}, ""},
	{"1", "POST /coverage/retention", acct, ObservationCurrencySet{DialMove{"observation currency", "30 days"}}, "Q4 split, row 1 of 2 from one submit"},
	{"1", "POST /coverage/retention", acct, DispatchCadenceSet{DialMove{"dispatch cadence", "4"}}, "Q4 split, row 2 of 2 from one submit"},
	{"1", "POST /settings/address-cap", acct, AddressCapSet{DialMove{"address-scope cap", "1024"}}, ""},
	{"1", "POST /settings/updates/check", acct, UpdateCheckMoved{ToggleMove{"update check", "off"}}, ""},
	{"1+2", "POST /settings/restore", acct, RestoreApplied{ArchiveRef{"verge-2026-09-08.tar.zst", "2026-09-08T14:02Z"}}, "hazard 3: written after the truncation (#1789)"},
	{"2", "POST /settings/api", acct, APIAccessMoved{ToggleMove{"API access", "on"}}, "ADR-0123"},

	{"2", "POST /settings/sso", acct, SSOProviderDeclared{ProviderRef{"okta"}}, ""},
	{"2", "POST /settings/sso/update", acct, SSOProviderUpdated{ProviderRef{"okta"}}, ""},
	{"2", "POST /settings/sso/secret", acct, SSOProviderSecretSet{ProviderRef{"okta"}}, "hazard 2: one field, and it is the slug"},
	{"2", "POST /settings/sso/delete", acct, SSOProviderWithdrawn{ProviderRef{"okta"}}, ""},
	{"2", "POST /settings/sso/identity/remove", acct, SSOBindingRemoved{BindingRef{"okta", "bob"}}, "ADR-0113 admin-remove"},

	{"1", "POST /settings/integrations/install", acct, IntegrationInstalled{IntegrationRef{"pagerduty"}}, ""},
	{"1", "POST /settings/integrations/remove", acct, IntegrationRemoved{IntegrationRef{"pagerduty"}}, "one class, two routes (+ /disconnect)"},
	{"1", "POST /settings/integrations/channel", acct, IntegrationChannelBound{IntegrationChannelRef{"pagerduty", ""}}, "an unbound channel renders, never an empty cell"},
	{"3", "POST /settings/integrations/test", acct, IntegrationTested{IntegrationChannelRef{"pagerduty", "events.pagerduty.com"}}, ""},

	{"2", "GET /profile/sso/{slug}/link/callback", acct, SSOBindingCreated{BindingRef{"okta", "alice"}}, "a GET; a POST-only sweep misses it (#1789)"},
	{"4", "GET /run/{id}/raw", acct, TranscriptDisclosed{JobRef{"9182", "412", "edge-01"}}, "limb 4; the subject outlives the Transcript it names (#1794)"},
	{"1", "goose.Up", System{}, MigrationApplied{MigrationRef{"25200", "seal_channel_and_sso_secrets"}}, "conditional on #1805"},
}

// label is the Action cell's copy. It is a read-time table and never a stored
// string, so a copy change never needs an UPDATE against an append-only corpus.
var label = map[string]string{
	"setup.completed":           "Instance set up",
	"password.reset":            "Password reset",
	"invite.accepted":           "Invite accepted",
	"onboarding.finished":       "Scan dispatched",
	"seed.declared":             "Seed declared",
	"seed.withdrawn":            "Seed withdrawn",
	"seed.custody.moved":        "Custody moved",
	"zone.declared":             "Zone declared",
	"zone.cadence.set":          "Dial moved",
	"dns.cadence.set":           "Dial moved",
	"exclusion.declared":        "Exclusion declared",
	"exclusion.lifted":          "Exclusion lifted",
	"cold.moved":                "Cold scan moved",
	"vantage.declared":          "Vantage declared",
	"vantage.resolver.set":      "Resolver set",
	"schedule.declared":         "Schedule declared",
	"schedule.edited":           "Schedule edited",
	"schedule.withdrawn":        "Schedule withdrawn",
	"annotation.declared":       "Annotation declared",
	"annotation.withdrawn":      "Annotation withdrawn",
	"proposal.queried":          "Registries queried",
	"proposal.confirmed":        "Proposal confirmed",
	"proposal.declined":         "Proposal declined",
	"proposal.decline.undone":   "Decline lifted",
	"scan.triggered":            "Scan triggered",
	"scan.stopped":              "Scan stopped",
	"scan.terminated":           "Scan terminated",
	"frequency.moved":           "Frequency moved",
	"source.moved":              "Source moved",
	"password.changed":          "Password changed",
	"token.minted":              "Token minted",
	"token.revoked":             "Token revoked",
	"sso.unlinked":              "SSO unlinked",
	"account.created":           "Account created",
	"totp.enrolled":             "TOTP enrolled",
	"invite.minted":             "Invite minted",
	"account.role.moved":        "Role moved",
	"totp.stripped":             "TOTP stripped",
	"account.removed":           "Account removed",
	"channel.declared":          "Channel declared",
	"channel.updated":           "Channel updated",
	"channel.withdrawn":         "Channel withdrawn",
	"channel.tested":            "Channel tested",
	"transcript.currency.set":   "Dial moved",
	"observation.currency.set":  "Dial moved",
	"dispatch.cadence.set":      "Dial moved",
	"address.cap.set":           "Dial moved",
	"update.check.moved":        "Update check moved",
	"restore.applied":           "Restore applied",
	"api.access.moved":          "API access moved",
	"sso.provider.declared":     "SSO provider declared",
	"sso.provider.updated":      "SSO provider updated",
	"sso.provider.secret.set":   "SSO secret set",
	"sso.provider.withdrawn":    "SSO provider withdrawn",
	"sso.binding.removed":       "SSO binding removed",
	"integration.installed":     "Integration installed",
	"integration.removed":       "Integration removed",
	"integration.channel.bound": "Channel bound",
	"integration.tested":        "Integration tested",
	"sso.binding.created":       "SSO linked",
	// #1794 worded limb 4 as a disclosure and refused the read register, so the
	// copy says disclosed.
	"transcript.disclosed": "Raw output disclosed",
	"migration.applied":    "Migration applied",
}

// decoders is built from the catalogue, so the reader can never drift from the
// writer's set.
var decoders = func() map[string]func() any {
	m := map[string]func() any{}
	for _, e := range catalogue {
		t := reflect.TypeOf(e.Act)
		m[e.Act.Class()] = func() any { return reflect.New(t).Interface() }
	}
	return m
}()
