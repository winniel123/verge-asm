package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type fakeSSOFlow struct {
	sub     string
	display string
	authErr error
	exchErr error

	lastCfg      ssoConfig
	lastState    string
	lastNonce    string
	lastVerifier string
	lastCode     string
}

func (f *fakeSSOFlow) AuthCodeURL(_ context.Context, cfg ssoConfig, state, nonce, verifier string) (string, error) {
	if f.authErr != nil {
		return "", f.authErr
	}
	f.lastCfg, f.lastState, f.lastNonce, f.lastVerifier = cfg, state, nonce, verifier
	return "https://idp.example/authorize?client_id=" + url.QueryEscape(cfg.ClientID) +
		"&state=" + url.QueryEscape(state) + "&nonce=" + url.QueryEscape(nonce), nil
}

func (f *fakeSSOFlow) Exchange(_ context.Context, cfg ssoConfig, code, verifier, nonce string) (ssoIdentity, error) {
	if f.exchErr != nil {
		return ssoIdentity{}, f.exchErr
	}
	f.lastCode = code
	return ssoIdentity{Sub: f.sub, Display: f.display}, nil
}

func startWithSSO(t *testing.T, f *fakeStore, flow ssoFlow) string {
	t.Helper()
	srv := newServer(f, testKey, "", fixedClock())
	srv.sso = flow
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)
	return ts.URL
}

func addSSOProvider(f *fakeStore, id int64, slug, name string) {
	f.ssoNextID = id
	f.ssoProviders = append(f.ssoProviders, fakeSSOProvider{
		id: id, slug: slug, name: name, issuer: "https://idp.example", clientID: "cid",
		enabled: true, createdBy: 1, createdAt: obsClock,
	})
}

func addSSOIdentity(f *fakeStore, providerID int64, sub string, accountID int64, display string) {
	f.ssoIdentNextID++
	f.ssoIdentities = append(f.ssoIdentities, fakeSSOIdentity{
		id: f.ssoIdentNextID, providerID: providerID, accountID: accountID,
		sub: sub, displayName: display, createdAt: obsClock,
	})
}

func stateFromRedirect(t *testing.T, loc string) string {
	t.Helper()
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse redirect %q: %v", loc, err)
	}
	return u.Query().Get("state")
}

func TestSignInRendersSSOButtons(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addSSOProvider(f, 1, "okta", "Okta")

	base := start(t, f, "")
	page := getBody(t, newClient(t), base+"/login", http.StatusOK)

	for _, want := range []string{`href="/login/sso/okta"`, "Continue with Okta"} {
		if !strings.Contains(page, want) {
			t.Errorf("sign-in page missing %q; body: %s", want, page)
		}
	}
	if strings.Contains(page, "Single sign-on not configured") {
		t.Errorf("sign-in still shows the not-configured state with a provider present")
	}
}

func TestSignInNoProviderShowsNotConfigured(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := start(t, f, "")
	page := getBody(t, newClient(t), base+"/login", http.StatusOK)
	if !strings.Contains(page, "Single sign-on not configured") {
		t.Errorf("sign-in missing the not-configured state with no provider; body: %s", page)
	}
}

