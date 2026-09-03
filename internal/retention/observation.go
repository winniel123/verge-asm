package retention

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

const SecondsPerDay int64 = 86400

func ObservationBoundSeconds(tightestCoveringCadenceSeconds int64) (bound int64, hasBound bool) {
	if tightestCoveringCadenceSeconds <= 0 {
		return 0, false
	}
	return FloorCadences * tightestCoveringCadenceSeconds, true
}

func ObservationFloorDays(tightestEnabledCadenceSeconds int64) (days int64, hasFloor bool) {
	// Below the tightest in-force bound the dial changes no row, so the floor is derived (ADR-0094).
	bound, ok := ObservationBoundSeconds(tightestEnabledCadenceSeconds)
	if !ok {
		return 0, false
	}
	// Rounding up keeps the floor out of the live tier.
	days = (bound + SecondsPerDay - 1) / SecondsPerDay
	return days, true
}

func BelowObservationFloor(dialDays, tightestEnabledCadenceSeconds int64) bool {
	floor, ok := ObservationFloorDays(tightestEnabledCadenceSeconds)
	if !ok {
		return false
	}
	return dialDays > 0 && dialDays < floor
}

// A tier names what may still read a row, never how old it is (ADR-0041).

type Tier int

const (
	Live Tier = iota
	Evidential
)

func TierOf(ageSeconds, boundSeconds int64, hasBound bool) Tier {
	if hasBound && ageSeconds <= boundSeconds {
		return Live
	}
	return Evidential
}

func RetainObservation(ageSeconds, boundSeconds, dialSeconds int64, hasBound, withdrawn bool) bool {
	// The SQL deletion query encodes this same row-level rule (db/queries/retention.sql).
	if withdrawn {
		// A withdrawn subject's timelines are closed, so nothing floors them (ADR-0094).
		if dialSeconds <= 0 {
			return true
		}
		return ageSeconds <= dialSeconds
	}
	// An undefined bound is not an expired one, so an uncovered timeline is never retired.
	if !hasBound {
		return true
	}
	// The live tier is what every derivation reads and may never be discarded (ADR-0041).
	if ageSeconds <= boundSeconds {
		return true
	}
	if dialSeconds <= 0 {
		return true
	}
	return ageSeconds <= dialSeconds
}

type AgedObservation struct {
	ID           int64
	AgeSeconds   int64
	BoundSeconds int64
	HasBound     bool
}

func LiveOnly(rows []AgedObservation) []AgedObservation {
	// The read gate, not the sweep, keeps an evidential row out of a derivation (#237).
	out := make([]AgedObservation, 0, len(rows))
	// The SQL twin ListLiveObservationsForDerivation must move with this gate.
	for _, r := range rows {
		if TierOf(r.AgeSeconds, r.BoundSeconds, r.HasBound) == Live {
			out = append(out, r)
		}
	}
	return out
}

type ObservationStore interface {
	GetRetentionSettings(ctx context.Context) (db.GetRetentionSettingsRow, error)
	DeleteExpiredObservations(ctx context.Context, arg db.DeleteExpiredObservationsParams) (int64, error)
}

type ObservationRetirer struct {
	store ObservationStore
	now   func() time.Time
	log   *log.Logger
}

func NewObservationRetirer(store ObservationStore, now func() time.Time, logger *log.Logger) *ObservationRetirer {
	if now == nil {
		now = time.Now
	}
	return &ObservationRetirer{store: store, now: now, log: logger}
}

func (r *ObservationRetirer) Sweep(ctx context.Context) (int64, error) {
	settings, err := r.store.GetRetentionSettings(ctx)
	if err != nil {
		return 0, err
	}
	dialSeconds := settings.ObservationCurrencyDays * SecondsPerDay
	if dialSeconds <= 0 {
		return 0, nil
	}
	// The delete evaluates each row's own bound, so no dial size can reach a live row.
	return r.store.DeleteExpiredObservations(ctx, db.DeleteExpiredObservationsParams{
		DialSeconds:   dialSeconds,
		FloorCadences: FloorCadences,
		AsOf:          pgtype.Timestamptz{Time: r.now().UTC(), Valid: true},
	})
}

func (r *ObservationRetirer) Run(ctx context.Context, interval time.Duration) error {
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

func (r *ObservationRetirer) sweepAndLog(ctx context.Context) {
	n, err := r.Sweep(ctx)
	if err != nil {
		if r.log != nil {
			r.log.Printf("retention: observation sweep: %v", err)
		}
		return
	}
	if n > 0 && r.log != nil {
		r.log.Printf("retention: retired %d evidential observation row(s)", n)
	}
}
