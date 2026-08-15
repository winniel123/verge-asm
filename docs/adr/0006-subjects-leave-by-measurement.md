# Subjects leave by measurement, never by decay

A subject leaves the estate only because something measured its absence. verge-asm ships no
decay: nothing ages out, no clock and no counter retires a subject, and there is no
`dormant` / `stale` / `unconfirmed` state between present and gone.

This looks like an omission, so the reasoning has to be written down. The problem it answers
is real: [`CONTEXT.md`](../../CONTEXT.md) permits only an `enumerable` source to assert an
absence, and only inside the scope its `Batch` recorded, so a subject that no enumerable
source covers can never die — an append-only inventory, which is half a drift product.

Three findings dissolve most of it rather than solving it.

**Our own resolver is `enumerable` over a single `Name`.** A Name Error is a complete answer
to whether that name exists, so the scope a resolution batch records is `{name, qtype}`. This
sets no new precedent — the prober was already enumerable over `(that address, that port
set)`, a scope we chose rather than one the operator declared. The rule was never *the
operator must declare the scope*; it was *the batch must record the scope its silence covers*,
and a singleton is the smallest such scope. Since every known name is re-resolved every cycle,
the premise that a certificate-discovered name has only `corroborative` sources bearing on it
is false for every name outside a wildcard.

**Removal is an observed value, not an invented event.** A `Name` going away already has one:
the `resolution` facet reporting a Name Error. Inventing a lifecycle state would mean
inventing a transition, and an invented transition is a threshold we chose sitting inside the
comparison path — the failure the three-layer split exists to prevent. So membership is a
Derived *view* over the latest observation per facet, the Observed layer stays append-only,
and "gone" is diffable with no new machinery.

**The problem was only ever a `Name` problem.** `Service` and `Endpoint` are known only by
measurement, so the prober's scope already licenses their absence. `Address` — alone among the
four subjects — has no lifecycle of its own, because nothing ever observes an address's
existence; it is in the estate exactly while a current resolution cites it or a `Seed` covers
it.

What remains is the genuine residue: names below a DNS wildcard, where the resolver can never
return a Name Error. That is where decay would have to earn its place, and it cannot.
Opportunity-counted decay — *N* batches that would have corroborated it and didn't — sounds
like the principled form, but the residue is precisely the set where the opportunity never
arrives: under a wildcard we query the name and get an answer every time, so corroboration
opportunities are infinite and all worthless. Wall-clock decay is the cardinal sin twice over,
reporting the passage of time as movement, and turning a vantage down for a week into an
estate-wide removal — *we stopped looking* rendered as *everything was removed*.

So the residue stays in the estate, visibly unconfirmed, and leaves by one of two honest
routes: the operator supplies coverage, or the operator declares it out of scope.

## Amendment — [#35](https://github.com/winniel123/verge-asm/issues/35): the residue has a
second member, and `resolution` has a fifth outcome

This ADR's residue was measured as one population — names below a wildcard — and that is
now known to be incomplete. Two changes, neither of which disturbs the decision above.

**`resolution` gains a fifth outcome, `Lame`.** Beside Name Error, NODATA, `Shadowed` and
*source error*, a `Name` whose parent zone delegates it to nameservers that were **reached
and do not serve it** holds `Lame` — RFC 8499 §7's lame delegation. This is an observed
value and not a source error, because the delegated authorities were queried **directly**
and answered; the distinction is only purchasable that way, since a recursive resolver's
SERVFAIL cannot separate a dead delegation from our own bad upstream. A delegation only
partly lame is not this value: the name still resolves, so `resolution` has not moved, and
the per-nameserver detail is recorded on `dns-record`.

**Names beneath a `Lame` delegation are a second un-disconfirmable population.** With
nobody left to answer, no Name Error can ever arrive for them, so they can no more leave
than a shadowed name can. They are **not** `Shadowed` and they do not inherit `Lame` —
under [ADR-0007](./0007-drift-is-a-timeline-of-spans.md) we hold no value for them at all,
which is a `Gap`. That is the correct record and it needs no new machinery: the `Gap` says
*we stopped being able to look*, this ADR's no-cascade-down rule already forbids inferring
absence beneath, and one lame delegation therefore produces one alert over an arbitrarily
large burst of `Gap`s — ADR-0007's alert-at-the-cause rule, unmodified.

