# ADR-0088: A re-resolution check recurs only where its target can move — and a citation that carries no live cell is in no check's population

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#150 Name G8's population](https://github.com/winniel123/verge-asm/issues/150)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)
([#125](https://github.com/winniel123/verge-asm/issues/125)) split the curator's watch into a
**gate** over what is closed and a **queue** over what is open, and fixed the boundary between them:

> A check is **closed** where its population is enumerable and its evidence is bytes the project
> already holds, **plus a finite named set of targeted re-fetches.**

Eleven checks were specified on that basis. [#133](https://github.com/winniel123/verge-asm/issues/133)
then ran all eleven over the composed table and completed ten. **G8 — *every citation resolves, and
its quoted string is still a token of the artefact at the tag named* — could not be completed**, and
[`sensitive-ports.md`](../research/sensitive-ports.md) §40.9 recorded why in the design rather than in
the run:

> §39.6 calls G8 *closed* on the ground that its re-fetch set is finite and named. **It is finite and
> it is not named** — no section enumerates the citations G8 must re-resolve, and the note carries
> several hundred. The named subset run here was chosen by this section, **which is exactly the
> discretion a closed check is supposed to remove.**

**The failure is precise and it is not laziness.** G8's population had never been cut on anything, so
the only two candidates were *every citation in the note* — 457 unique URLs, 394 of them external —
and *whatever the running session picks*. The first is unpayable and the second is not a closed
check. **[measured]** the corpus's one external G8 measurement, §40.5's character-for-character check
of `security-checklist.md`, was made *"in the raw markdown **at the pinned commit**"* — against a
target that cannot move.

**A second question arrived mid-flight and is answered here rather than deferred.**
[#153](https://github.com/winniel123/verge-asm/issues/153), landing in the same wave, measured a
**summarising fetch fabricating a word in RFC 4251 §1** at the clause its own ruling turned on, and
recorded at §45.10 that G7's rider *"should be read as reaching **any** intermediary"* — leaving open
whether that earns a gate check. Naming *which* citations a release re-resolves is worthless if
*re-resolve* is undefined, so the two questions meet in this ADR: see rider 4.

The full working, including the roster, is
[`sensitive-ports.md`](../research/sensitive-ports.md) **§44**.

## Decision

**G8 asks two questions, and only one of them recurs.**

| Half | What it finds | Recurs? |
| --- | --- | --- |
| **(a) the quoted string is a token of the artefact** | We quoted something the artefact does not say — transcription error, fabrication, a string read off the wrong page | **Only against a target that can move.** Against a fixed target it is **idempotent**: run once, settled forever |
| **(b) the citation resolves** | The URL no longer returns the artefact | Yes, and **it decides nothing on its own** — a `404` is a server response and never a withdrawal (`sensitive-ports.md` §40.10) |

**So the rule:**

> **A citation is in G8's population exactly where a LIVE cell of a curated table rests on a string
> quoted from it, and it is re-resolved *every release* only where its target can move under its own
> name. Where the target is content-addressed, the check is an **entry gate on the edit** and never
> recurs.**

Three populations follow, and they exhaust the citation set:

| Population | Cadence | Membership |
| --- | --- | --- |
| **Internal** | Every gate run, **no fetch** | Every citation resolvable against repository bytes — a relative markdown link, an `ADR-00NN` token, a `§N` cross-reference, a `#N` ticket link. Total and mechanical; no roster is needed or possible |
| **Entry** | **Once**, by the edit that introduces or moves the citation | A citation whose target is **content-addressed**. Inherited by every later run and re-fetched by none |
| **Recurring** | Every gate run | A citation whose target is **moving** and which carries a live cell. **This is §39.6's *finite named set*,** and it is named at `sensitive-ports.md` §44.4 |

**The mutability test is a string test on the URL and needs no retrieval.** A target is
**content-addressed** where the citation names an object that cannot change under that name: an
**RFC number**; a **git blob or tree at a tag or 40-hex commit**; a **released archive verified
against a published checksum**; a **vendor document named by its own part, document or revision
number**; a **frozen per-release page**. Everything else is **moving** — a branch, a
`latest`/`stable`/`current` path, a release-line path the owner edits in place, a wiki, a vendor
portal page, a live registry, and any page with no version marker at all.

**Three riders.**

1. **A pin counts only where the note records it.** A citation to a tag the note does not name is a
   **moving** citation, because a run has nothing to fetch. The error therefore runs towards a larger
   recurring roster and never towards a missed check.
2. **A citation naming more than one target is as many members as it names targets.**
3. **A cell whose ground is an *absence* is in no G8 population.** Half (a) has no string to test; a
   dated negative is re-swept under
   [ADR-0046](./0046-a-negatives-corpus-is-its-owners-class-list-and-only-a-sole-ground-negative-is-exposed.md)
   as amended by [#93](https://github.com/winniel123/verge-asm/issues/93), which is a **reading** and
   therefore a queue item.
4. **Every member is resolved as the artefact's own bytes.** A member resolved through any
   intermediary that does not return them — a summarising reader, a search snippet, a mirror, a
   paraphrase — is **UNRUN**, never green, per §40's own convention that an incomplete check is
   recorded as incomplete and never as passed.

**Rider 4 sites an old rule and mints nothing.** Its specifying site is `sensitive-ports.md` §33.11 —
*"cite the bytes the repository returns, because a renderer and a summariser are two different
substitutions"* — and **[measured]** it has fired at least six times in the corpus (§12.9, §22.10,
§30.10, §33.11, §37.14, and §45.10, where a summarising fetch reported a word RFC 4251 §1 does not
contain, at the clause [#153](https://github.com/winniel123/verge-asm/issues/153)'s ruling turned on).
**It is a precondition on the fetch and not a check**, and specifically it is **not G7's**: G7 asks
*did the **owner** publish this?*, its authority is
[ADR-0045](./0045-an-owners-documentation-is-what-it-has-issued.md), and §45.10 records that G7 was
**vacuous by construction** on the artefact where the hazard fired. G8's own words already exclude an
intermediary — *still a **token** of the artefact* is a claim about **bytes** — so a member read
through one has not tested the proposition at all. **This lives in the population statement because
§39.6 has specified *targeted re-fetches* since #125 without ever saying how a re-fetch is performed,
having had no list of them to attach the discipline to.** This is the first list.

**And the corollary that makes the rule worth having rather than merely smaller: pinning is a cure,
and it is free at the moment it is made.** A pass already reading an owner artefact is already
holding the tag; writing the tag into the citation moves that member from the recurring roster to the
entry set **permanently**. Nothing in the corpus previously rewarded pinning at all, which is why
nothing did it systematically.

## Rationale

**The two checks had been carrying each other's question.** *Is this still what the owner says?* is
the **tag comparison**, and **G11** owns it and ran it to completion over all 27 footing cells
(§40.4). A citation to a moving target answers G8's question and G11's at one fetch, so it reads as a
G8 obligation; a citation to a pinned target hands the currency question wholly to G11 and keeps none
of it. **G8's set looked unnameable because it was carrying G11's**, and the two separate cleanly the
moment the population is cut on mutability.

**Reliance, not readership, is the key.** A curated table's gate protects **cells**. A citation
behind a removed row (`161/udp`, `1433/tcp`, `9200`, `9300`), a withdrawn clause, a **corroborator**
that §2.3 already bars from being a ground, or a method paragraph carries **no cell** — so its
failure moves nothing the gate is for. That is not a hole grudgingly accepted; it is what
*corroborator* and *withdrawn* mean.

**Two options lost.**

- **Every citation in the note.** It is finite and nameable, so it satisfies ADR-0057's letter. It
  loses on **relevance first** — it re-resolves the artefacts behind rows this project has removed,
  where a failure has no consequence — and on **cost second**: it would make G8 dearer than the other
  ten checks together, and *a check a release cannot pay is a check a release skips*, which is the
  precise failure the closed/open partition exists to prevent.
- **The live cells' citations, undifferentiated.** The honest runner-up, and half of this decision. It
  loses because it re-runs, every release, checks that **cannot fail** — roughly half the live-cell
  set. A gate spending its budget on questions with known answers is `sensitive-ports.md` §39.3's
  *reading budget spent backwards*, one instrument over.

**A third option is worth naming because it is the one a session reaches for first.** A note's own
`## Sources` bibliography looks like a citation roster and is not one: it records what was **read**,
never what is **relied on**, and it deliberately carries entries citable only for what they do **not**
contain.

## Consequences

- **G8 is closed in ADR-0057's own sense for the first time**: its population is enumerable, its
  evidence is bytes the project holds, and its re-fetch set is finite **and named** —
  `sensitive-ports.md` §44.4, **36 members over 30 owners**, against G11's 27 cells over 16 owners.
- **The gate stays at eleven checks.** Naming a population is not adding a check, and this ADR adds
  none. Whether the gate needs a twelfth is
  [#149](https://github.com/winniel123/verge-asm/issues/149)'s and
  [#152](https://github.com/winniel123/verge-asm/issues/152)'s question and is untouched here.
- **`sensitive-ports.md` §39.6's G8 row is amended at its clause**, per ADR-0057's *a withdrawal that
  supplies no replacement does not hold* and
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s test: the
  row read alone in the present tense licensed a run to choose its own subset. §40.9's first bullet is
  marked **discharged**.
- **This reaches every curated table, not only the sensitive-port list.** ADR-0032 attaches an
  evidence standard to a table; ADR-0057's gate is the instrument over any of them. A table with no
  citations has an empty recurring roster, which is a legible state.
- **Two cells are unpinnable and therefore permanently recurring** — MongoDB publishes no revision
  marker of any kind (§36.14) and the memcached wiki has no releases. **[measured]** those are also
  the cells G11 cannot compare, which is the same fact and is raised at §44.11 rather than ruled on.
- **G7, G10 and G11 are confirmed by use and none is amended.** G10's named list of one is a member of
  the recurring roster, so **one fetch answers G8 and G10 for that cell**.
- **§45.10's open question — whether the summariser hazard earns a gate check — is answered in the
  negative.** It is rider 4, a precondition on the fetch. **G7 is not widened and not amended**, and
  the gate stays at eleven.
- **The partition pays a second time, and it was not foreseen.** The artefacts a session is most
  tempted to reach for an intermediary on are the ones plain fetching fails on — HPE's portal serving
  an empty body, Dell's PDF host returning `403`, the JavaScript shells at `learn.microsoft.com` and
  `docs.oracle.com`. **[measured]** all four vendor-document members are in the **entry** set, pinned
  by part and revision number, so **a release never fetches them**. The place the temptation is
  strongest is the place the recurring roster does not go.
- **Nothing on any operator-facing surface moves.** `CONTEXT.md` is not amended: the curator is not a
  subject in the model, and the product holds nothing about a citation. ADR-0008's rule version is not
  triggered, and no row, class, tier, coverage or exclusion figure moves.

## Alternatives rejected

- **Score citations by the queue's rung ladder (ADR-0057) instead of by mutability.** Tempting,
  because the ladder already grades volatility. It loses because the ladder grades **how loudly an
  artefact announces a change** and G8 asks **whether the artefact can change under its name at all** —
  a rung-5 specification and a rung-1 dataset are both immutable once published under a number, while
  a rung-3 shipped default is immutable at a tag and mutable at a branch. **The rung ladder is about
  the owner's act; this cut is about the citation's form.**
- **Sample the roster and disclose a bounded residue,** as the queue does under
  [ADR-0078](./0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md). Refused: a bounded residue
  is the instrument for a population that **cannot terminate**, and this one does. Sampling a
  terminating population is the discretion ADR-0057 built the gate to remove, and it would move G8
  from the gate to the queue on a cost argument alone.
- **Re-resolve on a slower cadence — every *n*th release — rather than partitioning.** It restores
  discretion in the time axis instead of the membership axis, and it makes *which release ran G8* a
  fact a later reader must reconstruct. A check that is either run or inherited is legible; a check
  that is *due* is not.
