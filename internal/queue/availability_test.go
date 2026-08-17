package queue

import (
	"testing"

	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

// A terminal resolution-walk Batch outcome derives its vantage's Availability: a
// completed Batch restores it, a dead-lettered one opens it, and a Batch with no
// vantage (the worker-read zone/ct Scans) moves nothing (ADR-0108).
func TestAvailabilityAfterOutcome(t *testing.T) {
	const dns = resolutionwalk.Kind
	cases := []struct {
		name         string
		vantageValid bool
		kind         string
		outcome      string
		want         availabilityAction
	}{
		{"completed dns batch at a vantage restores available", true, dns, outcomeCompleted, availabilityAvailable},
		{"dead-lettered dns batch at a vantage opens unavailable", true, dns, outcomeDeadLettered, availabilityUnavailable},
		{"completed with no vantage (zone/ct) moves nothing", false, dns, outcomeCompleted, availabilityUnchanged},
		{"dead-lettered with no vantage moves nothing", false, dns, outcomeDeadLettered, availabilityUnchanged},
		{"an unknown outcome moves nothing", true, dns, "retried", availabilityUnchanged},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := availabilityAfterOutcome(tc.vantageValid, tc.kind, tc.outcome); got != tc.want {
				t.Errorf("availabilityAfterOutcome(%v, %q, %q) = %d, want %d", tc.vantageValid, tc.kind, tc.outcome, got, tc.want)
			}
		})
	}
}

// The clobber guard: only a resolution-walk (dns) batch may move Availability. A
// completing port-probe (connect-outcome/tls-acceptance) batch at the same
// vantage must NOT re-mark it available, or it would erase the unavailable a
// dead-lettered dns batch set and silently re-mask the resolver outage #249
// exists to surface (ADR-0108).
func TestNonResolutionBatchDoesNotMoveAvailability(t *testing.T) {
	for _, kind := range []string{"connect-outcome", "tls-acceptance"} {
		for _, outcome := range []string{outcomeCompleted, outcomeDeadLettered} {
			if got := availabilityAfterOutcome(true, kind, outcome); got != availabilityUnchanged {
				t.Errorf("a %s batch (%s) moved Availability (%d); only a resolution-walk batch may", kind, outcome, got)
			}
		}
	}
}
