package drift

// A view clamps its horizon to the most recent Break rather than blanking; nothing crosses one.

type Break struct {
	Before      Span
	After       Span
	MovedLeaves []string
}

func Breaks(spans []Span) []Break {
	var out []Break
	// Derived on read and never stored, which is what lets a Break name the moved leaf (ADR-0008).
	for i := 1; i < len(spans); i++ {
		before, after := spans[i-1], spans[i]
		if before.Vector.Equal(after.Vector) {
			continue
		}
		// A flat, never-nested vector lets a Break name a leaf and not a path through the graph.
		out = append(out, Break{
			Before:      before,
			After:       after,
			MovedLeaves: MovedLeaves(before.Vector, after.Vector),
		})
	}
	return out
}

// Across a Break two values are not the same kind of thing, so nothing may compare them (ADR-0008).

func Comparable(a, b Span) bool { return a.Vector.Equal(b.Vector) }

type Kind string

const (
	KindNone     Kind = ""
	KindAppeared Kind = "appeared"
	KindReturned Kind = "returned"
	KindRevealed Kind = "revealed"
)

func MembershipReturn(priorClosure *Span, witnessBroke bool) Kind {
	// The aperture guard lives in ReEntryKind; these two are the membership guards alone (#1039).
	if priorClosure == nil {
		return KindAppeared
	}
	// A narrowing is not a decommission, so a descoped closure never reads returned (ADR-0087).
	if priorClosure.Reason == ReasonDescoped {
		return KindAppeared
	}
	// A Break on any relied-upon witness voids returned for the whole subject (ADR-0080, ADR-0097).
	if witnessBroke {
		return KindAppeared
	}
	return KindReturned
}

func ReEntryKind(priorClosure *Span, witnessBroke, apertureWidened bool) Kind {
	// The marker reports that the Declared aperture covers the subject and no Exclusion cuts it out.
	// It is stamped only where a Seed declares the subject, so no subject-kind switch is needed.
	if apertureWidened && priorClosure != nil && priorClosure.Reason == ReasonDescoped {
		// Widening a Declared scope is a Declared act: revealed, never appeared (ADR-0041, ADR-0047).
		return KindRevealed
	}
	return MembershipReturn(priorClosure, witnessBroke)
}

func OpeningKind(apertureWidened bool) Kind {
	// Membership alone owns appeared and returned, so a facet-timeline opening reads neither.
	if apertureWidened {
		// A widened aperture means we started looking; the world did not move (ADR-0014).
		return KindRevealed
	}
	return KindNone
}
