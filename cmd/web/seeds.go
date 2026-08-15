package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/seed"
)

// seedView is a declared Seed shaped for rendering: the scope collapsed to one
// display string, with the kind kept so name and address scopes stay visually
// distinct.
type seedView struct {
	ID        int64
	IsAddress bool
	Scope     string
	By        string
	At        string
	// CustodyExtension is the name scope's declared custody extension — the
	// operator's assertion that the addresses its names resolve to are under
	// their Custody. Off by default and meaningful on name scopes alone.
	CustodyExtension bool
}

// seedsForms carries the echo state of the two forms the Seeds screen hosts —
// the scope declaration and the exclusion — so a rejected submission on one
// leaves its own error and typed value in place without disturbing the other.
type seedsForms struct {
	seedError, seedKind, seedScope                  string
	exclError, exclKind, exclValue                  string
	custodyError                                    string
	proberError, proberHost, proberPort, proberUser string
	// The org-name lookup echo: an error keeps the search box populated on a
	// rejected submit, a notice reports a lookup that returned no candidates.
	proposalError, proposalNotice, proposalQuery string
}

// nameScopes returns the name-scope subset of a seed listing, in the same order.
// The custody-extension section is over name scopes alone: an address scope is
// its own complete enumeration and carries no extension.
func nameScopes(views []seedView) []seedView {
	out := make([]seedView, 0, len(views))
	for _, v := range views {
		if !v.IsAddress {
			out = append(out, v)
		}
	}
	return out
}

func (s *server) seedsPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderSeeds(w, r, acct, seedsForms{})
}

// declareSeed handles a scope declaration. It is reached only through
// requireAdmin, so a viewer can list seeds but never declare one.
func (s *server) declareSeed(w http.ResponseWriter, r *http.Request, acct db.Account) {
	kind := r.FormValue("kind")
	value := strings.TrimSpace(r.FormValue("scope"))
	fail := func(msg string) {
		s.renderSeeds(w, r, acct, seedsForms{seedError: msg, seedKind: kind, seedScope: value})
	}

	switch kind {
	case "name":
		domain, err := seed.NormalizeDomain(value)
		if err != nil {
			fail(err.Error())
			return
		}
		if _, err := s.store.CreateNameSeed(r.Context(), db.CreateNameSeedParams{
			NameDomain: pgtype.Text{String: domain, Valid: true}, CreatedBy: acct.ID,
		}); err != nil {
			fail(seedCreateError(err, "domain"))
			return
		}
	case "address":
		p, err := seed.ParseCIDR(value)
		if err != nil {
			fail(err.Error())
			return
		}
		if !seed.WithinCap(p, s.seedAddressCap) {
			fail(fmt.Sprintf(
				"%s covers %s addresses, over the cap of %d — declare a smaller block.",
				p, seed.AddressCount(p), s.seedAddressCap))
			return
		}
		if _, err := s.store.CreateAddressSeed(r.Context(), db.CreateAddressSeedParams{
			AddressCidr: &p, CreatedBy: acct.ID,
		}); err != nil {
			fail(seedCreateError(err, "block"))
			return
		}
	default:
		fail("Choose a scope type.")
		return
	}
	http.Redirect(w, r, "/seeds", http.StatusSeeOther)
}

func (s *server) renderSeeds(w http.ResponseWriter, r *http.Request, acct db.Account, f seedsForms) {
	rows, err := s.store.ListSeeds(r.Context())
	if err != nil {
		s.serverError(w, "list seeds", err)
		return
	}
	excl, err := s.store.ListExclusions(r.Context())
	if err != nil {
		s.serverError(w, "list exclusions", err)
		return
	}
	probers, err := s.store.ListVantages(r.Context())
	if err != nil {
		s.serverError(w, "list vantages", err)
		return
	}
	lookups, err := s.proposalLookups(r.Context())
	if err != nil {
		s.serverError(w, "list proposals", err)
		return
	}
	status := http.StatusOK
	if f.seedError != "" || f.exclError != "" || f.custodyError != "" || f.proberError != "" || f.proposalError != "" {
		status = http.StatusBadRequest
	}
	seeds := toSeedViews(rows)
	s.renderStatus(w, status, "seeds", map[string]any{
		"Title": "Seeds", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"Seeds": seeds, "AddressCap": s.seedAddressCap,
		"FormError": f.seedError, "FormKind": f.seedKind, "FormScope": f.seedScope,
		"Exclusions": toExclusionViews(excl),
		"ExclError":  f.exclError, "ExclKind": f.exclKind, "ExclValue": f.exclValue,
		// The custody-extension section reads name scopes alone — an address
		// scope can never carry one.
		"CustodyScopes": nameScopes(seeds), "CustodyError": f.custodyError,
		"Probers":     toProberViews(probers),
		"ProberError": f.proberError, "ProberHost": f.proberHost,
		"ProberPort": f.proberPort, "ProberUser": f.proberUser,
		// Pending Proposals and the org-name lookup echo (#210).
		"ProposalLookups": lookups,
		"ProposalError":   f.proposalError, "ProposalNotice": f.proposalNotice,
		"ProposalQuery": f.proposalQuery,
	})
}

func toSeedViews(rows []db.ListSeedsRow) []seedView {
	out := make([]seedView, 0, len(rows))
	for _, row := range rows {
		v := seedView{ID: row.ID, By: row.CreatedByUsername, CustodyExtension: row.CustodyExtension}
		if row.Kind == "address" && row.AddressCidr != nil {
			v.IsAddress = true
			v.Scope = row.AddressCidr.String()
		} else {
			v.Scope = row.NameDomain.String
		}
		if row.CreatedAt.Valid {
			v.At = row.CreatedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
		}
		out = append(out, v)
	}
	return out
}

func seedCreateError(err error, noun string) string {
	if isUniqueViolation(err) {
		return "That " + noun + " is already declared."
	}
	return "Could not declare the scope."
}
