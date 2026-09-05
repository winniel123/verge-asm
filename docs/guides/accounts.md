---
title: Accounts, invites & roles
section: Access
order: 1
description: The two-role model, inviting operators and the invite lifecycle, changing a role, re-enrolling a lost second factor, and removing an account — all admin-gated, and how it meets the /setup admin and SSO.
---

# Accounts, invites & roles

Every act in verge-asm has an author. This guide covers who those authors are, how
you add and remove them, and the exact line between the two roles — **admin** and
**viewer** — that decides which acts each may perform.

Account management lives under **Settings → Team** (`/settings?tab=team`). The
handlers are in [`cmd/web/settings.go`](../../cmd/web/settings.go) (team surface) and
[`cmd/web/auth.go`](../../cmd/web/auth.go) (roles, invite acceptance, setup). The
route table is [`cmd/web/handlers.go`](../../cmd/web/handlers.go).

---

## The two roles

There are exactly two roles, `admin` and `viewer` — **there is no operator role**. The
role is a column on the account row, read live on every request. So a demotion or a
removal takes effect on the target's very next click, not on their next sign-in.

- **admin** performs every *declared act* — the writes that change what the estate is or
  how it is measured. Each such write needs an author in the audit trail.
- **viewer** is **read-only across the product**. A viewer reads every page but cannot
  change a single declared row.

### What is admin-gated

An admin act is any handler mounted behind `requireAdmin`. A viewer hitting one is
refused with **403**. Concretely, admins alone may:

| Area | Admin-only act |
| --- | --- |
| Scope | Declare seeds and exclusions, upload zone files, set custody, opt a scope into `cold` |
| Vantages | Provision a prober (`POST /probers`) |
| Proposals | Run an org-name lookup, confirm a proposal into a seed, decline one |
| Scans | Trigger a scan (`POST /scans/trigger`), finish onboarding |
| Sources | Toggle a discovery source on or off ([sources.md](sources.md)) |
| Port aperture | Edit the verge-core frequency tier |
| Signals | Declare or withdraw an annotation |
| Reports | Declare a report schedule |
| Delivery | Create/update/delete channels, set the retention dials |
| SSO | Add/edit/delete an identity provider, remove a binding ([sso.md](sso.md)) |
| Integrations | Install or disconnect an integration |
| **Team** | **Invite, change a role, require re-enrollment, remove an account** |

**Settings** is admin-gated as a whole (`GET /settings` is behind
`requireSettingsAdmin`), so a viewer cannot even open the Team tab. This is stricter than a
read surface like `/sources`, which a viewer *can* open but not toggle. The one carve-out is
**API access** (`?tab=api`), which a viewer may read but not change (ADR-0173).

### What a viewer may still do

Read-only is product-wide, but every account governs **its own** credentials. Through
**Profile** (`/profile`, viewer-readable) any signed-in account may act on its own row:

- change its own password,
- enrol or hold its own two-factor,
- mint and revoke its own personal API tokens,
- link or unlink its own SSO identity,
- end its own session.

None of that is an admin act — it touches only the caller's own row.

---

## Inviting a new account

New operators are added by **invitation**, not by an admin typing someone else's
password. From **Settings → Team**, open the invite dialog. Choose the **role** the new
account will hold (admin or viewer). That is the only field. An invite binds a *role*,
never a username or an email. The invitee chooses their own username and password when
they accept.

### The invite lifecycle

1. **Mint** — the admin submits the dialog (`POST /settings/accounts`). A single-use,
   high-entropy token is generated, stored only as its SHA-256 hash, and the plaintext
   **join link is revealed once** on the page. It is also written to the `web` logs,
   exactly as the setup and password-reset tokens are. A self-hosted instance has no mail
   service to send it. The invite **expires in 7 days**.
2. **Deliver** — hand the copied link to the new operator out of band. There is no
   in-product delivery.
3. **Accept** — the invitee opens `GET /invite?token=…` (pre-auth, holding only the
   token and no session). They see the role being granted and set a **username and
   password** (`POST /invite`). Usernames are up to 64 characters. Passwords are 12–72.
