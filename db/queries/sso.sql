-- A render read takes presence, never the value; the token exchange is the exception (ADR-0112).

-- name: InsertSSOProvider :one
INSERT INTO sso_provider (slug, name, issuer, client_id, client_secret, enabled, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: ListSSOProviders :many
SELECT p.id, p.slug, p.name, p.issuer, p.client_id, p.enabled,
       (p.client_secret IS NOT NULL)::boolean AS has_secret,
       p.created_by, p.created_at, p.updated_at,
       a.username AS created_by_username
FROM sso_provider p
JOIN account a ON a.id = p.created_by
ORDER BY p.id DESC;

-- name: ListEnabledSSOProviders :many
SELECT id, slug, name
FROM sso_provider
WHERE enabled = TRUE
ORDER BY id DESC;

-- name: GetSSOProvider :one
SELECT id, slug, name, issuer, client_id, enabled,
       (client_secret IS NOT NULL)::boolean AS has_secret, created_by, created_at, updated_at
FROM sso_provider
WHERE id = $1;

-- name: GetSSOProviderForAuth :one
SELECT id, slug, name, issuer, client_id, client_secret
FROM sso_provider
WHERE slug = $1 AND enabled = TRUE;

-- name: UpdateSSOProvider :execrows
UPDATE sso_provider
SET slug = $2, name = $3, issuer = $4, client_id = $5, enabled = $6, updated_at = now()
WHERE id = $1;

-- name: SetSSOProviderSecret :one
-- The provider rides the set's own RETURNING, so an absent id sets nothing.
UPDATE sso_provider SET client_secret = $2, updated_at = now() WHERE id = $1
RETURNING slug;

-- name: DeleteSSOProvider :one
-- The provider rides the delete's own RETURNING, so an absent id withdraws nothing.
DELETE FROM sso_provider WHERE id = $1
RETURNING slug;

-- name: InsertSSOIdentity :exec
INSERT INTO sso_identity (provider_id, account_id, sub, display_name)
VALUES ($1, $2, $3, $4);

-- name: GetAccountBySSOIdentity :one
SELECT a.id, a.username, a.role, a.password_hash, a.totp_secret, a.totp_enabled, a.created_at, a.totp_last_step
FROM sso_identity i
JOIN account a ON a.id = i.account_id
WHERE i.provider_id = $1 AND i.sub = $2;

-- name: GetSSOIdentityBySub :one
SELECT id, account_id, display_name
FROM sso_identity
WHERE provider_id = $1 AND sub = $2;

-- name: ListSSOIdentitiesForAccount :many
SELECT i.id, i.provider_id, p.slug AS provider_slug, p.name AS provider_name,
       i.display_name, i.created_at
FROM sso_identity i
JOIN sso_provider p ON p.id = i.provider_id
WHERE i.account_id = $1
ORDER BY i.id DESC;

-- name: DeleteSSOIdentityForAccount :one
WITH unlinked AS (
    DELETE FROM sso_identity i WHERE i.id = $1 AND i.account_id = $2
    RETURNING i.provider_id
)
-- The provider rides the unlink's own RETURNING, so the binding is named while it exists.
SELECT p.slug
FROM unlinked u
JOIN sso_provider p ON p.id = u.provider_id;

-- name: ListSSOBindings :many
SELECT i.id, i.provider_id, p.slug AS provider_slug, p.name AS provider_name,
       i.account_id, a.username AS account_username, i.display_name, i.created_at
FROM sso_identity i
JOIN sso_provider p ON p.id = i.provider_id
JOIN account a ON a.id = i.account_id
ORDER BY i.id DESC;

-- name: DeleteSSOIdentity :one
WITH removed AS (
    DELETE FROM sso_identity i WHERE i.id = $1
    RETURNING i.provider_id, i.account_id
)
-- Both names ride the removal's own RETURNING, so neither is read after its row is gone.
SELECT p.slug, a.username
FROM removed rm
JOIN sso_provider p ON p.id = rm.provider_id
JOIN account a ON a.id = rm.account_id;
