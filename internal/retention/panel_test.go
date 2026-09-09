package retention

import (
	"testing"
	"time"
)

var shipped = []ScanCadence{
	{Kind: "dns", CadenceSeconds: 86400},
	{Kind: "hot", CadenceSeconds: 86400},
	{Kind: "tls-acceptance", CadenceSeconds: 7 * 86400},
	{Kind: "zone", CadenceSeconds: 30 * 86400},
}

func TestFloorsAreAMultipleAndANamedScan(t *testing.T) {
	obs := ObservationFloor(shipped)
	if !obs.Bounded() || obs.Multiple != FloorCadences || obs.ScanKind != "dns" {
		t.Fatalf("observation floor = %+v, want k=%d over dns", obs, FloorCadences)
	}
	if obs.Days() != 2 {
		t.Errorf("observation floor days = %d, want 2", obs.Days())
	}
	disp := DispatchFloor(shipped)
	if !disp.Bounded() || disp.ScanKind != "zone" {
		t.Fatalf("dispatch floor = %+v, want the slowest enabled Scan, zone", disp)
	}
	// The two floors come apart on any install with no zone file (ADR-0081, ADR-0094).
	if obs.ScanKind == disp.ScanKind {
		t.Errorf("the two floors collapsed onto one Scan %q", obs.ScanKind)
	}
}

func TestAFloorWithNoEnabledScanIsUnbounded(t *testing.T) {
	for _, scans := range [][]ScanCadence{nil, {{Kind: "zone", CadenceSeconds: 0}}} {
		if f := ObservationFloor(scans); f.Bounded() || f.Days() != 0 {
			t.Errorf("ObservationFloor(%v) = %+v, want unbounded", scans, f)
		}
	}
}

func TestPairFloorsSortTightestFirst(t *testing.T) {
	pairs := []PairFloor{
		{Facet: "dns-record", Source: "zone", Floor: FloorExpression{Multiple: 2, ScanKind: "zone", CadenceSeconds: 30 * 86400}},
		{Facet: "reachability", Source: "resolver", Floor: FloorExpression{}},
		{Facet: "dns-record", Source: "resolver", Floor: FloorExpression{Multiple: 2, ScanKind: "dns", CadenceSeconds: 86400}},
	}
	SortPairFloors(pairs)
	got := []string{pairs[0].Source, pairs[1].Source, pairs[2].Facet}
	want := []string{"resolver", "zone", "reachability"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pair order = %v, want %v (an unbounded pair sorts last)", got, want)
		}
	}
	// The one v1 facet-source pair carries two different covering Scans (ADR-0094).
	if pairs[0].Floor.ScanKind == pairs[1].Floor.ScanKind {
		t.Errorf("dns-record's two sources resolved to one Scan %q", pairs[0].Floor.ScanKind)
	}
}

func TestRetentionNeverTakesTheInk(t *testing.T) {
	now := time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)
	rows := OrderClamps([]Clamp{
		{Kind: ClampBatch, Label: "First batch", At: now.Add(-400 * 24 * time.Hour), Bounded: true},
		{Kind: ClampBreak, Label: "Break", Leaf: "resolution-walk", At: now.Add(-30 * 24 * time.Hour), Bounded: true},
		{Kind: ClampRetention, Label: "Observation retention", At: now.Add(-10 * 24 * time.Hour), Bounded: true},
		{Kind: ClampRetention, Label: "Dispatch retention", Bounded: false},
	})
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4", len(rows))
	}
	if rows[0].Kind != ClampRetention || !rows[0].Inert {
		t.Fatalf("the tightest retention row must sort first and render inert; got %+v", rows[0])
	}
	if !rows[1].Binding || rows[1].Leaf != "resolution-walk" {
		t.Fatalf("the ink must fall on the Break that names the leaf; got %+v", rows[1])
	}
	for i, r := range rows {
		if r.Binding && r.Inert {
			t.Errorf("row %d is both inert and in ink", i)
		}
		if r.Kind == ClampRetention && r.Binding {
			t.Errorf("row %d: retention took the ink", i)
		}
	}
	last := rows[len(rows)-1]
	if last.Bounded || !last.Inert {
		t.Errorf("an unbounded dial must sort last and render inert; got %+v", last)
	}
}

