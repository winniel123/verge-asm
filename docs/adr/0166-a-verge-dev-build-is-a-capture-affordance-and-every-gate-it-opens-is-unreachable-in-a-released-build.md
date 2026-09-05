# ADR-0166: a `VERGE_DEV` build is a capture affordance, and every gate it opens is unreachable in a released build

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1333 ADR gaps: cmd/web/devfixtures.go](https://github.com/winniel123/verge-asm/issues/1333), gaps 1, 4 and 5; [#1334 ADR gaps: cmd/web/auth.go](https://github.com/winniel123/verge-asm/issues/1334), gap 4
- **Sweep PRs that deleted the comments:** [#1335](https://github.com/winniel123/verge-asm/pull/1335), [#1337](https://github.com/winniel123/verge-asm/pull/1337)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md), which rules that a secret is held only where the act it authorises is performed. This ADR uses that test to say what a pinned fixture credential *is not*
- **Read with:** [ADR-0160 §5](./0160-a-backup-redacts-a-reversible-cleartext-credential-and-carries-a-hash-or-an-externally-keyed-ciphertext-and-restore-re-applies-the-same-redaction.md), which records that a restore rotates the key the archived `account.totp_secret` is sealed under, and files the resulting lockout as [#1419](https://github.com/winniel123/verge-asm/issues/1419). §3 below states the tension that finding creates
- **Not bound by:** [ADR-0116](./0116-the-design-package-is-normative-for-look-and-functionality.md) and [ADR-0109](./0109-design-system-components-are-authored-in-claude-design-and-imported.md), whose handoff workflow was retired on 2026-08-28 ([ADR-0145](./0145-design-system-is-the-shared-home-and-source-of-truth-for-ui-assets-and-a-session-edits-it-in-the-repo.md)). Every deleted comment behind this ADR justified itself out of that workflow — `SPEC-CHANGE` rulings, `G1`/`G2` goldens, and a `run.sh` capture script. None of the four is on disk

## Context

`cmd/web` ships a second personality. Under one environment variable the console serves pinned
fixtures instead of live derivations, pins the clock, pins the build version, accepts a fixed TOTP
code, mints a session with no credential, and empties the `account` table on a `GET`.

Nothing under `docs/` states what licenses that. A grep for `VERGE_DEV` across `docs/`,
`CONTEXT.md` and `CONTRIBUTING.md` returns one line — an ADR-0120 Consequences bullet that mentions
a fixture *"served byte-for-byte … under a `VERGE_DEV` build"* while ruling something else. The
comments that carried the rule cited `SPEC-CHANGE #24`, `#35`, `DF-F3`, `G2` and `run.sh`. `find`
returns no `run.sh` in the tree. So the four decisions below were argued entirely out of a workflow
that no longer exists, and each survives uncited under [`comment-policy.md`](../spec/comment-policy.md) §4.7.

**The flag's topology, counted.** `cmd/web` reads `VERGE_DEV` twice.
`cmd/web/main.go:107` sets the server field (`cmd/web/handlers.go:258`, wired at
`cmd/web/main.go:117`). `cmd/web/main.go:45` reads it again to refuse `-seed-fixtures` outside a dev
build — a separate gate because that path seeds and exits at `cmd/web/main.go:84`, before any
`server` exists to hold a flag. One variable, two gates, and the second guards a process that never
listens.

Downstream there are **36 guard sites** in `cmd/web`: 32 of the form `if s.devMode {` (including one
`else if` at `cmd/web/auth.go:1263`) and four compound — `cmd/web/auth.go:232`, `:301`, `:944`, and
`cmd/web/subjects.go:933`. **No affordance sits outside one.** The six `/dev` routes are registered
inside a single `if s.devMode` block (`cmd/web/handlers.go:484-491`), so a released build refuses
them at the mux rather than inside a handler.

`cmd/web/adr0130_contract_test.go:128-138` already reads the flag this way: `devModeBodies` skips
every `if s.devMode` body before checking the ADR-0130 contract, on the stated ground that *"a
dev-mode branch is reached by configuration, never by an operator refusing a form."* It matches the
plain `if` form only, so the four compound guards are invisible to it.

The worker's `devMode` (`cmd/worker/main.go:88`) is a different thing and out of scope: it
**suppresses** output — transcript capture at `internal/queue/worker.go:142`, message production at
`internal/queue/produce.go:56` — rather than opening a gate.

## Decision

> **A `VERGE_DEV` build is a capture affordance, not a deployment mode. Its only consumer is a
> deterministic capture of the console against a throwaway fixture database, and on that ground it
> may bypass an authentication factor, pin a credential value, pin an identifier, and empty a
> table. Every such affordance is gated on `devMode`, is unreachable while that flag is false, and
> is bounded by the database the build is pointed at — never by an ordering, and never by the code.
> A released build sets the flag nowhere.**

### 1. The dev session mint is not a bypass of a factor that exists

`devSessionMint` (`cmd/web/devfixtures.go:46-65`) maps a role to a fixture username, reads the
account, and calls `completeLogin` (`cmd/web/auth.go:423`) directly. No password, no TOTP challenge,
no rate limiter. `seedProfileFixtures` then seeds that same account
`totp_enabled = true` with no secret (`cmd/web/devfixtures.go:183`).

The two halves are one decision, and stated together they are not the contradiction the deleted
comment made them look like. The fixture account **has no second factor to satisfy**: the normal
path would reach `auth.DecryptTOTPSecret` at `cmd/web/auth.go:305` against a `NULL` column and fail.
The mint does not walk around a live gate; it is the only door the account has. `totp_enabled` is a
**render input** on that account — the Profile capture is internally consistent only at TOTP-on —
not a credential check.

The ruling: **a dev fixture account is a rendering subject, not a principal.** It is created only by
`-seed-fixtures` (`cmd/web/main.go:71-76`), it exists only in a database that flag has already
written to, and its session is minted rather than authenticated.

### 2. A pinned credential value is not a secret

Five pins ride the flag: the TOTP code at login (`cmd/web/auth.go:301`), the enrolment secret and
the enrolment reset (`cmd/web/auth.go:897-902`), the confirm code
(`cmd/web/auth.go:944`), the minted personal-token plaintext
(`cmd/web/auth.go:1657-1658`, `fixtureMintedToken` at `:1663`), and the build version
(`cmd/web/auth.go:173-174`).

ADR-0053's test is *where is the act it authorises performed*. **A pinned value authorises nothing
outside the fixture database**, so it is not a secret under that test and its presence in the source
is not a placement violation. That is why each carries a `#nosec G101` rather than a rotation
procedure (`cmd/web/devfixtures.go:114`, `:266`, `:268-269`, `:128`).

The pins are additive at the gate they cross, never subtractive: `cmd/web/auth.go:944` reads
`if !(s.devMode && code == devFixtureTOTPCode)`, so a live build runs the full RFC 6238 verification
and a dev build runs it for every code but one.

### 3. The tension ADR-0160 §5 exposes, stated and not resolved here

[ADR-0160 §5](./0160-a-backup-redacts-a-reversible-cleartext-credential-and-carries-a-hash-or-an-externally-keyed-ciphertext-and-restore-re-applies-the-same-redaction.md)
records that a restore rotates the session key (`cmd/web/restore.go:226`) and re-derives the TOTP key
from it (`cmd/web/restore.go:395`), so a restored `account.totp_secret` cannot be opened. An account
carrying `totp_enabled = true` across that restore reaches `cmd/web/auth.go:305`, fails to decrypt,
and gets a 500. The operator cannot log in.

`devResetTOTPEnroll` (`cmd/web/devfixtures.go:306-320`) is **the only TOTP reset in the tree**, and
the pinned code at `cmd/web/auth.go:301` fires *before* the decrypt at `:305`. So the recovery
today is: restart `web` with `VERGE_DEV=1`, type `482913`, then open the enrolment form to clear the
column. **An affordance that must never ship is currently the only escape from a live lockout.**

This strengthens the ruling rather than weakening it. The escape works precisely because the flag is
a whole second personality, which is the reason it may never ship. The fix belongs in
[#1419](https://github.com/winniel123/verge-asm/issues/1419) as an operator-facing reset on its own
authorisation, and this ADR licenses no widening of `devMode` toward it.

### 4. `GET /dev/seed/empty` is bounded by the database, not by an ordering

`devSetupSeedEmpty` (`cmd/web/devfixtures.go:361-375`) runs
`TRUNCATE account RESTART IDENTITY CASCADE` at `:366` and then pins `s.setupToken` to
`devFixtureSetupToken` at `:372-374`, reopening the first-run window under a token that is a
compile-time constant at `:359`.

The deleted comment bounded this by capture ordering — Setup is the last screen `run.sh` captures,
so nothing earlier is stranded. **That bound is refused.** `run.sh` is not in this repo, the ordering
is unverifiable from the tree, and a safety property no reader can check is not a safety property.

**The replacement bound is the database.** The route is safe because a `VERGE_DEV` build is pointed
at a throwaway fixture database, and everything it destroys was written by `-seed-fixtures`. Two
other dev routes write that database on the same ground: `devProfileSessionPrepare`
(`cmd/web/devfixtures.go:67-87`) reseeds Profile on every hit, and `devResetTOTPEnroll` clears three
columns and the recovery codes. State the limit honestly: **the code cannot distinguish a fixture
database from a deployment one.** The bound is a precondition on the build, not a property of the
handler, and no check restores it.

### 5. A dev identifier is reserved once, and the reservation covers the whole id space

The dev run ids are **three**, not two. `devRunDetailID = "1407"` (`cmd/web/devfixtures.go:571`) is
the concluded run detail; `1408` is the error screen's missing-run demo; `devRunningRunID = "1409"`
(`cmd/web/devfixtures.go:641`) is the Settings active dispatch's drill-in.

`1408` is **not a Go constant**. It lives in `design-system/fixtures/fixtures.json:253` and
`design-system/examples/console/ErrorPage.jsx:11`, and it works only because `runPage`'s dev branch
is total over the id space: two named ids and a catch-all to `renderMissingRun`
(`cmd/web/scans.go:350-360`). Its sole reservation in Go is the comment at
`cmd/web/devfixtures.go:639`.

The ruling: **the dev id space is allocated in `devfixtures.go` and never reused.** An id the design
package holds and Go does not is still reserved there. Because the dev branch of `runPage` cannot
404, a collision does not fail loudly — it silently swaps one screen for another, which is the whole
reason the reservation is worth a rule. The reservation binds the dev id space only: a deployment's
`Dispatch` ids are unconstrained, and the test corpus already uses `1408` as a live dispatch id
(`cmd/web/scans_stop_terminate_test.go:20`).

## Consequences

- **The residual risk is total and it is stated plainly.** A build shipped with `VERGE_DEV` set is a
  full authentication bypass. `GET /dev/session/admin` mints an admin session against a
  fixture account with no credential (`cmd/web/handlers.go:487`); the pinned code passes the second
  factor for **every** account (`cmd/web/auth.go:301`); and `GET /dev/seed/empty` empties `account`
  and reopens first-run under a published token. **Nothing but the flag prevents any of it.**
- **What does mitigate, today.** `docker-compose.yml` names `VERGE_DEV` on neither service
  (`:13-21`, `:59-63`), so it is not merely unset but has no passthrough — a bare
  `VERGE_DEV=1 docker compose up` does not reach the container. No workflow under `.github/workflows/`
  and no `ENV` line in `Dockerfile` sets it.
- **What does not mitigate.** There is no build tag, no startup refusal, no banner, and **no test
  asserting the six `/dev` routes 404 when the flag is false.** The claim in §1 that no affordance
  escapes the flag is true of the tree today and is held there by review alone.
- **ADR-0053 and ADR-0160 are amended nowhere.** §2 applies ADR-0053's existing test to a value class
  it never named. ADR-0160 §5 already declines to rule the second-factor path; §3 above adds only the
  fact that the dev affordance is the current workaround, which #1419 needs and ADR-0160 could not
  state.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Keep the `run.sh` capture-ordering bound for `/dev/seed/empty`** | The script is not in this repo and was retired with the handoff workflow on 2026-08-28. A reader cannot check the ordering, cannot restore it, and cannot tell whether a new capture was inserted after Setup. It reads as a safety argument and functions as none, which is worse than an honest precondition |
| **Split `devMode` into per-affordance flags** — one for the mint, one for the pins, one for the destructive seeds | It multiplies the number of switches that must all be off in a released build from one to several, and it makes "is this build safe?" a conjunction instead of a lookup. It also breaks `adr0130_contract_test.go:133`, which recognises exactly one flag when deciding what is outside the product surface. The cost lands on every future reader for a separation no consumer asked for |
| **Refuse to start when `VERGE_DEV` is set against a non-empty `account` table** | It reads as a guard and is not one. The capture flow itself boots against a seeded database and `/dev/seed/empty` deliberately empties it, so the check would have to permit the exact state it exists to catch. It also gives a false all-clear on a fresh deployment, which is the case where the bypass is most valuable to an attacker |
| **Gate the affordances behind a Go build tag instead of an environment variable** | It is the stronger containment and it costs the capture flow its single-image property: two binaries, two images, two CI matrices, and a dev build that no longer proves the released code paths render. The pinned-fixture branches sit inside live handlers (`cmd/web/auth.go:455`, `cmd/web/scans.go:350`, and 30 more), so a tag split would fork those functions rather than isolate a package |
| **Drive the capture with a real generated TOTP code instead of a pinned one** | The capture would then depend on a wall clock inside a flow whose whole point is a pinned clock (`cmd/web/main.go:109-114`), reintroducing the non-determinism the fixture clock exists to remove. It also leaves the mint, which is the larger affordance |
| **Give `1408` a Go constant beside 1407 and 1409** | The id is consumed only by the design package and by `runPage`'s catch-all, so the constant would be declared and never read. The repair that pays is a drift test pinning all three against `fixtures.json`, matching `cmd/web/devfixtures_test.go:636-648`, which pins `1407` alone |
