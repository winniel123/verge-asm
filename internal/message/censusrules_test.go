package message

import (
	"strings"
	"testing"
)

func TestFlagshipHeadlineNamesTheRulesThatOpenedAtFired(t *testing.T) {
	census := NewCensus(
		CensusEntry{Kind: KindRule, Key: "sensitive-port-reached-from-internet", Detail: "3306/tcp"},
		CensusEntry{Kind: KindFacet, Key: "certificate"},
		CensusEntry{Kind: KindRule, Key: "certificate-expired"},
	)
	msg := Flagship(ReachMove{
		ServiceKey: "198.51.100.1:3306/tcp", Class: ClassInternet,
		From: NotReached, To: Reached,
	}, census, t0)
	if msg == nil {
		t.Fatal("flagship must fire")
	}
	if !strings.Contains(msg.Headline, "1 facet opened beneath it") {
		t.Errorf("the facet count must count facet entries only, got %q", msg.Headline)
	}
	if !strings.Contains(msg.Headline, "2 rules opened at fired") {
		t.Errorf("the headline must count the rules that opened at fired, got %q", msg.Headline)
	}
	if !strings.Contains(msg.Headline, "sensitive-port-reached-from-internet (3306/tcp)") {
		t.Errorf("the sensitive-port entry must name the port, got %q", msg.Headline)
	}
	if !strings.Contains(msg.Headline, "certificate-expired") {
		t.Errorf("every rule entry is named, got %q", msg.Headline)
	}
	if ContainsValence(msg.Headline) {
		t.Errorf("headline carries a valence word: %q", msg.Headline)
	}
	if census.Entries[0].Kind != KindFacet || census.Entries[1].Kind != KindRule {
		t.Errorf("rule entries must follow facet entries, got %+v", census.Entries)
	}
	if got := census.Rules(); len(got) != 2 || got[0].Key != "certificate-expired" {
		t.Errorf("Rules() = %+v, want the two rule entries by key", got)
	}
}

func TestFlagshipHeadlineWithoutRulesIsUnchanged(t *testing.T) {
	census := NewCensus(CensusEntry{Kind: KindFacet, Key: "certificate"}, CensusEntry{Kind: KindFacet, Key: "http-identity"})
	if got := flagshipHeadline("198.51.100.1:443/tcp", census); got != "198.51.100.1:443/tcp reached from the internet · 2 facets opened beneath it" {
		t.Errorf("a census with no rule entry must render as before, got %q", got)
	}
}

func TestCensusEntryDetailRoundTrips(t *testing.T) {
	c := NewCensus(CensusEntry{Kind: KindRule, Key: "sensitive-port-reached-from-internet", Detail: "3306/tcp"})
	b, err := c.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	back, err := ParseCensus(b)
	if err != nil {
		t.Fatal(err)
	}
	if back.Entries[0].Detail != "3306/tcp" {
		t.Errorf("detail lost in the round trip: %+v", back.Entries)
	}
	plain, _ := NewCensus(CensusEntry{Kind: KindFacet, Key: "certificate"}).Marshal()
	if strings.Contains(string(plain), "detail") {
		t.Errorf("a facet entry must marshal as before, got %s", plain)
	}
}
