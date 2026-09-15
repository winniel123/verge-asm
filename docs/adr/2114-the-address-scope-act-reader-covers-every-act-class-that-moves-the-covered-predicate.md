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
---

# ADR-2114: The address-scope act reader covers every act class that moves the covered predicate

## Decision

> **The `Exposure` screen's address-scope act reader covers every act class that moves
> `addressScopeCovered`**: `seed.declared`, `seed.withdrawn`, `proposal.confirmed`, and the
> address-kind `exclusion.declared` and `exclusion.lifted`. The seven-day window, the five-row cap
> and the newest-first order are unchanged.
>
> Each class renders its own verb. A withdrawal and a declaration are different facts, and
> `ProposalConfirmed` and `SeedWithdrawn` both carry a `SeedScope`, so one shared row would flatten
> them.
>
> A reader asks why a panel beside the `Exposure` figure lists one of the four causes that move it.
> The panel's empty state is a statement, and a withdrawal or a confirmed proposal makes it false.
>
> Rejected: narrowing the panel's copy instead, which leaves it honest and useless.

## 1. Context

[ADR-1946](./1946-an-address-scope-edit-reaches-the-operator-through-an-act-reader-never-a-message.md)
§3 specified the reader precisely: `seed.declared` acts, over seven days, five rows, newest first,
linked to the audit tab. #1939 implemented exactly that in `cmd/web/scopeacts.go` and did not widen
past it, because the panel's read was the ADR's own text.

The panel exists to explain a movement in the `Exposure` figure beside it. That is what makes the
one-class read a defect rather than a narrow scope.

## 2. Four acts move the predicate, and the reader saw one

`addressScopeCovered` (`cmd/web/vantageclass.go`) decides every `Vantage class`, and therefore which
leg of `Exposure` an observation lands on. It reads the live address `Seed` set minus the live
address `exclusion` set.

| Act class | Handler | Moves `covered` | Read before this ADR |
| --- | --- | --- | --- |
| `seed.declared` | `declareSeed`, `cmd/web/seeds.go` | yes | yes |
| `seed.withdrawn` | `deleteSeed`, `cmd/web/seeds.go` | yes | no |
| `proposal.confirmed` | `confirmProposal`, `cmd/web/proposals.go` | yes | no |
| `exclusion.declared`, `exclusion.lifted` | `cmd/web/exclusions.go` | yes, through ADR-0133 §4's subtraction | no |

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

## 6. Consequences

Reversal returns a panel that reports no address-scope edit while one has just moved the figure
beside it.

The row shapes, the heading and the empty-state copy are implementation, settled on #2072 against
this ADR. This ADR rules the read alone.

## 7. Proof

`{ticket: 2072}`. #2072 is open and names this rule.
