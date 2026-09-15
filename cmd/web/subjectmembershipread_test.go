package main

import (
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

const (
	membershipDidNotResolve = "Membership did not resolve"
	withdrawnHeadline       = "Withdrawn by the world"
	withdrawnChip           = `class="sd-wmark"`
	rescanDisabled          = `<button class="sd-btn" disabled`
	rescanOffered           = `<a class="sd-btn" href="/scans"`
)

func membershipFixture(t *testing.T) (*fakeStore, string, db.Account) {
	t.Helper()
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addClassReachability(t, "203.0.113.99:5900/tcp", "internet", obsClock, `{"outcome":"reached","result":"open"}`)
	return f, start(t, f, ""), admin
}

func membershipPage(t *testing.T, base, path string) string {
	t.Helper()
	ac := login(t, base, "admin", "hunter2hunter2")
	return getBody(t, ac, base+path, http.StatusOK)
}

const serviceDetailPath = "/subjects/service?key=203.0.113.99%3A5900%2Ftcp"

func TestServiceMembershipReadFailureRendersTheNoteAndNotTheWithdrawalVerdict(t *testing.T) {
	f, base, _ := membershipFixture(t)
	f.nameCitingAddressErr = errors.New("find name citing address failed")
	f.coveringAddressSeedErr = errors.New("find covering address seed failed")

	page := membershipPage(t, base, serviceDetailPath)

	wantIn(t, page, membershipDidNotResolve, "failed membership reads")
	wantNotIn(t, page, withdrawnHeadline, "failed membership reads")
	wantNotIn(t, page, withdrawnChip, "failed membership reads")
	// An advisory read degrades its block and never refuses the act (ADR-0168 §3).
	wantIn(t, page, rescanOffered, "failed membership reads")
	wantNotIn(t, page, rescanDisabled, "failed membership reads")
	// The signals rail is gated on the withdrawal verdict, so a fault must not empty it.
	wantIn(t, page, "Signals here", "failed membership reads")
	wantIn(t, page, "sensitive-port-reached-from-internet", "failed membership reads")
}

func TestServiceMembershipNoRowsStaysTheWithdrawalVerdict(t *testing.T) {
	f, base, _ := membershipFixture(t)
	f.nameCitingAddressErr = pgx.ErrNoRows
	f.coveringAddressSeedErr = pgx.ErrNoRows

	page := membershipPage(t, base, serviceDetailPath)

	wantIn(t, page, withdrawnHeadline, "no rows at both membership reads")
	wantIn(t, page, withdrawnChip, "no rows at both membership reads")
	wantIn(t, page, rescanDisabled, "no rows at both membership reads")
	wantNotIn(t, page, membershipDidNotResolve, "no rows at both membership reads")
}

func TestServiceCoveringSeedSettlesMembershipThoughTheCitationReadFailed(t *testing.T) {
	f, base, admin := membershipFixture(t)
	scope := netip.MustParsePrefix("203.0.113.0/24")
	if _, err := f.CreateAddressSeed(t.Context(), db.CreateAddressSeedParams{
		AddressCidr: &scope, CreatedBy: pgtype.Int8{Int64: admin.ID, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	f.nameCitingAddressErr = errors.New("find name citing address failed")

	page := membershipPage(t, base, serviceDetailPath)

	// A Seed that covers the address settles membership, so the other read's failure claims nothing.
	wantNotIn(t, page, membershipDidNotResolve, "covering seed with a failed citation read")
	wantNotIn(t, page, withdrawnHeadline, "covering seed with a failed citation read")
	wantIn(t, page, "203.0.113.0/24", "covering seed with a failed citation read")
}

func TestEndpointMembershipReadFailureRendersTheNoteAndNotTheWithdrawalVerdict(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addHTTPIdentity(t, "api.example.com@203.0.113.99:443/tcp", obsClock,
		`{"outcome":"responded","status":200,"server":"nginx"}`)
	base := start(t, f, "")
	f.nameCitingAddressErr = errors.New("find name citing address failed")
	f.coveringAddressSeedErr = errors.New("find covering address seed failed")

	key := url.QueryEscape("api.example.com@203.0.113.99:443/tcp")
	page := membershipPage(t, base, "/subjects/endpoint?key="+key)

	wantIn(t, page, membershipDidNotResolve, "failed endpoint membership reads")
	wantNotIn(t, page, withdrawnHeadline, "failed endpoint membership reads")
	wantNotIn(t, page, withdrawnChip, "failed endpoint membership reads")
	wantIn(t, page, rescanOffered, "failed endpoint membership reads")
	wantNotIn(t, page, rescanDisabled, "failed endpoint membership reads")
	// The identity cell renders an em dash on a withdrawn endpoint, and the status here is read.
	wantIn(t, page, "200", "failed endpoint membership reads")
}

func TestAssetScopeDateReadFailureNotesTheHeaderChip(t *testing.T) {
	f, base := assetRegionFixture(t, "api.example.com")
	f.coveringNameSeedErr = errors.New("find covering name seed failed")

	page := assetRegionPage(t, f, base, "api.example.com")

	wantIn(t, page, "in scope since — the Seed read did not resolve", "failed scope-date read")
}

func TestAssetOutsideEveryScopeLeavesTheHeaderChipAbsent(t *testing.T) {
	// A Name no Seed covers is an honest absence, and it must not read as a failed read (#2050).
	f, base := assetRegionFixture(t, "api.example.org")

	page := assetRegionPage(t, f, base, "api.example.org")

	wantNotIn(t, page, "the Seed read did not resolve", "name outside every scope")
	wantNotIn(t, page, "in scope since", "name outside every scope")
}
