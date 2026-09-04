package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
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
