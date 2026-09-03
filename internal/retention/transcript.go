package retention

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// Nothing derives this floor: no derivation reads a Transcript, so no cadence pins it (#841).

const TranscriptFloorDays int64 = 1

func TranscriptWindowDays(dialDays int64) (days int64, bounded bool) {
	// Verbatim bytes are the volume problem, so this dial alone ships bounded at 14 days (ADR-0126).
	if dialDays <= 0 {
		return 0, false
	}
	if dialDays < TranscriptFloorDays {
		return TranscriptFloorDays, true
	}
	return dialDays, true
}

func TranscriptCutoff(now time.Time, dialDays int64) (cutoff time.Time, bounded bool) {
	days, bounded := TranscriptWindowDays(dialDays)
	if !bounded {
		return time.Time{}, false
	}
	window := time.Duration(days*SecondsPerDay) * time.Second
	return now.Add(-window), true
}

type TranscriptStore interface {
	GetRetentionSettings(ctx context.Context) (db.GetRetentionSettingsRow, error)
	DeleteExpiredTranscripts(ctx context.Context, before pgtype.Timestamptz) (int64, error)
}

type TranscriptRetirer struct {
	store TranscriptStore
	now   func() time.Time
	log   *log.Logger
}

func NewTranscriptRetirer(store TranscriptStore, now func() time.Time, logger *log.Logger) *TranscriptRetirer {
	if now == nil {
		now = time.Now
	}
	return &TranscriptRetirer{store: store, now: now, log: logger}
}

func (r *TranscriptRetirer) Sweep(ctx context.Context) (int64, error) {
	settings, err := r.store.GetRetentionSettings(ctx)
	if err != nil {
		return 0, err
	}
	cutoff, bounded := TranscriptCutoff(r.now().UTC(), settings.TranscriptCurrencyDays)
	if !bounded {
		return 0, nil
	}
	return r.store.DeleteExpiredTranscripts(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true})
}

func (r *TranscriptRetirer) Run(ctx context.Context, interval time.Duration) error {
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

func (r *TranscriptRetirer) sweepAndLog(ctx context.Context) {
	n, err := r.Sweep(ctx)
	if err != nil {
		if r.log != nil {
			r.log.Printf("retention: transcript sweep: %v", err)
		}
		return
	}
	if n > 0 && r.log != nil {
		r.log.Printf("retention: retired %d expired transcript row(s)", n)
	}
}
