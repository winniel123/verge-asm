package main

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

// --- fakeStore source-state methods ----------------------------------------

func (f *fakeStore) ListSourceStates(context.Context) ([]db.SourceState, error) {
	rows := make([]db.SourceState, 0, len(f.sourceStates))
	for _, st := range f.sourceStates {
		rows = append(rows, db.SourceState{Slug: st.Slug, Enabled: st.Enabled})
	}
	return rows, nil
}

func (f *fakeStore) UpsertSourceState(_ context.Context, arg db.UpsertSourceStateParams) (db.SourceState, error) {
	st := db.SourceState{Slug: arg.Slug, Enabled: arg.Enabled}
	f.sourceStates[arg.Slug] = st
	return st, nil
}

// --- helpers ----------------------------------------------------------------

func toggleSourceReq(t *testing.T, c *http.Client, base, slug, enabled string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/sources/toggle", url.Values{"slug": {slug}, "enabled": {enabled}})
}

func sourcesBody(t *testing.T, c *http.Client, base string) string {
	t.Helper()
	resp, err := c.Get(base + "/sources")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /sources status = %d, want 200", resp.StatusCode)
	}
	return body(t, resp)
}

// --- tests ------------------------------------------------------------------

// The modal renders every catalogued source, both marked groups, and the
// shipped defaults from §3.1 with no override in place.
func TestSourcesModalRendersCatalogueAndDefaults(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := sourcesBody(t, ac, base)

	// Every catalogued source appears.
	for _, c := range sourceCatalog {
		if !strings.Contains(page, c.Name) {
			t.Errorf("source %q missing from the modal", c.Name)
		}
	}
	// The three consent tiers render (the spec tier cards, #26).
	for _, tier := range []string{"unencumbered", "operator-accepted", "barred"} {
		if !strings.Contains(page, tier) {
			t.Errorf("consent tier %q not rendered; body: %s", tier, page)
		}
	}
	// A proposer is labelled a proposer, never a source (ADR-0012).
	if !strings.Contains(page, ">proposer<") || !strings.Contains(page, ">source<") {
		t.Errorf("source/proposer kinds not distinguished; body: %s", page)
	}
	// crt.sh ships on; RIPEstat ships off — the §3.1 defaults, unoverridden.
	if !strings.Contains(page, "crt.sh") || !strings.Contains(page, "RIPEstat") {
		t.Errorf("expected sources missing; body: %s", page)
	}
}

// LACNIC is catalogued-not-executing (#241, ruling #30): no proposer runner ships
// for it, so it renders in the barred/catalogued bucket, offers no toggle, and its
// ?consent dialog no longer opens — there is nothing to enable until a runner lands.
func TestLacnicCataloguedSourceNoConsent(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := sourcesBody(t, ac, base)
	if !strings.Contains(page, "LACNIC registry") {
		t.Fatalf("LACNIC not rendered; body: %s", page)
	}
	// A no-runner source opens no consent dialog even by ?consent=<id>.
	dlg := getBody(t, ac, base+"/sources?consent=lacnic-registry", http.StatusOK)
	if strings.Contains(dlg, "Nobody has been able to retrieve these terms.") {
		t.Errorf("LACNIC consent dialog rendered for a no-runner source; body: %s", dlg)
	}
	if strings.Contains(dlg, "Accept and enable") {
		t.Errorf("consent dialog opened for a catalogued-not-executing source; body: %s", dlg)
	}
}

