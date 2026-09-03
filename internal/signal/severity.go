package signal

// Severity is the five-level urgency ramp a `Signal` rule carries (P0.1;
// design-system/examples/console/SignalData.jsx `sev` / `SEV_ORDER`). It is a
// property of the RULE — assigned per rule and identical for every instance the
// rule raises — never a property of the subject or of the transition that
// surfaces it. A rule's severity moves with a deliberate re-rating, so it is not
// folded into the version vector (which tracks output-affecting evidence, not the
// operator-facing ramp): two censuses of the same rule at different severities are
// still comparable.
//
// This supersedes the old "a signal carries no severity" reading (the ruling in
// PARITY-CHART.md §"The ruling", collision #1 in SPEC-CHANGE.md): the design is
// normative for look AND functionality, and where the domain lacked the datum the
// fix is to build it, per rule, rather than drop the ramp.
type Severity string

const (
	SevCritical Severity = "critical"
	SevHigh     Severity = "high"
	SevMedium   Severity = "medium"
	SevLow      Severity = "low"
	SevInfo     Severity = "info"
)

var SevOrder = []Severity{SevCritical, SevHigh, SevMedium, SevLow, SevInfo}

// Rank is the severity's position in SevOrder — 0 for critical (most urgent),
// rising toward info. An unknown severity ranks past the ramp, so it sorts last
// rather than colliding with critical.
func (s Severity) Rank() int {
	for i, sv := range SevOrder {
		if sv == s {
			return i
		}
	}
	return len(SevOrder)
}

func (s Severity) String() string { return string(s) }

// SeverityFor returns the severity the named rule is assigned, across all three
// subject kinds, and whether the rule exists. It is the web layer's read: a
// per-instance signal's severity is exactly its rule's severity, looked up by
// name — the same key an `Annotation` and the census drill-down are resolved on.
// An unknown rule reports (SevInfo, false), never a panic, so a stale name folds
// to the calmest level rather than manufacturing urgency.
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
	return SevInfo, false
}
