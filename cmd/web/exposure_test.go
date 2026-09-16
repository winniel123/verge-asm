package main

import (
	"errors"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func TestExposureWithheldNamesCause(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp, err := ac.Get(base + "/exposure")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /exposure: status = %d, want 200", resp.StatusCode)
	}
	for _, want := range []string{"Exposure withheld.", "No internet vantage exists.", "Provision a prober"} {
		if !strings.Contains(got, want) {
			t.Fatalf("withheld state missing %q; body: %s", want, got)
		}
	}
}

func TestExposureBothLegsTable(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	f.vantages = append(f.vantages, db.Vantage{
		ID: f.vantageNextID, Name: "internet-prober", Class: "internet",
		Host:        pgtype.Text{String: "prober.example.com", Valid: true},
		Port:        pgtype.Int4{Int32: 22, Valid: true},
		Username:    pgtype.Text{String: "verge", Valid: true},
		DialledAddr: classPresentedDialled("internet"),
		CreatedBy:   pgtype.Int8{Int64: admin.ID, Valid: true},
	})
	f.vantageNextID++

	now := time.Now().UTC()
	const svc = "198.51.100.10:443/tcp"
	f.addClassReachability(t, svc, "internal", now, `{"outcome":"reached"}`)
	f.addClassReachability(t, svc, "internet", now, `{"outcome":"reached"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp, err := ac.Get(base + "/exposure")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /exposure: status = %d, want 200", resp.StatusCode)
	}
	for _, want := range []string{
		"Both legs", "Service exposure", "Internal leg", "Internet leg",
		"198.51.100.10", ":443 tcp",
		"Exposed to internet", "Edge only", "Firewalled", "Unreachable", "One-legged",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("both-legs table missing %q; body: %s", want, got)
		}
	}
	if strings.Contains(got, "Exposure withheld.") {
		t.Fatalf("board render still shows the WITHHELD state; body: %s", got)
	}
	if strings.Contains(got, `class="ex-delta `) {
		t.Fatalf("stat band rendered a delta chip with only one batch; body: %s", got)
	}
}

func TestExposureStatBandRendersDelta(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	// Only a host-set prober counts as provisioned, so one with no Host leaves the
	// board WITHHELD and the fixture proves nothing.
	f.vantages = append(f.vantages, db.Vantage{
		ID: f.vantageNextID, Name: "internet-prober", Class: "internet",
		Host:        pgtype.Text{String: "prober.example.com", Valid: true},
		Port:        pgtype.Int4{Int32: 22, Valid: true},
		Username:    pgtype.Text{String: "verge", Valid: true},
		DialledAddr: classPresentedDialled("internet"),
		CreatedBy:   pgtype.Int8{Int64: admin.ID, Valid: true},
	})
	f.vantageNextID++

	base0 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	t0 := base0
	t1 := base0.Add(1 * time.Hour)
	const svc = "198.51.100.10:443/tcp"
	f.addClassReachability(t, svc, "internal", t0, `{"outcome":"reached"}`)
	f.addClassReachability(t, svc, "internet", t0, `{"outcome":"not-reached"}`)
	f.addClassReachability(t, svc, "internet", t1, `{"outcome":"reached"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	resp, err := ac.Get(base + "/exposure")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /exposure: status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(got, `class="ex-delta bad"`) {
		t.Fatalf("exposed tile missing the bad-tone delta chip; body: %s", got)
	}
	if !strings.Contains(got, "+1") {
		t.Fatalf("exposed delta chip missing the +1 movement; body: %s", got)
	}
	if n := strings.Count(got, `class="ex-delta `); n != 1 {
		t.Fatalf("stat band rendered %d delta chips, want exactly 1 (exposed tile only); body: %s", n, got)
	}
}

