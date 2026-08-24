---
title: SSO with Google
section: Access
order: 5
description: Wire Google Workspace to verge-asm as an OpenID Connect provider — build the OAuth client, register verge's two callbacks, and restrict sign-in to your organisation.
---

# Single sign-on with Google Workspace

Google Workspace is one OpenID Connect (OIDC) provider among many, and verge-asm talks to
it through the same generic connector it uses for every provider — there is no Google
plug-in, only the standard discovery-and-verify path described in [sso.md](sso.md). This
guide is the Google-specific companion to that page: it walks the Google Cloud console
side, then hands back to the verge-side form. Read [sso.md](sso.md) first for the rules
that govern all of this — SSO **authenticates an existing account and never creates one**,
and a binding is keyed on the verified `(issuer, sub)`, never a username or email.

Two facts shape the whole setup and are worth stating before you start:

- **Google issues no per-tenant issuer.** Every Google and Workspace account — yours,
  another org's, a personal `@gmail.com` — is issued by the single issuer
  `https://accounts.google.com`. You therefore cannot lean on the issuer to fence sign-in
  to your organisation. The domain restriction is enforced at the **OAuth consent screen**
  (and, optionally, the `hd` claim), *not* the issuer URL.
- **verge already refuses strangers by construction.** Because SSO only ever authenticates
  an *existing, linked* account, a verified identity from outside your org that maps to no
  local binding is refused, not provisioned. The consent-screen restriction below is
  defence in depth on top of that, keeping the round-trip itself inside your org.

The verge config lives in [`cmd/web/settings_sso.go`](../../cmd/web/settings_sso.go); the
flow in [`cmd/web/sso.go`](../../cmd/web/sso.go).

---

## What you will need

