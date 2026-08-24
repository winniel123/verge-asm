---
title: SSO with Entra ID
section: Access
order: 6
description: A worked example of declaring a Microsoft Entra ID (formerly Azure AD) OpenID Connect provider — register the app, its two redirect URIs and a secret, and pin the tenant-scoped v2.0 issuer.
---

# Single sign-on with Microsoft Entra ID

This is a provider-specific companion to [sso.md](sso.md): a worked example of wiring
**Microsoft Entra ID** (formerly Azure Active Directory) to verge-asm's one generic OIDC
connector. Nothing here is Entra-specific on the verge side — the app discovers Entra's
endpoints and signing keys from `<issuer>/.well-known/openid-configuration`, exactly as it
would for Okta, Google Workspace or Keycloak. Read [sso.md](sso.md) first for the model
that governs everything below; this guide only fills in the Entra admin-centre half and the
one Entra footgun worth calling out — the **issuer URL**.

Two product rules from [sso.md](sso.md) carry over unchanged, and both matter here:

- **SSO authenticates an existing local account; it never creates one.** Provisioning a
  user in Entra does *not* provision them in verge-asm. Create the local account first
  (see [accounts.md](accounts.md)); the user links their Entra identity to it afterwards.
- **A binding is keyed on the verified `(issuer, sub)`, never a username or email.** For
  Entra the `sub` is the token's stable, per-issuer subject id — not the user's UPN,
  `email` or `preferred_username`, all of which are mutable and reassignable. verge-asm
  captures `email`/`preferred_username`/`name` for display only.

The Entra field labels below are named by their labels in the Microsoft Entra admin centre,
not by pixel position; Microsoft moves the furniture, but the labels are stable.

---

## What you need before you start

| You need | Why |
| --- | --- |
| A **Microsoft Entra admin centre** role that can create app registrations (Application Administrator, Cloud Application Administrator, or Global Administrator) | You are registering an application and minting a secret. |
| verge-asm's **`VERGE_EXTERNAL_URL`** already set and correct | The two redirect URIs you register at Entra are rooted at it, and it must match byte-for-byte (see [sso.md → `<base>` comes from `VERGE_EXTERNAL_URL`](sso.md#base-comes-from-verge_external_url)). |
| An **admin** account in verge-asm | Declaring the provider under Settings → SSO is admin-only. |
| A chosen **slug** for this provider (e.g. `entra`) | It rides the callback routes, so decide it now — both redirect URIs embed it. |

> **Note:** SSO's scopes are fixed at `openid profile email` and are not configurable in
> verge-asm. You do not need to add optional claims or extra Graph permissions for sign-in
> to work; the default OIDC scopes carry the `sub`, `email`, `preferred_username` and
> `name` claims the app reads.

---

## Step 1 — Register an application

In the **Microsoft Entra admin centre**, go to **Identity → Applications → App
registrations → New registration**.

