package queue

import (
	"slices"
	"testing"

	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
)

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

func TestNonResolutionBatchDoesNotMoveAvailability(t *testing.T) {
	for _, kind := range []string{"connect-outcome", "tls-acceptance"} {
		for _, outcome := range []string{outcomeCompleted, outcomeDeadLettered} {
			if got := availabilityAfterOutcome(true, kind, outcome); got != availabilityUnchanged {
				t.Errorf("a %s batch (%s) moved Availability (%d); only a resolution-walk batch may", kind, outcome, got)
			}
		}
	}
}

func TestRecoveryRetiresOnlyTheFacetsTheBatchReMeasured(t *testing.T) {
	// A dead-lettered walk opens a Gap on every facet the vantage fed, but a completed walk
	// re-reads only its own two. Closing the rest retires a reachability Gap with nothing to
	// replace it, and the leg then reads never-configured (ADR-2087, #2060).
	got := recoveredFacets(resolutionwalk.Kind)
	want := []string{resolutionwalk.FacetResolution, resolutionwalk.FacetDNSRecord}
	if !slices.Equal(got, want) {
		t.Fatalf("recoveredFacets(%q) = %v, want %v: a resolution-walk batch emits these two "+
			"facets and no other", resolutionwalk.Kind, got, want)
	}
	for _, facet := range []string{
		connectoutcome.FacetReachability,
		connectoutcome.FacetCertificate,
		tlsacceptance.Facet,
		httpexchange.FacetHTTPIdentity,
	} {
		if slices.Contains(got, facet) {
			t.Errorf("recovery retires the %s Gap, which no walk re-measures, so the span leaves "+
				"the open-span read and the leg reads never-configured (ADR-2087)", facet)
		}
	}
}

func TestEveryKindThatRestoresAvailabilityNamesItsFacets(t *testing.T) {
	// Two switches read the same kind. A kind that cleared an outage but named no facet would
	// move the column and leave every Gap it opened standing.
	for _, kind := range []string{
		resolutionwalk.Kind, connectoutcome.Kind, tlsacceptance.Kind, httpexchange.Kind,
	} {
		restores := availabilityAfterOutcome(true, kind, outcomeCompleted) == availabilityAvailable
		if restores != (len(recoveredFacets(kind)) > 0) {
			t.Errorf("kind %q restores availability = %v but names %d facets", kind,
				restores, len(recoveredFacets(kind)))
		}
	}
}
