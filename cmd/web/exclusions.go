package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/seed"
)

// exclusionView is a declared exclusion shaped for rendering: the value
// collapsed to one display string, with the kind kept so the three kinds stay
// visually distinct and the un-exclude control can carry the row's id.
type exclusionView struct {
	ID    int64
	Kind  string // "name", "subtree" or "address"
	Value string // the excluded name or address scope, rendered
	By    string
	At    string
}

// declareExclusion draws the boundary inwards: an exact name, a name subtree, or
// an address scope the operator declares is not theirs (v1 spec §3.2, §6.4). It
// is reached only through requireAdmin, so a viewer can list exclusions but never
// declare one.
//
// A narrowing act should, per §6.4, show a narrowing receipt — the count of what
// it would withdraw — before the operator commits, but only where a withdrawal
// message would actually fire. That preview depends on the Message model (#205),
// which does not exist yet in this sequence, so it is deferred (#166, #167): the
// exclusion mechanism and its management UI ship here, and the receipt is wired in
// once a withdrawal message can be honestly computed.
func (s *server) declareExclusion(w http.ResponseWriter, r *http.Request, acct db.Account) {
	kind := r.FormValue("kind")
	value := strings.TrimSpace(r.FormValue("value"))
	fail := func(msg string) {
		s.renderSeeds(w, r, acct, seedsForms{exclError: msg, exclKind: kind, exclValue: value})
	}

	switch kind {
	case "name", "subtree":
		name, err := seed.NormalizeExclusionName(value)
		if err != nil {
			fail(err.Error())
			return
		}
		if _, err := s.store.CreateNameExclusion(r.Context(), db.CreateNameExclusionParams{
			Kind: kind, Name: pgtype.Text{String: name, Valid: true}, CreatedBy: acct.ID,
		}); err != nil {
			fail(exclusionCreateError(err, "name"))
			return
		}
	case "address":
		p, err := seed.NormalizeExclusionCIDR(value)
		if err != nil {
			fail(err.Error())
			return
		}
		if _, err := s.store.CreateAddressExclusion(r.Context(), db.CreateAddressExclusionParams{
			AddressCidr: &p, CreatedBy: acct.ID,
		}); err != nil {
			fail(exclusionCreateError(err, "address scope"))
			return
		}
	default:
		fail("Choose an exclusion type.")
		return
	}
	http.Redirect(w, r, "/scope", http.StatusSeeOther)
}

// previewExclusion computes the narrowing receipt for a candidate exclusion and
// re-renders the Seeds screen with it, before the operator commits (#205 AC8,
// ADR-0074). This is ticket 4's deferred receipt, now honestly computable: the
// Message model can count what a withdrawal message would carry. The preview
// fires only where the message would — an address exclusion over ground nothing
// else cites shows the count and names the loss; a name or subtree whose names
// still resolve (the survives-via-`Gap` case) withdraws nothing and shows no
// receipt, because a preview for a message that will not fire is a promise the
// widening side never has to make good on.
func (s *server) previewExclusion(w http.ResponseWriter, r *http.Request, acct db.Account) {
	kind := r.FormValue("kind")
	value := strings.TrimSpace(r.FormValue("value"))
	fail := func(msg string) {
		s.renderSeeds(w, r, acct, seedsForms{exclError: msg, exclKind: kind, exclValue: value})
	}

	var receipt message.NarrowingReceipt
	switch kind {
	case "address":
		p, err := seed.NormalizeExclusionCIDR(value)
		if err != nil {
			fail(err.Error())
			return
		}
		// The message fires at the Seed whose scope moved — the address scope that
		// contains the excluded ground. Fall back to the excluded value itself
		// where no declared scope covers it (nothing is enumerated there, so the
		// count is zero and the receipt does not fire anyway).
		scope := p.String()
		if covering, err := s.store.FindCoveringAddressSeed(r.Context(), p.Addr()); err == nil && covering.AddressCidr != nil {
			scope = covering.AddressCidr.String()
		} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			s.serverError(w, "find covering seed", err)
			return
		}
		row, err := s.store.PreviewExclusionWithdrawal(r.Context(), db.PreviewExclusionWithdrawalParams{
			Cidr: p, Kind: "address",
		})
		if err != nil {
			s.serverError(w, "preview exclusion withdrawal", err)
			return
		}
		receipt = message.PreviewNarrowing(scope, p.String(), int(row.SubjectsWithdrawn), int(row.TimelinesRemoved))
	case "name", "subtree":
		if _, err := seed.NormalizeExclusionName(value); err != nil {
			fail(err.Error())
			return
		}
		// A name or subtree whose names still resolve survives and its Gap carries
		// it; the honest count is zero and no receipt fires (ADR-0074).
		receipt = message.PreviewNarrowing(value, value, 0, 0)
	default:
		fail("Choose an exclusion type.")
		return
	}
	s.renderSeeds(w, r, acct, seedsForms{exclKind: kind, exclValue: value, exclPreview: &receipt})
}

// unexclude withdraws an exclusion. It is admin-only and idempotent: deleting a
// row that is already gone is not an error, since the operator's intent — that
// the exclusion no longer stand — is satisfied either way.
func (s *server) unexclude(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.renderSeeds(w, r, acct, seedsForms{exclError: "That exclusion could not be found."})
		return
	}
	if err := s.store.DeleteExclusion(r.Context(), id); err != nil {
		s.serverError(w, "delete exclusion", err)
		return
	}
	http.Redirect(w, r, "/scope", http.StatusSeeOther)
}

func toExclusionViews(rows []db.ListExclusionsRow) []exclusionView {
	out := make([]exclusionView, 0, len(rows))
	for _, row := range rows {
		v := exclusionView{ID: row.ID, Kind: row.Kind, By: row.CreatedByUsername}
		if row.Kind == "address" && row.AddressCidr != nil {
			v.Value = row.AddressCidr.String()
		} else {
			v.Value = row.Name.String
		}
		if row.CreatedAt.Valid {
			v.At = row.CreatedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
		}
		out = append(out, v)
	}
	return out
}

func exclusionCreateError(err error, noun string) string {
	if isUniqueViolation(err) {
		return "That " + noun + " is already excluded."
	}
	return "Could not declare the exclusion."
}
