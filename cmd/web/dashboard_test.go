package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// The Dashboard renders full parity with Dashboard.jsx (P2.1, #447): the framed
// five-cell stat band, the by-severity bars, the census Coverage card, the Vantages
// card with its per-vantage latency reading (P0.5, #485), and the most-recent
// Signals register — and neither of the two re-skinned empty-state holds this
// ticket deletes.
func TestDashboardParityRegions(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// A non-empty estate so `/` renders the Dashboard, not the first-run checklist.
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	// A provisioned vantage with a measured connect latency (P0.5) so the Vantages
	// card renders a row with the spec's mono "34ms" reading, not the no-vantage
	// empty state and not a fabricated number.
	f.vantages = append(f.vantages, db.Vantage{
		ID: f.vantageNextID, Name: "eu-west-1", Class: "internet",
		Host:         pgtype.Text{String: "prober.example.com", Valid: true},
		Port:         pgtype.Int4{Int32: 22, Valid: true},
		Username:     pgtype.Text{String: "verge", Valid: true},
		Availability: pgtype.Text{String: "available", Valid: true},
		LatencyMs:    pgtype.Int4{Int32: 34, Valid: true},
		CreatedBy:    pgtype.Int8{Int64: admin.ID, Valid: true},
	})
	f.vantageNextID++
	// A second provisioned vantage whose prober has not been reached yet: latency
	// is still NULL, so its reading is the spec's pending em dash, never a number.
	f.vantages = append(f.vantages, db.Vantage{
		ID: f.vantageNextID, Name: "us-east-2", Class: "internet",
		Host:         pgtype.Text{String: "prober2.example.com", Valid: true},
		Port:         pgtype.Int4{Int32: 22, Valid: true},
		Username:     pgtype.Text{String: "verge", Valid: true},
		Availability: pgtype.Text{String: "pending", Valid: true},
		CreatedBy:    pgtype.Int8{Int64: admin.ID, Valid: true},
	})
	f.vantageNextID++

	base := start(t, f, "")
	c := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, c, base+"/", http.StatusOK)

	// The spec's regions render: the framed stat band with all five cells, the
	// by-severity ramp, the coverage/vantage cards, and the most-recent register.
	for _, want := range []string{
		"db-statgrid",  // one framed card, five Stat cells (frozen dashboard.tmpl)
		"Open signals", // stat cell labels
		"Critical",
		"Assets watched",
		"Exposed services",
		"Certs expiring ≤30d",
		"By severity",         // the real severity bars
		"Scan infrastructure", // the Vantages card
		"eu-west-1",           // the provisioned vantage
		"34ms",                // its measured connect latency renders the real value
		"us-east-2",           // the unmeasured vantage
		"Most recent",         // the most-recent Signals register
	} {
		if !strings.Contains(page, want) {
			t.Errorf("dashboard missing parity region %q", want)
		}
	}

	// The unmeasured vantage renders the spec's pending em dash (the frozen tmpl emits
	// the literal — glyph, not the old &#8212; entity), and the retired latency Skeleton
	// placeholder is gone — a real value or a dash, never a fabricated number.
	if !strings.Contains(page, "—") {
		t.Error("dashboard did not render the pending em dash for the unmeasured vantage latency")
	}
	if strings.Contains(page, "dash-skel") {
		t.Error("dashboard still renders the retired latency Skeleton placeholder")
	}

	// The two re-skinned empty-state holds this ticket deletes are gone, along with
	// the loose .kpi tiles and the per-rule firing table they replaced.
	for _, forbidden := range []string{
		"Signals carry no severity",
		"Coverage detail is on its own screen",
		"Firing now, by rule",
		"class=\"kpi-num\"", // the loose .kpi tiles are gone (markup, not the shared CSS rule)
	} {
		if strings.Contains(page, forbidden) {
			t.Errorf("dashboard still renders the deleted region %q", forbidden)
		}
	}
}
