package message

import (
	"fmt"
	"strings"
	"time"
)

// One edge, one message, cut per rule and never per facet (ADR-0026 §5).

func SignalEdge(subjectKind, subjectKey, rule string, moved []string, instant time.Time) *Message {
	return &Message{
		Cause:       CauseDrift,
		Class:       ClassForCause(CauseDrift),
		SubjectKind: subjectKind,
		FiredAt:     subjectKey,
		Instant:     instant,
		Headline:    signalEdgeHeadline(subjectKey, rule, moved),
	}
}

func signalEdgeHeadline(subjectKey, rule string, moved []string) string {
	return fmt.Sprintf("%s · %s moved · %s now fired", subjectKey, strings.Join(moved, ", "), rule)
}
