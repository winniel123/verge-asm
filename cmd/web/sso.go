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

const (
	ssoTxCookie = "verge_sso_tx"
	ssoTxTTL    = 10 * time.Minute
	ssoTxDomain = "sso-tx"
)

type ssoConfig struct {
	Slug         string
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// Auth keys on the non-reassignable sub, because a username is editable and recycled (ADR-0113).

type ssoIdentity struct {
	Sub     string
	Display string
}

type ssoFlow interface {
	AuthCodeURL(ctx context.Context, cfg ssoConfig, state, nonce, pkceVerifier string) (string, error)
	Exchange(ctx context.Context, cfg ssoConfig, code, pkceVerifier, nonce string) (ssoIdentity, error)
}

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
	if idToken.Subject == "" {
		return ssoIdentity{}, errors.New("oidc: id_token carried no sub")
	}
	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return ssoIdentity{}, fmt.Errorf("oidc id_token claims: %w", err)
	}
	return ssoIdentity{Sub: idToken.Subject, Display: ssoDisplayName(claims)}, nil
}

func ssoDisplayName(claims map[string]any) string {
	for _, k := range []string{"email", "preferred_username", "name"} {
		if v, ok := claims[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

type ssoTx struct {
	Slug     string    `json:"slug"`
	State    string    `json:"state"`
	Nonce    string    `json:"nonce"`
	Verifier string    `json:"pkce"`
	Expires  time.Time `json:"exp"`
	Link     bool      `json:"link,omitempty"`
}

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

func (s *server) ssoCallback(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	tx, ok := s.readSSOTxCookie(r)
	s.clearCookie(w, ssoTxCookie)
	// A link transaction replayed here would sign the caller in, so the flag must match the route.
	if !ok || tx.Slug != slug || tx.Link {
		s.render(w, r, "login", s.loginData(r.Context(), "That sign-on attempt expired. Try again."))
		return
	}
	if e := r.URL.Query().Get("error"); e != "" {
		s.render(w, r, "login", s.loginData(r.Context(), "Single sign-on was cancelled or refused."))
		return
	}
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
		// SSO never provisions, so enabling a broad IdP cannot silently mint an account (ADR-0112).
		log.Printf("web: sso: no binding for verified identity via %q", logSafe(slug)) // #nosec G706 (sanitized via logSafe)
		s.render(w, r, "login", s.loginData(r.Context(), "That identity is not linked to an account here. Sign in with your password, then link it on your Profile."))
		return
	case err != nil:
		log.Printf("web: sso: look up binding via %q: %v", logSafe(slug), err) // #nosec G706 (sanitized via logSafe)
		s.render(w, r, "login", s.loginData(r.Context(), "Single sign-on could not be completed. Sign in with your password."))
		return
	}
	// SSO adds a route and replaces no factor, so an enrolled account still owes its code (ADR-0112).
	if acct.TotpEnabled {
		if !s.setSignedCookie(w, r, pendingCookie, auth.KindPending, acct.ID, "", s.pendingTTL) {
			return
		}
		s.render(w, r, "totp", map[string]any{"Title": "Two-factor"})
		return
	}
	s.completeLogin(w, r, acct.ID)
}

func (s *server) ssoLinkStart(w http.ResponseWriter, r *http.Request, _ db.Account) {
	// Only a signed-in self-link binds; trust-on-first-use is a first-claimant race (ADR-0113).
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
	s.clearCookie(w, ssoTxCookie)
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
		// The sub was just confirmed free, so this is the one-link-per-provider bound (ADR-0113).
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

func (s *server) loginData(ctx context.Context, errMsg string) map[string]any {
	data := map[string]any{"Title": "Sign in", "SSOProviders": s.loginProviders(ctx, false)}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	return s.signinData(data)
}

func (s *server) enabledSSOProviders(ctx context.Context) []db.ListEnabledSSOProvidersRow {
	rows, err := s.store.ListEnabledSSOProviders(ctx)
	if err != nil {
		log.Printf("web: sso: list enabled providers: %v", err)
		return nil
	}
	return rows
}

func (s *server) setSSOTxCookie(w http.ResponseWriter, r *http.Request, tx ssoTx) bool {
	payload, err := json.Marshal(tx)
	if err != nil {
		s.serverError(w, "marshal sso transaction", err)
		return false
	}
	// Strict would drop this cookie on the top-level redirect back from the IdP, so Lax.
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
