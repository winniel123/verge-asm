package queue

import (
	"net/netip"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
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

// addressExcluded is the address analogue nameExcluded had no twin for (#1032):
// containment is the family-matched prefix test, and a name or subtree exclusion
// covers no Address at all.
func TestAddressExcluded(t *testing.T) {
	exclusions := []db.ListExclusionsRow{
		nameExclusion("example.com"),
		addressExclusion("198.51.100.128/25"),
		addressExclusion("2001:db8::/32"),
	}
	tests := []struct {
		addr string
		want bool
	}{
		{"198.51.100.200", true},
		{"198.51.100.127", false}, // below the excluded half
		{"203.0.113.5", false},
		{"2001:db8::1", true},
		{"2001:db9::1", false},
	}
	for _, tt := range tests {
		if got := addressExcluded(netip.MustParseAddr(tt.addr), exclusions); got != tt.want {
			t.Errorf("addressExcluded(%s) = %v, want %v", tt.addr, got, tt.want)
		}
	}
	// A name exclusion alone excludes no address, whatever the address is.
	if addressExcluded(netip.MustParseAddr("198.51.100.200"), []db.ListExclusionsRow{nameExclusion("example.com")}) {
		t.Error("a name exclusion covers no Address")
	}
}

// coveringExclusionKey names the mover for an ADDRESS departure too. Before #1032
// it skipped every exclusion row carrying no name, so an address exclusion that
// descoped a subject could name no Source.
func TestCoveringExclusionKeyNamesTheAddressExclusion(t *testing.T) {
	exclusions := []db.ListExclusionsRow{
		subtreeExclusion("example.com"),
		addressExclusion("198.51.100.128/25"),
	}
	tests := []struct {
		name        string
		subjectKind string
		subjectKey  string
		reason      drift.ClosureReason
		want        string
	}{
		{"an address inside the exclusion", "address", "198.51.100.200", drift.ReasonDescoped, "198.51.100.128/25"},
		{"a service on that address", "service", "198.51.100.200:443", drift.ReasonDescoped, "198.51.100.128/25"},
		{"an endpoint on that address", "endpoint", "198.51.100.200:80", drift.ReasonDescoped, "198.51.100.128/25"},
		{"an address no exclusion covers", "address", "203.0.113.5", drift.ReasonDescoped, ""},
		{"a name still reads the name limb", "name", "www.example.com", drift.ReasonDescoped, "example.com"},
		{"a world withdrawal names no mover", "address", "198.51.100.200", drift.ReasonMeasuredAbsent, ""},
		{"an unparseable address key", "address", "not-an-address", drift.ReasonDescoped, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := coveringExclusionKey(tt.subjectKind, tt.subjectKey, tt.reason, exclusions); got != tt.want {
				t.Errorf("coveringExclusionKey = %q, want %q", got, tt.want)
			}
		})
	}
}

// The withdrawal states its two counts with their factors, never as a product: a
// subject holding three timelines is ONE subject withdrawn and THREE timelines
// removed (message.NarrowingReceipt). The message fires at the declared Seed scope
// the excluded ground sits inside, which is the site the preview already names.
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

	spanIDs, narrowings := composeAddressWithdrawals(rows, in)

	if len(spanIDs) != 4 {
		t.Fatalf("every listed timeline closes, got %v", spanIDs)
	}
	if len(narrowings) != 1 {
		t.Fatalf("one narrowing per covering exclusion, got %d", len(narrowings))
	}
	n := narrowings[0]
	if n.Scope != "198.51.100.0/24" {
		t.Errorf("the message fires at the covering Seed scope, got %q", n.Scope)
	}
	if n.Removed != "198.51.100.128/25" {
		t.Errorf("the removed value is the declared exclusion, got %q", n.Removed)
	}
	if n.SubjectsWithdrawn != 3 {
		t.Errorf("three distinct subjects left, got %d", n.SubjectsWithdrawn)
	}
	if n.TimelinesRemoved != 4 {
		t.Errorf("four timelines closed, got %d", n.TimelinesRemoved)
	}
}

// Two declared exclusions are two acts, so they are two narrowings and two
// messages — never one merged count over a scope neither of them names.
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

	_, narrowings := composeAddressWithdrawals(rows, in)

	if len(narrowings) != 2 {
		t.Fatalf("want 2 narrowings, got %d", len(narrowings))
	}
	if narrowings[0].Removed != "198.51.100.128/25" || narrowings[1].Removed != "203.0.113.0/28" {
		t.Errorf("narrowings are keyed by their own exclusion, got %+v", narrowings)
	}
	if narrowings[0].Scope != "198.51.100.0/24" || narrowings[1].Scope != "203.0.113.0/24" {
		t.Errorf("each fires at its own covering scope, got %+v", narrowings)
	}
}

// The most specific covering scope wins where declared scopes nest, mirroring
// FindCoveringAddressSeed, so the act and the preview name the same firing site.
// Where no declared scope covers the exclusion the excluded value is the site.
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

// A row this fold cannot attribute to a declared Exclusion is DROPPED, not closed.
// A closure with no mover to name is a withdrawal the operator cannot trace back to
// their own act, so the safe reading is to leave the timeline open.
func TestComposeAddressWithdrawalsDropsAnUnattributableRow(t *testing.T) {
	in := membershipInputs{exclusions: []db.ListExclusionsRow{addressExclusion("198.51.100.128/25")}}
	rows := []db.ListAddressExclusionWithdrawalsRow{
		withdrawalRow(1, "address", "not-an-address"),
		withdrawalRow(2, "address", "203.0.113.5"), // no exclusion covers it
		withdrawalRow(3, "name", "www.example.com"),
		withdrawalRow(4, "address", "198.51.100.200"),
	}

	spanIDs, narrowings := composeAddressWithdrawals(rows, in)

	if len(spanIDs) != 1 || spanIDs[0] != 4 {
		t.Errorf("only the attributable timeline closes, got %v", spanIDs)
	}
	if len(narrowings) != 1 || narrowings[0].TimelinesRemoved != 1 {
		t.Errorf("the counts state what closed, got %+v", narrowings)
	}
}

// Nothing withdrawn is nothing to do: no span closes and no narrowing is collected,
// which is what makes the fold idempotent. The closure is what removes the row from
// the query's answer, so the batch after the withdrawal reads none.
func TestComposeAddressWithdrawalsIsEmptyWithNoRows(t *testing.T) {
	spanIDs, narrowings := composeAddressWithdrawals(nil, membershipInputs{
		exclusions: []db.ListExclusionsRow{addressExclusion("198.51.100.128/25")},
	})
	if len(spanIDs) != 0 || len(narrowings) != 0 {
		t.Errorf("an empty answer closes and collects nothing, got %v / %+v", spanIDs, narrowings)
	}
}
