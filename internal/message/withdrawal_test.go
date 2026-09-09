package message

import (
	"strings"
	"testing"
)

func TestWithdrawalIsADriftMessageAtTheSubject(t *testing.T) {
	msg := Withdrawal("name", "gone.example.com", 2, t0)
	if msg == nil {
		t.Fatal("a measured-absent withdrawal writes a message")
	}
	if msg.Cause != CauseDrift || msg.Class != ClassDrift {
		t.Errorf("the world withdrew it: cause=%q class=%q, want drift/drift (ADR-0026 mints nothing)", msg.Cause, msg.Class)
	}
	if msg.SubjectKind != "name" || msg.FiredAt != "gone.example.com" {
		t.Errorf("a withdrawal fires at the withdrawn subject, got %q/%q", msg.SubjectKind, msg.FiredAt)
	}
	if msg.LinkKind() != LinkObject {
		t.Errorf("a withdrawal links to the object, got %q", msg.LinkKind())
	}
	if !strings.Contains(msg.Headline, "gone.example.com") || !strings.Contains(msg.Headline, "2 timelines") {
		t.Errorf("the headline states the subject and its timeline count, got %q", msg.Headline)
	}
	if ContainsValence(msg.Headline) {
		t.Errorf("the headline carries a valence word: %q", msg.Headline)
	}
	if msg.Census != nil {
		t.Errorf("a withdrawal carries a count, not a census, got %+v", msg.Census)
	}
}

func TestWithdrawalOnlyAtAMembershipRoot(t *testing.T) {
	for _, kind := range []string{"service", "endpoint", "seed", ""} {
		if Withdrawal(kind, "k", 1, t0) != nil {
			t.Errorf("a %q is no membership root and writes no withdrawal", kind)
		}
	}
	one := Withdrawal("address", "198.51.100.7", 1, t0)
	if one == nil || !strings.Contains(one.Headline, "1 timeline ") {
		t.Errorf("an address withdrawal fires and counts one timeline, got %+v", one)
	}
}