func TestSSOStartRedirectsToIdP(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addSSOProvider(f, 1, "okta", "Okta")
	flow := &fakeSSOFlow{}
	base := startWithSSO(t, f, flow)

	c := newClient(t)
	resp, err := c.Get(base + "/login/sso/okta")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("sso start status = %d, want 303", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if !strings.HasPrefix(loc, "https://idp.example/authorize") {
		t.Errorf("sso start Location = %q, want the IdP authorize URL", loc)
	}
	if stateFromRedirect(t, loc) == "" || stateFromRedirect(t, loc) != flow.lastState {
		t.Errorf("redirect state %q does not match minted state %q", stateFromRedirect(t, loc), flow.lastState)
	}
	var txSet bool
	for _, ck := range resp.Cookies() {
		if ck.Name == ssoTxCookie && ck.Value != "" {
			txSet = true
		}
	}
	if !txSet {
		t.Errorf("sso start did not set the %s transaction cookie", ssoTxCookie)
	}
	if flow.lastVerifier == "" {
		t.Errorf("sso start did not mint a PKCE verifier")
	}
	if !strings.HasSuffix(flow.lastCfg.RedirectURL, "/login/sso/okta/callback") {
		t.Errorf("redirect URL = %q, want the okta callback", flow.lastCfg.RedirectURL)
	}
}

func TestSSOCallbackSignsInExistingAccount(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@example.com")
	flow := &fakeSSOFlow{sub: "okta-sub-alice"}
	base := startWithSSO(t, f, flow)

	c := newClient(t)
	r1, err := c.Get(base + "/login/sso/okta")
	if err != nil {
		t.Fatal(err)
	}
	r1.Body.Close()
	state := stateFromRedirect(t, r1.Header.Get("Location"))

	r2, err := c.Get(base + "/login/sso/okta/callback?state=" + url.QueryEscape(state) + "&code=abc123")
	if err != nil {
		t.Fatal(err)
	}
	r2.Body.Close()
	if r2.StatusCode != http.StatusSeeOther || r2.Header.Get("Location") != "/" {
		t.Fatalf("callback status=%d loc=%q, want 303 to /", r2.StatusCode, r2.Header.Get("Location"))
	}
	if flow.lastCode != "abc123" {
		t.Errorf("flow.Exchange got code %q, want abc123", flow.lastCode)
	}
	home := getBody(t, c, base+"/", http.StatusOK)
	if strings.Contains(home, "Sign in</h1>") {
		t.Errorf("post-SSO request was not authenticated; got the login page")
	}
}

func TestSSOCallbackStillRequiresTOTP(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	a := f.accounts[alice.ID]
	a.TotpEnabled = true
	f.accounts[alice.ID] = a

	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@example.com")
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "okta-sub-alice"})

	c := newClient(t)
	r1, _ := c.Get(base + "/login/sso/okta")
	r1.Body.Close()
	state := stateFromRedirect(t, r1.Header.Get("Location"))

	r2, err := c.Get(base + "/login/sso/okta/callback?state=" + url.QueryEscape(state) + "&code=abc")
	if err != nil {
		t.Fatal(err)
	}
	page := body(t, r2)
	if !strings.Contains(page, "Two-factor check") {
		t.Errorf("a TOTP-enrolled account should land on the two-factor step after SSO; body: %s", page)
	}
	r3, _ := c.Get(base + "/")
	r3.Body.Close()
	if r3.StatusCode != http.StatusSeeOther || r3.Header.Get("Location") != "/login" {
		t.Errorf("SSO completed the login without the second factor: status=%d loc=%q", r3.StatusCode, r3.Header.Get("Location"))
	}
}

func TestSSOCallbackRefusesUnlinkedIdentity(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	flow := &fakeSSOFlow{sub: "okta-sub-unlinked"}
	base := startWithSSO(t, f, flow)

	c := newClient(t)
	r1, _ := c.Get(base + "/login/sso/okta")
	r1.Body.Close()
	state := stateFromRedirect(t, r1.Header.Get("Location"))

	r2, err := c.Get(base + "/login/sso/okta/callback?state=" + url.QueryEscape(state) + "&code=abc")
	if err != nil {
		t.Fatal(err)
	}
	page := body(t, r2)
	if r2.StatusCode != http.StatusOK || !strings.Contains(page, "not linked to an account here") {
		t.Fatalf("unlinked identity: status=%d, want 200 with a refusal; body: %s", r2.StatusCode, page)
	}
	r3, _ := c.Get(base + "/")
	r3.Body.Close()
	if r3.StatusCode != http.StatusSeeOther || r3.Header.Get("Location") != "/login" {
		t.Errorf("a refused SSO login left a session: status=%d loc=%q", r3.StatusCode, r3.Header.Get("Location"))
	}
}

func TestSSOCallbackRejectsStateMismatch(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	flow := &fakeSSOFlow{sub: "okta-sub-alice"}
	base := startWithSSO(t, f, flow)

	c := newClient(t)
	r1, _ := c.Get(base + "/login/sso/okta")
	r1.Body.Close()

	r2, err := c.Get(base + "/login/sso/okta/callback?state=forged&code=abc")
	if err != nil {
		t.Fatal(err)
	}
	page := body(t, r2)
	if !strings.Contains(page, "could not be verified") {
		t.Errorf("state mismatch should be refused; body: %s", page)
	}
	r3, _ := c.Get(base + "/")
	r3.Body.Close()
	if r3.StatusCode != http.StatusSeeOther {
		t.Errorf("a state-mismatched callback issued a session")
	}
}

