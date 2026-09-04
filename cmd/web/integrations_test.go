package main

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func (f *fakeStore) ListIntegrationStates(context.Context) ([]db.IntegrationState, error) {
	rows := make([]db.IntegrationState, 0, len(f.integrationStates))
	for _, st := range f.integrationStates {
		rows = append(rows, st)
	}
	return rows, nil
}

func (f *fakeStore) UpsertIntegrationState(_ context.Context, arg db.UpsertIntegrationStateParams) (db.IntegrationState, error) {
	// The real ON CONFLICT omits channel_id, so a re-install keeps the bound Channel.
	st := db.IntegrationState{Slug: arg.Slug, State: arg.State}
	if prev, ok := f.integrationStates[arg.Slug]; ok {
		st.ChannelID = prev.ChannelID
	}
	f.integrationStates[arg.Slug] = st
	return st, nil
}

func (f *fakeStore) DeleteIntegrationState(_ context.Context, slug string) error {
	delete(f.integrationStates, slug)
	return nil
}

func (f *fakeStore) GetIntegrationChannel(_ context.Context, slug string) (pgtype.Int8, error) {
	st, ok := f.integrationStates[slug]
	if !ok {
		return pgtype.Int8{}, pgx.ErrNoRows
	}
	return st.ChannelID, nil
}

func (f *fakeStore) SetIntegrationChannel(_ context.Context, arg db.SetIntegrationChannelParams) error {
	// The real UPDATE is WHERE slug = $1, so binding an integration with no row touches nothing.
	st, ok := f.integrationStates[arg.Slug]
	if !ok {
		return nil
	}
	st.ChannelID = arg.ChannelID
	f.integrationStates[arg.Slug] = st
	return nil
}

func (f *fakeStore) GetChannelForDelivery(_ context.Context, id int64) (db.GetChannelForDeliveryRow, error) {
	for _, c := range f.channels {
		if c.id == id {
			return db.GetChannelForDeliveryRow{Url: c.url, Secret: c.secret}, nil
		}
	}
	return db.GetChannelForDeliveryRow{}, pgx.ErrNoRows
}

func skipIfIntegrationsHidden(t *testing.T) {
	t.Helper()
	if !integrationsEnabled {
		t.Skip("integrations surface hidden (#388)")
	}
}

func integrationsBody(t *testing.T, c *http.Client, base, extra string) string {
	t.Helper()
	resp, err := c.Get(base + "/settings?tab=integrations" + extra)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /settings?tab=integrations%s status = %d, want 200", extra, resp.StatusCode)
	}
	return body(t, resp)
}

func TestIntegrationsTabHidden(t *testing.T) {
	if integrationsEnabled {
		t.Skip("integrations surface is live; the tab is expected to render")
	}
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := settingsBody(t, ac, base)
	if strings.Contains(page, "tab=integrations") {
		t.Errorf("settings tab bar still links to the hidden Integrations tab; body: %s", page)
	}
	if strings.Contains(page, ">Integrations<") {
		t.Errorf("an Integrations tab still renders while the surface is hidden; body: %s", page)
	}
}

func TestIntegrationsDirectNavRedirects(t *testing.T) {
	if integrationsEnabled {
		t.Skip("integrations surface is live; the tab is expected to render")
	}
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp, err := ac.Get(base + "/settings?tab=integrations")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/settings?tab=scans" {
		t.Fatalf("direct nav to hidden tab: status=%d loc=%q, want 303 -> /settings?tab=scans",
			resp.StatusCode, resp.Header.Get("Location"))
	}
	page := body(t, resp)
	for _, absent := range []string{"Channels need no integration", "Install Slack", "Install PagerDuty"} {
		if strings.Contains(page, absent) {
			t.Errorf("the placeholder catalog rendered on the hidden tab; found %q; body: %s", absent, page)
		}
	}
}

func TestIntegrationsMutationRoutesUnregistered(t *testing.T) {
	if integrationsEnabled {
		t.Skip("integrations surface is live; the mutation routes are expected to be registered")
	}
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	for _, path := range []string{"/settings/integrations/install", "/settings/integrations/disconnect"} {
		resp := postForm(t, ac, base+path, url.Values{"slug": {"slack"}})
		got := resp.StatusCode
		resp.Body.Close()
		if got != http.StatusMethodNotAllowed && got != http.StatusNotFound {
			t.Errorf("POST %s while hidden: status=%d, want 405/404 (no write route registered)", path, got)
		}
	}
	if len(f.integrationStates) != 0 {
		t.Fatalf("a hidden-surface route wrote to integration_state; got %d rows", len(f.integrationStates))
	}
}

func TestIntegrationsTileGridRenders(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := integrationsBody(t, ac, base, "")

	for _, c := range integrationCatalog {
		if !strings.Contains(page, c.Name) {
			t.Errorf("integration %q missing from the tile grid", c.Name)
		}
	}
	for _, cat := range integrationCats {
		if !strings.Contains(page, ">"+cat+"<") {
			t.Errorf("category segment %q missing", cat)
		}
	}
	if !strings.Contains(page, ">available<") {
		t.Errorf("available install state not rendered; body: %s", page)
	}
	if strings.Contains(page, ">installed<") {
		t.Errorf("an integration reads installed with nothing installed (fabricated state); body: %s", page)
	}
	if !strings.Contains(page, "need no integration") || !strings.Contains(page, "Channels are built in") {
		t.Errorf("the channels-vs-integration callout is missing; body: %s", page)
	}
}

