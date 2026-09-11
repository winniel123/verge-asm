// Package queue is the Postgres-backed dispatch and worker loop: SELECT … FOR UPDATE SKIP
// LOCKED and LISTEN/NOTIFY, never a broker (v1-spec §4.1). One tick dispatches once on the
// unique (scan, scheduled_time) key; an overlap is skipped and recorded (ADR-0005).
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/scan"
)

const notifyChannel = "queue_job"

// A chunk bounds transaction duration and memory above the address-scope cap (ADR-0127).

const chunkCommitSize = 500

const DispatchStatusSkipped = "skipped"

type SkipReason string

const (
	SkipNone           SkipReason = ""
	SkipTickDispatched SkipReason = "tick-already-dispatched"
	SkipCadenceLag     SkipReason = "cadence-lag"
)

type Dispatcher struct {
	pool *pgxpool.Pool
	q    *db.Queries
	now  func() time.Time
	log  *log.Logger

	ctSource string

	staleJobThreshold time.Duration // zero means the reaper is off, never unset (ADR-0137 §4)

	// The enqueuer is injected because internal/delivery imports this package (ADR-0199 §1, #1316).

	enqueue func(ctx context.Context, q *db.Queries, messageID int64, class message.Class) (int, error)
}

func NewDispatcher(pool *pgxpool.Pool, now func() time.Time, logger *log.Logger) *Dispatcher {
	if now == nil {
		now = time.Now
	}
	return &Dispatcher{pool: pool, q: db.New(pool), now: now, log: logger, staleJobThreshold: DefaultStaleJobThreshold}
}

func (d *Dispatcher) WithStaleJobThreshold(threshold time.Duration) *Dispatcher {
	d.staleJobThreshold = threshold
	return d
}

// The fold enqueued no delivery for a held row, so the release poll owes it one (ADR-1806 §2).

func (d *Dispatcher) WithMessages(enqueue func(ctx context.Context, q *db.Queries, messageID int64, class message.Class) (int, error)) *Dispatcher {
	d.enqueue = enqueue
	return d
}

func (d *Dispatcher) WithCTSource(slug string) *Dispatcher {
	// The worker picks this at wire-time by operator-key presence (ct-source-replacement §2.3).
	d.ctSource = slug
	return d
}

func (d *Dispatcher) selectedCTSource() string {
	if d.ctSource == "" {
		return scan.CrtshSource
	}
	return d.ctSource
}

func (d *Dispatcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	d.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			d.tick(ctx)
		}
	}
}

func (d *Dispatcher) tick(ctx context.Context) {
	d.dispatchDue(ctx)
	// A hot fan-out commits its jobs after its dispatch row, so this runs second (ADR-1806 §3).
	d.releaseDue(ctx)
	d.settleDue(ctx)
}

func (d *Dispatcher) settleDue(ctx context.Context) {
	fired, err := d.settleRePoints(ctx)
	if err != nil {
		d.log.Printf("dispatcher: settle the re-point residue: %v", err)
		return
	}
	if fired > 0 {
		d.log.Printf("dispatcher: fired %d re-point message(s)", fired)
	}
}

func (d *Dispatcher) settleRePoints(ctx context.Context) (int, error) {
	// Settling on a Dispatcher that routes nothing would lose the message (ADR-0026 §2).
	if d.enqueue == nil {
		return 0, nil
	}
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	// The claim and the messages it owes commit together, so a lost pass re-derives them.
	defer tx.Rollback(ctx)
	qtx := d.q.WithTx(tx)

	fired, err := settleRePoints(ctx, qtx, d.now().UTC(), !HotLagGateArmed(d.staleJobThreshold), func(c context.Context, messageID int64, class message.Class) (int, error) {
		return d.enqueue(c, qtx, messageID, class)
	})
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return fired, nil
}

func (d *Dispatcher) releaseDue(ctx context.Context) {
	released, err := d.releaseHeld(ctx)
	if err != nil {
		d.log.Printf("dispatcher: release held message: %v", err)
		return
	}
	if released > 0 {
		d.log.Printf("dispatcher: released %d held message(s)", released)
	}
}

