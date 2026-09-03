package drift

import (
	"sort"
	"time"
)

type Reading struct {
	Value      string
	IsGap      bool
	Vector     Vector
	ObservedAt time.Time
}

func Fold(key TimelineKey, readings []Reading) []Span {
	// Ingest is a fold and never a diff: no two batches are ever paired (ADR-0007).
	if len(readings) == 0 {
		return nil
	}
	ordered := append([]Reading(nil), readings...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].ObservedAt.Before(ordered[j].ObservedAt)
	})

	var spans []Span
	open := spanFromReading(key, ordered[0])
	for _, r := range ordered[1:] {
		if sameSpan(open, r) {
			// A span is a period and never a heartbeat, so a repeated reading writes nothing.
			continue
		}
		open.ClosedAt = r.ObservedAt
		spans = append(spans, open)
		open = spanFromReading(key, r)
	}
	// No closure reason here: withdrawal is a subject-level fact the caller composes (ADR-0082).
	spans = append(spans, open)
	// The last span is left open, so a current-state query is a lookup and never a fold.
	return spans
}

func spanFromReading(key TimelineKey, r Reading) Span {
	return Span{
		Key:      key,
		Value:    r.Value,
		IsGap:    r.IsGap,
		Vector:   NewVector(r.Vector...),
		OpenedAt: r.ObservedAt,
	}
}

func sameSpan(open Span, r Reading) bool {
	// A differing vector opens a new span even where the value is byte-identical (ADR-0008).
	return open.IsGap == r.IsGap &&
		open.Value == r.Value &&
		open.Vector.Equal(NewVector(r.Vector...))
}

func FoldStep(open *Span, key TimelineKey, r Reading) (closeAt time.Time, opened Span, changed bool) {
	if open == nil {
		return time.Time{}, spanFromReading(key, r), true
	}
	if sameSpan(*open, r) {
		return time.Time{}, Span{}, false
	}
	return r.ObservedAt, spanFromReading(key, r), true
}