func TestSSOCallbackWithoutTransactionRefused(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addSSOProvider(f, 1, "okta", "Okta")
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "okta-sub-alice"})

	c := newClient(t)
	r, err := c.Get(base + "/login/sso/okta/callback?state=x&code=y")
	if err != nil {
		t.Fatal(err)
	}
	page := body(t, r)
	if !strings.Contains(page, "expired") {
		t.Errorf("a cookie-less callback should be refused; body: %s", page)
	}
}

func TestSettingsSSOCreateSecretWriteOnly(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/sso", url.Values{
		"slug": {"okta"}, "name": {"Okta"}, "issuer": {"https://idp.example"},
		"client_id": {"cid"}, "client_secret": {"super-secret-value"},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("create sso provider status = %d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()

	page := settingsTabBody(t, ac, base, "sso")
	for _, want := range []string{"Okta", "/okta", "https://idp.example", "cid"} {
		if !strings.Contains(page, want) {
			t.Errorf("sso settings missing %q; body: %s", want, page)
		}
	}
	if strings.Contains(page, "super-secret-value") {
		t.Errorf("sso settings leaked the client secret into the render path")
	}
}

func ssoLinkFlow(t *testing.T, ac *http.Client, base, slug string) *http.Response {
	t.Helper()
	r1, err := ac.Get(base + "/profile/sso/" + slug + "/link")
	if err != nil {
		t.Fatal(err)
	}
	r1.Body.Close()
	if r1.StatusCode != http.StatusSeeOther {
		t.Fatalf("link start status=%d, want 303 (loc %q)", r1.StatusCode, r1.Header.Get("Location"))
	}
	state := stateFromRedirect(t, r1.Header.Get("Location"))
	r2, err := ac.Get(base + "/profile/sso/" + slug + "/link/callback?state=" + url.QueryEscape(state) + "&code=linkcode")
	if err != nil {
		t.Fatal(err)
	}
	return r2
}

func TestSSOSelfLinkBindsIdentity(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "okta-sub-alice", display: "alice@corp"})
	ac := login(t, base, "alice", "unused-password-x")

	r := ssoLinkFlow(t, ac, base, "okta")
	r.Body.Close()
	if r.StatusCode != http.StatusSeeOther || r.Header.Get("Location") != "/profile?linked=1" {
		t.Fatalf("link callback status=%d loc=%q, want 303 to /profile?linked=1", r.StatusCode, r.Header.Get("Location"))
	}
	if len(f.ssoIdentities) != 1 {
		t.Fatalf("link recorded %d bindings, want 1", len(f.ssoIdentities))
	}
	got := f.ssoIdentities[0]
	if got.providerID != 1 || got.accountID != alice.ID || got.sub != "okta-sub-alice" || got.displayName != "alice@corp" {
		t.Errorf("binding = %+v, want provider 1 / account %d / sub okta-sub-alice / display alice@corp", got, alice.ID)
	}
}

func TestSSOSelfLinkThenLoginSucceeds(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "okta-sub-alice", display: "alice@corp"})

	ac := login(t, base, "alice", "unused-password-x")
	ssoLinkFlow(t, ac, base, "okta").Body.Close()

	lc := newClient(t)
	r1, _ := lc.Get(base + "/login/sso/okta")
	r1.Body.Close()
	state := stateFromRedirect(t, r1.Header.Get("Location"))
	r2, _ := lc.Get(base + "/login/sso/okta/callback?state=" + url.QueryEscape(state) + "&code=abc")
	r2.Body.Close()
	if r2.StatusCode != http.StatusSeeOther || r2.Header.Get("Location") != "/" {
		t.Fatalf("post-link SSO login status=%d loc=%q, want 303 to /", r2.StatusCode, r2.Header.Get("Location"))
	}
}

