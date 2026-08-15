# Only `revealed` generalises, and a `Gap` costs the model nothing

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#42 Does revealed cover a value appearing where a Gap was?](https://github.com/winniel123/verge-asm/issues/42)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0006](./0006-subjects-leave-by-measurement.md) split appearance into `appeared` and
`returned`; [ADR-0007](./0007-drift-is-a-timeline-of-spans.md) added `revealed` for a widened
aperture and called all three *opening kinds on the membership timeline*. That framing was
already leaking. ADR-0007's own [#36](https://github.com/winniel123/verge-asm/issues/36)
amendment rules that the queried qtype set and the TLS candidate set *"start timelines that did
not exist, so both yield `revealed` rather than `appeared`"* — and those are **facet** timelines,
not membership. So `revealed` had been doing work outside the family it was declared in for two
ADRs, with the glossary saying otherwise.

[#31](https://github.com/winniel123/verge-asm/issues/31) forced the question. Its dispatch table
sits in the aperture rather than in a rule version, which is what keeps the listener rule on the
right side of [#5](https://github.com/winniel123/verge-asm/issues/5)'s fingerprinting line. Adding
a protocol row is not a subject arriving: it is a `Service` that has been in the estate for
cadences acquiring a value it never had. [#40](https://github.com/winniel123/verge-asm/issues/40)
supplied a second instance and a harder one, because it is operator-caused and can be seconds
long — withdrawing a `custody extension` opens a `Gap` beneath addresses that are still cited, and
re-enabling it puts a value back.

Get it wrong in either direction and the failure is one the model exists to prevent. Treated as an
ordinary `Transition`, **shipping a release emits drift estate-wide**. Treated as nothing, the
operator gets a value with no account of why it appeared, which is what `revealed` was invented to
supply.

## Decision

| Concern | Decision |
| --- | --- |
| `appeared` / `returned` | **Membership only** — they describe a subject |
| `revealed` | **Any timeline** — it describes our looking, which is per-timeline |
| An opening with neither cause | **Recorded, unnamed, unalerted** |
| `Gap` → value | An **ordinary adjacency**, outside the opening family |
| A fourth family member | **Refused** — the `Gap` records its own cause |
| Aperture widening that only creates timelines | **No `Break`**, per ADR-0011's strictly-additive rule |
| `Custody` closing and currency | **Independent** — the gate stops feeding the timeline; currency opens the `Gap` |
| Attribution while an aged value renders | Carried by the **`Custody` timeline**, which is already current |
| Values either side of a `Gap` | **May be shown**, labelled as undatable — never a `Transition` |
| A `Gap` closing | **Notifies**, coverage class, at the cause |
| Aperture | The **`Batch` scope record**; inputs stay enumerated, the criterion is now stated |

## Rationale

### The ticket's premise fused two events, and both are real

Take #31's dispatch table gaining a protocol row. It produces **two edges, on two timelines, from
one cause**:

- The **observation timeline** for that `(Service, facet)` did not exist. We never spoke that
  protocol there, so nothing was ever folded and there is no span of any kind — not even a `Gap`,
  because ADR-0007 established that a batch whose recorded scope excludes a thing simply *does not
  touch* that timeline. A value arriving is a timeline **opening**.
- The **signal timeline** did exist and did hold a `Gap`, because a signal evaluated where its
  evidence is absent is `not-evaluable`, which ADR-0007 lists as a `Gap` instance. That one goes
  **`Gap` → value**.

So the question *"is `Gap` → value the same thing as a timeline opening?"* has a cleaner answer
than either alternative the ticket offered: they are different events that happen to share a
cause, and #31 produces one of each. Nothing needed choosing between them.

### Only `revealed` generalises, and the reason is not a carve-out

The obvious move is to widen the whole family to any timeline. A slow cadence shows why that
over-reaches. A `Service` appears at 02:00 and is exchanged with on the daily tier; its
`tls-acceptance` timeline opens six days later, when the weekly enumeration next runs. That opening
is not the world moving — the service was there all along — and not our aperture widening, since
the aperture always included it. We simply got round to looking. A uniformly widened family has no
name for it and would have to mint one.

*Amended by [ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md): this example
originally read "TLS is attempted on the weekly tier; its `certificate` timeline opens six days
later", which put the `certificate` handshake on a `Scan` it does not ride. The handshake is a step
in the exchange that produces `reachability`, so a `certificate` timeline opens with its `Service`.
The rule this example establishes is untouched, and `tls-acceptance`'s weekly enumeration is an
instance of the identical shape — same six days, same conclusion.*

The line that holds instead: **membership is a property of a subject, aperture is a property of
looking, and looking is per-timeline.** So `appeared` and `returned` stay where ADR-0006 put them,
`revealed` generalises to any timeline, and an opening caused by neither is recorded, unnamed and
unalerted — ~~the subject's own membership transition already carried that news, at the cause~~.

> **QUALIFIED TWICE by the [#63](https://github.com/winniel123/verge-asm/issues/63) amendment
> below**, which says this clause *"was doing more work than had been checked"*. The membership
> transition carries the news **only where the root of the entering sub-tree was a `Name` or an
> `Address`** — a `Service` or an `Endpoint` entering is never a message — and what carries it is the
> **census** on that message, never the bare fact that something appeared. Read alone, the clause
> licenses relying on a message that, for a `Service` or `Endpoint` root, does not exist. Marked at
> the sentence per
> [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened
> by [#106](https://github.com/winniel123/verge-asm/issues/106).

This is a correction to ADR-0007 rather than an extension of it. The ticket's instinct that *the
family is about subjects* was two-thirds right, and the third that is not is precisely the member
that had already escaped through the #36 amendment.

The rule is total because of [ADR-0011](./0011-a-facet-is-six-parts.md)'s closed unions. A facet
timeline opens exactly when its subject enters the estate, when our aperture widens to include it,
or when we first get round to it — never because the world changed underneath an existing
timeline, since every measured negative is a **value** and not an absence. A `Service` that starts
speaking TLS does not open a `certificate` timeline; it closes a `NoTLS` span on one that already
existed.

### `Gap` → value is an ordinary adjacency, and it is always about us

A `Gap` is a span. A value arriving after it is the next span. The adjacency between them is a
`Transition` by ADR-0007's own definition, derived on read like every other, and no machinery is
missing.

What makes it safe to leave unnamed is structural rather than a case analysis: **every `Gap` →
value edge is an observer event and never a world event**, because a `Gap` exists only where *we*
could not say. So it lands in the coverage class by construction, and no rule has to sort the
outage case from the operator case from the aperture case.

The consequence worth stating, because it looks like a defect and is not: **re-opening the probing
gate produces both kinds of edge at once** — `revealed` on addresses never probed before, a `Gap`
ending on addresses that were. One operator action, two vocabularies. That is tolerable only
because ADR-0007 already fires alerting at the cause rather than per affected subject, so the
operator never sees the seam.

### No fourth member, because the `Gap` already carries the account

The ticket asked whether a fourth family member is needed. It is not, and minting one would be a
second representation of one fact — the standing seam rule, and the same argument ADR-0007 used to
refuse storing `Transition`s beside spans.

The failure mode the fourth member was meant to prevent — *a value with no account of why it
appeared* — is answered by the `Gap` sitting immediately before it, which **records its cause**.
That follows ADR-0007's closure `reason` — ~~a one-member vocabulary, `cascaded`~~ **closed at three
by [#147](https://github.com/winniel123/verge-asm/issues/147) ·
[ADR-0087](./0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md):
`measured-absent` · `uncited` · `descoped`** — and
[#22](https://github.com/winniel123/verge-asm/issues/22)'s *one treatment, stated reasons*.

The whole `Gap` → value case therefore costs the model **nothing new**, which is the right price
for something that turned out to be an ordinary adjacency wearing an unfamiliar name.

### Creating timelines writes no `Break`

ADR-0011 already states the rule — *creating values where none existed costs `revealed` or
nothing; reinterpreting an input that already had a value costs a `Break`* — gated by a CI check
that every corpus row whose output moved previously produced no observation. This ADR restates it
because it is the load-bearing half of the ticket's first failure mode, and it had only ever been
asserted inside a discussion of union variants.

Two things are worth being explicit about. A `Break` is an edge between two spans, so on an
**opening** it is vacuous — there is no predecessor to break from. And the CI check is the *whole*
guarantee: a widening that turns out to also reinterpret something is not additive, and breaks
uniformly per [ADR-0008](./0008-derivation-versions-move-on-content.md). "We only added things" is
a claim the corpus verifies, never one a release asserts.

### Aperture is the `Batch` scope record

ADR-0007 named three inputs and its #36 amendment made it five. #31 would make it six, and every
facet added after v1 makes it more. Enumerating them one ticket at a time has been working, but
nothing said what qualifies.

**Aperture is what a `Batch` records as its completed scope.** Every named input is a dimension of
that record — enabled sources, port and transport tiers, the custody gate, the queried qtype set,
the TLS candidate set — and any future dimension is an aperture input by the same test.

The list stays **enumerated** even so, and that is not redundancy. A widening is detected by
comparing a batch's recorded scope against the prior one's for that scope, which requires each
dimension to be recorded under a name. What changes is that a future dimension is now *recognised*
rather than argued about — the criterion does the work the precedent chain was doing.

This is also the ticket's generalisation past #31, discharged: adding a facet after v1 opens a
timeline on every existing subject at once, and it costs `revealed` plus one message, which is
what [#7](https://github.com/winniel123/verge-asm/issues/7)'s *adding a facet must not mean
writing a drift implementation* promised.

### Custody closing does not touch currency

ADR-0007 says an `owned` → `third-party` transition puts services into a `Gap` at once. Read
literally that is a second mechanism doing `Gap`'s job, and it makes #40's seconds-apart toggle a
real event in the model: `value → Gap → value` on every timeline beneath the address, for a
`Gap` no measurement was ever missed during.

Currency already covers it. An observation is current within `k` cadences of the covering Declared
`Scan`; the gate closing stops *feeding* the timeline, the last value ages out normally, and a
`Gap` opens by the mechanism that was already there. A toggle inside one cadence is then a
non-event in the drift model, with no damping and no threshold — which matters, because damping is
exactly what this project refuses in the comparison path and exactly what this case invites.

The cost is real and is discharged elsewhere rather than accepted. ~~On a weekly `Scan` with `k`=2,
a disclaimed address keeps rendering its last measured `exposed` for up to a fortnight.~~
**Every sensitive-list pair sits in `verge-core`, so the covering cadence is `daily` and the window
is `k`=2 cadences — TWO DAYS — since [#78](https://github.com/winniel123/verge-asm/issues/78)
retired the weekly tier.** The Consequences section below stated this correction; it is repeated
here because this is the site that *specifies* the window, and per
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) read alone
and in the present tense the struck sentence would build a fortnight-wide window. That is
not a stale attribution, because **`Custody`'s own timeline is current**: it reads `third-party`
from the instant the operator toggled, so every surface says *this is not yours* beside the aged
value. A fact we measured is being shown next to a custody value that is correct. Forcing currency
instead would have a Declared input reaching into the staleness bound, which is the second
mechanism this section removed, re-entering one level down.

### Values either side of a `Gap` may be shown, and a closing `Gap` notifies

No `Transition` crosses a `Gap` — the two values are not adjacent, so the machinery never computes
one. But a service that went `not-reached` → *(gap)* → `reached` is the product's flagship event,
and it would otherwise disappear without trace.

The distinction from [#18](https://github.com/winniel123/verge-asm/issues/18)'s refused difference
set is what makes rendering it safe. Across a `Break` we lack the **licence**: the two values were
produced by different derivations and are not the same kind of thing. Across a `Gap` both values
are the same kind of thing under one derivation, and what we lack is knowledge of **when** it
moved. So the honest rendering is the pair plus an undatable label — the same shape as ADR-0008's
labelled floor, which is already this project's form for *true, but weaker than it looks*.
Refusing it would make a `Gap` strictly more destructive than a `Break`, inverting their severity.

A `Gap` closing therefore **notifies**, in the coverage class, at the cause, and carries the pair
only where the value actually differs. *We can see again and nothing moved* is a one-line
all-clear; *we can see again and eleven services differ from what we last saw* is the real message.
Suppressing it would leave a `Gap` as the one way a genuine `firewalled` → `exposed` vanishes
silently — the alert hole ADR-0008 refused to accept across a `Break`, arriving through `Gap`
instead.

### One message class, two triggers

ADR-0011 says a strictly-additive widening ships *"with no break and one re-baseline message"*,
borrowing #18's word for a trigger #18 never defined: a re-baseline fires when a `Derivation`
**vector** moves, and an aperture widening moves no vector at all.

They are **one class with ~~two~~ THREE triggers**, named for the cause and not the mechanism. Both say *we
changed how we look*, and the operator has no use for the distinction between *our rule changed*
and *our aperture widened* — both were already coverage-class under the five-precedent messages
partition.

> **A third trigger — [#130](https://github.com/winniel123/verge-asm/issues/130) ·
> [ADR-0074](./0074-an-aperture-narrowing-that-takes-its-carrier-with-it-fires-at-the-scope.md).** An
> aperture **narrowing** says this same sentence, so it is this cause and this class. It fires only
> where the act takes the carrier with it — a `Seed` narrowing withdraws the subjects that would have
> held the `Gap`, so one message fires **at the scope**; every other narrowing leaves a subject
> holding a `Gap`, which already carries it. **The *class* here is the CAUSE-class, and its triggers
> are separate coverage-class members** — `revealed` is member 2 and the vector move member 4 — so
> this third trigger is **member ten** and the coverage class stands at **ten**. Marked at the
> sentence per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md),
> because *two triggers* read alone specifies that a narrowing has none. **Nothing about `revealed`
> moves**: no transition name is minted in the closing family, in this or any other.

The **payloads differ and should**. A vector move carries a difference set computed across a
`Break`, with all of ADR-0008's caveats about never persisting it. An aperture widening carries a
count of timelines opened and **no comparison at all**, because per the additive rule there is
nothing to compare. Writing that down stops the safer payload inheriting the riskier one's
apparatus for no reason.

> **A ~~second~~ THIRD payload under this heading, and they must not be levelled** — #130 · ADR-0074.
> An aperture **narrowing** carries a count of **subjects withdrawn**, no comparison and no rows,
> **plus the loss named**: a listener appearing in the removed ground is invisible afterwards and no
> later message recovers it. That third element is the #121 rider below arriving on a second trigger,
> for the identical reason — there is no repair, so naming it is the whole of the remedy. The copy of
> all three is [#120](https://github.com/winniel123/verge-asm/issues/120)'s.

> **A third element on the vector-move payload —
> [#121](https://github.com/winniel123/verge-asm/issues/121) ·
> [ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md).** Where the
> vector that moved includes a leaf the **membership** timeline composes — `resolution-walk`, which
> every `Name`'s and every cited `Address`'s membership reads — the payload must also **state the
> loss**: until a subject has withdrawn and returned again under the new vector, a return reads
> `appeared`. A withdrawn subject's timelines are closed, so the reopening sits across the `Break`
> with nothing legally before it, no `Transition` is derived, and
> [ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)'s membership message
> fires with the wrong word. It cannot be corrected afterwards — history is never re-derived — so
> naming it is the whole of the remedy. This is a **third element on an existing payload**, not a
> fourth trigger and not a new class.

## Amendment — [#63](https://github.com/winniel123/verge-asm/issues/63): what the membership
transition actually carried

This ADR leaves an opening caused by neither the world nor our aperture *recorded, unnamed and
unalerted*, on the ground that *"the subject's own membership transition already carried that
news, at the cause"*. That clause was doing more work than had been checked, because `appeared`
had never been ruled a message at all.

[ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md) keeps the decision
and qualifies the clause twice. The membership transition carries the news only where the **root
of the entering sub-tree** was a `Name` or an `Address` — a `Service` or an `Endpoint` entering
is never a message — and what carries it is the **census** on that message, never the bare fact
that something appeared. That matters because a newly-entered subject has no transitions at all:
its `Reach` leg opens *at* `reached` rather than moving to it, its `Exposure` opens, and its
signal timelines open, so the flagship predicate, the projection and all ten rules are
transition-shaped and match none of them. The census is the only thing that reaches the operator,
which makes it load-bearing rather than decorative.

## Amendment — [#118](https://github.com/winniel123/verge-asm/issues/118): the deferred
rendering cost, discharged

*"Custody closing does not touch currency"* above says the cost *"is real and is **discharged
elsewhere** rather than accepted"*, and names the discharge: **"every surface says *this is not
yours* beside the aged value."** No surface had drawn it. #118 draws it
([`prototypes/custody-of-nothing/`](../../prototypes/custody-of-nothing/)) and settles four things
this section left to whoever got there first.

- **The retained value is a value, not a leftover, and is rendered as one** — ordinary ground,
  ordinary weight, with the as-of stamp and the **bound as a rule** beside it (`held for 2
  cadences · 2 days on this Scan`). It is current, rules keep reading it, and the flagship keeps
  firing off it. Every treatment that implies otherwise — a countdown, the fault band, blanking at
  the toggle, striking the value through — is a way of saying *this reading no longer counts*, and
  it counts.
- **The bound is stated and the date is not.** A deadline implies an act available before it, and
  the only act here is to re-assert custody the operator does not have — which is
  [#44](https://github.com/winniel123/verge-asm/issues/44)'s refused `Enable` button arriving as a
  clock. Note the true magnitude: every sensitive-list pair sits in `verge-core`, so the covering
  cadence is **daily** and the window is **two days**, not the fortnight the weekly tier implied
  before [#78](https://github.com/winniel123/verge-asm/issues/78) retired it.
- **The custody marker's altitude follows the extent of the move.** A withdrawn extension is one
  act over the whole scope, so `third-party` is true of every row identically and belongs **above**
  the board as an ink bar, per [#74](https://github.com/winniel123/verge-asm/issues/74)'s rule that
  what is the same on every row belongs above them. A **per-row** chip is correct only where
  custody is mixed — a resolution chain leaving the zone while the extension stays on — and there
  it marks the exception rather than the rule. A constant column would be noise on every install
  that is not this one.
- **The two ends of the window are not the same screen, and the far end is not the same screen as
  an install that never opened the gate.** `never` has no subject: no timeline, no `Gap`, no row,
  and the honest carrier is the standing aperture statement on `Coverage`. `aged` has the full
  subject population holding `Gap`s that record their cause and their last reading. Drawing them
  alike would say the operator's act never happened. What makes the second bearable is a fact the
  gate supplies for free: **the population can only shrink** — nothing can observe a new `Service`
  through a shut gate, so no row can join it, and rows leave as resolutions stop citing their
  addresses.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) is amended in three entries.** `Transition` now says that
  `appeared`/`returned` are membership-only and `revealed` is any timeline; `Gap` records its cause
  and its closing edge is an ordinary transition; `Custody` states that closing the gate stops
  feeding timelines rather than opening a `Gap` directly.
- **ADR-0007's *"a third opening kind on the membership timeline"* is corrected**, and its
  *"they enter a `Gap`"* sentence for a closing probing gate is amended to go via currency.
- **ADR-0011's additive rule is now cited from two places**, and its *"one re-baseline message"*
  is pinned to the class rather than to #18's trigger.
- **[#41](https://github.com/winniel123/verge-asm/issues/41) is unblocked with its pricing
  settled**: if the observation-driven listener rule ships, its dispatch table is an aperture
  input, its arrival costs `revealed` on the observation timelines and a `Gap` → value on the
  signal timelines, and it writes no `Break`. That is a scope cost to weigh, not a correctness
  objection.
- **[#44](https://github.com/winniel123/verge-asm/issues/44) inherits two inputs it did not have**
  — a `Gap` renders its recorded cause, and a closing `Gap` may render an undatable pair. The
  `not-evaluable` signal timeline is the worked example above.
- **[#45](https://github.com/winniel123/verge-asm/issues/45) and the `Subjects` screen inherit a
  second labelling obligation** beside ADR-0008's floor: a value that arrived after a `Gap` cannot
  be dated, and a surface rendering it beside its predecessor owes the operator that label.
- **The coverage class gains a member and a form of words it lacked** — *we can see again, and
  things differ* is neither drift, nor a clear, nor *we stopped looking*.
- **Nothing new is stored.** No fourth transition name, no marker on a span, no break on an
  opening. The `Gap` was already carrying everything this case needed.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Widen the whole family to any timeline | Leaves a first measurement on a slow cadence unnamed — neither the world nor our aperture caused it |
| Keep `revealed` membership-only and mint a fourth member for facet arrival | Two words for one predicate, differing only in which timeline they sit on — the `Host` defect |
| A fourth member for `Gap` → value | A second representation of a fact the `Gap` already records, with its cause |
| Treat `Gap` → value as `revealed` | The timeline has history; nothing was revealed, we resumed |
| Treat `Gap` → value as nothing | The ticket's second failure mode — a value with no account of why it appeared |
| A `Break` on an aperture widening that only creates timelines | Vacuous on an opening, and it would price a pure addition as a loss of licence |
| `Custody` closing forcing the currency bound | A Declared input reaching into the staleness bound; a second mechanism doing `Gap`'s job |
| Suppressing a `Gap` shorter than some duration | Damping in the model, refused by ADR-0007 and ADR-0006 before it |
| Refusing to show the values either side of a `Gap` | Makes a `Gap` more destructive than a `Break`, and drops a genuine `firewalled` → `exposed` silently |
| Emitting a `Transition` across a `Gap` | Asserts a date for a change we cannot date |
| A distinct message for aperture widening | Two messages for *we changed how we look*; the operator cannot use the distinction |
| Aperture as an open criterion with no enumerated list | A widening is detected by diffing recorded scope dimensions, which requires each to be named |
