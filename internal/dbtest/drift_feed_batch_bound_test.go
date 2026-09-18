package dbtest_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/dbtest"
)

// cmd/web is package main and cannot be imported, so its feed bounds are restated here (#2325).

const driftBatchBound = 50

const driftBatchRead int64 = driftBatchBound + 1

const driftFeedBound int32 = 500

func TestOneLargeFoldLeavesRoomForEveryOtherBatch2325(t *testing.T) {
	q, tx := dbtest.Queries(t)
	ctx := context.Background()

	vantage := insertVantage(t, tx, "drift-batch-bound")
	now := time.Now().UTC().Truncate(time.Microsecond)

	// A vantage going unavailable Gaps every span it fed, so one fold can outrun the feed (#2180).
	foldAt := now.Add(-time.Hour)
	fold := insertOutcomeBatch(t, tx, foldAt)
	openSpansInBatch(t, tx, vantage, fold, foldAt, "fold.bound.example.", int(driftFeedBound)+20)

	small := map[int64]bool{}
	for i := range 5 {
		at := now.Add(time.Duration(-6+i) * time.Hour)
		b := insertOutcomeBatch(t, tx, at)
		openSpansInBatch(t, tx, vantage, b, at, fmt.Sprintf("small%d.bound.example.", i), 2)
		small[b] = true
	}

	rows, err := q.ListRecentDriftEvents(ctx, db.ListRecentDriftEventsParams{
		Since:       pgtype.Timestamptz{Time: time.Time{}, Valid: true},
		MaxEvents:   driftFeedBound,
		MaxPerBatch: driftBatchRead,
	})
	if err != nil {
		t.Fatalf("ListRecentDriftEvents: %v", err)
	}

	perBatch := map[int64]int{}
	for _, row := range rows {
		perBatch[row.BatchID]++
	}
	if got := perBatch[fold]; int64(got) != driftBatchRead {
		t.Errorf("the fold contributed %d rows, want %d: the per-batch bound did not hold", got, driftBatchRead)
	}
	for id := range small {
		if perBatch[id] != 2 {
			t.Errorf("batch %d contributed %d rows, want 2: a large fold evicted it from the window", id, perBatch[id])
		}
	}

	// The same read without the per-batch bound is the defect, so it names what the bound buys.
	unbounded, err := q.ListRecentDriftEvents(ctx, db.ListRecentDriftEventsParams{
		Since:       pgtype.Timestamptz{Time: time.Time{}, Valid: true},
		MaxEvents:   driftFeedBound,
		MaxPerBatch: int64(driftFeedBound) + 1,
	})
	if err != nil {
		t.Fatalf("ListRecentDriftEvents unbounded: %v", err)
	}
	for _, row := range unbounded {
		if small[row.BatchID] {
			t.Fatalf("an unbounded read reached batch %d, so this case proves nothing", row.BatchID)
		}
	}
}

func TestABatchUnderTheBoundIsNotProbed2325(t *testing.T) {
	q, tx := dbtest.Queries(t)
	ctx := context.Background()

	vantage := insertVantage(t, tx, "drift-batch-exact")
	now := time.Now().UTC().Truncate(time.Microsecond)

	at := now.Add(-time.Hour)
	batch := insertOutcomeBatch(t, tx, at)
	openSpansInBatch(t, tx, vantage, batch, at, "exact.bound.example.", driftBatchBound)

	rows, err := q.ListRecentDriftEvents(ctx, db.ListRecentDriftEventsParams{
		Since:       pgtype.Timestamptz{Time: time.Time{}, Valid: true},
		MaxEvents:   driftFeedBound,
		MaxPerBatch: driftBatchRead,
	})
	if err != nil {
		t.Fatalf("ListRecentDriftEvents: %v", err)
	}

	got := 0
	for _, row := range rows {
		if row.BatchID == batch {
			got++
		}
	}
	// A batch of exactly the bound returns the bound, so the reader reports no truncation.
	if got != driftBatchBound {
		t.Errorf("a batch of exactly %d rows read back %d", driftBatchBound, got)
	}
}
