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
		"primary · Cert Spotter",
		"last ct scan · Cert Spotter",
		"4213 names admitted",
		"under bar",
		"Runtime failover is not built",
		"detected",
		"≥ 99%", "≤ 5 s",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("degraded-primary hero missing %q; body: %s", want, page)
		}
	}
}

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
		"fallback · crt.sh",
		"bar-exempt",
		"not set",
		"VERGE_CERTSPOTTER_TOKEN",
		"12 names admitted",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("fallback hero missing %q; body: %s", want, page)
		}
	}
}

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
		"More CT capabilities",
		"drift tail",
		"last ct-tail scan",
		"37 names admitted",
		"verification",
		"812 certificates captured",
		"ephemeral event",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("capabilities card missing %q; body: %s", want, page)
		}
	}
}

func TestSourcesCTCapabilitiesCardEmpty(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := sourcesBody(t, ac, base)
	for _, want := range []string{
		"More CT capabilities",
		"awaiting captures",
		"0 certificates captured",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("empty capabilities card missing %q; body: %s", want, page)
		}
	}
	if strings.Contains(page, "last ct-tail scan") {
		t.Errorf("tail with no run must not render a run readout; body: %s", page)
	}
}

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

const sourcesTab = "/settings?tab=sources"

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

func TestSourcesModalRendersCatalogueAndDefaults(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := sourcesBody(t, ac, base)

	for _, c := range sourceCatalog {
		if !strings.Contains(page, c.Name) {
			t.Errorf("source %q missing from the modal", c.Name)
		}
	}
	for _, tier := range []string{"unencumbered", "operator-accepted", "barred"} {
		if !strings.Contains(page, tier) {
			t.Errorf("consent tier %q not rendered; body: %s", tier, page)
		}
	}
	if !strings.Contains(page, ">proposer<") || !strings.Contains(page, ">source<") {
		t.Errorf("source/proposer kinds not distinguished; body: %s", page)
	}
	if !strings.Contains(page, "crt.sh") || !strings.Contains(page, "RIPEstat") {
		t.Errorf("expected sources missing; body: %s", page)
	}
}

func TestLacnicCataloguedSourceNoConsent(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := sourcesBody(t, ac, base)
	if !strings.Contains(page, "LACNIC registry") {
		t.Fatalf("LACNIC not rendered; body: %s", page)
	}
	dlg := getBody(t, ac, base+"/sources?consent=lacnic-registry", http.StatusOK)
	if strings.Contains(dlg, "Nobody has been able to retrieve these terms.") {
		t.Errorf("LACNIC consent dialog rendered for a no-runner source; body: %s", dlg)
	}
	if strings.Contains(dlg, "Accept and enable") {
		t.Errorf("consent dialog opened for a catalogued-not-executing source; body: %s", dlg)
	}
}

func TestToggleSourcePersistsOverride(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/sources/toggle", url.Values{
		"slug": {"ripestat"}, "enabled": {"true"}, "agreed": {"on"},
	})
	if loc := submitLoc(t, resp); loc != sourcesTab {
		t.Fatalf("refused toggle landed at %q, want %q", loc, sourcesTab)
	}
	if got := getBody(t, ac, base+sourcesTab, http.StatusOK); !strings.Contains(got, "could not be found") {
		t.Fatalf("refused toggle showed no callout; body: %s", got)
	}
	if _, ok := f.sourceStates["ripestat"]; ok {
		t.Fatalf("no-runner source wrote state: %+v", f.sourceStates["ripestat"])
	}

	toggleSourceReq(t, ac, base, "arin", "false").Body.Close()
	if st, ok := f.sourceStates["arin"]; !ok || st.Enabled {
		t.Fatalf("disable override not persisted: %+v", f.sourceStates["arin"])
	}

	toggleSourceReq(t, ac, base, "crtsh", "false").Body.Close()
	if st, ok := f.sourceStates["crtsh"]; !ok || st.Enabled {
		t.Fatalf("crtsh disable override not persisted: %+v", f.sourceStates["crtsh"])
	}
}

func TestCataloguedSourceOffersNoEnable(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/sources", http.StatusOK)
	if strings.Contains(page, "consent=ripestat") {
		t.Errorf("catalogued source rendered a consent-gated enable; body: %s", page)
	}
	dlg := getBody(t, ac, base+"/sources?consent=ripestat", http.StatusOK)
	if strings.Contains(dlg, "Accept and enable") {
		t.Errorf("consent dialog opened for a catalogued source; body: %s", dlg)
	}
	const from = "/settings?tab=sources"
	resp := postForm(t, ac, base+"/settings/sources", url.Values{
		"id": {"ripestat"}, "enable": {"true"}, "return": {from},
	})
	if loc := submitLoc(t, resp); loc != from {
		t.Fatalf("catalogued enable landed at %q, want %q", loc, from)
	}
	if st, ok := f.sourceStates["ripestat"]; ok && st.Enabled {
		t.Fatalf("catalogued source enabled: %+v", st)
	}
	if land := getBody(t, ac, base+from, http.StatusOK); !strings.Contains(land, "could not be found") {
		t.Fatalf("the refusal is not on the landing page; body: %s", land)
	}
}

