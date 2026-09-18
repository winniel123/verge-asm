package queue

import (
	"context"
	"go/ast"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
	"github.com/winniel123/verge-asm/internal/wire"
)

// The defect #2250 reports is a sequence, not a fold: gap, reached, gap, once per dispatch
// cycle, for as long as one resolver outage stands. So each case here drives several cycles.

var flapVantage = pgInt8(9)

var flapTarget = netip.AddrPortFrom(netip.MustParseAddr("203.0.113.77"), 443)

// The reachability branch of MarkVantageUnavailable's CASE, byte for byte: that facet alone
// decodes a lowercase outcome, and it carries a reason the other branches do not (#2183).

var outageGapValue = `{"outcome":"` + connectoutcome.GapOutcome + `","cause":"` + outageGapCause +
	`","reason":"we could not look from this position"}`

func flapReached() []wire.Observation {
	return []wire.Observation{
		connectoutcome.EmitService("flap", "vantage", flapTarget, connectoutcome.Reached, connectoutcome.ConnOpen),
	}
}

// What MarkVantageUnavailable writes: every open span closed, and a Gap carrying the outage
// cause opened behind each (db/queries/vantages.sql, ADR-2087).

func markOutage(store *foldSpanStore, at time.Time) {
	var closed []*foldedSpan
	for _, sp := range store.spans {
		if sp.closedAt.Valid || sp.VantageID != flapVantage {
			continue
		}
		// The guard is on the spans, so a second mark reaches what opened since (#2060).
		if sp.IsGap && string(sp.Value) == outageGapValue {
			continue
		}
		sp.closedAt = tstz(at)
		closed = append(closed, sp)
	}
	for _, sp := range closed {
		store.nextID++
		gap := &foldedSpan{OpenSpanParams: sp.OpenSpanParams, id: store.nextID}
		gap.Value = []byte(outageGapValue)
		gap.IsGap = true
		gap.OpenedAt = tstz(at)
		store.spans = append(store.spans, gap)
	}
}

func openFoldSpans(store *foldSpanStore) []*foldedSpan {
	var out []*foldedSpan
	for _, sp := range store.spans {
		if !sp.closedAt.Valid {
			out = append(out, sp)
		}
	}
	return out
}

func foldFlapBatch(t *testing.T, store *foldSpanStore, batchID int64, at time.Time, outage bool) []spanChange {
	t.Helper()
	var changes []spanChange
	err := foldObservationsIntoSpans(context.Background(), store, batchID, flapVantage, at,
		flapReached(), membershipInputs{}, outage, &changes)
	if err != nil {
		t.Fatalf("fold batch %d: %v", batchID, err)
	}
	return changes
}

func TestAStandingOutageGapsTheLegOnceRatherThanEveryDispatchCycle(t *testing.T) {
	store := &foldSpanStore{}
	at := produceT0

	// A healthy cycle first, so the leg holds a real reading when the resolver dies.
	if changes := foldFlapBatch(t, store, 1, at, false); len(changes) != 1 {
		t.Fatalf("the first connect batch opens the reachability span, got %+v", changes)
	}
	at = at.Add(time.Minute)
	markOutage(store, at)
	settled := len(store.spans)

	for cycle := int64(2); cycle <= 4; cycle++ {
		at = at.Add(time.Minute)
		// The prober is healthy, so the connect batch keeps reporting reached.
		if changes := foldFlapBatch(t, store, cycle, at, true); len(changes) != 0 {
			t.Errorf("cycle %d overwrote the outage Gap with a reading no resolver backs, so the "+
				"Exposure board flips and a GapClosed message fires: %s", cycle, changes[0].Value)
		}
		at = at.Add(time.Second)
		// The resolver is still down, so the next resolution walk dead-letters again.
		markOutage(store, at)
	}

	if got := len(store.spans); got != settled {
		t.Errorf("the span corpus grew from %d rows to %d behind one standing outage, which is a "+
			"closed pair per dispatch cycle recording no change in the estate (#2250)", settled, got)
	}
	open := openFoldSpans(store)
	if len(open) != 1 {
		t.Fatalf("want the outage Gap alone open, got %+v", open)
	}
	if !open[0].IsGap || string(open[0].Value) != outageGapValue {
		t.Errorf("the open span is %s, want the outage Gap: the leg reads stopped-looking for as "+
			"long as the outage stands, which #2165 ruled correct", open[0].Value)
	}
}

