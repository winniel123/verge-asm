package message

import (
	"fmt"
	"time"
)

// Dark declared space has no entering root, so revealed fires at the Seed (ADR-0047, ADR-0052).

func ScopeRevealed(scope string, census Census, instant time.Time) *Message {
	if scope == "" || census.Len() == 0 {
		return nil
	}
	c := census
	return &Message{
		Cause:       CauseAperture,
		Class:       ClassForCause(CauseAperture),
		SubjectKind: "seed",
		FiredAt:     scope,
		Instant:     instant,
		Census:      &c,
		Headline:    scopeRevealedHeadline(scope, c),
	}
}

func scopeRevealedHeadline(scope string, census Census) string {
	// One count at the scope, never a line per address (ADR-0047 Alternatives rejected).
	return fmt.Sprintf("%s · declared address scope · %s%s opened on ground no name cites",
		scope, factorsClause(kindCountFactors(census)), plural(census.Len(), "timeline", "timelines"))
}
