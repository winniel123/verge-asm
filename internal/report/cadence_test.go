package report

import (
	"testing"
	"time"
)

// CadenceWindow maps the stored cadence label to the coarse, model-owned window a
// run covers. The vocabulary is closed: a custom / cron cadence is never evaluated
// as a predicate (ADR-0091/ADR-0117) — it falls to the weekly default, exactly as
// Run-now already treats it.
func TestCadenceWindowMapsLabelsToWindows(t *testing.T) {
	cases := []struct {
		cadence string
		want    time.Duration
	}{
		{"every 6h", 6 * time.Hour},
		{"daily · 08:00", 24 * time.Hour},
		{"weekly · mon 09:00", 7 * 24 * time.Hour},
		{"monthly · 1st", 30 * 24 * time.Hour},
		// A custom cron string is not a preset — it falls to the weekly window, never
		// interpreted as an operator-authored predicate over a versioned rule set.
		{"0 9 * * 1", 7 * 24 * time.Hour},
		{"", 7 * 24 * time.Hour},
		{"anything unrecognised", 7 * 24 * time.Hour},
		// Case-insensitive: the label is lower-cased before matching.
		{"Every 6h", 6 * time.Hour},
		{"DAILY", 24 * time.Hour},
	}
	for _, c := range cases {
		if got := CadenceWindow(c.cadence); got != c.want {
			t.Errorf("CadenceWindow(%q) = %s, want %s", c.cadence, got, c.want)
		}
	}
}

// scheduledTick floors now to the cadence boundary, so two ticks in one window
// resolve to one key (the second dispatch conflicts and is a recorded skip) while a
// tick in the next window is a distinct key (a new run). This is the on-cadence
// idempotency contract, computed identically to package queue.
func TestScheduledTickIsIdempotentWithinAWindow(t *testing.T) {
	window := 24 * time.Hour
	base := time.Date(2026, 8, 15, 3, 0, 0, 0, time.UTC)

	a := scheduledTick(base, window)
	b := scheduledTick(base.Add(6*time.Hour), window)
	if !a.Equal(b) {
		t.Errorf("two ticks in one daily window differ: %s vs %s", a, b)
	}

	c := scheduledTick(base.Add(25*time.Hour), window)
	if a.Equal(c) {
		t.Errorf("next window collapsed onto this one: %s", c)
	}

	// The tick is floored to the window boundary — UTC midnight for a daily window.
	wantBoundary := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	if !a.Equal(wantBoundary) {
		t.Errorf("daily tick = %s, want floor to UTC midnight %s", a, wantBoundary)
	}
}

// The period bounds a run stamps are deterministic per tick: end at the tick, start
// one window before it. Every poll inside the window computes the same bounds, so a
// won claim and a skipped one agree on the window they name.
func TestPeriodBoundsAreDeterministicPerTick(t *testing.T) {
	window := 7 * 24 * time.Hour
	now1 := time.Date(2026, 8, 20, 4, 30, 0, 0, time.UTC)
	now2 := now1.Add(3 * time.Hour)

	t1 := scheduledTick(now1, window)
	t2 := scheduledTick(now2, window)
	if !t1.Equal(t2) {
		t.Fatalf("two polls in one weekly window floored differently: %s vs %s", t1, t2)
	}
	// end = tick, start = tick - window.
	if start := t1.Add(-window); !start.Equal(t2.Add(-window)) {
		t.Errorf("period start diverged across polls: %s vs %s", start, t2.Add(-window))
	}
}
