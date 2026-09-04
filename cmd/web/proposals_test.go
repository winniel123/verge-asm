package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/proposer"
)

type fakeProposer struct {
	candidates  []proposer.Candidate
	err         error
	lastQuery   string
	lastEnabled map[string]bool
	calls       int
}

func (p *fakeProposer) Propose(_ context.Context, org string, enabled map[string]bool) ([]proposer.Candidate, error) {
	p.calls++
	p.lastQuery = org
	p.lastEnabled = enabled
	return p.candidates, p.err
}

func startWithProposer(t *testing.T, f *fakeStore, p proposerRunner) string {
	t.Helper()
	srv := newServer(f, testKey, "", fixedClock())
	srv.proposer = p
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)
	return ts.URL
}

func lookup(t *testing.T, c *http.Client, base, query string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/proposals", url.Values{"query": {query}})
}

func get(t *testing.T, c *http.Client, rawURL string) *http.Response {
	t.Helper()
	resp, err := c.Get(rawURL)
	if err != nil {
		t.Fatalf("GET %s: %v", rawURL, err)
	}
	return resp
}

func twoCandidates() []proposer.Candidate {
	return []proposer.Candidate{
		{SourceSlug: proposer.SlugARIN, RecordKind: proposer.RecordRIRDelegation,
			Scope: netip.MustParsePrefix("203.0.113.0/24"), OrgName: "Example Org"},
		{SourceSlug: proposer.SlugARIN, RecordKind: proposer.RecordCompelledReassignment,
			Scope: netip.MustParsePrefix("198.51.100.8/29"), OrgName: "Renter LLC"},
	}
}

func TestLookupProducesProposalsNotSeeds(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{candidates: twoCandidates()}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := lookup(t, ac, base, "Example")
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("lookup status=%d, want 303", resp.StatusCode)
	}
	resp.Body.Close()

	if len(f.proposals) != 2 {
		t.Fatalf("proposals = %d, want 2", len(f.proposals))
	}
	if len(f.seeds) != 0 {
		t.Fatalf("seeds after lookup = %d, want 0 (a proposal is read by nothing)", len(f.seeds))
	}
	for _, p := range f.proposals {
		if p.Status != "pending" {
			t.Errorf("proposal %d status=%q, want pending", p.ID, p.Status)
		}
	}

	page := seedsBody(t, ac, base)
	for _, want := range []string{
		"203.0.113.0/24", "198.51.100.8/29",
		">Confirm<", "Decline selected",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("proposals section missing %q; body: %s", want, page)
		}
	}
}

