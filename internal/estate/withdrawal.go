package estate

import "sort"

type ClassWitness struct {
	Class    string
	Outcomes []string
}

func suppresses(outcome string) bool {
	// Shadowed suppresses as affirmatively as NameError: a fiction is indistinguishable (#192).
	return outcome == OutcomeNameError || outcome == OutcomeShadowed
}

func WithdrawnCrossClass(classes []ClassWitness) bool {
	// The drift engine reads this answer rather than re-deriving it: one computation (ADR-0080).
	if len(classes) == 0 {
		// A subject leaves by measurement or not at all: never a clock, never survivor-only (ADR-0006).
		return false
	}
	for _, c := range classes {
		// A vantage that did not ask is not a vantage that got nothing, so an empty class decides none.
		// The guard against withdrawing every Name the night every vantage goes unavailable.
		if len(c.Outcomes) == 0 {
			return false
		}
		for _, o := range c.Outcomes {
			if !suppresses(o) {
				return false
			}
		}
	}
	return true
}

func undecided(outcome string) bool {
	// These three are where no Name Error can ever arrive, so measurement never concludes (ADR-0006).
	return outcome == OutcomeShadowed || outcome == OutcomeGap || outcome == OutcomeLame
}

func DecidedAbsentCrossClass(classes []ClassWitness) bool {
	// A Seed admits a Name and holds it only where measurement cannot decide (ADR-0146 §2).
	if !WithdrawnCrossClass(classes) {
		return false
	}
	for _, c := range classes {
		for _, o := range c.Outcomes {
			if undecided(o) {
				return false
			}
		}
	}
	return true
}

func witnessesByClass(outcomes []classedOutcome) []ClassWitness {
	// An empty class label is the one default class of a single-class install (ADR-0080).
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
