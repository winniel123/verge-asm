package report

import (
	"testing"
	"time"
)

func TestCadenceWindowMapsLabelsToWindows(t *testing.T) {
	// The window is only the run's period; when it fires is a separate concern (ADR-0122).
	cases := []struct {
		cadence string
		want    time.Duration
	}{
		{"every 6h", 6 * time.Hour},
		{"daily · 08:00", 24 * time.Hour},
		{"weekly · mon 09:00", 7 * 24 * time.Hour},
		{"monthly · 1st", 30 * 24 * time.Hour},
		{"0 9 * * 1", 7 * 24 * time.Hour},
		{"", 7 * 24 * time.Hour},
		{"anything unrecognised", 7 * 24 * time.Hour},
		{"Every 6h", 6 * time.Hour},
		{"DAILY", 24 * time.Hour},
	}
	for _, c := range cases {
		if got := CadenceWindow(c.cadence); got != c.want {
			t.Errorf("CadenceWindow(%q) = %s, want %s", c.cadence, got, c.want)
		}
	}
}

func utc(y int, mo time.Month, d, h, mi int) time.Time {
	return time.Date(y, mo, d, h, mi, 0, 0, time.UTC)
}

func TestDispatchTickHonoursDeclaredTime(t *testing.T) {
	cases := []struct {
		name    string
		cadence string
		now     time.Time
		want    time.Time
	}{
		{"daily after 08:00", "daily · 08:00", utc(2026, 8, 15, 10, 0), utc(2026, 8, 15, 8, 0)},
		{"daily before 08:00 → yesterday", "daily · 08:00", utc(2026, 8, 15, 7, 59), utc(2026, 8, 14, 8, 0)},
		{"daily exactly at 08:00", "daily · 08:00", utc(2026, 8, 15, 8, 0), utc(2026, 8, 15, 8, 0)},

		{"6h mid-window", "every 6h", utc(2026, 8, 15, 13, 0), utc(2026, 8, 15, 12, 0)},
		{"6h before first boundary", "every 6h", utc(2026, 8, 15, 5, 59), utc(2026, 8, 15, 0, 0)},

		// 2026-08-17 is a Monday, which is what makes these two rows the weekly firing.
		{"weekly after Mon 09:00", "weekly · mon 09:00", utc(2026, 8, 18, 12, 0), utc(2026, 8, 17, 9, 0)},
		{"weekly before Mon 09:00 → prior Monday", "weekly · mon 09:00", utc(2026, 8, 17, 8, 0), utc(2026, 8, 10, 9, 0)},

		{"monthly mid-month", "monthly · 1st", utc(2026, 8, 15, 4, 0), utc(2026, 8, 1, 0, 0)},
		{"monthly on the 1st", "monthly · 1st", utc(2026, 9, 1, 5, 0), utc(2026, 9, 1, 0, 0)},

		{"cron 0 8 * * *", "0 8 * * *", utc(2026, 8, 15, 10, 0), utc(2026, 8, 15, 8, 0)},
		{"cron */15 mid-quarter", "*/15 * * * *", utc(2026, 8, 15, 10, 7), utc(2026, 8, 15, 10, 0)},
		{"cron */15 later in hour", "*/15 * * * *", utc(2026, 8, 15, 10, 47), utc(2026, 8, 15, 10, 45)},
		{"cron 0 9 * * 1 (Monday)", "0 9 * * 1", utc(2026, 8, 18, 12, 0), utc(2026, 8, 17, 9, 0)},
	}
	for _, c := range cases {
		got, ok := DispatchTick(c.now, c.cadence)
		if !ok {
			t.Errorf("%s: DispatchTick(%q) not ok", c.name, c.cadence)
			continue
		}
		if !got.Equal(c.want) {
			t.Errorf("%s: DispatchTick(%s, %q) = %s, want %s", c.name, c.now, c.cadence, got, c.want)
		}
	}
}

