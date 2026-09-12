package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/proposer"
	"github.com/winniel123/verge-asm/internal/seed"
)

type proposalsStore interface {
	ConfirmProposal(ctx context.Context, arg db.ConfirmProposalParams) (int64, error)
	CreateAddressExclusion(ctx context.Context, arg db.CreateAddressExclusionParams) (db.Exclusion, error)
	CreateAddressSeed(ctx context.Context, arg db.CreateAddressSeedParams) (db.Seed, error)
	CreateProposal(ctx context.Context, arg db.CreateProposalParams) (db.Proposal, error)
	CreateProposerLookup(ctx context.Context, arg db.CreateProposerLookupParams) (db.ProposerLookup, error)
	DeclineProposal(ctx context.Context, id int64) (int64, error)
	DeleteUnclaimedAddressExclusion(ctx context.Context, addressCidr netip.Prefix) (bool, error)
	GetPendingProposal(ctx context.Context, id int64) (db.Proposal, error)
	ListAddressExclusionCidrs(ctx context.Context) ([]*netip.Prefix, error)
	ListDeclinedProposalScopes(ctx context.Context) ([]db.ListDeclinedProposalScopesRow, error)
	ListPendingProposals(ctx context.Context) ([]db.ListPendingProposalsRow, error)
	RecordSourceAttempt(ctx context.Context, arg db.RecordSourceAttemptParams) (db.SourceHealth, error)
	UndoDeclineProposal(ctx context.Context, id int64) (netip.Prefix, error)
}

type proposalRow struct {
	ID      int64
	Value   string
	Kind    string
	Source  string
	OverCap bool
	Refusal string
}

func flattenProposals(lookups []proposalLookupView) []proposalRow {
	var out []proposalRow
	for _, l := range lookups {
		for _, p := range l.Proposals {
			out = append(out, proposalRow{
				ID: p.ID, Value: p.Scope, Kind: "range", Source: p.Source,
				OverCap: p.OverCap, Refusal: p.Refusal,
			})
		}
	}
	return out
}

type proposerRunner interface {
	Propose(ctx context.Context, orgName string, enabled map[string]bool) ([]proposer.Candidate, []proposer.Attempt, error)
}

type proposalView struct {
	ID          int64
	Scope       string
	Source      string
	RecordLabel string
	OrgName     string
	AddrCount   string
	OverCap     bool
	Refusal     string
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

func toProposalLookups(rows []db.ListPendingProposalsRow, addrCap int) []proposalLookupView {
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
		v := proposalView{
			ID: row.ID, Scope: row.AddressCidr.String(), Source: row.SourceSlug,
			RecordLabel: recordLabel(row.RecordKind), OrgName: row.OrgName,
			AddrCount: humanCount(row.AddressCidr),
		}
		// A confirm is a declaration, so the cap refuses before the click (ADR-0052, #1713).
		if !seed.WithinCap(row.AddressCidr, addrCap) {
			v.OverCap = true
			v.Refusal = overCapProposalNotice(row.AddressCidr, addrCap)
		}
		out[idx].Proposals = append(out[idx].Proposals, v)
		out[idx].Count = len(out[idx].Proposals)
	}
	return out
}

func (s *server) proposalLookups(ctx context.Context) ([]proposalLookupView, error) {
	rows, err := s.proposalsStore.ListPendingProposals(ctx)
	if err != nil {
		return nil, err
	}
	return toProposalLookups(rows, s.addressCap(ctx)), nil
}

func coveringExclusion(scope netip.Prefix, excl []*netip.Prefix) *netip.Prefix {
	for _, e := range excl {
		if e == nil {
			continue
		}
		// A decline claims its prefix, so a wider candidate is still offered (ADR-0012, #1714).
		if e.Bits() <= scope.Bits() && e.Contains(scope.Addr()) {
			return e
		}
	}
	return nil
}

func stillExcludedNotice(e *netip.Prefix) string {
	return fmt.Sprintf(
		"That scope sits under the exclusion %s, and an exclusion refuses ground. Undo every decline that claims %s, or remove the exclusion, then confirm it.",
		e, e,
	)
}

