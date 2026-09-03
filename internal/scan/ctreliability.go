package scan

// The measured reliability bar for the bulk CT primary (spec
// docs/spec/ct-source-replacement.md §3, map #854, ticket #879). A primary must
// clear all three limbs; crt.sh is exempt as the keyless fallback. This file holds
// the bar itself and the pure evaluation of one source's rolling-window aggregate
// into pass/fail per limb — the worker records the samples (internal/queue), the web
// reads the window and renders the report this returns.

// CTReliabilityWindowSize is the rolling sample size the bar is measured over: the
// newest this-many bulk-by-name queries per source. The worker trims each source to
// this count on write, and the web reads this many. A whole window of failures is
// only two allowed under the 99% success bar, so the window is wide enough that one
// slow or failed query does not flip the bar, yet narrow enough to reflect the
// source's current behaviour rather than its distant history. Stated in the runbook
// (docs/guides/sources.md) as the measurement method.
const CTReliabilityWindowSize = 200

// The three limbs of the bar (spec §3). A primary clears the bar only when it meets
// all three; the false-empty limb's bar is zero — none.
const (
	CTSuccessRateBar  = 0.99
	CTP95LatencyBarMS = 5000
)

// CTReliabilityWindow is one source's rolling-window aggregate — the shape the
// CTReliabilityWindow query returns, restated here so the evaluation stays pure and
// the internal/db types do not leak into it. Total is how many samples the window
// holds; Successes how many of those succeeded; Empties how many succeeded yet
// returned zero names (false-empty); P95LatencyMS the 95th-percentile latency.
type CTReliabilityWindow struct {
	Total        int64
	Successes    int64
	Empties      int64
	P95LatencyMS int64
}

// CTReliabilityReport is the bar's verdict for one source, shaped for the UI (spec
// §3, §6.2/§6.4). It carries the measured values and a pass/fail per limb. A source
// with no samples has HasData false, so the UI reads "no recent data" rather than a
// false failure. Exempt is true for the keyless fallback (crt.sh): its measured
// values are shown for contrast, muted, never as pass/fail (spec §3). Degraded is
// true only for a non-exempt primary with data that misses any limb — the state the
// UI surfaces without a silent swap to crt.sh (runtime failover is deferred, spec
// §7).
type CTReliabilityReport struct {
	Source  string
	Exempt  bool
	HasData bool
	Samples int64

	SuccessRate float64
	SuccessPass bool

	P95LatencyMS int64
	LatencyPass  bool

	FalseEmpty     int64
	FalseEmptyPass bool

	Degraded bool
}

// EvaluateCTReliability turns one source's window aggregate into its bar report
// (spec §3). It is pure: the bar thresholds and the exempt rule are the only inputs
// beside the aggregate. crt.sh is exempt as the keyless fallback; every other source
// is a primary held to all three limbs. A source with no samples reports HasData
// false and is never degraded, so a just-configured or standby source does not read
// as failing.
func EvaluateCTReliability(source string, w CTReliabilityWindow) CTReliabilityReport {
	r := CTReliabilityReport{
		Source:       source,
		Exempt:       source == CrtshSource,
		Samples:      w.Total,
		P95LatencyMS: w.P95LatencyMS,
		FalseEmpty:   w.Empties,
	}
	if w.Total > 0 {
		r.HasData = true
		r.SuccessRate = float64(w.Successes) / float64(w.Total)
	}
	r.SuccessPass = r.SuccessRate >= CTSuccessRateBar
	r.LatencyPass = w.P95LatencyMS <= CTP95LatencyBarMS
	r.FalseEmptyPass = w.Empties == 0
	// Degraded is a property of a primary that has data and misses a limb. An exempt
	// fallback is never degraded, and a source with no samples is not judged.
	if !r.Exempt && r.HasData {
		r.Degraded = !(r.SuccessPass && r.LatencyPass && r.FalseEmptyPass)
	}
	return r
}
