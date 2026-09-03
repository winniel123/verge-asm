package drift

import "time"

type Delta struct {
	Current  int
	Previous int
}

func (d Delta) Change() int { return d.Current - d.Previous }

func openAt(s Span, t time.Time) bool {
	// Half-open by choice, so a batch that opens one span and closes another nets the two out.
	if s.OpenedAt.After(t) {
		return false
	}
	return s.ClosedAt.IsZero() || s.ClosedAt.After(t)
}

func OpenAt(spans []Span, t time.Time) []Span {
	// Nothing new is stored: a past population is rebuilt from a never-compacted corpus (ADR-0041).
	out := make([]Span, 0, len(spans))
	for _, s := range spans {
		if openAt(s, t) {
			out = append(out, s)
		}
	}
	return out
}

func CurrentlyOpen(spans []Span) []Span {
	// One open span per timeline is guaranteed by the span_open_timeline_idx partial index.
	out := make([]Span, 0, len(spans))
	for _, s := range spans {
		if s.Open() {
			out = append(out, s)
		}
	}
	return out
}

func CountDelta(all []Span, prevBatchAt time.Time, count func(open []Span) int) Delta {
	// The batch instant is the caller's, so this package stays scoped to spans and time.
	// A caller passes the spans the most recent batch closed too, or the previous end is short.
	return Delta{
		Current:  count(CurrentlyOpen(all)),
		Previous: count(OpenAt(all, prevBatchAt)),
	}
}

func CountSpans(open []Span) int { return len(open) }

func DistinctSubjects(open []Span) int {
	type key struct{ kind, subject string }
	// The name/service filter is the caller's, so this counts whatever population it is handed.
	seen := make(map[key]struct{}, len(open))
	for _, s := range open {
		seen[key{s.Key.SubjectKind, s.Key.SubjectKey}] = struct{}{}
	}
	return len(seen)
}
