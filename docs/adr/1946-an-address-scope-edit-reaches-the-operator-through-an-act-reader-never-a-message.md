---
number: 1946
title: "An address-scope edit reaches the operator through an Act reader, never a Message"
slug: an-address-scope-edit-reaches-the-operator-through-an-act-reader-never-a-message
date: 2026-09-14
status: accepted
source: grilling
ticket: 1946
proof: {ticket: 1939}
relations:
  - {kind: amends, adr: 1895, clause: "9"}
  - {kind: rests-on, adr: 92}
  - {kind: rests-on, adr: 1891}
  - {kind: rests-on, adr: 1902}
---

# ADR-1946: An address-scope edit reaches the operator through an Act reader, never a Message

## Decision

**An address-scope edit mints no `Message`. The act reaches the operator through a bounded,
display-only `Act` reader beside the `Exposure` figure: `seed.declared` acts over seven days, five
rows, newest first, linked to the audit tab.**

A reader asks why, because the flagship figure moves between two page loads and nothing announces
it. ADR-0092 rules that the system announces what it found, never what the operator did, and no
`Message` constructor is reached outside `internal/queue`, so none can fire on the act.

The reader asserts no causal tie. None exists: the class is stored nowhere and the figure is
recomputed per read.

Rejected: a `Kind` field on the seed-scope act subject, as the exclusion reference already carries.
The `Act` corpus is never deleted, so historic rows would lack it and the fallback would survive.

Reversal reopens the four-cause enumeration and owes a message to every dial ADR-0092 left silent.

## 1. Context

[ADR-1895](./1895-a-vantage-class-reads-present-configuration-so-a-historic-observation-carries-none.md)
rules that a `Vantage class` is a reading of present configuration, and that both legs of one
comparison classify under one binding. So an address-scope edit moves the reading of the whole
history at once, and the `Exposure` flagship figure may move between two page loads with no batch
between them. ADR-1895 §6 states that cost rather than smoothing it.

