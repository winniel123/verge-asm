package message

import "testing"

func TestMembershipHeadlineRendersFactorsThenTotal(t *testing.T) {
	census := NewCensus(
		CensusEntry{Kind: "service", Key: "10.0.0.1:443/tcp"},
		CensusEntry{Kind: "endpoint", Key: "a.example.com@10.0.0.1:443/tcp"},
		CensusEntry{Kind: "service", Key: "10.0.0.1:22/tcp"},
		CensusEntry{Kind: "endpoint", Key: "b.example.com@10.0.0.1:443/tcp"},
		CensusEntry{Kind: "endpoint", Key: "c.example.com@10.0.0.1:443/tcp"},
		CensusEntry{Kind: "endpoint", Key: "d.example.com@10.0.0.1:443/tcp"},
	)
	want := "example.com entered the estate · 4 endpoints + 2 services · 6 timelines opened beneath it"
	if got := membershipHeadline(EntryAppeared, "example.com", census); got != want {
		t.Errorf("membership headline\n got %q\nwant %q", got, want)
	}
}

func TestMembershipHeadlineOmitsAnEmptyKind(t *testing.T) {
	census := NewCensus(CensusEntry{Kind: "endpoint", Key: "a.example.com@10.0.0.1:443/tcp"})
	want := "example.com returned to the estate · 1 endpoint · 1 timeline opened beneath it"
	if got := membershipHeadline(EntryReturned, "example.com", census); got != want {
		t.Errorf("one-kind headline\n got %q\nwant %q", got, want)
	}
	want = "203.0.113.0/24 came into view · 0 timelines opened beneath it"
	if got := membershipHeadline(EntryRevealed, "203.0.113.0/24", NewCensus()); got != want {
		t.Errorf("empty census carries no factor clause\n got %q\nwant %q", got, want)
	}
}

func TestMembershipFactorsSumToTheTotal(t *testing.T) {
	census := NewCensus(
		CensusEntry{Kind: "service", Key: "10.0.0.1:443/tcp"},
		CensusEntry{Kind: "endpoint", Key: "a.example.com@10.0.0.1:443/tcp"},
		CensusEntry{Kind: "address", Key: "10.0.0.1"},
	)
	want := "example.com entered the estate · 1 address + 1 endpoint + 1 service · 3 timelines opened beneath it"
	if got := membershipHeadline(EntryAppeared, "example.com", census); got != want {
		t.Errorf("every kind present is a factor, so the factors sum to the total\n got %q\nwant %q", got, want)
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
