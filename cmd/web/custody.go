package main

import (
	"net/http"
	"strconv"

	"github.com/winniel123/verge-asm/internal/db"
)

// setCustody declares or withdraws a custody extension on a name-scope Seed (v1
// spec §3.2, §6.4; ADR-0013): the operator's declaration that the addresses its
// names resolve to are inside the boundary, and therefore under their Custody.
// It is off by default and toggled by a single act — declaring flips the flag on,
// withdrawing flips it off — since the extension is Declared input with no
// timeline. It is reached only through requireAdmin, so a viewer can read a
// scope's custody state but never move it.
//
// This ticket (#186) implements the declaration mechanism alone. The custody
// *derivation* — which addresses the extension actually reaches, following
// resolution out of the declared zone and stopping at a foreign name or a
// non-globally-reachable address — is ticket 13, which reads this flag alongside
// the Seeds. Nothing here computes coverage.
func (s *server) setCustody(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.renderSeeds(w, r, acct, seedsForms{custodyError: "That scope could not be found."})
		return
	}
	// The form carries the intended end state, not a blind flip: a stale page
	// submitting "declare" on an already-declared scope is idempotent rather than
	// a surprising withdrawal.
	extend := r.FormValue("extend") == "true"
	if err := s.store.SetCustodyExtension(r.Context(), db.SetCustodyExtensionParams{
		ID: id, CustodyExtension: extend,
	}); err != nil {
		s.serverError(w, "set custody extension", err)
		return
	}
	http.Redirect(w, r, "/seeds", http.StatusSeeOther)
}
