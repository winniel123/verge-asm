package db

import (
	"strings"
	"testing"
)

// TestTheRePointReadReadsTheSameDrainTheReleaseReads guards ADR-1806 §3 and §6 on the second
// path. A held membership row and a re-point move both wait for the tier that admits the
// subtree, so the two must not disagree about when it has drained. The clauses below are
// ListReleasableHeldMessages' own, and TestTheReleasePredicateIsADrainedHotDispatch holds the
// other copy to the same list.
func TestTheRePointReadReadsTheSameDrainTheReleaseReads(t *testing.T) {
	q := strings.ToLower(listSettleableRePointBatches)
	for _, want := range []struct {
		clause, why string
	}{
		{"repoint_settled_at is null", "only an unread fold is a settle candidate"},
		{"as has_move", "a fold holding no move owes no pass of its own"},
		{"s.kind = 'hot'", "the bound names hot by Kind and does not derive the tier"},
		{"d.status = 'fanned-out'", "a skipped tick enqueues no job and would read as drained"},
		{"d.created_at >= b.created_at", "the dispatch must be one that fanned out after the batch"},
		{"limit 1", "the bound is the FIRST such dispatch, not any of them"},
		{"state in ('ready', 'running')", "drained means no job of that dispatch is non-terminal"},
		{"not exists", "the drain test is the complement of the cadence-lag gate's query"},
		{"first_hot.fanout_complete", "a streamed tier commits its dispatch row before its jobs, so the row carries that half"},
		{"::boolean", "the reaper's state is the caller's argument, as it is on the release path"},
		{"hs.kind = 'hot' and hs.enabled", "a disabled hot tier opens nothing beneath the move, ever"},
	} {
		if !strings.Contains(q, want.clause) {
			t.Errorf("listSettleableRePointBatches must carry %q — %s (ADR-1806 §3), got:\n%s", want.clause, want.why, listSettleableRePointBatches)
		}
	}
	if strings.Contains(q, "interval") {
		t.Errorf("the bound is a drained tier and never a clock (ADR-1806 §7, #27), got:\n%s", listSettleableRePointBatches)
	}
	assertNoJobCountProxy(t, "listSettleableRePointBatches", listSettleableRePointBatches)
}

// TestTwoPollPassesSettleAFoldOnce guards the claim. The spans ADR-0026 §2 reads stay true after
// the message fires, so nothing but this UPDATE stops the minute poll re-announcing the move
// (ADR-1806 §8). The guarded UPDATE takes the row lock and re-tests the column, so the pass that
// arrives second claims no batch and writes no second message.
func TestTwoPollPassesSettleAFoldOnce(t *testing.T) {
	q := strings.ToLower(settleRePointBatch)
	if !strings.Contains(q, "repoint_settled_at is null") {
		t.Errorf("the claim must re-test the column, so a second pass takes no fold, got:\n%s", settleRePointBatch)
	}
	if !strings.Contains(q, "set repoint_settled_at =") {
		t.Errorf("the claim must be the UPDATE itself and never a read, got:\n%s", settleRePointBatch)
	}
	if !strings.Contains(strings.ToLower(settleMovelessRePointBatches), "repoint_settled_at is null") {
		t.Errorf("the moveless settle must re-test the column too, got:\n%s", settleMovelessRePointBatches)
	}
}

// TestARePointMoveIsReadFromTheTwoAdjacentSpans guards ADR-0026 §2's predicate. The move is
// defined over the two adjacent resolution spans alone, and both are durable, so the poll owes
// no fold-local state (#1818). The pair is a move rather than an opening because the fold that
// opened one closed the other on the same timeline.
func TestARePointMoveIsReadFromTheTwoAdjacentSpans(t *testing.T) {
	q := strings.ToLower(listRePointMovesForBatch)
	for _, want := range []struct {
		clause, why string
	}{
		{"p.closed_batch_id = n.opened_batch_id", "one fold closed the previous span and opened this one"},
		{"p.vantage_id is not distinct from n.vantage_id", "a timeline is keyed by its vantage, and the shipped resolver carries none"},
		{"n.subject_kind = 'name'", "only a Name re-points"},
		{"n.facet = 'resolution'", "ADR-0026 §2 rules on the resolution facet"},
		{"n.is_gap = false", "a Gap opens no value"},
		{"p.is_gap = false", "a Gap-closing edge is coverage, which gapclose carries (ADR-0014)"},
	} {
		if !strings.Contains(q, want.clause) {
			t.Errorf("listRePointMovesForBatch must carry %q — %s, got:\n%s", want.clause, want.why, listRePointMovesForBatch)
		}
	}
}

// TestTheCiterReadIsTakenAtTheFoldsOwnInstant guards the partition. The Address root fired at the
// cause on what the estate held then, and the residue drops exactly what that root covers. A read
// of the currently-open spans would answer for today instead, and the two would disagree (#1730).
func TestTheCiterReadIsTakenAtTheFoldsOwnInstant(t *testing.T) {
	q := strings.ToLower(listResolutionCitersForAddressesAt)
	if !strings.Contains(q, "r.opened_at <= $2") {
		t.Errorf("a citer must have been open at the move's instant, got:\n%s", listResolutionCitersForAddressesAt)
	}
	if !strings.Contains(q, "r.closed_at > $2") {
		t.Errorf("a span the same fold closed is no citer after it, got:\n%s", listResolutionCitersForAddressesAt)
	}
}

// TestAFoldRootIsATimelineTheFoldFoundNoOpenSpanFor guards the second half of the partition.
// membershipMessages roots where the fold opened a resolution timeline it held no open span for,
// and a move is the case where it did hold one. Reading both as roots would empty every residue.
func TestAFoldRootIsATimelineTheFoldFoundNoOpenSpanFor(t *testing.T) {
	q := strings.ToLower(listNameRootsOpenedInBatch)
	if !strings.Contains(q, "not exists") || !strings.Contains(q, "p.closed_batch_id = n.opened_batch_id") {
		t.Errorf("a timeline the same fold closed moved and is no root, got:\n%s", listNameRootsOpenedInBatch)
	}
}
