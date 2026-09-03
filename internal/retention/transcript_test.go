package retention

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func TestTranscriptWindowDays(t *testing.T) {
	cases := []struct {
		dial    int64
		days    int64
		bounded bool
	}{
		{0, 0, false},
		{-3, 0, false},
		{1, 1, true},
		{14, 14, true},
		{90, 90, true},
	}
	for _, c := range cases {
		days, bounded := TranscriptWindowDays(c.dial)
		if days != c.days || bounded != c.bounded {
			t.Errorf("TranscriptWindowDays(%d) = (%d, %v), want (%d, %v)",
				c.dial, days, bounded, c.days, c.bounded)
		}
	}
	if TranscriptFloorDays != 1 {
		t.Fatalf("TranscriptFloorDays = %d, want 1", TranscriptFloorDays)
	}
}

func TestTranscriptCutoff(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	if _, bounded := TranscriptCutoff(now, 0); bounded {
		t.Error("dial 0 must be unbounded")
	}

	cutoff, bounded := TranscriptCutoff(now, 14)
	if !bounded {
		t.Fatal("dial 14 must be bounded")
	}
	want := now.Add(-14 * 24 * time.Hour)
	if !cutoff.Equal(want) {
		t.Errorf("TranscriptCutoff = %v, want %v (14 days back)", cutoff, want)
	}
}

type fakeTranscriptStore struct {
	dial          int64
	deleteCalled  bool
	deleteBefore  time.Time
	deletedReturn int64
	getErr        error
}

func (f *fakeTranscriptStore) GetRetentionSettings(context.Context) (db.GetRetentionSettingsRow, error) {
	if f.getErr != nil {
		return db.GetRetentionSettingsRow{}, f.getErr
	}
	return db.GetRetentionSettingsRow{TranscriptCurrencyDays: f.dial}, nil
}

func (f *fakeTranscriptStore) DeleteExpiredTranscripts(_ context.Context, before pgtype.Timestamptz) (int64, error) {
	f.deleteCalled = true
	f.deleteBefore = before.Time
	return f.deletedReturn, nil
}

func TestTranscriptSweepUnboundedDeletesNothing(t *testing.T) {
	f := &fakeTranscriptStore{dial: 0}
	r := NewTranscriptRetirer(f, func() time.Time { return time.Now() }, nil)
	n, err := r.Sweep(context.Background())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if n != 0 {
		t.Errorf("retired %d rows, want 0 when the dial is 0", n)
	}
	if f.deleteCalled {
		t.Error("delete must not be called when the dial is 0")
	}
}

func TestTranscriptSweepBoundedDeletesAtCutoff(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	f := &fakeTranscriptStore{dial: 14, deletedReturn: 5}
	r := NewTranscriptRetirer(f, func() time.Time { return now }, nil)

	n, err := r.Sweep(context.Background())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if n != 5 {
		t.Errorf("retired %d rows, want 5", n)
	}
	if !f.deleteCalled {
		t.Fatal("delete must be called when the dial is bounded")
	}
	want := now.Add(-14 * 24 * time.Hour)
	if !f.deleteBefore.Equal(want) {
		t.Errorf("delete cutoff = %v, want %v", f.deleteBefore, want)
	}
}

func TestTranscriptSweepFloorsPositiveBelowOne(t *testing.T) {
	// The dial column stores whole days, so a sub-floor value never reaches the sweep in production.
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	cutoff, bounded := TranscriptCutoff(now, TranscriptFloorDays)
	if !bounded {
		t.Fatal("the floor value must be bounded")
	}
	if !cutoff.Equal(now.Add(-24 * time.Hour)) {
		t.Errorf("floored cutoff = %v, want one day back", cutoff)
	}
}

func TestTranscriptSweepPropagatesReadError(t *testing.T) {
	sentinel := errors.New("boom")
	f := &fakeTranscriptStore{getErr: sentinel}
	r := NewTranscriptRetirer(f, nil, nil)
	if _, err := r.Sweep(context.Background()); !errors.Is(err, sentinel) {
		t.Errorf("Sweep error = %v, want %v", err, sentinel)
	}
	if f.deleteCalled {
		t.Error("delete must not run after a read error")
	}
}
