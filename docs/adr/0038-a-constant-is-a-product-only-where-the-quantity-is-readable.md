# ADR-0038: A constant is a product only where the quantity is readable — and the watch is defined by shape, not by cause

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#71 Which project-authored constants are products of a moving world quantity rather than the quantity itself?](https://github.com/winniel123/verge-asm/issues/71)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Findings:** [`docs/research/project-authored-constants.md`](../research/project-authored-constants.md)

## Context

[ADR-0034](./0034-derive-the-claim-before-looking-for-the-owner.md) §4 established a rule and found
its first instance by accident:

> **Where a project-authored constant is the product of a fraction and a moving world quantity, ship
> the fraction.** The product is a measurement of the world at a date, and it goes stale with
> nothing changing in the repository and no document anywhere being retracted.

It named the failure **silent staleness** and priced it against
[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §8's silent
de-attestation: **worse**, because de-attestation needs a maintainer to flip a default while this
needs nobody to do anything; **better**, because it is preventable by construction.

`certificate-expiring`'s `N = 30` was found because a retrieval happened to touch it. #71 is the
sweep nobody had run, and it carried an explicit instruction not to manufacture instances: *if the
sweep finds one product and six genuine constants, that is the answer, because it bounds the rule's
reach.*

**The sweep found one product, already fixed, and the bound is the result.** What made the sweep
decidable was that ADR-0034 states the rule without stating its **scope**, and the scope turns out
to be the whole of the question.

## Decision

| Concern | Decision |
|---|---|
| When is a constant a **product** in ADR-0034's sense | **Four limbs, all necessary** — form, motion, **reach**, silence. §1 |
| The limb ADR-0034 left implicit, and the one that decides every case | **Reach.** The moving quantity must be readable **from the subject, at evaluation time**, by machinery the rule already has |
| Where reach fails | The rule is **inapplicable, not violated**. Re-expressing as a fraction does not remove the staleness, it **relocates** it into a shipped value for the quantity |
| What to do where reach fails | **A scheduled edit.** [`weak-key-and-signature.md`](../research/weak-key-and-signature.md) §7.3 is promoted from a local note to the general counter-rule |
| The cure, stated generally | **Ship the rule that generated the number.** A fraction is the case where the rule's input is a magnitude the subject carries; §5 names two other shapes |
| How many instances exist in the repository | **One** — `certificate-expiring`'s `N`, fixed by [#67](https://github.com/winniel123/verge-asm/issues/67) before the sweep began |
| `verge-core`'s frequency half | **A product, already stale by eighteen years, measured — and not curable.** Fails reach; its residual cost is **aperture only**; stays, watched |
| `certificate-weak-key-or-signature`'s thresholds | **A product** — NIST writes the derivation into the normative sentence — **and it must not be re-expressed**, on two grounds [#68](https://github.com/winniel123/verge-asm/issues/68) had already written |
| The prober's timeout and retry budget | **Not a product.** And the re-expression was **already forbidden** by [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md): a deadline that reads a measured quantity makes a value depend on how busy the run was |
| `k` | **Not a product, and the rule already satisfied** — currency is `k` **cadences**, never a duration. The sweep's positive control |
| The availability window, and #61's staleness bound | **Not products.** `k × cadence` is a product of two project-authored quantities and cannot go stale while the repository does not change |
| Rate and concurrency figures, including #4 §6's 5 handshakes/s | **Not products, and not declared parameters** — ADR-0021 puts them outside every leaf, so they can never move a value. Anti-fragile besides |
| The coverage threshold | **Not in the population.** It is an operator's dial, which is not project-authored |
| Does the class need a home | **No.** ADR-0032 §8's watch is redefined **by shape rather than by cause**, and it already covers this. **No fourth pile** |
| What ADR-0034's *"third state"* is | **Not a pile — the absence of one.** A constant passing reach leaves the accounting; one failing it stays on the watch it was always on |
| ADR-0034's *"nothing to watch"* | **Scoped.** A fraction removes the **quantity** from the watch, never the **attestation** |
| The sweep's most damaging finding | **Not an instance of this rule.** ADR-0028's currency bound is a **stale argument**, not a stale constant. §6, and [#77](https://github.com/winniel123/verge-asm/issues/77) |

## Rationale

### 1. Four limbs, and reach is the one that decides

> A project-authored constant `C` is a **product** in ADR-0034's sense exactly where:
>
> 1. **Form.** `C = f(Q)` for a project-authored rule `f` and a quantity `Q` the project did not choose.
> 2. **Motion.** `Q` moves, and moves with no document the constant cites being retracted.
> 3. **Reach.** `Q` is readable **from the subject, at evaluation time**, by machinery the rule already has.
> 4. **Silence.** When `Q` moves, the constant's wrongness produces no signal — no failing build, no
>    `not-evaluable`, no version move, no visible symptom.

**Limbs 1 and 2 make a constant stale-able. Limb 3 makes it curable by construction. Limb 4 makes
it worth curing.**

Reach is already in ADR-0034, unstated: *"a row expressing the fraction reads the moving quantity
from the subject at evaluation time, so it cannot go stale."* The **cannot** is the entire cure, and
it is available only where the subject carries `Q`. A certificate carries `not_before` and
`not_after`. It does not carry NIST's strength table, the internet's port-frequency distribution, or
a calendar.

**Where reach fails, the fraction is not a lesser cure — it is not a cure.** The shipped artefact
must then carry a value for `Q`, and that value is the same stale thing wearing a formula. This is
why the rule cannot simply be applied harder: it has an edge, and §2 and §3 are what is on the other
side of it.

Silence is ADR-0034's own price argument read as a test. It ranks silent staleness worst *because*
nobody has to do anything; where something does happen, the constant is stale-able but not silently
so, and a defence already exists.

### 2. `verge-core`'s frequency half — the clearest product, and no fraction exists

`f` = *the top hundred by open-frequency, minus the ephemeral tail, plus a modern-services
supplement*. `Q` = the distribution of open ports across the internet.

Form, motion and silence all pass, and **the staleness is measured rather than argued — re-measured
here, not inherited.** The shipped file's header is byte-identical to what #4 read in
[`safe-active-probing.md`](../research/safe-active-probing.md) §2.2 —
`$Id: nmap-services 9746 2008-08-26 18:45:24Z fyodor $` — and the changelog corroborates that the
marker is not merely stale: the frequency data appears **once**, under `Nmap 4.75 [2008-9-7]`
(*"generated by scanning tens of millions of IPs on the Internet this summer"*), with **no second
entry across the entire 7.x series** through `7.991 [2026-08-06]`. Nmap issue **#2399**, asking this
exact question, has been open and unanswered since 2021-11-20. **Eighteen years, confirmed.**

**And the retrieval corrected #4's numbers in the direction that strengthens the finding.** Redis
and Docker are not *ranked low* — they carry `0.000076`, a filler plateau shared by **1,969 TCP
lines**, blame-dated to **2016-09-14** and the commit *"Merge latest IANA services"*. They entered
from IANA's **name** registry eight years after the only scan and **nothing was ever measured for
them**; the quoted ranks are tie-break artefacts of a block spanning 1442–3410. Kubelet is
confirmed genuinely absent.

> **The file does not under-weight modern exposure. It holds no information about it at all**, and
> #4 §2.3's signal-mapped supplement is the only reason those ports are in `verge-core`.

**Reach fails and there is nothing to ship instead.** No `Address` and no `Service` carries the
internet's port-frequency distribution. A fraction here would mean shipping a *table of
frequencies* rather than a *set of ports* — the same staleness, one indirection out, plus a table.

**Two things make this the right place to stop rather than a defeat.**

**The stale half is exactly the half not expressed as a rule.** §2.3's supplement is chosen because
each port *maps to a named v1 risk signal* — a project-internal criterion that does not move when
the world does — and [ADR-0009](./0009-verge-core-is-a-union.md) already made the other half a rule:
`verge-core = frequency-set ∪ sensitive-list`, membership by claim and attestation rather than by
frequency. ADR-0009 was built to make an invariant unfalsifiable; it also, unremarked, converted
half of a stale product into a rule-expressed set. That is §5's generalisation, found rather than
invented.

**The residual cost is aperture, so a stale frequency set makes us not look and can never make us
conclude wrongly.** ADR-0032 §5 already places it there, governed by
[#31](https://github.com/winniel123/verge-asm/issues/31)'s line, and #44's standing aperture
statement on `Coverage` is where the honesty already lands. It is not deleted because the job is
real: the frequency half catches a listener no signal names, which the rule-expressed half cannot do
by construction.

### 3. The weak-key table — a product whose re-expression two rulings already forbid

`f` = *below 112 bits of security strength*. `Q` = NIST's strength-to-key-size mapping and its dated
transitions.

**Form passes on the owner's own grammar, retrieved.** SP 800-131A Rev 2 (FINAL, March 2019) does
not merely tabulate 2048 against 112 — it writes the derivation into the normative sentence, with
*to meet* as the connective:

> *"**RSA:** The length of the modulus n **shall be 2048 bits or more to meet the minimum
> security-strength requirement of 112 bits** for Federal Government use."*

> *"**ECDSA and EdDSA:** The security strength provided by an elliptic-curve-based signature
> algorithm is no greater than 1/2 of the length of the domain parameter n. **Therefore, the length
> of n shall be at least 224 bits to meet the minimum security-strength requirement of 112 bits.**"*

The requirement is stated as a **strength** and never as a size — §1.2.1: *"a security strength of
at least 112 bits is required at this time"* — with a footnote conceding the vocabulary is not its
own: *"The term 'key size' is commonly used in other documents."* And `224` is `112 × 2` with the
multiplication written out, so the table ships a flattened derivation twice over. All three of our
key rows are **one row of SP 800-57 Part 1 Rev 5 Table 2**, transcribed.

**Reach fails**: no certificate carries NIST's mapping. **And two rulings already on the books
forbid reaching for it anyway**, both written by [#68](https://github.com/winniel123/verge-asm/issues/68)
without knowing it was answering #71 — §2.4, that the key may not be a bare bit count or gate 3
comes inside the domain and fails; and §7.3, that encoding a dated transition in the predicate moves
the rule's output at midnight with no version bump, which is release-coupling defeated from the
inside.

**And the retrieval found the reason the fraction would fail even if reach were satisfied.** The
scheduled move is not `2048 → 3072`. SP 800-57 Part 1 Rev 5 Table 4 (FINAL) puts 112-bit strength
**Disallowed for applying protection from 2031**, while Rev 4 → Rev 5 left the 112 row's *sizes*
numerically untouched. So what moves is the **tier's status**, not the mapping:

> **A table shipping `≥ 112 bits` would be exactly as stale as one shipping `≥ 2048`, because the
> quantity that moves is whether 112 is enough.** There is no fixed point to ship. `f` is NIST's
> floor, and NIST moves the floor.

That is the strongest available argument that a scheduled edit is the right answer here, and it is
measured rather than reasoned.

**Silence fails too, and that is the general point.** The transition is published, dated and already
diarised — the map carries *re-read SP 800-131A when a revision goes final and otherwise do
nothing*, against #68's measured cadence of about one edit per row per decade.

> **Where `Q` moves on a schedule its owner publishes but the rule cannot read `Q`, the answer is a
> scheduled edit, and ADR-0034 §4 is inapplicable rather than violated.**

The two rules do not compete. **They partition on reach.**

**Disclosed, per ADR-0032 as amended by ADR-0034 §7.** The dates are thinner than they look: SP
800-131A **Rev 3** and **NIST IR 8547** — which carry the 2030-deprecate/2035-disallow language —
are both **Initial Public Drafts** and have sat in draft for about twenty-one months
(`csrc.nist.gov/pubs/sp/800/131/a/r3/final` and `/pubs/ir/8547/final` both **404**). The only final
normative date is SP 800-57 Table 4's 2031, which the drafts propose to *soften*. **What remains
unestablished** is which date governs; **the condition that would move it** is either draft going
final. It changes no row today, because #68 refused to encode dates at all.

### 4. The negatives, and why each is stated as a rule rather than a verdict

The sweep's value is mostly in what it declines, so each negative names the reason that stops the
next session reopening it. [`project-authored-constants.md`](../research/project-authored-constants.md)
§6 walks all of them; four carry rulings.

**The connect timeout is not a product, and the re-expression was already forbidden.** #71 asked
whether 3 s is a fraction of round-trip time. It is not — it is a **classification boundary** whose
stated footing is another tool's shipped default, which ADR-0034 §5 already rules a corroborator.
Reach is technically available, since RTT is observable per connect, and **ADR-0021 rejected taking
it in terms**: *"Adaptive back-off inside `connect-outcome` — it halves the rate, never the deadline
— and had it moved the deadline, **a value would depend on how busy the run was**."* A deadline
reading a measured quantity makes `connect-outcome` non-deterministic and fails the golden-corpus
gate ADR-0021 exists to run. **This is #71's *genuinely should not* case, and it was already ruled;
the sweep only had to notice.**

**`k` is the rule already satisfied, and it is the sweep's positive control.**
[ADR-0007](./0007-drift-is-a-timeline-of-spans.md) set currency at **`k` cadences** rather than at a
number of hours, on **2026-08-13** — four days before ADR-0034 named the rule. `k` is the fraction;
`k × cadence` is the product; the cadence is read at evaluation time from `Scan`, which is Declared.
Had ADR-0007 shipped *an observation is current for 48 hours*, it would have been `N = 30`'s twin.
A sweep reporting only failures cannot tell whether its instrument works, so the positive control is
recorded.

**The rate and concurrency figures are not even in the comparison path.** ADR-0021 puts
*"concurrency, rate-limiting and adaptive back-off"* outside every leaf, so #4 §6's 5 handshakes/s
and its siblings **can never move a value** — at worst they move wall-clock and a `Batch`'s
completed scope, which is coverage, recorded and rendered. They are also **anti-fragile**: the
quantity they are budgeted against is a host's capacity to absorb connections, which moves upward
with hardware, so a stale budget becomes *more* conservative. #4 §6.2 already banks it: *"Our budget
is wall-clock hours, and we have plenty."*

**And the near-miss is why motion is not a formality.** The EDNS UDP payload size **1232** is
*literally* a product and says so — [`measurement-offers.md`](../spec/measurement-offers.md) §4:
*"derived from IPv6's required 1280-byte MTU."* Motion fails, decisively: RFC 8200 §5 fixes 1280 and
has since RFC 2460 in 1998. A Standards Track minimum is a **protocol constant, not a measurement**;
nobody publishes a schedule for it and changing it would break IPv6 rather than update a number. It
is independently attested as a *value* besides, by DNS Flag Day 2020 and by BIND, Unbound and Knot.

> **A constant can be a product in form and a constant in fact.** A sweep run on form alone would
> have re-expressed a correct, attested, frozen number as a formula over a quantity that never
> moves. That is the over-fitting #71 forbade, caught in the act, and it is the reason the test has
> four limbs rather than one.

### 5. The cure generalises past the fraction

Three cure shapes appeared, and they share a form.

| Cure | Applies where | Instance |
|---|---|---|
| **A fraction of a quantity on the subject** | `Q` is a magnitude the subject carries | `certificate-expiring`'s `N` (#67) |
| **A rule-expressed membership criterion** | `Q` is a population and membership follows from something project-internal | `verge-core`'s sensitive half — ADR-0009's union; §2.3's signal-mapped supplement |
| **A terminator instead of a budget** | `Q` bounds *how much to read* and the stopping condition is in the bytes | the capped body read |

> **Ship the rule that generated the number, wherever the rule's inputs are available at evaluation
> time. A fraction is the case where the input is a magnitude carried by the subject.**

This is stated as a generalisation of ADR-0034 rather than a correction of it: *ship the fraction*
is right and is the case that had an instance.

### 6. The sweep's most damaging finding is a stale **argument**, and it is not this rule

[ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md) (2026-08-13) named the clock
class as *"the only place in v1 where a rule reads an always-current wall clock against a possibly
stale observed value"*, bounded it with `k`, and priced the residue as acceptable: *"for up to two
months they can speak about a certificate no longer served. **That is honest rather than fixable.**"*

**One day later**, [#67](https://github.com/winniel123/verge-asm/issues/67) retrieved that Let's
Encrypt's short-lived certificates — *"valid for 160 hours, just over six days"* — have been
**generally available since 2026-01-15**, and that the `tlsserver` profile has issued **45-day**
certificates since 2026-05-13.

ADR-0028's guard is structural **only where the window is short against the lifetime**, and it was
checked against 90 days:

| Tier | Bound (`k`=2) | vs a 160-hour certificate | vs a 45-day certificate |
|---|---|---|---|
| daily (`verge-core`) | 2 days | 0.32 of its life | 0.04 of its life |
| **weekly** (top-1000 only) | **14 days** | **2.3 × its entire life** | **0.31 — one whole `certificate-expiring` threshold width** |
| monthly (full-range only) | 60 days | **≈ 10 generations** | 1.3 generations |

**Inside the bound the observation is current and the rules fire on it.** So a top-1000-only
`Service` presenting a six-day certificate has its observed `not_after` pass on day six, and
`certificate-expired` **fires true — on a current observation, about an endpoint serving a valid
certificate — and keeps firing for the remaining eight days.** It does not go quiet. It goes
**loudly wrong**, onto the census [#53](https://github.com/winniel123/verge-asm/issues/53) made the
thing the operator reads, in the class [#60](https://github.com/winniel123/verge-asm/issues/60) ruled
is `certificate-expiring`'s only carrier.

**And it is not an instance of this ADR's rule.** Neither `k` nor any cadence is a product of a
world quantity. What was computed against a world quantity is the **safety argument** that
`k × cadence` is short enough.

> **ADR-0034 §4 catches stale numbers. It does not catch stale arguments, and the argument is where
> the residual risk in this repository actually sits.**

Recording it as an instance would be the over-fitting #71 forbade; not recording it because it fails
the test would be worse. **The repair passes reach** — the clock rules already read `not_before` and
`not_after` — and is one extra evaluability predicate at a `Break` on three rules for one cadence,
vacuous before first install. It is **not ruled here**, because it changes three rules' evaluability
and interacts with [#72](https://github.com/winniel123/verge-asm/issues/72) and #44's
`not-evaluable` rendering. Opened as [#77](https://github.com/winniel123/verge-asm/issues/77).

### 7. The class needs no home, because the watch was misdefined rather than incomplete

#71 asks whether the class needs a home, since #67 found `N` in neither of ADR-0032 §8's two piles
and #68 has since added a third, **scope**.

**It does not, and a fourth pile would be the machinery the fix removes.**

ADR-0032 §8 defines its watch list by its **cause** — a maintainer flips a restricting default —
rather than by its **shape**. Read by shape it is *rows where somebody must notice something for the
row to stay right*, and that already covers a constant computed from a quantity the rule cannot
read: `verge-core`'s frequency half and the weak-key table both need somebody to notice, in exactly
the sense 5432/tcp and 5984/tcp do.

> **A weak row is *watched* wherever something must be noticed for it to stay right, whether what
> moves is an attestation or a quantity the row was computed from.** The names — **watched**,
> **chased**, **scope** — are causes of weakness, not kinds of watch.

And ADR-0034's *"a third state exists and it is the one to aim for"* is now placed:

> **It is not a pile. It is the absence of one.** `N` left both piles because its cure **removed the
> watch**, not because it was a new kind of weakness. A constant passing reach leaves the accounting
> entirely; a constant failing it stays on the watch it was always on.

This is the answer that mints nothing, which ADR-0034 already argued for when it refused to file the
staleness finding as a watch item: *"filing it as a watch would build the machinery the fix
removes."* The fix removes the machinery **where it reaches**; where it does not reach, the existing
machinery was always the right home.

### 8. The bound, stated as the result

**Eighteen project-authored constants and constant-sets were walked. One is a product passing all
four limbs, and it was fixed before the sweep began. Two are products failing reach, and both were
already correctly disposed of by tickets that did not know they were doing it. Fifteen are not
products. One near-miss passes form and fails motion.**

> **The rule's reach in this repository is one instance. That is the finding.** A rule that turned
> out to apply everywhere would have been evidence the test was wrong rather than that the
> repository was rotten — and the one place the world's movement has done real damage (§6) is a
> place the rule does not reach at all.

## Consequences

- **[ADR-0034](./0034-derive-the-claim-before-looking-for-the-owner.md) is amended twice, additively.**
  §4's rule gains the **reach** limb: it applies where the moving quantity is readable from the
  subject at evaluation time, and is **inapplicable** — not violated — where it is not. And its
  *"a row that reads the moving quantity from the subject at evaluation time has nothing to watch"*
  is **scoped**: a fraction removes the **quantity** from the watch, never the **attestation**.
  `⅓`, `½` and the 10-day threshold remain the issuer's published values and the issuer may revise
  any of them, which is an ordinary §8 attestation move. #67 already measured the live hazard — the
  CA/B Forum moved its short-lived definition from ≤10 days to **≤7 days on 2026-03-15** while the
  issuer, `boulder`, Certbot and lego all still use **10 days**.
- **The port-curation patch's discharge of `certificate-expiring`'s horizon stands.** Nobody revises
  a fraction when a lifetime changes. The amendment above concerns a different mover.
- **[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §8's watch list is
  redefined by shape**, gaining `verge-core`'s frequency half and the weak-key table without gaining
  a pile. Its three shipped-default rows — now two after
  [#69](https://github.com/winniel123/verge-asm/issues/69) — are unaffected.
- **[`weak-key-and-signature.md`](../research/weak-key-and-signature.md) §7.3 is promoted to a
  general rule** and **nothing is taken from it**. The file is owned by
  [#73](https://github.com/winniel123/verge-asm/issues/73) this round and is **not edited**; one
  optional addition is offered in #71's resolution comment for whoever holds it.
- **[`sensitive-ports.md`](../research/sensitive-ports.md) is not edited** — owned by
  [#70](https://github.com/winniel123/verge-asm/issues/70) this round — and needs no amendment. §2
  cites `safe-active-probing.md`, not it.
- **[ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md)'s *"honest rather than
  fixable"* is annotated, not rewritten.** It was correct against the lifetimes it was priced
  against and is re-priced on a retrieval performed the following day, which is
  [#37](https://github.com/winniel123/verge-asm/issues/37)'s move rather than a re-reading. The
  question it raises is [#77](https://github.com/winniel123/verge-asm/issues/77)'s and is
  deliberately not pre-empted here.
- **[#12](https://github.com/winniel123/verge-asm/issues/12) carries two shape decisions for
  parameters that have no value yet.** The **availability window** is expressed in **batches, not
  hours**, for `k`'s reason; the **capped body read** is a **terminator, not a byte count**, because
  a byte budget against where `<title>` sits is a world quantity that moves upward and whose failure
  is a `Transition` indistinguishable from a real one. Both cost nothing now and a `Break` later.
- **Nothing costs a `Break`, a `revealed`, an aperture change or a message.** No constant is
  re-expressed, no `Derivation` version moves, no table gains or loses a row. This ADR is an
  instrument and a bound.
- **`CONTEXT.md` is not edited**, deliberately — concurrent sessions are in that file and #37,
  ADR-0032 and ADR-0034 all set the precedent. The edit this ADR would make is one clause on
  `Derivation`: *a declared parameter expresses the rule that generated it wherever that rule's
  inputs are readable at evaluation time, and is a scheduled edit wherever they are not.*
- **[`safe-active-probing.md`](../research/safe-active-probing.md) §2.2's ranks are wrong and its
  conclusion is stronger than it knew.** Retrieval found that Redis and Docker carry `0.000076`, a
  filler value shared by **1,969 TCP lines**, imported from IANA's *name* registry on **2016-09-14**
  — eight years after the only scan. They were never **measured**, so the quoted ranks (1,683 and
  2,903) are tie-break artefacts of a block spanning ranks 1442–3410. The file contributes **no
  information whatever** about modern exposure, rather than under-weighting it. Amendment text is in
  #71's resolution comment; **the file is not edited here.**
- **[#78](https://github.com/winniel123/verge-asm/issues/78) is opened as by-catch and **blocks
  [#12](https://github.com/winniel123/verge-asm/issues/12)**.** Establishing that no redistributable
  replacement for `nmap-services` exists surfaced a prior question: nmap ships under the **NPSL**,
  reported to define a derivative work to include software that *"reads or includes Covered Software
  data files"*, and NPSL is GPLv2-derived against our AGPL-3.0. **No verdict is implied** — the NPSL
  was not retrieved against bytes in this sweep — but the map's standing instruction is to flag
  anything that would force a licence change, and [#27](https://github.com/winniel123/verge-asm/issues/27)
  killed bundling CAIDA on the neighbouring ground. It blocks #12 because a spec cannot carry a port
  tier whose licence to exist is unresolved.
- **The map's Notes carry an illustration without its formula.** #61's staleness bound is recorded
  as *"two days modally"*; it is `k × cadence` and the tiering has already moved once (#61's fourth
  `Scan`). The wording fix is in #71's resolution comment. No `Break` — nothing reads it.

## Alternatives rejected

| Alternative | Why not |
|---|---|
| **Apply ADR-0034 §4 as written, with no reach limb** | It would have re-expressed the EDNS payload size — a correct, attested, frozen number — as a formula over IPv6's minimum MTU, which RFC 8200 has fixed since 1998. And it would have demanded a fraction for the weak-key table and `verge-core`, where the "fraction" must ship a value for the quantity and is therefore the same staleness with extra steps. The rule needs an edge or it manufactures work |
| **Mint a fourth pile for products that cannot be re-expressed** | ADR-0034 refused to file its own finding as a watch because *"filing it as a watch would build the machinery the fix removes."* The same objection applies to a fourth pile: §8's watch already means *somebody must notice*, and these rows need noticing in exactly that sense. The defect was that §8 named its members by cause; naming by shape costs one sentence and no object |
| **Re-express the weak-key table as `≥ 112 bits of security strength`** | Refused on three independent grounds, two of them #68's and already written: §2.4 (a strength key is a surrogate and brings gate 3 inside the domain, where the same integer means incompatible things on a modulus and a curve) and §7.3 (release-coupling defeated from the inside). The third is measured here — SP 800-57 Table 4 disallows the **112-bit tier** from 2031 while its sizes stay numerically identical, so a strength-expressed table is **exactly as stale** as a size-expressed one. There is no fixed point to ship |
| **Make the connect timeout adaptive on measured RTT** | ADR-0021 rejected it in terms and for the decisive reason: a value would depend on how busy the run was, which fails the golden-corpus gate. Not reopened, and the sweep's job was to notice it had been answered |
| **Re-source `verge-core`'s frequency half from live scan data** | Retrieved and closed on measurement: **no free, redistributable ranking exists.** Shodan publishes one openly and grants **no data licence at all**, requiring instead that dependent materials *"clearly indicate Shodan's ownership and copyright"* — unshippable inside AGPL-3.0. Censys bars it in terms: *"Under no circumstances may any Customer … incorporate any Censys Data into its own software products or services that are distributed."* Project Sonar went commercial in 2022 (*"cannot be redistributed in bulk"*), scans.io is non-commercial. This is [#27](https://github.com/winniel123/verge-asm/issues/27)'s CAIDA finding a second time. And the half is **aperture** (ADR-0032 §5), so its staleness costs a missed look and never a false verdict |
| **Drop `verge-core`'s frequency half and keep only the rule-expressed union member** | The frequency half exists to catch a listener **no signal names**, which the rule-expressed half cannot do by construction, since a port joins it only where a signal already names it. Dropping it deletes the product's drift-detection job to fix a defect that costs aperture |
| **Rule [#77](https://github.com/winniel123/verge-asm/issues/77)'s repair here** | It changes three rules' evaluability and interacts with #72 and #44's rendering, and the fraction's size wants arguing both ways. #71 is a sweep; ruling a rule change inside it is the scope creep the map's own note about stopping for a scope change exists to catch |
| **Record §6 as an instance of ADR-0034 §4** | It is a stale **argument**, not a stale constant — no constant in it is a product of a world quantity. #71 forbade manufacturing instances, and the honest statement that the sweep's worst finding lies outside its own rule is worth more than a fourth tally mark |
| **Fold this into ADR-0034 as an amendment** | Two of its rulings are not about ADR-0034's population: the watch-by-shape ruling governs ADR-0032 §8, and §6 governs ADR-0028's safety argument. Wrong population — ADR-0034's own objection to folding into ADR-0032, and ADR-0032's to folding into ADR-0004 |
| **Leave the sweep unrecorded because it found nothing to change** | A bound is a result. Without it the next session re-runs the sweep, and the session after that applies the rule to the EDNS payload size because nobody wrote down why not |
</content>
