// Package drift is the v1 drift engine's facet-agnostic core: the Span timeline
// and everything derived on read from it — the Break between two spans under
// differing Derivation vectors, the Transition between two consecutive spans,
// and the Closure ground a withdrawal records. It is scoped to nothing: the same
// machinery serves `resolution` and `dns-record` (the two facets that exist so
// far) and every facet a later ticket adds, because a Span is one period a value
// was held and that shape does not vary by facet (v1 spec §5.1, ADR-0007).
//
// Three rules the package encodes structurally rather than by discipline:
//
//   - Two spans compare only where their Derivation vectors are equal. A
//     comparison across differing vectors is not merely inadvisable, it is a
//     Break — derived on read, never stored, naming the leaf that moved
//     (ADR-0008). The package offers no way to compare a value across a Break.
//   - A Span is never compacted or deleted. This package folds observations into
//     spans and reads spans back; it has no delete path, and the store it drives
//     has none either (ADR-0041).
//   - `appeared` and `returned` are membership-only, because they describe a
//     subject; `revealed` belongs to any timeline, because aperture is a property
//     of looking and looking is per-timeline (ADR-0014). The API keeps the two
//     families apart so a caller cannot name a facet-timeline opening `returned`.
package drift

import (
	"sort"
	"time"
)

// TimelineKey identifies one Span timeline: the five-part key a Span is held
// under (ADR-0007). `Discriminator` carries the qtype for `dns-record` and is
// empty for `resolution`; `Source` is `resolver` for our own resolver and the
// operator's zone file otherwise — one timeline per source, so two sources that
// disagree hold two true facts rather than an arbitration.
type TimelineKey struct {
	SubjectKind   string
	SubjectKey    string
	Facet         string
	Discriminator string
	Vantage       string
	Source        string
}

// Component is one named leaf's version within a Derivation vector — one leaf
// per named Derivation (ADR-0008).
type Component struct {
	Leaf    string `json:"leaf"`
	Version string `json:"version"`
}

// Vector is the flattened, deduped set of Derivation component versions a Span
// was produced under. Equality is a set comparison and a Break names a leaf, so
// the vector is kept flat (never nested) and sorted by leaf name. Composition is
// by absorbing the whole vector of every derivation read (ADR-0008): a
// `resolution` value is decided by two leaves jointly, so its vector is their
// union (ADR-0086).
type Vector []Component

// NewVector returns the canonical (sorted, deduped) vector over the given
// components. A later component with the same leaf wins, so a caller composing
// two derivations may append freely and let this fold the union.
func NewVector(components ...Component) Vector {
	byLeaf := make(map[string]string, len(components))
	for _, c := range components {
		byLeaf[c.Leaf] = c.Version
	}
	out := make(Vector, 0, len(byLeaf))
	for leaf, version := range byLeaf {
		out = append(out, Component{Leaf: leaf, Version: version})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Leaf < out[j].Leaf })
	return out
}

// Equal reports whether two vectors are the same set of (leaf, version) pairs.
// This is the whole comparability precondition: two spans compare only where
// Equal holds, and the boundary between two spans where it does not is a Break.
func (v Vector) Equal(o Vector) bool {
	if len(v) != len(o) {
		return false
	}
	a, b := NewVector(v...), NewVector(o...)
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// MovedLeaves names the leaves whose version differs between two vectors — the
// leaf a Break names as having moved. A leaf present in one vector and absent
// from the other has moved too (the membership vector is discovered, not
// declared, so a leaf entering it is a real move; ADR-0086). The result is
// sorted for a stable rendering.
func MovedLeaves(before, after Vector) []string {
	bv := map[string]string{}
	for _, c := range before {
		bv[c.Leaf] = c.Version
	}
	av := map[string]string{}
	for _, c := range after {
		av[c.Leaf] = c.Version
	}
	moved := map[string]struct{}{}
	for leaf, ver := range av {
		if bv[leaf] != ver {
			moved[leaf] = struct{}{}
		}
	}
	for leaf, ver := range bv {
		if av[leaf] != ver {
			moved[leaf] = struct{}{}
		}
	}
	out := make([]string, 0, len(moved))
	for leaf := range moved {
		out = append(out, leaf)
	}
	sort.Strings(out)
	return out
}

// ClosureReason is the ground a withdrawal closure records — a closed union of
// three, sorted on what the closure rests on (ADR-0087). It is carried only by a
// withdrawal's closure; an ordinary value move needs none (the next span is the
// fact) and a version change needs none (the Break is derived from the two
// vectors). Only `descoped` suppresses a later `returned`.
type ClosureReason string

const (
	// ReasonMeasuredAbsent: an observation about this subject says it is absent —
	// a Name measured NameError on a cross-class Vantage composition. The only
	// closure that is independent evidence.
	ReasonMeasuredAbsent ClosureReason = "measured-absent"
	// ReasonUncited: the subject's chain back to a Seed no longer holds — a child
	// beneath a withdrawn root, or a resolution that stopped citing an Address.
	// Covers both the cascade and de-citation.
	ReasonUncited ClosureReason = "uncited"
	// ReasonDescoped: our aperture stopped covering the subject — an exclusion, a
	// narrower Seed, or a release narrowing a composed population. The only ground
	// that blocks a subsequent `returned`, because a narrowing is not a
	// decommission.
	ReasonDescoped ClosureReason = "descoped"
)

// Valid reports whether a reason is one of the three closed grounds.
func (r ClosureReason) Valid() bool {
	switch r {
	case ReasonMeasuredAbsent, ReasonUncited, ReasonDescoped:
		return true
	default:
		return false
	}
}

// Span is one period a (subject, facet, discriminator, vantage, source) timeline
// held a single value — opened, current, closed. It carries the Derivation
// vector it was produced under, and — only where it is a withdrawal's closing
// side — the ground that closure rests on. A Span is immutable and carries no
// operator state; a drift record is a measurement, not a work item (ADR-0007).
type Span struct {
	Key      TimelineKey
	Value    string // the canonical value, structurally compared; empty when IsGap
	IsGap    bool   // a period over which the system could not say (never a withdrawal)
	Vector   Vector
	OpenedAt time.Time
	ClosedAt time.Time     // zero => open / current
	Reason   ClosureReason // set only on a withdrawal closure
}

// Open reports whether the span is the timeline's current one.
func (s Span) Open() bool { return s.ClosedAt.IsZero() }
