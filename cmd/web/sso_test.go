package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// fakeSSOFlow stands in for the OIDC seam so the login- and link-flow tests assert the
// handler's state/nonce/cookie handling and identity binding without a live identity
// provider. AuthCodeURL echoes the minted state into the returned URL so a test can
// read it back off the redirect; Exchange returns the configured verified identity.
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

// addSSOProvider seeds an enabled OIDC provider in the fake store.
func addSSOProvider(f *fakeStore, id int64, slug, name string) {
	f.ssoNextID = id
	f.ssoProviders = append(f.ssoProviders, fakeSSOProvider{
		id: id, slug: slug, name: name, issuer: "https://idp.example", clientID: "cid",
		enabled: true, createdBy: 1, createdAt: obsClock,
	})
}

// addSSOIdentity seeds a verified (provider, sub) → account binding, the state an
// authenticated Profile self-link would have recorded (#319, ADR-0113).
func addSSOIdentity(f *fakeStore, providerID int64, sub string, accountID int64, display string) {
	f.ssoIdentNextID++
	f.ssoIdentities = append(f.ssoIdentities, fakeSSOIdentity{
		id: f.ssoIdentNextID, providerID: providerID, accountID: accountID,
		sub: sub, displayName: display, createdAt: obsClock,
	})
}

// stateFromRedirect parses the state parameter out of an IdP redirect Location.
func stateFromRedirect(t *testing.T, loc string) string {
	t.Helper()
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse redirect %q: %v", loc, err)
	}
	return u.Query().Get("state")
}

// The sign-in screen renders a button per enabled provider once one is configured,
// linking to its flow route — replacing the "not configured" affordance (#293).
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

// With no provider configured the honest not-configured state renders (the empty-state
// SignIn kept from the migration).
func TestSignInNoProviderShowsNotConfigured(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := start(t, f, "")
	page := getBody(t, newClient(t), base+"/login", http.StatusOK)
	if !strings.Contains(page, "Single sign-on not configured") {
		t.Errorf("sign-in missing the not-configured state with no provider; body: %s", page)
	}
}

// GET /login/sso/{slug} mints the transaction and redirects to the IdP: a 303 to the
// authorization URL, a signed transaction cookie, and the state echoed in both.
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
	// The transaction cookie was set, and a PKCE verifier was minted for the exchange.
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
	// The redirect URL handed to the flow is the callback on this host.
	if !strings.HasSuffix(flow.lastCfg.RedirectURL, "/login/sso/okta/callback") {
		t.Errorf("redirect URL = %q, want the okta callback", flow.lastCfg.RedirectURL)
	}
}

// The full flow: start → callback with the echoed state and a code matches the verified
// (provider, sub) to the bound local account and issues a session.
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
	// The session is real: a follow-up request to a gated page renders rather than
	// bouncing to /login.
	home := getBody(t, c, base+"/", http.StatusOK)
	if strings.Contains(home, "Sign in</h1>") {
		t.Errorf("post-SSO request was not authenticated; got the login page")
	}
}

// SSO must not downgrade a local second factor: an account that enrolled TOTP still
// lands on the two-factor step after a verified SSO assertion, rather than being logged
// straight in (#293 review).
func TestSSOCallbackStillRequiresTOTP(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	// Enrol TOTP on the account SSO will map to.
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
	// Not yet fully signed in: a gated request still bounces to /login (the pending
	// cookie is not a session).
	r3, _ := c.Get(base + "/")
	r3.Body.Close()
	if r3.StatusCode != http.StatusSeeOther || r3.Header.Get("Location") != "/login" {
		t.Errorf("SSO completed the login without the second factor: status=%d loc=%q", r3.StatusCode, r3.Header.Get("Location"))
	}
}

// A verified identity with no binding is refused, not provisioned and never mapped by a
// username (ADR-0113: authentication keys on a stored (provider, sub) binding).
func TestSSOCallbackRefusesUnlinkedIdentity(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	// A verified subject that no binding points at — even though a same-named account
	// exists, no username fallback admits it.
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
	// No session was issued: a gated request bounces to /login.
	r3, _ := c.Get(base + "/")
	r3.Body.Close()
	if r3.StatusCode != http.StatusSeeOther || r3.Header.Get("Location") != "/login" {
		t.Errorf("a refused SSO login left a session: status=%d loc=%q", r3.StatusCode, r3.Header.Get("Location"))
	}
}

