package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
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

type Prober interface {
	Probe(ctx context.Context, spec wire.JobSpec) (wire.ProbeResult, error)
}

type ExecProber struct {
	Path string
}

func (p ExecProber) Probe(ctx context.Context, spec wire.JobSpec) (wire.ProbeResult, error) {
	var stdin bytes.Buffer
	// Nothing was sent and no exchange happened, so an encode failure carries no transcript (§2.2).
	if err := wire.EncodeJobSpec(&stdin, spec); err != nil {
		return wire.ProbeResult{}, err
	}
	// Run drains the buffer, so the bytes are copied first and the transcript holds the literal spec.
	sent := append([]byte(nil), stdin.Bytes()...)

	cmd := exec.CommandContext(ctx, p.Path) // #nosec G204 (Path is operator-configured; no argv args, spec via stdin per ADR-0001 — no tainted input)
	cmd.Stdin = &stdin
	// Even a local prober is untrusted for this bound: an uncapped stdout could OOM the worker (#772).
	stdout := wire.NewLimitedBuffer(wire.MaxProberStdout)
	var stderr bytes.Buffer
	cmd.Stdout = stdout
	cmd.Stderr = &stderr

	start := time.Now()
	runErr := cmd.Run()
	dur := time.Since(start)
	t := buildProberTranscript(spec, sent, stdout, stderr.Bytes(), dur, cmd.ProcessState, ctx.Err())

	// Raw output is highest-value when the job failed, so the transcript rides every outcome (§2.2).
	if runErr != nil {
		return wire.ProbeResult{Transcript: t}, fmt.Errorf("queue: exec prober: %w (stderr: %s)", runErr, stderr.String())
	}
	sc := wire.NewObservationScanner(bytes.NewReader(stdout.Bytes()))
	var obs []wire.Observation
	for sc.Next() {
		obs = append(obs, sc.Observation())
	}
	if err := sc.Err(); err != nil {
		return wire.ProbeResult{Transcript: t}, fmt.Errorf("queue: decode prober output: %w", err)
	}
	return wire.ProbeResult{Observations: obs, Transcript: t}, nil
}

func buildProberTranscript(spec wire.JobSpec, sent []byte, stdout *wire.LimitedBuffer, stderr []byte, dur time.Duration, ps *os.ProcessState, ctxErr error) wire.ProberTranscript {
	// The worker stamps the job id and instant at persist time, so they are left zero here (§1.2).
	return wire.ProberTranscript{
		TranscriptFrame: wire.TranscriptFrame{Kind: spec.Kind, Duration: dur},
		SentScope:       sent,
		Stdout:          stdout.Bytes(),
		Stderr:          stderr,
		Outcome:         classifyProberOutcome(ps, ctxErr),
		StdoutOverflow:  stdout.Overflowed(),
	}
}

func classifyProberOutcome(ps *os.ProcessState, ctxErr error) wire.ProberOutcome {
	// A prober killed by the deadline must not read as a clean exit, so the cancel test comes first.
	if ctxErr != nil {
		return wire.ProberContextCancelled{}
	}
	// A prober that never started has no clean exit, so exited(-1) is honest and a zero would not be.
	if ps == nil {
		return wire.ProberExited{Code: -1}
	}
	if code := ps.ExitCode(); code >= 0 {
		return wire.ProberExited{Code: code}
	}
	if sig := signalName(ps); sig != "" {
		return wire.ProberSignalled{Signal: sig}
	}
	return wire.ProberExited{Code: -1}
}

type Worker struct {
	pool   *pgxpool.Pool
	q      *db.Queries
	prober Prober
	now    func() time.Time
	log    *log.Logger

	// An unwired seam refuses rather than silently admitting nothing (ADR-0106).

	ctFetcher  CTFetcher
	ctThrottle CTThrottle

	// A nil source runs the keyless crt.sh path rather than refusing (spec §2.3).

	ctSource scan.CTSource

	ctTailFetcher CTFetcher

	ctVerifyFetcher CTFetcher

	router VantageRouter

	// The enqueuer is injected because internal/delivery imports this package (ADR-0199 §1, #1316).

	produceMsgs bool
	devMode     bool
	enqueue     func(ctx context.Context, q *db.Queries, messageID int64, class message.Class) (int, error)

	captureTranscripts bool
	transcriptKey      []byte

	probeTimeout time.Duration
}

func (w *Worker) WithTranscripts(key []byte, devMode bool) *Worker {
	w.captureTranscripts = true
	w.transcriptKey = key
	w.devMode = devMode
	return w
}

// A fixture-only install writes no transcript and no message, so no golden fixture moves.

func (w *Worker) captureOn() bool { return w.captureTranscripts && !w.devMode }

func (w *Worker) persistTranscript(ctx context.Context, qtx *db.Queries, jobID int64, t wire.Transcript) error {
	// A job with no capture inserts no row, so the absence stays legible beside an empty stream.
	if !w.captureOn() || t == nil {
		return nil
	}
	params, err := buildTranscriptParams(jobID, w.now().UTC(), t, w.transcriptKey)
	if err != nil {
		return err
	}
	return qtx.InsertTranscript(ctx, params)
}

