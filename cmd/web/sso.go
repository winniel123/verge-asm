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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"

	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
)

// Single sign-on — the OIDC authorization-code flow (#293, ADR-0112, ADR-0113). SSO is
// admitted as cryptographically-verified OIDC (the id_token's signature, issuer and a
// per-login nonce are checked) — never reverse-proxy header-trust, which stays refused.
// It authenticates an EXISTING local account by a stored binding of the verified,
// non-reassignable subject `(provider, sub)` — never a mutable username claim (ADR-0113,
// which closes the account-takeover surface a self-editable/recycled username opened).
// It never creates an account, so a broad IdP cannot silently mint one, and the
// session's role is still read from the local account row on every request.
//
// The sign-in flow is two hops, both unauthenticated by construction (a caller signing
// in has no session yet):
//
//   GET /login/sso/{slug}          → mint state/nonce/PKCE, redirect to the IdP.
//   GET /login/sso/{slug}/callback → verify state, exchange the code, verify the
//                                    id_token, match (provider, sub), issue the session.
//
// The binding itself is established by an already-authenticated user linking their own
// identity from their Profile (ssoLinkStart/ssoLinkCallback) — never trust-on-first-use,
// which would re-open the first-claimant window. That link runs the same OIDC round-trip
// inside the caller's session and records `(provider, sub) → their account`.
//
// The state/nonce/PKCE-verifier ride an HMAC-signed, short-lived cookie between the
// two hops (the transaction is carried in the signed cookie itself, not looked up
// server-side), so the callback can trust the state it echoes back was minted here —
// CSRF and replay are both closed. The cookie's Link flag keeps a self-link
// transaction from ever being replayed as a sign-in.

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
	Slug         string
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// ssoIdentity is the verified identity the OIDC exchange yields (ADR-0113): the stable,
// non-reassignable per-issuer subject that authentication keys on, plus a human label
// captured for display only. Sub is the matching key; Display never gates auth.
type ssoIdentity struct {
	Sub     string
	Display string
}

// ssoFlow is the OIDC seam. The real implementation (oidcFlow) uses go-oidc + oauth2
// over the network; tests inject a fake so a login flow asserts its state/nonce/cookie
// handling and identity binding without a live identity provider.
type ssoFlow interface {
	AuthCodeURL(ctx context.Context, cfg ssoConfig, state, nonce, pkceVerifier string) (string, error)
	// Exchange completes the callback: it exchanges the code, verifies the returned
	// id_token (signature, issuer, audience) and that its nonce matches, and returns the
	// verified identity — the stable `sub` and a display label. Login (match) and the
	// Profile self-link (bind) share this one extraction, so no crypto path is duplicated.
	Exchange(ctx context.Context, cfg ssoConfig, code, pkceVerifier, nonce string) (ssoIdentity, error)
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

func newOIDCFlow(client *http.Client) *oidcFlow {
	return &oidcFlow{httpClient: client, byIss: map[string]*oidc.Provider{}}
}

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

func (f *oidcFlow) Exchange(ctx context.Context, cfg ssoConfig, code, pkceVerifier, nonce string) (ssoIdentity, error) {
	ctx = oidc.ClientContext(ctx, f.httpClient)
	prov, err := f.provider(ctx, cfg.Issuer)
	if err != nil {
		return ssoIdentity{}, err
	}
	oc := f.oauth2Config(cfg, prov)
	tok, err := oc.Exchange(ctx, code, oauth2.VerifierOption(pkceVerifier))
	if err != nil {
		return ssoIdentity{}, fmt.Errorf("oidc code exchange: %w", err)
	}
	rawID, ok := tok.Extra("id_token").(string)
	if !ok || rawID == "" {
		return ssoIdentity{}, errors.New("oidc: token response carried no id_token")
	}
	idToken, err := prov.Verifier(&oidc.Config{ClientID: cfg.ClientID}).Verify(ctx, rawID)
	if err != nil {
		return ssoIdentity{}, fmt.Errorf("oidc id_token verify: %w", err)
	}
	if idToken.Nonce != nonce {
		return ssoIdentity{}, errors.New("oidc: id_token nonce mismatch")
	}
	// The subject is on the verified token itself, not the claims bag — the stable,
	// non-reassignable per-issuer identifier authentication binds to (ADR-0113). go-oidc
	// guarantees it is non-empty on a verified token, but guard anyway.
	if idToken.Subject == "" {
		return ssoIdentity{}, errors.New("oidc: id_token carried no sub")
	}
	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return ssoIdentity{}, fmt.Errorf("oidc id_token claims: %w", err)
	}
	return ssoIdentity{Sub: idToken.Subject, Display: ssoDisplayName(claims)}, nil
}

