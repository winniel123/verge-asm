# ADR-0087: A closure records the ground it rests on — and there are three grounds, not four and not one

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#147 What does a timeline closure record, and how many closure reasons are there?](https://github.com/winniel123/verge-asm/issues/147)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0082](./0082-a-withdrawn-subjects-timelines-close-and-the-withdrawn-period-is-on-no-timeline.md)
made the closure load-bearing and then said so: the withdrawn period is on **no timeline at all**,
neither a value nor a `Gap`, so the closure is the whole record of a departure. It also declined to
mint the vocabulary that record now needs — *"This ADR does not mint the vocabulary; it records that
the closure is now carrying weight that only one of its routes was built for."* **This is that
mint**, and ADR-0082's own thin note is the brief for it: *"the next session that needs to tell
measured absence from de-citation on the `Span` corpus will find they are the same row."*

Two questions arrive together and only look like one.

**How many reasons.** [ADR-0007](./0007-drift-is-a-timeline-of-spans.md) named exactly one — the
cascade closure's — and the model has at least three further routes to a closed timeline: a `Name`
measured absent at the root, a cited `Address` whose last citing resolution moves, and an `Address`
or `Name` leaving a declared scope. On the corpus they are one row.

**What a closure records.** The vocabulary is one field and the question is what the *field set* is.
[ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) rules the
`Span` corpus **never compacted, no policy, no dial, no default**, so whatever a closure records is
kept for the life of the install. A field admitted here is admitted forever, which is the right
posture for pricing one and the wrong one for adding four.

Three prior decisions arrive with obligations, and they do not all point the same way.

- **[ADR-0006](./0006-subjects-leave-by-measurement.md)** — *nothing leaves because time passed, and
  nothing leaves by a state we invented.* Its founding sentence is **subjects leave by measurement**,
  and three of the four routes above are not measurements of the subject that leaves.
- **ADR-0007** — the vocabulary test. `Break`, `Gap` and `Shadowed` were split because they *"sort by
  what we have"*, and one name over the family was refused because *"a name spanning them stops
  carrying the specific rule at any site."* That is a two-way test and it is the one used below.
- **[ADR-0072](./0072-absence-is-a-property-of-a-cell-and-withdrawn-is-the-only-population.md)** —
  `withdrawn` is a population reached **by key**, so there is a reader: the operator looks a departed
  subject up and asks *why is this gone?*, which is `Citation`'s question with the sign flipped.

## Decision

**A closure is not an object and records exactly one thing that is not already recorded elsewhere:
the ground it rests on. There are three grounds — an observation about this subject, an observation
about another subject, or our own aperture — so the reason vocabulary is closed at three:
`measured-absent`, `uncited`, `descoped`.**