4. **Create & spend** — a new account is created at the invite's role and the token is
   consumed. The link is inert forever after. Acceptance grants **no session**. The new
   operator arrives at `/login` with an *"Account created — sign in with your new
   credentials"* notice and signs in normally. They then enrol two-factor on first
   sign-in. No privileged session is ever minted straight from a token.

A token that is missing, already spent, or past its 7 days renders an *invitation
invalid* page rather than a form that would fail on submit.

---

## Changing an account's role

From **Settings → Team**, open the change-role dialog on a member. Save the dialog
(`POST /settings/accounts/role`). The Save control stays disabled until the selected role
actually differs, so a no-op never fires.

One invariant is enforced in code: **you cannot demote the last admin.** A change that
would leave the estate with zero admins is refused with *"promote another account
first."* Zero admins would leave every remaining account read-only and block every
mutation permanently. You cannot change your own role. A member never acts on their own
row in Team. Adjust your standing from another admin account.

---

## Re-enrolling a lost second factor

When an operator loses their authenticator, an admin can force a fresh enrolment. Go to
**Settings → Team → require re-enrollment** (`POST /settings/accounts/reenroll`). This
clears the member's TOTP secret and enabled flag. Their current authenticator **stops
working immediately**. Their next sign-in guides them through two-factor setup again. It
touches **neither their password nor any session** — it is only a second-factor reset.

This is the admin-side recovery. The operator's own **recovery codes** are the
self-service path that needs no admin at all — see
[authentication.md](authentication.md) for the recovery-code flow and how re-enrolment
issues a fresh set.

---

## Removing an account

Removal is the most destructive Team act, so it is gated hardest. From **Settings → Team**
open the remove dialog. **Type the member's exact username** to confirm
(`POST /settings/accounts/remove`). It is reached only through that dialog, never a menu
click.

Three refusals protect the estate:

- **You cannot remove yourself.**
- **You cannot remove the last admin** (same invariant as demotion).
- **An account that authored attributed acts cannot be removed.** Seeds, channels and
  other declared rows carry a `created_by` reference to their author. Rather than orphan
  that work, the database refuses the delete and tells you to *reassign or keep the
  account*. In practice a brand-new account that has declared nothing can be removed
  cleanly. A working admin's account usually cannot. Demote it to viewer instead to
  retire it while preserving its authorship.

There is no audit log of these acts — this build keeps no queryable admin-action feed
(the **Audit** tab is honestly empty). The operational records that *do* exist are the
delivery record and the message store.

---

## How this meets the first-run admin and SSO

**The first admin comes from `/setup`, not an invite.** On a fresh instance with zero
accounts, the single-use setup token opens `/setup`, where you create the **first account
at role admin**. The token is printed to the logs, or pinned with `VERGE_SETUP_TOKEN`.
Creating that account closes the setup window. Once any account exists, `/setup` redirects
to `/login` and the token is spent. Every later account descends from that first admin
through the invite flow above. See [first-run.md](first-run.md) and
[using.md](using.md).

**SSO never creates or roles an account.** An SSO identity is only an *additional
sign-in method* attached to an account that already exists here. A verified identity with
no binding is **refused at sign-in**, never provisioned. The user is told to sign in with
a password and link the identity from their own Profile. So the role always lives on the
local account, never on the identity provider. Inviting, re-roling, and removing are
purely local-account acts, regardless of how the operator signs in. Full detail is in
[sso.md](sso.md).

---

## Route reference

| Route | Method | Gate | Purpose |
| --- | --- | --- | --- |
| `/setup` | GET/POST | setup token | Create the first admin; closes once any account exists |
| `/settings?tab=team` | GET | admin | The Team management surface |
| `/settings/accounts` | POST | admin | Mint an invite at a role |
| `/settings/accounts/role` | POST | admin | Change an account's role (last-admin guard) |
| `/settings/accounts/reenroll` | POST | admin | Clear a member's second factor |
| `/settings/accounts/remove` | POST | admin | Remove an account (typed-name confirm, guards) |
| `/invite` | GET/POST | pre-auth token | Accept an invitation; set username + password |
| `/profile` | GET | viewer | Self-service credentials, tokens, 2FA, SSO links |
| `/settings?tab=api` | GET | viewer | Read whether the JSON API is enabled; the toggle is admin-only |
