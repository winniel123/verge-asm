package drift

import (
	"sort"
	"time"
)

// Reading is one observation as the fold sees it: a canonical value (or a gap),
// the Derivation vector it was produced under, and when the source spoke. The
// fold never pairs two batches — it folds a stream of readings on one timeline
// into spans, each reading either extending the open span or closing it and
// opening the next (ADR-0007). Ingest is a fold, not a diff.
type Reading struct {
	Value      string
	IsGap      bool
	Vector     Vector
	ObservedAt time.Time
}

// Fold turns an ordered stream of readings on one timeline into its spans. The
// last span is left open (its ClosedAt is zero) — it is the current state, which
// every current-state query reads as a lookup rather than a fold. A reading
// extends the open span where it carries the same value, the same gap-status and
// an equal vector; otherwise it closes the open span at its own instant and opens
// a new one. A version change (equal value, differing vector) therefore still
// closes and opens — the Break between the two is derived on read from their
// vectors (Breaks), never stored.
//
// Fold assigns no closure reason: an ordinary value move's closure records none
// (the next span is the fact). A withdrawal closure's reason is applied by the
// caller that decides the subject left — see CloseWithdrawal — because withdrawal
// is a subject-level fact composed across timelines, not a per-timeline value
// move.
func Fold(key TimelineKey, readings []Reading) []Span {
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
			// The reading extends the open span: one value held for longer. Nothing
			// is written — a span is a period, not a heartbeat.
			continue
		}
		open.ClosedAt = r.ObservedAt
		spans = append(spans, open)
		open = spanFromReading(key, r)
	}
	spans = append(spans, open)
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

// sameSpan reports whether a reading belongs to the currently-open span: the
// value, the gap-status and the vector must all match. A differing vector opens
// a new span even where the value is byte-identical, because the two are not the
// same kind of thing — that is what makes the Break structural rather than a
// matter of discipline (ADR-0008).
func sameSpan(open Span, r Reading) bool {
	return open.IsGap == r.IsGap &&
		open.Value == r.Value &&
		open.Vector.Equal(NewVector(r.Vector...))
}

// FoldStep is the incremental form of Fold, for the ingest path that folds one
// completed batch at a time (ADR-0007): given the timeline's currently-open span
// (nil where the timeline has none) and one new reading, it reports whether the
// open span must close, and the span the reading opens. Where the reading merely
// extends the open span, both are nil / zero and the caller writes nothing.
//
//   - open == nil: the timeline is new; the reading opens its first span.
//   - the reading extends the open span: closeAt is zero and opened.OpenedAt is
//     zero (the caller writes nothing — the open span already covers it).
//   - the reading differs: closeAt is the reading's instant and opened is the new
//     span. The caller closes the open span at closeAt and inserts opened.
func FoldStep(open *Span, key TimelineKey, r Reading) (closeAt time.Time, opened Span, changed bool) {
	if open == nil {
		return time.Time{}, spanFromReading(key, r), true
	}
	if sameSpan(*open, r) {
		return time.Time{}, Span{}, false
	}
	return r.ObservedAt, spanFromReading(key, r), true
}
