package main

import (
	"net/http"
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
