package main

import (
	"context"
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

// The Scope screen's Proposals section is now byte-served from the frozen scope.tmpl
// (batch 3, #574), which FOLDS the old repo-authored "proposals" define in. That
// define (proposalTemplates) and its markup are deleted; the handlers below keep their
// POST routes, and renderSeeds shapes the flat .Proposals[{ID,Value,Kind,Source}] rows
// the tmpl reads via flattenProposals.

// proposalRow is one pending Proposal shaped for the frozen scope.tmpl's Proposals
// section (#574): the row id (for confirm/decline forms), the proposed scope Value, its
// Kind label, and the Source that offered it. It is the flat replacement for the old
// per-lookup grouping the folded-away "proposals" define rendered.
type proposalRow struct {
	ID     int64
	Value  string
	Kind   string
	Source string
}

// flattenProposals collapses the per-lookup pending-proposal grouping into the flat
// rows scope.tmpl renders, preserving the query's order (lookup-newest, proposal-oldest).
// A proposer answers with address scopes, so every row's Kind is "range" (#574).
func flattenProposals(lookups []proposalLookupView) []proposalRow {
	var out []proposalRow
	for _, l := range lookups {
		for _, p := range l.Proposals {
			out = append(out, proposalRow{ID: p.ID, Value: p.Scope, Kind: "range", Source: p.Source})
		}
	}
	return out
}

// proposerRunner runs the enabled keyless proposer paths for one operator
// lookup. It is the seam ADR-0012's paths sit behind: production wires the real
// registry (real HTTP), tests inject a fake, so no lookup touches the network
// under test.
type proposerRunner interface {
	Propose(ctx context.Context, orgName string, enabled map[string]bool) ([]proposer.Candidate, error)
}

// proposalView is one pending Proposal shaped for rendering. It never reads as a
// Seed: it is a candidate the operator has not confirmed, and it carries which
// kind of record produced it so its caveat stays visible (ADR-0012). AddrCount
// is rendered inside the confirm affordance's own label, which is what makes the
// gesture identical at every size (ADR-0022).
type proposalView struct {
	ID          int64
	Scope       string
	Source      string
	RecordLabel string
	OrgName     string
	AddrCount   string
}

// proposalLookupView groups every candidate one operator search produced. The
// group is the unit a bulk decline operates over — declining is done over a
// whole lookup at once (ADR-0022) — while confirmation stays per-Proposal.
type proposalLookupView struct {
	LookupID  int64
	Query     string
	By        string
	At        string
	Count     int
	Proposals []proposalView
}

// recordLabel renders a Proposal's record kind in the operator's words.
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

// humanCount renders an address count with thousands separators so the confirm
// label reads "Confirm 8,388,608 addresses" rather than a wall of digits.
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

// toProposalLookups groups pending proposal rows (already ordered lookup-newest,
// proposal-oldest by the query) into the render shape.
func toProposalLookups(rows []db.ListPendingProposalsRow) []proposalLookupView {
	var out []proposalLookupView
	byLookup := map[int64]int{} // lookup id -> index in out
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

// proposalLookups reads the pending Proposals for rendering on the Seeds screen.
func (s *server) proposalLookups(ctx context.Context) ([]proposalLookupView, error) {
	rows, err := s.store.ListPendingProposals(ctx)
	if err != nil {
		return nil, err
	}
	return toProposalLookups(rows), nil
}

// runLookup answers an operator's org-name search: it runs every enabled keyless
// proposer and files each candidate as a pending Proposal under one lookup. It
// produces Proposal rows and never Observation rows — a proposer admits nothing.
// Proposals are produced only in answer to this act, never on a cadence.
func (s *server) runLookup(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// The frozen scope.tmpl's org-name search posts `org` (POST /proposals/search); the
	// legacy route posts `query`. Accept either so both the tmpl form and existing
	// callers reach the one lookup path (#574).
	query := strings.TrimSpace(r.FormValue("org"))
	if query == "" {
		query = strings.TrimSpace(r.FormValue("query"))
	}
	if query == "" {
		s.renderSeeds(w, r, acct, seedsForms{proposalError: "Enter an organisation name to search."})
		return
	}

	enabled, err := s.enabledProposers(r)
	if err != nil {
		s.serverError(w, "list source states", err)
		return
	}

	cands, perr := s.proposals().Propose(r.Context(), query, enabled)
	if perr != nil {
		// A proposer path errored. perr is a join of "<slug>: <err>" entries, so
		// logging it with the query names both the failing paths and the search
		// a maintainer needs to correlate against — the reason must not be
		// silently thrown away (#251).
		log.Printf("web: proposer lookup %q: %v", query, perr)
	}
	if len(cands) == 0 {
		// Distinguish a backend failure from a genuine no-match: a non-nil perr
		// means a path could not answer, so this is not "your name matched
		// nothing" and must not read as one.
		msg := "No candidate scopes matched that name."
		if perr != nil {
			msg = "The lookup could not be completed — a registry path errored and no candidates were found. See the server log for details."
		}
		s.renderSeeds(w, r, acct, seedsForms{proposalQuery: query, proposalNotice: msg})
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
		// Partial failure: some paths returned candidates (now filed) while
		// another errored, so this list may be missing scopes. Redirect exactly
		// like a clean success — the filed candidates persist, so a plain inline
		// render would re-file duplicates on refresh — but carry a flag that has
		// the Seeds page surface the incompleteness (see seedsPage).
		http.Redirect(w, r, "/scope?notice="+noticePartialProposals, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/scope", http.StatusSeeOther)
}

// noticePartialProposals is the /scope query flag a partial-failure lookup
// redirects with (retargeted from /seeds → /scope in #286), and
// partialProposalNotice is the message the Scope page then renders. Carrying the
// caveat through the redirect (rather than inline off the POST) keeps the search
// idempotent on refresh while still surfacing that some registry path errored (#251).
const (
	noticePartialProposals = "partial-proposals"
	partialProposalNotice  = "Showing partial results — one or more registry paths errored, so this list may be incomplete. See the server log for details."
)

// confirmProposal confirms exactly one Proposal into exactly one Seed. It is
// singular by construction — one id per request, no batch (ADR-0022) — and the
// resulting Seed retains the Proposal as provenance. Confirming a proposed scope
// is not bound by the operator's own address-scope cap: the cap governs scopes
// the operator types, while a Proposal is a registry-authored range the operator
// confirms whole (ADR-0022's "Confirm N addresses" is the same gesture at every
// size).
func (s *server) confirmProposal(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad proposal id", http.StatusBadRequest)
		return
	}

	p, err := s.store.GetPendingProposal(r.Context(), id)
	if err != nil {
		// Already confirmed, declined, or never existed — a repeat submit opens
		// no second gate. Return to the screen rather than erroring.
		http.Redirect(w, r, "/scope", http.StatusSeeOther)
		return
	}

	cidr := p.AddressCidr
	sd, err := s.store.CreateAddressSeed(r.Context(), db.CreateAddressSeedParams{
		AddressCidr: &cidr, CreatedBy: acct.ID,
	})
	if err != nil {
		if isUniqueViolation(err) {
			s.renderSeeds(w, r, acct, seedsForms{proposalError: "That scope is already declared as a seed."})
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
	http.Redirect(w, r, "/scope", http.StatusSeeOther)
}

// declineLookup declines every still-pending Proposal under one lookup in a
// single act (ADR-0022). Declining is safe to batch: a pending Proposal is read
// by nothing, so declining and never answering have the same effect on the gate.
//
// A decline is a boundary claim — the operator says these proposed scopes are
// not theirs — so each declined scope is recorded as an address exclusion
// (Scope.jsx: "declines are recorded as exclusions"). Recording the exclusion
// makes the decline durable: the same range does not silently re-enter the
// estate. The pending rows are read first, while they are still pending, so their
// scopes survive the decline; an already-excluded scope is left as-is.
func (s *server) declineLookup(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode {
		http.Redirect(w, r, "/scope", http.StatusSeeOther)
		return
	}
	// The frozen scope.tmpl declines the CHECKED proposals: the checkboxes post their
	// ids under `ids` (form-attribute association), so decline-many operates over the
	// selected proposals rather than a whole lookup (#574). Each still-pending scope is
	// read (while pending) so it can be recorded as an address exclusion, making the
	// decline durable — the same range does not silently re-enter the estate.
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	raw := r.Form["ids"]
	if len(raw) == 0 {
		// Nothing selected — a stale or empty submit is a no-op, not an error.
		http.Redirect(w, r, "/scope", http.StatusSeeOther)
		return
	}
	for _, idStr := range raw {
		id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
		if err != nil {
			continue
		}
		// Read the proposal while it is still pending so its scope survives the decline.
		p, gerr := s.store.GetPendingProposal(r.Context(), id)
		if gerr != nil {
			continue // already spent, or never existed — a repeat submit opens no gate.
		}
		if _, err := s.store.DeclineProposal(r.Context(), id); err != nil {
			s.serverError(w, "decline proposal", err)
			return
		}
		cidr := p.AddressCidr
		if _, err := s.store.CreateAddressExclusion(r.Context(), db.CreateAddressExclusionParams{
			AddressCidr: &cidr, CreatedBy: acct.ID,
		}); err != nil && !isUniqueViolation(err) {
			s.serverError(w, "record declined proposal as exclusion", err)
			return
		}
	}
	http.Redirect(w, r, "/scope", http.StatusSeeOther)
}

// enabledProposers returns the keyless proposer slugs the operator has left
// enabled, so a lookup runs only the paths the source-enablement state permits.
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

// proposals returns the proposer runner, defaulting to an empty registry so a
// server constructed without one simply proposes nothing rather than panicking.
func (s *server) proposals() proposerRunner {
	if s.proposer == nil {
		return proposer.NewRegistry()
	}
	return s.proposer
}
