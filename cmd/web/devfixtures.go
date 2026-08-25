package main

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
)

// Dev-only pixel-parity affordances for screen 2 (ErrorPage, package v3.5.0,
// WORK-ORDER-2-ERROR.md). Everything here is gated to a VERGE_DEV build — the /dev
// routes are registered only when s.devMode (handlers.go) and the seed helpers run
// only from the -seed-fixtures one-shot, which main.go bars outside VERGE_DEV — so no
// dev surface exists in a real deployment. It exists so the harness can drive the six
// error capture states states.json declares, deterministically.

// devFixtureIncidentID is the incident id the 500 error page shows in a VERGE_DEV build,
// so the harness's 500 golden is stable. It mirrors design-system/fixtures/fixtures.json
// → error.incident_id; TestDevFixturesMatchPackage asserts the two never drift. A real
// build keeps the crypto/rand id (newIncidentID, errors.go).
const devFixtureIncidentID = "err_9f3ka72c"

// devFixtureAccount is one design fixture login the harness signs in as. states.json
// carries a per-state `session` ("admin" | "viewer"); the dev session mint resolves that
// role to one of these accounts and opens a session before the state's route is captured.
// Dev-only — well-known passwords, seeded only under VERGE_DEV, mirroring fixtures.json →
// accounts (asserted by TestDevFixturesMatchPackage). The password is #nosec G101: not a
// real credential, only ever seeded into a throwaway dev database.
type devFixtureAccount struct {
	username string
	role     string
	password string // #nosec G101 -- dev-only fixture login, seeded only under VERGE_DEV
}

// devFixtureAccounts pins fixtures.json → accounts: an admin and a viewer. One account
// per role — the mint resolves states.json's `session` role to the account here.
var devFixtureAccounts = []devFixtureAccount{
	{username: "ola.perez", role: roleAdmin, password: "verge-dev-1"},
	{username: "sam.reader", role: roleViewer, password: "verge-dev-2"},
}

// devPanic is the /dev/panic handler (VERGE_DEV only): it panics so recoverPanics
// renders the 500 error page with the deterministic incident id — the harness's 500
// capture state. The panic value is fixed for a stable host-log line.
func (s *server) devPanic(http.ResponseWriter, *http.Request) {
	panic("fixture")
}

// devSessionMint is the /dev/session/{role} handler (VERGE_DEV only): it opens a session
// as the fixture account for the requested role ("admin" | "viewer") and hands back the
// session cookie via completeLogin, so the harness can establish states.json's per-state
// `session` before it navigates the state's route. An unknown role, or a role whose
// fixture account was not seeded, is a plain 404 — the mint never invents an account.
func (s *server) devSessionMint(w http.ResponseWriter, r *http.Request) {
	role := r.PathValue("role")
	username := ""
	for _, a := range devFixtureAccounts {
		if a.role == role {
			username = a.username
			break
		}
	}
	if username == "" {
		s.notFound(w, r)
		return
	}
	acct, err := s.store.GetAccountByUsername(r.Context(), username)
	if err != nil {
		s.notFound(w, r)
		return
	}
	s.completeLogin(w, r, acct.ID)
}

// seedDevFixtureAccounts makes the fixtures.json admin + viewer exist so the dev session
// mint can sign in as either. It is idempotent — an account already present (by username)
// is left untouched — so a re-seed is a no-op, and it runs only from the -seed-fixtures
// one-shot (main.go, VERGE_DEV-gated), beside seedInventoryFixtures/seedDevOperator.
func seedDevFixtureAccounts(ctx context.Context, pool *pgxpool.Pool) error {
	q := db.New(pool)
	for _, a := range devFixtureAccounts {
		if _, err := q.GetAccountByUsername(ctx, a.username); err == nil {
			continue // already seeded
		}
		hash, err := auth.HashPassword(a.password)
		if err != nil {
			return err
		}
		if _, err := q.CreateAccount(ctx, db.CreateAccountParams{
			Username: a.username, Role: a.role, PasswordHash: hash,
		}); err != nil {
			return err
		}
	}
	return nil
}
