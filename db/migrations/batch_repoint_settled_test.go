package migrations

import (
	"strings"
	"testing"
)

func TestABatchRecordsWhenItsRePointResidueWasRead(t *testing.T) {
	// Two durable spans stay true after the message fires, so the poll owes a bound (ADR-1806 §8).
	col := tableColumn(t, "batch", "repoint_settled_at")

	if !strings.Contains(col, "timestamptz") {
		t.Errorf("repoint_settled_at must be a TIMESTAMPTZ — the fold records an instant, got: %s", col)
	}
	if strings.Contains(col, "not null") {
		t.Errorf("repoint_settled_at must be nullable — an unread fold records no instant, got: %s", col)
	}
}

func TestTheUnsettledScanIsProportionalToWhatIsUnsettled(t *testing.T) {
	// One queue job is one Batch, so every row of the largest operational record is a candidate.
	idx := tableIndexes(t, "batch")
	for _, ix := range idx {
		if ix.keys("id") && ix.where == "repoint_settled_at is null" {
			return
		}
	}
	t.Errorf("no partial index over the unsettled batches — the poll would scan every batch "+
		"each minute, got: %v", idx)
}

func TestTheHistoryIsSettledByTheMigration(t *testing.T) {
	// Every move the estate folded still satisfies the predicate, so the first pass would fire all.
	up := strings.ToLower(upMigrations(t))
	if !strings.Contains(up, "update batch set repoint_settled_at = now()") {
		t.Error("the migration must settle the existing batches — the first pass would announce the whole history")
	}
}
