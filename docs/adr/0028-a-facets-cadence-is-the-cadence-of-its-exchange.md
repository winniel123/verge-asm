# A facet's cadence is the cadence of the exchange that measures it

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#61 On which scan tier is the TLS handshake that feeds `certificate` performed?](https://github.com/winniel123/verge-asm/issues/61)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

Three accepted documents disagreed about which `Scan` feeds the `certificate` facet, and
[ADR-0027](./0027-a-source-may-admit-without-observing.md) recorded the disagreement rather than
guessing at it.

- [ADR-0011](./0011-a-facet-is-six-parts.md) has TLS attempted on **every open `Service`,
  opportunistically** — deleting the implicit-TLS port list rather than pricing it — calls
  `certificate`'s evidence *"the daily handshake"*, and separates `tls-acceptance` as the facet fed
  by [#4](https://github.com/winniel123/verge-asm/issues/4)'s **weekly enumeration**. Its
  consequence *"`tls-1.0-accepted` is now the slowest-moving signal in the set, and a `Gap` opens
  two weeks after the weekly tier stops rather than two days"* is only coherent if `certificate` is
  **not** weekly.
- [ADR-0014](./0014-only-revealed-generalises.md) says the opposite in the worked example that
  carries its central rule: *"A `Service` appears at 02:00; TLS is attempted on the weekly tier; its
  `certificate` timeline opens six days later."*
- [`CONTEXT.md`](../../CONTEXT.md)'s `Transition` entry repeats ADR-0014's version, and
  [ADR-0007](./0007-drift-is-a-timeline-of-spans.md)'s `#42` amendment repeats it a third time.

Two further texts, neither cited by the ticket, sit on ADR-0011's side.
[ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md) builds its forcing
measurement on *"the **every-run handshake** that feeds `certificate`"*, and
[#4](https://github.com/winniel123/verge-asm/issues/4) §5 — the research that decided the tiers —
draws the contrast explicitly: *"One handshake per run yields expiry, issuer, SAN, version …
Version/cipher **enumeration** is N handshakes, so weekly not daily."*

It is not a cadence preference. [ADR-0007](./0007-drift-is-a-timeline-of-spans.md) makes currency
`k` cadences of the Declared `Scan` covering the thing, so the answer decides when a `Gap` opens on
every `certificate` timeline in the estate, and therefore when the expiry and self-signed rules from
[#16](https://github.com/winniel123/verge-asm/issues/16) go `not-evaluable`.

[#60](https://github.com/winniel123/verge-asm/issues/60) sharpened the stake while this ticket was
open, and it is worth stating before the decision rather than after. The **clock class** —
`certificate-expired`, `certificate-not-yet-valid` and `certificate-expiring` — is the only place in
v1 where a rule reads an **always-current wall clock** against a **possibly stale observed value**.
Everywhere else both sides age together, so staleness cannot make a comparison lie. Here it can, and
the lie is a rule firing on a certificate that has already been replaced. The width of that window
is exactly the currency bound this ADR sets, three rules wide.

## Decision

| Concern | Decision |
| --- | --- |
| On which tier is the `certificate` handshake performed? | ***The* tier is the wrong question.** The handshake is a **step in the same exchange** that produces `reachability` for the `Service`, so it rides whichever of [#4](https://github.com/winniel123/verge-asm/issues/4)'s ~~three~~ **two** `Scan`s ran it — [#78](https://github.com/winniel123/verge-asm/issues/78) retired the weekly top-1000 tier, per the Consequences below and [ADR-0005](./0005-scan-execution-model.md)'s #80 amendment |
| `certificate`'s currency bound | Whatever [ADR-0007](./0007-drift-is-a-timeline-of-spans.md) already computes for the `Service` — *the tightest cadence where several apply*. **Daily** for a `verge-core` port, ~~weekly for a top-1000-only port,~~ monthly for a full-range-only one. **The weekly top-1000 limb is WITHDRAWN** — [#78](https://github.com/winniel123/verge-asm/issues/78) retired that tier, so no port is covered by it and the bound has two limbs, not three |
| Do the ~~three~~ **two** tiers produce ~~three~~ **two** timelines? | **No.** One timeline — same subject, facet, discriminator, vantage and source — fed by all of them |
| The `certificate` handshake's declared candidate set | **One set, identical across all ~~three~~ two reachability `Scan`s.** Its content is [#62](https://github.com/winniel123/verge-asm/issues/62)'s |
| Is `tls-acceptance` on the weekly **tier**? | **No.** #4's *"weekly not daily"* is a **cadence**, decided on N-handshakes-versus-one. The weekly `Scan`'s scope is a **port set**, which is a different thing |
| Where `tls-acceptance` sits | A `Scan` **of its own** — ~~a **fourth `Scan`**~~, and **third** since [#78](https://github.com/winniel123/verge-asm/issues/78) retired the weekly top-1000 tier — weekly cadence, scope the open `Service` population plus the candidate set, **no port tier**. The ordinal is not load-bearing; *a `Scan` of its own* is |
| [ADR-0005](./0005-scan-execution-model.md)'s *"Three `Scan`s"* | **Amended to ~~four~~ three.** On ADR-0005's own rule, not against it — and ADR-0005's own #80 amendment restates the count as **three, not four** |
| May `certificate` and `tls-acceptance` sit on different cadences? | **Yes, and they must.** They are different exchanges against different subjects |
| ADR-0014's worked example | **The rule survives; the example moves to `tls-acceptance`** — same shape, same six days, and live under this ruling |
| Which text is repaired | ADR-0014's example, [ADR-0007](./0007-drift-is-a-timeline-of-spans.md)'s restatement of it, and `CONTEXT.md`'s `Transition` and `tls-acceptance` entries |
| Which text is vindicated | ADR-0011 and ADR-0025, and #4 §5. Nothing on that side is stale |

## Rationale

### `certificate`'s value space has no variant for a port that is not open

This is the argument that decides it, and it is structural rather than a scheduling preference.

`certificate` is `Presented(chain) │ TLSRefused │ NoTLS`
([ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md)). Every variant
presupposes something answering on the port. There is no variant meaning *the port was shut*, and
minting one would be `reachability` rebuilt inside a second facet.

So a handshake scheduled apart from the connect has two ways to decide whether it may emit anything
at all, and both are closed. It can **import** openness from another batch — a cross-observation
dependency with its own currency problem inside the comparison path, which ADR-0011 refused in terms
when it ruled that *a value needing two measurements is decided by the measurement binary inside one
batch, never assembled afterwards*. Or it can **re-derive** openness from its own connect, in which
case `reachability` and `certificate` hold two independent connect verdicts on one `Service` taken
at different times, and the map's seam rule bites: two measurements of one fact, with a change in
the observer surfacing as a change in the world.

[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s leaves say the same thing from the
other end. `connect-outcome`, `tls-handshake` and `http-exchange` are three decisions on one
endpoint exchange, and nothing anywhere proposes that `http-identity` runs on a slower clock than
`reachability`. `certificate` was the only one anybody thought was scheduled separately, and the
reason is traceable: ADR-0011 introduced `tls-acceptance`'s weekly enumeration and ADR-0014 read
*TLS is weekly* off it.

Note what the argument does **not** rest on. It is not that a handshake is cheap — #4 caps TLS at 5
handshakes/s against 50 pkt/s for connects, so it is the dearest per-service probe in the profile.
It is that the value cannot be interpreted without the connect it rides.

### The honest answer is that there is no single tier, and ADR-0007 already handles it

ADR-0011's *every open `Service`, opportunistically* reads as riding whatever tier discovered the
service, and that reading is correct. A `Service` on a `verge-core` port is exchanged with daily,
weekly and monthly; one only in nmap's top-1000 weekly and monthly; one reachable only by the
opt-in full-range sweep, monthly.

Three tiers do not make three timelines. The key is
`(subject, facet, discriminator, vantage, source)` and the tier is in none of those components, so
all three batches fold into one `certificate` timeline — which is why the tiers must declare **one**
candidate set between them. Differing sets would put observations under two apertures on one
timeline with nothing in the model to name the difference, and ADR-0025's *"each records its own
candidate set, and they need not match"* was said of `certificate` against `tls-acceptance`, not of
the reachability tiers against each other.

Currency needs no new machinery either. ADR-0007 already reads *"the tightest such cadence where
several apply"*, so `certificate`'s bound is `reachability`'s bound for the same `Service`, computed
by the rule that is already there. The estate-wide answer everyone wanted — *is it one day or
seven?* — does not exist, and the product is better for not inventing it: the tiers exist precisely
to buy non-uniform freshness, and `reachability` and `http-identity` have had non-uniform bounds
since #4.

### `tls-acceptance`'s "weekly" is a cadence, and reading it as a tier loses the sensitive ports

The conflation that produced this ticket is between *the weekly tier* — nmap's top-1000, a **port
set** — and *the weekly enumeration* — N handshakes per service, a **cost**. #4 §5 argues purely
from cost: one handshake every run, N handshakes weekly. Nothing in #4 ties the enumeration to a
port set, and ADR-0011 slid from *"#4's weekly enumeration"* in its rationale to *"the weekly tier"*
in its consequence within one document.

Reading it as the tier is not merely loose, it loses real estate.
[ADR-0009](./0009-verge-core-is-a-union.md) made `verge-core` the union `frequency-set ∪
sensitive-list`, and measured that **none of the four missing pairs belongs in the hot set on the
hot set's own evidence standard** — 512/tcp ranks 239th by open-frequency, and 4369, 25672 and 27019
rank nowhere. Ports that rank nowhere by frequency are not in nmap's top-1000. So on the tier
reading, `tls-acceptance` is never measured on the ports the product deliberately probes *because*
they are never legitimately internet-facing, and `tls-1.0-accepted` reports nothing on part of
exactly the population it exists for. That is
[ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md)'s forcing failure in a
second guise, arriving through a port tier instead of a library default.

It also rebuilds the thing ADR-0011 deleted. *A curated implicit-TLS port list* was rejected because
it *"prices an aperture input rather than deleting one"*; scoping the enumeration to top-1000 is the
same list wearing nmap-2008's clothes, and ADR-0009 is the standing lesson about hand-maintained
port tables nobody derives. `CONTEXT.md` already contradicts itself on this inside one sentence —
*"by enumeration on the weekly tier, one handshake per candidate, and attempted against every open
service rather than a curated implicit-TLS port list"* — since the top-1000 port set is not every
open service.

### A fourth `Scan`, and ADR-0005 is the authority for it rather than the obstacle

Having ruled that the enumeration is weekly and estate-wide, there are two places to put it, and
ADR-0005 has already chosen between them. *"Port tiers are three `Scan`s"* was decided against *one
`Scan` with tiered cadence*, because a cadence hidden inside another `Scan` makes the aperture a
hidden field rather than a configured object, and *"the operator disabled the weekly deep scan"*
stops being a legible state. Hanging a weekly enumeration off the daily reachability `Scan` as
"every seventh run" is that rejected shape exactly.

So the enumeration is a `Scan` of its own. Its scope is not a port set — it is the open `Service`
population together with the TLS candidate set, which is the scope its `Batch` records under
[ADR-0011](./0011-a-facet-is-six-parts.md) and
[ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md). ADR-0005's partitioning
rule is satisfied: the enumeration retains completeness over any subset of services and any subset
of candidates.

The obvious objection is the one this ADR used against a separately-scheduled `certificate`
handshake: the fourth `Scan` targets a population read from other batches' observations. The
distinction is between **targeting** and **value assembly**, and it is not a fine one. Every prober
batch already targets subjects discovered by earlier batches — an `Endpoint` takes its `Name` from a
`resolution` batch — and ADR-0011 forbids only a *value* assembled from two observations.
`tls-acceptance`'s value is decided wholly inside its own batch, because the enumeration makes its
own N connections: all connects failing is distinguishable in band from all handshakes being
rejected, so the batch never has to consume a connect verdict it did not make. A separately
scheduled `certificate` handshake would have to consume one, because deciding `NoTLS` against
*nothing to record* **is** a connect verdict wearing a value's clothes.

### ADR-0014's rule survives and keeps a live worked example

ADR-0014 used the six-days-later case to establish that **an opening caused by neither the world nor
our aperture is recorded, unnamed and unalerted** — a rule that binds map-wide and that
[ADR-0017](./0017-exposure-needs-both-legs.md) has already leaned on. The rule is untouched. Only
its instance was wrong, and an instance of the identical shape is live under this ruling:

> A `Service` appears at 02:00 and is exchanged with on the daily tier. Its `tls-acceptance`
> timeline opens when the weekly enumeration next runs — up to six days later. The service was
> there all along, so the world did not move; the enumeration always covered it, so the aperture
> did not widen. We simply got round to looking.

Same shape, same six days, same conclusion. What could **not** replace it is a slower reachability
tier: where the slower tier is what *discovers* the `Service`, the subject enters the estate and its
timelines open with it, so `appeared` carries the news at the cause and the opening is not unnamed
at all. ADR-0017's *"a leg's timeline opens when a slower tier first covers that port"* survives for
the same reason it was correct — there the subject already existed on the other vantage's leg.

*Amended by [#63](https://github.com/winniel123/verge-asm/issues/63) /
[ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md), landed the same day.*
The sentence *"`appeared` carries the news at the cause and the opening is not unnamed at all"* is
**withdrawn**. A `Service` `appeared` is never a message: membership alerts fire at the **root of
the entering sub-tree** and only a `Name` or an `Address` can be a root, so where a slower tier
discovers a `Service` beneath an `Address` already in the estate, the root walk terminates at that
`Address` and **nothing carries the discovery at all**. The Decision does not move, and the
correction **strengthens** it — the rejected alternative loses the news outright rather than having
membership carry it, which is a cost of the alternative rather than a relief. ADR-0017's clause is
untouched, its subject having already existed.

### Repaired or recorded: ADR-0020's test is whether a reader acting on the text goes wrong

[ADR-0020](./0020-a-conflict-needs-two-enumerable-sources.md) set the precedent for recording rather
than repairing where the substance is unaffected. Here the substance is affected, and on exactly one
side.

**Repaired**, because a session acting on them schedules the handshake on the wrong `Scan`:
ADR-0014's worked example, ADR-0007's `#42`-amendment restatement of it, `CONTEXT.md`'s `Transition`
entry, and `CONTEXT.md`'s `tls-acceptance` entry. Each gets a marked amendment quoting what it
originally said, following ADR-0011's own precedent for a single withdrawn clause — never a silent
rewrite, and placed at the sentence so that a session grepping `weekly tier` finds the correction at
the hit rather than in a footer.

**Vindicated and unchanged in substance**: ADR-0011's *"the daily handshake"*, its `tls-acceptance`
consequence (which was right that `tls-1.0-accepted` is the slowest-moving signal in the set, and
needs only *the weekly tier* → *its weekly `Scan`*), ADR-0025's *"every-run handshake"*, and #4 §5.
The research note stands unrewritten under ADR-0007's *a wrong record is corrected by a new entry*;
nothing in it was wrong.

### Where this was decided on thin ground

The `certificate` half is firm: four texts and the originating research on one side, one claim
repeated three times on the other, and a structural argument from the value space that does not
depend on counting texts.

The **fourth `Scan`** is thinner and is flagged as such. No document proposes one; it is derived
from ADR-0005's one-cadence-per-`Scan` rule plus the measured `verge-core ⊄ top-1000` gap. The
alternative that would survive is scoping the enumeration to the weekly tier and accepting that four
sensitive ports go unenumerated — a coverage loss the model can state honestly. It was rejected
because those four ports are the ones ADR-0009 went to the trouble of unioning in, and a facet whose
population silently differs from `certificate`'s by a 2008 frequency ranking is the defect this ADR
exists to remove, not a cheaper version of the fix.

## Consequences

- **[ADR-0005](./0005-scan-execution-model.md) is amended: three `Scan`s become four.** The fourth
  is not a port tier and must not be modelled as one — its scope is the open `Service` population
  and the TLS candidate set. Everything else in ADR-0005 applies to it unchanged: it is Declared, a
  manual dispatch does not reset its cadence, and its `Batch` records what it completed.
- **[ADR-0011](./0011-a-facet-is-six-parts.md) is amended in one word and vindicated in
  substance.** *"A `Gap` opens two weeks after the weekly tier stops"* becomes *after its weekly
  `Scan` stops*; `tls-1.0-accepted` remains the slowest-moving signal in the set, and that is now
  true for a stated reason rather than by accident.
- **[ADR-0014](./0014-only-revealed-generalises.md)'s rule is untouched and its worked example
  moves to `tls-acceptance`.** Anything downstream that cited the rule — ADR-0017's opening leg,
  ADR-0015's vacuous `Break`, [#42](https://github.com/winniel123/verge-asm/issues/42)'s
  coverage-class reasoning — is unaffected.
- **[ADR-0007](./0007-drift-is-a-timeline-of-spans.md) needs no new currency machinery.** *The
  tightest cadence where several apply* was already the rule, and it turns out to have been
  answering this question since it was written. Only its restatement of ADR-0014's example is
  corrected.
- **`certificate`'s currency is per-`Service` and there is no estate-wide number.** Modal `k`=2 on a
  daily tier is two days; ~~a top-1000-only port is two weeks;~~ a full-range-only port is two months.
  The expiry and self-signed rules from [#16](https://github.com/winniel123/verge-asm/issues/16) go
  `not-evaluable` on that bound, per-subject.
  > **The two-week limb is WITHDRAWN**: [#78](https://github.com/winniel123/verge-asm/issues/78)
  > retired the weekly top-1000 tier, so there is no *top-1000-only port* to bound — see the bullet
  > below recording that retirement, and [ADR-0005](./0005-scan-execution-model.md)'s #80 amendment.
  > Marked at the sentence per
  > [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened
  > by [#106](https://github.com/winniel123/verge-asm/issues/106).
- **This bound is the only one in the model doing *safety* work rather than coverage work, and
  this ruling is what makes it narrow.**
  [#60](https://github.com/winniel123/verge-asm/issues/60) found that the clock class — three rules
  after ADR-0024's domain table, `certificate-expired`, `certificate-not-yet-valid` and
  `certificate-expiring` — is the one place in v1 where a rule compares an **always-current wall
  clock** against a **possibly stale observed value**. Every other signal reads observed values
  against each other, so staleness moves both sides together and the comparison stays honest; here
  it does not, and the failure mode is all three rules asserting a fact about a certificate that has
  already been replaced. The exposure window is exactly `certificate`'s currency bound, which is
  what this ADR sets. Two things follow. **The guard is structural and already present**: past `k`
  cadences the value stops being current, so the rules go `not-evaluable` rather than firing on a
  stale `not_after` — the window is bounded by the mechanism ADR-0007 already built, not by
  discipline. And **the window is now two days on the modal estate** rather than the seven the
  withdrawn ADR-0014 reading implied, because the handshake rides the daily tier for any
  `verge-core` port. #60's amendment to ADR-0004 left this hanging and pointed here; it is answered
  in the narrowing direction, and no new machinery is needed.

  *Amended by [#77](https://github.com/winniel123/verge-asm/issues/77) /
  [ADR-0043](./0043-a-clock-reading-rule-bounds-its-evidence-in-the-subjects-own-units.md),
  2026-08-14.* The clause *"the guard is structural and already present"* is **withdrawn as a
  sufficient guard** and survives as a necessary one. It bounds the window in **our** units — how
  often we looked — where the hazard is measured in the subject's: the guard is safe only while
  `k × cadence` is small against the observed certificate's own validity period, a comparison this
  ADR made once, silently, against 90 days. **Inside** the bound the observation is current and the
  rules fire on it, so the failure is not late silence but a false assertion —
  `certificate-expired` firing true about an endpoint serving a valid certificate. The clock class
  now carries a second, per-rule gate: an observation feeds it only while its age is within the
  certificate's own horizon, `N = ⅓ × (not_after − not_before)` and `½ ×` below a 10-day validity,
  so the effective bound is `min(k × cadence, N)`. `k` does not move and no `Gap` opens on the new
  gate. **The sentence's second half is untouched** — on the hot tier the horizon binds only below a
  four-day validity period and none exists, so *two days on the modal estate* is exactly as true as
  when it was written. What moves this is the **six-day GA fact retrieved the next day** by
  [#67](https://github.com/winniel123/verge-asm/issues/67), not a re-reading of this ADR: against
  the lifetimes it was priced against, it was correct.
- **The opt-in full-range tier carries a stated cost, and it is the clock class that pays it.** A
  `Service` only that tier reaches holds a `certificate` value current for two months and
  `not-evaluable` the rest of the time — so on that population the three clock rules are silent far
  more often than they speak, and for up to two months they can speak about a certificate no longer
  served. That is honest rather than fixable: you cannot handshake a port you do not yet know is
  open, and narrowing the window would mean probing the full range more often, which is the cost
  #4 declined. It is a coverage rendering question, not a schedule one.

  *Amended by [#77](https://github.com/winniel123/verge-asm/issues/77) /
  [ADR-0043](./0043-a-clock-reading-rule-bounds-its-evidence-in-the-subjects-own-units.md),
  2026-08-14.* The **silence** half stands and is strengthened: the clock rules are silent on that
  population more often still, and silence is honest. The **speaking** half is **withdrawn**, and
  the wrong word is *fixable*. It rested on there being exactly one lever — the probe schedule — and
  that reasoning about the schedule remains correct and is not reopened. There was a second lever
  this ADR could not see, because the fraction expressing it was retrieved the following day: **stop
  reading the value earlier, in the certificate's units rather than ours**, which costs no probes at
  all. *Honest* does not survive the arithmetic either — a 60-day-old observation of a 160-hour
  certificate is nine generations stale, so the value is not possibly superseded but **certainly**
  superseded, and a rule asserting from certainly-superseded evidence is wrong rather than
  honest-but-stale. Two further figures in this bullet have moved underneath it:
  [#78](https://github.com/winniel123/verge-asm/issues/78) **retired the weekly top-1000 tier** on
  2026-08-14, leaving two tiers rather than three, and the CA/Browser Forum ceiling reaches **100
  days on 2027-03-15**, from which date every publicly-trusted certificate is inside the new gate on
  this tier.
- **The `certificate` handshake's candidate set is one declared set across all ~~three~~ **two**
  reachability `Scan`s.** [#62](https://github.com/winniel123/verge-asm/issues/62) decides its content;
  this ADR decides only that there is exactly one of it and that it is not the library's default, which
  ADR-0025 already settled.
  > **Two, not three**, since [#78](https://github.com/winniel123/verge-asm/issues/78) retired the
  > weekly top-1000 tier — recorded in the bullet above and, cross-document, in
  > [ADR-0005](./0005-scan-execution-model.md)'s #80 amendment, which names *this* sentence as
  > *"stranded the same way"* and yet was written only there. Marked at the sentence per
  > [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened
  > by [#106](https://github.com/winniel123/verge-asm/issues/106). **One set** is what the ruling is;
  > the count is incidental to it.
- **[`CONTEXT.md`](../../CONTEXT.md) changes in four places.** `Transition`'s worked example moves
  to `tls-acceptance`; `tls-acceptance` loses *the weekly tier*; `certificate` gains the clause
  saying its handshake rides the reachability exchange; and `Scan` records that not every `Scan` is
  a port tier.
- **Nothing costs a `Break`, a `revealed` or a message.** No aperture dimension moved, no
  `Derivation` version moved, and nothing has shipped. This is a defect in the documents, which is
  why [ADR-0015](./0015-the-value-space-is-the-commitment.md) gave it no deadline.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| `certificate` on the weekly tier, as ADR-0014 and `CONTEXT.md` say | The value space has no variant for a closed port, so the handshake cannot be interpreted apart from the connect it rides; and it makes ADR-0011's own *slowest-moving signal in the set* consequence incoherent, since `certificate` would tie |
| A dedicated `certificate` `Scan` on a uniform daily cadence | Buys a uniform currency bound with a second connect verdict on every `Service` — two measurements of one fact, the seam rule's exact shape — and its target population would have to be imported from the reachability timelines |
| `certificate` on the daily tier alone, ignoring the slower tiers | Silently drops every `Service` the daily tier does not cover, so a top-1000-only listener would hold a `Service` with no `certificate` timeline and no record saying why |
| `tls-acceptance` on the weekly top-1000 tier, as written | `verge-core`'s sensitive-only members rank nowhere by frequency and are not in top-1000, so `tls-1.0-accepted` never evaluates on the ports the product added because they are never legitimately internet-facing |
| The weekly enumeration as "every seventh run" of the daily `Scan` | ADR-0005's rejected *one `Scan` with tiered cadence*: the aperture becomes a hidden field and *the operator disabled the enumeration* stops being a legible state |
| Redefining the weekly tier as `top-1000 ∪ verge-core` so the enumeration can ride it | Fixes the symptom by editing the port tiers, which are a different object with a different evidence standard, and leaves the enumeration's population defined by a 2008 frequency ranking it has no reason to inherit |
| Recording ADR-0014's example as stale without moving it | The rule binds map-wide and a rule with a withdrawn example is a rule nobody can check; an instance of the identical shape exists and costs nothing to substitute |
| Deleting ADR-0014's example and abstracting the rule | Loses the six-days case that made the rule legible, in exchange for avoiding a two-line edit |
