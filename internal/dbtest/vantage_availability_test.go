package dbtest_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/dbtest"
)

// internal/vantage/availability_test.go proves the two availability writers by
// matching their text. These run them: the closure and the reopening are one
// data-modifying statement over a timeline held to one open span by a partial
// unique index, and no reading of the text settles what that statement does
// (ADR-2087, #2166).

type spanRow struct {
	facet      string
	subjectKey string
	source     string
	value      string
	isGap      bool
}

func valued(facet, subjectKey string) spanRow {
	return spanRow{facet: facet, subjectKey: subjectKey, source: "resolver", value: `{"outcome":"Resolved"}`}
}

func gap(facet, subjectKey, cause string) spanRow {
	return spanRow{
		facet:      facet,
		subjectKey: subjectKey,
		source:     "resolver",
		value:      `{"outcome":"Gap","cause":"` + cause + `"}`,
		isGap:      true,
	}
}

func insertVantage(t *testing.T, tx pgx.Tx, name string) int64 {
	t.Helper()
	var id int64
	err := tx.QueryRow(context.Background(),
		`INSERT INTO vantage (name, resolver, host, port, username, availability)
		 VALUES ($1, '9.9.9.9', $1 || '.example', 22, 'verge', 'available')
		 RETURNING id`, name).Scan(&id)
	if err != nil {
		t.Fatalf("insert vantage %s: %v", name, err)
	}
	return id
}

func insertSpan(t *testing.T, tx pgx.Tx, vantageID int64, row spanRow) int64 {
	t.Helper()
	var id int64
	err := tx.QueryRow(context.Background(),
		`INSERT INTO span (subject_kind, subject_key, facet, discriminator, vantage_id,
		                   source, value, is_gap, derivation, opened_at)
		 VALUES ('name', $1, $2, '', $3, $4, $5::jsonb, $6, '[]'::jsonb, now())
		 RETURNING id`,
		row.subjectKey, row.facet, vantageID, row.source, row.value, row.isGap).Scan(&id)
	if err != nil {
		t.Fatalf("insert %s span for %s: %v", row.facet, row.subjectKey, err)
	}
	return id
}

type storedSpan struct {
	id      int64
	facet   string
	outcome string
	cause   string
	isGap   bool
	closed  bool
}

