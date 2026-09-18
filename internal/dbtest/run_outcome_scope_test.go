package dbtest_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/dbtest"
)

// The run-detail outcome join counted a run's transitions out of the drift
// feed's newest 500 rows, so a run older than those rows stated Transitions: 0.
// cmd/web is package main and cannot be imported, so the proof runs the two
// statements the join now reads and the one it used to (#2247).

// cmd/web/drift.go's driftFeedLimit, which the old join passed as MaxEvents.
const outcomeFeedLimit int32 = 500

func insertOutcomeBatch(t *testing.T, tx pgx.Tx, at time.Time) int64 {
	t.Helper()
	var id int64
	err := tx.QueryRow(context.Background(),
		`INSERT INTO batch (scan_id, kind, outcome, offers, recorded_scope, created_at)
		 SELECT s.id, 'resolution-walk', 'completed', '{}'::jsonb, '{}'::jsonb, $1
		 FROM scan s WHERE s.kind = 'dns'
		 RETURNING id`, at).Scan(&id)
	if err != nil {
		t.Fatalf("insert batch at %s: %v", at, err)
	}
	return id
}

func openSpansInBatch(t *testing.T, tx pgx.Tx, vantageID, batchID int64, at time.Time, prefix string, n int) {
	t.Helper()
	_, err := tx.Exec(context.Background(),
		`INSERT INTO span (subject_kind, subject_key, facet, discriminator, vantage_id,
		                   source, value, is_gap, derivation, opened_at, opened_batch_id)
		 SELECT 'name', $1 || g::text, 'resolution', '', $2, 'resolver',
		        '{"outcome":"Resolved"}'::jsonb, FALSE, '[]'::jsonb, $3, $4
		 FROM generate_series(1, $5) AS g`,
		prefix, vantageID, at, batchID, n)
	if err != nil {
		t.Fatalf("open %d spans in batch %d: %v", n, batchID, err)
	}
}

func TestOlderRunReadsItsOwnTransitionCount2247(t *testing.T) {
	q, tx := dbtest.Queries(t)
	ctx := context.Background()

	vantage := insertVantage(t, tx, "run-outcome-scope")

	now := time.Now().UTC()
	oldAt := now.Add(-30 * 24 * time.Hour)
	recentAt := now.Add(-time.Hour)

	oldBatch := insertOutcomeBatch(t, tx, oldAt)
	openSpansInBatch(t, tx, vantage, oldBatch, oldAt, "old.outcome.example.", 3)

	// More than the feed limit, so the capped read reaches no row of the old run.
	recentBatch := insertOutcomeBatch(t, tx, recentAt)
	openSpansInBatch(t, tx, vantage, recentBatch, recentAt, "recent.outcome.example.", int(outcomeFeedLimit)+20)

	capped, err := q.ListRecentDriftEvents(ctx, db.ListRecentDriftEventsParams{
		Since:     pgtype.Timestamptz{Time: time.Time{}, Valid: true},
		MaxEvents: outcomeFeedLimit,
	})
	if err != nil {
		t.Fatalf("ListRecentDriftEvents: %v", err)
	}
	for _, row := range capped {
		if row.BatchID == oldBatch {
			t.Fatalf("the capped feed read reached the old run's batch, so this case proves nothing")
		}
	}

	scoped, err := q.ListDriftEventsForBatches(ctx, []int64{oldBatch})
	if err != nil {
		t.Fatalf("ListDriftEventsForBatches: %v", err)
	}
	if len(scoped) != 3 {
		t.Errorf("scoped transitions for the old run: got %d, want 3", len(scoped))
	}
	for _, row := range scoped {
		if row.BatchID != oldBatch {
			t.Errorf("scoped read returned batch %d, want only %d", row.BatchID, oldBatch)
		}
	}
}

func TestBatchWindowBoundsOnTheNextBatch2247(t *testing.T) {
	q, tx := dbtest.Queries(t)
	ctx := context.Background()

	vantage := insertVantage(t, tx, "run-outcome-window")

	// timestamptz keeps microseconds, so a nanosecond instant reads back changed.
	now := time.Now().UTC().Truncate(time.Microsecond)
	oldAt := now.Add(-30 * 24 * time.Hour)
	quietAt := now.Add(-20 * 24 * time.Hour)
	latestAt := now.Add(-time.Hour)

	oldBatch := insertOutcomeBatch(t, tx, oldAt)
	openSpansInBatch(t, tx, vantage, oldBatch, oldAt, "window.outcome.example.", 2)
	// This batch moved nothing, so no transitions list holds its instant.
	insertOutcomeBatch(t, tx, quietAt)
	latestBatch := insertOutcomeBatch(t, tx, latestAt)
	openSpansInBatch(t, tx, vantage, latestBatch, latestAt, "windowlate.outcome.example.", 1)

	windows, err := q.ListBatchWindows(ctx, []int64{oldBatch, latestBatch})
	if err != nil {
		t.Fatalf("ListBatchWindows: %v", err)
	}
	if len(windows) != 2 {
		t.Fatalf("windows: got %d rows, want 2", len(windows))
	}

	old := windows[0]
	if old.BatchID != oldBatch {
		t.Fatalf("first window: got batch %d, want %d", old.BatchID, oldBatch)
	}
	if !old.BatchAt.Valid || !old.BatchAt.Time.UTC().Equal(oldAt) {
		t.Errorf("old window start: got %v, want %v", old.BatchAt.Time, oldAt)
	}
	if !old.NextBatchAt.Valid || !old.NextBatchAt.Time.UTC().Equal(quietAt) {
		t.Errorf("old window end: got %v, want the quiet batch at %v", old.NextBatchAt, quietAt)
	}

	latest := windows[1]
	if latest.BatchID != latestBatch {
		t.Fatalf("second window: got batch %d, want %d", latest.BatchID, latestBatch)
	}
	if latest.NextBatchAt.Valid {
		t.Errorf("latest window end: got %v, want none", latest.NextBatchAt.Time)
	}
}
