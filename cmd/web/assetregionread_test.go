package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

const (
	certDidNotResolve = "The certificate read did not resolve on this load."
	citDidNotResolve  = "The citation read did not resolve on this load."
	dnsDidNotResolve  = "The dns-record read did not resolve on this load."

	certEmptyState = "No certificate-chain leaf holds a parsed value for this asset yet."
	citEmptyState  = "Nothing has cited this name into the estate yet."
	dnsEmptyState  = "This name holds no current resolution or dns-record value."

	kvCellSeed = `<span class="as-micro">Seed</span>`
	kvCellVia  = `<span class="as-micro">Via</span>`
)

func assetRegionFixture(t *testing.T, name string) (*fakeStore, string) {
	t.Helper()
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	// One address is what carries the certificate and DNS reads past their filters (#2029).
	f.addResolution(t, admin.ID, name, "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	return f, start(t, f, "")
}

func assetRegionPage(t *testing.T, f *fakeStore, base, name string) string {
	t.Helper()
	ac := login(t, base, "admin", "hunter2hunter2")
	return getBody(t, ac, base+"/asset/"+name, http.StatusOK)
}

func wantIn(t *testing.T, page, want, what string) {
	t.Helper()
	if !strings.Contains(page, want) {
		t.Errorf("%s: missing %q; body: %s", what, want, page)
	}
}

func wantNotIn(t *testing.T, page, banned, what string) {
	t.Helper()
	if strings.Contains(page, banned) {
		t.Errorf("%s: rendered %q; body: %s", what, banned, page)
	}
}

func TestAssetCertificateReadFailureRendersDidNotResolve(t *testing.T) {
	f, base := assetRegionFixture(t, "api.example.com")
	f.endpointCertsErr = errors.New("list endpoint certificates failed")

	page := assetRegionPage(t, f, base, "api.example.com")

	wantIn(t, page, certDidNotResolve, "failed certificate read")
	wantNotIn(t, page, certEmptyState, "failed certificate read")
	wantIn(t, page, "TLS certificate", "failed certificate read")
	// Every other region still renders, because a region read never fails the page (ADR-0168 §1).
	wantIn(t, page, "Open ports", "failed certificate read")
	wantIn(t, page, "198.51.100.1", "failed certificate read")
}

func TestAssetCitationReadFailureWithNoCoveringSeedRendersDidNotResolve(t *testing.T) {
	// A Name outside every declared scope leaves the provenance card with no other row (#2029).
	f, base := assetRegionFixture(t, "api.example.org")
	f.nameCitationErr = errors.New("get name citation failed")

	page := assetRegionPage(t, f, base, "api.example.org")

	wantIn(t, page, citDidNotResolve, "failed citation read, no covering seed")
	wantNotIn(t, page, citEmptyState, "failed citation read, no covering seed")
	// A bare label matches elsewhere on the page, so the row is anchored to its cell.
	wantNotIn(t, page, kvCellVia, "failed citation read, no covering seed")
}

func TestAssetCitationReadFailureWithCoveringSeedKeepsTheSeedRow(t *testing.T) {
	f, base := assetRegionFixture(t, "api.example.com")
	f.nameCitationErr = errors.New("get name citation failed")

	page := assetRegionPage(t, f, base, "api.example.com")

	// The Seed row comes from a second read that succeeded, so the note renders beside it.
	wantIn(t, page, kvCellSeed, "failed citation read, covering seed")
	wantIn(t, page, "example.com", "failed citation read, covering seed")
	wantIn(t, page, citDidNotResolve, "failed citation read, covering seed")
	wantNotIn(t, page, citEmptyState, "failed citation read, covering seed")
}

func TestAssetCoveringSeedReadFailureRendersTheProvenanceNote(t *testing.T) {
	f, base := assetRegionFixture(t, "api.example.com")
	f.coveringNameSeedErr = errors.New("find covering name seed failed")

	page := assetRegionPage(t, f, base, "api.example.com")

	wantIn(t, page, citDidNotResolve, "failed covering-seed read")
	wantNotIn(t, page, citEmptyState, "failed covering-seed read")
	// The citation read succeeded, so its own rows survive the seed read's failure.
	wantIn(t, page, kvCellVia, "failed covering-seed read")
}

func TestAssetSeedByIDReadFailureRendersTheProvenanceNote(t *testing.T) {
	f, base := assetRegionFixture(t, "api.example.com")
	// An admission carries a seed id, which is the only route into FindNameSeedByID (ADR-0107).
	f.addAdmittedName(t, "api.example.com", obsClock)
	f.nameSeedByIDErr = errors.New("find name seed by id failed")

	page := assetRegionPage(t, f, base, "api.example.com")

	wantIn(t, page, citDidNotResolve, "failed seed-by-id read")
	wantNotIn(t, page, citEmptyState, "failed seed-by-id read")
}

func TestAssetDNSReadFailureWithAddressesKeepsTheAddressRows(t *testing.T) {
	f, base := assetRegionFixture(t, "api.example.com")
	f.nameDNSErr = errors.New("list name dns records failed")

	page := assetRegionPage(t, f, base, "api.example.com")

	// The A row is built from the subject read, which already succeeded.
	wantIn(t, page, "198.51.100.1", "failed dns read, addresses present")
	wantIn(t, page, dnsDidNotResolve, "failed dns read, addresses present")
	wantNotIn(t, page, dnsEmptyState, "failed dns read, addresses present")
}

func TestAssetDNSReadFailureWithNoAddressesRendersDidNotResolve(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	// An empty address list leaves the panel with no row of its own, so it read as empty (#2029).
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":[]}`)
	base := start(t, f, "")
	f.nameDNSErr = errors.New("list name dns records failed")

	page := assetRegionPage(t, f, base, "api.example.com")

	wantIn(t, page, dnsDidNotResolve, "failed dns read, no addresses")
	wantNotIn(t, page, dnsEmptyState, "failed dns read, no addresses")
}

func TestWithdrawnAssetCitationReadFailureRendersDidNotResolve(t *testing.T) {
	f, base := withdrawnNameFixture(t)
	f.nameCitationErr = errors.New("get name citation failed")

	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/gone.example.com", http.StatusOK)

	wantIn(t, page, citDidNotResolve, "withdrawn asset, failed citation read")
	wantNotIn(t, page, citEmptyState, "withdrawn asset, failed citation read")
	wantIn(t, page, "no current member", "withdrawn asset, failed citation read")
}

func TestAssetEmptyRegionsKeepTheirOriginalEmptyStates(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.org", "dns", obsClock, `{"outcome":"Resolved","addresses":[]}`)
	base := start(t, f, "")
	// pgx.ErrNoRows is the Name that holds no citation, not a read that failed (ADR-0168 §4).
	f.nameCitationErr = pgx.ErrNoRows

	page := assetRegionPage(t, f, base, "api.example.org")

	for _, want := range []string{certEmptyState, citEmptyState, dnsEmptyState} {
		wantIn(t, page, want, "a Name that holds nothing")
	}
	for _, banned := range []string{certDidNotResolve, citDidNotResolve, dnsDidNotResolve} {
		wantNotIn(t, page, banned, "a Name that holds nothing")
	}
}

func TestAssetRegionReadFailuresNeverReachAnotherRegion(t *testing.T) {
	f, base := assetRegionFixture(t, "api.example.com")
	f.addDNSRecord(t, "api.example.com", "TXT", obsClock, `{"rrs":[{"name":"api.example.com","type":"TXT","data":"\"v=spf1 -all\""}]}`)
	f.addCertificate(t, "api.example.com@198.51.100.1:443/tcp", obsClock,
		`{"outcome":"presented","chain":["sha256:leaf01"],"not_after":"2027-03-01T12:00:00Z","issuer":"CN=R11","algorithm":"ECDSA-SHA256"}`)
	f.nameCitationErr = errors.New("get name citation failed")
	f.coveringNameSeedErr = errors.New("find covering name seed failed")

	page := assetRegionPage(t, f, base, "api.example.com")

	wantIn(t, page, citDidNotResolve, "failed provenance reads only")
	for _, banned := range []string{certDidNotResolve, dnsDidNotResolve} {
		wantNotIn(t, page, banned, "failed provenance reads only")
	}
	for _, want := range []string{"sha256:leaf01", "v=spf1 -all"} {
		wantIn(t, page, want, "failed provenance reads only")
	}
}
