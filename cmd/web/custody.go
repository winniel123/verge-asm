package main

import (
	"context"
	"net/http"
	"strconv"

	"github.com/winniel123/verge-asm/internal/db"
)

type custodyStore interface {
	SetCustodyExtension(ctx context.Context, arg db.SetCustodyExtensionParams) error
}

func (s *server) setCustody(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.flashScopeBack(w, r, seedsForms{custodyError: "That scope could not be found."})
		return
	}
	// A stale page posting the end state cannot surprise-withdraw the way a blind flip would.
	extend := r.FormValue("extend") == "true"
	if err := s.custodyStore.SetCustodyExtension(r.Context(), db.SetCustodyExtensionParams{
		ID: id, CustodyExtension: extend,
	}); err != nil {
		s.serverError(w, "set custody extension", err)
		return
	}
	s.backToScope(w, r)
}
