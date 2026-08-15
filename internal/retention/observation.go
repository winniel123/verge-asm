package retention

// Observation retention is the two-tier rule of v1 spec §4.6 (ADR-0041,
// ADR-0094), landed beside the Dispatch path in this same package but on its own
// structurally-separate discipline. Where Dispatch is retired by a plain wall
// clock, an Observation is retained by what may still READ it:
//
//   - Live — within k cadences of the TIGHTEST Scan covering its timeline — is
//     what every derivation reads and may NEVER be discarded.
//   - Evidential — past that bound — is read by no derivation and re-derives no
//     history, so discarding it moves no value on any timeline; it is kept only
//     for a person asking "what did we actually measure".
//
// The boundary is computed PER TIMELINE from that timeline's covering Scan's
// cadence, never globally, and keyed on the whole timeline tuple
// (subject, facet, discriminator, vantage, source). A row survives while its age
// is inside EITHER its own bound OR the operator's dial, whichever is longer: the
// control collapses to one number, the query never does (ADR-0094). Two
// populations fall opposite ways — a timeline no enabled Scan covers has an
// UNDEFINED bound and is never retired; a WITHDRAWN subject's timelines carry no
// floor at all and the dial alone governs them.
//
// Everything here is a pure function plus a thin Retirer over a delete-only Store,
// so the two-tier boundary, the tightest-bound floor and the two exceptions are
// provable without a database (observation_test.go), and the row-level decision is
// the same one the SQL deletion query encodes (db/queries/retention.sql).

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// SecondsPerDay converts the operator's day-stated observation dial into the
// seconds the per-timeline bounds are counted in. The dial is a whole number of
// days (retention_settings.observation_currency_days); a bound is k cadences, and
// a cadence is seconds.
const SecondsPerDay int64 = 86400

// ObservationBoundSeconds is a timeline's live/evidential boundary: k cadences of
// the tightest Scan covering it (FloorCadences × the cadence). hasBound is false
// when no enabled Scan covers the timeline — an UNDEFINED bound, which is never
// expired (never retired), not a loose one: undefined is not the same as very
// large. k is the same FloorCadences the Dispatch floor uses.
func ObservationBoundSeconds(tightestCoveringCadenceSeconds int64) (bound int64, hasBound bool) {
	if tightestCoveringCadenceSeconds <= 0 {
		return 0, false
	}
	return FloorCadences * tightestCoveringCadenceSeconds, true
}

// ObservationFloorDays is the tightest bound in force expressed in whole days: the
// least value the observation dial may take and still change a row. The tightest
// bound is k cadences of the tightest ENABLED Scan (the smallest cadence), because
// that is the smallest per-timeline bound any in-force timeline can carry — set
// the dial below it and every row's own bound already exceeds it, so the dial
// changes nothing (that is the derivation of the floor, never an operator choice;
// ADR-0094). hasFloor is false when no Scan is enabled: with no bound in force
// there is nothing to floor against. A fractional day rounds UP so the floor never
// lands inside the live tier.
func ObservationFloorDays(tightestEnabledCadenceSeconds int64) (days int64, hasFloor bool) {
	bound, ok := ObservationBoundSeconds(tightestEnabledCadenceSeconds)
	if !ok {
		return 0, false
	}
	days = (bound + SecondsPerDay - 1) / SecondsPerDay
	return days, true
}

// BelowObservationFloor reports whether a day-stated observation dial is under the
// tightest bound in force. 0 is the unbounded default and always allowed; any
// positive value below the floor is rejected, because below the tightest bound the
// control changes no row at all — the dial collapses to one number while the query
// still reads each row's own bound (ADR-0094). It mirrors BelowFloor on the
// Dispatch side; the difference is only which cadence the floor rests on (tightest
// here, slowest there). With no bound in force (no enabled Scan) nothing is
// floored and every non-negative dial is allowed.
func BelowObservationFloor(dialDays, tightestEnabledCadenceSeconds int64) bool {
	floor, ok := ObservationFloorDays(tightestEnabledCadenceSeconds)
	if !ok {
		return false
	}
	return dialDays > 0 && dialDays < floor
}

// Tier is an observation row's readability tier — what may still read it, never
// its age (ADR-0041).
type Tier int

const (
	// Live: within k cadences of the tightest Scan covering the timeline. Every
	// derivation reads it and it may never be discarded.
	Live Tier = iota
	// Evidential: past the bound (or on a timeline no enabled Scan covers). No
	// derivation may read it or re-derive history from it; it is retained only as
	// a record of what was measured.
	Evidential
)

// TierOf classifies a row by its age against its own timeline bound. A row with an
// undefined bound (hasBound false — no enabled Scan covers it) is Evidential:
// nothing is "within k cadences of a covering Scan" when no Scan covers it, so no
// derivation may read it — it is retained purely as evidence.
func TierOf(ageSeconds, boundSeconds int64, hasBound bool) Tier {
	if hasBound && ageSeconds <= boundSeconds {
		return Live
	}
	return Evidential
}