func (d *Dispatcher) releaseHeld(ctx context.Context) (int, error) {
	// Clearing the hold on a Dispatcher that routes nothing would lose the delivery (ADR-1806 §2).
	if d.enqueue == nil {
		return 0, nil
	}
	// With no reaper the drain test can never conclude, so the hold is off here too (ADR-1806 §6).
	rows, err := d.q.ListReleasableHeldMessages(ctx, !HotLagGateArmed(d.staleJobThreshold))
	if err != nil {
		return 0, fmt.Errorf("queue: list releasable held messages: %w", err)
	}
	released := 0
	for _, row := range rows {
		took, err := d.releaseOne(ctx, row)
		if err != nil {
			// One row must not hold the rest back, so the pass logs and continues (ADR-0141).
			d.log.Printf("dispatcher: release message %d: %v", row.ID, err)
			continue
		}
		if took {
			released++
		}
	}
	return released, nil
}

func (d *Dispatcher) releaseOne(ctx context.Context, row db.ListReleasableHeldMessagesRow) (bool, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	// One row is one transaction, so the census and its delivery commit together (ADR-1806 §2).
	defer tx.Rollback(ctx)
	qtx := d.q.WithTx(tx)

	took, err := releaseHeldMessage(ctx, qtx, row, func(c context.Context, messageID int64, class message.Class) (int, error) {
		return d.enqueue(c, qtx, messageID, class)
	})
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return took, nil
}

func (d *Dispatcher) dispatchDue(ctx context.Context) {
	scans, err := d.q.ListEnabledScans(ctx)
	if err != nil {
		d.log.Printf("dispatcher: list scans: %v", err)
		return
	}
	for _, s := range scans {
		switch s.Kind {
		case scan.DNSKind, scan.ZoneKind, scan.HotKind, scan.ColdKind, scan.TLSAcceptanceKind, scan.CTKind, scan.HTTPIdentityKind, scan.CTTailKind, scan.EdgeFanoutKind:
			// The cold Scan ships disabled and appears only once a Seed scope opts in (ADR-0044).
		default:
			continue
		}
		tick := scheduledTick(d.now(), time.Duration(s.CadenceSeconds)*time.Second)
		if _, _, err := d.fanOut(ctx, s, tick); err != nil {
			d.log.Printf("dispatcher: fan out %s: %v", s.Kind, err)
		}
	}
}

func (d *Dispatcher) Trigger(ctx context.Context, kind string) (int, SkipReason, error) {
	s, err := d.q.GetScanByKind(ctx, kind)
	if err != nil {
		return 0, SkipNone, fmt.Errorf("queue: get scan %q: %w", kind, err)
	}
	if !s.Enabled {
		return 0, SkipNone, fmt.Errorf("queue: %s Scan is disabled — a manual run dispatches an enabled Scan, never a one-off (ADR-0044)", kind)
	}
	return d.fanOut(ctx, s, d.now().UTC().Truncate(time.Second))
}

func (d *Dispatcher) fanOut(ctx context.Context, s db.Scan, scheduledTime time.Time) (int, SkipReason, error) {
	switch s.Kind {
	case scan.HotKind, scan.ColdKind, scan.EdgeFanoutKind:
		return d.fanOutStreamed(ctx, s, scheduledTime)
	default:
		return d.fanOutAtomic(ctx, s, scheduledTime)
	}
}