// ssoDisplayName picks a human label from the verified claims for display only — never
// an auth input. It prefers email, then preferred_username, then name; a provider that
// carries none leaves it empty (the Profile falls back to the subject).
func ssoDisplayName(claims map[string]any) string {
	for _, k := range []string{"email", "preferred_username", "name"} {
		if v, ok := claims[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
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
	// Link marks a Profile self-link transaction (bind) rather than a sign-in (match), so
	// a link's callback can never be replayed as a login or vice-versa: the callback
	// refuses a transaction whose Link flag does not match the route it arrives on.
	Link bool `json:"link,omitempty"`
}

// ssoRedirectURL is the absolute callback URL the IdP returns to, for the given callback
// path (the login callback, or the Profile self-link callback). It is taken from the
// configured external base URL (VERGE_EXTERNAL_URL) when set — the trusted origin the
// deployment is reached at — so the redirect_uri handed to the IdP never derives from
// the attacker-influenceable Host header. Where none is configured it falls back to the
// request host (https when the request arrived over TLS or a TLS-terminating proxy
// fronts it, the same signal the Secure cookie flag uses); the OIDC redirect_uri must
// still match the value registered at the IdP, which bounds that fallback.
func (s *server) ssoRedirectURL(r *http.Request, path string) string {
	if s.externalURL != "" {
		return strings.TrimRight(s.externalURL, "/") + path
	}
	scheme := "http"
	if r.TLS != nil || s.secureCookies {
		scheme = "https"
	}
	return scheme + "://" + r.Host + path
}

func ssoLoginCallbackPath(slug string) string { return "/login/sso/" + slug + "/callback" }
func ssoLinkCallbackPath(slug string) string  { return "/profile/sso/" + slug + "/link/callback" }

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
		s.render(w, r, "login", s.loginData(r.Context(), "Single sign-on is not available. Sign in with your password."))
		return
	}

	state, nonce := randToken(), randToken()
	verifier := oauth2.GenerateVerifier()
	cfg := ssoConfig{
		Slug: prov.Slug, Issuer: prov.Issuer, ClientID: prov.ClientID,
		ClientSecret: prov.ClientSecret.String,
		RedirectURL:  s.ssoRedirectURL(r, ssoLoginCallbackPath(prov.Slug)),
	}
	authURL, err := s.sso.AuthCodeURL(r.Context(), cfg, state, nonce, verifier)
	if err != nil {
		log.Printf("web: sso: auth url for %q: %v", logSafe(slug), err) // #nosec G706 (sanitized via logSafe)
		s.render(w, r, "login", s.loginData(r.Context(), "That identity provider could not be reached. Sign in with your password."))
		return
	}

	if !s.setSSOTxCookie(w, r, ssoTx{Slug: prov.Slug, State: state, Nonce: nonce, Verifier: verifier, Expires: s.now().Add(ssoTxTTL)}) {
		return
	}
	http.Redirect(w, r, authURL, http.StatusSeeOther)
}

