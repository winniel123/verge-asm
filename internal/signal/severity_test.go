package signal

import "testing"

func inRamp(s Severity) bool {
	for _, sv := range SevOrder {
		if sv == s {
			return true
		}
	}
	return false
}

// TestEveryRuleHasARampSeverity locks the invariant P0.1 builds: every shipped
// rule, across all three subject kinds, is assigned a severity on the five-level
// ramp — never an off-ramp value the SeverityBadge could not key a class off.
func TestEveryRuleHasARampSeverity(t *testing.T) {
	for _, r := range All() {
		if !inRamp(r.Severity()) {
			t.Fatalf("name rule %q severity %q is not on the ramp", r.Name(), r.Severity())
		}
	}
	for _, r := range AllEndpointRules() {
		if !inRamp(r.Severity()) {
			t.Fatalf("endpoint rule %q severity %q is not on the ramp", r.Name(), r.Severity())
		}
	}
	for _, r := range AllServiceRules() {
		if !inRamp(r.Severity()) {
			t.Fatalf("service rule %q severity %q is not on the ramp", r.Name(), r.Severity())
		}
	}
}

// TestSeverityForResolvesEveryRuleName checks the web layer's read: SeverityFor
// finds every shipped rule by name and returns its rule severity, and reports a
// stale name as unknown (folding to the calmest level, never a panic).
func TestSeverityForResolvesEveryRuleName(t *testing.T) {
	for _, name := range AllRuleNames() {
		sev, ok := SeverityFor(name)
		if !ok {
			t.Fatalf("SeverityFor(%q) reported unknown for a shipped rule", name)
		}
		if !inRamp(sev) {
			t.Fatalf("SeverityFor(%q) = %q, off the ramp", name, sev)
		}
	}
	if sev, ok := SeverityFor("no-such-rule"); ok || sev != SevInfo {
		t.Fatalf("SeverityFor(unknown) = (%q, %v), want (info, false)", sev, ok)
	}
}

// TestSevOrderMatchesSpec pins the ramp order to SignalData.jsx's SEV_ORDER
// (critical→info) so a severity-sorted view ranks exactly as the design does.
func TestSevOrderMatchesSpec(t *testing.T) {
	want := []Severity{SevCritical, SevHigh, SevMedium, SevLow, SevInfo}
	if len(SevOrder) != len(want) {
		t.Fatalf("SevOrder has %d levels, want %d", len(SevOrder), len(want))
	}
	for i, sv := range want {
		if SevOrder[i] != sv {
			t.Fatalf("SevOrder[%d] = %q, want %q", i, SevOrder[i], sv)
		}
		if sv.Rank() != i {
			t.Fatalf("%q.Rank() = %d, want %d", sv, sv.Rank(), i)
		}
	}
}
