package message

import (
	"strings"
	"testing"
)

func TestANameRootCountsOnTheAddressItCites(t *testing.T) {
	census := NewCensus(
		CensusEntry{Kind: "service", Key: "10.0.0.1:443/tcp"},
		CensusEntry{Kind: "endpoint", Key: "@10.0.0.1:443/tcp"},
	)
	want := "example.com entered the estate · 1 endpoint + 1 service · 2 timelines opened on an address it cites"
	if got := membershipHeadline(EntryAppeared, "name", "example.com", census); got != want {
		t.Errorf("a Name reaches its sub-tree by citation, and the headline says so\n got %q\nwant %q", got, want)
	}
}

func TestASecondNameOnOneAddressClaimsNoMoreThanTheFirst(t *testing.T) {
	// k Names citing one address each census all n subjects, so neither may claim them (#1809).
	census := NewCensus(
		CensusEntry{Kind: "service", Key: "10.0.0.1:443/tcp"},
		CensusEntry{Kind: "endpoint", Key: "@10.0.0.1:443/tcp"},
	)
	first := membershipHeadline(EntryAppeared, "name", "example.com", census)
	second := membershipHeadline(EntryAppeared, "name", "second.test", census)
	for _, h := range []string{first, second} {
		if want := "opened on an address it cites"; !strings.Contains(h, want) {
			t.Errorf("headline %q states the ground it counts over, want %q", h, want)
		}
		if strings.Contains(h, "beneath it") {
			t.Errorf("a Name owns none of a shared address, so it claims nothing beneath it: %q", h)
		}
	}
}

func TestAnAddressRootKeepsBeneathIt(t *testing.T) {
	// The address is the ground its sub-tree sits on, so the count is its own (ADR-1806 §4).
	census := NewCensus(
		CensusEntry{Kind: "service", Key: "10.0.0.1:443/tcp"},
		CensusEntry{Kind: "endpoint", Key: "@10.0.0.1:443/tcp"},
	)
	want := "10.0.0.1 entered the estate · 1 endpoint + 1 service · 2 timelines opened beneath it"
	if got := membershipHeadline(EntryAppeared, "address", "10.0.0.1", census); got != want {
		t.Errorf("an Address root's ground is the address\n got %q\nwant %q", got, want)
	}
}

func TestAnEmptyCensusStatesItsGroundToo(t *testing.T) {
	want := "example.com came into view · 0 timelines opened on an address it cites"
	if got := membershipHeadline(EntryRevealed, "name", "example.com", NewCensus()); got != want {
		t.Errorf("an empty census still names the ground it counted over\n got %q\nwant %q", got, want)
	}
}

func TestAReleasedHeadlineCarriesTheRootsGround(t *testing.T) {
	// Release recomputes the clause from the basis, which carries the root kind (ADR-1806 §4).
	census := NewCensus(CensusEntry{Kind: "endpoint", Key: "@10.0.0.1:443/tcp"})
	cause := membershipCauseClause(EntryAppeared, "example.com")
	want := "example.com entered the estate · 1 endpoint · 1 timeline opened on an address it cites"
	if got := ReleasedMembershipHeadline(cause, "name", census); got != want {
		t.Errorf("the released headline reads the basis's root kind\n got %q\nwant %q", got, want)
	}
}
