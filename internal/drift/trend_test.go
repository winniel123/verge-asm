package drift

import (
	"testing"
	"time"
)

var trendNow = time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)

const week = 7 * 24 * time.Hour

func TestHeatLevelsAllZeroIsAllLevelZero(t *testing.T) {
	got := HeatLevels([]int{0, 0, 0})
	for i, l := range got {
		if l != 0 {
			t.Fatalf("day %d: want level 0, got %d", i, l)
		}
	}
}

func TestHeatLevelsRampsInQuartilesOfTheBusiestDay(t *testing.T) {
	counts := []int{0, 1, 2, 3, 4, 5, 6, 7, 8}
	want := []int{0, 1, 1, 2, 2, 3, 3, 4, 4}
	got := HeatLevels(counts)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("count %d (day %d): want level %d, got %d", counts[i], i, want[i], got[i])
		}
	}
}

func TestHeatLevelsSingleBusyDayIsMaxLevel(t *testing.T) {
	got := HeatLevels([]int{0, 3, 0})
	if got[1] != heatSteps {
		t.Fatalf("the only non-zero day should be the max level %d, got %d", heatSteps, got[1])
	}
}

func TestHeatLevelsNeverExceedsTheRamp(t *testing.T) {
	for _, l := range HeatLevels([]int{1, 1, 1, 100}) {
		if l < 0 || l > heatSteps {
			t.Fatalf("level %d out of [0,%d]", l, heatSteps)
		}
	}
}

func TestSignalsOverTimeBucketsIncidenceByFirstSeen(t *testing.T) {
	start := windowStart(trendNow, week, 4)
	raises := []Raise{
		{At: start.Add(1 * time.Hour)},
		{At: start.Add(week + 1*time.Hour)},
		{At: start.Add(week + 2*time.Hour)},
		{At: start.Add(3*week + 1*time.Hour)},
	}
	pts := SignalsOverTime(raises, trendNow, week, 4)
	if len(pts) != 4 {
		t.Fatalf("want 4 buckets, got %d", len(pts))
	}
	wantCount := []int{1, 2, 0, 1}
	for i, w := range wantCount {
		if pts[i].Count != w {
			t.Fatalf("bucket %d incidence: want %d, got %d", i, w, pts[i].Count)
		}
	}
}

func TestSignalsOverTimeStandingAccumulatesAndCountsPreWindowRaises(t *testing.T) {
	start := windowStart(trendNow, week, 4)
	raises := []Raise{
		{At: start.Add(-2 * week)},
		{At: start.Add(1 * time.Hour)},
		{At: start.Add(2*week + 1*time.Hour)},
	}
	pts := SignalsOverTime(raises, trendNow, week, 4)
	wantStanding := []int{2, 2, 3, 3}
	for i, w := range wantStanding {
		if pts[i].Standing != w {
			t.Fatalf("bucket %d standing: want %d, got %d", i, w, pts[i].Standing)
		}
	}
	if pts[0].Count != 1 {
		t.Fatalf("bucket 0 incidence should exclude the pre-window raise, got %d", pts[0].Count)
	}
}

func TestSignalsOverTimeSplitsElevated(t *testing.T) {
	start := windowStart(trendNow, week, 2)
	raises := []Raise{
		{At: start.Add(1 * time.Hour), Elevated: true},
		{At: start.Add(2 * time.Hour), Elevated: false},
		{At: start.Add(week + 1*time.Hour), Elevated: true},
	}
	pts := SignalsOverTime(raises, trendNow, week, 2)
	if pts[0].Count != 2 || pts[0].Elevated != 1 {
		t.Fatalf("bucket 0: want count 2 elevated 1, got count %d elevated %d", pts[0].Count, pts[0].Elevated)
	}
	if pts[1].Elevated != 1 || pts[1].StandingElevated != 2 {
		t.Fatalf("bucket 1: want elevated 1 standingElevated 2, got %d / %d", pts[1].Elevated, pts[1].StandingElevated)
	}
}

func TestSignalsOverTimeEmptyIsAllZeroBuckets(t *testing.T) {
	pts := SignalsOverTime(nil, trendNow, week, 3)
	if len(pts) != 3 {
		t.Fatalf("want 3 buckets, got %d", len(pts))
	}
	for i, p := range pts {
		if p.Count != 0 || p.Standing != 0 || p.Elevated != 0 {
			t.Fatalf("bucket %d should be empty, got %+v", i, p)
		}
	}
}

func TestDiscoverySeriesBucketsAppearancesByInstant(t *testing.T) {
	const day = 24 * time.Hour
	start := windowStart(trendNow, day, 4)
	apps := []Appearance{
		{At: start.Add(1 * time.Hour)},
		{At: start.Add(day + 1*time.Hour)},
		{At: start.Add(day + 2*time.Hour), Service: true},
		{At: start.Add(3*day + 1*time.Hour)},
		{At: start.Add(-day)},
	}
	pts := DiscoverySeries(apps, trendNow, day, 4)
	if len(pts) != 4 {
		t.Fatalf("want 4 buckets, got %d", len(pts))
	}
	want := []int{1, 2, 0, 1}
	for i, w := range want {
		if pts[i].Count != w {
			t.Fatalf("bucket %d: want %d, got %d", i, w, pts[i].Count)
		}
	}
}

