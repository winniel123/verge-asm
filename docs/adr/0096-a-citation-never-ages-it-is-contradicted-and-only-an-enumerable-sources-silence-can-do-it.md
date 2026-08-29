# ADR-0096: A `Citation` never ages — it is contradicted, and only an `enumerable` source's silence can contradict one

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#176 Certificate transparency's admissions have no covering `Scan`, so a `Name` a SAN once carried never leaves](https://github.com/winniel123/verge-asm/issues/176)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[#142](https://github.com/winniel123/verge-asm/issues/142) ·
[ADR-0084](./0084-a-scan-is-a-cadence-over-an-exchange-and-an-uncovered-facet-has-no-currency-bound.md)
minted the fifth `Scan`, `dns`, and closed the currency hole for `resolution` and `dns-record`. Its
opening premise named **two** populations — our own resolver's DNS **and the passive sources** — and
it closed only the first, stating rather than absorbing the second:

> A source that admits without observing holds no facet timeline, so it has no *currency bound* in
> the sense this ADR is about. What it has is a **`Citation` that goes stale** … The consequence
> there is **subject withdrawal**, not a `Gap`.

[ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) reached the
same seam from the retention side and stepped over it in one sentence — *"a `Batch` cited by a
source that admits without observing is not an observation and does not age the same way."*
**Nothing since has said how it ages**, and
[`CONTEXT.md`](../../CONTEXT.md)'s `Citation` entry has meanwhile carried a live consequence that
presupposes it does: *"a subject whose last citation goes stale has no chain back to a `Seed`, which
withdraws it **and** closes the probing gate on it."*

So the model contains a withdrawal route with a trigger nobody has defined, over the population
[ADR-0027](./0027-a-source-may-admit-without-observing.md) created when it ruled that certificate
transparency admits `Name`s and observes nothing.

The ticket put the question in the form that matters: **not what cadence covers CT, but does a
citation age at all, and against which `completeness` value.**

## Decision

**A `Citation` does not age. Nothing in the model retires one by a clock, and the question *against
which `completeness` value does it age* is malformed for the same reason ADR-0012's *what is a
proposer's `authority`* was malformed — it asks for the rate of a process that does not run.**

**What `completeness` decides is whether a later `Batch` can *contradict* a citation. Only an
`enumerable` source's silence can, and that is not aging — it is
[ADR-0006](./0006-subjects-leave-by-measurement.md)'s measured absence, arriving at the citation
hop.**

| Concern | Decision |
| --- | --- |
| Does a `Citation` age on a clock | **No.** Not on any hop, not for any source, not at any `completeness` |
| What *"a citation goes stale"* means where the hop is an **observation** | The observation left the currency bound. That is the observation's clock, never the citation's, and it is the only live instance |
| What it means where the hop is a **`Batch`** (a source that admits without observing) | **Nothing. There is no observation, so there is no bound, and there is nothing to define** |
| Can a `corroborative` source's silence retire its own citation | **No.** `corroborative` *is* the value that says silence is not evidence; retiring a citation on silence is concluding absence from it |
| Can an `enumerable` source's silence | **Yes — and it is not aging.** A later `Batch` over a covering recorded scope that omits the subject is a **measured absence**, and the subject leaves by ADR-0006's existing route with no new machinery |
| Does v1 hold any `enumerable` source that admits without observing | **No.** The population is empty; the limb is written for the source that arrives later |
| Is a `Citation` sufficient for membership | **No — necessary only**, except for `Address`, the one subject whose existence nothing observes |
| A sixth `Scan` over the CT poll, as an instrument of withdrawal | **Refused permanently, on capability.** Its every firing would be a defect of our own instrument |
| A `Scan` over a source that admits without observing, in general | **Carries no currency bound and no withdrawal power** — the currency rule quantifies over observations, and there are none |
| Is the CT poll's **cadence** decided here | **No.** It is a separate hole, it is real, and its scope cannot be drawn yet — see §7 and the successor |
| A `Gap` | **None. Not opened, not reachable** — CT holds no timeline for one to sit on ([ADR-0027](./0027-a-source-may-admit-without-observing.md)) |
| Cost | **`Name`s admitted by CT beneath a wildcard or a `Lame` delegation stay in the estate permanently** — which is ADR-0006's already-ruled residue, at a second door |

