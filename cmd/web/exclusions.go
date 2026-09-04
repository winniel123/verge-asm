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

type exclusionView struct {
	ID    int64
	Kind  string
	Value string
	By    string
	At    string
}

func (s *server) declareExclusion(w http.ResponseWriter, r *http.Request, acct db.Account) {
	kind := r.FormValue("kind")
	value := strings.TrimSpace(r.FormValue("value"))
	fail := func(msg string) {
		s.flashScopeBack(w, r, seedsForms{exclError: msg, exclKind: kind, exclValue: value})
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
	s.backToScope(w, r)
}

func (s *server) previewExclusion(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode {
		s.render(w, r, "scope", s.scopeFixtureDataPreview(acct))
		return
	}
	kind := r.FormValue("kind")
	value := strings.TrimSpace(r.FormValue("value"))
	fail := func(msg string) {
		s.flashScopeBack(w, r, seedsForms{exclError: msg, exclKind: kind, exclValue: value})
	}

	var receipt message.NarrowingReceipt
	switch kind {
	case "address":
		p, err := seed.NormalizeExclusionCIDR(value)
		if err != nil {
			fail(err.Error())
			return
		}
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
		// A narrowing that withdraws no subject is silent, so a survivor gets no receipt (ADR-0074).
		receipt = message.PreviewNarrowing(value, value, 0, 0)
	default:
		fail("Choose an exclusion type.")
		return
	}
	s.flashScopeBack(w, r, seedsForms{exclKind: kind, exclValue: value, exclPreview: &receipt})
}

func (s *server) unexclude(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.flashScopeBack(w, r, seedsForms{exclError: "That exclusion could not be found."})
		return
	}
	if err := s.store.DeleteExclusion(r.Context(), id); err != nil {
		s.serverError(w, "delete exclusion", err)
		return
	}
	s.backToScope(w, r)
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