func TestSSOSelfLinkRefusesCollision(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	bob := seedAccount(t, f, "bob", roleViewer, "unused-password-y")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "shared-sub", bob.ID, "bob@corp")
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "shared-sub", display: "someone"})
	ac := login(t, base, "alice", "unused-password-x")

	r := ssoLinkFlow(t, ac, base, "okta")
	r.Body.Close()
	if r.Header.Get("Location") != "/profile?linkerr=elsewhere" {
		t.Fatalf("collision link loc=%q, want /profile?linkerr=elsewhere", r.Header.Get("Location"))
	}
	if len(f.ssoIdentities) != 1 || f.ssoIdentities[0].accountID != bob.ID {
		t.Errorf("collision link mutated the existing binding: %+v (alice=%d bob=%d)", f.ssoIdentities, alice.ID, bob.ID)
	}
}

func TestSSOSelfLinkIdempotent(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@corp")
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "okta-sub-alice", display: "alice@corp"})
	ac := login(t, base, "alice", "unused-password-x")

	r := ssoLinkFlow(t, ac, base, "okta")
	r.Body.Close()
	if r.Header.Get("Location") != "/profile?linked=exists" {
		t.Fatalf("re-link loc=%q, want /profile?linked=exists", r.Header.Get("Location"))
	}
	if len(f.ssoIdentities) != 1 {
		t.Errorf("re-link created a duplicate binding: %d rows", len(f.ssoIdentities))
	}
}

func TestSSOSelfLinkOnePerProvider(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@corp")
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "okta-sub-alice-2", display: "alice.alt@corp"})
	ac := login(t, base, "alice", "unused-password-x")

	// The Profile hides the Link button once linked, so the route must refuse the direct URL too.
	r := ssoLinkFlow(t, ac, base, "okta")
	r.Body.Close()
	if r.Header.Get("Location") != "/profile?linkerr=provider" {
		t.Fatalf("second link for a provider loc=%q, want /profile?linkerr=provider", r.Header.Get("Location"))
	}
	if len(f.ssoIdentities) != 1 {
		t.Errorf("a second identity was bound for an already-linked provider: %d rows", len(f.ssoIdentities))
	}
}

func TestSSOLinkRequiresLogin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addSSOProvider(f, 1, "okta", "Okta")
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "x"})

	c := newClient(t)
	r, err := c.Get(base + "/profile/sso/okta/link")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusSeeOther || r.Header.Get("Location") != "/login" {
		t.Errorf("anonymous link start status=%d loc=%q, want 303 to /login", r.StatusCode, r.Header.Get("Location"))
	}
}

func TestSSOUnlinkRemovesOwnBindingOnly(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	bob := seedAccount(t, f, "bob", roleViewer, "unused-password-y")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@corp")
	addSSOIdentity(f, 1, "okta-sub-bob", bob.ID, "bob@corp")
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "okta-sub-alice"})
	ac := login(t, base, "alice", "unused-password-x")

	postForm(t, ac, base+"/profile/sso/unlink", url.Values{"id": {"2"}}).Body.Close()
	if len(f.ssoIdentities) != 2 {
		t.Fatalf("alice unlinked another account's binding: %d rows left", len(f.ssoIdentities))
	}
	r := postForm(t, ac, base+"/profile/sso/unlink", url.Values{"id": {"1"}})
	r.Body.Close()
	if r.Header.Get("Location") != "/profile?unlinked=1" {
		t.Fatalf("unlink loc=%q, want /profile?unlinked=1", r.Header.Get("Location"))
	}
	for _, i := range f.ssoIdentities {
		if i.accountID == alice.ID {
			t.Errorf("alice's binding survived the unlink")
		}
	}
}

func TestSSOAdminRemoveBinding(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@corp")
	base := start(t, f, "")

	vc := login(t, base, "viewer", "hunter2hunter2")
	vr := postForm(t, vc, base+"/settings/sso/identity/remove", url.Values{"id": {"1"}})
	vr.Body.Close()
	if vr.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer remove-binding status=%d, want 403", vr.StatusCode)
	}
	if len(f.ssoIdentities) != 1 {
		t.Fatalf("viewer removed a binding despite the admin gate")
	}
	ac := login(t, base, "admin", "hunter2hunter2")
	ar := postForm(t, ac, base+"/settings/sso/identity/remove", url.Values{"id": {"1"}})
	ar.Body.Close()
	if len(f.ssoIdentities) != 0 {
		t.Errorf("admin remove left the binding: %d rows", len(f.ssoIdentities))
	}
}