func (d *Dispatcher) fanOutAtomic(ctx context.Context, s db.Scan, scheduledTime time.Time) (int, SkipReason, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, SkipNone, err
	}
	defer tx.Rollback(ctx)
	qtx := d.q.WithTx(tx)

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", s.ID); err != nil {
		return 0, SkipNone, err
	}

	dispatchID, err := qtx.TryFanOut(ctx, db.TryFanOutParams{ScanID: s.ID, ScheduledTime: tstz(scheduledTime)})
	if errors.Is(err, pgx.ErrNoRows) {
		d.log.Printf("dispatcher: %s tick %s already dispatched, skipped", s.Kind, scheduledTime.Format(time.RFC3339))
		return 0, SkipTickDispatched, tx.Commit(ctx)
	}
	if err != nil {
		return 0, SkipNone, fmt.Errorf("queue: try fan out: %w", err)
	}
	gated, gerr := gateHotTick(ctx, qtx, s.Kind, s.ID, dispatchID, d.staleJobThreshold, d.log)
	if gerr != nil {
		return 0, SkipNone, gerr
	}
	if gated != SkipNone {
		d.log.Printf("dispatcher: %s tick %s overtakes an undrained dispatch, recorded skipped", s.Kind, scheduledTime.Format(time.RFC3339))
		return 0, gated, tx.Commit(ctx)
	}

	enqueued := 0
	switch s.Kind {
	case scan.ZoneKind:
		enqueued, err = d.fanOutZone(ctx, qtx, s.ID, dispatchID)
	case scan.TLSAcceptanceKind:
		enqueued, err = d.fanOutTLSAcceptance(ctx, qtx, s.ID, dispatchID)
	case scan.HTTPIdentityKind:
		enqueued, err = d.fanOutHTTPIdentity(ctx, qtx, s.ID, dispatchID)
	case scan.CTKind:
		enqueued, err = d.fanOutCT(ctx, qtx, s.ID, dispatchID)
	case scan.CTTailKind:
		enqueued, err = d.fanOutCTTail(ctx, qtx, s.ID, dispatchID)
	default:
		enqueued, err = d.fanOutDNS(ctx, qtx, s.ID, dispatchID)
	}
	if err != nil {
		return 0, SkipNone, err
	}

	if enqueued > 0 {
		if _, err := tx.Exec(ctx, "SELECT pg_notify($1, '')", notifyChannel); err != nil {
			return 0, SkipNone, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, SkipNone, err
	}
	d.log.Printf("dispatcher: %s fanned out %d job(s) at %s", s.Kind, enqueued, scheduledTime.Format(time.RFC3339))
	return enqueued, SkipNone, nil
}

func (d *Dispatcher) fanOutStreamed(ctx context.Context, s db.Scan, scheduledTime time.Time) (int, SkipReason, error) {
	// A crash between chunks under-covers the claimed tick; the currency surfaces report it (#847).
	dispatchID, skip, err := d.claimDispatch(ctx, s, scheduledTime)
	if err != nil || skip != SkipNone {
		return 0, skip, err
	}

	var enqueued int
	switch s.Kind {
	case scan.HotKind:
		enqueued, err = d.fanOutHot(ctx, s.ID, dispatchID)
	case scan.ColdKind:
		enqueued, err = d.fanOutCold(ctx, s.ID, dispatchID)
	case scan.EdgeFanoutKind:
		enqueued, err = d.fanOutEdgeFanout(ctx, s.ID, dispatchID)
	}
	if err != nil {
		return enqueued, SkipNone, err
	}
	d.log.Printf("dispatcher: %s fanned out %d job(s) at %s", s.Kind, enqueued, scheduledTime.Format(time.RFC3339))
	return enqueued, SkipNone, nil
}

func (d *Dispatcher) claimDispatch(ctx context.Context, s db.Scan, scheduledTime time.Time) (dispatchID int64, skip SkipReason, err error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, SkipNone, err
	}
	defer tx.Rollback(ctx)
	qtx := d.q.WithTx(tx)

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", s.ID); err != nil {
		return 0, SkipNone, err
	}
	id, err := qtx.TryFanOut(ctx, db.TryFanOutParams{ScanID: s.ID, ScheduledTime: tstz(scheduledTime)})
	if errors.Is(err, pgx.ErrNoRows) {
		d.log.Printf("dispatcher: %s tick %s already dispatched, skipped", s.Kind, scheduledTime.Format(time.RFC3339))
		return 0, SkipTickDispatched, tx.Commit(ctx)
	}
	if err != nil {
		return 0, SkipNone, fmt.Errorf("queue: try fan out: %w", err)
	}
	// The lock ends at this commit, so a Trigger racing the cadence poll can still pass the gate.
	gated, gerr := gateHotTick(ctx, qtx, s.Kind, s.ID, id, d.staleJobThreshold, d.log)
	if gerr != nil {
		return 0, SkipNone, gerr
	}
	if gated != SkipNone {
		// A rollback leaves the window unclaimed and a later poll defers it (ADR-0137 §4).
		d.log.Printf("dispatcher: %s tick %s overtakes an undrained dispatch, recorded skipped", s.Kind, scheduledTime.Format(time.RFC3339))
		return 0, gated, tx.Commit(ctx)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, SkipNone, err
	}
	return id, SkipNone, nil
}