| Field | What to enter |
| --- | --- |
| **Name** | A human label for the app registration (e.g. `verge-asm`). This is Entra's own label; it need not match verge's **Display name**. |
| **Supported account types** | Choose **single-tenant** ("Accounts in this organizational directory only") unless you have a deliberate reason to federate other tenants — see [Single-tenant vs multi-tenant](#single-tenant-vs-multi-tenant) below. |
| **Redirect URI** | Leave blank for now — you will add both URIs as a **Web** platform in Step 2. |

Select **Register**. You land on the app's **Overview**, which shows the
**Application (client) ID** and the **Directory (tenant) ID** — you will copy both shortly.

---

## Step 2 — Register verge's two redirect URIs (Web platform)

verge-asm uses **two** callback paths per provider — one for sign-in, one for the Profile
self-link — and **both** must be registered at Entra or it refuses the round-trip. Both are
rooted at your `VERGE_EXTERNAL_URL`, with your chosen `<slug>` embedded:

| Purpose | Redirect URI to register |
| --- | --- |
| Sign-in callback | `<VERGE_EXTERNAL_URL>/login/sso/<slug>/callback` |
| Profile self-link callback | `<VERGE_EXTERNAL_URL>/profile/sso/<slug>/link/callback` |

For a provider slugged `entra` reached at `https://verge.example.com`, register
`https://verge.example.com/login/sso/entra/callback` and
`https://verge.example.com/profile/sso/entra/link/callback`.

In the app registration, open **Authentication → Add a platform → Web**. Enter the first
URI, save, then use **Add URI** to add the second. Both must sit under the **Web** platform
— **not** "Single-page application" (SPA). verge-asm completes the code exchange
server-side; registering the URIs as SPA opts them into the browser-only PKCE rules and the
server-side exchange will be refused.

> **Note:** Entra matches `redirect_uri` **exactly** — scheme, host, path and trailing
> slash. `https://` is required (Entra rejects plaintext `http` redirect URIs for anything but
> `localhost`), and verge-asm sends no trailing slash on either path. If your
> `VERGE_EXTERNAL_URL` carries a path prefix or a non-default port, it must appear in the
> registered URI too. A mismatch surfaces at the IdP as `AADSTS50011` (redirect URI
> mismatch), never as a verge error.

You may leave the **Front-channel logout URL** and the implicit-grant **tokens** checkboxes
untouched — verge-asm uses neither the implicit flow nor OIDC front-channel logout.

---

## Step 3 — Copy the Client ID, and create a Client secret

verge-asm supports both a **confidential client** (client id *and* secret) and a **public,
PKCE-only client** (client id, no secret). PKCE (S256) is used on every login regardless;
the secret only distinguishes the two client types.

**Client ID.** On the app's **Overview**, copy **Application (client) ID**. This is
verge's **Client ID**. It is not a secret.

**Client secret (confidential client — recommended for Entra).** Go to **Certificates &
secrets → Client secrets → New client secret**. Give it a description and an expiry, then
**Add**. Copy the secret's **Value** immediately — Entra shows it once and never again;
after you leave the blade only its "Secret ID" remains visible. This **Value** is verge's
**Client secret**.

> **Note:** Entra client secrets **expire** (24 months maximum). When one expires, sign-in
> starts failing at the token exchange. Mint a new secret ahead of the expiry and paste it
> into verge via **Update secret** (see [sso.md → Edit, disable, re-key, remove](sso.md#edit-disable-re-key-remove));
> the field is write-only, so replacing it is the only way to change it. Diarise the expiry.

**Public PKCE-only client (optional alternative).** If you would rather not manage a
secret, skip creating one and, under **Authentication → Advanced settings**, set **Allow
public client flows** to **Yes**. Then leave verge's **Client secret** field blank — a blank
secret makes verge a public PKCE client. This trades secret-rotation toil for a client that
proves possession of the code only via PKCE. For a server-side deployment, prefer the
confidential client with a secret.

---

## Step 4 — Pin the tenant-scoped v2.0 issuer

This is the one field where Entra reliably trips people up. verge-asm discovers everything
from `<issuer>/.well-known/openid-configuration`, and go-oidc then verifies that the
`issuer` claim inside every `id_token` **equals the issuer you configured, exactly**. Entra
exposes more than one issuer shape, and only one of them matches.

Use the **tenant-scoped v2.0 issuer**, with your **Directory (tenant) ID** (the GUID from
the Overview) substituted in:

```
https://login.microsoftonline.com/<tenant-id>/v2.0
```

Its discovery document is at
`https://login.microsoftonline.com/<tenant-id>/v2.0/.well-known/openid-configuration`, and
that document's own `issuer` field is `https://login.microsoftonline.com/<tenant-id>/v2.0`
— so the tokens Entra mints carry a matching `iss`, and verification passes.

> **Note:** Do **not** use the **v1.0** (Azure AD) endpoint
> `https://login.microsoftonline.com/<tenant-id>` (no `/v2.0`). Its issuer is the
> `https://sts.windows.net/<tenant-id>/` form, which will **not** equal the v2.0 issuer you
> configured — verification fails with an issuer mismatch. Only the **v2.0** endpoint's
> discovery and token issuer line up. If sign-in fails with an issuer-mismatch error, this
> is almost always the cause: confirm the `/v2.0` suffix is present.

### Verify against the discovery URL before you save

Fetch the discovery document and confirm its `issuer` matches what you are about to paste:

```
curl -s https://login.microsoftonline.com/<tenant-id>/v2.0/.well-known/openid-configuration | grep -o '"issuer":"[^"]*"'
```

It should return `"issuer":"https://login.microsoftonline.com/<tenant-id>/v2.0"`. If the
returned issuer contains the literal placeholder `{tenantid}` instead of your GUID, you are
looking at a multi-tenant (`common`/`organizations`) endpoint — see below.

---

## Single-tenant vs multi-tenant

Your choice of **Supported account types** in Step 1 decides which issuer is even
available, so it is really an issuer decision:

| Registration | Issuer to configure | Notes |
| --- | --- | --- |
| **Single-tenant** (this directory only) | `https://login.microsoftonline.com/<tenant-id>/v2.0` | The straightforward, recommended case. The discovery issuer carries your real GUID and matches the tokens exactly. |
| **Multi-tenant** (`common` / `organizations`) | `https://login.microsoftonline.com/common/v2.0` *and its discovery say* `issuer` *is* `https://login.microsoftonline.com/{tenantid}/v2.0` | The `common` discovery document returns a **templated** issuer containing the literal `{tenantid}` placeholder, while each user's token carries their own tenant GUID. A strict OIDC verifier that expects one fixed issuer string will reject those tokens. |

For almost every deployment, register **single-tenant** and configure the tenant-scoped
issuer. Only reach for multi-tenant if you deliberately intend to admit users from tenants
you do not control — and note that verge-asm still authenticates only **existing** local
accounts keyed on `(issuer, sub)`, so a multi-tenant app does not, by itself, let anyone new
in. Given the `{tenantid}`-placeholder mismatch, a single-tenant registration per admitted
tenant is the cleaner path.

---

## Step 5 — Admin consent (if your tenant requires it)

The default sign-in scopes (`openid`, `profile`, `email`) are delegated permissions that
normally allow user consent, so most users can consent on their first sign-in. If your
tenant disables user consent, an administrator must consent once for the whole directory:
in the app registration, **API permissions → Grant admin consent for `<tenant>`**. After
that, individual sign-ins proceed without a per-user consent prompt.

No Microsoft Graph application permissions are needed for sign-in — verge-asm reads only the
standard OIDC claims from the `id_token`, and calls no Graph API.

---

## Step 6 — Declare the provider in verge-asm

With the Entra side registered, add the provider under **Settings → Single sign-on**
(`/settings?tab=sso`), admin-only. The **Add an OpenID Connect provider** form takes five
fields — map them from what you gathered above:

| verge field | Value from Entra |
| --- | --- |
| **Display name** | The label on the sign-in button (e.g. `Microsoft`). Free text. |
| **Slug** | The `<slug>` you chose in Step 2 (e.g. `entra`). Lowercase letters, digits and internal hyphens; it must match the slug embedded in the two redirect URIs you registered. |
| **Issuer URL** | `https://login.microsoftonline.com/<tenant-id>/v2.0` (Step 4). Must be `https`. |
| **Client ID** | The **Application (client) ID** (Step 3). |
| **Client secret** | The secret **Value** (Step 3) for a confidential client — or leave blank for a public PKCE client. Write-only once saved. |

Save. A new provider is created **enabled**, and its button appears on the sign-in screen.
A duplicate slug is reported plainly rather than as a raw error.

---

## Step 7 — End-to-end: account, then link, then sign in

Because SSO authenticates an *existing* account and never provisions one, the order matters:

1. **Create the local account** in verge-asm first (see [accounts.md](accounts.md)). Set
   its role there — the session role is always read from the local account row, never from
   any Entra claim.
2. **The user signs in with their password once**, then links their Entra identity from
   **Profile → Linked identities**. This runs the same OIDC round-trip inside their own
   session and records `(provider, sub) → their account`. Linking (never trust-on-first-use)
   is what binds the verified subject to the right account.
3. **From then on, the user signs in via the Microsoft button.** verge-asm verifies the
   `id_token` (signature, issuer, audience and per-login nonce), resolves the `(issuer, sub)`
   binding, and issues the session. An identity that is not yet linked is refused with an
   honest message directing the user to sign in with a password and link it first — it is
   never auto-provisioned.

> **Note:** SSO does not replace a second factor. An account with TOTP enrolled still lands
> on the two-factor step after the Entra round-trip, exactly as a password login would. And
> because SSO only ever authenticates existing accounts, password + TOTP login always
> remains available alongside it — disabling or removing the Entra provider never strands
> anyone out of their account. See [sso.md → Signing in with SSO](sso.md#signing-in-with-sso).

---

## Troubleshooting quick reference

| Symptom | Likely cause |
| --- | --- |
| `AADSTS50011` redirect URI mismatch at Microsoft's screen | A registered redirect URI does not match byte-for-byte. Re-check both URIs, the slug, scheme, port and trailing slash against your `VERGE_EXTERNAL_URL`. |
| Sign-in fails after the Microsoft round-trip with an issuer-mismatch error | The v1.0 endpoint (issuer `https://sts.windows.net/<tenant-id>/`) was configured instead of the `/v2.0` issuer. Fix the **Issuer URL** to end in `/v2.0`. |
| Token exchange started failing after months of working fine | The Entra **client secret expired**. Mint a new one and paste it via **Update secret**. |
| "That identity is not linked to an account here." | Expected for a first sign-in — the user must sign in with a password and link the identity from Profile first. Not an Entra misconfiguration. |
| Server-side exchange refused / consent loops on an SPA-registered URI | The redirect URIs were registered under the **Single-page application** platform. Move them to **Web**. |

For the verge-side route reference, secret storage, disabling and admin identity removal,
see [sso.md](sso.md).
