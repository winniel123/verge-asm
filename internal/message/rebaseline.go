package message

import (
	"fmt"
	"strings"
	"time"
)

// A re-baseline and an aperture widening are one class: both say we changed how we look (ADR-0014).

func Rebaseline(derivation string, movedLeaves []string, membershipLeaf bool, instant time.Time) *Message {
	// An empty difference set is a fact about values, not a licence to compare them (ADR-0008).
	if len(movedLeaves) == 0 {
		return nil
	}
	entries := make([]CensusEntry, 0, len(movedLeaves))
	for _, leaf := range movedLeaves {
		entries = append(entries, CensusEntry{Kind: "leaf", Key: leaf})
	}
	c := NewCensus(entries...)
	return &Message{
		Cause:       CauseAperture,
		Class:       ClassForCause(CauseAperture),
		SubjectKind: "derivation",
		FiredAt:     derivation,
		Instant:     instant,
		Census:      &c,
		Headline:    rebaselineHeadline(derivation, movedLeaves, membershipLeaf),
	}
}

func rebaselineHeadline(derivation string, movedLeaves []string, membershipLeaf bool) string {
	// A break forbids the comparison, so the copy states the horizon and grades nothing (ADR-0008).
	head := fmt.Sprintf("%s derivation moved · %s moved: %s · earlier values are not compared",
		derivation, plural(len(movedLeaves), "leaf", "leaves"), strings.Join(movedLeaves, ", "))
	if !membershipLeaf {
		return head
	}
	// History is never re-derived, so the loss of returned is stated at the cause (ADR-0041).
	return head + " · a return reads appeared until a subject withdraws and returns under this vector"
}
