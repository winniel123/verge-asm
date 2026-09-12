package retention

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// A model of the two statements answers for a held row whose first hot dispatch the dial
// expired, which no unit test can reach through a fake that only records its arguments.

type modelDispatch struct {
	id        int64
	scheduled time.Time
	created   time.Time
	complete  bool
	abandoned bool
}

// Each pending instant is one held row's batch, the instant its bound counts from (ADR-1806 §3).

type sweepModel struct {
	dispatches []modelDispatch
	pending    []time.Time
	deleted    []int64
}

func (m *sweepModel) GetRetentionSettings(context.Context) (db.GetRetentionSettingsRow, error) {
	return db.GetRetentionSettingsRow{DispatchCadenceMultiple: 2}, nil
}

func (m *sweepModel) SlowestEnabledScanCadenceSeconds(context.Context) (int64, error) {
	return 86400, nil
}

func (m *sweepModel) ListDispatchesAPendingReleaseMayRead(_ context.Context, before pgtype.Timestamptz) ([]int64, error) {
	var out []int64
	for _, at := range m.pending {
		if !at.Before(before.Time) {
			continue
		}
		for _, id := range []int64{m.pick(at, false), m.pick(at, true)} {
			if id != 0 {
				out = append(out, id)
			}
		}
	}
	return out, nil
}

func (m *sweepModel) pick(at time.Time, finished bool) int64 {
	var best modelDispatch
	for _, d := range m.dispatches {
		if d.abandoned || d.created.Before(at) || (finished && !d.complete) {
			continue
		}
		if best.id == 0 || d.created.Before(best.created) {
			best = d
		}
	}
	return best.id
}

func (m *sweepModel) DeleteExpiredDispatches(_ context.Context, arg db.DeleteExpiredDispatchesParams) (int64, error) {
	exempt := map[int64]bool{}
	for _, id := range arg.StillRead {
		exempt[id] = true
	}
	var retired int64
	for _, d := range m.dispatches {
		if exempt[d.id] || !d.scheduled.Before(arg.Before.Time) || !d.created.Before(arg.Before.Time) {
			continue
		}
		m.deleted = append(m.deleted, d.id)
		retired++
	}
	return retired, nil
}

func (m *sweepModel) sweep(t *testing.T, now time.Time) {
	t.Helper()
	if _, err := NewRetirer(m, func() time.Time { return now }, nil).Sweep(context.Background()); err != nil {
		t.Fatalf("Sweep: %v", err)
	}
}

func (m *sweepModel) survives(id int64) bool {
	for _, gone := range m.deleted {
		if gone == id {
			return false
		}
	}
	return true
}

// The held row's first hot dispatch is older than the cutoff, so the dial retired it and both
// predicates then read no dispatch at all: a message held forever, and a fold that never
// settles (#1853).

func TestAnArmedDialLeavesTheHeldRowsOwnDispatchStanding(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	held := now.Add(-10 * 24 * time.Hour)
	m := &sweepModel{
		pending: []time.Time{held},
		dispatches: []modelDispatch{
			{id: 1, scheduled: held.Add(-time.Hour), created: held.Add(-time.Hour), complete: true},
			{id: 2, scheduled: held.Add(time.Hour), created: held.Add(time.Hour), complete: true},
			{id: 3, scheduled: held.Add(25 * time.Hour), created: held.Add(25 * time.Hour), complete: true},
		},
	}

	m.sweep(t, now)

	if !m.survives(2) {
		t.Error("the held row's first hot dispatch must survive an armed dial — the row releases on it (#1853)")
	}
	for _, id := range []int64{1, 3} {
		if m.survives(id) {
			t.Errorf("dispatch %d is read by no pending release, so the dial retires it (ADR-0041)", id)
		}
	}
}

// The bound skips an abandoned row for the next live one, so exempting the abandoned row would
// keep what no predicate reads and retire what one does (ADR-1851 §3).

func TestAnAbandonedFanOutIsNotTheRowThePredicateReads(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	held := now.Add(-10 * 24 * time.Hour)
	m := &sweepModel{
		pending: []time.Time{held},
		dispatches: []modelDispatch{
			{id: 1, scheduled: held.Add(time.Hour), created: held.Add(time.Hour), abandoned: true},
			{id: 2, scheduled: held.Add(2 * time.Hour), created: held.Add(2 * time.Hour), complete: true},
		},
	}

	m.sweep(t, now)

	if m.survives(1) {
		t.Error("an abandoned fan-out is skipped by the bound, so the dial retires it (ADR-1851 §3)")
	}
	if !m.survives(2) {
		t.Error("the bound moved to the next live dispatch, so that one must survive (ADR-1851 §3)")
	}
}

// The exemption costs one row per held row, and a settled estate pays none of it (ADR-0041).

func TestWithNothingHeldTheDialRetiresEveryExpiredDispatch(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	m := &sweepModel{dispatches: []modelDispatch{
		{id: 1, scheduled: now.Add(-9 * 24 * time.Hour), created: now.Add(-9 * 24 * time.Hour), complete: true},
		{id: 2, scheduled: now.Add(-8 * 24 * time.Hour), created: now.Add(-8 * 24 * time.Hour), complete: true},
	}}

	m.sweep(t, now)

	for _, id := range []int64{1, 2} {
		if m.survives(id) {
			t.Errorf("dispatch %d is expired and read by nothing, so the dial retires it (ADR-0041)", id)
		}
	}
}

// A dispatch claiming a stale tick is created after the exempt read, so no read could have
// exempted it, and a row held in that instant bounds on it (#1853).

func TestALateFanOutSurvivesTheTickItMissed(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	m := &sweepModel{
		// The exempt read drops this instant, and the delete spares the row it bounds on.
		pending: []time.Time{now.Add(-2 * time.Minute)},
		dispatches: []modelDispatch{
			{id: 1, scheduled: now.Add(-9 * 24 * time.Hour), created: now.Add(-time.Minute), complete: true},
		},
	}

	m.sweep(t, now)

	if !m.survives(1) {
		t.Error("a fan-out created after the exempt read must survive the pass that read (#1853)")
	}
}

// A tick retires an unfinished pick, and the bound then moves to the first finished row. Retiring
// that row defers the release by a cadence, so the exempt set carries it too (ADR-1851 §3).

func TestTheRowAnUnfinishedPickMovesToSurvivesAsWell(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	held := now.Add(-10 * 24 * time.Hour)
	m := &sweepModel{
		pending: []time.Time{held},
		dispatches: []modelDispatch{
			{id: 1, scheduled: held.Add(time.Hour), created: held.Add(time.Hour)},
			{id: 2, scheduled: held.Add(2 * time.Hour), created: held.Add(2 * time.Hour), complete: true},
		},
	}

	m.sweep(t, now)

	if !m.survives(1) {
		t.Error("the bound's own pick must survive, unfinished or not (ADR-1806 §3)")
	}
	if !m.survives(2) {
		t.Error("a tick may retire the pick, so the row the bound moves to must survive (ADR-1851 §3)")
	}
}
