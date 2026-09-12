package main

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/seed"
)

type exclusionsStore interface {
	CreateAddressExclusion(ctx context.Context, arg db.CreateAddressExclusionParams) (db.Exclusion, error)
	CreateNameExclusion(ctx context.Context, arg db.CreateNameExclusionParams) (db.Exclusion, error)
	DeleteExclusion(ctx context.Context, id int64) (db.DeleteExclusionRow, error)
	FindCoveringAddressSeed(ctx context.Context, address netip.Addr) (db.FindCoveringAddressSeedRow, error)
	PreviewExclusionWithdrawal(ctx context.Context, arg db.PreviewExclusionWithdrawalParams) (db.PreviewExclusionWithdrawalRow, error)
}

type exclusionView struct {
	ID    int64
	Kind  string
	Value string
	At    string

	UndoProposalIDs []int64
}

func (s *server) declareExclusion(w http.ResponseWriter, r *http.Request, acct db.Account) {
	kind := r.FormValue("kind")
	value := strings.TrimSpace(r.FormValue("value"))
	fail := func(msg string) {
		s.flashScopeBack(w, r, seedsForms{exclError: msg, exclKind: kind, exclValue: value})
	}

	var scope string
	switch kind {
	case "name", "subtree":
		name, err := seed.NormalizeExclusionName(value)
		if err != nil {
			fail(err.Error())
			return
		}
		if _, err := s.exclusionsStore.CreateNameExclusion(r.Context(), db.CreateNameExclusionParams{
			Kind: kind, Name: pgtype.Text{String: name, Valid: true}, CreatedBy: pgtype.Int8{Int64: acct.ID, Valid: true},
		}); err != nil {
			fail(exclusionCreateError(err, "name"))
			return
		}
		scope = name
	case "address":
		p, err := seed.NormalizeExclusionCIDR(value)
		if err != nil {
			fail(err.Error())
			return
		}
		if _, err := s.exclusionsStore.CreateAddressExclusion(r.Context(), db.CreateAddressExclusionParams{
			AddressCidr: &p, CreatedBy: pgtype.Int8{Int64: acct.ID, Valid: true},
		}); err != nil {
			fail(exclusionCreateError(err, "address scope"))
			return
		}
		scope = p.String()
	default:
		fail("Choose an exclusion type.")
		return
	}
	// The normalized form and not the operator's, so the Act and the row name one scope.
	s.recorder().Record(r.Context(), actingAccount(acct), act.ExclusionDeclared{
		ExclusionRef: act.ExclusionRef{Kind: kind, Scope: scope},
	})
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
		if covering, err := s.exclusionsStore.FindCoveringAddressSeed(r.Context(), p.Addr()); err == nil && covering.AddressCidr != nil {
			scope = covering.AddressCidr.String()
		} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			s.serverError(w, "find covering seed", err)
			return
		}
		row, err := s.exclusionsStore.PreviewExclusionWithdrawal(r.Context(), db.PreviewExclusionWithdrawalParams{
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
		// A narrowing that withdraws nothing is silent, so a survivor gets no receipt (ADR-0074).
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
	// The scope rides the delete's RETURNING, so no read can blank the Act (spec §4.2).
	row, err := s.exclusionsStore.DeleteExclusion(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		// An unknown id lifts nothing, and a refused act directed nothing (spec §7.6).
		s.backToScope(w, r)
		return
	}
	if err != nil {
		s.serverError(w, "delete exclusion", err)
		return
	}
	s.recorder().Record(r.Context(), actingAccount(acct), act.ExclusionLifted{
		ExclusionRef: act.ExclusionRef{Kind: row.Kind, Scope: liftedExclusionScope(row)},
	})
	s.backToScope(w, r)
}

// The pair toExclusionViews renders, so the Act and the screen name one scope.

func liftedExclusionScope(row db.DeleteExclusionRow) string {
	if row.AddressCidr != nil {
		return row.AddressCidr.String()
	}
	return row.Name.String
}

func hasAddressExclusion(rows []db.Exclusion) bool {
	for _, row := range rows {
		if row.Kind == "address" && row.AddressCidr != nil {
			return true
		}
	}
	return false
}

func toExclusionViews(rows []db.Exclusion, declined map[string][]int64) []exclusionView {
	out := make([]exclusionView, 0, len(rows))
	for _, row := range rows {
		v := exclusionView{ID: row.ID, Kind: row.Kind}
		if row.Kind == "address" && row.AddressCidr != nil {
			v.Value = row.AddressCidr.String()
			v.UndoProposalIDs = declined[v.Value]
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
