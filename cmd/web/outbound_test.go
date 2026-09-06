package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/proposer"
)

func TestOutboundClientRefusesRedirects(t *testing.T) {
	c := newOutboundClient(30 * time.Second)
	if c.CheckRedirect == nil {
		t.Fatal("no CheckRedirect set — redirects would be followed")
	}
	if err := c.CheckRedirect(nil, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Errorf("CheckRedirect = %v, want ErrUseLastResponse (do not follow)", err)
	}
}

func TestNewServerBuildsAnOIDCClientThatRefusesRedirects(t *testing.T) {
	srv := newServer(newFakeStore(), testKey, "", fixedClock())
	flow, ok := srv.sso.(*oidcFlow)
	if !ok {
		t.Fatalf("sso is %T, want *oidcFlow", srv.sso)
	}
	if flow.httpClient.CheckRedirect == nil {
		t.Fatal("the OIDC back-channel client has no CheckRedirect — the client secret would be replayed")
	}
	if err := flow.httpClient.CheckRedirect(nil, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Errorf("CheckRedirect = %v, want ErrUseLastResponse (do not follow)", err)
	}
}

func TestOIDCExchangeDoesNotFollowARedirect(t *testing.T) {
	var hopReached bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hopReached = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"a","token_type":"bearer","id_token":"pwned"}`))
	}))
	defer target.Close()

	var issuer string
	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			http.Redirect(w, r, target.URL+"/latest/meta-data/", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 issuer,
			"authorization_endpoint": issuer + "/authorize",
			"token_endpoint":         issuer + "/token",
			"jwks_uri":               issuer + "/jwks",
		})
	}))
	defer idp.Close()
	issuer = idp.URL

	f := newOIDCFlow(newOutboundClient(5 * time.Second))
	cfg := ssoConfig{
		Slug: "idp", Issuer: issuer, ClientID: "cid", ClientSecret: "shhh",
		RedirectURL: "https://verge.example/sso/idp/callback",
	}

	_, err := f.Exchange(context.Background(), cfg, "the-code", "the-verifier", "the-nonce")
	if hopReached {
		t.Fatal("the token exchange followed the redirect: the client secret left the issuer")
	}
	if err == nil {
		t.Fatal("Exchange succeeded on an unfollowed 302, so x/oauth2 read the 3xx as a token")
	}
	if !strings.Contains(err.Error(), "oidc code exchange") {
		t.Errorf("err = %v, want the code-exchange wrap", err)
	}
}

func TestProposerClientDoesNotFollowARedirect(t *testing.T) {
	var hopReached bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hopReached = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"entitySearchResults":[{"handle":"PWNED"}]}`))
	}))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/latest/meta-data/", http.StatusFound)
	}))
	defer redirector.Close()

	cands, err := proposer.NewARIN(newOutboundClient(5*time.Second), redirector.URL).
		Propose(context.Background(), "acme")
	if hopReached {
		t.Fatal("the RDAP lookup followed the redirect and reached the next hop: blind SSRF")
	}
	if err == nil {
		t.Fatalf("Propose returned %v on a 302, want the unfollowed 3xx refused as a non-200", cands)
	}
	if !strings.Contains(err.Error(), "302") {
		t.Errorf("err = %v, want the 302 surfaced as an ordinary non-200", err)
	}
}

func TestCheckHealthDoesNotFollowARedirect(t *testing.T) {
	var hopReached bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hopReached = true
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	probed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/latest/meta-data/", http.StatusFound)
	}))
	defer probed.Close()

	err := checkHealth(probed.Listener.Addr().String())
	if hopReached {
		t.Fatal("the health probe followed the redirect and reached the next hop")
	}
	if err == nil {
		t.Fatal("checkHealth passed on a 302, so a redirected listener reads as healthy")
	}
	if !strings.Contains(err.Error(), "healthz returned 302") {
		t.Errorf("err = %v, want the 302 surfaced as an ordinary non-200", err)
	}
}