## Rationale

### 1. The ticket's premise is false for every name outside a population ADR-0006 already owns

The title says *a `Name` a SAN once carried never leaves*. It is worth stating plainly that this is
**not true**, because the whole shape of the answer follows from where it stops being true.

A CT-admitted `Name` is a `Name` like any other, and `CONTEXT.md`'s `Name` entry says how one
departs: *"it leaves when our own resolver measures a Name Error on a cross-class `Vantage
composition` … never because time passed."* ADR-0006 measured the point in terms, against this exact
population:

> Since every known name is re-resolved every cycle, the premise that a certificate-discovered name
> has only `corroborative` sources bearing on it is **false for every name outside a wildcard**.

ADR-0027 restated it as the reason CT needs no timeline at all:

> A CT-admitted name acquires a `resolution` timeline from our own `enumerable` resolver within one
> cadence, and leaves by Name Error like any other. **CT needs no timeline for the model to work,
> and never had one.**

And #142 has since supplied the clock that sentence was written against but did not have: our own
resolver's `resolution` timeline is now covered by the `dns` `Scan`, daily, `k` = 2. So a
CT-admitted name that stops existing is withdrawn within two days, **by measurement**, and its
citation has nothing to do with it.

What survives is the residue where the resolver can never return a Name Error, and it has exactly
two members, both enumerated by ADR-0006 and its [#35](https://github.com/winniel123/verge-asm/issues/35)
amendment: names under a `Shadowed` answer, and names beneath a `Lame` delegation. `CONTEXT.md`
already says so at `Name`, in the present tense.

**So the population the ticket is worried about is not *CT-admitted names*. It is ADR-0006's residue,
reached through a second door.** ADR-0006 ruled it stays — *"the residue stays in the estate, visibly
unconfirmed, and leaves by one of two honest routes: the operator supplies coverage, or the operator
declares it out of scope"*, with the #35 amendment adding a third, repair the delegation.

That is the first half of the answer to the ticket's *immortal by decision rather than by omission*
objection. **These `Name`s are not immortal by this decision.** They are immortal by ADR-0006's
decision, which was made on the merits, and nothing here is asked to re-litigate it. What is asked
is whether to add a **second** withdrawal route beside it. §2 and §3 are why not.

### 2. A citation clock over CT could never fire on the world moving — only on our own instrument

This is the decisive argument and it is stronger than the one the ticket supplies. The ticket's
framing is that a citation clock *"would mass-withdraw an estate on a bad afternoon"* — a noise
argument. The honest statement is worse than that: **the instrument has zero evidential content in
the withdrawal direction.**

Certificate transparency is append-only. A `Name` that appeared in a logged SAN in 2019 is returned
by the same `crt.sh` query in 2026, whether or not it ever resolved, whether or not the certificate
expired, and whether or not the operator decommissioned it. So a citation clock ticking against CT
re-arms on every poll for every name that is really there, and can only ever expire a citation in
these cases:

1. **The source errored.** Already barred, twice: the map's standing rule that *a source that errors
   must produce no observation, never an observation of absence*, and
   [ADR-0005](./0005-scan-execution-model.md)'s *"a dead-lettered batch records an empty scope and
   licenses no absence whatsoever."* **[measured 2026-07-31, quoted from
   [`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §7 and NOT re-run for
   this ticket]** crt.sh returned **4 successes in 8 identical requests**, including two spurious
   **404**s against the same URL that had served a valid 95 KB body seconds earlier. It is a **dated
   record, not a current value**, and see §9 for why nothing here rests on it.
2. **The answer truncated.** ADR-0027 already names the hazard: *"a documented 999-row cap that
   truncates silently under an HTTP 200."* This one is not caught by rule 1, because it **is** a
   well-formed 200 — the only case where a citation clock would fire, silently, on nothing.
3. **The name was only ever carried by a wildcard SAN.** Then it was never admitted in the first
   place ([ADR-0060](./0060-a-wildcard-san-is-a-pattern-over-names-and-admits-none-of-them.md)), so
   there is no citation to expire.

