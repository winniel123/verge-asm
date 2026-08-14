# ADR-0048: A convention is evidenced by placement, never by catalogue

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#82 §2.4's determinacy gate has no evidence standard — what may establish that a convention is contested?](https://github.com/winniel123/verge-asm/issues/82)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0042](./0042-a-squat-is-contested-where-the-other-convention-is-live.md)
([#75](https://github.com/winniel123/verge-asm/issues/75)) settled the **criterion** for
`sensitive-ports.md` §2.4's determinacy gate: *a squat is contested where the other convention is
live*, liveness read off the competing owner's own current documentation and never off a frequency
source. It reproduced six existing verdicts unchanged and closed the `9200`-versus-`9100` tension the
note had carried since [#21](https://github.com/winniel123/verge-asm/issues/21).

It left the other half open, and said so in its own Consequences: **it does not say which classes of
source may establish a convention at all**, the way §2.2 says which sources may attest a claim. Every
sibling gate on this table has had that treatment and this one never has:

| Gate | Its evidence standard |
|---|---|
| §2.1, the claim | Closed **by construction** over what an internet vantage supplies, with three candidate fourths tested and refused — §10.2, [#37](https://github.com/winniel123/verge-asm/issues/37) |
| §2.2, the attestation | Three named forms; an **owner** definition ([ADR-0035](./0035-a-cryptographic-primitives-owner-is-its-specifier.md), §10.5); a one-way rule for shipped defaults (§10.4); an artefact test ([ADR-0036](./0036-a-shipped-default-is-the-configuration-that-takes-effect.md)); an extent rule ([ADR-0037](./0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md)) |
| §2.3, the corroborators | A named population and one rule: **corroborate, never carry** |
| §2.4, determinacy | *"Uncontested convention has to be"* the test — plus ADR-0042's criterion, and **no source rule of any kind** |

`sensitive-ports.md` §14.6 records what that cost the ruling that needed it: #75 *"had to weigh a
vendor's product-port table, a vendor's container image README, an IANA annotation and a rendered
manual against each other with nothing in the standard to rank them."*

**The hazard is §2.3's, one gate across, and it is worse here in one specific way.** §2.3's whole
finding is that a gate with no source rule gets cleared by whichever authoritative-looking document is
nearest. An unfenced determinacy gate can be cleared by a weak source to admit a row **or blocked by
one to refuse a row that belongs** — and a refusal leaves no artefact behind to argue with. #75's is
the second shape. It is also the shape a reader cannot audit, because the note's negative space
records verdicts and, until §14.8, did not always record the document that produced them.

## Decision

**A determinacy finding is made on placement statements and on nothing else.**

> **Placement statement.** A party's statement, in its **own current documentation** or its **own
> shipped or compiled bytes**, that **its own software listens on a given `(port, transport)` pair by
> default**.

Five limbs, plus one procedural rule.

1. **The positive limb — what establishes a convention.** A row's convention is established by the
   **candidate service's own owner's** placement statement, in §10.5's sense of owner. We never
   establish one, and a catalogue never establishes one. In practice this limb is already satisfied
   everywhere on the list, which is why nobody noticed it was unstated: every listed project
   documents its own default port.
2. **The negative limb — what defeats it.** A convention is **contested** where **another party**
   has a current placement statement on the same pair for a **different protocol**. It is
   **displaced** where the **candidate's own owner** has one putting its service on a different pair,
   or making the pair depend on which version is running. **One statement suffices**, and a survey is
   neither required nor claimed (`sensitive-ports.md` §14.9).
3. **The unit is the protocol, not the vendor.** Two parties placing the **same** protocol on a pair
   are **one** convention, not two. Compatibility is read off the second party's own declaration —
   ScyllaDB declaring CQL, OpenSearch declaring the Elasticsearch REST API — never judged by us. This
   limb is not a softening: it is what §4.6 already meant by *"one port, two completely different
   services"*.
4. **Current means the party still presents the statement as applicable** — bytes at a supported
   release, documentation for a supported product, or a specification **in force**: not obsoleted,
   withdrawn, or reclassified Historic. **Currency, not size, is what *live* means.** A competitor
   with one deployment and a current page contests; a competitor with a large installed base and no
   current document does not.
5. **Everything else corroborates and never carries.** §2.3's rule with a **descriptive** population
   in place of a normative one: IANA registry rows and the `Unauthorized Use Reported` field,
   `nmap-services`' name column, cloud-provider and government port tables, third-party port
   references, and **this project's own frequency half**.

> **The refusal artefact rule.** Determinacy is a **defeasible presumption**: once limb 1 is met it
> holds until a document defeats it. So **every refusal on determinacy names the artefact that
> defeated it**, quoted and dated, in the note's negative space. A determinacy refusal with no cited
> document is not a finding.

**Two riders.**

**First-party is a property of the row, not of the document.** Apple's *TCP and UDP ports used by
Apple software products* is first-party about `AirPlay · 7000 · TCP` and third-party about anything
else it happens to tabulate. This is [ADR-0037](./0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md)'s
shape applied to the other gate.

**A registration reaches the limbs only through its registrant.** An IANA row is a **record of a
registrant's own placement declaration**, not an independent authority — the registry says so itself,
in capitals, and §2.4 has ruled since #21 that registration cannot be the determinacy test. So a
registration is followed to the registrant's current documents and stands or falls with them. That is
why `79/tcp` is defeated by **RFC 4146** — the IETF's own in-force specification, saying that a
process listens on the finger port for a protocol that is not finger — with IANA's annotation
corroborating and not carrying, which is a strengthening of §9.3.4's footing rather than a change to
its verdict.

## Rationale

### 1. A determinacy finding is a *conclusion*, not an *assertion*, and that is the whole of the source rule

[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §6 cut three instruments
that partition on where a project-authored table sits relative to the wire: **before** it (what we
ask — an offer), **at** it (what a byte means — a conclusion), **after** it (what a value means
normatively — an assertion). The ticket asks whether determinacy's inputs need a fourth. **They do
not, and the third instrument is the wrong one.**

The sensitive list is an **assertion** table and stays under exactly one instrument; nothing here
moves it. But the three gates inside it do not all ask assertion-shaped questions. §2.1 and §2.2 ask
*may this normative statement be made, and by whom* — after the wire. §2.4 asks *does an observed
listener on this pair tell the operator what they are looking at* — **at** the wire. ADR-0032 already
named that: determinacy is the **surrogate** gate, and the surrogate is exactly where an assertion
table has to answer a conclusion question, because the rule *"cannot read the fact it is named for"*.

Read #31's line at the object: *"a table deciding where to look is aperture; a table deciding what an
answer means is a signature database"*, whose failure mode is *"a false verdict that moves when a
vendor's banner moves"*. A port-to-service mapping asserted by a third party **is** a signature
database — `nmap-services` is literally one. So the discipline #31 imposes is the discipline
determinacy needs: pin the meaning to something the **party itself defines**, because a catalogue's
version of it is precisely the object #31 refused. A placement statement is the port-number analogue
of #31's spec-defined field: the field is defined by the party that defines the software.

**That placement decides the rest of the ADR without further argument.** It is why gate 2's
apparatus — owner, attestation forms, one-way defaults — does not travel to gate 3, and why nothing
new had to be minted for it.

### 2. There is no second sense of *owner*, and minting one would have leaked a normative entitlement

The ticket's sharpest sub-question is whether Apple, which owns AirPlay but not port 7000, needs to
be named an **owner of a convention** the way ADR-0035 named the cryptographic-primitive case.
**No — and the reason is the reason ADR-0035 said yes to its case.**

§10.5's *owner* is a **normative entitlement**: it answers *who is entitled to say that exposing this
is wrong*, and it exists so that a hardening opinion cannot stand where a position should. ADR-0035
extended it because a cryptographic primitive genuinely has a party whose standard **is** the
algorithm, and that party's statements are of the same normative kind.

Determinacy asks nothing normative. It asks a **factual** question — *what listens there* — and no
party is entitled to answer that for a number. IANA disclaims exactly that entitlement in capitals.
Minting *owner of a convention* would say somebody is, and it would leak: once Apple owns `7000`,
Apple's prose about `7000` becomes admissible under §2.2 for a row we would then have to argue out
again on other grounds.

What Apple's table actually is is a **self-declaration**, which is a different and stronger object
for this purpose. It is falsifiable by running the software, and it is self-interested in the
direction that makes it reliable: a vendor that misstates its own default port ships a product that
does not work. That is §12.2's own move — *"the test is read off the file rather than judged — nine
artefacts, nine self-declarations"* — and ADR-0036's, replacing a judgement with a sentence read off
a document. **The class does the work that a second owner definition would have done, and carries
none of the entitlement.**

### 3. Limb 4 is where frequency would have got back in, and *currency* is the fence

ADR-0042 already refused deployment share in both directions. But its own reconstruction table
explains `9200` with the words **"WAP has no deployed population"**, and that is a population
sentence sitting inside a rule that forbids population sentences. Left alone it is the crack the next
session widens: if *no deployed population* can establish uncontestedness, *a large deployed
population* is one short step from establishing contestedness, and §1's exclusion of frequency is
gone.

**Liveness is the currency of a declaration, not the size of a population.** Concretely and
deliberately non-monotone in deployment size:

- If AirPlay had a thousand installs and Apple's current page still tabulated `AirPlay · 7000 · TCP`,
  `7000/tcp` would still be refused.
- If WAP had a hundred million handsets in the field and no party published a current placement
  document for WSP, `9200/tcp` would still be listed.

The second bullet is the uncomfortable one and it is stated rather than smoothed. It is defensible for
the same reason limb 2 of ADR-0042 was: a criterion that asks a reviewer to judge whether a protocol
is *still a thing* is the ownerless counterfactual §10.1 deleted and §12.4 refused. Currency is a
**retrieval** — open the party's current documentation and find the number, or record that you did not
— and the bar to defeating a row is therefore one document, which is as low as a falsifiable bar gets.

`sensitive-ports.md` §15.3 restates the `9200` finding on that footing.

### 4. Limb 3 was forced by the walk, and it is the finding the ADR-0042 table never had to face

**[retrieved]** Walking the rows that rest on convention (`sensitive-ports.md` §15.4) turns up
something ADR-0042's six verdicts did not contain: **`9200/tcp` and `9042/tcp` both have current
first-party placement statements from parties other than the row's own owner.** OpenSearch's own
network-settings documentation defaults `http.port` to the `9200-9300` range, the Wazuh indexer is
distributed on the same number, and ScyllaDB's configuration reference defaults `native_transport_port`
to `9042`.

Under limbs 1, 2 and 4 alone, both rows are contested and both leave the list — which would be two
rows moving on a standard whose whole cost estimate was zero, and it would be **wrong**. A firing that
names Elasticsearch and finds OpenSearch has named the service correctly at the granularity the row
asserts: same wire protocol, same claim, same remediation, and the second party declares the
compatibility itself. §4.6 already said this in the sentence it used to exclude `9100` — *"one port,
two **completely different** services, opposite populations"* — and limb 3 is that adjective written
down.

The discriminator is mechanical and is read off the competitor rather than judged: **does the second
party declare that it speaks the first's protocol?** ScyllaDB declares CQL; OpenSearch declares the
Elasticsearch REST API. Apple does not declare that AirPlay speaks Cassandra's internode protocol, HP
does not declare that JetDirect speaks Prometheus exposition, and Oracle does not declare that the
WebLogic AdminServer speaks Cassandra's. Where the declaration is absent, the services are different
and the convention is contested.

### 5. It reproduces every verdict, and the six were checked rather than assumed

ADR-0042's reconstruction table, re-run under this ADR's source rule. The **ground** moves on two rows
and no **verdict** moves on any.

| Row | Defeating artefact under this ADR | Verdict |
|---|---|---|
| `9200/tcp` Elasticsearch | **None.** No current placement statement for WSP on `9200/tcp` was found; the WAP suite's own bearer ports are WDP/**UDP** (§15.3), a different key. OpenSearch and the Wazuh indexer are the **same protocol** — limb 3 | **Listed**, ground restated |
| `9300`, `2181`, `9042`, `10250`, `10255`, `623/udp` | **None found** — §15.4 walks each and names what was searched. `9042`'s ScyllaDB is limb 3; `623/udp`'s `asf-rmcp` is the transport IPMI rides, so it is limb 3 as well | **Listed** |
| `9100/tcp` node_exporter | HP's own best-practices document — *"9100 Printing should always be enabled"* | **Excluded** |
| `6443/tcp` kube-apiserver | Kubernetes' own *"the API serves on port 443"* — the **displacement** half of limb 2 | **Excluded** |
| `79/tcp` finger | **RFC 4146**, in force, specifying a listener on 79 for a protocol that is not finger. IANA's annotation corroborates | **Excluded**, ground strengthened |
| `7000/tcp` Cassandra | Apple's *TCP and UDP ports used by Apple software products* | **Excluded** ([#75](https://github.com/winniel123/verge-asm/issues/75)) |
| `7001/tcp` Cassandra | Oracle's `ADMIN_LISTEN_PORT` *(default: 7001)*, plus the owner's own deprecation — both limbs of limb 2 at once | **Excluded** |
| `9090/tcp` Prometheus (§4.3) | Red Hat's RHEL 8 and RHEL 9 web-console documentation putting **Cockpit** on 9090 | **Excluded**, ground **replaced** — §4.3 rested it on `nmap-services`' name column, which limb 5 makes inadmissible as grounds |

**Nine rulings, one source rule, no verdict moved.** That is the same test ADR-0042 set itself, run on
the layer beneath it. The two rows whose *ground* changed both got **stronger**: `9090` moves from a
2008-vintage catalogue to a current first-party document, and `79` from a registry-hygiene field to an
in-force RFC.

### 6. This project may not supply its own determinacy evidence, and that is now a rule

#14.4 fact 3 refused to let `safe-active-probing.md` §2.3's *HTTP-ish alternates* label decide `7001`,
in passing. **Stated as a rule** under limb 5, on three grounds, of which only the first was given.

- **§6's invariant is one-directional.** Deriving the sensitive list from the hot set would make
  frequency a precondition of normativity; letting the hot set's **labels** refuse a sensitive row is
  the same laundering in the negative direction.
- **The labels are frequency artefacts.** *HTTP-ish alternate* is a probe-scheduling grouping produced
  by [#4](https://github.com/winniel123/verge-asm/issues/4)'s frequency question. It is not a claim
  about what listens, and it was never retrieved from anybody.
- **§2.2's first sentence already binds.** *"The claim may not be asserted by us."* Determinacy
  inherits it, in **both** directions: we may not establish a convention either. A row may not be
  admitted because our own list says a number means something.

The **product-coherence** use §14.4 made of it survives untouched: recording that our two port lists
disagree about `7001` is worth seeing, and is not evidence.

## Consequences

- **No `(port, transport)` pair moves.** The list stays at **37**; §1's count, §3's class totals
  (12 / 7 / 18), §2.2's footing table and its 19-of-37 denominator, §6.1's containment arithmetic and
  [ADR-0009](./0009-verge-core-is-a-union.md)'s union are unchanged, each checked in
  `sensitive-ports.md` §15.6 rather than asserted. No rule version bump and no `Break` under
  [ADR-0008](./0008-derivation-versions-move-on-content.md).
- **Two grounds move and both strengthen** — `9090/tcp` (§4.3) and `79/tcp` (§9.3.4). Both are
  exclusions, both keep their verdict, and in both cases the replacement is a first-party document
  where the old ground was a catalogue.
- **`sensitive-ports.md` §8 question 10 is closed.** §2.4 now carries an evidence standard, so the
  asymmetry that opened the question — three attestation forms against none — is gone.
- **Every future determinacy refusal owes a cited artefact.** This is the operative change for the next
  session: a refusal that says *the port is conventionally anything* without naming who says so is no
  longer a finding. §4.3's ten generic ports are grandfathered as a **class** and each owes its
  artefact the first time it is individually relied on — §15.5.
- **It does not generalise to other curated tables**, and that is ADR-0032's ruling rather than a
  choice here: determinacy binds where a key is a surrogate, and
  `sensitive-port-reached-from-internet` is the only surrogate v1 has. The weak-key table
  ([#68](https://github.com/winniel123/verge-asm/issues/68), [#79](https://github.com/winniel123/verge-asm/issues/79))
  has no determinacy gate and gains none.
- **[ADR-0042](./0042-a-squat-is-contested-where-the-other-convention-is-live.md) is amended once**,
  in one phrase: *"WAP has no deployed population"* is restated as *no current placement statement*,
  and limb 3 is added to its reconstruction table. Its criterion is untouched.
- **[#12](https://github.com/winniel123/verge-asm/issues/12) is unaffected.** The pair count stays 37
  and definite, as #75 predicted when it wired no blocking edge for this ticket.
- **First application, [#87](https://github.com/winniel123/verge-asm/issues/87) — the standard's first
  use on an *excluded* row's squat, and it reproduces both verdicts.** §15.4 walked the nine **listed**
  rows resting on convention; the two exclusions resting on an untested squat — `5601/tcp` Kibana on
  `esmagent` and `8500/tcp` Consul on `fmtp` — were left to
  [#87](https://github.com/winniel123/verge-asm/issues/87), which ran them.
  **[measured] Both registrations are live and neither row moves; the list stays at 37**
  (`sensitive-ports.md` §18). Three things this ADR predicted are confirmed by use rather than by
  argument. **The registrant rider did the work**: `esmagent`'s IANA contact record resolves to AXENT
  Technologies, whose successor Broadcom places the *CCS Windows Agent* on `5601` in current supported
  documentation — so following the registration to its registrant beat reading the registry's service
  name, which had gone stale through a product rename. **Limb 4's *currency, not size* held under
  pressure in both directions**: Symantec ESM has been out of support since 2016 and the number is still
  live, while a 2007 EUROCONTROL specification is still current because its owner's 2026 standards
  catalogue carries it — neither answer is available from deployment share. And **limb 3 fired on a
  sibling rather than on the row**, disposing of OpenSearch Dashboards on `5601` exactly as it disposed
  of OpenSearch on `9200`. The **refusal artefact rule** is now satisfied for every determinacy refusal
  in `sensitive-ports.md` §4.6 except the Hadoop entry, which stays grandfathered on §15.5's pricing.
- **`CONTEXT.md` is not edited**, deliberately — concurrent sessions are in that file and #37 and
  ADR-0032 both set the precedent. The edit this ADR would make is one clause on `Signal`: *where a
  curated table's key is a surrogate for the fact the rule names, what the key means is evidenced by
  the parties that ship on it, never by a catalogue.*

## Alternatives rejected

| Alternative | Why not |
|---|---|
| **Mint a second sense of *owner* — an owner of a convention** | The ticket's own suggestion, and it is the one that leaks. §10.5's *owner* is a normative entitlement answering *who may say exposure is wrong*; determinacy asks a factual question nobody is entitled to answer, which the registry disclaims in capitals. Making Apple an owner of `7000` would make Apple's prose about `7000` admissible under §2.2. The self-declaration class produces every verdict without inventing the entitlement |
| **Rank sources by authority — IANA above vendors above catalogues** | Puts the registry on top of the one question it explicitly refuses to answer, and resurrects *registration is the test* through the back door: `9200`, `9042`, `2181`, `10250`, `10255` and `25672` all rest on squats or on unassigned ranges and would fall. §2.4 has refused that reading since #21 |
| **Admit deployment evidence — scan studies, `nmap-services` frequency, our own hot set** | ADR-0042 limb 3, §1's framing, §2.5, §2.7, §11.5 and §12.3, now stated as a gate-3 rule rather than applied in passing. It is the one substitution this whole note exists to prevent, and *live* is the word it would have arrived through |
| **Require a survey — prove no other service listens** | Unbounded and unfalsifiable. §14.9 declined to claim it for `7000` and was right to; [ADR-0040](./0040-a-specifications-silence-is-not-the-owners-silence.md)'s discipline is about **bounding** a negative, not completing one. It would also price every admission at an impossible retrieval and collapse the list to the registered rows |
| **Treat determinacy evidence under §2.2's forms** | Wrong instrument. Gate 2's forms carry normative statements by entitled parties; a placement statement is a factual self-declaration. Applied here it demands the owner fiction above, and it makes an in-force RFC about somebody else's usage of port 79 inadmissible — which loses `79` and `623/udp`'s reasoning together |
| **Count every party on a number as a separate convention** | Falsified by the walk: it deletes `9200` on OpenSearch and `9042` on ScyllaDB, two rows nobody thinks are wrong, on parties that declare compatibility with the row's own protocol. §4.6's *"completely different services"* already carried limb 3 unstated |
| **Leave it unwritten, as ADR-0042 left it** | The status quo, and #75 recorded in §14.6 that it had to rank four source classes with nothing to rank them by. The asymmetry is the tell: §2.1 is closed by construction and §2.2 has four ADRs, while the gate that decided the note's two most recent refusals had none. An unstated criterion cannot be argued against, which is the failure §2 was built to prevent and the ground on which #37 repaired Claim 1 |
