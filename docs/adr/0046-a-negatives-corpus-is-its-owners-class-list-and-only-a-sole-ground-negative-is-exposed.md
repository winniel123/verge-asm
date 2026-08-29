# ADR-0046: A negative's corpus is its owner's class list, and only a sole-ground negative is exposed

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#79 Which curated-table negatives were established over specifications alone, and does ADR-0040 reopen any?](https://github.com/winniel123/verge-asm/issues/79)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0040](./0040-a-specifications-silence-is-not-the-owners-silence.md) ruled that **a specification's
silence is not the owner's silence**, and required that a negative about an owner enumerate the
owner's **document classes** rather than its documents. It was measured on a failure:
[#68](https://github.com/winniel123/verge-asm/issues/68) read five documents, every one a
specification or a method, and recorded that the IETF sets no certificate key-size floor.
[#73](https://github.com/winniel123/verge-asm/issues/73) found three, one of them in an appendix of a
document #68 had cited by name.

Every negative load-bearing in the two curated tables was established before that rule existed. This
ticket asked which of them were established over specifications alone, whether re-asking the other
classes reopens any, and — the question the map's curation patch has been carrying since
[#66](https://github.com/winniel123/verge-asm/issues/66) — **whether the sweep has an end.**

The full working, with the enumerated corpus and every quote checked against retrieved bytes, is
[`sensitive-ports.md`](../research/sensitive-ports.md) §17 and
[`weak-key-and-signature.md`](../research/weak-key-and-signature.md) §14.

## Decision

| Concern | Decision |
|---|---|
| How many negatives are exposed | **Ten of the sixteen load-bearing negatives in the two tables**, and the filter that produces ten is the rule below. The other six cannot move on any document, because a gate no document reaches already refuses the row. Of the ten: **seven were searched**, **one had its class list exhausted before this ticket**, and **two are the named residue** |
| The exposure test | **A negative is exposed to ADR-0040's class sweep only where it is the row's *sole* ground.** An **overdetermined** negative — one sitting beside a determinacy failure, a closed-claim-set failure, or an owner sentence pointing the other way — is **bounded on arrival** and is swept for the record, never for the verdict |
| Does the sweep have an end | **Yes, and the end is countable in advance.** The corpus of a negative is its **owner's class list**, and the class count is a property of **the owner**, not of the subject: a standards body has three, a single-project owner usually two, a project with a reference implementation three. There is no open set to exhaust |
| Does any row reopen | **No row moves. The list stays at 37 pairs and the weak-key table at five rows.** Two negatives came back **stronger** than they went in, one came back with a sentence that is **not shipped**, and one was measured for the first time |
| The `161/udp` question, answered directly | **It does not re-open.** The IETF's operational class — RFC 3512, RFC 3871, RFC 4778, none of them opened by #66 — contains no placement sentence, and RFC 3871 contains the opposite: *"There are many situations where in-band management makes sense, is used, and/or is the only option."* #66's negative is now **bounded** rather than merely unsearched |
| The new thing the sweep found | **An owner's document can exist and not be shipped.** Apache Kafka's `docs/security/security-model.md` carries the prohibition-shaped sentence `sensitive-ports.md` §4.6 says does not exist — and it is absent from release tag `4.3.1` and 404s on the published documentation site. §12 ruled what an **example configuration** attests; nothing rules on **unreleased prose**. Routed, not decided |
| What a breadth weakness is | **Not a class weakness, and ADR-0040 does not reach it.** #67's *no second CA was retrieved stating a fraction* is a corpus of one **instance** of a class that was searched. Enumerating classes cannot cure it, and calling it a class weakness would make ADR-0040's sweep unbounded |
| Is a re-admission a reversal | **No — [ADR-0009](./0009-verge-core-is-a-union.md)'s removal plus addition, priced separately**, as §11.8 already wrote for `161/udp`. Nothing here re-admits anything, and the rule is restated because the first row that reopens will be tempted to call itself a correction |

## Rationale

### 1. The population is sixteen and six of them cannot move, and the arithmetic is the finding

ADR-0040's disclosure rule binds this ticket's own output, so the population is listed rather than
asserted: `sensitive-ports.md` §17.1 tabulates fourteen load-bearing negatives with the classes
searched and the classes that exist, and `weak-key-and-signature.md` §14.1 tabulates two more.

Six of the sixteen are **overdetermined**, and that is not luck. It is the shape of an evidence
standard with more than one gate. `111/tcp` has no owner sentence *and* is refused at
[§10.1](../research/sensitive-ports.md) Step 1, because the operations answerable anonymously are the
ones the specification exists to answer. `79/tcp` has no implementation position *and* fails
determinacy against RFC 4146. `389/tcp` has RFC 4513's silence about `ldaps://` *and* OpenLDAP naming
*"the global Internet"* as an intended environment. In each case, an owner sentence retrieved
tomorrow from a class nobody opened would change the note's footing and not its verdict.

> **A gate a document cannot reach makes the negative beside it un-exposed.** Determinacy is a fact
> about the registry and about competing conventions; the closed claim set is a fact about what an
> internet vantage supplies; an owner sentence naming the internet as supported is a positive. None
> of the three is a silence, so none of the three has a class list.

**This is what makes the sweep affordable, and it is the reason to state it as a rule rather than to
note it in passing.** Read without the filter, ADR-0040 obliges a session to enumerate three document
classes for every negative in every curated table, forever. Read with it, six of sixteen fall away
before any retrieval is planned, one more turns out to have had its class list exhausted already, and
seven retrievals remain — of which **one** returned a sentence. The filter is not the difference
between a big job and a small one. It is the difference between an obligation with an end and a
standing one.

**And the filter is a *rule* rather than a heuristic**, because it is falsifiable in the direction that
matters. A reader who thinks a negative was wrongly called un-exposed has a specific thing to attack:
the second gate. §17.1's *Sole ground?* column names it in every row, so the disagreement is about a
stated proposition rather than about how hard someone looked.

### 2. The corpus is bounded because a class list is a property of the owner

ADR-0040 §3 established that a scope weakness is tractable because *"the set of bodies that can
resolve one is small and enumerable rather than open."* The same argument runs one level down and it
is what answers #66's backlog question.

| Owner shape | Classes | Instance measured here |
|---|---|---|
| **Standards body** | specification · deployment or operational recommendation · implementation guidance | IETF: RFC 3411 (spec) · RFC 3871, RFC 3512 (operational) · net-snmp (implementation) |
| **Single project, documentation only** | its manual · its shipped bytes | PostgreSQL: the manual and `postgresql.conf.sample`/`pg_hba.conf.sample`. **Both were already open** |
| **Project with a reference implementation** | its manual · its shipped bytes · the implementation's own prose | Apache Kafka: `security-overview.md` · `server.properties` · `security-model.md` |
| **Vendor** | product documentation · deployment guidance, usually cloud-scoped | Microsoft: `winrm-security` · Azure best practices |

The list is short, it is fixed before the search starts, and **an owner cannot invent a fourth class
to defeat a sweep**. That is the difference between this and a general *somebody may have said this
somewhere*, and it is why ADR-0040's *bounded* rather than *permanent* is available here at all.

> **Read *fixed* as fixing the TAXONOMY, never the MEMBERSHIP** — the
> [#93](https://github.com/winniel123/verge-asm/issues/93) amendment below, which says in terms that
> this phrase *"could be read as making a finished sweep finished for all time"* and then left it
> readable that way. An owner's class list is **fixed at the time of the sweep, not for all time**; a
> class sweep's output is a **dated** negative, disclosed as *the class had no member at release R,
> dated D*. Marked at the sentence per
> [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened
> by [#106](https://github.com/winniel123/verge-asm/issues/106).

**The measurement that makes the point cheapest.** PostgreSQL is the note's weakest row and its
negative is the strongest-sounding in the corpus — *upstream states no position at all on network
placement*. It has **two** classes and #70 read both. There is nothing left to open, so §4.5's
disclosure was already complete before ADR-0040 existed. It just could not say so.

### 3. What the sweep actually returned, and most of it is not what a sweep is supposed to return

Recorded because the yield pattern is more useful than any single result.

- **`161/udp` — survives, and the operational class points the other way.** Three IETF operational
  documents were opened for the first time. RFC 3512 *Configuring Networks and Devices with SNMP* is
  the SNMP family's own deployment guide and its security sections are about community strings,
  notification defaults and MIB object sensitivity — no placement sentence. RFC 3871's SNMP paragraph
  is the ubiquitous boilerplate *"deployment of SNMP versions prior to SNMPv3 is NOT RECOMMENDED"*,
  which is a **version** statement whose remedy is SNMPv3 **on 161** — §11.5's structure exactly, and
  refused for §11.5's reason. And RFC 3871 §2.2 says *"There are many situations where in-band
  management makes sense, is used, and/or is the only option."* This is the
  [#30](https://github.com/winniel123/verge-asm/issues/30) shape a third time: the negative moves from
  *nobody looked* to *evidence found pointing the other way*.
- **RFC 4778 is frequency wearing a floor's clothes, and the ticket predicted it.** *Operational
  Security Current Practices in Internet Service Provider Environments* is the most placement-shaped
  title in the corpus and it is a **survey**: *"In all large ISPs that were interviewed"*, and
  *"SNMPv2 is primarily deployed since it is easier to set up than v3."* It reports what operators do.
  #73 met this in RFC 8422 §5.1.1 and [#75](https://github.com/winniel123/verge-asm/issues/75) refused
  it again on `7000`. It is met here in the one document class most likely to contain it, which is
  now the recorded reason to expect it there.
- **`5672`/`15672` — survives, and it strengthens.** RabbitMQ's production checklist, the deployment
  class, was never opened. It divides the ports into client-library ports and everything else, and
  says of the first category that they *"should be accessible to hosts that run applications, which in
  some cases can mean **public networks**, for example, behind a load balancer."* §4.6's negative — *the
  prohibition names 4369 and 25672, not these* — becomes an owner naming public networks as supported
  for exactly the two ports at issue, which is §10.3's failure condition met from the other side.
- **MD4's exclusion was measured for the first time.** #68 cited RFC 6149, which is about **MD2**.
  **RFC 6150, the MD4 document, was never retrieved by anyone.** It has now been read end to end and
  it does not carry a certificate prohibition: its strongest sentence is *"It cannot be used as a
  one-way function. For example, it must not be used to hash a cryptographic key of 80 bits or
  longer"* — the right modality on the wrong artefact, which is ADR-0035's artefact test failing in
  #73's *right number in the wrong sentence* shape. Its recommendation is *"implementations should
  strongly consider removing support"*, and it opens with a frequency clause, *"Despite MD4 seeing
  some deployment on the Internet"*.
- **`10255/tcp` — the fourteenth negative, created while the sweep was running, and the one that
  returned a sentence.** [#76](https://github.com/winniel123/verge-asm/issues/76) moved the kubelet
  read-only port into the weak tier on a restricting default alone, recording that *"the owner states
  no position on `10255` anywhere that was retrieved"* over Kubernetes' ports reference and its
  authn/authz page. The class those two do not belong to is Kubernetes' **security documentation**, and
  its checklist says *"The Kubernetes API, kubelet API and etcd are not exposed publicly on Internet"*
  and *"The kubelet API access should be restricted and not exposed publicly."* That is the owner
  naming a boundary about its own component — **by category rather than by port**, which is a question
  §2.3 has never had to answer because every previous category statement in the corpus came from a
  **non-owner**. It is reported to the curator rather than applied, because it moves a footing cell in
  another pass's table (`sensitive-ports.md` §17.6).

### 4. The one that did not survive its sweep, and why it still does not move a row

Apache Kafka's `docs/security/security-model.md` says, in the owner's own words:

> "**Security is off by default.** A freshly-installed Apache Kafka cluster accepts unauthenticated
> `PLAINTEXT` connections on every listener and applies no authorization. This is appropriate only for
> closed test environments. Production deployments **must** explicitly configure authentication,
> authorization, and transport encryption before being exposed to any untrusted network."

`sensitive-ports.md` §4.6 excludes `9092/tcp` on *"Upstream declines to take any network posture. Its
only relevant sentence is neutral"*, and §10.3 and §12.6 each re-confirmed it on that ground. The
sentence above is not neutral, and finding it is a **retrieval** on ADR-0040 §4's test: the claim is
about a different document's **content**, not about the **import** of one already held.

**It does not admit the row, on two independent grounds, and the second is new.**

1. **ADR-0037 limb 2 governs.** *A subject the artefact names and the table lacks is a finding, and it
   is ticketed rather than admitted* — the artefact supplies the **attestation limb only**, and the
   claim and determinacy gates were not what this retrieval was scoped to answer. `7000/tcp` is the
   precedent and it is the same shape: the strongest sentence in the corpus, and a different gate
   sank it.
2. **[measured] The sentence is not shipped.** `docs/security/security-model.md` is present on
   `apache/kafka` `trunk`, **absent from release tag `4.3.1`**, and
   `kafka.apache.org/documentation/security/security-model/` returns **404**. The document that *is*
   at `4.3.1` and *is* published is `security-overview.md`, still carrying the neutral sentence §4.6
   quotes, verified against retrieved bytes.

The second ground is a gap in the standard rather than an application of it. §12 ruled what an
**example configuration** attests — nothing, in either direction — and §2.2's second form reads *"the
project's or vendor's own documentation"* without saying whether that means what the project
**publishes** or what is in its **tree**. The two answers give opposite verdicts here, and the
question has never been asked because until now no case turned on it.

> **Under the standard as it stands, `9092/tcp` stays excluded and the list is definitively 37.** The
> new document is unreleased and unpublished, so it does not reach §2.2's second form on the reading
> §12 already committed the note to for configuration — operativeness is read off what ships. The
> ticket that follows asks whether prose should be read the same way. **It is not a blocker for
> [#12](https://github.com/winniel123/verge-asm/issues/12)**: the count is not in doubt under the
> current standard, and if the standard changes, the row is a new admission priced then, under
> ADR-0009.

### 5. Breadth is not class, and the distinction is what keeps ADR-0040 finite

[#67](https://github.com/winniel123/verge-asm/issues/67)'s negatives are the ones the ticket expected
to survive, and they do — but the reason is worth separating from the result.

`acme-renewal-timing.md` §3's *RFC 8555 says nothing about renewal timing* is a specification-class
negative of exactly #68's shape. It is harmless because the retrieval **did not stop there**: RFC 9773
(specification), the Let's Encrypt Integration Guide (issuer documentation), `boulder` (issuer code)
and the Certbot user guide (reference client) are four more classes, and three of them came back
positive. The row rests on the positives, so the negative carries nothing.

§14's other disclosure is different in kind: *"No second CA was retrieved stating a different fraction,
and none was retrieved stating the same one."* That is a corpus of **one instance** of a class that
was searched — not an unopened class.

> **A weakness of breadth is not a weakness of class, and ADR-0040 does not reach it.** Enumerating
> document classes cures a negative that asked the wrong *kind* of document. It cannot cure a negative
> that asked the right kind and only asked one party, because the set of parties is open where the set
> of classes is closed. Conflating them would make ADR-0040's sweep unbounded, which is the property
> ADR-0040 exists to deny.

## Consequences

- **No `(port, transport)` pair moves and no row of the weak-key table moves.** `sensitive-ports.md`
  stays at **37 pairs**. §3's class totals (12 / 7 / 18) are unchanged. §2.2's footing table as
  [#76](https://github.com/winniel123/verge-asm/issues/76)'s §16.7 restated it is untouched in every
  cell by this ADR, and its denominator stays **37**. `weak-key-and-signature.md` stays at **five
  rows**. No `Break` ([ADR-0008](./0008-derivation-versions-move-on-content.md)), no aperture change,
  no amendment to ADR-0009's union.
- **One footing cell has a candidate and it is not moved here.** `10255/tcp`'s promotion out of the
  weak tier rests on whether an owner's **category** statement reaches a port the owner has not
  numbered, which is a question about §2.2 and §2.3 rather than about kubelet, and the cell belongs to
  #76's pass. Reported to the curator. `sensitive-ports.md` §17.6 carries the measurement and the
  counter-argument.
- **#66's backlog question is answered: for these two tables, the class sweep is finished — as of a
  stated table state.** The map's curation patch has carried *a row's footing was never checked against
  the standard it is now held to* as *a backlog with an end* since #66, and the end is here. What
  remains is named rather than left open, and it belongs to a **different gate**.
- **The qualifier is load-bearing, and it earned itself during this ticket.** The population was fixed
  at thirteen, and #76's §16.5 moved `10255/tcp` into the weak tier mid-sweep on a restricting default
  alone — manufacturing a fourteenth sole-ground negative out of a row that had not had one, and it was
  the one that returned a sentence.

  > **A class sweep is complete as of a table state, and a *footing* move re-arms it for exactly the row
  > that moved.** The backlog's end is a **fixed point** rather than a date: it is finished when no
  > row's footing has moved since the last sweep. That is cheap to check, because the footing table
  > names the tiers — and it is a **different obligation** from the watch the map's curation patch
  > already prices, which fires when the **world** moves. This one fires when **we** move, and it had
  > never been named.
- **The residue is two rows and it is a determinacy question, not a class one.** `5601/tcp` Kibana and
  `8500/tcp` Consul are excluded partly on owner-silence and partly on a **squat** — `esmagent` and
  `fmtp` — and [ADR-0042](./0042-a-squat-is-contested-where-the-other-convention-is-live.md) made
  *contested* testable by keying it to the competing owner's current documentation, and
  [ADR-0048](./0048-a-convention-is-evidenced-by-placement-never-by-catalogue.md) has just supplied the
  source rule for it. **[measured] Nobody has run that test on either registration** — #82's §15.4
  walked the nine **listed** rows resting on convention and did not reach the **excluded** ones. That is
  §2.4's backlog, it is the smallest extension of this ticket's corpus that could change an answer, and
  it is named here so the residue is falsifiable rather than decorative.
- **ADR-0037 limb 1 pays out a second time, on a table rather than a file.** RFC 9325, RFC 9846 and
  RFC 8551 were all held by #73 and all read for **key sizes**. Read for the rest of the table's
  domain, the strings `MD2` and `MD4` occur in them **zero times** — which converts #68's MD2/MD4
  exclusion from an inference into a measurement without any new retrieval. This is the second
  instance of limb 1 finding something in an artefact already held. ADR-0037's own criterion for
  forcing a general re-read named *a second unlisted subject found beside a listed one* as the trigger,
  and this is a **confirmation** rather than that trigger, because nothing was found.
- **[ADR-0040](./0040-a-specifications-silence-is-not-the-owners-silence.md) is confirmed and
  narrowed.** Its class rule held on its first application beyond the table that produced it — three
  classes, three notes, seven searches, one hit and two negatives that strengthened. What is added is
  the **exposure test** that keeps it from being a standing obligation, the **breadth/class
  distinction** that keeps its corpus closed, and the **table-state qualifier** that says when the
  obligation recurs.
- **A question the standard has never been asked is now on the record**, with the first case that
  turns on it: *is an owner's unreleased document the owner's documentation?* Ticketed. Until it is
  answered, `9092/tcp`'s exclusion is disclosed as resting on an artefact test rather than on the
  absence of a sentence — which is a materially different footing from the one §4.6 records, and the
  note says so.
- **`CONTEXT.md` is not edited**, on ADR-0032's, ADR-0035's and ADR-0040's precedent and for their
  reason. Nothing here changes a glossary term.
- **No new owner is admitted anywhere.** Every party quoted in this sweep — the IETF, RabbitMQ,
  Apache Kafka, CouchDB — was already an owner under §10.5 for the artefact it is quoted about.

## Alternatives rejected

| Alternative | Why not |
|---|---|
| **Sweep every negative's three classes, without the sole-ground filter** | It is the honest-looking option and it is refused on measurement: eleven of the fifteen negatives sit beside a gate no document can reach, so eleven of fifteen retrievals were guaranteed to be verdict-neutral before they were run. An obligation whose expected yield is zero is how a disclosure rule decays into the ceremony ADR-0032 forbids. The classes are still **enumerated** for all fifteen — that part is cheap and it is what ADR-0040 actually requires — but only four were **searched** |
| **Admit `9092/tcp` on the sentence found** | Two independent refusals. ADR-0037 limb 2 says an attestation retrieval answers one gate, and `7000/tcp` is the measured precedent for what happens when a session sweeps the rest in with it. And the sentence is not in the release or on the site, so admitting it would decide the unreleased-document question silently, in the direction that happens to add a row — the worst way to settle a standard |
| **Refuse `9092/tcp` outright and close the question, on the ground that the file is unreleased** | Symmetrically wrong for the same reason. §2.2's second form does not say whether *documentation* means published or committed, §12 answered it only for configuration, and the answer that reaches for the nearest analogy without stating it is the failure ADR-0036 was written about. The verdict today is *excluded*, and the reason is disclosed as provisional |
| **Block [#12](https://github.com/winniel123/verge-asm/issues/12) on the new ticket, on ADR-0037's precedent for `7000`** | ADR-0037 blocked #12 because the count was genuinely in doubt: two subjects had passed the attestation gate on **shipped** bytes and only determinacy stood between them and the list. Here the count is **not** in doubt under the standard as written — the artefact fails §2.2 before any gate is reached. Blocking would price a standards question as a count question and would delay the spec for a row that cannot be admitted without a rule change |
| **Record the surviving negatives as *permanent*** | ADR-0040 forbids it and it would be false. Each survivor names the classes searched and the class list is the owner's, so any reader can falsify it by naming a document of a class not in the list — which is the property *permanent* lacks |
| **Treat #67's one-issuer corpus as a class weakness and open a retrieval on other CAs** | It is a breadth weakness. The class — issuer documentation — was searched, and searching it again at a second issuer tests **generality**, not **completeness**. Folding the two together would give ADR-0040's corpus no boundary, since there is always another party. §14's disclosure already states it correctly and needs no change |
| **Fold this into ADR-0040 as an amendment** | ADR-0040's subject is *where to look before recording that an owner is silent*. This ADR's is *which negatives that obligation applies to, and how the resulting backlog terminates*. The second is a rule about the **scheduling** of the first and reaches the map's curation patch, which ADR-0040 explicitly left carrying an open cadence question |
| **Re-admit `161/udp` on RFC 3871's `NOT RECOMMENDED` boilerplate** | It is a statement about the **version**, not the **placement**, and its remedy — SNMPv3 — is reached on `161/udp` itself. §11.5 already refused this exact structure for RFC 6353's 10161, and RFC 3410 §8.2's *"framework of choice"* is the same sentence from the same body nine years earlier, which #66 held and correctly declined. Admitting it now would be re-reading held text with a new document stapled to it |

## Amendment — [#93](https://github.com/winniel123/verge-asm/issues/93), 2026-08-14

Limb 2 reads *"the list is short, it is **fixed before the search starts**, and an owner cannot invent
a fourth class to defeat a sweep."* [#84](https://github.com/winniel123/verge-asm/issues/84) supplies
the case that sentence does not name, and it is a measurement rather than an argument: **[measured]**
Erlang/OTP's `system/doc/design_principles/secure_coding.md` — the document that re-founded `4369/tcp`'s
footing from a class [#76](https://github.com/winniel123/verge-asm/issues/76)'s corpus did not contain —
is **absent at `OTP-28.4` and earlier and present at the tag `OTP-28.5`**, released **2026-04-23**, four
months before it was read. Before that release the owner's *Deployment / security guidance* class had
**no member**, and the sweep that found nothing there was correct.

> **Limb 2 closes the *taxonomy*, not the *membership*.** There are four kinds of document —
> specification, deployment or operational recommendation, implementation guidance, shipped default —
> and Ericsson invented no fifth; it published a **member** into a class that existed and was **empty**.
> A taxonomy is a property of the owner's *shape* and does not move. A class's **membership** is a
> property of the owner's *release*, and moves whenever the owner ships.
>
> **So an owner's class list is fixed at the time of the sweep, not for all time**, and a class sweep's
> output is a **dated** negative. The disclosure form a sweep owes is not *the class is empty* but **the
> class had no member at release R, dated D**.

This adds no machinery. [ADR-0045](./0045-an-owners-documentation-is-what-it-has-issued.md) already
rules that an owner's documentation is what it has **issued** and that **issuance is per version**, so
a negative *"names the release and the date and never says never issued"*. A class sweep is a search
over issued documents and inherits that rule unamended. What this amendment does is say so, because
limb 2's *fixed before the search starts* could be read as making a finished sweep finished for all
time — a claim `sensitive-ports.md` §17.8 never made.

**What discharges the recurrence is [ADR-0040](./0040-a-specifications-silence-is-not-the-owners-silence.md)'s
falsification clause, not a watch, and this is the load-bearing half of the amendment.** A dated class
list is *"falsifiable by naming one document outside the boundary"*, and §20.4 records the clause firing
for the first time: §16.9 stated its corpus, #84 named a document outside it, the residue was withdrawn.
The clause costs nothing until somebody actually holds a document, which is the only moment at which
anything can move.

**A ninth curation-watch trigger was argued and lost, on cost rather than on coverage.** The map's
curation patch carries *a drafted owner document becoming issued* ([#86](https://github.com/winniel123/verge-asm/issues/86))
as *"the cheapest, because it is **announced, by a release**"* — and that pricing holds only because the
**draft is already known**: Kafka's `security-model.md` sits on a branch with no tag, so the watch is
*check whether that one file's branch gets tagged*. **Nothing announced `secure_coding.md`.** It was
authored and released in a single motion, into a class that had never had a member, in a repository
nobody was watching. The corresponding watch is *read every release of every owner in the table, forever,
for a document that did not previously exist*, which is the *"somebody may have said this somewhere"*
standing obligation the Rationale above exists to deny. **A watch that cannot be run is worse than a
bounded disclosure, because it converts an unbounded obligation into a checked box.** The existing
trigger keeps its cheapness by keeping its scope, and *a row's footing was never checked against the
standard it is now held to* stays **a backlog with an end** — the end being §17.8's fixed point, which
this amendment does not touch.

**Thin ground.** The taxonomy/membership distinction is doing all the work and nobody had to draw it
before #84. A reader who holds that a class with no member is no class concludes the opposite — that
limb 2 **was** falsified and the taxonomy is open. That reading is refused because it makes the class
count depend on the subject's publication history rather than on the owner's shape, which is what the
table of owner shapes in §2 asserts. It is a ruling, not a measurement.

**A second, independent instance was measured while this was being decided**, and it makes the same
point without Erlang. `sensitive-ports.md` §26.4 enumerates NFS's two owners: **[measured]** the IETF's
**operational class is empty for NFS** — not one RFC in the family carries status `BEST CURRENT
PRACTICE` — and nfs-utils publishes **no deployment or security document at all**. Both classes exist in
the taxonomy above and neither has a member. **A class list is a set of slots, and slots can be empty at
a release and filled at the next one.**

**No `(port, transport)` pair moves, no row moves and no figure in either curated table changes**
(`sensitive-ports.md` §26.7). This amendment mints no ADR of its own, on the test §20.9 and §16.6 both
applied: the general rules it composes — limb 2 here, ADR-0045's per-version issuance, ADR-0040's
falsification clause — were all already available.
