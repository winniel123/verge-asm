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

`unknown` failing closed is the whole point. An address we cannot classify is an address
whose owner we have not identified, which is the case where scanning is least defensible.

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

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Ownership as a label only; scan everything discovered | Wider surface map, but it scans hosts the operator cannot consent for, and ships that behaviour as the default to every self-hosted deployment |
| Ask the operator to confirm each discovered address before probing | Consent machinery the map explicitly scoped out, and it makes discovery a foreground task |
| Probe `third-party` addresses at a gentler rate | Rate is not the objection — authority is. A slow scan of a stranger's host is still a scan of a stranger's host |
