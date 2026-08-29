# ADR-0123: a token-authed API is read-only always, opt-in and off by default, and a bearer path fully separate from sessions

- **Status:** Accepted
- **Date:** 2026-08-26
- **Ticket:** [#660 A1 — ADR-0123: reverse #6, admit read-only opt-in token API + withdraw the refusal](https://github.com/winniel123/verge-asm/issues/660)
- **Map:** [#658 Consume v3.18.0 — API token surfaces (#390) + Backup & updates (#391)](https://github.com/winniel123/verge-asm/issues/658)
- **Reverses, narrowly:** [ADR-0001](./0001-stack-and-runtime.md)'s *"No JSON API in v1"* row and its *"Read-only or full JSON API"* rejected-alternative — but only for a **read-only** surface, and only under the containment this ADR builds. ADR-0001's decision to ship no *mutating* API stands.
- **Withdraws the refusal at its specifying sites** ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) discipline): spec §4.1/§4.3/§7, ADR-0001, [ADR-0039](./0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md), and the `CONTEXT.md` Channel entry.
- **Leaves untouched:** [ADR-0039](./0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md)'s outbound `Channel` (still one-way, bearer-free, opens no read of the instance), and the `internal/delivery` *"no bearer, ever"* rule — that bearer is **us→receiver** and is unrelated to this one. [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md)'s secret-placement rule is preserved in full.

## Context

