package signal

// A property of the rule, identical for every instance it raises, and never of the subject (P0.1).
// A re-rating is deliberate and stays out of the version vector, so censuses stay comparable.

type Severity string

const (
	SevCritical Severity = "critical"
	SevHigh     Severity = "high"
	SevMedium   Severity = "medium"
	SevLow      Severity = "low"
	SevInfo     Severity = "info"
)

var SevOrder = []Severity{SevCritical, SevHigh, SevMedium, SevLow, SevInfo}

func (s Severity) Rank() int {
	for i, sv := range SevOrder {
		if sv == s {
			return i
		}
	}
	// An unknown severity sorts last rather than colliding with critical at rank zero.
	return len(SevOrder)
}

func (s Severity) String() string { return string(s) }

func SeverityFor(ruleName string) (Severity, bool) {
	for _, r := range All() {
		if r.Name() == ruleName {
			return r.Severity(), true
		}
	}
	for _, r := range AllEndpointRules() {
		if r.Name() == ruleName {
			return r.Severity(), true
		}
	}
	for _, r := range AllServiceRules() {
		if r.Name() == ruleName {
			return r.Severity(), true
		}
	}
	// A stale name folds to the calmest level rather than manufacturing urgency.
	return SevInfo, false
}
