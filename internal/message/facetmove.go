package message

import (
	"fmt"
	"strings"
	"time"
)

// Fires at the move and never at the rule: one cause, one message (ADR-0033 §3).

func FacetMove(subjectKind, subjectKey string, facets []string, census Census, instant time.Time) *Message {
	// An opening with no rule beneath it is recorded, unnamed and unalerted (ADR-0033 §4).
	if len(census.Rules()) == 0 {
		return nil
	}
	c := census
	return &Message{
		Cause:       CauseDrift,
		Class:       ClassDrift,
		SubjectKind: subjectKind,
		FiredAt:     subjectKey,
		Instant:     instant,
		Census:      &c,
		Headline:    facetMoveHeadline(subjectKey, facets, census),
	}
}

func facetMoveHeadline(subjectKey string, facets []string, census Census) string {
	return fmt.Sprintf("%s · %s moved%s", subjectKey, strings.Join(facets, ", "), rulesOpenedClause(census))
}
