// Package report holds the on-cadence report machinery: the cadence→window
// vocabulary a schedule dispatches on, and the Dispatcher that renders each due
// schedule's artifact and stamps its in-instance receipt (#502/T3). It is the
// report-side twin of package queue's measurement dispatch, and shares queue's
// idempotency shape — a floored tick keys one run of a window, and a second poll
// inside the window is a recorded skip rather than a second run.
package report

import (
	"strings"
	"time"
)

// CadenceWindow is the period a run of a schedule covers, derived from the
// schedule's stored cadence label: the run cuts the artifact for the window the
// cadence implies (6h / daily / weekly / monthly), defaulting to a week for an
// unrecognised label. It is the single source of truth for that mapping — both the
// Run-now handler (cmd/web) and the on-cadence Dispatcher compute the same window
// from this one function, so a scheduled run and a manual run of the same schedule
// cover the same period.
//
// The vocabulary is a closed, model-owned set: a custom / cron cadence is NOT
// evaluated as an operator-authored predicate (ADR-0091) — it simply falls to the
// weekly window, exactly as Run-now already treats it. The dispatcher dispatches on
// this window and never interprets a cron expression, so no versioned-rule-set
// predicate is ever evaluated (ADR-0117).
func CadenceWindow(cadence string) time.Duration {
	c := strings.ToLower(cadence)
	switch {
	case strings.Contains(c, "6h"):
		return 6 * time.Hour
	case strings.Contains(c, "daily"):
		return 24 * time.Hour
	case strings.Contains(c, "monthly"):
		return 30 * 24 * time.Hour
	default:
		return 7 * 24 * time.Hour
	}
}

// scheduledTick floors now to the cadence boundary, so two dispatcher ticks inside
// one cadence window resolve to the same (schedule, scheduled_tick) key and the
// second conflicts rather than dispatching again. Missed ticks are not caught up —
// the boundary is always the current window, never a past one. Copied verbatim from
// package queue's scheduledTick so the report side floors identically to the
// measurement side.
func scheduledTick(now time.Time, cadence time.Duration) time.Time {
	if cadence <= 0 {
		return now.UTC().Truncate(time.Second)
	}
	secs := int64(cadence / time.Second)
	if secs <= 0 {
		secs = 1
	}
	floored := (now.UTC().Unix() / secs) * secs
	return time.Unix(floored, 0).UTC()
}
