package main

import (
	"bytes"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/secretseal"
)

func mustDeriveKey(label string) []byte {
	key, err := secretseal.DeriveKey(testTranscriptKey, label)
	if err != nil {
		panic(err)
	}
	return key
}

func mustSeal(label, raw string) string {
	sealed, err := secretseal.Seal(mustDeriveKey(label), raw)
	if err != nil {
		panic(err)
	}
	return sealed
}

func openTestChannelSecret(t *testing.T, stored pgtype.Text) string {
	t.Helper()
	plain, err := secretseal.OpenText(mustDeriveKey(secretseal.LabelChannelSecret), stored)
	if err != nil {
		t.Fatalf("stored channel secret did not open: %v", err)
	}
	return string(plain)
}

func openTestSSOSecret(t *testing.T, stored string) string {
	t.Helper()
	plain, err := secretseal.Open(mustDeriveKey(secretseal.LabelSSOClientSecret), stored)
	if err != nil {
		t.Fatalf("stored sso client secret did not open: %v", err)
	}
	return plain
}

func TestChannelSecretSealedAtRest(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	sender := &fakeChannelSender{status: http.StatusOK}
	base := startWithChannelSender(t, f, sender)
	ac := login(t, base, "admin", "hunter2hunter2")

	const secret = "sign-me-please"
	postForm(t, ac, base+"/settings/channels", url.Values{
		"url": {"https://ops.example/hook"}, "secret": {secret}, "drift": {"on"},
	}).Body.Close()
	if len(f.channels) != 1 {
		t.Fatalf("channels = %d, want 1", len(f.channels))
	}
	stored := f.channels[0].secret
	if !stored.Valid || stored.String == "" {
		t.Fatal("no channel secret was stored")
	}
	if stored.String == secret || strings.Contains(stored.String, secret) {
		t.Fatalf("channel secret stored in cleartext: stored=%q", stored.String)
	}
	if got := openTestChannelSecret(t, stored); got != secret {
		t.Fatalf("stored ciphertext opened to %q, want %q", got, secret)
	}

	idStr := "1"
	postForm(t, ac, base+"/settings/channels/update", url.Values{
		"id": {idStr}, "url": {"https://ops.example/hook"}, "drift": {"on"}, "secret": {"rotated"},
	}).Body.Close()
	if s := f.channels[0].secret.String; s == "rotated" || strings.Contains(s, "rotated") {
		t.Fatalf("rotated channel secret stored in cleartext: %q", s)
	}
	if got := openTestChannelSecret(t, f.channels[0].secret); got != "rotated" {
		t.Fatalf("rotated ciphertext opened to %q, want rotated", got)
	}

	postForm(t, ac, base+"/settings/channels/test", url.Values{"id": {idStr}}).Body.Close()
	if sender.calls != 1 || string(sender.lastSecret) != "rotated" {
		t.Fatalf("test send did not sign with the opened secret: calls=%d secret=%q", sender.calls, sender.lastSecret)
	}
}

func TestChannelLegacyCleartextSecretFailsLoud(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.channels = append(f.channels, fakeChannel{
		id: 5, url: "https://ops.example/hook", drift: true, enabled: true, createdBy: pgtype.Int8{Int64: 1, Valid: true},
		secret: pgtype.Text{String: "legacy-cleartext-secret", Valid: true},
	})
	f.chanNextID = 6
	sender := &fakeChannelSender{status: http.StatusOK}
	base := startWithChannelSender(t, f, sender)
	ac := login(t, base, "admin", "hunter2hunter2")

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	resp := postForm(t, ac, base+"/settings/channels/test", url.Values{"id": {"5"}})
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("legacy cleartext channel secret was tolerated: status=%d, want 500", resp.StatusCode)
	}
	if sender.calls != 0 {
		t.Fatalf("a delivery was signed with an unopened secret; calls=%d", sender.calls)
	}
	if strings.Contains(buf.String(), "legacy-cleartext-secret") {
		t.Fatalf("cleartext secret leaked into the log: %s", buf.String())
	}
}

func TestSSOClientSecretSealedAtRest(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	flow := &fakeSSOFlow{sub: "u1", display: "Admin"}
	base := startWithSSO(t, f, flow)
	ac := login(t, base, "admin", "hunter2hunter2")

	const secret = "super-secret-value"
	postForm(t, ac, base+"/settings/sso", url.Values{
		"slug": {"okta"}, "name": {"Okta"}, "issuer": {"https://idp.example"},
		"client_id": {"cid"}, "client_secret": {secret},
	}).Body.Close()
	if len(f.ssoProviders) != 1 {
		t.Fatalf("providers = %d, want 1", len(f.ssoProviders))
	}
	stored := f.ssoProviders[0].secret
	if !f.ssoProviders[0].hasSecret || stored == "" {
		t.Fatal("no client secret was stored")
	}
	if stored == secret || strings.Contains(stored, secret) {
		t.Fatalf("client secret stored in cleartext: stored=%q", stored)
	}
	if got := openTestSSOSecret(t, stored); got != secret {
		t.Fatalf("stored ciphertext opened to %q, want %q", got, secret)
	}

	c := newClient(t)
	resp, err := c.Get(base + "/login/sso/okta")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if flow.lastCfg.ClientSecret != secret {
		t.Fatalf("the OIDC flow received %q, want the original client secret", flow.lastCfg.ClientSecret)
	}
}

func TestSSOLegacyCleartextSecretFailsLoud(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.ssoNextID = 1
	f.ssoProviders = append(f.ssoProviders, fakeSSOProvider{
		id: 1, slug: "okta", name: "Okta", issuer: "https://idp.example", clientID: "cid",
		secret: "legacy-cleartext-secret", hasSecret: true, enabled: true, createdBy: pgtype.Int8{Int64: 1, Valid: true}, createdAt: obsClock,
	})
	flow := &fakeSSOFlow{sub: "u1", display: "Admin"}
	base := startWithSSO(t, f, flow)

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	c := newClient(t)
	resp, err := c.Get(base + "/login/sso/okta")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("legacy cleartext client secret was tolerated: status=%d, want 500", resp.StatusCode)
	}
	if flow.lastCfg.ClientSecret != "" {
		t.Fatalf("the OIDC flow received an unopened secret %q", flow.lastCfg.ClientSecret)
	}
	if strings.Contains(buf.String(), "legacy-cleartext-secret") {
		t.Fatalf("cleartext secret leaked into the log: %s", buf.String())
	}
}
