package main

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/auth"
)

// Limb 2 — who may act on this instance (spec §1.2, §2.1). The gate proves a Record is reachable;
// it proves nothing about ordering, cardinality or content, so these do (spec §7.5).

func actActor(t *testing.T, f *fakeStore, class string) act.Actor {
	t.Helper()
	for _, a := range f.acts {
		if a.Action != class {
			continue
		}
		actor, _ := decodeAct(t, a)
		return actor
	}
	t.Fatalf("no %s row was recorded; recorded %v", class, actClasses(f))
	return nil
}

// The Actor cell and the Subject cell are what the admin reads, so both are asserted (spec §3.5).

func wantOneActBy(t *testing.T, f *fakeStore, class, actorCell, subject string) {
	t.Helper()
	wantOneAct(t, f, class, subject)
	if got := act.ActorCell(actActor(t, f, class)); got != actorCell {
		t.Errorf("%s actor cell = %q, want %q", class, got, actorCell)
	}
}

// The three GrantHolder variants. A grant carries the grant's id exactly when a second Act names
// the same grant, so only the invite carries one (spec §3.3).

func TestSetupRecordsTheTokenHolderAndNamesNoAccountAsActor(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "the-setup-token")

	postForm(t, newClient(t), base+"/setup", url.Values{
		"token": {"the-setup-token"}, "username": {"root"}, "password": {"hunter2hunter2"},
	}).Body.Close()

	wantOneActBy(t, f, "setup.completed", "setup token", "root · admin")
	if _, ok := actActor(t, f, "setup.completed").(act.GrantHolder); !ok {
		t.Errorf("setup recorded actor %T, want a GrantHolder", actActor(t, f, "setup.completed"))
	}
}

func TestARefusedSetupTokenRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "the-setup-token")

	postForm(t, newClient(t), base+"/setup", url.Values{
		"token": {"wrong"}, "username": {"root"}, "password": {"hunter2hunter2"},
	}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a refused setup token recorded %v, want nothing", got)
	}
}

// A row rendering the account would assert that ola reset her own password, when the host
// operator may have read the link out of the instance's web log (spec §3.3).

func TestAResetRecordsTheGrantHolderAndNeverTheAccount(t *testing.T) {
	f := newFakeStore()
	acct := seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	addReset(t, f, acct.ID, "live-token", serverClock.Add(time.Hour))

	postForm(t, newClient(t), base+"/reset", url.Values{
		"token": {"live-token"}, "password": {"brand-new-pass"}, "confirm": {"brand-new-pass"},
	}).Body.Close()

	wantOneActBy(t, f, "password.reset", "password-reset link", "ola")
	if actor := actActor(t, f, "password.reset"); strings.Contains(act.ActorCell(actor), "ola") {
		t.Errorf("the reset's actor cell names the account: %q", act.ActorCell(actor))
	}
}

func TestASpentResetTokenRecordsNothing(t *testing.T) {
	f := newFakeStore()
	acct := seedAccount(t, f, "ola", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	addReset(t, f, acct.ID, "stale-token", serverClock.Add(-time.Hour))

	postForm(t, newClient(t), base+"/reset", url.Values{
		"token": {"stale-token"}, "password": {"brand-new-pass"}, "confirm": {"brand-new-pass"},
	}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a spent reset token recorded %v, want nothing", got)
	}
}

func TestAnInviteAcceptanceCarriesTheInviteId(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	addInvite(t, f, roleViewer, "invite-token", serverClock.Add(24*time.Hour))
	id := f.invites[0].ID

	postForm(t, newClient(t), base+"/invite", url.Values{
		"token": {"invite-token"}, "username": {"bob"}, "password": {"hunter2hunter2"},
	}).Body.Close()

	wantOneActBy(t, f, "invite.accepted", "invite "+itoa(id), "bob · viewer")
}

func TestALostInviteRaceRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	addInvite(t, f, roleViewer, "invite-token", serverClock.Add(24*time.Hour))

	first := postForm(t, newClient(t), base+"/invite", url.Values{
		"token": {"invite-token"}, "username": {"bob"}, "password": {"hunter2hunter2"},
	})
	first.Body.Close()
	f.acts = nil

	second := postForm(t, newClient(t), base+"/invite", url.Values{
		"token": {"invite-token"}, "username": {"carol"}, "password": {"hunter2hunter2"},
	})
	second.Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a spent invite recorded %v, want nothing", got)
	}
}

// The profile acts. Each is the account's own session acting on its own credentials (spec §3.2).