| Concern | Decision |
| --- | --- |
| Is a `Closure` an object | **No.** It is the closing side of a `Span` — no new row, no new table, no second representation |
| What a closure records | **One new field: its reason.** Everything else it might record is already held somewhere with a rule attached |
| The reason's value space | **A closed union of three**: `measured-absent` │ `uncited` │ `descoped` |
| What the three sort on | **The ground the closure rests on**, exactly as `Break`/`Gap`/`Shadowed` sort on what we have |
| `measured-absent` | An observation **about this subject** says it is absent — a `Name` measured `NameError` on a cross-class `Vantage composition`. The only closure that is independent evidence |
| `uncited` | The subject's chain back to a `Seed` no longer holds. Covers **both** the cascade and de-citation |
| `descoped` | **Our aperture stopped covering it** — an exclusion, a narrower `Seed`, or a release narrowing a composed population |
| ADR-0007's `cascaded` | **Renamed and widened to `uncited`.** The rule it states holds verbatim at both of `uncited`'s sites; the *word* holds at only one |
| The ticket's four routes | **Three reasons.** The cascade and de-citation are one; the other two are each their own |
| Which closures carry a reason | **A withdrawal's, and no other.** An ordinary value move needs none — the next span is the fact. A version change needs none — the `Break` is derived on read from the two vectors |
| The **derivation vector in force at closure** | **Already on the `Span`, and no field is added.** A version change closes every open span of that derivation, so a `Span` is wholly inside one vector *by construction* and its vector **is** the vector in force at its closure |
| The closing instant | Already what makes a `Span` a period. Not new, not priced |
| An actor — who closed it | **Refused.** [#127](https://github.com/winniel123/verge-asm/issues/127)'s operator-act record arriving through the span corpus |
| A pointer to the subject it went with | **Refused.** The subject's own `Citation` already names it |
| The `NameError` observation itself | **Refused.** It is retained in corpus 1 on ADR-0041's terms; the closure is *"the boundary the measurement drew"* |
| Price against a corpus never compacted | **One enum on a row that already exists** — under 2% of the span corpus at the shipped ceiling, and carried only by withdrawal closures |
| The options that lost | **Four** — one reason per route — and **two** — the reader-only cut. Both are argued below rather than dismissed |
| [`CONTEXT.md`](../../CONTEXT.md) | **Gains one term, `Closure`**, and is amended in three entries |

## Rationale

### The vocabulary sorts on the ground, and in this model there are exactly three grounds

ADR-0007 settled a vocabulary question of this exact shape once, and its method is the method here.
Eight separately-argued instances of *we cannot say* were sorted into `Break`, `Gap` and `Shadowed`
**by what we have**, and a single spanning name was refused because it *"stops carrying the specific
rule at any site."* The test cuts both ways: where a spanning name **does** carry the specific rule
at every site, one name is correct and two are a distinction the model does not make.

A closure asks the same kind of question one level down: not *what do we have* but **what does this
closure rest on**. In a three-layer model whose Declared layer does not drift, a fact can come from
exactly three places, and so can a closure:

| Ground | Reason | Route |
| --- | --- | --- |
| An observation **about this subject** | `measured-absent` | A `Name` our resolver measures `NameError` for, on a cross-class `Vantage composition` |
| An observation **about another subject** | `uncited` | A `Service` or `Endpoint` beneath a withdrawn root; a cited `Address` whose last citing resolution moves |
| **Our own aperture** | `descoped` | An excluded `Name`, an excluded or narrowed address scope, a release narrowing a composed population |

That is not a taxonomy invented for this ticket. It is the same axis the model already sorts
**openings** on — `appeared` and `returned` for the world, `revealed` for our aperture, and an
opening caused by neither recorded but unnamed
([ADR-0014](./0014-only-revealed-generalises.md)). Closures get the third member named where
openings leave it unnamed, and the asymmetry has a cause rather than being an inconsistency: **an
unnamed opening is followed by a span you can read, and an unnamed closure is followed by nothing at
all.** That is precisely what ADR-0082 created when it put the withdrawn period on no timeline.

### The cascade and de-citation are one reason, and the glossary had already written the sentence

This is the merge the ticket asked to be established rather than assumed, and the argument is that
the project has already made it, in one sentence, without noticing it covered two routes.
`CONTEXT.md`'s `Citation` entry:

> a subject whose last citation goes stale has no chain back to a `Seed`, which withdraws it *and*
> closes the probing gate on it

That is true of an `Endpoint` whose `Name` went and true of an `Address` whose citing resolution
moved, in the same words, for the same reason. Run ADR-0007's test at both sites:

- **Is the closure independent evidence?** No, at both. ADR-0007's whole warrant for naming the
  cascade's reason was *"the closure is not independent evidence"*. Nothing observed the de-cited
  `Address` either — ADR-0082 is explicit that a cited `Address` leaving *"is not an observation
  about the address"*, the `Address` being alone among the subjects in having no lifecycle of its
  own.
- **Does the subject return when its ground returns?** Yes, at both. ADR-0007's stated payoff was
  that the reason *"lets the endpoints return coherently if the name does"*. A de-cited `Address`
  returns when a resolution cites it again, by the same mechanism and with the same coherence.

The name carries the specific rule at both sites, which is ADR-0007's own condition for one name.

What the split would have bought is *which* thing the subject went with — and that is already stored.
`Citation` is a single-hop link the subject carries, so putting the answer in the reason word too is
a second representation of one fact, which is the standing seam rule and the argument ADR-0007 used
to refuse storing `Transition`s beside spans. **The reason names the class; the `Citation` names the
thing.**

`cascaded` is then the wrong word for the merged member, because nothing cascades when a resolution
changes its answer: the `Name` is alive and well and simply points somewhere else. A name that is
false at one of its two sites is the defect ADR-0007 refused a spanning name for, arriving from the
other direction. It is renamed **`uncited`**, which is `Citation`'s own word and true at both sites,
and it is withdrawn at every site that specifies it per
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).

### Leaving a declared scope is *not* the same reason, and the model had already sorted that too

