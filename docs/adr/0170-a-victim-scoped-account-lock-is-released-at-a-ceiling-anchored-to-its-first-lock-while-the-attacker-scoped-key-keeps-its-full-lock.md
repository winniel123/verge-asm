# ADR-0170: a victim-scoped account lock is released at a ceiling anchored to its first lock, while the attacker-scoped key keeps its full lock

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1374 ADR gaps: cmd/web production 16/17 (#1217)](https://github.com/winniel123/verge-asm/issues/1374), gap 1
- **Sweep PR that deleted the comment:** [#1375](https://github.com/winniel123/verge-asm/pull/1375)
- **Rests on:** [ADR-0159](./0159-an-unnamed-proxy-is-never-trusted-so-the-client-ip-is-the-immediate-peer-and-a-fronted-deployment-must-name-its-proxies.md), which supplies the `ip:` key this ADR calls attacker-scoped: how the client IP is derived, and that it reaches nothing else. **This ADR restates none of that** and rules only the ceiling and the asymmetry
- **Narrows:** the same ADR's Context, which reports *"The 15-minute release ceiling applies only to keys with the `acct:` prefix. An `ip:` key has no release ceiling"* as a measured cost of a misnamed proxy set. Its Decision rules neither sentence — [`comment-policy.md`](../spec/comment-policy.md) §8.3 shape 3 — so the gap stands and this ADR rules them
- **Not bound by:** [ADR-0160](./0160-a-backup-redacts-a-reversible-cleartext-credential-and-carries-a-hash-or-an-externally-keyed-ciphertext-and-restore-re-applies-the-same-redaction.md), whose "lockout" is the post-restore second-factor lockout ([#1419](https://github.com/winniel123/verge-asm/issues/1419)) — a key-rotation fact, not a rate-limiter fact
- **Bounds:** [`authentication.md`](../guides/authentication.md):64-68, at its own site, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)

## Context

[`cmd/web/ratelimit.go`](../../cmd/web/ratelimit.go) stated this rule five times — on
`acctPrefix`/`acctLockCeiling`, on `firstLockedAt`, on the constructor, in `locked` and in `fail` —
until #1375 deleted them. Two compressed survivors remain, at `:55` and `:82`. Both are **uncited**,
and between them they state the rule in nineteen words and carry none of the ground:

```go
// A capped account lock stops a pre-auth attacker locking a known username out for good.
// Anchoring to the first lock stops repeated re-locks sliding the ceiling forward.
```

**The citation is dead.** All five blocks cited `#738`, which returns HTTP 410, *"This issue was
deleted"*. [`comment-policy.md`](../spec/comment-policy.md) §8.3 rules that a deleted issue
suppresses nothing.

**The live citation on this file argues the other way.** `ratelimit.go:9` cites `#322`, live, closed
and titled *"TOTP/login has no rate limiting or lockout — brute-forceable 2FA"*. §8.3 requires
reading the body first. #322 asks for *"per-account (and per-IP from `RemoteAddr`, never a proxy
header) failed-attempt counting with exponential backoff / temporary lockout"* and says nothing about
releasing one. It grounds the limiter, not the ceiling — it is this ADR's rejected alternative 1.

**The five-place search returns one near miss and no hit.** `acctLockCeiling`, `firstLockedAt`,
"lock ceiling", "victim-scoped" and "attacker-scoped" appear zero times across `docs/spec/`,
`docs/adr/`, `docs/research/`, `docs/guides/` and `CONTEXT.md`. The near miss is
[`authentication.md`](../guides/authentication.md):64-68, which tells an operator *"Both the password
step and the TOTP step are throttled per-account **and** per source IP. Too many failures locks the
key for a few minutes"* — and states neither the ceiling, nor its anchor, nor the asymmetry, nor the
denial-of-service trade the ceiling exists to make.

### What the code does today

The limiter carries `acctPrefix` and `acctLockCeiling` (`ratelimit.go:19-20`), set to `"acct:"` and
`15 * time.Minute` at `:40-41`. **Fifteen minutes is still the real value.** The rest of the schedule
is `maxFailures: 5`, `window: 5 * time.Minute`, `baseLockout: 5 * time.Minute`,
`maxLockout: time.Hour` (`:36-39`).

`lockoutFor` (`:101-113`) doubles the base once per failure past the threshold and clamps at the
maximum, so one key escalates:

| Failures | Lock span |
| ---: | --- |
| 5 | 5 minutes |
| 6 | 10 minutes |
| 7 | 20 minutes |
| 8 | 40 minutes |
| 9 or more | 1 hour |

`fail` (`:65-91`) sets `firstLockedAt` to the current instant at `:83-85`, and only when it is zero.
It clears `firstLockedAt` and the failure count at `:75-78`, where a key that is **not currently
locked** has been idle for longer than `window`. `reset` (`:93-98`) deletes the entry outright, which
clears the anchor with everything else; `cmd/web/auth.go` calls it on every successful password
(`:263`), dev bypass (`:302`) and second factor (`:336`).

`locked` (`:46-63`) reads the anchor at `:56-59`: where the ceiling is positive, the prefix is
non-empty, the key carries that prefix and the anchor is set, a key past its ceiling is skipped
instead of reported locked.

`cmd/web/auth.go:345` builds the account key — `func loginAccountKey(username string) string
{ return "acct:" + strings.ToLower(username) }` — and `cmd/web/clientip.go:83-85` builds the IP key
as `"ip:" + s.clientIP(r)`. Both handlers pass the pair together (`auth.go:245`, `:292`).

## Decision

> **The login limiter throttles on two axes and the two are deliberately asymmetric. The per-IP key
> is attacker-scoped and keeps its full escalating lock. The per-account key is victim-scoped: its
> lock is honoured only within `acctLockCeiling` of that key's FIRST lock, so an unauthenticated
> attacker who fails a known username cannot deny the real operator indefinitely. The anchor is the
> first lock, so repeated re-locks cannot slide the window forward. A zero ceiling disables the cap
> and every key then locks fully.**

### 1. The two axes name two different parties, and that is what licenses the asymmetry

An `ip:` key names the host doing the guessing: locking it costs the attacker and costs the operator
nothing. An `acct:` key names the person being attacked: locking it costs the attacker one of a
million usernames and costs the victim their whole console.

A username is not a secret, so the account axis is reachable **pre-auth** by anyone who can spell
one, and a control an unauthenticated stranger trips on someone else's behalf is a denial-of-service
lever. The IP axis is not: an attacker can only lock the address they attack from. The two axes are
not two instances of one control. They are two controls, and only one is bounded.

### 2. The ceiling truncates the escalation, and the truncation is the point

§Context's schedule runs to an hour and an `acct:` key never reaches it. The first lock is five
minutes; past fifteen minutes from that first lock, `locked` stops honouring the key whatever
`lockedUntil` says. The 20-minute, 40-minute and 1-hour steps are unreachable on the account axis,
and the 10-minute step is reachable only for the part of it inside the ceiling. The `ip:` key runs
the schedule in full, and nothing releases it early.

### 3. The anchor is the FIRST lock, and only two acts clear it

Anchoring to the last lock would let an attacker re-arm the ceiling by failing again, which makes the
bound unbounded. `fail:83-85` writes `firstLockedAt` only where it is zero, so every re-lock in a
sustained attack keeps the original anchor. Two acts clear it, and both are correct.

- **A success** (`reset`, `ratelimit.go:93-98`). The operator got in, so the whole entry goes.
- **An idle window** (`fail`, `:75-78`). The key must be **unlocked** and untouched for longer than
  `window`. An attacker who wants a fresh ceiling must therefore stop attacking for more than five
  minutes while the account is reachable, and must then re-earn five failures. The victim's
  availability is what the attacker pays.

### 4. The ceiling is a read-side release, and it spends budget to buy availability

`locked` skips the key. It does not delete the entry, reset the count or stop the escalation. So a
sustained attack past the ceiling leaves the account axis **un-honoured for the whole duration of
that attack**, with only the `ip:` key throttling. That is the trade at its strongest.

It is affordable because the guessing host is still bound by its own uncapped `ip:` key. An attacker
distributed widely enough to defeat the IP axis was never going to be stopped by an account lock
either — they would have chosen a different victim. A zero `acctLockCeiling` refuses the trade and
restores the full lock (`:56` guards on `> 0`). Nothing sets it today.

### 5. `acct:` is the only thing that makes a key victim-scoped, so the prefix is one constant

The rule above is enforced by a string comparison. `ratelimit.go:40` and `auth.go:345` hold that
string twice, in two packageless files, with no type and no shared constant between them. **They are
one fact and must have one home.** A key builder that emits a prefix the limiter does not recognise
produces an account key that locks like an attacker key: for its full escalating span, forever, on a
victim the attacker chose. The failure is silent at compile time and invisible at run time.

## Consequences

- **[`authentication.md`](../guides/authentication.md):64-68 gains the ceiling and the asymmetry.**
  Read alone, its *"locks the key for a few minutes"* describes both axes identically and tells an
  operator nothing about the bound on a lockout someone else triggered. Marked at the bullet per
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).
- **`cmd/web/ratelimit.go:55` and `:82` gain this ADR's citation.** Both survivors are uncited under
  [`comment-policy.md`](../spec/comment-policy.md) §4.7 route 3 today, and route 3 rules the dead
  token, never the rule.
- **The prefix agreement is pinned by behaviour, not by type.** `cmd/web/clientip_test.go:138-173`
  builds its key with `loginAccountKey` and asserts the release, so a drift in either literal fails
  it. The pin is real and incidental: it sits in the client-IP test file, its name is about the
  ceiling, and nothing declares that the agreement is what it protects. The fix is one shared
  constant, and it is not applied here.
- **No production behaviour changes**, and **`#738`'s ground is unrecoverable.** This ADR restates
  the rule from the code and the five deleted blocks. No session should cite #738 for it.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** A rate-limit key is a console term, not a term
  in the measurement domain.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **No ceiling at all — the account key locks for its full escalating span**, which is what [#322](https://github.com/winniel123/verge-asm/issues/322) asked for | A username is public and the axis is reachable pre-auth, so any stranger holds any known account at the 1-hour step indefinitely by failing it five times an hour. The victim's only recovery is an out-of-band unlock this product does not have. It converts a brute-force control into a targeted denial-of-service lever against the operator |
| **Anchor the ceiling to the LAST lock instead of the first** | It is the same alternative wearing a bound. Every fresh lock restarts the fifteen minutes, and an attacker failing once every few minutes holds the account forever while never exceeding the ceiling on any single lock. A window an attacker can slide is not a window |
| **A CAPTCHA or a proof-of-work challenge in place of the lock** | It buys the availability the ceiling buys and pays in currencies this product will not spend. A CAPTCHA is a third-party dependency on the login path of a self-hosted console ([ADR-0001](./0001-stack-and-runtime.md)) and hands a vendor the sign-in page. A proof-of-work is priced for the honest operator's laptop and free for the attacker's fleet, so it moves cost the wrong way at the scale that matters. Both add surface to the one screen an operator must complete under duress, and both would still need a lock behind them, because neither bounds an attacker who solves the challenge |
| **Drop the account axis entirely and throttle per-IP only** | The account axis bounds a distributed online guess against one username, the credential-stuffing half of #322. The ceiling keeps that bound for the first fifteen minutes of any attack, which is where a burst guess lives. Removing the axis gives it up permanently to buy availability the ceiling already buys |
| **Give the `ip:` key a ceiling too, for symmetry** | The `ip:` key names the attacker. Releasing it hands the guessing host an unthrottled retry every fifteen minutes and removes the only control that survives §4's un-honoured account axis. Symmetry between two keys naming two different parties is a property nobody wants |
| **Unlock the account out of band instead — an admin action or an emailed link** | An admin unlock needs a second admin who is not locked out, and a single-operator instance is the common shape. An emailed link needs mail, which `web` does not send. Both replace a bounded wait with a dependency that can itself be unavailable |
