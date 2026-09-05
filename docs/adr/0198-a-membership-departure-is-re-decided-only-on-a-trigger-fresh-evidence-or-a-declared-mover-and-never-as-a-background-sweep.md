# ADR-0198: A membership departure is re-decided only on a trigger — fresh evidence, or a declared mover — and never as a background sweep

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1315 ADR gaps: internal/queue (2/7)](https://github.com/winniel123/verge-asm/issues/1315), gap 4
- **PR that deleted the comment:** [#1314](https://github.com/winniel123/verge-asm/pull/1314)
- **Sibling of, and not ruled by:** [ADR-0006](./0006-subjects-leave-by-measurement.md). That ADR rules that a subject leaves the estate only because something measured its absence, and that verge-asm ships no decay. It rules out a **clock and a counter**. It does not rule out a background pass, and it calls membership a Derived view over the latest observation per facet — a definition of the value, not a rule about when the value is recomputed
- **Supplies a premise for:** [ADR-0133](./0133-an-address-exclusion-is-a-limb-of-the-custody-derivation.md). That ADR rules that an address exclusion is a limb of the `Custody` derivation and cuts the `Seed` limb alone. Its §8.1 reasons *from* this rule — "a withdrawal scoped to the subjects a batch observed — the rule `foldEstateTransitions` applies to Names — could therefore never reach it" — and never rules it
- **Supplies a premise for:** [ADR-0134](./0134-a-seed-withdrawal-is-recorded-by-a-tombstone-because-the-mover-does-not-survive-the-act.md). That ADR rules that withdrawing a `Seed` writes a tombstone, because the delete destroys the mover the fold would read. Its §5 repeats the same premise — "a fold scoped to the subjects a batch observed could never reach the address" — and its §5.1 rests on `foldEstateTransitions` deciding departures for Names alone
- **Supplies a premise for:** [ADR-0135](./0135-a-name-seed-withdrawal-states-one-act-and-its-tombstone-carries-the-domain-alone.md). That ADR rules that a name `Seed` withdrawal states one aggregate `Narrowing` and its tombstone carries the domain alone. Its last Alternatives row refuses to let the ordinary fold decide, because "the fold would need a background sweep of the estate"
- **Rests on:** [ADR-0111](./0111-a-span-cites-the-batch-that-folded-it.md), which rules that a `Span` cites the `Batch` that folded it. A closure therefore needs a batch in hand, which is why no web handler closes a timeline and why every trigger is resolved inside a batch transaction
- **Rests on:** [ADR-0087](./0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md), which rules that a closure records the ground it rests on and that there are exactly three grounds. A sweep cannot name a ground, which is §3's argument

## Context

`internal/queue/membership.go:68` carried this, until [#1314](https://github.com/winniel123/verge-asm/pull/1314) deleted it:

```go
// It is scoped to the Names the batch actually observed (a `resolution` fold): a
// membership decision is only re-made where fresh evidence about the subject
// arrived, never as a background sweep of the whole estate.
```

The compressed survivor sits at `internal/queue/membership.go:39`, on the loop head it explains. It
is uncited, because nothing states the rule.

### The code is exactly as the comment says

`foldEstateTransitions` (`internal/queue/membership.go:38`) iterates
`observedResolutionNames(obs)` — the deduplicated `Subject` of every observation in this batch whose
`Facet` is `resolutionwalk.FacetResolution` (`internal/queue/membership.go:103`). For each such Name
it lists the open spans, calls `decideNameDeparture`, and closes the timelines only where that
returns `left`. It has one production caller, `internal/queue/worker.go:432`, inside the batch
transaction.

There is no other route by which a Name's membership is re-decided. `estate.AddressClosure` has no
production caller, a fact ADR-0134 §5.1 already measures.

### Three ADRs already rest on the rule, and none rules it

This is the third shape [`comment-policy.md`](../spec/comment-policy.md) §8.3 records: a source names
the rule while its own Decision never rules it.

| ADR | Where it uses the rule | What it rules instead |
| --- | --- | --- |
| [ADR-0133](./0133-an-address-exclusion-is-a-limb-of-the-custody-derivation.md) §8.1 | "§3 stops enumerating an excluded address, so no observation about one need ever arrive again. A withdrawal scoped to the subjects a batch observed — the rule `foldEstateTransitions` applies to Names — could therefore never reach it." | That an address exclusion cuts the `Seed` limb alone, and that its withdrawal is driven from the declaration side |
| [ADR-0134](./0134-a-seed-withdrawal-is-recorded-by-a-tombstone-because-the-mover-does-not-survive-the-act.md) §5, §5.1 | "a withdrawn scope stops being enumerated, so a fold scoped to the subjects a batch observed could never reach the address"; "`foldEstateTransitions` decides departures for **Names**" | That a `Seed` withdrawal writes a tombstone, spent only once exhausted |
| [ADR-0135](./0135-a-name-seed-withdrawal-states-one-act-and-its-tombstone-carries-the-domain-alone.md) Alternatives | "Keep the name limb enumerated and let the ordinary fold decide … The fold would need a background sweep of the estate, which ADR-0133 §8.1 and ADR-0134 both rejected for want of a mover." | That a name `Seed` withdrawal states one aggregate `Narrowing` |

Each of the three takes the rule as given in order to reach a decision about something else. Each
argues *because the observation-scoped fold cannot reach this act, the act needs its own fold*. That
argument is only sound if the observation scope really is the rule, and no accepted document says it
is.

`ADR-0135`'s row also mis-attributes the refusal slightly: ADR-0133 §8.1 and ADR-0134 reject a sweep
**for want of a mover**, which is an attribution ground. The rule the deleted comment states is a
**trigger** ground — fresh evidence, or nothing. The two grounds agree in every shipped case and are
not the same argument, and §3 below keeps them distinct.

### ADR-0006 does not suppress this

[ADR-0006](./0006-subjects-leave-by-measurement.md) is the nearest source and it rules a different
thing. Its subject is **decay**: "verge-asm ships no decay. Nothing ages out, no clock and no counter
retires a subject." It refuses opportunity-counted decay because under a wildcard the opportunities
are "infinite and all worthless", and it refuses wall-clock decay as "the cardinal sin twice over".
Every argument it makes is about time or about a count of missed chances.

A background sweep is neither. A sweep that ran every fold, re-evaluated every open Name against the
current `Seed` and exclusion corpora, and closed the ones no limb covers would introduce no clock and
no counter. ADR-0006 would permit it. ADR-0006 also calls membership "a Derived *view* over the
latest observation per facet", which fixes what the value **is** and says nothing about when it is
recomputed. A view can be recomputed on any schedule.

### The estate fold is one of five, and the other four are not observation-scoped

The batch transaction runs five folds in order (`internal/queue/worker.go:419-443`):

| Order | Fold | What it iterates | Trigger |
| --- | --- | --- | --- |
| 1 | `foldEdgeFanoutObservations` | this batch's observations | fresh evidence |
| 2 | `foldObservationsIntoSpans` | this batch's observations | fresh evidence |
| 3 | `foldEstateTransitions` | the Names this batch observed | fresh evidence |
| 4 | `foldNameSeedWithdrawals` | pending name `seed_withdrawal` tombstones | a declared mover |
| 5 | `foldAddressExclusionWithdrawals` | the live `exclusion` corpus | a declared mover |
| 6 | `foldSeedWithdrawals` | pending address `seed_withdrawal` tombstones | a declared mover |

`internal/queue/worker.go:435` carries the survivor that names the second trigger:

```go
// A withdrawn Seed stops its Names being enumerated, so a batch-scoped fold misses them (#1045).
```

**The rule stated over the observation scope alone would therefore be wrong**, and it would appear to
contradict three accepted ADRs. Folds 4, 5 and 6 re-decide membership for subjects this batch never
observed. What they are not is a sweep: each iterates a bounded set of movers the operator wrote, and
each closes only what it can attribute to one. `composeAddressWithdrawals`
(`internal/queue/withdrawal.go:44`) refuses to close a span it cannot attribute to a declared act.

Both triggers share one property, and it is the property that matters: **something new arrived.**
Either a measurement arrived about the subject, or a declaration arrived about the scope. Neither
fold ever asks "what is still open?"

## Decision

> **A membership departure is re-decided only where a trigger arrived, and there are exactly two
> triggers. Fresh evidence about the subject — an observation this batch carries — re-decides that
> subject. A declared mover the operator wrote — an `exclusion` row, or a `seed_withdrawal` tombstone
> — re-decides the scope that mover names. Nothing else re-decides membership. No pass over the whole
> estate, on any schedule, ever asks whether an open subject should still be open.**

### 1. The measurement trigger is the observation scope, and it is per subject

`foldEstateTransitions` iterates the Names this batch observed and nothing else. A Name the batch did
not resolve is not looked at, its open spans are not listed, and `decideNameDeparture` is not called
for it.

This is the limb the deleted comment states and the limb ADR-0133 §8.1 and ADR-0134 §5 name when they
explain why their own acts need a separate fold.

The scope is the batch's own observations, not the batch's declared scope. A batch that intended to
resolve a Name and failed carries no observation about it, so it re-decides nothing about it. That
follows from [ADR-0027](./0027-a-source-may-admit-without-observing.md)'s separation of admission
from observation and needs no new rule.

### 2. The declaration trigger is a mover, and it is per act

An operator act that narrows the aperture does not produce an observation, and it stops the affected
subjects being enumerated in the same transaction. ADR-0133 §8.1 measures this: "§3 stops enumerating
an excluded address, so no observation about one need ever arrive again."

So the act itself is the trigger. The three withdrawal folds each read a bounded mover set — the live
`exclusion` corpus, or the unconsumed `seed_withdrawal` tombstones — and act only on the scopes those
movers name. [ADR-0134](./0134-a-seed-withdrawal-is-recorded-by-a-tombstone-because-the-mover-does-not-survive-the-act.md)
exists because a `Seed` delete destroys its own mover, and the tombstone is the mover it writes in its
place. That ADR is the proof that this limb is a real second trigger rather than a special case of the
first: the product added a table so the trigger would exist.

The two triggers are disjoint in the shipped code but the rule does not require them to be. A subject
can be re-decided by both in one batch. `internal/queue/worker.go:443`'s ordering handles that: the
withdrawal folds run last, and an address a prior fold already closed is no longer open, so it is
never counted or attributed twice.

### 3. A background sweep is refused, and the ground is that it cannot name what moved

The refusal is not about cost. A sweep over the open span corpus is cheap and needs no migration,
which is the only merit ADR-0134's Alternatives table grants it.

**A sweep cannot name a mover, so its closure cannot record a ground.**
[ADR-0087](./0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md) fixes three
closure grounds and keeps them distinct. A sweep that finds an open Name no `Seed` covers and no
resolution cites cannot tell a `descoped` departure from an `uncited` one, so it blurs two of the
three. ADR-0134's Alternatives row states exactly this and gives the operator's side of it: they
cannot trace the closure back to anything.

**A sweep also makes the estate move when nothing happened.** ADR-0064 rules that a message names what
moved and reads it from the fold. A departure a sweep produced has no cause to name, so either the
message lies about a cause or the departure fires no message and the operator loses the trace of a
subject leaving. ADR-0006's *the cardinal sin* — "reporting the passage of time as movement" — is the
same failure reached by a different route, which is why ADR-0006 is a sibling here and not a parent.

**And a sweep re-opens what it closes.** ADR-0133 §8.1 and ADR-0134 §4 both measure this on the
custody-extension survivor: close an address a limb still holds, and the next batch reopens it and the
one after closes it again — "a `descoped` departure and a `Narrowing` message every cadence, for
ever". A sweep has no survivor test unless it re-derives every limb, at which point it is not a sweep
but a full re-derivation of the estate on every fold.

### 4. Every trigger is resolved inside the batch transaction

[ADR-0111](./0111-a-span-cites-the-batch-that-folded-it.md) rules that a `Span` cites the `Batch` that
folded it. A web handler holds no batch, so no operator act closes a timeline at the moment it is
performed. Both triggers therefore land on the next completed job.

The accepted bound is ADR-0133 §8.1's and ADR-0134 §5's, unchanged and restated here so it is not
lost: **an estate running no jobs holds its spans open until one completes.** That is a latency, not a
sweep, and it is the price of the closure citing a real batch.

### 5. What this rule does not reach

- **What `decideNameDeparture` decides.** This ADR rules *when* the question is asked. The answer —
  which open spans mean a Name left, and under which ground — is ruled by
  [ADR-0006](./0006-subjects-leave-by-measurement.md), [ADR-0087](./0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md)
  and `CONTEXT.md`'s disjunctive membership rule.
- **Which limbs keep a subject in the estate.** ADR-0133 §1 and ADR-0134 §4 rule the survivors.
- **The value fold.** `foldObservationsIntoSpans` folds a reading into a timeline.
  [ADR-0007](./0007-drift-is-a-timeline-of-spans.md) rules it. It is observation-scoped for its own
  reasons and this ADR does not restate them.
- **Retention.** [ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md)
  runs passes over corpora by age of readership. A retention pass deletes rows and decides no
  membership, so it is not a sweep in this rule's sense.
- **A re-derivation for a different purpose.** A future need to recompute a derived view over the whole
  estate is not forbidden. What is forbidden is such a pass **closing a timeline**.

## Consequences

- **This ADR changes no Go code.** All five folds already behave as ruled.
- **ADR-0133, ADR-0134 and ADR-0135 are left alone, and none is amended.** Each rests on this rule as
  an unstated premise and none contradicts it. This ruling supplies the premise they already assume
  and changes nothing any of them says. The relation rows above name all three, so a reader arriving
  from any of them finds the premise, and a reader arriving here finds the three consumers.
- **No [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
  withdrawal is owed, at any of the three sites.** ADR-0058 requires a withdrawal where a new ruling
  changes what a reader may believe about an earlier ADR. Read alone and in the present tense, every
  sentence in ADR-0133 §8.1, ADR-0134 §5 and §5.1 and ADR-0135's Alternatives table stays true and
  still specifies a mechanism that exists. **A later reader should not go looking for a withdrawal
  here.**
- **The rule is stated in one place.** Amending three accepted ADRs to add a premise none of them
  disputed would put the rule in three documents that can then drift apart, and would spend three
  reviews on text that already reads correctly.
- **ADR-0135's Alternatives row keeps a narrower attribution of the refusal.** It says ADR-0133 §8.1
  and ADR-0134 rejected a sweep "for want of a mover". That is their ground and it is stated
  accurately. §3 here adds the trigger ground beside it. No correction is owed.
- **Nothing enforces this.** No check fires on a query that reads the open span corpus without a
  trigger. Review carries the rule, and the shape to watch for is a fold whose iteration source is a
  corpus rather than this batch's observations or a mover set.
- **`internal/queue/membership.go:39`'s survivor gains a citation.** It is uncited today. The edit is
  recorded in `docs/adr/.pending/1315.md`.
- **`CONTEXT.md` gains nothing.** It already states what membership *is*, in the **Narrowing receipt**
  entry and in the disjunctive rule for an `Address`. When a derived value is recomputed is a fold
  property, not a domain term.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Amend ADR-0133, ADR-0134 and ADR-0135 to state the premise each relies on** | Each of the three rests on the rule and none contradicts it, so ADR-0058 is not engaged: no reader of any of the three currently believes anything this ruling makes false. The amendment would write one rule into three accepted documents, where a later change to one leaves the other two stating the old version. It also spends three reviews to add a sentence nobody disputed |
| **State the rule over the observation scope alone**, as the deleted comment does | It is true of `foldEstateTransitions` and false of the estate. Three of the five folds in the same transaction re-decide membership for subjects the batch never observed, driven from the declaration side. A rule that named only the first trigger would read as a contradiction of ADR-0133 §8.1, ADR-0134 §5 and ADR-0135, and a session applying it would delete the three withdrawal folds |
| **Rely on [ADR-0006](./0006-subjects-leave-by-measurement.md)** | Its subject is decay, and every argument it makes is about a clock or a count of missed corroboration opportunities. A background sweep uses neither, so ADR-0006 permits one. Its "membership is a Derived view over the latest observation per facet" fixes what the value is, and a view can be recomputed on any schedule |
| **A background sweep of the open span corpus each fold** | It cannot name a mover, so its closure cannot record one of ADR-0087's three grounds and blurs `descoped` into `uncited`. ADR-0064's message would then have no cause to name. It also re-opens what it closes over any subject a survivor limb still holds — a `descoped` departure and a `Narrowing` message every cadence, for ever — unless it re-derives every limb, at which point it is a full estate re-derivation and not a sweep |
| **A periodic sweep on a slower cadence than the batch** | Every objection above holds unchanged, and it adds ADR-0006's cardinal sin back on top: the interval becomes a threshold we chose, sitting inside the comparison path, reporting the passage of time as movement |
| **Close the timelines in the web handler at the moment the operator acts** | Refused by ADR-0111 and already refused by ADR-0133 §8.1 and ADR-0134's Alternatives table. A closure cites the folding batch and a handler holds none. It would also make the act non-atomic with the fold that reads its mover |
| **Drive the measurement trigger from the batch's declared scope rather than its observations** | A batch that intended to resolve a Name and failed asserts no absence. Scoping by intent would let a failed job produce a departure, which is the shape [#1316](https://github.com/winniel123/verge-asm/issues/1316) gap 2 records separately: a failed job records an empty scope, never the attempted one |
| **State the rule in `CONTEXT.md` beside the membership definition** | `CONTEXT.md` states what a subject's membership is and which limbs hold it. When the derivation is recomputed is a property of the fold, and putting it in the glossary would invite a reader to take the recomputation schedule as part of the model's definition |
