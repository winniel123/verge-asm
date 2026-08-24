---
title: SSO with Keycloak
section: Access
order: 7
description: Configure Keycloak as an OpenID Connect provider for verge-asm — pick a realm, register a client and its two callback URLs, and let users link their verified identity to a local account.
---

# Single sign-on with Keycloak

This is the Keycloak-specific companion to [sso.md](sso.md), which covers the
verge-asm side of single sign-on in full. Read that first: it explains what SSO is
here (**OpenID Connect only**, the authorization-code flow with PKCE), where the
config lives, and the two rules that never bend — **SSO authenticates an existing
account, it never creates one**, and **a binding is keyed on the verified
`(issuer, sub)`, never a username or email**. Nothing on the Keycloak side changes
those; this page only maps verge's five fields and two callback URLs onto Keycloak's
admin console.

verge-asm ships **one generic OIDC connector**, not a Keycloak plug-in. Keycloak is
admitted the same way any OIDC provider is: by its **issuer URL**, from which the app
discovers every endpoint and signing key at
`<issuer>/.well-known/openid-configuration`. Keycloak simply happens to be a standards
-clean OIDC provider, so the generic path fits it exactly.

> **Note:** These steps follow the current Keycloak admin console (the account-console
> UI shipped from Keycloak 19 onward). Older releases arrange the same settings under
> different tab names, but the fields — client ID, client authentication, valid
> redirect URIs, credentials — carry the same meaning throughout.

---

## What you will hand verge-asm

verge's **Add an OpenID Connect provider** form (Settings → Single sign-on) takes
five fields. Every one of them comes out of Keycloak below — there is no sixth.

