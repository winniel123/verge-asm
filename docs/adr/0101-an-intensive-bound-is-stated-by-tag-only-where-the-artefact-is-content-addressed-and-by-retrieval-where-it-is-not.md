# ADR-0101: An intensive bound is stated by tag only where the artefact is content-addressed, and by retrieval where it is not

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#157 The first ambiguous class boundary in an intensive bound](https://github.com/winniel123/verge-asm/issues/157)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0078](./0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md) gave each residue-ledger entry an
**intensive bound** — per item read, *"the artefacts opened, and the class boundary of that opening"* —
worked as *"this owner's issued documentation at tag `v1.37.0` and its release notes, and no other
class."* [`curated-table-watch.md`](../spec/curated-table-watch.md) §2.2 restates the form and its
falsification test: naming one artefact **inside** the stated class that the reading did not open.
Drawing that class is a judgement, not a lookup, and two prior sections flagged the same failure mode for
it without ruling it.

**`sensitive-ports.md` §39.9**, on the register's own rung ladder: *"the boundary between rungs 2 and 4 is
where it will first fail. Continuously published with no version pin and issued in a versioned
documentation set are distinguishable on today's five instances and will not stay so: a vendor that
versions its documentation site and then edits a prior version in place sits in neither. The ladder
survives that by being read off the artefact — is there a retrievable prior version? — which is a
question with a byte answer. It is flagged because the first ambiguous case should be **ticketed rather
than decided by whoever meets it**, which is [ADR-0061](./0061-a-comment-is-a-position-only-where-it-outlives-the-value-it-annotates.md)'s
own precedent."*

**`sensitive-ports.md` §42.9** read the same failure mode one instrument over, into the intensive bound
specifically: *"The class boundary in the intensive bound is a judgement, and it is the same judgement
the rung ladder makes. … The ladder survives by being read off the artefact, and so does this: is there a
retrievable prior version of the thing I opened? is a question with a byte answer. The first ambiguous
case should be ticketed rather than decided by whoever meets it."* Neither section ruled the case; both
deferred it. This ticket is that deferral, discharged.

**The first instance is measured, not hypothetical.** [`curated-table-watch.md`](../spec/curated-table-watch.md)
§1.1 carries `3306/tcp`'s footing and claim cells at **rung 4**, grounded on Oracle's *Security
Guidelines*, MySQL Reference Manual §8.1.1 — the page at `dev.mysql.com/doc/refman/8.4/en/security-guidelines.html`.
That same citation is independently a member of [ADR-0088](./0088-a-re-resolution-check-recurs-only-where-its-target-can-move.md)'s
G8-recurring roster (`sensitive-ports.md` §44.4, item 7), described there in terms: *"A release-line path
the owner edits in place."* The path names a release (`8.4`) exactly as rung 4's *"issued prose in a
versioned documentation set"* expects, but ADR-0088's own mutability test — already run over this exact
citation, for an unrelated purpose — finds it **moving**, never content-addressed: rung 4's defining half,
*"the prior version stays retrievable"*, is not merely undemonstrated for this artefact, it is measured
false. §39.9's predicted failure is sitting on a live register member.

**Not re-derived.** Whether `3306/tcp`'s **rung** should move, and whether `sensitive-ports.md` §39.3's or
[ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)'s rung ladder should itself be
redrawn on ADR-0088's mutability test rather than on the *"announced"* language it currently uses, are
both left as live successor questions and are not ruled here. This ADR rules only how the **intensive
bound's class** is stated once an item is read — the judgement §2.1 part 3 leaves open, not the judgement
the queue's rung ladder makes.

The full working is [`curated-table-watch.md`](../spec/curated-table-watch.md) **§2.2.1**.

## Decision

