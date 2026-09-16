package main

import (
	"net/netip"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/exposure"
	"github.com/winniel123/verge-asm/internal/signal"
)

// ADR-2087 rules Alternative B: a vantage that becomes unavailable closes its open spans
// at write time, and no composition read gains an availability predicate. These read the
// corpus that writer leaves.

const unavailableGap = `{"outcome":"gap","cause":"vantage-unavailable","reason":"we could not look from this position"}`

func classCovered() func(netip.Addr) bool {
	scope := netip.MustParsePrefix("10.0.0.0/8")
	return func(a netip.Addr) bool { return scope.Contains(a.Unmap()) }
}

func TestTheWrittenGapCarriesTheCauseTheLedgerDedupsOn(t *testing.T) {
	// The writer spells the cause in SQL and the Coverage ledger reads it back here, so a
	// rename on one side would silently double-count the Gap (#2090).
	if !strings.Contains(unavailableGap, `"cause":"`+vantageUnavailableCause+`"`) {
		t.Fatalf("the Gap MarkVantageUnavailable writes must carry cause %q; got %s",
			vantageUnavailableCause, unavailableGap)
	}
	if decodeReachability([]byte(unavailableGap)).Cause != vantageUnavailableCause {
		t.Error("the reach decoder must read the writer's cause off the span value")
	}
}

func TestADeadProbersGapStopsPinningTheClassToReached(t *testing.T) {
	const svc = "198.51.100.9:443/tcp"
	opened := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	live := reachLegRow{
		subject: svc, dialled: "203.0.113.2",
		value: []byte(`{"outcome":"not-reached"}`), openedAt: opened.Add(time.Hour), id: 2,
	}

	// #2060: the dead prober's span is open at `reached`, so the existential fold pins the
	// leg to reached however long it stays dark.
	stale := reachLegRow{
		subject: svc, dialled: "203.0.113.1",
		value: []byte(`{"outcome":"reached"}`), openedAt: opened, id: 1,
	}
	before := collapseReachLegs([]reachLegRow{stale, live}, classCovered())
	if got := legFrom(before[svc]["internet"]); got.Value != exposure.Reached {
		t.Fatalf("the defect this ticket names must reproduce over an open reached span; leg = %+v", got)
	}

	// ADR-2087's writer closed that span and opened a Gap in its place.
	gapped := stale
	gapped.value, gapped.isGap, gapped.id = []byte(unavailableGap), true, 3
	after := collapseReachLegs([]reachLegRow{gapped, live}, classCovered())

	want := exposure.Leg{Status: exposure.LegValued, Value: exposure.NotReached}
	if got := legFrom(after[svc]["internet"]); got != want {
		t.Errorf("leg = %+v, want %+v: a vantage that stopped answering casts no vote, so the "+
			"live prober's not-reached settles the class (ADR-0080, #2060)", got, want)
	}
}

func TestAClassWhoseOnlyVantageWentUnavailableStoppedLooking(t *testing.T) {
	const svc = "198.51.100.9:443/tcp"
	rows := []reachLegRow{{
		subject: svc, dialled: "203.0.113.1", value: []byte(unavailableGap),
		isGap: true, openedAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), id: 1,
	}}
	info := collapseReachLegs(rows, classCovered())[svc]["internet"]

	if got := legFrom(info); got.Status != exposure.LegGap {
		t.Fatalf("leg status = %q, want %q: an empty in-scope set is not vacuously not-reached "+
			"(ADR-0080 decision rule 3)", got.Status, exposure.LegGap)
	}
	if !slices.Contains(info.causes, "vantage-unavailable") {
		t.Errorf("leg causes = %v, want the vantage-unavailable cause: ADR-0017 decision 4 "+
			"separates a leg that stopped looking from one never configured", info.causes)
	}
}

func TestTheRunningClassSetKeepsEveryVantageRow(t *testing.T) {
	// ADR-2087 refuses a read-time predicate here: applyAvailability restores a vantage only
	// from a completed resolution-walk batch, so a filtered dispatch read would strand it.
	txt := func(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }
	rows := []db.ListVantagesForDispatchRow{
		{ID: 1, DialledAddr: txt("203.0.113.1")},
		{ID: 2, DialledAddr: txt("10.0.0.5")},
	}
	got := runningVantageClasses(rows, classCovered())
	want := []string{"internal", "internet"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runningVantageClasses = %v, want %v", got, want)
	}
}