func TestAPasswordChangeRecordsTheAccountItMoved(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/profile/password", url.Values{
		"current_password": {"hunter2hunter2"}, "new_password": {"brand-new-pass"},
	}).Body.Close()

	wantOneActBy(t, f, "password.changed", "@admin", "admin")
}

func TestAWrongCurrentPasswordRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/profile/password", url.Values{
		"current_password": {"not-the-password"}, "new_password": {"brand-new-pass"},
	}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a refused password change recorded %v, want nothing", got)
	}
}

var mintedTokenRE = regexp.MustCompile(`vg_pat_[0-9a-f]{48}`)

// The public prefix and never the token value: a secret being minted is auditable, its value is
// not (spec §4.2).

func TestATokenMintAndRevokeRecordTheLabelAndItsPrefix(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/profile/tokens", url.Values{"name": {"ci reader"}}).Body.Close()
	if len(f.personalTokens) != 1 {
		t.Fatalf("tokens minted = %d, want 1", len(f.personalTokens))
	}
	tok := f.personalTokens[0]
	wantOneActBy(t, f, "token.minted", "@admin", "ci reader ("+tok.Prefix+")")

	f.acts = nil
	postForm(t, ac, base+"/profile/tokens/revoke", url.Values{"id": {itoa(tok.ID)}}).Body.Close()
	wantOneActBy(t, f, "token.revoked", "@admin", "ci reader ("+tok.Prefix+")")
}

func TestRevokingAnUnknownTokenRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/profile/tokens/revoke", url.Values{"id": {"4242"}}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("an unknown token id recorded %v, want nothing", got)
	}
}

func TestATOTPEnrolmentRecordsOnTheConfirmAndNotTheStage(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	page := body(t, postForm(t, ac, base+"/account/totp/enable", nil))
	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("staging a secret recorded %v, want nothing (spec §2.3)", got)
	}
	secret := secretRE.FindStringSubmatch(page)[1]
	code, err := auth.TOTPCode(secret, serverClock)
	if err != nil {
		t.Fatal(err)
	}

	postForm(t, ac, base+"/account/totp/confirm", url.Values{"code": {code}}).Body.Close()
	wantOneActBy(t, f, "totp.enrolled", "@admin", "admin")
}

func TestAWrongTOTPCodeRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	postForm(t, ac, base+"/account/totp/enable", nil).Body.Close()
	f.acts = nil

	postForm(t, ac, base+"/account/totp/confirm", url.Values{"code": {"000000"}}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a wrong TOTP code recorded %v, want nothing", got)
	}
}

// The team acts. The admin's session acted; the subject is the account that moved (spec §2.1).

func TestCreatingAnAccountRecordsTheNameAndItsRole(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/accounts", url.Values{
		"username": {"bob"}, "password": {"hunter2hunter2"}, "role": {roleViewer},
	}).Body.Close()

	wantOneActBy(t, f, "account.created", "@admin", "bob · viewer")
}

func TestARefusedAccountCreationRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/accounts", url.Values{
		"username": {"admin"}, "password": {"hunter2hunter2"}, "role": {roleViewer},
	}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a duplicate username recorded %v, want nothing", got)
	}
}

// The mint creates no account, so the subject names the role alone (spec §2.1).

func TestAnInviteMintRecordsTheRoleAlone(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/settings/accounts", url.Values{"role": {roleViewer}}).Body.Close()

	wantOneActBy(t, f, "invite.minted", "@admin", "invite · viewer")
}

func TestARoleMoveRecordsTheAccountAtItsNewRole(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	bob := seedAccount(t, f, "bob", roleViewer, "hunter2hunter2")
	f.acts = nil

	postForm(t, ac, base+"/settings/accounts/role", url.Values{
		"id": {itoa(bob.ID)}, "role": {roleAdmin},
	}).Body.Close()

	wantOneActBy(t, f, "account.role.moved", "@admin", "bob · admin")
}

func TestARefusedLastAdminDemotionRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	admin := f.accounts[f.byName["admin"]]
	f.acts = nil

	postForm(t, ac, base+"/settings/accounts/role", url.Values{
		"id": {itoa(admin.ID)}, "role": {roleViewer},
	}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a refused demotion recorded %v, want nothing", got)
	}
}

func TestStrippingTOTPRecordsTheAccountItStripped(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	bob := seedAccount(t, f, "bob", roleViewer, "hunter2hunter2")
	enrolTOTP(f, bob.ID)
	f.acts = nil

	postForm(t, ac, base+"/settings/accounts/reenroll", url.Values{"id": {itoa(bob.ID)}}).Body.Close()

	wantOneActBy(t, f, "totp.stripped", "@admin", "bob")
}

