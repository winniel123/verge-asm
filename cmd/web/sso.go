package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
)

// Single sign-on — the OIDC authorization-code flow (#293, ADR-0112). SSO is admitted
// as cryptographically-verified OIDC (the id_token's signature, issuer and a per-login
// nonce are checked) — never reverse-proxy header-trust, which stays refused. It
// authenticates an EXISTING local account matched by the provider's configured
// username claim; it never creates one, so a broad IdP cannot silently mint accounts,
// and the session's role is still read from the local account row on every request.
//
// The flow is two hops, both unauthenticated by construction (a caller signing in has
// no session yet):
//
//   GET /login/sso/{slug}          → mint state/nonce/PKCE, redirect to the IdP.
//   GET /login/sso/{slug}/callback → verify state, exchange the code, verify the
//                                    id_token, match the username, issue the session.
//
// The state/nonce/PKCE-verifier ride an HMAC-signed, short-lived cookie between the
// two hops (the app keeps no server-side session store), so the callback can trust the
// state it echoes back was minted here — CSRF and replay are both closed.

const (
	ssoTxCookie = "verge_sso_tx"
	ssoTxTTL    = 10 * time.Minute
	// ssoTxDomain is the type tag mixed into the transaction cookie's signature, so a
	// session cookie can never verify as an ssoTx (or vice-versa) even under the same
	// signing key — the signed-value analogue of the session cookie's Kind.
	ssoTxDomain = "sso-tx"
)

// ssoConfig is the resolved provider config the flow needs, free of DB types so the
// ssoFlow seam (and its test fake) never depend on the store layer.
type ssoConfig struct {
	Slug          string
	Issuer        string
	ClientID      string
	ClientSecret  string
	UsernameClaim string
	RedirectURL   string
}

// ssoFlow is the OIDC seam. The real implementation (oidcFlow) uses go-oidc + oauth2
// over the network; tests inject a fake so a login flow asserts its state/nonce/cookie
// handling and account mapping without a live identity provider.
type ssoFlow interface {
	// AuthCodeURL returns the IdP authorization URL to redirect the user to, for the
	// given config and the state/nonce/PKCE verifier the caller minted.
	AuthCodeURL(ctx context.Context, cfg ssoConfig, state, nonce, pkceVerifier string) (string, error)
	// Exchange completes the callback: it exchanges the code, verifies the returned
	// id_token (signature, issuer, audience) and that its nonce matches, and returns
	// the username carried in cfg.UsernameClaim.
	Exchange(ctx context.Context, cfg ssoConfig, code, pkceVerifier, nonce string) (username string, err error)
}

// oidcFlow is the production ssoFlow: OpenID Connect discovery + authorization-code
// exchange with PKCE, all token verification delegated to the vetted go-oidc/oauth2
// libraries rather than hand-rolled (ADR-0112). A discovered *oidc.Provider is built
// once per issuer and cached — it is meant to be reused, and go-oidc's verifier caches
// the JWKS behind it — so a login is not two discovery/JWKS round-trips.
type oidcFlow struct {
	httpClient *http.Client

	mu    sync.Mutex
	byIss map[string]*oidc.Provider
}

// newOIDCFlow builds the production flow with an initialised provider cache.
func newOIDCFlow(client *http.Client) *oidcFlow {
	return &oidcFlow{httpClient: client, byIss: map[string]*oidc.Provider{}}
}

// provider returns the cached provider for an issuer, discovering it once on first use.
func (f *oidcFlow) provider(ctx context.Context, issuer string) (*oidc.Provider, error) {
	f.mu.Lock()
	prov, ok := f.byIss[issuer]
	f.mu.Unlock()
	if ok {
		return prov, nil
	}
	prov, err := oidc.NewProvider(oidc.ClientContext(ctx, f.httpClient), issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}
	f.mu.Lock()
	f.byIss[issuer] = prov
	f.mu.Unlock()
	return prov, nil
}

// oauth2Config builds the oauth2 config for a provider over its discovered endpoints.
func (f *oidcFlow) oauth2Config(cfg ssoConfig, prov *oidc.Provider) oauth2.Config {
	return oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     prov.Endpoint(),
		RedirectURL:  cfg.RedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
}

func (f *oidcFlow) AuthCodeURL(ctx context.Context, cfg ssoConfig, state, nonce, pkceVerifier string) (string, error) {
	prov, err := f.provider(ctx, cfg.Issuer)
	if err != nil {
		return "", err
	}
	oc := f.oauth2Config(cfg, prov)
	// state defeats CSRF, nonce binds the id_token to this login, S256 PKCE binds the
	// code to this client — all three are verified on the callback.
	return oc.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(pkceVerifier)), nil
}

