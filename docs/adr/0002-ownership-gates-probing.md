# ADR-0002: Ownership gates probing, it does not merely label it

- **Status:** Accepted
- **Date:** 2026-08-01
- **Ticket:** [#7 Core domain model and ubiquitous language](https://github.com/winniel123/verge-asm/issues/7)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

verge-asm's defensive posture rests on a premise the map states plainly: *the operator
owns the assets being scanned, so no scope-consent machinery is needed.* Every safety
decision in [#4](https://github.com/winniel123/verge-asm/issues/4) — TCP connect over
SYN, per-host rate limits, credentials never submitted — is calibrated to that premise.

Discovery does not respect it. It follows records outward by construction:
`shop.example.com` → `d1x2y3.cloudfront.net` → an address in Amazon's range. Shared
hosting, managed SaaS and CDN fronting all produce the same shape — an address inside the
operator's *discovery reach* and outside their *authority*.

If discovery reach and ownership are the same thing, then the moment verge-asm resolves a
CNAME to a CDN, it port-scans infrastructure the operator cannot consent for, and the
premise is quietly false. This matters more than it would for an internal tool: verge-asm
is AGPL-3.0 and self-hosted, so it is run by people we never meet, against estates we
never see, with defaults nobody reviews.

## Decision

**`Ownership` is derived per `Address` — `owned`, `third-party`, or `unknown` — and it
gates active probing.**

| Ownership | Probing permitted |
| --- | --- |
| `owned` | The full tiered port sets from [#4](https://github.com/winniel123/verge-asm/issues/4) |
| `third-party` | Only the ports the `Name` implies (443, 80), or nothing pending explicit operator opt-in |
| `unknown` | Treated as `third-party` |

It is *derived*, not declared: computed from the operator's address-scope seeds plus RDAP
expansion of their organisation's ranges. That machinery is already required —
[#14](https://github.com/winniel123/verge-asm/issues/14) computes exactly this to verify
that a vantage sits outside every operator-owned range.

> **Amended 2026-08-13 by [#27](https://github.com/winniel123/verge-asm/issues/27) — the
> registry half of that sentence is withdrawn.** `Ownership` is computed from **`Seed`s
> alone**. Registry expansion does not feed the derivation; it *proposes* address scopes
> that the operator confirms into `Seed`s, and an unconfirmed proposal is read by nothing.
>
> The measurement that forced it: the RIR delegated-stats opaque-id groups every held
> prefix by resource holder, and one address-scope seed inside AWS shares its opaque-id
> with **177 resources totalling 76,046,336 IPv4 addresses**. Under the original sentence
> that expansion derives `owned` and this ADR's gate opens on all of it — 96 ARIN holders
> exceed a million addresses, against a median holder of 512. Since
> [#26](https://github.com/winniel123/verge-asm/issues/26) established the cloud-resident
> operator as the *modal* case, that is the common path and not an edge.
>
> A size cap and a hyperscaler exclusion list were both rejected: the first is an invented
> threshold sitting inside the safety path, the second is a signature database wearing a
> different hat, which [#31](https://github.com/winniel123/verge-asm/issues/31) drew a line
> against. Operator confirmation was already this ADR's escape hatch — *"the fix is the
> operator declaring an address scope, which is a `Seed`, so the mechanism already
> exists"* — and this uses it in the widening direction rather than only the correcting one.
>
> **The safety property gets stronger, not weaker.** With registry data out of the
> derivation, no third party's file can open the gate, and a stale table can no longer
> silently change what we probe. The cost is that address space the operator acquires and
> never declares stays `unknown` and is picked up more slowly; it surfaces on
> [#22](https://github.com/winniel123/verge-asm/issues/22)'s `Coverage` rather than
> vanishing.

`unknown` failing closed is the whole point. An address we cannot classify is an address
whose owner we have not identified, which is the case where scanning is least defensible.

> **Amended 2026-08-13 by [#40](https://github.com/winniel123/verge-asm/issues/40) — see
> [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md), which carries the
> reasoning in full rather than being inlined here.** Four things in this ADR are superseded:
>
> - **`Ownership` is renamed `Custody`** and means **control of the listener**, not registry
>   title. This ADR's text is left unrewritten; read every `Ownership` below as `Custody`.
> - **`unknown` is deleted.** With the derivation reading `Seed`s alone there is no lookup left
>   to fail, so the sentence above has no producer. Its *intent* is unchanged — everything not
>   covered by a `Seed` is `third-party`, which is the closed direction.
> - **A name-scope `Seed` may carry a `custody extension`**, declaring that the addresses its
>   names resolve to are under the operator's custody. Off by default; transitivity stops where
>   the resolution chain leaves the declared zone. This is what lets the cloud-resident modal
>   operator of [#26](https://github.com/winniel123/verge-asm/issues/26) be described at all.
> - **The derivation's inputs are no longer all Declared**, so the gate now moves on
>   measurement. The rejected alternatives below are all unaffected — in particular *"ask the
>   operator to confirm each discovered address"* stays rejected, and ADR-0013 rejects it again
>   in its own terms.

> **Amended 2026-08-14 by [#81](https://github.com/winniel123/verge-asm/issues/81) — see
> [ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md).** This ADR's whole vocabulary is
> gate-shaped — *classify*, *opens on*, *permitted* — and it never said whether the `operator` side
> of the table has an **extension** the prober walks. It does, for one of the two `Seed` kinds:
>
> - **An address-scope `Seed` enumerates.** Every address inside a declared CIDR is a subject from
>   the declaration and is probed under the tiers this table permits, whether or not any resolution
>   ever cited it. A name-scope `Seed` does not enumerate; its addresses arrive by measured
>   resolution, and only under a `custody extension`.
> - **The `/22` range size cap is checked at declaration**, per scope, and is operator-configurable.
>   It is **not** the size cap the #27 amendment above rejected, and the difference is structural:
>   #27's cap sat *inside the derivation*, deciding on a third party's file whose addresses are
>   `operator`. This one sits in front of a **Declared act**, adjudicates cost rather than truth,
>   and cannot fail silently — its only failure mode is a declaration that does not take. `Custody`
>   itself remains the total lookup ADR-0013 §2 made it, uncapped and unchanged.

## Consequences

- **Exposure attribution splits by ownership.** For an `owned` address, exposure belongs
  to the `Address`. For a `third-party` one, the operator's exposure belongs to the
  `Name` — *"this name is served from infrastructure you do not control"* — and the open
  ports on that address are largely another tenant's story. This is a finding worth
  surfacing in its own right, and it is available without scanning a stranger.
- **Ownership is load-bearing vocabulary**, not a display flag. It appears in
  [`CONTEXT.md`](../../CONTEXT.md) under Derived.
- **Coverage is deliberately narrower** than a tool that scans everything it reaches. An
  operator whose estate is mostly SaaS-fronted sees fewer ports. That is the intended
  trade.
- **RDAP accuracy becomes a safety property**, not just a discovery nicety. A wrong
  expansion either scans someone else's range or fails to scan the operator's own.
- **Classification needs an operator escape hatch.** Addresses will be misclassified, and
  the fix is the operator declaring an address scope — which is a `Seed`, so the
  mechanism already exists.
- **RDAP accuracy is no longer a safety property** *(amended by
  [#27](https://github.com/winniel123/verge-asm/issues/27))*. It was, while expansion fed
  the derivation; now a wrong expansion produces a wrong *proposal*, which the operator
  declines. Accuracy governs how much typing onboarding saves, not what gets probed. The
  registry files' own caveat — that they record where a range was *allocated*, not who
  uses it now — is thereby survivable rather than load-bearing.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Ownership as a label only; scan everything discovered | Wider surface map, but it scans hosts the operator cannot consent for, and ships that behaviour as the default to every self-hosted deployment |
| Ask the operator to confirm each discovered address before probing | Consent machinery the map explicitly scoped out, and it makes discovery a foreground task. **Still rejected after [#27](https://github.com/winniel123/verge-asm/issues/27)**, which is a different act and must not be read as reversing this row: #27 confirms a *proposed address scope* once, at onboarding, producing a `Seed` — an authoring step the operator already performs — whereas this row confirms *every discovered address*, forever, inside the discovery loop. One is a boundary declaration; the other is a per-subject approval queue |
| Cap the expansion by group size, or exclude known hyperscaler holders | Considered and rejected in [#27](https://github.com/winniel123/verge-asm/issues/27). A cap is an invented threshold inside the safety path — the model refuses those everywhere else — and it fails silently in both directions: too low and a genuine /16 holder is denied, too high and a mid-sized provider's tenants are scanned. An exclusion list is reference data deciding what an answer *means*, which is [#31](https://github.com/winniel123/verge-asm/issues/31)'s signature-database line, and it would need updating out of band forever |
| Probe `third-party` addresses at a gentler rate | Rate is not the objection — authority is. A slow scan of a stranger's host is still a scan of a stranger's host |