The residue's escape routes are unchanged, and it is worth noting the asymmetry: unlike a
wildcard, a lame delegation is a **defect the operator can fix**, so the third route out is
the one this ADR could not offer before — repair the delegation and the names beneath
become measurable again.

## Amendment — [#36](https://github.com/winniel123/verge-asm/issues/36): a key component may be
absent from the start

The cascade rule below — *a subject leaves when a component of its natural key leaves* — is
general and stands unchanged. What needs saying is that its worked example named only one
component, and a reader carrying the example rather than the rule now gets the wrong answer.

[ADR-0011](./0011-a-facet-is-six-parts.md) lets an `Endpoint`'s `Name` be **absent**, meaning
*the default response to a client that names nothing* — the only endpoint available on an
address-scope `Seed` before any name is known. Such an endpoint has no `Name` to withdraw, so
*"an `Endpoint` whose `Name` has gone is not a measurable thing"* never fires and, read as the
whole rule, it would keep a live endpoint under a dead port forever.

The `Service` leg was always there and always necessary; it went unnoticed because a withdrawing
`Service` normally arrives alongside a withdrawing `Name`. Stated explicitly: **an `Endpoint`
closes when either its `Name` or its `Service` withdraws**, and a nameless one simply has one
leg. Reason `cascaded` is unchanged, so the endpoints return coherently if the port comes back.

## Amendment — [#63](https://github.com/winniel123/verge-asm/issues/63): the appearance family
fires on two subject kinds, not four

*Appearance splits in two* below rules `returned` alertable and `withdrawn` not, and leaves
`appeared` unstated. [ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)
settles `appeared` and adds a restriction the split needs and never had: **a membership message
fires only on a `Name` or an `Address`, and never on a `Service` or an `Endpoint`.**

Those two have keys the model composes from what it already holds — a `Service` from an `Address`
in the estate and `verge-core`, which is ours; an `Endpoint` from two subjects already in the
estate — so their membership is another subject's membership restated, and a message for one is
a second representation of one fact.

The restriction governs the **whole** family and not only `appeared`, which is where it earns its
place: without it an `Address` returning fires one alertable `Service` `returned` per
`(port, transport)` in `verge-core`, a burst on the one member of the family this ADR made
alertable. The cascade rule below already writes those spans with reason `cascaded`; they are now
also silent.

## Consequences

