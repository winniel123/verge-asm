# ADR-0017: `Exposure` needs both legs, and a one-legged reading is not a state

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#45 Which of the four unnamed Exposure cells need names, and what does the board render for the rest?](https://github.com/winniel123/verge-asm/issues/45)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0010](./0010-exposure-composes-two-reaches.md) composed `Exposure` from two `Reach`es, one
per `Vantage class`, and projected the composition onto [#14](https://github.com/winniel123/verge-asm/issues/14)'s
five state names. It drew the projection as a 3×3 — two `Reach` values plus *leg absent* on each
axis — and left four cells unnamed:

| internal ↓ · internet → | `reached` | `not-reached` | *(leg absent)* |
| --- | --- | --- | --- |
| **`reached`** | `exposed` | `firewalled` | `internal-only` |
| **`not-reached`** | `edge-only` | `unreachable` | *unnamed* |
| ***(leg absent)*** | *unnamed* | *unnamed* | no `Exposure` |

ADR-0010 recorded that one of the holes is dangerous: internal `not-reached` with no internet leg
currently reads `unreachable` — *"no vantage reaches it"* — asserted while nobody looked from
outside. It handed the naming to this ticket.

Two things established after ADR-0010 change what the third row and column even are.

**[ADR-0014](./0014-only-revealed-generalises.md) separated *no timeline* from `Gap`.** A `Gap` is
a span holding no value; a timeline that never existed holds nothing at all, and the two are
different objects. ADR-0010's third row and column were labelled *(`Gap`)*, and its Consequences
say `firewalled` versus `internal-only` is *"precisely value versus `Gap`"*. Under ADR-0014 that
is not right: §5 of the same ADR says the never-configured case has **no timeline**, and only the
went-silent case is a `Gap`. So the third row and column were carrying two different absences
under one header, and `internal-only` sits on the *no timeline* one.

**A `Gap` on a leg is not a cell.** ADR-0010 §5 already ruled that where a leg's timeline existed
and went silent, `Exposure` itself opens a `Gap`. A `Gap` is the absence of a value, so there is
no `Exposure` value to project and nothing to put in a cell. That empties the went-silent half of
the third row and column before this ADR touches them, leaving only *no timeline* — which is a
fact about which vantage classes the operator runs, not about the `Service`.

## Decision

**`Exposure` exists only where both legs hold a value. Its projection is the 2×2 of two valued
legs — four states — and a one-legged reading is not an `Exposure` at all.**

**1. The enum loses a value. `internal-only` is withdrawn.** The four states are `exposed`
(both `reached`), `edge-only` (internet `reached`, internal `not-reached`), `firewalled` (internet
`not-reached`, internal `reached`) and `unreachable` (both `not-reached`). Every one is a verdict
about the `Service` drawn from two measurements.

**2. None of the four unnamed cells is named, because they are not cells of `Exposure`.** They
were cells of a 3×3 whose third row and column are not `Reach` values. What was in them is
rendered as the surviving leg's `Reach`, under a statement naming the conclusion that is
unavailable and why.

**3. `unreachable` does not split.** The distinction the split would have drawn — *we measured
nothing from outside* versus *we never looked from outside* — is drawn one level up, by whether
an `Exposure` exists at all. `unreachable` is left meaning exactly what it says: both legs
measured, neither reached.

**4. The two absences keep their two statements.** Where a vantage class was **never configured**,
the surviving leg's `Reach` renders and the statement is *we never looked* — nothing is broken and
the inventory is complete for the class that runs. Where a configured leg **went silent**, the
`Exposure` timeline holds a `Gap`, the surviving leg's `Reach` still renders, and the statement is
*we stopped looking*. [#32](https://github.com/winniel123/verge-asm/issues/32) required the
`Subjects` screen never to render these two alike; they now have one shape and two statements.

**5. The board groups both axes by the internet leg.** With four states the transition matrix is a
2×2 of 2×2s: the outer square is the internet leg's transition, the inner one the internal leg's.
The internet leg's boundary is a single structural rule across both axes, and **the flagship
escalation is the block on one side of it**, not a cell.

## Consequences

- **ADR-0010's own consequence is discharged rather than restated.** It found `firewalled` versus
  `internal-only` to be *"one state and a hole, named twice"* and said the mechanism now has a
  name — but left the second name in the enum, so the hole stayed named. Withdrawing
  `internal-only` is that finding carried out. The correction ADR-0010 could not make is that the
  hole is *no timeline* and not a `Gap`; ADR-0014 supplied the distinction two tickets later.
- **The dangerous cell is removed rather than patched.** Internal `not-reached` with no internet
  leg no longer reads `unreachable`, because it no longer reads as an `Exposure`. The fix is at
  the class of defect — no one-legged reading gets a state name — rather than at the instance, so
  a sixth name cannot re-introduce it.
- **The state axis stops being movable by our own aperture.** A leg's timeline **opens** when a
  slower tier first covers that port — an *opening* under ADR-0014, recorded, unnamed and
  unalerted, explicitly not `revealed`, because neither the world nor our aperture moved. With
  one-legged names on the axis those openings are ordinary cell moves, indistinguishable from
  escalations, which is the failure [#28](https://github.com/winniel123/verge-asm/issues/28)'s
  `not comparable` band exists to prevent, re-entering through the state axis. With no one-legged
  names there is no cell for an opening to move into: the `Exposure` timeline simply opens.
- **The first vantage of a class `Break`s every `Exposure` timeline, and that is the aperture
  cause.** Adding an internet vantage to a running install does not move a `Derivation` version —
  the derivation is unchanged — but it widens what the composition covers, which is
  [ADR-0007](./0007-drift-is-a-timeline-of-spans.md)'s second `Break` cause. Nothing crosses it,
  so no `Transition` is emitted and the estate does not appear to escalate overnight; the census
  still renders, per [ADR-0008](./0008-derivation-versions-move-on-content.md). Vantage class
  therefore joins the enumerated aperture inputs, which ADR-0014 licensed as *recognised rather
  than argued*.

  **Amended by [#58](https://github.com/winniel123/verge-asm/issues/58)
  ([ADR-0029](./0029-an-alert-fires-on-a-leg.md)): the `Break` half of this consequence is
  withdrawn.** This ADR's own Decision 1 says `Exposure` exists only where both legs hold a value,
  so on an install that has never run a vantage of the arriving class there are **no `Exposure`
  timelines to break** — the first vantage of a class **opens** them. A `Break` is an edge between
  two spans, so on an opening it is vacuous, which is
  [ADR-0014](./0014-only-revealed-generalises.md)'s rule and the same reasoning this ADR applied to
  its own value-space narrowing two consequences below. The case does not arise on a returning
  vantage either: a vantage of a class that already exists is no aperture change at all, and its
  `Exposure` timelines hold a `Gap` that closes by the ordinary mechanism. What survives unchanged
  is the part that carries the protection — **`Vantage class` remains an enumerated aperture
  input**, so the widening is detected, yields `revealed` on the `Reach` and `Exposure` timelines it
  opens, and fires one coverage-class message under *we changed how we look*. The estate still does
  not appear to escalate overnight, and it never did so because of the `Break`: the third
  consequence above already states the real mechanism — with no one-legged names there is no cell
  for an opening to move into.
- **A precondition panel and a board may co-exist.** #28's third density is *"the `precond`
  treatment. No board at all"*, which is right where the precondition removes every value and
  wrong where it removes only the composition. Rendering the ink panel alone would empty the board
  #14 deliberately built the no-prober install to have, and would report less than we measured. So
  the density splits by what the precondition costs: **the value is gone** → panel, no board;
  **only the composition is gone** → ink bar plus the surviving leg's board.
- **`Reach` gains an operator-facing rendering it did not have.** ADR-0010 made it a named
  `Derivation` leaf so a rule could compose one leg; this ADR reads it as the object a one-legged
  install's board is built over. There was no such object before ADR-0010, which is why #14's
  promise of *"a complete, honest internal reachability inventory"* and #10's *"worthless with one
  vantage"* could not both be honoured. Under #14's own noun they both are.
- **No rule is affected.** ADR-0010 already made every rule read a leg, and
  `sensitive-port-reached-from-internet` is the only v1 signal that touches `Exposure` at all. Its
  predicate is the internet `Reach`, which this ADR does not move: it is `not-evaluable` where
  that leg has no timeline or holds a `Gap`, exactly as before. This is ADR-0010's *"re-labelling
  which cell counts as what is a presentation change"* collecting on its promise — the enum
  shrinks and the signal estate does not break.
- **The value-space change is priced and the price is vacuous.** Narrowing a Derived value's
  projection is output-affecting for every service that read `internal-only`, so the `Exposure`
  derivation version moves and every `Exposure` timeline takes a `Break`. Nothing has shipped, so
  there are no timelines: this is [ADR-0009](./0009-verge-core-is-a-union.md)'s pre-release
  exemption **refused as vacuous rather than waived** — no exemption is claimed, and none is
  needed, because `Break` is an edge between two spans and there are none.
- **The `Subjects` screen's population list is corrected, not lengthened.** #32 told it that where
  no internet vantage was ever configured *"the subject holds an honest `internal-only`"*. It
  holds an honest internal `Reach` and no `Exposure`. The population is the same and the count is
  unchanged; what it holds is not.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Name all four holes** — an eight-value enum | Four of the eight would be facts about which vantage classes had a timeline over that port, sharing an axis with four facts about the `Service`. Rendered, 67 of the board's moves were leg-timeline **openings** — ours, not the estate's — drawn identically to the escalations beside them. The traffic is also one-way, since a leg that stops ages into a `Gap` and never back into a one-legged name, so four columns can be entered and never left. An axis whose values are not the same kind of thing |
| **Name only the dangerous cell** — a six-value enum, `unreachable` split | Fixes the instance and leaves the class. It also forces the asymmetry the ticket flagged: `firewalled`/`internal-only` and `unreachable`/*new name* would draw one distinction with two ad-hoc pairs, and naming them symmetrically means renaming `firewalled` — the most-quoted string on the map — to buy consistency between two names that should not exist |
| **Fold, as today** — `unreachable` absorbs the one-legged reading | The defect this ticket opened on. Rendered, the drill-down has to print `no timeline` in a column headed *Internet Reach*, in the same table as a measured `not-reached`, under a cell that already claimed they were the same |
| **Render the one-legged column as a `Gap`** | Re-invents the confusion ADR-0014 removed. A never-configured class has no timeline, and a `Gap` is a span. It would also route the modal no-prober install into the coverage class permanently, which is *we stopped looking* said about something we never started |
| **Keep `internal-only` and name nothing else** | The minimal move, and it keeps one one-legged reading on the state axis for no reason except that it was named first. Every argument against the other three names is an argument against this one; it survives only by seniority |
| **Two hero cells** — promote `firewalled` → `exposed` and `firewalled` → `edge-only` identically | Answers ADR-0010's complaint and leaves the operator adding two scattered numbers to get the headline. It also misses `unreachable` → `exposed` and `unreachable` → `edge-only`, which are the same leg move on a service that was closed internally too. Grouping the axes makes the count exhaustive instead of enumerated, which is [ADR-0009](./0009-verge-core-is-a-union.md)'s definition-over-list move at the presentation layer |
