# ADR-0084: A `Scan` is a cadence over an exchange — and an uncovered facet's currency bound is undefined, not loose

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#142 `resolution` and `dns-record` have no covering `Scan`, so no currency bound](https://github.com/winniel123/verge-asm/issues/142)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[#124](https://github.com/winniel123/verge-asm/issues/124) minted the fourth `Scan`, `zone`, on
[ADR-0005](./0005-scan-execution-model.md)'s own reasoning — *a measurement needing a cadence of its
own takes a `Scan` of its own* — and then wrote down, rather than absorbed, the hole it could see and
would not guess at:

> **A hole this amendment does not close, stated rather than absorbed:** `resolution` and our own
> resolver's `dns-record` still have **no covering `Scan`** either, so they have no currency bound
> for the same reason. That is wider than the packaging patch and is ticketed rather than guessed at
> here.

This is that ticket. Four `Scan`s exist — two port tiers, `tls-acceptance`, `zone` — and the two DNS
facets are covered by none of them.

The upstream that decides the shape:

- **[ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md)** — a facet's cadence is
  the cadence of the **exchange** that measures it, and a measurement needing a cadence of its own
  takes a `Scan` of its own. `certificate` takes none, because its handshake is a step *inside* the
  exchange that produces `reachability`.
- **[ADR-0044](./0044-a-one-off-measurement-has-no-currency.md)** — there is no one-off measurement
  in the model, because a measurement with no cadence has no currency.
- **[ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md)** — the
  observation retention floor **is** the currency bound. A corpus with no covering `Scan` therefore
  has no floor either, which is what makes this urgent rather than tidy.
- **[ADR-0019](./0019-the-probing-gate-is-total-over-an-address.md)** — an install holding custody of
  nothing measures `resolution` and `dns-record` at full aperture, **a query is not a connect**, and
  measures none of the four facets that ride a TCP connect.

## Decision

| Concern | Decision |
| --- | --- |
| Covering `Scan` | **A fifth `Scan`, `dns`.** One `Scan`, covering **both** `resolution` and `dns-record` on our own resolver's timelines |
| Name | **`dns`** — for the exchange. Not `discovery`, and the refused noun is the smaller of the two reasons |
| Scope | **The name scopes**, unconditionally — independent of `Custody`, and independent of whether a zone file was supplied |
| Port list | **None.** The second `Scan` with none, after `zone`. `Scan` count 5, port tiers still **2** |
| Vantages | **Every configured `Vantage`** — unlike `zone`, which has none at all |
| Cadence | The operator's, **shipped at daily** — the tightest shipped port tier's |
| Currency bound | `k` cadences of it: **two days** at shipped settings. `k` = 2 is untouched |
| Aperture inputs | **Seven, unchanged.** A cadence is not a dimension of a recorded scope |
| Declared parameters | **None added.** Every parameter of `resolution-walk` and `wildcard-discrimination` already has a value |
| The zone source's `dns-record` timeline | **Already covered, by `zone`.** One timeline per source, and only *our resolver's* was uncovered |
| The passive sources' admissions | **Not covered here.** Their hop is a `Citation` and not an observation; ticketed, not guessed at |

## Rationale

### Undefined is not loose, and it reaches the `Address` membership predicate

The ticket's framing — *nothing bounds the currency of the facts every membership decision reads* —
is right and understates it. The failure is not that DNS observations age slowly. It is that the
predicate has **no truth value at all**.

[ADR-0007](./0007-drift-is-a-timeline-of-spans.md)'s currency rule reads *"an observation is current
while it is within `k` cadences of the Declared `Scan` whose scope covers that `(subject, facet,
port, vantage)`"*. Where no `Scan` covers the tuple there is no cadence to multiply, so *is this
observation current?* is not answered *no* and is not answered *yes* — it is unanswerable. Three
things downstream are functions of that unanswerable question:

1. **`Address` membership.** `CONTEXT.md`: an `Address` *"is in the estate exactly while a **current**
   resolution cites it or a `Seed` covers it"*. On a name-scope install there is no `Seed` limb, so
   the entire `Address` population — and therefore every `Service`, every `Endpoint`, and every
   timeline hanging off them — is defined by a predicate with no value.
2. **The probing gate.**
   [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) §: a resolution that *"has
   aged past its currency bound cites nothing, so no address exists"*. Under a `custody extension`,
   `Custody` is *"a function of measured resolutions"* — so the one Derived value carrying a safety
   consequence reads an input whose staleness cannot be computed.
