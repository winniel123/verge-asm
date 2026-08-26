package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/wire"
)

// Prober runs one job spec and returns the observations it produced. The
// production impl execs the measurement binary; a test supplies a fake.
type Prober interface {
	Probe(ctx context.Context, spec wire.JobSpec) ([]wire.Observation, error)
}

// ExecProber execs the measurement binary at Path, writing the spec to its
// stdin and decoding every NDJSON observation it writes back — ADR-0001's
// job-spec-in / NDJSON-out contract.
type ExecProber struct {
	Path string
}

// Probe implements Prober.
func (p ExecProber) Probe(ctx context.Context, spec wire.JobSpec) ([]wire.Observation, error) {
	var stdin bytes.Buffer
	if err := wire.EncodeJobSpec(&stdin, spec); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, p.Path) // #nosec G204 (Path is operator-configured; no argv args, spec via stdin per ADR-0001 — no tainted input)
	cmd.Stdin = &stdin
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("queue: exec prober: %w (stderr: %s)", err, stderr.String())
	}
	sc := wire.NewObservationScanner(&stdout)
	var obs []wire.Observation
	for sc.Next() {
		obs = append(obs, sc.Observation())
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("queue: decode prober output: %w", err)
	}
	return obs, nil
}

// Worker claims jobs off the queue and runs them, committing each job's outcome
// and its observations in one transaction.
type Worker struct {
	pool   *pgxpool.Pool
	q      *db.Queries
	prober Prober
	now    func() time.Time
	log    *log.Logger

	// The CT runner's seams (ADR-0106), wired via WithCT. Nil on a worker built
	// without them: a `ct` job then refuses rather than silently admitting
	// nothing. Separate from NewWorker so the measurement-only construction stays
	// unchanged.
	ctFetcher  CTFetcher
	ctThrottle CTThrottle
}

// NewWorker builds a Worker over pool driving prober.
func NewWorker(pool *pgxpool.Pool, prober Prober, now func() time.Time, logger *log.Logger) *Worker {
	if now == nil {
		now = time.Now
	}
	return &Worker{pool: pool, q: db.New(pool), prober: prober, now: now, log: logger}
}

// Run drains the queue, then waits on LISTEN/NOTIFY (with a ticker fallback so a
// missed notification still makes progress) until ctx is done.
func (w *Worker) Run(ctx context.Context) error {
	conn, err := w.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "LISTEN "+notifyChannel); err != nil {
		return err
	}

	for {
		if err := w.drain(ctx); err != nil && !errors.Is(err, context.Canceled) {
			w.log.Printf("worker: drain: %v", err)
		}
		waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := conn.Conn().WaitForNotification(waitCtx)
		cancel()
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				return ctx.Err()
			}
			// A timeout is the ticker fallback: loop and drain again.
		}
	}
}

// Drain runs every claimable job until none remain. It is what a manual trigger
// calls to run a fan-out to completion synchronously.
func (w *Worker) Drain(ctx context.Context) error { return w.drain(ctx) }

// drain runs claimable jobs until none remain.
func (w *Worker) drain(ctx context.Context) error {
	for {
		ran, err := w.RunOnce(ctx)
		if err != nil {
			return err
		}
		if !ran {
			return nil
		}
	}
}

// RunOnce claims one job and processes it. It returns false when the queue had
// no claimable job.
func (w *Worker) RunOnce(ctx context.Context) (bool, error) {
	job, err := w.q.ClaimJob(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("queue: claim: %w", err)
	}
	if err := w.process(ctx, job); err != nil {
		return true, fmt.Errorf("queue: process job %d: %w", job.ID, err)
	}
	return true, nil
}