The symmetric temptation is to merge the other way — a closure is a closure, the subject is gone,
one word. The model refuses it, and again it refuses it in a sentence already written.
[ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md)'s four-object table gives the
descoping route its own row and then qualifies the parity it was granted:

> **the parity holds for the span mechanism and NOT for the message**: ADR-0013 §5 is a table about
> `Custody` moves caused by **measurement**, and this row is a **Declared act**

That is the cut, made and marked, on the axis this ADR names. And it has since been paid for:
[#130](https://github.com/winniel123/verge-asm/issues/130) ·
[ADR-0074](./0074-an-aperture-narrowing-that-takes-its-carrier-with-it-fires-at-the-scope.md) minted
the **tenth member of the coverage class** for exactly this route, because the narrowing takes the
subject that would have carried the news away with it and so fires **one message at the scope**.

So a `descoped` closure and a `measured-absent` closure already sit in **different message classes**
under [ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md) — coverage
against membership — and already differ in whether the message is per subject or one at the scope.
A name spanning them stops carrying the specific rule at one of its sites. ADR-0007's test, failed.

It is named **`descoped`** on the **aperture** rather than on the operator, and that choice is
load-bearing. An operator's exclusion and a release narrowing a composed population are the same act
with a different author — ADR-0044 already rules that *a tier bounds which subjects exist* — and a
reason called `operator-excluded` would need a fourth member the first time a release removed a pair
from `verge-core`. Naming the ground rather than the actor also keeps
[#127](https://github.com/winniel123/verge-asm/issues/127)'s refusal intact: **the closure records
that our aperture stopped covering the subject and never who narrowed it.**

### The forcing argument: without the reason, an `Address` we stopped declaring comes back reading `returned`

Everything above establishes that three names are *warranted*. This establishes that the field is
**necessary**, and it is the argument that decides the ticket.

An `Address` is in the estate *"exactly while a current resolution cites it **or** a `Seed` covers
it"*, and `CONTEXT.md` is explicit that the two limbs are **disjunctive**. So for the whole `Address`
population — one of the two membership-bearing subject kinds — **leaving by one route and returning
by the other is the ordinary shape and not a corner.** Work it:

1. `203.0.113.5` is in the estate because a declared address scope covers it. Its timelines run.
2. The operator narrows the scope. Nothing cites the address, so it withdraws; its timelines close;
   no `Gap` is left behind, there being no subject to hold one (ADR-0047, ADR-0074).
3. Weeks later a `Name` in an unrelated name scope resolves to it. The address is cited, re-enters
   the estate, and a span opens.

At step 3 the fold sees a closed span, a hole no object holds, and a new span opened by an ordinary
measured resolution — **with no aperture act at the instant of the opening**, because the operator
did not touch the `Seed`. ADR-0082 leaves the two spans adjacent by construction and the vectors are
equal, so no `Break` sits between them. The fold therefore reads a `Transition`, and on the spans
alone the only word available is **`returned`** — *a decommission undone*, for a decommission that
never happened. The subject left because we stopped looking.

That is [ADR-0014](./0014-only-revealed-generalises.md)'s distinction — membership is a property of a
**subject**, aperture is a property of **looking** — collapsing at the storage layer, and it is the
same shape as ADR-0041's *"deleting the span before an open one converts `returned` into
`appeared`"*: **a false statement about the world manufactured by what storage failed to keep.**
ADR-0041 refused that one on the ground that a clock may not move a value; the same ground refuses
this one, because a missing field moves it just as effectively as a deleted row.

So `returned` is **not derivable from the spans alone**, and the closure's reason is the input that
makes it derivable. This ADR rules the record; it does not write the predicate. What follows
directly, because both halves are already ruled elsewhere, is only this: **`returned` and `appeared`
are membership-only (ADR-0014) and a scope act is aperture yielding `revealed` (ADR-0047), so a
`descoped` closure does not license `returned` across it.** Everything else about the predicate —
which of a subject's many timelines it reads, and what it says when their last closed spans
disagree — is [#148](https://github.com/winniel123/verge-asm/issues/148)'s and is deliberately not
touched here.

### ADR-0006 says subjects leave by measurement, and exactly one of the three reasons is a measurement

Worth stating on its own because it inverts how the question reads. The model's founding refusal is
*subjects leave by measurement, and nothing leaves because time passed*. Under the three-member
vocabulary, **one** reason is a measurement of the subject that left. The other two are bookkeeping —
the chain broke — and aperture — we stopped declaring it.

With no vocabulary, those three are the same row, and a departure nobody measured is indistinguishable
from one we did. That is not a cosmetic loss: it is ADR-0006's founding sentence defeated by a missing
field rather than by an argument, which is precisely the failure mode ADR-0041 identified when it read
the same refusal at the storage layer. **The reason vocabulary is what keeps ADR-0006 true on the
corpus.**

One naming note, because a collision here would be silent: the member is `measured-absent` and not
`measured`, since `authority: measured` is a live value in this model and a reason sharing its
spelling would read as a claim about the source rather than about the subject.

### The one-fact-in-*n*-representations objection, which is the strongest one against this ADR

ADR-0082 killed the open `withdrawn` span partly on this ground: withdrawal needs **every available
vantage** to agree, so it is a fact about the *subject*, while a `Span` is keyed
`(subject, facet, discriminator, vantage, source)` — *"one fact in n representations, which is the
standing seam rule broken n times, and each copy individually false of the timeline it sits on."*
A withdrawal closes every timeline the subject held, so the reason is written *n* times too. Does
this ADR re-admit what ADR-0082 refused?

No, on two independent grounds.

**A closure is not a value.** ADR-0082's objection lands because `withdrawn` would have to be a value
of each facet, and it is a value of none of them — an `internal-only`-shaped defect, a name given to
a reading that is not about the thing the key holds. A closure is the **boundary** of a span, and
*this timeline stopped, on this ground* is true of the timeline it sits on, at every one of the *n*
keys. Nothing is asserted about a facet that the facet did not do.

**And the model already writes one cause onto *n* objects, and has never called it a seam.** A
`Vantage` going `unavailable` opens a `Gap` on every timeline that vantage fed, and *"it records its
cause"* — one operator-visible event, *n* objects, each honestly recording why it exists. ADR-0014
drew the parallel first, in the sentence this ADR renames: the `Gap` records its cause the way a
closure records its reason. **The `Gap`'s cause is the precedent, it is exact, and it predates the
question.**

### What the closure does not record, and every refusal has an owner

The field set is the half of this ticket that is priced rather than argued, so each exclusion names
the rule that already holds the thing.

- **The derivation vector.** Already on the `Span` (ADR-0007), and — this is the part worth being
  exact about, because it is what #148 needs — **a closure cannot carry a different one.** A version
  change *"closes every open span of that derivation and opens new ones under the new version, with a
  `Break` between them"*, so a `Span` is wholly inside one vector by construction. Its open-vector
  and its close-vector are necessarily identical. *The vector in force at closure* is a field that
  cannot differ from one that already exists, so it is not added.
- **An actor.** #127 ruled the operator-act record out of scope — *named accounts create identity,
  never a log* — and a `who` on a `descoped` closure is that record arriving through the corpus that
  is never compacted. Refused, and named here so a later session does not build it as a detail.
- **A pointer to the subject it went with.** The `Citation` is that pointer and it is already stored.
- **The observation.** ADR-0082 settled it: the `NameError` observation is real, is retained in
  corpus 1 on ADR-0041's terms, and *"the closure is not a discarded measurement. It is the boundary
  the measurement drew."*
- **A marker distinct from the reason.** ADR-0082 already refused this by name — *"a second
  representation of a fact the closure already is"*.
- **A reason on any closure that is not a withdrawal.** An ordinary value move closes A and opens B in
  one fold step, and the adjacency is the fact. A version change closes A and opens C across a
  `Break` that is *"derived on read from the two spans' vectors and never stored, which is what lets
  it name the leaf that moved"* — storing a reason there would be the second representation ADR-0007
  built the derived `Break` to avoid.
- **Free text of any kind.** Not argued; the corpus is never compacted.

### The price, stated rather than waved through

ADR-0041 sizes the span corpus at **~672,000 rows and flat** at the shipped ceiling, ~200 bytes each,
~135 MB once — against ~98M observation rows a year. A closed three-member enum costs a byte and, at
four bytes with alignment, **under 2%** of that, carried only by the closures that are withdrawals.
It is affordable on the arithmetic that already ruled the corpus affordable.

That number is not the argument, though, and it should not be read as one. The reason the field set is
**one** field is that every other candidate had an owner already, not that four fields would have
overflowed a disk. A pointer and an actor would each be affordable too, and both are refused above on
structure.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) gains one term — `Closure` — and is amended in three entries.**
  `Closure` states that it is not an object, carries the closed three-member enumeration and the
  ground each member names, and says what it does not record. `Span` gains that a withdrawal's
  closure carries a reason and an ordinary one does not. `Transition` gains that a `descoped` closure
  does not license `returned` across it. `Citation` gains that the stale-chain sentence it already
  states is the `uncited` reason and covers both the cascade and de-citation.
- **[ADR-0007](./0007-drift-is-a-timeline-of-spans.md) is amended at three sites**, per ADR-0058 and
  [#106](https://github.com/winniel123/verge-asm/issues/106)'s rule that a document supersedes itself:
  its Decision row *"Cascade closure — writes a span, reason `cascaded`"*, the *Alert on the cause*
  paragraph that states the reason and its warrant, and the [#42](https://github.com/winniel123/verge-asm/issues/42)
  amendment's *"the `Gap` records its cause the way a cascaded closure records `cascaded`"*. Read alone
  and in the present tense, each would have a session build a one-member vocabulary under the old name.
- **[ADR-0006](./0006-subjects-leave-by-measurement.md) is amended at two sites** — its `Endpoint`
  two-leg amendment and its #63 amendment, both of which name the reason and both of which state a
  rule that survives verbatim under the new name.
- **[ADR-0014](./0014-only-revealed-generalises.md) and
  [ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) are amended at
  one site each**, both of them pointers to the vocabulary rather than statements of it.
- **[ADR-0082](./0082-a-withdrawn-subjects-timelines-close-and-the-withdrawn-period-is-on-no-timeline.md)'s
  deferral is discharged and its thin note is struck at the site that states it.** *"Only one closure
  reason has ever been named"* and *"the next session … will find they are the same row"* would each,
  read alone, send a session to run work that has been run.
- **ADR-0072's named debt is discharged, and this is the consequence worth reading twice.** It records
  that *"the operator's durable record of a departure is the message store, not a screen"* and prices
  its decision 3 against the risk that *"if the message store ever acquires a retention horizon, the
  durable record goes with it."* It no longer does. With a reason on the closure, the durable record
  of a departure — the fact and its ground — sits on the **corpus that is never compacted**, and the
  message store is the operator's convenient carrier rather than the model's only one. The debt is
  paid, not deferred.
- **The notification path gains no vocabulary and no case.** All three reasons' message behaviour was
  ruled before this ADR: a cascade closure is silent (ADR-0006's #63 amendment), a membership
  withdrawal writes a message that is not routed (ADR-0006's *appearance splits in two*, ADR-0039's
  store/channel split), and a descoping fires one coverage-class message at the scope (ADR-0074). What
  was missing was the field those rules key on.
- **No prototype owes a mark.** Per [ADR-0075](./0075-a-prototype-is-a-dated-record-of-a-reading-never-of-a-rule.md),
  a drawn state a ruling makes **unreachable** is owed the mark and a figure it moves is owed nothing.
  This ADR adds a field nothing has drawn and makes no drawn state unreachable; `Subjects`' `withdrawn`
  population and its by-key lookup are untouched.
- **Nothing new is stored beyond one enum.** No new object, no new table, no new value on any facet,
  no membership timeline object, no fourth transition name, and no field on a closure that is not a
  withdrawal.
- **[#148](https://github.com/winniel123/verge-asm/issues/148) is handed a record and not a
  predicate.** It receives: a closure carries a reason from a closed three-member set; a `Span`
  carries the vector it was produced under and that vector is by construction the one in force at its
  closure, so no vector field is added and none is needed; and `returned`'s predicate now has two
  per-timeline inputs to compose rather than one.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Four reasons — one per route, keeping `cascaded` narrow and adding a de-citation member** — the reading the ticket's own enumeration most naturally supports | **The losing option, and it lost on ADR-0007's own test rather than on cost** — four enum members cost what three do. At both sites the merged name carries the specific rule verbatim: the closure is not independent evidence, and the subject returns when its ground returns. The only thing the split buys is *which* thing the subject went with, and the subject's `Citation` already holds that — so the fourth member is a second representation of a stored fact, wearing a vocabulary's clothes |
| **Two reasons — *the estate took it* against *our aperture stopped covering it*** — the minimal cut, and the only one with a reader in the `returned`/`revealed` sort | The other serious contender. It reverses ADR-0007, which named the cascade's reason precisely **because** *"the closure is not independent evidence"*, with no warrant for the reversal; and it puts a measured departure and an inferred one on one row, which is ADR-0006's founding sentence read at the storage layer and refused there by ADR-0041 already. The `withdrawn` population is reached by key so that the operator can ask *why is this gone?*, and under two names one of the three honest answers is unavailable |
| **One reason — no vocabulary at all; a closure is a closure and the departure is inferred from the hole** | Deletes a member ADR-0007 named and gave a stated consequence, with nothing offered in its place. It also does not work: an `Address` that left `descoped` and returns by an ordinary measured resolution reads `returned` on the spans alone, telling the operator a decommission was undone that never happened |
| **Store the vector in force at closure as its own field**, which is the field #148's predicate looks like it needs | It cannot differ from a field that already exists. A version change closes every open span of that derivation, so a `Span` is wholly inside one vector by construction and its vector **is** the vector at its closure. Adding it would be a second representation of one fact on a corpus that is never compacted — and worse, a *derivable* second representation, which is the class that goes silently wrong when the two disagree |
| **Record an actor on a `descoped` closure** — *the operator narrowed this scope* | #127's operator-act record arriving through the span corpus rather than through a screen. Every Declared term is a current value with no timeline precisely so that no operator act is written down with an actor on it, and the modal install has one mutating role, so every row would carry one author |
| **Record a pointer to the subject the closure went with** | The `Citation` is that pointer, is a single hop, and is already stored and already retained (ADR-0041). The reason names the class; the citation names the thing |
| **Put a reason on every closure, including ordinary value moves and version changes** | An ordinary move closes A and opens B in one step and the adjacency is the fact; a version change's `Break` is derived on read from the two vectors, *"which is what lets it name the leaf that moved"*. A stored reason at either site is the second representation the derived `Break` was built to avoid, on a corpus that keeps it forever |
| **Keep the name `cascaded` and widen it to cover de-citation** | The word is false at one of its two sites — nothing cascades when a live `Name` merely resolves elsewhere — which is exactly the defect ADR-0007 refused a spanning name for, arriving from the other direction. A withdrawal that supplies no replacement does not hold (ADR-0057), so the rename supplies one |
| **Name the third reason for the operator — `operator-excluded`** | It would need a fourth member the first time a **release** narrowed a composed population, which ADR-0044's *a tier bounds which subjects exist* already admits as the same act with a different author. Naming the ground rather than the actor also keeps #127's refusal intact for free |
| **Mint the `Gap`'s cause vocabulary here at the same time**, since it is the same shape one object over | Tempting and out of scope. `Gap`'s causes are enumerated descriptively in the glossary and have never been ruled closed, and its readers are different — a `Gap` sits on **one** timeline of a subject that is otherwise alive (ADR-0072), where a closure's reason is uniform across a departing subject's timelines. Named as a successor rather than settled in passing |

## Where this is thin, stated rather than smoothed

- **`uncited` on a `Name` has never been walked.** `CONTEXT.md`'s `Citation` sentence is general, so a
  `Name` whose last citation goes stale is `uncited` by the same rule — but ADR-0027 argues a
  CT-admitted `Name` *"acquires a `resolution` timeline from our own `enumerable` resolver within one
  cadence, and leaves by Name Error like any other"*, and ADR-0041 argues the citation exemption is
  subsumed because a subject in the estate has a citation inside the currency bound by construction.
  Between them it is unclear whether the route is reachable for a `Name` at all. **Nothing here turns
  on it** — the member exists for the `Address`, `Service` and `Endpoint` cases, which are not in
  doubt — but a session that needs the `Name` case will find it argued in two directions.
- **`descoped`'s release limb is argued and not observed.** A release narrowing `verge-core` and
  withdrawing `Service` subjects follows from ADR-0044's tier rule and ADR-0074's shape, and no
  decision names it directly. It is why the member is named on the aperture rather than on the
  operator, so if the limb turns out to be unreachable the name is still correct and merely covers one
  case fewer.
- **The `Gap`'s own cause vocabulary is in exactly the position this ticket found the closure's in** —
  recorded as *"it records its cause"*, enumerated descriptively in five or six places, never ruled
  closed and never named. This ADR does not fix it and it is now the sharper of the two gaps, since
  the closure's is closed.
