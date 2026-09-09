package message

import (
	"strings"
	"testing"
	"time"
)

func TestExtensionGainedIsACoverageFiringAtTheScope(t *testing.T) {
	at := time.Date(2026, 9, 9, 3, 0, 0, 0, time.UTC)
	m := ExtensionGained("example.com", []ExtensionGain{
		{Address: "52.1.2.4", Names: []string{"www.example.com", "api.example.com"}},
		{Address: "52.1.2.3", Names: []string{"api.example.com"}},
	}, 17, at)
	if m == nil {
		t.Fatal("a gain fires")
	}
	if m.Cause != CauseAperture || m.Class != ClassCoverage {
		t.Errorf("cause=%q class=%q, want aperture/coverage", m.Cause, m.Class)
	}
	if m.SubjectKind != "seed" || m.FiredAt != "example.com" {
		t.Errorf("fired at %q/%q, want seed/example.com", m.SubjectKind, m.FiredAt)
	}
	if m.LinkKind() != LinkSeed {
		t.Errorf("link kind %q, want seed", m.LinkKind())
	}
	if m.Instant != at {
		t.Errorf("instant %v, want %v", m.Instant, at)
	}
	want := "example.com custody extension gained 2 addresses: " +
		"52.1.2.3 (api.example.com), 52.1.2.4 (api.example.com, www.example.com) · 17 addresses covered"
	if m.Headline != want {
		t.Errorf("headline\n got %q\nwant %q", m.Headline, want)
	}
	if ContainsValence(m.Headline) {
		t.Errorf("the headline grades nothing, got %q", m.Headline)
	}
	if m.CensusLen() != 2 || m.Census.Entries[0].Kind != KindAddress || m.Census.Entries[0].Key != "52.1.2.3" {
		t.Errorf("the census carries the gained addresses, got %+v", m.Census)
	}
	if strings.Contains(m.Headline, "may") || strings.Contains(m.Headline, "over-assert") {
		t.Errorf("the headline reports a boundary, not a suspicion: %q", m.Headline)
	}
}

func TestExtensionGainedWithNoGainIsNoMessage(t *testing.T) {
	if m := ExtensionGained("example.com", nil, 3, time.Now()); m != nil {
		t.Errorf("no gain fires nothing, got %+v", m)
	}
}
