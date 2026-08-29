# ADR-0022: Confirmation is singular; declining may be plural

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#50 What does the operator see when confirming or declining a Proposal?](https://github.com/winniel123/verge-asm/issues/50)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Constrains:** [ADR-0002](./0002-ownership-gates-probing.md), [ADR-0012](./0012-a-proposer-is-not-a-source.md)

## Context

[#27](https://github.com/winniel123/verge-asm/issues/27) amended
[ADR-0002](./0002-ownership-gates-probing.md) so that `Custody` reads `Seed`s alone and registry
lookups only **propose** scopes. That made confirming a `Proposal` **the only route by which the
probing gate opens on address space the operator did not type by hand.** Its forcing measurement is
the one worth carrying: a single address-scope `Seed` inside AWS shares its delegated-stats
opaque-id with **177 resources totalling 76,046,336 IPv4 addresses**, against a median ARIN holder
of **512**.

[ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) then established that for the
modal, cloud-resident install the correct response to that expansion is to **decline all of it**.
So declining is not this surface's unhappy path — it is the expected one for the majority of
installs, and confirming is the minority act with the whole consequence attached.

Drawing the surface ([#50](https://github.com/winniel123/verge-asm/issues/50)) put those two facts
in tension. A list of 177 rows with a per-row control is a list that demands a batch affordance, and
the obvious way to supply one is a checkbox column with `Confirm selected` beside `Decline
selected`. That affordance would silently undo #27: it restores, in one gesture, exactly the
76-million-address gate opening that the amendment exists to prevent.

## Decision

**Confirming a `Proposal` is one act per address scope, always. Declining may be done over a whole
lookup or a whole registered holder in one act. The two are deliberately not symmetric, and no
surface may draw them as peers.**

### The asymmetry is not a preference — the two acts fail in opposite directions

The decisive property is what the *wrong* answer costs, and it is not the same magnitude in each
direction:

| Act | Wrong answer costs |
| --- | --- |
| Confirm | The probing gate opens on space the operator does not control. #27's 76,046,336 addresses, in one gesture, against a third party's production |
| Decline | Coverage. The scope was already outside every `Seed`, so its `Custody` was already `third-party` — declining adds durability and non-re-offer, nothing else |

The second row is the part that makes batching safe rather than merely convenient. An unconfirmed
`Proposal` is **read by nothing** ([ADR-0012](./0012-a-proposer-is-not-a-source.md)), so *declined*
and *never answered* have the **same effect on the gate**. Batching a decline batches an act whose
worst case is a coverage loss the operator can reverse on the same screen. Batching a confirm
batches the one act in the product that cannot be quietly reversed, because it has already sent
packets.

### The cost of no bulk confirm is bounded, and it is measured

[#26](https://github.com/winniel123/verge-asm/issues/26) measured the distribution the objection
depends on: **79.1% of ASN-less holders hold exactly one IPv4 block, and 97.5% of all holders hold
ten or fewer.** For the modal case the affordance saves zero clicks. For 97.5% of the population
that can use this path at all — itself under 1% of ADR-0003's persona, per #26 — it saves at most
nine, once, over the life of the install. That is the entire benefit being weighed against the
76-million-address failure, and it is why the counts are the argument rather than a mitigation.

### It is not a threshold, a cap, or a typed confirmation

Nothing branches on the address count. **The gesture is identical at every size**. What changes is
the sentence the operator reads, because the count is rendered *in the affordance's own label* —
`Confirm 256 addresses` and `Confirm 8,388,608 addresses` are the same control. That distinction
matters, because #27 refused a size cap as *an invented threshold inside the safety path* and
refused an exclusion list as [#31](https://github.com/winniel123/verge-asm/issues/31)'s
signature-database line. A typed confirmation over some magnitude is a threshold wearing a keyboard
and is refused for the same reason.

### It is also not ADR-0002's rejected per-address queue

The unit stays the **scope**, which is #27's *confirm-a-scope-once* — a boundary declaration the
operator already authors — and never the address. ADR-0002's *"confirm each discovered address"*
stays rejected, and [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) kept it
rejected for the `custody extension` on the same grounds.

## Consequences

- **A `confirm all` affordance may not be added later as an ergonomic improvement.** It is the
  specific thing this ADR exists to forbid, and the request for it will arrive as a usability
  complaint rather than as a safety proposal — which is how it would get through.
- **The decline path must stay cheap, and that is a requirement rather than a nicety.** A surface
  that makes 177 declines expensive pushes the operator into abandoning the lookup. Abandonment is
  *safe* — nothing is confirmed — but it leaves 177 proposals to be re-offered and it is a coverage
  failure, so cheap declining is what buys the asymmetry its ergonomics back.
- **Undo may not become a confirm shortcut.** Reversing a decline returns the scope to the pending
  proposals. Confirming it afterwards is a fresh, singular confirmation. Otherwise the one
  deliberate act per scope is recoverable in bulk through the undo path.
- **This binds any future proposer.** It is a rule about the confirmation act, not about the ARIN
  SWIP path or about delegated-stats sibling expansion, so a proposer added in a later release
  inherits it without re-argument — which matters because
  [#43](https://github.com/winniel123/verge-asm/issues/43) established that a proposer may be added
  or dropped in any release with no drift consequence whatever, and therefore with very little
  scrutiny.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Symmetric bulk — `Confirm selected` beside `Decline selected` | The affordance that undoes [#27](https://github.com/winniel123/verge-asm/issues/27). Drawn as variant A of the prototype specifically so the argument is visible: with `fill=cloud` the button reads `Confirm 177 selected` and opens the gate on 76,046,336 addresses |
| Symmetric per-scope — no bulk anywhere | Loses nothing on safety and loses the modal case. ADR-0013 makes declining 177 scopes the expected act, and 177 clicks to do the right thing is a surface that teaches the operator to stop reading |
| Bulk confirm above a size *floor* — batch only small scopes | An invented threshold inside the safety path, refused by [#27](https://github.com/winniel123/verge-asm/issues/27). It is also backwards in practice: the operator with one /24 does not need it, and the estate that would use it is the one with enough scopes to stop reading them |
| Bulk confirm behind a typed confirmation | A threshold wearing a keyboard. It branches on magnitude, and every version of it needs a number nobody can attest |
| Bulk confirm restricted to one registered holder | The AWS case *is* one registered holder. The grouping that makes declining safe is exactly the grouping that would make confirming catastrophic |
| Leave it unstated — a rendering decision that lives in [#50](https://github.com/winniel123/verge-asm/issues/50) | The map's own record is that sessions resurrect withdrawn framings; #39 was filed on a premise #27 had withdrawn hours earlier. A safety rule whose only home is a closed prototype ticket is one a usability pass reverses without noticing |
