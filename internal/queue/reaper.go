package queue

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// A legitimately mid-probe job must never be reaped, so this clears one probe timeout with room.

const DefaultStaleJobThreshold = 15 * time.Minute

func StaleCutoff(now time.Time, threshold time.Duration) (cutoff time.Time, bounded bool) {
	if threshold <= 0 {
		return time.Time{}, false
	}
	return now.Add(-threshold), true
}

type ReaperStore interface {
	ReapStaleRunningJobs(ctx context.Context, cutoff pgtype.Timestamptz) (int64, error)
}

// The sweep is one guarded UPDATE over the queue, so it runs off the worker's per-job transaction.

type Reaper struct {
	store     ReaperStore
	threshold time.Duration
	now       func() time.Time
	log       *log.Logger
}

func NewReaper(store ReaperStore, threshold time.Duration, now func() time.Time, logger *log.Logger) *Reaper {
	if now == nil {
		now = time.Now
	}
	return &Reaper{store: store, threshold: threshold, now: now, log: logger}
}

func (r *Reaper) Sweep(ctx context.Context) (int64, error) {
	// Nothing else exits 'running', so a dead worker strands its job and Dispatch forever (#853).
	cutoff, bounded := StaleCutoff(r.now().UTC(), r.threshold)
	if !bounded {
		return 0, nil
	}
	return r.store.ReapStaleRunningJobs(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true})
}

func (r *Reaper) Run(ctx context.Context, interval time.Duration) error {
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

func (r *Reaper) sweepAndLog(ctx context.Context) {
	// A failed sweep costs one interval, so the loop logs and continues rather than ending (ADR-0141).
	n, err := r.Sweep(ctx)
	if err != nil {
		if r.log != nil {
			r.log.Printf("queue: reaper sweep: %v", err)
		}
		return
	}
	if n > 0 && r.log != nil {
		r.log.Printf("queue: reclaimed %d stale running job(s)", n)
	}
}

var _ ReaperStore = (*db.Queries)(nil)