func (w *Worker) WithMessages(enqueue func(ctx context.Context, q *db.Queries, messageID int64, class message.Class) (int, error), devMode bool) *Worker {
	w.produceMsgs = true
	w.devMode = devMode
	w.enqueue = enqueue
	return w
}

func (w *Worker) changeCollector(changes *[]spanChange) *[]spanChange {
	// A nil collector stops the fold doing bookkeeping the measurement-only path would discard.
	if !w.produceMsgs {
		return nil
	}
	return changes
}

func (w *Worker) departureCollector(deps *[]departure) *[]departure {
	if !w.produceMsgs {
		return nil
	}
	return deps
}

func (w *Worker) narrowingCollector(narrowings *[]message.NarrowingReceipt) *[]message.NarrowingReceipt {
	// A withdrawal is a fact about the estate, so the fold closes its timelines with a nil collector (ADR-0219 §1).
	if !w.produceMsgs {
		return nil
	}
	return narrowings
}

func (w *Worker) produce(ctx context.Context, qtx *db.Queries, batchID int64, observedAt time.Time, changes []spanChange, departures []departure, narrowings []message.NarrowingReceipt, in membershipInputs) error {
	// An unwired producer writes no message rather than refusing, so a measurement-only build works.
	if !w.produceMsgs {
		return nil
	}
	var enqueue enqueueFunc
	if w.enqueue != nil {
		enqueue = func(c context.Context, messageID int64, class message.Class) (int, error) {
			return w.enqueue(c, qtx, messageID, class)
		}
	}
	return produceMessages(ctx, qtx, batchID, observedAt, changes, departures, narrowings, in, enqueue, w.devMode)
}

// A provisioned Vantage measures from its own position; handled=false runs it locally (ADR-0103).

type VantageRouter interface {
	ProbeVantage(ctx context.Context, vantageID pgtype.Int8, spec wire.JobSpec) (res wire.ProbeResult, handled bool, err error)
}

// The drain loop is single-threaded, so a hung prober would block every later job (#853).

const DefaultProbeTimeout = 5 * time.Minute

func NewWorker(pool *pgxpool.Pool, prober Prober, now func() time.Time, logger *log.Logger) *Worker {
	if now == nil {
		now = time.Now
	}
	return &Worker{pool: pool, q: db.New(pool), prober: prober, now: now, log: logger, probeTimeout: DefaultProbeTimeout}
}

func (w *Worker) WithProbeTimeout(d time.Duration) *Worker {
	w.probeTimeout = d
	return w
}

func (w *Worker) WithRouter(router VantageRouter) *Worker {
	w.router = router
	return w
}

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
		// A notification can be missed, so the timeout is a fallback that still makes progress.
		waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := conn.Conn().WaitForNotification(waitCtx)
		cancel()
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				return ctx.Err()
			}
		}
	}
}

func (w *Worker) Drain(ctx context.Context) error { return w.drain(ctx) }

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

	// A worker-read Scan runs no prober and admits without observing, so it opens no span (ADR-0027).
	if spec.Kind == scan.ZoneKind {
		return w.completeZone(ctx, job, spec)
	}

	// A crt.sh fetch is a network step, so this worker-read Scan retries and dead-letters (ADR-0106).
	if spec.Kind == scan.CTKind {
		return w.completeCT(ctx, job, spec)
	}

	if spec.Kind == scan.CTTailKind {
		return w.completeCTTail(ctx, job, spec)
	}

	res, probeErr := w.probe(ctx, job.VantageID, spec)
	if probeErr != nil {
		if exhaustedRetries(job.Attempt, job.MaxAttempts) {
			return w.deadLetter(ctx, job, res.Transcript, probeErr)
		}
		return w.retry(ctx, job, res.Transcript, probeErr)
	}
	return w.complete(ctx, job, res)
}

func (w *Worker) probe(ctx context.Context, vantageID pgtype.Int8, spec wire.JobSpec) (wire.ProbeResult, error) {
	// The bracket wraps only the probe, so the terminal transaction runs under the parent ctx (#853).
	if w.probeTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, w.probeTimeout)
		defer cancel()
	}
	if w.router != nil {
		res, handled, err := w.router.ProbeVantage(ctx, vantageID, spec)
		if err != nil {
			// A failed remote probe still carries its transcript, so the error must not discard res (#867).
			return res, err
		}
		if handled {
			return res, nil
		}
	}
	return w.prober.Probe(ctx, spec)
}

// A terminate discards staged work, so a zero-row guard rolls the tx back (raw-job-output §2.4).

var errJobCanceled = errors.New("queue: job canceled mid-flight")

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

func (w *Worker) runJobTx(ctx context.Context, jobID int64, fn func(*db.Queries) error) error {
	err := w.inTx(ctx, fn)
	// The cancellation already recorded the job's terminal state, so nothing more is owed or logged.
	if errors.Is(err, errJobCanceled) {
		w.log.Printf("worker: job %d canceled mid-flight; uncommitted work discarded", jobID)
		return nil
	}
	return err
}