func TestConfirmIsSingularWithNoBatchAffordance(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{candidates: twoCandidates()}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()

	page := seedsBody(t, ac, base)
	for _, forbidden := range []string{"Confirm all", "Confirm selected", `type="checkbox" name="confirm`} {
		if strings.Contains(page, forbidden) {
			t.Errorf("batch-confirm affordance present (%q); ADR-0022 forbids it", forbidden)
		}
	}

	confirmID := f.proposals[0].ID
	resp := postForm(t, ac, base+"/proposals/confirm", url.Values{"id": {itoa(confirmID)}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("confirm status=%d, want 303", resp.StatusCode)
	}
	resp.Body.Close()

	if len(f.seeds) != 1 {
		t.Fatalf("seeds after one confirm = %d, want exactly 1", len(f.seeds))
	}
	if f.seeds[0].AddressCidr.String() != "203.0.113.0/24" {
		t.Errorf("confirmed seed scope = %s, want 203.0.113.0/24", f.seeds[0].AddressCidr)
	}
	var pending, confirmed int
	for _, p := range f.proposals {
		switch p.Status {
		case "pending":
			pending++
		case "confirmed":
			confirmed++
			if !p.ConfirmedSeedID.Valid || p.ConfirmedSeedID.Int64 != f.seeds[0].ID {
				t.Errorf("confirmed proposal did not retain its seed as provenance: %+v", p)
			}
		}
	}
	if pending != 1 || confirmed != 1 {
		t.Errorf("after one singular confirm: pending=%d confirmed=%d, want 1 and 1", pending, confirmed)
	}
}

func TestConfirmOrgSourcedCIDRIsCanonicalAndScanEligible(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{candidates: []proposer.Candidate{
		{SourceSlug: proposer.SlugARIN, RecordKind: proposer.RecordRIRDelegation,
			Scope: netip.MustParsePrefix("198.51.100.130/25"), OrgName: "Org Discovery Co"},
	}}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Org Discovery Co").Body.Close()

	resp := postForm(t, ac, base+"/proposals/confirm", url.Values{"id": {itoa(f.proposals[0].ID)}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("confirm status=%d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()

	if len(f.seeds) != 1 {
		t.Fatalf("seeds after confirm = %d, want exactly 1", len(f.seeds))
	}
	seedCIDR := f.seeds[0].AddressCidr
	if seedCIDR == nil {
		t.Fatal("confirmed org-sourced seed has no address_cidr")
	}
	if got, want := seedCIDR.String(), "198.51.100.128/25"; got != want {
		t.Fatalf("confirmed org-sourced seed = %s, want canonical %s (parity with a manually-added CIDR)", got, want)
	}

	// The hot and cold fan-out consult this same custody gate, so Operator proves eligibility.
	target := netip.MustParseAddr("198.51.100.200")
	if got := (custody.Estate{AddressScopes: []netip.Prefix{*seedCIDR}}).Derive(target); got != custody.Operator {
		t.Errorf("in-range address custody = %q, want %q (org-sourced range must be scan-eligible)", got, custody.Operator)
	}
}

func TestConfirmRefusesOverCapProposalUntilCapAdmitsIt(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{candidates: []proposer.Candidate{
		{SourceSlug: proposer.SlugAFRINIC, RecordKind: proposer.RecordRIRDelegation,
			Scope: netip.MustParsePrefix("10.0.0.0/8"), OrgName: "Big Holder"},
	}}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Big").Body.Close()

	resp := postForm(t, ac, base+"/proposals/confirm", url.Values{"id": {itoa(f.proposals[0].ID)}})
	page := refusalPage(t, ac, base, resp)
	if len(f.seeds) != 0 {
		t.Fatalf("over-cap proposal was confirmed into a seed despite the cap: %+v", f.seeds)
	}
	if f.proposals[0].Status != "pending" {
		t.Errorf("refused proposal status=%q, want pending (a refused confirm spends nothing)", f.proposals[0].Status)
	}
	for _, want := range []string{"over your cap", "Settings · Scans", "decline"} {
		if !strings.Contains(page, want) {
			t.Errorf("over-cap refusal missing %q; body: %s", want, page)
		}
	}
	if !strings.Contains(page, "Decline selected") {
		t.Errorf("decline affordance gone after an over-cap refusal; body: %s", page)
	}

	f.instanceConfig.SeedAddressCap = 16777216
	resp = postForm(t, ac, base+"/proposals/confirm", url.Values{"id": {itoa(f.proposals[0].ID)}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("confirm within a raised cap status=%d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	if len(f.seeds) != 1 || f.seeds[0].AddressCidr.String() != "10.0.0.0/8" {
		t.Fatalf("raised-cap proposal was not confirmed into its seed: %+v", f.seeds)
	}
}

func TestDeclineIsBulkOverALookup(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{candidates: twoCandidates()}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()

	var ids []string
	for _, p := range f.proposals {
		ids = append(ids, itoa(p.ID))
	}
	resp := postForm(t, ac, base+"/proposals/decline", url.Values{"ids": ids})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("decline status=%d, want 303", resp.StatusCode)
	}
	resp.Body.Close()

	for _, p := range f.proposals {
		if p.Status != "declined" {
			t.Errorf("proposal %d status=%q, want declined", p.ID, p.Status)
		}
	}
	if len(f.seeds) != 0 {
		t.Errorf("seeds after decline = %d, want 0", len(f.seeds))
	}
	if page := seedsBody(t, ac, base); !strings.Contains(page, "No open proposals") {
		t.Errorf("declined proposals still shown as pending; body: %s", page)
	}
}

func TestDeclineRecordsEachScopeAsAnExclusion(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{candidates: twoCandidates()}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()

	var ids []string
	for _, p := range f.proposals {
		ids = append(ids, itoa(p.ID))
	}
	resp := postForm(t, ac, base+"/proposals/decline", url.Values{"ids": ids})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("decline status=%d, want 303", resp.StatusCode)
	}
	resp.Body.Close()

	want := map[string]bool{"203.0.113.0/24": false, "198.51.100.8/29": false}
	for _, e := range f.exclusions {
		if e.Kind == "address" && e.AddressCidr != nil {
			if _, ok := want[e.AddressCidr.String()]; ok {
				want[e.AddressCidr.String()] = true
			}
		}
	}
	for scope, seen := range want {
		if !seen {
			t.Errorf("declined scope %s was not recorded as an exclusion; exclusions: %+v", scope, f.exclusions)
		}
	}

	if page := seedsBody(t, ac, base); !strings.Contains(page, "203.0.113.0/24") {
		t.Errorf("declined scope not shown among exclusions; body: %s", page)
	}
}

func TestLookupRunsOnlyEnabledProposers(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{candidates: twoCandidates()}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")

	if _, err := f.UpsertSourceState(context.Background(), db.UpsertSourceStateParams{
		Slug: "arin", Enabled: false,
	}); err != nil {
		t.Fatal(err)
	}
	lookup(t, ac, base, "Example").Body.Close()

	if fp.lastEnabled["arin"] {
		t.Errorf("arin was passed as enabled after being toggled off: %v", fp.lastEnabled)
	}
	if !fp.lastEnabled[proposer.SlugAFRINIC] || !fp.lastEnabled[proposer.SlugAPNIC] {
		t.Errorf("default-on keyless proposers not enabled: %v", fp.lastEnabled)
	}
}

func captureLog(t *testing.T) *bytes.Buffer {
	// No test in this package runs in parallel, so the process-global logger is safe to borrow.
	t.Helper()
	var buf bytes.Buffer
	prevOut, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prevOut); log.SetFlags(prevFlags) })
	return &buf
}

func TestLookupBackendFailureIsNotAMiss(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{err: errors.New("arin: registry unreachable")}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")

	logs := captureLog(t)
	page := refusalPage(t, ac, base, lookup(t, ac, base, "Example"))
	if strings.Contains(page, "No candidate scopes matched that name.") {
		t.Errorf("a backend failure was rendered as a no-match; body: %s", page)
	}
	if !strings.Contains(page, "could not be completed") {
		t.Errorf("backend failure not surfaced to the operator; body: %s", page)
	}
	if len(f.proposals) != 0 {
		t.Errorf("a failed lookup filed %d proposals, want 0", len(f.proposals))
	}
	if got := logs.String(); !strings.Contains(got, "registry unreachable") || !strings.Contains(got, "Example") {
		t.Errorf("underlying perr not logged with the query; log: %q", got)
	}
}

func TestLookupGenuineMissStillReadsAsAMiss(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")

	page := refusalPage(t, ac, base, lookup(t, ac, base, "Nonesuch"))
	if !strings.Contains(page, "No candidate scopes matched that name.") {
		t.Errorf("a genuine no-match lost its message; body: %s", page)
	}
	if len(f.proposals) != 0 {
		t.Errorf("a no-match filed %d proposals, want 0", len(f.proposals))
	}
}

func TestLookupPartialFailureFilesAndFlags(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{candidates: twoCandidates(), err: errors.New("apnic-caida: timeout")}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := lookup(t, ac, base, "Example")
	if len(f.proposals) != 2 {
		resp.Body.Close()
		t.Fatalf("partial lookup filed %d proposals, want 2 (the candidates that returned)", len(f.proposals))
	}
	if resp.StatusCode != http.StatusSeeOther {
		resp.Body.Close()
		t.Fatalf("partial failure did not redirect (status=%d); a refresh would re-file duplicates", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if loc != "/scope" {
		t.Fatalf("partial-failure redirect %q is not the submitting URL", loc)
	}

	page := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(page, "partial") {
		t.Errorf("partial failure was not flagged to the operator; body: %s", page)
	}
	if !strings.Contains(page, "203.0.113.0/24") {
		t.Errorf("filed candidates not rendered on the partial-failure page; body: %s", page)
	}
}

func TestViewerCannotLookupConfirmOrDecline(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	fp := &fakeProposer{candidates: twoCandidates()}
	base := startWithProposer(t, f, fp)

	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()

	vc := login(t, base, "viewer", "hunter2hunter2")
	for _, ep := range []struct {
		path string
		form url.Values
	}{
		{"/proposals", url.Values{"query": {"Example"}}},
		{"/proposals/confirm", url.Values{"id": {itoa(f.proposals[0].ID)}}},
		{"/proposals/decline", url.Values{"lookup_id": {itoa(f.proposals[0].LookupID)}}},
	} {
		resp := postForm(t, vc, base+ep.path, ep.form)
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("viewer POST %s: status=%d, want 403", ep.path, resp.StatusCode)
		}
	}
	if len(f.seeds) != 0 {
		t.Errorf("viewer opened the gate: seeds=%d", len(f.seeds))
	}
	page := seedsBody(t, vc, base)
	if !strings.Contains(page, "203.0.113.0/24") {
		t.Errorf("viewer cannot read pending proposals; body: %s", page)
	}
	if strings.Contains(page, `action="/proposals/confirm"`) {
		t.Errorf("confirm control shown to a viewer; body: %s", page)
	}
}