func TestExposureRequiresLogin(t *testing.T) {
	base := start(t, newFakeStore(), "")
	c := newClient(t)

	resp, err := c.Get(base + "/exposure")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon GET /exposure: status=%d location=%q, want redirect to /login",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestExposureFailsLoudlyWhenItsBoardReadFails(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	f.vantages = append(f.vantages, db.Vantage{
		ID: f.vantageNextID, Name: "internet-prober", Class: "internet",
		Host:        pgtype.Text{String: "prober.example.com", Valid: true},
		Port:        pgtype.Int4{Int32: 22, Valid: true},
		Username:    pgtype.Text{String: "verge", Valid: true},
		DialledAddr: classPresentedDialled("internet"),
		CreatedBy:   pgtype.Int8{Int64: admin.ID, Valid: true},
	})
	f.vantageNextID++
	f.reachSpansErr = errors.New("reachability read failed")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/exposure", http.StatusInternalServerError)
	if strings.Contains(page, "Service exposure") {
		t.Errorf("a failed board read rendered an empty board, which reads as 0 exposed; body: %s", page)
	}
}

func seedInternetVantage(t *testing.T, f *fakeStore, admin db.Account) {
	t.Helper()
	f.vantages = append(f.vantages, db.Vantage{
		ID: f.vantageNextID, Name: "internet-prober", Class: "internet",
		Host:        pgtype.Text{String: "prober.example.com", Valid: true},
		Port:        pgtype.Int4{Int32: 22, Valid: true},
		Username:    pgtype.Text{String: "verge", Valid: true},
		DialledAddr: classPresentedDialled("internet"),
		CreatedBy:   pgtype.Int8{Int64: admin.ID, Valid: true},
	})
	f.vantageNextID++
}

// The date is the chip's sibling, so this regex stays chip-only and servicedetailcard_test can
// still assert that a card draws none (#2149).

var legChipCell = regexp.MustCompile(`<span class="vg-leg ([a-z]+)">([^<]*)</span>`)

func legChips(t *testing.T, page string) []legChip {
	t.Helper()
	var chips []legChip
	for _, m := range legChipCell.FindAllStringSubmatch(page, -1) {
		chips = append(chips, legChip{Tone: m[1], Label: m[2]})
	}
	if len(chips) == 0 {
		t.Fatalf("the service table rendered no leg chip; body: %s", page)
	}
	return chips
}

var exposureLegCell = regexp.MustCompile(
	`<span class="ex-legcell"><span class="vg-leg ([a-z]+)">([^<]*)</span>` +
		`(?:<span class="ex-legdate">since ([^<]*)</span>)?</span>`)

func exposureLegCells(t *testing.T, page string) []legChip {
	t.Helper()
	var cells []legChip
	for _, m := range exposureLegCell.FindAllStringSubmatch(page, -1) {
		cells = append(cells, legChip{Tone: m[1], Label: m[2], Date: m[3]})
	}
	if len(cells) == 0 {
		t.Fatalf("the service table rendered no leg cell; body: %s", page)
	}
	return cells
}

