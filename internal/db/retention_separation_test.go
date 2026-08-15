package db

import (
	"strings"
	"testing"
)

// TestDeleteExpiredDispatchesTouchesOnlyDispatch guards the structural half of
// the Dispatch retention ACs (v1 spec §4.6, ADR-0041): retiring an expired
// Dispatch may never reach the Observation or Span corpus. The deletion query
// must name the dispatch table and no measured-data table, so a clock retiring
// Dispatch can move no value on any timeline. The FK change in migration 20900
// makes the sever a SET NULL on the operational back-references rather than a
// cascade into measured data.
func TestDeleteExpiredDispatchesTouchesOnlyDispatch(t *testing.T) {
	sql := strings.ToLower(deleteExpiredDispatches)

	if !strings.Contains(sql, "delete from dispatch") {
		t.Fatalf("deletion query must delete from dispatch, got:\n%s", deleteExpiredDispatches)
	}
	for _, forbidden := range []string{"observation", "span", "batch", "queue_job"} {
		if strings.Contains(sql, forbidden) {
			t.Errorf("dispatch deletion query must never name %q — it reaches measured/operational data:\n%s", forbidden, deleteExpiredDispatches)
		}
	}
}
