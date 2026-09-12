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
		{"first_hot.fanout_complete", "a streamed tier commits its dispatch row before its jobs, so the row carries that half"},
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
	assertNoJobCountProxy(t, "listReleasableHeldMessages", listReleasableHeldMessages)
}

// assertNoJobCountProxy catches the one spelling of the proxy #1816 used for the fan-out-finished
// half of ADR-1806 §3's bound, and it catches no other. A rewrite under a different alias or as a
// count comparison passes it. The load-bearing half is the positive assertion the two callers make
// on first_hot.fanout_complete; this one names the exact shape #1851 removed, so a revert of that
// commit fails a test rather than passing every one.
func assertNoJobCountProxy(t *testing.T, name, query string) {
	t.Helper()
	flat := strings.Join(strings.Fields(strings.ToLower(query)), " ")
	// The drain half keeps its own EXISTS, so the proxy is the one carrying no job-state test.
	if strings.Contains(flat, "exists ( select 1 from queue_job j where j.dispatch_id = first_hot.id )") {
		t.Errorf("%s must not read a job count for the fan-out-finished half (ADR-1851 §2, #1851), got:\n%s", name, query)
	}
}

// TestTheFanOutMarkIsWrittenOnceAndCarriesNoInstant guards ADR-1851 §2. The column answers the
// first half of ADR-1806 §3's bound, and it answers nothing else. A timestamp here would invite
// the duration test ADR-1806 §7 and #27 both refuse, so the column is a boolean and the mark is
// an unconditional write that the caller places after its last chunk commits.
func TestTheFanOutMarkIsWrittenOnceAndCarriesNoInstant(t *testing.T) {
	q := strings.ToLower(markFanOutComplete)
	if !strings.Contains(q, "fanout_complete = true") {
		t.Errorf("the mark sets the fan-out-finished half (ADR-1851 §2), got:\n%s", markFanOutComplete)
	}
	for _, forbidden := range []string{"now()", "interval", "fanout_completed_at"} {
		if strings.Contains(q, forbidden) {
			t.Errorf("the mark carries no instant, so %q may not appear (ADR-1806 §7, #27), got:\n%s", forbidden, markFanOutComplete)
		}
	}
}

// TestAnAbandonedDispatchIsRetiredByATickAndNeverByAClock guards ADR-1851 §3. A streamed
// fan-out that crashes marks itself never, so its dispatch would hold the release bound for
// good. The next claimed tick of the same Scan marks it abandoned, and a tick is a cadence
// rather than a duration, so no clock reaches the release predicate. Three clauses keep a live
// fan-out safe: the claimed dispatch is excluded by id, a dispatch still streaming holds ready
// jobs, and an already-abandoned row is not re-marked.
//
// The mark is a column of its own and never a fifth dispatch.status token, because ADR-0164 §1
// rules that status carries the operator's disposition. stopped and terminated each cancel jobs
// and mean a person acted, and abandonment does neither.
func TestAnAbandonedDispatchIsRetiredByATickAndNeverByAClock(t *testing.T) {
	q := strings.ToLower(abandonUnfinishedDispatches)
	for _, want := range []struct {
		clause, why string
	}{
		{"set fanout_abandoned = true", "the mark is a column, so no operator disposition is minted"},
		{"dispatch.status = 'fanned-out'", "a skipped row or an operator's disposition is not an abandoned fan-out"},
		{"dispatch.fanout_complete = false", "a finished fan-out is never abandoned"},
		{"dispatch.id <> ", "the tick doing the marking may never mark itself"},
		{"dispatch.fanout_abandoned = false", "an already-abandoned row is not re-marked"},
		{"j.state in ('ready', 'running')", "a fan-out still streaming holds ready jobs, so it is left alone"},
	} {
		if !strings.Contains(q, want.clause) {
			t.Errorf("abandonUnfinishedDispatches must carry %q — %s (ADR-1851 §3), got:\n%s", want.clause, want.why, abandonUnfinishedDispatches)
		}
	}
	for _, forbidden := range []string{"now()", "interval", "age(", "status = 'terminated'", "status = 'stopped'"} {
		if strings.Contains(q, forbidden) {
			t.Errorf("a tick marks a column, never a clock and never a disposition, so %q may not appear (ADR-0164 §1, ADR-1851 §3), got:\n%s", forbidden, abandonUnfinishedDispatches)
		}
	}
}

// TestTheBoundPassesOverAnAbandonedDispatch guards the other half of ADR-1851 §3. Marking a
// dispatch abandoned moves nothing on its own: the inner select that picks the first hot
// dispatch has to skip it, or the bound still names a row that will never be marked complete.
// This is not #1817's reverted shape. That one skipped a dispatch for holding no job, which a
// dispatch mid-fan-out also does. This one skips a dispatch for a recorded fact, so no dispatch
// still streaming is ever passed over.
func TestTheBoundPassesOverAnAbandonedDispatch(t *testing.T) {
	for _, c := range []struct{ name, query string }{
		{"listReleasableHeldMessages", listReleasableHeldMessages},
		{"listSettleableRePointBatches", listSettleableRePointBatches},
	} {
		if !strings.Contains(strings.ToLower(c.query), "d.fanout_abandoned = false") {
			t.Errorf("%s must skip an abandoned dispatch when choosing the first hot one (ADR-1851 §3), got:\n%s", c.name, c.query)
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

// TestEachDegradationLeavesAPathOutOfTheHold guards ADR-1806 §6. The hold reads
// the state of the hot tier, and two operator configurations make that state
// meaningless. With no stale-running reaper the drain test reads a job set that
// nothing reaps, so the caller passes the flag and every held row leaves at
// once. With the hot Scan disabled nothing will ever open beneath the root, so
// waiting for a drain waits forever. A held row that can never release is the
// failure #1817 exists to prevent.
//
// Neither arm forces the census empty. The census read is bounded below by the
// root's own batch, inclusive (#1816), so a released row still names whatever
// the root's own fold opened beneath it. For a Name root that is nothing, since
// its Service and Endpoint open in a later hot fold — which is the empty census
// ADR-1806 §6 describes, reached by reading rather than by assertion.
func TestEachDegradationLeavesAPathOutOfTheHold(t *testing.T) {
	q := strings.ToLower(listReleasableHeldMessages)
	for _, want := range []struct {
		clause, why string
	}{
		{"$1::boolean", "the caller states that the reaper is disabled, so no drain test can conclude"},
		{"not exists (\n          select 1 from scan hs where hs.kind = 'hot' and hs.enabled", "a disabled hot tier opens nothing beneath the root, ever"},
	} {
		if !strings.Contains(q, want.clause) {
			t.Errorf("listReleasableHeldMessages must carry %q — %s (ADR-1806 §6), got:\n%s", want.clause, want.why, listReleasableHeldMessages)
		}
	}
	// Each arm is an alternative to the drain test, never a narrowing of it.
	if strings.Count(q, " or ") < 2 {
		t.Errorf("the two degradations are disjuncts beside the drain test (ADR-1806 §6), got:\n%s", listReleasableHeldMessages)
	}
}
