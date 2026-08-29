# ADR-0094: A retention control collapses and a retention query never does — and a bound is keyed on the timeline it bounds

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#171 Is one floor over a corpus with per-timeline currency bounds the right collapse?](https://github.com/winniel123/verge-asm/issues/171)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) made the
observation retention floor **the currency bound**, and
[ADR-0007](./0007-drift-is-a-timeline-of-spans.md) makes that bound **per timeline** — `k` cadences
of the tightest covering `Scan`. [#139](https://github.com/winniel123/verge-asm/issues/139) ·
[ADR-0081](./0081-a-floor-is-territory-and-an-unbounded-default-is-a-position.md) then had to draw
**one** control over that corpus, and collapsed the per-timeline bound into one floor by taking the
**longest** bound in force, so the dial could never discard a live observation anywhere. ADR-0081
flagged the step as its own thinnest ground: *"turning one per-timeline bound into one floor over one
corpus is this ADR's step, and the conservative direction was chosen without an argument that the
tight direction is unsafe for every timeline it would touch."*

This ticket is that argument, and three things arrive fixed.

- **The floor is the currency bound and the bound is per-timeline** — ADR-0041, not re-argued.
- **A `Scan` does not have a port list**, and **the floor may never be presented as an operator
  choice** ([ADR-0084](./0084-a-scan-is-a-cadence-over-an-exchange-and-an-uncovered-facet-has-no-currency-bound.md),
  ADR-0081).
- **An invisible horizon is the thing to avoid.** That is the whole of #139's ruling, and it is why a
  per-facet dial set — *n* controls where the product has one — is not the alternative.

And one thing arrived **after** #139 resolved.
[#142](https://github.com/winniel123/verge-asm/issues/142) · ADR-0084 minted the fifth `Scan`, `dns`,
which covers `resolution` and our own resolver's `dns-record`. Those were the two facets ADR-0081's
enumeration existed to exclude. **The exclusion list is now empty, so the collapse's stated
justification is spent while the collapse itself is still in force.**

## Decision

| Concern | Decision |
| --- | --- |
| The number of controls | **One, unchanged.** A per-facet dial set is refused: *n* controls is *n* horizons, and every one is invisible |
| The collapse | **Withdrawn.** One floor over the longest bound in force is a guard placed over the wrong hazard |
| What the retirement query reads | **Each row's own timeline bound.** A row is retained while `age < max(bound(row), dial)` |
| The dial's floor, as drawn territory | **The *tightest* bound in force**, because below it the control has no effect on any row — which is what *not the operator's territory* means, derived rather than asserted |
| The key the bound is resolved on | **The timeline key** — `(subject, facet, discriminator, vantage, source)`. ADR-0007's currency rule states a *different* tuple, and that tuple cannot resolve the bound |
| Subject **in the estate**, no covering `Scan` | Bound **undefined**, per ADR-0084 — so the row is **never retired**. Undefined is not expired |
| Subject **withdrawn** | Timelines are closed and no derivation may read the row, so it has **no floor**: the dial alone governs. Tighter than the collapse it replaces |
| `Batch` retirement | Unchanged and **not** per-row: a `Batch` is retained while **any** observation it produced is retained, or while it is a current `Citation` (ADR-0041) |
| Where a per-row instant renders | **The clamp list ADR-0081 already built**, which is per-timeline already. No new object |
| Whether the dial may state a horizon a row outlives | **Yes, and it is rendered.** The dial names the horizon the operator is buying; the floor is what the product will not sell below |

**The rule, generalised beyond retention: a control may collapse a per-row quantity, and a query may
never. *n* computed instants under one control is not *n* controls, and only the second is an
invisible horizon.**

## Rationale

### The collapse was a guard, and the hazard it guards against arrives somewhere else

#139 collapsed to the longest bound so the dial *"never discards a live observation anywhere."* That
is the right fear. It is guarding the wrong door.

**ADR-0007 states two different tuples for two things that must be the same tuple.** Its decision
table gives the **timeline key** as `(subject, facet, discriminator, vantage, source)` — *"one per
source"* — and its currency section gives the bound as *"within `k` cadences of the Declared `Scan`
whose scope covers that `(subject, facet, port, vantage)`."* Five components against four, and they
are not the same four: the currency tuple has a `port` the timeline key does not, and lacks the
`source` the timeline key turns on.

`port` is harmless — it is already inside the subject. A `Service` **is** `(Address, port, transport)`
and an `Endpoint` is `(Name, Service)`, so naming the port beside the subject restates it. The two
`Name` facets have no port at all, which is also why *a `Scan` does not have a port list* costs this
lookup nothing. Coverage over a port is decided by the port list on the two `Scan`s whose exchange is
a **connect**, exactly where ADR-0084 put it.

`source` is not harmless, and v1 has exactly one instance of it — which `CONTEXT.md` names in terms:
*"In v1 exactly one such pair exists — the operator's zone file against our own resolver, on
`dns-record`."* On a `Name` inside a name scope holding a supplied zone file, `dns-record` holds
**two timelines**, and they have **two different covering `Scan`s**:

| Timeline | Covering `Scan` | Cadence | Bound at `k` = 2 |
| --- | --- | --- | --- |
| `dns-record`, source = our resolver | `dns` | daily | 2 days |
| `dns-record`, source = the zone file | `zone` | the operator's re-supply interval, shipped monthly | ~2 months |

**The four-tuple cannot tell them apart.** A competent session building the retirement query from the
currency sentence as written resolves both rows to one `Scan` and takes *"the tightest such cadence
where several apply"* — which hands the zone-sourced row a two-day bound and discards it while it is
**still live**, on the corpus the map calls *the product's spine*. Read alone and in the present tense
that sentence builds the exact failure #139 collapsed the floor to prevent, and the collapse does not
stop it: the collapse governs the **dial**, and this is the **bound**, which is under the dial and
under the floor and is what *live* means. It is withdrawn at ADR-0007's site.

So the finding is not that the collapse is dangerous. It is that **the collapse was never what stood
between the corpus and a discarded live row** — the bound is, and the bound was under-keyed. Fix the
key and the collapse has nothing left to do.

### After ADR-0084 the collapse's own justification is spent, and what is left is a false affordance

ADR-0081 justified the collapse and its rendered enumeration on a fact: *"Four facets have a covering
`Scan`. `resolution` and `dns-record` do not."* An uncovered observation has no bound, never becomes
evidential, and the dial never reaches it. ADR-0084 closed that hole. **All six facets now have a
covering `Scan`, so every observation row is inside the dial's reach and every one of them is governed
by the maximum.** The exclusion list is empty and the enumeration's stated reason is gone.

What that leaves is worth walking, because it is the case against the collapse and it is ADR-0081's
own criterion turned around. Bounds in force at shipped settings — the daily port tier, the full-range
tier shipped configured and disabled with an empty scope list, `tls-acceptance` weekly, `zone` at the
operator's monthly re-supply promise, `dns` daily:

| Install | Tightest bound | Longest bound = #139's floor | Ratio |
| --- | --- | --- | --- |
| Name scope, **no** zone file | 2 days (`dns`, the daily tier) | 14 days (`tls-acceptance`) | 7× |
| Name scope **with** a zone file — the recommended install | 2 days | ~2 months (`zone`) | ~30× |
| Cold tier enabled with a scope | 2 days | ~2 months (the full-range tier) | ~30× |

On the recommended install the floor holds every `reachability` row for two months because **one
facet, on one source, on the smallest and slowest population in the corpus**, is re-supplied monthly.
The floor is set by the corpus's slowest member and applied to its fastest one, and nothing about the
rows being retained is readable from the number retaining them —
[ADR-0038](./0038-a-constant-is-a-product-only-where-the-quantity-is-readable.md)'s shape, arriving in
retention clothing.

**And it fails ADR-0081's own test for why a floor is territory rather than an error.** ADR-0081:
reaching below the floor *"produces the derivation — `k` × cadence, and the name of the `Scan`
supplying the cadence — which is the honest answer to why can I not go there, and which names the
thing the operator **would** change to move the floor. A floor that names the `Scan` that sets it is a
floor the operator can move by changing the world rather than by arguing with a form."*

Under the collapse, on the recommended install, that answer names **`zone`** — and the thing the
operator would change to shorten the retention horizon on their `reachability` evidence is **their own
promise about how often they re-export their zone file**. The control's one honest affordance points
at degrading a `Source`'s declared freshness to buy disk on an unrelated facet. That is not a floor
the operator can move by changing the world. It is a floor they can only move by making the product
worse, and ADR-0081 built the territory rendering precisely so the answer to *why can I not go there*
would be a true one.

Under a per-row query the same answer names **`dns` at daily** — an ordinary operator cadence dial,
whose consequence ADR-0084 has already written down (*"a `dns` cadence looser than the port tier
produces subject withdrawal, not stale probing"*). Same rendering, same rule, and now the affordance
is honest.

### The load-bearing check: is the per-row query expressible?

The ticket calls this the work, and it is. Three requirements, and the answer is **yes, after one
repair**.

**One — can the bound be resolved per row?** Yes, on the timeline key and only on the timeline key.
The lookup is: take the row's timeline, find every enabled `Scan` whose scope covers it, take the
tightest cadence, multiply by `k`. Every component is Declared and already recorded — a `Batch`
records the source and the vantage, the subject carries the port where there is one, and coverage over
a port is decided by a port list that exists on exactly the two `Scan`s whose exchange is a connect.
The `discriminator` is in the key and does not move the answer, since `dns` covers the whole declared
qtype set. Carrying it costs nothing and dropping it would put a third tuple in the model.

**Two — what happens where the bound is undefined?** ADR-0084 ruled that an uncovered facet's bound is
*"undefined rather than loose"*, and the retirement query is the place where that ruling has to be
executed rather than stated. It splits on the **subject**, not on the bound, and the two halves fall
opposite ways:

- **The subject is in the estate and no `Scan` covers the timeline.** Nothing can say when the row
  stops being readable, so nothing may discard it: the row is **never retired**. Post-ADR-0084 this
  population is **empty at shipped settings** and becomes non-empty only where an operator disables a
  `Scan` — so it is a standing reason to leave no facet uncovered rather than a retention policy. It
  must be written down anyway, because the naive expression of *retire where `age > bound`* over an
  undefined bound is a null comparison, which most engines will silently decline to delete. Accidentally
  correct is worse than deliberately correct: the next refactor makes it accidentally wrong.
- **The subject is withdrawn.** Its timelines are **closed**
  ([ADR-0082](./0082-a-withdrawn-subjects-timelines-close-and-the-withdrawn-period-is-on-no-timeline.md)),
  the withdrawn period is on no timeline at all, and no derivation may read the row. ADR-0041's unit of
  the rule is *the reader, never the date* — and the only reader left is the person asking *what did we
  actually measure*, whose horizon **is** the dial. So the row has **no floor** and the dial alone
  governs it. This is **tighter** than the collapse, which held those rows at the corpus-wide floor for
  no reader at all, and it is the direct answer to the ticket's *chosen without an argument that the
  tight direction is unsafe*: the tight direction is unsafe in exactly one place, that place is the
  first bullet, and it is closed by name.

**Three — does a per-row instant re-introduce the invisible horizon?** No, and this is the check that
decides it.

ADR-0081 made the clamp invariant a sort order: *"one ordered list of every clamp in force, tightest
first, the binding one alone drawn in ink,"* whose members are **the first `Batch`, a `Break`, and each
retention dial**. The first two are **per-timeline objects**. So the list ADR-0081 built to hold the
horizon is *already* rendered per timeline, and a per-timeline retention instant sorts into it at its
own instant with no new machinery, still inert in v1 because both dials ship unbounded.

It does one thing more. ADR-0081 recorded an asymmetry as the reason the list works: *"the binding row
names the leaf that moved. A retention row can never name anything."* Under a per-row bound **the
retention row names the `Scan` that set it** — the one thing a collapsed retention row could never do.
The per-row query does not weaken the horizon rendering. It is the only version of it in which every
row in the list carries a cause.

So the distinction the whole ticket turns on is **controls versus computed instants**. *n* controls is
*n* horizons an operator must find, hold in their head and reconcile — which is what kills the
per-facet dial set, and #139 is right about it. *n* computed instants under **one** control, each
rendered where the timeline it bounds is already rendered, is not a horizon at all. It is the one
horizon, evaluated. Collapsing the control is required. Collapsing the query was never required, and
buying the second with the first is what made the floor thirty times too high.

### What the dial means, now that a row may outlive it

This is the strongest objection to the ruling and it must be answered rather than absorbed. Under the
collapse the dial can never over-promise, because the floor stops the operator setting a value any row
would outlive. Under a per-row query, an operator setting seven days will have zone-sourced
`dns-record` rows living two months. Does the control then lie?

It does if it is drawn as *discard at D*. So it is not. **The dial names the horizon the operator is
buying, and the floor is what the product will not sell below** — which is what it already meant. The
collapse merely hid the gap by raising the floor until no row could exceed it. The rendering
obligation is discharged by an object ADR-0081 already built: **the enumeration beneath the dial.**

It was a two-way split — facets the dial reaches, facets it does not — and ADR-0084 emptied the second
half. It becomes a **row per facet–source pair, each with its own floor and the `Scan` supplying it**:
six rows, on the recommended install one of them reading `dns-record` (zone file) · `k` × monthly ·
`zone`. Same object, same place, same #47 ground — *a dial whose reach is silent is a dial the operator
believes covers everything* — with the silent half replaced by a stated one. The list ADR-0081 built
for a reason ADR-0084 removed acquires a better job, and this ruling costs **no new object anywhere**.

### Two floors, and they have already come apart

ADR-0081 kept the `Dispatch` floor and the observation floor as two rules that *"on every install
drawn here produce the same number"*, and named the case that would separate them: *"a `Scan` that is
enabled and covers nothing — which `zone` on an install with no name scope holding a zone file is
already close to."*

Walking it against the corpus rather than quoting it: `zone`'s scope is *"the name scopes holding a
supplied zone file"*, and *"a `Scan` whose scope list is empty is a legible state."* So on any install
with no zone file, `zone` is **enabled and covers nothing**. The `Dispatch` floor — slowest **enabled**
— is `k` × monthly. The observation floor — slowest **covering** — is `k` × weekly, from
`tls-acceptance`. **They differ by a factor of four, today, at shipped settings, on an ordinary
install.** *Already close to* understates it. They are already apart, and ADR-0081's decision to keep
them as two rules is confirmed by the case rather than merely by the argument. Recorded here because
this ruling makes the observation floor move further from the `Dispatch` one — to the **tightest**
covering `Scan` — and nothing in the model should read the two as one number again.

### Where this is decided on thin ground

- **The cost of over-retention is not disk, and the ruling does not claim it is.** ADR-0041 already
  established that the disk is not under pressure — ~13 GB a year at the ceiling, under a gigabyte on
  the modal install — so a 30× over-retention costs roughly 2 GB on the one install where anyone would
  notice. The case against the collapse is the **false affordance** and the **spent justification**,
  not the volume, and if a later session finds the affordance argument unpersuasive the arithmetic will
  not rescue this ruling.
- **The per-row query is checked for expressibility, not for cost.** Resolving a covering `Scan` per
  row is a join against a Declared object on every retirement pass, and nobody has measured it against a
  98M-row evidential corpus. It is bounded work — the `Scan` set is five rows — and the pass runs on a
  schedule with nothing waiting on it, but that is an argument and not a measurement.
- **The bound is evaluated against the *current* `Scan` configuration**, so tightening a cadence
  retroactively shortens what a row's floor protects. That is not new and is not a defect — it is how
  currency already behaves, and ADR-0084 has already priced the loosening direction as subject
  withdrawal — but the collapse hid it behind one number and a per-row query does not.

## Consequences

- **[ADR-0007](./0007-drift-is-a-timeline-of-spans.md)'s currency tuple is withdrawn at the site that
  specifies it**, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md),
  **with a replacement rather than a strike**, per ADR-0057. The bound is resolved on the **timeline
  key**. This is the only change in this ruling that could produce a wrong value if it is missed.
- **[ADR-0081](./0081-a-floor-is-territory-and-an-unbounded-default-is-a-position.md) is amended in
  four places**: the collapse to the longest bound, the enumeration's stated reason, the *two floors
  agree on every install* claim, and its own thin-ground flag, which is discharged here. **Its three
  general rules are untouched and are what this ruling is built on** — a floor is territory, an
  unbounded default is a position, and a corpus with no reader travels with the corpus that reads it.
- **ADR-0084's consequence row is corrected at its site.** It applies the `Dispatch` floor's rule —
  *slowest **enabled*** — to the observation floor, which is *slowest **covering***. The two differ
  today on any install with no zone file.
- **[`CONTEXT.md`](../../CONTEXT.md)'s `Observation` entry is amended and gains no term.** The bound is
  keyed on the timeline. The dial's floor is the tightest bound in force. The query applies each
  timeline's own. The two undefined cases are named.
- **The number of dials, corpus rows and controls is unchanged** — two dials, seven corpus rows, one
  observation control. **No aperture input, no declared parameter, no new number, and `k` = 2 is
  untouched.**
- **`prototypes/coverage-retention/` is superseded in two of its computed values** — it computes the
  observation floor as the slowest covering `Scan` and draws the two-facet exclusion list. Ticketed
  rather than redrawn here: it is #139's artefact and redrawing it is prototype work.
- **The floor is still never an operator choice.** Nothing on the control is a floor the operator may
  type. The territory boundary moves and stays territory, and the per-row floors are display in two
  places that already exist.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Keep #139's collapse — one floor at the longest bound in force** — the status quo, the conservative direction, and the option that lost | Its stated justification is spent: ADR-0084 gave the two uncovered facets a covering `Scan`, so the exclusion list the enumeration was built for is empty and every row is now governed by the maximum. It guards the wrong door — what would discard a live row is the under-keyed currency tuple, not the dial. And it fails ADR-0081's own territory test, since the honest answer to *why can I not go there* names `zone`, and the only thing the operator can change to move it is their own zone re-supply promise |
| **A per-facet dial set** — the obvious alternative the ticket names | *n* controls where the product has one, and every one invisible. #139's whole ruling is that an invisible horizon is the thing to avoid, and it is right. This ADR's point is that the per-row **query** was never the same object as a per-facet **control** |
| **Enlarge the dial's territory to the tightest bound and stop there**, leaving the query collapsed | It is the ruling with the safety removed. With no per-row floor in the engine, a dial set below a row's own bound discards a **live** observation, which ADR-0041 forbids outright. The territory move is only legal *because* the engine holds each row's floor |
| **Repair the currency tuple and keep the collapse** — the minimal change, and it fixes the one thing that can produce a wrong value | It fixes the bug and leaves the floor thirty times too high on the recommended install, with an affordance pointing at the operator's zone file. Half of this ticket, shipped as though it were all of it — and the half it leaves is the half ADR-0081 flagged as thin |
| **Take the *median* or a *weighted* bound rather than the longest or the tightest** | It is a number derived from the shape of the corpus rather than from what may read a row, which is ADR-0041's unit of the rule discarded. It discards live rows in the tail, and no rendering can state what it is |
| **Retire a withdrawn subject's rows on the corpus floor like everything else** | A withdrawn subject's timelines are closed and no derivation may read them (ADR-0082), so the floor protects a reader that does not exist while the person who *does* read them is the one the dial is for. The collapse held them for no reader at all |
| **Treat an undefined bound as expired, since an uncovered facet has no currency** | ADR-0084's ruling is that undefined is *not loose*, and reading it as expired inverts it exactly. The population it would delete is live observations on a facet whose `Scan` an operator disabled — the case where we can least say when the row stops being readable |
| **Make the observation floor and the `Dispatch` floor one computed number**, since both move to a `Scan` cadence | Refused by ADR-0081 and now refuted by an install rather than by an argument: slowest **enabled** and slowest **covering** differ by 4× at shipped settings wherever no zone file is supplied. This ruling moves the observation floor to the *tightest* covering `Scan` and takes them further apart |
| **State the dial's meaning in copy — *or its own bound, whichever is later*** | An invariant stated in prose has to be believed — ADR-0081's own objection to a copy line beneath the horizon. The per-facet floor enumeration already exists beneath the dial and states it structurally |
