package message

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

// The four causes each map to exactly one class, and the three classes are the
// four causes with aperture and declared-input merged (ADR-0091).
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

// Each message row links per its mover (v1 spec §5.3): drift and threshold to an
// object page, declared-input to the Source, aperture-widening to the Seed.
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

// A Message is computed once at the cause and never recomputed: the package
// exposes construction alone, and a constructed message is a frozen value whose
// headline, census and instant do not move when the world later does. This pins
// the invariant that no code path recomputes an existing message across a Break.
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

	// The world moves on: a later census with more facets, a later instant. The
	// already-computed message must be untouched — there is no recompute path.
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

// No valence word appears in any rendered message copy — flagship, membership,
// and narrowing headlines all pass the guard.
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

// The valence guard matches whole words only — it must not fire on a substring
// like `ok` inside `looked`.
func TestValenceGuardWordBoundaries(t *testing.T) {
	if ContainsValence("we looked and the port is reached") {
		t.Error("the guard fired on a substring — it must match whole words only")
	}
	if !ContainsValence("this is resolved") {
		t.Error("the guard must catch a real valence word")
	}
}
