package queue

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

// availabilityAction is the Vantage Availability mutation a terminal Batch
// outcome implies for the vantage that produced it (ADR-0108). Availability is
// concluded from recent batch outcomes rather than measured directly: a
// completed Batch is proof the position can observe, a dead-lettered one is
// proof it could not.
type availabilityAction int

const (
	availabilityUnchanged availabilityAction = iota
	availabilityAvailable
	availabilityUnavailable
)

// availabilityAfterOutcome maps a terminal Batch outcome to the Availability it
// implies. Two guards keep it honest:
//
//   - It derives nothing where the Batch carries no vantage (zone/ct), so a
//     null-vantage outcome never moves an availability that does not exist.
//   - It derives only from a **resolution-walk** (dns) Batch — the one that
//     exercises the vantage's recursive resolver, and the only capability a
//     resolver outage impairs. A completing port-probe (hot/cold/tls,
//     `connect-outcome` / `tls-acceptance`) Batch at the same vantage says
//     nothing about resolver health, so it must not re-mark the vantage
//     `available` and clobber the `unavailable` a dead-lettered dns Batch set —
//     which would silently re-mask the very resolver outage this exists to
//     surface (ADR-0108). Availability is a single scalar per vantage, so the
//     capability that may move it is scoped rather than shared.
//
// The window Availability is "concluded from recent batch outcomes over" is, in
// v1, the retry sequence within one dispatch: a dead-letter already means every
// attempt across that window failed, and a completed Batch is immediate proof of
// recovery.
func availabilityAfterOutcome(vantageValid bool, kind, outcome string) availabilityAction {
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

// The Batch outcome vocabulary (db/migrations/18803_measurement_batch.sql), named
// once so the worker's writes and this derivation cannot drift apart.
const (
	outcomeCompleted    = "completed"
	outcomeDeadLettered = "dead-lettered"
)
