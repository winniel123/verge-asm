package dbtest_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/dbtest"
)

// ListCitedAddressSpansForNames decides whether an Address still has a citer, and
// uncitedAddresses withdraws it on an empty set. MarkVantageUnavailable opens a Gap over
// every timeline a dark vantage fed, so what the citer read makes of a Gap is a data-loss
// question no text match settles (ADR-0006, #2164).

func cites(subjectKey string, addrs ...string) spanRow {
	quoted := make([]string, 0, len(addrs))
	for _, a := range addrs {
		quoted = append(quoted, `"`+a+`"`)
	}
	return spanRow{
		facet:      "resolution",
		subjectKey: subjectKey,
		source:     "resolver",
		value:      `{"outcome":"Resolved","addresses":[` + strings.Join(quoted, ",") + `]}`,
	}
}

func serviceUnder(addr string) spanRow {
	return spanRow{
		facet:       "reachability",
		subjectKind: "service",
		subjectKey:  addr + ":443/tcp",
		source:      "prober",
		value:       `{"outcome":"open"}`,
	}
}

func closeAsMeasuredAbsent(t *testing.T, tx pgx.Tx, id int64) {
	t.Helper()
	if _, err := tx.Exec(context.Background(),
		`UPDATE span SET closed_at = now(), closure_reason = 'measured-absent' WHERE id = $1`, id); err != nil {
		t.Fatalf("close span %d: %v", id, err)
	}
}

func citersOf(t *testing.T, rows []db.ListCitedAddressSpansForNamesRow, addr string) []string {
	t.Helper()
	for _, r := range rows {
		if r.SubjectKey == addr {
			return r.Citers
		}
	}
	t.Fatalf("%s is not a candidate at all, so the read dropped it: %+v", addr, rows)
	return nil
}

func TestAVantageGapLeavesTheAddressItsGappedNameStillCites(t *testing.T) {
	q, tx := dbtest.Queries(t)
	ctx := context.Background()

	const addr = "203.0.113.164"
	dark := insertVantage(t, tx, "gap-citer-dark")
	steady := insertVantage(t, tx, "gap-citer-steady")

	// The Name whose departure this fold just recorded: its timelines are closed
	// before the citer read runs (ADR-0087).
	closeAsMeasuredAbsent(t, tx, insertSpan(t, tx, steady, cites("departing.example", addr)))
	insertSpan(t, tx, steady, serviceUnder(addr))
	insertSpan(t, tx, dark, cites("survivor.example", addr))

	if err := q.MarkVantageUnavailable(ctx, dark); err != nil {
		t.Fatalf("MarkVantageUnavailable: %v", err)
	}

	rows, err := q.ListCitedAddressSpansForNames(ctx, []string{"departing.example"})
	if err != nil {
		t.Fatalf("ListCitedAddressSpansForNames: %v", err)
	}
	citers := citersOf(t, rows, addr)
	if !slices.Contains(citers, "survivor.example") {
		t.Errorf("citers of %s = %v, want survivor.example: one vantage going dark is the absence "+
			"of a measurement, and an Address leaves only by one (ADR-0006, ADR-0080)", addr, citers)
	}
}

func TestAGapCitesTheAddressItsOwnPreGapValueHeldAndNoOther(t *testing.T) {
	q, tx := dbtest.Queries(t)
	ctx := context.Background()

	const left = "203.0.113.165"
	const elsewhere = "203.0.113.166"
	dark := insertVantage(t, tx, "gap-citer-scope-dark")
	steady := insertVantage(t, tx, "gap-citer-scope-steady")

	closeAsMeasuredAbsent(t, tx, insertSpan(t, tx, steady, cites("departing.example", left)))
	insertSpan(t, tx, steady, serviceUnder(left))
	insertSpan(t, tx, dark, cites("survivor.example", elsewhere))

	if err := q.MarkVantageUnavailable(ctx, dark); err != nil {
		t.Fatalf("MarkVantageUnavailable: %v", err)
	}

	rows, err := q.ListCitedAddressSpansForNames(ctx, []string{"departing.example"})
	if err != nil {
		t.Fatalf("ListCitedAddressSpansForNames: %v", err)
	}
	if citers := citersOf(t, rows, left); len(citers) != 0 {
		t.Errorf("citers of %s = %v, want none: a Gap carries the value beneath it and never a "+
			"blanket citation (#2164)", left, citers)
	}
}

func TestTheSoleCiterDepartingStillLeavesTheAddressUncited(t *testing.T) {
	q, tx := dbtest.Queries(t)
	ctx := context.Background()

	const addr = "203.0.113.167"
	steady := insertVantage(t, tx, "gap-citer-sole-steady")

	closeAsMeasuredAbsent(t, tx, insertSpan(t, tx, steady, cites("departing.example", addr)))
	insertSpan(t, tx, steady, serviceUnder(addr))

	rows, err := q.ListCitedAddressSpansForNames(ctx, []string{"departing.example"})
	if err != nil {
		t.Fatalf("ListCitedAddressSpansForNames: %v", err)
	}
	if citers := citersOf(t, rows, addr); len(citers) != 0 {
		t.Errorf("citers of %s = %v, want none: a measured departure with no Gap anywhere still "+
			"withdraws the Address (ADR-0006)", addr, citers)
	}
}
