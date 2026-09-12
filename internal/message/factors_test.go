package message

import (
	"testing"
	"time"
)

func TestMembershipHeadlineRendersFactorsThenTotal(t *testing.T) {
	census := NewCensus(
		CensusEntry{Kind: "service", Key: "10.0.0.1:443/tcp"},
		CensusEntry{Kind: "endpoint", Key: "a.example.com@10.0.0.1:443/tcp"},
		CensusEntry{Kind: "service", Key: "10.0.0.1:22/tcp"},
		CensusEntry{Kind: "endpoint", Key: "b.example.com@10.0.0.1:443/tcp"},
		CensusEntry{Kind: "endpoint", Key: "c.example.com@10.0.0.1:443/tcp"},
		CensusEntry{Kind: "endpoint", Key: "d.example.com@10.0.0.1:443/tcp"},
	)
	want := "example.com entered the estate · 4 endpoints + 2 services · 6 timelines opened on an address it cites"
	if got := membershipHeadline(EntryAppeared, "name", "example.com", census); got != want {
		t.Errorf("membership headline\n got %q\nwant %q", got, want)
	}
}

func TestMembershipHeadlineOmitsAnEmptyKind(t *testing.T) {
	census := NewCensus(CensusEntry{Kind: "endpoint", Key: "a.example.com@10.0.0.1:443/tcp"})
	want := "example.com returned to the estate · 1 endpoint · 1 timeline opened on an address it cites"
	if got := membershipHeadline(EntryReturned, "name", "example.com", census); got != want {
		t.Errorf("one-kind headline\n got %q\nwant %q", got, want)
	}
	want = "203.0.113.0/24 came into view · 0 timelines opened beneath it"
	if got := membershipHeadline(EntryRevealed, "address", "203.0.113.0/24", NewCensus()); got != want {
		t.Errorf("empty census carries no factor clause\n got %q\nwant %q", got, want)
	}
}

func TestMembershipFactorsSumToTheTotal(t *testing.T) {
	census := NewCensus(
		CensusEntry{Kind: "service", Key: "10.0.0.1:443/tcp"},
		CensusEntry{Kind: "endpoint", Key: "a.example.com@10.0.0.1:443/tcp"},
		CensusEntry{Kind: "address", Key: "10.0.0.1"},
	)
	want := "example.com entered the estate · 1 address + 1 endpoint + 1 service · 3 timelines opened on an address it cites"
	if got := membershipHeadline(EntryAppeared, "name", "example.com", census); got != want {
		t.Errorf("every kind present is a factor, so the factors sum to the total\n got %q\nwant %q", got, want)
	}
}

func TestMembershipHeadlineIsItsTwoClauses(t *testing.T) {
	// The release path appends the census clause without recomputing the cause (ADR-1806 §2).
	census := NewCensus(CensusEntry{Kind: "service", Key: "10.0.0.1:443/tcp"})
	cause := membershipCauseClause(EntryAppeared, "example.com")
	if cause != "example.com entered the estate" {
		t.Errorf("the cause clause is a sentence on its own, got %q", cause)
	}
	if got := cause + membershipCensusClause("name", census); got != membershipHeadline(EntryAppeared, "name", "example.com", census) {
		t.Errorf("the two clauses compose the headline, got %q", got)
	}
}

func TestAHeldMembershipStatesNoCount(t *testing.T) {
	// A held row states no census, so its headline may not state one either (ADR-1806 §2).
	m := HeldMembership(EntryAppeared, "name", "example.com", "", CensusPending{AfterBatch: 7}, time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))
	if m == nil {
		t.Fatal("a Name root fires a membership message")
	}
	if m.Headline != "example.com entered the estate" {
		t.Errorf("the headline holds the cause clause alone, got %q", m.Headline)
	}
	if m.Census != nil {
		t.Errorf("a held firing carries no census, got %+v", m.Census)
	}
	if m.CensusPending == nil || m.CensusPending.AfterBatch != 7 {
		t.Errorf("the firing states the batch its release waits on, got %+v", m.CensusPending)
	}
	if m.Class != ClassDrift || m.FiredAt != "example.com" {
		t.Errorf("holding the census moves neither the class nor the subject, got %+v", m)
	}
}

func TestFlagshipHeadlineStaysABareFacetCount(t *testing.T) {
	census := NewCensus(
		CensusEntry{Kind: KindFacet, Key: "certificate"},
		CensusEntry{Kind: KindFacet, Key: "http-identity"},
	)
	want := "198.51.100.1:443/tcp reached from the internet · 2 facets opened beneath it"
	if got := flagshipHeadline("198.51.100.1:443/tcp", census); got != want {
		t.Errorf("a facet count's factors are its rows, which the body may not carry\n got %q\nwant %q", got, want)
	}
}
