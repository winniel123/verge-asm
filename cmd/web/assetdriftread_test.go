package main

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

const (
	driftDidNotResolve = "The span read did not resolve on this load."
	driftEmptyState    = "This asset holds no folded span yet, so there is no transition to trace."
)

func TestAssetDriftReadFailureRendersDidNotResolve(t *testing.T) {
	f, base := assetRegionFixture(t, "api.example.com")
	f.subjectSpansErr = errors.New("list spans for subject failed")

	page := assetRegionPage(t, f, base, "api.example.com")

	wantIn(t, page, driftDidNotResolve, "failed span read")
	wantNotIn(t, page, driftEmptyState, "failed span read")
	wantIn(t, page, "Drift trail", "failed span read")
	// A region read never takes another region away with it (ADR-0168 §1).
	wantIn(t, page, "198.51.100.1", "failed span read")
	wantIn(t, page, "How it got here", "failed span read")
}

func TestAssetDriftKeepsItsEmptyStateWhenNoSpanExists(t *testing.T) {
	for _, empty := range []struct {
		name string
		set  func(*fakeStore)
	}{
		{"zero rows", func(f *fakeStore) { f.subjectSpansEmpty = true }},
		// No row is the honest empty answer, so it must not raise the failed flag (ADR-0168 §4).
		{"no rows error", func(f *fakeStore) { f.subjectSpansErr = pgx.ErrNoRows }},
	} {
		t.Run(empty.name, func(t *testing.T) {
			f, base := assetRegionFixture(t, "api.example.com")
			empty.set(f)

			page := assetRegionPage(t, f, base, "api.example.com")

			wantIn(t, page, driftEmptyState, "an asset that holds no span")
			wantNotIn(t, page, driftDidNotResolve, "an asset that holds no span")
		})
	}
}

func TestAssetDriftRendersItsTrailOnAResolvedSpanRead(t *testing.T) {
	f, base := assetRegionFixture(t, "api.example.com")

	page := assetRegionPage(t, f, base, "api.example.com")

	wantIn(t, page, "Drift trail", "resolved span read")
	wantNotIn(t, page, driftDidNotResolve, "resolved span read")
	wantNotIn(t, page, driftEmptyState, "resolved span read")
}
