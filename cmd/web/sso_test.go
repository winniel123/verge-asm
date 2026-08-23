package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// fakeSSOFlow stands in for the OIDC seam so the login-flow tests assert the
// handler's state/nonce/cookie handling and account mapping without a live identity
// provider. AuthCodeURL echoes the minted state into the returned URL so a test can
// read it back off the redirect; Exchange returns the configured username.
type fakeSSOFlow struct {
	username string
	authErr  error
	exchErr  error

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

func (f *fakeSSOFlow) Exchange(_ context.Context, cfg ssoConfig, code, verifier, nonce string) (string, error) {
	if f.exchErr != nil {
		return "", f.exchErr
	}
	f.lastCode = code
	return f.username, nil
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
		claim: "preferred_username", enabled: true, createdBy: 1, createdAt: obsClock,
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

// The full flow: start → callback with the echoed state and a code maps the verified
// identity to an existing local account and issues a session.
func TestSSOCallbackSignsInExistingAccount(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	flow := &fakeSSOFlow{username: "alice"}
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

// A verified identity with no matching local account is refused, not provisioned
// (ADR-0112: SSO authenticates existing accounts, never creates them).
func TestSSOCallbackRefusesUnknownIdentity(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addSSOProvider(f, 1, "okta", "Okta")
	flow := &fakeSSOFlow{username: "ghost"} // no such local account
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
	if r2.StatusCode != http.StatusOK || !strings.Contains(page, "No account here matches that identity") {
		t.Fatalf("unknown identity: status=%d, want 200 with a refusal; body: %s", r2.StatusCode, page)
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
	flow := &fakeSSOFlow{username: "alice"}
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
	base := startWithSSO(t, f, &fakeSSOFlow{username: "alice"})

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
		"username_claim": {"preferred_username"},
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
