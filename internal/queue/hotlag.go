package queue

// The hot cadence-lag gate (#1114, decided by ADR-0137 §4).
//
// Dispatch idempotency is keyed on (scan, scheduled_time), which admits one dispatch
// per tick window. Nothing gates a new tick on the PREVIOUS dispatch draining. The
// hot Scan fans out one job per (Vantage, Address) pair, which within one dispatch
// carries the per-target safety ceiling at any worker count (ADR-0137 §3). Across two
// dispatches it carries nothing: where a hot scan does not finish inside its cadence,
// the next tick enqueues a fresh job for every pair while the first dispatch's jobs
// are still 'ready' or 'running', two workers run the same pair concurrently, and the
// target receives up to twice the declared rate.
//
// The gate skips such a tick and records it. Skipping, not deferring: a deferred tick
// would carry a scheduled_time it did not run at, and a recorded skip makes cadence
// lag visible — the signal telling the operator that the raised address-scope cap has
// a cost.
//
// TWO BOUNDS, BOTH LOAD-BEARING.
//
// hot only. cold and edge-fanout fan out streamed too and lag the same way, but only
// hot connects to a target and this gate exists for a safety property. Extending it to
// the other two would be a scheduling change under a safety ticket.
//
// It does not arm when the stale-'running' reaper is disabled. A non-positive stale-job
// timeout switches the reaper off (reaper.go, StaleCutoff). With it off, a job wedged in
// 'running' never terminates, so the gate would skip EVERY future hot tick forever on one
// stuck row — turning an operator's tuning choice into a silent stop of all active
// measurement. A safety mechanism that can permanently stop measurement is worse than the
// doubling it prevents, and its failure would surface only as an absence. In that
// configuration the gate logs a warning and falls through to the pre-#1114 behaviour.

import (
	"context"
	"log"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
)

// hotLagGateApplies is the tier bound, held in one place so it is provable. Only the
// hot Scan connects to a target, so only it carries the per-target safety ceiling the
// gate protects. cold and edge-fanout fan out streamed too and lag the same way; their
// dispatch behaviour must stay exactly as it was.
func hotLagGateApplies(kind string) bool {
	return kind == scan.HotKind
}

// HotLagStore is the whole surface the gate can reach: the one read asking whether a
// Scan holds a non-terminal job from another dispatch. It exposes no write and no read
// of measured data, so a bug here can only decide a skip. *db.Queries satisfies it.
type HotLagStore interface {
	ScanHasNonTerminalJobs(ctx context.Context, arg db.ScanHasNonTerminalJobsParams) (bool, error)
}

// HotLagGateArmed reports whether the gate may hold a tick, given the worker's
// stale-job timeout. It mirrors StaleCutoff's rule exactly — a non-positive threshold
// disables the reaper — because the gate is only safe while something can terminate a
// wedged 'running' job. Pure, so the arming boundary is provable without a database.
func HotLagGateArmed(staleJobThreshold time.Duration) bool {
	_, bounded := StaleCutoff(time.Time{}, staleJobThreshold)
	return bounded
}

// hotTickLags answers whether this hot tick must be skipped because an earlier dispatch
// of the same Scan has not drained. It returns false — dispatch normally — whenever the
// gate is not armed, after logging the warning that says why the protection is off.
func hotTickLags(ctx context.Context, q HotLagStore, scanID, dispatchID int64, staleJobThreshold time.Duration, logger *log.Logger) (bool, error) {
	if !HotLagGateArmed(staleJobThreshold) {
		if logger != nil {
			logger.Printf("dispatcher: the stale-running reaper is disabled (stale job timeout %s), so the hot cadence-lag gate is not armed; "+
				"a hot tick that overtakes an undrained dispatch can double the rate at one target", staleJobThreshold)
		}
		return false, nil
	}
	return q.ScanHasNonTerminalJobs(ctx, db.ScanHasNonTerminalJobsParams{ScanID: scanID, DispatchID: dispatchID})
}

// compile-time proof *db.Queries is a HotLagStore.
var _ HotLagStore = (*db.Queries)(nil)
