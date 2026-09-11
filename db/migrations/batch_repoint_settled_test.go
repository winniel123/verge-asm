package migrations

import (
	"regexp"
	"strings"
	"testing"
)

func TestABatchRecordsWhenItsRePointResidueWasRead(t *testing.T) {
	// Two durable spans stay true after the message fires, so the poll owes a bound (ADR-1806 §8).
	up := strings.ToLower(upMigrations(t))

	col := regexp.MustCompile(`repoint_settled_at\s+timestamptz[^,;]*`).FindString(up)
	if col == "" {
		t.Fatal("no batch.repoint_settled_at column found — a move would fire on every poll pass (ADR-1806 §8)")
	}
	if strings.Contains(col, "not null") {
		t.Errorf("repoint_settled_at must be nullable — an unread fold records no instant, got: %s", strings.TrimSpace(col))
	}
}

func TestTheUnsettledScanIsProportionalToWhatIsUnsettled(t *testing.T) {
	// One queue job is one Batch, so every row of the largest operational record is a candidate.
	up := strings.ToLower(upMigrations(t))
	if !strings.Contains(up, "on batch (id) where repoint_settled_at is null") {
		t.Error("no partial index over the unsettled batches — the poll would scan every batch each minute")
	}
}

func TestTheHistoryIsSettledByTheMigration(t *testing.T) {
	// Every move the estate folded still satisfies the predicate, so the first pass would fire all.
	up := strings.ToLower(upMigrations(t))
	if !strings.Contains(up, "update batch set repoint_settled_at = now()") {
		t.Error("the migration must settle the existing batches — the first pass would announce the whole history")
	}
}