func TestToggleRejectsBarredAndUnknown(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	for _, slug := range []string{"hackertarget", "no-such-source"} {
		if loc := submitLoc(t, toggleSourceReq(t, ac, base, slug, "true")); loc != sourcesTab {
			t.Errorf("refused toggle %q landed at %q, want %q", slug, loc, sourcesTab)
		}
		if got := getBody(t, ac, base+sourcesTab, http.StatusOK); !strings.Contains(got, "could not be found") {
			t.Errorf("refused toggle %q showed no callout; body: %s", slug, got)
		}
	}
	if len(f.sourceStates) != 0 {
		t.Fatalf("no override should have been written; got %d", len(f.sourceStates))
	}
}

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

func TestCrtshShownAsExecuting(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := sourcesBody(t, ac, base)

	if !strings.Contains(page, "crt.sh") {
		t.Fatalf("crt.sh missing from the modal; body: %s", page)
	}
	if !strings.Contains(page, `value="crtsh"`) {
		t.Errorf("no toggle control offered for crt.sh; body: %s", page)
	}
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

func TestCoverageApertureMeters(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	declare(t, ac, base, "address", "203.0.113.0/24").Body.Close()

	page := coverageBody(t, ac, base)

	for _, want := range []string{
		"example.com", "203.0.113.0/24", "census", "addresses", "declared names",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("aperture meter is missing %q; body: %s", want, page)
		}
	}
}

func TestCoverageLaggingScopeAsOfNoStalenessMessage(t *testing.T) {
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startAt(t, f, now)
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "address", "203.0.113.0/24").Body.Close()

	oldest := now.Add(-30 * time.Hour)
	f.addReachability(t, "203.0.113.10:443/tcp", now.Add(-1*time.Hour), "reached")
	f.addReachability(t, "203.0.113.20:443/tcp", oldest, "reached")
	f.addReachability(t, "203.0.113.30:443/tcp", now.Add(-3*time.Hour), "reached")
	f.addReachability(t, "203.0.113.98:443/tcp", now.Add(-96*time.Hour), "reached")
	f.addReachability(t, "203.0.113.99:443/tcp", now.Add(-96*time.Hour), "reached")

	page := coverageBody(t, ac, base)

	if !strings.Contains(page, "3 / 256") {
		t.Errorf("lagging scope must show counted/total 3 / 256; body: %s", page)
	}
	if !strings.Contains(page, "oldest still current") {
		t.Errorf("the oldest-current as-of line is missing; body: %s", page)
	}
	if iso := oldest.UTC().Format(time.RFC3339); !strings.Contains(page, iso) {
		t.Errorf("the as-of ISO instant %q is missing; body: %s", iso, page)
	}
	for _, aged := range []string{"203.0.113.98", "203.0.113.99"} {
		if strings.Contains(page, aged) {
			t.Errorf("an aged-out address (%s) surfaced; the trailing-edge Gap must fold to declared/current, never a per-address message; body: %s", aged, page)
		}
	}
	if !strings.Contains(page, "No coverage messages") {
		t.Errorf("a lagging scope must not add a coverage message; body: %s", page)
	}
}

func TestCoverageSurfacesUnavailableVantages(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// A resolver-only vantage has no host, so the prober list drops it but its outage still shows.
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
	if !strings.Contains(page, "could not look") {
		t.Errorf("the unavailable-vantage message does not distinguish blindness from emptiness; body: %s", page)
	}
}

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

func TestCoverageHasNoProportionOfEstate(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)

	// The shared stylesheet's layout rules carry "100%", so a whole-page search would false-fail.
	main := page
	if i := strings.Index(page, "<main"); i >= 0 {
		main = page[i:]
	}
	for _, banned := range []string{"%", "estate completeness", "% covered", "% of your estate"} {
		if strings.Contains(main, banned) {
			t.Errorf("a proportion-of-estate figure appeared (%q); body: %s", banned, main)
		}
	}
}

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
	if !strings.Contains(page, `href="/scope"`) {
		t.Errorf("the aperture empty-state does not point at Scope; body: %s", page)
	}
}

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
