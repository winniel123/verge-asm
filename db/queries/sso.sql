-- Reads and writes behind the OIDC single-sign-on config (#293, ADR-0112) and the
-- verified-identity bindings that authentication keys on (#319, ADR-0113). The client
-- secret is write-only at the interface, mirroring the channel secret (ADR-0053): the
-- list/get reads expose only whether one is set, and exactly one read path
-- (GetSSOProviderForAuth) hands the secret to the token exchange.

-- name: InsertSSOProvider :one
-- Declare one OIDC provider. Returns the id only; the secret is write-only and no
-- read query hands it back. A public (PKCE-only) client passes a NULL secret.
INSERT INTO sso_provider (slug, name, issuer, client_id, client_secret, enabled, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: ListSSOProviders :many
-- Every configured provider, newest-first, for the Settings tab. Never selects the
-- secret: it exposes only whether one is set, so the render path cannot leak it.
SELECT p.id, p.slug, p.name, p.issuer, p.client_id, p.enabled,
       (p.client_secret IS NOT NULL)::boolean AS has_secret,
       p.created_by, p.created_at, p.updated_at,
       a.username AS created_by_username
FROM sso_provider p
JOIN account a ON a.id = p.created_by
ORDER BY p.id DESC;

-- name: ListEnabledSSOProviders :many
-- The enabled providers the SignIn screen renders a button for, newest-first. No
-- secret, and no created-by join — SignIn is pre-auth and needs only what a button
-- carries: the slug (its route) and the display name.
SELECT id, slug, name
FROM sso_provider
WHERE enabled = TRUE
ORDER BY id DESC;

-- name: GetSSOProvider :one
-- One provider for the Settings edit form. Omits the secret; a caller reads presence,
-- never the value.
SELECT id, slug, name, issuer, client_id, enabled,
       (client_secret IS NOT NULL)::boolean AS has_secret, created_by, created_at, updated_at
FROM sso_provider
WHERE id = $1;

-- name: GetSSOProviderForAuth :one
-- The ONE read path that selects the secret: the server-side OIDC flow (both a login
-- match and a Profile self-link) needs the issuer, client id and client secret to
-- complete the confidential-client token exchange. Keyed by slug (the flow route) and
-- gated on enabled, so a disabled provider's flow resolves no row and is refused.
SELECT id, slug, name, issuer, client_id, client_secret
FROM sso_provider
WHERE slug = $1 AND enabled = TRUE;

-- name: UpdateSSOProvider :execrows
-- Updates everything but the secret; the secret has its own write path, so an edit
-- that leaves it blank keeps the existing one untouched (exactly the channel pattern).
-- Returns the rows affected so the handler can tell a stale edit (an id deleted in
-- another tab) from a real update, rather than reporting a phantom success.
UPDATE sso_provider
SET slug = $2, name = $3, issuer = $4, client_id = $5, enabled = $6, updated_at = now()
WHERE id = $1;

-- name: SetSSOProviderSecret :exec
-- Set, replace or clear the secret. A NULL clears it (a public PKCE-only client); the
-- value is written and never read back through any interface query.
UPDATE sso_provider SET client_secret = $2, updated_at = now() WHERE id = $1;

-- name: DeleteSSOProvider :exec
DELETE FROM sso_provider WHERE id = $1;

-- name: InsertSSOIdentity :exec
-- Record a verified (provider, sub) → account binding, established by an authenticated
-- Profile self-link (ADR-0113). The UNIQUE(provider_id, sub) guards a second account
-- from claiming an identity already bound; the caller checks for an existing binding
-- first so it can distinguish "already yours" from "bound elsewhere".
INSERT INTO sso_identity (provider_id, account_id, sub, display_name)
VALUES ($1, $2, $3, $4);

-- name: GetAccountBySSOIdentity :one
-- The SSO login match: resolve the local account a verified (provider, sub) is bound to.
-- Keyed on the stable, non-reassignable subject — never a username. No row is an honest
-- refusal (the identity is unlinked), never a provision.
SELECT a.id, a.username, a.role, a.password_hash, a.totp_secret, a.totp_enabled, a.created_at
FROM sso_identity i
JOIN account a ON a.id = i.account_id
WHERE i.provider_id = $1 AND i.sub = $2;

-- name: GetSSOIdentityBySub :one
-- Whether a (provider, sub) is already bound, and to whom — so the self-link flow can
-- no-op an identity already linked to the caller and refuse one linked elsewhere,
-- rather than surfacing a raw unique-violation.
SELECT id, account_id, display_name
FROM sso_identity
WHERE provider_id = $1 AND sub = $2;

-- name: ListSSOIdentitiesForAccount :many
-- An account's own linked identities for its Profile, newest-first. Joined to the
-- provider for the display name/slug; sub is not surfaced (opaque, of no use to a human).
SELECT i.id, i.provider_id, p.slug AS provider_slug, p.name AS provider_name,
       i.display_name, i.created_at
FROM sso_identity i
JOIN sso_provider p ON p.id = i.provider_id
WHERE i.account_id = $1
ORDER BY i.id DESC;

-- name: DeleteSSOIdentityForAccount :execrows
-- A user unlinks their OWN identity (Profile). Scoped to the account so one user can
-- never unlink another's; returns rows so a stale or foreign id no-ops honestly.
DELETE FROM sso_identity WHERE id = $1 AND account_id = $2;

-- name: ListSSOBindings :many
-- Every binding for the admin SSO settings — the offboarding / seat-reassignment view.
-- Joined to provider and account so the admin sees which identity maps to whom, newest
-- first.
SELECT i.id, i.provider_id, p.slug AS provider_slug, p.name AS provider_name,
       i.account_id, a.username AS account_username, i.display_name, i.created_at
FROM sso_identity i
JOIN sso_provider p ON p.id = i.provider_id
JOIN account a ON a.id = i.account_id
ORDER BY i.id DESC;

-- name: DeleteSSOIdentity :exec
-- An admin removes any binding by id (offboarding / seat reassignment). Idempotent:
-- removing a row already gone satisfies the intent either way.
DELETE FROM sso_identity WHERE id = $1;
