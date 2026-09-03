package scan

// A single failed query must not flip the 99% bar (ct-source-replacement.md §3).

const CTReliabilityWindowSize = 200

const (
	CTSuccessRateBar  = 0.99
	CTP95LatencyBarMS = 5000
)

type CTReliabilityWindow struct {
	Total        int64
	Successes    int64
	Empties      int64 // succeeded yet returned zero names — the false-empty limb (§3)
	P95LatencyMS int64
}

// A degraded primary is surfaced, never silently swapped: runtime failover is deferred (§7).

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

func EvaluateCTReliability(source string, w CTReliabilityWindow) CTReliabilityReport {
	// crt.sh is exempt because it is the keyless fallback: there is nothing to fall back to (§3).
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
	if !r.Exempt && r.HasData {
		r.Degraded = !(r.SuccessPass && r.LatencyPass && r.FalseEmptyPass)
	}
	return r
}