| Concern | Decision |
| --- | --- |
| **The first ambiguous case** | `3306/tcp`'s footing and claim cells (rung 4, `curated-table-watch.md` §1.1), grounded on MySQL's `dev.mysql.com/doc/refman/8.4/en/security-guidelines.html` — a page [ADR-0088](./0088-a-re-resolution-check-recurs-only-where-its-target-can-move.md) already measured as **moving** (`sensitive-ports.md` §44.4 item 7), despite sitting at rung 4 |
| **The general rule** | **An intensive bound states its class by version only where the opened artefact is content-addressed under ADR-0088's own mutability test** — a tag, a commit, a checksum, a vendor part or revision number, or a frozen per-release page the owner does not revise. **Where the artefact is a moving target under that same test** — including one whose path names a release line the owner edits in place — **the class is stated by the artefact and the reading's own retrieval** (its URL or path, and our basis commit or fetch date), **never by the path's nominal version** |
| **Is §39.9's "retrievable prior version?" a new test** | **No.** It is [ADR-0088](./0088-a-re-resolution-check-recurs-only-where-its-target-can-move.md)'s mutability test, asked of a second consumer. *A retrievable prior version exists* and *the target is content-addressed* are the same fact, stated twice for two different instruments |
| **Does the rung decide the class's form** | **No.** ADR-0088's own Alternatives already refused the rung ladder as G8's axis — *"the rung ladder is about the owner's act; this cut is about the citation's form"* — and this ADR extends the same refusal to the intensive bound. `3306/tcp` sits at rung 4 and its class is drawn as a moving target's, on the artefact's own measured mutability, not on the ladder's placement |
| **Does this move `3306/tcp`'s rung** | **No.** Rung is unchanged; `sensitive-ports.md` §39.3 and ADR-0057's ladder are not reopened |
| **Does this reopen ADR-0088** | **No.** Confirmed by use, at a second consumer. Its ruling on G8's population and cadence is untouched |
| **Does this reopen ADR-0078 or §2.1's entry form** | **No.** Part 3's shape — a class boundary, falsified by naming one un-opened artefact inside it — is unchanged. This rules how the class is **drawn** where §2.1 left it a judgement, exactly as [#151](https://github.com/winniel123/verge-asm/issues/151)/[ADR-0077](./0077-a-second-ground-counts-only-where-it-would-have-carried-the-cells-proposition-alone.md) ruled a gloss on ADR-0077's own step 2 without reopening its four-step test |
| **Does a moving artefact lose its place in the queue** | **No.** [ADR-0088](./0088-a-re-resolution-check-recurs-only-where-its-target-can-move.md) §44.7 already names members with **no version to pin at all** — MongoDB, the memcached wiki — as permanently moving, and neither is thereby refused a reading. This ADR refuses only the **tag form** of the class, never the reading itself |

## Rationale

### 1. The byte question already had an instrument, built for a different consumer

§39.9 and §42.9 each independently asked for a test with *"a byte answer"* and neither built one, because
[ADR-0088](./0088-a-re-resolution-check-recurs-only-where-its-target-can-move.md) had already built it,
eleven sections earlier in the same note, for G8's re-fetch cadence: *"a target is content-addressed where
the citation names an object that cannot change under that name … everything else is moving — a branch, a
`latest`/`stable`/`current` path, **a release-line path the owner edits in place**, a wiki, a vendor
portal page, a live registry."* That phrase is not an analogy to §39.9's *"a vendor that versions its
documentation site and then edits a prior version in place"* — it is the same fact, named first. **This
ADR does not mint a test; it points two flagged questions at an answer the corpus already paid for.**

### 2. Rung and mutability are already known to disagree, and this ADR only names where

[ADR-0088](./0088-a-re-resolution-check-recurs-only-where-its-target-can-move.md)'s Alternatives-rejected
table considered and refused scoring G8 by the rung ladder: *"a rung-5 specification and a rung-1 dataset
are both immutable once published under a number, while a rung-3 shipped default is immutable at a tag
and mutable at a branch. The rung ladder is about the owner's act; this cut is about the citation's
form."* `3306/tcp` is the measured instance of exactly that disagreement running the other direction: a
rung-**4** item — the ladder's own *"issued prose in a versioned documentation set"* — whose citation is
mutability's **moving**, not content-addressed. Reading the intensive bound's class form off the rung
would get this case wrong on the same axis ADR-0088 already refused to conflate, which is why this ADR
extends ADR-0088's refusal rather than re-deriving a rung-shaped answer.

### 3. A class over a moving target does not terminate, and that is the disease the intensive bound was built to cure

[ADR-0078](./0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md)'s founding sentence: *"we read item
3" is unfalsifiable, because reading an owner's corpus does not terminate.* A class stated as *"MySQL's
Reference Manual at 8.4, and no other class"* reintroduces exactly that non-termination one level down:
the owner can add or edit pages inside that path indefinitely without a version bump, so *"name one
artefact inside it we did not open"* never closes — the path is not a boundary, it is a label on an
open-ended set. Stated instead as *"this page, as our repository's basis held it at commit `<sha>` (or
fetched on `<date>`), and no other page,"* the class is a fixed set of bytes again, exactly as a genuine
tag makes it, and the falsification test terminates the same way it does for a content-addressed artefact.
**[measured]** this is the same near-miss `curated-table-watch.md` §2.4 already caught once, in miniature,
for breadth rather than mutability — the `623/udp` class drafted as *"HPE's, Dell's and NEC's BMC-security
documentation"* admitted an unopened document (HPE's iLO 6 brief) and had to be narrowed to the three
documents actually opened. A class too wide on **breadth** and a class unpinned on **mutability** are the
same failure — an unfalsifiable class — measured on two different axes of the same form.