func excludeCandidates(cands []proposer.Candidate, excl []*netip.Prefix) []proposer.Candidate {
	var out []proposer.Candidate
	for _, c := range cands {
		if coveringExclusion(c.Scope.Masked(), excl) == nil {
			out = append(out, c)
		}
	}
	return out
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

	cands, attempts, perr := s.proposals().Propose(r.Context(), query, enabled)
	// The registries were reached, so a lookup that matches nothing still acted (spec §1.3).
	s.recorder().Record(r.Context(), actingAccount(acct), act.ProposalQueried{
		OrgQuery: act.OrgQuery{Term: query},
	})
	// The write precedes every return, so a lookup that finds nothing still records (ADR-0223 §4).
	s.recordProposerAttempts(r.Context(), attempts)
	if perr != nil {
		log.Printf("web: proposer lookup %q: %v", logSafe(query), perr) // #nosec G706 (sanitized via logSafe)
	}
	excl, err := s.proposalsStore.ListAddressExclusionCidrs(r.Context())
	if err != nil {
		s.serverError(w, "list address exclusions", err)
		return
	}
	// Excluded candidates drop before the miss check, so all-excluded is a miss (#1714).
	cands = excludeCandidates(cands, excl)
	if len(cands) == 0 {
		msg := "No candidate scopes matched that name."
		if perr != nil {
			msg = "The lookup could not be completed — a registry path errored and no candidates were found. See the server log for details."
		}
		s.flashScopeBack(w, r, seedsForms{proposalQuery: query, proposalNotice: msg})
		return
	}

	lookup, err := s.proposalsStore.CreateProposerLookup(r.Context(), db.CreateProposerLookupParams{
		Query: query, CreatedBy: acct.ID,
	})
	if err != nil {
		s.serverError(w, "create lookup", err)
		return
	}
	for _, c := range cands {
		if _, err := s.proposalsStore.CreateProposal(r.Context(), db.CreateProposalParams{
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

	p, err := s.proposalsStore.GetPendingProposal(r.Context(), id)
	if err != nil {
		s.backToScope(w, r)
		return
	}

	// A confirmed Proposal is a Seed, so the declaration path's own cap gates it too (#892).
	if addrCap := s.addressCap(r.Context()); !seed.WithinCap(p.AddressCidr, addrCap) {
		s.flashScopeBack(w, r, seedsForms{proposalNotice: overCapProposalNotice(p.AddressCidr, addrCap)})
		return
	}

	// A cidr column rejects host bits, so masking gives an org range dispatch parity (#755).
	cidr := p.AddressCidr.Masked()

	excl, err := s.proposalsStore.ListAddressExclusionCidrs(r.Context())
	if err != nil {
		s.serverError(w, "list address exclusions", err)
		return
	}
	// The queue refuses ground an exclusion covers, so this seed would measure nothing (#1777).
	if e := coveringExclusion(cidr, excl); e != nil {
		s.flashScopeBack(w, r, seedsForms{proposalNotice: stillExcludedNotice(e)})
		return
	}

	sd, err := s.proposalsStore.CreateAddressSeed(r.Context(), db.CreateAddressSeedParams{
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
	if _, err := s.proposalsStore.ConfirmProposal(r.Context(), db.ConfirmProposalParams{
		ID: id, ConfirmedSeedID: pgtype.Int8{Int64: sd.ID, Valid: true},
	}); err != nil {
		s.serverError(w, "confirm proposal", err)
		return
	}
	// What confirmation declares is the scope, so the seed's masked form is the subject (§2.1).
	s.recorder().Record(r.Context(), actingAccount(acct), act.ProposalConfirmed{
		SeedScope: act.SeedScope{Scope: cidr.String()},
	})
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
		p, gerr := s.proposalsStore.GetPendingProposal(r.Context(), id)
		if gerr != nil {
			continue
		}
		if _, err := s.proposalsStore.DeclineProposal(r.Context(), id); err != nil {
			s.serverError(w, "decline proposal", err)
			return
		}
		cidr := p.AddressCidr
		// A decline records an exclusion, so the same range is not proposed again (ADR-0012).
		if _, err := s.proposalsStore.CreateAddressExclusion(r.Context(), db.CreateAddressExclusionParams{
			AddressCidr: &cidr, CreatedBy: acct.ID,
		}); err != nil && !isUniqueViolation(err) {
			s.serverError(w, "record declined proposal as exclusion", err)
			return
		}
		// One row per subject: this loop bails mid-batch, leaving the rest committed (spec §7.6).
		s.recorder().Record(r.Context(), actingAccount(acct), act.ProposalDeclined{
			ExclusionRef: act.ExclusionRef{Kind: "address", Scope: cidr.String()},
		})
	}
	s.backToScope(w, r)
}

func (s *server) declinedProposalScopes(ctx context.Context) map[string][]int64 {
	// No screen lists the declined tail, so the decline's own exclusion row carries it (#1721).
	rows, err := s.proposalsStore.ListDeclinedProposalScopes(ctx)
	if err != nil {
		return nil
	}
	out := make(map[string][]int64, len(rows))
	for _, row := range rows {
		// A proposal scope has no unique constraint, so one range holds two declines (#1777).
		scope := row.AddressCidr.String()
		out[scope] = append(out[scope], row.ID)
	}
	return out
}

func (s *server) undoDecline(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad proposal id", http.StatusBadRequest)
		return
	}
	scope, err := s.proposalsStore.UndoDeclineProposal(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		// A pending or confirmed Proposal is no decline to reverse, so nothing moves (ADR-0022).
		s.backToScope(w, r)
		return
	}
	if err != nil {
		s.serverError(w, "undo declined proposal", err)
		return
	}
	// A declined scope was never declared, so lifting it admits no ground (ADR-0133, #1721).
	claimed, err := s.proposalsStore.DeleteUnclaimedAddressExclusion(r.Context(), scope)
	if err != nil {
		s.serverError(w, "lift declined proposal exclusion", err)
		return
	}
	// Both paths returned the scope to pending, so both recorded the lift (ADR-0022).
	s.recorder().Record(r.Context(), actingAccount(acct), act.ProposalDeclineUndone{
		ExclusionRef: act.ExclusionRef{Kind: "address", Scope: scope.String()},
	})
	if claimed {
		// Both paths return the scope to pending, so both carry the rider (ADR-0022).
		s.toastRedirectBack(w, r, "/scope", "neutral",
			scope.String()+" returned to pending.",
			"Its exclusion stays — another declined proposal still claims that scope. Confirming it is a fresh act.")
		return
	}
	// The second sentence is the rider: undo may not become a confirm shortcut (ADR-0022).
	s.toastRedirectBack(w, r, "/scope", "neutral",
		scope.String()+" returned to pending.", "Confirming it is a fresh act.")
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
