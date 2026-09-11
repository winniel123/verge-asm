package migrations

import (
	"regexp"
	"strings"
	"testing"
)

func TestMessageCarriesTheBasisAHeldCensusIsComputedFrom(t *testing.T) {
	// The release poll reads what the fold saw, never live resolution (ADR-1806 §4).
	up := strings.ToLower(upMigrations(t))

	col := regexp.MustCompile(`census_basis\s+jsonb[^,;]*`).FindString(up)
	if col == "" {
		t.Fatal("no message.census_basis column found — a held census owes the basis it is computed from (ADR-1806 §4)")
	}
	if strings.Contains(col, "not null") {
		t.Errorf("census_basis must be nullable — a firing that owes no census records no basis, got: %s", strings.TrimSpace(col))
	}
	if !strings.Contains(up, "census_pending_after_batch is null or census_basis is not null") {
		t.Error("a held row must owe its basis — no CHECK ties census_basis to census_pending_after_batch (ADR-1806 §4)")
	}
}
