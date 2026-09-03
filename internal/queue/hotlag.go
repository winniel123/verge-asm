package queue

import (
	"context"
	"log"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
)

func hotLagGateApplies(kind string) bool {
	// cold and edge-fanout lag alike, but only hot connects to a target (ADR-0137 §4).
	return kind == scan.HotKind
}

type HotLagStore interface {
	ScanHasNonTerminalJobs(ctx context.Context, arg db.ScanHasNonTerminalJobsParams) (bool, error)
}

func HotLagGateArmed(staleJobThreshold time.Duration) bool {
	// With no reaper one wedged row would skip every hot tick forever (ADR-0137 §4).
	_, bounded := StaleCutoff(time.Time{}, staleJobThreshold)
	return bounded
}

func hotTickLags(ctx context.Context, q HotLagStore, scanID, dispatchID int64, staleJobThreshold time.Duration, logger *log.Logger) (bool, error) {
	// Two dispatches of one Scan run a pair concurrently, doubling a target's rate (ADR-0137 §4).
	if !HotLagGateArmed(staleJobThreshold) {
		if logger != nil {
			logger.Printf("dispatcher: the stale-running reaper is disabled (stale job timeout %s), so the hot cadence-lag gate is not armed; "+
				"a hot tick that overtakes an undrained dispatch can double the rate at one target", staleJobThreshold)
		}
		return false, nil
	}
	return q.ScanHasNonTerminalJobs(ctx, db.ScanHasNonTerminalJobsParams{ScanID: scanID, DispatchID: dispatchID})
}

var _ HotLagStore = (*db.Queries)(nil)
