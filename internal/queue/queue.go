// Package queue is the Postgres-backed dispatch and worker loop (v1 spec §4.1):
// `SELECT … FOR UPDATE SKIP LOCKED` + `LISTEN/NOTIFY`, not a broker. One queue
// job is one Batch; job outcome and observation data commit together. Retry is
// always a new Batch, never a resumption; a dead-lettered Batch records an empty
// scope. Dispatch fires under a Postgres advisory lock, idempotent on
// (scan, scheduled_time); overlapping ticks are skipped and recorded, never run
// concurrently.
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
)

const notifyChannel = "queue_job"

// Dispatcher fans a Scan out into queue jobs on its cadence and on demand.
type Dispatcher struct {
	pool *pgxpool.Pool
	q    *db.Queries
	now  func() time.Time
	log  *log.Logger
}

// NewDispatcher builds a Dispatcher over pool. now is injectable so tests and
// manual triggers can control the scheduled tick.
func NewDispatcher(pool *pgxpool.Pool, now func() time.Time, logger *log.Logger) *Dispatcher {
	if now == nil {
		now = time.Now
	}
	return &Dispatcher{pool: pool, q: db.New(pool), now: now, log: logger}
}

// Run fans out every enabled Scan once per cadence window until ctx is done. It
// polls each minute; the idempotency key makes a second poll inside one window a
// recorded skip rather than a second fan-out.
func (d *Dispatcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	d.dispatchDue(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			d.dispatchDue(ctx)
		}
	}
}

func (d *Dispatcher) dispatchDue(ctx context.Context) {
	scans, err := d.q.ListEnabledScans(ctx)
	if err != nil {
		d.log.Printf("dispatcher: list scans: %v", err)
		return
	}
	for _, s := range scans {
		switch s.Kind {
		case scan.DNSKind, scan.ZoneKind, scan.HotKind, scan.ColdKind, scan.TLSAcceptanceKind, scan.CTKind:
			// The dns Scan (worker-probed, per Vantage), the zone Scan
			// (worker-read, no Vantage), the hot Scan (Custody-gated, per Vantage),
			// the cold Scan (Custody-gated, opt-in per Seed scope, full range) and
			// the tls-acceptance Scan (weekly enumeration over the open Service
			// population, no port list — ADR-0028) and the ct Scan (worker-read
			// crt.sh poll that admits Names without observing, no port list, no
			// vantage — ADR-0106) each fan out on their own cadence.
			// The cold Scan reaches this switch only while at least one Seed scope has
			// opted in — that is what flips it enabled and into ListEnabledScans
			// (ADR-0044); shipped disabled, it is skipped here and never fires unasked.
		default:
			continue
		}
		tick := scheduledTick(d.now(), time.Duration(s.CadenceSeconds)*time.Second)
		if _, err := d.fanOut(ctx, s, tick); err != nil {
			d.log.Printf("dispatcher: fan out %s: %v", s.Kind, err)
		}
	}
}

// Trigger fans a Scan out immediately, regardless of where the cadence window
// sits, by keying the Dispatch on the current instant. It is the manual-run
// entrypoint (v1 spec §3.4: a manual run dispatches an existing Scan).
//
// A DISABLED Scan is refused: a manual dispatch of a disabled Scan is exactly
// the ad-hoc one-off ADR-0005/ADR-0044 forbid — a batch whose scope no enabled
// configured object accounts for, and (for the cold tier) a full-range sweep
// with no cadence and therefore no currency bound. The onboarding baseline
// ("Run the first batch") dispatches only the Scans that exist AND are enabled,
// so the shipped-disabled cold Scan never fires unasked. Once a Seed scope opts
// in, the cold Scan is enabled and this manual path dispatches it as the ordinary
// configured object it has become.
func (d *Dispatcher) Trigger(ctx context.Context, kind string) (int, error) {
	s, err := d.q.GetScanByKind(ctx, kind)
	if err != nil {
		return 0, fmt.Errorf("queue: get scan %q: %w", kind, err)
	}
	if !s.Enabled {
		return 0, fmt.Errorf("queue: %s Scan is disabled — a manual run dispatches an enabled Scan, never a one-off (ADR-0044)", kind)
	}
	return d.fanOut(ctx, s, d.now().UTC().Truncate(time.Second))
}