func TestOnlyOneRowTakesTheInk(t *testing.T) {
	now := time.Now()
	rows := OrderClamps([]Clamp{
		{Kind: ClampBatch, Label: "First batch", At: now.Add(-100 * time.Hour), Bounded: true},
		{Kind: ClampBreak, Label: "Break", Leaf: "a", At: now.Add(-50 * time.Hour), Bounded: true},
		{Kind: ClampBreak, Label: "Break", Leaf: "b", At: now.Add(-20 * time.Hour), Bounded: true},
	})
	ink := 0
	for _, r := range rows {
		if r.Binding {
			ink++
		}
	}
	if ink != 1 || rows[0].Leaf != "b" {
		t.Fatalf("ink=%d, first leaf=%q; want exactly one, on the most recent Break", ink, rows[0].Leaf)
	}
}

func TestProjectionStatesItHasNoDenominator(t *testing.T) {
	p := Project(1000, 0, 0)
	if p.HasDenominator || p.RowsPerYear != 0 {
		t.Fatalf("undeclared scope produced a forecast: %+v", p)
	}
	if p.RowsHeld != 1000 || p.BytesHeld != 1000*BytesPerObservation {
		t.Errorf("what is held must still render: %+v", p)
	}
	d := Project(1000, 256, 4)
	if !d.HasDenominator || d.RowsPerYear != 1024 {
		t.Fatalf("declared scope = %+v, want a priced forecast", d)
	}
}

func TestBelowFloorIsUnreachableRatherThanRefused(t *testing.T) {
	stops := DialStops(2, []int64{1, 2, 30, 90, 365})
	if stops[0] != 2 {
		t.Fatalf("stops[0] = %d, want the floor", stops[0])
	}
	for _, s := range stops {
		if s < 2 {
			t.Fatalf("stop %d lies in the ground below the floor", s)
		}
	}
	if got := ClampToFloor(1, 2); got != 2 {
		t.Errorf("ClampToFloor(1,2) = %d, want 2 — raised, never refused", got)
	}
	// Zero is the terminal stop the handle parks on, not a below-floor value.
	if got := ClampToFloor(0, 2); got != 0 {
		t.Errorf("ClampToFloor(0,2) = %d, want 0 (keep everything)", got)
	}
	if got := ClampToFloor(90, 2); got != 90 {
		t.Errorf("ClampToFloor(90,2) = %d, want 90", got)
	}
}

func TestABreakNamesEveryLeafThatMoved(t *testing.T) {
	prev := []byte(`[{"leaf":"resolution-walk","version":"3"},{"leaf":"wildcard-discrim","version":"1"}]`)
	cur := []byte(`[{"leaf":"resolution-walk","version":"4"},{"leaf":"wildcard-discrim","version":"1"}]`)
	if got := MovedLeaves(prev, cur); len(got) != 1 || got[0] != "resolution-walk" {
		t.Errorf("MovedLeaves = %v, want [resolution-walk]", got)
	}
	// Two leaves moving in one transition are two Breaks, and both must be named.
	both := []byte(`[{"leaf":"resolution-walk","version":"4"},{"leaf":"wildcard-discrim","version":"2"}]`)
	if got := MovedLeaves(prev, both); len(got) != 2 || got[0] != "resolution-walk" || got[1] != "wildcard-discrim" {
		t.Errorf("MovedLeaves on a two-leaf move = %v, want both, sorted", got)
	}
	added := []byte(`[{"leaf":"resolution-walk","version":"3"},{"leaf":"edge-fanout","version":"1"}]`)
	if got := MovedLeaves(prev, added); len(got) != 2 || got[0] != "edge-fanout" || got[1] != "wildcard-discrim" {
		t.Errorf("an added and a dropped leaf both moved: MovedLeaves = %v", got)
	}
	if got := MovedLeaves(prev, prev); len(got) != 0 {
		t.Errorf("MovedLeaves on an unchanged vector = %v, want none", got)
	}
	if got := MovedLeaves([]byte(`not json`), cur); got != nil {
		t.Errorf("MovedLeaves on unreadable input = %v, want nil", got)
	}
}

func TestAProjectionNeverOverflows(t *testing.T) {
	// A /65 passes the int64 gate on the address count and would overflow the year.
	huge := int64(1) << 62
	p := Project(10, huge, 1460)
	if p.HasDenominator || p.RowsPerYear != 0 || p.BytesPerYear != 0 {
		t.Fatalf("an overflowing scope produced a forecast: %+v", p)
	}
	if p.RowsHeld != 10 {
		t.Errorf("what is held must still render: %+v", p)
	}
}
