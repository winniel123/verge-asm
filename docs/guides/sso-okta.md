---
title: Single sign-on with Okta
section: Operating
order: 9
description: Wire Okta to verge-asm's generic OIDC connector — register an app integration, point verge at the right issuer, and let an existing account link its verified Okta identity.
---

# Single sign-on with Okta

verge-asm ships **one generic OpenID Connect connector, not an Okta plug-in** — Okta is
just one IdP that speaks standard OIDC discovery, wired in exactly like any other (see
[sso.md](sso.md) for the general model, the security rationale, and the Settings and
Profile screens). This guide is the Okta-specific half: what to create in the Okta Admin
Console, which of Okta's two issuer flavours to point verge at, and how the pieces line up
end to end.

Two rules from [sso.md](sso.md) carry over unchanged, and both matter before you touch
Okta:

- **SSO authenticates an existing account; it never creates one.** Assigning a user to the
  Okta app does *not* provision them in verge-asm. Create the local account first (see
  [accounts.md](accounts.md)); the user links their Okta identity to it from their Profile.
- **A binding is keyed on the verified `(provider, sub)`, never a username or email.** For
  Okta the `sub` is the immutable per-user subject its `id_token` carries — not the Okta
  username, not the email, both of which are mutable and reassignable.

The five verge-side fields, the two callback URLs, the fixed `openid profile email` scopes,
and the write-only client secret are all covered in [sso.md](sso.md). Everything below maps
Okta onto them.

---

## What you will need

- Admin access to your Okta org (the **Admin Console**), and admin access to verge-asm's
  **Settings → Single sign-on**.
- verge's external origin fixed in **`VERGE_EXTERNAL_URL`** on the `web` service — the two
  redirect URIs you register at Okta are rooted at it, and Okta rejects any round-trip whose
  `redirect_uri` was not registered verbatim. Pin it before you register anything.
