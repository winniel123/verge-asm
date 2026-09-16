package db

import (
	"strings"
	"testing"
)

func TestEveryCiterReadTakesTheGapFallback(t *testing.T) {
	// The weaker of the two guards, and never the proof: a text match cannot see
	// all three reads move together, and internal/dbtest runs them against a
	// database inside the same required job now (ADR-0006, #2164, #2255).
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
