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
	IsAddress bool
	Scope     string
	By        string
	At        string
}

func (s *server) seedsPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderSeeds(w, r, acct, "", "", "")
}

// declareSeed handles a scope declaration. It is reached only through
// requireAdmin, so a viewer can list seeds but never declare one.
func (s *server) declareSeed(w http.ResponseWriter, r *http.Request, acct db.Account) {
	kind := r.FormValue("kind")
	value := strings.TrimSpace(r.FormValue("scope"))

	switch kind {
	case "name":
		domain, err := seed.NormalizeDomain(value)
		if err != nil {
			s.renderSeeds(w, r, acct, err.Error(), kind, value)
			return
		}
		if _, err := s.store.CreateNameSeed(r.Context(), db.CreateNameSeedParams{
			NameDomain: pgtype.Text{String: domain, Valid: true}, CreatedBy: acct.ID,
		}); err != nil {
			s.renderSeeds(w, r, acct, seedCreateError(err, "domain"), kind, value)
			return
		}
	case "address":
		p, err := seed.ParseCIDR(value)
		if err != nil {
			s.renderSeeds(w, r, acct, err.Error(), kind, value)
			return
		}
		if !seed.WithinCap(p, s.seedAddressCap) {
			s.renderSeeds(w, r, acct, fmt.Sprintf(
				"%s covers %s addresses, over the cap of %d — declare a smaller block.",
				p, seed.AddressCount(p), s.seedAddressCap), kind, value)
			return
		}
		if _, err := s.store.CreateAddressSeed(r.Context(), db.CreateAddressSeedParams{
			AddressCidr: &p, CreatedBy: acct.ID,
		}); err != nil {
			s.renderSeeds(w, r, acct, seedCreateError(err, "block"), kind, value)
			return
		}
	default:
		s.renderSeeds(w, r, acct, "Choose a scope type.", kind, value)
		return
	}
	http.Redirect(w, r, "/seeds", http.StatusSeeOther)
}

func (s *server) renderSeeds(w http.ResponseWriter, r *http.Request, acct db.Account, formError, formKind, formScope string) {
	rows, err := s.store.ListSeeds(r.Context())
	if err != nil {
		s.serverError(w, "list seeds", err)
		return
	}
	status := http.StatusOK
	if formError != "" {
		status = http.StatusBadRequest
	}
	s.renderStatus(w, status, "seeds", map[string]any{
		"Title": "Seeds", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"Seeds": toSeedViews(rows), "FormError": formError, "FormKind": formKind,
		"FormScope": formScope, "AddressCap": s.seedAddressCap,
	})
}

func toSeedViews(rows []db.ListSeedsRow) []seedView {
	out := make([]seedView, 0, len(rows))
	for _, row := range rows {
		v := seedView{By: row.CreatedByUsername}
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
