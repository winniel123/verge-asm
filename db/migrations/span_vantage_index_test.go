package migrations

import (
	"strings"
	"testing"
)

const openSpanPredicate = "closed_at is null"

func TestSpanIndexesTheVantageTheAvailabilityWritersFilterOn(t *testing.T) {
	// MarkVantageAvailable runs on every completed resolution-walk batch inside the job
	// transaction, and span_open_timeline_idx leads on subject_key, which it does not
	// constrain (#2179).
	idx, names := indexesLeadingOn(t, "span", "vantage_id")

	if len(names) == 0 {
		t.Fatalf("no span index leads on vantage_id; MarkVantageUnavailable and "+
			"MarkVantageAvailable both filter on it, and span_open_timeline_idx can only be "+
			"scanned whole to apply it, got: %v", idx)
	}
	for _, name := range names {
		if idx[name].where == openSpanPredicate {
			return
		}
	}
	// An equality and not a Contains: `closed_at IS NULL AND is_gap`, the predicate #2179
	// proposed, and `... AND facet = ...` each drop rows MarkVantageUnavailable closes.
	t.Errorf("a span index leads on vantage_id but none is partial on exactly %q, so the "+
		"availability writers cannot seek their open spans, got: %v", openSpanPredicate, idx)
}

func TestSpanKeepsItsOpenTimelineUniqueIndex(t *testing.T) {
	// The new vantage_id index carries the same partial predicate, so it reads as a
	// replacement for a reader who does not know this one is UNIQUE. "The open span is the
	// current state" is that uniqueness (19000, ADR-0105 §3, ADR-0208).
	idx := tableIndexes(t, "span")

	timeline := []string{"subject_key", "facet", "discriminator", "vantage_id", "source"}
	for _, ix := range idx {
		if ix.unique && ix.where == openSpanPredicate && ix.keys(timeline...) {
			return
		}
	}
	t.Errorf("no span index is UNIQUE on (%s) WHERE %s; at most one open span per timeline "+
		"is a structural guarantee and not a query convention, got: %v",
		strings.Join(timeline, ", "), openSpanPredicate, idx)
}
