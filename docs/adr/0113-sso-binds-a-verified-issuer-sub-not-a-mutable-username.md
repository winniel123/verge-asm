# ADR-0113: SSO binds a verified `(issuer, sub)`, established by authenticated self-link, never a mutable username

- **Status:** Accepted
- **Date:** 2026-08-23
- **Ticket:** [#319 SSO: identity mapped by mutable preferred_username claim enables account takeover](https://github.com/winniel123/verge-asm/issues/319)
- **Supersedes in part:** [ADR-0112](./0112-single-sign-on-is-admitted-as-verified-oidc-never-header-trust.md) — only its identity-mapping clause ("a verified identity's configured username claim is matched to an existing `account` by username"). Everything else in ADR-0112 stands: OIDC-not-header-trust, write-only client secret, existing-accounts-only, role read from the local row every request.

## Context

ADR-0112 admitted OIDC and matched the verified identity to a local `account` by its
configured **username claim**, defaulting to `preferred_username`. The token crypto is
sound — signature, issuer, audience, nonce and PKCE are all verified — but the *mapping*
keys on a claim that is **mutable and reassignable** on common IdPs:

- Providers that let a user self-edit `preferred_username` let an attacker set it to a
  local username — including `admin` — and be admitted as that account with its role.
- Username **recycling** (a departed employee's name handed to a new hire) silently
  points the new identity at the old account.

The `id_token` is genuinely signed, so every cryptographic check passes; the takeover
rides in on a *true* claim about a *reassignable* name. ADR-0112 bounds the blast radius
— SSO authenticates existing accounts and never provisions — but does not close it: the
matching key itself is the flaw.

The stable OIDC identity is the **`sub` claim**, unique and non-reassignable *per issuer*
(so the key is the pair `(issuer, sub)`). But `sub` is an opaque provider-assigned
identifier; it cannot be matched against a human `account.username`. Bridging it to a
local account requires a **binding** — and the security of the whole scheme reduces to
how that binding is first established. A "first SSO login records the pair" bootstrap
(trust-on-first-use) was rejected: it re-opens exactly the trusted-`preferred_username`
window this decision closes — whoever logs in *first* claiming a username captures the
binding.

## Decision

**SSO authenticates against a stored binding of a verified `(issuer, sub)` to a local
account. The binding is established only by an already-authenticated user linking their
own identity — never by trusting a username claim.**

- **Identity key.** `(issuer, sub)`, both taken from the verified `id_token`, compared
  exactly. `sub` is opaque and per-issuer, so it is only ever interpreted relative to the
  configured provider it arrived through.

- **Binding store.** A new `sso_identity` table maps a `(provider, sub)` to one
  `account`. An account may hold **many** identities (one per provider); a
  `UNIQUE(provider_id, sub)` guarantees an external identity binds to **at most one**
  account. A `display_name` (from `email`/`preferred_username` at link time) is stored
  **for display only** and never gates authentication. Both foreign keys are
  `ON DELETE CASCADE`, so deleting a provider or an account leaves no orphan binding.

- **Bootstrap = authenticated self-link (not trust-on-first-use).** A user already signed
  in (password + TOTP) links a provider from their **Profile** page — the same per-user
  security surface that hosts TOTP enrollment. The link runs the OIDC round-trip inside
  their authenticated session and records `(provider, sub) → their account`. No username
  is ever trusted. Every account here reaches SSO through a password login it already has
  (accounts are created only by admin invite), so no one is stranded by requiring a link
  first.

- **Login is match-only.** The SSO login callback looks the account up by
  `(provider, sub)`. No binding is an honest refusal directing the user to sign in with a
  password and link — not a provision, and not a username fallback. The TOTP second factor
  after a hit is unchanged (ADR-0112).

- **`username_claim` retired as an auth input.** It no longer selects the matching key;
  the column and its config field are removed. Nothing user-facing keys on a mutable
  claim any longer.

- **Management & audit.** A user unlinks their own identity; an admin can remove any
  binding (the offboarding / seat-reassignment case this bug is about) from the admin
  SSO settings. Link, self-unlink and admin-remove are audited.

## Consequences

- **A one-time link step precedes SSO** for every user — an intended cost. It is the step
  that removes all username trust; TOFU would have been cheaper and is exactly what is
  rejected.
- **Schema migration is forward-only.** #293 merged the same day on a single-estate,
  pre-release build with no live SSO bindings: add `sso_identity`, drop
  `sso_provider.username_claim`, no backfill.
- **The token verification path is untouched.** `Exchange` is refactored to return the
  verified `{sub, display}` instead of a username string; login (match) and link (bind)
  share that one extraction, so no crypto path is duplicated.
- **Auto-provisioning and identity→role mapping remain out of scope** (ADR-0112) — a
  binding attaches a verified identity to an account that an admin already created; it
  never mints one or changes its role.
