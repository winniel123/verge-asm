-- +goose Up
-- An sso_identity binds a verified OIDC identity to a local account (#319, ADR-0113).
-- ADR-0112 matched a verified identity to an account by its configured username_claim
-- (defaulting to preferred_username); that claim is mutable and reassignable on common
-- IdPs, so a genuinely-signed id_token could take over any local account — including
-- admin — by carrying a true claim about a *reassignable* name. ADR-0113 supersedes only
-- that identity-mapping clause: authentication now keys on the stable, non-reassignable
-- per-issuer subject, held as the pair (provider, sub).
--
-- The binding is established ONLY by an already-authenticated user linking their own
-- identity from their Profile (never trust-on-first-use, which would re-open the same
-- first-claimant window). Login is match-only: no binding is an honest refusal, never a
-- provision and never a username fallback.
--
-- Forward-only, no backfill: #293 merged the same day on a single-estate, pre-release
-- build with no live SSO bindings, so the username_claim column is simply dropped.
CREATE TABLE sso_identity (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- The provider the identity arrived through. sub is opaque and per-issuer, so it is
    -- only ever interpreted relative to its provider. ON DELETE CASCADE so removing a
    -- provider leaves no orphan binding.
    provider_id  BIGINT NOT NULL REFERENCES sso_provider (id) ON DELETE CASCADE,
    -- The local account this identity authenticates as. ON DELETE CASCADE so deleting an
    -- account takes its bindings with it.
    account_id   BIGINT NOT NULL REFERENCES account (id) ON DELETE CASCADE,
    -- The verified OIDC subject (the id_token `sub`), unique and non-reassignable per
    -- issuer. This is the matching key — never a human-facing username.
    sub          TEXT NOT NULL,
    -- A human label (email / preferred_username captured at link time) shown in the
    -- Profile and admin lists. For DISPLAY ONLY — it never gates authentication.
    display_name TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- One external identity binds to AT MOST ONE account: (provider, sub) is unique. An
    -- account may still hold many identities, but at most ONE per provider — the model
    -- offers a single link per provider (ADR-0113), so (provider, account) is unique too.
    -- This enforces the invariant in the schema, not just the UI, closing the direct-link
    -- route as a way to bind a second subject for a provider already linked.
    UNIQUE (provider_id, sub),
    UNIQUE (provider_id, account_id)
);

-- The Profile page lists an account's own linked identities; index the lookup key.
CREATE INDEX sso_identity_account ON sso_identity (account_id);

-- username_claim is retired as an auth input (ADR-0113): nothing keys on a mutable claim
-- any longer, so the column and its config field go.
ALTER TABLE sso_provider DROP COLUMN username_claim;

-- +goose Down
ALTER TABLE sso_provider
    ADD COLUMN username_claim TEXT NOT NULL DEFAULT 'preferred_username';
DROP TABLE sso_identity;
