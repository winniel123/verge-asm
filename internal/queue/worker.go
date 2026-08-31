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
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/wire"
)

// Prober runs one job spec and returns a ProbeResult — the observations it produced
// and the verbatim Transcript of the exchange (#838). The production impl execs the
// measurement binary; a test supplies a fake. The Transcript is ABSENT until #840
// captures it at the prober boundary; this seam only carries the shape.
type Prober interface {
	Probe(ctx context.Context, spec wire.JobSpec) (wire.ProbeResult, error)
}

// ExecProber execs the measurement binary at Path, writing the spec to its
// stdin and decoding every NDJSON observation it writes back — ADR-0001's
// job-spec-in / NDJSON-out contract.
type ExecProber struct {
	Path string
}

// Probe implements Prober. It returns the observations wrapped in a ProbeResult with
// an ABSENT Transcript: this ticket (#863) reshapes the seam only — #840 captures the
// stdin/stdout/stderr and the exit outcome into the Transcript at this boundary.
func (p ExecProber) Probe(ctx context.Context, spec wire.JobSpec) (wire.ProbeResult, error) {
	var stdin bytes.Buffer
	if err := wire.EncodeJobSpec(&stdin, spec); err != nil {
		return wire.ProbeResult{}, err
	}
	cmd := exec.CommandContext(ctx, p.Path) // #nosec G204 (Path is operator-configured; no argv args, spec via stdin per ADR-0001 — no tainted input)
	cmd.Stdin = &stdin
	// Fail-closed stdout sink: even the local prober is treated as untrusted
	// output for this bound (#772) — its stdout is capped at MaxProberStdout
	// during the copy rather than buffered without limit, so a runaway or
	// compromised prober binary cannot OOM the worker. Exceeding the cap makes
	// cmd.Run return a write error, driving the normal retry/dead-letter path.
	stdout := wire.NewLimitedBuffer(wire.MaxProberStdout)
	var stderr bytes.Buffer
	cmd.Stdout = stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return wire.ProbeResult{}, fmt.Errorf("queue: exec prober: %w (stderr: %s)", err, stderr.String())
	}
	sc := wire.NewObservationScanner(bytes.NewReader(stdout.Bytes()))
	var obs []wire.Observation
	for sc.Next() {
		obs = append(obs, sc.Observation())
	}
	if err := sc.Err(); err != nil {
		return wire.ProbeResult{}, fmt.Errorf("queue: decode prober output: %w", err)
	}
	return wire.ProbeResult{Observations: obs}, nil
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

	// The ct-tail runner's fetcher (spec §4), wired via WithCTTail. Nil on a worker
	// built without it: a `ct-tail` job then refuses rather than silently admitting
	// nothing. It reuses the CTFetcher seam, pointed at the RFC 6962 log endpoints.
	ctTailFetcher CTFetcher

	// The off-host measurement router (ADR-0103, #683), wired via WithRouter. A
	// provisioned internet Vantage measures from its OWN position: its jobs are pushed
	// to and exec'd on the prober host over SSH, not run locally on the instance. Nil
	// on a worker built without it — every job then runs on the local prober, exactly
	// as before this seam — so the measurement-only construction and its tests are
	// unchanged.
	router VantageRouter

	// The message producer's seam (P0.7), wired via WithMessages. When enabled the
	// batch tx folds each signal/drift transition into a message and routes it to its
	// bound channels via enqueue (delivery.EnqueueForMessage, injected to avoid the
	// delivery→queue import cycle). Off on a worker built without it — the
	// measurement-only construction and its tests write no message. devMode suppresses
	// production entirely even when enabled: a fixture-only install never writes a
	// message (AL-25), so the golden fixtures stay message-free and G2 does not move.
	produceMsgs bool
	devMode     bool
	enqueue     func(ctx context.Context, q *db.Queries, messageID int64, class message.Class) (int, error)
}

// WithMessages enables the message producer (P0.7): after a batch's folds commit,
// the same transaction folds each flagship / membership transition into a Message and
// routes it to its bound Channels through enqueue — delivery.EnqueueForMessage bound
// to the batch tx by the producer. devMode is the VERGE_DEV guard: when true the
// producer is a no-op, so a fixture install writes no message. Returns the worker for
// chaining beside WithCT.
func (w *Worker) WithMessages(enqueue func(ctx context.Context, q *db.Queries, messageID int64, class message.Class) (int, error), devMode bool) *Worker {
	w.produceMsgs = true
	w.devMode = devMode
	w.enqueue = enqueue
	return w
}

