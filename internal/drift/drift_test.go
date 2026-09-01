package drift

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 8, 10, 9, 0, 0, 0, time.UTC)

func day(n int) time.Time { return t0.Add(time.Duration(n) * 24 * time.Hour) }

func vec(pairs ...string) Vector {
	var cs []Component
	for i := 0; i+1 < len(pairs); i += 2 {
		cs = append(cs, Component{Leaf: pairs[i], Version: pairs[i+1]})
	}
	return NewVector(cs...)
}

var resKey = TimelineKey{SubjectKind: "name", SubjectKey: "api.example.com", Facet: "resolution", Source: "resolver"}

// --- Fold: span open / close (AC1, AC6) ---

func TestFoldOpensOneSpanForOneValue(t *testing.T) {
	v := vec("resolution-walk", "v1")
	spans := Fold(resKey, []Reading{
		{Value: `{"outcome":"Resolved"}`, Vector: v, ObservedAt: day(0)},
		{Value: `{"outcome":"Resolved"}`, Vector: v, ObservedAt: day(1)},
		{Value: `{"outcome":"Resolved"}`, Vector: v, ObservedAt: day(2)},
	})
	if len(spans) != 1 {
		t.Fatalf("one held value = %d spans, want 1", len(spans))
	}
	if !spans[0].Open() {
		t.Error("the sole span is the current state and must be open")
	}
	if !spans[0].Vector.Equal(v) {
		t.Error("the span must carry the derivation vector it was produced under")
	}
}

func TestFoldChangedAnswerClosesOldAndOpensNew(t *testing.T) {
	// AC6: re-running the dns Scan with a changed answer produces a new Span and
	// the old one closes.
	v := vec("resolution-walk", "v1")
	spans := Fold(resKey, []Reading{
		{Value: `{"outcome":"Resolved","addresses":["203.0.113.1"]}`, Vector: v, ObservedAt: day(0)},
		{Value: `{"outcome":"Resolved","addresses":["203.0.113.2"]}`, Vector: v, ObservedAt: day(1)},
	})
	if len(spans) != 2 {
		t.Fatalf("a changed answer = %d spans, want 2", len(spans))
	}
	if spans[0].Open() {
		t.Error("the first span must close when the answer moves")
	}
	if !spans[0].ClosedAt.Equal(day(1)) {
		t.Errorf("first span closes at the new reading's instant, got %v", spans[0].ClosedAt)
	}
	if !spans[1].Open() {
		t.Error("the new span is current and open")
	}
	if spans[0].Reason != "" {
		t.Error("an ordinary value move records no closure reason — the next span is the fact")
	}
}

func TestFoldVersionChangeClosesAndOpensEvenWhenValueEqual(t *testing.T) {
	// A version change closes every open span of that derivation and opens a new
	// one, with a Break between — even where the value is byte-identical (ADR-0007).
	before := vec("resolution-walk", "v1")
	after := vec("resolution-walk", "v2")
	spans := Fold(resKey, []Reading{
		{Value: `{"outcome":"Resolved"}`, Vector: before, ObservedAt: day(0)},
		{Value: `{"outcome":"Resolved"}`, Vector: after, ObservedAt: day(1)},
	})
	if len(spans) != 2 {
		t.Fatalf("a version change = %d spans, want 2", len(spans))
	}
	if spans[0].Open() {
		t.Error("the pre-upgrade span must close")
	}
}

func TestFoldStepExtendsOpenSpan(t *testing.T) {
	v := vec("resolution-walk", "v1")
	open := &Span{Key: resKey, Value: `{"outcome":"NoData"}`, Vector: v, OpenedAt: day(0)}
	_, _, changed := FoldStep(open, resKey, Reading{Value: `{"outcome":"NoData"}`, Vector: v, ObservedAt: day(1)})
	if changed {
		t.Error("an identical reading extends the open span — nothing is written")
	}
}

func TestFoldStepOpensFirstSpan(t *testing.T) {
	closeAt, opened, changed := FoldStep(nil, resKey, Reading{Value: `x`, Vector: vec("l", "v1"), ObservedAt: day(0)})
	if !changed || !opened.Open() || !closeAt.IsZero() {
		t.Errorf("a new timeline opens its first span with no close, got changed=%v closeAt=%v", changed, closeAt)
	}
}

// --- Break on read names the moved leaf (AC2) ---

func TestBreakNamesMovedLeaf(t *testing.T) {
	before := vec("resolution-walk", "v1", "wildcard-discrimination", "v1")
	after := vec("resolution-walk", "v1", "wildcard-discrimination", "v2")
	spans := []Span{
		{Vector: before, OpenedAt: day(0), ClosedAt: day(1)},
		{Vector: after, OpenedAt: day(1)},
	}
	breaks := Breaks(spans)
	if len(breaks) != 1 {
		t.Fatalf("differing vectors = %d breaks, want 1", len(breaks))
	}
	if len(breaks[0].MovedLeaves) != 1 || breaks[0].MovedLeaves[0] != "wildcard-discrimination" {
		t.Errorf("break must name the moved leaf, got %v", breaks[0].MovedLeaves)
	}
}

func TestNoBreakWhenVectorsEqual(t *testing.T) {
	v := vec("resolution-walk", "v1")
	spans := []Span{
		{Vector: v, Value: "a", OpenedAt: day(0), ClosedAt: day(1)},
		{Vector: v, Value: "b", OpenedAt: day(1)},
	}
	if len(Breaks(spans)) != 0 {
		t.Error("equal vectors are comparable — no Break, only an ordinary value move")
	}
	if !Comparable(spans[0], spans[1]) {
		t.Error("equal-vector spans must be Comparable")
	}
}

