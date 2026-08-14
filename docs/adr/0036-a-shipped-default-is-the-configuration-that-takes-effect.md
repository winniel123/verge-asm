# ADR-0036: A shipped default is the configuration that takes effect — and installing one transfers operativeness, not ownership

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#69 An upstream example config a distributor ships as the default: documentation, or a shipped default?](https://github.com/winniel123/verge-asm/issues/69)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) ruled that an evidence
standard attaches to a **curated table** rather than to a rule, and that of
[#21](https://github.com/winniel123/verge-asm/issues/21)'s three gates only the **middle** one —
*state the claim, and cite the source that owns it* — generalises to other tables. That gate is
`sensitive-ports.md` §2.2, and it admits three forms of attestation: a specification, the project's
own documentation, and **the project's shipped default, as documented by the project**.

The third form has a seam ADR-0032 did not touch, because it is not about reach. **The same bytes can
be an example in one party's hand and an operative configuration in another's.**
[#66](https://github.com/winniel123/verge-asm/issues/66) met it while removing `161/udp`: net-snmp
ships `EXAMPLE.conf.def` with the loopback line active and the all-interfaces line commented out, and
Debian installs that file verbatim as `/etc/snmp/snmpd.conf`. Under one reading the file is upstream
documentation — an example, not a default — and the agent's real default is permissive and therefore
**silent** under §10.4's one-way rule. Under the other it is a distributor's shipped default that
**restricts**, which §10.5 admits *"on exactly the same terms as any other shipped default"*, and so
it **admits**. Two readings, opposite answers, one file, and §2.2, §10.4 and §10.5 each speak to one
hand without saying which governs. #66 recorded it as a general defect rather than deciding it,
because both readings lost on that row for an unrelated reason.

The pattern is common enough to decide the next case rather than the last one: an upstream
`EXAMPLE.conf` / `*.conf.sample` / `*.conf.example` that a distribution installs as-is. And it bears
on the **weakest** part of the one curated table this repo has already published — §2.2's tier of
rows that rest on a shipped default and nothing else.

## Decision

**A shipped default is the configuration that takes effect, in the hand of the party being quoted.
Installing another party's bytes transfers operativeness and not ownership.**

Four limbs. They are stated in full, with their walk, as `sensitive-ports.md` **§12**; this ADR
carries the part that travels.

1. **The third form reads what takes effect, and what the party documents as its default — both.** A
   file the party does not install is not that party's shipped default, whoever wrote it. Where the
   file's own text disclaims operativeness, that disclaimer is the author's statement and settles it.
   Where the party documents a default elsewhere and the two disagree, **the documented default
   governs**.
2. **A directive is not a position; prose in a config file may be.** The documentation form takes a
   quotable position wherever the owner wrote it, and a comment in a shipped configuration file is
   the owner's prose exactly as a manual page is. It does not take a **directive** — an instruction
   to software attests only through the shipped-default form — nor a **label** describing what the
   directive beneath it does.
3. **Installation transfers operativeness, not ownership.** A distributor that installs upstream's
   bytes as its package's operative configuration holds a shipped default **of that package**, which
   attests **about that package**. Where the claims a table admits are claims about a thing the
   distributor did not design, its packaging **corroborates and is never sole grounds**.
4. **An example is silent in both directions.** It cannot admit and it cannot exclude, and in
   particular it is not a restricting act for the purposes of any exclusion route: an act that takes
   effect nowhere was not taken.

## Amendment — [#83](https://github.com/winniel123/verge-asm/issues/83), 2026-08-14

Limb 1 was written for a party's **artefact** disagreeing with its **documentation**. #83 met the
case it does not name: a party shipping **two defaults for one setting, both operative in one
binary**, with the operator's invocation selecting between them. Two readings of limb 1 travel out of
that case, and both are readings rather than new limbs.

1. **A default the party's own bytes label *legacy* or *deprecated* is not that party's shipped
   default, however operative it is.** Limb 1 requires both halves — the configuration that **takes
   effect** *and* the one the party **documents as its default**. A compatibility shim the owner
   routes operators away from satisfies the first half and fails the second. The test stays read off
   the artefact: the party says which is which.
2. **§10.4's one-way rule reads the default, not the artefact class that records it.** A permissive
   default is silent because *"the absence of an act is not a position"*, and that reason is a
   property of the default rather than of where it is written down. An owner's prose describing its
   own permissive default is the same silence in prose. Otherwise every documented permissive default
   re-enters through the documentation form and §10.4 is inoperative. This is limb 2 in its other
   direction: a **directive** is not a position, a **label** is not a position, and a **description
   of a permissive default** is not one either.

**[measured]** `kubernetes/kubernetes` at `v1.34.1` and `v1.36.3`: the kubelet's config API defaults
`anonymous-auth` to `false`, `authorization-mode` to `Webhook` and documents `readOnlyPort`
*"Default: 0 (disabled)"*, while `cmd/kubelet/app/options/options.go`'s `applyLegacyDefaults`
overwrites all three with `true`, `AlwaysAllow` and `10255` *"in order to preserve the command line
API"* — flags the same file marks deprecated with *"This parameter should be set via the config file
specified by the Kubelet's `--config` flag."* Both take effect; one is labelled legacy. Under this
amendment the shipped default is the restricting set, which moves `10250/tcp` from Class A to Class C
in `sensitive-ports.md` **without moving a `(port, transport)` pair**. The walk is `sensitive-ports.md`
**§18**; the argument for making this a new ADR instead lost on placement, since it refines limb 1
and travels exactly where limb 1 travels.

**Thin ground, flagged.** Limb 1 was measured across nine configuration artefacts from as many
projects before it was stated. This amendment has **one project** behind it — the deprecation label
appears three times, but all three are Kubernetes. It is stated because it is a reading of limb 1
rather than a new limb; the shallow measurement is worth knowing before it is relied on elsewhere.

## Amendment — [#105](https://github.com/winniel123/verge-asm/issues/105), 2026-08-14

Limb 1 was written for artefacts a party **writes**. It never asks *who performed the act that made
the configuration take effect*, because in a configuration file the party always did: a line is in
the file or it is not. [ADR-0061](./0061-a-comment-is-a-position-only-where-it-outlives-the-value-it-annotates.md)
limb 3 puts limb 1's two halves in two artefacts for a config API, and that split exposes the case
limb 1 does not name — **a struct field with no defaulting line still has a value**, produced by the
language rather than by the party. Three readings travel out of it, and all three are readings rather
than new limbs.

1. **A default's provenance is not read.** *Takes effect* is satisfied by the value the operator meets
   at first run, **however the shipped software arrives at it** — an assignment, a constructor, a build
   flag, or **a language's zero value**. Where inside the party's toolchain the value came from is not
   read, for the reason [ADR-0059](./0059-a-footing-tier-grades-evidential-distance-never-the-owners-conviction.md)
   limb 4 does not read the artefact a sentence was written in: the mechanism is a fact about the
   utterance, not a step between the utterance and the proposition. `sensitive-ports.md` §10.4's cost
   is **friction at first run**, and friction is a property of the running software rather than of its
   source's syntax — so a rule reading the assignment instead would be satisfied by a no-op line that
   buys nothing and refused by an identical binary that buys the same friction. Limb 4's *"an act that
   takes effect nowhere was not taken"* is the **converse** case and does not carry here; a zero value
   takes effect everywhere.
2. **The conjunction is where a zero value fails when it fails.** Limb 1 requires both halves, and a
   value the party has not published as its default fails the second one: there is no utterance to
   quote, and supplying one from the language's type system is `sensitive-ports.md` §2.2's opening
   sentence failing — *the claim may not be asserted by us*. This is the #83 amendment's half read
   for an absence rather than for a label. It is not a rule that documentation **converts** a value
   into an act; a documented default's comment may well be a **label** under ADR-0061 limb 1, and a
   label is exactly what the documentation half asks for.
3. **§10.4's one-way rule governs this form and not a claim's own measurement.** A permissive default
   is silent *as a position*. A value read to establish what the shipped artefact **does** —
   `sensitive-ports.md` §10.1's two steps, §10.4.3's *did the remedy reach the port?* — asks for no
   position, is governed by §20.6's owner rule and
   [ADR-0054](./0054-a-claim-step-is-answered-only-by-evidence-about-that-step.md)'s step rule, and
   reads a zero value in **both** directions, documented or not. **This limb is table-local**: §10.1's
   claim set is a theorem about one rule and
   [ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) rules it does not
   travel. Readings 1 and 2 travel with limb 1 as usual.

**[measured]** `kubernetes/kubernetes` at `v1.34.0`. `ReadOnlyPort` occurs **zero** times in
`pkg/kubelet/apis/config/v1beta1/defaults.go`, so `readOnlyPort`'s documented *"Default: 0
(disabled)"* is Go's zero value; the only place any maintainer writes a `ReadOnlyPort` value is
`applyLegacyDefaults`, which writes the **permissive** `10255`. In the same release
`k8s.io/kube-proxy`'s `enableProfiling` is also a zero value and carries **no documented default at
all**. **The two separate on the second half of limb 1, in the owner's own bytes** — one is published
as the default and the other is not — and `sensitive-ports.md` §16.5's and §27.2's opposite-looking
uses of a zero value are both confirmed. **No `(port, transport)` pair, row, class or footing tier
moves.** The walk is `sensitive-ports.md` **§34**; the argument for minting a new ADR lost on
placement, exactly as it did for #83 — this refines limb 1 and travels where limb 1 travels — and on
reading 3's reach, which ADR-0032 confines to one table.

**Thin ground, flagged.** The two candidate rules — *provenance is not read* and *only a written act
counts* — have the **same extension over the table today**, because the one cell a provenance rule
would refuse was superseded three sections earlier. The ruling is therefore decided on principle
rather than on a measured cell move, which is weaker than every prior ruling in this line
(ADR-0059, ADR-0061 and §32 each measured their alternative taking a neighbour with it). The nearest
thing to a measured cost is that a provenance rule cannot assess a default whose owner ships firmware
rather than source — `623/udp`'s, which `sensitive-ports.md` §28.10 admitted — and that is an argument
about reach rather than a cell. `sensitive-ports.md` §34.11 records this and the criterion that would
change the verdict.

## Rationale

**The costly-act test decides it, and it decides against the reading it appears to support.**
`sensitive-ports.md` §10.4 admits a restricting default because a restriction is a **costly act** —
it *"buys friction at first run and the maintainer paid for it anyway"*. The tempting argument for
admitting an example is that arranging which line is active is also a choice. It is, and it is a free
one: a file no daemon reads produces no first run, so its author pays nothing and supports nobody.
The cost is the whole reason the third form exists, and where the cost **is** paid — because a
distributor installed the file — the party who paid is the distributor, which is limb 3.

**The test is read off the artefact, not judged.** The objection that would sink limb 1 is that
*takes effect* sounds like an adjudication. **[measured]** across the nine configuration artefacts
retrieved for #69, every one declares its own status in its own bytes, and the two categories never
blur: net-snmp's *"An example configuration file … deliberately commented out, and will need to be
explicitly activated"* and RabbitMQ's *"This file is AN EXAMPLE. It is NOT MEANT TO BE USED IN
PRODUCTION."* against PostgreSQL's *"The commented-out settings shown in this file represent the
default values."*, CouchDB's *"Upgrading CouchDB will overwrite this file."* and Redis's *"So by
default we uncomment the following bind directive"*. **Surface syntax carries no information at all**
— a commented `listen_addresses` line in PostgreSQL's sample **is** the default, a commented
`agentAddress` line in net-snmp's example is the branch not taken, and a commented `-h 127.0.0.1` in
Debian's `rpcbind.default` is an offer to the operator. Only the file's own prose separates them,
which is why limb 1 keys on that and nothing else.

**Limb 3 is composed rather than invented.** #21's claims are answered from the specification or from
the owner — §10.1 says so in terms for the unauthenticated claim, §10.3 says so for the boundary
claim — so a distributor's artefact cannot reach any of them however operative it is. §10.5's
*"artefact, not party"* survives intact and keeps a distributor's **packaging** distinct from its
**security-guide prose**; what limb 3 adds is that the artefact route terminates at the owner gate.

**Why this is an ADR and not only a note edit.** ADR-0032 ruled that the attestation gate is the one
instrument that travels to other curated tables. This is a refinement of that gate, so it travels
with it: any future curated table admitting an owner's shipped default inherits all four limbs. Where
such a table's claims are claims about a **distributor's own artefact**, limb 3 does not bite —
because there the distributor *is* the owner, and limb 3 keys on who owns the claim rather than on
the word "distributor".

## Consequences

- **No `(port, transport)` pair moves.** `sensitive-ports.md` stays at **37 pairs**; §1's count, §3's
  class totals (12 / 7 / 18), §6.1's containment arithmetic and
  [ADR-0009](./0009-verge-core-is-a-union.md)'s union are all unchanged, each checked rather than
  asserted (§12.7). **No rule version moves and no evaluation `Break`s**
  ([ADR-0008](./0008-derivation-versions-move-on-content.md)) — this is free in the strong sense, not
  merely the vacuous-before-first-install sense.
- **One footing changes.** **[measured]** Apache Cassandra's shipped `conf/cassandra.yaml` carries
  *"For security reasons, you should not expose this port to the internet. Firewall it if needed."*
  immediately above both `native_transport_port: 9042` and `rpc_address: localhost`. §2.2's footing
  table says the weak tier has *"no prohibition … upstream"*; that is false for this row on the
  shipped bytes. **`9042/tcp` leaves the weak tier, which is now two rows rather than three.** A
  footing is evidence for a claim and not a claim, so the row keeps its class and its place.
- **`161/udp` does not come back.** #66 removed it on the owner requirement and the closed claim set,
  and this ADR touches neither. Had the ruling gone the other way the row would still fail.
- **Nothing in the negative space re-opens.** All sixteen of §4.6's exclusions were walked against the
  admitting reading, `111/tcp` first because it is the artefact the question was built on:
  **[measured]** Debian's `rpcbind.default` carries the loopback line **commented** under *"Uncomment
  the following line to restrict…"* while `rpcbind.socket` ships `ListenStream=0.0.0.0:111`
  **active** — the restriction was offered to the operator and not taken.
- **A published claim is now known to have been built the wrong way.** §2.2's footing table was
  derived from the projects' web documentation, and one file's bytes falsified one of its cells. The
  other rows were assessed the same way and their configuration bytes have never been read. Routed to
  [#70](https://github.com/winniel123/verge-asm/issues/70), which does **not** block
  [#12](https://github.com/winniel123/verge-asm/issues/12): a footing is not a claim, so a re-derived
  tier moves no row by itself.
- **Thin ground, flagged.** Limb 2's directive-versus-label line is the thinnest part. A comment that
  argues for its own directive sits between *"# Listen for connections from the local system only"*
  and *"you should not expose this port to the internet"*, and the only instrument offered for placing
  it is the existing position-versus-preference discrimination. A case in that gap should be ticketed
  rather than decided by whoever meets it.
  > **Discharged — [#100](https://github.com/winniel123/verge-asm/issues/100), 2026-08-14.** The case was
  > met by [#95](https://github.com/winniel123/verge-asm/issues/95) in a **third artefact class** — a
  > published config-API doc comment — ticketed rather than decided, and decided there.
  > [ADR-0061](./0061-a-comment-is-a-position-only-where-it-outlives-the-value-it-annotates.md): **a
  > comment takes a position only where it outlives the value it annotates.** Change the value: a comment
  > that remains true and operative is a **candidate position**; one that goes false or moot is a
  > **label**. Necessary and not sufficient — the position-versus-preference discrimination runs
  > **second**, on the survivor. **Limb 2 is unchanged.** ADR-0061 supplies the instrument limb 2 said it
  > lacked and explains limb 2's poles rather than correcting them: *a comment that argues for its own
  > directive* turns out **not** to be in the middle, because Cassandra's sentence goes on binding the
  > operator who overrides `rpc_address` while net-snmp's simply stops describing. **[measured]** the test
  > reproduces all nine comment verdicts in [`sensitive-ports.md`](../research/sensitive-ports.md), across
  > five projects and three artefact classes, eight of them at step one (§31.5). ADR-0061 also carries
  > what limb 1 does not reach: in a **config API** the third form's two limbs live in two artefacts, the
  > doc comment answers *documented as its default* only, and **the costly act is paid entirely by the
  > defaulting code**. **No row, footing, tier or figure moves.**

## Alternatives rejected

**The artefact reading — the distributor's install governs.** §10.5 keys on the artefact, and the
artefact the operator's daemon reads is the distributor's. It is a real argument and it is why the
question was ticketed. It loses three ways. It **answers a different question**: the forms are routes
to a claim about the protocol, not a description of the modal install, and deployment share is what
corroborator sources were refused for. It is **not stable under repackaging** — **[measured]** the
same Debian archive ships net-snmp bound to loopback and rpcbind bound to `0.0.0.0:111`, so the
verdict would depend on which of one distributor's files a reader opened. And it **re-opens the door
the corroborator rule holds shut**, one level down: a packaging choice would carry a normative row,
arriving through a `.deb` instead of a government PDF.

**A coherence gate — a restricting default attests only where the restriction describes a deployment
somebody actually runs.** This is the sharpest distinction available between PostgreSQL's
`listen_addresses = localhost`, which describes a real and common deployment, and a loopback-only
SNMP agent, which can be polled by no management station at all. **Refused on ADR-precedent ground**:
*does anybody run this?* is a judgement about deployment reality with no owner, and it is exactly the
shape §10.1 deleted when it removed *"would otherwise require authority"* for asking a reviewer to
imagine a counterfactual. It would also need frequency evidence, which the note excludes as a matter
of framing. **The criterion that would reopen it:** a candidate row passing on the owner's own
operative default where the same owner's documentation elsewhere calls that configuration unusable —
attested incoherence rather than reasoned incoherence.

**Withdrawing §10.5's distributor admission outright.** Limb 3 makes it inert for this table, so
deleting it is tempting. Refused: the sentence does real work distinguishing a distributor's
**packaging** from its **security-guide prose**, which is the distinction it was written for, and it
is live for any future table whose claims are about a distributor's own artefacts. Its reach is
stated instead of removed.