func (w *Worker) complete(ctx context.Context, job db.ClaimJobRow, res wire.ProbeResult) error {
	// A prober can name any subject, so a line outside the job's authorised scope is dropped (#773).
	obs := parseAuthorizedScope(job.AttemptedScope).gate(res.Observations, w.log, job.ID)
	// The outcome, its observations and its raw output must commit together (ADR-0007, spec §2.4).
	if err := w.runJobTx(ctx, job.ID, func(qtx *db.Queries) error {
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
		// A completing port probe must not clobber a resolver outage, so this is dns-scoped (ADR-0108).
		if err := applyAvailability(ctx, qtx, job.VantageID, job.Kind, outcomeCompleted); err != nil {
			return err
		}
		observedAt := w.now().UTC()
		for _, p := range toObservationParams(batchID, job.VantageID, tstz(observedAt), obs) {
			if err := qtx.InsertObservation(ctx, p); err != nil {
				return err
			}
		}
		// Material rides beside the facet value, never inside it, so the fence stays closed (ADR-0027).
		for _, o := range obs {
			if o.CertMaterial == nil {
				continue
			}
			if err := qtx.InsertCertificateMaterial(ctx, db.InsertCertificateMaterialParams{
				Fingerprint: o.CertMaterial.Fingerprint,
				Der:         o.CertMaterial.DER,
				Scts:        o.CertMaterial.SCTs,
				IssuerSpki:  o.CertMaterial.IssuerSPKI,
			}); err != nil {
				return err
			}
		}
		if err := foldEdgeFanoutObservations(ctx, qtx, job, batchID, tstz(observedAt), obs, w.log); err != nil {
			return err
		}
		membership, err := readMembershipInputs(ctx, qtx)
		if err != nil {
			return err
		}
		var changes []spanChange
		var departures []departure
		var narrowings []message.NarrowingReceipt
		if err := foldObservationsIntoSpans(ctx, qtx, batchID, job.VantageID, observedAt, obs, membership, w.changeCollector(&changes)); err != nil {
			return err
		}
		if err := foldEstateTransitions(ctx, qtx, batchID, observedAt, obs, membership, w.departureCollector(&departures)); err != nil {
			return err
		}
		// A withdrawn Seed stops its Names being enumerated, so a batch-scoped fold misses them (#1045).
		if err := foldNameSeedWithdrawals(ctx, qtx, batchID, observedAt, membership, w.narrowingCollector(&narrowings)); err != nil {
			return err
		}
		if err := foldAddressExclusionWithdrawals(ctx, qtx, batchID, observedAt, membership, w.narrowingCollector(&narrowings)); err != nil {
			return err
		}
		// The delete destroys the mover, so an address withdrawal is driven from a tombstone (ADR-0134).
		if err := foldSeedWithdrawals(ctx, qtx, batchID, observedAt, membership, w.narrowingCollector(&narrowings)); err != nil {
			return err
		}
		if err := w.produce(ctx, qtx, batchID, observedAt, changes, departures, narrowings, membership); err != nil {
			return err
		}
		// The live stream carries the count alone, never the observations, and persists nothing (#780).
		w.emitJobEvent(ctx, qtx, job, "", countLabel(len(obs), "observation", "observations"))
		if err := w.persistTranscript(ctx, qtx, job.ID, res.Transcript); err != nil {
			return err
		}
		return markDone(ctx, qtx, job.ID, batchID)
	}); err != nil {
		return err
	}
	// A CT verification does network I/O, which must never ride a database transaction (ADR-0215 §1).
	w.autoVerifyCerts(ctx, job, obs)
	return nil
}

func (w *Worker) deadLetter(ctx context.Context, job db.ClaimJobRow, t wire.Transcript, cause error) error {
	// A failed job asserts no absence, so the Batch records an empty scope, never the attempted one.
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
		if err := applyAvailability(ctx, qtx, job.VantageID, job.Kind, outcomeDeadLettered); err != nil {
			return err
		}
		w.emitJobEvent(ctx, qtx, job, "error", deadLetterLabel(job.Attempt, cause))
		if err := w.persistTranscript(ctx, qtx, job.ID, t); err != nil {
			return err
		}
		return markDead(ctx, qtx, job.ID, batchID)
	})
}

func (w *Worker) retry(ctx context.Context, job db.ClaimJobRow, t wire.Transcript, cause error) error {
	w.log.Printf("worker: job %d attempt %d failed, retrying: %v", job.ID, job.Attempt, cause)
	return w.runJobTx(ctx, job.ID, func(qtx *db.Queries) error {
		// A retry is a new Batch, never a resumption, so no partial batch is ever resumed.
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
		w.emitJobEvent(ctx, qtx, job, "warn", retryLabel(job.Attempt, cause))
		// The transcript belongs to the failed attempt's row, not the fresh job just enqueued (§1.1).
		if err := w.persistTranscript(ctx, qtx, job.ID, t); err != nil {
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