func TestBreakNamesLeafEnteringVector(t *testing.T) {
	// The membership vector is discovered, not declared: a leaf entering it is a
	// real move a Break names (ADR-0086).
	before := vec("resolution-walk", "v1")
	after := vec("resolution-walk", "v1", "wildcard-discrimination", "v1")
	got := MovedLeaves(before, after)
	if len(got) != 1 || got[0] != "wildcard-discrimination" {
		t.Errorf("a leaf entering the vector has moved, got %v", got)
	}
}

// --- Transition on read (AC3, AC4) ---

func TestAppearedWhenNoPriorSpan(t *testing.T) {
	if MembershipReturn(nil, false) != KindAppeared {
		t.Error("a first discovery reads appeared")
	}
}

func TestReturnedAcrossCleanHistory(t *testing.T) {
	prior := &Span{Reason: ReasonMeasuredAbsent, ClosedAt: day(1)}
	if MembershipReturn(prior, false) != KindReturned {
		t.Error("a measured-absent departure with a clean witness reads returned")
	}
}

func TestDescopedClosureSuppressesReturned(t *testing.T) {
	// AC4: only descoped suppresses a later returned.
	prior := &Span{Reason: ReasonDescoped, ClosedAt: day(1)}
	if MembershipReturn(prior, false) != KindAppeared {
		t.Error("a descoped closure blocks returned — a narrowing is not a decommission")
	}
	// uncited and measured-absent do NOT suppress it.
	for _, r := range []ClosureReason{ReasonMeasuredAbsent, ReasonUncited} {
		if MembershipReturn(&Span{Reason: r, ClosedAt: day(1)}, false) != KindReturned {
			t.Errorf("%s must not suppress returned", r)
		}
	}
}

func TestBreakOnWitnessVoidsReturned(t *testing.T) {
	// AC3/ADR-0097: a Break on any relied-upon witness voids returned; the subject
	// re-enters reading appeared.
	prior := &Span{Reason: ReasonMeasuredAbsent, ClosedAt: day(1)}
	if MembershipReturn(prior, true) != KindAppeared {
		t.Error("a Break on a witness re-enters as appeared, not returned")
	}
}

func TestRevealedIsAnyTimelineNotMembership(t *testing.T) {
	if OpeningKind(true) != KindRevealed {
		t.Error("a widened aperture opens a timeline reading revealed")
	}
	if OpeningKind(false) != KindNone {
		t.Error("an ordinary opening is unnamed")
	}
	// OpeningKind never returns a membership-only kind.
	if k := OpeningKind(true); k == KindAppeared || k == KindReturned {
		t.Error("revealed belongs to any timeline; appeared/returned are membership-only")
	}
}

// --- Closure grounds are a closed union of three ---

func TestClosureReasonsAreClosedAtThree(t *testing.T) {
	for _, r := range []ClosureReason{ReasonMeasuredAbsent, ReasonUncited, ReasonDescoped} {
		if !r.Valid() {
			t.Errorf("%s must be a valid ground", r)
		}
	}
	for _, bad := range []ClosureReason{"cascaded", "measured", "gone", ""} {
		if ClosureReason(bad).Valid() {
			t.Errorf("%q must not be a valid ground", bad)
		}
	}
}

func TestCloseWithdrawalClosesEveryTimelineWithReason(t *testing.T) {
	open := []Span{
		{Key: TimelineKey{Facet: "resolution"}, OpenedAt: day(0)},
		{Key: TimelineKey{Facet: "dns-record", Discriminator: "A"}, OpenedAt: day(0)},
	}
	closed := CloseWithdrawal(open, day(2), ReasonMeasuredAbsent)
	for _, s := range closed {
		if s.Open() {
			t.Error("a withdrawal closes every timeline the subject held")
		}
		if s.Reason != ReasonMeasuredAbsent {
			t.Errorf("closure records its ground on every timeline, got %q", s.Reason)
		}
	}
}

// ReEntryKind composes the aperture marker over the membership pair (#1039). A
// re-entry behind a `descoped` closure reads `revealed` where the operator widened a
// Declared scope back over the subject, and `appeared` where the world cited the
// subject again. Neither input alone decides the word.
func TestReEntryKindRevealedOnDeclaredWidening(t *testing.T) {
	descoped := &Span{Reason: ReasonDescoped, ClosedAt: day(1)}

	// ADR-0041: a Declared widening yields revealed, never appeared.
	if got := ReEntryKind(descoped, false, true); got != KindRevealed {
		t.Errorf("a marked re-entry across a descoped closure = %q, want revealed", got)
	}
	// Unmarked: the world cited the subject again and no scope widened.
	if got := ReEntryKind(descoped, false, false); got != KindAppeared {
		t.Errorf("an unmarked re-entry across a descoped closure = %q, want appeared", got)
	}
	// The marker never promotes a decommission undone. Those grounds keep returned.
	for _, r := range []ClosureReason{ReasonMeasuredAbsent, ReasonUncited} {
		if got := ReEntryKind(&Span{Reason: r, ClosedAt: day(1)}, false, true); got != KindReturned {
			t.Errorf("a marked re-entry across a %s closure = %q, want returned", r, got)
		}
	}
	// A Break on a witness still voids returned, and the marker does not restore it.
	if got := ReEntryKind(&Span{Reason: ReasonMeasuredAbsent, ClosedAt: day(1)}, true, true); got != KindAppeared {
		t.Errorf("a broken witness = %q, want appeared", got)
	}
	// No prior span is a first discovery. This function defers to MembershipReturn
	// for that rather than reading a marker as a widening.
	if got := ReEntryKind(nil, false, true); got != KindAppeared {
		t.Errorf("no prior closure = %q, want appeared", got)
	}
}