| verge field | Where it comes from in Keycloak |
| --- | --- |
| **Display name** | Your label for the sign-in button (e.g. `Keycloak`). Free text; not read by Keycloak. |
| **Slug** | Your short URL-safe id (lowercase letters, digits, internal hyphens). It rides verge's flow routes and **shapes the two callback URLs** you register in Keycloak — decide it before you configure the client. |
| **Issuer URL** | `https://<keycloak-host>/realms/<realm>` (see [The issuer URL](#the-issuer-url)). Must be `https`. |
| **Client ID** | The **Client ID** you set when creating the client. |
| **Client secret** | *Confidential client:* the secret from the client's **Credentials** tab. *Public (PKCE-only) client:* leave blank. |

Scopes are **fixed** on verge's side at `openid profile email` — you do not, and
cannot, choose them in verge. Keep the matching Keycloak client scopes assigned (see
[Client scopes](#client-scopes-email-and-profile)).

---

## 1. Choose or create a realm

In Keycloak, **the realm is the issuer**. Every realm publishes its own OIDC
discovery document, its own signing keys, and its own set of users. Pick the realm
whose users should sign in to verge-asm, or create one (top-left realm switcher →
**Create realm**).

Whatever realm you choose, its name becomes part of the issuer URL you hand verge —
so settle the realm before you go further. Signing in through the wrong realm means
verge is verifying tokens against the wrong issuer and will refuse every login.

---

## 2. Create the OpenID Connect client

From the chosen realm: **Clients → Create client**.

1. **Client type** — choose **OpenID Connect**. (SAML is not supported; verge is
   OIDC-only.)
2. **Client ID** — set a stable identifier, e.g. `verge-asm`. This is the exact
   string you will paste into verge's **Client ID** field. It is **not** a secret.
3. **Name** — optional human label inside Keycloak; it has no bearing on verge.

Continue to the **Capability config** step.

### Confidential vs public — pick one

This single toggle decides whether verge stores a client secret.

| Keycloak setting | verge client type | verge **Client secret** field |
| --- | --- | --- |
| **Client authentication** = **On** | Confidential | Set it (copied from Credentials) |
| **Client authentication** = **Off** | Public, PKCE-only | Leave it blank |

- Leave **Standard flow** enabled — that is the authorization-code flow verge uses.
  You can safely turn off *Direct access grants*, *Implicit flow* and *Service
  accounts*; verge uses none of them.
- **PKCE is used on every login regardless of this choice.** verge always sends an
  S256 PKCE challenge; the client-authentication toggle only decides whether a secret
  is *also* required. For a public client you may additionally pin
  **Advanced → Proof Key for Code Exchange Code Challenge Method = `S256`** so
  Keycloak enforces it, but verge sends S256 either way.

> **Note:** A confidential client is the stronger default when verge can keep a
> secret (it can — see [sso.md → Where the client secret is stored](sso.md#where-the-client-secret-is-stored)).
> Reach for a public PKCE-only client only when you have a deliberate reason not to
> provision a secret; PKCE alone still binds the code to this login.

---

## 3. Register verge's two callback URLs

verge uses **two** redirect URIs per provider — one for sign-in, one for the Profile
self-link — and **both** must be registered on the Keycloak client or Keycloak
refuses the round-trip. In the client's **Login settings** (the **Access settings**
section on newer consoles), add both under **Valid redirect URIs**:

| Purpose | Register (`<VERGE_EXTERNAL_URL>` + this path) |
| --- | --- |
| Sign-in callback | `/login/sso/<slug>/callback` |
| Profile self-link callback | `/profile/sso/<slug>/link/callback` |

Both paths are rooted at **`VERGE_EXTERNAL_URL`** — the trusted external origin verge
is reached at, set on the `web` service. verge builds the `redirect_uri` it sends
Keycloak from this value, *not* from the incoming `Host` header, so set it before you
register anything and make the registered URIs match it exactly. See
[sso.md → `<base>` comes from `VERGE_EXTERNAL_URL`](sso.md#base-comes-from-verge_external_url).

For a provider slugged `keycloak` at `https://verge.example.com`, register:

```
https://verge.example.com/login/sso/keycloak/callback
https://verge.example.com/profile/sso/keycloak/link/callback
```

> **Note:** **Register the exact URIs — do not use a wildcard.** Keycloak accepts a
> trailing `*` in a redirect pattern, but a broad wildcard weakens the one check that
> stops an authorization code being returned to an attacker-chosen URL. Add the two
> literal paths above and nothing wider. verge only ever sends these two exact
> `redirect_uri` values, so a wildcard buys you nothing and costs you a guarantee.

Leave **Valid post logout redirect URIs** and **Web origins** at their defaults
unless you have a separate reason to set them; verge's flow does not require them.
**Save** the client.

---

## 4. Copy the client id and (confidential only) the secret

- **Client ID** — the identifier you set in step 2 goes straight into verge's
  **Client ID** field.
- **Client secret** — *confidential clients only.* Open the client's **Credentials**
  tab, copy the **Client secret**, and paste it into verge's **Client secret** field.
  For a public (PKCE-only) client there is no Credentials tab and no secret; leave
  verge's field blank.

verge stores the secret **write-only**: no screen ever reads it back, and the form
shows only whether a secret *is set*. If you rotate the secret in Keycloak
(Credentials → **Regenerate**), update it in verge through **Update secret** on the
provider's row — see [sso.md → Edit, disable, re-key, remove](sso.md#edit-disable-re-key-remove).

---

## The issuer URL

verge's **Issuer URL** is the realm's issuer, and it depends on your Keycloak version:

| Keycloak version | Issuer URL |
| --- | --- |
| **17 and newer** | `https://<keycloak-host>/realms/<realm>` |
| **16 and older** | `https://<keycloak-host>/auth/realms/<realm>` |

The `/auth` base path was dropped by default in Keycloak 17 (the Quarkus
distribution); an upgraded install may still serve it if the operator set it back.
**Do not guess — verify against the published discovery document.** Fetch:

```
<issuer>/.well-known/openid-configuration
```

A `200` whose `"issuer"` value equals exactly what you are about to paste into verge
confirms the URL. verge itself discovers the same document, so if that URL does not
resolve to a matching issuer, no login will complete. The Issuer URL **must be
`https`** — verge refuses a plaintext issuer at validation time, since discovery over
`http` is unprotected.

> **Note:** The `"issuer"` field in the discovery document is authoritative and must
> match byte-for-byte. A trailing slash, an `http` scheme, or an internal hostname
> that differs from the browser-facing one will all fail token verification even
> though the endpoints resolve. Register the exact external `https` issuer.

---

## Client scopes: email and profile

verge always requests `openid profile email` and reads the `sub` from the verified
`id_token`. Keycloak assigns the built-in **email** and **profile** client scopes to
new clients by default, which is all verge needs. Confirm under the client's **Client
scopes** tab that `email` and `profile` are present (as *Default*, not just
*Optional*). The `openid` scope is implicit to every OIDC request and needs no
configuration.

verge's account binding keys on the token **subject (`sub`)** — Keycloak's stable,
non-reassignable per-user identifier — never on the email or username claim. Those
human claims are captured for display only. This is what makes a renamed or recycled
Keycloak username unable to take over a verge account; see
[sso.md](sso.md) and ADR-0113.

---

## Putting it together — end to end

Because SSO authenticates an **existing** account and never creates one, the order of
operations matters:

1. **Create the local verge-asm account first.** SSO will refuse a verified identity
   that maps to no local account — it never provisions one. See
   [accounts.md](accounts.md).
2. **Declare the provider in verge** — Settings → Single sign-on → *Add an OpenID
   Connect provider*, filling the five fields from this guide. The provider is
   created enabled.
3. **The user links their identity themselves** — from **Profile → Linked
   identities**, they run the Keycloak round-trip inside their own session, which
   records `(issuer, sub) → their account`. An operator cannot link on a user's
   behalf; trust-on-first-use is deliberately not offered.
4. **Sign in with SSO.** From then on the Keycloak button on the sign-in screen
   authenticates that account. A local second factor is **not** bypassed: a
   TOTP-enrolled account still lands on the two-factor step, and password + TOTP login
   always remains available alongside SSO.

To offboard a user, an admin removes the binding under **Settings → SSO → Linked
identities**; to remove the account itself, see [accounts.md](accounts.md). Both are
covered in [sso.md](sso.md).
