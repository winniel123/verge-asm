package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/proposer"
)

type fakeProposer struct {
	candidates  []proposer.Candidate
	attempts    []proposer.Attempt
	err         error
	lastQuery   string
	lastEnabled map[string]bool
	calls       int
}

func (p *fakeProposer) Propose(_ context.Context, org string, enabled map[string]bool) ([]proposer.Candidate, []proposer.Attempt, error) {
	p.calls++
	p.lastQuery = org
	p.lastEnabled = enabled
	return p.candidates, p.attempts, p.err
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

func oneScopeFromTwoSources() []proposer.Candidate {
	return []proposer.Candidate{
		{SourceSlug: proposer.SlugARIN, RecordKind: proposer.RecordRIRDelegation,
			Scope: netip.MustParsePrefix("203.0.113.0/24"), OrgName: "Example Org"},
		{SourceSlug: proposer.SlugAPNIC, RecordKind: proposer.RecordRIRDelegation,
			Scope: netip.MustParsePrefix("203.0.113.0/24"), OrgName: "Example Org"},
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
	for _, slug := range []string{proposer.SlugAFRINIC, proposer.SlugAPNIC} {
		if !fp.lastEnabled[slug] {
			t.Errorf("%s ships on and was not toggled off, but was passed as disabled: %v", slug, fp.lastEnabled)
		}
	}

	for _, slug := range []string{"arin", proposer.SlugAFRINIC, proposer.SlugAPNIC} {
		if _, err := f.UpsertSourceState(context.Background(), db.UpsertSourceStateParams{
			Slug: slug, Enabled: true,
		}); err != nil {
			t.Fatal(err)
		}
	}
	lookup(t, ac, base, "Example").Body.Close()

	if !fp.lastEnabled["arin"] {
		t.Errorf("arin was not passed as enabled after being toggled on: %v", fp.lastEnabled)
	}
	for _, slug := range []string{proposer.SlugAFRINIC, proposer.SlugAPNIC} {
		if !fp.lastEnabled[slug] {
			t.Errorf("%s was toggled on and was still passed as disabled: %v", slug, fp.lastEnabled)
		}
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

func (f *fakeStore) CreateProposerLookup(_ context.Context, arg db.CreateProposerLookupParams) (db.ProposerLookup, error) {
	l := db.ProposerLookup{
		ID: f.lookupNextID, Query: arg.Query, CreatedBy: arg.CreatedBy,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	f.lookups = append(f.lookups, l)
	f.lookupNextID++
	return l, nil
}

func (f *fakeStore) CreateProposal(_ context.Context, arg db.CreateProposalParams) (db.Proposal, error) {
	p := db.Proposal{
		ID: f.proposalNext, LookupID: arg.LookupID, SourceSlug: arg.SourceSlug,
		RecordKind: arg.RecordKind, AddressCidr: arg.AddressCidr, OrgName: arg.OrgName,
		Status:    "pending",
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	f.proposals = append(f.proposals, p)
	f.proposalNext++
	return p, nil
}

func (f *fakeStore) ListPendingProposals(context.Context) ([]db.ListPendingProposalsRow, error) {
	if f.pendingPropsErr != nil {
		return nil, f.pendingPropsErr
	}
	lookupByID := map[int64]db.ProposerLookup{}
	for _, l := range f.lookups {
		lookupByID[l.ID] = l
	}
	rows := []db.ListPendingProposalsRow{}
	for _, p := range f.proposals {
		if p.Status != "pending" {
			continue
		}
		l := lookupByID[p.LookupID]
		rows = append(rows, db.ListPendingProposalsRow{
			ID: p.ID, LookupID: p.LookupID, SourceSlug: p.SourceSlug,
			RecordKind: p.RecordKind, AddressCidr: p.AddressCidr, OrgName: p.OrgName,
			LookupQuery: l.Query, LookupAt: l.CreatedAt,
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].LookupID != rows[j].LookupID {
			return rows[i].LookupID > rows[j].LookupID
		}
		return rows[i].ID < rows[j].ID
	})
	return rows, nil
}

func (f *fakeStore) GetPendingProposal(_ context.Context, id int64) (db.Proposal, error) {
	for _, p := range f.proposals {
		if p.ID == id && p.Status == "pending" {
			return p, nil
		}
	}
	return db.Proposal{}, pgx.ErrNoRows
}

func (f *fakeStore) ConfirmProposal(_ context.Context, arg db.ConfirmProposalParams) (int64, error) {
	for i, p := range f.proposals {
		if p.ID == arg.ID && p.Status == "pending" {
			f.proposals[i].Status = "confirmed"
			f.proposals[i].ConfirmedSeedID = arg.ConfirmedSeedID
			return 1, nil
		}
	}
	return 0, nil
}

func (f *fakeStore) DeclineProposal(_ context.Context, id int64) (int64, error) {
	for i, p := range f.proposals {
		if p.ID == id && p.Status == "pending" {
			f.proposals[i].Status = "declined"
			return 1, nil
		}
	}
	return 0, nil
}

func (f *fakeStore) UndoDeclineProposal(_ context.Context, id int64) (netip.Prefix, error) {
	if f.undoDeclineErr != nil {
		return netip.Prefix{}, f.undoDeclineErr
	}
	for i, p := range f.proposals {
		if p.ID == id && p.Status == "declined" {
			f.proposals[i].Status = "pending"
			return p.AddressCidr, nil
		}
	}
	return netip.Prefix{}, pgx.ErrNoRows
}

func (f *fakeStore) ListDeclinedProposalScopes(context.Context) ([]db.ListDeclinedProposalScopesRow, error) {
	f.declinedRead++
	lookupAt := map[int64]pgtype.Timestamptz{}
	for _, l := range f.lookups {
		lookupAt[l.ID] = l.CreatedAt
	}
	out := []db.ListDeclinedProposalScopesRow{}
	for _, p := range f.proposals {
		if p.Status == "declined" {
			out = append(out, db.ListDeclinedProposalScopesRow{
				ID: p.ID, AddressCidr: p.AddressCidr, SourceSlug: p.SourceSlug,
				RecordKind: p.RecordKind, OrgName: p.OrgName, LookupAt: lookupAt[p.LookupID],
			})
		}
	}
	return out, nil
}

func (f *fakeStore) DeleteUnclaimedAddressExclusion(_ context.Context, addressCidr netip.Prefix) (db.DeleteUnclaimedAddressExclusionRow, error) {
	var out db.DeleteUnclaimedAddressExclusionRow
	row := -1
	for i, e := range f.exclusions {
		if e.Kind == "address" && e.AddressCidr != nil && e.AddressCidr.String() == addressCidr.String() {
			row = i
			break
		}
	}
	if row < 0 {
		return out, nil
	}
	out.DeclaredByHand = !f.exclusions[row].ProposalID.Valid
	for _, p := range f.proposals {
		if p.Status == "declined" && p.AddressCidr.String() == addressCidr.String() {
			out.StillClaimed = true
			break
		}
	}
	if !out.DeclaredByHand && !out.StillClaimed {
		f.exclusions = append(f.exclusions[:row], f.exclusions[row+1:]...)
	}
	return out, nil
}

func declineOne(t *testing.T, c *http.Client, base string, id int64) {
	t.Helper()
	resp := postForm(t, c, base+"/proposals/decline", url.Values{"ids": {itoa(id)}})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("decline status=%d, want 303", resp.StatusCode)
	}
}

func toastText(t *testing.T, resp *http.Response) string {
	t.Helper()
	// The ruling fixes one sentence, so the two toast fields are read back joined.
	loc := submitLoc(t, resp)
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse redirect %q: %v", loc, err)
	}
	raw := u.Query().Get("toast")
	if raw == "" {
		t.Fatalf("the redirect to %q carries no toast", loc)
	}
	blob, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode toast %q: %v", raw, err)
	}
	var got struct {
		Tone        string `json:"tone"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(blob, &got); err != nil {
		t.Fatalf("unmarshal toast %q: %v", blob, err)
	}
	return strings.TrimSpace(got.Title + " " + got.Description)
}

const undoDeclineFlash = "203.0.113.0/24 returned to pending. Confirming it is a fresh act."

func statusOf(f *fakeStore, id int64) string {
	for _, p := range f.proposals {
		if p.ID == id {
			return p.Status
		}
	}
	return ""
}

func addressExclusions(f *fakeStore) []string {
	var out []string
	for _, e := range f.exclusions {
		if e.Kind == "address" && e.AddressCidr != nil {
			out = append(out, e.AddressCidr.String())
		}
	}
	sort.Strings(out)
	return out
}

func TestUndoDeclineReturnsTheProposalAndLiftsItsExclusion(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: twoCandidates()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()

	declined := f.proposals[0]
	declineOne(t, ac, base, declined.ID)
	if got := addressExclusions(f); len(got) != 1 || got[0] != "203.0.113.0/24" {
		t.Fatalf("exclusions after the decline = %v, want [203.0.113.0/24]", got)
	}

	const from = "/scope?seen=1"
	resp := postForm(t, ac, base+"/proposals/undo-decline",
		url.Values{"id": {itoa(declined.ID)}, "return": {from}})
	if got := toastText(t, resp); got != undoDeclineFlash {
		t.Errorf("undo flash = %q, want %q", got, undoDeclineFlash)
	}
	if loc := resp.Header.Get("Location"); !strings.HasPrefix(loc, from) {
		t.Errorf("undo landed at %q, want the submitting URL %q", loc, from)
	}

	if got := statusOf(f, declined.ID); got != "pending" {
		t.Errorf("proposal %d status=%q, want pending", declined.ID, got)
	}
	if got := addressExclusions(f); len(got) != 0 {
		t.Errorf("exclusions after the undo = %v, want none", got)
	}
	if len(f.messages) != 0 {
		t.Errorf("the undo fired %d messages; a declined scope was never declared", len(f.messages))
	}
	if page := seedsBody(t, ac, base); !strings.Contains(page, `action="/proposals/confirm"`) {
		t.Errorf("the returned scope is not offered for a fresh confirm; body: %s", page)
	}
}

const undoDeclineKeptFlash = "203.0.113.0/24 returned to pending. " +
	"Its exclusion stays — another declined proposal still claims that scope. Confirming it is a fresh act."

func declineOneScopeFromTwoSources(t *testing.T, f *fakeStore) (string, *http.Client, int64, int64) {
	t.Helper()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: oneScopeFromTwoSources()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()

	if len(f.proposals) != 2 {
		t.Fatalf("proposals = %d, want 2; one scope from two sources is the ordinary case", len(f.proposals))
	}
	first, second := f.proposals[0].ID, f.proposals[1].ID
	declineOne(t, ac, base, first)
	declineOne(t, ac, base, second)
	if got := addressExclusions(f); len(got) != 1 || got[0] != "203.0.113.0/24" {
		t.Fatalf("exclusions after both declines = %v, want the one [203.0.113.0/24] row", got)
	}
	return base, ac, first, second
}

func TestBothDeclinesOfOneScopeRenderAnUndoControl(t *testing.T) {
	f := newFakeStore()
	base, ac, first, second := declineOneScopeFromTwoSources(t, f)

	page := seedsBody(t, ac, base)
	if got := strings.Count(page, `action="/proposals/undo-decline"`); got != 2 {
		t.Fatalf("undo controls = %d, want 2 (one per declined proposal); body: %s", got, page)
	}
	for _, id := range []int64{first, second} {
		if !strings.Contains(page, `<input type="hidden" name="id" value="`+itoa(id)+`">`) {
			t.Errorf("no undo control carries declined proposal %d; body: %s", id, page)
		}
	}
}

func undoControlFor(t *testing.T, page string, id int64) string {
	t.Helper()
	const open = `<form method="post" action="/proposals/undo-decline">`
	for _, chunk := range strings.Split(page, open)[1:] {
		form, _, ok := strings.Cut(chunk, "</form>")
		if ok && strings.Contains(form, `name="id" value="`+itoa(id)+`"`) {
			return form
		}
	}
	t.Fatalf("no undo control carries declined proposal %d; body: %s", id, page)
	return ""
}

func TestUndoControlsDateEveryDeclineOfOneScope(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	agoMin := func(m int) pgtype.Timestamptz {
		return pgtype.Timestamptz{Time: now.Add(-time.Duration(m) * time.Minute), Valid: true}
	}
	ago := func(h int) pgtype.Timestamptz { return agoMin(h * 60) }
	crowded := netip.MustParsePrefix("203.0.113.0/24")
	lone := netip.MustParsePrefix("198.51.100.0/24")

	got := toUndoControls([]db.ListDeclinedProposalScopesRow{
		{ID: 1, AddressCidr: crowded, SourceSlug: proposer.SlugARIN, OrgName: "Acme Corp",
			RecordKind: proposer.RecordRIRDelegation, LookupAt: ago(3)},
		{ID: 2, AddressCidr: crowded, SourceSlug: proposer.SlugAPNIC, OrgName: "Acme Corp",
			RecordKind: proposer.RecordRIRDelegation, LookupAt: ago(3)},
		// One ARIN lookup answers with both kinds, so this row ties with row 1 on every other axis.
		{ID: 3, AddressCidr: crowded, SourceSlug: proposer.SlugARIN, OrgName: "Acme Corp",
			RecordKind: proposer.RecordCompelledReassignment, LookupAt: ago(3)},
		{ID: 4, AddressCidr: crowded, SourceSlug: proposer.SlugARIN, OrgName: "Acme Corp",
			RecordKind: proposer.RecordRIRDelegation, LookupAt: ago(50)},
		// One ARIN response lists two holders of one range, so only the holder parts these (#1873).
		{ID: 6, AddressCidr: crowded, SourceSlug: proposer.SlugARIN, OrgName: "Globex Ltd",
			RecordKind: proposer.RecordRIRDelegation, LookupAt: ago(3)},
		// A repeated lookup of one holder lands in row 1's relative bucket, 14 minutes off (#1873).
		{ID: 7, AddressCidr: crowded, SourceSlug: proposer.SlugARIN, OrgName: "Acme Corp",
			RecordKind: proposer.RecordRIRDelegation, LookupAt: agoMin(3*60 + 14)},
		{ID: 5, AddressCidr: lone, SourceSlug: proposer.SlugARIN,
			RecordKind: proposer.RecordRIRDelegation},
	})

	if len(got) != 2 {
		t.Fatalf("scopes = %d, want 2", len(got))
	}
	want := map[string][]undoControlView{
		crowded.String(): {
			{ProposalID: 1, Org: "Acme Corp", Source: proposer.SlugARIN, Record: "RIR delegation", ISO: "2026-08-15 09:00 UTC"},
			{ProposalID: 2, Org: "Acme Corp", Source: proposer.SlugAPNIC, Record: "RIR delegation", ISO: "2026-08-15 09:00 UTC"},
			{ProposalID: 3, Org: "Acme Corp", Source: proposer.SlugARIN, Record: "compelled reassignment", ISO: "2026-08-15 09:00 UTC"},
			{ProposalID: 4, Org: "Acme Corp", Source: proposer.SlugARIN, Record: "RIR delegation", ISO: "2026-08-13 10:00 UTC"},
			{ProposalID: 6, Org: "Globex Ltd", Source: proposer.SlugARIN, Record: "RIR delegation", ISO: "2026-08-15 09:00 UTC"},
			{ProposalID: 7, Org: "Acme Corp", Source: proposer.SlugARIN, Record: "RIR delegation", ISO: "2026-08-15 08:46 UTC"},
		},
		// A lookup with no instant still reaches its decline, so the control drops the date alone.
		lone.String(): {{ProposalID: 5, Source: proposer.SlugARIN, Record: "RIR delegation"}},
	}
	for scope, controls := range want {
		if len(got[scope]) != len(controls) {
			t.Fatalf("controls for %s = %+v, want %+v", scope, got[scope], controls)
		}
		for i, c := range controls {
			if got[scope][i] != c {
				t.Errorf("control %d of %s = %+v, want %+v", i, scope, got[scope][i], c)
			}
		}
	}

	// The title is mouse-only, so only the rendered label parts two controls (#1873).
	for scope, controls := range got {
		seen := map[string]int64{}
		for _, c := range controls {
			if prior, dup := seen[c.Label()]; dup {
				t.Errorf("proposals %d and %d both read %q on the %s row, so the choice is a guess",
					prior, c.ProposalID, c.Label(), scope)
			}
			seen[c.Label()] = c.ProposalID
		}
	}
}

func TestUndoControlNamesTheSourceThatProposedItsScope(t *testing.T) {
	f := newFakeStore()
	base, ac, first, second := declineOneScopeFromTwoSources(t, f)

	page := seedsBody(t, ac, base)
	for _, c := range []struct {
		id      int64
		mine    string
		sibling string
	}{
		{first, proposer.SlugARIN, proposer.SlugAPNIC},
		{second, proposer.SlugAPNIC, proposer.SlugARIN},
	} {
		control := undoControlFor(t, page, c.id)
		if !strings.Contains(control, c.mine) {
			t.Errorf("undo control for proposal %d does not name %q: %s", c.id, c.mine, control)
		}
		if strings.Contains(control, c.sibling) {
			t.Errorf("undo control for proposal %d names the sibling source %q: %s", c.id, c.sibling, control)
		}
	}
}

func TestUndoControlCarriesTheLookupInstantItsProposalCameFrom(t *testing.T) {
	f := newFakeStore()
	base, ac, first, second := declineOneScopeFromTwoSources(t, f)

	if len(f.lookups) != 1 {
		t.Fatalf("lookups = %d, want 1", len(f.lookups))
	}
	want := `at ` + f.lookups[0].CreatedAt.Time.UTC().Format("2006-01-02 15:04 UTC") + `"`

	page := seedsBody(t, ac, base)
	for _, id := range []int64{first, second} {
		control := undoControlFor(t, page, id)
		if !strings.Contains(control, want) {
			t.Errorf("undo control for proposal %d carries no %s: %s", id, want, control)
		}
	}
}

func TestUndoDeclineKeepsAnExclusionASiblingDeclineStillClaims(t *testing.T) {
	f := newFakeStore()
	base, ac, first, second := declineOneScopeFromTwoSources(t, f)

	resp := postForm(t, ac, base+"/proposals/undo-decline", url.Values{"id": {itoa(first)}})
	if got := toastText(t, resp); got != undoDeclineKeptFlash {
		t.Errorf("first undo flash = %q, want %q", got, undoDeclineKeptFlash)
	}
	if got := statusOf(f, first); got != "pending" {
		t.Errorf("proposal %d status=%q, want pending", first, got)
	}
	if got := statusOf(f, second); got != "declined" {
		t.Errorf("sibling proposal %d status=%q, want declined", second, got)
	}
	if got := addressExclusions(f); len(got) != 1 || got[0] != "203.0.113.0/24" {
		t.Fatalf("exclusions after the first undo = %v, want the sibling's [203.0.113.0/24] row", got)
	}

	resp = postForm(t, ac, base+"/proposals/undo-decline", url.Values{"id": {itoa(second)}})
	if got := toastText(t, resp); got != undoDeclineFlash {
		t.Errorf("second undo flash = %q, want %q", got, undoDeclineFlash)
	}
	if got := addressExclusions(f); len(got) != 0 {
		t.Errorf("exclusions after the last undo = %v, want none", got)
	}
}

func TestConfirmRefusesAScopeASiblingDeclineStillExcludes(t *testing.T) {
	f := newFakeStore()
	base, ac, first, second := declineOneScopeFromTwoSources(t, f)

	postForm(t, ac, base+"/proposals/undo-decline", url.Values{"id": {itoa(first)}}).Body.Close()
	if got := addressExclusions(f); len(got) != 1 {
		t.Fatalf("exclusions after the first undo = %v, want the sibling's row", got)
	}

	resp := postForm(t, ac, base+"/proposals/confirm", url.Values{"id": {itoa(first)}})
	page := refusalPage(t, ac, base, resp)
	if len(f.seeds) != 0 {
		t.Fatalf("a seed was declared under a standing exclusion, so its ground measures nothing: %+v", f.seeds)
	}
	if got := statusOf(f, first); got != "pending" {
		t.Errorf("refused proposal %d status=%q, want pending", first, got)
	}
	if got := statusOf(f, second); got != "declined" {
		t.Errorf("sibling proposal %d status=%q, want declined", second, got)
	}
	for _, want := range []string{"203.0.113.0/24", "refuses ground", "Undo every decline"} {
		if !strings.Contains(page, want) {
			t.Errorf("still-excluded refusal missing %q; body: %s", want, page)
		}
	}

	postForm(t, ac, base+"/proposals/undo-decline", url.Values{"id": {itoa(second)}}).Body.Close()
	resp = postForm(t, ac, base+"/proposals/confirm", url.Values{"id": {itoa(first)}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("confirm after the last undo status=%d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	if len(f.seeds) != 1 || f.seeds[0].AddressCidr.String() != "203.0.113.0/24" {
		t.Fatalf("the scope was not confirmed once no decline claimed it: %+v", f.seeds)
	}
}

func TestUndoDeclineWithNoExclusionStillReturnsTheProposal(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: twoCandidates()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()

	declined := f.proposals[0]
	declineOne(t, ac, base, declined.ID)
	// An operator may lift the exclusion on its own row first, which leaves the decline alone.
	f.exclusions = nil

	resp := postForm(t, ac, base+"/proposals/undo-decline", url.Values{"id": {itoa(declined.ID)}})
	if got := toastText(t, resp); got != undoDeclineFlash {
		t.Errorf("undo flash = %q, want %q", got, undoDeclineFlash)
	}
	if got := statusOf(f, declined.ID); got != "pending" {
		t.Errorf("proposal %d status=%q, want pending", declined.ID, got)
	}
	if len(f.messages) != 0 {
		t.Errorf("the undo fired %d messages", len(f.messages))
	}
}

func TestUndoDeclineSurfacesAStoreFailure(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: twoCandidates()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()

	id := f.proposals[0].ID
	declineOne(t, ac, base, id)
	f.undoDeclineErr = errors.New("connection refused")

	resp := postForm(t, ac, base+"/proposals/undo-decline", url.Values{"id": {itoa(id)}})
	defer resp.Body.Close()
	// A silent redirect would read to the operator as "nothing to undo" (#1721).
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("a store failure answered %d, want 500", resp.StatusCode)
	}
}

func TestUndoDeclineRefusesAPendingOrConfirmedProposal(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: twoCandidates()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()

	pending := f.proposals[0].ID
	confirmed := f.proposals[1].ID
	postForm(t, ac, base+"/proposals/confirm", url.Values{"id": {itoa(confirmed)}}).Body.Close()
	seedsBefore := len(f.seeds)

	for _, id := range []int64{pending, confirmed} {
		resp := postForm(t, ac, base+"/proposals/undo-decline", url.Values{"id": {itoa(id)}})
		loc := submitLoc(t, resp)
		if strings.Contains(loc, "toast=") {
			t.Errorf("undo of proposal %d answered with a receipt at %q", id, loc)
		}
	}
	if got := statusOf(f, pending); got != "pending" {
		t.Errorf("pending proposal moved to %q", got)
	}
	if got := statusOf(f, confirmed); got != "confirmed" {
		t.Errorf("confirmed proposal moved to %q", got)
	}
	if len(f.seeds) != seedsBefore {
		t.Errorf("seeds = %d, want %d; the undo touched the gate", len(f.seeds), seedsBefore)
	}
}

func TestViewerCannotUndoADecline(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: twoCandidates()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()
	declined := f.proposals[0]
	declineOne(t, ac, base, declined.ID)

	vc := login(t, base, "viewer", "hunter2hunter2")
	resp := postForm(t, vc, base+"/proposals/undo-decline", url.Values{"id": {itoa(declined.ID)}})
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("viewer POST /proposals/undo-decline: status=%d, want 403", resp.StatusCode)
	}
	if got := statusOf(f, declined.ID); got != "declined" {
		t.Errorf("proposal %d status=%q, want declined", declined.ID, got)
	}
	if page := seedsBody(t, vc, base); strings.Contains(page, `action="/proposals/undo-decline"`) {
		t.Errorf("undo control shown to a viewer; body: %s", page)
	}
}

func TestOnlyTheDeclinedScopesExclusionRowCarriesTheUndoControl(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: twoCandidates()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()
	declineOne(t, ac, base, f.proposals[0].ID)
	postForm(t, ac, base+"/exclusions", url.Values{
		"kind": {"address"}, "value": {"192.0.2.0/24"},
	}).Body.Close()

	page := seedsBody(t, ac, base)
	if got := strings.Count(page, `action="/proposals/undo-decline"`); got != 1 {
		t.Fatalf("undo controls on /scope = %d, want 1 (a hand-typed exclusion carries none); body: %s", got, page)
	}
	if !strings.Contains(page, `<input type="hidden" name="id" value="`+itoa(f.proposals[0].ID)+`">`) {
		t.Errorf("the undo control does not carry the declined proposal's id; body: %s", page)
	}
	if !strings.Contains(page, "Undo decline") {
		t.Errorf("the undo control has no label; body: %s", page)
	}
}

func TestScopeSkipsTheDeclinedReadWithoutAnAddressExclusion(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: twoCandidates()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()
	declineOne(t, ac, base, f.proposals[0].ID)
	f.exclusions = nil

	f.declinedRead = 0
	page := seedsBody(t, ac, base)
	if f.declinedRead != 0 {
		t.Errorf("declined reads on an empty exclusions list = %d, want 0", f.declinedRead)
	}
	if strings.Contains(page, `action="/proposals/undo-decline"`) {
		t.Errorf("an undo control rendered without an exclusion row; body: %s", page)
	}

	postForm(t, ac, base+"/exclusions", url.Values{
		"kind": {"name"}, "value": {"example.com"},
	}).Body.Close()

	f.declinedRead = 0
	page = seedsBody(t, ac, base)
	if f.declinedRead != 0 {
		t.Errorf("declined reads with only a name exclusion = %d, want 0", f.declinedRead)
	}
	if strings.Contains(page, `action="/proposals/undo-decline"`) {
		t.Errorf("an undo control rendered beside a name exclusion; body: %s", page)
	}
}

func TestScopeReadsTheDeclinedTailForAnAddressExclusion(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: twoCandidates()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()
	declineOne(t, ac, base, f.proposals[0].ID)

	f.declinedRead = 0
	page := seedsBody(t, ac, base)
	if f.declinedRead != 1 {
		t.Errorf("declined reads with an address exclusion = %d, want 1", f.declinedRead)
	}
	if !strings.Contains(page, `<input type="hidden" name="id" value="`+itoa(f.proposals[0].ID)+`">`) {
		t.Errorf("the undo control does not carry the declined proposal's id; body: %s", page)
	}
}

func TestOverCapProposalRowRendersRefusalAndNoConfirm(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	fp := &fakeProposer{candidates: append(twoCandidates(), proposer.Candidate{
		SourceSlug: proposer.SlugAFRINIC, RecordKind: proposer.RecordRIRDelegation,
		Scope: netip.MustParsePrefix("10.0.0.0/8"), OrgName: "Big Holder",
	})}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()

	page := seedsBody(t, ac, base)
	for _, want := range []string{"10.0.0.0/8", "over your cap", "Settings · Scans", "decline", "Decline selected"} {
		if !strings.Contains(page, want) {
			t.Errorf("over-cap row missing %q; body: %s", want, page)
		}
	}
	if got := strings.Count(page, `action="/proposals/confirm"`); got != 2 {
		t.Errorf("confirm forms = %d, want 2 (one per in-cap row; the over-cap row carries none)", got)
	}
	if got := strings.Count(page, `name="ids"`); got != 3 {
		t.Errorf("decline checkboxes = %d, want 3 (bulk decline stays open to the over-cap row)", got)
	}

	f.instanceConfig.SeedAddressCap = 16777216
	page = seedsBody(t, ac, base)
	if strings.Contains(page, "over your cap") {
		t.Errorf("a raised cap still renders the refusal; body: %s", page)
	}
	if got := strings.Count(page, `action="/proposals/confirm"`); got != 3 {
		t.Errorf("confirm forms after a raised cap = %d, want 3", got)
	}
}

func pendingProposals(f *fakeStore) int {
	n := 0
	for _, p := range f.proposals {
		if p.Status == "pending" {
			n++
		}
	}
	return n
}

func TestLookupDoesNotRefileADeclinedScope(t *testing.T) {
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
	postForm(t, ac, base+"/proposals/decline", url.Values{"ids": ids}).Body.Close()
	if pendingProposals(f) != 0 {
		t.Fatalf("decline left %d pending", pendingProposals(f))
	}

	page := refusalPage(t, ac, base, lookup(t, ac, base, "Example"))
	if pendingProposals(f) != 0 {
		t.Errorf("a second lookup re-filed %d declined scopes as pending", pendingProposals(f))
	}
	if !strings.Contains(page, "No candidate scopes matched that name.") {
		t.Errorf("an all-excluded lookup did not read as a miss; body: %s", page)
	}
	if strings.Contains(page, "could not be completed") {
		t.Errorf("an all-excluded lookup read as a backend failure; body: %s", page)
	}
}

func TestLookupSkipsCandidateInsideAnExclusionButOffersAWiderOne(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	for _, raw := range []string{"198.51.100.0/24", "203.0.113.0/25"} {
		p := netip.MustParsePrefix(raw)
		f.exclusions = append(f.exclusions, db.Exclusion{ID: f.exclNextID, Kind: "address", AddressCidr: &p, CreatedBy: pgtype.Int8{Int64: 1, Valid: true}})
		f.exclNextID++
	}
	fp := &fakeProposer{candidates: twoCandidates()}
	base := startWithProposer(t, f, fp)
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := lookup(t, ac, base, "Example")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("lookup status=%d, want 303", resp.StatusCode)
	}
	if len(f.proposals) != 1 {
		t.Fatalf("filed %d proposals, want 1: %+v", len(f.proposals), f.proposals)
	}
	if got := f.proposals[0].AddressCidr.String(); got != "203.0.113.0/24" {
		t.Errorf("filed %s; want the candidate wider than its exclusion, 203.0.113.0/24", got)
	}
}

const undoDeclineDeclaredFlash = "203.0.113.0/24 returned to pending. " +
	"Its exclusion stays — no decline recorded that row. Lift it on the exclusions screen. Confirming it is a fresh act."

func declareAddressExclusion(t *testing.T, c *http.Client, base, cidr string) {
	t.Helper()
	postForm(t, c, base+"/exclusions", url.Values{"kind": {"address"}, "value": {cidr}}).Body.Close()
}

func TestUndoDeclineKeepsAHandDeclaredExclusionOfTheSameScope(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: twoCandidates()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()
	declined := f.proposals[0]

	// excludeCandidates drops a candidate its own exclusion covers, so the declaration
	// lands after the lookup (#1799).
	declareAddressExclusion(t, ac, base, "203.0.113.0/24")
	if got := addressExclusions(f); len(got) != 1 || got[0] != "203.0.113.0/24" {
		t.Fatalf("exclusions after the declaration = %v, want [203.0.113.0/24]", got)
	}
	declaredID := f.exclusions[0].ID

	declineOne(t, ac, base, declined.ID)
	if got := addressExclusions(f); len(got) != 1 {
		t.Fatalf("exclusions after the decline = %v, want the one declared row", got)
	}

	resp := postForm(t, ac, base+"/proposals/undo-decline", url.Values{"id": {itoa(declined.ID)}})
	if got := toastText(t, resp); got != undoDeclineDeclaredFlash {
		t.Errorf("undo flash = %q, want %q", got, undoDeclineDeclaredFlash)
	}
	if got := addressExclusions(f); len(got) != 1 || got[0] != "203.0.113.0/24" {
		t.Fatalf("the undo deleted the operator's own declaration: exclusions = %v", got)
	}
	if f.exclusions[0].ID != declaredID {
		t.Errorf("the declaration row was replaced: id = %d, want %d", f.exclusions[0].ID, declaredID)
	}
	if got := statusOf(f, declined.ID); got != "pending" {
		t.Errorf("proposal %d status=%q, want pending", declined.ID, got)
	}
}

const undoDeclineDeclaredAndClaimedFlash = "203.0.113.0/24 returned to pending. " +
	"Its exclusion stays — no decline recorded that row, and another declined proposal " +
	"still claims that scope. Confirming it is a fresh act."

func TestUndoDeclineWithdrawsTheLiftAdviceWhileASiblingDeclineStands(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: oneScopeFromTwoSources()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()
	first, second := f.proposals[0].ID, f.proposals[1].ID

	declareAddressExclusion(t, ac, base, "203.0.113.0/24")
	declineOne(t, ac, base, first)
	declineOne(t, ac, base, second)
	if got := addressExclusions(f); len(got) != 1 {
		t.Fatalf("exclusions after both declines = %v, want the one declared row", got)
	}

	resp := postForm(t, ac, base+"/proposals/undo-decline", url.Values{"id": {itoa(first)}})
	got := toastText(t, resp)
	if got != undoDeclineDeclaredAndClaimedFlash {
		t.Errorf("undo flash = %q, want %q", got, undoDeclineDeclaredAndClaimedFlash)
	}
	// Lifting the row takes the only undo control the sibling has (scope.tmpl, #1799).
	if strings.Contains(got, "Lift it on the exclusions screen") {
		t.Error("the flash advised a lift that would strand the sibling decline with no undo control")
	}
	if got := statusOf(f, second); got != "declined" {
		t.Errorf("sibling proposal %d status=%q, want declined", second, got)
	}
}

func TestDeclineRecordsTheProposalOnTheExclusionItWrites(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: twoCandidates()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()

	declined := f.proposals[0]
	declineOne(t, ac, base, declined.ID)
	if len(f.exclusions) != 1 {
		t.Fatalf("exclusions after the decline = %d, want 1", len(f.exclusions))
	}
	// Only this column tells a later undo the row is a decline's and not a declaration (#1799).
	got := f.exclusions[0].ProposalID
	if !got.Valid || got.Int64 != declined.ID {
		t.Errorf("the decline's exclusion carries proposal_id %+v, want %d", got, declined.ID)
	}
}

func TestHandDeclaredExclusionCarriesNoProposal(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: twoCandidates()})
	ac := login(t, base, "admin", "hunter2hunter2")

	declareAddressExclusion(t, ac, base, "198.51.100.0/24")
	if len(f.exclusions) != 1 {
		t.Fatalf("exclusions after the declaration = %d, want 1", len(f.exclusions))
	}
	if got := f.exclusions[0].ProposalID; got.Valid {
		t.Errorf("a hand-declared exclusion carries proposal_id %+v, want NULL", got)
	}
}