func TestAClassThatStoppedSupplyingAValueLeavesTheComparisonUnmade(t *testing.T) {
	// #2077: the internet vantage went unavailable and its observations aged past the
	// cadence floor, so the class names itself with no current value.
	running := []string{"internal", "internet"}
	classes := map[string]resolutionValue{"internal": {Outcome: signal.Resolved, Addresses: []string{"10.0.0.5"}}}

	got := composeResolution(classes, running, false)
	if got.outcome != signal.ResolutionNotEvaluable {
		t.Errorf("outcome = %q, want %q: a class with no available vantage leaves the "+
			"comparison unmade (ADR-0080 decision rule 1)", got.outcome, signal.ResolutionNotEvaluable)
	}
	if !got.inEstate {
		t.Error("a Name must not leave the estate because one vantage went dark (ADR-0006, ADR-0080)")
	}
}

func TestCollapseNameResolutionsComposesWithinTheClass(t *testing.T) {
	txt := func(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }
	at := func(h int) pgtype.Timestamptz {
		return pgtype.Timestamptz{Time: time.Date(2026, 9, 1, h, 0, 0, 0, time.UTC), Valid: true}
	}
	row := func(id int64, dialled, value string, h int) db.ListNameResolutionsByClassRow {
		return db.ListNameResolutionsByClassRow{
			SubjectKey: "www.example.com", ID: id, DialledAddr: txt(dialled),
			Value: []byte(value), ObservedAt: at(h),
		}
	}

	cases := []struct {
		name string
		rows []db.ListNameResolutionsByClassRow
		want resolutionValue
	}{
		{
			// ADR-0070 measured this: one wildcarded name drew two disjoint pools in one week.
			name: "two pools within the class union rather than the newest winning",
			rows: []db.ListNameResolutionsByClassRow{
				row(1, "203.0.113.1", `{"outcome":"Resolved","addresses":["203.0.113.50"]}`, 1),
				row(2, "203.0.113.2", `{"outcome":"Resolved","addresses":["203.0.113.51"]}`, 2),
			},
			want: resolutionValue{Outcome: signal.Resolved, Addresses: []string{"203.0.113.50", "203.0.113.51"}},
		},
		{
			name: "one answer establishes the presence claim, whatever answered last",
			rows: []db.ListNameResolutionsByClassRow{
				row(1, "203.0.113.1", `{"outcome":"Resolved","addresses":["203.0.113.50"]}`, 1),
				row(2, "203.0.113.2", `{"outcome":"NameError"}`, 2),
			},
			want: resolutionValue{Outcome: signal.Resolved, Addresses: []string{"203.0.113.50"}},
		},
		{
			name: "an absence claim needs every vantage of the class",
			rows: []db.ListNameResolutionsByClassRow{
				row(1, "203.0.113.1", `{"outcome":"NameError"}`, 1),
				row(2, "203.0.113.2", `{"outcome":"NameError"}`, 2),
			},
			want: resolutionValue{Outcome: signal.NameError},
		},
		{
			name: "two absences that disagree are variance no quantifier settles",
			rows: []db.ListNameResolutionsByClassRow{
				row(1, "203.0.113.1", `{"outcome":"NameError"}`, 1),
				row(2, "203.0.113.2", `{"outcome":"NoData"}`, 2),
			},
			want: resolutionValue{Outcome: signal.ResolutionNotEvaluable},
		},
		{
			name: "a vantage that could not look casts no vote",
			rows: []db.ListNameResolutionsByClassRow{
				row(1, "203.0.113.1", `{"outcome":"Gap"}`, 2),
				row(2, "203.0.113.2", `{"outcome":"NameError"}`, 1),
			},
			want: resolutionValue{Outcome: signal.NameError},
		},
		{
			name: "a class that only gapped holds the Gap",
			rows: []db.ListNameResolutionsByClassRow{row(1, "203.0.113.1", `{"outcome":"Gap"}`, 1)},
			want: resolutionValue{Outcome: signal.Gap},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := collapseNameResolutions(c.rows, classCovered())["www.example.com"]["internet"]
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("class value = %+v, want %+v: the quantifiers close at two, and recency "+
					"is a third rule (ADR-0080 decision rule 1, #2058)", got, c.want)
			}
		})
	}
}
