package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// /exposure is repurposed from the #286 redirect-to-/reports into the first-class
// Exposure page (#300, T5). With no internet vantage the page renders the WITHHELD
// state, which must NAME its cause — no internet vantage — rather than 500 or fall
// back to a redirect. It is a rendered state, not an error.
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

// With an internet vantage provisioned and both reach legs concluded, the page
// renders the both-legs table: a Service per row with its internal and internet
// legs side by side, and the summary band.
func TestExposureBothLegsTable(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	// A provisioned internet prober — this is what "an internet vantage exists" means
	// (probers.go), so the board renders instead of the WITHHELD state.
	f.vantages = append(f.vantages, db.Vantage{
		ID: f.vantageNextID, Name: "internet-prober", Class: "internet",
		Host:      pgtype.Text{String: "prober.example.com", Valid: true},
		Port:      pgtype.Int4{Int32: 22, Valid: true},
		Username:  pgtype.Text{String: "verge", Valid: true},
		CreatedBy: pgtype.Int8{Int64: admin.ID, Valid: true},
	})
	f.vantageNextID++

	// Both legs conclude `reached` for one Service — an Exposed derivation.
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
	// The WITHHELD state must NOT show while an internet vantage exists.
	if strings.Contains(got, "Exposure withheld.") {
		t.Fatalf("board render still shows the WITHHELD state; body: %s", got)
	}
	// A single batch has no previous batch to compare against (HasDeltas false), so
	// the stat band shows no delta chip — the spec's own no-delta pattern, never a
	// fabricated +0.
	if strings.Contains(got, `class="exdelta `) {
		t.Fatalf("stat band rendered a delta chip with only one batch; body: %s", got)
	}
}

// The stat band renders the spec's vs-last-batch delta chip on the exposed tile
// (P0.2 #443, P2.6 #452, Exposure.jsx): a Service firewalled at the previous batch
// and exposed now moves exposure +1, so the exposed tile shows a "+1" chip in the
// danger tone (a rise in exposure is bad). Only the exposed tile is chipped, matching
// the spec's stat band — firewalled and not-reached carry no delta.
func TestExposureStatBandRendersDelta(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	// A provisioned (host-set) internet prober — what the WITHHELD gate reads as "an
	// internet vantage exists" (ListVantages is scoped to host-set probers), so the
	// board renders instead of the WITHHELD state.
	f.vantages = append(f.vantages, db.Vantage{
		ID: f.vantageNextID, Name: "internet-prober", Class: "internet",
		Host:      pgtype.Text{String: "prober.example.com", Valid: true},
		Port:      pgtype.Int4{Int32: 22, Valid: true},
		Username:  pgtype.Text{String: "verge", Valid: true},
		CreatedBy: pgtype.Int8{Int64: admin.ID, Valid: true},
	})
	f.vantageNextID++

	base0 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	t0 := base0
	t1 := base0.Add(1 * time.Hour)
	const svc = "198.51.100.10:443/tcp"
	// Previous batch (t0): internal reached, internet not-reached — firewalled.
	f.addClassReachability(t, svc, "internal", t0, `{"outcome":"reached"}`)
	f.addClassReachability(t, svc, "internet", t0, `{"outcome":"not-reached"}`)
	// Latest batch (t1): internet flips to reached — now exposed (internal span holds).
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
	// The exposed tile carries a signed delta chip in the danger tone.
	if !strings.Contains(got, `class="exdelta bad"`) {
		t.Fatalf("exposed tile missing the bad-tone delta chip; body: %s", got)
	}
	if !strings.Contains(got, "+1") {
		t.Fatalf("exposed delta chip missing the +1 movement; body: %s", got)
	}
	// Only the exposed tile is chipped — firewalled and not-reached carry no delta.
	if n := strings.Count(got, `class="exdelta `); n != 1 {
		t.Fatalf("stat band rendered %d delta chips, want exactly 1 (exposed tile only); body: %s", n, got)
	}
}

// An unauthenticated hit lands on /login — the page keeps its login gate.
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
