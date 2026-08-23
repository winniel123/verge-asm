# ADR-0117: a session is a server-side record, so it can be revoked

- **Status:** Accepted
- **Date:** 2026-08-23
- **Ticket:** [#393 Wayfinder: active sessions — a server-side registry with view and revoke](https://github.com/winniel123/verge-asm/issues/393)
- **Supersedes in part:** the stateless-session design as it was **stated in code and UI copy** — `cmd/web/auth.go`, `cmd/web/templates_profile.go`, `cmd/web/templates_signin.go`, `cmd/web/sso.go`, `internal/auth/password.go` — wherever they assert that a session "lapses when it expires rather than being revoked" or that the build "keeps no server-side session store." No ADR or the v1 spec mandated statelessness; the statements live in code and are withdrawn at their sites by the copy-sweep child (#409), per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).
- **Preserves:** [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md) in full.

## Context

Sessions were stateless HMAC-signed cookies (`internal/auth/session.go`): the cookie carried
`{account_id, kind, expires_at}`, signed with the file-backed key on the `web-state` volume,
and `currentAccount` verified the signature and re-read the account row. There was **no record
of an issued session anywhere on the server.** The consequence was stated honestly throughout
the UI: logout and the "end session" action only cleared the caller's *own* cookie, a copy of
the cookie elsewhere stayed valid until `exp`, and a password change could not sign other
sessions out. An operator could neither **see** their active sessions nor **revoke** one — the
capability [#393](https://github.com/winniel123/verge-asm/issues/393) exists to add, both
per-user and application-wide for an admin (offboarding).

Revocation is impossible against a credential the server does not record. Some server-side
state per session is therefore unavoidable. The design tension is with ADR-0053, whose whole
point is that **a read-only database leak must not convert into live admin sessions** — which
is why the signing key lives on a volume and never in Postgres. A naive "store the signed
cookie in a table" would not weaken that (the key is still needed to forge one), but storing
anything the cookie itself contains invites re-use from a dump. The resolution has to *add*
revocability without *removing* the leak-model property.

## Decision

**A session is a row in a `session` table. The cookie carries an opaque, high-entropy session
token; the row stores only that token's hash plus metadata; and `currentAccount` validates the
row — present, not revoked, not expired — on every request.** Revoking a session is setting
`revoked_at`, and it takes effect on that session's next request, exactly as a role change or
account deletion already does.

- **Registry.** `session(id, account_id, token_hash, created_at, last_seen_at, user_agent, ip,
  expires_at, revoked_at)`. `account_id` is `ON DELETE CASCADE` (deleting an account takes its
  sessions with it). Indexed on `token_hash` (the per-request lookup) and `account_id` (the
  personal and admin listings).

- **Cookie.** The cookie carries the opaque token, still wrapped in the existing HMAC signature
  so it stays tamper-evident and the signing path is unchanged. Authentication now needs **both**
  a valid signature **and** a live row — either alone is refused.

- **ADR-0053 is preserved, not weakened.** The HMAC signing key still lives only on `web-state`,
  never in Postgres, so a read-only database leak still cannot **forge** a cookie. And because the
  row keeps only the token's **hash**, a leak cannot **replay** a stored token either — the
  preimage is only ever in the cookie on the client. The registry strictly *adds* revocability;
  it removes nothing the leak model relied on. (The one changed property is intentional and
  desirable: a restore of an old dump no longer silently re-animates sessions — a revoked or
  absent row is dead.)

- **Per-request cost.** One indexed lookup by `token_hash` per request — the account row is
  already read every request, so this is one more point read on the hot path, not a new tier.
  `last_seen_at` is refreshed at most once per minute per session to avoid write amplification.

- **Lifecycle.** Login (password and SSO) inserts a row and sets the cookie to its token; logout
  and "end session" set `revoked_at`; a password **change** revokes every *other* session for the
  account; a password **reset** revokes *all* of them. An admin can revoke any single session or
  every session for an account.

- **Two surfaces.** Personal (any signed-in account, owner-scoped in SQL like personal tokens):
  the Profile lists this account's sessions and revokes them. Admin (`requireAdmin`): a Settings
  surface lists every account's sessions and revokes any, or all for one account.

## Consequences

- **Revocation is real.** "Sign out other sessions", admin offboarding, and password-change
  invalidation all work; the UI copy that said they could not is withdrawn at its sites (#409).
- **A dead cookie fails closed.** An old signed cookie with no live row simply fails validation
  and redirects to `/login` — no crash, no ambiguous state.
- **The leak model is intact.** No secret moves into Postgres; the DB holds only hashes and
  metadata. A dump discloses who was signed in and when, never a usable credential — the same
  posture ADR-0053 already sets for every other stored auth artifact.
- **Sessions are now Operational state with a lifecycle**, so they carry a retirement story:
  expired and revoked rows are dead to auth immediately and may be reaped by a later cadence;
  this ADR ships them unreaped (one row per login is nothing to retire yet), matching the
  `Dispatch` stance in spec §4.6.
- **Forward-only migration.** A single-estate pre-release build with stateless cookies in flight:
  add the table; existing cookies simply fail the new row check once and re-login. No backfill.
