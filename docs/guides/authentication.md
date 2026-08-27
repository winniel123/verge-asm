---
title: Authentication
section: Access
order: 2
description: Manage your own account security — two-factor enrollment and the login challenge, recovery and re-enrollment, API tokens, active sessions, and the password change and reset flows.
---

# Authentication

This guide covers the credential surface you manage for **your own account**:
enabling two-factor, the login-time TOTP challenge, recovering a lost
authenticator, minting and revoking API tokens, ending a session, and changing or
resetting your password. Admin acts on *other* accounts — inviting a member,
changing a role, forcing a re-enrollment — live in [accounts.md](accounts.md);
single sign-on lives in [sso.md](sso.md).

Identity comes only from the signed session cookie, never from a proxy-supplied
header — there is no reverse-proxy forward-auth path to trust. The primitives are
in [`internal/auth/`](../../internal/auth/) (`totp.go`, `password.go`, `key.go`)
and the handlers in [`cmd/web/auth.go`](../../cmd/web/auth.go).

---

## Two-factor (TOTP)

Two-factor is **per-account and self-service** — you enrol your own, from your
Profile. It is standard TOTP (RFC 6238): SHA-1, 6 digits, a 30-second step, the
values every authenticator app defaults to. These are fixed, not dials — changing
them would silently invalidate every enrolled device.

### Enrolling

1. On **Profile → Credentials**, start enrollment (`POST /account/totp/enable`).
   A fresh 160-bit secret is generated and rendered as a **QR code** to scan and
   as **base32 text** for manual entry. The QR is drawn in-process — the secret
   never leaves the origin. At rest the secret is **encrypted** with a file-backed
   key before it touches Postgres; a database dump discloses no usable secret.
2. Scan the QR (or type the secret) into your authenticator app.
3. Enter the current 6-digit code to confirm (`POST /account/totp/confirm`). A
   wrong code re-renders the enrollment screen and **does not** enable two-factor.
