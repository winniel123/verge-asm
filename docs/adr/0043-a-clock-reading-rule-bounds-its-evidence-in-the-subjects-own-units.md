# A clock-reading rule bounds its evidence in the subject's own units

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#77 Is `certificate`'s currency bound still safe for the clock class, now that six-day certificates are generally available?](https://github.com/winniel123/verge-asm/issues/77)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md) settled that `certificate`
rides the `reachability` exchange, so its currency bound is whatever
[ADR-0007](./0007-drift-is-a-timeline-of-spans.md) already computes for the `Service` — `k`
cadences of the tightest Declared `Scan` covering the port. It then priced the **clock class** —
`certificate-expired`, `certificate-not-yet-valid`, `certificate-expiring` — and pronounced the
residue acceptable:

> **The guard is structural and already present**: past `k` cadences the value stops being current,
> so the rules go `not-evaluable` rather than firing on a stale `not_after`.

> A `Service` only that tier reaches holds a `certificate` value current for **two months** … for up
> to two months they can speak about a certificate no longer served. **That is honest rather than
> fixable.**

[#71](https://github.com/winniel123/verge-asm/issues/71) swept for constants that are products of a
moving world quantity, found that neither `k` nor any cadence is one, and recorded the sharper
finding in [ADR-0038](./0038-a-constant-is-a-product-only-where-the-quantity-is-readable.md) §6:

> **ADR-0034 §4 catches stale numbers. It does not catch stale arguments, and the argument is where
> the residual risk in this repository actually sits.**

The stale argument is ADR-0028's. `k × cadence` is a duration in **our** units — how often we
looked. What it is asked to bound is the drift between an always-current wall clock and an observed
`not_after`, and that drift is only small if the window is small **against the certificate's own
life**. ADR-0028 performed that comparison once, silently, against a certificate lifetime with a
published expiry date.

Three facts retrieved *after* it move the price. [#67](https://github.com/winniel123/verge-asm/issues/67)
([`acme-renewal-timing.md`](../research/acme-renewal-timing.md) §7.1–§7.3) established that
certificate lifetimes are plural and shrinking on published schedules: Let's Encrypt's 160-hour
profile went **generally available on 2026-01-15**; its `tlsserver` profile has issued 45-day
certificates since **2026-05-13**; its default `classic` profile drops to **64 days on 2027-02-10**
and 45 days on 2028-02-16; and the CA/Browser Forum ceiling, already **200 days since 2026-03-15**,
drops to **100 days on 2027-03-15** and 47 days on 2029-03-15. In the same retrieval,
`certificate-expiring`'s horizon stopped being a day count and became
`N = ⅓ × (not_after − not_before)`, and `½ ×` that where the validity period is 10 days or less.

This is a **re-pricing on a retrieval, not a re-reading**, which is what
[#37](https://github.com/winniel123/verge-asm/issues/37)'s precedent requires. Nothing in ADR-0028
is reinterpreted. Against the lifetimes it was priced against, ADR-0028 was correct, and it is
recorded here as correct.

## Decision

| Concern | Decision |
| --- | --- |
| Does the clock class need an evaluability guard that reads the observed certificate's own validity period? | **Yes.** A `certificate` observation feeds the clock class only while its **age is within the certificate's own horizon**; otherwise the three rules are `not-evaluable` |
| Is the fraction one number, or the rule's own horizon? | **The rule's own horizon.** The bound is the same `N` — `⅓ × (not_after − not_before)`, `½ ×` below a 10-day validity — that `certificate-expiring` already computes. **No new fraction is minted** |
| The declared parameter set | **Unchanged at `{⅓, ½, 10 days}`** ([#67](https://github.com/winniel123/verge-asm/issues/67)). This ADR adds no declared parameter and no constant |
| Does it replace `k`, or sit beside it? | **Beside it.** The effective bound is `min(k × cadence, N)`. `k` is untouched, ADR-0007 is unamended, and ADR-0038 §4 is not reopened |
| Does it fire per rule or per facet? | **Per rule, and only over the clock class.** `certificate-self-signed`, `certificate-hostname-san-mismatch` and `certificate-weak-key-or-signature` are untouched |
| Does it open a `Gap`? | **No.** The value is present and current; one class of rule declines to read it. `Gap` remains ADR-0007's, opened at `k × cadence` alone |
| Which register does it land in? | **`not-evaluable`**, on [#44](https://github.com/winniel123/verge-asm/issues/44)'s fourth cause — *we measured; this rule cannot read the answer*. **Not** *we stopped looking*, so #44's fault treatment does not fire: nothing failed |
| A non-positive validity period (`not_after ≤ not_before`) | The guard **fails**, and the class is `not-evaluable`. A malformed validity period is a different defect and the clock class is not its carrier |
| Is it a dial? | **No.** [#60](https://github.com/winniel123/verge-asm/issues/60)'s three grounds are re-checked below and answered exactly as [`acme-renewal-timing.md`](../research/acme-renewal-timing.md) §12 answered them |
| Is the honest answer partly a tiering change? | **No — and the tiering question has moved out from under the ticket.** [#78](https://github.com/winniel123/verge-asm/issues/78) retired the weekly top-1000 tier on 2026-08-14. What remains of the coverage question is [#80](https://github.com/winniel123/verge-asm/issues/80)'s |
| What it costs | **One `Break` on three rules for one cadence, vacuous before the first install.** No new measurement, no facet change, no ADR-0011 cost, no new declared parameter. One field the corpus row must carry that it did not: the **observation instant** |
| ADR-0028 | **Amended at the two sentences, not withdrawn.** Its decision, its cadence ruling and its fourth `Scan` all stand |

## Rationale

### The bound is in the wrong units, and that is the whole defect

ADR-0028's guard is a duration measured in **how often we look**. The hazard it is asked to bound
is measured in **how fast the subject changes**. Those are independent quantities, and the guard is
safe only where the first is small against the second — which is a comparison ADR-0028 made once,
against 90 days, and which nothing in the repository re-makes when the second quantity moves.

The failure is not that the rules go quiet too late. It is that **inside** the bound the observation
is *current*, so the rules fire on it. `certificate-expired` fires **true** — on a current
observation, about an endpoint serving a perfectly valid certificate. It does not go quiet. It goes
loudly wrong, onto the census [#53](https://github.com/winniel123/verge-asm/issues/53) made the
operator's denominator, in the class [#60](https://github.com/winniel123/verge-asm/issues/60) ruled
is `certificate-expiring`'s **only** carrier.

The structural form of the defect is worth stating, because it shows the hazard is not incidental to
short lifetimes. `certificate-expired` fires exactly when `now > not_after`, which given
`now = t_obs + age` is exactly when

> `age > not_after − t_obs` — the certificate's **remaining life at the moment we observed it**.

So `certificate-expired` can *only ever* fire on an observation that has outlived the remaining life
of the certificate it observed. On any endpoint whose automation works, that is the interval in
which a replacement was due. The rule is a lie generator by construction, and the only thing
standing between it and a false assertion is how large `age` is allowed to grow. ADR-0028 let it
grow to a number chosen with no reference to the certificate at all.

### The horizon is the right bound because it is the same quantity, not because it is symmetric

The ticket offers two shapes: a fresh fraction of validity (*age < ½ × validity*), or the comparison
*age < `certificate-expiring`'s own horizon*. The second wins, and the attraction the ticket names —
that the rule reading the horizon comes to bound its own evidence — is the *consequence* rather than
the argument. The argument is that they are **one quantity doing one job**.

`N` is the issuer's attested replacement window: the interval before expiry within which, on the
issuer's own published rule and its own reference server's arithmetic, the certificate is expected
to be replaced. Read forwards it says *replace it by now*. Read backwards over an observation of
age `N` it says exactly the same thing about that observation: **a replacement was due inside the
interval this evidence spans.** There is no second measurement and no second claim — it is #67's
attestation used once, in both directions.

Three properties follow, and each is checkable rather than asserted.

**It mints nothing.** The declared parameter set stays `{⅓, ½, 10 days}`. That matters more than it
looks: [ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md)'s gate applies to
project-authored content in the comparison path, and
[`acme-renewal-timing.md`](../research/acme-renewal-timing.md) §14 already discloses that ⅓ is **one
CA's number**, extended beyond its attestation when applied to a commercial, internal or self-signed
certificate. A fresh ½ for the evidence bound would be a **second** project-authored number in the
same comparison path with **no** owner behind it at all — widening the disclosed weakness one day
after #67 went to the trouble of removing the last unattested number from this rule. ADR-0038 §11's
rule is *ship the fraction*; it is not a licence to ship a new one.

**It nests the two rules correctly.** Under the guard, `certificate-expired` can fire only where
`remaining-at-observation < age < N` — that is, **only on an observation that already showed the
certificate inside `certificate-expiring`'s horizon.** We assert *expired* only about certificates
we last saw already in their replacement window, with at most one replacement window's worth of
staleness behind the claim. The two rules stop being independent readings of one value and become
nested, which is what one would want of a pair whose whole subject matter is one certificate's
approach to one date. A flat ½ does not produce this; it is looser than the horizon above a 10-day
validity and identical to it below, so it buys nothing at the short end and gives away the nesting
at the long end.

**It introduces no degeneracy.** With `age < N` and the rule firing at `remaining-now < N`, the rule
can still fire (`remaining-at-observation` anywhere below `N + age`) and can still be false
(`remaining-at-observation − age ≥ N`). The guard does not silently convert `certificate-expiring`
into a predicate that cannot partition — which is precisely the failure §7.3 caught in `N = 30`, and
it would have been an ugly way to reintroduce it.

### Beside `k`, not instead of it — and both gates bind on real populations

Replacing `k` for this facet would make `k` facet-dependent, which is a large change to ADR-0007 for
one facet and would reopen what ADR-0038 §4 records as *the staleness rule already satisfied*. It is
also wrong on the objects. The two gates answer different questions:

- `k × cadence` asks **are we still looking?** It is coverage. It is what opens a `Gap`, which is a
  fact about the timeline, and ADR-0008 records that `k` "decides where a `Gap` opens, which decides
  where spans close". A guard that replaced it would have to take over `Gap` opening, and a `Gap`
  opened because a certificate is short-lived would be false: we looked, we got an answer, we hold
  it.
- The horizon asks **may this rule read what we hold?** It is safety, per rule, and it opens
  nothing.

ADR-0007 already rejected *staleness as a fixed wall-clock duration* — "wrong for a weekly tier and
wrong for a daily one; cadence is already Declared and already correct". That refusal is untouched
and is a reason for the shape taken here: this is not a duration, it is not fixed, and it is not the
currency rule. It is a second gate above it.

The decisive check is that **neither gate is redundant** — each is the binding one on a real
population. Effective bound is `min(k × cadence, N)`; the tiers after
[#78](https://github.com/winniel123/verge-asm/issues/78) are two, hot (`verge-core`, daily, `k ×
cadence` = **2 days**) and cold (full-range, opt-in, monthly ceiling, `k × cadence` = **60 days**):

| Validity period | `N` | Hot tier — bound is | Cold tier — bound is |
| --- | --- | --- | --- |
| 160 hours (6.67 d) — GA 2026-01-15 | 80 h (3.33 d) | `k` — 2 d | **horizon — 3.33 d** |
| 45 days — `tlsserver` since 2026-05-13 | 15 d | `k` — 2 d | **horizon — 15 d** |
| 64 days — `classic` from 2027-02-10 | 21.33 d | `k` — 2 d | **horizon — 21.33 d** |
| 90 days — `classic` today | 30 d | `k` — 2 d | **horizon — 30 d** |
| 200 days — BR ceiling since 2026-03-15 | 66.67 d | `k` — 2 d | `k` — 60 d |
| 398 days — legacy commercial | 132.67 d | `k` — 2 d | `k` — 60 d |

Two results fall out of the table and both are load-bearing.

**The hot tier is untouched, for every profile that exists or is scheduled.** The horizon binds there
only where `N < 2 days`, which needs a validity period under **four days**. The shortest certificate
generally available is 160 hours, whose horizon is 80 hours — forty times the gap it would have to
close. So the modal estate sees no change whatever, ADR-0028's *two days modally* survives intact,
and the `Break` this ADR costs is paid on a population that is currently zero.

**On the cold tier the horizon binds below a 180-day validity period** — and the CA/Browser Forum
ceiling drops to **100 days on 2027-03-15**. From that date, *every publicly-trusted certificate in
existence* is inside this guard on the cold tier, by a schedule its own owner has published. The
defect does not need six-day certificates to arrive; six-day certificates are only where it arrives
first and worst.

### Only the clock class, checked rather than assumed

ADR-0028's own reasoning is that the clock class is special because it "is the only place in v1 where
a rule reads an **always-current wall clock** against a **possibly stale observed value**. Everywhere
else both sides age together, so staleness cannot make a comparison lie." The ticket is right that
this should be checked against the other `certificate` rules rather than inherited.

It holds. `certificate-self-signed` compares issuer against subject, both inside one observed chain.
`certificate-hostname-san-mismatch` compares the SAN set against the name the handshake was made to,
and ADR-0028 established that the handshake is a step in the `reachability` exchange — so both sides
are decided inside one `Batch` and carry one instant. `certificate-weak-key-or-signature`
([ADR-0035](./0035-a-cryptographic-primitives-owner-is-its-specifier.md)) ranges over the chain
against a curated table that moves at release cadence, not at wall-clock. None of the three has a
clock on either side. Their staleness is **ordinary** staleness, of exactly the kind every facet in
v1 carries and `k` already prices, and applying a second and tighter gate to them would be pricing a
hazard nobody has shown to exist.

The cost of getting this wrong in the permissive direction is one-sided and large. A facet-level
guard would take `certificate-self-signed` `not-evaluable` on a three-day-old observation of a
six-day certificate, deleting a true and still-actionable finding to buy nothing — and it would do so
on precisely the growing population. Meanwhile a per-rule guard is no novelty: ADR-0024 already makes
the `not-evaluable` case one of a rule's **four parts**, cut per rule and not per facet, and #64
already ruled the message cut "per predicate, not per facet".

### `not-evaluable`, and why it does not destroy the word

[#53](https://github.com/winniel123/verge-asm/issues/53) left a standing warning that this ADR must
clear: "**`not-evaluable` is a coverage word, and a rule whose inapplicable population lands there
destroys it.**" Routing `NoTLS` there was rejected because it filled the band permanently with rows
where nothing failed and no action existed.

This is not that failure, on three counts. The population here is **inside** the domain — the
`certificate` is `Presented` and the question genuinely arises, where `NoTLS` was a population of
which the question could not be asked. The cause is **real and specific** — we measured, and this
rule cannot read what we measured — which is #44's fourth cause verbatim, already named and already
rendered. And it is **not permanent**: on the cold tier the class alternates, evaluable for `N` of
every cadence and silent for the rest, rather than sitting in the band forever.

It follows that #44's **fault treatment does not fire**: that treatment is reserved for *we stopped
looking*, and only where something failed. Nothing failed here. The scan ran, the batch completed,
the value is current. This is the one place in v1 where a `not-evaluable` is produced by the
certificate being short-lived rather than by anything about our own machinery, and the copy must not
read as an outage.

What this ADR does **not** decide is what `certificate-expiring`'s census then says
([#72](https://github.com/winniel123/verge-asm/issues/72)'s), where the `not-evaluable` renders
([#44](https://github.com/winniel123/verge-asm/issues/44)'s), or whether a member that becomes most
of a population is drillable ([#74](https://github.com/winniel123/verge-asm/issues/74)'s, open). It
moves an input to all three and is handed to them as such.

### Not a dial, re-checked

[#60](https://github.com/winniel123/verge-asm/issues/60) killed an operator-configurable `N` on three
grounds, all about **per-install** variation. The horizon reopens none of them, and the answers are
[`acme-renewal-timing.md`](../research/acme-renewal-timing.md) §12's unchanged, because it is the
same parameter: it is project-authored and fixed at the release; nothing is handed outward, since the
guard narrows and widens with the certificate's own validity period, which is a measurement and not a
preference; and CI gates exactly the function every install runs, over a declared parameter set that
has not grown. #22's configurable/fixed line is inside-versus-outside the comparison path, and this
stays inside and stays ours.

### What ADR-0028 got right, and the one word that was wrong

ADR-0028's cost sentence has two halves and they fare differently.

*Silence* was priced correctly. "On that population the three clock rules are silent far more often
than they speak" is true, was true, and this ADR makes it more true. Silence is honest and ADR-0028
was right to accept it.

*Speaking* was priced wrongly, and the wrong word is **"fixable"**. "For up to two months they can
speak about a certificate no longer served — that is honest rather than fixable" rested on there
being exactly one lever, the probe schedule: you cannot handshake a port you do not yet know is open,
so narrowing the window means probing the full range more often, which is #4's declined cost. That
reasoning about the schedule is still correct. But there was a second lever, and ADR-0028 did not see
it because it could not: **stop reading the value earlier, in the certificate's units instead of
ours.** It costs no probes at all. It was not available on 2026-08-13 in any usable form, because the
fraction that expresses it was retrieved on 2026-08-14.

And *honest* does not survive contact with the arithmetic. A 60-day-old observation of a 160-hour
certificate is **nine generations** stale; the observation is not merely possibly superseded, it is
certainly superseded, since one certificate cannot be served for nine times its own validity. A rule
asserting a fact from evidence that is certainly superseded is not being honest-but-stale. It is
wrong, and it does not know it.

### The ticket's tier framing is stale, and the ruling survives it

This must be recorded rather than quietly corrected, because it is the ticket's own headline
argument. #77 argues that "**the population is not exotic** — the *weekly* tier, not the opt-in
monthly one, is where this bites", and prices the failure at 14 days against a six-day certificate,
2.3 lifetimes.

**There is no weekly tier.** [#78](https://github.com/winniel123/verge-asm/issues/78) retired the
weekly top-1000 tier on 2026-08-14 — the same day, on licence grounds, from a different direction
entirely — leaving two tiers, hot and cold. The ~900 ports it covered fell to the cold tier's 30-day
latency, and the cold tier is opt-in. So the population this ADR is about is **narrower** than the
ticket claims: services outside `verge-core`, on installs that opted into the full-range sweep.

Two figures in the ticket's table are also slightly out, and the corrected ones are used above: 160
hours is **6.67** days, not six, so a 14-day bound was **2.1** lifetimes rather than 2.3, a 2-day
bound is **0.30** of a life rather than 0.32, and a 60-day bound is **9** generations rather than 10.

None of it changes the answer, and the reason is the point. The urgency argument was population size;
the ruling's argument is that a rule returns an assertion that is false. A narrow population is not a
defence for a wrong answer, and it is a poor one here in any case, because the CA/Browser Forum's
100-day ceiling on **2027-03-15** puts the entire publicly-trusted WebPKI inside the guard on the cold
tier, and Let's Encrypt's 64-day `classic` profile on **2027-02-10** puts the modal ACME certificate
there a month earlier. Those are published dates, not forecasts, and they arrive with nothing in this
repository changing and no document it cites being retracted — which is ADR-0038 §11's silent
staleness, one level up, in the argument rather than in a number, exactly as #71 predicted.

### Where this was decided on thin ground

**The coupling between the horizon and the evidence bound is an argument, not a retrieval.** Let's
Encrypt attests ⅓ as a *renewal* trigger. Reading the same interval as the span over which an
observation may have been superseded is this ADR's inference. It is a good one — the two are the same
world quantity and the same claim read in two directions — but if a future ticket moves `{⅓, ½, 10
days}` for a reason about renewal, the evidence bound moves with it automatically, and whether that
is still right will need re-checking rather than inheriting. The dependency is stated rather than
buried, per ADR-0034's own practice.

**Every weakness `acme-renewal-timing.md` §14 discloses about `N` is inherited whole.** ⅓ is one CA's
number; applying it to a commercial, internal or self-signed certificate is an extension beyond the
attestation; and the 10-day halving threshold has diverged from the CA/Browser Forum's short-lived
definition, which moved to ≤7 days on 2026-03-15. This ADR adds no weakness of its own, and cures
none.

**The affected population is arithmetic, not measurement.** As #71 disclosed, "§7's arithmetic is
certain; its population is not measured and cannot be before an install." How many services sit
outside `verge-core` on installs that opt into the full-range sweep, presenting short-lived
certificates, is unknown and unknowable today. The ruling rests on the assertion being false where it
occurs, and on the published lifetime schedule, rather than on how often it occurs.

## Consequences

- **Three rules change their predicate, so three rules `Break`.** `certificate-expired`,
  `certificate-not-yet-valid` and `certificate-expiring` each gain the guard in their
  `not-evaluable` case, which is one of ADR-0024's four parts, so each version vector moves.
  ADR-0008's price applies: **one `Break` on three rules for one cadence**, breaks are per timeline,
  the census still renders across one, and nothing has shipped — so the cost today is **zero** and
  after v1 it is a comparability cycle.
- **No new measurement and no facet change.** [ADR-0011](./0011-a-facet-is-six-parts.md)'s six parts
  are untouched and `certificate`'s value space is unchanged. The rules already read `not_before`
  and `not_after` ([#67](https://github.com/winniel123/verge-asm/issues/67)), which is what made this
  repair available at all: ADR-0038's **reach** limb is satisfied because the moving quantity is on
  the subject.
- **The corpus row gains one field.** #60 established that a clock-reading rule's corpus row carries
  its **evaluation instant** as an input, and #67 widened that to carry `not_before`. The guard needs
  one more: the **observation instant**. It is a measurement and already sits on the observation, so
  nothing new is measured — but the row must carry it, or the row's outcome moves with nothing
  declared having moved and ADR-0021's bidirectional gate passes by accident.
- **`k` does not move, and `Gap` does not move.** ADR-0007 is unamended. ADR-0038 §4's record of `k`
  as the staleness rule already satisfied stands. A `certificate` observation that fails the horizon
  is **still current** and still holds an open `Span`; only three rules decline to read it.
- **[ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md) is amended in two
  sentences and stands in everything else.** Its cadence ruling, its fourth `Scan`, its one-candidate-set
  rule and its per-`Service` currency are all untouched. What is withdrawn is *the guard is structural
  and already present* as a **sufficient** guard, and the word *fixable*.
- **The modal estate sees nothing.** On the hot tier the horizon binds only below a four-day validity
  period, and none exists. ADR-0028's *two days modally* is exactly as true after this ADR as before,
  which is why the `Break` is affordable.
- **An input moves for three open questions and is decided by none of them here.**
  [#72](https://github.com/winniel123/verge-asm/issues/72) owns what the census says,
  [#44](https://github.com/winniel123/verge-asm/issues/44) owns where a `not-evaluable` renders, and
  [#74](https://github.com/winniel123/verge-asm/issues/74) — open — owns whether a member that is most
  of a population is drillable. On the cold tier this ruling can make `not-evaluable` the largest of
  the clock class's three census members, which is #74's question arriving with a concrete instance.
- **An input moves for [#80](https://github.com/winniel123/verge-asm/issues/80).** #78 left the ~900
  tail ports covered by the cold tier, which is opt-in, so on a default install they are reached only
  by the onboarding baseline sweep. A single observation ages past its horizon within `N` and past
  `k × cadence` after that, so on a default install the clock class over those ports is effectively
  never evaluable. Whether that is acceptable is #80's question, not this one's.
- **[`CONTEXT.md`](../../CONTEXT.md) changes in two places.** `Certificate` gains the guard beside its
  currency clause and the note that failing it is not a `Gap`; `Signal`'s `not-evaluable` sentence
  widens from *evidence absent* to evidence absent **or held and unreadable by this rule**.
- **Nothing here is a dial, a tier change, or a fold of the clock class into the drift class.** #60's
  three grounds are re-checked and unmoved; the tiering question belongs to #78 (settled) and #80
  (open); and the clock class remains `certificate-expiring`'s only carrier.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Do nothing — the residue is acceptable, as ADR-0028 priced it** | The residue ADR-0028 priced was *silence*, and silence is honest. What it also licensed was *speaking*, and a 60-day-old observation of a 160-hour certificate is nine generations stale — the value is not possibly superseded, it is certainly superseded. A rule that asserts from certainly-superseded evidence is wrong, not stale. And the word "fixable" was false: a second lever existed and cost no probes |
| **A fresh fraction — *age < ½ × validity*** | Mints a second project-authored number in the comparison path with no owner behind it, one day after #67 removed the last unattested number from this rule, and widens the weakness §14 already discloses. Below a 10-day validity it is *identical* to the horizon, so it buys nothing where the problem is worst; above it, it is looser and gives away the nesting of `certificate-expired` inside `certificate-expiring`. Its one genuine advantage — more headroom before the cold tier's gate flips from `k` to the horizon — is the residue itself, bought by tolerating a staler observation. Loosening a guard until the schedule passes is fitting the evidence standard to the budget, which is the move ADR-0028 itself refused when it declined to redefine the weekly tier so the enumeration could ride it |
| **`age < remaining life at observation`** — the strictest sound bound | Degenerate. `certificate-expired` fires *exactly* when `age > remaining-at-observation`, so this guard makes it unfireable by construction. It is the obvious strongest option and it deletes a rule |
| **Replace `k` for `certificate`** | Makes `k` facet-dependent — a large change to ADR-0007 for one facet, and a reopening of what ADR-0038 §4 records as already satisfied. Wrong on the objects besides: `k` decides where a `Gap` opens, and a `Gap` opened because a certificate is short-lived would assert we stopped looking when we did not |
| **Apply the guard per facet, to every `certificate` rule** | The other three rules compare observed values against each other inside one `Batch`, so staleness moves both sides and cannot make them lie — ADR-0028's own reason for the clock class being special, checked here rather than inherited. Per-facet would take `certificate-self-signed` `not-evaluable` on a three-day-old observation of a six-day certificate, deleting a true finding to price a hazard nobody has shown |
| **Fix it by tiering — probe the affected population more often** | ADR-0028's refusal stands on its own reasoning, which lifetimes do not touch: you cannot handshake a port you do not know is open, so it means probing the full range more often, which is #4's declined cost. It is also no longer this ticket's to take — #78 retired the weekly tier on 2026-08-14 and #80 owns what remains. And it would redraw the scan schedule, which is a scope change rather than a preference |
| **Route the failed guard to a `Gap`** | The value is present and current and its `Span` is open. A `Gap` is the absence of a value, and ADR-0007's cause taxonomy would have to say *we stopped looking* about a batch that ran and answered |
| **Route it outside the `Predicate domain` instead of to `not-evaluable`** | The domain is the extension of the rule's name (ADR-0024), and *this certificate could be expired* is squarely assertable of the subject. Excluding a subject the rule might have fired on is the model-layer damping `Drift` refuses, whatever it is called — and outside-the-domain renders nothing at all, hiding the population instead of counting it |
