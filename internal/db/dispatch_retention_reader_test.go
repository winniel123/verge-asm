package db

import (
	"strings"
	"testing"
)

// TestTheDispatchSweepKeepsWhatAPendingReleaseMayRead guards #1853. ADR-0041
// retains a corpus by what may still read it, and ADR-1806 §3 made one hot
// Dispatch the bound a held membership message and an unsettled re-point batch
// both read. The wall-clock sweep of that corpus must therefore exempt the row
// each predicate still reads. Deleting it breaks the predicate twice over: the
// inner select finds no row, and migration 20900 nulls the dispatch_id of every
// surviving job, so no job can match a dispatch id again. Either route holds the
// row, and #1814 keeps a held row out of the panel and out of the unread badge,
// so the hold is silent.
func TestTheDispatchSweepKeepsWhatAPendingReleaseMayRead(t *testing.T) {
	sweep := strings.ToLower(deleteExpiredDispatches)
	if !strings.Contains(sweep, "not (id = any(") {
		t.Errorf("the sweep must exempt the ids a pending release may read (ADR-0041, #1853), got:\n%s", deleteExpiredDispatches)
	}
	// A dispatch created after the caller's read is no row it could have exempted.
	if !strings.Contains(sweep, "created_at < $1") {
		t.Errorf("the sweep must also age a dispatch by created_at, the instant the bound orders by (#1853), got:\n%s", deleteExpiredDispatches)
	}
}

// TestTheExemptSetIsBothReleasePredicatesOwnRead guards the exempt set against
// drift. It is not a second answer to the release question: it is the same inner
// select, so a change to either predicate's choice of dispatch must reach the
// sweep in the same commit. A clause that appears in one and not the other would
// exempt a row no predicate reads, or retire a row one of them does.
func TestTheExemptSetIsBothReleasePredicatesOwnRead(t *testing.T) {
	exempt := strings.ToLower(listDispatchesAPendingReleaseMayRead)
	for _, clause := range []string{
		"s.kind = 'hot'",
		"d.status = 'fanned-out'",
		"d.fanout_abandoned = false",
		"d.fanout_complete",
		"d.created_at >=",
		"order by d.created_at, d.id",
		"limit 1",
	} {
		for name, predicate := range map[string]string{
			"listReleasableHeldMessages":   listReleasableHeldMessages,
			"listSettleableRePointBatches": listSettleableRePointBatches,
		} {
			if !strings.Contains(strings.ToLower(predicate), clause) {
				t.Errorf("%s must carry %q — the exempt set mirrors it (#1853), got:\n%s", name, clause, predicate)
			}
		}
		if !strings.Contains(exempt, clause) {
			t.Errorf("the exempt set must carry %q, so it picks the row the predicates pick (#1853), got:\n%s", clause, listDispatchesAPendingReleaseMayRead)
		}
	}
	// Both held sources are candidates: a membership row holds a column, a fold holds an instant.
	for _, source := range []string{
		"census_pending_after_batch is not null",
		"repoint_settled_at is null",
	} {
		if !strings.Contains(exempt, source) {
			t.Errorf("the exempt set must read the rows held by %q (ADR-1806 §3, #1853), got:\n%s", source, listDispatchesAPendingReleaseMayRead)
		}
	}
	// The dial is the sweep's clock. The set it exempts is read without one (ADR-1806 §7, #27).
	for _, forbidden := range []string{"interval", "now()"} {
		if strings.Contains(exempt, forbidden) {
			t.Errorf("the exempt set is a read of the predicates and never a clock, so %q may not appear (#27), got:\n%s", forbidden, listDispatchesAPendingReleaseMayRead)
		}
	}
}

// TestTheExemptSetReachesThePickATickMovesTo covers ADR-1851 §3. A tick retires an
// unfinished pick, and both predicates then bound on the first finished row at or
// after the held batch. Retiring that row defers the release by a cadence, so the
// exempt set names both picks. The pick stops there: AbandonUnfinishedDispatches
// crosses no row that holds the mark, so a finished pick is final.
func TestTheExemptSetReachesThePickATickMovesTo(t *testing.T) {
	exempt := strings.ToLower(listDispatchesAPendingReleaseMayRead)
	if got := strings.Count(exempt, "limit 1"); got != 2 {
		t.Errorf("the exempt set names two picks, the bound's own and the one a tick moves it to, got %d (ADR-1851 §3):\n%s", got, listDispatchesAPendingReleaseMayRead)
	}
	// The read costs one lateral per held instant, so the sweep's own cutoff bounds it (#1853).
	if !strings.Contains(exempt, "b.created_at < $1") {
		t.Errorf("the exempt set must drop a batch at or after the cutoff, whose pick the delete already spares (#1853), got:\n%s", listDispatchesAPendingReleaseMayRead)
	}
}
