---
number: 2114
title: "The address-scope act reader covers every act class that moves the covered predicate"
slug: the-address-scope-act-reader-covers-every-act-class-that-moves-the-covered-predicate
date: 2026-09-15
status: accepted
source: grilling
ticket: 2114
proof: {ticket: 2072}
relations:
  - {kind: amends, adr: 1946, clause: "3"}
  - {kind: rests-on, adr: 1895}
  - {kind: rests-on, adr: 133}
  - {kind: rests-on, adr: 1644}
---

# ADR-2114: The address-scope act reader covers every act class that moves the covered predicate

## Decision

> **The `Exposure` screen's address-scope act reader covers every act class that moves
> `addressScopeCovered`**: `seed.declared`, `seed.withdrawn`, `proposal.confirmed`, and the
> address-kind `exclusion.declared` and `exclusion.lifted`. The seven-day window, the five-row cap
> and the newest-first order are unchanged.
>
> A reader asks why a panel beside the `Exposure` figure lists one of the five classes that move it.
> The panel's empty state is a statement, and a withdrawal or a confirmed proposal makes it false.
>
> Rejected: narrowing the panel's copy instead, which leaves it honest and useless.
>
> Reversal returns a panel that reports no address-scope edit while one has just moved the figure.

## 1. Context

[ADR-1946](./1946-an-address-scope-edit-reaches-the-operator-through-an-act-reader-never-a-message.md)
§3 specified the reader precisely: `seed.declared` acts, over seven days, five rows, newest first,
linked to the audit tab. #1939 implemented exactly that in `cmd/web/scopeacts.go` and did not widen
past it, because the panel's read was the ADR's own text.

The panel exists to explain a movement in the `Exposure` figure beside it. That is what makes the
one-class read a defect rather than a narrow scope.

## 2. Five act classes move the predicate, and the reader saw one

> **Amended** by [ADR-2171: Declining an address proposal moves the covered predicate, so the act reader covers seven classes](./2171-declining-an-address-proposal-moves-the-covered-predicate-so-the-act-reader-covers-seven-classes.md), 2026-09-16. <!-- adr-marker amends 2171 -->

`addressScopeCovered` (`cmd/web/vantageclass.go`) decides every `Vantage class`, and therefore which
leg of `Exposure` an observation lands on. It reads the live address `Seed` set minus the live
address `exclusion` set.

| Act class | Handler | Moves `covered` | Read before this ADR |
| --- | --- | --- | --- |
| `seed.declared` | `declareSeed`, `cmd/web/seeds.go` | yes | yes |
| `seed.withdrawn` | `deleteSeed`, `cmd/web/seeds.go` | yes | no |
| `proposal.confirmed` | `confirmProposal`, `cmd/web/proposals.go` | yes | no |
| `exclusion.declared` | `cmd/web/exclusions.go` | yes, through ADR-0133 §4's subtraction | no |
| `exclusion.lifted` | `cmd/web/exclusions.go` | yes, through ADR-0133 §4's subtraction | no |

`confirmProposal` calls `CreateAddressSeed` and records `act.ProposalConfirmed`, which carries the
same `act.SeedScope` payload a declaration carries. The act is recorded. Only the reader was narrow.

## 3. The false statement

Confirm an address proposal. The `Exposure` counts move between two page loads, and the panel headed
*Recent address scopes* reports that no seed declaration in the window named an address scope —
although the confirmation just declared one.

The withdrawal case is sharper.
[ADR-1895](./1895-a-vantage-class-reads-present-configuration-so-a-historic-observation-carries-none.md)
§5 says declaring or withdrawing an address `Seed` is an operator act, and withdrawal moves the
reading in the direction `CONTEXT.md` names as the hazard: a vantage that read `internal` re-reads
`internet`, and observations move onto the leg that alerts.

## 4. Exclusions belong in the panel

They move `covered` through
[ADR-0133](./0133-an-address-exclusion-is-a-limb-of-the-custody-derivation.md) §4's subtraction,
which is the predicate this panel exists to explain. `ExclusionRef` carries a `Kind`, so the
address-kind acts are separable without re-parsing, and the name-kind ones stay out.

Where the heading no longer fits what the panel reads, the heading moves. Narrowing the read to fit
a heading is the tail wagging the dog.

## 5. The rejected alternative

**Keep the one class and repair only the empty-state copy**, so it claims no more than it read. It
costs no new query and needs no ADR.

It is rejected because it fixes the sentence and not the panel. An operator whose confirmed proposal
moved the figure would read an accurate statement that no seed declaration occurred, and would go
looking for an estate change that was their own edit. The panel would be honest and would no longer
do the job ADR-1946 §3 built it for.

## 6. What this ADR does not rule

**This ADR rules the read alone.** The row shapes, the heading and the empty-state copy are
implementation, and #2072's recorded ruling settles them against this read.

That ruling holds that each class renders its own verb, because a withdrawal and a declaration are
different facts, and because `ProposalConfirmed` and `SeedWithdrawn` both carry a `SeedScope` — so
one shared row would flatten them. The reasoning is recorded there and is not decided here.

Nothing in ADR-1946 moves except the class list. Its no-`Message` ruling, its seven-day and
five-row bound, its newest-first order, its audit-tab link and the reasoning for each all stand.

## 7. Consequences

Reversal returns a panel that reports no address-scope edit while one has just moved the figure
beside it.

## 8. How ADR-1946 is amended

[ADR-1644](./1644-for-an-adr-target-a-withdrawal-is-a-tool-written-marker-not-a-hand-edited-sentence.md)
rules that for an ADR target a withdrawal is a **tool-written marker, never a hand-edited sentence**,
and that the unit is the numbered clause. It coarsens ADR-0058's sentence rule for an ADR target.

So this ADR declares one relation, `{kind: amends, adr: 1946, clause: "3"}`, and the tool renders the
marker under ADR-1946 §3. **ADR-1946's own prose is not edited at all** — no strikethrough, and no
hand-written note. A hand-written marker fails by ADR-1644's own terms.

## 9. Proof

`{ticket: 2072}`. #2072 is open and names this rule.
