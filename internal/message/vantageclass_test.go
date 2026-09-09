package message

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

var widenT0 = time.Date(2026, 9, 9, 9, 0, 0, 0, time.UTC)

func TestVantageClassWidenedFiresAtTheClassWithTheStartedCensus(t *testing.T) {
	m := VantageClassWidened("internet", []CensusEntry{
		{Kind: KindService, Key: "10.0.0.2:443/tcp", Detail: "edge-only"},
		{Kind: KindService, Key: "10.0.0.1:443/tcp", Detail: "exposed"},
	}, widenT0)
	if m == nil {
		t.Fatal("a first vantage of a class fires one coverage-class message")
	}
	if m.Cause != CauseAperture || m.Class != ClassCoverage {
		t.Errorf("a widening is us changing how we look, got cause=%q class=%q", m.Cause, m.Class)
	}
	if m.SubjectKind != KindVantageClass || m.FiredAt != "internet" {
		t.Errorf("fires at the class, got %q/%q", m.SubjectKind, m.FiredAt)
	}
	if m.CensusLen() != 2 || m.Census.Entries[0].Key != "10.0.0.1:443/tcp" {
		t.Errorf("the census names every Exposure the widening started, got %+v", m.Census)
	}
	want := "an internet vantage is now configured · 2 Exposure timelines opened · 1 edge-only, 1 exposed"
	if m.Headline != want {
		t.Errorf("headline = %q, want %q", m.Headline, want)
	}
}

func TestVantageClassWidenedStatesAnEmptyCensusRatherThanStayingSilent(t *testing.T) {
	m := VantageClassWidened("internal", nil, widenT0)
	if m == nil {
		t.Fatal("the widening is the news, so an empty census still fires")
	}
	if m.CensusLen() != 0 {
		t.Errorf("census = %+v, want empty", m.Census)
	}
	if !strings.HasSuffix(m.Headline, "no Exposure timeline opened") {
		t.Errorf("headline = %q, want it to state the zero", m.Headline)
	}
}

func TestVantageClassWidenedRefusesAnUnnamedClass(t *testing.T) {
	if m := VantageClassWidened("", nil, widenT0); m != nil {
		t.Errorf("a class with no name keys nothing, got %+v", m)
	}
}

func TestVantageClassWidenedGroupsTheCountAndTalliesByValue(t *testing.T) {
	entries := make([]CensusEntry, 0, 1412)
	for i := range 1412 {
		v := "unreachable"
		if i < 37 {
			v = "exposed"
		}
		entries = append(entries, CensusEntry{Kind: KindService, Key: fmt.Sprintf("10.0.0.1:%d/tcp", i), Detail: v})
	}
	m := VantageClassWidened("internet", entries, widenT0)
	if !strings.Contains(m.Headline, "1,412 Exposure timelines opened") {
		t.Errorf("headline = %q, want the grouped count", m.Headline)
	}
	if !strings.Contains(m.Headline, "37 exposed, 1,375 unreachable") {
		t.Errorf("headline = %q, want the tally by value", m.Headline)
	}
}
