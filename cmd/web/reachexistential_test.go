package main

import (
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/exposure"
)

const reachFoldService = "198.51.100.1:443/tcp"

func reachFoldCovered() func(netip.Addr) bool {
	return custody.Estate{AddressScopes: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}}.CoversAddressScope
}

func reachFoldRow(id int64, dialled, value string, isGap bool, openedAt time.Time) reachLegRow {
	return reachLegRow{
		subject: reachFoldService, dialled: dialled, value: []byte(value),
		isGap: isGap, openedAt: openedAt, id: id,
	}
}

func reachFoldLeg(class string, rows ...reachLegRow) legInfo {
	return collapseReachLegs(rows, reachFoldCovered())[reachFoldService][class]
}

var (
	reachFoldOlder = time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	reachFoldNewer = time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC)
)

func TestReachFoldComposesOneClassExistentially(t *testing.T) {
	// One vantage of the class reaching settles the leg, whatever a sibling saw later (ADR-0080).
	leg := reachFoldLeg("internet",
		reachFoldRow(1, "198.51.100.10", `{"outcome":"reached","result":"open"}`, false, reachFoldOlder),
		reachFoldRow(2, "198.51.100.11", `{"outcome":"not-reached"}`, false, reachFoldNewer))

	if got := legFrom(leg); got != (exposure.Leg{Status: exposure.LegValued, Value: exposure.Reached}) {
		t.Fatalf("internet leg = %+v, want a valued reached", got)
	}
	if !leg.since.Equal(reachFoldOlder) {
		t.Errorf("leg date = %v, want the earliest span carrying reached (%v)", leg.since, reachFoldOlder)
	}
}

func TestReachFoldNotReachedOnlyWhereNoVantageReached(t *testing.T) {
	leg := reachFoldLeg("internet",
		reachFoldRow(1, "198.51.100.10", `{"outcome":"not-reached"}`, false, reachFoldOlder),
		reachFoldRow(2, "198.51.100.11", `{"outcome":"not-reached"}`, false, reachFoldNewer))

	if got := legFrom(leg); got != (exposure.Leg{Status: exposure.LegValued, Value: exposure.NotReached}) {
		t.Fatalf("internet leg = %+v, want a valued not-reached", got)
	}
	if !leg.since.Equal(reachFoldOlder) {
		t.Errorf("leg date = %v, want the earliest span carrying not-reached (%v)", leg.since, reachFoldOlder)
	}
}

func TestReachFoldGappedVantageDoesNotMaskASiblingReach(t *testing.T) {
	// A Gap span holds no value, so it is not a term the quantifier ranges over (ADR-0014).
	leg := reachFoldLeg("internet",
		reachFoldRow(1, "198.51.100.10", `{"outcome":"reached","result":"open"}`, false, reachFoldOlder),
		reachFoldRow(2, "198.51.100.11", `{"outcome":"gap","reason":"an edge answers for the origin"}`, true, reachFoldNewer))

	if got := legFrom(leg); got != (exposure.Leg{Status: exposure.LegValued, Value: exposure.Reached}) {
		t.Fatalf("internet leg = %+v, want a valued reached", got)
	}
	if !leg.since.Equal(reachFoldOlder) {
		t.Errorf("leg date = %v, want the reaching span's date (%v)", leg.since, reachFoldOlder)
	}
}

func TestReachFoldGapsWhereNoVantageOfTheClassDecided(t *testing.T) {
	leg := reachFoldLeg("internet",
		reachFoldRow(1, "198.51.100.10", `{"outcome":"gap","reason":"the control probe did not complete"}`, true, reachFoldOlder),
		reachFoldRow(2, "198.51.100.11", `{"outcome":"gap","reason":"an edge answers for the origin"}`, true, reachFoldNewer))

	if got := legFrom(leg); got != (exposure.Leg{Status: exposure.LegGap}) {
		t.Fatalf("internet leg = %+v, want a Gap", got)
	}
	if !leg.since.Equal(reachFoldOlder) {
		t.Errorf("leg date = %v, want the earliest gapped span (%v)", leg.since, reachFoldOlder)
	}
	want := []string{"the control probe did not complete", "an edge answers for the origin"}
	if len(leg.reasons) != 2 || leg.reasons[0] != want[0] || leg.reasons[1] != want[1] {
		t.Errorf("leg reasons = %q, want %q", leg.reasons, want)
	}
}

