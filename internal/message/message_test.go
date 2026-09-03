package message

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

func TestClassForCause(t *testing.T) {
	cases := map[Cause]Class{
		CauseDrift:         ClassDrift,
		CauseAperture:      ClassCoverage,
		CauseDeclaredInput: ClassCoverage,
		CauseThreshold:     ClassClock,
	}
	for cause, want := range cases {
		if got := ClassForCause(cause); got != want {
			t.Errorf("ClassForCause(%q) = %q, want %q", cause, got, want)
		}
		if !cause.Valid() {
			t.Errorf("%q should be a valid cause", cause)
		}
	}
}

func TestLinkKindPerMover(t *testing.T) {
	cases := map[Cause]LinkKind{
		CauseDrift:         LinkObject,
		CauseThreshold:     LinkObject,
		CauseDeclaredInput: LinkSource,
		CauseAperture:      LinkSeed,
	}
	for cause, want := range cases {
		if got := LinkKindForCause(cause); got != want {
			t.Errorf("LinkKindForCause(%q) = %q, want %q", cause, got, want)
		}
	}
}

func TestComputedOnceAtCause(t *testing.T) {
	census := NewCensus(CensusEntry{Kind: "facet", Key: "certificate"})
	msg := Flagship(ReachMove{
		ServiceKey: "198.51.100.1:443/tcp", Class: ClassInternet,
		From: NotReached, To: Reached,
	}, census, t0)
	if msg == nil {
		t.Fatal("flagship must fire")
	}
	snapHeadline, snapLen, snapInstant := msg.Headline, msg.CensusLen(), msg.Instant

	_ = Flagship(ReachMove{
		ServiceKey: "198.51.100.1:443/tcp", Class: ClassInternet,
		From: NotReached, To: Reached,
	}, NewCensus(
		CensusEntry{Kind: "facet", Key: "certificate"},
		CensusEntry{Kind: "facet", Key: "http-identity"},
	), t0.Add(24*time.Hour))

	if msg.Headline != snapHeadline || msg.CensusLen() != snapLen || !msg.Instant.Equal(snapInstant) {
		t.Error("a computed message must be frozen — nothing recomputes it across a later firing")
	}
}

func TestNoValenceInRenderedCopy(t *testing.T) {
	census := NewCensus(
		CensusEntry{Kind: "facet", Key: "certificate"},
		CensusEntry{Kind: "facet", Key: "http-identity"},
	)
	copies := []string{
		flagshipHeadline("198.51.100.1:443/tcp", census),
		membershipHeadline(EntryAppeared, "example.com", census),
		membershipHeadline(EntryRevealed, "203.0.113.0/24", census),
		membershipHeadline(EntryReturned, "example.com", census),
		narrowingHeadline("198.51.100.0/24", "198.51.100.128/25", 128, 17920),
		narrowingLoss("198.51.100.128/25"),
	}
	for _, c := range copies {
		if ContainsValence(c) {
			t.Errorf("rendered copy carries a valence word: %q", c)
		}
	}
}

func TestValenceGuardWordBoundaries(t *testing.T) {
	if ContainsValence("we looked and the port is reached") {
		t.Error("the guard fired on a substring — it must match whole words only")
	}
	if !ContainsValence("this is resolved") {
		t.Error("the guard must catch a real valence word")
	}
}
