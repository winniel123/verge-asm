# ADR-0072: Absence is a property of a cell, not of a row — and `withdrawn` is the only population

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#122 The Subjects screen: which populations render alongside the living?](https://github.com/winniel123/verge-asm/issues/122)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[#10](https://github.com/winniel123/verge-asm/issues/10) made *what do I own* the second-position
screen, deliberately: the landing view is the exposure board and `Subjects` is the lookup the board
refuses to be. Current membership is the **open `Span`**, so *what is here* is cheap.

What was never decided is **which populations render alongside the living**. The map's fog patch
handed the screen six of them, read off one record since every `Gap` states its cause
([#42](https://github.com/winniel123/verge-asm/issues/42) /
[ADR-0014](./0014-only-revealed-generalises.md)):

> withdrawn · `Shadowed` · *we never looked* · *we stopped looking* · *you stopped answering* ·
> a closed custody gate

and a second question: **which of them can be listed without a comparison at all.**

Six things had accumulated around this screen without composing.

- [ADR-0006](./0006-subjects-leave-by-measurement.md) makes *subjects leave only by measurement* a
  headline safety property, so a departure has to be **visible** somewhere or the property is
  unbacked in the product — the same defect [#44](https://github.com/winniel123/verge-asm/issues/44)
  found behind [#21](https://github.com/winniel123/verge-asm/issues/21) §6.1.
- [ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md) established that **only
  two of the four subjects can be a root** — a `Service`'s membership is its `Address`'s restated and
  an `Endpoint`'s is two subjects' restated — and nobody had asked what that implies for a listing.
- [ADR-0017](./0017-exposure-needs-both-legs.md) ruled, one screen over, that **an axis must hold one
  kind of thing**, and killed an eight-state enum for mixing facts about our vantages with facts
  about the world.
- [#51](https://github.com/winniel123/verge-asm/issues/51) left a warning rather than a rule:
  rendering the `custody extension` here as a `custody: operator` filter is the obvious
  implementation and is **wrong**. It said the rows are right and the screen is not; it did not say
  what makes a filter legal.
- [#74](https://github.com/winniel123/verge-asm/issues/74) ruled that a rule's census member and
  `Subjects` **may not be the same object**, and added the question nobody had asked: *given both
  are lists of bare subjects, what distinguishes them?* Nobody had drawn them together.
- Two presentational obligations bind anything drawn here — every duration and count must be able to
  declare itself a **floor** ([#18](https://github.com/winniel123/verge-asm/issues/18),
  [ADR-0008](./0008-derivation-versions-move-on-content.md)), and **a value arriving after a `Gap`
  cannot be dated** (ADR-0014).

## Decision

**Absence is a property of a cell, not of a row. Five of the six populations are states of one
facet timeline on a subject that is fully alive and already in the list; `withdrawn` is the sixth,
and it is a population. It is not rendered as a listing in any ordering, and it is reached by key
alone.**

Sorted by *what the absence is an absence of*, the six are not one kind of thing:

| Population | What is absent | On the screen | Needs history? |
| --- | --- | --- | --- |
| **withdrawn** | **the subject** | **Not a row and not a cell.** Reached by key, on search | **yes** |
| **`Shadowed`** | a rule's licence to apply a predicate — the **value is present** | A living row, with a **membership** annotation: *cannot be measured absent* | no |
| ***we never looked*** | the timeline, which never existed | A cell, cause stated, `never evaluated · no prior value` | no |
| ***we stopped looking*** | a value, under a `Gap` we caused | A cell, cause stated, last value in the muted role with an as-of, **fault keying only where something failed** | no |
| ***you stopped answering*** | a value, under a `Gap` the operator caused | A cell that **cites the rule already reporting it**, never a second fault | no |
| **a closed custody gate** | eventually a value; **not yet** | During the retention window an **ordinary current value**, with `third-party` above the table rather than per row. After it, a cell | no |

Four consequences, and the fourth is the one that had no home.

**1. The two halves of the question cut in the same place.** A `Gap` is a `Span` and it records its
cause, so all five cell states are read off **current state**, exactly like the living. `withdrawn`
has no open span — its timelines are **closed** — so it is found by reading a closed span with
nothing after it. It is the only population that is not a cell **and** the only one that cannot be
listed without a comparison. That coincidence is what makes this one cut rather than two rulings.

**2. `Subjects` has two lists, not four.** Only a `Name` and an `Address` can be the root of an
entering sub-tree (ADR-0031). `Service` and `Endpoint` are reached **through** a root and are never
peers in a listing, so the header states **two counts and never sums them**. A four-kind total counts
one membership fact up to four times, and there is no question an operator asks whose answer is that
number.

**3. `withdrawn` is reachable by key and never as a list.** Search takes a key and returns at most
one subject, so it is a lookup: no ordering to choose, no length that grows, no partition to state a
denominator for. **The listing is the estate alone; the search is not.** Refusing the search half as
well would manufacture a false absence at the text box — `No results` for a subject we measured gone
reads as *we have no record*, which is a source erroring and producing an observation of absence.

**4. A filter on `Subjects` is legal exactly where its predicate is a value the row already
renders.** `Custody` is a column, so `custody: third-party` is legal and narrows to rows whose
visible cell says so. A rule's verdict is not a column and may not become one — evaluating a rule on
a screen authors its population a second time, outside the leaf that versions it and outside the
`Break` that clamps it. This is the general form of #51's warning, and it discharges it without a
special case: the `custody extension`'s population is *addresses a name inside the scope resolves to
directly*, which is not a cell on any row here, and whose honest rendering includes the names whose
chains leave the zone — rows this screen does not have at all.

## Consequences

- **The operator's durable record of a departure is the message store, not a screen.** A withdrawal
  fires a membership message at the cause, and every `Message` is written and rendered
  **unconditionally** — the store is not a `Channel`, has no configuration, cannot be disabled and
  cannot fail. Ordered by time, which is correct for a log and wrong for an inventory. The board's
  `withdrawn` column carries the comparison window; the subject's own page still renders its closed
  timelines. **Three carriers, and this ADR leans on the first**: if the message store ever acquires
  a retention horizon, the durable record goes with it and decision 3 needs re-pricing.
  [ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) ships both
  retirable corpora unbounded in v1, so the debt is named and not due.
- **`Shadowed` is not read off a `Gap`, and the fog patch's wording is corrected here.** It is a
  **value** the model holds on `resolution`; what is absent is a rule's ability to apply a predicate
  to it — #44's fourth cause, *we measured; this rule cannot read the answer*, which is a `Gap` on
  the **signal's** timeline sitting above a good value on the facet's. The record all six are read
  off is **the open span**, not the `Gap`. That is a strictly wider claim and it is what makes the
  ruling total.
- **The floor obligation lands in one column and is inconsistent across it, correctly.** *In the
  estate since* is a duration over a derivation wherever the membership timeline can `Break`. A
  `Name`'s membership composes `resolution-walk`, so it clamps and reads `≥`. A `Seed`-covered
  `Address`'s membership composes **nothing at all** — a `Seed` is Declared and carries no vector —
  so its date is exact. Two renderings in one column on one screen, and the difference is a property
  of the subject rather than of the display.
- **The undatable pair lands in exactly one place**: a cell whose `Gap` has just closed shows both
  values and **no date for the move**. It may not read as `changed 3 days ago`, and it may not be
  silently dropped, or a genuine escalation that happened while a vantage was down vanishes without
  trace. It is worded distinctly from the floor beside it so a reader does not fuse a missing *when*
  with a clamped *how long*.
- **`Subjects` acquires no signal column.** A rule's population is versioned inside its own leaf; a
  column of verdicts on a list with no version is #74's refused object arriving as a cell rather than
  as a screen. A subject's own page carries its rules.
- **No per-row control and no nav badge**, per [ADR-0002](./0002-ownership-gates-probing.md), #51 §4
  and #74 decision 7. A row's key is a **link**, which carries no state and approves nothing.
- **An address inside a declared address scope is a subject before anything has been measured**
  ([ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md)), holding `Reach` = `not-reached`, a
  **measured value**. So `Subjects` is non-empty before the first batch on an address scope and empty
  on a name scope, and a single *no data yet* state would be false on one of them.

## Alternatives rejected

| Option | Why it lost |
| --- | --- |
| **One flat list of four kinds with a `state` column** | The conventional inventory, and the thing a build session writes first. The `state` axis holds two kinds of thing — *the subject is gone* is about the estate, *we could not measure this facet* is about one of its six timelines — which is ADR-0017's own defect one screen over. It counts membership up to four times, and a subject in two absences at once forces the column to pick |
| **Absence as a filter-chip partition over one list** | The closest loss. A partition owes a denominator, and the estate can never honestly have one (#28). Refuse the denominator and the chips lie by omission; supply it and the screen asserts estate completeness. Drawn, the chip counts also **fail to partition** — a name that is living *and* gapped is under two of them |
| **A `withdrawn` listing, ordered by when it left** | A reverse-chronological feed of change: [#10](https://github.com/winniel123/verge-asm/issues/10)'s rejected variant B rebuilt on the second screen |
| **A `withdrawn` listing, ordered by the subject** | Everything that has ever left this install. Monotone for the instance's whole life, and the `Span` corpus is never compacted (ADR-0041), so nothing bounds it. A screen whose length is a function of the install's **age** — #74's defect with a worse independent variable |
| **A `withdrawn` listing, truncated** | #74's refused row threshold: a number on a screen that nothing versions |
| **A `withdrawn` listing, filtered to a recent window** | A screen-side partition of a population, which is [ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md)'s refused object with a date in it instead of a predicate |
| **No withdrawn anywhere, including search** | Manufactures a false absence at the search box. `No results` for a subject we measured gone is the failure the project refuses everywhere else, arriving through a text box |
| **A `custody: operator` filter reproducing the extension census** | #51's named trap. The rows are right and the screen is wrong: what is **not** covered is absent here by construction, so the boundary rule is invisible exactly where you would audit it |
| **A signal column on the estate list** | #74's ground, arriving as a cell rather than as a screen: the same population authored twice, once inside a leaf that `Break`s and once on a page that does not |
| **A fifth `Gap` cause for the operator's own act** | Already refused by #44 decision 3, which enumerates *our gate* inside *we stopped looking*, and by [#118](https://github.com/winniel123/verge-asm/issues/118), which declined to put a correct operator act into the same enumeration as our failures |

## Thin ground

- **That the message store is a sufficient carrier for departures is argued, not measured.** It is
  durable, unconditional and permanent in v1, and it is ordered by time — which is right for a log.
  Whether an operator finds a three-week-old membership message when they want to know what happened
  to one name is a claim about reading, not a finding. The failure mode if it is wrong is recoverable
  and local: the repair is a search that also reaches the message store by key, not a listing.
- **Decision 4's filter test is mine.** #51 refused a specific filter and gave screen-level reasons;
  no prior ticket says what makes a filter legal. *Its predicate is a value the row already renders*
  is a clean line and it produces the right answer on all three cases available, but three cases is
  not a corpus. If it collapses, it collapses toward **no filters at all**, and the cost lands on
  search alone.
