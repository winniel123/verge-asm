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
and an owner's artefact is a document about **its own** subjects. Those two sets overlap; they are not
the same set; and a retrieval keyed on the first can only ever return the intersection.

## Decision

**An attestation is retrieved over the artefact, not over the row. Every subject the artefact names is
a candidate for the table, whether or not the table already holds it.**

Three limbs.

1. **Read the whole artefact.** Where an owner's artefact is opened to attest a row, it is read end to
   end for **every** subject in the table's domain that it names — every port it names, for
   `sensitive-ports.md`; every algorithm or key size, for a cryptographic table. A retrieval that
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
  `sensitive-ports.md` stays at **37 pairs**; §1's count, §3's class totals (12 / 7 / 18), §6.1's
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
  §11.9 and §12.9 each record *what was retrieved*; none states *how much of each artefact was read*.
  They are not withdrawn — nothing in them is measured wrong — but they may not be cited as evidence
  about the parts of those artefacts nobody looked at. §11.8 already wrote this rider for one negative
  finding; limb 3 makes it general.
- **It travels to every curated table under ADR-0032.** The five-row weak-key table
  ([#68](https://github.com/winniel123/verge-asm/issues/68)) is the live instance: a NIST document
  opened to attest SHA-1 names every other digest in the same tables, and
  [ADR-0035](./0035-a-cryptographic-primitives-owner-is-its-specifier.md)'s owner definition makes
  each of them a candidate subject.

## Alternatives rejected

**Leave it as a retrieval habit rather than a rule.** The objection is that *read the whole file* is
obvious enough not to need an ADR, and that writing it down is ceremony. It is refused on the
measurement: #69 was a careful, well-executed ticket, run under a standard that already demanded
shipped bytes, and it missed two ports in a file it had open. An instrument that a good session fails
to apply is not obvious; and the failure is silent, because a row-scoped retrieval produces a clean,
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
first; a second would make it a pattern rather than an instance.
