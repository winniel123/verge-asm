package message

import (
	"strings"
	"testing"
	"time"
)

var gapT0 = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

func TestGapClosedFiresAtTheScopeWithThePairOnlyWhereTheValueDiffers(t *testing.T) {
	closures := []GapClosure{
		{Kind: "service", Key: "198.51.100.1:443/tcp", Facet: "reachability", Before: "not-reached", After: "reached"},
		{Kind: "service", Key: "198.51.100.2:443/tcp", Facet: "reachability", Before: "reached", After: "reached"},
		{Kind: "name", Key: "www.example.com", Facet: "resolution", Before: "Resolved 198.51.100.1", After: "Resolved 198.51.100.1"},
	}
	m := GapClosed("example.com", "seed", closures, nil, gapT0)
	if m == nil {
		t.Fatal("a Gap closing notifies")
	}
	if m.Cause != CauseAperture || m.Class != ClassCoverage {
		t.Errorf("a Gap closing is coverage class, got cause=%q class=%q", m.Cause, m.Class)
	}
	if m.SubjectKind != "seed" || m.FiredAt != "example.com" {
		t.Errorf("fires at the covering scope, got %q/%q", m.SubjectKind, m.FiredAt)
	}
	if m.Census.Len() != 3 {
		t.Fatalf("one entry per subject whose sight was restored, got %+v", m.Census)
	}
	var detailed int
	for _, e := range m.Census.Entries {
		if e.Detail != "" {
			detailed++
			if e.Key != "198.51.100.1:443/tcp" || e.Detail != "reachability not-reached → reached" {
				t.Errorf("the pair rides the differing entry only, got %+v", e)
			}
		}
	}
	if detailed != 1 {
		t.Errorf("exactly one entry carries a pair, got %d", detailed)
	}
	want := "example.com · sight restored on 3 timelines across 3 subjects · 1 differs from the last value seen: 198.51.100.1:443/tcp (reachability not-reached → reached)"
	if m.Headline != want {
		t.Errorf("headline\n got %q\nwant %q", m.Headline, want)
	}
}

func TestGapClosedNothingMovedIsOneLine(t *testing.T) {
	closures := []GapClosure{
		{Kind: "name", Key: "a.example.com", Facet: "resolution", Before: "Resolved 198.51.100.1", After: "Resolved 198.51.100.1"},
		{Kind: "name", Key: "b.example.com", Facet: "resolution", Before: "NoData", After: "NoData"},
	}
	m := GapClosed("example.com", "seed", closures, nil, gapT0)
	want := "example.com · sight restored on 2 timelines across 2 subjects · none differs from the last value seen"
	if m.Headline != want {
		t.Errorf("headline\n got %q\nwant %q", m.Headline, want)
	}
	for _, e := range m.Census.Entries {
		if e.Detail != "" {
			t.Errorf("an unchanged value carries no pair, got %+v", e)
		}
	}
}

func TestGapClosedStatesTheUndatableValueAndNoPair(t *testing.T) {
	closures := []GapClosure{
		{Kind: "service", Key: "198.51.100.1:443/tcp", Facet: "reachability", Before: "", After: "reached"},
		{Kind: "service", Key: "198.51.100.1:443/tcp", Facet: "reachability", Before: "", After: "reached"},
	}
	m := GapClosed("198.51.100.0/24", "seed", closures, nil, gapT0)
	if m.Census.Len() != 1 {
		t.Fatalf("one subject across two vantages is one entry, got %+v", m.Census)
	}
	if !strings.Contains(m.Headline, "none differs") || !strings.HasSuffix(m.Headline, " · 1 has no earlier value") {
		t.Errorf("a timeline that opened as a Gap states no pair, got %q", m.Headline)
	}
	if strings.Contains(m.Headline, "→") {
		t.Errorf("no pair is stated without an earlier value, got %q", m.Headline)
	}
}

func TestGapClosedWithNothingRestoredIsNil(t *testing.T) {
	if m := GapClosed("example.com", "seed", nil, nil, gapT0); m != nil {
		t.Errorf("no closing, no message, got %+v", m)
	}
}

func TestGapClosedStatesNoPairAcrossADerivationMoveAndCarriesTheRules(t *testing.T) {
	closures := []GapClosure{
		{Kind: "service", Key: "198.51.100.1:3306/tcp", Facet: "reachability", Before: "", After: "reached", Broke: true},
	}
	rules := []CensusEntry{
		{Kind: KindRule, Key: "sensitive-port-reached-from-internet", Detail: "3306/tcp"},
		{Kind: KindRule, Key: "sensitive-port-reached-from-internet", Detail: "3306/tcp"},
	}
	m := GapClosed("198.51.100.0/24", "seed", closures, rules, gapT0)
	want := "198.51.100.0/24 · sight restored on 1 timeline across 1 subject · none differs from the last value seen" +
		" · 1 is not compared across a derivation move · 1 rule opened at fired: sensitive-port-reached-from-internet (3306/tcp)"
	if m.Headline != want {
		t.Errorf("headline\n got %q\nwant %q", m.Headline, want)
	}
	if len(m.Census.Rules()) != 1 || m.Census.Len() != 2 {
		t.Errorf("one rule entry beside the subject entry, got %+v", m.Census)
	}
	if strings.Contains(m.Headline, "no earlier value") {
		t.Errorf("a break is not an absent value, got %q", m.Headline)
	}
}
