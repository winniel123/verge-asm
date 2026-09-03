package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestStaleCutoff(t *testing.T) {
	now := time.Date(2026, 8, 29, 20, 50, 45, 0, time.UTC)

	if _, bounded := StaleCutoff(now, 0); bounded {
		t.Error("a zero threshold must disable the reaper (unbounded)")
	}
	if _, bounded := StaleCutoff(now, -5*time.Minute); bounded {
		t.Error("a negative threshold must disable the reaper (unbounded)")
	}

	cutoff, bounded := StaleCutoff(now, DefaultStaleJobThreshold)
	if !bounded {
		t.Fatal("a positive threshold must be bounded")
	}
	if want := now.Add(-DefaultStaleJobThreshold); !cutoff.Equal(want) {
		t.Errorf("StaleCutoff = %v, want %v (threshold back)", cutoff, want)
	}
}

// The reaper threshold must sit above the probe timeout, or a job that is
// legitimately mid-probe could be reaped as dead while its probe is still running.
func TestStaleThresholdExceedsProbeTimeout(t *testing.T) {
	if DefaultStaleJobThreshold <= DefaultProbeTimeout {
		t.Fatalf("DefaultStaleJobThreshold (%s) must exceed DefaultProbeTimeout (%s), else a "+
			"legitimately-running probe could be reaped as stale", DefaultStaleJobThreshold, DefaultProbeTimeout)
	}
}

type fakeReaperStore struct {
	reapCalled bool
	reapCutoff time.Time
	reapReturn int64
	reapErr    error
}

func (f *fakeReaperStore) ReapStaleRunningJobs(_ context.Context, cutoff pgtype.Timestamptz) (int64, error) {
	f.reapCalled = true
	f.reapCutoff = cutoff.Time
	return f.reapReturn, f.reapErr
}

func TestReaperSweepDisabledReclaimsNothing(t *testing.T) {
	f := &fakeReaperStore{reapReturn: 9}
	r := NewReaper(f, 0, func() time.Time { return time.Now() }, nil)
	n, err := r.Sweep(context.Background())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if n != 0 {
		t.Errorf("reclaimed %d rows, want 0 when the threshold disables the reaper", n)
	}
	if f.reapCalled {
		t.Error("the sweep must not run when the threshold is not positive")
	}
}

func TestReaperSweepReclaimsAtCutoff(t *testing.T) {
	now := time.Date(2026, 8, 29, 21, 50, 45, 0, time.UTC)
	f := &fakeReaperStore{reapReturn: 3}
	r := NewReaper(f, 15*time.Minute, func() time.Time { return now }, nil)

	n, err := r.Sweep(context.Background())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if n != 3 {
		t.Errorf("reclaimed %d rows, want 3", n)
	}
	if !f.reapCalled {
		t.Fatal("the sweep must run when the threshold is positive")
	}
	if want := now.Add(-15 * time.Minute); !f.reapCutoff.Equal(want) {
		t.Errorf("reap cutoff = %v, want %v (15m back)", f.reapCutoff, want)
	}
}

func TestReaperSweepPropagatesError(t *testing.T) {
	sentinel := errors.New("boom")
	f := &fakeReaperStore{reapErr: sentinel}
	r := NewReaper(f, 15*time.Minute, nil, nil)
	if _, err := r.Sweep(context.Background()); !errors.Is(err, sentinel) {
		t.Errorf("Sweep error = %v, want %v", err, sentinel)
	}
}
