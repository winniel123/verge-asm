# ADR-0097: `returned`'s predicate composes every witness a presence read rests on — one Break among them voids it for the whole subject

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#148 What is returned's predicate when a subject's timelines closed under different vectors?](https://github.com/winniel123/verge-asm/issues/148)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0082](./0082-a-withdrawn-subjects-timelines-close-and-the-withdrawn-period-is-on-no-timeline.md)
ruled the mechanism for a single resolution timeline: a withdrawn subject's timeline closes, the
period between closure and return holds no object at all, and the closed span and the reopening
are therefore consecutive spans on **one** timeline — a `Transition` by
[ADR-0007](./0007-drift-is-a-timeline-of-spans.md)'s own definition, unless a `Break` sits between
them, in which case the reopening has nothing legally before it and the membership message fires
reading `appeared`. [ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md)
wrote the same mechanism first and flagged it as thin. ADR-0082 confirmed it. Both write the
sentence for one timeline, and both name the gap in their own thin-ground sections: *"a subject
holds many timelines, at many `(vantage, source)` keys, whose last spans may have closed under
different vectors."*

Two decisions since have made the gap load-bearing rather than hypothetical.

- **[ADR-0080](./0080-a-vantage-composition-is-cross-class-or-class-scoped-and-only-one-takes-a-quantifier.md)**
  settled that `Name` membership reads a **set** of per-vantage `resolution` timelines, not one —
  a `Vantage composition`, cross-class, with no quantifier, where every class must hold a current
  value and **agree**. Disagreement is not evidence of absence. It is `not-evaluable`. A v1 install
  running the optional external vantage ([#14](https://github.com/winniel123/verge-asm/issues/14))
  therefore holds **two** `resolution` timelines per `Name` — internal and internet — and
  membership's presence read is a joint fact about both, not a fact read off either alone.
- **[ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md)** settled
  that each of those timelines' own vector is not one leaf but two — `resolution-walk` and
  `wildcard-discrimination`, which can now bump independently — so *"closed under a different
  vector"* has a second, per-timeline way to happen that ADR-0082 never considered. ADR-0086's own
  closing line hands this ticket the question exactly: *"the vector now has two members that can
  move independently — so `closed under a different vector` has a second way to happen. #148 owns
  it, and it reads this ruling rather than the other way round."*

Neither prior ADR is wrong about what it ruled. Both are silent about what happens once the thing
they called *the timeline* turns out to be a **set**, and a set needs a composition rule of its
own — the exact gap [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s leaf table
and ADR-0080 exist to close everywhere else in the model, now reached at the one place nobody had
looked yet: the `Transition`, not the value.

### Not to be re-derived

- ADR-0082's mechanism for one timeline is untouched and is the base case here, not a competing
  answer.
- [ADR-0087](./0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md)
  already added a second necessary condition for `returned` — the closure's ground may not be
  `descoped` — orthogonal to this ADR's condition and composed with it, not replaced by it.
- This does not reopen ADR-0086's decision table, its pin block, or [#143](https://github.com/winniel123/verge-asm/issues/143)
  / [ADR-0085](./0085-an-obligation-with-no-failing-test-has-no-owner-and-a-boundary-needs-a-row-on-each-side.md)'s
  golden-corpus rows. It settles how those rows' comparisons are applied once a presence read draws
  on more than one of them at once.

## Decision

**`returned`'s predicate is a conjunction over every witness the composed presence read actually
rests on — the same witnesses, chosen by the same quantifiers, that `Vantage composition` already
uses to read the current value. A Break on any one of them voids `returned` for the whole subject,
not only for the timeline it sits on.**

| Concern | Decision |
| --- | --- |
| What a presence read composes | A **set** of per-vantage `resolution` timelines, chosen by [ADR-0080](./0080-a-vantage-composition-is-cross-class-or-class-scoped-and-only-one-takes-a-quantifier.md)'s own quantifiers: **existential** within a class (any one available vantage of that class witnesses it), **agreement** across classes (every class's witness must concur) |
| What `returned` additionally requires, beyond ADR-0082's single-timeline case | **No `Break` on any witness the composed read is currently relying on.** A witness is Break-free where its immediately-prior closed span and its new open span share **both** vector components — [ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md)'s `resolution-walk` version **and** `wildcard-discrimination` version, unchanged |
| Which witnesses count | Exactly those the presence read is using to conclude *present* right now — one per class (the existential witness), across every class the cross-class composition needs. A class with no available vantage contributes no witness and is a missing term, per ADR-0080 rule 3, not a vacuous pass |
| One Break among several clean witnesses | **Voids `returned` for the subject.** The subject re-enters reading `appeared`, exactly as ADR-0082 rules for the one-witness case — this is that rule applied once per witness and conjoined, not a new rule |
| Why conjunction and not "check the deciding witness alone" | There is no single deciding witness to privilege. Cross-class composition itself concludes *present* only on **agreement** among all classes' witnesses — none is dispensable to the conclusion — so the Transition-generalisation inherits the same conjunction the value-composition already uses, at no new granularity |
| Interaction with ADR-0087's closure-ground condition | **Composed, not replaced.** `returned` now requires **both**: no `Break` on any witness (this ADR), **and** the closure's recorded ground was not `descoped` (ADR-0087). Either failing alone reads `appeared` |
| Does this reopen the single-vantage case | **No.** One class, one vantage — the modal v1 install, over 99% of them ([#26](https://github.com/winniel123/verge-asm/issues/26)) — has exactly one witness, and the rule collapses to ADR-0082's sentence exactly |
| Does this change what is stored | **No.** No new object, value, field or timeline. The witness set is read at evaluation time from `Vantage composition`'s own machinery; the conjunction is computed, not stored |
| Golden-corpus consequence | **None beyond what ADR-0086 already priced.** A leaf's version is one release-wide fact; the corpus protects the leaf, not the vantage. What moves here is only how many Break-checks a single return event runs — once per witness — never what a corpus row pins |

## Rationale

### The set was already named; only its consequence for `Transition` was left unwritten

ADR-0080 did the hard part already: it enumerated exactly which timelines a presence read
composes and by which quantifier, precisely so that a later session would not have to re-derive
the population. `Vantage composition`'s existing rule — *existential for presence within a class,
agreement across classes* — answers "which values does the read use" in full. What it does not
answer, because it was written for **values**, is "which of those same timelines' **comparability**
does the read's *history* depend on." That is a different question about the same set, and this ADR
answers it by pointing the identical quantifiers at comparability instead of at value.

This is not a new design choice dressed as a derivation. It falls out of what a cross-class
composition **is**: *"every `Vantage class` the install runs must hold a current value, and they
must agree."* If any one of those classes is currently supplying a value the model cannot legally
compare to its own past — a Break sitting on that one witness — then the sentence *"the subject was
absent, and now every witness agrees it is present, continuously"* is not fully assertable, because
one of its own conjuncts cannot be assessed as continuous at all. The read can still conclude
*present* — reading a current value never needed comparability, which is why membership composition
is untouched by any of this — but the **claim that this is the same subject picking up where it
left off** is exactly the claim a Break is built to refuse, and refusing it on a subset while
asserting it whole is the *one fact in several representations* defect this map has refused
everywhere else it has found it.

### Why AND, not "check the timeline that flipped"

The tempting shortcut is to find whichever single witness caused the composed value to change from
absent to present and check only its Break status, on the theory that the other witnesses were
"already present" and contribute nothing new. It fails on the composition's own terms.

Cross-class agreement does not have a distinguished "deciding" class. It is defined as *every*
class's witness concurring. A class that was already reading non-`NameError` before the others
caught up is not a bystander to the conclusion, it is one of the terms the conclusion is **made
of**, exactly as much as the witness that changed last. Treating it as inert because its value did
not move on this particular fold conflates *the value that changed* with *the fact that grounds the
conclusion* — the same distinction [ADR-0010](./0010-exposure-composes-two-reaches.md) draws
between a leg and a projection. A rule reads a leg, never a state. A composed presence read
composes its witnesses, not its deltas.

The one-timeline case already shows why: ADR-0082 does not ask whether the timeline's value
*changed* across the withdrawal, it asks whether the timeline's two spans sit under one vector.
Generalising that test by asking it once per contributing witness, and requiring it to pass on
every one that the value-read used, is the same test at the granularity the model already fixed —
not a new invented threshold.

### Why an existential-within-class witness, and not every vantage in the class

A class with several vantages composes existentially for a presence claim — *"one vantage
receiving an answer is enough to establish that the answer was served."* Extending this ADR's
conjunction to every vantage in every class, rather than to the existential witness alone, would
silently import a **unanimous** reading into a place ADR-0080 deliberately kept existential, making
`returned` harder to earn than the presence conclusion it rides on. That is backwards: a `Break` on
a vantage the composition is not currently relying on cannot be evidence against continuity, because
the model never asked that vantage to testify. The conjunction is over exactly the witnesses
`Vantage composition` selects, at the same quantifier, for the same reason it selects them there.

### Why the two are additive rather than one subsuming the other

ADR-0087's condition (no `descoped` closure) and this ADR's condition (no Break on any relied-upon
witness) fail for different reasons and neither predicts the other. A `descoped` closure can have a
perfectly clean, single-vector return waiting behind it — an operator narrows an address-scope
declaration and widens it back within one release, no leaf moving at all — and it still must not
read `returned`, because the intervening period was an aperture narrowing, not a fact about the
world, and `appeared`/`returned` are membership's family while a scope act is aperture's
(`revealed`). Conversely a `measured-absent` closure with a clean scope story can still cross a leaf
bump on one witness while the corpus was silent on the rest, in which case ADR-0087's condition is
satisfied and this one is not. The predicate is their conjunction because each guards against a
distinct way the model could overclaim continuity it does not have.

### What this buys, stated rather than oversold

Nothing is priced differently from what ADR-0086 already paid for. That ADR already established
that a `wildcard-discrimination` bump breaks `resolution` estate-wide exactly as a `resolution-walk`
bump does, on every witness it touches. This ADR does not add a new way for a leaf bump to hurt. It
states precisely which witnesses a *given return event* must check, so that an implementation
evaluating `returned` for a two-class install knows to look at both `resolution` timelines rather
than one, and knows a Break on the vantage that was not the one whose value flipped still costs the
word.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md)'s `Transition` entry is amended at the site that specifies the
  single-timeline mechanism**, per
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md): read
  alone and in the present tense, *"a withdrawn subject's timeline is closed, so a Break between the
  withdrawal and the return leaves the reopening with nothing legally before it"* names one timeline
  where a presence read may compose several. The sentence's conclusion is unchanged for the case it
  was written for (one witness) and gains the conjunction for the case ADR-0080 later made real
  (several).
- **No new object, value, field, message class or census member.** The conjunction is computed at
  evaluation time from `Vantage composition`'s existing witness selection. Nothing is stored beyond
  what ADR-0082 and ADR-0086 already store.
- **The golden corpus's job is unchanged in kind and unchanged in size.** [ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md)'s
  two pin blocks (`resolution-walk` 27, `wildcard-discrimination` 19) still protect exactly the leaf
  versions they protect. This ADR does not add rows, because a leaf's version is one release-wide
  fact regardless of how many vantages read it. What changes is only that an implementation checking
  `returned` for a multi-class subject runs that same per-leaf comparison **once per witness**
  instead of once.
- **The two membership-family conditions for `returned` are now stated as a pair.** No `Break` on
  any witness the presence read relies on (this ADR), and the closure's ground was not `descoped`
  ([ADR-0087](./0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md)). Both
  must hold. Either failing reads `appeared`.
- **The single-vantage-class install is untouched.** Its presence read has exactly one witness, so
  the conjunction is over a set of size one and collapses to ADR-0082's original sentence with no
  observable difference — which is also the modal v1 install ([#26](https://github.com/winniel123/verge-asm/issues/26)).

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Check only the witness whose value flipped from `NameError`** | Conflates the value that changed with the fact the composed conclusion is made of. Cross-class agreement has no dispensable term — every class's witness is jointly necessary for the conclusion, so every one of them is equally load-bearing for whether that conclusion can be narrated as *continuous* |
| **Require every vantage in every class to be Break-free, not just the existential witness** | Imports a unanimous reading into a place ADR-0080 fixed as existential for presence, making `returned` harder to earn than the presence conclusion it rides on. A vantage the composition never consulted cannot be evidence against continuity |
| **Treat disagreement among witnesses' Break status as itself `not-evaluable`, mirroring cross-class value disagreement** | Overloads a value-composition outcome onto a Transition question. `not-evaluable` is what a presence *read* returns when classes disagree on the *value*; comparability is a different axis, and the model already has the right word for "cannot legally compare" — `Break` — with no need to borrow another term's vocabulary |
| **Leave the single-timeline sentence as the general rule, and let implementations discover the multi-witness case on their own** | This is the state before this ticket, and it is exactly what [ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md) declined to leave standing when it named the gap explicitly and routed it here rather than silently assuming it away |
| **Score `returned` as a fraction — some witnesses clean, some broken — and render a partial-confidence transition** | Invents a state between `returned` and `appeared` that the model refuses everywhere else it has been tempted (ADR-0006's *no `dormant`, no `stale`, no `unconfirmed`*; ADR-0007's refusal of hysteresis). The honest binary is the smaller claim when any part of the evidence cannot support the larger one |

## Where this is thin, stated rather than smoothed

- **A vantage class enabled between a subject's withdrawal and its return is not settled here.**
  Such a class never held a prior timeline for this subject at all — nothing to Break, only an
  opening — so it is not a "witness with a Break" in this ADR's sense, but whether a presence read
  may conclude `returned` while depending on a witness that has no history is a question this ADR
  does not reach. It is closer to `revealed`'s territory (a widened `Vantage class` input,
  [ADR-0007](./0007-drift-is-a-timeline-of-spans.md)'s amendment) than to this one, and it stays
  fog rather than being folded in here.
- **No install has run two vantage classes against a subject that withdrew and returned.** The
  conjunction is argued from ADR-0080's and ADR-0086's own stated composition rules, not measured
  against a live multi-witness return.