3. **`Gap`.** A `Gap` opens when an observation ages past the bound. With no bound, **no `Gap` ever
   opens on either DNS facet**, so *we stopped being able to say* is unreachable on the two facets
   most likely to stop being sayable — the mirror of [#48](https://github.com/winniel123/verge-asm/issues/48)
   route 4, which is the exact defect `zone` was minted to close.

That is the whole warrant. The alternatives below are measured against it.

### The losing option: `resolution` rides the reachability exchange the way `certificate` does

This is the strongest thing that can be said against a fifth `Scan`, and it has real ADR behind it.
ADR-0028 refused a `certificate` `Scan` because the handshake is *"a step in the exchange that
produces `reachability`, not a scan of its own and not a tier"* — neither of its negatives can be
read without knowing the port was open. A prober must likewise resolve a `Name` before it can connect
to an `Endpoint` beneath it, so the DNS query looks like a step in the same exchange, and
`resolution` looks like it should ride whichever port tier ran.

It loses on three independent grounds, and the first is decisive on its own.

**One: ADR-0019 has already measured the relation, and it is the opposite one.** An install holding
custody of nothing *"measures `resolution` and `dns-record` at full aperture and measures none of the
four facets that ride a TCP connect; its `Service` population is empty"*. `certificate` rides
`reachability` because it **cannot** be read where the connect did not happen. `resolution` is
measured **precisely where the connect cannot happen**. A facet cannot take its cadence from an
exchange that is not made on the install the model spends a whole ADR describing — that install would
measure the only thing it measures on no clock whatsoever.

**Two: the populations are disjoint in both directions.** A port tier runs over the `Address`es in
custody; the DNS exchange runs over the `Name`s beneath the name scopes. A name scope *"enumerates
nothing"* and an address scope produces no names, so a name-scope-only install has an empty port-tier
scope and a full DNS scope, while an address-scope-only install has the reverse. Neither set contains
the other, so neither cadence can bound the other's facts.

**Three: it is the hidden-field failure, for the third time.** ADR-0005 refused a cadence hidden
inside another `Scan` because it makes the aperture a hidden field rather than a configured object;
#124 refused it again for the zone file, in terms — *"disabling the port tier would silently stop the
zone being read"*. Here it is worse: disabling the daily port tier would silently stop the estate
being **resolved**, on an install where the port tier had nothing to do anyway.

**And a fourth, which is ADR-0028's own rule applied rather than argued.** *A facet's cadence is the
cadence of the exchange that measures it.* The exchange that measures these two facets is
`resolution-walk`'s pair of queries — UDP/TCP to port 53, at the operator's own authorities and at
the `Vantage`'s recursive resolver, ungoverned by the probing gate. `CONTEXT.md` already names the
difference in four words: **a query is not a connect**. Two exchanges, so two cadences, so two
`Scan`s.

### One `Scan` and not two, because `Shadowed` and `Lame` are committed jointly

The next reasonable position is *two* new `Scan`s — `resolution` is the flagship every membership
decision reads and a `resolution` move is the one facet `Transition` that is a message in v1, while
`dns-record` *"has no channel at all"* and an MX or TXT change reaches nobody. Different urgency,
different cadence, and by ADR-0005's own one-cadence-per-`Scan` rule that is two `Scan`s.

It loses on the model, not on taste. **Two values are committed across both facets inside one
batch:**

- **`Shadowed`** is a value on `dns-record` as well as on `resolution`, and it is committed *"on
  `resolution` and on every `dns-record` discriminator or on none, because RFC 4592 blocks
  synthesis"* where the name exists.
- **`Lame`** is decided by one delegation walk and is *"also a value on `dns-record`, [since] the
  authorities were reached and refused to serve, so every qtype"*.

Split across two `Scan`s on two cadences, one facet's `Shadowed` becomes a function of a control
probe made in a **different batch at a different time** — which is
[ADR-0011](./0011-a-facet-is-six-parts.md)'s forbidden cross-batch value assembly, and it is the exact
objection ADR-0028 raised against a separately-scheduled `certificate` handshake. It would also split
the **queried qtype set** — one aperture input — across two recorded scopes, so the input would hold
two values at once with nothing in the model naming the difference, which is ADR-0028's objection to
differing candidate sets across the tiers, arriving one layer down.

One exchange, one batch, one recorded scope, one `Scan`.

### It is `dns` and not `discovery`, and the refused noun is the smaller of the two reasons

`Discovery` is a **refused noun** in `CONTEXT.md` — *"both are sources differing only in `authority`
and `completeness`; naming them as separate kinds of thing re-splits the pipeline that one
`Observation` exists to unify"*. A `Scan` is not a source, so a `discovery` `Scan` is not literally
the refused term returning; the refusal is about *kinds of source*, and it says in terms that the word
*"remains fine as a verb"*.

So the refusal alone would not settle it. What settles it is that **`discovery` would be a false
description of this `Scan`'s scope**. The `Scan` covers the DNS exchange and nothing else. Certificate
transparency, the registry paths and the zone file all put subjects in the estate and **none of them
rides this `Scan`** — CT observes no facet at all, and the zone file has had its own `Scan` since
#124. A `Scan` called `discovery` fails
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s test read
forwards: *a competent session reading the name alone, in the present tense, would hang the crt.sh
poll off it* — which is the thing this ADR deliberately does not decide.

`dns` names the exchange, which is what ADR-0028 says a cadence belongs to. It follows `zone`'s
precedent of naming a `Scan` for what it reads rather than for a port set, and being a protocol name
it cannot be mistaken for a subject or a layer. The near-collision with the `dns-record` **facet** is
accepted and is the price of naming the exchange honestly: the `Scan` covers `dns-record` *and*
`resolution`, and no name drawn from one facet could say so.

`resolution` as a name loses for that reason — a `Scan` named for one of the two facets it covers
under-claims the other. `authority` loses because half the exchange is made at the `Vantage`'s
recursive resolver and not at the delegated authorities (ADR-0070).

### No port list, and `CONTEXT.md`'s own enumeration is the ADR-0058 site

**A `Scan` does not have a port list**, and this ruling must not re-introduce the assumption. The
principled form of that: a `Scan` carries a port list exactly where its exchange is a **connect**, and
the port list is then an aperture input because it decides *what may be found*. The DNS exchange's
destination port is a property of the transport to the authority. It enumerates nothing about the
estate, no value carries a per-port negative over it, and widening it would find nothing — so it is
not a dimension of the recorded scope, and it is not on the `Scan`.

`zone` established that a `Scan` may have none. With `dns` there are **two of five**, and the
glossary's own opening enumeration — *"which scopes, which ports, which vantages, what cadence"* — is
now the site that specifies the assumption. Read alone and in the present tense it tells a session
that every `Scan` has ports. It is withdrawn at that site.

### It has vantages, where `zone` has none

`zone` has *"no port list and no vantage choice at all, the worker reading it"*. `dns` is not its
twin: a resolution's **answer is a function of where it was asked from**.

[ADR-0070](./0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md)
put the resolver standing at the query path **inside the `Vantage`** and therefore inside the
timeline key, on a measurement — one wildcarded name drew its addresses from two **disjoint** pools at
two vantages in one week. [ADR-0071](./0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md)
then made `non-globally-reachable-address-resolved-from-internet` read `resolution` **at the
internet-class vantage only**.

So the `Scan` runs at **every configured `Vantage`**, as the port tiers do, and its batches carry the
`vantage` component the key already requires. That is a schedule and an aperture, not a composition:
how a rule's leaf combines the per-vantage answers belongs to
[#138](https://github.com/winniel123/verge-asm/issues/138) and is untouched here. What this ruling
hands #138 is the precondition its two compositions need — **one `Scan`, one cadence, every vantage**
— because a unanimity claim over vantages measured on different clocks composes a live observation
with a stale one, and neither of #138's compositions is expressible against that.

### Daily, and loosening it withdraws `Address`es rather than probing stale ones

The shipped default is **the tightest shipped port tier's cadence, daily**, and the reason is a
dependency rather than a preference: the port tier's own subject population is computed from
`resolution`, and `Custody` under a `custody extension` is a function of measured resolutions. A DNS
cadence looser than the port tier's would have the tier connecting to an address set the model no
longer vouches for.

Faster than daily was considered and refused as an underived number. The cost driver is not the
resolution itself but the control probes — **nine random labels plus one structured label, each run
over the declared qtype set, per parent** ([ADR-0069](./0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md),
count raised by [#115](https://github.com/winniel123/verge-asm/issues/115)) — all of it landing on the
operator's own authorities under #4's safety frame. An hourly cadence multiplies that by 24 to buy
currency nothing reads at that resolution.

**The consequence of an operator loosening it needs no new machinery, and this is worth stating
because the first instinct is to add some.** A `dns` cadence looser than the port tier does not
produce stale probing; it produces **subject withdrawal**. The resolution ages past its bound, it
cites nothing, and the `Address` leaves the estate with its timelines — which is `CONTEXT.md`'s
`Address` entry and ADR-0013's gate working exactly as written. No invariant coupling two operator
dials is needed, and none is created: a coupling would be a second mechanism doing `Gap`'s job, which
[ADR-0014](./0014-only-revealed-generalises.md) has already refused by name.

### Minting a `Scan` widens no aperture, because a cadence is not aperture

Worth writing down once, since it was never stated and both `zone` and `dns` depend on it.

ADR-0014 fixed the criterion: **aperture is what a `Batch` records as its completed scope**, and every
aperture input is a dimension of that record. A **cadence is not a dimension of a recorded scope** — a
batch records *what we asked about*, never *how often we ask*. So a new `Scan` adds an aperture input
only if it introduces a new scope dimension.

`dns` introduces none. Its batches record enabled sources, the queried qtype set, the control-probe
population and the vantage — inputs 1, 4, 7 and 6 of the seven, all of them already recorded on the
DNS batches that exist today. **The aperture input count stays at seven**, and the same argument
retroactively explains why `zone` added none.

The honest residue, since the ticket asked for it loudly: this ruling **does** cause looking that does
not happen today, in exactly one sense — it makes the DNS exchange **recur**. Nothing in the corpus
today says when it is asked a second time. That is not an aperture widening (no scope dimension moves,
so no `revealed` fires and no coverage-class message is owed); it is the model's standing requirement
arriving where it had been skipped, since ADR-0044 already holds that a measurement that does not
recur is not expressible. An operator sees no new *kind* of query and no query against a subject we
were not already asking about.

### The passive sources are stated rather than absorbed — the #124 move, applied again

The ticket's opening sentence names a second population: *"the same hole is still open … for the
passive sources."* It is not closed here, and the reason is that it is a **different mechanism with a
different consequence**, not that it is inconvenient.

A source that admits without observing holds no facet timeline, so it has no *currency bound* in the
sense this ADR is about. What it has is a **`Citation` that goes stale** — and ADR-0041 saw the same
seam and set it aside in the same way: *"a `Batch` cited by a source that admits without observing is
not an observation and does not age the same way."* The consequence there is **subject withdrawal**,
not a `Gap`, and it turns on a question nobody has put: whether certificate transparency's silence may
retire a `Name` at all, against a source the map has **measured** returning spurious 404s and a
standing rule that a source that errors must produce no observation. A `Scan` covering crt.sh at a
cadence would, on a bad day at the source, mass-withdraw an estate.

That is worth its own interrogation and gets a successor ticket. Putting it on `dns` would also make
the name a lie about the scope, which is §4 above.

### Where this was decided on thin ground

**The daily default.** The *shape* — a `Scan`, one, over the exchange, with no port list — is derived
from ADR-0005, ADR-0019 and ADR-0028 and does not rest on a number. **Daily** rests on an argument
from the port tier's dependency, not on a measurement of DNS load, and no session has measured what a
day of control probes costs an operator's authorities. It is cheap to move — it is an operator dial
and moving it moves no version and `Break`s nothing — and it is flagged rather than smoothed over.

## Consequences

- **There are five `Scan`s and still two port tiers.** Every site saying *four* is wrong at the site
  it says it, per ADR-0058: [ADR-0005](./0005-scan-execution-model.md)'s decision table row and its
  #124 amendment, `CONTEXT.md`'s `Scan` entry, and
  [`packaging-and-configuration.md`](../spec/packaging-and-configuration.md) §7.2. The map's
  **CURRENT COMPOSED STATE** bullet carries the live absolute and is the orchestrator's to move.
- **`CONTEXT.md`'s `Scan` entry loses *which ports* from its opening enumeration**, since two of five
  `Scan`s have no port list. Withdrawn at the site.
- **`resolution` and `dns-record` gain a currency bound of `k` × the `dns` cadence — two days at
  shipped settings.** `Gap` becomes reachable on both facets, the `Address` membership predicate gains
  a truth value, and ADR-0013's gate gains a computable staleness for the resolutions it reads.
- **The observation retention floor becomes total.**
  [#139](https://github.com/winniel123/verge-asm/issues/139) is drawing `Coverage` against ADR-0041:
  the floor is the currency bound, so before this ruling it was a maximum taken over four of the six
  facets, silently under-retaining the two every membership decision reads. The floor's **number** does
  not move — it is set by the ~~slowest enabled~~ **slowest covering** `Scan`, and `dns` at daily is not
  it — but its **domain** does.

  > **Corrected at this site**
  > ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)) —
  > [#171](https://github.com/winniel123/verge-asm/issues/171) ·
  > [ADR-0094](./0094-a-retention-control-collapses-and-a-retention-query-never-does.md). *Slowest
  > **enabled*** is the **`Dispatch`** floor's rule; the observation floor is *slowest **covering***
  > (ADR-0081). The two differ by 4× today on any install with no zone file, where `zone` is enabled
  > and covers nothing. **And the sentence is now doubly superseded**: ADR-0094 withdraws the collapse
  > itself — the dial's floor is the **tightest** bound in force and the retirement query applies each
  > timeline's own bound beneath it. What this ruling moved is exactly the **domain**, as stated.
- ~~**ADR-0041's worked floor figure is stale, and not by this ruling.** It reads *"two weeks with
  `tls-acceptance` the slowest enabled one"*, which #124 falsified when it added `zone` at **monthly**
  and did not discharge. Reported to the map rather than repaired here, since #139 owns that surface.~~
  **REPAIRED, and struck at this site** (ADR-0058) — ADR-0041 no longer states the figure. It was
  withdrawn at ADR-0041's own site by the session that merged the fifteen-agent batch, with a note
  that `dns` at daily does not move the floor. Read alone and in the present tense this bullet sends a
  session to repair something already repaired. Walked against the corpus and confirmed by #171 ·
  ADR-0094.
- **`Coverage` gains a fifth `Scan` row**, and on the ADR-0019 install it is the only one with
  anything in its scope.
- **No aperture input, no declared parameter, no new number.** Seven inputs, `k` = 2, and the
  parameters of `resolution-walk` and `wildcard-discrimination` are untouched.
- **[#138](https://github.com/winniel123/verge-asm/issues/138) gets its precondition, not its
  answer.** One `Scan` over every vantage on one cadence is what makes a unanimity composition and an
  existential composition both expressible; which of them a leaf uses is #138's.
- **The passive sources' `Citation` staleness is open and ticketed**, and the estate of a source that
  errors is the hazard that ticket must price.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **`resolution` and `dns-record` ride the port tiers, as `certificate` rides `reachability`** — the option that lost, and the only one with an ADR behind it | ADR-0019's install measures these two facets **and nothing else**, so the exchange they would ride is not made there at all. `certificate` rides because it cannot be read without the connect; these are measured exactly where the connect cannot happen. Plus: disjoint populations in both directions, and the hidden-field failure ADR-0005 and #124 have each already refused |
| **Two new `Scan`s, one per facet** | `Shadowed` and `Lame` are committed across both facets inside one batch, so two cadences make one facet's value a function of another batch's control probe — ADR-0011's cross-batch assembly, and ADR-0028's own objection to a separately-scheduled handshake. It would also split the queried qtype set across two recorded scopes |
| **A DNS currency horizon as a declared parameter, with no `Scan`** | A currency bound that is not `k` cadences of a covering `Scan` is a fifth mechanism beside the one the whole model reads (`Observation`'s two tiers, ADR-0007, ADR-0041). And a measurement with a horizon but no cadence is ADR-0044's inexpressible object with a number bolted to it |
| **Call it `discovery`** | It would be a false description of the scope: CT, the registry paths and the zone file admit subjects and none of them rides this `Scan`. Read alone and in the present tense the name tells the next session to hang the crt.sh poll off it. The refused noun is the second reason, not the first |
| **Call it `resolution`** | A `Scan` named for one of the two facets it covers under-claims the other, and it collides with a facet name in the same sentence it would have to be read in |
| **Give it a port list of 53** | The port of the transport to the authority is not a dimension of what we may find; no value carries a per-port negative over it, widening it discovers nothing, and it would put a hidden aperture input on a `Scan` that has none. `zone` already established that a `Scan` need not have one |
| **Pin the cadence to the tightest enabled port tier rather than making it a dial** | A `Scan` with a computed cadence is the hidden field again, one layer up. And the failure it guards against is already handled: loosening `dns` withdraws `Address`es rather than probing stale ones |
| **Hourly** | An underived number, multiplying ten control labels per parent per qtype by 24 against the operator's own authorities, to buy currency nothing reads at that resolution |
| **Cover the passive sources' admissions on this `Scan` too** | Their hop is a `Citation`, not an observation; the consequence is subject withdrawal rather than a `Gap`; and it turns on whether a source the map measured returning spurious 404s may retire a `Name` by silence. A separate question, ticketed rather than guessed at — which is the move #124 made to produce this ADR |