func TestTheConnectBatchClosesTheOutageGapOnceTheVantageIsAvailableAgain(t *testing.T) {
	store := &foldSpanStore{}
	at := produceT0

	foldFlapBatch(t, store, 1, at, false)
	at = at.Add(time.Minute)
	markOutage(store, at)

	// Recovery retires no reachability Gap, because no walk re-measures a port (ADR-2087, #2060).
	// The next connect batch is what replaces it, and declining that would make the residual
	// #2189 records permanent rather than momentary.
	at = at.Add(time.Minute)
	changes := foldFlapBatch(t, store, 2, at, false)
	if len(changes) != 1 {
		t.Fatalf("the connect batch after recovery states the reading that retires the Gap, got %+v", changes)
	}
	if !changes[0].PrevIsGap || changes[0].IsGap {
		t.Errorf("the change must close the Gap and open a real reading, got prevGap=%v gap=%v",
			changes[0].PrevIsGap, changes[0].IsGap)
	}
	open := openFoldSpans(store)
	if len(open) != 1 || open[0].IsGap {
		t.Fatalf("want one real reading open, got %+v", open)
	}
}

func TestOnlyABatchThatCouldClearTheOutageMayRetireItsGap(t *testing.T) {
	for _, kind := range []string{connectoutcome.Kind, tlsacceptance.Kind, httpexchange.Kind} {
		if !outageStands(availabilityUnavailableValue, kind) {
			t.Errorf("a %s batch says nothing of resolver health, so it cannot clear the outage "+
				"(ADR-0108) and must not overwrite the Gap the outage opened", kind)
		}
	}

	// applyAvailability runs after this batch's own fold, so the column still reads unavailable
	// while the recovering walk folds. Declining its reading would leave MarkVantageAvailable
	// closing a Gap with nothing behind it, and the leg reads never-configured (ADR-2087 §6).
	if outageStands(availabilityUnavailableValue, resolutionwalk.Kind) {
		t.Error("a completed resolution-walk fold was declined, so recovery retires a Gap no " +
			"reading replaces")
	}

	for _, availability := range []string{"available", "pending", ""} {
		if outageStands(availability, connectoutcome.Kind) {
			t.Errorf("availability %q is no outage, so the fold must write normally", availability)
		}
	}
}

func foldCall(t *testing.T, fn *ast.FuncDecl) *ast.CallExpr {
	t.Helper()
	var found *ast.CallExpr
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "foldObservationsIntoSpans" && found == nil {
			found = call
		}
		return true
	})
	if found == nil {
		t.Fatal("Worker.complete folds no observations; the gate lost the tree, not the rule")
	}
	return found
}

func TestTheFoldIsToldWhetherTheOutageStands(t *testing.T) {
	// The guard sits in foldOne, so a caller that hands it a literal false re-opens the flap
	// with every case above still passing (#2250).
	fn := completeDecl(t)
	read := firstCallPos(t, fn, "availabilityForFold")
	call := foldCall(t, fn)
	if read > call.Pos() {
		t.Errorf("Worker.complete reads the vantage's availability at %d, after it folds at %d, so "+
			"the fold is told nothing", read, call.Pos())
	}
	for _, arg := range call.Args {
		inner, ok := arg.(*ast.CallExpr)
		if !ok {
			continue
		}
		if id, ok := inner.Fun.(*ast.Ident); ok && id.Name == "outageStands" {
			return
		}
	}
	t.Error("the fold is passed no outageStands call, so a batch that cannot clear the outage " +
		"retires its Gap again")
}

func TestTheOutageCauseIsTheOneTheWriterSpells(t *testing.T) {
	// The guard turns on a Go literal matching a SQL one, and nothing else holds the pair: a
	// rename in the query would make isOutageGap read false for every span, silently.
	body, err := os.ReadFile(filepath.Join("..", "..", "db", "queries", "vantages.sql"))
	if err != nil {
		t.Fatalf("read the vantage queries: %v", err)
	}
	want := `"cause":"` + outageGapCause + `"`
	if !strings.Contains(string(body), want) {
		t.Errorf("MarkVantageUnavailable opens no Gap spelling %s, so the fold guards a cause "+
			"nothing writes (#2250)", want)
	}
}
