package estate

import "sort"

// This file is the one home of the cross-class withdrawal composition — the
// decision that a Name has left the estate. The drift engine (internal/drift)
// closes a withdrawn subject's timelines and names the closure `measured-absent`,
// but it never re-derives *whether* the subject left: it reads the answer from
// here, so there is one membership computation rather than two (ADR-0080,
// ADR-0006). Membership above composes the estate census; WithdrawnCrossClass is
// the per-Name predicate underneath it, exported so drift shares it.

// ClassWitness is one Vantage class's contribution to a Name's withdrawal
// composition: the latest composed resolution outcome at each *available* vantage
// of that class. An unavailable vantage contributes no outcome — a vantage that
// did not ask is not a vantage that got nothing — so an empty Outcomes slice
// means the class currently holds no value.
type ClassWitness struct {
	Class    string
	Outcomes []string
}

// suppresses reports whether a composed resolution outcome suppresses a Name's
// membership — it neither admits the Name nor is a not-evaluable Gap. NameError
// (the name does not exist) and Shadowed (a wildcard-synthesised answer,
// indistinguishable from a fiction) both suppress the affected Name "as
// affirmatively as" each other (#192 AC; ADR-0086 — membership composes every
// leaf that decides the value it reads). Resolved / NoData / Lame admit the Name;
// Gap is not-evaluable and blocks withdrawal.
func suppresses(outcome string) bool {
	return outcome == OutcomeNameError || outcome == OutcomeShadowed
}

// WithdrawnCrossClass reports whether a Name is absent from the estate. A Name
// leaves only where every available Vantage class suppresses it, composed as a
// cross-class Vantage composition (ADR-0080):
//
//   - Every class the install runs must hold a current value and they must agree
//     that the Name is suppressed — every available vantage reads NameError or
//     Shadowed. A class with no available vantage leaves the comparison unmade —
//     a missing term, never a vacuous pass — so the Name does not withdraw (this
//     is the guard against withdrawing every Name the night every vantage goes
//     unavailable).
//   - Within a class an absence is unanimous: every available vantage of the
//     class must suppress. One vantage of the class still admitting the Name
//     (Resolved / NoData / Lame) keeps the class present, because presence
//     composes existentially; a Gap likewise blocks, being not-evaluable.
//
// Over an empty set of classes the Name does not withdraw — never survivor-only
// over an empty set, and never a clock. A subject leaves by measurement or not
// at all (ADR-0006).
func WithdrawnCrossClass(classes []ClassWitness) bool {
	if len(classes) == 0 {
		return false
	}
	for _, c := range classes {
		if len(c.Outcomes) == 0 {
			// The class holds no current value: the cross-class comparison cannot be
			// made, so it is not-evaluable and the Name does not withdraw.
			return false
		}
		for _, o := range c.Outcomes {
			if !suppresses(o) {
				// Some available vantage of this class admits the Name (Resolved /
				// NoData / Lame) or is not-evaluable (Gap): the class does not
				// conclude absence, so neither does the composition.
				return false
			}
		}
	}
	// Every class held a value and every one concluded the Name is suppressed.
	return true
}

// witnessesByClass groups a Name's per-vantage composed outcomes into one
// ClassWitness per class, so Membership and WithdrawnCrossClass share the same
// composition. An empty class label is one default class — the modal
// single-class install — so the cross-class rule collapses to the single-class
// case with no special path.
func witnessesByClass(outcomes []classedOutcome) []ClassWitness {
	byClass := map[string][]string{}
	order := []string{}
	for _, co := range outcomes {
		if _, seen := byClass[co.class]; !seen {
			order = append(order, co.class)
		}
		byClass[co.class] = append(byClass[co.class], co.outcome)
	}
	sort.Strings(order)
	out := make([]ClassWitness, 0, len(order))
	for _, cls := range order {
		out = append(out, ClassWitness{Class: cls, Outcomes: byClass[cls]})
	}
	return out
}

type classedOutcome struct {
	class   string
	outcome string
}
