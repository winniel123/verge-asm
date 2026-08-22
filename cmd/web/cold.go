package main

import (
	"net/http"
	"strconv"

	"github.com/winniel123/verge-asm/internal/db"
)

// coldScopeView is a declared Seed shaped for the cold-tier opt-in section: the
// scope, its kind, and whether it has opted into the full-range Scan.
type coldScopeView struct {
	ID        int64
	IsAddress bool
	Scope     string
	OptedIn   bool
}

// toColdScopeViews decorates each Seed with its cold-tier opt-in state, so the
// operator sees which scopes the full-range sweep covers. Every declared scope
// is shown — an un-opted scope as an invitation to opt in.
func toColdScopeViews(seeds []seedView, optedIn []int64) []coldScopeView {
	in := make(map[int64]bool, len(optedIn))
	for _, id := range optedIn {
		in[id] = true
	}
	out := make([]coldScopeView, 0, len(seeds))
	for _, s := range seeds {
		out = append(out, coldScopeView{
			ID: s.ID, IsAddress: s.IsAddress, Scope: s.Scope, OptedIn: in[s.ID],
		})
	}
	return out
}

// setColdScope opts a Seed scope into the full-range cold Scan, or back out (v1
// spec §3.4, ADR-0044). Enabling the cold tier is per-Seed, not global: this
// handler writes the scope opt-in and reconciles the Scan's enabled flag, and
// does NOTHING else — crucially it never dispatches a Scan. Adding a scope
// queues nothing; it only marks the tier enabled, and the cold Scan then fans
// out on its own monthly cadence, never on this config-save. It is reached only
// through requireAdmin, so a viewer can read the opt-in state but never move it.
func (s *server) setColdScope(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.renderSeeds(w, r, acct, seedsForms{coldError: "That scope could not be found."})
		return
	}
	// The form carries the intended end state, not a blind flip: a stale page is
	// idempotent rather than a surprising reversal.
	optIn := r.FormValue("opt_in") == "true"
	if optIn {
		if err := s.store.OptInColdScope(r.Context(), db.OptInColdScopeParams{
			SeedID: id, CreatedBy: acct.ID,
		}); err != nil {
			s.serverError(w, "opt in cold scope", err)
			return
		}
	} else {
		if err := s.store.OptOutColdScope(r.Context(), id); err != nil {
			s.serverError(w, "opt out cold scope", err)
			return
		}
	}
	// Reconcile the tier's enabled flag with its scope. This is the whole of the
	// enablement — the tier is enabled exactly while a scope is opted in — and it
	// is the only thing that puts the cold Scan on the dispatcher's cadence.
	if err := s.store.SyncColdScanEnabled(r.Context()); err != nil {
		s.serverError(w, "sync cold scan enabled", err)
		return
	}
	http.Redirect(w, r, "/scope", http.StatusSeeOther)
}
