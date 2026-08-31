package main

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

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

func (f *fakeStore) CTReliabilityWindow(_ context.Context, arg db.CTReliabilityWindowParams) (db.CTReliabilityWindowRow, error) {
	return f.ctReliability[arg.Source], nil
}

func (f *fakeStore) CTLastBatchAdmitCount(context.Context) (int64, error) {
	return f.ctAdmitCount, nil
}

func (f *fakeStore) CTTailLastBatch(context.Context) (db.CTTailLastBatchRow, error) {
	return f.ctTailBatch, nil
}

func (f *fakeStore) CountCertificateMaterial(context.Context) (int64, error) {
	return f.certMaterialCount, nil
}

// The reliability bar (spec §3, #879) reads each bulk CT source's rolling window,
// evaluates it, and shapes it for the card: the operator-keyed primary reports
// pass/fail per limb and degrades when it misses one; crt.sh reports exempt and is
// never degraded, its measured values kept for contrast.
func TestCTReliabilityViews(t *testing.T) {
	f := newFakeStore()
	f.ctReliability = map[string]db.CTReliabilityWindowRow{
		"certspotter": {Total: 200, Successes: 196, Empties: 0, P95LatencyMs: 3200},
		"crtsh":       {Total: 8, Successes: 4, Empties: 2, P95LatencyMs: 59600},
	}
	s := &server{store: f}

	views, err := s.ctReliabilityViews(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 {
		t.Fatalf("want 2 bulk-source views, got %d", len(views))
	}
	byslug := map[string]ctReliabilityView{}
	for _, v := range views {
		byslug[v.Slug] = v
	}

	// The operator-keyed primary at 196/200 is below the 99% success bar: not exempt,
	// degraded, and its measured values render.
	cs := byslug["certspotter"]
	if cs.Exempt {
		t.Errorf("certspotter must not be exempt")
	}
	if !cs.Degraded {
		t.Errorf("certspotter below the success bar must read degraded")
	}
	if cs.SuccessPass {
		t.Errorf("196/200 must fail the 99%% success bar")
	}
	if cs.SuccessPct != "98.0%" {
		t.Errorf("certspotter success = %q, want 98.0%%", cs.SuccessPct)
	}
	if cs.P95Display != "3.2 s" {
		t.Errorf("certspotter p95 = %q, want 3.2 s", cs.P95Display)
	}

	// crt.sh is exempt as the keyless fallback: never degraded, its measured values
	// shown for contrast, its name from the catalogue.
	sh := byslug["crtsh"]
	if !sh.Exempt {
		t.Errorf("crt.sh must be exempt")
	}
	if sh.Degraded {
		t.Errorf("an exempt fallback is never degraded")
	}
	if sh.SuccessPct != "50.0%" {
		t.Errorf("crtsh success = %q, want 50.0%%", sh.SuccessPct)
	}
	if sh.Name != "crt.sh" {
		t.Errorf("crtsh name = %q, want crt.sh", sh.Name)
	}
}

// A source with no recent samples reports no data — an em dash, never a fabricated
// zero — and is never degraded (#879).
func TestCTReliabilityViewsNoData(t *testing.T) {
	s := &server{store: newFakeStore()}

	views, err := s.ctReliabilityViews(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range views {
		if v.HasData {
			t.Errorf("%s reports data with no samples", v.Slug)
		}
		if v.Degraded {
			t.Errorf("%s with no samples must not be degraded", v.Slug)
		}
		if v.SuccessPct != "—" || v.P95Display != "—" {
			t.Errorf("%s no-data view = success %q p95 %q, want em dashes", v.Slug, v.SuccessPct, v.P95Display)
		}
	}
}

// The active-source hero (#880) derives which bulk source is live from the freshest
// reliability sample — web never reads the worker token. The keyed primary live means the
// key is set; crt.sh live means it is not. A below-bar primary reads in danger with no
// silent swap, and no samples at all means no bulk ct scan has run.
func TestNewCTSourceHero(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	const crtName = "crt.sh"
	const certName = "Cert Spotter (operator key)"

	t.Run("primary live", func(t *testing.T) {
		cert := ctReliabilityView{Slug: "certspotter", Name: certName, HasData: true, LastRun: now.Add(-5 * time.Minute)}
		crt := ctReliabilityView{Slug: "crtsh", Name: crtName, HasData: true, Exempt: true, LastRun: now.Add(-2 * time.Hour)}
		h := newCTSourceHero(crt, cert, 42, now)
		if !h.HasRun || !h.IsPrimary {
			t.Fatalf("Cert Spotter fresher must be the live primary: %+v", h)
		}
		if h.StatusLabel != "primary · Cert Spotter" || h.StatusClass != "accent" {
			t.Errorf("status = %q/%q, want primary · Cert Spotter/accent", h.StatusLabel, h.StatusClass)
		}
		if !h.KeyDetected || h.KeyLabel != "detected" {
			t.Errorf("keyed primary live => key detected; got %q", h.KeyLabel)
		}
		if h.DormantName != crtName || h.DormantRole != "fallback" {
			t.Errorf("dormant = %q/%q, want crt.sh/fallback", h.DormantName, h.DormantRole)
		}
		if h.LastRunRel != "5m" || h.Names != 42 {
			t.Errorf("readout = %q/%d, want 5m/42", h.LastRunRel, h.Names)
		}
		if h.Active.Slug != "certspotter" {
			t.Errorf("tiles must read the live source, got %q", h.Active.Slug)
		}
	})

	t.Run("primary degraded", func(t *testing.T) {
		cert := ctReliabilityView{Slug: "certspotter", Name: certName, HasData: true, Degraded: true, LastRun: now.Add(-time.Minute)}
		h := newCTSourceHero(ctReliabilityView{Slug: "crtsh", Name: crtName}, cert, 0, now)
		if !h.Degraded || h.StatusClass != "danger" {
			t.Errorf("a below-bar primary is danger+degraded, got %q degraded=%v", h.StatusClass, h.Degraded)
		}
	})

	t.Run("fallback live", func(t *testing.T) {
		crt := ctReliabilityView{Slug: "crtsh", Name: crtName, HasData: true, Exempt: true, LastRun: now.Add(-9 * time.Minute)}
		cert := ctReliabilityView{Slug: "certspotter", Name: certName}
		h := newCTSourceHero(crt, cert, 7, now)
		if !h.HasRun || h.IsPrimary {
			t.Fatalf("crt.sh-only must be the live fallback: %+v", h)
		}
		if h.StatusLabel != "fallback · crt.sh" || h.StatusClass != "neutral" {
			t.Errorf("status = %q/%q, want fallback · crt.sh/neutral", h.StatusLabel, h.StatusClass)
		}
		if h.KeyDetected || h.KeyLabel != "not set" {
			t.Errorf("no keyed run => key not set; got %q", h.KeyLabel)
		}
		if h.DormantName != "Cert Spotter" || h.DormantRole != "primary" {
			t.Errorf("dormant = %q/%q, want Cert Spotter/primary", h.DormantName, h.DormantRole)
		}
		if h.Active.Slug != "crtsh" {
			t.Errorf("tiles must read crt.sh, got %q", h.Active.Slug)
		}
	})

	t.Run("no run", func(t *testing.T) {
		h := newCTSourceHero(ctReliabilityView{Slug: "crtsh", Name: crtName}, ctReliabilityView{Slug: "certspotter", Name: certName}, 0, now)
		if h.HasRun {
			t.Errorf("no samples must not assert a run: %+v", h)
		}
		if h.KeyLabel != "not set" {
			t.Errorf("no run => key not set, got %q", h.KeyLabel)
		}
	})
}

// A live keyed primary that is below its bar renders the primary badge, the run readout
// with the admitted count, the failing limb, the honest no-failover callout, the detected
// key, and the measured targets — all from run data, never the worker token (#880).
func TestSourcesCTHeroRendersPrimaryDegraded(t *testing.T) {
	f := newFakeStore()
	f.ctReliability = map[string]db.CTReliabilityWindowRow{
		"certspotter": {Total: 200, Successes: 196, P95LatencyMs: 3200, LastAt: pgtype.Timestamptz{Time: time.Now(), Valid: true}},
	}
	f.ctAdmitCount = 4213
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := sourcesBody(t, ac, base)
	for _, want := range []string{
		"primary · Cert Spotter",        // status badge
		"last ct scan · Cert Spotter",   // run readout
		"4213 names admitted",           // the admitted count
		"under bar",                     // the failing success limb
		"Runtime failover is not built", // honest no-swap edge (§6.3)
		"detected",                      // operator-key presence, inferred
		"≥ 99%", "≤ 5 s",                // the measured targets
	} {
		if !strings.Contains(page, want) {
			t.Errorf("degraded-primary hero missing %q; body: %s", want, page)
		}
	}
}

// With only crt.sh recording, it is the live keyless fallback: bar-exempt tiles, the key
// not set, and a hint on how to promote Cert Spotter (#880).
func TestSourcesCTHeroRendersFallbackExempt(t *testing.T) {
	f := newFakeStore()
	f.ctReliability = map[string]db.CTReliabilityWindowRow{
		"crtsh": {Total: 8, Successes: 4, Empties: 2, P95LatencyMs: 59600, LastAt: pgtype.Timestamptz{Time: time.Now(), Valid: true}},
	}
	f.ctAdmitCount = 12
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := sourcesBody(t, ac, base)
	for _, want := range []string{
		"fallback · crt.sh",       // status badge
		"bar-exempt",              // the fallback is muted, not failed
		"not set",                 // key not set
		"VERGE_CERTSPOTTER_TOKEN", // the promote hint
		"12 names admitted",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("fallback hero missing %q; body: %s", want, page)
		}
	}
}

// The More-CT-capabilities card (#881, spec §6.1) renders the drift tail and the
// verification point-check beside the bulk hero, each from real state: the tail on with
// its last ct-tail run readout, verification with the captured-certificate pool. Neither
// reads the worker token or the bulk reliability bar.
func TestSourcesCTCapabilitiesCardLive(t *testing.T) {
	f := newFakeStore()
	f.sourceStates["ct-tail"] = db.SourceState{Slug: "ct-tail", Enabled: true}
	f.ctTailBatch = db.CTTailLastBatchRow{
		LastAt: pgtype.Timestamptz{Time: time.Now().Add(-4 * time.Minute), Valid: true},
		Names:  37,
	}
	f.certMaterialCount = 812
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := sourcesBody(t, ac, base)
	for _, want := range []string{
		"More CT capabilities",        // the card header
		"drift tail",                  // the tail capability
		"last ct-tail scan",           // the tail's own run readout
		"37 names admitted",           // the tail's admitted count
		"verification",                // the verification capability
		"812 certificates captured",   // verification's captured pool
		"ephemeral event",             // the honest no-durable-result edge (#878)
	} {
		if !strings.Contains(page, want) {
			t.Errorf("capabilities card missing %q; body: %s", want, page)
		}
	}
}

// With the tail off and nothing captured, the card is honest: the tail reads off with no
// run readout, and verification awaits its first capture (#881).
func TestSourcesCTCapabilitiesCardEmpty(t *testing.T) {
	f := newFakeStore()
	// ct-tail unset => ships off; no tail batch; zero captures.
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := sourcesBody(t, ac, base)
	for _, want := range []string{
		"More CT capabilities",
		"awaiting captures",         // verification with no captures
		"0 certificates captured",   // the truthful zero
	} {
		if !strings.Contains(page, want) {
			t.Errorf("empty capabilities card missing %q; body: %s", want, page)
		}
	}
	if strings.Contains(page, "last ct-tail scan") {
		t.Errorf("tail with no run must not render a run readout; body: %s", page)
	}
}

// No samples from either bulk source: the hero asserts no run, never a fabricated live
// source (#880).
func TestSourcesCTHeroNoRun(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := sourcesBody(t, ac, base)
	for _, want := range []string{"awaiting first scan", "No bulk ct scan has run yet"} {
		if !strings.Contains(page, want) {
			t.Errorf("no-run hero missing %q; body: %s", want, page)
		}
	}
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

// #890: a lagging address scope — one the batch cannot finish inside its cadence —
// renders the oldest-current as-of beside counted/total, and the addresses that have
// aged out of currency mint NO operator message (the trailing-edge staleness Gap folds
// to declared/current, decision #882; a Gap mints no message, ADR-0026/0104). Currency
// stays nominal: the numerator reads through the k×declared-cadence window (here
// 2×86400s), so an aged observation drops from the current set, never stretching the
// window to an effective cadence.
func TestCoverageLaggingScopeAsOfNoStalenessMessage(t *testing.T) {
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startAt(t, f, now)
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "address", "203.0.113.0/24").Body.Close()

	// Three subjects still current (inside the 2×86400s window), one of them the oldest.
	oldest := now.Add(-30 * time.Hour)
	f.addReachability(t, "203.0.113.10:443/tcp", now.Add(-1*time.Hour), "reached")
	f.addReachability(t, "203.0.113.20:443/tcp", oldest, "reached")
	f.addReachability(t, "203.0.113.30:443/tcp", now.Add(-3*time.Hour), "reached")
	// Two subjects aged out of currency — the trailing edge. They must mint no message.
	f.addReachability(t, "203.0.113.98:443/tcp", now.Add(-96*time.Hour), "reached")
	f.addReachability(t, "203.0.113.99:443/tcp", now.Add(-96*time.Hour), "reached")

	page := coverageBody(t, ac, base)

	// The honest lag: 3 still-current subjects over 256 declared addresses.
	if !strings.Contains(page, "3 / 256") {
		t.Errorf("lagging scope must show counted/total 3 / 256; body: %s", page)
	}
	// The oldest-current as-of, with its ISO instant as the tooltip.
	if !strings.Contains(page, "oldest still current") {
		t.Errorf("the oldest-current as-of line is missing; body: %s", page)
	}
	if iso := oldest.UTC().Format(time.RFC3339); !strings.Contains(page, iso) {
		t.Errorf("the as-of ISO instant %q is missing; body: %s", iso, page)
	}
	// The aged-out addresses mint no operator message — they never appear on the page.
	for _, aged := range []string{"203.0.113.98", "203.0.113.99"} {
		if strings.Contains(page, aged) {
			t.Errorf("an aged-out address (%s) surfaced; the trailing-edge Gap must fold to declared/current, never a per-address message; body: %s", aged, page)
		}
	}
	// No blanket responder and no unavailable vantage, so the currency card stays at its
	// empty state — the lag is the meter's figure, not a message.
	if !strings.Contains(page, "No coverage messages") {
		t.Errorf("a lagging scope must not add a coverage message; body: %s", page)
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
