package main

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/edgefanout"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/scan"
)

// Withdrawing a Seed closes timelines exactly as declaring an exclusion does (#1040),
// and until #1046 it was the one narrowing act that committed on a single click with
// no count shown. These tests read the two steps: the first click states what the
// withdrawal would take, the second performs it.

func candidateSpan(id int64, kind, key string) db.ListSeedWithdrawalCandidatesRow {
	return db.ListSeedWithdrawalCandidatesRow{ID: id, SubjectKind: kind, SubjectKey: key}
}

// previewChip clicks a chip's remove control and returns the landing GET's body.
func previewChip(t *testing.T, c *http.Client, base string, id int64) string {
	t.Helper()
	resp := postForm(t, c, base+"/seeds/preview", url.Values{"id": {intStr(id)}})
	if resp.StatusCode != http.StatusSeeOther {
		resp.Body.Close()
		t.Fatalf("preview: status=%d, want 303", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if !strings.HasPrefix(loc, "/scope") {
		t.Fatalf("preview landed at %q, want /scope", loc)
	}
	return followString(t, c, base+loc)
}

// The first click withdraws nothing. It is the whole point of the two-step act: the
// operator reads the count before the estate moves, and the Seed is still declared
// while they read it.
func TestSeedWithdrawalPreviewDoesNotWithdraw(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "address", "198.51.100.0/24").Body.Close()
	id := f.seeds[0].ID
	f.withdrawalCandidates = []db.ListSeedWithdrawalCandidatesRow{
		candidateSpan(1, "address", "198.51.100.200"),
	}

	page := previewChip(t, ac, base, id)

	if len(f.seeds) != 1 {
		t.Fatalf("the preview must leave the Seed declared, got %+v", f.seeds)
	}
	if len(f.seedWithdrawals) != 0 {
		t.Fatalf("the preview writes no tombstone, got %+v", f.seedWithdrawals)
	}
	if !strings.Contains(page, `action="/seeds/delete"`) {
		t.Errorf("the confirm step must ship the control that commits; body: %s", page)
	}
	if !strings.Contains(page, "198.51.100.0/24") {
		t.Errorf("the confirm step must name the scope; body: %s", page)
	}
	// The commit step's back field, which TestScopeFormsCarryTheSubmittingURL guarded
	// while the delete form shipped on the default render. It only ships inside the
	// confirm state now, so its ADR-0130 §3 contract is asserted here instead.
	if !strings.Contains(page, `name="return" value="/scope"`) {
		t.Errorf("the commit form must carry the submitting URL; body: %s", page)
	}
}

// A count read that fails must not strand the scope. The chip's control reaches the
// withdrawal only through this step, so a 500 here would leave the operator no way to
// withdraw at all — over a count that is advisory by construction (ADR-0134 §5). The
// block says the count did not resolve, and still offers the act.
func TestSeedWithdrawalPreviewDegradesOnAFailedCount(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "address", "198.51.100.0/24").Body.Close()
	id := f.seeds[0].ID
	f.citedErr = errors.New("resolution read failed")

	page := previewChip(t, ac, base, id)

	if !strings.Contains(page, "The count did not resolve") {
		t.Errorf("a failed count must say so rather than render a zero; body: %s", page)
	}
	if !strings.Contains(page, `action="/seeds/delete"`) {
		t.Errorf("a failed count must still offer the withdrawal; body: %s", page)
	}
}

// The count is the fold's count, stated in the fold's own words. The confirm step
// renders message.PreviewSeedWithdrawal's strings verbatim, so the receipt and the
// coverage message the fold writes read as one sentence and `message` gains no copy.
func TestSeedWithdrawalPreviewCountsSubjectsAndTimelines(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "address", "198.51.100.0/24").Body.Close()
	id := f.seeds[0].ID
	// Two subjects over three timelines: the receipt states the two factors, never
	// their product.
	f.withdrawalCandidates = []db.ListSeedWithdrawalCandidatesRow{
		candidateSpan(1, "address", "198.51.100.200"),
		candidateSpan(2, "address", "198.51.100.200"),
		candidateSpan(3, "service", "198.51.100.201:443"),
	}

	page := previewChip(t, ac, base, id)

	want := message.PreviewSeedWithdrawal("198.51.100.0/24", 2, 3)
	if !strings.Contains(page, want.Headline) {
		t.Errorf("headline %q missing; body: %s", want.Headline, page)
	}
	if !strings.Contains(page, want.Loss) {
		t.Errorf("loss %q missing; body: %s", want.Loss, page)
	}
}

// A SECOND live Seed still covering the address keeps it, so the count states what
// leaves rather than what the withdrawn scope contained (ADR-0134 §4). The preview
// applies the survivor against the estate MINUS the scope under withdrawal, which is
// the estate the fold will read.
func TestSeedWithdrawalPreviewSparesASecondLiveSeed(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "address", "198.51.100.0/24").Body.Close()
	declare(t, ac, base, "address", "198.51.100.128/25").Body.Close()
	var id int64
	for _, s := range f.seeds {
		if s.AddressCidr != nil && s.AddressCidr.String() == "198.51.100.0/24" {
			id = s.ID
		}
	}
	f.withdrawalCandidates = []db.ListSeedWithdrawalCandidatesRow{
		candidateSpan(1, "address", "198.51.100.200"), // the /25 still holds it
		candidateSpan(2, "address", "198.51.100.10"),
	}

	page := previewChip(t, ac, base, id)

	want := message.PreviewSeedWithdrawal("198.51.100.0/24", 1, 1)
	if !strings.Contains(page, want.Headline) {
		t.Errorf("headline %q missing — the second Seed's ground must not be counted; body: %s",
			want.Headline, page)
	}
}

