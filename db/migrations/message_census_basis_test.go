package migrations

import (
	"strings"
	"testing"
)

func TestMessageCarriesTheBasisAHeldCensusIsComputedFrom(t *testing.T) {
	// The release poll reads what the fold saw, never live resolution (ADR-1806 §4).
	col := tableColumn(t, "message", "census_basis")

	if !strings.Contains(col, "jsonb") {
		t.Errorf("census_basis must be JSONB — a held census owes the basis it is computed from, got: %s", col)
	}
	if strings.Contains(col, "not null") {
		t.Errorf("census_basis must be nullable — a firing that owes no census records no basis, got: %s", col)
	}

	const tie = "census_pending_after_batch is null or census_basis is not null"
	if len(constraintsMatching(t, "message", tie)) == 0 {
		t.Errorf("a held row must owe its basis — no surviving CHECK ties census_basis to "+
			"census_pending_after_batch (ADR-1806 §4), got: %v", tableConstraints(t, "message"))
	}
}