func spansOf(t *testing.T, tx pgx.Tx, vantageID int64) []storedSpan {
	t.Helper()
	rows, err := tx.Query(context.Background(),
		`SELECT id, facet, COALESCE(value ->> 'outcome', ''), COALESCE(value ->> 'cause', ''),
		        is_gap, closed_at IS NOT NULL
		 FROM span WHERE vantage_id = $1 ORDER BY id`, vantageID)
	if err != nil {
		t.Fatalf("read spans: %v", err)
	}
	defer rows.Close()

	var out []storedSpan
	for rows.Next() {
		var s storedSpan
		if err := rows.Scan(&s.id, &s.facet, &s.outcome, &s.cause, &s.isGap, &s.closed); err != nil {
			t.Fatalf("scan span: %v", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read spans: %v", err)
	}
	return out
}

func openSpans(spans []storedSpan) []storedSpan {
	var out []storedSpan
	for _, s := range spans {
		if !s.closed {
			out = append(out, s)
		}
	}
	return out
}

func availabilityOf(t *testing.T, tx pgx.Tx, vantageID int64) string {
	t.Helper()
	var availability string
	if err := tx.QueryRow(context.Background(),
		`SELECT availability FROM vantage WHERE id = $1`, vantageID).Scan(&availability); err != nil {
		t.Fatalf("read availability: %v", err)
	}
	return availability
}

func TestMarkVantageUnavailableClosesEveryOpenSpanAndReopensItAsAGap(t *testing.T) {
	q, tx := dbtest.Queries(t)
	ctx := context.Background()

	vantageID := insertVantage(t, tx, "unavailable-closes")
	before := map[int64]string{}
	for _, row := range []spanRow{
		valued("resolution", "a.example"),
		valued("dns-record", "a.example"),
		valued("reachability", "a.example:443"),
	} {
		before[insertSpan(t, tx, vantageID, row)] = row.facet
	}

	// The reopening INSERT writes the timeline the same statement's UPDATE just
	// closed, under span_open_timeline_idx. Whether the closed tuple is visible
	// to that uniqueness check inside one command is the question no text match
	// answers.
	if err := q.MarkVantageUnavailable(ctx, vantageID); err != nil {
		t.Fatalf("MarkVantageUnavailable did not execute: %v", err)
	}

	if got := availabilityOf(t, tx, vantageID); got != "unavailable" {
		t.Errorf("availability = %q, want unavailable", got)
	}

	spans := spansOf(t, tx, vantageID)
	if len(spans) != 6 {
		t.Fatalf("got %d spans, want the 3 originals plus one Gap each: %+v", len(spans), spans)
	}
	for _, s := range spans {
		if _, original := before[s.id]; original && !s.closed {
			t.Errorf("the %s span stayed open behind a dead vantage", s.facet)
		}
	}

	gaps := map[string]storedSpan{}
	for _, s := range openSpans(spans) {
		if _, original := before[s.id]; original {
			continue
		}
		if !s.isGap {
			t.Errorf("the %s replacement span is not a Gap: outcome %q", s.facet, s.outcome)
		}
		gaps[s.facet] = s
	}
	for facet, wantOutcome := range map[string]string{
		// Each facet's own writer owns the spelling, so the CASE branch and its
		// default are read back per facet rather than as one literal (#2183).
		"resolution":   "Gap",
		"dns-record":   "Gap",
		"reachability": "gap",
	} {
		g, ok := gaps[facet]
		if !ok {
			t.Errorf("%s opened no Gap, so its timeline is silent rather than unmeasurable", facet)
			continue
		}
		if g.outcome != wantOutcome {
			t.Errorf("the %s Gap spells the outcome %q, want %q", facet, g.outcome, wantOutcome)
		}
		if g.cause != "vantage-unavailable" {
			t.Errorf("the %s Gap carries the cause %q, want vantage-unavailable", facet, g.cause)
		}
	}
}

func TestMarkVantageUnavailableTouchesNoOtherVantagesSpans(t *testing.T) {
	q, tx := dbtest.Queries(t)
	ctx := context.Background()

	dying := insertVantage(t, tx, "unavailable-scoped-dying")
	healthy := insertVantage(t, tx, "unavailable-scoped-healthy")
	insertSpan(t, tx, dying, valued("resolution", "a.example"))
	insertSpan(t, tx, healthy, valued("resolution", "a.example"))

	if err := q.MarkVantageUnavailable(ctx, dying); err != nil {
		t.Fatalf("MarkVantageUnavailable: %v", err)
	}

	spans := spansOf(t, tx, healthy)
	if len(spans) != 1 || spans[0].closed {
		t.Errorf("one vantage's outage reached another's spans: %+v", spans)
	}
	if got := availabilityOf(t, tx, healthy); got != "available" {
		t.Errorf("the healthy vantage's availability = %q, want available", got)
	}
}

func TestMarkVantageUnavailableReachesTheSpansOpenedSinceTheFirstMark(t *testing.T) {
	q, tx := dbtest.Queries(t)
	ctx := context.Background()

	vantageID := insertVantage(t, tx, "unavailable-twice")
	insertSpan(t, tx, vantageID, valued("resolution", "a.example"))

	if err := q.MarkVantageUnavailable(ctx, vantageID); err != nil {
		t.Fatalf("first mark: %v", err)
	}

	// Availability clears only from a completed resolution-walk batch, so a
	// connect batch keeps opening reached spans while the position is dark
	// (#2060).
	late := insertSpan(t, tx, vantageID, valued("reachability", "a.example:443"))

	if err := q.MarkVantageUnavailable(ctx, vantageID); err != nil {
		t.Fatalf("second mark: %v", err)
	}

	spans := spansOf(t, tx, vantageID)
	for _, s := range spans {
		if s.id == late && !s.closed {
			t.Error("the second mark left a span opened during the outage voting")
		}
	}
	// The original span, its Gap, the late span, and the late span's Gap. A
	// re-marked Gap would add a fifth.
	if len(spans) != 4 {
		t.Errorf("got %d spans, want 4: a re-mark must reopen no Gap it already opened: %+v", len(spans), spans)
	}
	open := openSpans(spans)
	if len(open) != 2 {
		t.Errorf("got %d open spans, want one Gap per timeline: %+v", len(open), open)
	}
	for _, s := range open {
		if !s.isGap {
			t.Errorf("a valued %s span outlived the second mark", s.facet)
		}
	}
}

func TestMarkVantageAvailableRetiresTheOutageGapOnTheNamedFacetsAlone(t *testing.T) {
	q, tx := dbtest.Queries(t)
	ctx := context.Background()

	vantageID := insertVantage(t, tx, "available-scoped")
	if err := q.MarkVantageUnavailable(ctx, vantageID); err != nil {
		t.Fatalf("mark unavailable: %v", err)
	}
	for _, row := range []spanRow{
		valued("resolution", "a.example"),
		valued("dns-record", "a.example"),
		valued("reachability", "a.example:443"),
	} {
		insertSpan(t, tx, vantageID, row)
	}
	if err := q.MarkVantageUnavailable(ctx, vantageID); err != nil {
		t.Fatalf("mark unavailable again: %v", err)
	}

	// The recovering batch re-read two facets, so the reachability Gap has
	// nothing to replace it and must stay open (ADR-2087, #2060).
	before := len(spansOf(t, tx, vantageID))
	if err := q.MarkVantageAvailable(ctx, db.MarkVantageAvailableParams{
		ID:     vantageID,
		Facets: []string{"resolution", "dns-record"},
	}); err != nil {
		t.Fatalf("MarkVantageAvailable did not execute: %v", err)
	}

	if got := availabilityOf(t, tx, vantageID); got != "available" {
		t.Errorf("availability = %q, want available", got)
	}
	spans := spansOf(t, tx, vantageID)
	if len(spans) != before {
		t.Errorf("recovery states no reading, so it opens no span: %d spans, want %d", len(spans), before)
	}
	open := openSpans(spans)
	if len(open) != 1 {
		t.Fatalf("got %d open spans, want the reachability Gap alone: %+v", len(open), open)
	}
	if open[0].facet != "reachability" {
		t.Errorf("recovery retired the %s Gap, which no facet it names replaces", open[0].facet)
	}
}

func TestMarkVantageAvailableLeavesEveryOtherCauseAndValueAlone(t *testing.T) {
	q, tx := dbtest.Queries(t)
	ctx := context.Background()

	vantageID := insertVantage(t, tx, "available-other-causes")
	blanket := insertSpan(t, tx, vantageID, gap("resolution", "blanket.example", "blanket-responder"))
	reached := insertSpan(t, tx, vantageID, valued("resolution", "reached.example"))

	if err := q.MarkVantageAvailable(ctx, db.MarkVantageAvailableParams{
		ID:     vantageID,
		Facets: []string{"resolution", "dns-record", "reachability"},
	}); err != nil {
		t.Fatalf("MarkVantageAvailable: %v", err)
	}

	for _, s := range spansOf(t, tx, vantageID) {
		switch s.id {
		case blanket:
			if s.closed {
				t.Error("recovery closed a blanket-responder Gap, which outlives the outage")
			}
		case reached:
			if s.closed {
				t.Error("recovery ended a valued reading it never read")
			}
		}
	}
}
