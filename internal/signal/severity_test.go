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

func TestEveryRuleHasARampSeverity(t *testing.T) {
	// An off-ramp value is one the SeverityBadge could not key a class off (P0.1).
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

func TestSevOrderMatchesSpec(t *testing.T) {
	// Pinned to SignalData.jsx's SEV_ORDER so a severity-sorted view ranks as the design does.
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