[#6](https://github.com/winniel123/verge-asm/issues/6)/ADR-0001 refused a JSON API in v1 on two grounds, quoted here in full because the reversal is scoped to exactly one of them:

1. *"An API token is a bearer credential that **bypasses the TOTP** #11 just decided on — a weaker second authenticated surface guarding the full inventory."*
2. *"The integration need here is **push, not pull.** Nobody polls an ASM tool; they want to be told when `firewalled → exposed` fires ... so deferring the API costs no integration story."*

Ground 2 is a product-fit argument, not a security one, and it has weakened in the two years since: operators integrating verge-asm into their own dashboards, CMDBs, and ticketing do want to **pull** the current inventory on their own cadence, and the session-authed CSV/JSON export (§4.1) serves that only through a human with a browser. The design package **v3.18.0** surfaces a real API-token management screen (Profile last-used rows, a Settings·Access enable toggle), which is the operator-facing commitment that this integration story is now in scope.

Ground 1 is the one that must be answered on its own terms, because it is a **security** claim and it is correct as far as it goes: a bearer credential that can do everything a session can — including mutate the estate and its config — would be a second, TOTP-free surface over the most sensitive object this system holds. This ADR does not wave that away. It **removes the thing the argument is about.**

### The threat the refusal names, stated precisely

The instance's database is *"a complete, current map of the operator's attack surface"* (ADR-0001, ADR-0039). #11's TOTP guards **admin acts** — a `Seed`, an exclusion, a `Scan` config change, a `Channel` (§4.3: *"every Declared act ... is an authenticated admin act"*). TOTP's protection is therefore, precisely, over **mutation of the estate and its configuration**. A leaked bearer that could perform a Declared act would walk past exactly that guard — ground 1's hazard, verbatim.

But a bearer that can **only read** walks past nothing TOTP was defending, because a session can already read the whole inventory and export it as CSV/JSON **without a second TOTP challenge** (§4.1). The read is already available to any authenticated session. TOTP does not re-gate each read. So a read-only bearer exposes no capability the session model does not already expose to a logged-in operator — it changes *who holds the credential and how it is carried*, which is a credential-management problem (revocation, scoping, transport), not a bypass of an auth factor that was guarding reads. It never was.

This is the seam the reversal turns on: **read is not what TOTP protects. Mutation is.** Reverse #6 for reads, keep it absolute for mutation.

## Decision

> **A token-authed request is a read of the inventory and can never be anything else. The `/api/v1` surface it reaches is off until an admin turns it on, is served by a bearer path that shares no machinery with the session/cookie surface, reads the account's role live on every request, and can perform no mutation and no Declared act. #6's refusal is reversed for this surface and for no other; the mutating API, the outbound `Channel`'s bearer-freedom, and the `internal/delivery` "no bearer, ever" rule are all untouched.**

### 1. Read-only, always — the property that answers ground 1

A token-authed request can perform **no mutation and no Declared act**: no `Seed`, no exclusion, no `Scan`/config change, no `Channel`, no account change, no enable/disable — nothing that writes to the estate or its configuration. This is not a per-endpoint permission check that a future endpoint might forget. It is a property of the **bearer path itself** (§3): the path mounts only read handlers and the mutating verbs are not routed on it at all, so "a token can mutate" is not an expressible state rather than a guarded one — the same *make-the-violation-inexpressible* move ADR-0009/ADR-0058 prefer over a test.

A leaked read-only bearer therefore retrieves what the session-authed export (§4.1) already yields to any logged-in operator, and **cannot change the estate or its config** — which is where TOTP's protection actually lives (Context). The revocation and scoping story (tokens are revocable, live-role, coarsened last-used) below is what bounds the *read* credential. TOTP continues to guard every write, unbypassed.

### 2. Opt-in, off by default — and "off" means absent

`/api/v1` is **disabled unless an admin enables it**, a single instance-wide flag (`instance_config.api_enabled`, minted by map child F, carrying who/when like the `retention_settings` singleton). Minted tokens stay **inert** until the surface is enabled. A token is not a capability on its own.

**Disabled ⇒ every `/api/v1` path 404s.** Surface-off is made *indistinguishable from surface-absent*: a probe against a disabled instance cannot tell "API exists but is off" from "this build has no API," because both answer `404`, not `401`/`403`. This denies an unauthenticated scanner even the fact that the surface exists, and it means the default posture of every instance is byte-for-byte the pre-ADR-0001 posture — no API reachable at all — until an admin makes a deliberate, audit-recorded choice otherwise.

### 3. The bearer path is fully separate from sessions

The token path and the session path share **no** authentication machinery, in either direction:

- A token **never** mints a cookie, **never** establishes a session, and is **never** accepted on the HTML surface.
- The session cookie is **never** accepted on `/api` — a browser session cannot drive the API by riding its cookie, and a token cannot drive the HTML app.
- `currentAccount` (the session-derived request identity the HTML handlers read) is **untouched**. The API derives its own request principal from the verified token and does not enter the session middleware at all.

Two consequences follow that matter for the security argument. First, the API surface cannot be reached by CSRF or by a stolen cookie, because it does not read the cookie. Second, the session surface cannot be reached by a stolen token, because the token mints nothing the HTML app trusts. The two credential classes are disjoint, so a compromise of one is not a foothold on the other — which is the isolation ADR-0001's *"second authenticated surface"* worry was really about, delivered by construction rather than asserted away.

### 4. Live role, revocable, coarsened last-used — bounding the read credential

- **Role read live per request.** The token authenticates *which account*. The account's **role** (admin/viewer) is read from the account row **on every request**, never frozen into the token. This is the same rule §4.3 already states for sessions (*"a session's role is still read from the local account row on every request"*), applied identically here: demoting an account to viewer, or disabling it, takes effect on its tokens' very next request with no token reissue.
- **Revocable.** Tokens are individually revocable (the management surface already models this). Revocation is immediate on the next request for the same live-read reason.
- **`last_used_at`, coarsened.** Each successful token request records a `last_used_at` so the operator can spot a live-but-forgotten token, written **coarsened** (at most once per hour per token — the exact cadence is A4/A2's to implement) so the write is not one row per request and the timestamp is not a fine-grained access log of the operator's own integration traffic. A never-used token reads as **null** ("never"), which the Profile surface renders as the empty state.

### 5. What is explicitly *not* touched

- **The outbound `Channel` (ADR-0039) is unchanged.** It remains one-way, carries the message and never the estate, grants no read of the instance, and uses **no bearer header** — its signature authenticates *us to the receiver*. This ADR opens an **inbound read** surface. ADR-0039 governs an **outbound push** surface. They are different directions of a different credential, and admitting the former says nothing about the latter. ADR-0039's *"a pull feed is the one option #6's constraint genuinely kills"* is narrowed only in that the killed thing was an **unauthenticated** feed a reader polls with no credential. The surface admitted here is bearer-authed, opt-in, and off by default, which is not that feed.
- **`internal/delivery`'s "no bearer, ever" is unchanged and unrelated.** That rule is about the delivery POST to a receiver (us→receiver). It is not an authentication surface on this instance and this ADR does not reach it.
- **The mutating API stays refused.** ADR-0001's refusal of a *full* JSON API — one that can write — is preserved in full. Only the read-only surface is admitted.
- **ADR-0053 is preserved.** Token hashes are held where the act is performed. The shared store holds no plaintext secret. (Token *storage* — constant-time hash lookup — is A2's build, not this ADR's decision.)

## Consequences

- **`/api/v1` read-only endpoints become buildable** (map child A3), mounted as read mirrors of the reads the HTML surface already wraps, `404` when disabled or absent, with no mutating verb routed.
- **Token verification and the enable gate become buildable** (A2/A4): `GetPersonalTokenByHash` (constant-time), the coarsened `last_used_at` touch, the Bearer middleware, and the `api_enabled` gate.
- **#6 is reversed for reads and for nothing else.** The out-of-scope index (§7) loses the unqualified *"a JSON API and API tokens"* entry and gains a read-only, opt-in one pointing here. The mutating API remains out of scope.
- **The push story is undamaged.** ADR-0039's notification `Channel` still serves *"tell me when `firewalled → exposed` fires"*. This ADR adds *"let me pull the current inventory"* alongside it. Neither displaces the other.
- **The default instance is unchanged until an admin acts.** With `api_enabled` false — the ship default — every `/api/v1` path `404`s, so an instance that never enables the API is indistinguishable from one built before this ADR.
- **The withdrawals are written at their sites** (per ADR-0058): spec §4.1/§4.3/§7, ADR-0001's *"No JSON API"* row / rejected-alternative / pull-refusal amendment, ADR-0039's bearer-bypass and pull-feed clauses, and the `CONTEXT.md` Channel entry's *"opens no second authenticated surface / a pull feed ... there is none"* sentence — each struck-or-annotated in place with a dated pointer here, so no site still reads, alone and in the present tense, *"there is no JSON API / no second authenticated surface / no pull surface."*

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Keep #6 absolute — no API at all | The integration-need (ground 2) has real weight now and the design package commits to the surface; and ground 1 (the security claim) does not reach a **read-only** surface, because TOTP guards mutation, not reads a session can already export. Refusing on ground 1 over-reads what that ground defends |
| Admit a **full** (read+write) token API | This is the surface ground 1 correctly refuses: a write-capable bearer is a second, TOTP-free path to mutate the estate and its config. Kept out of scope; only reads are admitted |
| On by default | Every instance would ship a second authenticated surface over the full inventory that no admin chose. Off-by-default keeps the default posture identical to pre-ADR-0001, and `404`-when-off hides even the surface's existence |
| Let a token mint or ride a session (shared auth path) | Collapses the two credential classes, so a stolen cookie drives the API and a stolen token drives the HTML app — the coupling ground 1 fears. Full separation (§3) makes each compromise local to its own surface |
| Freeze the role into the token | A token minted while admin would keep admin after a demotion until reissued — a revocation hole. Live per-request role read (matching §4.3 for sessions) closes it with no reissue |
| Fine-grained `last_used_at` (one row per request) | A write amplifier and an access log of the operator's own integration traffic. Coarsening to ≤1/hr keeps the "is this token still live?" signal without either cost |
| Escalate as a doctrine conflict (AWAITING DESIGN) | Not available: the design package rules the surface **in** (v3.18.0 ships its management UI and the map directs the build), the reversal is scoped and reasoned on #6's own two grounds, and read-only + opt-in introduces no new mutation path. This is a recorded reversal, not an unspecced decision |
