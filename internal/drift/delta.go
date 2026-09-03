package drift

import "time"

// A vs-last-batch delta is the facet-agnostic read this file adds to the drift
// core: the value a stat tile shows now, paired with the value the same metric
// held one batch ago, so a tile can render the signed change (+3, -1, 0) the
// design's stat band shows (design-system PARITY-CHART.md P0.2). It is derived on
// read from the Span timeline exactly as a Transition and a Break are (ADR-0007):
// a Span carries the whole interval a value was held, so the population open at
// any past instant is reconstructable from the same never-compacted corpus
// (ADR-0041) — nothing new is stored. The batch boundary is the previous batch's
// commit instant; the caller reads it from the batch corpus (corpus 1, ADR-0041)
// and hands it in, keeping this package scoped to nothing but spans and time.

type Delta struct {
	Current  int
	Previous int
}

func (d Delta) Change() int { return d.Current - d.Previous }

// openAt reports whether a span was open at instant t: it had opened by t and had
// not yet closed — still open, or closed strictly after t. The interval is
// half-open [OpenedAt, ClosedAt): a span opened exactly at t counts as open at t,
// and a span closed exactly at t does not, so a batch that opens one span and
// closes another at the same instant moves the count by the net of the two.
func openAt(s Span, t time.Time) bool {
	if s.OpenedAt.After(t) {
		return false
	}
	return s.ClosedAt.IsZero() || s.ClosedAt.After(t)
}

func OpenAt(spans []Span, t time.Time) []Span {
	out := make([]Span, 0, len(spans))
	for _, s := range spans {
		if openAt(s, t) {
			out = append(out, s)
		}
	}
	return out
}

// CurrentlyOpen returns the spans that are the current value of their timeline —
// the one open span per timeline the span_open_timeline_idx guarantees. It is
// OpenAt evaluated at now, read as a lookup on ClosedAt rather than a comparison,
// so a fixed-clock caller needs no "now" to read current state.
func CurrentlyOpen(spans []Span) []Span {
	out := make([]Span, 0, len(spans))
	for _, s := range spans {
		if s.Open() {
			out = append(out, s)
		}
	}
	return out
}

// CountDelta computes a vs-last-batch Delta for a metric expressed as a count over
// an open-span population. `count` is applied to the currently-open spans (the
// value now) and to the spans open at `prevBatchAt` (the value a batch ago), both
// drawn from the same `all` set — so the two ends of the delta are one consistent
// reading of one corpus, never a live figure differenced against a reconstructed
// one. `all` must include the spans the most recent batch closed (see OpenAt).
func CountDelta(all []Span, prevBatchAt time.Time, count func(open []Span) int) Delta {
	return Delta{
		Current:  count(CurrentlyOpen(all)),
		Previous: count(OpenAt(all, prevBatchAt)),
	}
}

func CountSpans(open []Span) int { return len(open) }

// DistinctSubjects counts the distinct subjects an open-span population covers —
// one per (subject_kind, subject_key), so a subject holding several open facet
// timelines is one watched asset, not several. It is the counter an "assets
// watched" delta uses over the name/service spans the caller pre-filters in.
func DistinctSubjects(open []Span) int {
	type key struct{ kind, subject string }
	seen := make(map[key]struct{}, len(open))
	for _, s := range open {
		seen[key{s.Key.SubjectKind, s.Key.SubjectKey}] = struct{}{}
	}
	return len(seen)
}