// changeCollector is the transition collector the fold appends to — the real slice
// pointer where the producer is enabled and needs the feed, nil where it is off so
// the fold does no bookkeeping the measurement-only path would discard.
func (w *Worker) changeCollector(changes *[]spanChange) *[]spanChange {
	if !w.produceMsgs {
		return nil
	}
	return changes
}

// departureCollector is the estate-fold's withdrawal collector, twin to
// changeCollector: the real slice pointer where the producer is enabled and needs the
// declared-input feed (AL-2, #722), nil where it is off so foldEstateTransitions does
// no bookkeeping the measurement-only path would discard.
func (w *Worker) departureCollector(deps *[]departure) *[]departure {
	if !w.produceMsgs {
		return nil
	}
	return deps
}

// produce folds the batch's transitions into messages inside the batch tx. It binds
// the injected enqueuer to this transaction's queries so the producer never imports
// internal/delivery, and is a no-op unless the worker was built WithMessages. The
// devMode guard lives inside produceMessages (AL-25), so an enabled dev worker still
// writes nothing.
func (w *Worker) produce(ctx context.Context, qtx *db.Queries, batchID int64, observedAt time.Time, changes []spanChange, departures []departure, in membershipInputs) error {
	if !w.produceMsgs {
		return nil
	}
	var enqueue enqueueFunc
	if w.enqueue != nil {
		enqueue = func(c context.Context, messageID int64, class message.Class) (int, error) {
			return w.enqueue(c, qtx, messageID, class)
		}
	}
	return produceMessages(ctx, qtx, batchID, observedAt, changes, departures, in, enqueue, w.devMode)
}

// VantageRouter decides whether a job runs off-host and, if so, runs it there. It is
// consulted per job before the local prober: ProbeVantage reports handled=false for a
// vantage with no prober (the resolver-only `local` position), so that job falls
// through to the local ExecProber; handled=true means the observations came from the
// prober host over SSH. An error is a transient measurement failure (unreachable host,
// push failure) and drives the same retry/dead-letter path a local probe error does.
type VantageRouter interface {
	ProbeVantage(ctx context.Context, vantageID pgtype.Int8, spec wire.JobSpec) (res wire.ProbeResult, handled bool, err error)
}

// NewWorker builds a Worker over pool driving prober.
func NewWorker(pool *pgxpool.Pool, prober Prober, now func() time.Time, logger *log.Logger) *Worker {
	if now == nil {
		now = time.Now
	}
	return &Worker{pool: pool, q: db.New(pool), prober: prober, now: now, log: logger}
}

// WithRouter wires the off-host measurement router onto the Worker (ADR-0103, #683).
// It is separate from NewWorker so the local-only worker construction and its tests
// stay unchanged; a worker with no router runs every job on the local prober.
func (w *Worker) WithRouter(router VantageRouter) *Worker {
	w.router = router
	return w
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

	// The ct-tail Scan is worker-read too: it polls a CT log directly, forward-only,
	// and admits without observing — no observation, no span (spec §4, ADR-0027).
	if spec.Kind == scan.CTTailKind {
		return w.completeCTTail(ctx, job, spec)
	}

	res, probeErr := w.probe(ctx, job.VantageID, spec)
	if probeErr != nil {
		// A transient failure. Retry is a new Batch, never a resumption: while
		// attempts remain we enqueue a fresh job; past them we dead-letter. The
		// bound is single-sourced in exhaustedRetries, shared with the ct path.
		//
		// res.Transcript is absent today (#863 carries the shape only); #840 will
		// thread it onto the retry/dead-letter tx so a failed job keeps its raw
		// output, which is exactly when it is most wanted (§2.2).
		if exhaustedRetries(job.Attempt, job.MaxAttempts) {
			return w.deadLetter(ctx, job, probeErr)
		}
		return w.retry(ctx, job, probeErr)
	}
	return w.complete(ctx, job, res.Observations)
}

