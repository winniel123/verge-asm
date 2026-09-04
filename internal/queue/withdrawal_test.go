package queue

import (
	"net/netip"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

func addressExclusion(cidr string) db.ListExclusionsRow {
	p := netip.MustParsePrefix(cidr)
	return db.ListExclusionsRow{Kind: "address", AddressCidr: &p}
}

func nameExclusion(name string) db.ListExclusionsRow {
	return db.ListExclusionsRow{Kind: "name", Name: pgtype.Text{String: name, Valid: true}}
}

func withdrawalRow(id int64, kind, key string) db.ListAddressExclusionWithdrawalsRow {
	return db.ListAddressExclusionWithdrawalsRow{ID: id, SubjectKind: kind, SubjectKey: key}
}

func thirdParty(netip.Addr) custody.Custody { return custody.ThirdParty }

func TestCoveringAddressExclusion(t *testing.T) {
	exclusions := []db.ListExclusionsRow{
		nameExclusion("example.com"),
		addressExclusion("198.51.100.128/25"),
		addressExclusion("2001:db8::/32"),
	}
	tests := []struct {
		addr string
		want string
	}{
		{"198.51.100.200", "198.51.100.128/25"},
		{"198.51.100.127", ""},
		{"203.0.113.5", ""},
		{"2001:db8::1", "2001:db8::/32"},
		{"2001:db9::1", ""},
	}
	for _, tt := range tests {
		got := ""
		if p := coveringAddressExclusion(netip.MustParseAddr(tt.addr), exclusions); p != nil {
			got = p.String()
		}
		if got != tt.want {
			t.Errorf("coveringAddressExclusion(%s) = %q, want %q", tt.addr, got, tt.want)
		}
	}
	if coveringAddressExclusion(netip.MustParseAddr("198.51.100.200"), []db.ListExclusionsRow{nameExclusion("example.com")}) != nil {
		t.Error("a name exclusion covers no Address")
	}
}

func TestComposeAddressWithdrawalsCountsSubjectsAndTimelines(t *testing.T) {
	in := membershipInputs{
		seeds:      []db.ListSeedsRow{addressSeed("198.51.100.0/24")},
		exclusions: []db.ListExclusionsRow{addressExclusion("198.51.100.128/25")},
	}
	rows := []db.ListAddressExclusionWithdrawalsRow{
		withdrawalRow(1, "address", "198.51.100.200"),
		withdrawalRow(2, "address", "198.51.100.200"),
		withdrawalRow(3, "service", "198.51.100.200:443"),
		withdrawalRow(4, "address", "198.51.100.201"),
	}

	spanIDs, receipts := composeAddressWithdrawals(rows, in, thirdParty)

	if len(spanIDs) != 4 {
		t.Fatalf("every listed timeline closes, got %v", spanIDs)
	}
	if len(receipts) != 1 {
		t.Fatalf("one receipt per covering exclusion, got %d", len(receipts))
	}
	r := receipts[0]
	if r.Scope != "198.51.100.0/24" {
		t.Errorf("the message fires at the covering Seed scope, got %q", r.Scope)
	}
	if r.Removed != "198.51.100.128/25" {
		t.Errorf("the removed value is the declared exclusion, got %q", r.Removed)
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

func TestComposeAddressWithdrawalsKeepsAnExtensionReachedAddress(t *testing.T) {
	in := membershipInputs{
		seeds:      []db.ListSeedsRow{addressSeed("198.51.100.0/24")},
		exclusions: []db.ListExclusionsRow{addressExclusion("198.51.100.128/25")},
	}
	rows := []db.ListAddressExclusionWithdrawalsRow{
		withdrawalRow(1, "address", "198.51.100.200"),
		withdrawalRow(2, "address", "198.51.100.201"),
	}
	reached := netip.MustParseAddr("198.51.100.200")
	derive := func(a netip.Addr) custody.Custody {
		if a == reached {
			return custody.Operator
		}
		return custody.ThirdParty
	}

	spanIDs, receipts := composeAddressWithdrawals(rows, in, derive)

	if len(spanIDs) != 1 || spanIDs[0] != 2 {
		t.Errorf("the extension-reached address keeps its timeline, got %v", spanIDs)
	}
	if len(receipts) != 1 || receipts[0].SubjectsWithdrawn != 1 || receipts[0].TimelinesRemoved != 1 {
		t.Errorf("the counts state what actually left, got %+v", receipts)
	}
}

func TestComposeAddressWithdrawalsSilentWhereTheExtensionHoldsEverything(t *testing.T) {
	in := membershipInputs{exclusions: []db.ListExclusionsRow{addressExclusion("198.51.100.128/25")}}
	rows := []db.ListAddressExclusionWithdrawalsRow{withdrawalRow(1, "address", "198.51.100.200")}

	spanIDs, receipts := composeAddressWithdrawals(rows, in, func(netip.Addr) custody.Custody {
		return custody.Operator
	})

	if len(spanIDs) != 0 {
		t.Errorf("nothing closes, got %v", spanIDs)
	}
	if len(receipts) != 0 {
		t.Errorf("nothing is collected, so no message fires, got %+v", receipts)
	}
}

func TestComposeAddressWithdrawalsGroupsPerExclusion(t *testing.T) {
	in := membershipInputs{
		seeds: []db.ListSeedsRow{addressSeed("198.51.100.0/24"), addressSeed("203.0.113.0/24")},
		exclusions: []db.ListExclusionsRow{
			addressExclusion("198.51.100.128/25"),
			addressExclusion("203.0.113.0/28"),
		},
	}
	rows := []db.ListAddressExclusionWithdrawalsRow{
		withdrawalRow(1, "address", "198.51.100.200"),
		withdrawalRow(2, "address", "203.0.113.5"),
	}

	_, receipts := composeAddressWithdrawals(rows, in, thirdParty)

	if len(receipts) != 2 {
		t.Fatalf("want 2 receipts, got %d", len(receipts))
	}
	if receipts[0].Removed != "198.51.100.128/25" || receipts[1].Removed != "203.0.113.0/28" {
		t.Errorf("receipts are keyed by their own exclusion, got %+v", receipts)
	}
	if receipts[0].Scope != "198.51.100.0/24" || receipts[1].Scope != "203.0.113.0/24" {
		t.Errorf("each fires at its own covering scope, got %+v", receipts)
	}
}

func TestNarrowingScope(t *testing.T) {
	seeds := []db.ListSeedsRow{addressSeed("198.51.0.0/16"), addressSeed("198.51.100.0/24")}
	if got := narrowingScope(netip.MustParsePrefix("198.51.100.128/25"), seeds); got != "198.51.100.0/24" {
		t.Errorf("the most specific covering scope wins, got %q", got)
	}
	if got := narrowingScope(netip.MustParsePrefix("203.0.113.0/28"), seeds); got != "203.0.113.0/28" {
		t.Errorf("with no covering scope the excluded value is the site, got %q", got)
	}
	if got := narrowingScope(netip.MustParsePrefix("198.51.100.128/25"), nil); got != "198.51.100.128/25" {
		t.Errorf("with no seeds at all the excluded value is the site, got %q", got)
	}
}

func TestComposeAddressWithdrawalsDropsAnUnattributableRow(t *testing.T) {
	in := membershipInputs{exclusions: []db.ListExclusionsRow{addressExclusion("198.51.100.128/25")}}
	rows := []db.ListAddressExclusionWithdrawalsRow{
		withdrawalRow(1, "address", "not-an-address"),
		withdrawalRow(2, "address", "203.0.113.5"),
		withdrawalRow(3, "name", "www.example.com"),
		withdrawalRow(4, "address", "198.51.100.200"),
	}

	spanIDs, receipts := composeAddressWithdrawals(rows, in, thirdParty)

	if len(spanIDs) != 1 || spanIDs[0] != 4 {
		t.Errorf("only the attributable timeline closes, got %v", spanIDs)
	}
	if len(receipts) != 1 || receipts[0].TimelinesRemoved != 1 {
		t.Errorf("the counts state what closed, got %+v", receipts)
	}
}

func TestComposeAddressWithdrawalsIsEmptyWithNoRows(t *testing.T) {
	spanIDs, receipts := composeAddressWithdrawals(nil, membershipInputs{
		exclusions: []db.ListExclusionsRow{addressExclusion("198.51.100.128/25")},
	}, thirdParty)
	if len(spanIDs) != 0 || len(receipts) != 0 {
		t.Errorf("an empty answer closes and collects nothing, got %v / %+v", spanIDs, receipts)
	}
}

func TestHasAddressExclusion(t *testing.T) {
	if (membershipInputs{}).hasAddressExclusion() {
		t.Error("an empty corpus declares no address exclusion")
	}
	only := membershipInputs{exclusions: []db.ListExclusionsRow{nameExclusion("example.com"), subtreeExclusion("x.com")}}
	if only.hasAddressExclusion() {
		t.Error("name and subtree exclusions are not address exclusions")
	}
	with := membershipInputs{exclusions: []db.ListExclusionsRow{nameExclusion("example.com"), addressExclusion("198.51.100.0/24")}}
	if !with.hasAddressExclusion() {
		t.Error("a declared address exclusion is found")
	}
}

func TestComposeAddressWithdrawalsRendersThroughPreviewNarrowing(t *testing.T) {
	in := membershipInputs{
		seeds:      []db.ListSeedsRow{addressSeed("198.51.100.0/24")},
		exclusions: []db.ListExclusionsRow{addressExclusion("198.51.100.128/25")},
	}
	_, receipts := composeAddressWithdrawals(
		[]db.ListAddressExclusionWithdrawalsRow{withdrawalRow(1, "address", "198.51.100.200")}, in, thirdParty)

	want := message.PreviewNarrowing("198.51.100.0/24", "198.51.100.128/25", 1, 1)
	if receipts[0] != want {
		t.Errorf("the fold's receipt is PreviewNarrowing's own:\n got %+v\nwant %+v", receipts[0], want)
	}
}
