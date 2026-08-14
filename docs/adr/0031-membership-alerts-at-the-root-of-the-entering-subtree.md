# ADR-0031: A membership message fires at the root of the entering sub-tree — and a first run reveals rather than appears

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#63 Is `appeared` alertable, and what is the cause when an Address brings tens of Services with it?](https://github.com/winniel123/verge-asm/issues/63)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0006](./0006-subjects-leave-by-measurement.md) split appearance into `appeared` and
`returned` and ruled on **one** of the pair: `returned` alerts, `withdrawn` does not, because
decommissioning is the commonest intentional change in an estate.
[ADR-0014](./0014-only-revealed-generalises.md) settled `revealed`'s class and left `appeared`
exactly where ADR-0006 left it, which is unstated. Three later decisions then leaned on it.

- ADR-0014: an opening caused by neither the world nor our aperture is *"recorded, unnamed and
  unalerted — **the subject's own membership transition already carried that news, at the
  cause**"*.
- [ADR-0029](./0029-an-alert-fires-on-a-leg.md) §4: the worse readings of an internet
  `reached` → `not-reached` are each carried elsewhere, one of them being *"a service moving
  address, which is a `Service` `appeared` beside it"*.
- ADR-0029's forcing correction lists **membership drift** among the four things that keep a
  one-vantage-class install's alerting surface non-empty.