// The Team menu offers "Require re-enrollment" on an account the same row badges as not
// enrolled, and stripping no second factor is not an act (spec §7.6 ruling 2).

func TestStrippingTOTPOnANeverEnrolledAccountRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	bob := seedAccount(t, f, "bob", roleViewer, "hunter2hunter2")
	f.acts = nil

	postForm(t, ac, base+"/settings/accounts/reenroll", url.Values{"id": {itoa(bob.ID)}}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("stripping a never-enrolled account recorded %v, want nothing", got)
	}
}

func enrolTOTP(f *fakeStore, id int64) {
	acct := f.accounts[id]
	acct.TotpEnabled = true
	acct.TotpSecret = pgtype.Text{String: "sealed", Valid: true}
	f.accounts[id] = acct
}

func TestStrippingTOTPOnAnUnknownAccountRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/settings/accounts/reenroll", url.Values{"id": {"4242"}}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("an unknown account id recorded %v, want nothing", got)
	}
}

// The Act outlives the account, because §5.4 captures the name and never a foreign key.

func TestAccountRemovalRecordsTheNameAfterTheRowIsGone(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	bob := seedAccount(t, f, "bob", roleViewer, "hunter2hunter2")
	f.acts = nil

	postForm(t, ac, base+"/settings/accounts/remove", url.Values{
		"id": {itoa(bob.ID)}, "confirm_name": {"bob"},
	}).Body.Close()

	if _, ok := f.accounts[bob.ID]; ok {
		t.Fatalf("the account survived the removal")
	}
	wantOneActBy(t, f, "account.removed", "@admin", "bob")
}

func TestAMistypedRemovalConfirmationRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	bob := seedAccount(t, f, "bob", roleViewer, "hunter2hunter2")
	f.acts = nil

	postForm(t, ac, base+"/settings/accounts/remove", url.Values{
		"id": {itoa(bob.ID)}, "confirm_name": {"bobb"},
	}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a mistyped confirmation recorded %v, want nothing", got)
	}
}

// The API dial. The cell reads as the switch the operator threw (spec §2.1).

func TestTheAPIDialRecordsTheSwitchTheOperatorThrew(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/settings/api", url.Values{"enabled": {"true"}}).Body.Close()
	wantOneActBy(t, f, "api.access.moved", "@admin", "API access · on")

	f.acts = nil
	postForm(t, ac, base+"/settings/api", url.Values{"enabled": {"false"}}).Body.Close()
	wantOneAct(t, f, "api.access.moved", "API access · off")
}

// The four provider acts. The slug is the whole subject, so no field the secret could ride
// exists at all (spec §4.2).

func TestTheFourSSOProviderActsEachRecordTheSlug(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/settings/sso", url.Values{
		"slug": {"okta"}, "name": {"Okta"}, "issuer": {"https://idp.example"},
		"client_id": {"cid"}, "client_secret": {"super-secret-value"},
	}).Body.Close()
	wantOneActBy(t, f, "sso.provider.declared", "@admin", "okta")
	id := f.ssoProviders[0].id

	f.acts = nil
	postForm(t, ac, base+"/settings/sso/update", url.Values{
		"id": {itoa(id)}, "slug": {"okta"}, "name": {"Okta"},
		"issuer": {"https://idp.example"}, "client_id": {"cid"}, "enabled": {"on"},
	}).Body.Close()
	wantOneActBy(t, f, "sso.provider.updated", "@admin", "okta")

	f.acts = nil
	postForm(t, ac, base+"/settings/sso/secret", url.Values{
		"id": {itoa(id)}, "client_secret": {"rotated-secret-value"},
	}).Body.Close()
	wantOneActBy(t, f, "sso.provider.secret.set", "@admin", "okta")

	f.acts = nil
	postForm(t, ac, base+"/settings/sso/delete", url.Values{"id": {itoa(id)}}).Body.Close()
	wantOneActBy(t, f, "sso.provider.withdrawn", "@admin", "okta")
}

// The form promises that a blank field keeps the stored secret, so a blank submit sets nothing
// and an unmoved secret writes no row (spec §7.6 ruling 2).

func TestABlankSecretSubmissionRecordsNothing(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedSSOProviderWithSecret(f, 1, "okta", "stored-secret", admin.ID)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil

	postForm(t, ac, base+"/settings/sso/secret", url.Values{"id": {"1"}, "client_secret": {""}}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a blank secret submission recorded %v, want nothing", got)
	}
}

