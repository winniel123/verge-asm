package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func emptiedBoundedBatch(at time.Time) []db.ListRecentDriftEventsRow {
	rows := make([]db.ListRecentDriftEventsRow, 0, int(driftBatchLimit)+1)
	for i := range int(driftBatchLimit) + 1 {
		row := driftOpenedRow(1, at, fmt.Sprintf("d%03d.example.com", i),
			`{"outcome":"NXDOMAIN"}`, `{"outcome":"Resolved"}`)
		// A predecessor from another leaf is a row the classifier refuses, so the batch empties.
		row.PrevDerivation = []byte(`[{"leaf":"zone-transfer","version":"1"}]`)
		rows = append(rows, row)
	}
	return rows
}

func TestDriftFeedBoundedReportsABatchThatRendersNoGroup(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	rows := emptiedBoundedBatch(now.Add(-36 * time.Hour))

	groups, _ := buildDriftFeed(rows, now)
	if len(groups) != 0 {
		t.Fatalf("groups = %d, want 0: the fixture must leave the bound with no group to hang on", len(groups))
	}
	if !driftFeedBounded(rows) {
		t.Error("a bounded batch the classifier emptied reported no bound")
	}
}

func TestDriftTransitionDeltaIsSuppressedWhenABoundedBatchRendersNoGroup(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	prevStart := now.Add(-48 * time.Hour)
	rows := emptiedBoundedBatch(now.Add(-36 * time.Hour))

	earliest := pgtype.Timestamptz{Time: prevStart.Add(-time.Hour), Valid: true}
	if got := driftTransitionDelta(10, rows, false, earliest, prevStart, now); got != "" {
		t.Errorf("delta = %q, want empty: a bound whose batch renders no group still makes the count a floor", got)
	}
}