func (f *oidcFlow) Exchange(ctx context.Context, cfg ssoConfig, code, pkceVerifier, nonce string) (string, error) {
	ctx = oidc.ClientContext(ctx, f.httpClient)
	prov, err := f.provider(ctx, cfg.Issuer)
	if err != nil {
		return "", err
	}
	oc := f.oauth2Config(cfg, prov)
	tok, err := oc.Exchange(ctx, code, oauth2.VerifierOption(pkceVerifier))
	if err != nil {
		return "", fmt.Errorf("oidc code exchange: %w", err)
	}
	rawID, ok := tok.Extra("id_token").(string)
	if !ok || rawID == "" {
		return "", errors.New("oidc: token response carried no id_token")
	}
	idToken, err := prov.Verifier(&oidc.Config{ClientID: cfg.ClientID}).Verify(ctx, rawID)
	if err != nil {
		return "", fmt.Errorf("oidc id_token verify: %w", err)
	}
	if idToken.Nonce != nonce {
		return "", errors.New("oidc: id_token nonce mismatch")
	}
	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return "", fmt.Errorf("oidc id_token claims: %w", err)
	}
	username, _ := claims[cfg.UsernameClaim].(string)
	if username == "" {
		return "", fmt.Errorf("oidc: id_token carried no %q claim", cfg.UsernameClaim)
	}
	return username, nil
}

// ssoTx is the login transaction carried in the signed cookie between the redirect to
// the IdP and the callback. It is minted at /login/sso/{slug}, verified at the
// callback, and never trusted past its short expiry.
type ssoTx struct {
	Slug     string    `json:"slug"`
	State    string    `json:"state"`
	Nonce    string    `json:"nonce"`
	Verifier string    `json:"pkce"`
	Expires  time.Time `json:"exp"`
}

// ssoRedirectURL is the absolute callback URL the IdP returns to. It is taken from the
// configured external base URL (VERGE_EXTERNAL_URL) when set — the trusted origin the
// deployment is reached at — so the redirect_uri handed to the IdP never derives from
// the attacker-influenceable Host header. Where none is configured it falls back to the
// request host (https when the request arrived over TLS or a TLS-terminating proxy
// fronts it, the same signal the Secure cookie flag uses); the OIDC redirect_uri must
// still match the value registered at the IdP, which bounds that fallback.
func (s *server) ssoRedirectURL(r *http.Request, slug string) string {
	path := "/login/sso/" + slug + "/callback"
	if s.externalURL != "" {
		return strings.TrimRight(s.externalURL, "/") + path
	}
	scheme := "http"
	if r.TLS != nil || s.secureCookies {
		scheme = "https"
	}
	return scheme + "://" + r.Host + path
}

