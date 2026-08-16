package signal

import "testing"

// The engine is split by subject kind — a rule reads exactly one — so every
// shipped rule name must resolve to its kind, and an unknown name to "". The
// Signals web layer relies on this to build route-aware drill-down links (#248).
func TestSubjectKindForEveryRule(t *testing.T) {
	for _, r := range All() {
		if got := SubjectKindFor(r.Name()); got != "name" {
			t.Errorf("SubjectKindFor(%q) = %q, want name", r.Name(), got)
		}
	}
	for _, r := range AllEndpointRules() {
		if got := SubjectKindFor(r.Name()); got != "endpoint" {
			t.Errorf("SubjectKindFor(%q) = %q, want endpoint", r.Name(), got)
		}
	}
	for _, r := range AllServiceRules() {
		if got := SubjectKindFor(r.Name()); got != "service" {
			t.Errorf("SubjectKindFor(%q) = %q, want service", r.Name(), got)
		}
	}
	if got := SubjectKindFor("not-a-rule"); got != "" {
		t.Errorf("SubjectKindFor(unknown) = %q, want empty", got)
	}
}
