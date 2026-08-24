package drift

import "time"

// Trend series for the Reports screen (P0.3, PARITY-CHART.md; #444). The design
// (design-system/examples/console/Reports.jsx) renders three real series this file
// derives from the estate's own history rather than the fabricated mock the port
// once re-skinned away (SPEC-CHANGE.md collision #3): signals over time, the
// scans-per-day heatmap intensities, and the mean-time-to-withdrawal trend.
//
// Everything here is a PURE fold over instants and counts the web layer reads off
// the persisted corpus — the per-instance `signal_instance` first-seen ledger and
// the never-compacted `span` corpus (ADR-0041). The severity ramp is presentation,
// resolved by the web layer (internal/signal) and passed in as a bool, so this core
// carries no dependency on the severity model: the count is the datum, the colour is
// the render. No value is fabricated — a bucket with no data reports it as such so
// the caller renders the design's own empty/gap pattern rather than a standing zero.

// --- Windowing ---------------------------------------------------------------

// windowStart is the oldest instant a series over `buckets` buckets of width
// `bucket` ending at `now` covers. Bucket i (0 = oldest) spans
// [windowStart + i*bucket, windowStart + (i+1)*bucket); the last bucket ends at
// `now`. Kept in one place so the signals and withdrawal series bucket identically.
func windowStart(now time.Time, bucket time.Duration, buckets int) time.Time {
	return now.Add(-bucket * time.Duration(buckets))
}

// bucketIndex returns the bucket an instant falls in over the window, and whether it
// falls inside it at all. The window is half-open per bucket: an instant on a bucket
// boundary belongs to the newer bucket, and one at exactly `now` (the last bucket's
// upper bound) is outside.
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

// --- Signals over time -------------------------------------------------------

// Raise is one signal instance as the trend fold sees it: the instant the
// (rule, subject) pair was FIRST seen firing (signal_instance.first_seen) and
// whether the rule's severity is elevated — critical or high, the design's
// "Critical + high" series. Severity is the rule's, resolved by the web layer and
// passed as a bool so this core needs no severity import.
type Raise struct {
	At       time.Time
	Elevated bool
}

// SignalPoint is one bucket of the signals-over-time series. Count/Elevated are the
// signals FIRST raised within the bucket (incidence); Standing/StandingElevated are
// the running totals — every signal raised on or before the bucket's close,
// including any raised before the window — the standing level the design's "Open
// signals over time" line reads. Both are real: the per-instance corpus records when
// each signal was first raised and is never compacted.
type SignalPoint struct {
	Start            time.Time
	Count            int
	Elevated         int
	Standing         int
	StandingElevated int
}

