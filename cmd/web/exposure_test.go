package main

import (
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// v1 is the shipped connect-outcome derivation vector; v2 is a bumped one, used
// to force a Break on the composing derivation.
const (
	reachV1 = `[{"leaf":"connect-outcome","version":"connect-outcome/v1"}]`
	reachV2 = `[{"leaf":"connect-outcome","version":"connect-outcome/v2"}]`
)

func reachRow(service, host string, vantageID int64, outcome, derivation string, rn int64) db.ListReachabilitySpansForExposureRow {
	return db.ListReachabilitySpansForExposureRow{
		SubjectKey:   service,
		VantageID:    pgtype.Int8{Int64: vantageID, Valid: true},
		Host:         pgtype.Text{String: host, Valid: true},
		Availability: pgtype.Text{String: "available", Valid: true},
		Value:        []byte(`{"outcome":"` + outcome + `"}`),
		Derivation:   []byte(derivation),
		OpenedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Rn:           rn,
	}
}

// addProber seeds an available prober vantage the install-presence check reads.
func addProber(f *fakeStore, id int64, host string, createdBy int64) {
	f.vantages = append(f.vantages, db.Vantage{
		ID: id, Name: host, Class: "unverified",
		Host:         pgtype.Text{String: host, Valid: true},
		Port:         pgtype.Int4{Int32: 22, Valid: true},
		Username:     pgtype.Text{String: "scanner", Valid: true},
		Availability: pgtype.Text{String: "available", Valid: true},
		CreatedBy:    pgtype.Int8{Int64: createdBy, Valid: true},
		CreatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
}

// A two-legged install (a public prober and one inside a declared address scope)
// renders every Exposure cell, the flagship "what moved" panel, a co-existing
// one-legged reading, and a co-existing "rules changed" break — all on one screen.
func TestExposurePopulatedBoardAndCoexistingPreconditions(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	// A declared address scope makes the 10.0.0.0/8 prober verify `internal`; the
	// public prober verifies `internet`.
	if _, err := f.CreateAddressSeed(t.Context(), db.CreateAddressSeedParams{
		AddressCidr: mustPrefix("10.0.0.0/8"), CreatedBy: admin.ID,
	}); err != nil {
		t.Fatal(err)
	}
	const inet, intl = int64(1), int64(2)
	addProber(f, inet, "203.0.113.9", admin.ID) // internet
	addProber(f, intl, "10.0.0.5", admin.ID)    // internal

	f.reachSpans = []db.ListReachabilitySpansForExposureRow{
		// exposed: both reached.
		reachRow("a:443/tcp", "203.0.113.9", inet, "reached", reachV1, 1),
		reachRow("a:443/tcp", "10.0.0.5", intl, "reached", reachV1, 1),
		// edge-only, and the flagship: internet not-reached -> reached.
		reachRow("b:8080/tcp", "203.0.113.9", inet, "reached", reachV1, 1),
		reachRow("b:8080/tcp", "203.0.113.9", inet, "not-reached", reachV1, 2),
		reachRow("b:8080/tcp", "10.0.0.5", intl, "not-reached", reachV1, 1),
		// firewalled: internet not-reached, internal reached.
		reachRow("c:22/tcp", "203.0.113.9", inet, "not-reached", reachV1, 1),
		reachRow("c:22/tcp", "10.0.0.5", intl, "reached", reachV1, 1),
		// unreachable: both not-reached.
		reachRow("d:9000/tcp", "203.0.113.9", inet, "not-reached", reachV1, 1),
		reachRow("d:9000/tcp", "10.0.0.5", intl, "not-reached", reachV1, 1),
		// one-legged: only the internal side has a value; the internet leg is a Gap.
		reachRow("e:6379/tcp", "10.0.0.5", intl, "reached", reachV1, 1),
		// rules changed: the composing derivation moved between the two spans.
		reachRow("brk:443/tcp", "203.0.113.9", inet, "reached", reachV2, 1),
		reachRow("brk:443/tcp", "203.0.113.9", inet, "reached", reachV1, 2),
		reachRow("brk:443/tcp", "10.0.0.5", intl, "reached", reachV1, 1),
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/exposure", http.StatusOK)

	// All four Exposure cells render their names and their members.
	for _, cell := range []string{"exposed", "edge-only", "firewalled", "unreachable"} {
		if !strings.Contains(page, ">"+cell+"<") {
			t.Errorf("board missing the %q cell; body: %s", cell, page)
		}
	}
	for _, svc := range []string{"a:443/tcp", "b:8080/tcp", "c:22/tcp", "d:9000/tcp"} {
		if !strings.Contains(page, svc) {
			t.Errorf("board missing service %q; body: %s", svc, page)
		}
	}

	// The flagship "what moved" panel names the internet not-reached -> reached move.
	if !strings.Contains(page, "What moved") || !strings.Contains(page, "became reachable from the internet") {
		t.Errorf("what-moved panel missing; body: %s", page)
	}

	// The one-legged reading co-exists, labelled "we stopped looking" (a Gap, not
	// never-configured), and NEVER a fifth exposure value.
	if !strings.Contains(page, "e:6379/tcp") || !strings.Contains(page, "we stopped looking") {
		t.Errorf("one-legged reading missing; body: %s", page)
	}
	if strings.Contains(page, "internal-only") {
		t.Errorf("a withdrawn fifth value must never render; body: %s", page)
	}

	// The Break precondition co-exists with the board and names the broken service.
	if !strings.Contains(page, "brk:443/tcp") || !strings.Contains(strings.ToLower(page), "rules changed") {
		t.Errorf("break precondition missing; body: %s", page)
	}
}

// The modal no-prober install: only an internal vantage. No Exposure is
// constructible, the surviving internal Reach renders under "we never looked",
// and the screen never manufactures a false internal-only reading.
func TestExposureNoProberRendersNeverLooked(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	if _, err := f.CreateAddressSeed(t.Context(), db.CreateAddressSeedParams{
		AddressCidr: mustPrefix("10.0.0.0/8"), CreatedBy: admin.ID,
	}); err != nil {
		t.Fatal(err)
	}
	const intl = int64(1)
	addProber(f, intl, "10.0.0.5", admin.ID) // internal only — no prober on the internet

	f.reachSpans = []db.ListReachabilitySpansForExposureRow{
		reachRow("svc:6379/tcp", "10.0.0.5", intl, "reached", reachV1, 1),
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/exposure", http.StatusOK)

	if !strings.Contains(page, "no exposure constructible") {
		t.Errorf("the no-exposure precondition must render; body: %s", page)
	}
	if !strings.Contains(page, "svc:6379/tcp") || !strings.Contains(page, "we never looked") {
		t.Errorf("the surviving internal leg must render under 'we never looked'; body: %s", page)
	}
	if strings.Contains(page, "internal-only") {
		t.Errorf("no false internal-only reading may appear; body: %s", page)
	}
}

// An estate with no Service at all renders the no-service precondition, not a
// blank grid.
func TestExposureNoServices(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/exposure", http.StatusOK)
	if !strings.Contains(page, "No service in your estate yet") {
		t.Errorf("the no-service precondition must render; body: %s", page)
	}
}

func mustPrefix(s string) *netip.Prefix {
	p := netip.MustParsePrefix(s)
	return &p
}
