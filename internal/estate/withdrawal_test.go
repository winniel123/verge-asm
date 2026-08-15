package estate

import (
	"testing"

	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

func TestWithdrawnEmptySetNeverWithdraws(t *testing.T) {
	// Never survivor-only over an empty set, never a clock (ADR-0080 rule 3).
	if WithdrawnCrossClass(nil) {
		t.Error("no classes must never withdraw a Name")
	}
}

func TestWithdrawnAllClassesAgreeNameError(t *testing.T) {
	withdrawn := WithdrawnCrossClass([]ClassWitness{
		{Class: "internal", Outcomes: []string{OutcomeNameError}},
		{Class: "internet", Outcomes: []string{OutcomeNameError}},
	})
	if !withdrawn {
		t.Error("a Name reading NameError from every class withdraws")
	}
}

func TestWithdrawnOneClassStillResolvingKeepsName(t *testing.T) {
	// Split horizon: the internet class still resolves, so the classes disagree —
	// incommensurability, not evidence. The Name does not withdraw.
	if WithdrawnCrossClass([]ClassWitness{
		{Class: "internal", Outcomes: []string{OutcomeNameError}},
		{Class: "internet", Outcomes: []string{OutcomeResolved}},
	}) {
		t.Error("disagreement across classes must not withdraw a Name")
	}
}

func TestWithdrawnMissingClassLeavesComparisonUnmade(t *testing.T) {
	// The internet class holds no available vantage: a missing term, not a vacuous
	// pass. The Name does not withdraw even though the class present agrees.
	if WithdrawnCrossClass([]ClassWitness{
		{Class: "internal", Outcomes: []string{OutcomeNameError}},
		{Class: "internet", Outcomes: nil},
	}) {
		t.Error("a class with no available vantage leaves the comparison unmade")
	}
}

func TestWithdrawnWithinClassAbsenceIsUnanimous(t *testing.T) {
	// One vantage of the class still resolving keeps the class present (existential
	// presence), so the Name does not withdraw.
	if WithdrawnCrossClass([]ClassWitness{
		{Class: "internal", Outcomes: []string{OutcomeNameError, OutcomeResolved}},
	}) {
		t.Error("within a class, one resolving vantage keeps the Name present")
	}
	if !WithdrawnCrossClass([]ClassWitness{
		{Class: "internal", Outcomes: []string{OutcomeNameError, OutcomeNameError}},
	}) {
		t.Error("within a class, unanimous NameError withdraws")
	}
}

func TestMembershipCrossClassWithdrawalConvergesWithDrift(t *testing.T) {
	// Membership must read the same cross-class predicate the drift engine reads —
	// one membership computation. A Name reading NameError internally but still
	// resolving from the internet stays present.
	ne := Compose(OutcomeNameError, nil, wd.VerdictNotShadowed)
	ok := Compose(OutcomeResolved, []string{"203.0.113.5"}, wd.VerdictNotShadowed)
	est := Membership([]Observation{
		{Name: "split.example.com", Vantage: "in", Class: "internal", Resolution: ne},
		{Name: "split.example.com", Vantage: "out", Class: "internet", Resolution: ok},
	}, nil, nil)
	if !contains(est.Names, "split.example.com") {
		t.Error("cross-class disagreement keeps the Name present")
	}

	// Both classes agree on NameError: withdrawn.
	est2 := Membership([]Observation{
		{Name: "dead.example.com", Vantage: "in", Class: "internal", Resolution: ne},
		{Name: "dead.example.com", Vantage: "out", Class: "internet", Resolution: ne},
	}, nil, nil)
	if contains(est2.Names, "dead.example.com") {
		t.Error("cross-class agreement on NameError withdraws the Name")
	}
}