// Toggling persists the override and flips the effective state; the default is
// restored by toggling back.
func TestToggleSourcePersistsOverride(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// RIPEstat is catalogued-not-executing now (#241, ruling #30): no runner ships,
	// so it is non-toggleable — an enable POST is refused and persists nothing, even
	// carrying the acceptance field the old operator-accepted gate wanted.
	resp := postForm(t, ac, base+"/sources/toggle", url.Values{
		"slug": {"ripestat"}, "enabled": {"true"}, "agreed": {"on"},
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("catalogued-source toggle: status=%d, want 400", resp.StatusCode)
	}
	resp.Body.Close()
	if _, ok := f.sourceStates["ripestat"]; ok {
		t.Fatalf("no-runner source wrote state: %+v", f.sourceStates["ripestat"])
	}

	// ARIN ships on (a keyless proposer that executes); disabling is safe and persists.
	toggleSourceReq(t, ac, base, "arin", "false").Body.Close()
	if st, ok := f.sourceStates["arin"]; !ok || st.Enabled {
		t.Fatalf("disable override not persisted: %+v", f.sourceStates["arin"])
	}

	// crt.sh ships on and now executes (ADR-0106, #250): its runner is the ct Scan,
	// so it is a toggleable source again — disabling it persists and stops the poll.
	toggleSourceReq(t, ac, base, "crtsh", "false").Body.Close()
	if st, ok := f.sourceStates["crtsh"]; !ok || st.Enabled {
		t.Fatalf("crtsh disable override not persisted: %+v", f.sourceStates["crtsh"])
	}
}

// A catalogued-not-executing source (#241, ruling #30) offers no enable at all: its
// row carries no consent link, its ?consent dialog does not open, and an enable POST
// is refused rather than bounced to a terms dialog — there is nothing to enable until
// a real proposer.Source runner lands, at which point it returns to operator-accepted.
func TestCataloguedSourceOffersNoEnable(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// No consent-gated enable control renders for a no-runner source.
	page := getBody(t, ac, base+"/sources", http.StatusOK)
	if strings.Contains(page, "consent=ripestat") {
		t.Errorf("catalogued source rendered a consent-gated enable; body: %s", page)
	}
	// The consent dialog does not open for it.
	dlg := getBody(t, ac, base+"/sources?consent=ripestat", http.StatusOK)
	if strings.Contains(dlg, "Accept and enable") {
		t.Errorf("consent dialog opened for a catalogued source; body: %s", dlg)
	}
	// The settings twin refuses an enable POST — it renders the sources section with an
	// error (a 400 by renderSettings' section-error convention), never the consent-dialog
	// bounce, and persists nothing.
	resp := postForm(t, ac, base+"/settings/sources", url.Values{"id": {"ripestat"}, "enable": {"true"}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("catalogued enable: status=%d, want 400 (refusal render)", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "" {
		t.Errorf("catalogued enable bounced to %q rather than refusing", loc)
	}
	resp.Body.Close()
	if st, ok := f.sourceStates["ripestat"]; ok && st.Enabled {
		t.Fatalf("catalogued source enabled: %+v", st)
	}
}

// A source excluded on terms has no consent instrument the modal operator can
// satisfy, so it cannot be toggled; an unknown slug is refused too. (crt.sh was
// once here as a no-runner source; ADR-0106 gave it a runner, so it now toggles —
// asserted in TestToggleSourcePersistsOverride.)
func TestToggleRejectsBarredAndUnknown(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// hackertarget: excluded on terms. no-such-source: unknown.
	for _, slug := range []string{"hackertarget", "no-such-source"} {
		resp := toggleSourceReq(t, ac, base, slug, "true")
		got := resp.StatusCode
		resp.Body.Close()
		if got != http.StatusBadRequest {
			t.Errorf("toggle %q: status=%d, want 400", slug, got)
		}
	}
	if len(f.sourceStates) != 0 {
		t.Fatalf("no override should have been written; got %d", len(f.sourceStates))
	}
}

// Toggling is an admin act: a viewer is denied and no state is written, but a
// viewer may still read the modal — without a toggle control.
func TestViewerCannotToggleButCanView(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	vc := login(t, base, "viewer", "hunter2hunter2")
	resp := toggleSourceReq(t, vc, base, "ripestat", "true")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer toggle: status=%d, want 403", resp.StatusCode)
	}
	if len(f.sourceStates) != 0 {
		t.Fatalf("viewer toggle wrote state; got %d", len(f.sourceStates))
	}

	page := sourcesBody(t, vc, base)
	if !strings.Contains(page, "crt.sh") {
		t.Errorf("viewer cannot read the modal; body: %s", page)
	}
	if strings.Contains(page, `action="/sources/toggle"`) {
		t.Errorf("a toggle control was shown to a viewer; body: %s", page)
	}
}

// The V3 Coverage screen (#301, ADR-0110) renders its four regions: the aperture
// meters, the coverage messages, the gaps register, and the unevaluable rules.
func TestCoverageRendersRegions(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp, err := ac.Get(base + "/coverage")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /coverage status=%d, want 200", resp.StatusCode)
	}
	for _, region := range []string{
		"What the last batch walked", "Coverage messages",
		"Expected, not observed", "Unevaluable this batch",
	} {
		if !strings.Contains(got, region) {
			t.Errorf("Coverage region %q missing; body: %s", region, got)
		}
	}
}

// crt.sh has a runner again (ADR-0106, #250): the ct Scan queries certificate
// transparency, so it presents as an executing, toggleable source — with a toggle
// offered to an admin — and the not-yet-executing section (which had only crt.sh)
// no longer renders, its bucket now empty.
func TestCrtshShownAsExecuting(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := sourcesBody(t, ac, base)

	if !strings.Contains(page, "crt.sh") {
		t.Fatalf("crt.sh missing from the modal; body: %s", page)
	}
	// A toggle IS offered for crt.sh now, to an admin: it has an execution path.
	if !strings.Contains(page, `value="crtsh"`) {
		t.Errorf("no toggle control offered for crt.sh; body: %s", page)
	}
	// The not-yet-executing section had only crt.sh, so it no longer renders.
	if strings.Contains(page, "not yet executing") {
		t.Errorf("the not-yet-executing section rendered with no no-runner source; body: %s", page)
	}
}

func coverageBody(t *testing.T, c *http.Client, base string) string {
	t.Helper()
	resp, err := c.Get(base + "/coverage")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /coverage status = %d, want 200", resp.StatusCode)
	}
	return body(t, resp)
}

// The aperture meters render one CoverageMeter per declared scope in the census
// state — an address scope counting the addresses it enumerates, a name scope
// counting the owner names its zone declares. A census never claims a
// denominator, so the meter reads "census · …" and no percentage appears.
func TestCoverageApertureMeters(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	declare(t, ac, base, "address", "203.0.113.0/24").Body.Close()

	page := coverageBody(t, ac, base)

	// Both scopes surface as census meters, labelled by their scope key.
	for _, want := range []string{
		"example.com", "203.0.113.0/24", "census", "addresses", "declared names",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("aperture meter is missing %q; body: %s", want, page)
		}
	}
}