func streamEnqueue[J any](ctx context.Context, d *Dispatcher, jobs iter.Seq[J], enqueue func(context.Context, *db.Queries, J) error) (int, error) {
	total := 0
	chunk := make([]J, 0, chunkCommitSize)
	flush := func() error {
		if len(chunk) == 0 {
			return nil
		}
		tx, err := d.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		qtx := d.q.WithTx(tx)
		for i := range chunk {
			if err := enqueue(ctx, qtx, chunk[i]); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, "SELECT pg_notify($1, '')", notifyChannel); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		total += len(chunk)
		chunk = chunk[:0]
		return nil
	}
	for j := range jobs {
		chunk = append(chunk, j)
		if len(chunk) >= chunkCommitSize {
			if err := flush(); err != nil {
				return total, err
			}
		}
	}
	if err := flush(); err != nil {
		return total, err
	}
	return total, nil
}

func jobAddr(addrs []string) string {
	if len(addrs) == 0 {
		return "none"
	}
	return addrs[0]
}

func (d *Dispatcher) fanOutDNS(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64) (int, error) {
	// A discovered name widens the probes, never the gate, which stays at the Seeds (ADR-0066).
	seedDomains, err := nameSeedDomains(ctx, qtx)
	if err != nil {
		return 0, err
	}
	admitted, err := admittedNames(ctx, qtx)
	if err != nil {
		return 0, err
	}
	names := mergeResolutionNames(seedDomains, admitted)
	vantages, err := vantageList(ctx, qtx)
	if err != nil {
		return 0, err
	}
	jobs := scan.BuildDNSJobs(scanID, names, vantages.scanVantages())
	for i := range jobs {
		jobs[i] = jobs[i].WithResolver(vantages.resolver(jobs[i].VantageID)).WithSeeds(seedDomains)
	}
	return dispatchDNSJobs(jobs, d.log, func(j scan.Job) error {
		return enqueueJob(ctx, qtx, scanID, dispatchID, j)
	})
}

func dispatchDNSJobs(jobs []scan.Job, logger *log.Logger, enqueue func(scan.Job) error) (int, error) {
	enqueued, skipped := 0, 0
	for _, j := range jobs {
		err := enqueue(j)
		if errors.Is(err, scan.ErrNoResolver) {
			// One unconfigured position must not stop every configured one (ADR-0202 §2, #1433).
			logger.Printf("dispatcher: dns vantage %q (id %d) has no resolver, skipped", j.Vantage, j.VantageID)
			skipped++
			continue
		}
		if err != nil {
			return 0, err
		}
		enqueued++
	}
	if skipped > 0 {
		logger.Printf("dispatcher: dns fanned out past %d vantage(s) with no resolver", skipped)
	}
	return enqueued, nil
}

func (d *Dispatcher) fanOutZone(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64) (int, error) {
	files, err := zoneFiles(ctx, qtx)
	if err != nil {
		return 0, err
	}
	enqueued := 0
	for _, j := range scan.BuildZoneJobs(scanID, files) {
		if err := enqueueZoneJob(ctx, qtx, scanID, dispatchID, j); err != nil {
			return 0, err
		}
		enqueued++
	}
	return enqueued, nil
}

func enqueueJob(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64, j scan.Job) error {
	spec, err := j.JobSpec(fmt.Sprintf("scan:%d:vantage:%d", scanID, j.VantageID))
	if err != nil {
		return err
	}
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	scopeJSON, err := j.AttemptedScope()
	if err != nil {
		return err
	}
	offersJSON, err := j.OffersJSON()
	if err != nil {
		return err
	}
	_, err = qtx.EnqueueJob(ctx, db.EnqueueJobParams{
		ScanID:         scanID,
		VantageID:      pgInt8(j.VantageID),
		DispatchID:     pgInt8(dispatchID),
		Kind:           j.Kind,
		Spec:           specJSON,
		AttemptedScope: scopeJSON,
		Offers:         offersJSON,
		Attempt:        1,
		MaxAttempts:    5,
		RunAfter:       tstz(time.Now().UTC()),
	})
	return err
}
