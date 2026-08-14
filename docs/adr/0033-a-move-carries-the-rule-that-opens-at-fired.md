# ADR-0033: A move carries the rule that opens at `fired` — and nothing carries a schedule

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#65 Does a rule opening at `fired` need a carrier, and can one be built without a three-way case analysis?](https://github.com/winniel123/verge-asm/issues/65)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md) ruled that entering or leaving a
`Predicate domain` is not a `Transition`, and
[ADR-0026](./0026-the-facet-layer-is-evidence-not-a-channel.md) §4 confirmed it: a rule that
**opens at `fired`** is carried by the census of a message above it where one exists, and by
nothing where none does. ADR-0026 refused a carrier rather than leaving the question open, priced
the silence, and ticketed the residue as [#65](https://github.com/winniel123/verge-asm/issues/65).

The population that reaches nobody is three shapes, all on subjects **already in the estate**, so
no membership message covers them ([ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md) §1)
and no leg moved ([ADR-0029](./0029-an-alert-fires-on-a-leg.md) §2):

1. an existing `Name` gaining a **CNAME whose target does not exist** — `cname-target-name-error`
   opens at *fired*, which is the case [#35](https://github.com/winniel123/verge-asm/issues/35)
   called the highest-confidence dangling signal available;
2. an existing `Endpoint` going `NoTLS` → `Presented` with a certificate that is already expired,
   self-signed or weak — up to six certificate rules open at *fired* at once;
3. a `tls-acceptance` timeline opening at *fired* on its own weekly `Scan`, days after its
   `Service` entered — ADR-0031's stated cost, which
   [ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md) narrowed and did not
   remove.

ADR-0026 refused all three together because the two available carriers both lose. **The general
form** — *any rule opening at `fired` is a message* — fires on every internal deploy, since
`plaintext-http-no-https` opens at *fired* on most estates, which is the message-per-deploy
ADR-0029 §3 refused on [#17](https://github.com/winniel123/verge-asm/issues/17)'s ground. **The
narrow form** — suppress the ones caused by a leg opening — needs the three-way case analysis on
*why the timeline opened* (membership, aperture, slower tier) that ADR-0031 rejected by name in its
own alternatives table. **A per-rule enumeration** is sixteen judgements resting on unmeasured base
rates.

### The three shapes are not one kind of thing, and that is the whole answer

Shapes 1 and 2 have a facet `Transition` underneath them: a `dns-record` CNAME timeline moving
from `NoData` to an RRset, a `certificate` timeline moving from `NoTLS` to `Presented`. Both values
are values — [ADR-0011](./0011-a-facet-is-six-parts.md) made every measured negative a variant of a
closed union precisely so that they would be. **The world moved, and we watched it move.**

Shape 3 has nothing underneath it. The `tls-acceptance` timeline *opened*: there is no earlier
span, nothing was compared, and the only thing that happened is that a weekly `Scan` we authored
came round. **We got to it.**

So the residue is not one population needing one carrier. It is a world event and an observer
event filed together, and separating them costs no case analysis at all, because the model already
derives the discriminator on read.

## Decision

**A facet `Transition` is a message where a `Signal` span opens at `fired` beneath it in the same
fold and no message in that fold already covers that opening. It fires once per `Transition`,
carrying the census of every rule that opened at `fired` beneath it. Drift class, *the world
moved*. An opening with no `Transition` beneath it stays silent, which is v1's answer to shape 3
and stays v1's answer.**

This is ADR-0026's own Decision applied rather than overturned — *a `Transition` is a message
exactly where it is the sole carrier of a fact the operator asked for*. It is ADR-0026 §2's
construction one layer across: §2 fires where a move opens **ground** nothing else covers, and
this fires where a move opens a **fired rule** nothing else covers.

### 1. It is not a screen, and the screen is barred from carrying it

[#44](https://github.com/winniel123/verge-asm/issues/44) renders a rule's census as three members
over one population, and a rule that opens at `fired` does render — as a `fired` row, alongside
every other `fired` row. The tempting reading is therefore that the obligation is already
discharged and #65 is asking for a channel where a screen exists.

It is not, and the bar is in ADR-0024's own fourth binding rather than in anything this ruling
adds. **A rule's census is current state and may never be rendered as a delta, a trend or a
series.** The fact at stake is not *this rule is firing here* — the screen says that — it is *this
rule started firing here*, which is a difference between two censuses, which is the one thing that
screen is forbidden to compute. The operator cannot tell a row that opened at `fired` last night
from one that has been `fired` for eight months, and making them distinguishable is exactly the
delta ADR-0024 refused.

The cause is visible on the drift surfaces, since the facet `Transition` beneath is recorded and
queryable. But that renders the cause without the consequence: *this endpoint started presenting a
certificate* is on the board, and *the certificate it started presenting expired in March* is not.
[ADR-0007](./0007-drift-is-a-timeline-of-spans.md)'s *alert on the cause, record the consequence*
is satisfied by this ruling and violated by the screen answer, which records the cause and drops
the consequence.

So the screen possibility is ruled **out** before anything is minted, which is the order
[#65](https://github.com/winniel123/verge-asm/issues/65) required.

### 2. The predicate is one term, and the case analysis is dissolved rather than performed

The predicate never asks why the `Signal` timeline opened. It asks whether a `Transition` exists
beneath it, and everything else follows from what a `Transition` already is:

| Why the rule opened at `fired` | Is there a `Transition` beneath? | Result |
| --- | --- | --- |
| The subject entered the estate (membership) | No — every timeline beneath a new subject **opens**, and an opening emits no `Transition` ([ADR-0014](./0014-only-revealed-generalises.md)) | Silent here; carried by ADR-0031 §3's census |
| The aperture widened | No — a widening is a `Break` or an opening, and **nothing is compared across a `Break`, so it emits no `Transition`** | Silent here; carried by ADR-0029 §7's census |
| A `Gap` closed | No — **no `Transition` crosses a `Gap`** | Silent here; already coverage class, member 7, fired at the cause (ADR-0026 §5) |
| A slower tier first covered the facet | No — the timeline **opens** | **Silent, and stays silent.** §4 |
| The subject entered the rule's domain because a value it reads moved | **Yes** | **Message** |

Five causes, one test, no branch. The discriminator is not a fact about the notification layer's
reasoning; it is a fact the comparison path already derives on read from two adjacent spans. That
is the answer to the ticket's second question: **a carrier can be built without a three-way case
analysis, because the three ways the analysis would have enumerated are exactly the three ways a
`Transition` fails to exist.**

Two constraints are met by construction rather than by care. **No threshold and no count in the
predicate** ([ADR-0007](./0007-drift-is-a-timeline-of-spans.md)) — the test is the existence of an
adjacency and the inhabitance of a set the fold already computed; every number lives in the census,
which is payload. And **no free read** (ADR-0024) — the `Transition` is on evidence the rule
already declares and already composes into its own leaf, so no vector grows and no corpus row is
added.

### 3. The message fires at the move, not at the rule — one cause, one message

An `Endpoint` going `NoTLS` → `Presented` with a certificate that is expired, self-signed and
weak-keyed opens **three** rules at `fired` in one fold. Three messages for one cause is ADR-0007's
*never one per affected subject*, verbatim.

So the message fires at the `Transition` and carries the census of what opened beneath it:

> *`admin.example.com:443/tcp` began presenting a certificate; it had none before. Three rules
> opened at `fired` — `certificate-expired`, `certificate-self-signed`,
> `certificate-weak-key-or-signature` — and three opened at *did not fire*. Nothing is compared.*

This is **ADR-0029 §7's payload shape with a fifth producer**, after the aperture widening, the
membership entry, the flagship and ADR-0026 §2's re-point. It is computed once at the cause, is a
description and never a `Transition`, carries no difference set, and alerts nothing individually.
ADR-0031 wrote that a third producer *"would be a signal that the shape is right"*; this is the
fifth.

The residue clause — *no message in that fold already covers that opening* — is ADR-0026 §2's own
wording and does the same job: where the flagship's census, a membership census or a re-point
census already carries the opening, this fires nothing and there is no doubling.

### 4. What stays silent, and it is the half that would have needed the case analysis

**A `Signal` opening at `fired` with no `Transition` beneath it reaches nobody, and that is
confirmed rather than deferred.** A `tls-acceptance` timeline opening at `fired` on its own weekly
`Scan` is not the world moving. It is our own schedule arriving, on a `Service` whose acceptance
set was exactly the same the day before and the week before that. Alerting it would report a cron
expression as an event, and it would fire for every `Service` in the estate on the first weekly
`Scan` after any `Address` enters.

ADR-0031's cost therefore stands unchanged: *the entry census covers only the tier that admitted
the subject*, and whether the census is emitted incrementally per completed `Batch` or the
membership message waits for a defined set of tiers is still the notification patch's and
[#4](https://github.com/winniel123/verge-asm/issues/4)'s profile's. This ruling does not touch it,
and the honest statement is that shape 3's real remedy is that scheduling question, not a channel.

### 5. It does not fire on a deploy, and this is where ADR-0026's refusal was over-broad

ADR-0026's decisive objection was the burst: *an internal deploy opens `plaintext-http-no-https` at
`fired` on most estates*. That is true of the general form and false of this one, and it fails in
both of the shapes a deploy takes.

- **A deploy that mints new containers, ports or vhosts** mints new `Service` and `Endpoint`
  subjects. Every timeline beneath them opens, so there is no `Transition` and no message. The
  internal port opening still tells nobody, exactly as ADR-0029 §3 ruled and priced — this ruling
  cannot re-admit it, because the door it would come through is an opening.
- **A deploy onto endpoints that already exist** leaves `http-identity` at `Responded` and
  `certificate` where it was. No rule enters a domain, so no rule opens: the rules are already
  open, and if the deploy made something worse the edge is `not-fired` → `fired`, which ADR-0026 §5
  already made a message. Nothing here is new.

The one shape that does fire is `NoHTTPResponse` → `Responded` on a plaintext listener — *this
thing started speaking HTTP and speaks it in clear*. That is a world move on a subject already in
the estate, and it is the fact `plaintext-http-no-https` is named for.

The same check kills the second half of the objection. A closed or refused port yields no HTTP
exchange and no handshake, so `http-identity` and `certificate` hold a `Gap` rather than a measured
negative, and a `Gap` closing carries no `Transition` — so a service restarting, a port flapping
and an internal port opening are all outside this predicate by the same clause, and the ones that
carry news are already coverage class under ADR-0026 §5.

### 6. The growing direction only

Nothing here touches the closing side. A rule that was `fired` and **leaves** its domain — the
CNAME is deleted, the endpoint stops presenting a certificate — closes its span with no message,
which is ADR-0024's amendment unchanged. That is the shrinking direction
[ADR-0006](./0006-subjects-leave-by-measurement.md), ADR-0029 §4 and ADR-0026 §1 all silence, and
it is overwhelmingly the operator's own remediation. #35's *a clear may be the attack having
succeeded* is untouched and lands where it always did: on the four rules whose **within-domain**
clear is a message under ADR-0026 §5, which is where the takeover-relevant clear actually sits — a
CNAME's target starting to exist is `fired` → `not-fired`, not a domain exit.

### 7. Nothing is minted

No fifth cause: this is *the world moved*, drift class, which is the class both the move and the
rule were always in. No fourth class. No new coverage-class member — the class stays at nine. No
new `Transition` name, and no `Signal`-layer transition: **entering a `Predicate domain` is still
not a `Transition`**, which is ADR-0024's ruling and ADR-0026 §4's confirmation, and this ADR
relies on it rather than weakening it. The only additions are a predicate widening on an existing
message and a fifth producer of an existing payload shape.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) is amended in three entries** — `Transition` records that a
  move carries the rule that opens at `fired` beneath it; `Signal` replaces *"carried by the census
  of a message above it or by nothing"* with the move-or-schedule cut; `Predicate domain` restates
  the stated cost, which is now the schedule half alone.
- **ADR-0026 §1's table is amended in three rows, and its own reason column is where the residue
  was hiding.** `certificate`'s reason addresses `Presented(c1)` → `Presented(c2)` and
  `Presented` → `NoTLS` and never addresses `NoTLS` → `Presented` — the growing direction is
  simply absent from the argument while being covered by the row's *none*.
  `http-identity`'s reason routes `Responded` ↔ `NoHTTPResponse` to §4 in both directions at once.
  And `dns-record`'s reason — *growth reached through a record is a `Name` `appeared` and
  membership carries it* — is sound for a CNAME target that **exists** and false for one that does
  not, which is the only case that fires a rule. Each row now reads *none, except where the move
  opens a rule at `fired`*.
- **ADR-0026 §4 is narrowed, not withdrawn.** *A rule that opens at `fired` is carried by the
  census of a message above it where one exists and by nothing where none does* becomes: by the
  census where one exists, by the move beneath it where there is one, and by nothing where the
  timeline merely opened.
- **`dns-record` is still not a channel.** An MX, TXT or NS change reaches nobody, because no v1
  rule reads those qtypes and so none can open at `fired` beneath them. The map's **Out of scope**
  entry for a facet-level `dns-record` channel stands untouched; what this ruling adds is one
  `dns-record` move that is the sole carrier of a rule, not a channel for the facet.
- **The subdomain-takeover case #35 built a rule for is now covered in both of its shapes.** A
  CNAME whose target is deleted was already a message (`not-fired` → `fired`, ADR-0026 §5); a CNAME
  created already dangling is one now. Neither shape is bigger than the other on any measurement in
  this repo, and previously one of them reached nobody.
- **A fifth wording pair for the notification patch**, after
  [#28](https://github.com/winniel123/verge-asm/issues/28)/[#22](https://github.com/winniel123/verge-asm/issues/22)'s,
  [#55](https://github.com/winniel123/verge-asm/issues/55)/[#51](https://github.com/winniel123/verge-asm/issues/51)'s,
  ADR-0031's and ADR-0026's: this message and ADR-0026 §2's can arrive on one `Name` in one fold —
  a re-point that opens `Endpoint`s **and** a `dns-record` move that opens a rule — and they must
  not read as duplicates.
- **The census payload has five producers, not four**, so the patch's wording work is over five
  messages that all render a census at a cause.
- **Decided on thin ground in one place, and it is not dressed as a derivation.** The base rate of
  `NoHTTPResponse` → `Responded` on an internal estate is unmeasured, exactly as ADR-0029 and
  ADR-0031 left their volumes unmeasured. The argument that it is not the deploy shape is
  structural and holds; the argument that it is *rare* is an assertion. If it drowns the channel the
  remedy is coalescing in the notification patch and never a predicate change, because the
  `Transition` is recorded either way — one predicate in the notification layer, no timeline
  touched.
- **This does not block [#12](https://github.com/winniel123/verge-asm/issues/12) and did not
  need to.** v1's answer was silence with the cost stated, and #12 can carry either. What #12 must
  carry now is one more message and one more census producer.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **The v1 silence stands — close #65 with no change** | The strongest competitor, and it was ADR-0026's own ruling, so it starts ahead. It loses on the finding that the residue is two populations rather than one: the case ADR-0026 refused for needing a three-way case analysis needs none, because a `Transition` either exists beneath the opening or does not, and that is one boolean the model already derives. Refusing a carrier that costs one predicate, no new vocabulary, no threshold and no version leaf, in order to keep a silence whose stated price is the highest-confidence dangling signal in the set, is paying for a difficulty that turned out not to be there |
| **It is a screen, not a channel — #44's `Signals` rows already render it** | The ticket's own first-check option and the one that had to be ruled out before anything was minted. It loses on ADR-0024's fourth binding: a census is current state and may never be rendered as a delta, so the screen renders the row and is **forbidden** to render that it is new. The screen cannot carry the fact, so the rendering obligation is already discharged and the news is still uncarried |
| **The general form — any rule opening at `fired` is a message** | ADR-0026's refusal, and it stands. It fires on a subject entering the estate, on an aperture widening, on a `Gap` closing and on every weekly `Scan` reaching a `Service` for the first time — four populations the model has already routed elsewhere, three of them observer events, and the last of them a cron expression reported as news |
| **The narrow form as ADR-0026 framed it — suppress the openings caused by a leg opening** | Needs the three-way case analysis on *why the timeline opened* that ADR-0031 rejected by name, and the analysis has to run in the notification layer over causes it cannot see. §2 does not perform it; it tests one adjacency and lets the cases fall out |
| **Per-rule enumeration of which openings notify** | Sixteen judgements on unmeasured base rates, in a map that has flagged #17's unmeasured base rates three times. It is also unnecessary once the cut is on the move rather than on the rule: the same one test cuts all sixteen |
| **Fire at the `Signal` rather than at the `Transition`** | One `NoTLS` → `Presented` opens up to six certificate rules at `fired` at once, so this is six messages for one cause — ADR-0007's *never one per affected subject*, verbatim, and ADR-0031's root rule ignored one layer down |
| **Mint a fifth cause, or a coverage-class member, for *a rule started firing*** | The world moved: a certificate appeared on a port that had none, a CNAME was created that points nowhere. Our looking did not change. Filing it under an observer cause is ADR-0031's *rooting an appearance at the `Seed`* defect again, and the map's constraint is that a fifth cause needs a reason rather than a slot |
| **Include the slower-tier opening by testing whether the subject was already in the estate** | A simpler-looking predicate that reaches shape 3, and it is wrong in the direction that matters: it reports our own weekly `Scan` as an event, fires across the whole estate on the first enumeration after any entry, and makes the message's volume a function of the cadence table rather than of the world |
| **Extend it to the closing side — a rule leaving its domain while `fired` is a message** | Symmetry for its own sake, and it fires on every remediation in the estate: deleting a dangling CNAME, terminating TLS, taking an endpoint off HTTP. It is the shrinking direction ADR-0006, ADR-0029 §4 and ADR-0026 §1 each silence, and #35's *a clear is not always good news* is already served by ADR-0026 §5's four within-domain clears |
| **Let the message carry a difference set — which rules newly fire versus last fold** | A delta over a census by another name. ADR-0024 refuses it on the screen and nothing makes it legal in a payload; the census is computed once at the cause and asserts nothing about last cadence |
