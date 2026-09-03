// Package drift is the v1 drift engine's facet-agnostic core: one period a value
// was held, and everything derived on read from it (v1 spec §5.1, ADR-0007,
// ADR-0008, ADR-0014, ADR-0041).
package drift

import (
	"sort"
	"time"
)

// One timeline per source, so two disagreeing sources hold two true facts and not an arbitration.

type TimelineKey struct {
	SubjectKind   string
	SubjectKey    string
	Facet         string
	Discriminator string
	Vantage       string
	Source        string
}

type Component struct {
	Leaf    string `json:"leaf"`
	Version string `json:"version"`
}

// A caller absorbs each derivation's whole vector, so two leaves may decide a value (ADR-0086).

type Vector []Component

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

func MovedLeaves(before, after Vector) []string {
	// Membership's vector is discovered, not declared: a leaf entering it is a real move (ADR-0086).
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

type ClosureReason string

const (
	ReasonMeasuredAbsent ClosureReason = "measured-absent"
	ReasonUncited        ClosureReason = "uncited"
	ReasonDescoped       ClosureReason = "descoped"
)

func (r ClosureReason) Valid() bool {
	switch r {
	case ReasonMeasuredAbsent, ReasonUncited, ReasonDescoped:
		return true
	default:
		return false
	}
}

// A drift record is a measurement, never a work item, so no operator state rides here (ADR-0007).

type Span struct {
	Key      TimelineKey
	Value    string
	IsGap    bool
	Vector   Vector
	OpenedAt time.Time
	ClosedAt time.Time
	Reason   ClosureReason
}

func (s Span) Open() bool { return s.ClosedAt.IsZero() }
