# ADR-0037: An attestation is retrieved over the artefact, not over the row

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#70 Is §2.2's footing table right? It was built from web docs, and the shipped config bytes have never been read](https://github.com/winniel123/verge-asm/issues/70)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) ruled that an evidence
standard attaches to a **curated table**, and that the gate which travels is *state the claim, and
cite the source that owns it*. [ADR-0036](./0036-a-shipped-default-is-the-configuration-that-takes-effect.md)
then said **which artefacts count** as that source's shipped default and which are examples. Between
them they answer *whose bytes* and *which bytes*. Neither says **how much of a counting artefact must
be read**, and until now every retrieval on this effort has answered that by habit: open the file,
look at the row you came for.

[#69](https://github.com/winniel123/verge-asm/issues/69) did exactly that, correctly, and it worked —
it opened Apache Cassandra's shipped `conf/cassandra.yaml`, found *"For security reasons, you should
not expose this port to the internet. Firewall it if needed."* above `native_transport_port: 9042`
and above `rpc_address: localhost`, and moved `9042/tcp` out of `sensitive-ports.md` §2.2's weak tier
on it. It recorded the sentence as appearing **twice**.

**[measured]** It appears **four** times, at `cassandra-5.0.2` — the tag #69 read — and identically at
`cassandra-5.0.6` and `cassandra-5.0.9`. The two occurrences #69 did not record sit immediately above
`storage_port: 7000` and `ssl_storage_port: 7001`, and **neither port is on the list**. The retrieval
was scoped to the rows the table already held, so it could confirm those rows and could not find
anything else. The sentence that would admit two new rows was in the file the whole time, sixty lines
above the one that was quoted.

The generalisation is not about Cassandra and not about YAML. A curated table is a set of subjects,
and an owner's artefact is a document about **its own** subjects. Those two sets overlap. They are not
the same set. A retrieval keyed on the first can only ever return the intersection.

## Decision

**An attestation is retrieved over the artefact, not over the row. Every subject the artefact names is
a candidate for the table, whether or not the table already holds it.**

Three limbs.

1. **Read the whole artefact.** Where an owner's artefact is opened ~~to attest a row~~ **in the
   table's service — for any purpose the table's gates require: attestation, determinacy, ownership,
   class-list construction or claim analysis** (the narrow trigger is **WITHDRAWN** by the
   [#95](https://github.com/winniel123/verge-asm/issues/95) amendment below) — it is read end to
   end for **every** subject in the table's domain that it names — every port it names, for
   `sensitive-ports.md`, and every algorithm or key size, for a cryptographic table. A retrieval that
   stops at the row it came for is **not a negative result about the rest of the file**, and must not
   be recorded as one.
2. **A subject the artefact names and the table lacks is a finding, and it is ticketed rather than
   admitted.** The artefact supplies the attestation limb only. Every other gate the table imposes —
   the claim, determinacy, exclusion grounds — is unexamined at the moment of discovery, and the
   retrieval that found it was not scoped to answer them. The finding is **recorded with its
   measurement and routed**, never swept into the same ruling.
3. **A negative retrieval states its own scope.** [#66](https://github.com/winniel123/verge-asm/issues/66)
   established that *a negative retrieval is a verdict*. It is a verdict about what was read. So a
   negative finding records **the artefacts opened and the extent read**, and a session may rely on it
   only inside that extent.

## Rationale

**The asymmetry is the whole argument.** A row-scoped retrieval is sound in one direction and blind in
the other. It can falsify a cell — which is what #69 did, and what made it valuable — and it can
confirm one. It cannot discover a subject, because it never looks anywhere a subject it does not
already have could be. So a table built by row-scoped retrieval converges on being *correct about what
it contains* while its **coverage** never improves, and coverage is the half a curated list is actually
judged on: `sensitive-ports.md` §4.6 says in terms that *"the exclusions are as much of the deliverable
as the list"*.

**It is cheap where it matters.** The extra cost is reading a file already retrieved. The saving is
measured: the same pass that produced this ADR found `7000/tcp` and `7001/tcp`
([#75](https://github.com/winniel123/verge-asm/issues/75)) and, by the same discipline applied to a
table rather than a file, that §2.2's footing table places 19 of 37 pairs
([#76](https://github.com/winniel123/verge-asm/issues/76)). Neither needed a new retrieval.

**Limb 2 is what keeps it from being a licence.** The tempting reading of limb 1 is that a strong
sentence found beside an unlisted subject admits it. It does not, and `7000/tcp` is the demonstration:
IANA registers it to `afs3-fileserver`, so `sensitive-ports.md` §2.4's determinacy gate is live and may
sink the row **despite** the strongest kind of sentence the corpus has. Attestation is one gate of
several, and a retrieval answers one gate.

**Limb 3 already had a near-miss.** #69's *"nine artefacts, nine self-declarations"* is a genuine
finding and its scope is nine files. **[measured]** A tenth, `etcd.conf.yml.sample` at
`etcd-io/etcd` `v3.7.1`, opens *"This is the configuration file for the etcd server."* — an operative
self-description on a file named `.sample` that nothing installs. ADR-0036 limb 1's second sentence
resolves it without strain, because etcd documents its defaults elsewhere. But the 9-for-9 count
would have read as a law rather than as a sample of nine had its extent not been stated.

## Consequences

- **No `(port, transport)` pair moves and no rule version moves.** This is a method, not a table edit.
  `sensitive-ports.md` stays at **37 pairs**. §1's count, §3's class totals (12 / 7 / 18), §6.1's
  containment arithmetic and [ADR-0009](./0009-verge-core-is-a-union.md)'s union are unchanged, each
  checked rather than asserted (`sensitive-ports.md` §13.8). No `Break`
  ([ADR-0008](./0008-derivation-versions-move-on-content.md)).
- **§2.2's footing table is confirmed, cell by cell, against shipped bytes** — the first re-derivation
  of it from the artefacts it asserts about rather than from the web pages it was built from.
  `sensitive-ports.md` §13.
- **Two candidate rows are now on the record**, priced as an addition rather than a footing move —
  a version bump plus an aperture widening, vacuous before the first install — and therefore blocking
  [#12](https://github.com/winniel123/verge-asm/issues/12), on exactly the terms #66's removal was
  priced. [#75](https://github.com/winniel123/verge-asm/issues/75).
- **The retrieval records this repo already holds are, by their own new standard, scoped.** §9.5,
  §11.9 and §12.9 each record *what was retrieved*. None states *how much of each artefact was read*.
  They are not withdrawn — nothing in them is measured wrong — but they may not be cited as evidence
  about the parts of those artefacts nobody looked at. §11.8 already wrote this rider for one negative
  finding. Limb 3 makes it general.
- **It travels to every curated table under ADR-0032.** The five-row weak-key table
  ([#68](https://github.com/winniel123/verge-asm/issues/68)) is the live instance: a NIST document
  opened to attest SHA-1 names every other digest in the same tables, and
  [ADR-0035](./0035-a-cryptographic-primitives-owner-is-its-specifier.md)'s owner definition makes
  each of them a candidate subject.

### Note — [#91](https://github.com/winniel123/verge-asm/issues/91): limb 2 completes its first full cycle, and limb 1's trigger is narrower than its rationale

**Limb 2 has now been run end to end.** [#83](https://github.com/winniel123/verge-asm/issues/83) read
`ports-and-protocols.md` whole for `10250`'s claim and ticketed the three subjects it named that the
table lacked ([`sensitive-ports.md`](../research/sensitive-ports.md) §19.8).
[#91](https://github.com/winniel123/verge-asm/issues/91) disposed of all three — **`10259/tcp` and
`10257/tcp` admitted, `10256/tcp` refused** (§24). **Both halves of limb 2's design held.** The
candidates were not admitted where they were found, and the gates unexamined at discovery did real
work: determinacy passed for all three and decided none of them, while the **claim** gate — the one
the discovering retrieval was not scoped to answer — split the set two-to-one. This is the pattern
`7000/tcp` predicted, with the split falling on a different gate.

**A trigger question this ADR should be read as leaving open.** Limb 1 fires *"where an owner's
artefact is opened **to attest a row**"*. §24 opened `pkg/cluster/ports/ports.go` for **determinacy**,
not attestation, and it names three numbers the table does not hold — `10249` kube-proxy metrics,
`10248` kubelet healthz, `10258` cloud-controller-manager. Limb 1's *letter* does not reach a
determinacy retrieval. Its **rationale** — a retrieval keyed on the rows you already have returns only
the intersection — reaches it exactly. §24.11 records the by-catch on the rationale and routes it on
limb 2, and ~~declines to widen the limb, on the ground that one instance is not a measurement. **The
criterion that would force the widening:** a second determinacy retrieval turning up a candidate the
table lacks.~~ `10249` is the first, and it looks strong — **[measured]** kube-proxy's metrics server is
also entirely unauthenticated and its only protection is a loopback default.

> **The question is CLOSED and the stated criterion is SPENT** — the
> [#95](https://github.com/winniel123/verge-asm/issues/95) amendment below widened limb 1, and on a
> **different** and stronger measurement than the one named here: the *first* determinacy retrieval
> paid out. Nothing in this note is now open. Marked here rather than only at the amendment, per
> [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened by
> [#106](https://github.com/winniel123/verge-asm/issues/106).

### Amendment — [#95](https://github.com/winniel123/verge-asm/issues/95): limb 1's trigger is widened, and limb 2's second cycle is complete

> **Limb 1 fires wherever an owner's artefact is opened *in the table's service*, whatever the
> retrieval was for.** Read limb 1's *"where an owner's artefact is opened **to attest a row**"* as
> **where an owner's artefact is opened for any purpose the table's gates require** — attestation,
> determinacy, ownership, class-list construction or claim analysis. The narrow trigger is
> **withdrawn**.

**The forcing measurement is not the one the note above predicted, and it is stronger.** The criterion
asked for *a second determinacy retrieval turning up a candidate*. What happened instead is that the
**first one paid out**. [#95](https://github.com/winniel123/verge-asm/issues/95) disposed of all three
of §24.11's candidates: **`10249/tcp` admitted to Class A on Claim 1, `10248/tcp` admitted to Class C
on Claim 3, `10258/tcp` refused** ([`sensitive-ports.md`](../research/sensitive-ports.md) §27). §24's
determinacy retrieval was decisive of nothing by its own account — *"determinacy is not what decides
this ticket"* — and its by-catch produced **two pairs**, a second `Break` under
[ADR-0008](./0008-derivation-versions-move-on-content.md), and a second widening of
[ADR-0009](./0009-verge-core-is-a-union.md)'s union. **A trigger that would have permitted `ports.go`
to be read row-scoped is a trigger that would have cost the list two rows and left a third
unexamined.** The rationale was always purpose-blind. The letter's narrowness is now measured rather
than argued.

**#95 also found the same defect inside a file this note had already opened**, which is the second
independent ground. `pkg/proxy/apis/config/v1alpha1/defaults.go`'s `getDefaultAddresses` returns the
healthz and metrics addresses **in the same four lines**, and `sensitive-ports.md` §24.3 quoted it for
`10256` — the wildcard half — while the loopback half that founds `10249`'s footing sat beside it.
That is this ADR's own Cassandra case, in a Kubernetes file, met by a pass that was reading for
determinacy.

**Three riders keep the widening from being an expansion.**

1. **It adds no retrieval.** The obligation is to read a file already fetched — this ADR's own *"it is
   cheap where it matters"*. The rejected alternative *"require re-reading every artefact already
   retrieved, now"* is **not** re-opened and stays refused on cost and ordering.
2. **Limb 2 is unchanged and is what keeps it from being a licence.** #95 is the demonstration: two of
   the three candidates needed a **claim** analysis the discovering pass was not scoped to run, and
   the third needed a build-target check that produced
   [ADR-0056](./0056-a-port-constant-in-a-library-is-not-a-shipped-listener.md).
3. **The domain clause is untouched.** Limb 1 reads *every subject in the table's domain that it
   names*. A determinacy retrieval over an owner's artefact enumerates ports, not every identifier in
   the file.

**Limb 2 has now completed two full cycles, and both split the same way.** #83 → #91: three ticketed,
two admitted, one refused. #91 → #95: three ticketed, two admitted, one refused. In both, determinacy
passed for every admitted candidate and decided none of them, and the **claim** gate — the one the
discovering retrieval was not scoped to answer — did the separating. That is limb 2's design working
twice, and it is the argument for leaving limb 2 exactly as it is.

**The option that lost: widen limb 2 instead of limb 1.** Its case is that limb 2 is already
purpose-neutral in its own words and carried §24.11 without strain. **It loses because limb 2 is the
*disposal* rule and limb 1 is the *reading* rule** — limb 2 tells a session what to do with a subject
it found and cannot make it find one. The defect §24.11 exposed is that a session may legitimately
stop reading, and only limb 1 addresses that.

**Limb 3 gains an instance in the direction nobody reads it.** §24.11's by-catch table said `10249`
serves pprof. **[measured]** at `v1.34.0` it does not, `enableProfiling` having no defaulting function
and no documented default, and neither `flagz` nor `statusz` is installed either. A **by-catch record
is a scoped retrieval too**, and *state your own extent* binds a positive finding as much as a
negative one. `sensitive-ports.md` §27.2 carries the correction.

## Alternatives rejected

**Leave it as a retrieval habit rather than a rule.** The objection is that *read the whole file* is
obvious enough not to need an ADR, and that writing it down is ceremony. It is refused on the
measurement: #69 was a careful, well-executed ticket, run under a standard that already demanded
shipped bytes, and it missed two ports in a file it had open. An instrument that a good session fails
to apply is not obvious. The failure is silent, because a row-scoped retrieval produces a clean,
confident, correct-as-far-as-it-goes result with nothing in it to signal what was not read.

**State it as a rule about *files* rather than about *artefacts*.** Narrower and easier to check, and
it loses the case that produced the second finding: §2.2's footing **table** was read the same
row-scoped way — every cell in it was checked and nobody counted the cells against the list — and the
gap that fell out ([#76](https://github.com/winniel123/verge-asm/issues/76)) is the same defect one
level up. The word is *artefact* because the table is one too.

**Require re-reading every artefact already retrieved, now.** Sound in principle and refused on cost
and on ordering: the artefacts behind the rows on the list have been read for the rows on the list, and
what a full re-read would buy is **coverage** — new candidate rows — which is a different job from
verifying the list, has no deadline that ADR-0008 imposes, and belongs to the map's *how the tiered
port sets are curated* patch, where #66 already lodged *a row's footing was never checked against the
standard it is now held to* as a backlog with an end. **The criterion that would force it earlier:** a
second unlisted subject found beside a listed one in an artefact already retrieved. `7000/tcp` is the
first. A second would make it a pattern rather than an instance.
