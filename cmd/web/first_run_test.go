package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func TestFirstRunEmptyEstateRendersChecklist(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	c := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, c, base+"/", http.StatusOK)

	for _, want := range []string{
		"Welcome to Verge",
		"0 of 4 complete",
		"Declare your domain",
		"Upload a zone file",
		"Add an internet vantage",
		"Run the first batch",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("first-run checklist missing %q; body: %s", want, page)
		}
	}

	if !strings.Contains(page, "Needs an internet vantage first") {
		t.Fatalf("gated step 4 must name its gate; body: %s", page)
	}
	if !strings.Contains(page, "Step 4 stays gated until an internet vantage exists") {
		t.Fatalf("first-run footer must state the gate; body: %s", page)
	}
	if strings.Contains(page, `<a class="btn" href="/scans">Run first batch</a>`) {
		t.Fatalf("gated step 4 must not offer a live run action; body: %s", page)
	}

	for _, forbidden := range []string{"By severity", "Scan infrastructure"} {
		if strings.Contains(page, forbidden) {
			t.Fatalf("empty estate should not render the Dashboard region %q; body: %s", forbidden, page)
		}
	}
}

func TestFirstRunStepsReflectStateAndUngate(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.vantages = append(f.vantages, db.Vantage{
		ID: f.vantageNextID, Name: "internet-prober", Class: "internet",
		Host:        pgtype.Text{String: "prober.example.com", Valid: true},
		Port:        pgtype.Int4{Int32: 22, Valid: true},
		Username:    pgtype.Text{String: "verge", Valid: true},
		DialledAddr: classPresentedDialled("internet"),
		CreatedBy:   pgtype.Int8{Int64: admin.ID, Valid: true},
	})
	f.vantageNextID++

	base := start(t, f, "")
	c := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, c, base+"/", http.StatusOK)

	for _, want := range []string{
		"2 of 4 complete",
		"example.com declared",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("first-run checklist missing real-state %q; body: %s", want, page)
		}
	}

	if !strings.Contains(page, `action="/onboarding/finish"`) || !strings.Contains(page, "Run first batch") {
		t.Fatalf("step 4 should ungate into a live run POST once an internet vantage exists; body: %s", page)
	}
	if strings.Contains(page, "Needs an internet vantage first") {
		t.Fatalf("step 4 should not name the gate once an internet vantage exists; body: %s", page)
	}
}

func TestFirstRunHiddenForNonEmptyEstate(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)

	base := start(t, f, "")
	c := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, c, base+"/", http.StatusOK)

	if strings.Contains(page, "Welcome to Verge") {
		t.Fatalf("non-empty estate should render the Dashboard, not the first-run checklist; body: %s", page)
	}
	if !strings.Contains(page, "Open signals") {
		t.Fatalf("non-empty estate should render the Dashboard; body: %s", page)
	}
}