func TestClearingASecretRecordsTheProviderItCleared(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedSSOProviderWithSecret(f, 1, "okta", "stored-secret", admin.ID)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil

	postForm(t, ac, base+"/settings/sso/secret", url.Values{"id": {"1"}, "clear_secret": {"on"}}).Body.Close()

	wantOneActBy(t, f, "sso.provider.secret.set", "@admin", "okta")
}

// Both names ride the removal's own RETURNING, so the row reads after the binding is gone
// (spec §4.2 hazard 1).

func TestAdminBindingRemovalRecordsTheProviderAndTheAccount(t *testing.T) {
	f := newFakeStore()
	bob := seedAccount(t, f, "bob", roleViewer, "hunter2hunter2")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-bob", bob.ID, "bob@corp")
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/settings/sso/identity/remove", url.Values{"id": {"1"}}).Body.Close()

	if len(f.ssoIdentities) != 0 {
		t.Fatalf("the binding survived the removal: %d rows", len(f.ssoIdentities))
	}
	wantOneAct(t, f, "sso.binding.removed", "okta · bob")
	// The account half of the cell carries the mark, as every rendered username does (spec §3.5).
	if got := act.SubjectCell(mustDecodeOne(t, f, "sso.binding.removed")); got != "okta · @bob" {
		t.Errorf("subject cell = %q, want %q", got, "okta · @bob")
	}
}

func TestRemovingAnUnknownBindingRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/settings/sso/identity/remove", url.Values{"id": {"4242"}}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("an unknown identity id recorded %v, want nothing", got)
	}
}

func TestSelfUnlinkRecordsTheProviderItUnlinked(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@corp")
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "okta-sub-alice"})
	ac := login(t, base, "alice", "unused-password-x")
	f.acts = nil

	postForm(t, ac, base+"/profile/sso/unlink", url.Values{"id": {"1"}}).Body.Close()

	wantOneActBy(t, f, "sso.unlinked", "@alice", "okta")
}

func TestUnlinkingABindingTheCallerDoesNotHoldRecordsNothing(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	bob := seedAccount(t, f, "bob", roleViewer, "unused-password-y")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@corp")
	addSSOIdentity(f, 1, "okta-sub-bob", bob.ID, "bob@corp")
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "okta-sub-alice"})
	ac := login(t, base, "alice", "unused-password-x")
	f.acts = nil

	postForm(t, ac, base+"/profile/sso/unlink", url.Values{"id": {"2"}}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("unlinking another account's binding recorded %v, want nothing", got)
	}
}

// The one act on a GET. Without the widened harvest of ticket #1830 this route ships unrecorded
// and silent (spec §7.2).

func TestTheSSOLinkCallbackRecordsOnAGET(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "okta-sub-alice", display: "alice@corp"})
	ac := login(t, base, "alice", "unused-password-x")
	f.acts = nil

	ssoLinkFlow(t, ac, base, "okta").Body.Close()

	wantOneActBy(t, f, "sso.binding.created", "@alice", "okta · alice")
}

func TestARefusedLinkCallbackRecordsNothing(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "okta-sub-alice", display: "alice@corp"})
	ac := login(t, base, "alice", "unused-password-x")
	f.acts = nil

	resp, err := ac.Get(base + "/profile/sso/okta/link/callback?state=forged&code=linkcode")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a forged callback state recorded %v, want nothing", got)
	}
}

func mustDecodeOne(t *testing.T, f *fakeStore, class string) act.Act {
	t.Helper()
	for _, a := range f.acts {
		if a.Action == class {
			_, decoded := decodeAct(t, a)
			return decoded
		}
	}
	t.Fatalf("no %s row was recorded; recorded %v", class, actClasses(f))
	return nil
}

// A login, a failed login, a TOTP step, a logout and a session revoke move which credential is
// presented, never who may act. POST /login and POST /forgot are unauthenticated and take an
// attacker-supplied username, so auditing them makes the corpus writable from the sign-in
// screen (spec §1.2, §2.3).

func TestNoLoginFamilyRouteWritesARow(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := newClient(t)
	postForm(t, c, base+"/login", url.Values{"username": {"admin"}, "password": {"wrong"}}).Body.Close()
	postForm(t, c, base+"/forgot", url.Values{"username": {"attacker-supplied"}}).Body.Close()
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil

	for _, path := range []string{
		"/profile/sessions/revoke-others", "/profile/session/revoke", "/logout",
	} {
		postForm(t, ac, base+path, url.Values{}).Body.Close()
	}

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("the login family recorded %v, want nothing", got)
	}
}

