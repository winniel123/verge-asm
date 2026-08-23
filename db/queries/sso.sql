-- Reads and writes behind the OIDC single-sign-on config (#293, ADR-0112): the
-- SignIn buttons, the Settings → single-sign-on tab, and the server-side flow. The
-- client secret is write-only at the interface, mirroring the channel secret
-- (ADR-0053): the list/get reads expose only whether one is set, and exactly one
-- read path (GetSSOProviderForAuth) hands the secret to the token exchange.

-- name: InsertSSOProvider :one
-- Declare one OIDC provider. Returns the id only; the secret is write-only and no
-- read query hands it back. A public (PKCE-only) client passes a NULL secret.
INSERT INTO sso_provider (slug, name, issuer, client_id, client_secret, username_claim, enabled, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id;

-- name: ListSSOProviders :many
-- Every configured provider, newest-first, for the Settings tab. Never selects the
-- secret: it exposes only whether one is set, so the render path cannot leak it.
SELECT p.id, p.slug, p.name, p.issuer, p.client_id, p.username_claim, p.enabled,
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
SELECT id, slug, name, issuer, client_id, username_claim, enabled,
       (client_secret IS NOT NULL)::boolean AS has_secret, created_by, created_at, updated_at
FROM sso_provider
WHERE id = $1;

-- name: GetSSOProviderForAuth :one
-- The ONE read path that selects the secret: the server-side OIDC flow needs the
-- issuer, client id and client secret to complete the confidential-client token
-- exchange, plus the username claim to map the verified identity to a local account.
-- Keyed by slug (the flow route) and gated on enabled, so a disabled provider's flow
-- resolves no row and is refused.
SELECT id, slug, name, issuer, client_id, client_secret, username_claim
FROM sso_provider
WHERE slug = $1 AND enabled = TRUE;

-- name: UpdateSSOProvider :exec
-- Updates everything but the secret; the secret has its own write path, so an edit
-- that leaves it blank keeps the existing one untouched (exactly the channel pattern).
UPDATE sso_provider
SET slug = $2, name = $3, issuer = $4, client_id = $5, username_claim = $6,
    enabled = $7, updated_at = now()
WHERE id = $1;

-- name: SetSSOProviderSecret :exec
-- Set, replace or clear the secret. A NULL clears it (a public PKCE-only client); the
-- value is written and never read back through any interface query.
UPDATE sso_provider SET client_secret = $2, updated_at = now() WHERE id = $1;

-- name: DeleteSSOProvider :exec
DELETE FROM sso_provider WHERE id = $1;
