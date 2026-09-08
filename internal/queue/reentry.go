package queue

import (
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/message"
)

func reEntryInputs(rows []db.ListSpansForSubjectRow) (prior *drift.Span, witnessBroke bool) {
	var latest *db.ListSpansForSubjectRow
	// The conjunction is computed at the fold and never stored (ADR-0097).
	for i := range rows {
		r := &rows[i]
		if r.Facet != resolutionwalk.FacetResolution || !r.ClosedAt.Valid {
			continue
		}
		if latest == nil || r.ClosedAt.Time.After(latest.ClosedAt.Time) ||
			(r.ClosedAt.Time.Equal(latest.ClosedAt.Time) && r.ID > latest.ID) {
			latest = r
		}
	}
	// A value-move closure carries no ground, so the subject never left and this is no re-entry.
	if latest != nil && drift.ClosureReason(latest.ClosureReason.String).Valid() {
		prior = &drift.Span{
			Value:    string(latest.Value),
			IsGap:    latest.IsGap,
			Vector:   vectorFromJSON(latest.Derivation),
			OpenedAt: latest.OpenedAt.Time,
			ClosedAt: latest.ClosedAt.Time,
			Reason:   drift.ClosureReason(latest.ClosureReason.String),
		}
	}
	for i := range rows {
		w := &rows[i]
		if w.Facet != resolutionwalk.FacetResolution || w.ClosedAt.Valid {
			continue
		}
		if pred := timelinePredecessor(rows, w); pred != nil &&
			!vectorFromJSON(pred.Derivation).Equal(vectorFromJSON(w.Derivation)) {
			witnessBroke = true
		}
	}
	return prior, witnessBroke
}

func timelinePredecessor(rows []db.ListSpansForSubjectRow, w *db.ListSpansForSubjectRow) *db.ListSpansForSubjectRow {
	var pred *db.ListSpansForSubjectRow
	for i := range rows {
		p := &rows[i]
		if p.ID == w.ID || !p.ClosedAt.Valid || !sameTimeline(p, w) {
			continue
		}
		if p.OpenedAt.Time.After(w.OpenedAt.Time) || (p.OpenedAt.Time.Equal(w.OpenedAt.Time) && p.ID > w.ID) {
			continue
		}
		if pred == nil || p.OpenedAt.Time.After(pred.OpenedAt.Time) ||
			(p.OpenedAt.Time.Equal(pred.OpenedAt.Time) && p.ID > pred.ID) {
			pred = p
		}
	}
	return pred
}

func sameTimeline(a, b *db.ListSpansForSubjectRow) bool {
	return a.Facet == b.Facet && a.Discriminator == b.Discriminator &&
		a.VantageID == b.VantageID && a.Source == b.Source
}

func membershipEntry(root spanChange) message.Entry {
	kind := drift.OpeningKind(root.OpenedAperture)
	if root.PriorClosure != nil {
		kind = drift.ReEntryKind(root.PriorClosure, root.WitnessBroke, root.OpenedAperture)
	}
	switch kind {
	case drift.KindReturned:
		return message.EntryReturned
	case drift.KindRevealed:
		return message.EntryRevealed
	default:
		return message.EntryAppeared
	}
}
