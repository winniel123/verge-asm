package queue

import "testing"

// A terminal Batch outcome derives its vantage's Availability: a completed Batch
// restores it, a dead-lettered one opens it, and a Batch with no vantage (the
// worker-read zone/ct Scans) moves nothing (ADR-0108).
func TestAvailabilityAfterOutcome(t *testing.T) {
	cases := []struct {
		name         string
		vantageValid bool
		outcome      string
		want         availabilityAction
	}{
		{"completed at a vantage restores available", true, outcomeCompleted, availabilityAvailable},
		{"dead-lettered at a vantage opens unavailable", true, outcomeDeadLettered, availabilityUnavailable},
		{"completed with no vantage (zone/ct) moves nothing", false, outcomeCompleted, availabilityUnchanged},
		{"dead-lettered with no vantage moves nothing", false, outcomeDeadLettered, availabilityUnchanged},
		{"an unknown outcome moves nothing", true, "retried", availabilityUnchanged},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := availabilityAfterOutcome(tc.vantageValid, tc.outcome); got != tc.want {
				t.Errorf("availabilityAfterOutcome(%v, %q) = %d, want %d", tc.vantageValid, tc.outcome, got, tc.want)
			}
		})
	}
}
