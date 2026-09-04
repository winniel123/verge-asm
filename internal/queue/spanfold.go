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
	"github.com/winniel123/verge-asm/internal/measure/blanketdiscrim"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
	"github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
	"github.com/winniel123/verge-asm/internal/wire"
)

func foldObservationsIntoSpans(ctx context.Context, qtx *db.Queries, batchID int64, vantageID pgtype.Int8, observedAt time.Time, obs []wire.Observation, in membershipInputs, changes *[]spanChange) error {
	// Ingest is an incremental fold over completed batches, never a diff of two runs (ADR-0007).
	for _, o := range obs {
		if o.Facet == "" {
			continue
		}
		if err := foldOne(ctx, qtx, batchID, vantageID, observedAt, o, in, changes); err != nil {
			return err
		}
	}
	return nil
}

func foldOne(ctx context.Context, qtx *db.Queries, batchID int64, vantageID pgtype.Int8, observedAt time.Time, o wire.Observation, in membershipInputs, changes *[]spanChange) error {
	source := sourceFor(o.Facet)
	key := drift.TimelineKey{
		SubjectKind:   subjectKindFor(o.Facet),
		SubjectKey:    o.Subject,
		Facet:         o.Facet,
		Discriminator: o.Discriminator,
		Source:        source,
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
		Source:        source,
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
	// A re-entry behind a closure is marked like a first span, so drift reads it revealed (ADR-0041).
	openedAperture := open == nil && openedByAperture(key.SubjectKind, key.SubjectKey, in)
	if open != nil && !closeAt.IsZero() {
		// Departure is a cross-class composition, so the membership path closes those spans, not this.
		if err := qtx.CloseSpan(ctx, db.CloseSpanParams{ClosedAt: tstz(closeAt), ClosedBatchID: pgInt8(batchID), ID: openID}); err != nil {
			return err
		}
	}
	if _, err = qtx.OpenSpan(ctx, db.OpenSpanParams{
		SubjectKind:    key.SubjectKind,
		SubjectKey:     key.SubjectKey,
		Facet:          key.Facet,
		Discriminator:  key.Discriminator,
		VantageID:      vantageID,
		Source:         source,
		Value:          value,
		IsGap:          opened.IsGap,
		Derivation:     mustVectorJSON(opened.Vector),
		OpenedAt:       tstz(observedAt),
		OpenedBatchID:  pgInt8(batchID),
		OpenedAperture: openedAperture,
	}); err != nil {
		return err
	}
	if changes != nil {
		*changes = append(*changes, spanChange{
			SubjectKind:    key.SubjectKind,
			SubjectKey:     key.SubjectKey,
			Facet:          key.Facet,
			Opened:         open == nil,
			OpenedAperture: openedAperture,
			Value:          append([]byte(nil), value...),
		})
	}
	return nil
}

func facetVector(facet string) drift.Vector {
	// A vector composes the union of every leaf deciding the value, so a bump Breaks it (ADR-0086).
	switch facet {
	case connectoutcome.FacetReachability:
		return drift.NewVector(
			drift.Component{Leaf: connectoutcome.Kind, Version: connectoutcome.Version},
			drift.Component{Leaf: blanketdiscrim.Kind, Version: blanketdiscrim.Version},
		)
	case connectoutcome.FacetCertificate:
		// The connect gates whether the handshake runs but decides no chain, so it stays out of here.
		return drift.NewVector(
			drift.Component{Leaf: "tls-handshake", Version: connectoutcome.CertVersion},
		)
	case httpexchange.FacetHTTPIdentity:
		return drift.NewVector(
			drift.Component{Leaf: httpexchange.Kind, Version: httpexchange.Version},
		)
	case tlsacceptance.Facet:
		return drift.NewVector(
			drift.Component{Leaf: tlsacceptance.Kind, Version: tlsacceptance.Version},
		)
	default:
		return drift.NewVector(
			drift.Component{Leaf: "resolution-walk", Version: resolutionwalk.Version},
			drift.Component{Leaf: "wildcard-discrimination", Version: wildcarddiscrim.Version},
		)
	}
}

func isGapValue(facet string, data json.RawMessage) bool {
	var v struct {
		Outcome string `json:"outcome"`
	}
	_ = json.Unmarshal(data, &v)
	switch facet {
	case resolutionwalk.FacetResolution:
		return v.Outcome == string(resolutionwalk.OutcomeGap)
	case connectoutcome.FacetReachability:
		return v.Outcome == connectoutcome.GapOutcome
	default:
		return false
	}
}

func canonicalJSON(b []byte) []byte {
	// Postgres normalises JSONB key order, so an unstable rendering would read as a value move.
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
