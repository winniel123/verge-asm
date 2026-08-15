package retention

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func TestBelowFloor(t *testing.T) {
	cases := []struct {
		multiple int64
		below    bool
	}{
		{0, false}, // unbounded default — always allowed
		{1, true},  // one cadence is below the k=2 floor
		{2, false}, // exactly the floor
		{3, false}, // above the floor
		{100, false},
	}
	for _, c := range cases {
		if got := BelowFloor(c.multiple); got != c.below {
			t.Errorf("BelowFloor(%d) = %v, want %v", c.multiple, got, c.below)
		}
	}
	if FloorCadences != 2 {
		t.Fatalf("FloorCadences = %d, want 2 (the currency generation count k)", FloorCadences)
	}
}

func TestCutoff(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	const daily = int64(86400)

	if _, bounded := Cutoff(now, 0, daily); bounded {
		t.Error("multiple 0 must be unbounded")
	}
	if _, bounded := Cutoff(now, 3, 0); bounded {
		t.Error("no enabled Scan (cadence 0) must be unbounded")
	}

	cutoff, bounded := Cutoff(now, 3, daily)
	if !bounded {
		t.Fatal("multiple 3 over a daily Scan must be bounded")
	}
	want := now.Add(-3 * 24 * time.Hour)
	if !cutoff.Equal(want) {
		t.Errorf("Cutoff = %v, want %v (3 daily cadences back)", cutoff, want)
	}
}

// fakeStore is the whole surface the Retirer can reach. It has no Observation,
// Span or Batch method — the compiler will not let retention code touch measured
// data through it, which is the separation AC proved structurally.
type fakeStore struct {
	multiple      int64
	cadence       int64
	cadenceErr    error
	deleteCalled  bool
	deleteBefore  time.Time
	deletedReturn int64
}

func (f *fakeStore) GetRetentionSettings(context.Context) (db.GetRetentionSettingsRow, error) {
	return db.GetRetentionSettingsRow{DispatchCadenceMultiple: f.multiple}, nil
}

func (f *fakeStore) SlowestEnabledScanCadenceSeconds(context.Context) (int64, error) {
	return f.cadence, f.cadenceErr
}

func (f *fakeStore) DeleteExpiredDispatches(_ context.Context, before pgtype.Timestamptz) (int64, error) {
	f.deleteCalled = true
	f.deleteBefore = before.Time
	return f.deletedReturn, nil
}

func TestSweepUnboundedDeletesNothing(t *testing.T) {
	for _, tc := range []struct {
		name     string
		multiple int64
		cadence  int64
	}{
		{"dial at zero", 0, 86400},
		{"no enabled scan", 5, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeStore{multiple: tc.multiple, cadence: tc.cadence}
			r := NewRetirer(f, func() time.Time { return time.Now() }, nil)
			n, err := r.Sweep(context.Background())
			if err != nil {
				t.Fatalf("Sweep: %v", err)
			}
			if n != 0 {
				t.Errorf("retired %d rows, want 0 when unbounded", n)
			}
			if f.deleteCalled {
				t.Error("delete must not be called when retention is unbounded")
			}
		})
	}
}

func TestSweepBoundedDeletesAtCutoff(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	f := &fakeStore{multiple: 3, cadence: 86400, deletedReturn: 7}
	r := NewRetirer(f, func() time.Time { return now }, nil)

	n, err := r.Sweep(context.Background())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if n != 7 {
		t.Errorf("retired %d rows, want 7", n)
	}
	if !f.deleteCalled {
		t.Fatal("delete must be called when retention is bounded")
	}
	want := now.Add(-3 * 24 * time.Hour)
	if !f.deleteBefore.Equal(want) {
		t.Errorf("delete cutoff = %v, want %v", f.deleteBefore, want)
	}
}

func TestSweepPropagatesReadError(t *testing.T) {
	sentinel := errors.New("boom")
	f := &fakeStore{multiple: 3, cadenceErr: sentinel}
	r := NewRetirer(f, nil, nil)
	if _, err := r.Sweep(context.Background()); !errors.Is(err, sentinel) {
		t.Errorf("Sweep error = %v, want %v", err, sentinel)
	}
	if f.deleteCalled {
		t.Error("delete must not run after a read error")
	}
}
