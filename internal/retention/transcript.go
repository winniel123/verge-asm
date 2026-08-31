package retention

// Transcript retention is the plain-wall-clock rule of the raw-job-output spec §4
// (ADR-0126, amending ADR-0041), landed beside the Dispatch and Observation paths
// in this same package on its own structurally-separate discipline. Like Dispatch,
// a Transcript is retired by age alone: no derivation may read it (the spec §1
// fence), so a clock deleting a captured row by captured_at moves no value on any
// timeline. There is therefore no two-tier boundary and no coverage-style floor —
// nothing pins a minimum window the way the tightest covering Scan pins the
// observation floor.
//
// The one difference from the other two dials is the DEFAULT. Dispatch and
// Observation ship UNBOUNDED (dial 0, the sweep a no-op until the operator sets
// it). The transcript dial SHIPS BOUNDED at 14 days
// (retention_settings.transcript_currency_days, migration 23700): verbatim bytes
// are the volume problem on the address-scope installs that motivated retention,
// so the non-zero default IS the ADR-0041 reversal. 0 is still the explicit
// operator opt-out (unbounded), and a positive value below one day is floored up
// to one day — the tightest window the sweep honours.
//
// Everything here is a pure function plus a thin Retirer over a delete-only Store,
// so the floor and the unbounded sentinel are provable without a database
// (transcript_test.go), and the row-level decision is the same one the SQL
// deletion query encodes (db/queries/retention.sql, DeleteExpiredTranscripts).

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// TranscriptFloorDays is the least positive value the transcript dial may take. A
// positive dial below it is floored UP to it: one day is the tightest window the
// raw-output sweep honours (spec §4, #841). 0 is the unbounded opt-out and is
// never floored. Unlike the observation floor, this floor is a fixed constant, not
// a value derived from any Scan's cadence — no derivation reads a Transcript, so
// nothing pins a minimum window.
const TranscriptFloorDays int64 = 1

// TranscriptWindowDays applies the floor to the operator's day-stated dial. 0 stays
// unbounded (bounded false — the sweep retires nothing); any positive value below
// TranscriptFloorDays is raised to it; every other positive value passes through.
// It never returns a bounded window shorter than the floor.
func TranscriptWindowDays(dialDays int64) (days int64, bounded bool) {
	if dialDays <= 0 {
		return 0, false
	}
	if dialDays < TranscriptFloorDays {
		return TranscriptFloorDays, true
	}
	return dialDays, true
}

// TranscriptCutoff is the instant before which captured transcripts are retired:
// now minus the floored window. bounded is false when the dial is 0 — the explicit
// operator opt-out — and the caller must then retire nothing. The window is the
// floored day count from TranscriptWindowDays, counted in SecondsPerDay.
func TranscriptCutoff(now time.Time, dialDays int64) (cutoff time.Time, bounded bool) {
	days, bounded := TranscriptWindowDays(dialDays)
	if !bounded {
		return time.Time{}, false
	}
	window := time.Duration(days*SecondsPerDay) * time.Second
	return now.Add(-window), true
}

// TranscriptStore is the narrow slice of the data layer the TranscriptRetirer
// needs: the operator's dial and the transcript-only delete. It deliberately
// exposes no read of measured data and no write other than the retiring delete —
// the Retirer can only read the dial and delete expired transcripts, never move a
// value. *db.Queries satisfies it.
type TranscriptStore interface {
	GetRetentionSettings(ctx context.Context) (db.GetRetentionSettingsRow, error)
	DeleteExpiredTranscripts(ctx context.Context, before pgtype.Timestamptz) (int64, error)
}

// TranscriptRetirer sweeps captured transcripts older than the operator's window.
// It is landed beside the Dispatch and Observation Retirers, not folded into them:
// the delete reaches only the transcript table, so a bug here can retire
// transcript rows and only transcript rows.
type TranscriptRetirer struct {
	store TranscriptStore
	now   func() time.Time
	log   *log.Logger
}

// NewTranscriptRetirer builds a TranscriptRetirer over store. now is injectable so
// tests and manual runs can control the sweep instant.
func NewTranscriptRetirer(store TranscriptStore, now func() time.Time, logger *log.Logger) *TranscriptRetirer {
	if now == nil {
		now = time.Now
	}
	return &TranscriptRetirer{store: store, now: now, log: logger}
}

// Sweep retires every transcript captured before the floored window and returns how
// many it deleted. When the dial is 0 it deletes nothing and returns 0 — the
// explicit operator opt-out. Unlike the other two dials this sweep is ACTIVE on a
// fresh install: the dial ships bounded at 14 days (migration 23700).
func (r *TranscriptRetirer) Sweep(ctx context.Context) (int64, error) {
	settings, err := r.store.GetRetentionSettings(ctx)
	if err != nil {
		return 0, err
	}
	cutoff, bounded := TranscriptCutoff(r.now().UTC(), settings.TranscriptCurrencyDays)
	if !bounded {
		return 0, nil
	}
	return r.store.DeleteExpiredTranscripts(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true})
}

// Run sweeps once at start and then every interval until ctx is done. Like the
// Dispatch and Observation Retirers it never returns an error of its own beyond
// the context's: a failed sweep is logged and the loop continues, since a
// transient delete failure is retried on the next tick and transcript retirement
// is never on the measurement path.
func (r *TranscriptRetirer) Run(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	r.sweepAndLog(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			r.sweepAndLog(ctx)
		}
	}
}

func (r *TranscriptRetirer) sweepAndLog(ctx context.Context) {
	n, err := r.Sweep(ctx)
	if err != nil {
		if r.log != nil {
			r.log.Printf("retention: transcript sweep: %v", err)
		}
		return
	}
	if n > 0 && r.log != nil {
		r.log.Printf("retention: retired %d expired transcript row(s)", n)
	}
}
