# ADR-0080: A vantage composition is cross-class or class-scoped, and only the class-scoped kind takes a quantifier

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#138 `resolution` now has two compositions and neither is named](https://github.com/winniel123/verge-asm/issues/138)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0007](./0007-drift-is-a-timeline-of-spans.md) keys a timeline
`(subject, facet, discriminator, vantage, source)`. So any Derived value that reads a per-vantage
facet across more than one vantage performs a step nothing in the model has ever named: it turns a
**set** of per-vantage values into the **one** value a predicate reads.

`resolution` now carries two of those steps.

- [#48](https://github.com/winniel123/verge-asm/issues/48) ·
  [ADR-0020](./0020-a-conflict-needs-two-enumerable-sources.md) reused
  [ADR-0006](./0006-subjects-leave-by-measurement.md)'s membership rule — *"withdrawal needs every
  available vantage to agree, composing `Availability` exactly as `Exposure` does"* — for
  `zone-declared-name-returns-name-error` and `resolved-name-absent-from-zone`, and refused an
  asymmetric alternative because it *"fires `resolved-name-absent-from-zone` on every internal-only
  name in a split-horizon estate"*.
- [#128](https://github.com/winniel123/verge-asm/issues/128) ·
  [ADR-0071](./0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md) §3 wrote a
  second one for `non-globally-reachable-address-resolved-from-internet` — restricted to the
  **internet** class and composed **existentially** — and handed the naming question on, flagging its
  own ground as thin: *"the existential composition across internet-class vantages is argued, not
  measured, and no prior ticket rules on how a presence claim composes."*

[#6](https://github.com/winniel123/verge-asm/issues/6)'s rule is why that is not tidiness: **every
seam is a place drift can be manufactured**, and a step whose output can move while the world does
not is [`Derivation`](../../CONTEXT.md)'s own definition of a thing that needs a version and a name.
The ticket asks whether the two should be enumerated the way `Reach` names its class composition.

Three facts settle it, and all three were found by writing the population out rather than by
reasoning about the two known members — the same exercise
[ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md) ran on domains, and it found the
same class of defect.

**#128's "both sit inside their own rules' leaves" is already false for one of them.** ADR-0006's
composition has three consumers and one of them is **not a rule**: `Name` withdrawal. A shared
object with three consumers, one outside the `Signal` vocabulary entirely, cannot be described as a
step inside a predicate.

**`Reach` names its class and never its quantifier.** `CONTEXT.md` defines it as *"what vantages of
one `Vantage class` found for one `Service`"*. Two internet-class probers that disagree have no
written answer — not in `CONTEXT.md`, not in [ADR-0010](./0010-exposure-composes-two-reaches.md),
not in [ADR-0017](./0017-exposure-needs-both-legs.md). The ticket's premise that `Reach` names its
class composition is **half true**, and the missing half is the one this ticket is about. Adopting
the `Reach` treatment would have imported the hole.

**Two v1 rules state no composition at all.** `lame-delegation` and `cname-target-name-error`
([#35](https://github.com/winniel123/verge-asm/issues/35)) read `resolution` and predate both
rulings. Nothing anywhere says how they compose.

## Decision

**A reader of a per-vantage facet declares a `Vantage composition`, and there are exactly two kinds.
Which kind is available is decided by whether the fact the reader is named for is scoped to a vantage
([ADR-0071](./0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md)); only the
class-scoped kind takes a quantifier, and the cross-class kind cannot express one.**

### Decision rule 1 — the two kinds

> **Cross-class.** Every `Vantage class` the install runs must hold a current value, and they must
> agree. Disagreement is **incommensurability, not evidence** — the classes are looking at different
> worlds, which is what split horizon *is* — so the composed value is `not-evaluable`. A class with
> no available vantage leaves the comparison unmade, so that is `not-evaluable` too. **No quantifier
> is expressible on this kind**, because there is nothing to quantify over: agreement is the whole
> of it.
>
> **Class-scoped.** One named `Vantage class`; the composition reads the available vantages **of that
> class alone**, and the reader states a **quantifier** over them, from a closed union of two:
> `existential` (any) or `unanimous` (every). Disagreement here is **variance, not
> incommensurability** — geo-DNS, per-query rotation, anycast — and the quantifier is what says which
> way variance falls. An empty set of available vantages in the named class is `not-evaluable`.

The asymmetry is the whole ruling and it is not a convenience. **Across classes, a difference is a
fact about our aperture; within a class, a difference is a fact about the authority.** The model
already holds both halves separately and had never joined them: `Vantage class` is *"which side of
the operator's boundary a `Vantage` sits on"* — a property of **our** deployment — while ADR-0070
**[measured]** the within-class case on real authorities, one wildcarded name drawing addresses from
**two disjoint pools at two vantages in one week**.

### Decision rule 2 — which kind a reader takes

> **A reader takes the class-scoped kind exactly where the fact it is named for is scoped to a
> vantage, and the cross-class kind everywhere else** — ADR-0071's test, unchanged and now doing a
> second job.

And within the class-scoped kind:

> **A presence claim composes `existential`; an absence claim composes `unanimous`.** One vantage
> receiving an answer is enough to establish that the answer was served; no number of vantages
> failing to receive one establishes that none exists, because a vantage that did not ask is not a
> vantage that got nothing.

### Decision rule 3 — an empty in-scope set is never vacuous

> **Both kinds return `not-evaluable` on an empty in-scope set.** Stated because `unanimous` over an
> empty set is vacuously **true**, and the vacuity is not hypothetical: read literally, *"every
> available vantage agrees on `NameError`"* **withdraws every `Name` in the estate** on the night every
> vantage goes unavailable. ADR-0006 blocks it one sentence later — *"one vantage down makes
> membership not-comparable rather than concluded from the survivor"* — but the phrase **"every
> available vantage" travelled onward without that sentence**, into ADR-0004, ADR-0020, ADR-0071 and
> `CONTEXT.md`'s `Name` entry. Under Decision rule 1 the vacuity is closed by construction rather than
> by a neighbouring sentence: a cross-class composition needs a value from every class, so an absent
> class is a missing term and never an empty conjunction.

### The population, written out

| Reader | Kind | Quantifier | Where it is written |
| --- | --- | --- | --- |
| `Name` withdrawal / membership | Cross-class | — | ADR-0006; **not a rule**, which is why the composition is not a step inside a predicate |
| `zone-declared-name-returns-name-error` | Cross-class | — | #48 · ADR-0020, unchanged |
| `resolved-name-absent-from-zone` | Cross-class | — | #48 · ADR-0020, unchanged |
| `lame-delegation` | Cross-class | — | **Unstated until now.** A lame delegation is a fact about the delegation, not about where you asked, so it is not vantage-scoped |
| `cname-target-name-error` | Cross-class | — | **Unstated until now**, same ground |
| `non-globally-reachable-address-resolved-from-internet` | Class-scoped, `internet` | `existential` | #128 · ADR-0071 §3, unchanged |
| `Reach`, within one class | Class-scoped, per class | `existential` | **Unstated until now.** See below |
| `sensitive-port-reached-from-internet` | Reads a `Reach` leg | — | Composes `Reach`'s, not `resolution`'s |

`Exposure` is **not** in this table and is not a `Vantage composition`. It composes two `Reach`
**legs** — Derived values that have already been composed — and it is a projection of a 2×2 rather
than a quantified read of a set (ADR-0010, ADR-0017). The distinction is worth keeping: `Reach` is
the within-class step, `Exposure` the across-class one, and only the first is what this ADR governs.

### `Reach`'s quantifier is `existential`, and the direction was already chosen

Filled here rather than handed on, because leaving the hole open in the flagship value while closing
it in a peripheral one is precisely the inconsistency
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) exists to
prevent. `Reach` is class-scoped by its own definition and its value `reached` is a **presence**
claim, so Decision rule 2 gives `existential` — and the direction was independently fixed long ago
by `Vantage class`'s stated failure direction, *"a false `exposed` is investigated, a false quiet
reading is not."* Unanimity would report a service reachable from one internet position and
geo-blocked at another as `not-reached`, which is false in the closed direction and would under-report
the value the product is named for.

**This moves no version and breaks nothing.** Nothing was specified, so there is no corpus row whose
output moves, and [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s bidirectional
gate is satisfied by a leaf that does not move. The map plans; this is a spec hole filled before
implementation, not a behavioural change.

### What is refused: the `Reach` treatment

**The two `resolution` compositions do not become named `Derivation` leaves, gain no subject, no
value space, no `Span` timeline and no place in any version vector.** They stay inside their readers'
leaves exactly as ADR-0071 §3 left them.

`Reach` can be a leaf because `reachability` has **one consumer shape** — every reader wants the same
question answered, *is it reachable from this class* — so one stored composed value serves all of
them. `resolution` has **two consumer shapes**: membership wants agreement across classes, #128's rule
wants existence within one. A single stored composed `resolution` would have to pick one and be wrong
for the other consumer, every time, on every `Name`. That is not a cost comparison; the object does
not exist to be minted.

### What is refused: two proper nouns for the two occupied cells

The ticket offers *name these two*. **Declined.** Naming the two known compositions as two nouns
names **cells**, and this project has twice ruled that you name the axis and read a leg — ADR-0010's
*"a rule reads a leg, never a state — a state enum is a projection, not the thing"*
([#32](https://github.com/winniel123/verge-asm/issues/32)), and
[#45](https://github.com/winniel123/verge-asm/issues/45)'s *which of the four unnamed `Exposure`
cells need names? — **none of them***. Two nouns would also have frozen the wrong shape: it is not a
2×2 with two cells occupied, it is **1 + n**, and the cell #128 rejected (*class-scoped and
unanimous*) and the cell #48 rejected (*cross-class and existential*) are not symmetric — the first
is legal and empty in v1, the second is **inexpressible**.

That last point is the load-bearing test of Decision rule 1 and it is worth stating plainly. #48
rejected *"unanimity to assert an absence, one vantage to assert a presence"* on a measured-shaped
argument about split horizon. Under this ADR that option is not rejected, it **cannot be written**:
a cross-class composition has no quantifier place to put `existential` into. A rule that turns a
prior ticket's rejected alternative into an inexpressible one is stronger than the ticket that
rejected it, and it is how [#110](https://github.com/winniel123/verge-asm/issues/110) said a closed
enumeration gets repaired — *by writing down what a limb denotes, never by widening the enumeration*.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) gains `Vantage composition`** under Derived, and two existing
  entries are amended at the site: `Reach` gains its quantifier, and `Name` loses the ambiguous
  *"every available vantage"*.
- **No new rule, no new facet, no new field, no new value space, no `Break`, no aperture input, no
  declared parameter, no message, no census member and no coverage member.** The **v1 rule set stays
  at seventeen**; no rule's leaf moves, because every composition this ADR writes down is either
  unchanged or was never written at all.
- **The version vector does not widen.** ADR-0071 §3's statement that both compositions sit inside
  their readers' leaves is confirmed rather than withdrawn — what changes is that a leaf must now
  **declare** which kind it takes.
- **A rule is still four parts.** The composition is not a fifth: its **class** half is a fact about
  the rule's `Predicate domain` — ADR-0024's table already carries *internet-class* in #128's domain
  column — and its **quantifier** half is a fact about the predicate. What was missing was not a
  part, it was the requirement to state both.
- **ADR-0071's handed-on item is discharged**, and its thin-ground flag is **partly** discharged: the
  existential read for a **presence** claim within a class now has a rule behind it rather than one
  ticket's argument, and it is corroborated by `Reach` independently landing on the same value from
  the safety direction. What is **not** discharged is that no v1 reader takes the `unanimous`
  within-class cell, so that half of the closed union is untested — recorded below.
- **`lame-delegation` and `cname-target-name-error` are placed**, closing a hole neither #35, #48 nor
  #128 recorded.
- **`Reach` gains a quantifier**, closing the same hole in the flagship value.
- **Nothing is said about which `Scan` covers `resolution`** — that is
  [#142](https://github.com/winniel123/verge-asm/issues/142)'s, and a currency bound decides which
  observations are *available* to a composition without deciding how the available ones compose.
  The dependency runs one way and this ADR does not pre-empt it.

### Stated costs

- **The cross-class guard is inert on a one-class install.** A cross-class composition over one class
  is that class's value, agreed with nobody. So `resolved-name-absent-from-zone`'s protection against
  split horizon — ADR-0020's *"a non-event by construction"* — **does not exist on the modal
  install** ([#14](https://github.com/winniel123/verge-asm/issues/14): internal, no outside prober),
  which is the install most likely to hold a split zone and most likely to have supplied a file.
  #48 stated the noise floor and did not state that its mechanism is contingent on running two
  classes. **Disclosed rather than fixed**: the fix is to make the rule class-scoped to `internet`,
  which would make it `not-evaluable` on the modal install and gut #48's own structural argument that
  it is the only consumer of the zone file's `enumerable` — *"a spine nothing reads is not a spine."*
  Handed on as a ticket rather than decided here.
- **The `unanimous` within-class cell is empty in v1.** The union is closed at two and only one
  member has an instance, so the closure is analytic rather than measured.
- **An existential composition names no vantage.** Where two vantages of a class disagree and the
  existential read fires, nothing in the message says which one saw it — #119's message carries
  counts, keys and one link and no rows. Latent while the modal install runs one prober per class,
  and the model caps neither.
  **Ruled by [#170](https://github.com/winniel123/verge-asm/issues/170): worth carrying, and
  carried at the drill-down beneath the composed value the message's link already points to —
  never in the channel payload, which stays exactly as #119 fixed it.** The `Vantage` is already in
  the timeline key ([ADR-0007](./0007-drift-is-a-timeline-of-spans.md)), so the object's own detail
  rendering must enumerate the per-vantage values a class-scoped composition read, keyed by vantage
  — the same treatment a cross-source conflict already gets under ADR-0007's *reported, never
  resolved*. No new field, no new stored object, no version move: a rendering obligation on
  whichever ticket specifies the object's detail page, not a new `Derivation` leaf. Not live in v1
  either way — the modal install still runs one prober per class — but no longer unspecified for
  the install that runs two.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Promote both to named `Derivation` leaves, as `Reach` is** | The ticket's literal question. `Reach` can be a leaf because `reachability` has one consumer shape; `resolution` has two consumers wanting different compositions of the same timelines, so one stored composed value is wrong for one of them on every `Name`. It would also widen the version vector and mint a Derived value with a subject and a timeline, which nothing here forces |
| **Mint two proper nouns, one per existing composition** | Names cells. ADR-0010 and #45 both already ruled that you name the axis and read a leg, and two nouns freeze a 2×2 that is really 1 + n — hiding that #48's rejected option is *inexpressible* while #128's is merely *empty* |
| **Leave them unnamed; a composition is a step inside a predicate** | False for one of the three consumers: `Name` withdrawal is not a rule and has no predicate. And it leaves `lame-delegation`, `cname-target-name-error` and `Reach`'s quantifier unstated, which is how the next session mints a fourth composition by accident |
| **Make the composition a fifth part of a rule** | [#53](https://github.com/winniel123/verge-asm/issues/53) held the line at four parts for `Predicate domain` and was right. The class half is already a domain fact and the quantifier half a predicate fact; nothing needs a new place to live |
| **A free 2×2 — class axis and quantifier axis independent** | The tempting shape, and it is refuted by `resolved-name-absent-from-zone`: a direction-only rule makes it existential, which #48 killed on split horizon. The correction is that #48's unanimity is a **coherence guard** rather than a quantifier — ADR-0020 says so in terms — so disagreement means different things at the two scopes |
| **Make the quantifier follow the safety direction rather than the claim's direction** | Same answer on every v1 instance, so it is untestable here, and it is the worse rule: *err loud* is a tie-breaker, while *a vantage that did not ask is not a vantage that got nothing* is a statement about what the evidence can carry. ADR-0071 reached for the safety direction because it had no claim-direction rule to reach for; it now has one, and the two agree |
| **Change `resolved-name-absent-from-zone` to class-scoped `internet`, closing the inert-guard cost** | It fixes the split-horizon hole on the modal install and makes a **third** rule dark there — and the modal install is exactly the one that supplies a zone file. It would leave the zone file's `completeness` with no consumer on the install that has one, which is the argument #48 shipped the rule on. Disclosed as a cost and handed on, not taken |
| **Defer `Reach`'s quantifier to its own ticket** | Closing the hole in a peripheral value while leaving it open in the flagship is ADR-0058's shape exactly — a superseded reading left live at the site that specifies it. The direction is forced by `Vantage class`'s own stated failure direction, so there is no decision left to defer |
| **Treat `Exposure` as a third `Vantage composition`** | It composes Derived legs rather than per-vantage observations, and it is a projection rather than a quantified read. Folding it in would make the term mean *any composition touching a vantage*, which is the overload the term exists to end |
