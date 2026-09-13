package db

import (
	"strings"
	"testing"
)

// TestTheCensusReadStopsAtTheDrainedDispatch guards ADR-1870 §2. The read carried
// a lower bound only, so a release delayed past later dispatches swept in every
// subject those dispatches opened. ADR-1806 §4 freezes the basis so that a later
// cause cannot fold into an earlier message under that message's instant, and the
// open top reached the same outcome down the subject axis. The upper bound is a
// batch id and never an instant, because ADR-1806 §8 records that no column orders
// a dispatch against a batch soundly.
func TestTheCensusReadStopsAtTheDrainedDispatch(t *testing.T) {
	q := strings.ToLower(listSubjectsOpenedSinceBatch)
	if !strings.Contains(q, "opened_batch_id <=") {
		t.Errorf("the read stops at the dispatch that drained (ADR-1870 §2), got:\n%s", listSubjectsOpenedSinceBatch)
	}
	if !strings.Contains(q, "is null") {
		t.Errorf("the re-point residue passes no upper bound, so the bound is optional (ADR-1870 §4), got:\n%s", listSubjectsOpenedSinceBatch)
	}
}

// TestTheReleaseRowCarriesTheUpperBound guards ADR-1870 §3. The bound belongs to
// the same dispatch the release predicate already chose, so the two cannot drift
// apart. It reads batch.dispatch_id rather than a timestamp, for the reason
// ADR-1806 §8 records. A drained dispatch that opened no batch leaves the root's
// own fold as the whole census, which is what the fallback writes.
func TestTheReleaseRowCarriesTheUpperBound(t *testing.T) {
	q := strings.ToLower(listReleasableHeldMessages)
	if !strings.Contains(q, "census_upper_batch") {
		t.Errorf("the row carries the bound the release predicate's own dispatch fixes (ADR-1870 §3), got:\n%s", listReleasableHeldMessages)
	}
	if !strings.Contains(q, "cb.dispatch_id = first_hot.id") {
		t.Errorf("the bound reads the batches of the dispatch that drained (ADR-1870 §3), got:\n%s", listReleasableHeldMessages)
	}
	if !strings.Contains(q, "coalesce") {
		t.Errorf("a dispatch that opened no batch falls back to the root's own batch (ADR-1870 §3), got:\n%s", listReleasableHeldMessages)
	}
}
