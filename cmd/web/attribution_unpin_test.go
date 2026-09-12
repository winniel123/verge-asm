package main

import (
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// §9 withdraws the refusal that pinned an account to the objects it declared, and §9.2
// rules what an authorless object renders (docs/spec/audit-act.md).

func declaringAccount(t *testing.T, f *fakeStore) db.Account {
	t.Helper()
	alice := seedAccount(t, f, "alice", roleAdmin, "hunter2hunter2alice")

	if _, err := f.CreateChannel(t.Context(), db.CreateChannelParams{
		Url: "https://ops.example/hook", RouteDrift: true, Enabled: true,
		CreatedBy: pgtype.Int8{Int64: alice.ID, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.InsertSSOProvider(t.Context(), db.InsertSSOProviderParams{
		Slug: "okta", Name: "Okta", Issuer: "https://idp.example", ClientID: "cid",
		Enabled: true, CreatedBy: pgtype.Int8{Int64: alice.ID, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	scope := netip.MustParsePrefix("198.51.100.0/24")
	if _, err := f.CreateAddressSeed(t.Context(), db.CreateAddressSeedParams{
		AddressCidr: &scope, CreatedBy: pgtype.Int8{Int64: alice.ID, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	return alice
}

func removeAccountNamed(t *testing.T, c *http.Client, base string, acct db.Account) {
	t.Helper()
	postForm(t, c, base+"/settings/accounts/remove", url.Values{
		"id": {itoa(acct.ID)}, "confirm_name": {acct.Username},
	}).Body.Close()
}

func TestRemovingADeclaringAccountKeepsItsObjects(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	alice := declaringAccount(t, f)

	removeAccountNamed(t, ac, base, alice)

	if _, ok := f.accounts[alice.ID]; ok {
		t.Fatalf("the declaring account survived the removal; §9 withdraws that refusal")
	}
	// An INNER JOIN would drop each object rather than orphan it (§9.1).
	channels, err := f.ListChannels(t.Context())
	if err != nil || len(channels) != 1 {
		t.Fatalf("ListChannels = %d rows, %v; want the declared channel to survive", len(channels), err)
	}
	providers, err := f.ListSSOProviders(t.Context())
	if err != nil || len(providers) != 1 {
		t.Fatalf("ListSSOProviders = %d rows, %v; want the declared provider to survive", len(providers), err)
	}
	seeds, err := f.ListSeeds(t.Context())
	if err != nil || len(seeds) != 1 {
		t.Fatalf("ListSeeds = %d rows, %v; want the declared scope to survive", len(seeds), err)
	}
}

func TestAuthorlessTableCellsRenderTheRemovedAccountTag(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	alice := declaringAccount(t, f)

	removeAccountNamed(t, ac, base, alice)

	const tag = `<span class="st-tag">removed account</span>`
	for _, tab := range []string{"channels", "sso"} {
		page := getBody(t, ac, base+"/settings?tab="+tab, http.StatusOK)
		if !strings.Contains(page, tag) {
			t.Errorf("the %s tab does not render %q for an authorless row; body: %s", tab, tag, page)
		}
		if strings.Contains(page, ">alice<") {
			t.Errorf("the %s tab still names the removed account; body: %s", tab, page)
		}
	}
}

func TestAuthorlessSeedRendersRemovedAccountAsProse(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	alice := declaringAccount(t, f)
	f.addReachability(t, "198.51.100.1:443/tcp", obsClock, `{"outcome":"reached","result":"open"}`)

	drill := getBody(t, ac, base+"/subjects/service?key=198.51.100.1%3A443%2Ftcp", http.StatusOK)
	if !strings.Contains(drill, "declared by alice") {
		t.Fatalf("the citation chain does not name the live author; body: %s", drill)
	}

	removeAccountNamed(t, ac, base, alice)

	drill = getBody(t, ac, base+"/subjects/service?key=198.51.100.1%3A443%2Ftcp", http.StatusOK)
	// A dropped hop would leave the chain unterminated (§9.1).
	if !strings.Contains(drill, "address scope 198.51.100.0/24") {
		t.Errorf("the Declared · Seed hop vanished with its author; body: %s", drill)
	}
	if !strings.Contains(drill, "declared by a removed account") {
		t.Errorf("the citation chain does not read %q; body: %s", "declared by a removed account", drill)
	}
}

// Each dial resolves its author by scanning ListAccounts in Go, and the column itself goes
// NULL under ON DELETE SET NULL. The fake models both, so the branch under test is the one
// production reaches.

func TestAuthorlessDialsRenderRemovedAccountAsProse(t *testing.T) {
	for _, d := range []struct {
		name, tab, action, live, gone string
		form                          url.Values
	}{
		{
			name: "api", tab: "api", action: "/settings/api",
			live: "Enabled by alice", gone: "Enabled by a removed account",
			form: url.Values{"enabled": {"true"}},
		},
		{
			name: "address cap", tab: "scope", action: "/settings/address-cap",
			live: "by <span style=\"font-family:var(--font-mono)\">alice</span>",
			gone: "by a removed account",
			form: url.Values{"address_cap": {"2048"}},
		},
		{
			name: "retention", tab: "delivery", action: "/settings/retention",
			live: "by <span style=\"font-family:var(--font-mono)\">alice</span>",
			gone: "by a removed account",
			form: url.Values{"transcript_currency_days": {"14"}},
		},
	} {
		t.Run(d.name, func(t *testing.T) {
			f := newFakeStore()
			base, ac := adminSession(t, f)
			alice := seedAccount(t, f, "alice", roleAdmin, "hunter2hunter2alice")

			alicesClient := login(t, base, "alice", "hunter2hunter2alice")
			postForm(t, alicesClient, base+d.action, d.form).Body.Close()

			page := getBody(t, ac, base+"/settings?tab="+d.tab, http.StatusOK)
			if !strings.Contains(page, d.live) {
				t.Fatalf("the %s dial does not name the live author; body: %s", d.name, page)
			}

			removeAccountNamed(t, ac, base, alice)

			page = getBody(t, ac, base+"/settings?tab="+d.tab, http.StatusOK)
			if !strings.Contains(page, d.gone) {
				t.Errorf("the %s dial does not read %q; body: %s", d.name, d.gone, page)
			}
			if strings.Contains(page, "<no value>") {
				t.Errorf("the %s dial rendered a missing map key; body: %s", d.name, page)
			}
		})
	}
}