### 4. Why an ADR rather than a note applied in place

[ADR-0061](./0061-a-comment-is-a-position-only-where-it-outlives-the-value-it-annotates.md)'s own test for
minting rather than declining: *the load-bearing rule was available nowhere.* Two sections
(`sensitive-ports.md` §39.9 and §42.9) named the failure and could not state the rule that resolves it —
the same shape §31 and §16.5 were in before ADR-0061, reaching *label* by citation with no criterion
behind it. Here the resolving rule (ADR-0088's mutability test) existed but had never been pointed at this
question, so minting names the pointer rather than inventing new machinery, and travels the same way
ADR-0061 does: any future curated-table entry drawing an intensive bound on a moving artefact inherits it
without re-deriving it.

## Consequences

- **[`curated-table-watch.md`](../spec/curated-table-watch.md) gains §2.2.1**, stating this ruling at the
  intensive bound's own site and citing `3306/tcp` as the worked instance.
- **§1.1's `3306/tcp` rows gain a citation pointer** to §2.2.1, so a future reading of that cell does not
  re-discover the ambiguity from bytes alone.
- **No row, class, tier, rung, or coverage figure moves.** `3306/tcp` stays at rung 4 in the register;
  nothing in `sensitive-ports.md` is edited by this ADR.
- **[ADR-0088](./0088-a-re-resolution-check-recurs-only-where-its-target-can-move.md) is confirmed by use**
  at a second consumer and is not amended — its ruling on G8's population and cadence is untouched.
- **[ADR-0078](./0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md) and
  [ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) are confirmed by use and not
  amended.** Neither the entry form nor the rung ladder is reopened.
- **[ADR-0008](./0008-derivation-versions-move-on-content.md) is not triggered.** No reference-data figure
  moves; this is governance content.
- **`CONTEXT.md` is not amended**, on ADR-0057's and ADR-0078's own reason: the curator is not a subject
  in the model.
- **A successor question is named and left open**: whether `sensitive-ports.md` §39.3's rung ladder should
  itself be redrawn on ADR-0088's mutability test, rather than on its current *"announced"* language, now
  that the two are measured to disagree on a live register member. Not decided here — the ticket's own
  scope excludes re-deriving the ladder.

## Alternatives rejected

| Option | Why it lost |
| --- | --- |
| **Read the class's form off the item's rung** (rung 4 → tag form, rung 2 → retrieval form, no further test) | Tempting because rung is already computed per item and costs nothing further to read. It loses on the measured case: `3306/tcp` sits at rung 4 and is mutability's **moving**, so reading the class form off the rung gets exactly this item wrong — the predicted failure realized rather than avoided. [ADR-0088](./0088-a-re-resolution-check-recurs-only-where-its-target-can-move.md) already refused the rung ladder as the axis for this shape of question, for the same reason |
| **Leave it a judgement, decided by whoever writes the entry** | The status quo, and precisely what §39.9 and §42.9 each asked not to happen — [ADR-0061](./0061-a-comment-is-a-position-only-where-it-outlives-the-value-it-annotates.md)'s own history is the measured cost: two sections reached the same verdict by citation and neither could state the rule it was applying |
| **Add a fourteenth rung, or split rung 4 into pinned/moving sub-rungs** | Rung is [ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)'s own axis, keyed on the revision act for the queue's tie-break order. The intensive bound's need — how to phrase a class once an item is read — is orthogonal to queue priority, and folding it into the ladder reopens ADR-0057 for a purpose it was never built for, against [ADR-0088](./0088-a-re-resolution-check-recurs-only-where-its-target-can-move.md)'s own precedent of keeping the axes separate |
| **Bar a moving artefact from an intensive bound entirely, requiring a pinned artefact before an item may be read** | Too strong. [ADR-0088](./0088-a-re-resolution-check-recurs-only-where-its-target-can-move.md) §44.7 names members with **no version to pin at all** — MongoDB, the memcached wiki — as permanently moving; barring them from ever being read would make those cells permanently unreadable, which no ADR in this corpus has done and [ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) refused in general (*"direction of failure: toward more watching," never toward less*) |
| **State the class by the path's own nominal version anyway, and let G11 catch drift later** | Conflates two different instruments. G11 answers *is this still what the owner says now* — a currency question, re-run every release. The intensive bound answers *what did this reading actually open* — a one-time, dated fact. A class stated by a version the artefact does not actually hold is false the day it is written, not merely stale later, which is a different and worse failure than the one G11 exists to catch |
