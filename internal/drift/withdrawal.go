package drift

import "time"

// CloseWithdrawal closes each of a withdrawn subject's open spans at `at`,
// recording the ground the closure rests on. It is the drift-side half of a
// departure: whether the subject left is decided once, in the membership
// composition (internal/estate.WithdrawnCrossClass), and this applies the
// consequence to the subject's timelines.
//
// A withdrawal takes every timeline the subject held; the withdrawn period is
// then on no timeline at all — neither a value nor a Gap — which is precisely
// what leaves the span before the withdrawal and the span after a later return
// adjacent, so `returned` is derivable as their Transition (ADR-0082). The reason
// is written onto every closed timeline: that is one cause recorded on n objects,
// the same shape a Gap takes when a Vantage goes unavailable, and not a seam
// (ADR-0087).
//
// The reason must be one of the three grounds. A caller passes ReasonMeasuredAbsent
// where an observation about this subject measured its absence, ReasonUncited
// where the subject's chain back to a Seed broke, and ReasonDescoped where our
// aperture stopped covering it. An already-closed span is returned unchanged — a
// closure is written once and never rewritten.
func CloseWithdrawal(open []Span, at time.Time, reason ClosureReason) []Span {
	out := make([]Span, len(open))
	for i, s := range open {
		if !s.Open() {
			out[i] = s
			continue
		}
		s.ClosedAt = at
		s.Reason = reason
		out[i] = s
	}
	return out
}