- A decision on **which Okta issuer** verge will trust — the org authorization server or a
  custom one. This is the one choice that most often trips a first configuration; see
  [Choose the issuer](#choose-the-issuer-the-one-that-trips-people) below.

Pick the **slug** now too (e.g. `okta`): it is URL-safe (lowercase letters, digits, internal
hyphens) and rides both callback paths, so you need it to register the redirect URIs.

---

## Step 1 — Create the app integration in Okta

In the Okta **Admin Console**:

1. Go to **Applications → Applications** and click **Create App Integration**.
2. For **Sign-in method**, choose **OIDC - OpenID Connect**.
3. For **Application type**, choose **Web Application**.

> **Note:** Choose **Web Application** even if you intend to run verge as a *public*
> PKCE-only client (no secret). verge always uses server-side redirects and completes the
> code exchange from its backend, so "Web Application" is the correct Okta application type;
> the confidential-vs-public distinction is made by whether you give verge a client secret,
> not by picking Okta's SPA type. verge sends PKCE (S256) on every login regardless.

Click **Next** to reach the settings screen.

---

## Step 2 — Register verge's two redirect URIs

verge uses **two** callback paths per provider — one for sign-in, one for the Profile
self-link — and **both** must be registered as **Sign-in redirect URIs** on the Okta app.
Each is `VERGE_EXTERNAL_URL` + the path:

| Purpose | Path (append to `VERGE_EXTERNAL_URL`) |
| --- | --- |
| Sign-in callback | `/login/sso/<slug>/callback` |
| Profile self-link callback | `/profile/sso/<slug>/link/callback` |

For a slug of `okta` at `https://verge.example.com`, add both:

- `https://verge.example.com/login/sso/okta/callback`
- `https://verge.example.com/profile/sso/okta/link/callback`

Okta lets you list multiple **Sign-in redirect URIs** — add them as two entries. A missing
or mistyped entry surfaces as a "redirect URI mismatch" error at sign-in, not at
configuration time, so copy them exactly (scheme, host, and path all count).

verge does not use OIDC front-channel logout, so a **Sign-out redirect URI** is not
required — leave it blank unless your org convention needs one.

Under **Grant type**, keep **Authorization Code** enabled (the default). verge uses the
authorization-code flow only; it never uses the implicit or client-credentials grants.

---

## Step 3 — Assign the app to who may sign in

Under **Assignments**, assign the app to the users or groups permitted to sign in through
Okta. This is Okta's gate on *who can reach the flow*, and it is worth being precise about
what it does — and does not — do in verge:

- Being **assigned** in Okta lets a user *complete the Okta round-trip*. It does **not**
  create a verge account, and it does **not** by itself let them in: verge still refuses any
  verified identity that is not already bound to a local account.
- Being **unassigned** (or deactivated) in Okta stops the user obtaining an `id_token` at
  all — the front line of offboarding. Removing the verge-side binding
  (**Settings → SSO → Linked identities**, see [sso.md](sso.md)) is the second line.

Keep the Okta assignment list and your verge account list in step: an assigned Okta user
with no verge account simply gets the honest *"not linked to an account here"* refusal.

---

## Step 4 — Copy the client credentials

On the app's **General** tab, Okta shows the OAuth 2.0 credentials:

- **Client ID** — copy this into verge's **Client ID** field. It is not a secret.
- **Client secret** — for a **confidential client**, copy this into verge's **Client
  secret** field. For a **public, PKCE-only client**, configure the Okta app with **no
  client secret** (Okta's *"Proof Key for Code Exchange (PKCE)"* / public-client option) and
  leave verge's **Client secret** blank.

verge's client secret is **write-only**: once saved it is never displayed again, only
re-keyed or cleared through its own form (see [sso.md](sso.md)). If Okta issues the secret,
treat the copy carefully — it lands in verge's database (`sso_provider.client_secret`), so a
database dump carries it.

---

## Choose the issuer — the one that trips people

This is the field to get exactly right. Okta can present **two different issuers**, and
verge's **Issuer URL must match, character for character, the `iss` claim that Okta actually
stamps into the `id_token`** — because verge verifies the token's issuer against the value
you configure. Point verge at the wrong one and every login fails token verification even
though everything else is correct.

| Okta issuer | Issuer URL to enter in verge | Discovery document verge fetches |
| --- | --- | --- |
| **Org authorization server** | `https://<org>.okta.com` | `https://<org>.okta.com/.well-known/openid-configuration` |
| **Custom authorization server** | `https://<org>.okta.com/oauth2/<authServerId>` | `https://<org>.okta.com/oauth2/<authServerId>/.well-known/openid-configuration` |

- The **org** authorization server issues tokens whose `iss` is the bare org URL,
  `https://<org>.okta.com`. Enter exactly that.
- A **custom** authorization server (including Okta's built-in one named **`default`**)
  issues tokens whose `iss` is `https://<org>.okta.com/oauth2/<authServerId>` — for the
  built-in one, `https://<org>.okta.com/oauth2/default`. Enter the **full** path.

> **Note:** The issuer path and the discovery-document path are *not* the same shape. verge
> derives discovery itself by appending `/.well-known/openid-configuration` to the issuer you
> give it, so you only ever enter the **issuer**, never the discovery URL. Do not trim
> `/oauth2/default` off a custom-server issuer to make it "look like" the org one — that
> mismatched value is exactly what breaks verification.

To confirm the value before you save it, open the discovery document in a browser and read
its `"issuer"` field: whatever string appears there is precisely what belongs in verge's
**Issuer URL**. Okta requires the issuer to be **`https`**, which is also verge's own
validation rule — a non-`https` issuer is refused at save time.

Which to use is an Okta-org policy question: the org authorization server is the simplest for
plain sign-in; a custom authorization server lets you scope policies and claims per app. Both
work identically with verge as long as the Issuer URL matches the emitted `iss`.

---

## Step 5 — Declare the provider in verge

In verge, open **Settings → Single sign-on** (`/settings?tab=sso`, admin-only) and use
**Add an OpenID Connect provider**. The mapping from Okta to verge's five fields:

| verge field | What to enter |
| --- | --- |
| **Display name** | The label on the sign-in button and Settings row, e.g. `Okta`. |
| **Slug** | The URL-safe id you chose (e.g. `okta`) — must match the slug in the redirect URIs. |
| **Issuer URL** | The issuer from [Choose the issuer](#choose-the-issuer-the-one-that-trips-people), matching Okta's `iss` exactly. |
| **Client ID** | The Okta app's **Client ID**. |
| **Client secret** | The Okta app's **Client secret** for a confidential client; **blank** for a public PKCE-only client. |

The **scopes are fixed** at `openid profile email` and are not a field — verge requests
exactly those on every login, so no scope configuration is needed on the verge side. On the
Okta side, ensure those three standard OIDC scopes are granted to the app (they are enabled
by default for a new OIDC integration; a custom authorization server must include the
`openid`, `profile`, and `email` scopes in its access policy). verge reads the user's display
label from the `email` / `preferred_username` / `name` claims for the Settings and Profile
tables only — never as an authentication input.

A new provider is created **enabled**, so its button appears on the sign-in screen straight
away. A duplicate slug is reported plainly rather than as a raw error.

---

## Step 6 — Put it together, end to end

The pieces only authenticate someone once all of this is true, in this order:

1. **Create the local account** in verge (see [accounts.md](accounts.md)). SSO never
   provisions it.
2. **Declare the Okta provider** in Settings → SSO (Step 5), and confirm the user is
   **assigned** to the Okta app (Step 3).
3. **The user links their Okta identity**, from **Profile → Linked identities**. This runs
   the Okta round-trip inside their signed-in session and records
   `(provider, sub) → their account`. An account holds **one identity per provider**, and an
   Okta identity already bound to a *different* verge account is refused cleanly.
4. **The user signs in with Okta**: the **Okta** button on the sign-in screen starts the
   flow; verge verifies the `id_token` (signature, issuer, audience, per-login nonce),
   resolves the `(provider, sub)` binding, and issues the session.

If a user reaches Okta successfully but is not yet linked, verge refuses honestly — *"That
identity is not linked to an account here. Sign in with your password, then link it on your
Profile."* — which is the cue to complete step 3, not a misconfiguration.

> **Note:** SSO does not replace a local second factor. An account that enrolled TOTP still
> completes the TOTP step after the Okta round-trip, and its **role** is always read from the
> local account row, never from an Okta claim. Password + TOTP login always remains available
> alongside Okta, so disabling or removing the provider never strands anyone — see
> [sso.md](sso.md).

---

## Troubleshooting

| Symptom | Likely cause |
| --- | --- |
| Login fails at token verification, everything else correct | verge's **Issuer URL** does not match Okta's emitted `iss` — most often `/oauth2/default` trimmed off, or the org issuer used where a custom server issues the token. Read the discovery doc's `"issuer"` and match it exactly. |
| Okta shows a "redirect URI" / callback mismatch | A **Sign-in redirect URI** is missing or mistyped at Okta. Both callbacks must be registered verbatim, rooted at `VERGE_EXTERNAL_URL`, with the correct slug. |
| `redirect_uri` verge sends is wrong host/scheme | `VERGE_EXTERNAL_URL` is unset or wrong on the `web` service; it must equal the origin whose callbacks you registered. |
| User reaches Okta but is refused as "not linked" | No `(provider, sub)` binding yet — the account exists but the user has not linked from Profile (step 3). |
| Code exchange fails for a confidential client | Client secret missing or stale in verge — re-key it via **Update secret**. For a public client, ensure the Okta app is configured PKCE-only and verge's secret is blank. |

See [sso.md](sso.md) for the full Settings and Profile reference, the route table, and the
edit / disable / re-key / remove and admin-offboarding flows.
