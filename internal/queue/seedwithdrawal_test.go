package queue

import (
	"net/netip"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

func seedTombstone(id int64, cidr string) db.ListPendingSeedWithdrawalsRow {
	p := netip.MustParsePrefix(cidr)
	return db.ListPendingSeedWithdrawalsRow{ID: id, AddressCidr: &p}
}

func candidateRow(id int64, kind, key string) db.ListSeedWithdrawalCandidatesRow {
	return db.ListSeedWithdrawalCandidatesRow{ID: id, SubjectKind: kind, SubjectKey: key}
}

func TestCoveringSeedWithdrawal(t *testing.T) {
	pending := []db.ListPendingSeedWithdrawalsRow{
		seedTombstone(1, "198.51.100.0/24"),
		seedTombstone(2, "2001:db8::/32"),
	}
	tests := []struct {
		addr string
		want string
	}{
		{"198.51.100.200", "198.51.100.0/24"},
		{"203.0.113.5", ""},
		{"2001:db8::1", "2001:db8::/32"},
		{"2001:db9::1", ""},
	}
	for _, tt := range tests {
		got := ""
		if p := coveringSeedWithdrawal(netip.MustParseAddr(tt.addr), pending); p != nil {
			got = p.String()
		}
		if got != tt.want {
			t.Errorf("coveringSeedWithdrawal(%s) = %q, want %q", tt.addr, got, tt.want)
		}
	}
	if coveringSeedWithdrawal(netip.MustParseAddr("198.51.100.200"), nil) != nil {
		t.Error("no tombstone covers nothing")
	}
}

func TestComposeSeedWithdrawalsCountsSubjectsAndTimelines(t *testing.T) {
	pending := []db.ListPendingSeedWithdrawalsRow{seedTombstone(1, "198.51.100.0/24")}
	rows := []db.ListSeedWithdrawalCandidatesRow{
		candidateRow(1, "address", "198.51.100.200"),
		candidateRow(2, "address", "198.51.100.200"),
		candidateRow(3, "service", "198.51.100.200:443"),
		candidateRow(4, "address", "198.51.100.201"),
	}

	spanIDs, receipts := composeSeedWithdrawals(rows, pending, membershipInputs{}, thirdParty)

	if len(spanIDs) != 4 {
		t.Fatalf("every listed timeline closes, got %v", spanIDs)
	}
	if len(receipts) != 1 {
		t.Fatalf("one receipt per withdrawn scope, got %d", len(receipts))
	}
	r := receipts[0]
	if r.Scope != "198.51.100.0/24" || r.Removed != "198.51.100.0/24" {
		t.Errorf("the scope and the removed ground are the withdrawn CIDR, got %+v", r)
	}
	if r.SubjectsWithdrawn != 3 {
		t.Errorf("three distinct subjects left, got %d", r.SubjectsWithdrawn)
	}
	if r.TimelinesRemoved != 4 {
		t.Errorf("four timelines closed, got %d", r.TimelinesRemoved)
	}
	if !r.Fires {
		t.Error("an inhabited withdrawal fires")
	}
}

func TestComposeSeedWithdrawalsKeepsGroundASecondSeedCovers(t *testing.T) {
	pending := []db.ListPendingSeedWithdrawalsRow{seedTombstone(1, "198.51.100.0/25")}
	in := membershipInputs{seeds: []db.ListSeedsRow{addressSeed("198.51.100.0/24")}}
	rows := []db.ListSeedWithdrawalCandidatesRow{candidateRow(1, "address", "198.51.100.10")}

	spanIDs, receipts := composeSeedWithdrawals(rows, pending, in, thirdParty)

	if len(spanIDs) != 0 {
		t.Errorf("a wider live Seed still declares the address, got %v", spanIDs)
	}
	if len(receipts) != 0 {
		t.Errorf("nothing left, so no message fires, got %+v", receipts)
	}
}

func TestComposeSeedWithdrawalsKeepsARedeclaredScope(t *testing.T) {
	pending := []db.ListPendingSeedWithdrawalsRow{seedTombstone(1, "198.51.100.0/24")}
	in := membershipInputs{seeds: []db.ListSeedsRow{addressSeed("198.51.100.0/24")}}
	rows := []db.ListSeedWithdrawalCandidatesRow{candidateRow(1, "address", "198.51.100.200")}

	spanIDs, _ := composeSeedWithdrawals(rows, pending, in, thirdParty)

	if len(spanIDs) != 0 {
		t.Errorf("the re-declared scope holds its addresses, got %v", spanIDs)
	}
}

func TestComposeSeedWithdrawalsKeepsAnExtensionReachedAddress(t *testing.T) {
	pending := []db.ListPendingSeedWithdrawalsRow{seedTombstone(1, "198.51.100.0/24")}
	rows := []db.ListSeedWithdrawalCandidatesRow{
		candidateRow(1, "address", "198.51.100.200"),
		candidateRow(2, "address", "198.51.100.201"),
	}
	reached := netip.MustParseAddr("198.51.100.200")
	derive := func(a netip.Addr) custody.Custody {
		if a == reached {
			return custody.Operator
		}
		return custody.ThirdParty
	}

	spanIDs, receipts := composeSeedWithdrawals(rows, pending, membershipInputs{}, derive)

	if len(spanIDs) != 1 || spanIDs[0] != 2 {
		t.Errorf("the extension-reached address keeps its timeline, got %v", spanIDs)
	}
	if len(receipts) != 1 || receipts[0].SubjectsWithdrawn != 1 || receipts[0].TimelinesRemoved != 1 {
		t.Errorf("the counts state what actually left, got %+v", receipts)
	}
}

func TestComposeSeedWithdrawalsGroupsPerWithdrawnScope(t *testing.T) {
	pending := []db.ListPendingSeedWithdrawalsRow{
		seedTombstone(1, "198.51.100.0/24"),
		seedTombstone(2, "203.0.113.0/24"),
	}
	rows := []db.ListSeedWithdrawalCandidatesRow{
		candidateRow(1, "address", "198.51.100.200"),
		candidateRow(2, "address", "203.0.113.5"),
	}

	_, receipts := composeSeedWithdrawals(rows, pending, membershipInputs{}, thirdParty)

	if len(receipts) != 2 {
		t.Fatalf("want 2 receipts, got %d", len(receipts))
	}
	if receipts[0].Scope != "198.51.100.0/24" || receipts[1].Scope != "203.0.113.0/24" {
		t.Errorf("each fires at its own withdrawn scope, got %+v", receipts)
	}
}

func TestComposeSeedWithdrawalsMergesDuplicateTombstones(t *testing.T) {
	pending := []db.ListPendingSeedWithdrawalsRow{
		seedTombstone(1, "198.51.100.0/24"),
		seedTombstone(2, "198.51.100.0/24"),
	}
	rows := []db.ListSeedWithdrawalCandidatesRow{
		candidateRow(1, "address", "198.51.100.200"),
		candidateRow(2, "address", "198.51.100.201"),
	}

	_, receipts := composeSeedWithdrawals(rows, pending, membershipInputs{}, thirdParty)

	if len(receipts) != 1 {
		t.Fatalf("one scope is one act however many tombstones name it, got %d", len(receipts))
	}
	if receipts[0].SubjectsWithdrawn != 2 || receipts[0].TimelinesRemoved != 2 {
		t.Errorf("the counts are stated once over the whole scope, got %+v", receipts[0])
	}
}

func TestComposeSeedWithdrawalsDropsAnUnattributableRow(t *testing.T) {
	pending := []db.ListPendingSeedWithdrawalsRow{seedTombstone(1, "198.51.100.0/24")}
	rows := []db.ListSeedWithdrawalCandidatesRow{
		candidateRow(1, "address", "not-an-address"),
		candidateRow(2, "address", "203.0.113.5"),
		candidateRow(3, "name", "www.example.com"),
		candidateRow(4, "address", "198.51.100.200"),
	}

	spanIDs, receipts := composeSeedWithdrawals(rows, pending, membershipInputs{}, thirdParty)

	if len(spanIDs) != 1 || spanIDs[0] != 4 {
		t.Errorf("only the attributable timeline closes, got %v", spanIDs)
	}
	if len(receipts) != 1 || receipts[0].TimelinesRemoved != 1 {
		t.Errorf("the counts state what closed, got %+v", receipts)
	}
}

func TestComposeSeedWithdrawalsIsEmptyWithNoRows(t *testing.T) {
	spanIDs, receipts := composeSeedWithdrawals(nil,
		[]db.ListPendingSeedWithdrawalsRow{seedTombstone(1, "198.51.100.0/24")},
		membershipInputs{}, thirdParty)
	if len(spanIDs) != 0 || len(receipts) != 0 {
		t.Errorf("an empty answer closes and collects nothing, got %v / %+v", spanIDs, receipts)
	}
}

func TestComposeSeedWithdrawalsRendersThroughPreviewSeedWithdrawal(t *testing.T) {
	_, receipts := composeSeedWithdrawals(
		[]db.ListSeedWithdrawalCandidatesRow{candidateRow(1, "address", "198.51.100.200")},
		[]db.ListPendingSeedWithdrawalsRow{seedTombstone(1, "198.51.100.0/24")},
		membershipInputs{}, thirdParty)

	want := message.PreviewSeedWithdrawal("198.51.100.0/24", 1, 1)
	if receipts[0] != want {
		t.Errorf("the fold's receipt is PreviewSeedWithdrawal's own:\n got %+v\nwant %+v", receipts[0], want)
	}
}