4. On success, two-factor is on and **eight recovery codes** are shown **once**.
   Store them now — see [Recovery codes](#recovery-codes-and-a-lost-authenticator)
   below.

An account that is **already enrolled cannot re-roll its secret** through this
path — doing so would strip the second factor until a fresh confirm, a downgrade a
stolen session must not be able to perform. To rotate an authenticator you have
lost, use recovery or an admin re-enrollment.

### The login challenge

When a TOTP-enabled account signs in, a correct password is only the first step.
The server sets a short-lived **pending** grant (5 minutes) and presents the
two-factor screen (`POST /login/totp`). Enter the current authenticator code — or
a recovery code, which the same field accepts as a fallback.

Two protections sit on this step:

- **Replay guard.** A valid code is single-use within its own ~90-second validity
  window. The login path records the counter step it last accepted and refuses any
  code whose step is not strictly greater, so a captured-and-replayed code — even
  two concurrent logins carrying the *same* code — cannot both pass (RFC 6238 §5.2).
- **Lockout.** Both the password step and the TOTP step are throttled per-account
  **and** per source IP. Too many failures locks the key for a few minutes and
  invalidates the pending grant, so a 6-digit code is not brute-forceable and the
  attacker must start again from the password. The source IP is read from the
  connection, never from a forwarding header.

### Recovery codes and a lost authenticator

The eight recovery codes issued at enrollment are the self-service fallback when
the authenticator is gone. Each is high-entropy, single-use, and stored only as a
salted hash — a leaked database is not offline-crackable and the plaintext is never
persisted. At the two-factor screen, type a recovery code into the same field you
would type an authenticator code; a 6-digit code never matches a recovery hash and
vice versa, so the two never collide. Redeeming one spends it — it never redeems
twice.

Re-issuing recovery codes clears the prior set: any time TOTP is confirmed, the old
codes stop working and only the freshly shown set redeems.

If you have lost **both** the authenticator and your recovery codes, an admin must
**require re-enrollment** on your account — this clears your second factor, your
current authenticator stops working at once, and your next sign-in walks you
through TOTP setup again. It touches neither your password nor your session. See
[accounts.md → require re-enrollment](accounts.md).

---

## API tokens

Mint personal API tokens on **Profile → Personal API tokens**. A token lets **your
own automation** read this deployment with **your access** — it is scoped to your
account and carries its role, read live on every request. A token authenticates the
**read-only** [JSON API](api.md) at `/api/v1`, and nothing else: it can never change
the estate or its configuration, so an admin's token and a viewer's token can both
only read. There is no per-token scope narrowing; the surface a token reaches is the
read surface itself.

- **Create** (`POST /profile/tokens`) — give the token a name (unique within your
  account, ≤ 64 characters). The plaintext — a `vg_pat_…` string — is shown
  **once**, at creation. Verge keeps only its SHA-256 hash plus a non-secret prefix
  for display; a refresh re-loads the page without the value, so copy it then or
  mint a new one.
- **Revoke** (`POST /profile/tokens/revoke`) — reached through a confirm dialog
  that makes you **type the token's exact name**, guarding the page's most
  destructive act: a revoke is irreversible and silently breaks whatever automation
  held the token. Revocation is scoped to the owner, so no account can revoke
  another's token. Removing a member also revokes their tokens.

### How a token differs from a session

| | Session cookie | API token |
| --- | --- | --- |
| Form | Signed cookie, set at login | `vg_pat_…` bearer string, minted on demand |
| Reaches | The full HTML console (read and admin acts) | The read-only `/api/v1` JSON surface only |
| Capability | Everything your role allows, including mutation | **Read-only, always** — no write, no admin act |
| Lifetime | Expires after 12 hours | No expiry — lives until revoked |
| Second factor | Enforced at login | Not applicable — the token *is* the credential |
| Storage | Opaque token, stored as its SHA-256 hash in a session record | SHA-256 hash + prefix in the database |
| Revocation | End the session (see below) | Revoke by name |

The two credential paths share **no** authentication machinery: a token never mints
a cookie and is never accepted on the HTML surface, and the session cookie is never
accepted on `/api/v1` (see [ADR-0123](../adr/0123-a-token-api-is-read-only-opt-in-and-a-bearer-path-separate-from-sessions.md)).
So a stolen cookie cannot drive the API and a stolen token cannot drive the console.

> **The API is off by default.** The `/api/v1` surface a token reaches is **disabled
> until an admin enables it** under **Settings → API access** — so a freshly minted
> token is **inert** until then, and the Profile card says so. Once the surface is
> on, a token's **Last used** shows the coarsened time of its most recent API request
> (at most once per hour per token); a token that has never authenticated a request
> reads as **never**. The full endpoint reference — enabling the surface, the bearer
> header, the five read endpoints and their JSON shapes, and the 404-when-disabled /
> 405 / 401 semantics — is in **[api.md](api.md)**.

---

## Sessions

This build's sessions are **server-side records** (ADR-0117). The cookie carries an
opaque, random session token inside an HMAC-signed payload (account id, kind and
expiry); the server keeps only the token's **SHA-256 hash** as a row in the session
registry and checks it on every request. Storing only the hash preserves ADR-0053's
leak model — a dump of the registry hands out no usable token. Because a session is
a real record, it can be both **seen** and **revoked**.

### Active sessions

- **Personal** (`Profile`). The Sessions card lists every live session for your
  account — device and OS from the User-Agent, source IP from the connection (never
  a forwarding header), last-active time, and a **this device** badge on the current
  one. From it you can **revoke one** (`POST /profile/sessions/revoke` — that browser
  lands on `/login` on its next request), **end this session** (sign out here), or
  **sign out others** (revoke every session but this one).
- **Admin** (`Settings → Sessions`). An admin sees every account's live sessions
  across the deployment, grouped by account and newest activity first, and can
  **revoke any one** or **revoke all** for an account — the offboarding kill that
  signs a departing member out everywhere at once.
- **Credential change.** Changing your password (or completing a reset) **revokes
  your other sessions** — every other browser is signed out, leaving only the one
  that made the change.

A session also lapses on its own when its **12-hour** lifetime expires, and rotating
the session signing key (losing the `web-state` volume) invalidates **every** session
at once — see [running.md → Volumes](running.md#volumes).

---

## Password

### Changing your password

On **Profile → Credentials** (`POST /profile/password`) enter your current
password and a new one. The current password is re-verified against a fresh read,
the new one is bounded to **12–72 characters** (at least 12; bcrypt hashes no more
than 72 bytes), and your **second factor is left untouched** — a password change does not
strip TOTP. Your **other sessions are revoked** — every other browser is signed out
through the session registry, leaving only the one that made the change, and the
success notice says so (#408, ADR-0117).

### Forgot / reset

The reset flow is pre-auth — a caller who has lost their password has no session to
gate on — and **enumeration-safe**.

1. **Request** (`GET`/`POST /forgot`) — enter your account name. The page always
   renders the same "if that account exists, a link is on its way" state, whether
   or not the name exists, so it reveals nothing about which accounts do.
2. **Delivery.** A self-hosted host sends no mail, so the single-use reset link is
   delivered **out of band**. By default only the account name and the reset
   record's id are written to the web logs — **not** the link, which is a bearer
   credential that must not sit in a log (CWE-532). The operator resets on the host
   directly. Setting `VERGE_LOG_RESET_LINKS` to a non-empty value opts the plaintext
   `/reset?token=…` link into the logs, for an operator who has knowingly turned it
   on for their own logs.
3. **Reset** (`GET`/`POST /reset?token=…`) — a valid, unspent, unexpired token
   (links live **30 minutes**) renders a set-a-new-password form; a missing, spent
   or stale token renders an honest invalid state rather than a form that would fail
   on submit. Set the new password (same 12–72 bound, typed twice), and the token is
   spent so the link is single-use.

Like a password change, a reset **signs your other sessions out** — completing it
revokes every session for the account through the registry, and the done screen says
every session has been signed out (#408, ADR-0117).

---

## Where to go next

- Admin acts on other accounts — invite, change role, **require re-enrollment**,
  remove: [accounts.md](accounts.md).
- The read-only JSON API a token authenticates — enabling it and the endpoint
  reference: [api.md](api.md).
- Single sign-on and linking an external identity to your account:
  [sso.md](sso.md).
- Where the session signing key and other secrets live, and the security posture of
  the deployment: [running.md](running.md).
- The first-run setup token and the initial admin: [using.md](using.md),
  [first-run.md](first-run.md).
</content>
</invoke>