// ADR-0108 / #249: an unavailable vantage is surfaced on Coverage by name, so a
// resolver that went unreachable reads as "we could not look from here" rather
// than as an empty measurement — visible without inspecting observation.value.
// It includes the resolver-only `local` vantage, which the prober list excludes.
func TestCoverageSurfacesUnavailableVantages(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// The shipped resolver-only `local` vantage, marked unavailable — no host, so
	// the prober list would never show it, but the outage must still be loud.
	f.vantages = append(f.vantages, db.Vantage{
		ID: 1, Name: "local", Class: "internet",
		Resolver:     "127.0.0.11:53",
		Availability: pgtype.Text{String: "unavailable", Valid: true},
	})
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)

	if !strings.Contains(page, "local") || !strings.Contains(page, "127.0.0.11:53") {
		t.Errorf("unavailable `local` vantage not named on Coverage; body: %s", page)
	}
	if !strings.Contains(page, "unreachable") {
		t.Errorf("Coverage does not render the silent-vantage state; body: %s", page)
	}
	// The signal is distinct from an empty result: the message says we could not
	// look, never that nothing was found.
	if !strings.Contains(page, "could not look") {
		t.Errorf("the unavailable-vantage message does not distinguish blindness from emptiness; body: %s", page)
	}
}

// With no unavailable vantage the register does not render — an empty install is
// not told it cannot look from anywhere.
func TestCoverageOmitsUnavailableRegisterWhenAllAvailable(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)

	if strings.Contains(page, "unavailable vantages") {
		t.Errorf("the unavailable-vantage register rendered with no unavailable vantage; body: %s", page)
	}
}

