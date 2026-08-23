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

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/proposer"
)

// fakeProposer stands in for the real registry: it returns canned candidates and
// records the enabled set it was asked to run, so a test can assert that the
// source-enablement state gates which paths run without any network.
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

// get issues a GET and fails the test on a transport error, leaving the caller
// to read the body (with body()) and assert on status.
func get(t *testing.T, c *http.Client, rawURL string) *http.Response {
	t.Helper()
	resp, err := c.Get(rawURL)
	if err != nil {
		t.Fatalf("GET %s: %v", rawURL, err)
	}
	return resp
}

// twoCandidates is a delegation plus a compelled reassignment — the two record
// kinds ARIN returns in one response.
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

	// A proposer admits nothing: the lookup produced Proposal rows and nothing
	// entered the estate — no Seed was created.
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

	// The pending Proposals render on the Seeds screen, each with the record
	// kind that produced it and a singular, count-labelled confirm.
	page := seedsBody(t, ac, base)
	for _, want := range []string{
		"203.0.113.0/24", "198.51.100.8/29",
		"RIR delegation", "compelled reassignment",
		"Confirm 256 addresses", "Confirm 8 addresses",
		"Decline all 2",
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
	// The one act this surface must never draw: a batch confirm (ADR-0022).
	for _, forbidden := range []string{"Confirm all", "Confirm selected", `type="checkbox" name="confirm`} {
		if strings.Contains(page, forbidden) {
			t.Errorf("batch-confirm affordance present (%q); ADR-0022 forbids it", forbidden)
		}
	}

	// Confirming one Proposal creates exactly one Seed and retains the Proposal
	// as its provenance; the other stays pending.
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

func TestConfirmSkipsTheAddressScopeCap(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// A /8 is 16,777,216 addresses — far over the operator's own 1024 cap. A
	// Proposal is a registry-authored scope confirmed whole (ADR-0022), so the
	// cap that governs typed scopes does not bind it.
	fp := &fakeProposer{candidates: []proposer.Candidate{
		{SourceSlug: proposer.SlugAFRINIC, RecordKind: proposer.RecordRIRDelegation,
			Scope: netip.MustParsePrefix("10.0.0.0/8"), OrgName: "Big Holder"},
	}}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Big").Body.Close()

	resp := postForm(t, ac, base+"/proposals/confirm", url.Values{"id": {itoa(f.proposals[0].ID)}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("confirm over-cap proposal status=%d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	if len(f.seeds) != 1 || f.seeds[0].AddressCidr.String() != "10.0.0.0/8" {
		t.Fatalf("over-cap proposal was not confirmed into a seed: %+v", f.seeds)
	}
}

func TestDeclineIsBulkOverALookup(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{candidates: twoCandidates()}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()

	lookupID := f.proposals[0].LookupID
	resp := postForm(t, ac, base+"/proposals/decline", url.Values{"lookup_id": {itoa(lookupID)}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("decline status=%d, want 303", resp.StatusCode)
	}
	resp.Body.Close()

	// Every Proposal under the lookup is declined in one act, and no Seed was
	// created — declining is a boundary claim, not an admission.
	for _, p := range f.proposals {
		if p.Status != "declined" {
			t.Errorf("proposal %d status=%q, want declined", p.ID, p.Status)
		}
	}
	if len(f.seeds) != 0 {
		t.Errorf("seeds after decline = %d, want 0", len(f.seeds))
	}
	// The pending section is now empty.
	if page := seedsBody(t, ac, base); !strings.Contains(page, "No pending proposals") {
		t.Errorf("declined lookup still shown as pending; body: %s", page)
	}
}

func TestDeclineRecordsEachScopeAsAnExclusion(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{candidates: twoCandidates()}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()

	lookupID := f.proposals[0].LookupID
	resp := postForm(t, ac, base+"/proposals/decline", url.Values{"lookup_id": {itoa(lookupID)}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("decline status=%d, want 303", resp.StatusCode)
	}
	resp.Body.Close()

	// A decline is a boundary claim: each proposed scope becomes an address
	// exclusion so the same range is not silently re-admitted.
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

	// The declined scope now shows on the Scope screen as an exclusion.
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

	// Turn the ARIN keyless path off through the source-enablement state (#185).
	if _, err := f.UpsertSourceState(context.Background(), db.UpsertSourceStateParams{
		Slug: "arin", Enabled: false,
	}); err != nil {
		t.Fatal(err)
	}
	lookup(t, ac, base, "Example").Body.Close()

	if fp.lastEnabled["arin"] {
		t.Errorf("arin was passed as enabled after being toggled off: %v", fp.lastEnabled)
	}
	// The other keyless paths ship on and are still offered.
	if !fp.lastEnabled[proposer.SlugAFRINIC] || !fp.lastEnabled[proposer.SlugAPNIC] {
		t.Errorf("default-on keyless proposers not enabled: %v", fp.lastEnabled)
	}
}

// captureLog redirects the standard logger to a buffer for the duration of a
// test, so an assertion can prove a server-side failure was logged where a
// maintainer would find it. These tests do not run in parallel, so the
// process-global logger is safe to borrow.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prevOut, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prevOut); log.SetFlags(prevFlags) })
	return &buf
}

// TestLookupBackendFailureIsNotAMiss covers the #251 confusion: a lookup that
// errors on the backend with no candidates must read as a backend failure, not
// as "your org name matched nothing", must file nothing, and must log the
// underlying error with enough context for a maintainer to find it.
func TestLookupBackendFailureIsNotAMiss(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// Every enabled path errored and none returned a candidate.
	fp := &fakeProposer{err: errors.New("arin: registry unreachable")}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")

	logs := captureLog(t)
	page := body(t, lookup(t, ac, base, "Example"))
	if strings.Contains(page, "No candidate scopes matched that name.") {
		t.Errorf("a backend failure was rendered as a no-match; body: %s", page)
	}
	if !strings.Contains(page, "could not be completed") {
		t.Errorf("backend failure not surfaced to the operator; body: %s", page)
	}
	// A failed lookup admits nothing and files no Proposal.
	if len(f.proposals) != 0 {
		t.Errorf("a failed lookup filed %d proposals, want 0", len(f.proposals))
	}
	// The discarded perr is logged with the query so it is operator-findable.
	if got := logs.String(); !strings.Contains(got, "registry unreachable") || !strings.Contains(got, "Example") {
		t.Errorf("underlying perr not logged with the query; log: %q", got)
	}
}

// TestLookupGenuineMissStillReadsAsAMiss keeps the honest no-match message when
// the registries answered cleanly and simply held nothing.
func TestLookupGenuineMissStillReadsAsAMiss(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{} // no candidates, no error
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")

	page := body(t, lookup(t, ac, base, "Nonesuch"))
	if !strings.Contains(page, "No candidate scopes matched that name.") {
		t.Errorf("a genuine no-match lost its message; body: %s", page)
	}
	if len(f.proposals) != 0 {
		t.Errorf("a no-match filed %d proposals, want 0", len(f.proposals))
	}
}

// TestLookupPartialFailureFilesAndFlags covers the mixed case: some paths
// errored while others returned candidates. The candidates that did come back
// are filed, and the lookup still uses post-redirect-get (so a refresh cannot
// re-file duplicates) but carries a flag that surfaces the incompleteness.
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
	// Post-redirect-get, exactly like a clean success — a partial lookup persists
	// rows, so an inline render off the POST would re-file duplicates on refresh.
	if resp.StatusCode != http.StatusSeeOther {
		resp.Body.Close()
		t.Fatalf("partial failure did not redirect (status=%d); a refresh would re-file duplicates", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if !strings.Contains(loc, "notice=partial-proposals") {
		t.Fatalf("partial-failure redirect %q carries no incompleteness flag", loc)
	}

	// Following the redirect, the Seeds page shows the caveat and the candidates
	// that did come back.
	page := body(t, get(t, ac, base+loc))
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

	// Admin seeds a pending lookup so the viewer has something to read.
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
	// Nothing the viewer did changed state.
	if len(f.seeds) != 0 {
		t.Errorf("viewer opened the gate: seeds=%d", len(f.seeds))
	}
	// But the viewer can read the pending list, without a confirm control.
	page := seedsBody(t, vc, base)
	if !strings.Contains(page, "203.0.113.0/24") {
		t.Errorf("viewer cannot read pending proposals; body: %s", page)
	}
	if strings.Contains(page, `action="/proposals/confirm"`) {
		t.Errorf("confirm control shown to a viewer; body: %s", page)
	}
}
