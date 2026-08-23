-- +goose Up
-- An sso_provider is an operator's Declared OIDC identity provider (#293, ADR-0112).
-- Single sign-on is admitted as cryptographically-verified OIDC — the app verifies a
-- signed id_token against the provider's discovered keys — never reverse-proxy
-- header-trust, which stays refused (a misconfigured trusting proxy is a bypass class
-- OIDC's signature check does not have). A provider authenticates an EXISTING local
-- account matched by username_claim; it never creates one, so turning on a broad IdP
-- cannot silently mint accounts.
--
-- Tenancy follows the estate's per-record convention (seed, channel, report_schedule):
-- single-estate, so a provider is not account-scoped but is attributed to the admin who
-- declared it via created_by.
CREATE TABLE sso_provider (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- A short URL-safe id carried in the flow routes (/login/sso/<slug> and its
    -- callback). Unique so a route resolves one provider.
    slug           TEXT NOT NULL UNIQUE,
    -- The display label the SignIn button and the Settings row render (e.g. "Okta").
    name           TEXT NOT NULL,
    -- The OIDC issuer URL. The provider's endpoints and signing keys are discovered
    -- from its /.well-known/openid-configuration, so only the issuer is stored.
    issuer         TEXT NOT NULL,
    -- The OAuth2 client id the IdP assigned this deployment. Not a secret.
    client_id      TEXT NOT NULL,
    -- The client secret for the confidential-client token exchange. WRITE-ONLY,
    -- mirroring channel.secret (ADR-0053's shared-store precedent): the config reads
    -- never SELECT it — they expose only whether one is set — and a single dedicated
    -- server-side read hands it to the token exchange. NULL where the client is public
    -- (PKCE-only) and no secret was set.
    client_secret  TEXT,
    -- Which id_token claim carries the local username to match. Defaults to the OIDC
    -- standard preferred_username; accounts here are usernames, not emails.
    username_claim TEXT NOT NULL DEFAULT 'preferred_username',
    -- Whether the provider is offered on SignIn. A disabled provider keeps its config
    -- but renders no button and refuses its flow, so it can be turned off without
    -- deletion (and without stranding a password login).
    enabled        BOOLEAN NOT NULL DEFAULT TRUE,
    created_by     BIGINT NOT NULL REFERENCES account (id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- SignIn lists the enabled providers newest-first; a plain index over the ordering
-- key keeps that read cheap.
CREATE INDEX sso_provider_recent ON sso_provider (id DESC);

-- +goose Down
DROP TABLE sso_provider;