// RetainObservation reports whether an observation row survives a retention sweep.
// It is the row-level rule the deletion query encodes in SQL, stated here as a
// pure function so the two-tier boundary, the tightest-floor collapse and the two
// exception populations are provable without a database.
//
//	ageSeconds   now − observed_at, the row's age.
//	boundSeconds k cadences of the tightest Scan covering the row's timeline.
//	dialSeconds  the operator's dial in seconds; 0 == unbounded (the v1 default).
//	hasBound     false where no enabled Scan covers the timeline (undefined bound).
//	withdrawn    the row's subject has left the estate — its timelines are closed.
//
// A row is retained while its age is inside EITHER its own bound OR the dial,
// whichever is longer. Two populations fall outside that rule and opposite ways:
// an undefined bound is never expired (never retired); a withdrawn subject carries
// no floor at all, so the dial alone governs it.
func RetainObservation(ageSeconds, boundSeconds, dialSeconds int64, hasBound, withdrawn bool) bool {
	if withdrawn {
		// No floor: the row's own bound does not protect it, the dial alone governs.
		if dialSeconds <= 0 {
			return true // unbounded dial keeps everything
		}
		return ageSeconds <= dialSeconds
	}
	if !hasBound {
		return true // undefined bound is not expired — never retired
	}
	if ageSeconds <= boundSeconds {
		return true // live tier — never discarded
	}
	// Evidential tier: kept only while the operator's dial keeps it.
	if dialSeconds <= 0 {
		return true // unbounded default — the raw corpus grows without bound
	}
	return ageSeconds <= dialSeconds
}

// AgedObservation is the minimum a tier decision needs about one row: its identity,
// how old it is, and the bound of the timeline it sits on. A derivation folds these
// through LiveOnly, which is why a derivation cannot reach an evidential row.
type AgedObservation struct {
	ID           int64
	AgeSeconds   int64
	BoundSeconds int64
	HasBound     bool
}

// LiveOnly returns the live-tier subset of rows — the rows every derivation reads
// and the only rows any derivation may read. An evidential row (age past its own
// bound, or on a timeline no enabled Scan covers) is dropped here and is therefore
// unreadable by any derivation that reads through this gate. Proving that drop is
// the readability-separation AC (ADR-0041); the SQL half is
// ListLiveObservationsForDerivation, which filters by each row's own bound in the
// database.
func LiveOnly(rows []AgedObservation) []AgedObservation {
	out := make([]AgedObservation, 0, len(rows))
	for _, r := range rows {
		if TierOf(r.AgeSeconds, r.BoundSeconds, r.HasBound) == Live {
			out = append(out, r)
		}
	}
	return out
}

// ObservationStore is the narrow slice of the data layer the ObservationRetirer
// needs: the operator's dial and the observation-only delete. It deliberately
// exposes no read of measured data and no write other than the retiring delete —
// the Retirer can only settle the dial and delete evidential observations, never
// move a value. *db.Queries satisfies it.
type ObservationStore interface {
	GetRetentionSettings(ctx context.Context) (db.GetRetentionSettingsRow, error)
	DeleteExpiredObservations(ctx context.Context, arg db.DeleteExpiredObservationsParams) (int64, error)
}

// ObservationRetirer sweeps evidential observations the operator's dial no longer
// keeps. It never touches a live row: the delete query evaluates each row's own
// per-timeline bound and reaches only rows past BOTH that bound and the dial (or,
// for a withdrawn subject, past the dial alone). It is landed beside the Dispatch
// Retirer, not folded into it.
type ObservationRetirer struct {
	store ObservationStore
	now   func() time.Time
	log   *log.Logger
}

// NewObservationRetirer builds an ObservationRetirer over store. now is injectable
// so tests and manual runs can control the sweep instant.
func NewObservationRetirer(store ObservationStore, now func() time.Time, logger *log.Logger) *ObservationRetirer {
	if now == nil {
		now = time.Now
	}
	return &ObservationRetirer{store: store, now: now, log: logger}
}

// Sweep retires the evidential observations the dial no longer keeps and returns
// how many it deleted. When the dial is at 0 it deletes nothing and returns 0 —
// the v1 default keeps the raw corpus growing without bound; the operator opts
// into retiring evidence by setting the dial (floored at the tightest bound in the
// Settings screen). The delete evaluates each row's own bound, so a live row is
// never in the delete set no matter how large the dial.
func (r *ObservationRetirer) Sweep(ctx context.Context) (int64, error) {
	settings, err := r.store.GetRetentionSettings(ctx)
	if err != nil {
		return 0, err
	}
	dialSeconds := settings.ObservationCurrencyDays * SecondsPerDay
	if dialSeconds <= 0 {
		return 0, nil
	}
	return r.store.DeleteExpiredObservations(ctx, db.DeleteExpiredObservationsParams{
		DialSeconds:   dialSeconds,
		FloorCadences: FloorCadences,
		AsOf:          pgtype.Timestamptz{Time: r.now().UTC(), Valid: true},
	})
}

// Run sweeps once at start and then every interval until ctx is done. Like the
// Dispatch Retirer it never returns an error of its own beyond the context's: a
// failed sweep is logged and the loop continues, since a transient delete failure
// is retried on the next tick and evidential retirement is never on the
// measurement path.
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
