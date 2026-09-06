package queue

import (
	"context"
	"fmt"
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

type DispatchGateStore interface {
	HotLagStore
	SetDispatchStatus(ctx context.Context, arg db.SetDispatchStatusParams) error
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

func gateHotTick(ctx context.Context, q DispatchGateStore, kind string, scanID, dispatchID int64, staleJobThreshold time.Duration, logger *log.Logger) (SkipReason, error) {
	if !hotLagGateApplies(kind) {
		return SkipNone, nil
	}
	lagging, err := hotTickLags(ctx, q, scanID, dispatchID, staleJobThreshold, logger)
	if err != nil {
		return SkipNone, fmt.Errorf("queue: hot cadence-lag gate: %w", err)
	}
	if !lagging {
		return SkipNone, nil
	}
	// The claimed row would otherwise read as a run that fanned nothing out (#1120).
	arg := db.SetDispatchStatusParams{ID: dispatchID, Status: DispatchStatusSkipped}
	if err := q.SetDispatchStatus(ctx, arg); err != nil {
		return SkipNone, fmt.Errorf("queue: record cadence-lag skip: %w", err)
	}
	return SkipCadenceLag, nil
}

var _ DispatchGateStore = (*db.Queries)(nil)
