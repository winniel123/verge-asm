package message

import "time"

// VantageClass is which side of the operator's boundary a Reach leg was measured
// from. The flagship reads the internet leg alone; the internal leg is recorded
// and never alerted, either direction.
type VantageClass string

const (
	ClassInternet VantageClass = "internet"
	ClassInternal VantageClass = "internal"
)

// The two Reach values — total over a connection-oriented exchange (CONTEXT.md
// `Reach`). There is no value for *we did not look*; that absence is a Gap and
// never a move this predicate reads.
const (
	NotReached = "not-reached"
	Reached    = "reached"
)

// ReachMove is one Reach leg's transition on one Service — the input the
// flagship predicate reads. It carries the leg's class, the from/to values of
// the transition, and whether the leg opened at `reached` rather than moving.
type ReachMove struct {
	ServiceKey string
	Class      VantageClass
	From       string
	To         string
	// Opened is true where this is the first look the leg ever had — the timeline
	// opened at `reached` rather than moving into it. An opening emits no
	// Transition, so the flagship predicate does not match it: that news is
	// carried by the census on the entering subject's membership message and by
	// nothing else (CONTEXT.md `Reach`).
	Opened bool
}

// Flagship returns the flagship Message for an internet `Reach` leg going
// `not-reached` → `reached` — the move the product exists to catch — or nil
// where the predicate does not match. It encodes ADR-0029's alert-on-a-leg rule
// structurally:
//
//   - It reads the internet leg alone. An internal-leg move — a port opening or
//     closing, the commonest intentional change on that leg — is recorded and
//     never alerted in either direction, so a ClassInternal move returns nil.
//   - It fires whether or not an internal leg even exists: it reads one leg and
//     composes that leaf alone, so a one-legged install where no `Exposure`
//     exists still fires it. (This function never consults the internal leg, so
//     there is nothing to gate on.)
//   - Only `not-reached` → `reached` matches. The internet leg's `reached` →
//     `not-reached` is silent (the shrinking direction), and a leg that opened
//     at `reached` emits no Transition and returns nil here.
//   - The message carries the census of every facet opening beneath the
//     newly-reached Service — certificate, http-identity, tls-acceptance and
//     every rule over them open there, and an opening reaches nobody on its own.
//
// The firing is CauseDrift / ClassDrift: the estate's own object moved, and the
// row links to that Service's own page.
func Flagship(m ReachMove, census Census, instant time.Time) *Message {
	if m.Class != ClassInternet {
		return nil
	}
	if m.Opened {
		return nil
	}
	if m.From != NotReached || m.To != Reached {
		return nil
	}
	c := census
	return &Message{
		Cause:       CauseDrift,
		Class:       ClassDrift,
		SubjectKind: "service",
		FiredAt:     m.ServiceKey,
		Instant:     instant,
		Census:      &c,
		Headline:    flagshipHeadline(m.ServiceKey, census),
	}
}
