---
title: Single sign-on (SSO)
section: Operating
order: 9
description: Configure OpenID Connect single sign-on — declare a provider, register the callback URL, and let users link their verified identity to a local account.
---

# Single sign-on (SSO)

verge-asm can let people sign in through your own identity provider instead of a
per-account password. Single sign-on here is **OpenID Connect (OIDC), and only OIDC**:
the app completes the authorization-code flow and verifies the provider's signed
`id_token` against its published keys. Reverse-proxy header trust is **not** an option —
a misconfigured trusting proxy is a bypass class the signature check does not have, so it
stays refused ([ADR-0112](../adr/0112-single-sign-on-is-admitted-as-verified-oidc-never-header-trust.md)).

Two rules shape everything below, and both are deliberate:

- **SSO authenticates an existing account; it never creates one.** A verified identity
  that maps to no local account is refused, not provisioned — so turning on a broad IdP
  can never silently mint accounts. Create the account first (see
  [accounts.md](accounts.md)); the user links their identity to it afterward.
- **A binding is keyed on the verified `(provider, sub)`, never a username.** The `sub`
  is the provider's stable, non-reassignable subject id. Matching on it — rather than on a
  mutable, recyclable username or email claim — closes the account-takeover surface a
  reassigned username would open ([ADR-0113](../adr/0113-sso-binds-a-verified-issuer-sub-not-a-mutable-username.md)).

The config lives in [`cmd/web/settings_sso.go`](../../cmd/web/settings_sso.go); the flow
in [`cmd/web/sso.go`](../../cmd/web/sso.go).

---

## Which providers ship

**None ship pre-configured, and none are hard-coded.** verge-asm ships one *generic* OIDC
connector, not an Okta plug-in or a Google plug-in. Any provider that speaks standard OIDC
discovery — Okta, Google Workspace, Microsoft Entra ID, Keycloak, Auth0, and the rest —
works through the same path: you declare it by its **issuer URL**, and the app discovers
its endpoints and signing keys from `<issuer>/.well-known/openid-configuration`. "Okta"
and "google" appear only as example labels and slugs in the forms.

Both **confidential clients** (client id *and* secret) and **public, PKCE-only clients**
(client id, no secret) are supported. PKCE (S256) is used on every login regardless; the
secret is what distinguishes the two client types.

### Setting up a specific provider

The mechanics below are the same for every provider; the fiddly part is finding the right
values in each IdP's own console. These worked examples map a provider's admin console to
the five fields and two callback URLs this guide describes:

| Provider | Guide |
| --- | --- |
| Microsoft Entra ID (formerly Azure AD) | [sso-entra-id.md](sso-entra-id.md) |
| Google Workspace | [sso-google.md](sso-google.md) |
| Okta | [sso-okta.md](sso-okta.md) |
| Keycloak | [sso-keycloak.md](sso-keycloak.md) |

Any other standards-compliant OIDC provider works through the same generic path — declare
it by its issuer URL and register the two callback URLs.

---

## Configure a provider — Settings → SSO

Everything here is **admin-only** — each route is admin-gated, and a non-admin `POST` is
refused. Open **Settings → Single sign-on** (`/settings?tab=sso`).

### Add a provider

The **Add an OpenID Connect provider** form (`POST /settings/sso`) takes five fields:

| Field | Notes |
| --- | --- |
| **Display name** | The label rendered on the sign-in button and the Settings row (e.g. `Okta`). |
| **Slug** | A short URL-safe id that rides the flow routes (`/login/sso/<slug>`). Lowercase letters, digits and internal hyphens only; unique per install. |
| **Issuer URL** | The OIDC issuer. **Must be `https`** — discovery over plaintext is refused at validation time. |
| **Client ID** | The OAuth2 client id the IdP assigned this deployment. Not a secret. |
| **Client secret** | *Optional.* Set it for a confidential client; leave it blank for a public PKCE-only client. |

A new provider is created **enabled**. A duplicate slug is reported plainly rather than as
a raw error.

### Edit, disable, re-key, remove

Each row carries an **Edit** disclosure with three independent forms:

- **Save provider** (`POST /settings/sso/update`) edits the display name, slug, issuer,
  client id, and the **enabled** checkbox. Disabling a provider keeps its config but
  renders no sign-in button and **refuses its flow** — the clean way to turn SSO off
  without deleting it, and without stranding password login.
- **Update secret** (`POST /settings/sso/secret`) is the secret's own write path. Leaving
  the field blank **keeps** the stored secret; typing a value **replaces** it; ticking
  **clear the secret** removes it (making the client public/PKCE-only). The clear box wins
  over any typed value. This mirrors the notification-channel secret exactly.
- **Remove provider** (`POST /settings/sso/delete`) deletes the provider. Its identity
  bindings cascade away with it.

### Where the client secret is stored

The client secret is held in the database, in the `sso_provider.client_secret` column, and
it is **write-only at the interface**. No list or detail read ever selects it — those
queries expose only whether a secret **is set** (a `set` / `none` badge), never the value.
Exactly one server-side read path hands the secret to the OIDC token exchange. So the
Settings UI can display a provider without ever being able to leak its secret, and a
placeholder in the form reads *"set — leave blank to keep."*

