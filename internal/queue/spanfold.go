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
//
// It also appends every timeline it opened or moved to `changes` (P0.7): the estate
// / drift feed the message producer consumes downstream, in the same transaction, to
// fold a transition into a message. A nil collector folds spans without recording the
// feed — the measurement-only path that produces no messages.
func foldObservationsIntoSpans(ctx context.Context, qtx *db.Queries, batchID int64, vantageID pgtype.Int8, observedAt time.Time, obs []wire.Observation, in membershipInputs, changes *[]spanChange) error {
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
	// An opening with NO OPEN PREDECESSOR is marked where the operator's aperture is
	// why the fold looked at the subject — a Seed-declared Name/Address/Service the
	// exclusions do not cut back out (openedByAperture) — and unmarked where the world
	// brought a subject the aperture had not declared. The marker is the estate signal
	// the drift feed reads to tell the two apart (#637, ADR-0014). A value MOVE on an
	// existing timeline (open != nil) is `changed` and never carries it.
	//
	// `open == nil` is a first span OR a re-entry behind a closed one, and both are
	// marked. A first span reads the marker as `revealed`. A re-entry behind a
	// `descoped` closure reads it as the operator widening a Declared scope back over
	// the subject, which is `revealed` as well (drift.ReEntryKind, ADR-0041, #1039).
	openedAperture := open == nil && openedByAperture(key.SubjectKind, key.SubjectKey, in)
	if open != nil && !closeAt.IsZero() {
		// An ordinary value move or a version change: close with no reason, citing the
		// batch whose fold closed it (ADR-0111) so the estate-wide feed can pair this
		// close with the open below into one `changed` transition. A withdrawal's reason
		// is applied by the membership path, not here.
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
	// Record the transition for the message producer (P0.7): an opening (open == nil)
	// is a membership / census candidate, and a reachability change is a flagship
	// candidate. A nil collector is the measurement-only path that produces no message.
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

// facetVector is the Derivation vector for a facet's Span — the leaves that
// decide the value, composed as their union (ADR-0086), so a bump of any one
// moves the value and Breaks the timeline. `resolution` and `dns-record` are
// decided jointly by `resolution-walk` and `wildcard-discrimination`;
// `reachability` is decided jointly by `connect-outcome` and, since ADR-0104, the
// `blanket-discrimination` leaf that gaps a blanket responder's reaches — a bump of
// either Breaks the reach half of the estate once (ADR-0104 Consequences). The
// vector is keyed on the facet rather than hardcoded so a wave-4 facet adds its
// leaves here without forking the fold.
func facetVector(facet string) drift.Vector {
	switch facet {
	case connectoutcome.FacetReachability:
		return drift.NewVector(
			drift.Component{Leaf: connectoutcome.Kind, Version: connectoutcome.Version},
			drift.Component{Leaf: blanketdiscrim.Kind, Version: blanketdiscrim.Version},
		)
	case connectoutcome.FacetCertificate:
		// `certificate` is decided by the tls-handshake leaf alone — the reached
		// connect gates whether the handshake runs, but the presented chain (or
		// either TLS negative) is a function of the handshake, not of the connect
		// verdict — so a tls-handshake bump Breaks `certificate` timelines without
		// touching `reachability` (ADR-0086).
		return drift.NewVector(
			drift.Component{Leaf: "tls-handshake", Version: connectoutcome.CertVersion},
		)
	case httpexchange.FacetHTTPIdentity:
		return drift.NewVector(
			drift.Component{Leaf: httpexchange.Kind, Version: httpexchange.Version},
		)
	case tlsacceptance.Facet:
		// `tls-acceptance` is decided by the tls-acceptance leaf alone — its own
		// enumeration exchange, distinct from the tls-handshake leaf that feeds
		// `certificate` — so a tls-acceptance bump Breaks `tls-acceptance` timelines
		// without touching `certificate` ones (ADR-0028, ADR-0086).
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

// isGapValue reports whether an observation records a Gap — a period over which
// the system could not say. Two facets carry a gap outcome tag: `resolution`, where
// an undiscriminated DNS answer is a Gap (ADR-0066), and `reachability`, where an
// undiscriminated reach on a blanket responder is a Gap (ADR-0104,
// blanketdiscrim). A `dns-record` line is the RRset it measured and is never a gap
// here; a `reachability` value (`reached` / `not-reached`) is not a gap either.
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