// ssoStart begins the OIDC flow for the provider named in the path. It is
// unauthenticated (a caller signing in has no session); an already-signed-in caller is
// bounced home. An unknown or disabled slug, or a provider misconfiguration, falls
// back to the login form with an honest error rather than a raw 500.
func (s *server) ssoStart(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentAccount(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	slug := r.PathValue("slug")
	prov, err := s.store.GetSSOProviderForAuth(r.Context(), slug)
	if err != nil {
		s.render(w, "login", s.loginData(r.Context(), "Single sign-on is not available. Sign in with your password."))
		return
	}

	state, nonce := randToken(), randToken()
	verifier := oauth2.GenerateVerifier()
	cfg := ssoConfig{
		Slug: prov.Slug, Issuer: prov.Issuer, ClientID: prov.ClientID,
		ClientSecret: prov.ClientSecret.String, UsernameClaim: prov.UsernameClaim,
		RedirectURL: s.ssoRedirectURL(r, prov.Slug),
	}
	authURL, err := s.sso.AuthCodeURL(r.Context(), cfg, state, nonce, verifier)
	if err != nil {
		log.Printf("web: sso: auth url for %q: %v", slug, err)
		s.render(w, "login", s.loginData(r.Context(), "That identity provider could not be reached. Sign in with your password."))
		return
	}

	if !s.setSSOTxCookie(w, r, ssoTx{Slug: prov.Slug, State: state, Nonce: nonce, Verifier: verifier, Expires: s.now().Add(ssoTxTTL)}) {
		return
	}
	http.Redirect(w, r, authURL, http.StatusSeeOther)
}

// ssoCallback completes the flow. It verifies the transaction cookie and the echoed
// state, exchanges the code and verifies the id_token via the ssoFlow, then maps the
// verified username to an EXISTING local account and issues the session. Any failure
// returns to the login form with an honest, non-leaky message — SSO never creates an
// account, so an identity with no local match is refused, not provisioned.
func (s *server) ssoCallback(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	tx, ok := s.readSSOTxCookie(r)
	s.clearCookie(w, ssoTxCookie) // single-use: spend it whether or not it verifies
	if !ok || tx.Slug != slug {
		s.render(w, "login", s.loginData(r.Context(), "That sign-on attempt expired. Try again."))
		return
	}
	// The IdP reports a user-declined or error response in the query rather than a code.
	if e := r.URL.Query().Get("error"); e != "" {
		s.render(w, "login", s.loginData(r.Context(), "Single sign-on was cancelled or refused."))
		return
	}
	// state is the CSRF guard: the value the IdP echoes back must equal the one minted
	// into the signed cookie at the start of this transaction. Compared in constant
	// time, like the rest of the auth surface.
	if st := r.URL.Query().Get("state"); st == "" || !subtleConstantEqual(st, tx.State) {
		s.render(w, "login", s.loginData(r.Context(), "That sign-on attempt could not be verified. Try again."))
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		s.render(w, "login", s.loginData(r.Context(), "That sign-on attempt could not be verified. Try again."))
		return
	}

	prov, err := s.store.GetSSOProviderForAuth(r.Context(), slug)
	if err != nil {
		s.render(w, "login", s.loginData(r.Context(), "Single sign-on is not available. Sign in with your password."))
		return
	}
	cfg := ssoConfig{
		Slug: prov.Slug, Issuer: prov.Issuer, ClientID: prov.ClientID,
		ClientSecret: prov.ClientSecret.String, UsernameClaim: prov.UsernameClaim,
		RedirectURL: s.ssoRedirectURL(r, prov.Slug),
	}
	username, err := s.sso.Exchange(r.Context(), cfg, code, tx.Verifier, tx.Nonce)
	if err != nil {
		log.Printf("web: sso: exchange for %q: %v", slug, err)
		s.render(w, "login", s.loginData(r.Context(), "Single sign-on could not be completed. Sign in with your password."))
		return
	}

	acct, err := s.store.GetAccountByUsername(r.Context(), username)
	if err != nil {
		// A verified identity with no local account: refuse honestly. SSO authenticates
		// existing accounts; it does not create them (ADR-0112).
		log.Printf("web: sso: no local account for verified identity %q via %q", username, slug)
		s.render(w, "login", s.loginData(r.Context(), "No account here matches that identity. Ask an admin for an invite."))
		return
	}
	// SSO proves the IdP identity, but it must never DOWNGRADE a local second factor:
	// an account that enrolled TOTP still owes its code, exactly as the password path
	// requires. So a TOTP-enrolled account lands on the same TOTP step (a KindPending
	// cookie), and only an account without TOTP completes the login outright.
	if acct.TotpEnabled {
		if !s.setSignedCookie(w, r, pendingCookie, auth.KindPending, acct.ID, s.pendingTTL) {
			return
		}
		s.render(w, "totp", map[string]any{"Title": "Two-factor"})
		return
	}
	s.completeLogin(w, r, acct.ID)
}

// loginData is the login template's data with an optional error, so the SSO and
// password handlers can fall back to the sign-in form carrying a message. It re-lists
// the enabled providers so the buttons still render on the error re-paint.
func (s *server) loginData(ctx context.Context, errMsg string) map[string]any {
	data := map[string]any{"Title": "Sign in", "SSOProviders": s.enabledSSOProviders(ctx)}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	return data
}

// enabledSSOProviders lists the providers SignIn renders a button for. A read failure
// degrades to no buttons (password login still works) rather than failing the page.
func (s *server) enabledSSOProviders(ctx context.Context) []db.ListEnabledSSOProvidersRow {
	rows, err := s.store.ListEnabledSSOProviders(ctx)
	if err != nil {
		log.Printf("web: sso: list enabled providers: %v", err)
		return nil
	}
	return rows
}

// setSSOTxCookie writes the signed, short-lived login-transaction cookie. SameSite=Lax
// (like the session cookie) so it survives the top-level redirect back from the IdP.
func (s *server) setSSOTxCookie(w http.ResponseWriter, r *http.Request, tx ssoTx) bool {
	payload, err := json.Marshal(tx)
	if err != nil {
		s.serverError(w, "marshal sso transaction", err)
		return false
	}
	http.SetCookie(w, &http.Cookie{
		Name: ssoTxCookie, Value: auth.Sign(s.key, ssoTxDomain, payload), Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: s.secureCookies || r.TLS != nil, MaxAge: int(ssoTxTTL.Seconds()),
	})
	return true
}

// readSSOTxCookie verifies and decodes the transaction cookie, returning ok=false for
// a missing, forged, malformed, or expired cookie (all indistinguishable to the
// caller, exactly like the session path).
func (s *server) readSSOTxCookie(r *http.Request) (ssoTx, bool) {
	c, err := r.Cookie(ssoTxCookie)
	if err != nil {
		return ssoTx{}, false
	}
	payload, err := auth.Verify(s.key, ssoTxDomain, c.Value)
	if err != nil {
		return ssoTx{}, false
	}
	var tx ssoTx
	if err := json.Unmarshal(payload, &tx); err != nil {
		return ssoTx{}, false
	}
	if !s.now().Before(tx.Expires) {
		return ssoTx{}, false
	}
	return tx, true
}

// randToken returns a 256-bit URL-safe random token for the state and nonce values.
func randToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("web: sso: read random: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
