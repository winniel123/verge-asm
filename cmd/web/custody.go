package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/db"
)

type custodyStore interface {
	SetCustodyExtension(ctx context.Context, arg db.SetCustodyExtensionParams) (pgtype.Text, error)
}

func (s *server) setCustody(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.flashScopeBack(w, r, seedsForms{custodyError: "That scope could not be found."})
		return
	}
	// A stale page posting the end state cannot surprise-withdraw the way a blind flip would.
	extend := r.FormValue("extend") == "true"
	// The scope rides the move's RETURNING, so no read can blank the Act (spec §4.2).
	scope, err := s.custodyStore.SetCustodyExtension(r.Context(), db.SetCustodyExtensionParams{
		ID: id, CustodyExtension: extend,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// A custody extension is a name scope's property, so an address id moves nothing.
		s.backToScope(w, r)
		return
	}
	if err != nil {
		s.serverError(w, "set custody extension", err)
		return
	}
	s.recorder().Record(r.Context(), actingAccount(acct), act.SeedCustodyMoved{
		CustodyMove: act.CustodyMove{Scope: scope.String, Disposition: custodyDisposition(extend)},
	})
	s.backToScope(w, r)
}

func custodyDisposition(extend bool) string {
	if extend {
		return "custody extended"
	}
	return "custody ended"
}
