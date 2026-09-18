package queue

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

// Availability is concluded from batch outcomes, never measured directly (ADR-0108).

type availabilityAction int

const (
	availabilityUnchanged availabilityAction = iota
	availabilityAvailable
	availabilityUnavailable
)

func availabilityAfterOutcome(vantageValid bool, kind, outcome string) availabilityAction {
	// A port-probe batch says nothing of resolver health, so it cannot clear an outage (ADR-0108).
	if !vantageValid || kind != resolutionwalk.Kind {
		return availabilityUnchanged
	}
	switch outcome {
	case outcomeCompleted:
		return availabilityAvailable
	case outcomeDeadLettered:
		return availabilityUnavailable
	default:
		return availabilityUnchanged
	}
}

func recoveredFacets(kind string) []string {
	switch kind {
	case resolutionwalk.Kind:
		// The facets resolutionwalk.Emit writes, and the whole of what a completed walk re-reads.
		return []string{resolutionwalk.FacetResolution, resolutionwalk.FacetDNSRecord}
	default:
		return nil
	}
}

// db/queries/vantages.sql spells both, so a rename there must reach this file.

const (
	availabilityUnavailableValue = "unavailable"
	outageGapCause               = "vantage-unavailable"
)

func outageSurvives(kind string) bool {
	// A port probe says nothing of resolver health, so it clears no outage (ADR-0108).
	return availabilityAfterOutcome(true, kind, outcomeCompleted) != availabilityAvailable
}

func availabilityForFold(ctx context.Context, qtx *db.Queries, vantageID pgtype.Int8, kind string) (string, error) {
	// A kind that clears the outage folds whatever the column holds, so the read decides nothing.
	if !vantageID.Valid || !outageSurvives(kind) {
		return "", nil
	}
	v, err := qtx.GetVantage(ctx, vantageID.Int64)
	if err != nil {
		return "", err
	}
	return v.Availability.String, nil
}

func outageStands(availability, kind string) bool {
	return availability == availabilityUnavailableValue && outageSurvives(kind)
}

func applyAvailability(ctx context.Context, qtx *db.Queries, vantageID pgtype.Int8, kind, outcome string) error {
	switch availabilityAfterOutcome(vantageID.Valid, kind, outcome) {
	case availabilityAvailable:
		// The outage closed every facet; recovery retires this batch's own (ADR-2087, #2060).
		return qtx.MarkVantageAvailable(ctx, db.MarkVantageAvailableParams{
			ID:     vantageID.Int64,
			Facets: recoveredFacets(kind),
		})
	case availabilityUnavailable:
		return qtx.MarkVantageUnavailable(ctx, vantageID.Int64)
	default:
		return nil
	}
}

// Named from db/migrations/18803_measurement_batch.sql, so the worker and this fold cannot drift.

const (
	outcomeCompleted    = "completed"
	outcomeDeadLettered = "dead-lettered"
)