func TestTheGateStillReportsTheElevenLoginFamilyExemptions(t *testing.T) {
	c := parseWebPackage(t)
	rep := c.actAudit(liveActRules())

	var exempt []string
	for pattern, reason := range actExemptRoutes {
		if strings.HasPrefix(reason, "login family") {
			exempt = append(exempt, pattern)
		}
	}
	sort.Strings(exempt)
	if len(exempt) != 10 {
		t.Fatalf("actExemptRoutes names %d login-family POST routes, want 10: %v", len(exempt), exempt)
	}
	// GET /login/sso/{slug}/callback is the eleventh act and needs no entry: non-POST is exempt
	// by default, so the rule that holds it is that it reaches no Record call.
	for _, pattern := range append(exempt, "GET /login/sso/{slug}/callback") {
		if _, ok := c.routes[pattern]; !ok {
			t.Errorf("the harvest no longer holds %q", pattern)
			continue
		}
		if rep.auditable[pattern] {
			t.Errorf("%q now reads as auditable; §2.3's ruling is gone", pattern)
		}
		if rep.records[pattern] {
			t.Errorf("%q reaches a Record call", pattern)
		}
	}
}

// The round-trip test bars a payload field that is not a string or an int64, which is the hole a
// secret fits through. This holds the weaker claim §4.2 makes: no value typed into a limb-2
// secret field reaches a stored payload.

func TestNoLimb2SecretValueReachesAStoredSubject(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/settings/sso", url.Values{
		"slug": {"okta"}, "name": {"Okta"}, "issuer": {"https://idp.example"},
		"client_id": {"cid"}, "client_secret": {"super-secret-value"},
	}).Body.Close()
	postForm(t, ac, base+"/settings/sso/secret", url.Values{
		"id": {itoa(f.ssoProviders[0].id)}, "client_secret": {"rotated-secret-value"},
	}).Body.Close()
	page := body(t, postForm(t, ac, base+"/profile/tokens", url.Values{"name": {"ci reader"}}))
	enrol := body(t, postForm(t, ac, base+"/account/totp/enable", nil))
	totpSecret := secretRE.FindStringSubmatch(enrol)[1]
	code, err := auth.TOTPCode(totpSecret, serverClock)
	if err != nil {
		t.Fatal(err)
	}
	postForm(t, ac, base+"/account/totp/confirm", url.Values{"code": {code}}).Body.Close()

	minted := mintedTokenRE.FindAllString(page, -1)
	if len(minted) == 0 {
		t.Fatal("the mint revealed no token, so this proves nothing")
	}
	if len(f.acts) != 4 {
		t.Fatalf("recorded %v, want the four secret-adjacent acts", actClasses(f))
	}
	for _, a := range f.acts {
		payload := string(a.Subject) + string(a.Actor)
		for _, secret := range []string{
			"super-secret-value", "rotated-secret-value", "hunter2hunter2", totpSecret,
		} {
			if strings.Contains(payload, secret) {
				t.Errorf("%s carries the secret %q in its payload: %s", a.Action, secret, payload)
			}
		}
	}
	// The mint stores the public prefix, so the revealed plaintext reaches no payload at all.
	for _, a := range f.acts {
		if strings.Contains(string(a.Subject), minted[0]) {
			t.Errorf("%s carries the minted token plaintext: %s", a.Action, a.Subject)
		}
	}
}

// Every route in §2.1's limb-2 rows leaves the gate's pending set with this ticket (map #1826).

func TestNoLimb2RouteRemainsPending(t *testing.T) {
	for _, pattern := range []string{
		"POST /setup", "POST /reset", "POST /invite",
		"POST /profile/password", "POST /profile/tokens", "POST /profile/tokens/revoke",
		"POST /profile/sso/unlink", "POST /accounts", "POST /account/totp/confirm",
		"POST /settings/accounts", "POST /settings/accounts/role",
		"POST /settings/accounts/reenroll", "POST /settings/accounts/remove",
		"POST /settings/api", "POST /settings/sso", "POST /settings/sso/update",
		"POST /settings/sso/secret", "POST /settings/sso/delete",
		"POST /settings/sso/identity/remove", "GET /profile/sso/{slug}/link/callback",
	} {
		if owner, pending := actPending[pattern]; pending {
			t.Errorf("%q is still pending under %q", pattern, owner)
		}
	}
}
