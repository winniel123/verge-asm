package message

import (
	"fmt"
	"time"
)

func RePoint(nameKey string, census Census, instant time.Time) *Message {
	// An empty residue is no message, so an Address root is never doubled (ADR-0026 §2).
	if census.Len() == 0 {
		return nil
	}
	c := census
	return &Message{
		Cause:       CauseDrift,
		Class:       ClassForCause(CauseDrift),
		SubjectKind: "name",
		FiredAt:     nameKey,
		Instant:     instant,
		Census:      &c,
		Headline:    rePointHeadline(nameKey, census),
	}
}

func rePointHeadline(nameKey string, census Census) string {
	// "within" against membership's "entered" keeps the pair from reading as duplicates (ADR-0026 §2).
	return fmt.Sprintf("%s re-pointed within the estate · %s%s opened beneath it",
		nameKey, factorsClause(kindCountFactors(census)),
		plural(census.Len(), "timeline", "timelines"))
}