// SignalsOverTime folds a set of signal raises into a per-bucket series over the
// window ending at `now`, oldest bucket first. Incidence (Count/Elevated) counts the
// raises whose first-seen instant fell in the bucket; standing (Standing/
// StandingElevated) accumulates every raise on or before the bucket's close, so a
// signal raised before the window still lifts the standing level it is still part of.
// A nil/empty raise set yields a series of empty buckets, never a fabricated shape.
func SignalsOverTime(raises []Raise, now time.Time, bucket time.Duration, buckets int) []SignalPoint {
	start := windowStart(now, bucket, buckets)
	points := make([]SignalPoint, buckets)
	for i := range points {
		points[i].Start = start.Add(bucket * time.Duration(i))
	}
	for _, rs := range raises {
		// Incidence: only raises inside the window land in a bucket.
		if idx, ok := bucketIndex(rs.At, start, bucket, buckets); ok {
			points[idx].Count++
			if rs.Elevated {
				points[idx].Elevated++
			}
		}
		// Standing: a raise contributes to every bucket whose close is at or after
		// its instant — i.e. the first bucket whose upper bound is > rs.At onward.
		// Pre-window raises (rs.At < start) lift every bucket; in-window raises lift
		// from their own bucket forward.
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

// --- Scans-per-day heatmap intensities ---------------------------------------

// heatSteps is the number of non-zero intensity levels the scans-per-day heatmap
// ramps through, matching HeatmapCalendar.jsx (four steps above the empty day).
const heatSteps = 4

// HeatLevels maps a per-day scan-count series to the HeatmapCalendar intensity ramp:
// level 0 for a day with no scans, then 1..4 in equal quartiles of the busiest day
// (ceil(count/max * 4)). It is the pure core of the scans-per-day heatmap, shared so
// the page fold, the export and any later surface ramp identically off one rule. The
// busiest day is floored at 1, so an all-zero series is every day at level 0 rather
// than a divide-by-zero.
func HeatLevels(counts []int) []int {
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
		level := (c*heatSteps + max - 1) / max // ceil(c/max * heatSteps)
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

// --- Mean time to withdrawal -------------------------------------------------

// Withdrawal is one subject departure as the trend fold sees it: the instant the
// subject first appeared (its earliest span opened_at) and the instant it was
// withdrawn (the closure of its timelines, ADR-0082). Time-to-withdrawal is
// Withdrawn - Appeared. Both come from the never-compacted span corpus (ADR-0041),
// so the interval is measured, never fabricated. Signals are withdrawn by the world,
// never "resolved" by an operator — so this is time-to-WITHDRAWAL, not a resolve
// time (the mock's mean-time-to-resolve KPI, SPEC-CHANGE.md collision #3).
type Withdrawal struct {
	Appeared  time.Time
	Withdrawn time.Time
}

// Duration is the subject's time-to-withdrawal — how long it stood in the estate
// before departing — and whether it is a usable interval. A withdrawal with an
// unknown appearance, an unknown withdrawal, or a withdrawal not strictly after the
// appearance (clock skew, a partial corpus) reports (0, false) so the caller drops
// it from the mean rather than folding a zero or negative interval.
func (w Withdrawal) Duration() (time.Duration, bool) {
	if w.Appeared.IsZero() || w.Withdrawn.IsZero() || !w.Withdrawn.After(w.Appeared) {
		return 0, false
	}
	return w.Withdrawn.Sub(w.Appeared), true
}

// MeanTimeToWithdrawal is the mean time-to-withdrawal across a set of withdrawals,
// and whether any valid interval contributed. An empty set, or one whose every
// interval is unusable, reports (0, false) so the caller renders the KPI unavailable
// rather than a fabricated zero.
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

// WithdrawalPoint is one bucket of the mean-time-to-withdrawal trend: the bucket
// start, the mean time-to-withdrawal of the withdrawals that OCCURRED in it, the
// count of valid intervals folded, and whether the bucket carried any (HasMean=false
// renders a gap rather than a zero).
type WithdrawalPoint struct {
	Start   time.Time
	Mean    time.Duration
	Count   int
	HasMean bool
}

// --- New assets discovered ---------------------------------------------------

// Appearance is one subject's FIRST appearance as the discovery fold sees it: the
// instant it first entered the estate — its earliest span opened_at, the `appeared`
// drift-event classification — and whether that subject is a Service (vs a Name),
// so the caller can split the discovery count by kind for the card's caption. It
// comes from the never-compacted span corpus (ADR-0041), so the instant is
// measured, never fabricated. Only Name/Service subjects are counted — the same
// watched population the assets-watched census reads (DistinctSubjects) — so an
// Endpoint or Address facet moving is not itself a newly discovered asset.
type Appearance struct {
	At      time.Time
	Service bool
}

// DiscoveryPoint is one bucket of the new-assets-discovered series: the bucket
// start and the count of subjects that FIRST appeared within it — the daily bars
// of the Reports "New assets discovered" card (Reports.jsx DISCOVERY). A bucket
// with no appearance reports Count 0, a real empty bar, not a fabricated shape.
type DiscoveryPoint struct {
	Start time.Time
	Count int
}

// DiscoverySeries buckets first-appearances by their appearance instant over the
// window ending at `now`, oldest bucket first — the daily-discovery BarChart of the
// Reports screen. Same windowing as SignalsOverTime, so a discovery column lines up
// with the other trends. Only an appearance inside the window lands in a bucket; an
// empty set yields all-zero buckets rather than an invented series.
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

// DiscoveryTotals is the per-period discovery summary the card's KPI and caption
// read: the total subjects that first appeared in the window, split into Names and
// Services (Names + Services == Total). The count is the "New assets discovered"
// value; the split is its "N names · M services" caption.
type DiscoveryTotals struct {
	Total    int
	Names    int
	Services int
}

// DiscoveryCount folds the appearances whose instant is in [start, end) into the
// per-period totals — the count the KPI shows and the name/service split its caption
// reads. The half-open window matches DiscoverySeries' bucketing (an appearance
// exactly at `end` is outside), so counting [windowStart, now) sums to the series.
func DiscoveryCount(apps []Appearance, start, end time.Time) DiscoveryTotals {
	var t DiscoveryTotals
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

// WithdrawalSeries buckets withdrawals by their WITHDRAWAL instant over the window
// ending at `now`, oldest bucket first, and computes each bucket's mean
// time-to-withdrawal. A bucket with no valid withdrawal reports HasMean=false so the
// caller draws the design's gap rather than a standing zero. Same windowing as
// SignalsOverTime, so the two trends align column-for-column.
func WithdrawalSeries(ws []Withdrawal, now time.Time, bucket time.Duration, buckets int) []WithdrawalPoint {
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
