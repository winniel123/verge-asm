package dbtest_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/dbtest"
	"github.com/winniel123/verge-asm/internal/exposure"
)

// The two statements a restore runs over the corpus it just replayed, in the order it
// runs them: ADR-0124 promises the replayed vantage goes unavailable, and ADR-2087's
// writer is what closes the spans behind it.

const restoredService = "198.51.100.7:443/tcp"

func reachedSpan(subjectKey string) spanRow {
	return spanRow{
		facet:       "reachability",
		subjectKind: "service",
		subjectKey:  subjectKey,
		source:      "prober",
		value:       `{"outcome":"reached"}`,
	}
}

func reachLegOf(t *testing.T, q *db.Queries, subjectKey string) (outcomes []string, gaps int) {
	t.Helper()
	rows, err := q.ListServiceReachabilitySpansByClass(context.Background())
	if err != nil {
		t.Fatalf("ListServiceReachabilitySpansByClass: %v", err)
	}
	for _, row := range rows {
		if row.SubjectKey != subjectKey {
			continue
		}
		var v struct {
			Outcome string `json:"outcome"`
		}
		if err := json.Unmarshal(row.Value, &v); err != nil {
			t.Fatalf("decode span %d: %v", row.ID, err)
		}
		outcomes = append(outcomes, v.Outcome)
		if row.IsGap {
			gaps++
		}
	}
	return outcomes, gaps
}

func TestARestoredAvailableVantageGapsTheExposureLegItCanNoLongerTake(t *testing.T) {
	q, tx := dbtest.Queries(t)
	ctx := context.Background()

	replayed := insertVantage(t, tx, "restore-replayed")
	insertSpan(t, tx, replayed, reachedSpan(restoredService))

	// A restore drops the prober keypair, and this position owns none of it (#2200, #2244).
	resolverOnly := insertResolverOnlyVantage(t, tx, "restore-resolver-only")
	kept := insertSpan(t, tx, resolverOnly, valued("resolution", "a.example"))

	outcomes, _ := reachLegOf(t, q, restoredService)
	if v, ok := exposure.ComposeReach(outcomes); !ok || v != exposure.Reached {
		t.Fatalf("the replayed span composed %q/%v, so this case proves no over-report", v, ok)
	}

	ids, err := q.ListAvailableProberVantageIDs(ctx)
	if err != nil {
		t.Fatalf("ListAvailableProberVantageIDs: %v", err)
	}
	if !containsID(ids, replayed) {
		t.Fatalf("the replayed vantage %d is not in %v, so no restore would mark it", replayed, ids)
	}
	if containsID(ids, resolverOnly) {
		t.Errorf("the read lists the resolver-only vantage %d, so a restore Gaps a position "+
			"whose instrument it never dropped (ADR-2087 section 6, #2200)", resolverOnly)
	}
	for _, id := range ids {
		if err := q.MarkVantageUnavailable(ctx, id); err != nil {
			t.Fatalf("MarkVantageUnavailable(%d): %v", id, err)
		}
	}

	if got := availabilityOf(t, tx, replayed); got != "unavailable" {
		t.Errorf("availability = %q after the restore's mark, want unavailable (ADR-0124)", got)
	}

	outcomes, gaps := reachLegOf(t, q, restoredService)
	if v, ok := exposure.ComposeReach(outcomes); ok {
		t.Errorf("the leg composed %q behind a dead prober, want non-constructible (ADR-0005)", v)
	}
	if gaps != len(outcomes) || gaps == 0 {
		t.Errorf("the exposure read returns %d open spans and %d of them are Gaps", len(outcomes), gaps)
	}

	var closed, gapped bool
	for _, s := range spansOf(t, tx, replayed) {
		switch {
		case !s.isGap && s.closed:
			closed = true
		case s.isGap && !s.closed:
			gapped = true
			if s.cause != "vantage-unavailable" {
				t.Errorf("the Gap carries the cause %q, want vantage-unavailable", s.cause)
			}
			if s.outcome != "gap" {
				t.Errorf("the reachability Gap spells the outcome %q, want gap", s.outcome)
			}
		}
	}
	if !closed {
		t.Error("the replayed reached span stayed open, so it keeps voting under the fold (ADR-0080)")
	}
	if !gapped {
		t.Error("no Gap opened behind the closure, so the timeline reads silent rather than unmeasurable")
	}

	for _, s := range spansOf(t, tx, resolverOnly) {
		if s.id == kept && s.closed {
			t.Error("the resolver-only vantage's span closed, so the default position reads " +
				"unmeasurable until its next walk (ADR-2087 section 6)")
		}
	}
	if got := availabilityOf(t, tx, resolverOnly); got != "available" {
		t.Errorf("the resolver-only vantage reads %q, and the restore re-read nothing to move "+
			"it (#2200)", got)
	}
}

func insertResolverOnlyVantage(t *testing.T, tx pgx.Tx, name string) int64 {
	t.Helper()
	var id int64
	err := tx.QueryRow(context.Background(),
		`INSERT INTO vantage (name, resolver, availability)
		 VALUES ($1, '127.0.0.11:53', 'available')
		 RETURNING id`, name).Scan(&id)
	if err != nil {
		t.Fatalf("insert resolver-only vantage %s: %v", name, err)
	}
	return id
}

func containsID(ids []int64, want int64) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}