A fourth candidate exists and is unmeasured — whether `crt.sh` retains rows from **retired** logs.
The research note records Oak as retired. It does not weaken the argument, it repeats it: it is
another property of the instrument.

**Every route by which a CT citation clock could ever fire is a defect of our own reading, and none
is a fact about the estate.** A clock whose complete firing set is instrument defects is not a
measurement of anything, and the model already has a name for what it would produce: manufactured
drift. This refusal is therefore on **capability**, not on cost or on noise, and it does not reopen
on a better-behaved CT instrument — append-only is a property of certificate transparency, not of
`crt.sh`.

### 3. `completeness` was always the answer, and it had never been read at this hop

`CONTEXT.md`'s `Completeness` entry is one sentence long where it matters: *"Whether a source's
**silence** may mean absence."* It has always been read against observations, because that is where
its instances were. Read at the citation hop it decides the ticket's question without a new term:

> **Retiring a citation because a source stopped naming the subject is concluding absence from
> silence.** `corroborative` is exactly the value that forbids that conclusion. So a `corroborative`
> source's citation cannot be retired by that source's silence — not slowly, not on a cadence, not
> at all.

`CONTEXT.md` says it of CT specifically, in the `Completeness` entry itself: *"certificate
transparency is append-only, so a certificate's absence from a query means nothing, however the
query went."* **However the query went** is the whole ruling. §2 is what it looks like applied.

The `enumerable` limb is the half that keeps this from being a CT exemption, and it is why the rule
is stated as a property of `completeness` rather than as a carve-out. An `enumerable` source
*"returns a complete set over a declared scope, so silence within that scope is evidence"*. A later
`Batch` from such a source, over a **recorded scope covering the subject**, that does not name it,
is a **measured absence** — and ADR-0006's existing route handles it with no new machinery, because
that is precisely what ADR-0006 built. The citation is not expiring. It is being **contradicted**,
by evidence, in a `Batch` whose scope record is what makes the contradiction admissible.

Two things follow and both matter.

**The clock never appears, at any `completeness`.** Even in the `enumerable` case there is no
duration, no horizon and no `k` — there is a later batch that either named the subject or covered it
and did not. This is why the ruling is *a citation never ages* rather than *a citation ages only
against `enumerable`*: the second form invites a session to go looking for the rate.

**The population is empty in v1, and that is stated rather than hidden.** No shipped source is both
`enumerable` and admits without observing. The operator's zone file is `enumerable` and admits
`Name`s, but it **observes** `dns-record`, so its hop is an observation and it ages under `zone`
(`k` × the re-supply interval) by the ordinary machinery. The limb is written for the source that
arrives later, so that whoever adds one does not read the CT answer as the general one.

### 4. What *"a citation goes stale"* means, since the glossary says it and it is not wrong

The sentence is live and load-bearing and it must not simply be struck.

Where the hop is an **observation**, it is exact and it has a live instance: an `Address` *"is in the
estate exactly while a **current** resolution cites it or a `Seed` covers it"*. The resolution ages
past `k` cadences of `dns`, it is no longer current, it cites nothing, and the `Address` leaves with
its timelines. ADR-0084 spells the mechanism out as the consequence of loosening the `dns` dial, and
ADR-0006's *"de-citation closes the probing gate"* is the same event seen from the safety side.
**That clock belongs to the observation.** The citation is a link. Links do not have ages.

Where the hop is a **`Batch`**, there is no observation, so there is no clock to borrow, and §2 and
§3 say there must not be one of its own. The glossary sentence is therefore **true and total**
already — a citation that goes stale withdraws — and what this ADR supplies is that the antecedent is
**unreachable** on a `Batch` hop. It is amended at the site rather than struck, per
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md), because read
alone and in the present tense it invites a session to build the trigger.

### 5. A `Citation` is necessary for membership and not sufficient — which is what makes §1 and §4 consistent

This is the objection that has to be answered or the ruling collapses: if a CT citation never
retires, does it **keep alive** a `Name` the resolver has Name-Errored?