// A callback whose state does not match the transaction cookie is refused (CSRF
// guard), and no session is issued.
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

// A callback with no transaction cookie (a stray or replayed callback) is refused.
func TestSSOCallbackWithoutTransactionRefused(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addSSOProvider(f, 1, "okta", "Okta")
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "okta-sub-alice"})

	c := newClient(t) // never started a flow, so holds no tx cookie
	r, err := c.Get(base + "/login/sso/okta/callback?state=x&code=y")
	if err != nil {
		t.Fatal(err)
	}
	page := body(t, r)
	if !strings.Contains(page, "expired") {
		t.Errorf("a cookie-less callback should be refused; body: %s", page)
	}
}

// An admin can declare a provider; the client secret is stored but never rendered
// back (write-only), and the list shows only that a secret is set.
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

// ssoLinkFlow drives an authenticated Profile self-link end to end: the link start
// (303 to the IdP with a minted state) then the link callback echoing that state and a
// code. It returns the callback response for the caller to assert on.
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

// A signed-in user links their own verified identity: the callback records a
// (provider, sub) → their-account binding and returns to the Profile (#319, ADR-0113).
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

// After a self-link, the verified subject signs the account in — the end-to-end chain
// the binding exists to serve.
func TestSSOSelfLinkThenLoginSucceeds(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "okta-sub-alice", display: "alice@corp"})

	// alice links from her Profile...
	ac := login(t, base, "alice", "unused-password-x")
	ssoLinkFlow(t, ac, base, "okta").Body.Close()

	// ...then a fresh SSO sign-in with the same subject admits her.
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

// A subject already bound to ANOTHER account cannot be re-linked to yours: the exclusive
// (provider, sub) is the whole point — no takeover by re-linking.
func TestSSOSelfLinkRefusesCollision(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	bob := seedAccount(t, f, "bob", roleViewer, "unused-password-y")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "shared-sub", bob.ID, "bob@corp") // already bob's
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

// Re-linking a subject already bound to your OWN account is a benign no-op.
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

// An account holds at most one identity per provider (ADR-0113). Linking a second,
// different subject for a provider already linked is refused — even via the direct link
// URL that bypasses the hidden button — and records no second binding.
func TestSSOSelfLinkOnePerProvider(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@corp") // already linked
	// The IdP now returns a DIFFERENT subject for the same provider.
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "okta-sub-alice-2", display: "alice.alt@corp"})
	ac := login(t, base, "alice", "unused-password-x")

	r := ssoLinkFlow(t, ac, base, "okta")
	r.Body.Close()
	if r.Header.Get("Location") != "/profile?linkerr=provider" {
		t.Fatalf("second link for a provider loc=%q, want /profile?linkerr=provider", r.Header.Get("Location"))
	}
	if len(f.ssoIdentities) != 1 {
		t.Errorf("a second identity was bound for an already-linked provider: %d rows", len(f.ssoIdentities))
	}
}

// The self-link routes are authenticated: an anonymous caller is bounced to /login.
func TestSSOLinkRequiresLogin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addSSOProvider(f, 1, "okta", "Okta")
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "x"})

	c := newClient(t) // not signed in
	r, err := c.Get(base + "/profile/sso/okta/link")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusSeeOther || r.Header.Get("Location") != "/login" {
		t.Errorf("anonymous link start status=%d loc=%q, want 303 to /login", r.StatusCode, r.Header.Get("Location"))
	}
}

