package main

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
)

// --- fakeStore integration-state methods -----------------------------------

func (f *fakeStore) ListIntegrationStates(context.Context) ([]db.IntegrationState, error) {
	rows := make([]db.IntegrationState, 0, len(f.integrationStates))
	for _, st := range f.integrationStates {
		rows = append(rows, st)
	}
	return rows, nil
}

func (f *fakeStore) UpsertIntegrationState(_ context.Context, arg db.UpsertIntegrationStateParams) (db.IntegrationState, error) {
	st := db.IntegrationState{Slug: arg.Slug, State: arg.State}
	f.integrationStates[arg.Slug] = st
	return st, nil
}

func (f *fakeStore) DeleteIntegrationState(_ context.Context, slug string) error {
	delete(f.integrationStates, slug)
	return nil
}

// --- helpers ----------------------------------------------------------------

// integrationsBody fetches the Integrations sub-tab with an optional extra query
// fragment (e.g. "&open=slack"). Settings is admin-gated, so pass an admin client.
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

// --- tests ------------------------------------------------------------------

// The tile grid renders every catalogued integration with its install state, the
// category segments, and the channels-vs-integration callout — with no fabricated
// install state: a fresh install shows every tile available (not installed).
func TestIntegrationsTileGridRenders(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := integrationsBody(t, ac, base, "")

	// Every catalogued integration appears, by name.
	for _, c := range integrationCatalog {
		if !strings.Contains(page, c.Name) {
			t.Errorf("integration %q missing from the tile grid", c.Name)
		}
	}
	// The category segmented control renders every category.
	for _, cat := range integrationCats {
		if !strings.Contains(page, ">"+cat+"<") {
			t.Errorf("category segment %q missing", cat)
		}
	}
	// No fabricated install state: nothing installed, so no tile reads installed and
	// the available state is shown.
	if !strings.Contains(page, ">available<") {
		t.Errorf("available install state not rendered; body: %s", page)
	}
	if strings.Contains(page, ">installed<") {
		t.Errorf("an integration reads installed with nothing installed (fabricated state); body: %s", page)
	}
	// The channels-vs-integration distinction is stated, and never calls an
	// integration a webhook (channel != webhook, channel != integration).
	if !strings.Contains(page, "Channels need no integration") {
		t.Errorf("the channels-vs-integration callout is missing; body: %s", page)
	}
	if strings.Contains(page, "webhook") || strings.Contains(page, "Webhook") {
		t.Errorf("an integration/channel was called a webhook; body: %s", page)
	}
}