func TestDispatchTickIsIdempotentBetweenFirings(t *testing.T) {
	const cadence = "daily · 08:00"

	// The dispatcher's (schedule_id, scheduled_tick) key admits only the first poll after a firing.
	a, _ := DispatchTick(utc(2026, 8, 15, 8, 0), cadence)
	b, _ := DispatchTick(utc(2026, 8, 15, 8, 59), cadence)
	c, _ := DispatchTick(utc(2026, 8, 15, 23, 59), cadence)
	if !a.Equal(b) || !a.Equal(c) {
		t.Errorf("polls between firings differ: %s / %s / %s", a, b, c)
	}
	if want := utc(2026, 8, 15, 8, 0); !a.Equal(want) {
		t.Errorf("fire tick = %s, want %s", a, want)
	}

	next, _ := DispatchTick(utc(2026, 8, 16, 8, 0), cadence)
	if next.Equal(a) {
		t.Errorf("next firing collapsed onto this one: %s", next)
	}
	if want := utc(2026, 8, 16, 8, 0); !next.Equal(want) {
		t.Errorf("next fire tick = %s, want %s", next, want)
	}
}

func TestDispatchTickCronIdempotent(t *testing.T) {
	const cadence = "*/15 * * * *"
	a, _ := DispatchTick(utc(2026, 8, 15, 10, 15), cadence)
	b, _ := DispatchTick(utc(2026, 8, 15, 10, 29), cadence)
	if !a.Equal(b) {
		t.Errorf("two polls in one 15-min window differ: %s vs %s", a, b)
	}
	c, _ := DispatchTick(utc(2026, 8, 15, 10, 30), cadence)
	if c.Equal(a) {
		t.Errorf("next quarter collapsed onto this one: %s", c)
	}
}

func TestDispatchTickRejectsUninterpretable(t *testing.T) {
	for _, cadence := range []string{"custom", "not a cadence", "0 8 * *", ""} {
		if _, ok := DispatchTick(utc(2026, 8, 15, 10, 0), cadence); ok {
			t.Errorf("DispatchTick(%q) unexpectedly ok", cadence)
		}
	}
}

func TestValidateCron(t *testing.T) {
	valid := []string{
		"0 8 * * *", "*/15 * * * *", "0 9 * * 1", "0 0 1 * *", "0 */6 * * *",
		"5,35 8-17 * * 1-5", "0 8 * * 0", "0 8 * * 7", "30 2 15 6 *",
	}
	for _, e := range valid {
		if err := ValidateCron(e); err != nil {
			t.Errorf("ValidateCron(%q) = %v, want nil", e, err)
		}
	}
	invalid := []string{
		"", "0 8 * *", "0 8 * * * *", "60 8 * * *", "0 24 * * *",
		"0 8 32 * *", "0 8 * 13 *", "0 8 * * 8", "abc", "*/0 * * * *", "0 8 0 * *",
	}
	for _, e := range invalid {
		if err := ValidateCron(e); err == nil {
			t.Errorf("ValidateCron(%q) = nil, want error", e)
		}
	}
}

func TestDispatchTickDomDowUnion(t *testing.T) {
	// Vixie-cron day semantics: a restricted dom and dow union rather than intersect.
	// 2026-08-17 is a Monday and not the 1st, so only the dow limb can match it.
	cadence := "0 0 1 * 1"
	got, ok := DispatchTick(utc(2026, 8, 17, 12, 0), cadence)
	if !ok || !got.Equal(utc(2026, 8, 17, 0, 0)) {
		t.Errorf("dom/dow union: got %s (ok=%v), want 2026-08-17T00:00", got, ok)
	}
	// 2026-08-16 is a Sunday, so the most recent match is the Monday before it.
	got2, _ := DispatchTick(utc(2026, 8, 16, 12, 0), cadence)
	if !got2.Equal(utc(2026, 8, 10, 0, 0)) {
		t.Errorf("dom/dow union prior: got %s, want 2026-08-10T00:00", got2)
	}
}
