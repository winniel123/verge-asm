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

func candidateSpan(id int64, kind, key string) db.ListSeedWithdrawalCandidatesRow {
	return db.ListSeedWithdrawalCandidatesRow{ID: id, SubjectKind: kind, SubjectKey: key}
}

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
	if !strings.Contains(page, `name="return" value="/scope"`) {
		t.Errorf("the commit form must carry the submitting URL; body: %s", page)
	}
}

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

func TestSeedWithdrawalPreviewCountsSubjectsAndTimelines(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "address", "198.51.100.0/24").Body.Close()
	id := f.seeds[0].ID
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
		candidateSpan(1, "address", "198.51.100.200"),
		candidateSpan(2, "address", "198.51.100.10"),
	}

	page := previewChip(t, ac, base, id)

	want := message.PreviewSeedWithdrawal("198.51.100.0/24", 1, 1)
	if !strings.Contains(page, want.Headline) {
		t.Errorf("headline %q missing — the second Seed's ground must not be counted; body: %s",
			want.Headline, page)
	}
}

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
	// A leaf on one registrable domain is not a shared edge, so the extension reaches it.
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
	// An excluded name that still resolves survives; a withdrawn one does not.
	if strings.Contains(page, "its Gap carries it") {
		t.Errorf("the exclusion's non-firing sentence must not be reused here; body: %s", page)
	}
	if !strings.Contains(page, `action="/seeds/delete"`) {
		t.Errorf("a zero count must still offer the withdrawal; body: %s", page)
	}
}

func TestNameSeedWithdrawalPreviewCountsSubjectsAndTimelines(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Both survivor reads drop the withdrawn Seed, or every name count reads zero (ADR-0135 §3).
	declare(t, ac, base, "name", "example.com").Body.Close()
	id := f.seeds[0].ID
	f.nameWithdrawalCandidates = []db.ListNameSeedWithdrawalCandidatesRow{
		{ID: 1, SubjectKey: "www.example.com"},
		{ID: 2, SubjectKey: "www.example.com"},
		{ID: 3, SubjectKey: "api.example.com"},
	}
	// A name Seed reads its own limb, so these address rows are the negative control.
	f.withdrawalCandidates = []db.ListSeedWithdrawalCandidatesRow{
		candidateSpan(9, "address", "198.51.100.200"),
	}

	page := previewChip(t, ac, base, id)

	want := message.PreviewSeedWithdrawal("example.com", 2, 3)
	if !strings.Contains(page, want.Headline) {
		t.Errorf("headline %q missing; body: %s", want.Headline, page)
	}
	if !strings.Contains(page, want.Loss) {
		t.Errorf("loss %q missing; body: %s", want.Loss, page)
	}
}

func TestNameSeedWithdrawalPreviewSparesOnlyASurvivingSeedsAdmission(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	declare(t, ac, base, "name", "example.net").Body.Close()
	id := f.seeds[0].ID
	other := f.seeds[1].ID

	f.nameWithdrawalCandidates = []db.ListNameSeedWithdrawalCandidatesRow{
		{ID: 1, SubjectKey: "www.example.com"},
		{ID: 2, SubjectKey: "api.example.com"},
	}
	f.admitted = []db.AdmittedName{
		{Name: "www.example.com", SeedID: id},
		{Name: "api.example.com", SeedID: other},
	}

	page := previewChip(t, ac, base, id)

	want := message.PreviewSeedWithdrawal("example.com", 1, 1)
	if !strings.Contains(page, want.Headline) {
		t.Errorf("headline %q missing; body: %s", want.Headline, page)
	}
}

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
