# ADR-0082: A withdrawn subject's timelines close — an open span must be fed, and the withdrawn period is on no timeline at all

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#140 Does a withdrawn subject's timeline close, or hold an open withdrawn span?](https://github.com/winniel123/verge-asm/issues/140)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) needed one
sentence before it could state the mechanism by which a `Break` destroys `returned`, and it ruled
it in passing: *"A withdrawn subject's timelines close; they do not hold an open `withdrawn`
span."* It then flagged the sentence itself — *"ruled here on the glossary's own wording, not on a
prior decision"* — and [#121](https://github.com/winniel123/verge-asm/issues/121) called it **the
thinnest load-bearing sentence in that ADR** and asked for it to be tested on its own. This is that
test.

What hangs on it is narrow and expensive. Under *closes*, a `Break` between a subject's withdrawal
and its return leaves the reopening with nothing legally before it, so no `Transition` is derived
and [ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)'s membership message
fires reading **`appeared`** where the truth is `returned`. The leaf that reaches membership across
the whole estate is `resolution-walk`, which
[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) puts on a **dependency** cadence, so
the trigger is `go.mod`. Under the other answer — an open `withdrawn` span — the `Break` clamps that
span and re-opens it under the new vector, the return is an ordinary adjacency under one derivation,
and **`returned` survives**. Three release obligations exist only because of the first answer.

**That the other answer is materially cheaper is a reason to look at it hard, and no reason at all
to rule for it.** It was looked at hard, in its strongest form rather than the form the question
names, and it lost.

The test had to be run without re-deriving from the wording under test. `Span`'s *"it opens, it is
current, it closes; the open span is the current state"* is the sentence on trial and is therefore
inadmissible as its own warrant. Everything below is argued from prior decisions and from the
model's keys.

## Decision

**A withdrawn subject's timelines close. There is no `withdrawn` value on any timeline, and the
period between a withdrawal and a return is on no timeline at all — neither a value nor a `Gap` —
which is precisely what leaves the span before it and the span after it adjacent.**

ADR-0041's sentence is **CONFIRMED**, and its flag is discharged: the sentence does not rest on the
glossary's wording, and never only did.

| Concern | Decision |
| --- | --- |
| A withdrawn subject's timelines | **Close.** Every timeline the subject held, at every `(facet, discriminator, vantage, source)` key |
| An open `withdrawn` span | **Refused.** There is no such value, on any facet or any Derived timeline |
| The withdrawn period | **On no timeline.** Not a value, not a `Gap`, and no object holds it |
| What records the departure | The **closure**, carrying its reason — ~~ADR-0007's `cascaded` is the one member the model has already named~~ **a closed union of three: `measured-absent` · `uncited` · `descoped`** ([#147](https://github.com/winniel123/verge-asm/issues/147) · [ADR-0087](./0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md)) |
| Why the closure is enough | A closed span is **free to keep**; an open span must be **fed**, and nothing measures a subject that is not in the estate |
| Why `returned` is derivable at all | With no `Gap` between them, the closed span and the reopening are **consecutive spans on one timeline**, which is a `Transition` by ADR-0007's own definition |
| Why a `Break` destroys it | The two spans sit under different vectors, and nothing is compared across a break — the adjacency survives and the **licence** does not |
| The strongest competing reading | Not a `withdrawn` value but an **open `NameError` span**. Refused on two independent grounds below |
| ADR-0041's three release obligations | **All three stand, unchanged**, and are now founded rather than provisional |
| Is anything cheaper | **No.** The cost is stated rather than reduced |

## Rationale

