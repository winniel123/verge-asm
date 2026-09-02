package queue

import (
	"net/netip"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

func seedTombstone(id int64, cidr string) db.ListPendingSeedWithdrawalsRow {
	return db.ListPendingSeedWithdrawalsRow{ID: id, AddressCidr: netip.MustParsePrefix(cidr)}
}

func candidateRow(id int64, kind, key string) db.ListSeedWithdrawalCandidatesRow {
	return db.ListSeedWithdrawalCandidatesRow{ID: id, SubjectKind: kind, SubjectKey: key}
}

// coveringSeedWithdrawal is the tombstone analogue of coveringAddressExclusion:
// containment is the family-matched prefix test, so an IPv4 address is never read
// as inside an IPv6 withdrawal.
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

// The withdrawal states its two counts with their factors, never as a product: a
// subject holding two timelines is ONE subject withdrawn and TWO timelines removed
// (message.NarrowingReceipt). The message fires at the withdrawn CIDR itself,
// because an address Seed's display scope IS its CIDR — the scope that moved and
// the ground that left are one object.
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

// ADR-0134 §4, survivor two: an address a LIVE Seed still covers does not leave.
// This is the second declared Seed over the same ground — the case that separates
// this act from the exclusion act, which has no such overlap rule.
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

// ADR-0134 §4, survivor two again, and the reason it must read the LIVE Seed corpus
// rather than the tombstone: withdraw a scope, declare it again, and the addresses
// re-enter through the Seed limb while a stale tombstone still names the CIDR. The
// tombstone must not close ground that is declared again.
func TestComposeSeedWithdrawalsKeepsARedeclaredScope(t *testing.T) {
	pending := []db.ListPendingSeedWithdrawalsRow{seedTombstone(1, "198.51.100.0/24")}
	in := membershipInputs{seeds: []db.ListSeedsRow{addressSeed("198.51.100.0/24")}}
	rows := []db.ListSeedWithdrawalCandidatesRow{candidateRow(1, "address", "198.51.100.200")}

	spanIDs, _ := composeSeedWithdrawals(rows, pending, in, thirdParty)

	if len(spanIDs) != 0 {
		t.Errorf("the re-declared scope holds its addresses, got %v", spanIDs)
	}
}

// ADR-0134 §4, survivor three: an address custody.Estate.Derive still calls
// `operator` is reached by a custody extension, so it is still enumerated, still
// probed and still measured. Closing its timelines would reopen them on the next
// batch and close them again on the one after — a `descoped` departure every
// cadence over an address the gate never stopped probing.
func TestComposeSeedWithdrawalsKeepsAnExtensionReachedAddress(t *testing.T) {
	pending := []db.ListPendingSeedWithdrawalsRow{seedTombstone(1, "198.51.100.0/24")}
	rows := []db.ListSeedWithdrawalCandidatesRow{
		candidateRow(1, "address", "198.51.100.200"), // the extension reaches this one
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

// Two withdrawn scopes are two acts, so they are two receipts and two messages —
// never one merged count over a scope neither of them names.
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

// Two tombstones naming the same withdrawn CIDR — withdraw, re-declare, withdraw
// again before a job completes — state ONE act to the operator, because the
// receipts are keyed by CIDR and not by tombstone id.
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

// A row this fold cannot attribute to a pending tombstone is DROPPED, not closed.
// A closure with no mover to name is a withdrawal the operator cannot trace back to
// their own act, so the safe reading is to leave the timeline open.
func TestComposeSeedWithdrawalsDropsAnUnattributableRow(t *testing.T) {
	pending := []db.ListPendingSeedWithdrawalsRow{seedTombstone(1, "198.51.100.0/24")}
	rows := []db.ListSeedWithdrawalCandidatesRow{
		candidateRow(1, "address", "not-an-address"),
		candidateRow(2, "address", "203.0.113.5"), // no tombstone covers it
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

// Nothing withdrawn is nothing to do. A tombstone over ground that holds no open
// timeline closes nothing and collects no receipt — and it is still spent, because
// it has taken everything it was going to take.
func TestComposeSeedWithdrawalsIsEmptyWithNoRows(t *testing.T) {
	spanIDs, receipts := composeSeedWithdrawals(nil,
		[]db.ListPendingSeedWithdrawalsRow{seedTombstone(1, "198.51.100.0/24")},
		membershipInputs{}, thirdParty)
	if len(spanIDs) != 0 || len(receipts) != 0 {
		t.Errorf("an empty answer closes and collects nothing, got %v / %+v", spanIDs, receipts)
	}
}

// The receipt the fold collects is the SAME value the producer fires from, so one
// constructor renders the act's sentence.
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