// probe runs a job's measurement, routing it off-host when a provisioned prober owns
// the vantage and running it on the local prober otherwise. The router (when wired) is
// consulted first: it reports handled=false for a vantage with no prober, so that job
// falls through to the local ExecProber exactly as before this seam existed.
func (w *Worker) probe(ctx context.Context, vantageID pgtype.Int8, spec wire.JobSpec) (wire.ProbeResult, error) {
	if w.router != nil {
		res, handled, err := w.router.ProbeVantage(ctx, vantageID, spec)
		if err != nil {
			return wire.ProbeResult{}, err
		}
		if handled {
			return res, nil
		}
	}
	return w.prober.Probe(ctx, spec)
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
	// Re-gate the prober's self-reported subjects against what this job authorised
	// (#773): a compromised prober can put any string in an Observation's Subject —
	// the field written as SubjectKey and keyed on by the span/estate/message folds —
	// so any observation naming a subject outside job.AttemptedScope is dropped before
	// the write, rather than minting false spans/drift/messages for a subject the job
	// never dispatched. Dropped lines are logged; legitimate lines are untouched.
	obs = parseAuthorizedScope(job.AttemptedScope).gate(obs, w.log, job.ID)
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
		// Land any captured certificate material in its fingerprint-keyed side store, in
		// the same transaction (ADR-0027, spec §5.3). It rides certificate observation
		// lines BESIDE the facet value, never inside it, so the observation still records
		// only the fingerprint and the fence stays closed. The insert is deduped and
		// immutable — ON CONFLICT DO NOTHING keeps the first capture — so repeated
		// presentations of one certificate write its material once.
		for _, o := range obs {
			if o.CertMaterial == nil {
				continue
			}
			if err := qtx.InsertCertificateMaterial(ctx, db.InsertCertificateMaterialParams{
				Fingerprint: o.CertMaterial.Fingerprint,
				Der:         o.CertMaterial.DER,
				Scts:        o.CertMaterial.SCTs,
			}); err != nil {
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
		// commit together (ADR-0007). The fold also collects each transition it made
		// into `changes` — the estate/drift feed the message producer consumes below.
		var changes []spanChange
		var departures []departure
		if err := foldObservationsIntoSpans(ctx, qtx, batchID, job.VantageID, observedAt, obs, membership, w.changeCollector(&changes)); err != nil {
			return err
		}
		// Compose the subject-level departures the batch's evidence shows and close
		// their timelines with the estate-decided ground (internal/estate wired into
		// the spanfold closure, #637) — the withdrawn / descoped closures, and the
		// re-open that lets a later `returned` derive, all citing this batch.
		if err := foldEstateTransitions(ctx, qtx, batchID, observedAt, obs, membership, w.departureCollector(&departures)); err != nil {
			return err
		}
		// Fold each signal/drift transition into a Message and route it to its bound
		// channels, in this same transaction (P0.7): a flagship internet-leg move or a
		// membership entry becomes a Message row and its Deliveries. A no-op unless the
		// worker was built WithMessages, and always a no-op in devMode (AL-25).
		if err := w.produce(ctx, qtx, batchID, observedAt, changes, departures, membership); err != nil {
			return err
		}
		// Ephemeral per-job progress (#780, collision #40): a completed measurement rides a
		// count of what it observed onto the live stream — redacted to the count alone, never
		// the observations. Nothing is persisted by this; the state-derived .Log stands.
		w.emitJobEvent(ctx, qtx, job, "", countLabel(len(obs), "observation", "observations"))
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
		// Ephemeral per-job progress (#780, collision #40): the redacted dead-letter reason
		// rides the live stream as an appended line so the operator sees WHY the job gave up,
		// not a bare `dead`.
		w.emitJobEvent(ctx, qtx, job, "error", deadLetterLabel(job.Attempt, cause))
		return markDead(ctx, qtx, job.ID, batchID)
	})
}

// retry enqueues a new job (attempt+1) and marks the current one retried, so the
// eventual Batch is a fresh one and no partial batch is ever resumed.
func (w *Worker) retry(ctx context.Context, job db.ClaimJobRow, cause error) error {
	w.log.Printf("worker: job %d attempt %d failed, retrying: %v", job.ID, job.Attempt, cause)
	return w.runJobTx(ctx, job.ID, func(qtx *db.Queries) error {
		if _, err := qtx.EnqueueJob(ctx, db.EnqueueJobParams{
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
		}); err != nil {
			return err
		}
		// Ephemeral per-job progress (#780, collision #40): the redacted retry reason — the
		// crt.sh-502 the ticket cites — keyed to the job that failed, so the stream appends it
		// as a live line beside that job's state. This is the producer half of collision #40.
		w.emitJobEvent(ctx, qtx, job, "warn", retryLabel(job.Attempt, cause))
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