- **`Seed` gains exclusions** — exact names and subtrees, not patterns. Excluding a
  still-resolving name is legal: the operator is saying *not mine*, not *not there*, and a
  tool that refuses that is arguing with its owner about where their estate ends. An excluded
  name is no longer queried. This is deliberately **not** an `Annotation`, which is operator
  *opinion attached to a measured thing*; using opinion to remove a subject is the suppression
  [#22](https://github.com/winniel123/verge-asm/issues/22) refused, under a different name.
  Exclusions therefore appear on `Coverage`, because they are the one route by which an
  operator can silently shrink the estate until the board looks clean.
- **A wildcard-shadowed answer is its own observed value**, neither the synthesised answer nor
  a source error. Taking it at face value is estate-scale drift manufacture — repoint one
  wildcard and every fictional name below it reports a resolution change the same night.
  Discarding it loses the only fact the operator needs, which is that we cannot see here. As a
  value it keeps *we cannot say* in the Observed layer beside its evidence, and makes
  **resolving → shadowed** real drift: someone added a wildcard, and the operator just lost
  sight of a name they had.
- **Admission under a wildcard turns on provenance, not on the answer.** A brute-forced label
  is discarded — it was a hypothesis the wildcard failed to refute, never evidence. A
  certificate SAN or a zone-file entry is admitted as shadowed, because neither was derived
  from the resolution the wildcard poisoned.
- **One cascade rule, not several**: a subject leaves when a component of its natural key
  leaves. An `Endpoint` whose `Name` has gone is not a measurable thing; a `Service` under a
  departed `Address` likewise.
- **De-citation closes the probing gate.** An `Address` reached only through a name that has
  since gone has no `Citation` chain back to a `Seed` — the precise condition
  [ADR-0002](./0002-ownership-gates-probing.md) forbids, and by then the provider has probably
  reassigned it. Losing the last citation stops the probe, it does not merely hide a row.
- **Withdrawal needs ~~every available vantage~~ a cross-class `Vantage composition` to agree**,
  composing `Availability` exactly as `Exposure` does, or every split-DNS name flaps on every run.
  One vantage down makes membership not-comparable rather than concluded from the survivor.
  > **NAMED and REPAIRED by [#138](https://github.com/winniel123/verge-asm/issues/138) ·
  > [ADR-0080](./0080-a-vantage-composition-is-cross-class-or-class-scoped-and-only-one-takes-a-quantifier.md).**
  > This is the **cross-class** kind: every `Vantage class` the install runs holds a current value and
  > they agree, disagreement being incommensurability rather than evidence. The struck phrase is
  > **superseded here, at the site that specifies it**
  > ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)) for two
  > reasons. Read alone, *"every **available** vantage"* excludes the down vantage and concludes from
  > the survivor — the exact opposite of the sentence beside it, and it is the *phrase* rather than the
  > sentence that travelled onward, into ADR-0004, ADR-0020, ADR-0071 and `CONTEXT.md`'s `Name` entry.
  > And over an **empty** set unanimity is vacuously **true**, so read literally this bullet
  > **withdraws every `Name` in the estate** the night every vantage goes unavailable. Under ADR-0080
  > both are closed by construction: a cross-class composition needs a value from every class, so an
  > absent class is a missing term rather than an empty conjunction.
  >
  > **This composition's consumer is not a rule**, which is why ADR-0080 refuses to describe a
  > composition as a step inside a predicate.
- **No cascade down the tree.** A Name Error at one name licenses no absence below it. RFC
  8020 says it should, but compliance is not universal, and every inferred absence is a
  "removed" alert we never measured. Re-querying is free.
- **Appearance splits in two.** *Appeared* is discovery; *returned* is a decommission being
  undone, and it is the one of the pair worth an alert — withdrawal is the most common
  intentional change in an estate, and alerting on it trains the operator to ignore the
  channel.
  > **RESTRICTED by the [#63](https://github.com/winniel123/verge-asm/issues/63) amendment above**,
  > which names this split and adds the limb it *"needs and never had"*: **a membership message fires
  > only on a `Name` or an `Address`, and never on a `Service` or an `Endpoint`.** Read alone, this
  > bullet fires one alertable `Service` `returned` per `(port, transport)` in `verge-core` when one
  > `Address` returns. Marked at the sentence per
  > [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened
  > by [#106](https://github.com/winniel123/verge-asm/issues/106) — the amendment sits **above** this
  > sentence, which an order-based reading of the rule would miss.
  >
  > **QUALIFIED a second time by [#121](https://github.com/winniel123/verge-asm/issues/121) ·
  > [ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md): the split
  > is destroyed by a `Break` between the withdrawal and the return.** A withdrawn subject's
  > timelines **close**, so a return after a leaf bump reopens with nothing legally before it, no
  > `Transition` is derived, and the message fires reading **`appeared`** — the pair collapsing into
  > its uninteresting member exactly when a decommissioned host comes back. Retention cannot fix it
  > and history is never re-derived, so **the release owes the statement**: the re-baseline message
  > names the loss ([ADR-0014](./0014-only-revealed-generalises.md)), `resolution-walk`'s golden
  > corpus must pin the membership-deciding outcomes so a dependency upgrade provably does not bump
  > the leaf, and the membership vector may not be widened. Retention's own floor — never deleting
  > the span before an open one — is the storage-layer twin of *nothing leaves because time passed*,
  > and it is discharged by never compacting the `Span` corpus at all.

  The split is also where
  [ADR-0003](./0003-third-party-source-consent-bar.md)'s aperture rule can finally sit: *first
  observed under a widened aperture* is a third member of the same family, and a model with
  one appearance transition has nowhere to put it.
