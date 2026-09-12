// Package retention retires expired rows on three disciplines: Dispatch and transcript
// by wall clock, since no derivation reads either; observation by what may still read
// it (v1 spec §4.6, ADR-0041, ADR-0094, ADR-0126). A pending release reads one hot
// Dispatch, so the Dispatch sweep exempts the row it reads (ADR-1806 §3, #1853).
package retention

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

const FloorCadences int64 = 2 // k, the model's currency generation count (ADR-0044, ADR-0028).

func BelowFloor(multiple int64) bool {
	// Below k cadences Coverage cannot answer whether the slowest Scan ran (v1 spec §4.6).
	return multiple > 0 && multiple < FloorCadences
}

func Cutoff(now time.Time, multiple, slowestCadenceSeconds int64) (cutoff time.Time, bounded bool) {
	// Dial 0 is the v1 unbounded default; with no enabled Scan there is no cadence to count.
	if multiple <= 0 || slowestCadenceSeconds <= 0 {
		return time.Time{}, false
	}
	// The dial is a multiple of the cadence, not a day count, so the floor is unit-free.
	window := time.Duration(multiple*slowestCadenceSeconds) * time.Second
	return now.Add(-window), true
}

// Each Retirer's Store is narrow so the corpus separation holds by type, not discipline.

type Store interface {
	GetRetentionSettings(ctx context.Context) (db.GetRetentionSettingsRow, error)
	SlowestEnabledScanCadenceSeconds(ctx context.Context) (int64, error)
	ListDispatchesAPendingReleaseMayRead(ctx context.Context, before pgtype.Timestamptz) ([]int64, error)
	DeleteExpiredDispatches(ctx context.Context, arg db.DeleteExpiredDispatchesParams) (int64, error)
}

type Retirer struct {
	store Store
	now   func() time.Time
	log   *log.Logger
}

func NewRetirer(store Store, now func() time.Time, logger *log.Logger) *Retirer {
	if now == nil {
		now = time.Now
	}
	return &Retirer{store: store, now: now, log: logger}
}

func (r *Retirer) Sweep(ctx context.Context) (int64, error) {
	settings, err := r.store.GetRetentionSettings(ctx)
	if err != nil {
		return 0, err
	}
	cadence, err := r.store.SlowestEnabledScanCadenceSeconds(ctx)
	if err != nil {
		return 0, err
	}
	cutoff, bounded := Cutoff(r.now().UTC(), settings.DispatchCadenceMultiple, cadence)
	if !bounded {
		return 0, nil
	}
	before := pgtype.Timestamptz{Time: cutoff, Valid: true}
	// ADR-1806 §3 hung a product-visible message on this corpus, so its read set survives (#1853).
	stillRead, err := r.store.ListDispatchesAPendingReleaseMayRead(ctx, before)
	if err != nil {
		return 0, err
	}
	return r.store.DeleteExpiredDispatches(ctx, db.DeleteExpiredDispatchesParams{
		Before:    before,
		StillRead: stillRead,
	})
}

func (r *Retirer) Run(ctx context.Context, interval time.Duration) error {
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

func (r *Retirer) sweepAndLog(ctx context.Context) {
	n, err := r.Sweep(ctx)
	// The next tick retries the sweep, so a failed pass logs and the loop continues (ADR-0141).
	if err != nil {
		if r.log != nil {
			r.log.Printf("retention: dispatch sweep: %v", err)
		}
		return
	}
	if n > 0 && r.log != nil {
		r.log.Printf("retention: retired %d expired dispatch row(s)", n)
	}
}