func (w *Worker) process(ctx context.Context, job db.ClaimJobRow) error {
	var spec wire.JobSpec
	if err := json.Unmarshal(job.Spec, &spec); err != nil {
		return fmt.Errorf("decode spec: %w", err)
	}

	// The zone Scan is worker-read: no prober exec, no Vantage, and its
	// observations are stamped at the operator's supply instant (v1 spec §3.4).
	if spec.Kind == scan.ZoneKind {
		return w.completeZone(ctx, job, spec)
	}

	// The ct Scan is worker-read too, but it fetches crt.sh (a network step, so it
	// retries/dead-letters) and admits without observing — no observation, no span
	// (ADR-0027, ADR-0106).
	if spec.Kind == scan.CTKind {
		return w.completeCT(ctx, job, spec)
	}

	obs, probeErr := w.prober.Probe(ctx, spec)
	if probeErr != nil {
		// A transient failure. Retry is a new Batch, never a resumption: while
		// attempts remain we enqueue a fresh job; past them we dead-letter.
		if job.Attempt < job.MaxAttempts {
			return w.retry(ctx, job, probeErr)
		}
		return w.deadLetter(ctx, job, probeErr)
	}
	return w.complete(ctx, job, obs)
}

// errJobCanceled signals that a job's guarded terminal write matched no row because a
// stop or terminate (DF-F4) cancelled the job out from under the worker. Returned from
// inside a job's transaction, it rolls the whole transaction back — discarding the
// staged batch and observations, so a terminate's "uncommitted work is discarded"
// holds — and is then swallowed as a benign outcome by runJobTx: the cancellation
// already recorded the job's terminal ('cancelled') state, so nothing more is owed.
var errJobCanceled = errors.New("queue: job canceled mid-flight")

// markDone applies the guarded done transition, turning a zero-row result (the job was
// cancelled mid-flight) into errJobCanceled so the caller's transaction rolls back.
func markDone(ctx context.Context, qtx *db.Queries, jobID, batchID int64) error {
	n, err := qtx.MarkJobDone(ctx, db.MarkJobDoneParams{ID: jobID, BatchID: pgInt8(batchID)})
	if err != nil {
		return err
	}
	if n == 0 {
		return errJobCanceled
	}
	return nil
}

// markDead is markDone's dead-letter twin: a job cancelled mid-flight does not
// dead-letter, so a zero-row result rolls the transaction back.
func markDead(ctx context.Context, qtx *db.Queries, jobID, batchID int64) error {
	n, err := qtx.MarkJobDead(ctx, db.MarkJobDeadParams{ID: jobID, BatchID: pgInt8(batchID)})
	if err != nil {
		return err
	}
	if n == 0 {
		return errJobCanceled
	}
	return nil
}

// markRetried marks the current attempt retired. A zero-row result means the job was
// cancelled mid-flight, so the fresh attempt the caller enqueued in the same tx is
// rolled back with it — a terminated run does not retry.
func markRetried(ctx context.Context, qtx *db.Queries, jobID int64) error {
	n, err := qtx.MarkJobRetried(ctx, jobID)
	if err != nil {
		return err
	}
	if n == 0 {
		return errJobCanceled
	}
	return nil
}

// runJobTx runs a job's terminal transaction and treats a mid-flight cancellation as a
// benign no-op: errJobCanceled means the tx already rolled back (its work discarded)
// and the job's terminal state is recorded by the cancellation, so there is nothing
// left to do or to log as a failure.
func (w *Worker) runJobTx(ctx context.Context, jobID int64, fn func(*db.Queries) error) error {
	err := w.inTx(ctx, fn)
	if errors.Is(err, errJobCanceled) {
		w.log.Printf("worker: job %d canceled mid-flight; uncommitted work discarded", jobID)
		return nil
	}
	return err
}