This is the same write-only treatment the notification-channel secret gets under
[ADR-0053](../adr/0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md).
Note the distinction from the session-signing and prober SSH keys, which live on
per-service state volumes and never touch Postgres (see
[running.md → Where secrets live](running.md#where-secrets-live)): the OIDC client secret
*does* live in the database, so a database dump carries it. Treat `pgdata` accordingly.

---

## The callback URL to register with your IdP

Every OIDC provider needs the exact **redirect URI(s)** registered on its side, or the IdP
refuses the round-trip. verge-asm uses **two** callback paths per provider — one for
sign-in, one for the Profile self-link — and both must be registered:

| Purpose | Path (register `<base>` + this) |
| --- | --- |
| Sign-in callback | `/login/sso/<slug>/callback` |
| Profile self-link callback | `/profile/sso/<slug>/link/callback` |

For a provider slugged `okta` at `https://verge.example.com`, register
`https://verge.example.com/login/sso/okta/callback` and
`https://verge.example.com/profile/sso/okta/link/callback`.

### `<base>` comes from `VERGE_EXTERNAL_URL`

The app builds the `redirect_uri` it sends the IdP from **`VERGE_EXTERNAL_URL`** — the
trusted origin the deployment is reached at (e.g. `https://verge.example.com`), set on the
`web` service. Deriving it from a fixed, configured origin — rather than the incoming
`Host` header — keeps an attacker-influenceable header out of the `redirect_uri`.

**Set `VERGE_EXTERNAL_URL` before you register anything.** When it is unset the app falls
back to the request's own host and scheme (`https` when TLS terminates in front, matching
the `VERGE_SECURE_COOKIES` signal), but the value it produces must still match what you
registered at the IdP — so pin it explicitly for any real deployment. Configure it the
same way as the other environment variables in
[running.md → Environment variables](running.md#environment-variables).

> **Note:** `VERGE_EXTERNAL_URL` (web, the SSO redirect origin) is a *different* variable
> from `VERGE_PUBLIC_URL` (worker, the base for links in notification bodies). SSO uses
> the former; setting only the latter does not configure the callback origin.

---

## Signing in with SSO

Each **enabled** provider renders its own button on the sign-in screen (`GET /login/sso/<slug>`).
The flow is two hops, both unauthenticated by construction:

1. `GET /login/sso/<slug>` mints a state / nonce / PKCE transaction into a short-lived
   signed cookie and redirects to the IdP.
2. `GET /login/sso/<slug>/callback` verifies the state, exchanges the code, verifies the
   `id_token` (signature, issuer, audience, and the per-login nonce), resolves the
   `(provider, sub)` binding, and issues the session.

If the verified identity is bound to no account, the login is refused with an honest
message — *"That identity is not linked to an account here. Sign in with your password,
then link it on your Profile."*

### SSO does not replace your second factor

SSO proves the IdP identity, but it **never downgrades a local second factor**. An account
that has enrolled TOTP still lands on the same two-factor step after the SSO round-trip and
owes its code, exactly as a password login would — only an account without TOTP completes
the login outright. The session's **role** is always read from the local account row, not
from any IdP claim. And because SSO authenticates *existing* accounts, **password + TOTP
login always remains available** alongside it; disabling or removing a provider never
strands anyone out of their account. See [authentication.md](authentication.md) for the
password and TOTP mechanics.

---

## Linking and unlinking an identity — Profile

A binding is established by the account holder themselves, from **Profile → Linked
identities** — never trust-on-first-use, which would re-open a first-claimant window.

- **Link** (`GET /profile/sso/<slug>/link` → `…/link/callback`) runs the same OIDC
  round-trip *inside the caller's session*, then records `(provider, sub) → this account`.
  Re-linking your own identity is a no-op; an identity already bound to a **different**
  account is refused cleanly. An account may hold **one identity per provider**.
- **Unlink** (`POST /profile/sso/unlink`) removes one of your own bindings. It is scoped to
  your account, so you can only ever unlink your own — a stale or foreign id simply no-ops.

---

## Admin removal of a binding — offboarding

Under **Settings → SSO → Linked identities**, an admin sees **every** verified identity
bound to any local account (provider, account, display label, and when it was linked).
**Remove** (`POST /settings/sso/identity/remove`) revokes a binding outright — the
offboarding / seat-reassignment case: a departed user's linked identity, or a recycled one,
must stop authenticating as that account. Removal is idempotent, and it does not delete the
local account — only the identity's ability to sign in as it. To take away the account
itself, see [accounts.md](accounts.md).

---

## Route reference

| Route | Who | What |
| --- | --- | --- |
| `GET /login/sso/{slug}` | anyone | Start SSO sign-in. |
| `GET /login/sso/{slug}/callback` | anyone | Complete sign-in; issue the session. |
| `GET /profile/sso/{slug}/link` | logged in | Start a self-link. |
| `GET /profile/sso/{slug}/link/callback` | logged in | Complete the self-link binding. |
| `POST /profile/sso/unlink` | logged in | Unlink one of your own identities. |
| `POST /settings/sso` | admin | Add a provider. |
| `POST /settings/sso/update` | admin | Edit a provider / toggle enabled. |
| `POST /settings/sso/secret` | admin | Set, replace or clear the client secret. |
| `POST /settings/sso/delete` | admin | Remove a provider. |
| `POST /settings/sso/identity/remove` | admin | Remove any identity binding. |
