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

// Kind is a Transition's name. Three named kinds live in two families:
// `appeared` and `returned` are membership-only (they describe a subject), and
// `revealed` belongs to any timeline (aperture is a property of looking). An
// ordinary value move between two spans is a Transition with no name — the
// adjacency is the fact and nothing is alerted per consequence.
type Kind string

const (
	// KindNone is an ordinary adjacency — a value moved, and the move is recorded
	// and unnamed.
	KindNone Kind = ""
	// KindAppeared is discovery: a subject entering the estate with no prior
	// membership span to return from. Membership-only.
	KindAppeared Kind = "appeared"
	// KindReturned is a decommission undone: a subject re-entering across a clean
	// history. Membership-only, and the more alertable of the pair.
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
