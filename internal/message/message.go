// Package message is the operational record of what the operator was told: each
// firing is computed once at its cause and never recomputed, and the comparison
// path may never read one (CONTEXT.md, v1 spec §5.3, ADR-0064, ADR-0091).
package message

import "time"

type Cause string

const (
	CauseDrift         Cause = "drift"
	CauseAperture      Cause = "aperture"
	CauseDeclaredInput Cause = "declared-input"
	CauseThreshold     Cause = "threshold"
)

func (c Cause) Valid() bool {
	switch c {
	case CauseDrift, CauseAperture, CauseDeclaredInput, CauseThreshold:
		return true
	default:
		return false
	}
}

// A Channel routes on the class alone, never on the cause, which the operator reads (ADR-0091).

type Class string

const (
	ClassDrift    Class = "drift"
	ClassCoverage Class = "coverage"
	ClassClock    Class = "clock"
)

func ClassForCause(c Cause) Class {
	switch c {
	case CauseDrift:
		return ClassDrift
	case CauseAperture, CauseDeclaredInput:
		return ClassCoverage
	case CauseThreshold:
		return ClassClock
	default:
		return ""
	}
}

// Derived from the cause and never stored, so a row cannot drift from its cause (v1 spec §5.3).

type LinkKind string

const (
	LinkObject LinkKind = "object"
	LinkSource LinkKind = "source"
	LinkSeed   LinkKind = "seed"
)

func LinkKindForCause(c Cause) LinkKind {
	switch c {
	case CauseDrift, CauseThreshold:
		return LinkObject
	case CauseDeclaredInput:
		return LinkSource
	case CauseAperture:
		// Never Coverage's standing aperture statement, which is constant and loses which act fired.
		return LinkSeed
	default:
		return LinkObject
	}
}

// Read-state is the only operator state this may hold (CONTEXT.md Message).

type Message struct {
	ID int64

	Cause Cause
	Class Class

	SubjectKind string
	FiredAt     string // The unit of alerting is the message, so this is one key and never a set.

	Instant time.Time

	Census *Census

	Headline string

	Read bool
}

func (m Message) LinkKind() LinkKind { return LinkKindForCause(m.Cause) }

func (m Message) CensusLen() int {
	if m.Census == nil {
		return 0
	}
	return m.Census.Len()
}
