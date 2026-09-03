// Package retention retires expired Dispatch rows — the one corpus a wall clock
// may retire (v1 spec §4.6, ADR-0041). Dispatch carries no observations and the
// comparison path is structurally barred from reading it, so a clock deleting it
// moves no value on any timeline.
//
// The retention window is an operator dial (#206's
// retention_settings.dispatch_cadence_multiple) stated as a multiple of the
// slowest enabled Scan's cadence, not a day count. Because the dial and its
// floor are both counted in that same cadence, the floor is unit-free: a value
// of 0 is the unbounded v1 default and any positive value below k is rejected
// (see BelowFloor). Nothing here reads or writes the Observation, Span or Batch
// corpus — the Store this package depends on exposes no method that could, which
// is the structural half of the separation ACs.
package retention

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// FloorCadences is k: the number of the slowest enabled Scan's cadences below
// which the Dispatch window may not be set. Below k cadences, Coverage cannot
// answer whether the slowest scan ran (v1 spec §4.6). k is the model's currency
// generation count, fixed at 2 (ADR-0044, ADR-0028).
const FloorCadences int64 = 2

// BelowFloor reports whether a Dispatch dial value is under the floor. The dial
// is a multiple of the slowest enabled Scan's cadence and the floor is k of that
// same cadence, so the comparison needs no cadence at all: 0 is the unbounded
// default and always allowed; any positive value under k is rejected. Stating
// the dial as a multiple rather than a day count is precisely what makes this
// floor independent of which Scan is slowest.
func BelowFloor(multiple int64) bool {
	return multiple > 0 && multiple < FloorCadences
}

func Cutoff(now time.Time, multiple, slowestCadenceSeconds int64) (cutoff time.Time, bounded bool) {
	if multiple <= 0 || slowestCadenceSeconds <= 0 {
		return time.Time{}, false
	}
	window := time.Duration(multiple*slowestCadenceSeconds) * time.Second
	return now.Add(-window), true
}

// Store is the narrow slice of the data layer the Retirer needs. It is
// deliberately the whole of what retention may touch: a settings read, the
// slowest enabled Scan's cadence, and the Dispatch-only delete. It exposes no
// Observation, Span or Batch method, so no code path through the Retirer can
// reach measured data — the separation is enforced by the interface, not by
// discipline. *db.Queries satisfies it.
type Store interface {
	GetRetentionSettings(ctx context.Context) (db.GetRetentionSettingsRow, error)
	SlowestEnabledScanCadenceSeconds(ctx context.Context) (int64, error)
	DeleteExpiredDispatches(ctx context.Context, before pgtype.Timestamptz) (int64, error)
}

type Retirer struct {
	store Store
	now   func() time.Time
	log   *log.Logger
}

// NewRetirer builds a Retirer over store. now is injectable so tests and manual
// runs can control the sweep instant.
func NewRetirer(store Store, now func() time.Time, logger *log.Logger) *Retirer {
	if now == nil {
		now = time.Now
	}
	return &Retirer{store: store, now: now, log: logger}
}

// Sweep retires every Dispatch row scheduled before the operator's window and
// returns how many it deleted. When retention is unbounded it deletes nothing
// and returns 0 — the v1 default. It is the only code path that deletes Dispatch
// rows, and it can reach nothing else: Store has no Observation, Span or Batch
// method, so a bug here can delete Dispatch rows and only Dispatch rows.
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
	return r.store.DeleteExpiredDispatches(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true})
}

// Run sweeps once at start and then every interval until ctx is done. It never
// returns an error of its own beyond the context's: a failed sweep is logged and
// the loop continues, since a transient delete failure is retried on the next
// tick and Dispatch retention is never on the measurement path.
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
