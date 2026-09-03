package queue

// The stale-`running` reaper (#853). A queue job stuck in state 'running' is never
// reclaimed on its own: ClaimJob claims 'ready' rows alone, and the only exits from
// 'running' are the owning worker's guarded terminal writes (MarkJobDone/Dead/
// Retried) or an operator Terminate (CancelActiveJobsForDispatch). So when the
// worker process dies or hangs mid-job, that job stays 'running' forever and its
// Dispatch stays "in flight", clearable only by a manual Terminate.
//
// The reaper closes that gap. It sweeps on a period and returns every 'running' job
// whose lease (queue_job.claimed_at, stamped by ClaimJob) is older than a threshold
// to the claimable set — 'ready' with attempt bumped while attempts remain, or
// 'dead' past the budget. A live worker then re-claims and re-runs the readied job.
// This makes a worker crash recoverable without operator action.
//
// The threshold must sit comfortably above the per-job probe timeout (worker.go), so
// a job that is legitimately mid-probe is never mistaken for a dead one: the probe
// timeout bounds how long ONE claim may run, and the reaper only fires well past
// that. Mirrors the retention Retirers' shape — a pure cutoff, a narrow Store, and a
// Run loop that logs and continues — but reclaims queue work rather than retiring a
// corpus, so it lives beside the worker it serves, not in internal/retention.

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// DefaultStaleJobThreshold is the age past which a 'running' job's lease is treated
// as stale and the job is reclaimed. It sits well above the default probe timeout
// (DefaultProbeTimeout) so a job that is legitimately mid-probe — up to one probe
// timeout of exec plus its terminal commit — is never reaped as dead. The operator
// may override it (cmd/worker, VERGE_STALE_JOB_TIMEOUT).
const DefaultStaleJobThreshold = 15 * time.Minute

// StaleCutoff is the instant before which a 'running' job's lease is stale: now
// minus the threshold. bounded is false when the threshold is not positive — a
// zero or negative threshold disables the reaper, and the caller must then reclaim
// nothing. Pure so the boundary is provable without a database (reaper_test.go), the
// same discipline the retention cutoffs follow.
func StaleCutoff(now time.Time, threshold time.Duration) (cutoff time.Time, bounded bool) {
	if threshold <= 0 {
		return time.Time{}, false
	}
	return now.Add(-threshold), true
}

// ReaperStore is the narrow slice of the data layer the Reaper needs: the single
// reclaiming sweep. It exposes no read of measured data and no other write, so a bug
// here can only reclaim stale 'running' jobs, never move a value on any timeline.
// *db.Queries satisfies it.
type ReaperStore interface {
	ReapStaleRunningJobs(ctx context.Context, cutoff pgtype.Timestamptz) (int64, error)
}

// Reaper reclaims stale 'running' jobs on a period. It is landed beside the worker,
// not folded into it: the sweep is a single guarded UPDATE that reaches only the
// queue, so it can run on its own goroutine without touching the worker's per-job
// transaction.
type Reaper struct {
	store     ReaperStore
	threshold time.Duration
	now       func() time.Time
	log       *log.Logger
}

// NewReaper builds a Reaper over store with the stale-lease threshold. A threshold
// that is not positive disables the sweep (StaleCutoff reports unbounded). now is
// injectable so tests and manual runs can control the sweep instant.
func NewReaper(store ReaperStore, threshold time.Duration, now func() time.Time, logger *log.Logger) *Reaper {
	if now == nil {
		now = time.Now
	}
	return &Reaper{store: store, threshold: threshold, now: now, log: logger}
}

func (r *Reaper) Sweep(ctx context.Context) (int64, error) {
	cutoff, bounded := StaleCutoff(r.now().UTC(), r.threshold)
	if !bounded {
		return 0, nil
	}
	return r.store.ReapStaleRunningJobs(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true})
}

// Run sweeps once at start and then every interval until ctx is done. Like the
// retention Retirers it never returns an error of its own beyond the context's: a
// failed sweep is logged and the loop continues, since a transient failure is
// retried on the next tick and a stuck job costs only the extra interval before the
// next attempt reclaims it.
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
