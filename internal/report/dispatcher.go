package report

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

// Dispatcher cuts every declared report_schedule's artifact once per firing and
// stamps an in-instance receipt for the run. It mirrors package queue's Dispatcher
// closely: a minute poll, a per-item advisory lock, and idempotency on a fire tick —
// the partial-unique (schedule_id, scheduled_tick) admits only the first poll after a
// firing, so a second poll before the next firing is a recorded skip, never a second
// run. Unlike queue's epoch-floored tick, the report fire tick is the operator's
// declared clock time (preset time-of-day, or Custom cron — DispatchTick, ADR-0122).
//
// A run generates but does not leave: state is 'generated', delivered_at NULL. The
// off-instance send is a later ticket (#508/T7, ADR-0039 stands), so this loop
// confirms each due schedule is cuttable and records that it was cut, nothing more.
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

// Run dispatches every due schedule once per cadence window until ctx is done. It
// polls each minute; the idempotency key makes a second poll inside one window a
// recorded skip rather than a second run.
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
	schedules, err := d.q.ListReportSchedules(ctx)
	if err != nil {
		d.log.Printf("report dispatcher: list schedules: %v", err)
		return
	}
	for _, sc := range schedules {
		// The window is the artifact PERIOD (how much of the estate the run covers);
		// the tick is the fire INSTANT (when it runs, at the operator's declared clock
		// time — ADR-0122). They are computed separately and are orthogonal.
		window := CadenceWindow(sc.Cadence)
		tick, ok := DispatchTick(d.now(), sc.Cadence)
		if !ok {
			// An uninterpretable cadence (neither a known preset nor a parseable cron)
			// has no firing to dispatch. Create/edit refuses an invalid Custom cron, so
			// this only guards legacy or hand-edited rows; skip it rather than firing on
			// a wrong default.
			d.log.Printf("report dispatcher: schedule %d cadence %q is uninterpretable, skipped", sc.ID, sc.Cadence)
			continue
		}
		if err := d.dispatchOne(ctx, sc, tick, window); err != nil {
			d.log.Printf("report dispatcher: dispatch schedule %d: %v", sc.ID, err)
		}
	}
}

func (d *Dispatcher) dispatchOne(ctx context.Context, sc db.ReportSchedule, tick time.Time, window time.Duration) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := d.q.WithTx(tx)

	// The advisory lock serialises concurrent dispatches of the same schedule; the
	// partial-unique (schedule_id, scheduled_tick) key is the durable backstop.
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", sc.ID); err != nil {
		return err
	}

	no, err := qtx.NextReportDeliveryNo(ctx, sc.ID)
	if err != nil {
		return fmt.Errorf("report dispatcher: next delivery no: %w", err)
	}

	// Period bounds are deterministic per tick: the window ends at the tick and opens
	// one window before it, so every poll inside the window computes the same bounds.
	periodEnd := tick
	periodStart := tick.Add(-window)

	inserted, err := qtx.TryInsertScheduledDelivery(ctx, db.TryInsertScheduledDeliveryParams{
		ScheduleID:    sc.ID,
		PeriodStart:   tstz(periodStart),
		PeriodEnd:     tstz(periodEnd),
		DeliveryNo:    no,
		State:         "generated",
		DeliveredAt:   pgtype.Timestamptz{}, // generated, not delivered — the ready-message send is T7.
		ScheduledTick: tstz(tick),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// Overlapping tick: the window is already dispatched. Recorded by the existing
		// receipt; we render nothing and stamp nothing.
		d.log.Printf("report dispatcher: schedule %d tick %s already dispatched, skipped", sc.ID, tick.Format(time.RFC3339))
		return tx.Commit(ctx)
	}
	if err != nil {
		return fmt.Errorf("report dispatcher: try insert scheduled delivery: %w", err)
	}

	// Won the claim: cut the artifact for the window. The delivered document recomputes
	// from the period bounds at render time, so this render confirms the report is
	// cuttable; the receipt snapshots nothing. The canonical renderer draws the current
	// period, mirroring Run-now.
	_ = message.RenderArtifact(message.Artifact{
		Title:       sc.Name,
		PeriodStart: periodStart.Format("2006-01-02"),
		PeriodEnd:   periodEnd.Format("2006-01-02"),
		DeliveryNo:  int(no),
		GeneratedAt: d.now().UTC().Format("2006-01-02"),
		Format:      sc.Format,
	})

	// If the schedule binds a Channel, enqueue exactly ONE link-only ready-message for
	// this run, in the SAME transaction that stamped the receipt — so a won tick and its
	// notification are one atomic act (#508/T7). A download-only schedule (NULL
	// channel_id) binds nothing and enqueues nothing; the artifact simply stays viewable
	// in-instance. The receipt is left 'generated' — the notify runner flips it to
	// 'delivered' only once the Channel accepts the ready-message (ADR-0039).
	if shouldNotify(sc.ChannelID) {
		if err := qtx.InsertReportNotification(ctx, db.InsertReportNotificationParams{
			ReportDeliveryID: inserted.ID,
			ChannelID:        sc.ChannelID.Int64,
		}); err != nil {
			return fmt.Errorf("report dispatcher: enqueue notification: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	d.log.Printf("report dispatcher: schedule %d dispatched delivery #%d for tick %s", sc.ID, no, tick.Format(time.RFC3339))
	return nil
}

func tstz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