// ssoCallback completes the sign-in flow. It verifies the transaction cookie and the
// echoed state, exchanges the code and verifies the id_token via the ssoFlow, then maps
// the verified (provider, sub) to the local account it is BOUND to and issues the
// session. Any failure returns to the login form with an honest, non-leaky message — SSO
// authenticates existing bindings only, so a verified identity with no binding is
// refused (directed to sign in with a password and link), never provisioned.
func (s *server) ssoCallback(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	tx, ok := s.readSSOTxCookie(r)
	s.clearCookie(w, ssoTxCookie) // single-use: spend it whether or not it verifies
	if !ok || tx.Slug != slug || tx.Link {
		s.render(w, r, "login", s.loginData(r.Context(), "That sign-on attempt expired. Try again."))
		return
	}
	// The IdP reports a user-declined or error response in the query rather than a code.
	if e := r.URL.Query().Get("error"); e != "" {
		s.render(w, r, "login", s.loginData(r.Context(), "Single sign-on was cancelled or refused."))
		return
	}
	// state is the CSRF guard: the value the IdP echoes back must equal the one minted
	// into the signed cookie at the start of this transaction. Compared in constant
	// time, like the rest of the auth surface.
	if st := r.URL.Query().Get("state"); st == "" || !subtleConstantEqual(st, tx.State) {
		s.render(w, r, "login", s.loginData(r.Context(), "That sign-on attempt could not be verified. Try again."))
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		s.render(w, r, "login", s.loginData(r.Context(), "That sign-on attempt could not be verified. Try again."))
		return
	}

	prov, err := s.store.GetSSOProviderForAuth(r.Context(), slug)
	if err != nil {
		s.render(w, r, "login", s.loginData(r.Context(), "Single sign-on is not available. Sign in with your password."))
		return
	}
	cfg := ssoConfig{
		Slug: prov.Slug, Issuer: prov.Issuer, ClientID: prov.ClientID,
		ClientSecret: prov.ClientSecret.String,
		RedirectURL:  s.ssoRedirectURL(r, ssoLoginCallbackPath(prov.Slug)),
	}
	ident, err := s.sso.Exchange(r.Context(), cfg, code, tx.Verifier, tx.Nonce)
	if err != nil {
		log.Printf("web: sso: exchange for %q: %v", logSafe(slug), err) // #nosec G706 (sanitized via logSafe)
		s.render(w, r, "login", s.loginData(r.Context(), "Single sign-on could not be completed. Sign in with your password."))
		return
	}

	acct, err := s.store.GetAccountBySSOIdentity(r.Context(), db.GetAccountBySSOIdentityParams{
		ProviderID: prov.ID, Sub: ident.Sub,
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// A verified identity with no binding: refuse honestly. The subject is not linked
		// to any local account, and SSO never provisions or falls back to a username
		// (ADR-0113). The user signs in with a password and links this identity on Profile.
		log.Printf("web: sso: no binding for verified identity via %q", logSafe(slug)) // #nosec G706 (sanitized via logSafe)
		s.render(w, r, "login", s.loginData(r.Context(), "That identity is not linked to an account here. Sign in with your password, then link it on your Profile."))
		return
	case err != nil:
		// A transient read failure is NOT an unlinked identity: don't misdirect a
		// legitimately-linked user to re-link during a DB blip. Fail generically.
		log.Printf("web: sso: look up binding via %q: %v", logSafe(slug), err) // #nosec G706 (sanitized via logSafe)
		s.render(w, r, "login", s.loginData(r.Context(), "Single sign-on could not be completed. Sign in with your password."))
		return
	}
	// SSO proves the IdP identity, but it must never DOWNGRADE a local second factor:
	// an account that enrolled TOTP still owes its code, exactly as the password path
	// requires. So a TOTP-enrolled account lands on the same TOTP step (a KindPending
	// cookie), and only an account without TOTP completes the login outright.
	if acct.TotpEnabled {
		if !s.setSignedCookie(w, r, pendingCookie, auth.KindPending, acct.ID, "", s.pendingTTL) {
			return
		}
		s.render(w, r, "totp", map[string]any{"Title": "Two-factor"})
		return
	}
	s.completeLogin(w, r, acct.ID)
}

// ssoLinkStart begins an authenticated self-link (ADR-0113): the signed-in caller runs
// the OIDC round-trip inside their session so its callback can bind the verified
// (provider, sub) to THEIR account. It mints the same state/nonce/PKCE transaction as a
// sign-in but flags it Link and points the redirect at the link callback. Failures
// return to the Profile with an honest error rather than a raw 500.
func (s *server) ssoLinkStart(w http.ResponseWriter, r *http.Request, _ db.Account) {
	slug := r.PathValue("slug")
	prov, err := s.store.GetSSOProviderForAuth(r.Context(), slug)
	if err != nil {
		http.Redirect(w, r, "/profile?linkerr=unavailable", http.StatusSeeOther)
		return
	}
	state, nonce := randToken(), randToken()
	verifier := oauth2.GenerateVerifier()
	cfg := ssoConfig{
		Slug: prov.Slug, Issuer: prov.Issuer, ClientID: prov.ClientID,
		ClientSecret: prov.ClientSecret.String,
		RedirectURL:  s.ssoRedirectURL(r, ssoLinkCallbackPath(prov.Slug)),
	}
	authURL, err := s.sso.AuthCodeURL(r.Context(), cfg, state, nonce, verifier)
	if err != nil {
		log.Printf("web: sso: link auth url for %q: %v", logSafe(slug), err) // #nosec G706 (sanitized via logSafe)
		http.Redirect(w, r, "/profile?linkerr=unavailable", http.StatusSeeOther)
		return
	}
	if !s.setSSOTxCookie(w, r, ssoTx{Slug: prov.Slug, State: state, Nonce: nonce, Verifier: verifier, Expires: s.now().Add(ssoTxTTL), Link: true}) {
		return
	}
	http.Redirect(w, r, authURL, http.StatusSeeOther)
}

func (s *server) ssoLinkCallback(w http.ResponseWriter, r *http.Request, acct db.Account) {
	slug := r.PathValue("slug")
	tx, ok := s.readSSOTxCookie(r)
	s.clearCookie(w, ssoTxCookie) // single-use
	if !ok || tx.Slug != slug || !tx.Link {
		http.Redirect(w, r, "/profile?linkerr=expired", http.StatusSeeOther)
		return
	}
	if e := r.URL.Query().Get("error"); e != "" {
		http.Redirect(w, r, "/profile?linkerr=cancelled", http.StatusSeeOther)
		return
	}
	if st := r.URL.Query().Get("state"); st == "" || !subtleConstantEqual(st, tx.State) {
		http.Redirect(w, r, "/profile?linkerr=failed", http.StatusSeeOther)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, "/profile?linkerr=failed", http.StatusSeeOther)
		return
	}

	prov, err := s.store.GetSSOProviderForAuth(r.Context(), slug)
	if err != nil {
		http.Redirect(w, r, "/profile?linkerr=unavailable", http.StatusSeeOther)
		return
	}
	cfg := ssoConfig{
		Slug: prov.Slug, Issuer: prov.Issuer, ClientID: prov.ClientID,
		ClientSecret: prov.ClientSecret.String,
		RedirectURL:  s.ssoRedirectURL(r, ssoLinkCallbackPath(prov.Slug)),
	}
	ident, err := s.sso.Exchange(r.Context(), cfg, code, tx.Verifier, tx.Nonce)
	if err != nil {
		log.Printf("web: sso: link exchange for %q: %v", logSafe(slug), err) // #nosec G706 (sanitized via logSafe)
		http.Redirect(w, r, "/profile?linkerr=failed", http.StatusSeeOther)
		return
	}

	// Resolve any existing binding first so a re-link of one's own identity no-ops and a
	// collision with another account is a clean refusal — never a raw unique violation.
	existing, err := s.store.GetSSOIdentityBySub(r.Context(), db.GetSSOIdentityBySubParams{
		ProviderID: prov.ID, Sub: ident.Sub,
	})
	switch {
	case err == nil && existing.AccountID == acct.ID:
		http.Redirect(w, r, "/profile?linked=exists", http.StatusSeeOther)
		return
	case err == nil:
		log.Printf("web: sso: link refused: identity via %q already bound to another account", logSafe(slug)) // #nosec G706 (sanitized via logSafe)
		http.Redirect(w, r, "/profile?linkerr=elsewhere", http.StatusSeeOther)
		return
	case !errors.Is(err, pgx.ErrNoRows):
		s.serverError(w, "look up sso identity", err)
		return
	}

	if err := s.store.InsertSSOIdentity(r.Context(), db.InsertSSOIdentityParams{
		ProviderID: prov.ID, AccountID: acct.ID, Sub: ident.Sub, DisplayName: ident.Display,
	}); err != nil {
		// The exact (provider, sub) was just confirmed free, so a UNIQUE violation here is
		// the (provider, account) constraint: this account already holds an identity for
		// this provider (the model allows one). Report that honestly rather than 500ing.
		if isUniqueViolation(err) {
			http.Redirect(w, r, "/profile?linkerr=provider", http.StatusSeeOther)
			return
		}
		s.serverError(w, "insert sso identity", err)
		return
	}
	log.Printf("web: sso: account %d linked an identity via %q", acct.ID, logSafe(slug)) // #nosec G706 (sanitized via logSafe)
	http.Redirect(w, r, "/profile?linked=1", http.StatusSeeOther)
}