// fanOut inserts a Dispatch for (scan, scheduledTime) under a per-scan advisory
// lock and enqueues its jobs — per Vantage for the dns Scan, per supplied zone
// file for the zone Scan. It returns 0 with no error when the tick was already
// dispatched: the overlap is skipped (nothing runs, nothing is enqueued) and
// recorded by the pre-existing fanned-out Dispatch that owns the tick — the
// unique (scan, scheduled_time) key admits only one.
func (d *Dispatcher) fanOut(ctx context.Context, s db.Scan, scheduledTime time.Time) (int, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	qtx := d.q.WithTx(tx)

	// The advisory lock serialises concurrent fan-outs of the same Scan; the
	// unique (scan, scheduled_time) key is the durable idempotency backstop.
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", s.ID); err != nil {
		return 0, err
	}

	dispatchID, err := qtx.TryFanOut(ctx, db.TryFanOutParams{ScanID: s.ID, ScheduledTime: tstz(scheduledTime)})
	if errors.Is(err, pgx.ErrNoRows) {
		// Overlapping tick: the window is already dispatched. Recorded by the
		// existing Dispatch row; we run nothing and enqueue nothing.
		d.log.Printf("dispatcher: %s tick %s already dispatched, skipped", s.Kind, scheduledTime.Format(time.RFC3339))
		return 0, tx.Commit(ctx)
	}
	if err != nil {
		return 0, fmt.Errorf("queue: try fan out: %w", err)
	}

	enqueued := 0
	switch s.Kind {
	case scan.ZoneKind:
		enqueued, err = d.fanOutZone(ctx, qtx, s.ID, dispatchID)
	case scan.HotKind:
		enqueued, err = d.fanOutHot(ctx, qtx, s.ID, dispatchID)
	case scan.ColdKind:
		enqueued, err = d.fanOutCold(ctx, qtx, s.ID, dispatchID)
	case scan.TLSAcceptanceKind:
		enqueued, err = d.fanOutTLSAcceptance(ctx, qtx, s.ID, dispatchID)
	case scan.CTKind:
		enqueued, err = d.fanOutCT(ctx, qtx, s.ID, dispatchID)
	default:
		enqueued, err = d.fanOutDNS(ctx, qtx, s.ID, dispatchID)
	}
	if err != nil {
		return 0, err
	}

	if enqueued > 0 {
		if _, err := tx.Exec(ctx, "SELECT pg_notify($1, '')", notifyChannel); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	d.log.Printf("dispatcher: %s fanned out %d job(s) at %s", s.Kind, enqueued, scheduledTime.Format(time.RFC3339))
	return enqueued, nil
}

// fanOutDNS enqueues one dns job per Vantage over the name-scope Seeds.
func (d *Dispatcher) fanOutDNS(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64) (int, error) {
	names, err := nameSeedDomains(ctx, qtx)
	if err != nil {
		return 0, err
	}
	vantages, err := vantageList(ctx, qtx)
	if err != nil {
		return 0, err
	}
	enqueued := 0
	for _, j := range scan.BuildDNSJobs(scanID, names, vantages.scanVantages()) {
		j = j.WithResolver(vantages.resolver(j.VantageID)).WithSeeds(names)
		if err := enqueueJob(ctx, qtx, scanID, dispatchID, j); err != nil {
			return 0, err
		}
		enqueued++
	}
	return enqueued, nil
}

// fanOutZone enqueues one worker-read job per supplied zone file. No Vantage is
// read: the zone Scan has no vantage choice at all (v1 spec §3.4).
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