// complete writes the Batch, its Observations and the job's done state in one
// transaction — the outcome and the observation data commit together.
func (w *Worker) complete(ctx context.Context, job db.ClaimJobRow, obs []wire.Observation) error {
	return w.runJobTx(ctx, job.ID, func(qtx *db.Queries) error {
		batchID, err := qtx.InsertBatch(ctx, db.InsertBatchParams{
			ScanID:        job.ScanID,
			DispatchID:    job.DispatchID,
			VantageID:     job.VantageID,
			Kind:          job.Kind,
			Outcome:       outcomeCompleted,
			Offers:        job.Offers,
			RecordedScope: job.AttemptedScope,
		})
		if err != nil {
			return err
		}
		// A completed resolution-walk Batch is proof the vantage's resolver can
		// observe: derive its Availability back to 'available' in the same
		// transaction (ADR-0108). Scoped to the dns kind so a completing port
		// probe never clobbers a resolver outage.
		if err := applyAvailability(ctx, qtx, job.VantageID, job.Kind, outcomeCompleted); err != nil {
			return err
		}
		observedAt := w.now().UTC()
		for _, p := range toObservationParams(batchID, job.VantageID, tstz(observedAt), obs) {
			if err := qtx.InsertObservation(ctx, p); err != nil {
				return err
			}
		}
		// The declared-input context (Seeds, Exclusions) the fold composes membership
		// against — read once for both the aperture-widened opening marker and the
		// withdrawal closure below (internal/estate).
		membership, err := readMembershipInputs(ctx, qtx)
		if err != nil {
			return err
		}
		// Fold the completed batch's observations into the Span corpus in the same
		// transaction — the outcome, its observations and the drift they move all
		// commit together (ADR-0007).
		if err := foldObservationsIntoSpans(ctx, qtx, batchID, job.VantageID, observedAt, obs, membership); err != nil {
			return err
		}
		// Compose the subject-level departures the batch's evidence shows and close
		// their timelines with the estate-decided ground (internal/estate wired into
		// the spanfold closure, #637) — the withdrawn / descoped closures, and the
		// re-open that lets a later `returned` derive, all citing this batch.
		if err := foldEstateTransitions(ctx, qtx, batchID, observedAt, obs, membership); err != nil {
			return err
		}
		return markDone(ctx, qtx, job.ID, batchID)
	})
}

// deadLetter records a Batch whose scope is empty — never the attempted one —
// and marks the job dead, together.
func (w *Worker) deadLetter(ctx context.Context, job db.ClaimJobRow, cause error) error {
	w.log.Printf("worker: job %d dead-lettered after %d attempts: %v", job.ID, job.Attempt, cause)
	return w.runJobTx(ctx, job.ID, func(qtx *db.Queries) error {
		batchID, err := qtx.InsertBatch(ctx, db.InsertBatchParams{
			ScanID:        job.ScanID,
			DispatchID:    job.DispatchID,
			VantageID:     job.VantageID,
			Kind:          job.Kind,
			Outcome:       outcomeDeadLettered,
			Offers:        job.Offers,
			RecordedScope: []byte(`{"names":[]}`),
		})
		if err != nil {
			return err
		}
		// A dead-lettered resolution-walk Batch failed every attempt across the
		// retry window: the vantage's resolver could not observe, so derive its
		// Availability to 'unavailable' (ADR-0108). Scoped to the dns kind — a
		// dead-lettered port probe is its own durable failure, not a resolver
		// outage, and does not move this scalar.
		if err := applyAvailability(ctx, qtx, job.VantageID, job.Kind, outcomeDeadLettered); err != nil {
			return err
		}
		return markDead(ctx, qtx, job.ID, batchID)
	})
}

// retry enqueues a new job (attempt+1) and marks the current one retried, so the
// eventual Batch is a fresh one and no partial batch is ever resumed.
func (w *Worker) retry(ctx context.Context, job db.ClaimJobRow, cause error) error {
	w.log.Printf("worker: job %d attempt %d failed, retrying: %v", job.ID, job.Attempt, cause)
	return w.runJobTx(ctx, job.ID, func(qtx *db.Queries) error {
		_, err := qtx.EnqueueJob(ctx, db.EnqueueJobParams{
			ScanID:         job.ScanID,
			VantageID:      job.VantageID,
			DispatchID:     job.DispatchID,
			Kind:           job.Kind,
			Spec:           job.Spec,
			AttemptedScope: job.AttemptedScope,
			Offers:         job.Offers,
			Attempt:        job.Attempt + 1,
			MaxAttempts:    job.MaxAttempts,
			RunAfter:       tstz(w.now().UTC().Add(backoff(job.Attempt + 1))),
		})
		if err != nil {
			return err
		}
		return markRetried(ctx, qtx, job.ID)
	})
}

func (w *Worker) inTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(w.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