func TestExposureLegToneKeysOnValueAndClass(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedInternetVantage(t, f, admin)

	at := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	const svc = "198.51.100.10:443/tcp"
	f.addClassReachability(t, svc, "internal", at, `{"outcome":"reached"}`)
	f.addClassReachability(t, svc, "internet", at, `{"outcome":"reached"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/exposure", http.StatusOK)
	want := []legChip{
		{Tone: "neutral", Label: "reached"},
		{Tone: "danger", Label: "reached"},
	}
	if got := legChips(t, page); !slices.Equal(got, want) {
		t.Errorf("leg chips = %+v, want %+v; body: %s", got, want, page)
	}
}

func TestExposureLegsNeverCarryAnExposureWord(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedInternetVantage(t, f, admin)

	at := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	f.addClassReachability(t, "198.51.100.10:443/tcp", "internal", at, `{"outcome":"reached"}`)
	f.addClassReachability(t, "198.51.100.10:443/tcp", "internet", at, `{"outcome":"not-reached"}`)
	f.addClassReachability(t, "198.51.100.11:22/tcp", "internal", at, `{"outcome":"not-reached"}`)
	f.addClassReachability(t, "198.51.100.11:22/tcp", "internet", at, `{"outcome":"reached"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/exposure", http.StatusOK)
	legWords := []string{"reached", "not reached", "never looked", "stopped looking"}
	for _, c := range legChips(t, page) {
		if !slices.Contains(legWords, c.Label) {
			t.Errorf("a leg column rendered %q, which is not one of %q", c.Label, legWords)
		}
	}
}

func TestExposureAbsentLegsKeepTheirTwoWords(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedInternetVantage(t, f, admin)

	at := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	// One Service was never looked at from the internet; the other stopped being looked at.
	f.addClassReachability(t, "198.51.100.10:443/tcp", "internal", at, `{"outcome":"reached"}`)
	f.addClassReachability(t, "198.51.100.11:22/tcp", "internal", at, `{"outcome":"reached"}`)
	f.addClassReachability(t, "198.51.100.11:22/tcp", "internet", at, `{"outcome":"gap"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/exposure", http.StatusOK)
	want := []legChip{
		{Tone: "neutral", Label: "reached"},
		{Tone: "absent", Label: "never looked"},
		{Tone: "neutral", Label: "reached"},
		{Tone: "warn", Label: "stopped looking"},
	}
	if got := legChips(t, page); !slices.Equal(got, want) {
		t.Fatalf("leg chips = %+v, want %+v; body: %s", got, want, page)
	}
}

// A date belongs to the value it sits beside, so each leg carries its own and the row carries
// none (#2034).

func TestExposureRowDatesEachLegBesideItsOwnChip(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedInternetVantage(t, f, admin)

	early := time.Date(2026, 8, 1, 9, 30, 0, 0, time.UTC)
	late := time.Date(2026, 8, 3, 9, 30, 0, 0, time.UTC)
	f.addClassReachability(t, "198.51.100.10:443/tcp", "internal", early, `{"outcome":"reached"}`)
	f.addClassReachability(t, "198.51.100.10:443/tcp", "internet", late, `{"outcome":"reached"}`)
	// A never-configured leg holds no value for a date to belong to.
	f.addClassReachability(t, "198.51.100.11:22/tcp", "internal", early, `{"outcome":"reached"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/exposure", http.StatusOK)
	want := []legChip{
		{Tone: "neutral", Label: "reached", Date: "2026-08-01"},
		{Tone: "danger", Label: "reached", Date: "2026-08-03"},
		{Tone: "neutral", Label: "reached", Date: "2026-08-01"},
		{Tone: "absent", Label: "never looked"},
	}
	if got := exposureLegCells(t, page); !slices.Equal(got, want) {
		t.Errorf("leg cells = %+v, want %+v; body: %s", got, want, page)
	}
}

// The column the date replaced was class-blind, and so was the banner that stated its absence
// (#2034).

func TestExposureDrawsNoClassBlindSinceColumn(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedInternetVantage(t, f, admin)

	at := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	f.addClassReachability(t, "198.51.100.10:443/tcp", "internal", at, `{"outcome":"reached"}`)
	f.addClassReachability(t, "198.51.100.10:443/tcp", "internet", at, `{"outcome":"reached"}`)
	// The board no longer reads the open spans at all, so their failure reaches no column.
	f.openSpansErr = errors.New("open spans read failed")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/exposure", http.StatusOK)
	if !strings.Contains(page, "Service exposure") {
		t.Fatalf("the board did not render; body: %s", page)
	}
	for _, refused := range []string{`class="mono ex-since"`, "Since is unknown"} {
		if strings.Contains(page, refused) {
			t.Errorf("the table still carries the class-blind Since column: %q", refused)
		}
	}
	if got := exposureLegCells(t, page); got[0].Date != "2026-08-01" {
		t.Errorf("a leg lost its own date with the column; cells: %+v", got)
	}
}
