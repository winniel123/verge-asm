package message

import (
	"fmt"
	"time"

	"github.com/winniel123/verge-asm/internal/signal"
)

// Every clock-class message states that no measurement moved, as a clause (ADR-0064 §1).

func ClockEdge(subjectKind, subjectKey, rule string, clock signal.CertClock, instant time.Time) *Message {
	return Threshold(subjectKind, subjectKey, clockEdgeHeadline(subjectKey, rule, clock), instant)
}

func clockEdgeHeadline(subjectKey, rule string, clock signal.CertClock) string {
	head := fmt.Sprintf("%s · no measurement moved · %s now fired", subjectKey, rule)
	switch rule {
	case "certificate-expired":
		return head + " · not_after " + clockDate(clock.NotAfter) + " has passed"
	case "certificate-expiring":
		horizon, ok := signal.CertHorizon(clock.NotBefore, clock.NotAfter)
		if !ok {
			return head
		}
		return fmt.Sprintf("%s · not_after %s is inside the %s horizon", head, clockDate(clock.NotAfter), horizonLabel(horizon))
	case "certificate-not-yet-valid":
		return head + " · not_before " + clockDate(clock.NotBefore) + " has not arrived"
	default:
		return head
	}
}

func clockDate(t time.Time) string { return t.UTC().Format("2006-01-02") }

func horizonLabel(d time.Duration) string {
	if d%(24*time.Hour) == 0 {
		return fmt.Sprintf("%d-day", int(d/(24*time.Hour)))
	}
	return fmt.Sprintf("%d-hour", int(d.Round(time.Hour)/time.Hour))
}
