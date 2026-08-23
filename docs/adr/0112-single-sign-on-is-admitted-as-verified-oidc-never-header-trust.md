# ADR-0112: single sign-on is admitted as cryptographically-verified OIDC, never header-trust

- **Status:** Accepted
- **Date:** 2026-08-23
- **Ticket:** [#293 Auth: SSO / IdP backend](https://github.com/winniel123/verge-asm/issues/293)
- **Origin:** #282 (SignIn) and #281 (Settings → access), V2 console migration (map #275)
- **Amends:** the v1 spec §4.3 "Auth & access" and §7 non-goal that refused SSO/OIDC — and only the OIDC clause of it.

## Context

The v1 spec refused single sign-on outright:

> Local accounts only — no SSO/OIDC/reverse-proxy forward-auth (a misconfigured
> trusting proxy is a whole bypass class, and it is refused rather than risked; see §7).

Read closely, that clause bundles **two different mechanisms** under one refusal:

1. **Reverse-proxy forward-auth** — the app trusts an upstream header (`X-Auth-User`)
   to name the caller. The named risk is exact and real: *the moment the proxy is
   misconfigured*, anyone who can reach the app directly, or set that header, is
   whoever they claim. There is no cryptographic check the app can make — trust is
   positional, and a position is easy to get wrong. This is the "whole bypass class."

2. **OIDC** — the app redirects to an identity provider and receives a **signed
   `id_token`** back. The app verifies the token's signature against the provider's
   published keys and checks a `nonce` it minted, so a forged or replayed assertion is
   rejected by construction. The trust is cryptographic, not positional.

The spec's stated reasoning — "a misconfigured trusting proxy is a whole bypass class"
— is an argument against (1), not against (2). It was conservatively applied to both.

The SignIn and Settings screens (#282, #281) meanwhile render an honest "SSO not
configured" state. #293 asks to make it real.

## Decision

Single sign-on is **admitted, as OIDC only.** The operator (this ticket) has decided
SSO is wanted; this ADR admits the mechanism that the spec's own security argument does
not actually exclude, and holds the line on the one it does.

- **OIDC authorization-code flow with PKCE**, using the vetted `github.com/coreos/go-oidc`
  and `golang.org/x/oauth2` libraries. No hand-rolled token crypto: the `id_token`
  signature and issuer are verified by the library against the provider's discovered
  JWKS, and a per-login `nonce` and `state` (carried in an HMAC-signed, short-lived
  cookie under the session key) defeat replay and CSRF.
- **Reverse-proxy / forward-auth header-trust remains refused.** The bypass class the
  spec named is untouched: this build never trusts an upstream identity header. §7 keeps
  that non-goal.
- **SSO authenticates existing local accounts; it does not create them.** A verified
  identity's configured username claim is matched to an existing `account` by username;
  no match is an honest refusal, not an auto-provision. Admins stay in control of who
  exists (the invite path is the only account-creation route), so turning on a broad
  IdP cannot silently mint accounts. Role is still read from the local account row every
  request (unchanged), so SSO changes *how* a session is proven, never *what* it may do.
- **The IdP client secret is stored write-only**, mirroring the channel secret
  (ADR-0053's shared-store precedent): the config reads and lists expose only whether a
  secret is set, never its value; a dedicated server-side read hands it to the token
  exchange alone.
- **Local password + TOTP login stays.** SSO is an additional authentication route on
  the same accounts, not a replacement; a provider can be disabled without stranding
  anyone.

## Consequences

- The v1 spec §4.3 and §7 are updated: SSO is no longer a blanket non-goal; OIDC is
  supported and forward-auth header-trust remains refused, with the distinction above
  recorded.
- The SignIn screen renders a button per enabled provider and initiates the flow; the
  Settings → single-sign-on tab configures providers (add / edit / enable / delete).
  The "not configured" empty state now appears only when no provider is configured.
- New dependencies enter the tree (`go-oidc`, `oauth2`, and their `go-jose`
  transitive) — vetted, widely-used libraries, the responsible alternative to
  hand-rolling `id_token` verification for a security-critical path.
- Auto-provisioning and identity→role mapping are deliberately **not** built here; they
  are a larger policy surface and can be added as explicit dials later without changing
  this decision.