// An address custody.Estate.Derive still calls `operator` keeps its timelines, so the
// preview must not count it (ADR-0134 §4). The custody extension lives on a NAME Seed,
// which withdrawing an address Seed leaves standing — such an address is still
// enumerated, still probed and still measured.
func TestSeedWithdrawalPreviewSparesACustodyExtendedAddress(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "address", "93.184.216.0/24").Body.Close()
	declareExtendedZone(t, f, ac, base, "example.com")
	var id int64
	for _, s := range f.seeds {
		if s.AddressCidr != nil && s.AddressCidr.String() == "93.184.216.0/24" {
			id = s.ID
		}
	}
	f.scans = append(f.scans, db.Scan{ID: 99, Kind: scan.EdgeFanoutKind, Enabled: true, CadenceSeconds: 86400})
	f.completedBatchKinds[scan.EdgeFanoutKind] = true
	f.cited = []db.NameCitedAddressesRow{{SubjectKey: "shop.example.com", Address: "93.184.216.10"}}
	// A leaf on ONE registrable domain is not a shared edge, so the extension reaches
	// this address and Derive answers `operator` from the second limb.
	f.measuredEdge("93.184.216.10", string(edgefanout.Presented), edgeDER(t, 1))

	f.withdrawalCandidates = []db.ListSeedWithdrawalCandidatesRow{
		candidateSpan(1, "address", "93.184.216.10"),
		candidateSpan(2, "address", "93.184.216.20"),
	}

	page := previewChip(t, ac, base, id)

	want := message.PreviewSeedWithdrawal("93.184.216.0/24", 1, 1)
	if !strings.Contains(page, want.Headline) {
		t.Errorf("headline %q missing — the extended address must not be counted; body: %s",
			want.Headline, page)
	}
}

// Where the withdrawal takes nothing, NO receipt block renders. The exclusion
// preview's non-firing sentence is not reused: it states that an excluded name which
// still resolves survives and its Gap carries it, which is true there and false here.
// The withdrawal is still available — a zero count is not a refusal.
func TestSeedWithdrawalPreviewOfZeroCountRendersNoReceipt(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "address", "198.51.100.0/24").Body.Close()
	id := f.seeds[0].ID

	page := previewChip(t, ac, base, id)

	if strings.Contains(page, "taken out of the estate") {
		t.Errorf("a zero count states no headline; body: %s", page)
	}
	if strings.Contains(page, "its Gap carries it") {
		t.Errorf("the exclusion's non-firing sentence must not be reused here; body: %s", page)
	}
	if !strings.Contains(page, `action="/seeds/delete"`) {
		t.Errorf("a zero count must still offer the withdrawal; body: %s", page)
	}
}

// A NAME Seed takes the same two-step act, so the same click behaves alike on both
// kinds — but it closes nothing today (the gap ADR-0134 §7 leaves open), so its honest
// count is zero and its confirm step states none.
func TestNameSeedConfirmStepStatesNoCount(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	id := f.seeds[0].ID
	// Rows the fake would return for an address act. A name Seed must not read them.
	f.withdrawalCandidates = []db.ListSeedWithdrawalCandidatesRow{
		candidateSpan(1, "address", "198.51.100.200"),
	}

	page := previewChip(t, ac, base, id)

	if !strings.Contains(page, `action="/seeds/delete"`) {
		t.Errorf("a name chip still gets the confirm step; body: %s", page)
	}
	if strings.Contains(page, "taken out of the estate") {
		t.Errorf("a name withdrawal closes nothing, so it states no count; body: %s", page)
	}
}

// The confirm state is ONE-SHOT. It rides the session flash the exclusion preview
// rides, and the landing GET consumes it, so reloading or navigating away abandons the
// withdrawal and leaves the Seed declared.
func TestSeedWithdrawalConfirmIsOneShot(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "address", "198.51.100.0/24").Body.Close()
	id := f.seeds[0].ID

	if page := previewChip(t, ac, base, id); !strings.Contains(page, `action="/seeds/delete"`) {
		t.Fatalf("the first landing renders the confirm state; body: %s", page)
	}
	again := getBody(t, ac, base+"/scope", http.StatusOK)
	if strings.Contains(again, `action="/seeds/delete"`) {
		t.Errorf("a reload must abandon the confirm state; body: %s", again)
	}
	if len(f.seeds) != 1 {
		t.Fatalf("abandoning leaves the Seed declared, got %+v", f.seeds)
	}
}

// Previewing a chip whose row is already gone redirects cleanly, matching the
// withdrawal's own idempotency: the operator's intent is satisfied either way.
func TestSeedWithdrawalPreviewOfAGoneSeedRedirects(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/seeds/preview", url.Values{"id": {"4242"}})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("stale chip: status=%d, want 303", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); !strings.HasPrefix(loc, "/scope") {
		t.Fatalf("stale chip landed at %q, want /scope", loc)
	}
}

// The preview reads the estate and names what an irreversible act would take, so it is
// admin-only exactly as the withdrawal it fronts is.
func TestSeedWithdrawalPreviewRefusesNonAdmin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declare(t, ac, base, "address", "198.51.100.0/24").Body.Close()
	id := f.seeds[0].ID

	vc := login(t, base, "viewer", "hunter2hunter2")
	resp := postForm(t, vc, base+"/seeds/preview", url.Values{"id": {intStr(id)}})
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusSeeOther && strings.HasPrefix(resp.Header.Get("Location"), "/scope") {
		t.Fatalf("a viewer reached the preview: status=%d location=%q",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}
