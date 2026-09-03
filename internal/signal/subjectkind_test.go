package signal

import "testing"

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
