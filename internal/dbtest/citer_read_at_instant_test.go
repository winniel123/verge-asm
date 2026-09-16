package dbtest_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/dbtest"
)

// ListResolutionCitersForAddressesAt is the third read carrying the Gap fallback,
// and the re-point fold reads the estate through it alone (internal/queue/
// repointsettle.go). #2204 pinned the other two against each other and left this
// one held by a text match, which cannot see all three limbs move together
// (ADR-0006, #2255).

type citerAgreementCase struct {
	addr string
	want []string
	why  string
}

var citerAgreementCases = []citerAgreementCase{
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
}

func citersByInstantRead(rows []db.ListResolutionCitersForAddressesAtRow) map[string][]string {
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

func txInstant(t *testing.T, tx pgx.Tx) pgtype.Timestamptz {
	t.Helper()
	var at pgtype.Timestamptz
	// The transaction's own now(), which every seeded span carries: a case reading
	// the wall clock would sit past the Gap by a margin nothing here controls.
	if err := tx.QueryRow(context.Background(), `SELECT now()`).Scan(&at); err != nil {
		t.Fatalf("read transaction instant: %v", err)
	}
	return at
}

func TestTheInstantCiterReadNamesTheSameCitersAsBothOthersBehindAGap(t *testing.T) {
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
	instantRows, err := q.ListResolutionCitersForAddressesAt(ctx, db.ListResolutionCitersForAddressesAtParams{
		Addresses: []string{agreeCitedAddr, agreeOtherAddr},
		At:        txInstant(t, tx),
	})
	if err != nil {
		t.Fatalf("ListResolutionCitersForAddressesAt: %v", err)
	}

	byTimeline := citersByTimelineRead(timelineRows)
	byInstant := citersByInstantRead(instantRows)

	for _, c := range citerAgreementCases {
		bySpan := sortedUnique(citersOf(t, spanRows, c.addr))
		want := sortedUnique(c.want)

		// Each read is pinned to the answer, not only to the other two: agreement
		// alone is satisfiable by regressing all three limbs together, which is the
		// mutation the text match in internal/db survives (#2204).
		for read, got := range map[string][]string{
			"ListCitedAddressSpansForNames":      bySpan,
			"ListResolutionCitersForAddresses":   byTimeline[c.addr],
			"ListResolutionCitersForAddressesAt": byInstant[c.addr],
		} {
			if !slices.Equal(got, want) {
				t.Errorf("%s says %v cites %s, want %v: %s", read, got, c.addr, want, c.why)
			}
		}

		if !slices.Equal(byInstant[c.addr], byTimeline[c.addr]) {
			t.Errorf("the instant read and the timeline read disagree about who cites %s: %v against %v. "+
				"The re-point fold reads the estate through the instant read alone, so a Gap means the "+
				"same to it or the fold calls a held Address new (#1730, #1818)",
				c.addr, byInstant[c.addr], byTimeline[c.addr])
		}
	}
}
