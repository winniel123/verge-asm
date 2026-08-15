# ADR-0099: A stated horizon is a second comparand, and a tag match does not discharge it

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#152 G11 against a stated horizon, not only a tag](https://github.com/winniel123/verge-asm/issues/152)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) fixed **G11** as one of the
eleven gate checks: *"For every footing cell, the owner's current release tag against the tag the cell
was read at; a difference marks the cell **stale-against-tag**."* [#133](https://github.com/winniel123/verge-asm/issues/133)
ran the gate whole for the first time and, at `sensitive-ports.md` §40.4, met a cell where the check's
own stated comparison — read-at tag against current tag — returns **current**, and something is still
wrong:

> `2375/tcp`'s Class A ground cites Docker's deprecation page for *"Unauthenticated TCP connections"*,
> recorded as **deprecated v26.0, target removal v28.0**. Docker is at `docker-v29.7.2` — **past its own
> stated removal target** — and §13.1 read the shipped bytes at exactly that tag while §35 measured the
> enforcement directly (*"yes — a code path"*), so the pair demonstrably still exists. **The row stands
> and nothing moves.** What is missing is a sentence saying so, and G11 does not supply one: it compares
> a tag to a tag, never a tag to a horizon the artefact itself states.

This is not the shape [#149](https://github.com/winniel123/verge-asm/issues/149) and
[ADR-0077](./0077-a-second-ground-counts-only-where-it-would-have-carried-the-cells-proposition-alone.md)'s
Rationale 2 name. That gap is **G11 having no comparand at all** — *"vacuous for an artefact with no tag
to diff … a continuously-published page with no version pin"* — which is a rung-1/rung-2 artefact
problem, and `sensitive-ports.md` §43.6 item 5 measures it as reaching the register's whole rung-1 head.
`2375/tcp` is not that shape. It is tagged, G11's fetch succeeds, and the tag comparison is correctly
**equal** (§40.4's table: `2376+2375 Docker | as read | equal | Current`). **The gap here is a second,
independent fact the same ground states — a horizon — that a passing tag comparison gives no reason to
look at.**

The full working is `sensitive-ports.md` **§49**.

## Decision

| Concern | Decision |
| --- | --- |
| **Is the row wrong?** | **No.** §35 measured the enforcement directly, at the current tag. This ADR does not touch the row, the class, the footing tier or the coverage figure |
| **Does G11 need to change?** | **No — G11's own definition is left exactly as ADR-0057 wrote it and exactly as §40 ran it.** §40 is the gate's baseline; redefining G11 after its baseline run would retroactively change what that green/red verdict meant, which is the silent-regeneration failure ADR-0057 §5 exists to prevent |
| **What closes the gap** | **A new, sibling gate check — named here but not numbered.** Numbering a twelfth check is reserved to the merge session per the batch's shared constraint (ADR-0057: a twelfth check must terminate, and at most one may be minted). This ADR calls it **(new gate check, number pending merge)** |
| **What the new check tests** | For every graded cell whose held ground quotes an owner-declared **horizon** — a specific future release, version or date at which the owner states a named behaviour will change (a deprecation-to-removal target, an end-of-life or sunset date, a support-window close) — compare the owner's **current** tag or date, already fetched by G11's own re-fetch, against that horizon. Where current ≥ horizon and no subsequent retrieval records whether the promised change occurred, mark the cell **horizon-passed-unverified** |
| **Is it closed (terminating)?** | **Yes.** Its population is enumerable from bytes already held — cells whose cited ground text contains explicit horizon language — and it introduces **no new re-fetch class**: the "current" value is the same tag G11 already retrieves. Measured today, the population is **one**: `2375/tcp` |
| **Is it the same check #149 needs?** | **No, on the evidence read for this ticket — see Rationale 2.** #149's gap is the *absence* of any comparand for a whole artefact class (rung 1/2, no tag at all). This gap is a comparand that *exists and passes*, alongside a second, independent fact the same ground states that nothing compares at all. Flagged for the merge session rather than asserted as settled, because a generalised formulation exists that could subsume both — named in Rationale 3 and deliberately not chosen here |
| **First run over its own population** | **Green.** `2375/tcp` is horizon-passed, and §35 already performed — by hand, on a ticket — exactly the retrieval this check would have raised as a question. Formalising the check does not change the row; it makes the next occurrence of this shape find itself rather than wait for a ticket that happens to be reading the cell |

## Rationale

### 1. A tag match is not a currency claim about every fact the ground carries

G11's proposition is narrow by design — *has the owner's release line moved past the tag we read* —
and ADR-0057 built it that way on purpose: a check that tried to re-verify arbitrary prose against
current bytes would not terminate (ADR-0057's Decision table: *"two not at all"* among the eight
triggers, one of them *"a primary source changing position"*). A **horizon** is not arbitrary prose. It
is a structured, self-contained proposition the owner states about its **own** future release line, in
the owner's own units — exactly ADR-0043's *bound the evidence in the subject's own units* — and it
sits right next to the value G11 already fetches. Refusing to compare it because G11's existing
comparison happened to pass is a category error: *stale-against-tag* and *past-the-owner's-own-stated-
horizon* are two different predicates over the same two numbers.

### 2. Why this is not #149's gap, argued against the temptation to merge them

The two look alike from a distance — both are *"published in the owner's own metadata, missed by our
tag comparison"* — and the batch's own working note (`sensitive-ports.md` §43.6 item 5) files both
under one heading, *the gate's reach*. Argued closely, they diverge on the axis that decides whether a
check terminates cheaply:

- **#149's artefact class has no tag at all.** The check that would close it must manufacture *some*
  currency signal for a rung-1/rung-2 artefact — §133's own discovery that Microsoft Learn's `ms.date`
  and `updated_at` disagree is the shape of what that signal would be. Its population is *every
  rung-1/rung-2 register member*, which §43.6 item 5 measures as the register's **whole head** — large,
  and growing as the register grows.
- **This ticket's artefact is tagged, and the tag comparison is not the failure.** The population is
  *cells whose ground quotes a horizon*, which is a much narrower predicate — measured at **one** cell
  today — and is orthogonal to whether the artefact carries a tag at all. A rung-5 specification could
  state a horizon just as easily as a rung-3 shipped default.

A single check that tried to do both — *"check every temporal fact a ground states, tagged or not"** —
is the generalised version the merge session might reach for, and this ADR names it so it is visible
rather than silently pre-empted, but does not choose it: making the population *every temporal
comparand a ground happens to state* trades a finite, named population for one that can only be found
by re-reading prose with judgement, which is the closed/open line ADR-0057 draws precisely to keep the
gate terminating. Keeping the two checks separate keeps both populations enumerable by a mechanical
rule (*has a version tag* / *cites a horizon*) rather than by a reader's discretion.

### 3. Why a sibling check and not a G11 amendment

G11 was **run to completion** at §40 before this ticket existed, and its baseline verdict — the thing
"a later run is compared to" (ADR-0057 §40.6's own words) — is stated over a specific test: tag against
tag. Widening that test's definition after the fact would change what §40's recorded **GREEN**/`Current`
verdict for `2375/tcp` means without re-running anything, which is exactly the *sentence that names no
successor is re-derived by the next session that needs one* failure ADR-0057's own rationale 5 catalogs.
A new, separately-named check keeps §40's baseline legible: G11 stays *tag against tag*, and a second
check owns *tag against stated horizon*.

### 4. The population is thin, and thin is not the same as absent

One measured instance is not much to build an instrument on. It is kept anyway, on the same warrant
ADR-0057 used for G3, whose population was also one row at first run (`sensitive-ports.md` §40.5): a
check with a population of one is fully specified and fully falsifiable — a second horizon-bearing cell
either matches its test or it doesn't — and a check is not required to have a large population to
terminate. The alternative, folding this into a disclosure sentence rather than a mechanical check, is
addressed in Alternatives rejected.

## Consequences

- **No row, class, footing tier or coverage figure moves.** `2375/tcp` stands exactly as §35 and §40.4
  measured it.
- **G11's own definition is unchanged**, and its `sensitive-ports.md` §40 baseline is unchanged and
  unre-interpreted.
- **A new gate check is described but not numbered.** `sensitive-ports.md` §49 states it in full; the
  merge session assigns the final number, in the same pass that resolves whether it is one check with
  [#149](https://github.com/winniel123/verge-asm/issues/149)'s or two.
- **[`docs/spec/curated-table-watch.md`](../spec/curated-table-watch.md) §3's *"the gate is eleven
  checks, G1–G11"* is not edited here.** That document's gate-ledger section is explicitly reserved to
  [#133](https://github.com/winniel123/verge-asm/issues/133)'s shape and is the merge session's to
  update once the check is numbered, not this ticket's.
- **`CONTEXT.md` is not amended**, on ADR-0057's own last Decision row: the curator is not a subject in
  the model.
- **ADR-0057 and ADR-0077 are confirmed by use and neither is amended.** This ADR adds a check
  alongside G11; it does not redefine either ADR's decisions.

## Alternatives rejected

| Option | Why it lost |
| --- | --- |
| **Redefine G11 to also compare against a stated horizon** | Rewrites a check's meaning after its baseline run (§40) without re-running it, which is the silent-regeneration shape ADR-0057's rationale 5 was written to stop. A reader of §40's `2375/tcp` row *Current* verdict would no longer know which test that verdict answered |
| **A single merged check covering both this gap and #149's** | The generalised formulation is real and is named in Rationale 2, but its population — *every temporal comparand any ground states, tagged artefact or not* — is not enumerable by a mechanical rule; it requires reading prose with judgement to find, which is the open/closed line ADR-0057 draws to keep the gate terminating. Kept separate so each check's population stays a finite, named, mechanically-discoverable set |
| **Leave it as a disclosure sentence, not a check** | §40.4 already is that disclosure, and it already shows the failure mode: a sentence written once, in a walk section, has no mechanism that re-asks the question on the next edit. ADR-0057's rationale 5 — *a sentence that names no successor is re-derived by the next session that needs one* — is exactly the risk a prose-only fix leaves open |
| **Defer entirely — population of one is too thin to act on** | Refused on the same warrant G3 was kept at population one (`sensitive-ports.md` §40.5). A check does not need a large population to be fully specified, and the population is expected to grow as more grounds are read for horizon language, not shrink |
| **Route it to the queue instead of the gate** | The row is not de-attested and no artefact needs re-founding — this is squarely a currency question about an already-standing row, which ADR-0057 assigns to the gate (G11's own job description) and not to the de-attestation queue |
