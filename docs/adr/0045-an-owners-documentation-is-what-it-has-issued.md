# ADR-0045: An owner's documentation is what it has issued, not what it has drafted

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#86 Is an owner's unreleased document the owner's documentation? — and does 9092/tcp become a row if it is?](https://github.com/winniel123/verge-asm/issues/86)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) ruled that of
[#21](https://github.com/winniel123/verge-asm/issues/21)'s three gates only the middle one — *state
the claim, and cite the source that owns it* — generalises to other curated tables. That gate is
[`sensitive-ports.md`](../research/sensitive-ports.md) §2.2, and it admits three forms: a
**specification**, the project's or vendor's **own documentation**, and the project's **shipped
default, as documented by the project**.

[ADR-0036](./0036-a-shipped-default-is-the-configuration-that-takes-effect.md) settled the **third**
form's seam — the same bytes can be an example in one party's hand and an operative configuration in
another's — by keying it to what **takes effect without the operator acting**.
[ADR-0037](./0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md) then settled
**how much** of a counting artefact must be read.

**Nobody has settled the second form's equivalent seam.** *The project's or vendor's own
documentation* has never said whether *documentation* means what the project **publishes** or what
sits in its **tree**, and until now no case turned on it.

A case does now. [#79](https://github.com/winniel123/verge-asm/issues/79) §17.5 opened the one
document class nobody had opened for Apache Kafka and found the sentence `sensitive-ports.md` §4.6
says does not exist:

> "**Security is off by default.** A freshly-installed Apache Kafka cluster accepts unauthenticated
> `PLAINTEXT` connections on every listener and applies no authorization. This is appropriate only for
> closed test environments. Production deployments **must** explicitly configure authentication,
> authorization, and transport encryption before being exposed to any untrusted network."
> — `apache/kafka`, `docs/security/security-model.md`, `trunk`

§4.6 excludes `9092/tcp` on *"Upstream declines to take any network posture"*, and §10.3 and §12.6
each re-confirmed it on that ground. The sentence exists and **the document is not issued**: it is in
no release tag, in no release artefact, and on no published page. Two readings of one clause, opposite
verdicts, one file — which is [#69](https://github.com/winniel123/verge-asm/issues/69)'s shape exactly,
one form over.

The full working, with the enumerated corpus and every quote and absence checked against retrieved
bytes, is [`sensitive-ports.md`](../research/sensitive-ports.md) **§18**.

## Decision

**An owner's documentation is what the owner has issued. A document the owner has drafted but not
issued attests nothing, in either direction.**

Five limbs. They are stated in full, with their walk, as `sensitive-ports.md` §18; this ADR carries the
part that travels.

1. **The second form reads an *issued* document.** A document reaches the second form where the owner
   has put it in front of readers: carried by a release artefact the owner publishes — a source or
   binary distribution, or a documentation artefact released alongside one — **or** served on the
   owner's own published documentation site. **The prose analogue of ADR-0036's *takes effect without
   the operator acting* is *reaches a reader without the reader acting on the owner's behalf*:** a
   reader following the owner's own documentation route, at a version the owner has released, arrives
   at the sentence.
2. **A document in a development or release branch is a draft until it is issued** — whatever it
   contains, whoever committed it, and however settled it looks. A version-control history is the
   owner's **working record**, not the owner's documentation.
3. **Issuance is per version, and a negative states its version.** A finding under this limb records
   the release and the retrieval date and claims nothing beyond them — *not issued at `X`, not served
   on `<site>` on `<date>`* — never *never issued*. This is
   [ADR-0037](./0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md) limb 3 applied
   to an absence.
4. **Silent in both directions, and not corroboration.** An unissued document cannot admit a row and
   cannot exclude one, and it is **not** a corroborator under §2.3 — the corroborator tier is for the
   wrong **party** speaking correctly, and this is the right party at the wrong **time**. It is
   recorded as a **named reopening criterion** carrying its measurement, which is
   [ADR-0040](./0040-a-specifications-silence-is-not-the-owners-silence.md)'s bounded residue rather
   than a caveat.
5. **Issuance is not rendering, and publication is not weakened by also being committed.** Either
   route suffices and neither requires that any reader use it: a document carried in a release artefact
   but absent from the owner's website is issued, and a document served on the website and absent from
   the artefact is issued. The ordinary case is both, and that case was never in doubt.

## Rationale

**The costly-act test decides it, and it is the same instrument that decided ADR-0036.**
`sensitive-ports.md` §10.4.2 admits a restricting default because a restriction is a **costly act** —
it *"buys friction at first run and the maintainer paid for it anyway"* — and ADR-0036 refused the
example config because *"a file nobody's daemon reads produces no first run, so it costs its author
nothing and it is cheap talk"*. Transpose it with one word changed. **A document nobody's reader
reaches produces no reader.** What a *published* position costs its author is precise: it is what users
cite back, what downstream distributors are held to, and what the maintainers must defend when somebody
deploys against it. An unissued file can be softened, rescoped or reverted before any of that is owed,
so it is cheap talk in exactly ADR-0036's sense. **[measured]** the commit that created the Kafka file
is titled *"MINOR: A starting point for a formal security model"* — the project's own word for it is *a
starting point*, and its own severity for it is `MINOR`.

**Limb 2 is refused-alternative-shaped, and the alternative fails ADR-0036's own stability test.**
ADR-0036 refused the distributor reading partly because it was **not stable under repackaging** —
**[measured]** one Debian archive shipping net-snmp on loopback and rpcbind on `0.0.0.0:111`, so the
verdict would depend on which of one distributor's files a reader opened. The committed reading fails
the same test in a sharper form, because the instability is *within one project*: **which ref?**
**[measured]** `docs/security/security-model.md` is present at `trunk` and at branch `4.4` and absent at
branches `4.3`, `4.2` and `4.1` — one project, five refs, two answers, on the day of retrieval. And the
committed reading has no non-arbitrary stopping point on the way down: a merged commit on `trunk` is
authored by the owner's committers under the owner's review, and so is an open pull request, and so is a
commit that is reverted next week. Issuance has a stopping point the project itself draws, with a vote
and a signature.

**It does not read a limb in that was not there.** ADR-0036's headline is that upstream's reading
governs *"because §2.2's third form already has two limbs"*. The second form has one, and the work is
done by the noun: **a *documentation* is a thing an owner issues to readers**, and a file in a branch is
a draft *of* one. Reading *documentation* as *any prose in the owner's repository* is the reading that
adds something — it would make the second form reach an artefact class the phrase has never named,
and it would do so in the admitting direction, which is the direction §2's whole apparatus exists to
resist.

**Limb 4 is the option that came closest, and it is stated as a refusal rather than left implicit.**
The tempting middle is to give an unissued owner document the status ADR-0036 limb 3 gives a
distributor's packaging: corroborates under §2.3, never sole grounds. It loses twice. It is **inert
wherever it would matter** — corroboration cannot carry a row, and every row this could touch is a row
with *no other owner statement*, which is exactly why the question arose; so its output is identical to
limb 4's on every case in the corpus while costing a tier a later session must reason about. And it
**mislabels the object**: §2.3's corroborators are parties without ownership speaking correctly.
Recording an unissued owner document as corroboration says the problem is *who* spoke, when the problem
is that nobody has been spoken to.

**The direction-of-error argument was made and it is answered by pricing, not by the gate.** A false
negative on a *sensitive* list means the product is silent about a real exposure, and an open
unauthenticated Kafka is the product's own use case. But the silence is **this release's**, not
permanent: it is cured automatically when the owner issues the document, at which point the row is a
candidate under a criterion that is now written down and cheap to check, and priced as a **new
admission** under [ADR-0009](./0009-verge-core-is-a-union.md) exactly as §17.5 already routed it.
Against that, the committed reading admits rows on sentences the owner has not decided to stand behind,
and a curated list cannot defend one of those when the file is reverted. `sensitive-ports.md` §4.6 says
in terms that *"the exclusions are as much of the deliverable as the list"*.

**Why this is an ADR and not only a note edit**, on ADR-0036's own ground. ADR-0032 ruled the
attestation gate is the one instrument that travels to other curated tables. This is a refinement of
that gate, so it travels with it: any future curated table admitting an owner's documentation inherits
all five limbs. The live instance is already on the record — **[measured]**
[ADR-0040](./0040-a-specifications-silence-is-not-the-owners-silence.md) §2 records #73's corpus as
including *"17 active drafts"*, and an Internet-Draft is the IETF's unissued document class. No row in
`weak-key-and-signature.md` rests on one — that table's three resolving statements are RFC 9325 §4.5,
RFC 9846 §C.2 and RFC 8550/8551 §6, all published RFCs — so nothing moves there; but the table had no
rule saying a draft cannot carry, and now it does.

## Consequences

- **No `(port, transport)` pair moves and no rule version moves.** `sensitive-ports.md` stays at **37
  pairs**; §1's count, §3's class totals (12 / 7 / 18), §6.1's containment arithmetic (28 + 4 + 5),
  §2.2's footing table (13 + 10 + 3 + 11 = 37) and [ADR-0009](./0009-verge-core-is-a-union.md)'s union
  are unchanged, each checked rather than asserted (`sensitive-ports.md` §18.6). No `Break`
  ([ADR-0008](./0008-derivation-versions-move-on-content.md)) — `sensitive-port-reached-from-internet`
  is byte-identical, so this is free in the strong sense and not merely the vacuous-before-first-install
  sense.
- **`9092/tcp` remains excluded and the list is definitively 37.** §4.6's stated reason narrows a
  second time: *upstream has taken a network posture, in its repository, and the document taking it is
  in no release artefact and on no published page.* **[measured]** absent from
  `kafka_2.13-4.3.1-site-docs.tgz` — the ASF's own released documentation artefact, SHA-512 verified
  against the published sum — and 404 at every `kafka.apache.org` path tried, against 200 and 59,446
  bytes of real prose for the page that carries the neutral sentence.
- **The reopening criterion is live and near, which is new.** **[measured]** the file is on the `4.4`
  **release branch**, byte-identical to `trunk` at 15,499 bytes, and **no `4.4.x` tag exists** — the
  latest tag is `4.3.1`. §17.5 wrote the criterion as a hypothetical; it is one release away.
- **Two findings about the same document are recorded and deliberately not relied on**, per ADR-0037
  limb 2. **[measured]** the document **names no port number at all** — `9092`, `port` and `internet`
  each occur zero times — so an issued version of it would meet
  [#88](https://github.com/winniel123/verge-asm/issues/88)'s question before it reached a row. And read
  end to end it carries a *Known Non-Findings* section stating that *"an open `PLAINTEXT` listener with
  no authorizer is a deployment choice, not a defect"*, while its headline sentence's remedy sits **on
  the port** — which is §11.5's refused structure. Both bear on the **claim** gate, which this
  retrieval was not scoped to answer. `sensitive-ports.md` §18.4.
- **It reaches four tickets running concurrently, and re-decides none of them.**
  [#83](https://github.com/winniel123/verge-asm/issues/83) — limb 5 settles that a generated reference
  page served on the owner's site is *issued*, leaving #83's actual axis (is a generated page the
  owner's documentation or a rendering of code?) untouched.
  [#87](https://github.com/winniel123/verge-asm/issues/87) —
  [ADR-0042](./0042-a-squat-is-contested-where-the-other-convention-is-live.md) keys *live* to the
  competing owner's **current documentation**, and limb 1 says which of that owner's documents count.
  [#88](https://github.com/winniel123/verge-asm/issues/88) — this ADR is **upstream** of it: limb 1 asks
  whether the document counts, #88 asks whether its category sentence reaches a number, and both must
  pass. [#84](https://github.com/winniel123/verge-asm/issues/84) — reached only as a check to run before
  quoting an `erlang/otp` `master` page rather than a released one.
- **It gives ADR-0032 §8's etcd flag its vocabulary, and moves no row.** §16.9 flags `2379`/`2380` on a
  prohibition *"three months old and absent from the `3.5` and `3.6` lines"*. Limb 3 says that precisely:
  the footing is **issued at `v3.7.1` and not at `3.5` or `3.6`**. Whether the watch list should key on
  tier or on evidence age is the map's patch and is untouched here, and the row is
  [#76](https://github.com/winniel123/verge-asm/issues/76)'s.
- **The map's curation patch gains a seventh trigger, and it is the cheapest one it has.** A drafted
  owner document **becoming issued** changes a footing with nothing in the world having moved. It is
  watchable in the sense the patch requires — the criterion names the artefact and the release — and
  unlike ADR-0032 §8's silent de-attestation it is announced, by a release.
- **`CONTEXT.md` is not edited**, on ADR-0036's and ADR-0040's precedent and for their reason. Nothing
  here changes a glossary term.

## Alternatives rejected

| Alternative | Why not |
|---|---|
| **The committed reading — a file in the owner's repository is the owner's documentation** | The argument is real: a merged commit is unambiguously the owner speaking, where an example config is unambiguously not, so the analogy to ADR-0036 is imperfect. It loses on ADR-0036's **own** instruments. The costly-act test refuses it — an unissued document costs its author nothing until a reader can reach it. The stability test refuses it — **[measured]** one project, five refs, two answers. And it has no stopping point above an open pull request, which is the door §2.3 holds shut arriving one level further down than ADR-0036 found it |
| **A corroboration tier — an unissued owner document corroborates under §2.3, never sole grounds** | The closest option and the one worth stating. **Inert where it matters**, because every row it could touch has no other owner statement, so its output equals limb 4's on the whole corpus; and it **mislabels the object**, since §2.3's tier is for the wrong party speaking correctly and this is the right party at the wrong time. The honest home is a named reopening criterion, which ADR-0040 already provides |
| **Decide it per document, on how settled the draft looks** | This is *does anybody run this?* in prose — a judgement about a document's maturity with no owner, which is the shape `sensitive-ports.md` §10.1 deleted and ADR-0036 refused a second time as a coherence gate. Issuance is read off the artefact: a tag, a distribution, an HTTP status |
| **Admit `9092/tcp` on the sentence and route the standards question afterwards** | ADR-0037 limb 2 forbids it, and it would settle this question silently in the direction that happens to add a row. #79 named this as the option it refused, and it is refused again on the same ground |
| **Refuse `9092/tcp` on the ground that the file is unreleased, without stating the rule** | Symmetrically wrong, and it is what #79 declined to do. The verdict is unchanged either way; what a rule buys is that the next case is decided by a standard rather than by whoever meets it |
| **Fold this into ADR-0036** | ADR-0036's subject is which **configuration** counts as a party's shipped default — its four limbs are about operativeness, directives and installation. This one is about which **prose** counts as an owner's documentation. They are siblings under ADR-0032's gate and they answer different forms; merging them would bury the second form's rule inside a section about config files, which is where a session looking for it would not look |
| **Require both routes — a document must be in a release artefact *and* on the published site** | Refused as over-tight and unstable in the other direction. **[measured]** projects legitimately publish only one way: the ASF ships `site-docs` artefacts and serves a site, but many owners do one or the other, and a rule that failed a document served on the owner's site because no tarball carries it would exclude most vendor documentation this table already rests on. Limb 5 states the disjunction explicitly so no later session tightens it by accident |
