package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/winniel123/verge-asm/internal/db"
)

// The bearer path shares no machinery with the session path, down to the type (ADR-0123 §3).

type apiHandler func(w http.ResponseWriter, r *http.Request, acct db.Account)

type apiAuthStore interface {
	GetInstanceConfig(ctx context.Context) (db.GetInstanceConfigRow, error)
	GetPersonalTokenByHash(ctx context.Context, tokenHash string) (db.PersonalToken, error)
	GetAccountByID(ctx context.Context, id int64) (db.Account, error)
	UpdatePersonalTokenLastUsed(ctx context.Context, id int64) error
}

func (s *server) apiStore() (apiAuthStore, bool) {
	st, ok := s.store.(apiAuthStore)
	return st, ok
}

func (s *server) apiBearer(next apiHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, ok := s.apiStore()
		if !ok {
			// A wired store always satisfies this; the miss fails closed, never a panic on the request path.
			log.Printf("web: api: store lacks bearer capability")
			apiNotFound(w, r)
			return
		}
		// Off is indistinguishable from absent: 404 on every path, never 401 or 403 (ADR-0123 §2).
		cfg, err := st.GetInstanceConfig(r.Context())
		if err != nil {
			log.Printf("web: api: read instance config: %v", err)
			apiNotFound(w, r)
			return
		}
		if !cfg.ApiEnabled {
			apiNotFound(w, r)
			return
		}

		// The router registers GET-only, so this belts a verb no route expresses (ADR-0123 §1).
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		acct, tokenID, ok := resolveAPIToken(r, st)
		if !ok {
			apiUnauthorized(w)
			return
		}

		// The SQL predicate holds the cadence, so a busy token is not a write per request (ADR-0123 §4).
		if err := st.UpdatePersonalTokenLastUsed(r.Context(), tokenID); err != nil {
			log.Printf("web: api: touch token last-used: %v", err)
		}

		next(w, r, acct)
	}
}

func resolveAPIToken(r *http.Request, st apiAuthStore) (acct db.Account, tokenID int64, ok bool) {
	raw, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		return db.Account{}, 0, false
	}
	tok, err := st.GetPersonalTokenByHash(r.Context(), hashToken(raw))
	if err != nil {
		return db.Account{}, 0, false
	}
	// The account is read live, so a demotion binds the next request with no reissue (ADR-0123 §4).
	acct, err = st.GetAccountByID(r.Context(), tok.AccountID)
	if err != nil {
		return db.Account{}, 0, false
	}
	return acct, tok.ID, true
}

func bearerToken(header string) (string, bool) {
	const scheme = "bearer "
	// The auth scheme is case-insensitive (RFC 7235 §2.1).
	if len(header) < len(scheme) || !strings.EqualFold(header[:len(scheme)], scheme) {
		return "", false
	}
	tok := strings.TrimSpace(header[len(scheme):])
	if !strings.HasPrefix(tok, "vg_pat_") {
		return "", false
	}
	return tok, true
}

func apiNotFound(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func apiUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="verge-asm api"`)
	w.WriteHeader(http.StatusUnauthorized)
}