ADR-1895 §9 then names what is left open: the existing widening message does not step in, and the
case is *"unaddressed rather than handled"*.
[#1939](https://github.com/winniel123/verge-asm/issues/1939) carried that gap into a ticket and asks
four questions. Two of them are decisions rather than implementation: is the operator told at all,
and what carries the telling. This ADR answers both.

## 2. No `Message` can fire on the act, by construction

Three facts hold, and each is independent of the other two.

**The rule.**
[ADR-0092](./0092-an-operator-dials-movement-is-not-a-cause-and-an-annotation-never-lapses.md) rules
that a `Message` is one firing of one cause, and that an operator dial's movement is none of the
four causes
([ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md) §1 fixes the
four). Declaring or withdrawing an address `Seed` is an operator act, so the same rule reaches it.
ADR-1895 §5 already applied it to this exact act.

**No fold runs.** A scope edit runs no batch. `produceMessages` in `internal/queue/produce.go` is
reached from one call site, `Worker.produce` in `internal/queue/worker.go`, inside the fold that
follows a completed batch. So at the instant of the act there is no fold for a message to come out
of.

**No constructor is reachable from the act's path.** Sixteen functions return a `*Message`, and all
sixteen live in `internal/message`. Fifteen are called from `internal/queue`. The sixteenth,
`Threshold` in `internal/message/render.go`, is called only by `ClockEdge` in
`internal/message/clockedge.go`, which is itself called from `internal/queue`. So **no `Message`
constructor is reached outside `internal/queue`**, and nothing on the web handler's write path can
mint one.

The third fact is the one that makes this a decision about the model rather than about one handler.
Minting a message for a scope edit would not be a new call. It would be the first `Message` minted
outside the fold.

## 3. The carrier is a bounded `Act` reader

The act is already recorded. `seed.declared` is an `Act` class
([ADR-1891](./1891-an-operator-act-is-recorded-as-a-fifth-operational-corpus-under-a-four-limb-predicate.md),
limb 1), written one row per scope under
[ADR-1902](./1902-one-act-row-per-subject-never-one-per-request.md). Nothing new is stored.

What is added is a reader, beside the figure the act moved:

- `seed.declared` acts, over seven days
- five rows, newest first
- a link to the audit tab, which holds the unbounded list

The bound is deliberate. The panel is a pointer at the audit trail, not a second audit trail. Seven
days and five rows keep it a glance, and the link carries any reader who wants more.

This satisfies #1939's third question without a message: the silence is no longer recorded only in
an ADR's §9, because the act is on the screen whose figure moved.

## 4. The reader asserts no causal tie

The panel says *these scopes were declared*. It does not say *this is why the figure moved*.

That restraint is not caution. No tie is available to assert. The class is stored nowhere — it is
derived wherever it is used, and `vantage.class` is vestigial (ADR-1895 §1) — and the flagship figure
is recomputed per read. There is no recorded before-value for the panel to diff against, so a causal
claim would be manufactured rather than read.

ADR-0092 gives the same answer from the other end: the system announces what it found, never what
the operator did. A panel that narrated *your edit changed this number* would be a message about an
operator act, wearing a panel's clothes.

## 5. The ADR-1891 boundary holds

ADR-1891's Decision carries one prohibition this feature runs close to: *"No derivation may read an
`Act`."*

The reader is display-only. It reads `Act` rows to render them and feeds nothing. The `Exposure`
fold reads no `Act` input, and #1939 owns a test that asserts it. The boundary is therefore stated
here rather than assumed: an `Act` may be rendered beside a derived figure, and may not be read into
one.

## 6. What this refines in ADR-1895 §9

ADR-1895 §9 names **one** suppressor of the widening message: `ReachFoldedBeforeAtVantages` finds an
earlier completed batch that folded `Reach` at a vantage of the candidate class. That is correct, and
it is the second of two.

**The first gate is earlier and fires first.** `vantageClassMessages` in
`internal/queue/vantageclass.go` shortlists with `widenedClasses(legs.cur, legs.prev)`, whose
candidate set is the classes present in `cur` and absent from `prev`. `readBatchLegs` in
`internal/queue/produce.go` derives one `covered` and applies it to both legs, which is ADR-1895 §4's
one-binding rule in the code. So a vantage the edit moved to `internal` reads `internal` on **both**
legs. The class is already in `prev`, so it is never a candidate, and the function returns before it
reads a vantage at all.

The refinement is bounded. Where a batch's candidate services carried no leg on the previous batch,
`prev` is empty and every class reads as new — the fresh-`Service` shortlist the code's own comment
names. Gate 2 decides that case. Gate 1 is what suppresses the scope-edit case, which is the case
ADR-1895 §9 was writing about.

This ADR carries that refinement under an `amends` relation scoped to ADR-1895 §9. ADR-1895's text
is not edited: an ADR never amends itself in place
(`docs/spec/adr-governance.md` §3). The marker is tool-written.

## 7. The rejected alternative

**A `Kind` field on `act.SeedScope`, as `act.ExclusionRef` already carries.** `ExclusionRef` in
`internal/act/subject.go` holds `Kind` and `Scope`, and renders `Kind + " " + Scope`. `SeedScope`
holds `Scope` alone. A `Kind` would let the reader tell an address scope from a name scope by
reading the row, rather than by re-parsing the value.

It is rejected on one ground that does not improve with time: **the `Act` corpus is never deleted.**
Every row written before the field exists would lack it. The reader would therefore keep the
fallback for the life of the corpus, and the field would buy a second code path rather than remove
the first. The payload change is paid once and the fallback is paid forever.

The reader re-applies the existing test instead. `isAddressValue` in `cmd/web/seeds.go` is the
function the declare path already uses to decide the same question about the same string.

## 8. Three near neighbours, and why none of them rules this

| ADR | Why it differs |
| --- | --- |
| [ADR-0092](./0092-an-operator-dials-movement-is-not-a-cause-and-an-annotation-never-lapses.md) | Rules that a dial's movement fires no `Message`, over an `Annotation`. This ADR **rests on** that rule and answers the question ADR-0092 never faced: what carries the act when the act moves a **number on a dashboard**. ADR-0092's act moved nothing an operator reads as a finding |
| [ADR-1891](./1891-an-operator-act-is-recorded-as-a-fifth-operational-corpus-under-a-four-limb-predicate.md) | Rules **that** an operator act is recorded, and bars a derivation from reading one. It says nothing about where an `Act` may be rendered. §5 above states the rendering side of that boundary for the first time |
| [ADR-1895](./1895-a-vantage-class-reads-present-configuration-so-a-historic-observation-carries-none.md) | Rules what a `Vantage class` **is**, and its §5 rules out a new `Message` for a scope edit. It names no carrier, and its §9 leaves the gap open in its own words. This ADR supplies the carrier and refines §9's suppressor count |

## 9. Consequences

- **No migration, no `Message` class, no cause.** The causes stay at four. Nothing is stored that
  was not stored before.
- **ADR-1895 §9 gains a tool-written marker** and its text is unchanged. The tree cites ADR-1895
  twice, both as `ADR-1895 §4`, and neither citation names a line, so the marker moves no citation.
- **The `Exposure` page gains one bounded reader.**
  [#1939](https://github.com/winniel123/verge-asm/issues/1939) owns the panel, the filtered query and
  the fold test. This ADR owns none of them.
- **`act.SeedScope` keeps its single field.** A later ticket that wants the `Kind` reopens this
  section rather than the payload.
- **The silence is now stated where a dashboard reader finds it**, which was #1939's third question.

## 10. Proof

`{ticket: 1939}`.

[#1939](https://github.com/winniel123/verge-asm/issues/1939) is open and states the rule this ADR
decides. Its title names the finding — an address-scope edit moves the `Exposure` reading and no
`Message` carries it — and its *What this issue decides* list carries the two questions answered
here: whether the operator is told at all, and what carries the telling.

A `test` proof is not available yet. The assertion this ADR would quote is the fold test that the
`Exposure` derivation reads no `Act` input, and that test lands with the panel in #1939. Naming a
test that does not exist would fail the check that resolves a `test` proof, so the ticket proof is
the honest one until #1939 merges.
