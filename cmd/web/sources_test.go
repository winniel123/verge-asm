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
	// Both marked groups render.
	if !strings.Contains(page, "What you may be able to resolve") ||
		!strings.Contains(page, "What nobody has been able to resolve") {
		t.Errorf("the two marked groups are not both rendered; body: %s", page)
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

// LACNIC's actionable group is empty by construction, and it still renders — the
// #47 "render even when empty" requirement.
func TestEmptyMarkedGroupStillRenders(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := sourcesBody(t, ac, base)
	if !strings.Contains(page, "LACNIC registry") {
		t.Fatalf("LACNIC not rendered; body: %s", page)
	}
	if !strings.Contains(page, "every open question here is one nobody has been able to answer") {
		t.Errorf("empty actionable group did not render its empty state; body: %s", page)
	}
}

// Toggling persists the override and flips the effective state; the default is
// restored by toggling back.
func TestToggleSourcePersistsOverride(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// RIPEstat ships off; enable it.
	resp := toggleSourceReq(t, ac, base, "ripestat", "true")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/sources" {
		t.Fatalf("toggle: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	if st, ok := f.sourceStates["ripestat"]; !ok || !st.Enabled {
		t.Fatalf("override not persisted: %+v", f.sourceStates["ripestat"])
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

// Coverage is the source-enablement modal's entry point and links to it.
func TestCoverageStubLinksToModal(t *testing.T) {
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
	if !strings.Contains(got, `href="/sources"`) {
		t.Errorf("Coverage stub does not link to the source modal; body: %s", got)
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

// The aperture statement renders one line per aperture input (§3.2, §6.3): the
// seven inputs, the qtype set spelled out, and the dns cadence.
func TestCoverageRendersApertureStatement(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)

	for _, input := range []string{
		"Enabled sources", "Port sets", "Vantages", "TLS candidate set",
		"Qtype set", "Control-probe population", "Queried address scope",
	} {
		if !strings.Contains(page, input) {
			t.Errorf("aperture input %q missing from the statement", input)
		}
	}
	// The qtype set is spelled out, not summarised, and the dns cadence is stated.
	for _, q := range dnsQtypeSet {
		if !strings.Contains(page, q) {
			t.Errorf("qtype %q missing from the aperture statement", q)
		}
	}
	if !strings.Contains(page, "daily") {
		t.Errorf("dns cadence (daily) not stated; body: %s", page)
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
	if !strings.Contains(page, "unavailable") {
		t.Errorf("Coverage does not render the unavailable state; body: %s", page)
	}
	// The signal is distinct from an empty result: the register says we could not
	// look, never that nothing was found.
	if !strings.Contains(page, "could not look") {
		t.Errorf("the unavailable-vantage register does not distinguish blindness from emptiness; body: %s", page)
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
	// carries "100%" in its layout rules, which is not a coverage figure.
	main := page
	if i := strings.Index(page, "<main>"); i >= 0 {
		main = page[i:]
	}
	// No percentage figure, and no estate-completeness score phrasing, in the body.
	for _, banned := range []string{"%", "estate completeness", "% covered", "% of your estate"} {
		if strings.Contains(main, banned) {
			t.Errorf("a proportion-of-estate figure appeared (%q); body: %s", banned, main)
		}
	}
}

// The "Enabled sources" aperture line counts every source with an execution
// path. crt.sh has a runner again (ADR-0106, #250), so it is both a numerator and
// a denominator: the four ship-on sources (ARIN, AFRINIC, APNIC-CAIDA, crt.sh) of
// the eight toggleable ones read "4 of 8", reversing the "3 of 7" that excluded
// crt.sh while it produced nothing.
func TestCoverageCountsCrtshAsExecuting(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)

	if !strings.Contains(page, "4 of 8 sources enabled") {
		t.Errorf("enabled-sources line should read \"4 of 8 sources enabled\"; body: %s", page)
	}
	if strings.Contains(page, "3 of 7") {
		t.Errorf("crt.sh is still excluded from the active-source count; body: %s", page)
	}
}

// The zero-coverage state renders the four-step day-one checklist. Each of the
// first three steps links to the surface that performs it (Seeds); running the
// first batch has no surface yet, so it names the capability without a link.
func TestCoverageZeroCoverageChecklist(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)

	for _, step := range []string{
		"Declare your domain", "Upload a zone file", "Add an internet vantage", "Run the first batch",
	} {
		if !strings.Contains(page, step) {
			t.Errorf("checklist step %q missing in the zero-coverage state", step)
		}
	}
	if !strings.Contains(page, `href="/scope"`) {
		t.Errorf("a checklist step does not link to the Scope surface; body: %s", page)
	}
	// Running the first batch is the worker's job at cadence, not a button.
	if !strings.Contains(page, "Runs automatically at cadence") {
		t.Errorf("run-the-first-batch step should name the capability without a link; body: %s", page)
	}
}

// Once a scope is declared the estate is no longer at zero coverage: the
// checklist retires and the queried-scope line states the declared counts.
func TestCoverageChecklistRetiresWhenScopeDeclared(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()

	page := coverageBody(t, ac, base)
	if strings.Contains(page, "Declare your domain") {
		t.Errorf("the day-one checklist should retire once a scope is declared; body: %s", page)
	}
	if !strings.Contains(page, "1 name") {
		t.Errorf("queried-scope line should state the declared scope count; body: %s", page)
	}
}

// The retention section exists as a stub (real dials are #26/#28/#29).
func TestCoverageRetentionStub(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)
	if !strings.Contains(page, "Retention") {
		t.Errorf("retention stub section missing; body: %s", page)
	}
}

// Coverage surfaces blanket responders (ADR-0104 §4): when an address answers on
// every port its reach is a Gap, and Coverage states the proxy-edge finding in
// prose with the address, and points at declaring an address scope. Absent any
// blanket responder the section does not render.
func TestCoverageSurfacesBlanketResponders(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addClassReachability(t, "104.21.61.6:443/tcp", "internet", obsClock,
		`{"outcome":"gap","cause":"blanket-responder","reason":"this address answers on all ports — it is a proxy edge, not your origin"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)
	for _, want := range []string{
		"answer TCP on all ports", "104.21.61.6", "proxy edge", "address scope",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("Coverage blanket-responder statement missing %q; body: %s", want, page)
		}
	}
}

// With no blanket responder, the Coverage blanket-responder section does not
// render — it is a standing statement only when there is something to state.
func TestCoverageOmitsBlanketSectionWhenNone(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addClassReachability(t, "198.51.100.1:443/tcp", "internet", obsClock, `{"outcome":"reached"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)
	if strings.Contains(page, "answer TCP on all ports") {
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