func TestDiscoverySeriesEmptyIsAllZeroBuckets(t *testing.T) {
	pts := DiscoverySeries(nil, trendNow, 24*time.Hour, 3)
	if len(pts) != 3 {
		t.Fatalf("want 3 buckets, got %d", len(pts))
	}
	for i, p := range pts {
		if p.Count != 0 {
			t.Fatalf("bucket %d should be empty, got %d", i, p.Count)
		}
	}
}

func TestDiscoveryCountSplitsNamesAndServicesInWindow(t *testing.T) {
	winStart := trendNow.Add(-week)
	apps := []Appearance{
		{At: winStart.Add(1 * time.Hour)},
		{At: winStart.Add(2 * time.Hour), Service: true},
		{At: winStart.Add(3 * time.Hour), Service: true},
		{At: winStart.Add(-time.Hour)},
		{At: trendNow},
	}
	got := DiscoveryCount(apps, winStart, trendNow)
	if got.Total != 3 || got.Names != 1 || got.Services != 2 {
		t.Fatalf("want total 3 / names 1 / services 2, got %+v", got)
	}
}

func TestWithdrawalDurationValidity(t *testing.T) {
	appeared := trendNow.Add(-3 * week)
	withdrawn := trendNow.Add(-1 * week)
	if d, ok := (Withdrawal{Appeared: appeared, Withdrawn: withdrawn}).Duration(); !ok || d != 2*week {
		t.Fatalf("want 2 weeks valid, got %v ok=%v", d, ok)
	}
	for _, w := range []Withdrawal{
		{Withdrawn: withdrawn},
		{Appeared: appeared},
		{Appeared: withdrawn, Withdrawn: appeared},
		{Appeared: appeared, Withdrawn: appeared},
	} {
		if _, ok := w.Duration(); ok {
			t.Fatalf("expected %+v to be unusable", w)
		}
	}
}

func TestMeanTimeToWithdrawalAveragesValidIntervalsOnly(t *testing.T) {
	ws := []Withdrawal{
		{Appeared: trendNow.Add(-4 * week), Withdrawn: trendNow.Add(-2 * week)},
		{Appeared: trendNow.Add(-6 * week), Withdrawn: trendNow.Add(-2 * week)},
		{Withdrawn: trendNow},
	}
	mean, ok := MeanTimeToWithdrawal(ws)
	if !ok || mean != 3*week {
		t.Fatalf("want mean 3 weeks, got %v ok=%v", mean, ok)
	}
}

func TestMeanTimeToWithdrawalEmptyIsUnavailable(t *testing.T) {
	if _, ok := MeanTimeToWithdrawal(nil); ok {
		t.Fatal("empty set should report unavailable, not a zero mean")
	}
	if _, ok := MeanTimeToWithdrawal([]Withdrawal{{Withdrawn: trendNow}}); ok {
		t.Fatal("all-invalid set should report unavailable")
	}
}

func TestWithdrawalSeriesBucketsByWithdrawalInstant(t *testing.T) {
	start := windowStart(trendNow, week, 4)
	ws := []Withdrawal{
		{Appeared: start.Add(-week + 1*time.Hour), Withdrawn: start.Add(week + 1*time.Hour)},
		{Appeared: start.Add(2*week + 1*time.Hour), Withdrawn: start.Add(3*week + 1*time.Hour)},
	}
	pts := WithdrawalSeries(ws, trendNow, week, 4)
	if len(pts) != 4 {
		t.Fatalf("want 4 buckets, got %d", len(pts))
	}
	if pts[0].HasMean || pts[2].HasMean {
		t.Fatal("buckets with no withdrawal should have HasMean=false (a gap, not a zero)")
	}
	if !pts[1].HasMean || pts[1].Mean != 2*week {
		t.Fatalf("bucket 1: want mean 2 weeks, got %v hasMean=%v", pts[1].Mean, pts[1].HasMean)
	}
	if !pts[3].HasMean || pts[3].Mean != 1*week {
		t.Fatalf("bucket 3: want mean 1 week, got %v hasMean=%v", pts[3].Mean, pts[3].HasMean)
	}
}

func TestBucketIndexBoundaryBelongsToNewerBucket(t *testing.T) {
	start := trendNow.Add(-4 * week)
	if idx, ok := bucketIndex(start.Add(week), start, week, 4); !ok || idx != 1 {
		t.Fatalf("boundary want bucket 1, got %d ok=%v", idx, ok)
	}
	if _, ok := bucketIndex(trendNow, start, week, 4); ok {
		t.Fatal("instant at now should fall outside the window")
	}
	if _, ok := bucketIndex(start.Add(-time.Hour), start, week, 4); ok {
		t.Fatal("instant before the window should be outside")
	}
}
