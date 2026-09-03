package message

import "time"

type VantageClass string

const (
	ClassInternet VantageClass = "internet"
	ClassInternal VantageClass = "internal"
)

// There is no value for "we did not look": that absence is a Gap, never a move (CONTEXT.md Reach).

const (
	NotReached = "not-reached"
	Reached    = "reached"
)

type ReachMove struct {
	ServiceKey string
	Class      VantageClass
	From       string
	To         string
	Opened     bool
}

func Flagship(m ReachMove, census Census, instant time.Time) *Message {
	// An internal-leg move is recorded and never alerted, in either direction (ADR-0029).
	if m.Class != ClassInternet {
		return nil
	}
	// An opening emits no Transition, so this news rides the membership census (CONTEXT.md Reach).
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
