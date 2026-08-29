# ADR-0050: An owner's category statement reaches the members its own artefacts place inside it

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#88 Does an owner's category statement reach a port the owner has not numbered?](https://github.com/winniel123/verge-asm/issues/88)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[`sensitive-ports.md`](../research/sensitive-ports.md) §2.3 rules that a source attesting a
**category** rather than a port *"is therefore excellent corroboration and useless as the sole
grounds for any individual row"*. The rule was written against CISA and the cloud providers —
*"management interface"*, *"out-of-band interface"* — and **every category statement in the corpus
at the time came from a party that does not own the protocol**.

[#79](https://github.com/winniel123/verge-asm/issues/79) §17.6 produced the first one that does.
Sweeping Kubernetes' security-documentation class for `10255/tcp`'s negative, it retrieved:

> "- [ ] The Kubernetes API, kubelet API and etcd are not exposed publicly on Internet."

> "The kubelet API access should be restricted and not exposed publicly, the default authentication
> and authorization settings, when no configuration file specified with the `--config` flag, are
> permissive."

— both `content/en/docs/concepts/security/security-checklist.md`, `kubernetes/website`
`release-1.34`.

Neither sentence writes `10255`. Both name *the kubelet API*, which is a category — and Kubernetes
both **owns** the kubelet and **defines** what the kubelet API contains, which is precisely what
CISA does not do for *management interface*. #79 reported the finding rather than applying it and
named the question: **is §2.3's refusal about the grammar of a category statement, or about the
standing of the party making it?**

The full working is [`sensitive-ports.md`](../research/sensitive-ports.md) §18.

## Decision

| Concern | Decision |
|---|---|
| Does an owner's category statement reach an unnumbered member | **Yes — §2.3's refusal is about *standing*, never about *grammar*.** A statement by the protocol's owner about a category the owner defines reaches every member of that category, on three limbs, all three required |
| Limb 1 — **standing** | The speaker **owns the protocol** under §10.5: it designed the protocol or authors the reference implementation, speaking about the thing it designed or wrote. This is the limb CISA, the cloud providers, Red Hat and NSA fail, so **§2.3 is untouched** and no government or hyperscaler category statement gains an inch |
| Limb 2 — **membership** | **The owner's own artefact must place the member inside the category.** The mapping from category to `(port, transport)` may not be the reader's inference — it is a second owner statement, and the row is the concatenation of two. A number supplied by a **corroborator** does not close the gap, on §2.3's own terms |
| Limb 3 — **defeat, tested per member** | Reach **fails for any member whose internet-facing deployment the owner elsewhere names as supported** — §10.3's failure condition, applied per member rather than per sentence. One sentence may reach one member and be defeated for another. This is the sibling of [#76](https://github.com/winniel123/verge-asm/issues/76)'s *ownership is tested per port, not per sentence* |
| The unit of the category | **A protocol or an interface, never a vendor's product line** — [ADR-0048](./0048-a-convention-is-evidenced-by-placement-never-by-catalogue.md)'s unit rule, which travels here unchanged. *The kubelet API* is an interface of one component and qualifies; *Kubernetes*, *our appliances* or *Dell's DRACs* are product categories and do not |
| Is a hardening document disqualified by its grammar | **No, and the objection proves too much.** *"A checklist item is an instruction, not a legitimacy statement"* is §9.1's second ground against Red Hat and §4.4's against NSA/CISA. **[measured]** MySQL's *Security Guidelines*, MongoDB's *security hardening*, Microsoft's *Security considerations for a SQL Server installation* and ZooKeeper's *Administrator's Guide* are hardening documents in the owner's voice and each already **carries a row** in §3.4. Against a **non-owner** the shape is fatal because limb 1 already refuses the party; against an **owner** it adds nothing limb 3 does not test |
| `10255/tcp` kubelet | **Promoted out of the weak tier into explicit prohibition**, and it is the **thinnest member of that tier** (§18.7). Conditional on [#83](https://github.com/winniel123/verge-asm/issues/83), which may remove the row |
| `10250/tcp` kubelet | **Also promoted, from trusted-network scoping into explicit prohibition** — the same two sentences, and its membership evidence is the **stronger** of the two (`ports-and-protocols.md` numbers `10250` as *Kubelet API*). This discharges §16.9's flag that `10250`'s cell was the thinnest placement in the table. Conditional on #83 |
| What the rule re-opens elsewhere | **One row is newly exposed and is ticketed, not moved: `623/udp` IPMI**, whose owner sentence (Dell's, about *DRACs*) numbers no port and whose number is supplied by **CISA**, a corroborator — limb 2 unsatisfied on the artefacts the note holds. **`4369/tcp` is not rescued**, because limb 2 fails there too and §16.9's live argument survives intact |
| What the rule ratifies | **Most of the prohibition tier**, which has been running this rule unnamed since #21. §16.4 already ruled Elasticsearch's *"never expose an unprotected node"* reaches `9300` on exactly these limbs |
| Does any `(port, transport)` pair move | **No. The list stays at 37 pairs**, §3's class totals are unchanged, no rule version moves, no `Break`, no aperture change, and this does not block [#12](https://github.com/winniel123/verge-asm/issues/12). A footing is evidence for a claim and not a claim ([ADR-0036](./0036-a-shipped-default-is-the-configuration-that-takes-effect.md), §12.7) |

## Rationale

### 1. §2.3's refusal has two things in it, and only one of them was ever load-bearing

§2.3 gives its own reason twice, and the two readings come apart exactly here:

> "Both attest a **category** — 'management interface', 'out-of-band interface'. Neither attests a
> port."

> "The mapping from its protocol list to port numbers is **the reader's inference**, not CISA's."

The first sentence is grammatical. The second is not: it says the defect is that **nobody with
standing closed the gap**, so the reader closed it. Where the owner closes the gap in its own
artefacts, the reader infers nothing — it reads two owner statements and concatenates them. The
grammatical reading and the inference reading agree on every case §2.3 was written about, because a
non-owner cannot close the gap by definition: CISA does not get to say which ports are *management
interfaces*.

**The rule that follows is not new behaviour. It is the behaviour the table already has, named.**

### 2. The grammatical reading is refused because it guts the table, and §2.2 already refused its failure mode

This is the decisive argument and it is a measurement over §3.4's own quotes rather than a
principle. Of the 26 pairs §16.7's footing table places, the number of owner sentences that write
the port number is small:

| The sentence carrying the row | Does the owner write the number? |
|---|---|
| Redis — *"expose the Redis instance directly to the internet … the Redis TCP port"* | **No** — a definite description. `redis.conf` ships `port 6379` |
| memcached — *"you must not expose memcached directly to the internet"* | **No.** The software, not the port |
| Microsoft SQL Server — *"don't connect your SQL Server instances directly to the Internet"* | **No.** Instances, not ports. **[#107](https://github.com/winniel123/verge-asm/issues/107): reach runs through this rule and then **fails at limb 3** — Microsoft names internet-facing TCP/1433 as supported** |
| Elasticsearch — *"Never expose an unprotected node to the public internet"* | **No** — and §16.4 ruled it reaches `9300` anyway. **[#114](https://github.com/winniel123/verge-asm/issues/114): both rows are now REMOVED and §16.4's placement is withdrawn at its own site**; this rule was **confirmed by use** in doing it, carrying Elastic's ECE affirmation to `9300/tcp` on three of the owner's own artefacts. `sensitive-ports.md` §38.5, §38.8 |
| rsync — *"Do not expose a cleartext daemon to an untrusted network"* | **No.** The daemon |
| SMB — *"unlikely that any SMB communication … is legitimate"* | **No** in the sentence; the same document tabulates 445/139/137/138 |
| MongoDB — *"your `mongod` and `mongos` instances are only accessible on trusted networks"* | **No.** Binaries, not ports |
| NFS — *"on a trusted physical network between trusted hosts"* | **No** |
| ZooKeeper — *"a ZooKeeper ensemble … behind a firewall"* | **No.** The ensemble |
| Docker — *"reachable only from a trusted network or VPN"* | **No** |
| IPMI — Dell, *"DRAC's are intended to be on a separate management network"* | **No** — and the number comes from CISA (§18.5) |
| MySQL, etcd, Cassandra, RabbitMQ `25672`, kubelet `10250` | **Yes** — five cells |

**Under the grammatical reading the sensitive list is the set of ports whose maintainers happened to
type a number.** §2.2's founding paragraph refuses that outcome in advance, in its own words: a list
built on *"an asymmetry driven by a documentation accident rather than by any difference in the two
services' deployment models"* is *"exactly the kind of arbitrariness that destroys a curated list's
credibility."* The grammatical reading manufactures that asymmetry at scale.

It is also not a live option in the way the ticket framed it, and that is worth stating plainly:
**adopting it would overturn a ruling already made.** §16.4 admitted `9300` on a sentence about
*nodes*, wrote out the owner's own definition of a node, and called the result *"an inference of one
step"* — which is limb 2, unnamed. Refusing reach would remove `9300` from the prohibition tier, ~~put
`139`/`137`/`138` outside the SMB cell that carries them~~, and leave `623` with no footing at all.

> **The struck clause is CORRECTED by [#107](https://github.com/winniel123/verge-asm/issues/107).**
> **[measured]** the carrying document's own table gives the `Application protocol` of `137`/`138`/`139`
> as **NetBIOS Name Resolution**, **NetBIOS Datagram Service** and **NetBIOS Session Service** — only
> `445` reads `SMB` — and the same document adds that *"the use of NetBIOS for SMB transport ended in
> Windows Vista, Windows Server 2008, and in all later Microsoft operating systems"*. **So the three
> NetBIOS pairs were never riding this rule at all**: the owner's own artefact places them *outside*
> the category, and reading them in would be the reader's inference, which limb 2 forbids. They are
> carried instead by **enumeration** — the same document's perimeter directive governs a table that
> numbers all four pairs. The grammatical reading would therefore not unseat them, and the three cells
> stand on a stronger warrant than this ADR gave them. `sensitive-ports.md` §33.3.

### 3. The hardening-instruction objection is real, and it fails on its own consistency

The objection deserved better than dismissal, and it is the ticket's stated counter-argument: the
retrieved sentence is a **checklist item**, grammatically an instruction to a deployer, which is
§9.1's Red Hat shape and §4.4's NSA/CISA shape. Both were refused. A checklist states a floor for a
hardened deployment. It does not say that every other deployment is illegitimate, and this list's
claim is the stronger one — *never legitimately internet-facing*.

Three answers, in increasing order of force.

1. **§9.1's three grounds were not equal, and only one of them is at issue.** Red Hat lost on
   standing (ground 1), on shape (ground 2) and on self-contradiction across its own products
   (ground 3). Ground 1 is cured here. Ground 3 is **absent for this member and present for
   another**, which is §18.4. Ground 2 is what remains, and it never stood alone.
2. **The standard already admits weaker modality than this.** §2.2's third form — a restricting
   shipped default — asserts nothing at all about legitimacy: PostgreSQL's `listen_addresses =
   localhost` says only what the software does. It is nonetheless the **sole** footing for two rows.
   A prose sentence in the owner's voice saying the interface *"is not exposed publicly on
   Internet"* is not weaker than a config default. It is the same position stated explicitly.
3. **[measured] Applied consistently, the objection removes rows nobody proposes removing.** MySQL's
   `security-guidelines.html`, MongoDB's `security-hardening/`, Microsoft's *Security considerations
   for a SQL Server installation* and ZooKeeper's *Administrator's Guide* are all hardening
   documents, all in the owner's voice, and all four carry rows in §3.4 today. The genre is not the
   defect. **The party is**, and where the party has standing the genre question collapses into limb
   3: an instruction is a *preference* exactly where the architecture it advises against is one the
   owner supports.

> **Against a non-owner, "this is a hardening instruction" is fatal because limb 1 has already
> refused the party. Against an owner, it adds nothing that §10.3's failure condition does not
> already test.**

### 4. The same sentence reaches three members and is defeated for one of them, which is the cleanest possible demonstration of limb 3

The retrieved checklist item names **the Kubernetes API, the kubelet API and etcd**. §4.4 excludes
`6443/tcp` kube-apiserver, and quotes this very sentence while doing so — then quotes what upstream
says immediately after it:

> "Be careful, as **many managed Kubernetes distributions are publicly exposing the API server by
> default**."

So the owner, in one document, states the category prohibition and concedes that its own ecosystem's
dominant deployment violates it for one member. **For `6443` the checklist item is a preference
expressed against a real supported architecture — §4.4's words, and now demonstrably the owner's own
architecture.** For the kubelet API the owner concedes nothing of the kind anywhere retrieved:
`10250` is *Used By: Self, Control plane* and `10255` is disabled by default.

This is what makes limb 3 a rule rather than an escape hatch. It is falsifiable, it is keyed to a
document rather than to judgement, and it is tested **per member**:

> **A category sentence is tested per member, not per sentence** — the exact sibling of
> [#76](https://github.com/winniel123/verge-asm/issues/76)'s *ownership is tested per port, not per
> sentence*, and produced by the same defect: a sentence is not a unit of evidence, a claim about a
> subject is.

**`6443` does not re-open.** Its exclusion rests on three independent grounds and the attestation
limb was never one of them: Claim 3 fails on the facts, determinacy fails against `sun-sr-https` and
against Kubernetes' own *"the API serves on port 443"*, and §4.4's third ground is a product
judgement about false firings. What changes is the disclosure — §4.4 may no longer describe the
upstream quote as merely *a hardening preference*. It is an owner prohibition that this row defeats
on other grounds.

### 5. What limb 2 costs, stated as the row it does not save and the row it newly exposes

A rule that only ratified would be worthless, so both directions were walked.

**`4369/tcp` epmd is not rescued.** Erlang/OTP's sentence is about *distributed nodes* and the
distribution transport. epmd is the mechanism's **registry** rather than the transport, and the
owner's own epmd page distinguishes them and declines to prohibit anything. So the owner does not
place `4369` inside the category it spoke about, limb 2 fails, and §16.9's *"a reader who says
Erlang/OTP has stated a position about the distribution port and no position about `4369` has a live
argument"* survives this ADR completely intact.

**`623/udp` IPMI is newly exposed, and it is ticketed rather than moved.** Dell's sentence — *"DRAC's
are intended to be on a separate management network"* — is a **product** category, not a protocol
one, so it fails the ADR-0048 unit check on its face. The number that connects it to the row
comes from CISA's *"usually UDP port 623"*, a corroborator, which limb 2 forbids. That is §10.6's
`161/udp` shape — a corroborator standing where an owner should — in a second instance, and it sits
in the **prohibition** tier. It is not decided here: this ticket's work is the rule, a row moves on a
retrieval and never on a re-reading ([#37](https://github.com/winniel123/verge-asm/issues/37)), and
ADR-0037 limb 2 requires it be ticketed.

### 6. The rule travels, and moves nothing where it travels

[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) makes §2.2's
attestation gate the one part of this standard that generalises to other curated tables, so this
rule reaches [`weak-key-and-signature.md`](../research/weak-key-and-signature.md) whether or not
anything there needs it.

**Checked against §13.5's restated table: nothing moves.** That table's subjects are named
algorithms and named parameters, and every surviving footing names its subject — RFC 9846 §C.2
*"naming the number and the population"*, SP 800-131A Table 2's `N ≥ 224`, RFC 9325 §4.5's MUST over
exactly the artefact at issue. The shape this ADR governs — a specifier's **security-strength band**
reaching a key size it does not enumerate — is not how any row there is founded, and the weak tier
that survives is weak on **modality and scope** ([#73](https://github.com/winniel123/verge-asm/issues/73)),
which is a different axis. The check is recorded so the next session does not re-run it.

## Consequences

- **No `(port, transport)` pair moves.** 37 pairs. §3.1/§3.2/§3.3's 12/7/18 unchanged. §6.1's
  containment arithmetic unchanged. [ADR-0009](./0009-verge-core-is-a-union.md)'s union unchanged.
  No rule-version move and no `Break` under
  [ADR-0008](./0008-derivation-versions-move-on-content.md). No aperture change. **#12 is not
  blocked**.
- **Two footing cells move, both in the same direction, and both are conditional on
  [#83](https://github.com/winniel123/verge-asm/issues/83).** §2.2's tiers become **prohibition 15
  pairs · scoping 9 pairs · weak 2 rows**, coverage unchanged at **26 of 37**. If #83 removes either
  kubelet row, its cell leaves with the row and the tier counts fall accordingly — the cell is
  evidence for a claim, so it cannot survive the claim.
- ~~**The curator's watch list returns to `5432/tcp` and `5984/tcp`.**~~ **The *weak tier* returns to
  `5432/tcp` and `5984/tcp`. The identity with the watch list is SUPERSEDED by
  [#125](https://github.com/winniel123/verge-asm/issues/125)** — the watch keys on the **revision act**
  ([ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md),
  [`sensitive-ports.md`](../research/sensitive-ports.md) §39). This bullet's own next sentence is why:
  it records the third correction in a week and the *compare members, not counts* lesson, and #125
  takes that to its conclusion — the unit is a `(cell, artefact, revision act)` triple and the count is
  barred as an indicator. **`10255/tcp` does leave the queue** on this ADR's evidence, but on **support
  count** — it acquires a second independent ground — and not because its tier moved. ADR-0032 §8's enumeration is
  amended for the **third** time in a week, and the sequence is now 3 → 2 → 3 → 2 with the
  membership changing every time. §8's own lesson — *a reader comparing counts rather than members
  sees nothing move* — has a second instance, and it is this one.
- **[#79](https://github.com/winniel123/verge-asm/issues/79)'s fourteenth sole-ground negative
  retires.** #79 recorded that #76 manufactured one mid-sweep by moving `10255` into the weak tier,
  and that it was the one that returned a sentence. Applying the sentence removes the negative
  rather than answering it: `10255` no longer rests on an absence, so under
  [ADR-0046](./0046-a-negatives-corpus-is-its-owners-class-list-and-only-a-sole-ground-negative-is-exposed.md)
  limb 1 it leaves the exposed population. **ADR-0046's table-state qualifier fires again in the
  opposite direction** — a footing move re-arms the sweep for the row that moved, and here it
  *disarms* it.
- **§16.9's thinnest-cell flag is discharged for `10250` and a new one opens for `10255`.** `10250`
  no longer rests on a table cell for its *position*. The table cell now does **membership** work
  instead, which is the job it is actually fit for. `10255` becomes the thinnest member of the
  prohibition tier — the only one resting on a checklist item, in a documentation release branch
  that can be edited as easily as a default can be flipped.
- **One ticket is opened and nothing is admitted:** `623/udp`'s membership limb
  ([#90](https://github.com/winniel123/verge-asm/issues/90)), which **blocks
  [#12](https://github.com/winniel123/verge-asm/issues/12)** because it can reach a row removal.
- **§2.3 is narrowed and not weakened.** Every category statement from CISA, the CPG, NSA, AWS,
  Google Cloud and CIS still carries nothing, for the reason §2.3 gave. What is withdrawn is the
  implication that the *grammar* was the defect.
- **`CONTEXT.md` is not edited**, on ADR-0032's, ADR-0035's, ADR-0040's and ADR-0046's precedent and
  for their reason. **[checked]** the glossary holds no term for attestation, footing or category.
  The only occurrence of *attestation* is [#62](https://github.com/winniel123/verge-asm/issues/62)'s
  unrelated clause about offers. Nothing here changes a domain term.
- **No new owner is admitted.** Kubernetes was already the kubelet's owner under §10.5 and §16.5
  says so. This ADR changes what its sentences reach, never who may speak.

## Amendment — [#90](https://github.com/winniel123/verge-asm/issues/90): `623/udp`'s membership limb is satisfied, and this ADR was not needed for it

The one item this ADR left open is discharged. **The Decision, all three limbs and the unit rule are
unchanged**. What moves is a factual premise about one row, and it moves because the retrieval this
ADR declined to perform was performed.

[#90](https://github.com/winniel123/verge-asm/issues/90) retrieved the **IPMI Specification, Second
Generation, v2.0, Document Revision 1.1, 1 October 2013** (Intel, Hewlett-Packard, NEC, **Dell**) and
the **DMTF Alert Standard Format Specification DSP0136 v2.0, 23 April 2003** as shipped bytes.
**[measured]** IPMI §13.1.2 and Table 13-1 place the *Primary RMCP Port* at **623 (26Fh)** under *"two
well-known ports under **UDP**"*. DSP0136 §3.2.1 reserves it independently and **owns** the assignment,
which IPMI cites normatively and inherits. [`sensitive-ports.md`](../research/sensitive-ports.md) §23.

- **The Rationale §5 finding *"the number that connects it to the row comes from CISA's 'usually UDP
  port 623', a corroborator"* is withdrawn.** The number is two specifications', one of them Dell's
  own. **This ADR's qualifier was exactly right and is why it ticketed rather than ruled** — *"limb 2
  unsatisfied **on the artefacts the note holds**"*. The artefacts it did not hold satisfy it.
- **The unit-rule finding is not withdrawn and is not needed.** *"DRAC's"* is still a product line and
  a product line is still not a category. What changes is that **nothing has to reach**: the owner
  numbers the port, so `623/udp` belongs in §18.5's *"numbered by the owner, so the rule is not
  needed"* bucket, alongside `3306`, `2379`, `2380`, `9042`, `25672` and `10250`. The row is a
  concatenation of three owner statements — Dell placing the DRAC inside IPMI by enumerating which
  IPMI version each DRAC firmware ships (VU#843044, read in full for the first time), the
  specifications placing IPMI-over-LAN on `623/udp`, and Dell stating the position — with no reader
  inference at any hop. **§23.6 argues the objection rather than dismissing it.**
- **`4369/tcp` is still not rescued**, for the reason Rationale §5 gives, unchanged.
- **No `(port, transport)` pair moves and no footing cell moves between tiers.** `623/udp` stays in the
  explicit prohibition tier, re-founded where it sits — §20's shape rather than §18's. The list stays
  at **37**. Classes **11 / 7 / 19**. Tiers **prohibition 15 · scoping 9 · weak 2**. Coverage **26 of
  37**. [ADR-0009](./0009-verge-core-is-a-union.md)'s union unchanged. No rule version moves and no
  `Break`. **[#12](https://github.com/winniel123/verge-asm/issues/12) is unblocked** — the row removal
  it was priced against does not happen.
- **No new ADR was minted**, on #76's and #84's precedent. The rule the case might have wanted — *a
  vendor that co-authored the protocol's specification owns the protocol, not merely its product* — is
  [`sensitive-ports.md`](../research/sensitive-ports.md) §10.5 as already written, *"the party that
  **designed the protocol**"*, and it is what separates this case from §11.8's Cisco-and-SNMP fence.

## Confirmed by use — [#107](https://github.com/winniel123/verge-asm/issues/107), 2026-08-14

**Limb 3 has demoted a prohibition-tier cell for the first time, and the rule is unchanged.** #107
walked the footing tier's membership test per row and found `1433/tcp` MS SQL failing reach: the
carrying page numbers no port — **[measured]** the string `1433` occurs **zero** times on it — so
limb 2 must carry the pair through this rule, and **[measured]** Microsoft's own current documentation
names internet-facing **TCP/1433 SQL Server** a supported, portal-provisioned option (*"| **Public** |
Connect to SQL Server over the internet. |"*; *"Any client with internet access can connect to the SQL
Server instance"*). That is limb 3's defeat condition met by the owner, for the member, in the present
tense. [`sensitive-ports.md`](../research/sensitive-ports.md) §33.4.

> **And the row has since gone too — [#109](https://github.com/winniel123/verge-asm/issues/109),
> 2026-08-14, on a retrieval scoped to the row.** `1433/tcp` is **removed from the sensitive list**;
> the list is 40 pairs and the pair is in `sensitive-ports.md` §4.6. **This rule is not what removed
> it, and the distinction is now written down.** Limb 3 is a **defeat test on this rule's reach** and
> is answered at the **attestation** gate; the row fell on `sensitive-ports.md` §10.3's failure
> condition, which fires on the owner's **affirmative** naming and is answered at the **claim** gate —
> [ADR-0067](./0067-a-claim-fails-on-the-owners-affirmative-naming-not-on-the-reach-of-its-own-prohibition.md)
> limb 1. The consequence that matters for a reader of *this* ADR: **an argument that narrows the
> owner's category statement rescues the footing and does not rescue the row.** `sensitive-ports.md`
> §35.
>
> > **And two more rows have gone the same way — [#114](https://github.com/winniel123/verge-asm/issues/114),
> > 2026-08-14, on a retrieval scoped to the pair.** `9200/tcp` and `9300/tcp` are **removed**; the
> > list is **38 pairs** and both are in `sensitive-ports.md` §4.6. **This rule is what carried the
> > affirmation to `9300/tcp`** — Elastic's ECE statement numbers no port, and limb 2 places the pair
> > inside it on three of the owner's own artefacts — so the rule ran here **in the admitting
> > direction at the claim gate**, which no prior section had done. **Limb 3 was never reached**: there
> > is no defeat test at the claim gate (`sensitive-ports.md` §37.8). **The rule is confirmed by use
> > and is not amended.** `sensitive-ports.md` §38.5.

**It is a stronger instance than the one this ADR was argued from.** Rationale §4 turns on Kubernetes
**conceding** that *"many managed Kubernetes distributions are publicly exposing the API server by
default"* — the owner reporting what third parties do. Here the owner **ships the provisioning option
itself** and configures the operator's firewall and network security group to permit it. **The
addressee is identical in both documents** — the operator running its own SQL Server — which is what
separates this from a vendor operating its own hardened managed service on the same number, and that
distinction is what spares `445/tcp` on the same test (§33.5).

**Limb 3's *per member* discipline is what keeps the result narrow.** Run across all sixteen
prohibition-tier members the defeat test is met **once**, disposed-of-on-direction once, and refused
fourteen times — Redis, Elastic, memcached and Oracle each spared on a retrieved artefact rather than
on judgement. **A defeat test that fired on every owner with a cloud product would be the wide
aperture this ADR's second rejected alternative names. It does not fire that way.**

**No limb is amended and no other cell moves.**

## Alternatives rejected

| Alternative | Why not |
|---|---|
| **The grammatical reading — a category statement carries no port whoever makes it, so `10255` stays in the weak tier** | **This is the option that lost, and it lost on measurement rather than on taste.** Of the sentences carrying §16.7's 26 placed pairs, five write the port number; the rest name a daemon, a binary, a node, an ensemble or a protocol. The reading would leave the sensitive list resting on which maintainers happened to type a number — the *documentation accident* §2.2's founding paragraph names as the thing that destroys a curated list's credibility — and would overturn §16.4's `9300` ruling, unseat `139`/`137`/`138` from the SMB cell and strip `623` of any footing. It is a coherent reading of §2.3's first sentence and it is incompatible with §2.3's second |
| **Reach, with no membership limb — an owner's category sentence carries any port the owner would agree is in the category** | The wide aperture the ticket warned about, and it re-admits exactly what §2.3 refused: *the reader's inference*, with the owner's name on it. It would rescue `4369` on our reading of what *distributed nodes* includes, which is the judgement §16.9 flagged as live and unsettled, and it would make the standard unfalsifiable — there is no artefact to attack |
| **Reach, with no defeat limb** | It would put `6443/tcp` kube-apiserver's attestation on the same footing as the kubelet's while the owner concedes public exposure is the managed default for it. §4.4's other two grounds would still exclude the row, so nothing visible would break today — which is the worst reason to leave a rule underspecified, because the next case would arrive without §4.4's second and third grounds to catch it |
| **A fourth footing tier for category-derived prohibitions** | Tempting and refused. The tiers discriminate **what the owner said**, not **how many owner statements it took to say it** — `9300`, `445` and `6379` are all category-derived and none is labelled. A tier is a disclosure of strength; a two-statement concatenation is not weaker than a one-statement one when both statements are the owner's. Where a cell *is* weak, §18.7 flags it individually, which is what the note already does for `10250`, `4369` and `2379`/`2380` |
| **Put `10255` in the trusted-network scoping tier instead of prohibition** | The second quote — *"access should be restricted and not exposed publicly"* — is scoping-shaped in its first clause. But the first quote is a flat unqualified negative about internet exposure and names **no network** that would make it a scoping statement, which is what the scoping tier means. It is a prohibition, and it is disclosed as the tier's thinnest member rather than demoted to a tier it does not fit |
| **Wait for [#83](https://github.com/winniel123/verge-asm/issues/83) before moving either cell** | #76 declined to re-tier a row placed by a **concurrently running** pass, and that was right. #83 is concurrent, but it decides the **rows** while this decides their **footings**, and a footing that would be correct for as long as the row exists can be written now with its conditionality stated. Waiting would also leave the rule unstated, which is what the ticket is for; the rule does not depend on kubelet surviving |
| **Decide [#86](https://github.com/winniel123/verge-asm/issues/86)'s published-versus-committed question here, since both are about §2.2** | Different gates, and they compose rather than overlap. #86 asks **which artefacts** are the owner's documentation; this asks **how far a sentence in one reaches**. The kubelet quote is on a release branch of the docs repo and is published as a rendered page, so it clears #86 on either answer — but Kafka's `9092` sentence, which is #86's actual case, names **no port** either, so it needs *this* rule as well as #86's. Ruling both here would settle #86's question on a case that does not test it |
