package dbtest_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/dbtest"
)

// ListCitedAddressSpansForNames and ListResolutionCitersForAddresses answer one
// question — which Names cite this Address — and they parted over a Gap until
// #2164. internal/db.TestEveryCiterReadTakesTheGapFallback matches the repaired
// text; matching text cannot say the two reads return the same Names, which is
// what uncitedAddresses and the re-point fold each rely on (#2204).

const (
	agreeCitedAddr   = "203.0.113.168"
	agreeOtherAddr   = "203.0.113.169"
	agreeValuedName  = "valued.agree.example"
	agreeGappedName  = "gapped.agree.example"
	agreeMovedName   = "movedaway.agree.example"
	agreeRetiredName = "retired.agree.example"
)

func seedCiterAgreementEstate(t *testing.T, tx pgx.Tx, q *db.Queries) {
	t.Helper()
	ctx := context.Background()

	steady := insertVantage(t, tx, "citer-agree-steady")
	dark := insertVantage(t, tx, "citer-agree-dark")

	// ListCitedAddressSpansForNames drops a candidate holding nothing, so without
	// these the comparison reaches neither Address (#2033).
	insertSpan(t, tx, steady, serviceUnder(agreeCitedAddr))
	insertSpan(t, tx, steady, serviceUnder(agreeOtherAddr))

	insertSpan(t, tx, steady, cites(agreeValuedName, agreeCitedAddr))

	// A timeline that ended rather than went dark: no Gap stands over it.
	closeAsMeasuredAbsent(t, tx, insertSpan(t, tx, steady, cites(agreeRetiredName, agreeCitedAddr)))

	insertSpan(t, tx, dark, cites(agreeGappedName, agreeCitedAddr))
	insertSpan(t, tx, dark, cites(agreeMovedName, agreeOtherAddr))
	// The real writer, so the Gaps under test are the ones ADR-2087 opens rather
	// than a shape this file invented.
	if err := q.MarkVantageUnavailable(ctx, dark); err != nil {
		t.Fatalf("MarkVantageUnavailable: %v", err)
	}
}

func citersByTimelineRead(rows []db.ListResolutionCitersForAddressesRow) map[string][]string {
	names := map[string][]string{}
	for _, r := range rows {
		names[r.Addr] = append(names[r.Addr], r.SubjectKey)
	}
	out := map[string][]string{}
	for addr, n := range names {
		out[addr] = sortedUnique(n)
	}
	return out
}

func sortedUnique(in []string) []string {
	out := slices.Clone(in)
	slices.Sort(out)
	return slices.Compact(out)
}

func TestBothCiterReadsNameTheSameCitersBehindAGap2204(t *testing.T) {
	q, tx := dbtest.Queries(t)
	ctx := context.Background()

	seedCiterAgreementEstate(t, tx, q)

	spanRows, err := q.ListCitedAddressSpansForNames(ctx,
		[]string{agreeValuedName, agreeGappedName, agreeMovedName, agreeRetiredName})
	if err != nil {
		t.Fatalf("ListCitedAddressSpansForNames: %v", err)
	}
	timelineRows, err := q.ListResolutionCitersForAddresses(ctx, []string{agreeCitedAddr, agreeOtherAddr})
	if err != nil {
		t.Fatalf("ListResolutionCitersForAddresses: %v", err)
	}
	byTimeline := citersByTimelineRead(timelineRows)

	for _, c := range []struct {
		addr string
		want []string
		why  string
	}{
		{
			addr: agreeCitedAddr,
			want: []string{agreeGappedName, agreeValuedName},
			why: "the dark vantage's Gap still cites what its pre-Gap value held, and the timeline " +
				"closed measured-absent cites nothing",
		},
		{
			addr: agreeOtherAddr,
			want: []string{agreeMovedName},
			why:  "a Gap cites the Address its own pre-Gap value named and no other",
		},
	} {
		bySpan := sortedUnique(citersOf(t, spanRows, c.addr))
		want := sortedUnique(c.want)

		if !slices.Equal(bySpan, byTimeline[c.addr]) {
			t.Errorf("the two reads disagree about who cites %s: ListCitedAddressSpansForNames says %v, "+
				"ListResolutionCitersForAddresses says %v. A Gap means the same to both, or uncitedAddresses "+
				"withdraws an Address the re-point fold still sees cited (ADR-0006, #2164)",
				c.addr, bySpan, byTimeline[c.addr])
		}

		// Agreement alone is satisfiable by regressing both reads together, and
		// the text match in internal/db survives that mutation, so each read is
		// also pinned to the answer.
		for read, got := range map[string][]string{
			"ListCitedAddressSpansForNames":    bySpan,
			"ListResolutionCitersForAddresses": byTimeline[c.addr],
		} {
			if !slices.Equal(got, want) {
				t.Errorf("%s says %v cites %s, want %v: %s", read, got, c.addr, want, c.why)
			}
		}
	}
}
