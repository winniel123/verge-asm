---
number: 2171
title: "Declining an address proposal moves the covered predicate, so the act reader covers seven classes"
slug: declining-an-address-proposal-moves-the-covered-predicate-so-the-act-reader-covers-seven-classes
date: 2026-09-16
status: accepted
source: grilling
ticket: 2171
proof: {ticket: 2169}
relations:
  - {kind: amends, adr: 2114, clause: "2"}
  - {kind: rests-on, adr: 133}
---

# ADR-2171: Declining an address proposal moves the covered predicate, so the act reader covers seven classes

## Decision

> **Declining an address proposal moves the covered predicate, so the `Exposure` act reader covers
> seven classes.** `proposal.declined` and `proposal.decline.undone` join the five ADR-2114 names.
> The seven-day window, the five-row cap and the newest-first order are unchanged, and each class
> renders its own verb.
>
> A reader asks why ADR-2114's principle says every act class that moves `addressScopeCovered`, and
> its table lists five. The enumeration is wrong, not the principle.
>
> Rejected: narrowing the principle to an operator scope edit. That leaves an operator looking at a
> moved figure beside a panel reporting no address-scope edit, which is the defect ADR-2114 removed.
>
> Reversal returns a panel that reports no edit while a decline has just moved the figure.

## 1. Context

[ADR-2114](./2114-the-address-scope-act-reader-covers-every-act-class-that-moves-the-covered-predicate.md)
rules that the `Exposure` screen's address-scope act reader covers **every act class that moves
`addressScopeCovered`**. Its §2 then names five, and its own heading counts them: *Five act classes
move the predicate, and the reader saw one*.

Two more move it. The principle and the enumeration disagree, and the principle is the part that was
reasoned.

## 2. The two classes, and the proof they move the predicate

| Act class | Handler | Writes | Moves `covered` |
| --- | --- | --- | --- |
| `proposal.declined` | `declineLookup`, `cmd/web/proposals.go` | `CreateDeclinedProposalExclusion` | yes, through ADR-0133 §4's subtraction |
| `proposal.decline.undone` | `undoDecline`, `cmd/web/proposals.go` | `DeleteUnclaimedAddressExclusion` | yes, through the same subtraction |

The decisive fact is in the read, not the write. `ListAddressExclusionCidrs`
(`db/queries/exclusions.sql`) selects `WHERE kind = 'address' AND address_cidr IS NOT NULL`. **There
is no `proposal_id` clause.** An exclusion a decline created is therefore indistinguishable, to the
predicate, from one an operator declared. It subtracts from `covered` in exactly the same way.

Both acts already carry `act.ExclusionRef{Kind: "address"}`, the same payload ADR-2114 §4 relies on
to separate address-kind from name-kind without re-parsing. Both already render a label:
`proposal.declined` is *Proposal declined*, `proposal.decline.undone` is *Decline lifted*. The
widening is mechanical. Only the ruling was missing.

## 3. The counter-argument in the code, and why it does not bite

`cmd/web/proposals.go` carries a comment above the undo path: *"A declined scope was never declared,
so lifting it admits no ground (ADR-0133, #1721)."*

That comment is true and this ADR does not disturb it. It answers whether declining **declares**
scope, and the answer is no: a decline records an exclusion so the same range is not proposed again,
and lifting it returns the scope to pending rather than admitting it.

But the panel is not about declaration. ADR-2114's own principle is *moves `addressScopeCovered`*,
and §2 already admits `exclusion.declared` and `exclusion.lifted` on exactly that ground, neither of
which declares a `Seed` either. An exclusion moves the predicate by subtraction, whoever wrote it.

Naming this here matters. Without it, the next reader meets a comment that looks like a contradiction
of this ADR and has to re-derive why it is not.

## 4. The rejected alternative

**Keep five, and narrow ADR-2114's principle to an operator scope edit.**

It is coherent: the panel is headed at the operator, and a decline is a proposal-queue action rather
than an estate edit.

It is rejected because it makes the panel worse at the job ADR-2114 built it for. An operator who
declines an address proposal moves the `Exposure` figure, then reads a panel reporting that no
address-scope edit occurred in the window, and goes looking for an estate change that was their own
click. That is ADR-2114 §3's defect exactly, surviving in two classes it did not name.

It would also require editing ADR-2114's principle sentence, which is the part that was right.

## 5. Sequencing

**#2167 lands before this widening.** The panel already issues five `ListActsOfClassSince` reads per
`/exposure` render, and #2073 measured that `act` carries one index, `act_created_at_idx`, with none
on `action`. Going to seven compounds a measured cost.

#2167 replaces the five reads with one `action = ANY(...)` read, which also restores a single cap
bound over the merged set. This ADR may land first. Its implementation should not.

## 6. What this ADR does not rule

**The five-row cap's semantics.** #2188 already ruled that `capped` reads from each query returning
its own `LIMIT` rather than from a filled render. Seven reads do not change that.

**The index.** Whether `act` earns a composite `(action, created_at DESC, id DESC)` is #2073, and it
is a cost question rather than a correctness one.

**Name-kind exclusions.** They stay out, as ADR-2114 §4 rules. `ExclusionRef` carries a `Kind`, and
only the address kind reaches `addressScopeCovered`.

**Whether any further class moves the predicate.** This ADR names two because two were measured. The
principle, not this list, is what governs the next one found.

## 7. Consequences

An operator who declines or un-declines an address proposal sees the act that moved the figure.

The panel's heading and copy already moved with the read when #2072 widened it to five classes.
Seven classes need no further move.

Reversal returns a panel that reports no address-scope edit while a decline has just moved the
figure beside it.

## 8. How ADR-2114 §2 is amended

ADR-2114 numbers its headings. §2 is *Five act classes move the predicate, and the reader saw one*,
and it holds the enumeration table. Its heading carries the false count, which is what makes it the
clause this ADR reaches rather than any other.

So this ADR declares `{kind: amends, adr: 2114, clause: "2"}`, and the tool writes the marker under
that heading. ADR-2114's own prose is not hand-edited.

Nothing else in ADR-2114 moves. Its principle, its seven-day window, its five-row cap, its
newest-first order, its one-verb-per-class ruling and its exclusion of the name kind all stand. This
ADR corrects the count beneath the principle.

## 9. Proof

`{ticket: 2169}`. #2169 is open and names this rule: ADR-2114 names five act classes, and
`proposal.declined` and `proposal.decline.undone` also move `addressScopeCovered`.
