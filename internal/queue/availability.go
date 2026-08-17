package queue

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// availabilityAction is the Vantage Availability mutation a terminal Batch
// outcome implies for the vantage that produced it (ADR-0108). Availability is
// concluded from recent batch outcomes rather than measured directly: a
// completed Batch is proof the position can observe, a dead-lettered one is
// proof it could not.
type availabilityAction int

const (
	// availabilityUnchanged: derive nothing. The Batch carried no real vantage —
	// the worker-read zone and ct Scans have none — or the outcome is not
	// terminal.
	availabilityUnchanged availabilityAction = iota
	availabilityAvailable
	availabilityUnavailable
)

// availabilityAfterOutcome maps a terminal Batch outcome to the Availability it
// implies. It derives nothing where the Batch carries no vantage (zone/ct), so
// a null-vantage outcome never moves an availability that does not exist. The
// window Availability is "concluded from recent batch outcomes over" is, in v1,
// the retry sequence within one dispatch: a dead-letter already means every
// attempt across that window failed, and a completed Batch is immediate proof of
// recovery (ADR-0108).
func availabilityAfterOutcome(vantageValid bool, outcome string) availabilityAction {
	if !vantageValid {
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

// applyAvailability derives and applies the vantage's Availability from a
// terminal batch outcome, inside the same transaction that wrote the Batch.
func applyAvailability(ctx context.Context, qtx *db.Queries, vantageID pgtype.Int8, outcome string) error {
	switch availabilityAfterOutcome(vantageID.Valid, outcome) {
	case availabilityAvailable:
		return qtx.MarkVantageAvailable(ctx, vantageID.Int64)
	case availabilityUnavailable:
		return qtx.MarkVantageUnavailable(ctx, vantageID.Int64)
	default:
		return nil
	}
}

// The Batch outcome vocabulary (db/migrations/18803_measurement_batch.sql), named
// once so the worker's writes and this derivation cannot drift apart.
const (
	outcomeCompleted    = "completed"
	outcomeDeadLettered = "dead-lettered"
)
