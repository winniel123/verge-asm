---
number: 1892
title: "A record of the act is not residue, so an Annotation keeps its instant"
slug: a-record-of-the-act-is-not-residue-so-an-annotation-keeps-its-instant
date: 2026-09-13
status: accepted
source: grilling
ticket: 1892
map: 1786
proof: {ticket: 1826}
relations:
  - {kind: amends, adr: 73, clause: "2"}
  - {kind: amends, adr: 73, clause: "4"}
  - {kind: amends, adr: 93}
---

# ADR-1892: A record of the act is not residue, so an Annotation keeps its instant

## Decision

**A record of the act is not residue.** ADR-0093's limb 2 tests what the act **moved** — a
measurement the estate produced, or a record of the system acting on what the act changed — and a
consumer that exists in v1 must need the date. Both halves are required.

The `Act` corpus transcribes every Declared act, with an instant. It dates no consequence, because
it is not one. Were a transcript to count, limb 2 would be false for every Declared term at once, and
the test could never fire again. `Annotation` keeps its instant.

Rejected: a derivation fence, refused because two of limb 2's own rows already rest on Operational
residue. And a reader-relative reading, which hangs a modelled field on a permissions gate.

Reversal costs ADR-0093's eleven-row table, which rests on this reading. Read literally, it strips a
modelled field the day the corpus ships.

## 1. Context

[ADR-0093](./0093-an-instant-on-a-declared-term-is-earned-by-an-act-nothing-else-dates.md) earns an
instant on a Declared term over two limbs together. Limb 1: the term is replaced rather than edited.
Limb 2 today:

> *"Nothing else in the Observed or Operational corpus may already date the act, **and** a consumer
> that exists in v1 must need the date."*

[ADR-1891](./1891-an-operator-act-is-recorded-as-a-fifth-operational-corpus-under-a-four-limb-predicate.md)
records an operator act as an `Act`, the fifth **Operational** corpus. Under its limb 1 the
declaration of an `Annotation` writes one, and that row carries the act and its instant.

So limb 2 read literally fails on `Annotation` the day the corpus ships, and the only instant the
Declared layer carries is struck by the corpus that records the act.
[`docs/spec/audit-act.md`](../spec/audit-act.md) §8 · D measures the collision and writes the repair.
[`CONTEXT.md`](../../CONTEXT.md) already carries the restated limb at its Declared-layer preamble,
and cites ADR-0093 for it. This ADR is the record that makes the two agree.

## 2. The restated limb