- A **Google Workspace** organisation and access to the **Google Cloud console**
  (<https://console.cloud.google.com>) for a project in that organisation, with rights to
  edit the OAuth consent/branding screen and create credentials.
- Your deployment's external origin, `VERGE_EXTERNAL_URL` (e.g.
  `https://verge.example.com`) — **set this before you register anything**, because the two
  redirect URIs are rooted at it and must match byte-for-byte on both sides. See
  [sso.md → `<base>` comes from `VERGE_EXTERNAL_URL`](sso.md#base-comes-from-verge_external_url).
- Admin access to verge-asm's **Settings → Single sign-on**, which is admin-only.
- A **slug** you will use for this provider. This guide uses `google` throughout; pick your
  own (lowercase letters, digits and internal hyphens). The slug rides the callback paths,
  so decide it now — changing it later means re-registering both redirect URIs.

---

## Step 1 — Configure the OAuth consent screen

Google calls this the **Branding** / **OAuth consent screen** (under **APIs & Services →
OAuth consent screen** in the Google Cloud console). Configure it before creating the
client — Google requires it first.

Choose the **User type**:

| User type | Who can sign in | Use it when |
| --- | --- | --- |
| **Internal** | Only accounts in **your** Workspace organisation. | This is the one you want. It confines sign-in to your org at Google's own gate, and needs no verification/publishing review. |
| **External** | Any Google account, subject to the app's publishing status and test-user list. | Only if your project cannot be Internal (e.g. no Workspace org on the project). You then carry the domain restriction yourself — see the note below. |

> **Note:** **Internal is the domain restriction.** An Internal consent screen is the clean
> way to keep the Google round-trip to your organisation: accounts outside your Workspace
> cannot complete it at all. If you are forced to use **External**, verge still refuses any
> identity that is not linked to a local account, but the OAuth flow itself will accept
> out-of-org Google accounts up to that point — a weaker posture. Prefer Internal.

Fill in the app name, support email and the required links. No sensitive or restricted
scopes are involved: verge requests only `openid`, `profile` and `email`, which are
non-sensitive, so an Internal app needs no Google verification.

---

## Step 2 — Create the OAuth client ID

In **APIs & Services → Credentials**, choose **Create credentials → OAuth client ID**, and
set:

| Field | Value |
| --- | --- |
| **Application type** | **Web application.** verge completes the server-side authorization-code flow with PKCE; it is a web app, not a desktop, mobile or "Web (client-side)" type. |
| **Name** | Anything that identifies this deployment to your admins, e.g. `verge-asm (prod)`. Internal only; it is not shown to signing-in users. |

Leave **Authorized JavaScript origins** empty — verge performs no browser-side token flow.
The **Authorized redirect URIs** are the part that matters, and they come next.

---

## Step 3 — Register verge's two redirect URIs

verge uses **two** callback paths per provider — one for sign-in, one for the Profile
self-link — and **both** must be registered as **Authorized redirect URIs** on the client,
or Google refuses the round-trip with `redirect_uri_mismatch`. Each is your
`VERGE_EXTERNAL_URL` followed by the path:

| Purpose | Redirect URI to register |
| --- | --- |
| Sign-in callback | `<VERGE_EXTERNAL_URL>/login/sso/<slug>/callback` |
| Profile self-link callback | `<VERGE_EXTERNAL_URL>/profile/sso/<slug>/link/callback` |

For a deployment at `https://verge.example.com` with the slug `google`, register both of:

- `https://verge.example.com/login/sso/google/callback`
- `https://verge.example.com/profile/sso/google/link/callback`

> **Note:** Google requires every redirect URI to use **`https`** (the sole exception is
> `localhost` for local testing) and forbids raw IP hosts. This aligns with verge deriving
> the `redirect_uri` from `VERGE_EXTERNAL_URL` — set that to your real `https` origin, and
> the values match on both sides. A trailing-slash or scheme mismatch is the usual cause of
> `redirect_uri_mismatch`; register exactly the two strings above.

---

## Step 4 — Copy the Client ID and Client secret

On saving the client, Google shows its **Client ID** and **Client secret**. A Web
application client always has a secret, so configure verge as a **confidential client**:
paste both.

- The **Client ID** is not a secret; verge stores and displays it plainly.
- The **Client secret** is write-only in verge — it is stored in the database
  (`sso_provider.client_secret`) and never read back to any screen. See
  [sso.md → Where the client secret is stored](sso.md#where-the-client-secret-is-stored).
  Because it lives in Postgres, a database dump carries it; treat `pgdata` accordingly.

> **Note:** verge also supports public, PKCE-only clients (no secret), and it applies S256
> PKCE on every login regardless. But a Google **Web application** client issues a secret,
> so use it — leave the secret blank in verge only if you have a specific reason to run the
> client public.

---

## Step 5 — Declare the provider in verge

Open **Settings → Single sign-on** (`/settings?tab=sso`) and use the **Add an OpenID
Connect provider** form. It takes exactly **five** fields:

| Field | Value for Google Workspace |
| --- | --- |
| **Display name** | The label on the sign-in button, e.g. `Google` or `Company Google`. |
| **Slug** | The slug you chose in Step 1 (this guide: `google`). It must match the slug in the redirect URIs you registered. |
| **Issuer URL** | `https://accounts.google.com` — **not** a per-tenant URL. Must be `https`. |
| **Client ID** | The Client ID from Step 4. |
| **Client secret** | The Client secret from Step 4. (Optional in general; supply it here.) |

verge validates the issuer is an `https` URL and then discovers Google's endpoints and
signing keys from `<issuer>/.well-known/openid-configuration` — for Google, that is
`https://accounts.google.com/.well-known/openid-configuration`. You can open that document
in a browser to confirm the issuer before saving; its `issuer` field must read
`https://accounts.google.com` exactly, which is the value verge verifies in each
`id_token`. The scopes are **fixed** at `openid profile email` and are not configurable.

A new provider is created **enabled** and immediately renders a button on the sign-in
screen. A duplicate slug is reported plainly.

---

## Step 6 — What verge verifies about the Google identity

On each sign-in verge exchanges the code and verifies Google's `id_token` — signature,
issuer, audience and a per-login nonce — before trusting anything in it. Two claims matter:

- **`sub`** is the binding key. Google documents it as "unique among all Google Accounts
  and never reused" — exactly the stable, non-reassignable subject
  [ADR-0113](../adr/0113-sso-binds-a-verified-issuer-sub-not-a-mutable-username.md) requires.
  verge binds `(provider, sub)`; it never authenticates on `email`.
- **`email`** and **`email_verified`** ride along for the display label only. Google warns
  the email "may not be unique to this account and could change over time", which is
  precisely why verge does not key on it. Treat `email_verified` as a data-quality signal,
  not the auth decision — the auth decision is the verified `sub` matching an existing
  binding.

Because Google's issuer is shared across all orgs, verge cannot tell one Workspace from
another by the issuer. Org confinement therefore rests on the **Internal consent screen**
from Step 1 (and, if you use External, on the fact that verge admits only linked accounts).
Google also exposes an `hd` (hosted-domain) hint, but verge does not send an `hd`
authorization parameter and does not gate on the `hd` claim; do not rely on the request-side
`hd` hint for security in any case — Google itself notes it can be altered client-side, so
only the signed claim would ever be trustworthy.

---

## Step 7 — End-to-end: create, declare, link, sign in

SSO authenticates an *existing* account, so the order is fixed:

1. **Create the local account first.** In verge, create the account the person will sign in
   as (see [accounts.md](accounts.md)). SSO will never mint it for them.
2. **Declare the Google provider** (Steps 1–5 above), if you have not already. It appears
   as a button on the sign-in screen.
3. **The user links their identity from their Profile.** Signed in with their password, the
   user goes to **Profile → Linked identities** and links Google. That runs the Google
   round-trip inside their own session and records `(provider, sub) → their account`. This
   is deliberate: linking is never trust-on-first-use, so no first claimant can seize an
   account. An account holds at most one identity per provider.
4. **They sign in with Google thereafter.** The Google button now completes to a session.
   If the account has TOTP enrolled, SSO still lands on the two-factor step — SSO proves the
   Google identity but never downgrades a local second factor, and the session's role is
   always read from the local account row. Password + TOTP login remains available
   alongside Google at all times, so disabling or removing the provider never strands
   anyone. See [sso.md → Signing in with SSO](sso.md#signing-in-with-sso).

To offboard someone, an admin removes their identity binding under **Settings → SSO →
Linked identities**, which stops the Google identity authenticating as that account without
deleting the account itself. See
[sso.md → Admin removal of a binding](sso.md#admin-removal-of-a-binding--offboarding).

---

## Troubleshooting

| Symptom | Likely cause |
| --- | --- |
| `redirect_uri_mismatch` from Google | The redirect URI verge sent is not registered on the client. Confirm `VERGE_EXTERNAL_URL` and that **both** Step 3 URIs are registered exactly (scheme, host, no stray trailing slash). |
| Sign-in ends at *"That identity is not linked to an account here."* | Expected for an unlinked identity. Create the local account, sign in with a password, then link Google from Profile (Step 7). |
| Out-of-org Google accounts can start the flow | Your consent screen is **External**. Switch to **Internal** to confine the round-trip to your Workspace (Step 1). |
| *"The issuer must be an https URL"* on save | The Issuer URL is wrong. It is `https://accounts.google.com` — a plain, non-tenant `https` value. |

See [sso.md → Route reference](sso.md#route-reference) for the full route table shared by
every provider.
