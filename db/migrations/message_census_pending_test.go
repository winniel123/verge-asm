package migrations

import (
	"regexp"
	"strings"
	"testing"
)

func TestMessageCarriesANullableCensusPendingColumn(t *testing.T) {
	// NULL is the released state every row already written holds (ADR-1806 §2).
	up := strings.ToLower(upMigrations(t))

	col := regexp.MustCompile(`census_pending_after_batch\s+bigint[^,;]*`).FindString(up)
	if col == "" {
		t.Fatal("no message.census_pending_after_batch column found — a message whose census is pending must be held (ADR-1806 §2)")
	}
	if strings.Contains(col, "not null") {
		t.Errorf("census_pending_after_batch must be nullable — NULL is the released state, got: %s", strings.TrimSpace(col))
	}
	// The release poll orders the first hot dispatch against the root's own batch (ADR-1806 §3).
	if !regexp.MustCompile(`references\s+batch\s*\(\s*id\s*\)`).MatchString(col) {
		t.Errorf("census_pending_after_batch must reference batch (id), got: %s", strings.TrimSpace(col))
	}
}
