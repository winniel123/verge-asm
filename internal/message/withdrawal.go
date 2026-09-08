package message

import (
	"fmt"
	"time"
)

// Written and never routed: the store carries it, no channel does (ADR-0087, ADR-0039).

func Withdrawal(subjectKind, subjectKey string, timelines int, instant time.Time) *Message {
	if !RootFires(subjectKind) {
		return nil
	}
	return &Message{
		Cause:       CauseDrift,
		Class:       ClassForCause(CauseDrift),
		SubjectKind: subjectKind,
		FiredAt:     subjectKey,
		Instant:     instant,
		Headline:    withdrawalHeadline(subjectKey, timelines),
	}
}

func withdrawalHeadline(subjectKey string, timelines int) string {
	// A withdrawal is the world's act, so the headline carries no valence (ADR-0064).
	return fmt.Sprintf("%s withdrawn · measured absent · %s taken out of the estate",
		subjectKey, plural(timelines, "timeline", "timelines"))
}