// ssoUnlink removes one of the caller's OWN identity bindings (Profile). It is scoped to
// the account, so a caller can only ever unlink their own; a stale or foreign id no-ops.
func (s *server) ssoUnlink(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}
	rows, err := s.store.DeleteSSOIdentityForAccount(r.Context(), db.DeleteSSOIdentityForAccountParams{
		ID: id, AccountID: acct.ID,
	})
	if err != nil {
		s.serverError(w, "unlink sso identity", err)
		return
	}
	if rows > 0 {
		log.Printf("web: sso: account %d unlinked identity %d", acct.ID, id)
		http.Redirect(w, r, "/profile?unlinked=1", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

// loginData is the login template's data with an optional error, so the SSO and
// password handlers can fall back to the sign-in form carrying a message. It re-lists
// the enabled providers so the buttons still render on the error re-paint.
func (s *server) loginData(ctx context.Context, errMsg string) map[string]any {
	data := map[string]any{"Title": "Sign in", "SSOProviders": s.loginProviders(ctx, false)}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	// The frozen login.tmpl's authfoot reads the build version; stamp it so an
	// SSO/password error re-paint carries it too.
	return s.signinData(data)
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
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- Secure conditional (r.TLS != nil || s.secureCookies); HttpOnly + SameSite=Lax always set.
		Name: ssoTxCookie, Value: auth.Sign(s.key, ssoTxDomain, payload), Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: s.secureCookies || r.TLS != nil, MaxAge: int(ssoTxTTL.Seconds()),
	})
	return true
}

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

func randToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("web: sso: read random: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