All three assume `appeared` is a message. None of them ruled it, and the third is the ruling
[#58](https://github.com/winniel123/verge-asm/issues/58) took yesterday.

### The fact that decides it, and it is textual rather than a base rate

**A subject that has just entered the estate has no transitions at all.** Every timeline
beneath it *opens*, and ADR-0014 is unambiguous that an opening emits no `Transition`. So on a
newly-entered `Service`:

- the internet `Reach` leg opens **at `reached`** where the port answers — it does not go
  `not-reached` → `reached`, so ADR-0029's flagship predicate does not match;
- `Exposure` opens rather than moving, which is ADR-0017's third consequence working as
  designed;
- `sensitive-port-reached-from-internet` opens at *fired*, which is not a `Signal` transition
  either.

A brand-new internet-reachable Redis therefore produces, under a rule that silences membership,
**no message of any kind**. That is the product's headline event arriving through the one door
its headline predicate cannot see, and it is why *`appeared` is simply `withdrawn`'s twin* — the
tidiest-looking answer — cannot be right.

### The second half: one `Address` mints tens of `Service`s

A `Service` is `(Address, port, transport)` and it is the subject reachability is measured
against, so it exists for probed ports that come back closed —
[ADR-0017](./0017-exposure-needs-both-legs.md)'s `unreachable` is *"both legs measured, neither
reached"* and needs a subject to be a verdict about, and
[ADR-0011](./0011-a-facet-is-six-parts.md)'s *"every **open** `Service`"* is only a meaningful
qualifier if closed ones are in the estate too. So one `Address` entering mints one `Service`
per `(port, transport)` in the recorded scope, most of them closed, and
[ADR-0007](./0007-drift-is-a-timeline-of-spans.md)'s *alert on the cause, record the
consequence* forbids a message each. That rule needs the cause **named**, and the chain runs
`Seed` → `Name` appears → `Name` resolves → `Address` appears → tens of `Service`s appear →
`Endpoint`s appear. Read literally, an entire first run is one message.

## Decision

**A membership message fires once, at the root of the entering sub-tree, and it carries the
census of what entered beneath it. A subject first observed under a widened aperture is not
`appeared` at all.**

**1. The root of an entering sub-tree is the entering subject whose own `Citation` points at
something already in the estate** — an existing subject, a `Seed`, or a `Batch`. Everything
that entered beneath it in the same fold is its consequence and is recorded, not messaged. One
cause, one message, exactly as ADR-0007 requires, with no count anywhere in the predicate.

**2. Only `Name` and `Address` can be roots, and this is a theorem rather than a second rule.**
The four subjects split on **who supplies the key**:

| Subject | Key | Supplied by |
| --- | --- | --- |
| `Name` | the FQDN | the world — a source names it |
| `Address` | the IP | the world — a resolution or a `Seed` names it |
| `Service` | `(Address, port, transport)` | the model — an `Address` already in the estate, and `verge-core`, which is ours |
| `Endpoint` | `(Name, Service)` | the model — two subjects already in the estate |

`Service` and `Endpoint` have keys built entirely from things the model already holds, so
neither can bring ground the model was not already accounting for, and neither can ever satisfy
§1 — a `Service`'s citation runs to its `Address`, an `Endpoint`'s to its `Name` or its
`Service`. **`Service` and `Endpoint` membership is recorded and is never a message, in any
direction**, and a message announcing one would be ADR-0007's second representation of one
fact.

**3. The message carries the census of every value the entry started.** This is
ADR-0029 §7's payload shape with a second producer, and for the identical reason: everything
beneath an entering subject *opens*, nothing compares, and a bare count cannot tell the operator
whether to get out of bed. It is computed once at the cause, is a description and never a
`Transition`, and carries no difference set.

> *`admin.example.com` entered the estate, cited by a `crt.sh` observation. It resolves to
> `203.0.113.9`, which is new. 214 `Service` timelines opened; 3 read `reached` from the
> internet — 22/tcp, 443/tcp, 6379/tcp — and `sensitive-port-reached-from-internet` opened
> *fired* on 6379/tcp. Nothing is compared.*

The 3 are **not** alerted individually. They are openings.

**4. A first run is not `appeared`. It is `revealed`, coverage class, one message.** Declaring
the first `Seed` moves the **custody gate**, and enabling the discovery sources moves the
**enabled source set** — two of the six enumerated aperture inputs — so the whole of a first run
is a zero-to-declared aperture widening, and
[ADR-0003](./0003-third-party-source-consent-bar.md)'s rule already governs it: *a subject first
observed under a widened aperture is not "appeared"*. It fires one coverage-class message under
*we changed how we look*, **trigger 1**, carrying §3's census. **No first-run special case is
needed and none is admitted** — a predicate reading *is this the first `Batch`?* would be a
state we invented, sitting in the notification path for a case the general rule already covers.
Adding a `Seed` later behaves identically, for the same reason.

**5. Nothing is added to the notification vocabulary.** `appeared` and `returned` are *the world
moved*, in the drift class they were always in. No fifth cause, no third trigger, no fourth
class, and no tenth member of the coverage class — §4 uses `revealed`, which is already member
two.

## Rationale

### Why the cause sits at the root, and not one hop up

One hop up is the `Seed`. It loses twice. It is a **Declared** act and its cause is *we changed
how we look*, a different cause in a different class which §4 already routes — so rooting an
appearance there files a world event under an observer cause. And every subject's `Citation`
chain terminates at a `Seed` by construction, so rooting there is not a rule about first run at
all: it collapses **every** appearance the estate will ever make into one message per `Seed`,
forever. That is [#22](https://github.com/winniel123/verge-asm/issues/22)'s refused suppression
arriving as a coalescing rule, and it is the reading the ticket flagged.

The two answers agree at the boundary, which is evidence the cut is on the right joint: on a
first run nothing is already in the estate, so §1's root walk runs all the way to the `Seed` and
yields one message — and §4 independently says that message is `revealed`, in the coverage
class. Same count, and §4 supplies the class §1 would have got wrong.

### Why not one hop down, at the `Service`

Because `Service` membership carries no measured fact. Given an `Address` in the estate and the
`Batch`'s recorded port scope, the set of `Service` subjects is **computed**. The port set is
`verge-core`, which [ADR-0009](./0009-verge-core-is-a-union.md) made a *definition*; narrowing
it does not withdraw a `Service` but stops feeding its timeline and opens a `Gap`; widening it
is an aperture input and yields `revealed`. So across its whole life a `Service`'s membership is
`Address` membership, restated. A message for it is a second representation of one fact.

### The version of *`Service` appeared* that is not a restatement is worse, not better

The ticket's sharpest bullet asks whether a **closed** `Service` appearing is an event at all,
noting that if it is not, the population that appears is smaller than the population that
exists. It is not an event — but the reason is that **no** `Service` appearing is an event, and
the two populations stay equal. Making them unequal would mean membership beginning when a port
answers, and that fails three ways at once:

- it makes membership a projection of `Reach`, so a port opening presents as `appeared` and a
  port closing as `withdrawn` — two representations of one fact, and the second one is
  *alertable* under ADR-0006 while the first is the very thing ADR-0029 ruled silent on the
  internal leg. The internal port opening would walk back in through membership, which is
  exactly what #63 was told not to permit;
- it deletes `unreachable`. ADR-0017's fourth state is *both legs measured, neither reached*,
  and a `Service` that never answered would not exist to hold it;
- it makes the `Batch`'s recorded scope stop meaning what [ADR-0005](./0005-scan-execution-model.md)
  says it means. A scope of `verge-core` licenses an absence claim over `verge-core`; a
  population defined by what answered licenses nothing.

### `Name` and `Address` both, and why kind is the wrong axis for the question

The ticket asked whether the answer is *the root* or *the shallowest subject kind that has its
own lifecycle*. Kind is the wrong axis, because the same kind is sometimes a root and sometimes
a consequence: a `Name` that a source has just admitted is a root, and the `Address` it resolves
to is that root's consequence — but an `Address` reached by a `Name` that has been in the estate
for months is a root in its own right, and `Address` is the one subject ADR-0006 says has **no
lifecycle of its own**. Rooting on kind therefore gets the two commonest cases in a mature
estate wrong in opposite directions.

Both kinds are needed, and each is the sole carrier for one real case:

- **`Name` only** loses the redeploy. An existing name re-pointing at a new address is the
  commonest way new ground enters a mature, cloud-resident estate
  ([#26](https://github.com/winniel123/verge-asm/issues/26)), and it is precisely where new
  ports become internet-reachable. No `Name` appears, so nothing would fire.
- **`Address` only** loses the vhost. A new name on an address already in the estate mints
  `Endpoint`s and therefore new `certificate` and `http-identity` timelines, and the operator
  thinks in names.

### What this does to #58, stated plainly

[#58](https://github.com/winniel123/verge-asm/issues/58)'s ruling rests on membership drift
being live on a one-legged install. **It is, and the ruling stands** — `Name` and `Address`
`appeared`, and `returned`, all fire there. Two of its sentences are nonetheless wrong as
written and are corrected in the amendment to ADR-0029:

- its Context bullet *"Membership drift runs — `Name`s, `Address`es, `Service`s and `Endpoint`s
  appear and withdraw"* is true of the **drift** and false of the **messages**; and
- its §4 carrier *"a service moving address, which is a `Service` `appeared` beside it"* names a
  message that does not exist. The carrier is an **`Address`** `appeared`, and it is
  **narrower** than #58 had it: where the service moves to an address already in the estate,
  there is no membership message, and the residue is a `resolution` `Transition` recorded on the
  `Name`'s own timeline.

That narrowing touches one of three carriers for one direction cut. The other two — a closing
custody gate and a `Vantage` becoming `unavailable` — are untouched.

### What this does to [#61](https://github.com/winniel123/verge-asm/issues/61)

[ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md), resolved in parallel with
this ticket, rejects a slower *reachability* tier as a replacement worked example for ADR-0014
with a claim about this ticket's subject: *"where the slower tier is what discovers the
`Service`, the subject enters the estate and its timelines open with it, so `appeared` carries
the news at the cause and the opening is not unnamed at all."*

**That sentence is withdrawn, and its conclusion gets stronger rather than weaker.** A `Service`
appearing is never a message under §2, so where a slower reachability tier discovers a `Service`
beneath an `Address` already in the estate, the root walk terminates at that `Address` and **no
membership message fires at all** — the opening is unnamed and unalerted after all, exactly as
ADR-0014 describes. So a slower reachability tier is not disqualified as ADR-0014's worked
example for the reason ADR-0028 gives; it is disqualified only by ADR-0028's own primary
argument, that `certificate`'s value space has no variant meaning *the port was shut*.

The correction cuts ADR-0028's way. It means scheduling `reachability` slowly would lose the
discovery news outright rather than having it carried by membership, which is a **cost** of the
alternative ADR-0028 rejected, not a relief. Nothing in ADR-0028's Decision moves; one
rejected-alternative sentence does.

### The constraint about damping is met by construction

No threshold and no count appears in any predicate here. §1 walks a `Citation` chain, which is
structure the model already holds; §2 falls out of the key shapes; §4 diffs a `Batch`'s recorded
scope. The counts all live in §3's census, which is payload. Coalescing and flap suppression
stay where ADR-0007 put them, in the notification patch.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) is amended in three entries.** `Subject` gains the
  who-supplies-the-key split, the kind restriction and the root rule — the better home for it
  than `Transition`, since ADR-0014's own line is that *membership is a property of a subject*;
  `Service` states that its membership is `Address` membership restated, so a port opening is a
  `Reach` move and never a membership event; `Reach` states
  that a leg **opening** at `reached` emits no `Transition` and is carried by the entering
  subject's membership message and its census, never by the flagship predicate.
- **ADR-0029 §4 and its Context bullet are corrected**, and its stated cost survives with a
  stronger reason: an internal port opening is not covered by membership, and now it could not
  be, because `Service` membership is never a message at all.
- **ADR-0014's *"the subject's own membership transition already carried that news, at the
  cause"* is qualified.** It carried the news only where the root was a `Name` or an `Address`,
  and what carries it is the **census** on that message, not the bare fact that something
  appeared.
- **ADR-0006's appearance split gains a kind restriction that applies to `returned` too.**
  Without it, an `Address` returning fires one alertable `Service` `returned` per port in
  `verge-core` — a burst on the one member of the family ADR-0006 made alertable, which nobody
  had noticed.
- **The census is load-bearing, not decoration.** It is the **only** carrier for a newly-entered
  subject's first values, because every timeline beneath that subject opens: the flagship leg
  predicate, the `Exposure` projection and all ten signal rules are transition-shaped and none
  of them matches an opening. Any surface or channel that drops the census drops the product's
  headline event on everything new.
- **One payload shape, two producers.** ADR-0029 §7's census-at-the-cause now serves an aperture
  widening and a membership entry, which are the two ways a burst of openings arrives. A third
  would be a signal that the shape is right.
- **[#55](https://github.com/winniel123/verge-asm/issues/55) composes rather than colliding.**
  An `Address` entering beneath a live `custody extension` fires that ruling's coverage-class
  message (*the gate your declaration holds open has moved*) and this one's drift-class message
  (*the estate grew, and here is what answered*). Two facts, two classes, two subjects. They
  need to be worded so they do not read as duplicates — a third wording pair for the
  notification patch, after [#28](https://github.com/winniel123/verge-asm/issues/28)/#22's and
  #55/[#51](https://github.com/winniel123/verge-asm/issues/51)'s.
- **The stated cost is that the entry census covers only the tier that admitted the subject.**
  ADR-0017 rules a leg's timeline opening *"when a slower tier first covers that port"*
  recorded, unnamed and unalerted, so a sensitive port that first answers on a slower tier days
  after its `Address` entered is silent: it opened, so no `Transition` exists, and the
  membership message has already fired. [ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md)
  narrows this rather than removing it: `certificate` now rides the `reachability` exchange and
  so opens *with* its `Service`, inside the census — but there are still three port tiers, and
  `tls-acceptance` has a weekly `Scan` of its own whose openings the census cannot contain.
  Whether the census is emitted incrementally per completed `Batch`, or the membership message
  waits for a defined set of tiers, is the notification patch's and
  [#4](https://github.com/winniel123/verge-asm/issues/4)'s profile's, and it is recorded as fog
  rather than guessed at here.
- **Decided on thin ground, in one place.** That provisioning a name is a common enough
  intentional change to have made *`appeared` is silent* tempting is an unmeasured base rate, of
  exactly the kind ADR-0006 asserted about decommissioning and ADR-0029 flagged twice. This
  ruling does not rest on it — it rests on the opening argument, which is textual — but the
  **volume** of `Name` `appeared` messages on an estate with per-branch preview environments is
  unmeasured, and if it turns out to drown the channel the remedy is coalescing in the
  notification patch and never a predicate change, because every one of these entries is
  recorded either way.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **`appeared` is silent, the twin of `withdrawn`** — provisioning is as common and as intentional as decommissioning, so it falls on #17's silent side | The tidiest answer and the one that breaks the product. A newly-entered subject has no transitions: its `Reach` leg *opens* at `reached`, its `Exposure` opens, its signals open. The flagship predicate, which is a transition, matches none of them, so a brand-new internet-reachable Redis would reach nobody through any channel. ADR-0014 and ADR-0029 both already say in terms that the membership transition is what carries this |
| **Every subject kind fires** | One `Address` becomes one message per `(port, transport)` in `verge-core`, and with names, per `Endpoint` on top. ADR-0007's *never one per affected subject*, verbatim |
| **Only *open* `Service`s appear** — the ticket's fourth bullet taken as a proposal, shrinking the appearing population below the existing one | Fails three ways: it makes membership a projection of `Reach`, so an internal port opening re-enters as `appeared` through the door #58 closed; it deletes ADR-0017's `unreachable`, which needs a subject to be a verdict about; and it makes the `Batch`'s recorded scope stop licensing the absence it was built to license |
| **`Name` only** | Silent on an existing name re-pointing at a new address — the modal way new ground enters a cloud-resident estate, and the case where ports become internet-reachable |
| **`Address` only** | Silent on a new name landing on an address already in the estate, which mints `Endpoint`s and therefore new `certificate` and `http-identity` timelines. It also inverts the operator's own model of their estate |
| **The root is always the `Seed`, so a first run is one message** | True of a first run and catastrophic afterwards: every `Citation` chain terminates at a `Seed`, so this is one message per `Seed` forever, which is #22's refused suppression wearing a coalescing rule's clothes |
| **Special-case the first run in the notification predicate** | An invented state in the comparison-adjacent path for a case the aperture rule already covers. ADR-0003's *first observed under a widened aperture is not "appeared"* was written for exactly this and had never been applied to the largest widening the product makes |
| **Extend the flagship to *the internet leg opens at `reached`*, and leave membership silent** | Genuinely close, and it loses on the burst rather than on the principle. It fires one message per newly-open service, so one `Address` with three open ports is three messages for one cause; it needs a three-way case analysis on *why* the timeline opened (membership, aperture, slower tier) at the notification layer; and it still tells the operator nothing about a new subject on which nothing answered, which is a fact they asked for when they declared the scope |
| **A count or threshold in the predicate** — suppress an appearance burst above *N* | ADR-0007 puts damping in notification and refuses it in the model, and [#27](https://github.com/winniel123/verge-asm/issues/27) refuses invented numbers in a safety path. The census gives the operator the count as a fact instead of spending it as a threshold |
