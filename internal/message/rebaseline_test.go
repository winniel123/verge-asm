package message

import (
	"strings"
	"testing"
)

func TestRebaselineIsOneCoverageMessageAtTheDerivation(t *testing.T) {
	msg := Rebaseline("resolution", []string{"resolution-walk", "wildcard-discrimination"}, true, t0)
	if msg == nil {
		t.Fatal("a moved vector on an alerting derivation must fire")
	}
	if msg.Cause != CauseAperture || msg.Class != ClassCoverage {
		t.Errorf("a re-baseline is our own release: cause=%q class=%q, want aperture/coverage", msg.Cause, msg.Class)
	}
	if msg.SubjectKind != "derivation" || msg.FiredAt != "resolution" {
		t.Errorf("a re-baseline fires at the derivation, got %q/%q", msg.SubjectKind, msg.FiredAt)
	}
	if !strings.Contains(msg.Headline, "resolution-walk") || !strings.Contains(msg.Headline, "wildcard-discrimination") {
		t.Errorf("the headline names every moved leaf, got %q", msg.Headline)
	}
	if !strings.Contains(msg.Headline, "appeared") {
		t.Errorf("a membership leaf move states that a return reads appeared, got %q", msg.Headline)
	}
	if msg.CensusLen() != 2 || msg.Census.Entries[0].Kind != "leaf" {
		t.Errorf("the census enumerates the moved leaves, got %+v", msg.Census)
	}
	if ContainsValence(msg.Headline) {
		t.Errorf("the headline carries a valence word: %q", msg.Headline)
	}
	if msg.Instant != t0 {
		t.Errorf("instant = %v, want %v", msg.Instant, t0)
	}
}

func TestRebaselineWithoutMembershipLeafSaysNothingAboutReturns(t *testing.T) {
	msg := Rebaseline("reachability", []string{"connect-outcome"}, false, t0)
	if msg == nil {
		t.Fatal("a moved reachability vector must fire")
	}
	if strings.Contains(msg.Headline, "appeared") {
		t.Errorf("a non-membership leaf move says nothing about returns, got %q", msg.Headline)
	}
	if msg.CensusLen() != 1 || msg.Census.Entries[0].Key != "connect-outcome" {
		t.Errorf("the census names the one moved leaf, got %+v", msg.Census)
	}
}

func TestRebaselineEmptyDifferenceSetIsSuppressed(t *testing.T) {
	if Rebaseline("resolution", nil, true, t0) != nil {
		t.Error("an empty difference set suppresses the message entirely (ADR-0008)")
	}
}