> **Limb 2 — nothing the act moved dates it, and a named v1 reader needs it dated**
>
> Nothing the act **moved** may already carry the instant — a measurement the estate produced, or a
> record of the system acting on what the act changed — **and** a consumer that exists in v1 must
> need the date. Both halves are required: a wish is not a reader, and
> [#127](https://github.com/winniel123/verge-asm/issues/127) §4 is the rule.
>
> **A record of the act itself is not residue.** The `Act` corpus transcribes every Declared act,
> with an instant. It dates no consequence, because it is not one. Were a transcript to count, this
> limb would be false for every Declared term at once, and the test could never fire again. The
> residue that defeats an instant is **what the act moved**.

The heading moves with the text. Limb 2's heading reads *"the act leaves no dated residue"*, and the
restatement names whose residue it is.

## 3. Why a transcript cannot be residue

Residue is a **consequence** the act left behind, and the test asks whether that consequence already
carries the date. A `Message` dates a `Seed` narrowing because the narrowing produced the message. A
`Delivery` dates a `Channel` because the channel received one. An `Act` row dates nothing of the
kind. It is the act written down.

Admit a transcript and the limb dies whole. Every Declared act acquires an `Act` on the same day, so
every row of the eleven fails limb 2 at once, and no later term can ever pass it. A test that cannot
fire is not a test. **The enumeration of corpora is what carried the defect**, and an enumeration
goes stale the next time a corpus lands. That is what just happened, and it is why the restated limb
names no corpus.

## 4. The two readings that lost

| Rejected | Why |
| --- | --- |
| **A derivation fence** — limb 2 excludes the `Act` because no derivation reads it | **Refused on a measurement.** Limb 2 reads *"Observed **or Operational**"*, and two of its own rows rest on Operational residue: a `Message` dates a `Seed` narrowing, a `Delivery` dates a `Channel`. The fence never discriminated |
| **A reader-relative reading** — what dates the act *for its named reader* | **Demoted to a supporting fact.** It measures true today: `GET /signals` is `requireLogin`, so a viewer reads the annotation date, while `audit` refuses a viewer outright. But it hangs a Declared field on a permissions gate. Open the audit tab to a viewer one day and `Annotation` loses its instant. **A settings change must not strip a modelled field** |

Neither is re-tried. Both are recorded here so a later session reads the ground rather than the
outcome.

## 5. Stripping the instant loses twice

The alternative to restating limb 2 is to let it fire and take `Annotation`'s instant away.

**ADR-0073 §2 stands unreversed.** An undated standing mute, on an object the model gives no expiry,
cannot be reviewed at all. *"A mute with no stated reason is one nobody can review later"*, and an
undated reason is not reviewable either.

**And limb 2 would become unconditionally false.** ADR-0093's reopening condition restates the limb,
so a limb that no term can pass is a condition that can never fire. The ADR would keep its table and
lose its rule.

## 6. No verdict moves, and one enumeration is repaired

**Exactly one row was ever in play.** Residue only ever **defeats**, so no Declared term gains an
instant under any reading of limb 2. The eleven-row table stands cell for cell, and `Annotation`
alone passes both limbs.

**One repair rides along, as a side effect rather than a claim.** The incumbent wording says
*"Observed or Operational"*. The custody-withdrawal row's residue is a `Gap`, a `Span` holding no
value, and **every `Span` is Derived** ([`CONTEXT.md`](../../CONTEXT.md), the layer preamble on
`Span`). The incumbent wording never covered its own table. *"What the act moved"* reaches a residue
at any layer.

**A free strengthening is declined.** A literal reading would have given every row unconditional
residue, including `Channel` — ADR-0093's own *"thinnest cell"*, whose limb-2 residue is conditional
because a `Channel` created and never delivered to has none. Under the restatement that cell stays
exactly as thin as ADR-0093 left it, with the same mitigation: every such cell also fails on a limb
that is not thin. No ticket is owed.

## 7. The three targets, and what each relation carries

Each relation is declared in the front matter above. The marker tool writes every marker
([ADR-1644](./1644-for-an-adr-target-a-withdrawal-is-a-tool-written-marker-not-a-hand-edited-sentence.md)).
None is hand-written.

| Target | What the relation carries |
| --- | --- |
| ADR-0093, whole file | Limb 2's heading; *"Nothing else in the Observed or Operational corpus may already date the act"*; *"an `Annotation` can live its entire life with no dated residue anywhere in the model"*; the reopening condition, which restates the limb. ADR-0093 numbers no heading, so the relation is whole-ADR and the marker sits under the H1 |
| ADR-0073, clause 2 | ADR-0093's own amendment block, inside §2, restates limb 2 verbatim — *"the act leaves no dated residue anywhere in the model while a named v1 reader needs it dated"*. #1787's sweep did not reach it |
| ADR-0073, clause 4 | §8 below. §4 rules the `Signals` annotation list, and not every surface an annotation's date may appear on |

**The outcome of every target stands. Only the ground moves.** ADR-0093 keeps its rule, its table and
its reopening condition. ADR-0073 §2 keeps the instant it kept.

## 8. Consequences

- **ADR-0073 §4 is narrowed at its site, and this ADR carries the narrowing rather than a second
  record.** §4 requires `Annotation`'s instant to render *"mono, absolute, uncoloured"*, because
  *"accepted 412 days ago"* is *"an expiry the operator implements by eye"*. The `Act` reader renders
  `When` as a relative stamp, so an `annotation.declared` row renders `412d` beside it and the column
  stays uniform. §4 rules the `Signals` annotation list. It does not rule every surface on which an
  annotation's date may appear. This is not a second decision about the model. It is what the first
  decision costs once the corpus renders a date, and it does not arise at all if the instant is
  stripped.
- `[thin]` **The ground is not that the hazard is absent, and this ADR must not claim that it is.**
  §4's own word is *"**particularly** with a colour that deepens"*, so the colour is aggravation and
  the bare age is the ground. **A bare age does arrive here.** Two of §4's four requirements are
  nonetheless absent — no deepening colour, and an audit log sorts by recording time rather than by
  staleness — and the datum differs, because `When` dates **the recording** and not the declaration.
  **A 1-in-61 carve-out to pre-empt a reading is worse than the reading.** This follows ADR-0073's
  own precedent, which conceded the keystroke objection in the same terms.
- **`Annotation` keeps its `declared` instant, and no migration changes.** Nothing is stored that was
  not stored, and nothing that was stored is removed.
- **The reopening condition is unchanged in force and repaired in wording.** A Declared term that is
  replaced rather than edited, and whose act moved nothing that carries the instant, is an addition
  under this rule rather than an exception to it.
- **The withdrawal record is not a Declared history.** `POST /annotations/withdraw` is auditable, and
  withdrawal removes the row, so the `Act` corpus becomes the only surviving record that a given
  annotation ever existed. This does not reopen #127 §5, which refuses **reconstructible prior
  values**. The two variants carry `<subject key> · <signal name>` and no prose. **The limit, stated:
  the day any variant carries the prose, this becomes a Declared history for the one Declared term
  that holds prose. The prose is the reopening condition, and nothing else.**

## 9. Where this is thin, stated rather than smoothed

**One sentence the SPEC drafts for ADR-0093's body is not written into it.**
[`docs/spec/audit-act.md`](../spec/audit-act.md) §8 · D.1 adds a line above ADR-0093's enumeration:
*"Every row acquires an `Act` the day that corpus ships. No verdict moves, because a transcript is
not residue."* That line is a hand edit to a target ADR, which
[`docs/spec/adr-governance.md`](../spec/adr-governance.md) §3 and §5 forbid, and which ADR-1644
exists to prevent. **The marker discharges the site and this ADR holds the sentence**, at §6 above.

The price is named. A session that greps ADR-0093's table finds a marker under the H1 and no line
beside the rows. It reads one file further. The SPEC was written before the mechanism landed, and
the mechanism wins.

**The `Act` reader's relative stamp is unbuilt, so §8's first bullet prices a rendering nobody has
seen.** The copy is `verge-asm-design`'s to confirm when the implementation map lands. A deepening
colour on that column would reopen §4's hazard on its own ground, and this ADR grants no licence
for one.

## 10. Proof

`{ticket: 1826}` — [#1826](https://github.com/winniel123/verge-asm/issues/1826), the implementation
map that builds the `Act` corpus from
[`docs/spec/audit-act.md`](../spec/audit-act.md), whose §8 · D states this rule. No test can prove it
today. No migration, no table, no handler and no copy has landed, so nothing yet exhibits a
transcript that a limb-2 test could decline to count.
