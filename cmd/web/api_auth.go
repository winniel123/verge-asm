package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/winniel123/verge-asm/internal/db"
)

// apiHandler is a read-only /api/v1 handler that has already had its caller resolved
// from a Bearer personal token. It is the API twin of authedHandler, and deliberately
// NOT the same type: the bearer path shares no machinery with the session path (ADR-0123
// §3), so the two request-principal shapes stay distinct at the type level too.
type apiHandler func(w http.ResponseWriter, r *http.Request, acct db.Account)

// apiAuthStore is the narrow slice of the store the bearer path needs: the enable gate,
// the by-hash token lookup, the live account read, and the coarsened last-used touch.
// It is declared here rather than folded into cmd/web/handlers.go's shared store
// interface on purpose — that interface is an append-only, serial-merge friction point
// across the parallel A/B children (#658 seams), so A2 keeps its two new methods out of
// it and reaches them through a local capability interface the concrete *db.Queries and
// the test fake both satisfy. A3 mounts the routes; this stays A2's own seam.
type apiAuthStore interface {
	GetInstanceConfig(ctx context.Context) (db.GetInstanceConfigRow, error)
	GetPersonalTokenByHash(ctx context.Context, tokenHash string) (db.PersonalToken, error)
	GetAccountByID(ctx context.Context, id int64) (db.Account, error)
	UpdatePersonalTokenLastUsed(ctx context.Context, id int64) error
}

// apiStore adapts the server's store to the bearer path's capability view. The concrete
// store (*db.Queries in production, the fake in tests) satisfies apiAuthStore in full, so
// this never fails in a wired server; the comma-ok guard fails closed — treating a store
// that somehow lacks the capability as a surface that cannot be served — rather than
// panicking on the request path.
func (s *server) apiStore() (apiAuthStore, bool) {
	st, ok := s.store.(apiAuthStore)
	return st, ok
}

// apiBearer is the authentication spine for the read-only /api/v1 surface (#390,
// ADR-0123). A3 wraps its read handlers in it; no route is mounted here. The path is
// fully separate from the session/cookie surface — it never reads or sets a cookie,
// never mints a session, never accepts the session cookie, and never calls
// currentAccount — so a stolen cookie cannot drive the API and a stolen token cannot
// drive the HTML app. It enforces, in this order:
//
//  1. Surface-off beats auth. When instance_config.api_enabled is false — the ship
//     default — every path 404s, byte-indistinguishable from a build with no API at
//     all (ADR-0123 §2). The check precedes any token work, and a config-read failure
//     fails closed the same way rather than leaking the surface's existence.
//  2. Read-only is a property of the surface, not a per-endpoint permission. Any
//     non-GET verb is refused 405 with no body detail, before the credential is even
//     read — no mutating verb is expressible under /api/v1 (ADR-0123 §1).
//  3. The token authenticates which account; the account's role is read LIVE from its
//     row on every request (ADR-0123 §4), so a demotion or deletion takes effect on the
//     token's very next request with no reissue. A malformed or absent header, an
//     unknown token, or a token whose account no longer exists is a uniform 401.
//  4. A resolved request writes the coarsened (at most once per hour per token)
//     last-used touch, best-effort, then proceeds to the read handler.
func (s *server) apiBearer(next apiHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, ok := s.apiStore()
		if !ok {
			// A server whose store lacks the bearer capability cannot serve the surface;
			// fail closed as surface-off rather than panic. Unreachable in a wired build.
			log.Printf("web: api: store lacks bearer capability")
			apiNotFound(w, r)
			return
		}
		// (1) Surface-off beats auth — checked before token validity. A disabled or
		// unreadable surface answers 404 for every path and every verb, so a probe
		// cannot distinguish "API exists but is off" from "this build has no API": no
		// 401/403 is ever emitted for a disabled instance.
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

		// (2) Read-only surface: no mutating verb is routed here. Reject any non-GET
		// with 405 and no body detail, before the credential is touched.
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// (3) Resolve the bearer credential to a live account. Every failure — no
		// header, wrong scheme, unknown token, removed account — is the same 401.
		acct, tokenID, ok := resolveAPIToken(r, st)
		if !ok {
			apiUnauthorized(w)
			return
		}

		// (4) Coarsened last-used touch: at most once per hour per token (the SQL
		// predicate enforces the cadence, so a busy token is not one write per request
		// and last_used_at never regresses). Best-effort — a failure logs and never
		// fails the read, since it feeds only the "is this token still live?" display.
		if err := st.UpdatePersonalTokenLastUsed(r.Context(), tokenID); err != nil {
			log.Printf("web: api: touch token last-used: %v", err)
		}

		next(w, r, acct)
	}
}

// resolveAPIToken turns a request's Authorization header into the live account behind
// its Bearer personal token, plus the token's id for the last-used touch. It reads ONLY
// the header (never the session cookie), hashes the presented plaintext with the same
// SHA-256 keeper the token was stored under, looks the row up by that hash, and then
// reads the account row LIVE so the role is never frozen into the token (ADR-0123 §4). A
// removed account — whose token row is gone, or whose account lookup fails — resolves to
// ok=false, which the caller renders as 401.
func resolveAPIToken(r *http.Request, st apiAuthStore) (acct db.Account, tokenID int64, ok bool) {
	raw, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		return db.Account{}, 0, false
	}
	tok, err := st.GetPersonalTokenByHash(r.Context(), hashToken(raw))
	if err != nil {
		return db.Account{}, 0, false
	}
	acct, err = st.GetAccountByID(r.Context(), tok.AccountID)
	if err != nil {
		return db.Account{}, 0, false
	}
	return acct, tok.ID, true
}

// bearerToken extracts a personal API token from an Authorization header value. It
// accepts only the `Bearer vg_pat_…` shape: the scheme is matched case-insensitively
// (RFC 7235 §2.1), the credential must carry the vg_pat_ prefix every minted token has,
// and anything else — a Basic header, a bare token, an empty value — is refused. Nothing
// here consults the session cookie; this is the sole credential the API path reads.
func bearerToken(header string) (string, bool) {
	const scheme = "bearer "
	if len(header) < len(scheme) || !strings.EqualFold(header[:len(scheme)], scheme) {
		return "", false
	}
	tok := strings.TrimSpace(header[len(scheme):])
	if !strings.HasPrefix(tok, "vg_pat_") {
		return "", false
	}
	return tok, true
}

// apiNotFound answers a disabled or absent /api/v1 surface with a bare 404 that shares
// no machinery with the HTML error page (which reads the session account) — so a probe
// cannot tell a disabled instance from one built with no API at all (ADR-0123 §2). It
// mirrors net/http's own "404 page not found", the exact body an unrouted path returns.
func apiNotFound(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func apiUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="verge-asm api"`)
	w.WriteHeader(http.StatusUnauthorized)
}
