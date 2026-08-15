// Package message is the v1 `Message` model: one firing of one cause, computed
// once at the cause and never recomputed — recomputing one would reach back
// across a `Break` (CONTEXT.md `Message`, v1 spec §5.3, ADR-0064). It is
// Operational: a message records that the operator was told, never what is true
// of the estate, so the comparison path may never read one and nothing is ever
// concluded by comparing two. The fact itself lives in the timelines; if the two
// ever disagree the timeline wins and the message is still a true record of what
// we said.
//
// The package is a set of pure constructors — one per cause — plus the rendering
// grammar. Each constructor reads the fold at the cause and returns a frozen
// Message value: there is deliberately NO method that recomputes an existing
// message from live state, which is how "computed once at the cause" is enforced
// structurally rather than by discipline. The web layer stores the returned
// value verbatim and reads it back verbatim.
//
// Four causes, no valence word:
//
//   - the estate's own object moved (drift);
//   - us — our own aperture widened, or a rule of ours (aperture);
//   - the operator's own declared input (declared-input);
//   - nothing at all — only a clock or threshold was crossed (threshold).
//
// The three routing classes are those four with two merged (ADR-0091): aperture
// and declared-input both say *we changed how we look* and merge into coverage;
// drift is its own class; threshold is the clock class.
package message

import "time"

// Cause is the closed union of the four movers a Message can fire from. It is
// carried on every message and read by the operator; the router never reads it
// (routing keys on the class, ADR-0091).
type Cause string

const (
	// CauseDrift: the estate's own object moved. The row links to that object's
	// own page.
	CauseDrift Cause = "drift"
	// CauseAperture: us — our own aperture widened (or a rule of ours). The row
	// links to the Seed whose scope moved, never to Coverage's standing aperture
	// statement.
	CauseAperture Cause = "aperture"
	// CauseDeclaredInput: the operator's own declared input moved. The row links
	// to the Source the rule reads.
	CauseDeclaredInput Cause = "declared-input"
	// CauseThreshold: nothing at all — only a clock or threshold was crossed. The
	// row links to the object whose span the rule read.
	CauseThreshold Cause = "threshold"
)

// Valid reports whether a cause is one of the closed four.
func (c Cause) Valid() bool {
	switch c {
	case CauseDrift, CauseAperture, CauseDeclaredInput, CauseThreshold:
		return true
	default:
		return false
	}
}

// Class is the routing unit — one of three, and the only thing a Channel routes
// on (ADR-0091). It is a property of the firing, not of the rule.
type Class string

const (
	// ClassDrift: the estate's own object moved.
	ClassDrift Class = "drift"
	// ClassCoverage: *we changed how we look* — the two causes aperture and
	// declared-input merged.
	ClassCoverage Class = "coverage"
	// ClassClock: only a clock or threshold was crossed and no measurement moved.
	ClassClock Class = "clock"
)

// ClassForCause maps a firing's cause to its routing class — the merge that
// turns four causes into three classes (ADR-0091). `aperture` (us) and
// `declared-input` (the operator) both merge into coverage; `drift` stands
// alone; `threshold` is the clock class. Class is a property of the firing, so a
// clock-reading rule that finds its span moved fires CauseDrift and lands in the
// drift class through this same map — the class is never read off the rule.
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

// LinkKind is where a message row points, decided by the mover (v1 spec §5.3).
// It is derived from the cause and never stored, so a row can never drift out of
// step with its cause.
type LinkKind string

const (
	// LinkObject: an estate object's own page. Both a drift firing (the object
	// that moved) and a threshold firing (the object whose span the rule read)
	// link here — to a subject page.
	LinkObject LinkKind = "object"
	// LinkSource: the Source the rule reads — the declared-input mover.
	LinkSource LinkKind = "source"
	// LinkSeed: the Seed whose scope moved — the aperture-widening mover. NEVER
	// Coverage's standing aperture statement, which is constant and would lose
	// which act the message was about.
	LinkSeed LinkKind = "seed"
)

// LinkKindForCause resolves the row's link target kind per §5.3's per-mover
// rule. drift and threshold both link to an object page (the object that moved,
// or the object whose span the rule read); declared-input links to the Source;
// aperture links to the Seed whose scope moved.
func LinkKindForCause(c Cause) LinkKind {
	switch c {
	case CauseDrift, CauseThreshold:
		return LinkObject
	case CauseDeclaredInput:
		return LinkSource
	case CauseAperture:
		return LinkSeed
	default:
		return LinkObject
	}
}

// Message is one firing of one cause. It carries its class, the key of the
// subject or scope it fired at, the instant of the cause, and its census where
// it has one; it holds read-state and no other operator state (CONTEXT.md
// `Message`). It is a frozen value: once a constructor returns one, nothing in
// this package recomputes it.
type Message struct {
	// ID is the store's identity, zero for a freshly-computed message not yet
	// written.
	ID int64

	Cause Cause
	Class Class

	// SubjectKind names what FiredAt keys, so the web layer can build the right
	// drill-down URL: "name" / "address" / "service" / "endpoint" for an object
	// link, "source" for a Source link, "seed" for a Seed link.
	SubjectKind string
	// FiredAt is the key of the subject or scope the message fired at — the unit
	// of alerting is the message and never the affected subject, so this is one
	// key, not a set.
	FiredAt string

	// Instant is the instant of the cause, read from the fold. It is frozen at
	// construction and never re-derived — re-deriving it would reach back across
	// a Break.
	Instant time.Time

	// Census is the payload where the firing has one — the rows that opened
	// beneath the fired-at subject (a flagship or a membership root), or the
	// count a narrowing withdrew. Nil where the firing carries none.
	Census *Census

	// Headline is the rendered sentence, computed at the cause from the fold. It
	// carries no valence word and no severity (ADR-0064).
	Headline string

	// Read is the operator's read-state — the one piece of operator state a
	// message holds. It is not part of the fact and never reaches the comparison
	// path.
	Read bool
}

// LinkKind is the row's link target kind, derived from the cause on read.
func (m Message) LinkKind() LinkKind { return LinkKindForCause(m.Cause) }

// CensusLen is the census size, or zero where the message carries none — the
// count a row renders beside a flagship or membership headline.
func (m Message) CensusLen() int {
	if m.Census == nil {
		return 0
	}
	return m.Census.Len()
}
