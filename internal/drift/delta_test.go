package drift

import (
	"testing"
	"time"
)

func spanAt(kind, subject string, open, close time.Time) Span {
	s := Span{
		Key:      TimelineKey{SubjectKind: kind, SubjectKey: subject, Facet: "resolution"},
		OpenedAt: open,
	}
	if !close.IsZero() {
		s.ClosedAt = close
	}
	return s
}

func TestChangeIsCurrentMinusPrevious(t *testing.T) {
	cases := []struct {
		d    Delta
		want int
	}{
		{Delta{Current: 5, Previous: 3}, 2},
		{Delta{Current: 3, Previous: 5}, -2},
		{Delta{Current: 4, Previous: 4}, 0},
	}
	for _, c := range cases {
		if got := c.d.Change(); got != c.want {
			t.Errorf("Delta%+v.Change() = %d, want %d", c.d, got, c.want)
		}
	}
}

func TestOpenAtHalfOpenInterval(t *testing.T) {
	base := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	prev := base.Add(1 * time.Hour)
	latest := base.Add(2 * time.Hour)

	all := []Span{
		spanAt("name", "held.example", base, time.Time{}),
		spanAt("name", "new.example", latest, time.Time{}),
		spanAt("name", "gone.example", base, latest),
	}

	atPrev := OpenAt(all, prev)
	if got := DistinctSubjects(atPrev); got != 2 {
		t.Fatalf("distinct subjects open at prev = %d, want 2 (held + gone)", got)
	}
	if got := DistinctSubjects(CurrentlyOpen(all)); got != 2 {
		t.Fatalf("distinct subjects open now = %d, want 2 (held + new)", got)
	}

	edgeOpen := spanAt("name", "edge.example", prev, time.Time{})
	if !openAt(edgeOpen, prev) {
		t.Error("span opened exactly at t should be open at t")
	}
	edgeClosed := spanAt("name", "edge2.example", base, prev)
	if openAt(edgeClosed, prev) {
		t.Error("span closed exactly at t should not be open at t")
	}
}

func TestCountDeltaNetChangeAcrossBatch(t *testing.T) {
	base := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	prev := base.Add(1 * time.Hour)
	latest := base.Add(2 * time.Hour)

	all := []Span{
		spanAt("service", "a", base, time.Time{}),
		spanAt("service", "b", base, time.Time{}),
		spanAt("service", "c", latest, time.Time{}),
		spanAt("service", "d", base, latest),
	}
	d := CountDelta(all, prev, DistinctSubjects)
	if d.Current != 3 {
		t.Errorf("current = %d, want 3 (a, b, c)", d.Current)
	}
	if d.Previous != 3 {
		t.Errorf("previous = %d, want 3 (a, b, d)", d.Previous)
	}
	if d.Change() != 0 {
		t.Errorf("change = %d, want 0 (one appeared, one withdrawn)", d.Change())
	}
}

func TestCountDeltaGrowth(t *testing.T) {
	base := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	prev := base.Add(1 * time.Hour)
	latest := base.Add(2 * time.Hour)

	all := []Span{
		spanAt("name", "a", base, time.Time{}),
		spanAt("name", "b", latest, time.Time{}),
		spanAt("name", "c", latest, time.Time{}),
	}
	d := CountDelta(all, prev, CountSpans)
	if d.Current != 3 || d.Previous != 1 || d.Change() != 2 {
		t.Errorf("got %+v (change %d), want current 3 / previous 1 / change +2", d, d.Change())
	}
}