// A user unlinks their OWN identity; it can no longer sign in. The unlink is
// account-scoped, so it never removes another account's binding.
func TestSSOUnlinkRemovesOwnBindingOnly(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	bob := seedAccount(t, f, "bob", roleViewer, "unused-password-y")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@corp") // id 1
	addSSOIdentity(f, 1, "okta-sub-bob", bob.ID, "bob@corp")       // id 2
	base := startWithSSO(t, f, &fakeSSOFlow{sub: "okta-sub-alice"})
	ac := login(t, base, "alice", "unused-password-x")

	// alice cannot unlink bob's binding (id 2): scoped to her account, it no-ops.
	postForm(t, ac, base+"/profile/sso/unlink", url.Values{"id": {"2"}}).Body.Close()
	if len(f.ssoIdentities) != 2 {
		t.Fatalf("alice unlinked another account's binding: %d rows left", len(f.ssoIdentities))
	}
	// alice unlinks her own (id 1).
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

// An admin removes any binding (offboarding / seat reassignment); the identity then
// fails to sign in. A viewer is refused the route.
func TestSSOAdminRemoveBinding(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@corp") // id 1
	base := start(t, f, "")

	// A viewer cannot remove a binding.
	vc := login(t, base, "viewer", "hunter2hunter2")
	vr := postForm(t, vc, base+"/settings/sso/identity/remove", url.Values{"id": {"1"}})
	vr.Body.Close()
	if vr.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer remove-binding status=%d, want 403", vr.StatusCode)
	}
	if len(f.ssoIdentities) != 1 {
		t.Fatalf("viewer removed a binding despite the admin gate")
	}
	// The admin removes it.
	ac := login(t, base, "admin", "hunter2hunter2")
	ar := postForm(t, ac, base+"/settings/sso/identity/remove", url.Values{"id": {"1"}})
	ar.Body.Close()
	if len(f.ssoIdentities) != 0 {
		t.Errorf("admin remove left the binding: %d rows", len(f.ssoIdentities))
	}
}

// The admin SSO settings tab lists each binding — provider, account and label — so an
// admin can see and revoke who an identity authenticates as.
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

// The Profile shows an account's linked identities and offers a Link button for an
// enabled provider it has not linked yet.
func TestProfileShowsLinkedIdentitiesAndLinkButton(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOProvider(f, 2, "google", "Google") // enabled, not linked
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@corp")
	base := start(t, f, "")
	ac := login(t, base, "alice", "unused-password-x")

	page := getBody(t, ac, base+"/profile", http.StatusOK)
	// The linked identity renders with its label and an unlink control.
	for _, want := range []string{"Linked identities", "alice@corp", `action="/profile/sso/unlink"`} {
		if !strings.Contains(page, want) {
			t.Errorf("profile missing linked-identity element %q", want)
		}
	}
	// The unlinked provider offers a Link button; the already-linked one does not reappear.
	if !strings.Contains(page, `href="/profile/sso/google/link"`) {
		t.Errorf("profile missing a Link button for the unlinked provider")
	}
	if strings.Contains(page, `href="/profile/sso/okta/link"`) {
		t.Errorf("profile offered a Link button for an already-linked provider")
	}
}

// seedSSOProviderWithSecret seeds an enabled provider that already stores a client
// secret, so the update-secret write path can be exercised against a real starting
// state.
func seedSSOProviderWithSecret(f *fakeStore, id int64, slug, secret string, createdBy int64) {
	f.ssoNextID = id
	f.ssoProviders = append(f.ssoProviders, fakeSSOProvider{
		id: id, slug: slug, name: "Okta", issuer: "https://idp.example", clientID: "cid",
		secret: secret, hasSecret: true,
		enabled: true, createdBy: createdBy, createdAt: obsClock,
	})
}

// A blank secret field with the clear box unchecked must KEEP the stored secret — the
// form promises "set — leave blank to keep". Submitting it wiped the secret before the
// #318 fix; now it no-ops (mirroring the channel-secret form).
func TestSettingsSSOSecretBlankKeepsStored(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedSSOProviderWithSecret(f, 1, "okta", "stored-secret", admin.ID)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/sso/secret", url.Values{
		"id": {"1"}, "client_secret": {""}, // no clear_secret, blank field
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

// The explicit clear box removes a stored secret (the only way to, now).
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

// A non-blank secret field replaces the stored secret.
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

// The SSO config mutations are admin acts: a viewer is refused.
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