func TestSettingsSSOListsBindings(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	_ = admin
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@corp")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := settingsTabBody(t, ac, base, "sso")
	for _, want := range []string{"Linked identities", "alice", "alice@corp", "/settings/sso/identity/remove"} {
		if !strings.Contains(page, want) {
			t.Errorf("sso settings bindings view missing %q", want)
		}
	}
}

func TestProfileShowsLinkedIdentitiesAndLinkButton(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOProvider(f, 2, "google", "Google")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@corp")
	base := start(t, f, "")
	ac := login(t, base, "alice", "unused-password-x")

	page := getBody(t, ac, base+"/profile", http.StatusOK)
	for _, want := range []string{"Linked identities", "alice@corp", `action="/profile/sso/unlink"`} {
		if !strings.Contains(page, want) {
			t.Errorf("profile missing linked-identity element %q", want)
		}
	}
	if !strings.Contains(page, `href="/profile/sso/google/link"`) {
		t.Errorf("profile missing a Link button for the unlinked provider")
	}
	if strings.Contains(page, `href="/profile/sso/okta/link"`) {
		t.Errorf("profile offered a Link button for an already-linked provider")
	}
}

func seedSSOProviderWithSecret(f *fakeStore, id int64, slug, secret string, createdBy int64) {
	f.ssoNextID = id
	f.ssoProviders = append(f.ssoProviders, fakeSSOProvider{
		id: id, slug: slug, name: "Okta", issuer: "https://idp.example", clientID: "cid",
		secret: secret, hasSecret: true,
		enabled: true, createdBy: createdBy, createdAt: obsClock,
	})
}

func TestSettingsSSOSecretBlankKeepsStored(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedSSOProviderWithSecret(f, 1, "okta", "stored-secret", admin.ID)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// The form promises that a blank field keeps the stored secret, so submitting one must no-op.
	resp := postForm(t, ac, base+"/settings/sso/secret", url.Values{
		"id": {"1"}, "client_secret": {""},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("update secret status = %d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()

	if !f.ssoProviders[0].hasSecret || f.ssoProviders[0].secret != "stored-secret" {
		t.Errorf("a blank update-secret submission wiped the stored secret: hasSecret=%v secret=%q",
			f.ssoProviders[0].hasSecret, f.ssoProviders[0].secret)
	}
}

func TestSettingsSSOSecretClearBoxRemoves(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedSSOProviderWithSecret(f, 1, "okta", "stored-secret", admin.ID)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/sso/secret", url.Values{
		"id": {"1"}, "clear_secret": {"on"},
	})
	resp.Body.Close()
	if f.ssoProviders[0].hasSecret {
		t.Errorf("the clear box did not remove the stored secret")
	}
}

func TestSettingsSSOSecretValueReplaces(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedSSOProviderWithSecret(f, 1, "okta", "stored-secret", admin.ID)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/sso/secret", url.Values{
		"id": {"1"}, "client_secret": {"rotated-secret"},
	})
	resp.Body.Close()
	if !f.ssoProviders[0].hasSecret || f.ssoProviders[0].secret != "rotated-secret" {
		t.Errorf("a typed secret did not replace the stored one: hasSecret=%v secret=%q",
			f.ssoProviders[0].hasSecret, f.ssoProviders[0].secret)
	}
}

func TestSettingsSSORequiresAdmin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	vc := login(t, base, "viewer", "hunter2hunter2")

	resp := postForm(t, vc, base+"/settings/sso", url.Values{
		"slug": {"okta"}, "name": {"Okta"}, "issuer": {"https://idp.example"}, "client_id": {"cid"},
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer POST /settings/sso status = %d, want 403", resp.StatusCode)
	}
	if len(f.ssoProviders) != 0 {
		t.Errorf("viewer created an SSO provider despite the admin gate")
	}
}