// Opening a tile shows its consent grants; installing from that drawer is the
// consent, and it persists real install state. Consent is gated: the grants are
// shown with the install action, and grants are all-or-nothing.
func TestIntegrationsConsentGatingAndInstall(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// The drawer for an available integration shows its consent list and the
	// consent-gating copy alongside the install action.
	drawer := integrationsBody(t, ac, base, "&open=pagerduty")
	for _, want := range []string{
		"This integration can", "Read signals", "Write annotations", "writes",
		"Installing grants the access above", "Install PagerDuty",
	} {
		if !strings.Contains(drawer, want) {
			t.Errorf("consent drawer missing %q; body: %s", want, drawer)
		}
	}

	// Install persists real state.
	resp := postForm(t, ac, base+"/settings/integrations/install", url.Values{"slug": {"pagerduty"}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/settings?tab=integrations" {
		t.Fatalf("install: status=%d loc=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	if st, ok := f.integrationStates["pagerduty"]; !ok || st.State != integrationInstalled {
		t.Fatalf("install state not persisted: %+v", f.integrationStates["pagerduty"])
	}

	// The grid now shows it installed.
	page := integrationsBody(t, ac, base, "")
	if !strings.Contains(page, ">installed<") {
		t.Errorf("installed state not rendered after install; body: %s", page)
	}

	// An unknown slug is refused rather than written.
	resp = postForm(t, ac, base+"/settings/integrations/install", url.Values{"slug": {"no-such-integration"}})
	got := resp.StatusCode
	resp.Body.Close()
	if got != http.StatusBadRequest {
		t.Errorf("install unknown slug: status=%d, want 400", got)
	}
}

// A destructive disconnect never fires on the tile click: the installed tile's
// drawer offers a Disconnect link to a ConfirmDialog, and only the dialog's
// confirm button POSTs the disconnect. An available integration offers no
// disconnect target at all.
func TestIntegrationsDisconnectRoutesThroughConfirm(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Install so there is something to disconnect.
	postForm(t, ac, base+"/settings/integrations/install", url.Values{"slug": {"slack"}}).Body.Close()

	// The installed tile's drawer routes to the confirm step, and does NOT carry a
	// direct disconnect form — destruction is never fired on the drawer's action.
	drawer := integrationsBody(t, ac, base, "&open=slack")
	if !strings.Contains(drawer, "confirm=slack") {
		t.Errorf("installed drawer has no Disconnect link to the confirm step; body: %s", drawer)
	}
	if strings.Contains(drawer, `action="/settings/integrations/disconnect"`) {
		t.Errorf("the drawer fired disconnect directly instead of routing through confirm; body: %s", drawer)
	}

	// The ConfirmDialog renders the disconnect POST behind the confirm button.
	confirm := integrationsBody(t, ac, base, "&confirm=slack")
	for _, want := range []string{
		"Disconnect Slack", "Nothing was deleted on the",
		`action="/settings/integrations/disconnect"`, `name="slug" value="slack"`,
	} {
		if !strings.Contains(confirm, want) {
			t.Errorf("confirm dialog missing %q; body: %s", want, confirm)
		}
	}

	// A confirm param on an available (not installed) integration offers no
	// destructive act — there is nothing to disconnect.
	postForm(t, ac, base+"/settings/integrations/disconnect", url.Values{"slug": {"slack"}}).Body.Close()
	noTarget := integrationsBody(t, ac, base, "&confirm=slack")
	if strings.Contains(noTarget, `action="/settings/integrations/disconnect"`) {
		t.Errorf("a disconnect confirm rendered for an available integration; body: %s", noTarget)
	}

	// The confirm button's POST performs the disconnect, returning to available.
	resp := postForm(t, ac, base+"/settings/integrations/disconnect", url.Values{"slug": {"slack"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("disconnect: status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	if _, ok := f.integrationStates["slack"]; ok {
		t.Fatalf("disconnect did not return the integration to available: %+v", f.integrationStates)
	}
}

// The needs-config install state renders as its own state on the tile and carries
// a configuration callout in the drawer. Seeded directly: needs-config is a real
// stored state the render must handle, never fabricated into the catalogue.
func TestIntegrationsNeedsConfigRenders(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.integrationStates["jira"] = db.IntegrationState{Slug: "jira", State: integrationNeedsConfig}
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := integrationsBody(t, ac, base, "")
	if !strings.Contains(page, ">needs config<") {
		t.Errorf("needs-config state not rendered on the grid; body: %s", page)
	}
	drawer := integrationsBody(t, ac, base, "&open=jira")
	if !strings.Contains(drawer, "Configuration needed") {
		t.Errorf("needs-config drawer missing its configuration callout; body: %s", drawer)
	}
}

// The category segment and search box narrow the catalogue.
func TestIntegrationsFilterAndSearch(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// The SIEM category shows Splunk/Elastic but not Slack (a Notify integration).
	siem := integrationsBody(t, ac, base, "&cat=SIEM")
	if !strings.Contains(siem, "Splunk") || strings.Contains(siem, ">Slack<") {
		t.Errorf("SIEM filter did not narrow the grid; body: %s", siem)
	}
	// Search narrows to a name match.
	search := integrationsBody(t, ac, base, "&q=jira")
	if !strings.Contains(search, "Jira") || strings.Contains(search, "Splunk") {
		t.Errorf("search did not narrow the grid; body: %s", search)
	}
}

// The Integrations tab is admin-gated (it is reached only through /settings), and
// its mutations are admin acts. A source is distinct from an integration: the
// Sources tab carries the discovery-source catalogue, the Integrations tab the
// third-party tiles, and neither bleeds into the other.
func TestIntegrationsAdminGatedAndDistinctFromSources(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	// A viewer cannot install or disconnect, and no state is written.
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

	// An anonymous mutation is bounced to login.
	anon := newClient(t)
	resp := postForm(t, anon, base+"/settings/integrations/install", url.Values{"slug": {"slack"}})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon install: status=%d loc=%q", resp.StatusCode, resp.Header.Get("Location"))
	}

	// Sources and Integrations are distinct tabs. The tab bar carries both.
	ac := login(t, base, "admin", "hunter2hunter2")
	tabBar := settingsBody(t, ac, base)
	for _, tab := range []string{"tab=sources", "tab=integrations"} {
		if !strings.Contains(tabBar, tab) {
			t.Errorf("settings tab bar missing %q", tab)
		}
	}
	// The Integrations tab has the tiles, not the source catalogue's proposers.
	integ := integrationsBody(t, ac, base, "")
	if strings.Contains(integ, "RIPEstat") || strings.Contains(integ, ">proposer<") {
		t.Errorf("the Integrations tab bled the source catalogue in; body: %s", integ)
	}
	// The Sources tab still carries the discovery catalogue and not the tiles. It
	// stays viewer-reachable at /sources.
	src := settingsTabBody(t, ac, base, "sources")
	if !strings.Contains(src, "crt.sh") || !strings.Contains(src, ">proposer<") {
		t.Errorf("the Sources tab lost the discovery catalogue; body: %s", src)
	}
	if strings.Contains(src, "Install Slack") {
		t.Errorf("the Sources tab bled the integration tiles in; body: %s", src)
	}
}
