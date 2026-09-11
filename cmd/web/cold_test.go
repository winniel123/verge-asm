package main

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/winniel123/verge-asm/internal/db"
)

func setColdScope(t *testing.T, c *http.Client, base string, id int64, optIn bool) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/settings/cold", url.Values{
		"id": {strconv.FormatInt(id, 10)}, "opt_in": {strconv.FormatBool(optIn)},
	})
}

func coldBody(t *testing.T, c *http.Client, base string) string {
	t.Helper()
	return getBody(t, c, base+"/settings?tab=scans", http.StatusOK)
}

func coldScanEnabled(f *fakeStore) bool {
	for _, sc := range f.scans {
		if sc.Kind == "cold" {
			return sc.Enabled
		}
	}
	return false
}

func TestColdScanOptInEnablesPerScope(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	id := f.seeds[0].ID

	if coldScanEnabled(f) {
		t.Fatalf("cold Scan enabled on a fresh install, want disabled")
	}
	page := coldBody(t, ac, base)
	if !strings.Contains(page, "tier off") || !strings.Contains(page, "Opt in") {
		t.Errorf("cold tier not shown off with an opt-in control; body: %s", page)
	}

	resp := setColdScope(t, ac, base, id, true)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/settings?tab=scans" {
		t.Fatalf("opt in cold scope: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	if !f.coldScopes[id] {
		t.Fatalf("scope not recorded as opted into the cold tier")
	}
	if !coldScanEnabled(f) {
		t.Fatalf("cold Scan not enabled after a scope opted in")
	}
	if len(f.batches) != 0 || len(f.observations) != 0 {
		t.Fatalf("opting a scope in fired a scan on config save: %d batches, %d observations",
			len(f.batches), len(f.observations))
	}

	page = coldBody(t, ac, base)
	if !strings.Contains(page, "tier on") || !strings.Contains(page, "opted in") || !strings.Contains(page, "Opt out") {
		t.Errorf("opted-in tier not reflected with an opt-out control; body: %s", page)
	}

	setColdScope(t, ac, base, id, false).Body.Close()
	if f.coldScopes[id] {
		t.Fatalf("scope not removed from the cold tier on opt-out")
	}
	if coldScanEnabled(f) {
		t.Fatalf("cold Scan still enabled after the last scope opted out")
	}
}

func TestColdScanOptInAddressScope(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "address", "203.0.113.0/24").Body.Close()
	id := f.seeds[0].ID

	setColdScope(t, ac, base, id, true).Body.Close()
	if !f.coldScopes[id] || !coldScanEnabled(f) {
		t.Fatalf("address scope did not opt into the cold tier")
	}
}

func TestViewerCannotOptIntoColdTier(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	ac := login(t, base, "admin", "hunter2hunter2")
	declare(t, ac, base, "name", "example.com").Body.Close()
	id := f.seeds[0].ID

	vc := login(t, base, "viewer", "hunter2hunter2")
	resp := setColdScope(t, vc, base, id, true)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer opt in cold scope: status=%d, want 403", resp.StatusCode)
	}
	if f.coldScopes[id] || coldScanEnabled(f) {
		t.Fatalf("viewer's denied act still enabled the cold tier")
	}

	resp = get(t, vc, base+"/settings?tab=scans")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("viewer reached the admin-only cold-tier surface: status=%d, want 403", resp.StatusCode)
	}
}

func TestSetColdScopeRequiresLogin(t *testing.T) {
	base := start(t, newFakeStore(), "")
	c := newClient(t)
	resp := setColdScope(t, c, base, 1, true)
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon opt in: status=%d location=%q, want redirect to /login", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func (f *fakeStore) OptInColdScope(_ context.Context, arg db.OptInColdScopeParams) (db.OptInColdScopeRow, error) {
	// ON CONFLICT DO NOTHING returns no row, so a repeat opt-in enrols nothing.
	if f.coldScopes[arg.SeedID] {
		return db.OptInColdScopeRow{}, pgx.ErrNoRows
	}
	sd, ok := f.seedByID(arg.SeedID)
	if !ok {
		return db.OptInColdScopeRow{}, pgx.ErrNoRows
	}
	f.coldScopes[arg.SeedID] = true
	return db.OptInColdScopeRow{AddressCidr: sd.AddressCidr, NameDomain: sd.NameDomain}, nil
}

func (f *fakeStore) OptOutColdScope(_ context.Context, seedID int64) (db.OptOutColdScopeRow, error) {
	sd, ok := f.seedByID(seedID)
	if !f.coldScopes[seedID] || !ok {
		return db.OptOutColdScopeRow{}, pgx.ErrNoRows
	}
	delete(f.coldScopes, seedID)
	return db.OptOutColdScopeRow{AddressCidr: sd.AddressCidr, NameDomain: sd.NameDomain}, nil
}

func (f *fakeStore) seedByID(id int64) (db.Seed, bool) {
	for _, s := range f.seeds {
		if s.ID == id {
			return s, true
		}
	}
	return db.Seed{}, false
}

func (f *fakeStore) SyncColdScanEnabled(context.Context) error {
	enabled := len(f.coldScopes) > 0
	for i := range f.scans {
		if f.scans[i].Kind == "cold" {
			f.scans[i].Enabled = enabled
		}
	}
	return nil
}

func (f *fakeStore) ListBlanketedReachServices(_ context.Context) ([]string, error) {
	seen := map[string]struct{}{}
	for k, o := range f.currentReachByVantage() {
		if reachOutcomeIsGap(o.Value) {
			seen[k.svc] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for svc := range seen {
		out = append(out, svc)
	}
	sort.Strings(out)
	return out, nil
}
