package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/proposer"
	"github.com/winniel123/verge-asm/internal/seed"
)

type proposalRow struct {
	ID     int64
	Value  string
	Kind   string
	Source string
}

func flattenProposals(lookups []proposalLookupView) []proposalRow {
	var out []proposalRow
	for _, l := range lookups {
		for _, p := range l.Proposals {
			out = append(out, proposalRow{ID: p.ID, Value: p.Scope, Kind: "range", Source: p.Source})
		}
	}
	return out
}

type proposerRunner interface {
	Propose(ctx context.Context, orgName string, enabled map[string]bool) ([]proposer.Candidate, error)
}

type proposalView struct {
	ID          int64
	Scope       string
	Source      string
	RecordLabel string
	OrgName     string
	AddrCount   string
}

type proposalLookupView struct {
	LookupID  int64
	Query     string
	By        string
	At        string
	Count     int
	Proposals []proposalView
}

func recordLabel(kind string) string {
	switch kind {
	case proposer.RecordCompelledReassignment:
		return "compelled reassignment"
	case proposer.RecordRIRDelegation:
		return "RIR delegation"
	default:
		return kind
	}
}

func humanCount(p netip.Prefix) string {
	n := seed.AddressCount(p).String()
	var b strings.Builder
	for i, c := range n {
		if i > 0 && (len(n)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

func overCapProposalNotice(p netip.Prefix, cap int) string {
	return fmt.Sprintf(
		"That proposed scope spans %s addresses — over your cap of %s. Raise your cap in Settings · Scans to confirm it whole, or decline it.",
		humanCount(p), commaInt(cap),
	)
}

func toProposalLookups(rows []db.ListPendingProposalsRow) []proposalLookupView {
	var out []proposalLookupView
	byLookup := map[int64]int{}
	for _, row := range rows {
		idx, ok := byLookup[row.LookupID]
		if !ok {
			at := ""
			if row.LookupAt.Valid {
				at = row.LookupAt.Time.UTC().Format("2006-01-02 15:04 UTC")
			}
			out = append(out, proposalLookupView{
				LookupID: row.LookupID, Query: row.LookupQuery, By: row.LookupBy, At: at,
			})
			idx = len(out) - 1
			byLookup[row.LookupID] = idx
		}
		out[idx].Proposals = append(out[idx].Proposals, proposalView{
			ID: row.ID, Scope: row.AddressCidr.String(), Source: row.SourceSlug,
			RecordLabel: recordLabel(row.RecordKind), OrgName: row.OrgName,
			AddrCount: humanCount(row.AddressCidr),
		})
		out[idx].Count = len(out[idx].Proposals)
	}
	return out
}

func (s *server) proposalLookups(ctx context.Context) ([]proposalLookupView, error) {
	rows, err := s.store.ListPendingProposals(ctx)
	if err != nil {
		return nil, err
	}
	return toProposalLookups(rows), nil
}

func (s *server) runLookup(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// scope.tmpl posts org and the older /proposals route posts query, so both reach here (#574).
	query := strings.TrimSpace(r.FormValue("org"))
	if query == "" {
		query = strings.TrimSpace(r.FormValue("query"))
	}
	if query == "" {
		s.flashScopeBack(w, r, seedsForms{proposalError: "Enter an organisation name to search."})
		return
	}

	enabled, err := s.enabledProposers(r)
	if err != nil {
		s.serverError(w, "list source states", err)
		return
	}

	cands, perr := s.proposals().Propose(r.Context(), query, enabled)
	if perr != nil {
		log.Printf("web: proposer lookup %q: %v", logSafe(query), perr) // #nosec G706 (sanitized via logSafe)
	}
	if len(cands) == 0 {
		msg := "No candidate scopes matched that name."
		if perr != nil {
			msg = "The lookup could not be completed — a registry path errored and no candidates were found. See the server log for details."
		}
		s.flashScopeBack(w, r, seedsForms{proposalQuery: query, proposalNotice: msg})
		return
	}

	lookup, err := s.store.CreateProposerLookup(r.Context(), db.CreateProposerLookupParams{
		Query: query, CreatedBy: acct.ID,
	})
	if err != nil {
		s.serverError(w, "create lookup", err)
		return
	}
	for _, c := range cands {
		if _, err := s.store.CreateProposal(r.Context(), db.CreateProposalParams{
			LookupID: lookup.ID, SourceSlug: c.SourceSlug, RecordKind: c.RecordKind,
			AddressCidr: c.Scope, OrgName: c.OrgName,
		}); err != nil {
			s.serverError(w, "create proposal", err)
			return
		}
	}
	if perr != nil {
		// The candidates are already filed, so an inline render would re-file them on refresh.
		s.flashScopeBack(w, r, seedsForms{proposalNotice: partialProposalNotice})
		return
	}
	s.backToScope(w, r)
}

const partialProposalNotice = "Showing partial results — one or more registry paths errored, so this list may be incomplete. See the server log for details."

func (s *server) confirmProposal(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad proposal id", http.StatusBadRequest)
		return
	}

	p, err := s.store.GetPendingProposal(r.Context(), id)
	if err != nil {
		s.backToScope(w, r)
		return
	}

	// A confirmed Proposal is a Seed, so the declaration path's own cap gates it too (#892).
	if addrCap := s.addressCap(r.Context()); !seed.WithinCap(p.AddressCidr, addrCap) {
		s.flashScopeBack(w, r, seedsForms{proposalNotice: overCapProposalNotice(p.AddressCidr, addrCap)})
		return
	}

	// A cidr column rejects host bits, so masking is what gives an org range dispatch parity (#755).
	cidr := p.AddressCidr.Masked()
	sd, err := s.store.CreateAddressSeed(r.Context(), db.CreateAddressSeedParams{
		AddressCidr: &cidr, CreatedBy: acct.ID,
	})
	if err != nil {
		if isUniqueViolation(err) {
			s.flashScopeBack(w, r, seedsForms{proposalError: "That scope is already declared as a seed."})
			return
		}
		s.serverError(w, "create seed from proposal", err)
		return
	}
	if _, err := s.store.ConfirmProposal(r.Context(), db.ConfirmProposalParams{
		ID: id, ConfirmedSeedID: pgtype.Int8{Int64: sd.ID, Valid: true},
	}); err != nil {
		s.serverError(w, "confirm proposal", err)
		return
	}
	s.backToScope(w, r)
}

func (s *server) declineLookup(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode {
		s.backToScope(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	// scope.tmpl posts the checked ids, so this declines a selection and not a lookup (#574).
	raw := r.Form["ids"]
	if len(raw) == 0 {
		s.backToScope(w, r)
		return
	}
	for _, idStr := range raw {
		id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
		if err != nil {
			continue
		}
		p, gerr := s.store.GetPendingProposal(r.Context(), id)
		if gerr != nil {
			continue
		}
		if _, err := s.store.DeclineProposal(r.Context(), id); err != nil {
			s.serverError(w, "decline proposal", err)
			return
		}
		cidr := p.AddressCidr
		// A decline records an exclusion, so the same range is not proposed again (ADR-0012).
		if _, err := s.store.CreateAddressExclusion(r.Context(), db.CreateAddressExclusionParams{
			AddressCidr: &cidr, CreatedBy: acct.ID,
		}); err != nil && !isUniqueViolation(err) {
			s.serverError(w, "record declined proposal as exclusion", err)
			return
		}
	}
	s.backToScope(w, r)
}

func (s *server) enabledProposers(r *http.Request) (map[string]bool, error) {
	views, err := s.sourceViews(r)
	if err != nil {
		return nil, err
	}
	enabled := make(map[string]bool, len(views))
	for _, v := range views {
		if v.KindLabel == "proposer" && v.Consent == consentUnencumbered {
			enabled[v.Slug] = v.Enabled
		}
	}
	return enabled, nil
}

func (s *server) proposals() proposerRunner {
	if s.proposer == nil {
		return proposer.NewRegistry()
	}
	return s.proposer
}