func TestReachFoldClassWithNoSpanStaysNeverConfigured(t *testing.T) {
	leg := reachFoldLeg("internal",
		reachFoldRow(1, "198.51.100.10", `{"outcome":"reached","result":"open"}`, false, reachFoldOlder))

	if got := legFrom(leg); got != (exposure.Leg{Status: exposure.LegNeverConfigured}) {
		t.Fatalf("internal leg = %+v, want never configured", got)
	}
}

func TestServiceReachCardInternetLegComposesExistentially(t *testing.T) {
	older := obsClock.Add(-168 * time.Hour)
	f := legProbeStore(t)
	a := f.addVantagePresenting("internet-a", "198.51.100.10")
	b := f.addVantagePresenting("internet-b", "198.51.100.11")
	f.addReachabilityAtVantage(t, legProbeService, a, older, `{"outcome":"reached","result":"open"}`)
	f.addReachabilityAtVantage(t, legProbeService, b, obsClock, `{"outcome":"not-reached"}`)

	page := serviceDetailBody(t, f, http.StatusOK)

	if got := reachCardLeg(t, page, "Internet leg"); got != (legChip{Tone: "danger", Label: "reached"}) {
		t.Fatalf("internet leg = %+v, want a danger reached", got)
	}
	assertReachCardLegDate(t, page, "Internet leg", older)
}

func TestServiceReachCardGapBannerStatesEveryVantageReason(t *testing.T) {
	f := legProbeStore(t)
	a := f.addVantagePresenting("internet-a", "198.51.100.10")
	b := f.addVantagePresenting("internet-b", "198.51.100.11")
	f.addReachabilityAtVantage(t, legProbeService, a, obsClock, `{"outcome":"gap","reason":"the control probe did not complete"}`)
	f.addReachabilityAtVantage(t, legProbeService, b, obsClock, `{"outcome":"gap","reason":"an edge answers for the origin"}`)

	card := reachCard(t, serviceDetailBody(t, f, http.StatusOK))

	// An existential composition names no vantage, so the drill-down carries every cause (ADR-0080).
	for _, want := range []string{"the control probe did not complete", "an edge answers for the origin"} {
		if !strings.Contains(card, want) {
			t.Errorf("the Gap banner is missing %q; card: %s", want, card)
		}
	}
}

func TestExposureBoardReadsAnExistentialInternetLeg(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	const svc = "198.51.100.10:443/tcp"
	now := time.Now().UTC()

	a := addInternetProber(f, admin.ID, "internet-a", "198.51.100.200")
	b := addInternetProber(f, admin.ID, "internet-b", "198.51.100.201")
	f.addReachabilityAtVantage(t, svc, a, now.Add(-168*time.Hour), `{"outcome":"reached","result":"open"}`)
	f.addReachabilityAtVantage(t, svc, b, now, `{"outcome":"not-reached"}`)
	f.addClassReachability(t, svc, "internal", now, `{"outcome":"reached"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	got := getBody(t, ac, base+"/exposure", http.StatusOK)

	// A service one internet vantage reaches is exposed, whatever a later sibling saw (ADR-0080).
	if !strings.Contains(got, "Exposed to internet") {
		t.Fatalf("the board does not read exposed; body: %s", got)
	}
}

func addInternetProber(f *fakeStore, adminID int64, name, dialled string) int64 {
	v := db.Vantage{
		ID: f.vantageNextID, Name: name, Class: "internet",
		Host:        pgtype.Text{String: name + ".example.com", Valid: true},
		Port:        pgtype.Int4{Int32: 22, Valid: true},
		Username:    pgtype.Text{String: "verge", Valid: true},
		DialledAddr: pgtype.Text{String: dialled, Valid: true},
		CreatedBy:   pgtype.Int8{Int64: adminID, Valid: true},
	}
	f.vantages = append(f.vantages, v)
	f.vantageNextID++
	return v.ID
}