// §6.3, and this ticket's own AC: no proportion-of-estate figure appears on the
// Coverage screen. ADR-0095 — the statement counts what the instrument looks at,
// never how much of the estate it covers.
func TestCoverageHasNoProportionOfEstate(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)

	// Scope the check to the rendered body — the shared stylesheet legitimately
	// carries "100%" in its layout rules, which is not a coverage figure. The V3
	// main carries attributes, so anchor on the opening tag prefix.
	main := page
	if i := strings.Index(page, "<main"); i >= 0 {
		main = page[i:]
	}
	// No percentage figure, and no estate-completeness score phrasing, in the body.
	for _, banned := range []string{"%", "estate completeness", "% covered", "% of your estate"} {
		if strings.Contains(main, banned) {
			t.Errorf("a proportion-of-estate figure appeared (%q); body: %s", banned, main)
		}
	}
}

// On a fresh install every region falls to its design-system empty-state — no
// scope to walk, no coverage messages, no gaps, and every rule able to evaluate.
// No fabricated data stands in for a read that has nothing to report.
func TestCoverageEmptyStates(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)
	for _, want := range []string{
		"No scope to walk yet", "No coverage messages", "No gaps this batch", "Every rule could evaluate",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("Coverage empty-state %q missing; body: %s", want, page)
		}
	}
	// The aperture empty-state points at the next action — declaring a scope.
	if !strings.Contains(page, `href="/scope"`) {
		t.Errorf("the aperture empty-state does not point at Scope; body: %s", page)
	}
}

// Coverage surfaces blanket responders (ADR-0104 §4): when an address answers on
// every port its reach is a Gap. It surfaces both as a "no origin" gap row keyed
// on the address and as a currency message naming the proxy edge and pointing at
// declaring an address scope. Absent any blanket responder neither renders.
func TestCoverageSurfacesBlanketResponders(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addClassReachability(t, "104.21.61.6:443/tcp", "internet", obsClock,
		`{"outcome":"gap","cause":"blanket-responder","reason":"this address answers on all ports — it is a proxy edge, not your origin"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)
	for _, want := range []string{
		"TCP on all ports", "104.21.61.6", "proxy edge", "address scope", "no origin",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("Coverage blanket-responder gap missing %q; body: %s", want, page)
		}
	}
}

// With no blanket responder, no gap or proxy-edge message renders — the gaps
// register is a statement only when there is something to state.
func TestCoverageOmitsBlanketSectionWhenNone(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addClassReachability(t, "198.51.100.1:443/tcp", "internet", obsClock, `{"outcome":"reached"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)
	if strings.Contains(page, "proxy edge") {
		t.Errorf("Coverage rendered a blanket-responder statement with no blanket responder; body: %s", page)
	}
}

// The web layer's mirror of the qtype set never drifts from the leaf's authored
// set (resolutionwalk.DefaultOffers). If the leaf's set moves, this fails.
func TestDNSQtypeSetMatchesLeaf(t *testing.T) {
	want := resolutionwalk.DefaultOffers().Qtypes
	if len(want) != len(dnsQtypeSet) {
		t.Fatalf("qtype set length: web mirror has %d, leaf has %d", len(dnsQtypeSet), len(want))
	}
	for i, q := range want {
		if string(q) != dnsQtypeSet[i] {
			t.Errorf("qtype[%d]: web mirror %q, leaf %q", i, dnsQtypeSet[i], string(q))
		}
	}
}

// Both mutating and reading source routes require a login.
func TestSourceRoutesRequireLogin(t *testing.T) {
	base := start(t, newFakeStore(), "")
	c := newClient(t)

	resp := toggleSourceReq(t, c, base, "ripestat", "true")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon toggle: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}

	resp, err := c.Get(base + "/sources")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon GET /sources: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
}
