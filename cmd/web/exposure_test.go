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
		"198.51.100.10", ":443 tcp", "Exposed to internet",
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

var exposureSinceCell = regexp.MustCompile(`<td class="mono ex-since"[^>]*>([^<]*)</td>`)

func exposureSinceCells(t *testing.T, page string) []string {
	t.Helper()
	var cells []string
	for _, m := range exposureSinceCell.FindAllStringSubmatch(page, -1) {
		cells = append(cells, m[1])
	}
	if len(cells) == 0 {
		t.Fatalf("the service table rendered no Since cell; body: %s", page)
	}
	return cells
}

func TestExposureSinceIsUnknownWhenTheOpenSpansReadFails(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedInternetVantage(t, f, admin)

	at := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	for _, svc := range []string{"198.51.100.10:443/tcp", "198.51.100.11:22/tcp"} {
		f.addClassReachability(t, svc, "internal", at, `{"outcome":"reached"}`)
		f.addClassReachability(t, svc, "internet", at, `{"outcome":"reached"}`)
	}
	f.openSpansErr = errors.New("open spans read failed")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/exposure", http.StatusOK)
	for _, want := range []string{"Service exposure", "198.51.100.10", "198.51.100.11", "Exposed to internet"} {
		if !strings.Contains(page, want) {
			t.Fatalf("a failed open-spans read took down the board; missing %q; body: %s", want, page)
		}
	}
	if !strings.Contains(page, "Since is unknown") {
		t.Errorf("the card head does not say the open-spans read failed; body: %s", page)
	}
	cells := exposureSinceCells(t, page)
	if len(cells) != 2 {
		t.Fatalf("Since cells = %d, want 2; body: %s", len(cells), page)
	}
	for _, got := range cells {
		if got != exposureSinceUnknown {
			t.Errorf("Since cell = %q, want the unknown token %q", got, exposureSinceUnknown)
		}
	}
}

func TestExposureSinceDatesEveryRowWhenTheOpenSpansReadSucceeds(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedInternetVantage(t, f, admin)

	at := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	for _, svc := range []string{"198.51.100.10:443/tcp", "198.51.100.11:22/tcp"} {
		f.addClassReachability(t, svc, "internal", at, `{"outcome":"reached"}`)
		f.addClassReachability(t, svc, "internet", at, `{"outcome":"reached"}`)
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/exposure", http.StatusOK)
	if strings.Contains(page, "Since is unknown") {
		t.Errorf("a healthy open-spans read still declared the column unknown; body: %s", page)
	}
	want := []string{"2026-08-01", "2026-08-01"}
	if got := exposureSinceCells(t, page); !slices.Equal(got, want) {
		t.Errorf("Since cells = %q, want %q; body: %s", got, want, page)
	}
}

func TestExposureSinceNoteStaysAwayFromTheEmptyState(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedInternetVantage(t, f, admin)
	f.openSpansErr = errors.New("open spans read failed")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/exposure", http.StatusOK)
	if !strings.Contains(page, "No service exposure measured yet") {
		t.Fatalf("the empty state did not render; body: %s", page)
	}
	if strings.Contains(page, "Since is unknown") {
		t.Errorf("the card head names a column the empty state does not draw; body: %s", page)
	}
}

func TestSinceDisplaySeparatesAbsentFromUnknown(t *testing.T) {
	// Both store reads gate on the same open spans, so no page fixture is absent (#1947).
	for _, tc := range []struct {
		name    string
		since   string
		unknown bool
		want    string
	}{
		{name: "a dated open span", since: "2026-08-01", want: "2026-08-01"},
		{name: "no open span", since: "", want: exposureSinceAbsent},
		{name: "a failed read", since: "", unknown: true, want: exposureSinceUnknown},
		{name: "a failed read outranks a stale date", since: "2026-08-01", unknown: true, want: exposureSinceUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := sinceDisplay(tc.since, tc.unknown); got != tc.want {
				t.Errorf("sinceDisplay(%q, %v) = %q, want %q", tc.since, tc.unknown, got, tc.want)
			}
		})
	}
}