func TestIntegrationsConsentGatingAndInstall(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drawer := integrationsBody(t, ac, base, "&view=pagerduty")
	for _, want := range []string{
		"This integration can", "Read signals", "Write annotations", "writes",
		"Install PagerDuty",
	} {
		if !strings.Contains(drawer, want) {
			t.Errorf("consent drawer missing %q; body: %s", want, drawer)
		}
	}

	resp := postForm(t, ac, base+"/settings/integrations/install", url.Values{"slug": {"pagerduty"}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/settings?tab=integrations" {
		t.Fatalf("install: status=%d loc=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	if st, ok := f.integrationStates["pagerduty"]; !ok || st.State != integrationInstalled {
		t.Fatalf("install state not persisted: %+v", f.integrationStates["pagerduty"])
	}

	page := integrationsBody(t, ac, base, "")
	if !strings.Contains(page, ">installed<") {
		t.Errorf("installed state not rendered after install; body: %s", page)
	}

	resp = postForm(t, ac, base+"/settings/integrations/install", url.Values{"slug": {"no-such-integration"}})
	got := resp.StatusCode
	resp.Body.Close()
	if got != http.StatusBadRequest {
		t.Errorf("install unknown slug: status=%d, want 400", got)
	}
}

func TestIntegrationsDrawerRemoveAndTest(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	postForm(t, ac, base+"/settings/integrations/install", url.Values{"slug": {"slack"}}).Body.Close()

	drawer := integrationsBody(t, ac, base, "&view=slack")
	for _, want := range []string{
		`action="/settings/integrations/remove"`,
		`name="id" value="slack"`, "Remove", "Send test", "Connect a channel to test",
	} {
		if !strings.Contains(drawer, want) {
			t.Errorf("installed drawer missing %q; body: %s", want, drawer)
		}
	}
	if strings.Contains(drawer, `action="/settings/integrations/test"`) {
		t.Errorf("an unbound integration offered an active Send-test form; body: %s", drawer)
	}

	avail := integrationsBody(t, ac, base, "&view=jira")
	if strings.Contains(avail, `action="/settings/integrations/remove"`) {
		t.Errorf("an available integration's drawer offered Remove; body: %s", avail)
	}
	if !strings.Contains(avail, "Install Jira") {
		t.Errorf("an available integration's drawer has no Install action; body: %s", avail)
	}

	resp := postForm(t, ac, base+"/settings/integrations/remove", url.Values{"id": {"slack"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("remove: status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	if _, ok := f.integrationStates["slack"]; ok {
		t.Fatalf("remove did not return the integration to available: %+v", f.integrationStates)
	}
}

func TestIntegrationsNeedsConfigRenders(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.integrationStates["jira"] = db.IntegrationState{Slug: "jira", State: integrationNeedsConfig}
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := integrationsBody(t, ac, base, "")
	if !strings.Contains(page, "needs attention") {
		t.Errorf("needs-config (attention) state not rendered on the grid; body: %s", page)
	}
	drawer := integrationsBody(t, ac, base, "&view=jira")
	if !strings.Contains(drawer, "needs attention") {
		t.Errorf("needs-config drawer missing its attention state; body: %s", drawer)
	}
}

func TestIntegrationsFilterAndSearch(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	siem := integrationsBody(t, ac, base, "&cat=SIEM")
	if !strings.Contains(siem, "Splunk") || strings.Contains(siem, ">Slack<") {
		t.Errorf("SIEM filter did not narrow the grid; body: %s", siem)
	}
	search := integrationsBody(t, ac, base, "&q=jira")
	if !strings.Contains(search, "Jira") || strings.Contains(search, "Splunk") {
		t.Errorf("search did not narrow the grid; body: %s", search)
	}
}

func TestIntegrationsAdminGatedAndDistinctFromSources(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	vc := login(t, base, "viewer", "hunter2hunter2")
	for _, path := range []string{"/settings/integrations/install", "/settings/integrations/disconnect"} {
		resp := postForm(t, vc, base+path, url.Values{"slug": {"slack"}})
		got := resp.StatusCode
		resp.Body.Close()
		if got != http.StatusForbidden {
			t.Errorf("viewer POST %s: status=%d, want 403", path, got)
		}
	}
	if len(f.integrationStates) != 0 {
		t.Fatalf("a viewer mutated integration state; got %d", len(f.integrationStates))
	}

	anon := newClient(t)
	resp := postForm(t, anon, base+"/settings/integrations/install", url.Values{"slug": {"slack"}})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon install: status=%d loc=%q", resp.StatusCode, resp.Header.Get("Location"))
	}

	ac := login(t, base, "admin", "hunter2hunter2")
	tabBar := settingsBody(t, ac, base)
	for _, tab := range []string{"tab=sources", "tab=integrations"} {
		if !strings.Contains(tabBar, tab) {
			t.Errorf("settings tab bar missing %q", tab)
		}
	}
	integ := integrationsBody(t, ac, base, "")
	if strings.Contains(integ, "RIPEstat") || strings.Contains(integ, ">proposer<") {
		t.Errorf("the Integrations tab bled the source catalogue in; body: %s", integ)
	}
	src := settingsTabBody(t, ac, base, "sources")
	if !strings.Contains(src, "crt.sh") || !strings.Contains(src, ">proposer<") {
		t.Errorf("the Sources tab lost the discovery catalogue; body: %s", src)
	}
	if strings.Contains(src, "Install Slack") {
		t.Errorf("the Sources tab bled the integration tiles in; body: %s", src)
	}
}
