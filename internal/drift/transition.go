package drift

// Break is the boundary between two consecutive spans whose Derivation vectors
// differ — derived on read from the two spans' vectors, never stored, which is
// what lets it name the leaf that moved (ADR-0008). Nothing is ever compared
// across a Break, so no Transition is emitted and nothing is alerted; a view
// clamps its horizon to the most recent Break rather than blanking.
type Break struct {
	Before Span
	After  Span
	// MovedLeaves names the leaves whose version differs across the boundary.
	// Flattened vectors mean a Break names a leaf rather than a path through the
	// composition graph — a sentence an operator can act on.
	MovedLeaves []string
}

// Breaks derives the Breaks over an ordered run of spans on one timeline (as
// Fold returns them). A Break sits between every adjacent pair whose vectors are
// not equal. Nothing is stored: this reads the vectors both spans already carry.
func Breaks(spans []Span) []Break {
	var out []Break
	for i := 1; i < len(spans); i++ {
		before, after := spans[i-1], spans[i]
		if before.Vector.Equal(after.Vector) {
			continue
		}
		out = append(out, Break{
			Before:      before,
			After:       after,
			MovedLeaves: MovedLeaves(before.Vector, after.Vector),
		})
	}
	return out
}

// Comparable reports whether two spans may be compared at all — the precondition
// every drift comparison rests on. It is exactly vector equality: across a Break
// the values on either side are not the same kind of thing, so a query cannot
// accidentally compare them (ADR-0008).
func Comparable(a, b Span) bool { return a.Vector.Equal(b.Vector) }

type Kind string

const (
	KindNone     Kind = ""
	KindAppeared Kind = "appeared"
	KindReturned Kind = "returned"
	// KindRevealed is a widened aperture — we started looking, the world did not
	// move. It belongs to any timeline, never to a subject's membership alone.
	KindRevealed Kind = "revealed"
)

// MembershipReturn decides `appeared` versus `returned` for a subject re-entering
// the estate. It is the membership-only half of the Transition grammar and the
// place two independent guards compose (ADR-0087, ADR-0097):
//
//   - priorClosure is the closed membership span the reopening sits behind, or
//     nil where the subject has no prior span — a first discovery reads
//     `appeared`.
//   - A `descoped` closure blocks `returned`: the intervening period was an
//     aperture narrowing, not a decommission, so an ordinary measured return
//     across it reads `appeared`, never `returned` (ADR-0087).
//   - witnessBroke is true where a Break sits on any witness the composed
//     presence read currently relies on — one per Vantage class, existential
//     within a class, agreed across classes (ADR-0080). A Break on any one voids
//     `returned` for the whole subject, which re-enters reading `appeared`
//     (ADR-0097). For the modal single-class, single-vantage install the witness
//     set has size one and this collapses to ADR-0082's sentence.
//
// The predicate is the conjunction of both conditions: either failing alone
// reads `appeared`, because each guards a distinct way the model could overclaim
// a continuity it does not have.
//
// It reads the two membership guards ALONE. A re-entry asks one further question
// this function cannot answer — whether the operator's aperture widened back over
// the subject — and ReEntryKind composes that third guard over this one.
func MembershipReturn(priorClosure *Span, witnessBroke bool) Kind {
	if priorClosure == nil {
		return KindAppeared
	}
	if priorClosure.Reason == ReasonDescoped {
		return KindAppeared
	}
	if witnessBroke {
		return KindAppeared
	}
	return KindReturned
}

// ReEntryKind names the Transition for a timeline re-opening behind a closed span.
// It is the one place the membership pair and `revealed` meet, and it composes the
// two functions on either side of it rather than restating their guards.
//
// A re-entry behind a `descoped` closure asks a question the membership grammar
// alone cannot answer: WHY are we looking again. MembershipReturn refuses `returned`
// across the narrowing and reads `appeared`, which is right where the world brought
// the subject back — a resolution citing the Address again. It is wrong where the
// operator widened the Declared scope back over the subject. ADR-0041 holds that
// widening or narrowing a Declared scope is a Declared act yielding `revealed`
// (ADR-0047), never `appeared`, for exactly that population.
//
// apertureWidened answers the question. It is the aperture-opening marker the fold
// stamps on the opening, which reports that the Declared aperture covers the subject
// NOW and that no Exclusion cuts it back out (queue.openedByAperture). The guard is
// the CONJUNCTION with the `descoped` ground, because neither input is sufficient:
//
//   - The marker alone is not. A `measured-absent` or `uncited` re-entry under a
//     covering aperture is a decommission undone and stays `returned`. The aperture
//     did not move; the world did.
//   - The `descoped` ground alone is not. An unmarked re-entry is the world citing a
//     subject the operator has not re-widened over, and stays `appeared`.
//
// The rule reads the marker and never the subject kind. The fold stamps the marker
// only where a Seed declares the subject, so it already carries ADR-0041's
// `Seed`-covered population without a kind switch, and a Name or Service the
// operator re-widened over reads the same word an Address does.
func ReEntryKind(priorClosure *Span, witnessBroke, apertureWidened bool) Kind {
	if apertureWidened && priorClosure != nil && priorClosure.Reason == ReasonDescoped {
		return KindRevealed
	}
	return MembershipReturn(priorClosure, witnessBroke)
}

// OpeningKind names the Transition for a timeline opening that is not a
// subject's membership entering. `revealed` where a widened aperture opened the
// timeline — an enabled source, a widened qtype set, an opened ownership gate —
// and unnamed otherwise (an opening caused by neither the world nor our aperture
// is recorded and unnamed; ADR-0014). It never returns `appeared` or `returned`,
// which are membership's alone: this is the guardrail that keeps a facet-timeline
// opening from being narrated as a subject returning.
func OpeningKind(apertureWidened bool) Kind {
	if apertureWidened {
		return KindRevealed
	}
	return KindNone
}
