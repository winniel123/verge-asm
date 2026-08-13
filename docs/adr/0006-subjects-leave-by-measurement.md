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
- **Withdrawal needs every available vantage to agree**, composing `Availability` exactly as
  `Exposure` does, or every split-DNS name flaps on every run. One vantage down makes
  membership not-comparable rather than concluded from the survivor.
- **No cascade down the tree.** A Name Error at one name licenses no absence below it. RFC
  8020 says it should, but compliance is not universal, and every inferred absence is a
  "removed" alert we never measured. Re-querying is free.
- **Appearance splits in two.** *Appeared* is discovery; *returned* is a decommission being
  undone, and it is the one of the pair worth an alert — withdrawal is the most common
  intentional change in an estate, and alerting on it trains the operator to ignore the
  channel. The split is also where
  [ADR-0003](./0003-third-party-source-consent-bar.md)'s aperture rule can finally sit: *first
  observed under a widened aperture* is a third member of the same family, and a model with
  one appearance transition has nowhere to put it.
