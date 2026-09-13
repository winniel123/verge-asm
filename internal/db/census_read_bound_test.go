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
	flat := strings.Join(strings.Fields(strings.ToLower(listSubjectsOpenedSinceBatch)), " ")
	// The upper bound reads the same column as the lower one. Bounding on any other column
	// would pass a bare "opened_batch_id <=" test while reading different ground.
	if !strings.Contains(flat, "s.opened_batch_id <= $2") {
		t.Errorf("the read stops at the drained dispatch's last batch (ADR-1870 §2), got:\n%s", listSubjectsOpenedSinceBatch)
	}
	if !strings.Contains(flat, "$2::bigint is null or") {
		t.Errorf("the residue passes no upper bound, so the bound is optional (ADR-1870 §4), got:\n%s", listSubjectsOpenedSinceBatch)
	}
	if !strings.Contains(flat, "s.opened_batch_id >= $1") {
		t.Errorf("the lower bound is the root's own batch, inclusive (#1816), got:\n%s", listSubjectsOpenedSinceBatch)
	}
}

// TestTheReleaseRowCarriesTheUpperBound guards ADR-1870 §3. The bound belongs to
// the same dispatch the release predicate already chose, so the two cannot drift
// apart. It reads batch.dispatch_id rather than a timestamp, for the reason
// ADR-1806 §8 records. A drained dispatch that opened no batch leaves the root's
// own fold as the whole census, which is what the fallback writes.
func TestTheReleaseRowCarriesTheUpperBound(t *testing.T) {
	flat := strings.Join(strings.Fields(strings.ToLower(listReleasableHeldMessages)), " ")
	// The whole expression, not its parts. Swapping the COALESCE arguments, or reading some
	// other dispatch's batches, leaves every substring of this query in place while every
	// release bounds at the wrong batch.
	if !strings.Contains(flat, "coalesce(drained.census_upper_batch, 0)::bigint as census_upper_batch") {
		t.Errorf("zero is the bound where the dispatch opened no batch, and the order says so (ADR-1870 §3), got:\n%s", listReleasableHeldMessages)
	}
	if !strings.Contains(flat, "(select max(cb.id) from batch cb where cb.dispatch_id = first_hot.id) as census_upper_batch") {
		t.Errorf("the bound is the last batch of the dispatch that drained (ADR-1870 §3), got:\n%s", listReleasableHeldMessages)
	}
	if !strings.Contains(flat, "drained.drained_dispatch is not null") {
		t.Errorf("one dispatch fixes the release arm and the bound (ADR-1870 §3), got:\n%s", listReleasableHeldMessages)
	}
}
