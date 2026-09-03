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

type Dispatcher struct {
	pool *pgxpool.Pool
	q    *db.Queries
	now  func() time.Time
	log  *log.Logger
}

func NewDispatcher(pool *pgxpool.Pool, now func() time.Time, logger *log.Logger) *Dispatcher {
	if now == nil {
		now = time.Now
	}
	return &Dispatcher{pool: pool, q: db.New(pool), now: now, log: logger}
}

func (d *Dispatcher) Run(ctx context.Context) error {
	// A second poll inside one window is a recorded skip, so a minute ticker is safe.
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
		window := CadenceWindow(sc.Cadence)
		tick, ok := DispatchTick(d.now(), sc.Cadence)
		if !ok {
			// A legacy or hand-edited row is skipped rather than fired on a wrong default (ADR-0122).
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

	// The durable backstop is the partial-unique (schedule_id, scheduled_tick) key, not this lock.
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", sc.ID); err != nil {
		return err
	}

	no, err := qtx.NextReportDeliveryNo(ctx, sc.ID)
	if err != nil {
		return fmt.Errorf("report dispatcher: next delivery no: %w", err)
	}

	periodEnd := tick
	periodStart := tick.Add(-window)

	inserted, err := qtx.TryInsertScheduledDelivery(ctx, db.TryInsertScheduledDeliveryParams{
		ScheduleID:    sc.ID,
		PeriodStart:   tstz(periodStart),
		PeriodEnd:     tstz(periodEnd),
		DeliveryNo:    no,
		State:         "generated",
		DeliveredAt:   pgtype.Timestamptz{},
		ScheduledTick: tstz(tick),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		d.log.Printf("report dispatcher: schedule %d tick %s already dispatched, skipped", sc.ID, tick.Format(time.RFC3339))
		return tx.Commit(ctx)
	}
	if err != nil {
		return fmt.Errorf("report dispatcher: try insert scheduled delivery: %w", err)
	}

	// Discarded on purpose: the artifact recomputes at render time, so nothing is snapshotted.
	_ = message.RenderArtifact(message.Artifact{
		Title:       sc.Name,
		PeriodStart: periodStart.Format("2006-01-02"),
		PeriodEnd:   periodEnd.Format("2006-01-02"),
		DeliveryNo:  int(no),
		GeneratedAt: d.now().UTC().Format("2006-01-02"),
		Format:      sc.Format,
	})

	// The enqueue rides the receipt's transaction, so a won tick and its notice are one act (#508).
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
