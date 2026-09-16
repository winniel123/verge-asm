package db

import (
	"strings"
	"testing"
)

// TestEveryCiterReadTakesTheGapFallback holds the three citer reads to one answer about a
// Gap. internal/dbtest runs the statements, but that tier is advisory and skips wherever no
// DSN is set, so the shape is also matched here, inside a required check (ADR-0006, #2164).
func TestEveryCiterReadTakesTheGapFallback(t *testing.T) {
	for _, read := range []struct {
		name, query string
	}{
		{"listCitedAddressSpansForNames", listCitedAddressSpansForNames},
		{"listResolutionCitersForAddresses", listResolutionCitersForAddresses},
		{"listResolutionCitersForAddressesAt", listResolutionCitersForAddressesAt},
	} {
		lowered := strings.ToLower(read.query)
		for _, want := range []struct{ clause, why string }{
			{"is_gap = true and exists", "a Gap is the absence of a measurement, so it withdraws no Address"},
			{"q.closed_at is not null", "the fallback is the value the Gap replaced, never a live one"},
			{"q.is_gap = false", "a Gap behind a Gap held no value to fall back to"},
			{"q.vantage_id is not distinct from r.vantage_id", "the fallback stays on the gapped timeline"},
		} {
			if !strings.Contains(lowered, want.clause) {
				t.Errorf("%s must carry %q — %s, got:\n%s", read.name, want.clause, want.why, read.query)
			}
		}
	}
}
