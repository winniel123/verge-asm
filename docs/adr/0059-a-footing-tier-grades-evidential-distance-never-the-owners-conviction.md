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
   in the strongest tier where the owner's sentence entails that with **no** premise supplied by the
   reader; in the next tier where it entails it through **one** — characteristically *the internet lies
   outside the boundary the owner named*; and in the weakest tier where the owner has published no
   sentence at all and the footing is a restricting default.
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

## Rationale

**It is the rule the table has been applying, and the reconstruction reproduces every verdict.** That
is the test ADR-0042 set for itself and it is the test this has to pass. Read across the tier
assignments as they stand:

| Footing | Owner's sentence | Premises the reader supplies | Tier, and it is the existing one |
|---|---|---|---|
| `6379` Redis | *"not a good idea to expose the Redis instance directly to **the internet**"* | none | **Prohibition** — and it is **hedged**, which limb 2 says is irrelevant |
| `11211/tcp`+`/udp` memcached | *"you **must not** expose memcached directly to **the internet**"* | none | **Prohibition** |
| `3306` MySQL | *"Try to scan your ports **from the Internet** using a tool such as `nmap`. MySQL uses port 3306 by default. This port should not be accessible from untrusted hosts."* | none | **Prohibition** |
| `1433` MS SQL | *"don't connect your SQL Server instances directly to **the Internet**"* | none | **Prohibition** |
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

## Consequences

- **`873/tcp` rsync moves from the explicit prohibition tier to the explicit trusted-network scoping
  tier.** §2.2's tiers read **prohibition 14 · scoping 12 · weak 2**, coverage **28 of 39** unchanged,
  and 14 + 12 + 2 + 11 = 39. `sensitive-ports.md` §30.
- **No `(port, transport)` pair moves and no row moves.** The list stays at **39 pairs**, class totals
  `11 / 7 / 21`, §6.1's `28 + 6 + 5 = 39` and §4.6's 19 exclusions are untouched.
  [ADR-0009](./0009-verge-core-is-a-union.md)'s union is unchanged and
  [ADR-0008](./0008-derivation-versions-move-on-content.md) is **not** triggered — the rule's reference
  data is byte-identical. A tier records how strong a footing is, never whether a row qualifies.
- **§20.8's universal becomes true.** *"Every prohibition-tier sentence names the public internet"* is
  now a correct statement of a **fourteen**-member tier, made true by moving the counterexample rather
  than by widening the sentence. §20.8's `4369` ruling is **unchanged and re-founded**: `DEP-001` is one
  premise away, and its advisory mood — which §20.8 gave as a second reason — is withdrawn as a ground
  under limb 2.
- **[#12](https://github.com/winniel123/verge-asm/issues/12) is not touched.** The spec carries the
  list, the claims and the containment arithmetic; none of them reads a tier.
- **The weak tier, and therefore [ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md)
  §8's watch list, is unchanged at `5432/tcp` and `5984/tcp`.**
- **It does not generalise beyond a graded footing column**, and that is ADR-0032's ruling rather than a
  choice made here. [`weak-key-and-signature.md`](../research/weak-key-and-signature.md) has no footing
  tier and gains none.
- **The sufficient condition for the prohibition tier is still unstated**, and two scoping-tier rows
  now visibly bear on it. Routed, not answered.

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
