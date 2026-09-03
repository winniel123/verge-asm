package drift

import "time"

func CloseWithdrawal(open []Span, at time.Time, reason ClosureReason) []Span {
	// The withdrawn period is on no timeline, so returned is derivable across it (ADR-0082).
	out := make([]Span, len(open))
	for i, s := range open {
		// A closure is written once and never rewritten.
		if !s.Open() {
			out[i] = s
			continue
		}
		// One cause recorded on n objects is the shape a Gap already takes, not a seam (ADR-0087).
		s.ClosedAt = at
		// The ground is the caller's: measured absence, a broken Seed chain, or a narrowed aperture.
		s.Reason = reason
		out[i] = s
	}
	return out
}
