package drift

import "time"

func windowStart(now time.Time, bucket time.Duration, buckets int) time.Time {
	// Shared so the signals, discovery and withdrawal series all bucket identically.
	return now.Add(-bucket * time.Duration(buckets))
}

func bucketIndex(at, start time.Time, bucket time.Duration, buckets int) (int, bool) {
	if at.Before(start) {
		return 0, false
	}
	idx := int(at.Sub(start) / bucket)
	if idx < 0 || idx >= buckets {
		return 0, false
	}
	return idx, true
}

// The instant is signal_instance.first_seen, when the (rule, subject) pair was first seen firing.

type Raise struct {
	At       time.Time
	Elevated bool
}

type SignalPoint struct {
	Start            time.Time
	Count            int
	Elevated         int
	Standing         int
	StandingElevated int
}

func SignalsOverTime(raises []Raise, now time.Time, bucket time.Duration, buckets int) []SignalPoint {
	// The series come from the estate's own history, never the port's fabricated mock (#444).
	start := windowStart(now, bucket, buckets)
	points := make([]SignalPoint, buckets)
	for i := range points {
		points[i].Start = start.Add(bucket * time.Duration(i))
	}
	// Severity is resolved by internal/signal and passed as a bool, so no dependency rides here.
	for _, rs := range raises {
		if idx, ok := bucketIndex(rs.At, start, bucket, buckets); ok {
			points[idx].Count++
			if rs.Elevated {
				points[idx].Elevated++
			}
		}
		// A signal raised before the window still stands, so the standing level counts it.
		for i := 0; i < buckets; i++ {
			bucketEnd := start.Add(bucket * time.Duration(i+1))
			if rs.At.Before(bucketEnd) {
				points[i].Standing++
				if rs.Elevated {
					points[i].StandingElevated++
				}
			}
		}
	}
	return points
}

const heatSteps = 4

func HeatLevels(counts []int) []int {
	// The ramp is the HeatmapCalendar's: level 0, then 1..4 in quartiles of the busiest day.
	// The 0-4 ramp is one rule, and a second surface calls this and never re-derives it.
	max := 1
	for _, c := range counts {
		if c > max {
			max = c
		}
	}
	levels := make([]int, len(counts))
	for i, c := range counts {
		if c <= 0 {
			continue
		}
		level := (c*heatSteps + max - 1) / max
		if level < 1 {
			level = 1
		}
		if level > heatSteps {
			level = heatSteps
		}
		levels[i] = level
	}
	return levels
}

// A subject is withdrawn by the world, never resolved by an operator: this is not a resolve time.

type Withdrawal struct {
	Appeared  time.Time
	Withdrawn time.Time
}

func (w Withdrawal) Duration() (time.Duration, bool) {
	// Clock skew and a partial corpus both produce an unusable interval the caller must drop.
	if w.Appeared.IsZero() || w.Withdrawn.IsZero() || !w.Withdrawn.After(w.Appeared) {
		return 0, false
	}
	return w.Withdrawn.Sub(w.Appeared), true
}

func MeanTimeToWithdrawal(ws []Withdrawal) (time.Duration, bool) {
	var total time.Duration
	var n int
	for _, w := range ws {
		if d, ok := w.Duration(); ok {
			total += d
			n++
		}
	}
	if n == 0 {
		return 0, false
	}
	return total / time.Duration(n), true
}

type WithdrawalPoint struct {
	Start   time.Time
	Mean    time.Duration
	Count   int
	HasMean bool
}

// Only Name and Service subjects count, so an Endpoint or Address facet moving is not a discovery.

type Appearance struct {
	At      time.Time
	Service bool
}

type DiscoveryPoint struct {
	Start time.Time
	Count int
}

func DiscoverySeries(apps []Appearance, now time.Time, bucket time.Duration, buckets int) []DiscoveryPoint {
	start := windowStart(now, bucket, buckets)
	points := make([]DiscoveryPoint, buckets)
	for i := range points {
		points[i].Start = start.Add(bucket * time.Duration(i))
	}
	for _, a := range apps {
		if idx, ok := bucketIndex(a.At, start, bucket, buckets); ok {
			points[idx].Count++
		}
	}
	return points
}

type DiscoveryTotals struct {
	Total    int
	Names    int
	Services int
}

func DiscoveryCount(apps []Appearance, start, end time.Time) DiscoveryTotals {
	var t DiscoveryTotals
	// The window is half-open, so counting [windowStart, now) sums to the series exactly.
	for _, a := range apps {
		if a.At.Before(start) || !a.At.Before(end) {
			continue
		}
		t.Total++
		if a.Service {
			t.Services++
		} else {
			t.Names++
		}
	}
	return t
}

func WithdrawalSeries(ws []Withdrawal, now time.Time, bucket time.Duration, buckets int) []WithdrawalPoint {
	// A bucket with no data reports it, so the caller draws the design's gap and never a zero.
	start := windowStart(now, bucket, buckets)
	points := make([]WithdrawalPoint, buckets)
	totals := make([]time.Duration, buckets)
	for i := range points {
		points[i].Start = start.Add(bucket * time.Duration(i))
	}
	for _, w := range ws {
		d, ok := w.Duration()
		if !ok {
			continue
		}
		idx, in := bucketIndex(w.Withdrawn, start, bucket, buckets)
		if !in {
			continue
		}
		totals[idx] += d
		points[idx].Count++
	}
	for i := range points {
		if points[i].Count > 0 {
			points[i].Mean = totals[i] / time.Duration(points[i].Count)
			points[i].HasMean = true
		}
	}
	return points
}
