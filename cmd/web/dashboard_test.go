package main

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/retention"
)

func TestDashboardParityRegions(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// An empty estate renders the first-run checklist at `/` instead of the Dashboard.
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.vantages = append(f.vantages, db.Vantage{
		ID: f.vantageNextID, Name: "eu-west-1", Class: "internet",
		Host:         pgtype.Text{String: "prober.example.com", Valid: true},
		Port:         pgtype.Int4{Int32: 22, Valid: true},
		Username:     pgtype.Text{String: "verge", Valid: true},
		Availability: pgtype.Text{String: "available", Valid: true},
		LatencyMs:    pgtype.Int4{Int32: 34, Valid: true},
		DialledAddr:  classPresentedDialled("internet"),
		CreatedBy:    pgtype.Int8{Int64: admin.ID, Valid: true},
	})
	f.vantageNextID++
	f.vantages = append(f.vantages, db.Vantage{
		ID: f.vantageNextID, Name: "us-east-2", Class: "internet",
		Host:         pgtype.Text{String: "prober2.example.com", Valid: true},
		Port:         pgtype.Int4{Int32: 22, Valid: true},
		Username:     pgtype.Text{String: "verge", Valid: true},
		Availability: pgtype.Text{String: "pending", Valid: true},
		DialledAddr:  classPresentedDialled("internet"),
		CreatedBy:    pgtype.Int8{Int64: admin.ID, Valid: true},
	})
	f.vantageNextID++

	base := start(t, f, "")
	c := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, c, base+"/", http.StatusOK)

	for _, want := range []string{
		"db-statgrid",
		"Open signals",
		"Critical",
		"Assets watched",
		"Exposed services",
		"Certs expiring ≤30d",
		"By severity",
		"Scan infrastructure",
		"eu-west-1",
		"34ms",
		"us-east-2",
		"Most recent",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("dashboard missing parity region %q", want)
		}
	}

	if !strings.Contains(page, "—") {
		t.Error("dashboard did not render the pending em dash for the unmeasured vantage latency")
	}
	if strings.Contains(page, "dash-skel") {
		t.Error("dashboard still renders the retired latency Skeleton placeholder")
	}

	for _, forbidden := range []string{
		"Signals carry no severity",
		"Coverage detail is on its own screen",
		"Firing now, by rule",
		"class=\"kpi-num\"",
	} {
		if strings.Contains(page, forbidden) {
			t.Errorf("dashboard still renders the deleted region %q", forbidden)
		}
	}
}

var dashStatCellRe = regexp.MustCompile(
	`(?s)<span class="lb">([^<]*)</span>\s*<span class="vr"><span class="num">([^<]*)</span>(.*?)</span>\s*<span class="cap">`)

func dashStatCell(t *testing.T, page, label string) (value, chip string) {
	t.Helper()
	for _, m := range dashStatCellRe.FindAllStringSubmatch(page, -1) {
		if m[1] == label {
			return m[2], m[3]
		}
	}
	t.Fatalf("dashboard has no %q stat cell; body: %s", label, page)
	return "", ""
}

func TestDashboardAssetsWatchedFoldsTheOpenSpanCorpus(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")

	// The fixed clock is 2026-08-15T12:00Z and the fake scan cadence is daily, so k=2 admits 48h.
	stale := time.Date(2026, 8, 11, 9, 0, 0, 0, time.UTC)
	prev := time.Date(2026, 8, 14, 8, 0, 0, 0, time.UTC)
	last := time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC)

	f.addResolution(t, admin.ID, "stale.example.com", "dns", stale, `{"outcome":"Resolved","addresses":["198.51.100.5"]}`)
	f.addResolution(t, admin.ID, "broken.example.com", "dns", prev, `{"outcome":"NameError"}`)
	f.addResolution(t, admin.ID, "api.example.com", "dns", last, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addClassReachability(t, "198.51.100.1:443/tcp", "internet", last, `{"outcome":"reached"}`)

	ctx := t.Context()
	prevAt, err := f.PreviousBatchTime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !prevAt.Valid || !prevAt.Time.Equal(prev) {
		t.Fatalf("previous batch instant = %v, want %v", prevAt, prev)
	}
	rows, err := f.ListSpansOpenSince(ctx, prevAt)
	if err != nil {
		t.Fatal(err)
	}
	assets := []drift.Span{}
	for _, r := range rows {
		if r.SubjectKind == "name" || r.SubjectKind == "service" {
			assets = append(assets, spanFromOpenSinceRow(r))
		}
	}
	want := drift.DistinctSubjects(drift.CurrentlyOpen(assets))
	wantPrev := drift.DistinctSubjects(drift.OpenAt(assets, prevAt.Time))
	if want != 4 || wantPrev != 2 {
		t.Fatalf("fixture fold = %d current / %d previous, want 4 / 2", want, wantPrev)
	}

	names, err := f.ListCurrentNameSubjects(ctx, db.ListCurrentNameSubjectsParams{
		Search: "", AsOf: pgtype.Timestamptz{Time: fixedClock()(), Valid: true}, FloorCadences: retention.FloorCadences,
	})
	if err != nil {
		t.Fatal(err)
	}
	services, err := f.ListCurrentServiceSubjects(ctx, db.ListCurrentServiceSubjectsParams{
		Search: "", AsOf: pgtype.Timestamptz{Time: fixedClock()(), Valid: true}, FloorCadences: retention.FloorCadences,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(names)+len(services) == want {
		t.Fatalf("fixture does not separate the two reads: listing sum = %d, fold = %d", len(names)+len(services), want)
	}

	base := start(t, f, "")
	c := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, c, base+"/", http.StatusOK)

	value, chip := dashStatCell(t, page, "Assets watched")
	if value != strconv.Itoa(want) {
		t.Errorf("assets watched value = %q, want %q (DistinctSubjects(CurrentlyOpen(spans)))", value, strconv.Itoa(want))
	}
	change := want - wantPrev
	if !strings.Contains(chip, "+"+strconv.Itoa(change)) {
		t.Errorf("assets watched delta chip = %q, want a +%d arrow", chip, change)
	}
	got, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("assets watched value %q is not a number: %v", value, err)
	}
	if got-change != wantPrev {
		t.Errorf("value minus delta = %d, want the previous-instant count %d", got-change, wantPrev)
	}
}

func TestDashboardAssetsWatchedWithheldWhenDeltasDegrade(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")

	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addClassReachability(t, "198.51.100.1:443/tcp", "internet", obsClock, `{"outcome":"reached"}`)

	if d := newServer(f, testKey, "", fixedClock()).dashboardDeltas(t.Context(), nil); d.Known {
		t.Fatal("dashboardDeltas Known = true, want false with a single batch instant")
	}

	base := start(t, f, "")
	c := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, c, base+"/", http.StatusOK)

	value, chip := dashStatCell(t, page, "Assets watched")
	if value != "—" {
		t.Errorf("assets watched value = %q, want the em dash when the fold is withheld", value)
	}
	if chip != "" {
		t.Errorf("assets watched chip = %q, want no delta beside a withheld value", chip)
	}
}
