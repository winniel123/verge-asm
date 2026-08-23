---
title: Authentication
section: Operating
order: 8
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

Mint personal API tokens on **Profile → Personal API tokens**. A token is intended
to let **your own automation** act with **your access** — it is scoped to your
account and inherits your role, so a viewer's token reads and an admin's token
carries admin. There is no per-token scope narrowing; a token is as broad as the
account that owns it.

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
| Lifetime | Expires after 12 hours | No expiry — lives until revoked |
| Second factor | Enforced at login | Not applicable — the token *is* the credential |
| Storage | Nothing server-side (stateless, signed) | SHA-256 hash + prefix in the database |
| Revocation | End the session (see below) | Revoke by name |

> **Divergence worth knowing.** As of this build the personal-token store exists —
> you can mint, list and revoke tokens — but **no request-authentication path
> consumes one yet**. The web listener authenticates callers by session cookie
> only; there is no `Authorization`-header lookup that resolves a `vg_pat_…` token
> to an account, and a token's *Last used* therefore always reads as an em dash.
> Treat token management as forward-provisioning for a machine API that is not yet
> wired to accept them, not as a live second way to call the deployment today.

---

## Sessions

This build's sessions are **stateless signed cookies** — an HMAC over the account
id and an expiry, verified against the signing key on the `web-state` volume, with
**no server-side registry**. That shapes what "manage sessions" can honestly mean.

- **Viewing.** Profile shows only the session making the request — its browser and
  OS derived from your User-Agent, and its source IP from the connection (never a
  forwarding header). There is no roster of your other logins, because none is
  recorded.
- **Revoking** (`POST /profile/session/revoke`) — the one session honestly
  revocable is the current one. Revoking it clears the cookie and returns you to
  `/login`, exactly as signing out does. Because there is no registry, a session on
  another device cannot be revoked from here; it lapses when its 12-hour lifetime
  expires. Rotating the session signing key (losing the `web-state` volume)
  invalidates **every** session at once — see [running.md → Volumes](running.md#volumes).

---

## Password

### Changing your password

On **Profile → Credentials** (`POST /profile/password`) enter your current
password and a new one. The current password is re-verified against a fresh read,
the new one is bounded to **8–72 characters** (bcrypt hashes no more than 72
bytes), and your **second factor is left untouched** — a password change does not
strip TOTP. Other sessions are **not** invalidated; the success notice says so
plainly rather than implying a global sign-out.

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
   on submit. Set the new password (same 8–72 bound, typed twice), and the token is
   spent so the link is single-use.

Like a password change, a reset does not sign your other sessions out — a stateless
signed cookie has no registry to revoke against, and the done copy says so rather
than implying a global sign-out.

---

## Where to go next

- Admin acts on other accounts — invite, change role, **require re-enrollment**,
  remove: [accounts.md](accounts.md).
- Single sign-on and linking an external identity to your account:
  [sso.md](sso.md).
- Where the session signing key and other secrets live, and the security posture of
  the deployment: [running.md](running.md).
- The first-run setup token and the initial admin: [using.md](using.md),
  [first-run.md](first-run.md).
</content>
</invoke>