> **The reason `cascaded` is quoted several times below, from ADR-0007 and about it, and the name is
> superseded.** [#147](https://github.com/winniel123/verge-asm/issues/147) ·
> [ADR-0087](./0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md) closed
> the vocabulary at three — **`measured-absent` · `uncited` · `descoped`** — and the cascade's member
> is now **`uncited`**, widened to cover de-citation because the rule ADR-0007 states holds verbatim
> at both sites and the word does not. Marked once here rather than at each quotation, per
> [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md), because
> every instance below is a **quotation of ADR-0007 in support of a different point** and each is
> struck at ADR-0007's own sites; the arguments they carry are untouched.

### The flag's premise was wrong: ADR-0007 had already ruled this, for most of the estate

ADR-0041 recorded that the sentence rested on the glossary alone and on no prior decision. It does
not. [ADR-0007](./0007-drift-is-a-timeline-of-spans.md), settling ADR-0006's cascade rule, says it
outright and gives its own reason:

> ADR-0006's cascade rule takes every `Endpoint` beneath a withdrawn `Name`; **those spans must
> close, or a dead endpoint keeps an open span and every current-state query returns it as live** —
> but the closure is not independent evidence, so it records reason `cascaded`, which is what lets
> the endpoints return coherently if the name does.

That is a decision, with a stated consequence, on the same question. It is narrower than ADR-0041's
sentence — it governs the **cascade** population, the `Service`s and `Endpoint`s beneath a withdrawn
root, and not the root itself — which is why ADR-0041 was right that the *general* sentence was
unfounded and wrong that nothing prior bore on it. Two things follow immediately.

The reason ADR-0007 gives is subject-kind-agnostic. *Every current-state query returns it as live*
is true of a `Name` holding an open span exactly as it is of an `Endpoint`. The board, the census,
`Coverage`'s denominators and every count in the product read current state; a subject with an open
span is in all of them. The only way to hold an open span and stay out of those answers is to make
every current-state query special-case one value, which is
[#32](https://github.com/winniel123/verge-asm/issues/32)'s refusal — *a rule reads a leg, never a
state* — arriving as a query predicate instead of a rule.

And the two readings **cannot coexist in one tree**. ADR-0007 fixes the cascade population at
*close*. An open `withdrawn` span at the root would leave a `Name` that every current-state query
returns as live sitting above `Endpoint`s whose spans are closed, and the membership message's
census — the only carrier a returning sub-tree has
([ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)) — would be computed
over a half-dead tree. Reversing this sentence therefore does not cost one sentence; it reopens
ADR-0007's cascade closure and the reason `cascaded` exists.

### The decisive asymmetry: a closed span is free to keep, an open span must be fed

This is the argument that ends the question, and it is arithmetic on the model's own currency rule
rather than an appeal to any wording.

An open span holds the **current** value of a timeline, and ADR-0041's own corpus-1 ruling bounds
what current means: *"an observation is current while it is within `k` cadences of the Declared
`Scan` whose scope covers that `(subject, facet, port, vantage)`"*, `k` fixed at 2. Past the bound
the value is non-constructible and a `Gap` opens. So an open span is not a free record — it is a
standing claim that something is still being measured.

Nothing measures a withdrawn subject. It is not in the estate, so it is in no `Batch`'s recorded
scope, and ADR-0007 is explicit that *a batch whose recorded scope excludes a thing does not touch
that timeline at all*. Two consequences, and both are fatal to the open span:

1. **The open span rots within `k` cadences.** On the daily cadence that covers `verge-core` that is
   two days. Whatever the span held — `withdrawn`, or the `NameError` beneath it — the value ages
   out and the timeline holds a `Gap`. And **no `Transition` crosses a `Gap`** either, so the
   cheaper answer does not even deliver the thing it was wanted for: a return after two days would
   be `Gap` → value, an ordinary adjacency in the coverage class, and not `returned`. The open span
   buys `returned` for two days and loses it thereafter, which is worse than losing it predictably
   at a release.
2. **Keeping it current means querying the dead forever.** To hold the span open and current, every
   withdrawn `Name` must stay in the resolution scope of every `Batch`, on every cadence, in
   perpetuity. That is the append-only inventory ADR-0006 called *half a drift product*, arriving in
   the aperture record instead of the subject list — a monotonically growing scope, one query per
   dead name per cadence, and a `Batch` scope that can never shrink. ADR-0047's exact denominator
   and every coverage statement computed against it would be arithmetic over names the operator
   decommissioned years ago.

The closure has neither problem. A closed span needs no measurement to stay closed, and
ADR-0041 already rules the `Span` corpus **never compacted**, so it is there for the return to be
adjacent to, at ~200 bytes, forever. **The model's cheap object is the closed span and its expensive
object is the open one, and the competing answer picks the expensive one to save a word.**

### Withdrawal is a property of the subject, and a `Span` is not a subject-level object

[ADR-0014](./0014-only-revealed-generalises.md) settled the line this question is a special case of:
**membership is a property of a subject, aperture is a property of looking, and looking is
per-timeline.** That is why `appeared` and `returned` are membership-only while `revealed`
generalises to any timeline.

A `Span` is keyed `(subject, facet, discriminator, vantage, source)`. Withdrawal is not any of the
four trailing components' business: [ADR-0006](./0006-subjects-leave-by-measurement.md) requires
**every available vantage** to agree before a subject leaves, *composing `Availability` exactly as
`Exposure` does*, and one vantage down makes membership **not-comparable rather than concluded from
the survivor**. So a `withdrawn` value would be a fact about the join across every vantage and every
source, written identically onto every per-vantage per-source timeline the subject holds — one fact
in *n* representations, which is the standing seam rule broken *n* times, and each copy individually
false of the timeline it sits on. It is ADR-0017's `internal-only` defect exactly: a name given to a
reading that is not about the thing the key holds.

The same test refuses it a home among the Derived timelines. `Custody`, `Reach` and `Exposure` hold
spans, and each is a value **about a subject that is in the estate**. There is no membership
timeline object to put a span on — ADR-0006 rules membership *a Derived view over the latest
observation per facet*, and the map's own instruction is that a state enum is a projection and not
the thing.

### The strongest form of the other answer is the open `NameError` span, and it fails twice

The question names an open `withdrawn` span, and in that form it is easy: `withdrawn` is a state we
invented, sitting between present and gone, which is the enumerated refusal ADR-0006 opens with —
*no `dormant`, no `stale`, no `unconfirmed`*. Ruling against a straw man proves nothing, so the
better version was built and tested.

**The better version needs no new value at all.** `resolution`'s value space already contains
`NameError`; a measured negative *"is a value and must not collapse into we did not look"*; a `Name`
leaves when our resolver measures a Name Error from every available vantage. So the honest record of
a departure is the timeline holding an open `NameError` span. Nothing is invented, ADR-0006's
*removal is an observed value, not an invented event* is honoured more literally than by the
closure, and a `Break` clamps and re-opens that span under the new vector so that `NameError` →
`Resolved` is an ordinary adjacency and `returned` survives. This is the real competing option and
it is a good one.

It fails on two grounds that have nothing to do with the wording under test.

**It is inexpressible for half the population it must serve.** `Address` — alone among the four
subjects — *has no lifecycle of its own, because nothing ever observes an address's existence*. A
cited `Address` leaves when the resolution that cited it stops citing it, and an address leaves a
declared scope by a Declared act. Neither is an observation **about the address**, so there is no
timeline of that subject that could hold an open value meaning *gone*. The open-`NameError` reading
therefore rescues `returned` for `Name`s and cannot state the `Address` case at all — and the cited
`Address` is precisely the second population whose membership ADR-0041 traced to `resolution-walk`.
A rule that works on one of the two membership-bearing subject kinds and is inexpressible on the
other is not a rule; it is an accident of which facet happened to carry the evidence.

**And it rots, for the reason in the section above.** A withdrawn name is not re-queried, so its
`NameError` observation ages past the currency bound within `k` cadences and the span becomes a
`Gap`. No `Transition` crosses a `Gap`, so `returned` is lost anyway — not at the next release, but
two days after every decommission, on every install, with no release involved and nothing to state
the loss on.

What survives of it, and is worth keeping, is that the `NameError` **observation** is real and is
retained: it is the evidence that the departure was measured, it lives in the observation corpus on
corpus-1's terms, and it is what the closure's reason names. The closure is not a discarded
measurement. It is the boundary the measurement drew.

### What *close* leaves behind, stated exactly, because ADR-0041 left it implicit

Two sites already rule that a withdrawal **takes the subject's timelines with it and leaves no
`Gap` behind** — `Custody`'s account of a resolution ceasing to cite an address, and `Address`'s
account of leaving a declared scope. Neither is the sentence under test, and between them they
settle the one thing ADR-0041 needed and never said: **the withdrawn period is on no timeline at
all.**

That is not a detail. It is what makes the mechanism work. Because no object holds the gap between
them, the span before the withdrawal and the span after the return are **consecutive spans on one
timeline**, and a `Transition` is *the adjacency between two consecutive spans* — derived on read,
by machinery that already exists, with no case for absence anywhere in it. `returned` is that
adjacency where the two spans share a vector; where a release moved a leaf between them, the
adjacency is still there and the **licence** is not, which is what a `Break` withdraws.

It also explains why the model's contrast between a closed gate and a withdrawal is not
redundant. A closed probing gate leaves the subject in the estate: it *stops feeding* the timeline,
the last value ages out, and what follows the bound is a `Gap` naming the operator's own act. A
withdrawal removes the subject: the timelines close and there is no `Gap`. If a withdrawn subject
held an open span, those two paths would be the same path — the value would age out and a `Gap`
would open — and `Custody`'s explicit *leaves no `Gap` behind* would be false. The competing answer
does not merely add a value; it collapses a distinction two glossary entries draw deliberately.

### The cost is stated rather than reduced

Nothing here makes anything cheaper, and the honest form of that is to say so.

- A `Break` between a withdrawal and a return **destroys `returned`**, estate-wide, for every `Name`
  and every cited `Address`, and it cannot be repaired afterwards because history is never
  re-derived. The operator is told a machine they decommissioned last quarter is a new machine.
- The trigger is a **dependency upgrade**. `resolution-walk` holds the DNS library among its
  declared parameters on ADR-0021's dependency cadence, so `go.mod` reaches membership across the
  whole estate.
- The one immune population is the **`Seed`-covered `Address`**, in the estate exactly while a
  `Seed` covers it. A `Seed` is Declared and carries no vector, so that membership timeline cannot
  break. It is a property to preserve, not an accident.

The remedy is the three obligations ADR-0041 already wrote, and they now rest on a tested sentence.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) is amended in one entry and gains no term.** `Span` records
  that the clause was tested on its own and confirmed, on grounds independent of its own wording,
  and gains the corollary that the withdrawn period is on **no timeline** — not a value and not a
  `Gap` — which is what leaves the spans either side of it adjacent.
- **ADR-0041 is amended at two sites**, per
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) and
  [#106](https://github.com/winniel123/verge-asm/issues/106)'s rule that a document supersedes
  itself. Its *"ruled here on the glossary's own wording, not on a prior decision"* and the
  *"thinnest load-bearing sentence"* flag are **discharged at the clauses that make them** — read
  alone and in the present tense, either would send a competent session to re-run a test that has
  been run.
- **The three release obligations are unchanged and are no longer provisional.** ADR-0014's
  re-baseline payload keeps its third element; `resolution-walk`'s golden corpus keeps the
  membership-deciding rows ([#143](https://github.com/winniel123/verge-asm/issues/143) owns them);
  and the membership vector may not be widened.
- ~~**The closure is now the sole record of a departure, and only one closure reason has ever been
  named.** ADR-0007 gives `cascaded` for the cascade route. The model has at least three more —
  measured absence at the root, an `Address` losing its last citation, and an `Address` leaving a
  declared scope — and none of them has a reason. This ADR does not mint the vocabulary; it records
  that the closure is now carrying weight that only one of its routes was built for.~~
  **DISCHARGED at the clause that defers it**
  ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)), by
  [#147](https://github.com/winniel123/verge-asm/issues/147) ·
  [ADR-0087](./0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md) — read
  alone and in the present tense it sends a session to mint a vocabulary that has been minted. **The
  closure is the sole record of a departure and its reason is a closed union of three**, sorted on
  the ground the closure rests on: **`measured-absent`** (an observation about this subject),
  **`uncited`** (an observation about another subject — the cascade and de-citation are **one**
  reason, since ADR-0007's own splitting test finds the name carries its rule at both sites and the
  `Citation` already names *which* thing was lost), and **`descoped`** (our aperture stopped
  covering it — a route ADR-0047 had already cut off from the measured ones and ADR-0074 had already
  given its own message class). The four routes named above are therefore **three reasons**. Nothing
  else is added to the closure: no vector, no actor, no pointer, no marker.
- **An open item is raised against obligation 3, and it is not settled here.** ADR-0041 states that
  membership composes `resolution-walk` and nothing else. But a `Name` under a `Shadowed` answer
  *cannot leave at all*, and `Shadowed` is `wildcard-discrimination`'s output rather than
  `resolution-walk`'s (ADR-0021's leaf table). If deciding that a subject does **not** leave is
  deciding presence, then `wildcard-discrimination` is in the membership vector — and its
  control-label count has already moved `5` → `9`
  ([#115](https://github.com/winniel123/verge-asm/issues/115)). This bears directly on *never widen
  the membership vector* and on which leaf's golden corpus owes the `Shadowed` rows, and it is a
  different question from this one.
- **Nothing new is stored.** No value, no marker on a span, no membership timeline object, no
  fourth transition name.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **An open `NameError` span on the departing `Name`'s `resolution` timeline** — the strongest form of the losing option, and the one that needs no invented value | **The losing option.** It is inexpressible for a cited `Address`, which has no lifecycle of its own and no timeline that could hold a value meaning *gone*, so it rescues `returned` on one of the two membership-bearing subject kinds and cannot state the other. And it rots: a withdrawn name is not re-queried, so the span ages past the currency bound within `k` cadences and becomes a `Gap`, and no `Transition` crosses a `Gap` — it buys `returned` for two days and then loses it silently on every install, with no release to state the loss on |
| **An open `withdrawn` span, as the question names it** | A state we invented between present and gone, which ADR-0006 refuses by name; a subject-level fact written onto per-`(vantage, source)` timelines, each copy false of the timeline it sits on; and it would need a membership timeline object the model does not have |
| **Keep the withdrawn subject in every `Batch`'s recorded scope so the open span stays current** | The append-only inventory ADR-0006 called half a drift product, arriving in the aperture record: a scope that can only grow, a query per dead name per cadence forever, and ADR-0047's exact denominator computed over decommissioned names |
| **A `Gap` over the withdrawn period** | A `Gap` is *the period over which we could not say*, and we said. It would also make a withdrawal indistinguishable from a closed probing gate, which the model contrasts deliberately, and `Gap` *never withdraws a subject* — so the object that records the departure would be the one object ruled incapable of causing it |
| **Rule *open span* because the release obligations get cheaper** | The obligations are a consequence of what is true of the model, not an input to it. Pricing them first is how a release buys itself a word it has not earned — and, on the arithmetic above, it does not even buy it |
| **Close the timelines but store the withdrawal as a marker on the closing span** | A second representation of a fact the closure already is, and the reason vocabulary ADR-0007 built for `cascaded` is the shape that already fits |

## Where this is thin, stated rather than smoothed

- ~~**The closure's reason vocabulary is one member long and is now load-bearing.** With the
  withdrawn period on no timeline, the closure is the whole record of the departure, and only
  `cascaded` has ever been named. Nothing in this ADR turns on the missing members, but the next
  session that needs to tell *measured absence* from *de-citation* on the `Span` corpus will find
  they are the same row.~~
  **NO LONGER THIN — struck at the site that states the thinness**
  ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)), by
  [#147](https://github.com/winniel123/verge-asm/issues/147) ·
  [ADR-0087](./0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md). The
  vocabulary is **closed at three** and the worked example above is settled — *measured absence* and
  *de-citation* are **not** the same row, `measured-absent` being the one closure in the model that
  is independent evidence. What the note did not anticipate is that the field turns out to be
  **necessary rather than tidy**: an `Address`'s two membership limbs are disjunctive, so a subject
  that leaves `descoped` and returns by an ordinary measured resolution reads **`returned`** on the
  spans alone — a decommission undone that never happened, which is ADR-0014's membership/aperture
  distinction collapsing at the storage layer.
- **`returned`'s own predicate has never been written down.** A subject holds many timelines, at
  many `(vantage, source)` keys, whose last spans may have closed under different vectors. Which
  closed span the return is compared against — and what the answer is when they disagree — is
  assumed rather than stated, here and in ADR-0041 alike, which both write as though there were one
  timeline. The ruling does not depend on it; the message's word does.
