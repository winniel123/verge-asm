# ADR-0067: A claim fails on the owner's affirmative naming, not on the reach of its own prohibition

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#109 `1433/tcp`'s footing is gone and §10.3's failure condition is met — does the row survive?](https://github.com/winniel123/verge-asm/issues/109)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[`sensitive-ports.md`](../research/sensitive-ports.md) §10.3 gave Claim 3 a boundary limb and a
failure condition in one breath:

> "**The boundary must be named by the owner.** Where the owner names the public internet as a
> **supported deployment environment**, Claim 3 fails however strongly a third party disapproves."

The corpus then spent eighteen sections building a **footing** apparatus over the same artefacts:
[ADR-0050](./0050-an-owners-category-statement-reaches-the-members-its-own-artefacts-place-inside-it.md)'s
three limbs of *reach*, whose limb 3 is a **defeat test**; and
[ADR-0059](./0059-a-footing-tier-grades-evidential-distance-never-the-owners-conviction.md)'s four,
whose limb 4 branch (c) refuses *"a preference against an architecture the owner elsewhere names as
supported"*. Both cite §10.3. **Nobody wrote down whether they are the same test.**

§33 ([#107](https://github.com/winniel123/verge-asm/issues/107)) read them as one measurement met at
two places. Walking the prohibition tier per row, it found `1433/tcp` MS SQL failing ADR-0050 limb 3
and ADR-0059 limb 4(c) *"on the same measurement"*, demoted the cell out of the graded table, and
**routed the row** — under [ADR-0037](./0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md)
limb 2, §26.4's `2049/tcp` precedent, and [#37](https://github.com/winniel123/verge-asm/issues/37)'s
rule that a row moves only on a retrieval **scoped to the row**.

§33.10 left a live counter-argument behind it, and the ticket required it to be heard again for the
row: the word *directly* in Microsoft's *"don't connect your SQL Server instances **directly** to the
Internet"* may carve out a **filtered** exposure, in which case the Azure Public option sits outside
the forbidden class and the prohibition is not defeated at all.

**Running that argument at the row exposes the gap.** The carve-out is an argument about **what the
prohibition forbids**. §10.3's failure condition asks about **what the owner affirms**. They are not
the same proposition, and a session that treats them as one gets the direction of the carve-out
exactly backwards: granting it does not rescue the claim, it removes the only sentence that was in
tension with the affirmation.

The corpus has the shape of this distinction twice already and has never generalised it.
[ADR-0054](./0054-a-claim-step-is-answered-only-by-evidence-about-that-step.md) rules that a claim
**step** is answered only by evidence about **that step**. `sensitive-ports.md` §34.3 limb (c)
([#105](https://github.com/winniel123/verge-asm/issues/105)) rules that §10.4's one-way rule
*"governs the attestation gate and not the claim gate"* — a value read for what the software **does**
is read in both directions. This ADR is those two moved up one level: **the claim gate and the
attestation gate read the same artefacts for different propositions.**

A second gap surfaced in the same retrieval and had no home anywhere. **[measured]**, retrieved
2026-08-14, Microsoft's guidance is partitioned by `applies to` banner: the prohibition sits on a
page scoped to **SQL Server on Windows** carrying a single moniker icon; the Public-connectivity
guidance sits on pages scoped to **SQL Server on Azure VM**; and **the two sets never cross-reference
each other**. §10.5 keys ownership on the **artefact** rather than the party, and §33.2's *addressee*
rider keeps an owner's own managed service from defeating a statement addressed to operators — but
neither answers whether a **deployment-scoped** banner narrows the owner's voice.

## Decision

**A claim fails on the owner's affirmative naming, not on the reach of its own prohibition — and an
`applies to` banner scopes the artefact, not the pair.**

### Limb 1 — the two gates read the same artefacts for different propositions

**`sensitive-ports.md` §10.3's failure condition is answered by the owner's *affirmative* statement**
naming the public internet as a supported deployment environment for the pair. Whether the owner's
own **prohibition** reaches the pair, and whether that reach is defeated, is an **attestation**
question — ADR-0050 limb 3, ADR-0059 limb 4 — and it is answered at the footing gate.

Three consequences, and the second is the operative one.

1. **The prohibition's job at the claim gate is to *name the boundary*, and nothing else.** §10.3
   requires *the boundary must be named by the owner*; the failure condition fires on a **second,
   independent utterance**.
2. **A reading that narrows the prohibition does not rescue the claim.** It removes the tension and
   leaves the affirmative naming standing **alone**, which strengthens the failure condition rather
   than weakening it. A carve-out argument is therefore evidence **for** the claim's failure at the
   row, whatever it does at the tier.
3. **Symmetrically, an affirmative naming that defeats a footing does not by itself remove a row.**
   The row falls only where **no claim in §10.2's closed set fits**, tested per claim. A footing
   demotion and a row removal are two rulings and each needs its own retrieval — which is exactly
   what ADR-0037 limb 2 and #37 already require, and this limb is why they are not a formality.

> **Amended at this clause by [#112](https://github.com/winniel123/verge-asm/issues/112)**
> (`sensitive-ports.md` §37), per
> [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened by
> [#106](https://github.com/winniel123/verge-asm/issues/106). **Limb 1 sorts `sensitive-ports.md`
> §33.2's riders, and it turns out not all of them travel.**
>
> **A rider on §33.2's discriminator travels to the claim gate if and only if it bounds the
> *affirmation*.**
>
> - A rider fixing **which listener the statement is about** — *the addressee is part of the class*
>   (§33.2), *the direction is part of the class* (§36.8) — **travels**. §10.3's failure condition asks
>   whether the owner names the internet as supported **for the pair**, and the pair is a listener on
>   the addressed estate.
> - A rider **bounding the affirmation** in the owner's own words — *"You **must** restrict the
>   authorized public IP addresses to a single IP address or a small range"* — **travels**. An endpoint
>   answering an enumerated source set is not reached from an **internet vantage**
>   ([ADR-0010](./0010-exposure-composes-two-reaches.md),
>   [ADR-0017](./0017-exposure-needs-both-legs.md)). It is a **scope** read off a string, so
>   [ADR-0059](./0059-a-footing-tier-grades-evidential-distance-never-the-owners-conviction.md) limb 2
>   is untouched.
> - A rider narrowing the **prohibition's own class** — *an "unprotected" node* — **does not travel**.
>   Consequence 2 above makes carrying it across not merely wrong but **backwards**: narrowing the
>   prohibition removes the tension and leaves the affirmative naming standing alone.
>
> **[measured]** §33.3 spared `9200`/`9300` on the rider that does not travel and `3306` on the rider
> that does, and that single distinction is the whole of #112's result. **Consequence 3's converse is
> instantiated for the first time**: `9200`/`9300`'s **footing survives** — §32.2 limb 4 is asked of the
> carrying statement, whose class has no supported instance — while the **rows** meet §10.3's failure
> condition and are **routed** under ADR-0037 limb 2. A footing surviving on a rider the claim gate does
> not read.
>
> > **Amended again at this clause by [#114](https://github.com/winniel123/verge-asm/issues/114)**
> > (`sensitive-ports.md` §38), per ADR-0058. **A fourth candidate rider was tested against the
> > taxonomy above and it does NOT travel, because it bounds nothing — so the taxonomy is applied
> > rather than extended and it stays an *iff*.**
> >
> > **[measured]** Elastic **disclaims** the component that supplies ECE's internet reach: *"ECE does
> > not include a built-in load balancer, so **you must provision and configure one** in front of the
> > ECE proxies"*, and *"Provisioning and configuring the load balancer is the customer's
> > responsibility and is **outside the scope of this documentation**."* A session could read that as
> > bounding the affirmation. **It is not the second bullet's shape**: that one travels because an
> > endpoint answering an **enumerated source set** is not reached from an internet vantage
> > ([ADR-0010](./0010-exposure-composes-two-reaches.md),
> > [ADR-0017](./0017-exposure-needs-both-legs.md)), and a disclaimer of responsibility enumerates no
> > source set — it bounds **who writes the configuration**, which is not a scope. It is not the first
> > or third bullet either.
> >
> > **The underlying point is about this ADR's own decision sentence: the element is *naming*, not
> > *supplying*.** *"Microsoft ships the provisioning option"* was corroboration in Consequences below,
> > never a limb. **[measured] the reading that makes supply an element is falsified by
> > `sensitive-ports.md` §17.4**, where RabbitMQ's *"[client ports] should be accessible to hosts that
> > run applications, **which in some cases can mean public networks, for example, behind a load
> > balancer**"* was ruled to meet §10.3's failure condition — and RabbitMQ ships no load balancer
> > either. Adding the element withdraws that finding and reopens two §4.6 cells. `sensitive-ports.md`
> > §10.3 is amended at its own clause to say so. §38.7.
> >
> > **And consequence 3's converse, recorded above as a first, survived exactly one pass.** #114
> > removed both rows on a pair-scoped retrieval, and the footing cells left §2.2's graded table **with
> > them** — *a cell cannot outlive its row*. That is consequence 3 read literally rather than a
> > counter-example to it: two rulings, two retrievals, and the second disposed of both.

### Limb 2 — an `applies to` banner scopes the artefact, not the pair

**An owner's deployment-scoped document is the owner speaking about the pair wherever the sentence's
subject is the protocol's clients and its addressee is *the operator*.** The banner is admissible
evidence of **whose deployment is described** and is never a boundary around the owner's voice.

**Where the addressee is the owner's own managed service, §33.2's addressee rider governs instead**
and the document corroborates without defeating — which is what keeps Azure Files from defeating
`445/tcp`, and Azure SQL Database's public 1433 gateway from bearing on this ruling at all.

> **Amended at this clause by [#112](https://github.com/winniel123/verge-asm/issues/112)**
> (`sensitive-ports.md` §37.8), per
> [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).
> **This limb and §33.2's addressee rider are TWO tests sharing one coordinate, not one instrument
> stated twice, and they must not be folded into one clause.**
>
> They read one fact — *whose listener is this document about* — and partition its two values: **the
> operator's**, where this limb admits the document, and **the owner's own**, where the rider governs.
> The sentence above is the hand-off. **Neither subsumes the other:**
>
> - **§33.2's rider decides `445/tcp`**, where this limb's other three grounds are all satisfied — the
>   Azure Files pages are Microsoft's, addressed to an operator, about SMB on TCP/445. Only *whose
>   listener* disposes of them.
> - **This limb decides `1433/tcp`** against an argument the rider is **silent** on: that an
>   `applies to` banner makes the Azure VM pages a different **subject** (§10.5's distributor position).
>   That is a question about standing and reach, and ground 1 — *the artefacts live in
>   `MicrosoftDocs/sql-docs`* — is what answers it.
>
> **Folding them would convert the rider's silence about banners into a boundary**, which is
> [ADR-0040](./0040-a-specifications-silence-is-not-the-owners-silence.md)'s failure mode with the
> subject changed — the sharpest risk this ADR's own thin-ground section names against limb 2.
>
> **[measured]** across #112's thirty-two members the addressee coordinate is load-bearing **twice** —
> `445/tcp` at the attestation gate and **`27017`/`27018`/`27019` MongoDB at the claim gate**, where
> §32.2's limb 3 does not exist and Atlas supplies the affirmation — and the banner half **once**
> (`1433/tcp`, and it is what admits Elastic's ECE documents at §37.5). **No member is decided by
> both.**

The limb rests on four grounds, each about the artefact rather than the party, per §10.5's
*artefact, not party*:

| Ground | What it reads |
|---|---|
| **Where the artefact lives** | The Azure VM pages are in `MicrosoftDocs/sql-docs`, the same repository as the prohibition page — SQL Server documentation about SQL Server, not platform documentation that mentions it |
| **Who is addressed** | The operator provisions the VM, administers the instance and holds its credentials. The owner is telling **the operator** how to make **the operator's own** listener internet-reachable |
| **What the sentence is about** | *"Any client with internet access can connect to **the SQL Server instance**"* is a proposition about the protocol's **intended clients**, which is what Claim 3 is, and what ADR-0054 requires of evidence read at a claim step |
| **What the list is keyed on** | `(port, transport)` pairs, not services and not deployments (§2.1). A claim must be true **of the pair**, and the list has no cell for *except on deployment X* — §5 refused the middle band that would carry one |

## Consequences

- **`1433/tcp` Microsoft SQL Server leaves the sensitive list** — `sensitive-ports.md` §35. Claim 3
  fails on §10.3's failure condition; Claim 1 is unavailable (SQL Server ships admitting no anonymous
  command, and Microsoft's own page makes SQL authentication **required** for public access); Claim 2
  fails on its successor clause (TDS negotiates TLS on the **same** port). §10.2 closed the set, so
  *no claim* is *no row*. **The list is 40 pairs, class totals `12 / 7 / 21`**, and the pair is in
  §4.6 as the table's **21st** entry.
  > **Superseded as an absolute by [#114](https://github.com/winniel123/verge-asm/issues/114)** —
  > **the list is 38 pairs, class totals `12 / 7 / 19`, and §4.6 is ~~23~~ 23 entries**, `9200/tcp` and
  > `9300/tcp` having been removed on this ADR's own limbs. The `1433/tcp` verdict is unchanged.
  > **§4.6 is 24 entries since [#125](https://github.com/winniel123/verge-asm/issues/125)**, which
  > added `9443/tcp` on this ADR's limb 1 without moving a pair — `sensitive-ports.md` §39.5. The list,
  > the class totals and every `1433/tcp` and `9200`/`9300` verdict are unchanged.
- **[ADR-0008](./0008-derivation-versions-move-on-content.md) is triggered.**
  `sensitive-port-reached-from-internet`'s content moves, its leaf bumps and every span of that
  derivation `Break`s. Pre-install this is vacuous and **not waived**, which is the same sentence
  [ADR-0009](./0009-verge-core-is-a-union.md)'s #91 and #95 amendments wrote in the adding direction.
- **[ADR-0009](./0009-verge-core-is-a-union.md)'s union does not move, and that is worth stating
  because it is counter-intuitive.** **[measured]** `1433/tcp` is in the frequency half
  (`sensitive-ports.md` §29.2, *top-100, retained*), so `|F| + |S \ F|` is unchanged at **136 pairs —
  131 TCP, 5 UDP**, the daily probe budget is unchanged, and the port is still probed every day.
  **A pair leaving the sensitive list costs the union nothing wherever the frequency half already
  carries it.** What stops is the `Signal`, not the measurement.
- **The aperture *narrows*.** `safe-active-probing.md` §2.4's line reads
  ~~`0 of 41 sensitive pairs unread`~~ ~~`0 of 40`~~ **`0 of 38`** (#114) — a **denominator**, with the
  numerator `0` for every `|S|`, and `0 of 16 rules unevaluable` unchanged. Nothing becomes unread, so
  [ADR-0014](./0014-only-revealed-generalises.md) does not bite and no timeline opens.
- **ADR-0050, ADR-0054, ADR-0059 and ADR-0037 are all confirmed by use and none is amended.** Limb 1
  does not narrow ADR-0050 limb 3 or ADR-0059 limb 4; it says what they are **not** also deciding.
- ~~**Limb 2 reaches rows other than `1433/tcp` and the sweep has not been run.**~~ Every
  prohibition-tier owner documents a managed offering on its own port — Elastic Cloud on `9200`,
  MongoDB Atlas on `27017`, Azure Files on `445`, Oracle on `3306`. §33.2's two riders dispose of each
  **as the note currently reads them**, and none was tested against a limb that did not exist.
  Ticketed under ADR-0037 limb 2 rather than swept, and it **blocks
  [#12](https://github.com/winniel123/verge-asm/issues/12)** because it can reach a row.
  > **DISCHARGED by [#112](https://github.com/winniel123/verge-asm/issues/112)**
  > (`sensitive-ports.md` §37). **The sweep is run, over all thirty-two members limb 2 can reach** —
  > the two graded tiers for the attestation gate and **Class C** for the claim gate, a population the
  > tiers do not describe and which contains **six rows outside them**. **Thirty hold.**
  > **[measured] `9200/tcp` and `9300/tcp` Elasticsearch meet §10.3's failure condition** on Elastic
  > Cloud Enterprise documentation addressed to the operator — *"By default, all your deployments are
  > accessible over the public internet"* — and are **routed rather than removed**, this retrieval
  > having been scoped to a tier. **`27017` is disposed of on addressee, `445` on addressee and
  > direction, `3306` on a bounded affirmation, and `6379` on the pair.** The blocking relationship to
  > #12 passes to the successor ticket; **the rider taxonomy that decided it all is at limb 1's clause
  > above.**
  > > **The routing is DISCHARGED by [#114](https://github.com/winniel123/verge-asm/issues/114)**
  > > (`sensitive-ports.md` §38), on a retrieval **scoped to the pair** across all four of Elastic's
  > > corpora — self-managed, ECK, ECE and the shipped `elasticsearch.yml` at `v9.5.1`. **Both rows are
  > > REMOVED**; the list is **38 pairs**, class totals **`12 / 7 / 19`**, §4.6 **23** entries, tiers
  > > **13 / 11 / 3 / 11**, coverage **27 of 38**, §6.1 **`25 + 8 + 5 = 38`**, aperture denominator
  > > **38**. **`verge-core` is unchanged at 136 pairs — 131 TCP, 5 UDP**, both pairs being in the
  > > frequency half. **[ADR-0008](./0008-derivation-versions-move-on-content.md) is triggered.** This
  > > ADR is **applied and not extended**; **no ADR is minted and `0070` is left unused**.
  > >
  > > **[measured] the corpus that decides `9200/tcp` is self-managed, and #112 had not opened it**:
  > > Elastic's own security documentation says *"**The HTTP layer**: Used for communication between
  > > your cluster or deployment **and the internet**"*, of the interface `networking-settings.md`
  > > binds with *"`http.port` … Defaults to `9200-9300`"*. That is an **affirmation**, not a default,
  > > so §10.4's one-way rule is never reached for it — the whole *"ECE describes a default, not an
  > > offering"* counter-argument is unavailable against that pair. **The same sentence's next bullet
  > > separates the two pairs**: *"**The transport layer**: Used mainly for inter-node communications,
  > > and in certain cases for cluster to cluster communication"* is Claim 3's boundary stated by the
  > > owner, so **`9300/tcp` falls on ECE alone**, through ADR-0050 limb 2 on three of Elastic's own
  > > artefacts. **Claim 1 and Claim 2 were measured off the shipped bytes rather than assumed**, per
  > > `sensitive-ports.md` §35.6's precedent, and both are unavailable.
  > >
  > > **Limb 2's second live instance is confirmed and it now carries a removal.** #112 recorded
  > > Elastic's banner partition as limb 2's second instance; #114 spends it — and **[measured]** the
  > > counter-instance this ADR's thin-ground paragraph names, an owner disclaiming a deployment *in
  > > its own words*, is **still not met** across twelve further Elastic artefacts.
- **This is a *detectable* defect and not a curation trigger**, on the reasoning
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) uses for
  its own: both sides are written down — a row's claim, and the owner's affirmative sentences — and
  running limb 1 is a re-reading of artefacts the note already holds.

## Thin ground, flagged

**Limb 1 is a separation nobody had written down, and it is where the `1433` ruling actually lives.**
It reconstructs §26.4 (an **aspiration** is not an affirmation — the IETF names the Internet as an
NFSv4 *goal* and records in the same document that the goal is unmet), §4.4 (a **concession** is an
affirmation — Kubernetes conceding that managed distributions publicly expose the API server) and
§17.4 (a scoping sentence's other half is an affirmation — RabbitMQ's *"which in some cases can mean
public networks"*), and it reproduces all three verdicts. **But no prior section states it**, and a
reader who holds that the failure condition is defeated whenever the prohibition is narrowed reaches
the opposite verdict on `1433/tcp`. That reader's argument is stated in full at
`sensitive-ports.md` §35.4 rather than summarised, so it can be attacked.

~~**Limb 2 is stated from one instance.**~~ Its four grounds are strong and they are grounds about
**this** artefact set. A counter-instance would be an owner that disclaims a deployment **in its own
words** — *this guidance does not apply to X* — and **[measured]** Microsoft's partition is not of
that shape: the two document sets do not disclaim each other, they simply never mention each other.
**Silence is the whole of the partition**, and reading a silence as a boundary is
[ADR-0040](./0040-a-specifications-silence-is-not-the-owners-silence.md)'s failure mode with the
subject changed — which is limb 2's best argument and also the sharpest thing against it, because a
corpus partitioned only by silence could be **drift** rather than doctrine.

> **The struck sentence is WITHDRAWN by [#112](https://github.com/winniel123/verge-asm/issues/112)**
> (`sensitive-ports.md` §37) **and the paragraph beneath it stands.** **[measured]** limb 2 now has a
> **second** live instance, in a second owner's corpus and on a partition of exactly the shape this
> paragraph describes: Elastic's prohibition sits on a page bannered `Elastic Stack` and its
> public-internet statements on pages bannered `Elastic Cloud Enterprise` and `Elastic Cloud on
> Kubernetes`, **and the two sets do not disclaim each other**. The counter-instance the paragraph
> names — an owner disclaiming a deployment *in its own words* — is **still not met in either corpus**,
> which is why the sentence about silence is left standing rather than struck with the count.

**And the owner's freshest security document says nothing at all.** **[measured]** *SQL Server
security best practices*, `ms.date` **2026-05-07** — the most recently authored SQL Server security
page in Microsoft's corpus — contains neither `1433` nor the word *internet*, in any casing. A reader
could take that as the prohibition no longer being maintained, which reaches this ADR's verdict by a
different route; or as the security corpus moving on while operative guidance stands, which reaches
nothing. It is recorded because it is the strongest evidence that the prohibition is stale, and
because a session finding it later and thinking it was missed would be right to distrust the rest.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Keep the row on §33.10's *"directly"* carve-out** | It answers the **footing** question. Grant it in full and the affirmative naming stands **unopposed** — the carve-out strengthens the row's case rather than weakening it, which is limb 1. Two supporting grounds: the bullet's first clause, *"Install databases in the secure zone of the corporate intranet"*, is a **placement** requirement no firewall satisfies; and **[measured]** nothing on the page joins its *Use firewalls* bullets to the isolation bullet, so the join is the reader's inference and §2.2's *the claim may not be asserted by us* bars it |
| **Keep the row because an `applies to` banner narrows the owner's subject, the Azure pages being §10.5's distributor position** | The best argument against this ADR, and it is not the one the ticket named. Refused on limb 2's four grounds: the artefacts are in `MicrosoftDocs/sql-docs`; the addressee is the operator, not Microsoft, so §33.2's addressee rider does not reach them; the sentences' subject is the SQL Server **instance**; and the list is keyed on the **pair** |
| **Keep the row and re-found it on another claim** | Unavailable rather than refused. Claim 1 fails on its antecedent, and §28.5 already refused the *supports an unauthenticated mode* reading **by name for this port** — the reading that admits it *"moves `5432/tcp` to Class A on its `trust` method, `3306` and `1433` on their no-password accounts"* and *"admits half the list"*. Claim 2 fails on the successor clause, which is §9.2's LDAP disposal |
| **Keep the row with a stated exception for Azure VM deployments** | §5 refused the middle band and gave `6443/tcp` as the case that made the question real. A row whose exception is a **supported architecture** is the *rarely correct, worth mentioning* shape §5 exists to refuse, and a `Signal` is binary (ADR-0004) |
| **Amend §10.3 in place and mint nothing**, on #101's, #105's and #106's precedent | The strongest procedural objection, and the precedent is real. It loses on three counts: this ruling **removes a row**, the corpus's most consequential act, and the instrument that removes it should be met on its own by a session that never opens the note; limb 2 is a **new** instrument with no existing home in §10.3, §10.5 or §33.2; and the map's rules list wants a citable line. Recorded as the close call it is |
| **Route the row a second time**, on the ground that limb 2 is minted and spent in one pass | §33 routed because its **retrieval** was tier-scoped, which #37 makes dispositive. This retrieval is row-scoped, so the condition that forced the routing is discharged. Routing again because the ruling is hard leaves [#12](https://github.com/winniel123/verge-asm/issues/12) blocked on a question retrieved twice, and is the *leaving the question hanging* the map's standing instruction forbids. The instrument is stated **before** it is applied and flagged as this pass's own act — the discipline §33.2 used for the discriminator it minted and spent in one pass |
| **Treat the failure condition and the footing defeat test as one measurement**, as §33 did | Defensible at the tier, where both gates happen to be crossed by the same sentence. It is what makes the carve-out look like a rescue, and it inverts the direction of a narrowing argument. §33's tier verdict is unaffected either way; the row's is not |
