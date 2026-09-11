package db

import (
	"strings"
	"testing"
)

// TestTheReleasePredicateIsADrainedHotDispatch guards ADR-1806 §3. The bound is
// state the model already holds — one hot dispatch that fanned out after the
// root's batch, with every job it enqueued in a terminal state. #27 refuses an
// invented number in a safety path, so no duration window may appear here. The
// status test is separate: a tick the cadence-lag gate recorded skipped enqueues
// no job, and a drain test alone would read it as drained the instant the row
// exists.
func TestTheReleasePredicateIsADrainedHotDispatch(t *testing.T) {
	q := strings.ToLower(listReleasableHeldMessages)
	for _, want := range []struct {
		clause, why string
	}{
		{"census_pending_after_batch is not null", "only a held row is a release candidate"},
		{"s.kind = 'hot'", "the bound names hot by Kind and does not derive the tier"},
		{"d.status = 'fanned-out'", "a skipped tick enqueues no job and would read as drained"},
		{"d.created_at >= b.created_at", "the dispatch must be one that fanned out after the batch"},
		{"limit 1", "the bound is the FIRST such dispatch, not any of them"},
		{"state in ('ready', 'running')", "drained means no job of that dispatch is non-terminal"},
		{"not exists", "the drain test is the complement of the cadence-lag gate's query"},
		{"where exists", "hot commits its dispatch row before its jobs, so an empty job set is not a drained one"},
	} {
		if !strings.Contains(q, want.clause) {
			t.Errorf("listReleasableHeldMessages must carry %q — %s (ADR-1806 §3), got:\n%s", want.clause, want.why, listReleasableHeldMessages)
		}
	}
	for _, forbidden := range []string{"interval", "now()"} {
		if strings.Contains(q, forbidden) {
			t.Errorf("the bound is a drained tier and never a clock, so %q may not appear (ADR-1806 §7, #27), got:\n%s", forbidden, listReleasableHeldMessages)
		}
	}
}

// TestTwoPollPassesReleaseARowOnce guards the claim. Two dispatcher instances
// poll the same table, and a released row must not be enqueued for delivery
// twice or gain its census clause twice. The guarded UPDATE is the claim: it
// takes the row lock and re-tests the pending column, so the pass that arrives
// second affects no row and its caller enqueues nothing.
func TestTwoPollPassesReleaseARowOnce(t *testing.T) {
	if !strings.Contains(strings.ToLower(releaseHeldMessage), "census_pending_after_batch is not null") {
		t.Errorf("the release update must re-test the pending column, so a second pass takes no row (ADR-1806 §2), got:\n%s", releaseHeldMessage)
	}
	if !strings.Contains(strings.ToLower(releaseHeldMessage), "census_pending_after_batch = null") {
		t.Errorf("release clears the pending column, so the panel and the badge see the row (ADR-1806 §2), got:\n%s", releaseHeldMessage)
	}
}

// TestTheCensusReadIsBoundedByTheRootsOwnBatch guards the lower bound. The
// re-point residue and the scope reveal each drop a subject a membership root of
// the same fold covers, and the frozen basis records no list of what opened in
// that fold. A strictly-after bound would lose such a subject from every census
// (#1816). The bound is the opening batch and not a timestamp, because the span
// table cites the batch that opened each span.
func TestTheCensusReadIsBoundedByTheRootsOwnBatch(t *testing.T) {
	q := strings.ToLower(listSubjectsOpenedSinceBatch)
	if !strings.Contains(q, "opened_batch_id >=") {
		t.Errorf("the bound is the root's own batch, INCLUSIVE, and reads the opening batch (#1816), got:\n%s", listSubjectsOpenedSinceBatch)
	}
	if !strings.Contains(q, "subject_kind in ('service', 'endpoint')") {
		t.Errorf("a census admits a service or an endpoint only (ADR-1806 §4), got:\n%s", listSubjectsOpenedSinceBatch)
	}
	if strings.Contains(q, "opened_at") || strings.Contains(q, "created_at") {
		t.Errorf("the query bounds on the opening batch and never on a timestamp (#1816), got:\n%s", listSubjectsOpenedSinceBatch)
	}
}
