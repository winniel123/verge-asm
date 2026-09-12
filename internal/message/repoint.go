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

// One address carries many Names, so a move counts on the one it now cites (ADR-1809 §4).

func rePointHeadline(nameKey string, census Census) string {
	// "within" against membership's "entered" keeps the pair distinct (ADR-0026 §2).
	return fmt.Sprintf("%s re-pointed within the estate · %s%s opened on an address it now cites",
		nameKey, factorsClause(kindCountFactors(census)),
		plural(census.Len(), "timeline", "timelines"))
}
