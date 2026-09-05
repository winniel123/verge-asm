# ADR-0146: A `Seed` admits a `Name`, and holds it only where measurement cannot decide

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1282 ADR gaps: internal/delivery, internal/report, internal/estate](https://github.com/winniel123/verge-asm/issues/1282) — Gap 2
- **Rests on:** [ADR-0006](./0006-subjects-leave-by-measurement.md) (subjects leave by measurement; the disjunction is `Address`'s alone; the residue stays visibly unconfirmed), [ADR-0007](./0007-drift-is-a-timeline-of-spans.md) (`authority` governs **admission** and is *"**not** an ordering"*)
- **Bounded by:** [ADR-0080](./0080-a-vantage-composition-is-cross-class-or-class-scoped-and-only-one-takes-a-quantifier.md) (the composition this ruling reads is cross-class), [ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md) (`wildcard-discrimination` is in the vector, so `Shadowed` is a membership-deciding outcome), [ADR-0087](./0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md) (the closure vocabulary is closed at three grounds and states no precedence among them)
- **Bounded away from:** #1282's Gap 1, the `delivery.Doer` seam, which is deduped into
  [#1272](https://github.com/winniel123/verge-asm/issues/1272) §2 and ruled elsewhere. The
  pure-core / impure-runner file split inside `internal/delivery` is package taste and is ruled
  nowhere

## Context

`internal/estate/membership.go` carried an uncited assertion — *"A Seed-covered Name is Declared
and in the estate regardless of resolution"* — and the code enacted it at two sites.

`estate.Membership` runs the withdrawal predicate over every observed `Name`, and then re-adds
**every** Seed-covered name to the present set afterwards. The second loop cannot lose: whatever
the first decided, a covering `Seed` puts the name back.

`queue.decideNameDeparture` does the same thing in the fold. It returns *stays* on `seedCovered`
**before** it reads the absence predicate at all, under a comment asserting an ordering: *"The
operator's narrowing outranks declared input, which outranks measurement (ADR-0087)."*

**ADR-0087 states no ordering.** It closes the closure-reason vocabulary at three grounds —
`measured-absent`, `uncited`, `descoped` — and sorts them by *the ground the closure rests on*.
Ground is not rank. The citation supported a claim its target does not make.

**ADR-0007 refuses this exact ordering, by name.** Its Alternatives-rejected table carries the row
*"`authority` as a precedence ordering | Would let a zone file keep a dead name alive, contradicting
ADR-0006"*, and its body says what `authority` does instead: it is *"**not** an ordering: if it
were, `declared` would outrank `measured` and a zone file would keep a dead name alive… What it
governs is **admission** — whose word is enough to put a subject in the estate at all."*

A `Seed` is the operator's zone file by another route. The code re-adopted the rejected ordering
and cited, for it, an ADR about something else.

**ADR-0006 grants the disjunctive rule to `Address` and states why it cannot travel.** `Address` is
*"alone among the four subjects"* in having no lifecycle of its own, *"because nothing ever observes
an address's existence"*, and so is in the estate *"exactly while a current resolution cites it or a
`Seed` covers it."* A `Name` has a lifecycle. Our own resolver is `enumerable` over one `Name` and a
Name Error is a complete answer about it. **That asymmetry is the whole rationale for the
disjunction, and it does not hold for a `Name`.**

`CONTEXT.md`'s `Name` entry already described the behaviour this ADR rules, and never the behaviour
the code had: a `Name` *"leaves when our own resolver measures a Name Error on a cross-class `Vantage
composition`"*, and under `Shadowed` or beneath a `Lame` delegation *"it cannot leave at all"*. It
names no `Seed` carve-out anywhere. **The documents were right and the code was wrong**, which is why
nothing below is a supersession.

## Decision

> **A `Seed` ADMITS a `Name` into the estate. It HOLDS that `Name` only where measurement CANNOT
> decide — the undecided outcomes `Shadowed`, `Gap` and `Lame`, and the no-witness case. Where a
> cross-class composition decides an absence out of decided outcomes alone, the `Name` LEAVES with
> the reason `measured-absent`, Seed coverage notwithstanding.**

### 1. Admission and persistence are two questions, and only the first is the `Seed`'s

| Question | Who answers it |
| --- | --- |
| **Is this `Name` a subject at all?** | The `Seed`. That is ADR-0007's `authority`, and it is why a covering `Seed` puts a never-yet-resolved `Name` in the estate before any measurement exists |
| **Is this `Name` still there?** | The cross-class composition over `resolution`. That is ADR-0006's founding sentence, and a `Seed` is not a witness to it |

The old code answered both with the `Seed`, which collapses the two into one and makes the
declaration a permanent veto over its own measurements. **The split is the ruling.** A `Seed` keeps
its full force at the moment a subject enters, and no force at the moment measurement concludes.

### 2. What "cannot decide" means, outcome by outcome

The `resolution` value space is `Resolved`, `NoData`, `NameError`, `Lame`, `Shadowed`, `Gap`.

| Outcome | Decides? | What the `Seed` limb does |
| --- | --- | --- |
| `Resolved` | Decides **present** | Nothing. The name stays on its own evidence |
| `NoData` | Decides. The name exists and holds no record of this type | Nothing |
| `NameError` | **Decides absent.** The complete answer ADR-0006's *removal is an observed value* rests on | Nothing. The name leaves |
| `Shadowed` | **Cannot decide.** A wildcard's fiction is indistinguishable from a name ([#192](https://github.com/winniel123/verge-asm/issues/192)) | **Holds.** This is the limb's real work |
| `Lame` | **Cannot decide.** The delegated authorities were reached and do not serve it, so no Name Error can arrive | **Holds** |
| `Gap` | **Cannot decide.** We stopped being able to look | **Holds** |
| *no witness* | Nothing has measured this `Name` yet | **Holds.** This is admission, not persistence |

**Two of those three held outcomes are already held by the predicate**, and this ADR says so rather
than implying the limb carries them. `Lame` and `Gap` do not suppress, so a class witnessing either
already blocks a cross-class withdrawal for **every** `Name`, Seed-covered or not. Naming them here
costs nothing and states the *rule* instead of today's predicate: were the suppressing set ever
widened, the `Seed` limb must still hold through them.

`Shadowed` is the one outcome that both suppresses and cannot decide, so it is where the two
predicates come apart, and it is the case the new test pins.

### 3. The honest-unconfirmed residue is preserved, and it is the point

ADR-0006 named a residue it could not measure — names below a wildcard, and later names beneath a
lame delegation — and ruled that it *"stays in the estate, visibly unconfirmed, and leaves by one of
two honest routes: the operator supplies coverage, or the operator declares it out of scope."*

**That residue is exactly the set §2 holds.** Stripping the `Seed` limb outright would evict a
shadowed `Name` the operator declared, which is the estate silently shrinking on an answer that says
*we cannot see here* — the outcome ADR-0006 spent its whole argument refusing. The limb survives
because the residue does.

### 4. The exclusion limb is untouched

`descoped` still fires ahead of everything, and that is not a precedence ordering. An exclusion is
an act on **our aperture**, not a claim about the world (ADR-0087), and ADR-0006's Consequences make
excluding a still-resolving name legal on purpose: *"the operator is saying **not mine**, not **not
there**."* Nothing in this ruling touches it, and nothing here touches `openedByAperture`, which
attributes an appearance rather than deciding a departure.

### 5. What this changes in code

- `internal/estate/membership.go` — the Seed loop admits, and skips a name whose cross-class
  witnesses decided its absence.
- `internal/queue/membership.go` — `decideNameDeparture` reads the absence predicate **before** the
  `Seed` limb, and the limb narrows to the undecided case.
- The predicate the `Seed` limb yields to is new and lives beside `WithdrawnCrossClass` in
  `internal/estate/withdrawal.go`, so both membership sites read one computation — ADR-0080's rule
  that the drift engine does not re-derive this answer.
- The ordering comment at the fold is **deleted rather than re-cited.** The ordering it asserted no
  longer exists, and ADR-0087 never carried it.

## Consequences

- **`CONTEXT.md`'s `Name` entry gains the carve-out**, stated as the `Name`'s two limbs beside the
  `Address` entry's two. The entry did not read *wrong* alone in the present tense — it read
  **silent** on the `Seed`, and the code read that silence as licence.
- **ADR-0006 is amended nowhere.** Its disjunction is written as `Address`'s alone, with the
  no-lifecycle reason attached, and read in isolation it licenses nothing for a `Name`. There is no
  superseded sentence, so [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s
  obligation does not fall due there.
- **ADR-0007 is amended nowhere**, for the stronger reason: its `authority` section and its
  rejected-alternative row already **are** this ruling. This ADR applies them to a subject they
  never named, and its own body is the pointer a reader arriving at the code now finds.
- **`cmd/web/subjects.go`'s `suppressesNameMembership` is unchanged.** It renders one stored outcome
  on one subject page and has never consulted seed coverage, so it was never on the wrong side of
  this.
- **This is a behaviour change with a visible edge.** A Seed-covered `Name` reading `NameError` from
  every class was previously immortal and now closes its timelines with `measured-absent`. On an
  install holding such a name, the first fold after this lands fires one membership message per
  name. That is a **correction of the record**, not a widening of the membership vector, so
  ADR-0086's rider applies and no golden corpus is re-escrowed by it.
- **The estate can now shrink below its declaration**, and that is intended: a `Seed` says *this is
  mine to watch*, never *this exists*.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Ratify the unconditional override** — write the ADR the comment implies and keep the code | It re-adopts the precedence ordering [ADR-0007](./0007-drift-is-a-timeline-of-spans.md) rejected in its own Alternatives table, *"would let a zone file keep a dead name alive"*, and it defeats ADR-0006's founding sentence with a declaration rather than an argument. An estate that cannot shrink below its `Seed` set is an append-only inventory over that set, *"which is half a drift product"* |
| **Strip the `Seed` limb entirely** — a `Name` is decided by measurement alone | Loses the residue. A shadowed or lame-delegated `Name` would be evicted on an outcome that says *we cannot see here*, turning ADR-0006's *visibly unconfirmed* into *gone*, and it also evicts a declared `Name` that nothing has resolved **yet** — which is the admission half, and `authority`'s actual job |
| **Extend `Address`'s disjunction to `Name`** | ADR-0006 gives the disjunction one reason — an `Address` has no lifecycle of its own — and a `Name` has one. Copying the conclusion without the premise is how the code got here |
| **Keep the override and give the operator a dial for it** | A dial that silently makes the operator's whole board non-comparable, which is ADR-0005's own argument, and ADR-0007 already refused an operator-configurable term inside the comparison path |
| **Stop `Shadowed` suppressing**, then keep the unconditional limb | Reverses [ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md) and #192, which put `wildcard-discrimination` in the membership vector precisely so `Shadowed` decides *"as affirmatively as NameError"* for a name **nothing declared**. The undecided question is about the `Seed` limb, not about the leaf |
| **A new lifecycle state — `declared-unconfirmed` — between present and gone** | ADR-0006 refuses exactly this: an invented state needs an invented transition, *"a threshold we chose sitting inside the comparison path"*. The residue is already representable as present with an undecided value |
| **Rule it as an amendment on [ADR-0006](./0006-subjects-leave-by-measurement.md)** | Under [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s split an amendment carries a claim about the world that has changed. Nothing about the world changed; the code diverged from two standing ADRs. That is a new ruling on a subject neither named, not a correction to either |
| **Fix the code and cite ADR-0007 from the fold** | ADR-0007 rules on `authority` and timeline keying, and says nothing about a `Seed`'s two roles over a `Name`'s membership. The next reader would arrive at the same silence that produced the override |
