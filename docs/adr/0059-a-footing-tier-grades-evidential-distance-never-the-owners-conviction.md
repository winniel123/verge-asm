# ADR-0059: A footing tier grades evidential distance, never the owner's conviction

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#98 §20.8's prohibition-tier criterion is refuted by `873/tcp` — apply it or restate it](https://github.com/winniel123/verge-asm/issues/98)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[`sensitive-ports.md`](../research/sensitive-ports.md) §2.2 grades every row's attestation into three
tiers — **explicit prohibition**, **explicit trusted-network scoping** *"slightly weaker than a
prohibition"*, and **shipped default only**. The table has carried that grading since the note was
written and has never said **in what respect** one tier is weaker than another. §2.2 says *slightly
weaker*; it does not say weaker at what.

Two sections have since had to answer the question in passing, and each answered it in a different
vocabulary.

- **§20.8** ([#84](https://github.com/winniel123/verge-asm/issues/84)) refused to promote `4369/tcp`
  epmd out of the scoping tier on a **lexical** reading: *"The tier boundary in this note is drawn on
  **vocabulary**, not on force. Every prohibition-tier sentence names *the public internet*."*
- **§18.6** ([#88](https://github.com/winniel123/verge-asm/issues/88)) moved `10255/tcp` into the
  prohibition tier on a **grammatical** one: *"an unqualified negative about internet exposure and
  names **no trusted network**, which is what makes it a prohibition rather than a scoping."*

Both are reconstructions from the members. Neither states what the members were sorted **by**, and the
gap has now produced a measured falsehood in the note's own text.

**[measured]** [#93](https://github.com/winniel123/verge-asm/issues/93) §26.3 walked the whole
prohibition tier against §20.8's sentence and found it **refuted as a universal**. `873/tcp` rsync's
footing is *"Do not expose a cleartext daemon to an untrusted network"* (`rsyncd.conf.5.md`) and *"do
not send sensitive data across an untrusted network"* (`rsync.1.md`), both at `v3.5.0`; rsync scopes by
**trust boundary** in every artefact it ships and the string `internet` occurs **zero** times in
`rsync.1.md`, `rsyncd.conf.5.md` and `SECURITY.md`. The class list is exhausted, so no retrieval can
cure it. #93 reported the defect rather than applying it, because applying it moves tier counts and
§17.6's house precedent binds a pass to the question it was run for.

#98 is that question, and it cannot be answered without saying what the tier grades — which is the
thing nobody had written down. An unstated criterion is precisely what §2 exists to prevent, and
[ADR-0042](./0042-a-squat-is-contested-where-the-other-convention-is-live.md) refused
*"leave it unwritten and decide case by case"* on exactly that ground.

## Decision

**A footing tier grades the distance between the owner's own sentence and the proposition the row
asserts, counted in premises the reader has to supply. It does not grade the owner's conviction.**

Three limbs.

1. **The unit is a reader-supplied premise, and the tier is the count.** The row asserts *this
   `(port, transport)` pair being reachable from an internet vantage is never correct*. A footing sits
   in the strongest tier where the owner's ~~sentence~~ **statement of the port's permitted network**
   entails that with **no** premise supplied by the
   reader; in the next tier where it entails it through **one** — characteristically *the internet lies
   outside the boundary the owner named*; and in the weakest tier where the owner has published no
   ~~sentence~~ **statement** at all and the footing is a restricting default.
   > **The unit is WIDENED from *sentence* to *the owner's statement of the port's permitted network*
   > by the [#101](https://github.com/winniel123/verge-asm/issues/101) amendment below.** The narrower
   > word was written because every footing this ADR walked was prose; a boundary named in a
   > ports-table cell or an issued document is the same statement in another form. Marked at the
   > sentence per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
   > as widened by [#106](https://github.com/winniel123/verge-asm/issues/106) — this ADR marked limb 3
   > and the *"does not reach"* paragraph in place and left limb 1's own unit standing.
2. **Mood, force, hedging and priority label are inadmissible, in both directions.** Whether the
   sentence is an imperative or an advisory, whether it is hedged, and what severity the owner attaches
   to it are not read. A hedged sentence naming the row's own network outranks an unhedged imperative
   naming a trust boundary, and that is the rule working rather than failing. This is
   [ADR-0042](./0042-a-squat-is-contested-where-the-other-convention-is-live.md)'s refusal of
   *"gate on the candidate's attestation strength"* — *"it converts the gates into a ranking, which is
   the severity-shaped reasoning [ADR-0004](./0004-signals-are-release-coupled-rules.md) and §2 exist
   to keep out"* — applied to a **disclosure column** rather than to a gate.
3. **The lexical test is *necessary* for the top tier and is not *sufficient*.** For
   `sensitive-port-reached-from-internet`'s table the zero-premise test is **the owner's sentence names
   the public internet**, because the public internet is the network the rule reads. A footing whose
   sentence fails it **cannot** be in the prohibition tier. A footing whose sentence passes it is
   **not thereby promoted**: tier membership still requires the sentence to be the owner's (§10.5), to
   be about the port (§2.3, [ADR-0050](./0050-an-owners-category-statement-reaches-the-members-its-own-artefacts-place-inside-it.md)),
   and to state a **position** rather than an aspiration (§26.4). **[measured]** two scoping-tier rows —
   `2181/tcp` ZooKeeper and `25672/tcp` RabbitMQ — carry owner sentences that name the Internet and are
   **not** promoted by this ADR; the note has never stated the sufficient condition and this ADR does
   not invent one.
   > **Second sentence WITHDRAWN by the #101 amendment below.** The lexical test **is** sufficient in
   > conjunction with the three conditions this limb itself names, the list is **closed**, and
   > `2181/tcp` and `25672/tcp` are **promoted**. The first sentence — *a footing whose sentence fails
   > the lexical test cannot be in the prohibition tier* — stands unchanged.

## Rationale

**It is the rule the table has been applying, and the reconstruction reproduces every verdict.** That
is the test ADR-0042 set for itself and it is the test this has to pass. Read across the tier
assignments as they stand:

| Footing | Owner's sentence | Premises the reader supplies | Tier, and it is the existing one |
|---|---|---|---|
| `6379` Redis | *"not a good idea to expose the Redis instance directly to **the internet**"* | none | **Prohibition** — and it is **hedged**, which limb 2 says is irrelevant |
| `11211/tcp`+`/udp` memcached | *"you **must not** expose memcached directly to **the internet**"* | none | **Prohibition** |
| `3306` MySQL | *"Try to scan your ports **from the Internet** using a tool such as `nmap`. MySQL uses port 3306 by default. This port should not be accessible from untrusted hosts."* | none | **Prohibition** |
| ~~`1433` MS SQL~~ | *"don't connect your SQL Server instances directly to **the Internet**"* | none | ~~**Prohibition**~~ — **DEMOTED out of the graded table by [#107](https://github.com/winniel123/verge-asm/issues/107)** on limbs 2 and 4(c): Microsoft names internet-facing TCP/1433 as supported. `sensitive-ports.md` §33.4 |
| `9200`, `9300` Elasticsearch | *"Never expose an unprotected node to **the public internet**"* | none | **Prohibition** |
| `445` SMB | *"unlikely that any SMB communication originating from **the internet** … is legitimate"* | none | **Prohibition** |
| `623/udp` IPMI | *"not designed nor intended to be placed on or connected to **the internet**"* | none | **Prohibition** |
| `9042` Cassandra | *"you should not expose this port to **the internet**"* | none | **Prohibition** |
| `2379`, `2380` etcd | *"**must not** be exposed to untrusted networks or **the public internet**"* | none | **Prohibition** |
| `10250`, `10255` kubelet | *"not exposed publicly on **Internet**"* | none | **Prohibition** |
| **`873` rsync** | *"Do not expose a cleartext daemon to an **untrusted network**"* — imperative, unhedged | **one** — *the internet is an untrusted network* | **Scoping** — **the one cell this ADR moves** |
| `27017`/`27018`/`27019` MongoDB | *"only accessible on **trusted networks**"* | one | **Scoping** |
| `2049` NFS | *"on a **trusted physical network** between trusted hosts, it is entirely adequate"* | one | **Scoping** |
| `2375`, `2376` Docker | *"reachable only from a **trusted network** or VPN"*; *"not advisable on an **open network**"* | one | **Scoping** |
| `4369` epmd | *"should **only** be used in a **trusted network**"* | one | **Scoping** |
| `5432` PostgreSQL, `5984` CouchDB | no sentence; `listen_addresses = localhost` | n/a — no sentence to stand at a distance | **Weak** |

**Twenty-four prose footings, one criterion, one cell moved — and the moved cell is the one the
criterion was already measured to refute.** `873/tcp` is not an exception the rule accommodates; it is
the row the rule was found wrong on, moving to where the rule puts it.

**Limb 2 is the load-bearing half and it is the one a reader will resist.** rsync's *"Do not expose"* is
flatter and more imperative than Redis's *"usually it is not a good idea"*, and this ADR puts rsync
**below** Redis. That feels wrong, and the feeling is ADR-0042's `7000/tcp` feeling exactly — *"better
attested than `5432/tcp`, which is on the list … and it is refused"*. The answer is the same and it is
structural: **the column does not measure how much the owner minds.** It measures how much of the
row's proposition the owner said. rsync said *not on an untrusted network*; the step from there to *not
from an internet vantage* is the reader's, correct and obvious and still the reader's. That step is
exactly the thing §2.2 was built to disclose — *"the two forms are not equally strong, and the list must
not hide the difference"* — and hiding it because the sentence is forceful is the arbitrariness §2.2's
founding paragraph says *"destroys a curated list's credibility"*.

**Limb 2 also keeps the column falsifiable, which is limb 2 of ADR-0042 wearing a different hat.** *Is
this sentence forceful enough?* is a judgement, and a reviewer cannot be wrong about it. *Does the
sentence name the public internet?* is a **retrieval**: open the artefact, find the word or do not.
ADR-0042 made determinacy a retrieval for that reason, and `sensitive-ports.md` §15.7 recorded the
matching temptation by name — *"the word **live** invites the substitution and the next session will
feel the same pull"*. Here the word **prohibition** invites reading the column as a force ranking, and
this ADR records that pull rather than yielding to it.

**Limb 3 is where the rule would over-claim if it were left out.** §20.8's sentence is a **universal
over the prohibition tier**, not a membership test for it. Two scoping-tier rows name the Internet in
their owners' voices — ZooKeeper's `security.html` (*"not exposed directly to the Internet"*) and
RabbitMQ's networking guide (*"not exposed to the public Internet"*) — and reading the criterion as
sufficient would promote both, which is a cell move on a question nobody asked. It is recorded as
by-catch and routed, per [ADR-0037](./0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md)
limb 2 and §17.6's routing precedent.

**What this ADR does not reach, stated so its extent is in the artefact.** Limb 1 counts premises
between a **sentence** and a proposition, so it governs the **prose** footings only. §24
([#91](https://github.com/winniel123/verge-asm/issues/91)) placed `10259/tcp` and `10257/tcp` in the
scoping tier on a footing that is **not a sentence** — the owner's ports table plus a restricting
loopback default — reasoning relatively rather than lexically (*"strictly more than the shipped-default-only
footing carrying `5432` and `5984`"*). That placement stands and this ADR neither confirms nor
disturbs it; the scoping tier now holds two populations sorted on two different dimensions, and that is
disclosed rather than smoothed.

> **This paragraph is WITHDRAWN by the #101 amendment below.** Limb 1's unit is the owner's
> **statement of the port's permitted network**, not a sentence, so limb 1 **does** reach the non-prose
> footings; §24's placement is **confirmed** rather than left unreached; and the scoping tier holds
> **one** population, not two. `sensitive-ports.md` §32.7.

## Consequences

- **`873/tcp` rsync moves from the explicit prohibition tier to the explicit trusted-network scoping
  tier.** §2.2's tiers read **prohibition 14 · scoping 12 · weak 2**, coverage **28 of 39** unchanged,
  and 14 + 12 + 2 + 11 = 39. `sensitive-ports.md` §30.
  > **Composed with [#95](https://github.com/winniel123/verge-asm/issues/95), merged alongside this
  > one: prohibition 14 · scoping 13 · weak 3 · outside-subject 11, coverage 30 of 41, and 14 + 13 + 3 +
  > 11 = 41.** `10249/tcp` took a scoping cell and `10248/tcp` a weak one. **The move this ADR licenses
  > is the same move on either baseline** — which is why §30.8 stated it parametrically.
- **No `(port, transport)` pair moves and no row moves.** The list stays at **39 pairs**, class totals
  `11 / 7 / 21`, §6.1's `28 + 6 + 5 = 39` and §4.6's 19 exclusions are untouched.
  [ADR-0009](./0009-verge-core-is-a-union.md)'s union is unchanged and
  [ADR-0008](./0008-derivation-versions-move-on-content.md) is **not** triggered — the rule's reference
  data is byte-identical. A tier records how strong a footing is, never whether a row qualifies.
  > **Composed: 41 pairs, `12 / 7 / 22`, §6.1's `28 + 8 + 5 = 41`, §4.6's 20 exclusions** — all of it
  > #95's, none of it this ADR's. *No pair moves and no row moves* remains true of #98's own act.
- **§20.8's universal becomes true.** *"Every prohibition-tier sentence names the public internet"* is
  now a correct statement of a **fourteen**-member tier, made true by moving the counterexample rather
  than by widening the sentence. §20.8's `4369` ruling is **unchanged and re-founded**: `DEP-001` is one
  premise away, and its advisory mood — which §20.8 gave as a second reason — is withdrawn as a ground
  under limb 2.
- **[#12](https://github.com/winniel123/verge-asm/issues/12) is not touched.** The spec carries the
  list, the claims and the containment arithmetic; none of them reads a tier.
- **The weak tier, and therefore [ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md)
  §8's watch list, is unchanged at `5432/tcp` and `5984/tcp`.** **Unchanged by this ADR; composed it is
  three rows** — `10248/tcp` joined the weak tier at #95 in the same merge, and ADR-0032 §8 carries the
  amendment.
- **It does not generalise beyond a graded footing column**, and that is ADR-0032's ruling rather than a
  choice made here. [`weak-key-and-signature.md`](../research/weak-key-and-signature.md) has no footing
  tier and gains none.
- **The sufficient condition for the prohibition tier is still unstated**, and two scoping-tier rows
  now visibly bear on it. Routed, not answered.
  > **Answered by the #101 amendment below.** The condition is the conjunction limb 3 already
  > enumerated, the list is closed, and both rows are promoted.

### Amendment — [#101](https://github.com/winniel123/verge-asm/issues/101): limb 3 is closed, and limb 1's unit is a statement rather than a sentence

> **Limb 3's second half is withdrawn. The lexical test is necessary *and*, in conjunction with the
> three conditions limb 3 itself names, sufficient.** A footing is in the top tier **if and only if**
> the owner has published a statement that is (1) the **owner's for this port** (§10.5, §16.6), (2)
> about **this `(port, transport)` pair** (§2.3, [ADR-0050](./0050-an-owners-category-statement-reaches-the-members-its-own-artefacts-place-inside-it.md)),
> (3) naming **the public internet** (§20.8), and (4) taking a **position** rather than recording an
> aspiration (§26.4) or a label (§27.6). **Such a statement leaves the reader nothing to supply, and
> zero supplied premises is the top tier by limb 1.**
>
> > **Condition (2) is RESTATED by the [#110](https://github.com/winniel123/verge-asm/issues/110)
> > amendment below, and it is marked here as well as there** because
> > [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s unit is
> > the **sentence** per [#106](https://github.com/winniel123/verge-asm/issues/106), so an appended
> > block does not discharge this clause. **(2) about this `(port, transport)` pair *as the endpoint
> > being reached, on the estate the statement addresses*** — a **listener**, reached from outside that
> > estate. A statement about traffic **leaving** that estate on the same number, or about a listener
> > the **owner itself** runs and the addressed party does not, is about a different endpoint and does
> > not satisfy this condition. **The conjunction stays at four**; this states what *"this pair"* has
> > always denoted. `sensitive-ports.md` §36.8.
>
> **Limb 1's unit is *the owner's statement of the port's permitted network*, not a *sentence*.** The
> narrower word was written because every footing this ADR walked was prose. Prose is one form of the
> statement; a boundary the owner names in a ports-table cell or an issued document is another; a
> restricting default that names no network is not one at all. **The *"What this ADR does not reach"*
> paragraph is withdrawn** — limb 1 reaches the non-prose footings, and §24's placement of `10259/tcp`
> and `10257/tcp` is **confirmed** on the principal rather than left unreached.
>
> **Limb 1's premise-count mechanism and limb 2 are untouched.** Mood, force, hedging and priority
> label remain inadmissible in both directions. [#101](https://github.com/winniel123/verge-asm/issues/101)
> fenced both and the amending section reopened neither.

**Why the withdrawal happens here and not only in the note.**
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s test is
whether the superseded sentence, *read alone and in the present tense*, would cause a competent session
to act on it. *"The lexical test is necessary and not sufficient"* read alone would cause the next
session to refuse a promotion `sensitive-ports.md` §32 licenses, and *"it governs the **prose**
footings only"* read alone would cause it to treat the scoping tier as holding two populations that no
rule sorts. Both are load-bearing sentences of this ADR's own Decision, so both are withdrawn at the
site that specifies them. **This is also why no second ADR was minted:** ADR-0062 was reserved and is
**left unused**, as `0039`, `0041`, `0052`, `0053` and `0057` are.

**What forced it, and it is a measurement rather than a tidying.** **[measured]** this ADR recorded
`2181/tcp` ZooKeeper and `25672/tcp` RabbitMQ as *"carry[ing] owner sentences that name the Internet and
are **not** promoted by this ADR"*. §32 re-retrieved both sentences in their own context and both
satisfy all four conditions; the conjunction promotes exactly those two and, run across the whole
scoping tier, promotes **nothing else** — `4369/tcp` epmd is refused on condition 1, RabbitMQ being a
non-owner for that port, and `2049/tcp` NFS on condition 4, RFC 7530 §1.2's Internet sentence being a
goal RFC 8881 records as unmet. ~~Run in the demoting direction the conjunction refuses **none** of the
fourteen sitting members.~~ **The tiers read ~~prohibition 16 · scoping 11 · weak 3 · outside-subject 11,
coverage 30 of 41~~ — two cells, no rows, and ADR-0008 is not triggered.**

> **Both struck clauses are SUPERSEDED by [#107](https://github.com/winniel123/verge-asm/issues/107).**
> This amendment ran the demoting direction as a **re-reading of §18.5's ratification**, not as a
> retrieval — `sensitive-ports.md` §32.12 recorded the dependency against itself in terms. #107 walked
> limbs 2 and 4 **per row over all sixteen members**, and **[measured] one fails**: `1433/tcp` MS SQL,
> on [ADR-0050](./0050-an-owners-category-statement-reaches-the-members-its-own-artefacts-place-inside-it.md)
> limb 3 and independently on limb 4's third branch, because the carrying page numbers no port and
> Microsoft elsewhere names internet-facing TCP/1433 SQL Server as a supported, portal-provisioned
> option. **The conjunction refuses one of sixteen, and the tiers read prohibition 15 · scoping 11 ·
> weak 3 · outside-subject 11 · uncovered-in-subject 1, coverage 29 of 41.** Still no rows, and
> ADR-0008 is still not triggered. `sensitive-ports.md` §33.

**What this amendment does not reach, stated so its extent is in the artefact.** The closure claim is a
**ruling**: it holds that the row's proposition has four terms and that a reader-supplied premise can
only be needed for a term. It is modelled on `sensitive-ports.md` §10.2's closure of the claim set by
construction, and it is falsified by exhibiting a **fifth kind** of gap rather than a new instance of
one of the four. §32.12 names the standing candidate — **direction**, inbound versus outbound — and
~~records that no member of either tier turns on it today~~.

> **The struck clause is WITHDRAWN — twice over — and the extent statement is DISCHARGED by the
> [#110](https://github.com/winniel123/verge-asm/issues/110) amendment below.** §33
> ([#107](https://github.com/winniel123/verge-asm/issues/107)) withdrew the clause in the note on
> `445/tcp` and **left this recital of it standing here**, which is the ADR-0058 defect #106 widened
> the rule to catch; it is struck at its own clause now. **[measured]** the clause was in any case
> **false when written**: §28.9 had spared `623/udp` from a candidate defeater — HPE's *iLO Direct
> Connect*, which is outbound — four sections earlier. **And the closure survives the candidate.**
> Direction introduces no term the proposition lacks; it fixes which of conditions (2) and (3)'s named
> objects fills which place of the directed relation *reachable from*, so it is repaired **inside
> condition (2)** and the conjunction stays at **four**. `sensitive-ports.md` §36.4, §36.7.

## Confirmed by use — [#100](https://github.com/winniel123/verge-asm/issues/100), 2026-08-14

**Limb 2 is read one column over, and it holds.** #100 asked whether the **artefact class** carrying a
footing — a doc comment in a published config API, against prose on a documentation site — changes the
tier as well as admissibility. **It does not, and limb 1 is why:** the tier counts premises the reader
supplies between the owner's **sentence** and the row's proposition, and **the artefact carrying the
sentence is not a premise**. A reader who has the sentence has it. *Where it was written* is the same kind
of fact as *how firmly it was written* — a fact about the **utterance**, not about the gap between the
utterance and the proposition — so limb 2's refusal reaches it without being widened.

**[measured] a class discount applied honestly moves two cells nobody asked to move**: `9042/tcp`
Cassandra's **prohibition**-tier footing is a comment in a YAML file inside a source tarball, and
`5984/tcp` CouchDB's is a line in `rel/overlay/etc/default.ini` — the artefact
[`sensitive-ports.md`](../research/sensitive-ports.md) §12.5 calls *"the cleanest instance of §2.2's
third form in the note"*. That is the same shape as this ADR's own *a criterion that stretches to fit its
one counterexample takes its neighbours with it*, met in the artefact dimension.

**What artefact class does bear on is volatility, which is a different column** —
[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §8's watch list, where a
default changeable in one commit with no release note is a different exposure from one on a released
documentation page. #95's amendment to ADR-0032 §8 already said so for `10248/tcp`. That observation is
correct where it sits and is not a reason to move a tier; it gives the map's open *should the watch list
key on tier or on volatility* patch a third candidate axis and is reported rather than acted on
(`sensitive-ports.md` §31.7, §31.12).

**Limb 3's *necessary and not sufficient* structure is inherited by a second instrument.**
[ADR-0061](./0061-a-comment-is-a-position-only-where-it-outlives-the-value-it-annotates.md)'s survival
test separates a **label** from a **candidate position** and expressly does not promote the survivor;
§2.3's and §4.4's position-versus-preference discrimination runs second, exactly as this ADR's lexical
test leaves the sufficient condition unstated. **[measured]** the kubelet's `readOnlyPort` comment
survives its value and is still refused — the one instance on that side.

**No cell moves, and this ADR's rule is unchanged.**

## Confirmed by use — [#107](https://github.com/winniel123/verge-asm/issues/107), 2026-08-14

**The four-limb conjunction was run as a retrieval over all sixteen prohibition-tier members, one row
at a time. Fifteen hold. One does not, and this ADR's own limb 2 is what refuses the objection to
demoting it.**

**Limb 2's inadmissibility of mood and force is load-bearing in the demoting direction for the first
time.** Microsoft's Azure VM page hedges its internet-facing option — *"this doesn't imply that anyone
can connect to your SQL Server instance. Outside clients have to use the correct username and
password"* — and a reader will reach for that hedge to save the cell. **It is a fact about the
utterance, not about the gap between the utterance and the proposition**, so limb 2 refuses it in the
demoting direction exactly as it refuses Redis's *"usually it is not a good idea"* in the admitting
one. The column still does not measure how much the owner minds.

**Limb 1's premise count is untouched and was not needed for the verdict.** `1433/tcp` does not move
from zero premises to one — it stops being **evidence of the row's proposition at all**, which is the
disposition the #101 amendment's own limb-4 sentence specifies. A footing failing limbs 1, 2 or 4 does
not enter the grading, so the cell leaves the table rather than descending a tier. **[measured]** MS
SQL ships no configuration artefact (`sensitive-ports.md` §13.1), so no weaker form catches it and the
row becomes the table's only uncovered in-subject member.

**[measured] The demotion takes no neighbour, which is the test this ADR set for itself at §30.5 and
#101 re-set at §32.3.** Redis, Elastic, memcached and Oracle all document managed or public offerings
and all four are **spared on an artefact rather than on judgement** — Redis Cloud endpoints are on
`10000-19999` and `6379` occurs zero times in the owner's port table; Elastic's managed `9200` is a
**proxy** port with nodes on `18000-19999`; memcached operates no service; MySQL's class is conditioned
on *"untrusted hosts"*. **The discriminator is that the architecture the owner supports must be an
instance of the class the owner's own statement forbids**, and it is `sensitive-ports.md` §33.2's act
rather than this ADR's.

**One cell moves. The rule is unchanged in every limb.**

## Amendment — [#110](https://github.com/winniel123/verge-asm/issues/110), 2026-08-14: condition (2) names the endpoint being reached, and the conjunction stays at four

> **Direction is not a fifth limb. It is what condition (2)'s *"this `(port, transport)` pair"*
> denotes, and condition (2) is restated — at its own clause above as well as here, per
> [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened
> by [#106](https://github.com/winniel123/verge-asm/issues/106).**
>
> **(2) Reach, restated.** The statement is about **this `(port, transport)` pair as the endpoint being
> reached, on the estate the statement addresses** — a **listener**, reached from outside that estate.
> Reach is established exactly as before: either the owner numbers the pair, or
> [ADR-0050](./0050-an-owners-category-statement-reaches-the-members-its-own-artefacts-place-inside-it.md)'s
> three limbs carry it there on the owner's own artefacts. Two consequences, both of which the note has
> been applying unstated since §3.4: a statement about traffic **leaving** that estate on the same
> number is about the other end of the connection and satisfies this condition no more than a statement
> about a different port would; and a listener the **owner itself** runs, which the addressed party
> does not, is a different endpoint from the row's subject, so a statement about it neither carries
> this condition nor defeats it.
>
> **Addressee is not a sixth condition.** It is condition (2)'s second coordinate — *whose* endpoint,
> beside *which end* of the relation — and at the one member where it is load-bearing it is the same
> disposal as direction read from the other end of one TCP connection.
>
> **The four-limb closure is unchanged and is confirmed rather than weakened.** A reader-supplied
> premise can only be needed for a **term**, and direction introduces no term: *reachable from* is a
> directed relation with two argument places, condition (2) names the endpoint reached and condition
> (3) the vantage reached from, and direction is the **assignment** of those two named objects to those
> two places. A candidate introducing an object no condition reaches would still falsify the closure.
>
> **Limb 1's premise-count mechanism and limb 2 are untouched**, as #101 left them. Mood, force,
> hedging and priority label remain inadmissible in both directions.

**What forced it, and it is a retrieval rather than a tidying.** **[measured]** `sensitive-ports.md`
§32.12 wrote, as a hypothetical, that *"an owner sentence forbidding **outbound** traffic to the
internet would satisfy all four limbs as written while entailing nothing about the row"*. It is not
hypothetical. The document carrying `445/tcp`'s footing — *Secure SMB Traffic in Windows Server*,
`ms.date` **2024-10-25**, commit `00769866`, `word_count` 1602, retrieved 2026-08-14 — has a section
headed **`## Block outbound SMB access`**: *"Block TCP port 445 outbound to the internet at your
corporate firewall."* Microsoft's, numbering the pair, naming the internet, an unhedged imperative.
**Four conditions as written, satisfied; nothing entailed about the row.** The conjunction as written
was defective and the defect is measured rather than argued.

**Why it is repaired here and not only in the note.** ADR-0058's test is whether the superseded
sentence, *read alone and in the present tense*, would cause a competent session to act on it.
*"(2) about **this `(port, transport)` pair**"* read alone admits the outbound sentence, which is a
promotion `sensitive-ports.md` §36 refuses — and the #101 amendment's own extent paragraph was still
reciting §32.12's *"no member of either tier turns on direction today"* after §33 had withdrawn it in
the note, which is the intra-document defect #106 widened the rule to catch, on this file, one section
after it was widened. Both are struck at their own clauses. **This is also why no second ADR was
minted:** ADR-0068 was reserved and is **left unused**, as `0039`, `0041`, `0052`, `0053`, `0057`,
`0062`, `0063` and `0064` are.

**[measured] and nothing moves.** Direction and addressee were run as candidate conditions across all
**26** members of both graded tiers. The owner supplies the direction in **24** of the 26 carrying
statements — **six in the word**, three in the carrying verb's preposition, fifteen in a verb whose
object is a listener — and it is absent from the two, `2049/tcp` and `4369/tcp`, that already fail
condition (3). **Two owners write the direction in a table column literally headed `Direction`**:
Kubernetes for `10250`/`10259`/`10257`, and **HPE for `623/udp`** — *"`IPMI/DCMI over LAN port | 623 |
UDP | Inbound⁴`"*, footnote 4 *"An external client initiates the connection to iLO"*, against
*"`Remote support port | 7906 | TCP | Outbound¹`"*, footnote 1 *"iLO initiates the connection to an
external server"* (*iLO 6 User Guide*, part number 30-7A345B12-032, **July 2026**). **Two verdicts
turn on direction** — `445/tcp` (§33.5) and `623/udp` (§28.9, which no prior section counted) — **and
one on addressee**, `445/tcp`. **[measured]** the one case where addressee comes apart from direction
cleanly — MongoDB Atlas, an inbound internet-reachable listener on `27017` that MongoDB scopes away
from its own trusted-networks guidance by page title — is **not** load-bearing, `27017` failing
condition (3) a limb earlier. **No cell moves, no row moves, and the cost is nothing.**
`sensitive-ports.md` §36.

## Alternatives rejected

**Restate the criterion to read on the imperative and its scope — *a prohibition-tier sentence forbids
deployment beyond a trust boundary the owner names*.** The strongest losing option, and the one #98
put first. It admits `873` on its face, leaves the other thirteen where they are, and moves nothing.
**It loses on limb 2.** *Imperative* is a mood, and mood is force wearing grammar's clothes: the
restatement promotes rsync's *"Do not expose"* over Redis's *"usually it is not a good idea"* and can
only do so by ranking how firmly each owner speaks. That is the ranking ADR-0042 refused by name and
that ADR-0004 and §2 exist to keep out, arriving through the one column nobody had fenced — the same
shape as ADR-0042 limb 3's frequency leak. It loses a second time on **extension**: applied honestly it
also promotes `4369` epmd, whose `DEP-001` is titled *Do Not Expose Default Erlang Distribution on
Untrusted Networks*, carried at `Critical`, and advises disabling the daemon **unconditionally** — a
sentence at least as imperative as rsync's and scoped to a boundary Erlang/OTP names. §20.8 argued that
promotion at length and refused it; the restatement reverses it as a side effect. **A criterion that
stretches to fit its one counterexample takes its neighbours with it**, and this one is measured doing
so.

**Refuse both and leave §20.8's sentence standing as an approximation, marked as such.** Cheapest, and
it survives because the tier moves nothing downstream. **It loses because the note's own text would then
assert a universal its own table refutes** — the defect §2 exists to prevent, with a known repair
costing one cell and no rows. #93 already declined it once for a good reason that has now expired: it
was not that pass's question. It is this one's.

**Move `873` and mint no ADR** — record the move as a §-level ruling, as §16.6, §20.9, §23, §24, #90 and
#91 all did. It is the house default and it nearly wins. **It loses on the reason the ticket exists.**
Every one of those sections declined an ADR because *"both general rules this section applies were
available"*; here the load-bearing rule — **what the tier grades** — was available nowhere, which is why
two sections reconstructed it in two vocabularies and a third had to report the note contradicting
itself. ADR-0042 refused *"leave it unwritten"* on exactly this ground, and a reader who finds rsync's
flat imperative filed below Redis's hedge should be able to read a decision rather than conclude one of
them is a mistake.
