package queue

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
	"github.com/winniel123/verge-asm/internal/wire"
)

// foldObservationsIntoSpans folds one completed Batch's observations into the
// Span corpus, inside the Batch's own transaction (ADR-0007): ingest is an
// incremental fold, never a diff of two runs. For each observation's timeline it
// reads the open span and, where the canonical value or the Derivation vector
// moved, closes that span and opens a new one. Re-running the dns Scan with a
// changed answer therefore produces a new Span and closes the old — an ordinary
// value move, so the closure records no reason (the next span is the fact).
//
// A withdrawal is NOT decided here: whether a subject left the estate is a
// subject-level cross-class composition (internal/estate), and closing its
// timelines with a `measured-absent`/`uncited`/`descoped` ground is that path's
// job. This fold only tracks per-timeline value movement.
func foldObservationsIntoSpans(ctx context.Context, qtx *db.Queries, vantageID pgtype.Int8, observedAt time.Time, obs []wire.Observation) error {
	for _, o := range obs {
		if o.Facet == "" {
			continue
		}
		if err := foldOne(ctx, qtx, vantageID, observedAt, o); err != nil {
			return err
		}
	}
	return nil
}

func foldOne(ctx context.Context, qtx *db.Queries, vantageID pgtype.Int8, observedAt time.Time, o wire.Observation) error {
	key := drift.TimelineKey{
		SubjectKind:   subjectKindFor(o.Facet),
		SubjectKey:    o.Subject,
		Facet:         o.Facet,
		Discriminator: o.Discriminator,
		Source:        "resolver",
	}
	value := canonicalJSON(o.Data)
	reading := drift.Reading{
		Value:      string(value),
		IsGap:      isGapValue(o.Facet, o.Data),
		Vector:     facetVector(o.Facet),
		ObservedAt: observedAt,
	}

	row, err := qtx.GetOpenSpan(ctx, db.GetOpenSpanParams{
		SubjectKey:    o.Subject,
		Facet:         o.Facet,
		Discriminator: o.Discriminator,
		VantageID:     vantageID,
		Source:        "resolver",
	})
	var open *drift.Span
	var openID int64
	switch {
	case err == nil:
		openID = row.ID
		open = &drift.Span{
			Value:  string(canonicalJSON(row.Value)),
			IsGap:  row.IsGap,
			Vector: vectorFromJSON(row.Derivation),
		}
	case errors.Is(err, pgx.ErrNoRows):
		open = nil
	default:
		return err
	}

	closeAt, opened, changed := drift.FoldStep(open, key, reading)
	if !changed {
		return nil
	}
	if open != nil && !closeAt.IsZero() {
		// An ordinary value move or a version change: close with no reason. A
		// withdrawal's reason is applied by the membership path, not here.
		if err := qtx.CloseSpan(ctx, db.CloseSpanParams{ClosedAt: tstz(closeAt), ID: openID}); err != nil {
			return err
		}
	}
	_, err = qtx.OpenSpan(ctx, db.OpenSpanParams{
		SubjectKind:   key.SubjectKind,
		SubjectKey:    key.SubjectKey,
		Facet:         key.Facet,
		Discriminator: key.Discriminator,
		VantageID:     vantageID,
		Source:        "resolver",
		Value:         value,
		IsGap:         opened.IsGap,
		Derivation:    mustVectorJSON(opened.Vector),
		OpenedAt:      tstz(observedAt),
	})
	return err
}

// facetVector is the Derivation vector for a facet's Span. Both `resolution` and
// `dns-record` are decided jointly by `resolution-walk` and
// `wildcard-discrimination`, so a reader of either value composes the union of
// the two leaves (ADR-0086); a bump of either leaf moves the value and Breaks the
// timeline.
func facetVector(facet string) drift.Vector {
	return drift.NewVector(
		drift.Component{Leaf: "resolution-walk", Version: resolutionwalk.Version},
		drift.Component{Leaf: "wildcard-discrimination", Version: wildcarddiscrim.Version},
	)
}

// isGapValue reports whether a resolution observation records a Gap — a period
// over which the system could not say. Only `resolution` carries the outcome tag;
// a `dns-record` line is the RRset it measured and is never a gap here.
func isGapValue(facet string, data json.RawMessage) bool {
	if facet != resolutionwalk.FacetResolution {
		return false
	}
	var v struct {
		Outcome string `json:"outcome"`
	}
	_ = json.Unmarshal(data, &v)
	return v.Outcome == string(resolutionwalk.OutcomeGap)
}

// canonicalJSON renders a JSON value in a stable form (object keys sorted by
// Go's encoder) so a JSONB round-trip that reorders keys does not read as a value
// move. The leaf already emits deterministic values; this guards the comparison
// against Postgres's own normalisation.
func canonicalJSON(b []byte) []byte {
	if len(b) == 0 {
		return []byte("null")
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return b
	}
	out, err := json.Marshal(v)
	if err != nil {
		return b
	}
	return out
}

func vectorFromJSON(b []byte) drift.Vector {
	var comps []drift.Component
	_ = json.Unmarshal(b, &comps)
	return drift.NewVector(comps...)
}

func mustVectorJSON(v drift.Vector) []byte {
	out, err := json.Marshal(v)
	if err != nil {
		panic("queue: marshal derivation vector: " + err.Error())
	}
	return out
}
