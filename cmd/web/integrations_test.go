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

// --- fakeStore integration-state methods -----------------------------------

func (f *fakeStore) ListIntegrationStates(context.Context) ([]db.IntegrationState, error) {
	rows := make([]db.IntegrationState, 0, len(f.integrationStates))
	for _, st := range f.integrationStates {
		rows = append(rows, st)
	}
	return rows, nil
}

func (f *fakeStore) UpsertIntegrationState(_ context.Context, arg db.UpsertIntegrationStateParams) (db.IntegrationState, error) {
	// Re-install keeps the existing channel binding (the real ON CONFLICT omits
	// channel_id), so mirror that: preserve any bound Channel on the prior row.
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
	// Only an installed integration has a row to bind; binding one with no row is a
	// no-op, exactly as the WHERE slug = $1 UPDATE touches nothing.
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

// --- helpers ----------------------------------------------------------------

// skipIfIntegrationsHidden skips a behavioural Integrations test while the surface
// is hidden (#388, integrationsEnabled == false). Flipping the flag to revive the
// surface makes these tests run again.
func skipIfIntegrationsHidden(t *testing.T) {
	t.Helper()
	if !integrationsEnabled {
		t.Skip("integrations surface hidden (#388)")
	}
}

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

// --- hidden-surface tests ---------------------------------------------------
//
// The Integrations surface is hidden (#388, integrationsEnabled == false). These
// tests assert nothing integration-related is reachable in that shipped state.
// When the surface is revived (the flag flipped to true) they skip, and the
// behavioural tests below take over — so both states are covered in the tree and
// flipping the one constant flips which set runs.

// No Integrations tab appears in the Settings navigation while the surface is
// hidden, and there is no link to the tab.
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

// Navigating directly to /settings?tab=integrations does not render the
// placeholder catalog: it redirects to another tab.
func TestIntegrationsDirectNavRedirects(t *testing.T) {
	if integrationsEnabled {
		t.Skip("integrations surface is live; the tab is expected to render")
	}
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// The test client does not follow redirects, so the 303 is observable here.
	resp, err := ac.Get(base + "/settings?tab=integrations")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/settings?tab=scans" {
		t.Fatalf("direct nav to hidden tab: status=%d loc=%q, want 303 -> /settings?tab=scans",
			resp.StatusCode, resp.Header.Get("Location"))
	}
	page := body(t, resp)
	// The catalog and its scaffolding must not render on the redirect body.
	for _, absent := range []string{"Channels need no integration", "Install Slack", "Install PagerDuty"} {
		if strings.Contains(page, absent) {
			t.Errorf("the placeholder catalog rendered on the hidden tab; found %q; body: %s", absent, page)
		}
	}
}

// No user-facing route can write to integration_state: the install and disconnect
// routes are unregistered while the surface is hidden, so a POST is a 404 and
// nothing is written.
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
		// With no POST handler registered, the mux answers 405 — the path is only
		// matched by the catch-all "GET /" subtree, so it is method-not-allowed
		// rather than not-found. Either way there is no route that writes: the
		// definitive check is that integration_state stays empty below.
		if got != http.StatusMethodNotAllowed && got != http.StatusNotFound {
			t.Errorf("POST %s while hidden: status=%d, want 405/404 (no write route registered)", path, got)
		}
	}
	if len(f.integrationStates) != 0 {
		t.Fatalf("a hidden-surface route wrote to integration_state; got %d rows", len(f.integrationStates))
	}
}

// --- behavioural tests (dormant while the surface is hidden) -----------------
//
// These exercise the full Integrations surface and run only when it is revived
// (integrationsEnabled == true). They are kept in the tree so the future real
// build inherits the coverage by flipping the one constant.

// The tile grid renders every catalogued integration with its install state, the
// category segments, and the channels-vs-integration callout — with no fabricated
// install state: a fresh install shows every tile available (not installed).
func TestIntegrationsTileGridRenders(t *testing.T) {
	skipIfIntegrationsHidden(t)
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
	// The channels-vs-integration distinction is stated (the spec callout names that
	// built-in channels deliver raw JSON while integrations add formatting on top).
	if !strings.Contains(page, "need no integration") || !strings.Contains(page, "Channels are built in") {
		t.Errorf("the channels-vs-integration callout is missing; body: %s", page)
	}
}

// Opening a tile shows its consent grants; installing from that drawer is the
// consent, and it persists real install state. Consent is gated: the grants are
// shown with the install action, and grants are all-or-nothing.
func TestIntegrationsConsentGatingAndInstall(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// The drawer for an available integration shows its consent grants alongside the
	// install action (opened by ?view=, the spec PRG drawer).
	drawer := integrationsBody(t, ac, base, "&view=pagerduty")
	for _, want := range []string{
		"This integration can", "Read signals", "Write annotations", "writes",
		"Install PagerDuty",
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

// The spec drawer for an installed integration offers Remove and Send-test acts
// that POST to their own routes (#26j); an available integration offers no remove
// target. Remove returns the integration to available.
func TestIntegrationsDrawerRemoveAndTest(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Install so there is something to remove.
	postForm(t, ac, base+"/settings/integrations/install", url.Values{"slug": {"slack"}}).Body.Close()

	// The installed tile's drawer (?view=) carries the Remove act and the Send-test
	// affordance. Freshly installed it is unbound (no delivery Channel), so Send test is
	// gated OFF — the disabled button with the "connect a channel" hint, not an active
	// POST form (#39b: Send test is gated on a delivery-Channel binding). The active
	// test form and its bound behaviour are covered in integrations_channel_test.go.
	drawer := integrationsBody(t, ac, base, "&view=slack")
	for _, want := range []string{
		`action="/settings/integrations/remove"`,
		`name="id" value="slack"`, "Remove", "Send test", "Connect a channel to test",
	} {
		if !strings.Contains(drawer, want) {
			t.Errorf("installed drawer missing %q; body: %s", want, drawer)
		}
	}
	// Unbound, there is no active Send-test POST form.
	if strings.Contains(drawer, `action="/settings/integrations/test"`) {
		t.Errorf("an unbound integration offered an active Send-test form; body: %s", drawer)
	}

	// An available integration's drawer offers Install, not Remove.
	avail := integrationsBody(t, ac, base, "&view=jira")
	if strings.Contains(avail, `action="/settings/integrations/remove"`) {
		t.Errorf("an available integration's drawer offered Remove; body: %s", avail)
	}
	if !strings.Contains(avail, "Install Jira") {
		t.Errorf("an available integration's drawer has no Install action; body: %s", avail)
	}

	// The Remove POST returns the integration to available.
	resp := postForm(t, ac, base+"/settings/integrations/remove", url.Values{"id": {"slack"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("remove: status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	if _, ok := f.integrationStates["slack"]; ok {
		t.Fatalf("remove did not return the integration to available: %+v", f.integrationStates)
	}
}

// A needs-config install state renders as the spec "needs attention" state on the
// tile and in the drawer. Seeded directly: needs-config is a real stored state the
// render must handle, never fabricated into the catalogue.
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

// The category segment and search box narrow the catalogue.
func TestIntegrationsFilterAndSearch(t *testing.T) {
	skipIfIntegrationsHidden(t)
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
	skipIfIntegrationsHidden(t)
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
