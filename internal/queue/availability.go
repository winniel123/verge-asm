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
	// A port-probe batch says nothing of resolver health, so it must not clear an outage (ADR-0108).
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

func applyAvailability(ctx context.Context, qtx *db.Queries, vantageID pgtype.Int8, kind, outcome string) error {
	switch availabilityAfterOutcome(vantageID.Valid, kind, outcome) {
	case availabilityAvailable:
		return qtx.MarkVantageAvailable(ctx, vantageID.Int64)
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