**No, and the asymmetry is already in the model.** ADR-0006 makes membership *"a Derived view over
the latest observation per facet"*. A `Citation` answers *why is this here* — it is the chain that
makes provenance traversable and the probing gate legible. Losing it withdraws a subject. **Holding
it asserts nothing**. So a withdrawn `Name` and a live CT citation coexist without contradiction:
the name is gone by measurement, and the record of what once introduced it is still the honest answer
to how it ever entered.

The one exception proves the rule and is confined. `Address` *"has no lifecycle of its own, because
nothing ever observes an address's existence"*, so for `Address` alone the current citation carries
the whole membership — *exactly while* a current resolution cites it. That is the sufficient kind,
and it is the kind that ages, because its hop is an observation.

**Certificate transparency admits `Name`s and nothing else** (ADR-0027, and
[ADR-0060](./0060-a-wildcard-san-is-a-pattern-over-names-and-admits-none-of-them.md) for the
boundary). A `Name`'s membership is measured. So a non-aging CT citation can never be the thing
holding a subject in the estate against the evidence, on any subject kind CT can reach.

### 6. The losing option, stated at its strongest: a sixth `Scan` over the CT poll, carrying a citation bound

This is the answer the ticket's title points at and it had real support. Four `Scan`s existed and
#124 minted a fifth on the reasoning that *a measurement needing a cadence of its own takes a `Scan`
of its own*. #142 applied the same reasoning a third time. CT's admissions are the last row in
[`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §16.2 with no covering
`Scan`, and the pattern-completion is obvious: mint `ct`, give it a cadence, let `k` cadences bound
the citation, and the estate stops being append-only.

It loses, and it loses in two separable pieces, which is the finding of this ticket.

**The withdrawal half loses permanently and on capability** — §2. There is no cadence at which it
becomes safe, because its firings do not become truthful at any rate. This is a stronger refusal
than the *bad afternoon* framing the ticket supplied, and it means the option does not return with a
better-behaved instrument.

**The scheduling half does not lose. It is a different question, and it is still open** — §7.

Two further readings were considered and rejected on the way:

- **Let CT's citation take its bound from the `dns` `Scan`**, since a CT-admitted name is resolved on
  it anyway. This is the hidden-field failure ADR-0005 refused, #124 refused again for the zone file
  and ADR-0084 refused a third time — and ADR-0084 refused the *name* `discovery` specifically
  because *"read alone and in the present tense the name tells the next session to hang the crt.sh
  poll off it."* Doing it under a name that does not even claim to cover CT is worse.
- **A citation horizon as a declared parameter, with no `Scan`.** ADR-0084 rejected the identical
  shape for DNS: a bound that is not `k` cadences of a covering `Scan` is a fifth mechanism beside
  the one the whole model reads. Here it is emptier still — it would be a number attached to a
  process §2 shows can never be evidence.

### 7. What is NOT decided here: the CT poll has no cadence anywhere in the corpus, and its scope cannot be drawn yet

Stated rather than absorbed, which is the move #124 made to produce ADR-0084 and ADR-0084 made to
produce this ticket.

**[verified against the corpus]** Nothing in `CONTEXT.md`, `docs/adr/` or `docs/spec/` says when
`crt.sh` is polled. ADR-0003 clears it and ships it enabled, ADR-0005 gives it a 5 req/min
instance-wide throttle, ADR-0084 declines to hang it off `dns` — and no document gives v1's flagship
keyless discovery source a schedule of any kind. That is a real hole and it is **not** the one this
ticket was opened about: with withdrawal off the table it has no currency consequence at all. Its
warrant is **admission latency** (a name issued a certificate today should enter on some stated
clock) and **`Coverage`**, whose whole job is *did it run, and when*.

One rule is established here so that whoever closes it inherits the safety property rather than
re-deriving it:

> **A `Scan` covering a source that admits without observing carries no currency bound and no
> withdrawal power.** ADR-0007's currency rule quantifies over **observations** — *"an observation is
> current while it is within `k` cadences of the Declared `Scan` whose scope covers that `(subject,
> facet, port, vantage)`"* — and such a source produces none. The bound has an empty domain. It
> schedules, and it gives `Coverage` a row; it bounds nothing.

**Its scope cannot be drawn today, and that is why it is not minted here.** §16.2's residue is
enumerated as *exactly one row*, and walking that enumeration against the corpus rather than quoting
it turns up a second candidate the table mis-files — see §8. A `Scan` cannot be named for a scope
whose membership is undecided. ADR-0084's own objection to `discovery` was precisely that a `Scan`
name must describe its scope truthfully.

### 8. The §16.2 enumeration is one row short, and the extra row is undecided rather than missing

The brief for this ticket said to verify the residue against the corpus rather than quote it. Walking
it:

- **Own recursive resolver** — observes. Covered by `dns` (#142). Holds.
- **Wildcard detection** — admits nothing of its own. Rides `dns` in the same batch (ADR-0070). Holds.
- **Zone file** — observes `dns-record` with `declared` authority. Covered by `zone` (#124). Holds,
  and it is the reason the `enumerable` limb of §3 has no v1 instance.
- **Our own prober** — observes four facets. Covered by the two port tiers and `tls-acceptance`. Holds.
- **The registry paths** — not sources. They yield `Proposal`s (ADR-0012), produced *"only in answer
  to an operator act … never on a cadence"*. Holds, and none is owed a `Scan`.
- **Certificate transparency** — admits without observing, no covering `Scan`. Holds. This ADR.
- **Common Crawl index (CDX)** — §16.2 files it as *"either a proposer … or a transport for the
  resolver's own exchange"*. **It is neither.** It yields *"hostnames linked from the crawled web"*,
  and a `Proposal` is defined as *"a candidate **address scope**"* — a hostname is not one, so
  ADR-0012's route is closed to it — while it makes no DNS query, so it is not a transport for the
  resolver either. **A source yielding hostnames and observing no facet is a second instance of
  ADR-0027's shape.**

Whether that second row exists in v1 is **undecided in the corpus, not decided against**.
[`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §1 lists Common Crawl as
**Tier 1, on by default**. ADR-0003 disqualified two sources on terms and Common Crawl is neither.
And it appears in **no ADR, no glossary entry and no spec document** — grepped for the name across
all three, zero hits.

Nothing in this ruling turns on it, and that is deliberate: §3 is a property of `completeness`, not a
CT carve-out, so **it already governs Common Crawl** — a crawl index is `corroborative` beyond
argument, a hostname's absence from a crawl meaning nothing — and its citations do not age either. It
is §7's `Scan` that cannot be sized until the row is settled.

### 9. Where this is thin, stated rather than smoothed

- **The `enumerable` limb of §3 has no v1 instance.** It is derived from `Completeness`'s own
  wording and from ADR-0006's measured-absence route, and it is not tested against a real source. If
  the shape is wrong, it is wrong in a direction that costs nothing today and would be found by the
  first `enumerable` admitting source anyone adds.
- **§2's fourth route is unmeasured.** Whether `crt.sh` retains rows from retired logs was not
  measured for this ticket. It is named as a candidate rather than asserted, and the argument does
  not rest on it — the three measured or documented routes already close it.
- **Common Crawl's shipping status is read off a research note's recommendation, not off a ruling.**
  §8 states it as undecided precisely because no decision exists to read. A session that finds one
  should correct §8 at its site.
- **The crt.sh failure rate is quoted, not re-run — and the ruling deliberately does not rest on
  it.** §7's series is dated **2026-07-31** and is a dated record like every other figure in a closed
  record. This matters more than usual here, because the tempting version of §2 is *the instrument is
  too unreliable to retire subjects with*, which **is** a rate argument and would inherit the
  figure's staleness: a session that re-measured crt.sh at 99% would think it had reopened the
  question. **It would not have.** §2 is an argument from **append-only**, a property of certificate
  transparency itself, and it holds at every failure rate including zero. The measurement is cited as
  the second reason and never as the first, and it is **owed a re-run** on its own account — it is
  the instrument-reliability input to a source that ships enabled — but that re-run is
  [#142](https://github.com/winniel123/verge-asm/issues/142)-era work and belongs to whoever
  specifies the crt.sh client, not here.
- **No retrieval was performed for this ticket.** Every quoted artefact is a file in this repository,
  read directly. Recorded because [#153](https://github.com/winniel123/verge-asm/issues/153) has just
  established at `sensitive-ports.md` §45.10 that *G7's rider reaches **any intermediary***, having
  caught a summarising fetch asserting a sentence RFC 4251 §1 does not contain. Nothing here went
  through one, and a session re-checking §7 must retrieve **raw** — an intermediary that
  manufactures presence is the same failure as a 404 read as absence, with the sign flipped.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) is amended in three entries and gains no term.** `Citation`
  records that it never ages, that the observation hop's clock belongs to the observation, that a
  `Batch` hop has none, and that a citation is necessary for membership and not sufficient.
  `Completeness` records that it governs the citation hop as well as the observation, and that only
  an `enumerable` source's silence can contradict a citation. `Batch` records that a `Batch` held as
  the current `Citation` of a subject in the estate is retained until a later `Batch` from that
  source supersedes it in that role, there being no clock that will release it.
- **[ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md)'s seam
  sentence is closed at its own site.** *"Does not age the same way"* now has an answer — it does not
  age at all — marked per ADR-0058, because read alone it sends a session to work out the rate.
- **[ADR-0084](./0084-a-scan-is-a-cadence-over-an-exchange-and-an-uncovered-facet-has-no-currency-bound.md)'s
  open handoff is discharged in part.** Its *"the passive sources' `Citation` staleness is open and
  ticketed"* is answered. Its decision-table row and its *cover the passive sources' admissions on
  this `Scan`* rejection are marked. **The poll's cadence is not discharged** and is re-ticketed with
  a drawn boundary rather than left as prose.
- **[ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)'s worked message is
  repaired.** Its example read *"cited by a `crt.sh` observation"*, which ADR-0027 made impossible
  and did not sweep. Read alone and in the present tense it would cause a competent session to build
  a `crt.sh` observation — ADR-0058's test, on a defect **created by ADR-0027's own ruling and
  missed by its own consequence sweep**, which is [#102](https://github.com/winniel123/verge-asm/issues/102)'s
  shape rather than a new one.
- **No `Scan` is minted, no aperture input moves, no declared parameter is added, and `k` does not
  move.** Five `Scan`s, two port tiers, seven aperture inputs — all unchanged.
- **No `Gap` is opened and none becomes reachable.** CT holds no timeline (ADR-0027), so there is
  nothing for a `Gap` to sit on. This ruling adds no route to a `Gap` of any kind.
- **The map's `Out of scope` entry for [#56](https://github.com/winniel123/verge-asm/issues/56) —
  *certificate-transparency mis-issuance detection, and any CT-fed facet* — is not reopened.** This
  ADR mints no facet, no timeline, no value and no observation from CT. It rules on the `Citation`
  hop, which is the object ADR-0027 put there precisely **because** there is no facet.
- **The cost is stated: `Name`s admitted by CT beneath a `Shadowed` answer or a `Lame` delegation
  stay in the estate permanently**, with the three operator routes out that ADR-0006 and its #35
  amendment name. This is ADR-0006's cost and not a new one, and `CONTEXT.md`'s `Name` entry already
  says it in the present tense.
- **A retention consequence lands, and it is small.** A CT `Batch` is retained under ADR-0041's
  second limb — *the current `Citation` of a subject in the estate* — and with no clock releasing it,
  it is released only when a later CT `Batch` re-admits the same `Name` and takes over the role. CT
  being append-only, that is the normal case on every poll. The residue is names dropped from a
  **truncated** answer, whose citation stays pinned to an older `Batch` retained indefinitely. It is
  bounded by the 999-row cap and it is stated rather than modelled around.
- **[#147](https://github.com/winniel123/verge-asm/issues/147) inherits a narrowed population, not a
  new closure.** ADR-0082 names *an `Address` losing its last citation* as one of the closure routes
  it does not mint vocabulary for. This ruling settles that **de-citation has exactly one producer —
  the observation hop** — so #147 specifies one route rather than two, and no departure arises from a
  `Batch` hop for it to record.
- **A `Scan` over an admitting source is pre-armed as safe.** §7's rule stands whether or not §7's
  successor mints one, so no future session has to re-derive that a CT `Scan` would not carry a
  currency bound.
- **One term collision, disambiguated rather than renamed.**
  [ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) uses *citation staleness*
  for a curated table's **supporting artefact** ceasing to say what a cell claims — its G8 check. That
  is a different object in a different corpus (a document reference in
  [`sensitive-ports.md`](../research/sensitive-ports.md)), and **nothing in this ADR reaches it**.
  Recorded because the two phrases are identical and a session grepping for one will find the other.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A sixth `Scan` over the CT poll, with `k` cadences bounding the citation** — the ticket's own title, the pattern-completion after #124 and #142, and the only option with three ADRs of precedent behind it | **The losing option, and it loses on capability rather than on cost or noise.** CT is append-only, so a citation clock re-arms on every poll for every name that is really there. Its complete firing set is: the source errored (barred twice over, and **[measured 2026-07-31, quoted]** at 4/8 with spurious 404s), the 999-row cap truncated silently under a 200, or the name was only ever in a wildcard SAN and was never admitted. **Every firing is a defect of our own instrument and none is a fact about the estate**, so there is no cadence at which it becomes truthful and it does not reopen on a better instrument — the argument is from append-only and survives the failure rate going to zero |
| **An exemption: CT's citations are declared exempt from a staleness rule that otherwise exists** | The right answer wearing the wrong shape. It concedes that citations age and then carves out the one source we happen to ship, which makes the CT names *"immortal by decision"* exactly as the ticket feared, and hands the next admitting source no rule at all. The property is `completeness`'s, it was always there, and stating it as such covers Common Crawl and everything after it for free |
| **A citation ages against `enumerable` and not against `corroborative`** — the ticket's own framing of the question, and very nearly right | It keeps the word *ages*, and the word is the defect: it invites the next session to look for the rate. In the `enumerable` case there is no duration and no `k` — there is a later `Batch` that covered the subject's scope and did not name it, which is ADR-0006's measured absence and needs no new mechanism. A citation is **contradicted**, never expired |
| **Hang the CT citation's bound off the `dns` `Scan`**, since a CT-admitted name is resolved on it anyway | The hidden-field failure, refused by ADR-0005, again by #124 for the zone file, and a third time by ADR-0084 — which refused the very *name* `discovery` on the ground that it would tell the next session to hang the crt.sh poll off `dns`. Doing it under a name that does not even claim to cover CT is the same failure with the sign-post removed |
| **A citation horizon as a declared parameter, with no `Scan`** | ADR-0084 rejected the identical shape for DNS: a bound that is not `k` cadences of a covering `Scan` is a fifth mechanism beside the one the whole model reads. Here it is worse — a number governing a process §2 shows can never produce evidence |
| **Mint the scheduling `Scan` here, severed from any bound** | Correct in substance and premature in fact. §8 shows §16.2's residue is one row short and that the extra row's shipping status is undecided, so the `Scan`'s **scope** cannot be drawn — and ADR-0084's own objection to `discovery` is that a `Scan` name must describe its scope truthfully. Stated and ticketed rather than guessed at, which is the move that produced ADR-0084 |
| **Treat CT admissions as `Proposal`s so the operator confirms them and the question dissolves** | Refused by ADR-0027 already: ADR-0022 makes confirmation singular and permanent, so a 400-name CT answer becomes 400 clicks and every CT name is read by nothing until one lands. It is also ADR-0012 read literally, which ADR-0012's own rationale contradicts |
| **Say nothing, and leave the trigger undefined as ADR-0041 and ADR-0084 did** | `CONTEXT.md`'s `Citation` entry carries a live withdrawal consequence in the present tense with no trigger, which is ADR-0058's test failed at the glossary — the site every session reads. An undefined trigger on a withdrawal route is the shape that gets invented by the next session that needs one, which is ADR-0057's *a withdrawal that supplies no replacement does not hold* pointed at an antecedent instead of a consequent |
