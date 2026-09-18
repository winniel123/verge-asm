package migrations

import (
	"regexp"
	"strings"
	"testing"
)

func TestMessageCarriesANullableCensusPendingColumn(t *testing.T) {
	// NULL is the released state every row already written holds (ADR-1806 §2).
	col := tableColumn(t, "message", "census_pending_after_batch")

	if !strings.Contains(col, "bigint") {
		t.Errorf("census_pending_after_batch must be a BIGINT, got: %s", col)
	}
	if strings.Contains(col, "not null") {
		t.Errorf("census_pending_after_batch must be nullable — NULL is the released state, got: %s", col)
	}
	// The release poll orders the first hot dispatch against the root's own batch (ADR-1806 §3).
	if !regexp.MustCompile(`references\s+batch\s*\(\s*id\s*\)`).MatchString(col) {
		t.Errorf("census_pending_after_batch must reference batch (id), got: %s", col)
	}
}
