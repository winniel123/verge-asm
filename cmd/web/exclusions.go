package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
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
	http.Redirect(w, r, "/seeds", http.StatusSeeOther)
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
	http.Redirect(w, r, "/seeds", http.StatusSeeOther)
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
