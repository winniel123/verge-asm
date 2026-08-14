# ADR-0013: Custody is control, not title — and a name scope may extend it

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#40 What is the right Seed primitive for a cloud-resident estate?](https://github.com/winniel123/verge-asm/issues/40)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Amends:** [ADR-0002](./0002-ownership-gates-probing.md)

## Context

[#26](https://github.com/winniel123/verge-asm/issues/26) established that the modal operator
holds **no registry resources at all** — 128,233 organisations worldwide hold any RIR
delegation, against 6.4M US employer firms and 32.3M EU27 enterprises — and that their estate
is cloud-resident. All 52,680 published AWS/Azure/GCP IPv4 prefixes belong to **48 registry
holders**, none of whom is the operator.

Read against [ADR-0002](./0002-ownership-gates-probing.md), that looked fatal. If the
cloud-resident estate is reached only through name scopes, its addresses derive `third-party`,
the probing gate closes, and the modal install never evaluates
`sensitive-port-reached-from-internet` — the product's sharpest signal — for the life of the
deployment.

Two of the three routes [#40](https://github.com/winniel123/verge-asm/issues/40) proposed had
already collapsed into each other before the ticket was worked, and this is worth recording
because the framing survived into the ticket itself. Its route 2 asked for *"a route to `owned`
that does not depend on registry delegation"* — but
[#27](https://github.com/winniel123/verge-asm/issues/27) had already withdrawn registry data
from the derivation entirely. An address-scope `Seed` over `52.1.2.3/32` derives `owned` today,
with **no registry check to fail**. *"Asks them to declare a CIDR they do not hold"* is a
pre-#27 sentence surviving in a post-#27 world.

What survives is narrower, and neither half is what the ticket led with.

## Decision

### 1. `Ownership` is renamed `Custody`, and it means control of the listener

The gate protects **consent to be scanned**. The party who can consent to a scan of
`52.1.2.3:6379` is whoever controls what listens there — not whoever appears in ARIN's file.
Amazon holds title to every EC2 address and cannot tell you whether your Redis should be open.

`Seed` was never a title claim: [`CONTEXT.md`](../../CONTEXT.md) defines it as *an operator's
assertion of where the estate ends*. But the derived value was named `Ownership`, which reads as
title, and the cost of that is on the record — [#40](https://github.com/winniel123/verge-asm/issues/40)
was **written in terms the name manufactured** (*"asserts ownership over other tenants"*, *"a
CIDR they do not hold"*), by someone holding the whole model. That is the `Host` defect and the
`sensitive-port-exposed` defect, and both were renamed.

`Custody` takes **two** values, `operator` and `third-party`.

### 2. `unknown` is deleted

It dated from the RDAP era, where it meant *the lookup failed or returned nothing*, and it
failed closed. After [#27](https://github.com/winniel123/verge-asm/issues/27) the derivation
reads `Seed`s alone, and *is this address covered by a `Seed`?* is a **total** question with no
lookup left to fail.

Every candidate producer under this ADR's extension collapses too: a name whose resolution is in
a `Gap`, is `Shadowed`, or has aged past its currency bound cites nothing, so no address exists
to hold an indeterminate value — and per
[ADR-0006](./0006-subjects-leave-by-measurement.md) an address either has a current citation or
has left. ADR-0002's *"`unknown` failing closed is the whole point"* survives intact: everything
not covered is `third-party`, which **is** the closed direction.

A value nothing can produce is worse than an invented state — it reads as a real distinction to
every future session.

### 3. A name-scope `Seed` may carry a `custody extension`

The operator declares, once, per name scope, that the addresses its names resolve to are within
the boundary. It is **off by default**, it is a property on the name-scope `Seed` — `Seed` keeps
exactly **two kinds** — and it sits beside `exclusions`, which
[ADR-0012](./0012-a-proposer-is-not-a-source.md) has just extended to address scopes. Both are
the operator adjusting the boundary of a scope they already declared.

**Transitivity stops where the resolution chain leaves the declared zone.** `api.example.com A
52.1.2.3` extends; `shop.example.com` → `d1x2y3.cloudfront.net` → `13.32.x.x` does not, because
the CNAME target is outside every name-scope `Seed`. That test is a **measurement**, not a list
of providers to recognise — which is what keeps it clear of
[#31](https://github.com/winniel123/verge-asm/issues/31)'s signature-database line and
[#27](https://github.com/winniel123/verge-asm/issues/27)'s refused threshold.

This is **not** ADR-0002's rejected *"confirm each discovered address"*. It is one declaration on
a scope the operator already authors — [#27](https://github.com/winniel123/verge-asm/issues/27)'s
confirm-a-scope-once, which that ADR's amendment already distinguished.

### 4. The declaration is self-correcting, and that is a safety property

A literal `/32` `Seed` does not merely go stale. AWS returns a released elastic address to a
public pool where another account can allocate it, so a stale literal declaration holds the gate
**open on a stranger's machine** — and nothing notices, because Declared input does not drift. It
is the only overreach in the model that grows silently over time.

A computed extension fails the other way: when the name stops resolving there, the address leaves
on its own. **Perishability was an argument for the extension, not against it** — the ticket
filed it as an ergonomic cost.

*(The AWS reallocation behaviour is reasoning about the platform, not a measured claim. It is
motivating rather than load-bearing — the decision stands on the self-correction property alone —
but if it is ever quoted as evidence it wants a citation from AWS's own documentation, per the
map's cite-the-owner habit.)*

### 5. `Custody` now moves on measurement, and the churn is signal

Its inputs are two Declared — address scopes, and the extension declaration — and one
**Observed**: the resolutions of names in an extending scope. It is the first Derived gate in the
model whose value moves because the world moved, and
[`CONTEXT.md`](../../CONTEXT.md)'s *"its inputs never drift, so it moves exactly when the operator
moves it"* is **withdrawn**.

Two existing rules collide at the instance-replacement case, and they split cleanly by **cause**:

| Cause | Outcome |
| --- | --- |
| The operator withdraws the extension | The addresses are still cited, so they stay; the gate closes and a **`Gap`** opens beneath them — [#8](https://github.com/winniel123/verge-asm/issues/8)'s case, unchanged |
| The resolution stops citing the address | The **address leaves** and takes its timelines with it — [ADR-0006](./0006-subjects-leave-by-measurement.md)'s case, no `Gap`, because there is no subject left to hold one |

So every `Custody` move is co-caused either by an operator act or by an `Address` membership
change the model already surfaces. **No new noise class**, and nothing new to damp — what flap
exists was already routed to the notification question by
[ADR-0007](./0007-drift-is-a-timeline-of-spans.md).

### 6. `Vantage class` reads literal address-scope `Seed`s only, never the extension

A prober in the operator's VPC must not verify `internal` in one batch and `internet` in the next
because **a DNS answer changed**. `Reach` is measured per class, so a vantage changing class
silently moves observations between the two legs of `Exposure` and manufactures drift in the
flagship value — the seam rule with the seam inside our own boundary check.

The cost is that [ADR-0012](./0012-a-proposer-is-not-a-source.md)'s stated failure survives
unchanged: a vantage inside undeclared operator space verifies as `internet` and **over-reports
`exposed`**, which is the loud, intended failure landing on
[#22](https://github.com/winniel123/verge-asm/issues/22)'s `Coverage`.

This is the one place where two things both fairly called *"the operator's addresses"*
deliberately mean **different sets**. It is stated here because a future session will otherwise
unify them.

### 7. The apex-flattening hole is carried by the declaration and by rendering, not by a check

The hazard the zone-boundary test does not close: `CNAME` is illegal at a zone apex, so DNS
providers serve `ALIAS`/`ANAME` there — flattened into A records pointing at a CDN. The chain
never leaves the declared zone, the boundary test passes, and the operator asserted something
their DNS provider silently made false. This is the common configuration, not an edge.

Requiring the zone file for an extending scope was considered and **rejected**, and the reason
generalises: it only buys the guarantee if every export distinguishes `ALIAS` from `A`. Route 53's
API carries `AliasTarget` explicitly; what a Cloudflare export shows with flattening enabled has
**not been measured**. A safety precondition whose reliability we have not established is worse
than a stated cost, because it fails *silently* and the operator believes they are covered.

So the hole is carried two ways:

- **The declaration says what it is asserting**, in the operator's terms, naming the apex case
  explicitly.
- **The operator can see the extension's current extension** — the addresses it covers right now,
  each with the name citing it. A CloudFront address in that list is legible to the person who
  knows their own estate.

The second is **display, not approval**. The declaration remains one act per scope, so ADR-0002's
rejected per-address approval queue stays rejected.

Both carriers are **point-in-time and the extension is continuous**, which is a stated cost this
section originally left unstated. It is repaired by the amendment below.

## Amendment — [#55](https://github.com/winniel123/verge-asm/issues/55): §7 gains a third carrier, because the first two fire once and the extension is recomputed forever

[#51](https://github.com/winniel123/verge-asm/issues/51) wrote §7's declaration and drew §7's
census, and found that the pair **discharges the tick-time hazard fully and the continuous hazard
not at all**. The operator reads the declaration once, checks the address list once, and ticks.
Everything after that belongs to whoever administers their DNS: point a name at a CDN on Tuesday,
and on Wednesday's cadence an edge address enters the extension, the gate opens over it, and on
the model's own accounting nothing unusual happened — an `Address` `appeared` and its `Service`s
`appeared`, which is the ordinary shape of an estate growing.

**So a live `custody extension` gaining an address notifies**, in the **coverage class**, one
message per extending scope per cadence, fired at the cause.

### Why it is a message rather than a stated cost

A live extension is the one place in this model where the probing gate opens over an address with
**no Declared act at all**. This ADR's own rejected-alternatives table refuses automatic extension
on the ground that *"every legitimate widening in this model traces back to a Declared act"* — and
the declaration is such an act, made **once**, about a set that is **recomputed**. Between one
cadence and the next, the party who moves the boundary is the operator's DNS administrator or
their provider. Leaving that silent is not a stated cost of the same kind as the ones this project
has accepted elsewhere; those all state a cost the operator can *see*.

### What it may assert, and what it may not

The message states the **difference and nothing else**: which addresses entered the extension this
cadence, the name citing each, and the resulting count — the same count the census renders, read
from the **same computation** and never counted independently
([#51](https://github.com/winniel123/verge-asm/issues/51)). It carries **no verdict**. It does not
say the addresses may not be the operator's, does not name a provider, does not flag the apex, and
contains **no number the product chose**. Its job is to return the operator's attention to §7's
second carrier at the moment that carrier's content changed — the census is the check, and the
message is the reason to look at it.

Three predicates were considered and refused, each on an instrument this project has already
declined:

| Refused predicate | Why |
| --- | --- |
| *The address falls outside every address scope declared or proposed* | On the modal cloud-resident install there **are** no address scopes ([#26](https://github.com/winniel123/verge-asm/issues/26), and §6 of this ADR), so it is true of the whole extension and discriminates nothing — and its second limb reads a `Proposal`, which [ADR-0012](./0012-a-proposer-is-not-a-source.md) says is read by nothing |
| *The address is cited by the apex specifically* | This ADR's own rejected-alternatives table already calls *apex* a heuristic standing in for a rule, because Route 53 aliases and Cloudflare flattening **both apply below the apex** — so it misses the motivating case (`www` re-pointed at a CDN) while reading as coverage |
| *Only entries caused by an existing name re-pointing* | The most attractive of the three, and it fails on **§7's own criterion**: it is silent on a new name pointed at somebody else's edge, so it claims a property it does not have, which is *"a safety precondition whose reliability we have not established is worse than a stated cost"* one level down |

### Why it is the coverage class and not a fifth cause

The coverage class already holds **a closing custody gate**, and this ADR's Consequences already
put the gate *opening by operator act* there too — `revealed`, one message at the cause. What had
no message was the gate opening **because the world moved under a standing declaration**. That is
the same aperture input in the same direction, so **no fifth cause, no fourth class, and no sixth
aperture input** — this ADR's *"no new notification class"* survives unamended. What is new is the
**agency**: it is the first coverage-class member caused by neither our act nor the operator's,
after [#48](https://github.com/winniel123/verge-asm/issues/48) added the first one caused by the
operator's own input.

### The nag test

An estate that legitimately grows every week must not be told weekly that it may have
over-asserted, or the message trains the operator to ignore the one channel carrying a safety
property. Three things carry it, none of them a form of words alone:

- **The operator who was right to leave the extension off never sees this message at all.** There
  is no live extension, so there is nothing to gain an address. §7's hazard and §7's message have
  the same population.
- **It fires at the cause, carrying a set** ([ADR-0007](./0007-drift-is-a-timeline-of-spans.md)),
  so a CDN cutover that moves a whole zone in one night is one message and not forty.
- **It reports a boundary, not a suspicion.** *The addresses `example.com` covers have changed* is
  the register; *you may have over-asserted* is not, and the second is what the nag test forbids.
  An operator who added two machines this week reads two addresses they recognise, which is the
  same five-second act §7 already asks of them once.

Two riders. **A departure does not fire it** — an address leaving is §4's self-correction working
and the gate narrowing, so the message triggers on a gain and may then state the whole difference.
And **an address re-entering within the currency bound does not re-fire**, reusing the bound
already in the model rather than a second constant — a blue-green flip between two addresses is
the toggle-inside-one-cadence non-event under another name.

### It is not the approval queue arriving through the notification layer

This is the objection worth answering in terms, because a per-cadence list of new addresses looks
exactly like ADR-0002's rejected queue delivered by mail.
[#51](https://github.com/winniel123/verge-asm/issues/51)'s six census/queue discriminators bind the
**message** as well as the panel: **no per-address control and no per-address act**, no pending
state (every address listed is already in force and already being probed), no count that goes down
as the operator works, and the only act remains the one act on the whole declaration. A message
that offered *approve* or *exclude* per row would be the queue, and it is refused here explicitly
so that nobody re-derives it as an ergonomic improvement —
[ADR-0022](./0022-confirmation-is-singular.md)'s reason verbatim: the request arrives as a usability
complaint rather than as a safety proposal, which is how it would get through. That ADR's unit rule
also settles the message's shape without further argument: **the unit is the scope, never the
address**, which is why this fires once per extending scope and not once per entry.

### Thin ground, stated rather than smoothed

The message inherits [#51](https://github.com/winniel123/verge-asm/issues/51)'s unmeasured
assumption whole: it works only if the operator reads it and recognises their own addresses. It
converts the continuous half of §7's hazard from *no carrier* to *a carrier resting on the same
unmeasured assumption as the tick-time one*. That is a real improvement and it is not a proof, and
the spec should say so in those words rather than claiming §7 is discharged. The sharpest available
repair remains the one [#51](https://github.com/winniel123/verge-asm/issues/51) recorded and left
unpriced — a PTR on an entering address is a **measurement** rather than a list and so clears
[#31](https://github.com/winniel123/verge-asm/issues/31)'s line, but it is a new measurement
carrying [ADR-0011](./0011-a-facet-is-six-parts.md)'s six obligations and is a scope decision.

## Consequences

- **The modal install can evaluate the flagship signal.** A cloud-resident operator types the
  domain they were going to type anyway and ticks one box.
- **Turning the extension on widens the aperture through an input that already exists.**
  [#36](https://github.com/winniel123/verge-asm/issues/36) counts the custody gate among the five
  aperture inputs, so services first seen on those addresses are **`revealed`**, never
  `appeared`, and it fires **one message at the cause** rather than one per service
  ([ADR-0007](./0007-drift-is-a-timeline-of-spans.md)). **No sixth aperture input and no new
  notification class.**
- **Withdrawing the extension and re-enabling it puts a value where a `Gap` was**, which is a
  second live instance of the question open as
  [#42](https://github.com/winniel123/verge-asm/issues/42).
- **Exposure attribution follows.** ADR-0002 splits attribution by the gate — a custody-extended
  address is `operator`, so its exposure belongs to the `Address` rather than to the `Name`.
- **The registry paths lose their last claim to being load-bearing.** The cloud-resident entry
  point is a name scope plus a tick; no org→prefix lookup is on the path. This is an input to
  [#43](https://github.com/winniel123/verge-asm/issues/43).
- **An operator on shared hosting loses nothing by leaving it off**, and gains a false assertion
  by turning it on. That is the intended shape: being wrong here is an ordinary false declaration,
  the same class as declaring `3.0.0.0/8`, which the model has never tried to prevent because it
  cannot prevent a false name-scope seed either.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Literal address-scope `Seed`s only — the operator types `/32`s | Perishes *dangerously*, not merely inconveniently (§4), and it is precisely the manual inventory the map's Destination says the product exists to remove |
| Name scopes extend custody **automatically**, with no declaration | Widens the gate by derivation onto shared hosting and flattened apexes with nobody having asserted anything. Every legitimate widening in this model traces back to a Declared act |
| A test for whether an address is *dedicated* rather than *shared* | Every version is either an invented threshold inside the safety path ([#27](https://github.com/winniel123/verge-asm/issues/27)) or a list of hyperscaler ranges ([#31](https://github.com/winniel123/verge-asm/issues/31)). Both already refused |
| Exempt the zone apex from the extension | Route 53 aliases and Cloudflare flattening both apply below the apex, so "apex" is a heuristic standing in for a rule — the same refused shape one level down |
| Require the zone file for a custody-extending scope | §7. Buys the *appearance* of a guarantee that depends on unmeasured export behaviour, and fails silently |
| Propose each resolved address for confirmation, reusing [ADR-0012](./0012-a-proposer-is-not-a-source.md)'s `Proposal` | This is ADR-0002's per-address approval queue exactly — forever, inside the discovery loop — and that row stays rejected |
| A third `Seed` kind | Nothing here needs a new key or lifecycle, and `Seed`'s two-kind shape is quoted across [ADR-0002](./0002-ownership-gates-probing.md), [ADR-0012](./0012-a-proposer-is-not-a-source.md), [#26](https://github.com/winniel123/verge-asm/issues/26), [#27](https://github.com/winniel123/verge-asm/issues/27) and [#39](https://github.com/winniel123/verge-asm/issues/39) |
| Keep the name `Ownership` and fix only the definition | The evidence that the word misleads is the ticket that produced this ADR. [ADR-0010](./0010-exposure-composes-two-reaches.md) set the precedent that old names stand unrewritten in closed tickets, so the rename is cheap |
