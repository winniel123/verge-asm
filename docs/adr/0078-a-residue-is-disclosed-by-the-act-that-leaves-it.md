# ADR-0078: A residue is disclosed by the act that leaves it — and a residue over our own list is enumerated rather than described

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#136 Where does the curated-table queue's bounded-residue disclosure live?](https://github.com/winniel123/verge-asm/issues/136)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)
([#125](https://github.com/winniel123/verge-asm/issues/125)) split the curator's watch into two
instruments — a **gate** over what is closed and a **queue** over what is open — and gave the release
one obligation towards the queue:

> What a release owes the queue is a **disclosure**, never a quantum. … What is owed is
> [ADR-0040](./0040-a-specifications-silence-is-not-the-owners-silence.md)'s **bounded residue** —
> the queue's head, how far down it read, and what it did not read, falsifiable by naming one item.

That sentence states an obligation and sites nothing. **Three things were left unstated and #136 is
all three**:

- which *document* carries the disclosure
- whether it is a per-release statement, a standing section of a curated table's own note, or something
  the **gate** emits
- and — the part that decides whether it is a disclosure at all — **how it states its bound**.

**The screen question is not this question and is not reopened.**
[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §7 keeps the weak-tier
disclosure off every interface on four grounds — it is *severity arriving labelled as honesty*, it
does not vary per subject, it names no act the operator can take, and its real consumer is the
**curator**. #125 confirmed §7 untouched and its condition unchanged, and
[ADR-0016](./0016-an-annotation-moves-a-message-never-a-number.md)
([#117](https://github.com/winniel123/verge-asm/issues/117)) closed the `Annotation` question without
firing §7's conditional. **This ADR moves nothing on to any screen and touches neither the condition
nor the four grounds.**

**The corpus has exactly one residue form, and every instance of it is over somebody else's corpus.**
ADR-0040 fixed it — *the corpus actually searched, enumerated; what was found and which rows it
reached; and the smallest extension of the corpus that could still change the answer* — and ADR-0032
§7 carries it as the disclosure obligation's third limb, under the #73 amendment. It has fired four
times: `weak-key-and-signature.md` §13's ~340-document sweep, ADR-0069's control-label corpus,
`passive-discovery-sources.md` §11's four probed encodings, and
[`sensitive-ports.md`](../research/sensitive-ports.md) §16.9's `4369` paragraph — which is the one
instance that has been **falsified in the way the form promises**: §16.9 named the two documents it
read, [#84](https://github.com/winniel123/verge-asm/issues/84) named a third outside that boundary,
and §20.4 withdrew the residue. *"The falsification clause is the watch"* (§20.4).

**What nobody has asked is what changes when the population is our own.** All four instances bound a
corpus that exists independently of us and cannot be enumerated, so the disclosure can only *describe*
its boundary. The queue is a list **we author**, over cells **we hold**, and it is enumerable. Reading
a boundary description on to an enumerable population is how a **count** gets back in — and §39.2 bars
the queue's length from being quoted as an indicator, because five consecutive membership changes were
invisible to it.

The full working is [`sensitive-ports.md`](../research/sensitive-ports.md) **§42**.

## Decision

| Concern | Decision |
| --- | --- |
| **Which document carries the disclosure** | **The release's own account of its curated tables** — [`docs/spec/curated-table-watch.md`](../spec/curated-table-watch.md), one **dated entry per release**, appended. Not a curated table's research note, not an ADR, and not the operator-facing release note |
| **Per-release statement, standing section, or gate-emitted** | **A per-release statement, appended and never rewritten.** The other two lose, and they lose for different reasons |
| Why not a standing section of `sensitive-ports.md` | **A residue is a fact about a spend, not about a state.** Read in the present tense one release later, a standing residue statement is *false* — it asserts a boundary that has moved. That is [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s failure shape written into the instrument on purpose, in the corpus where it is most measured: G4 found two superseded sentences standing in the present tense **in the document that specifies G4** (§39.8) |
| Why not **gate-emitted** | **Three independent reasons, and the third is fatal.** *Wrong clock* — the gate fires on the **edit** and the residue is owed by the **release**, so a release that edits no table (which is exactly when the residue is most informative) would owe nothing. *Wrong faculty* — a machine can attest what it **checked**, never what a person **read**; emitting *"we did not read these"* from *"no ticket cites these"* asserts a claim on a proxy, which §2.2 bars. *Wrong definition* — the gate **is** what terminates ([ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md): *a release can check what terminates and can check nothing else*); making it emit the disclosure of what does **not** terminate collapses the distinction the whole instrument is built on, and what a machine would emit is a **length** |
| What the gate does contribute | **G11's `stale-against-tag` marks are an input and are cited in the entry** as the reason an item was read or passed over. Refused as the emitter, kept as an input — ADR-0057's own pattern with support count and evidence age |
| **One disclosure, or one per curated table** | **One.** The queue is **one order across four curated tables**; *how far down it read* is a fact about the **order**, and no table's document can state it. A per-table residue re-creates the per-table sub-queues ADR-0057 refused, and `weak-key-and-signature.md` — which contributes **zero** items — would owe a statement of nothing, which is a **caveat**, which ADR-0032 §7 forbids |
| **Is ADR-0032 §7 reopened** | **No, in either half.** Its *no screen* half binds this disclosure and is restated unchanged. Its *stays in the instrument's own document* half is about the **weak tier**, whose subject is a **row of that table**; the residue's subject is a **reading budget over a cross-table order**, which is nobody's table. §7 is **distinguished, not amended** — and it is marked at its own site so the distinction does not have to be re-derived |
| **How the bound is stated** | **Three parts, and part 3 is what ADR-0040 never had to say.** (1) **The order read against** — the queue register as of the release's tag, cited, never re-enumerated. (2) **The extensive residue** — every item **not read**, **named** by its `(cell, artefact, revision act)` triple. (3) **The intensive residue** — per item **read**, which of the owner's artefacts were opened and the **class boundary** of that opening |
| **Why the extensive residue is named rather than described** | **Because the population is ours.** ADR-0040 describes a boundary because somebody else's corpus cannot be enumerated. The queue can be, so a described boundary is a **count in disguise** and §39.2 bars it. **Where the population is our own and enumerable, the residue is named member by member; a described boundary is a weaker disclosure than the population permits** |
| **Why an intensive residue is needed at all** | **Because reading an item does not terminate.** *We read item 3* is unfalsifiable. *We opened this artefact at this tag and its release notes, and no other artefact of this owner* is falsified by naming one artefact **inside** the stated class that the reading did not open. ADR-0040's form, unchanged, applied one level down — at the item rather than at the list |
| **The falsification test, stated as a test** | **Two, one per residue.** The extensive residue is falsified by naming **one queue item** the entry neither read nor listed as unread. The intensive residue is falsified by naming **one artefact inside a stated class boundary** that the reading did not open. A statement that survives neither test is a caveat |
| What a release that read **nothing** owes | **The same entry**, with an empty head and **the whole register as its extensive residue**, saying why it is empty — [#47](https://github.com/winniel123/verge-asm/issues/47)'s rule that a group rendering when populated must render when empty |
| Is a quantum set | **No.** ADR-0057's refusal is unchanged: no number of items per release, and the entry's obligation is discharged by an empty head as fully as by a full one |
| May the entry quote a **length** | **No** — not the queue's, not the head's, not the residue's. §39.2's bar is on the count as an **indicator**, and an entry that ranked itself by how many items it got through would restore exactly the meaning the bar removes |
| **Who writes it** | **The release** — the same act that revises a curated table (ADR-0057). A machine may **prepare** the entry from G11's marks; only the release may **sign** it, for the same reason a machine may raise and only a release may rule |
| **Does its absence look like its discharge** | **No, and that is the form's whole justification.** Entries are **tag-ordered and append-only**, so a release with no entry is a visible hole in the ledger. A rewritten standing section has no such property: a release that silently skips its residue statement leaves the previous one standing, reading true. ADR-0057 refused the standing curator because *a duty whose discharge and whose absence look identical* is unfalsifiable; the same test decides the **form** of the disclosure, not only the **office** |
| Where the register lives today | **[`sensitive-ports.md`](../research/sensitive-ports.md) §39.4, and it is provisional** — §39.9 says so, and the per-cell independence walk over all 38 pairs plus the frequency half is [#134](https://github.com/winniel123/verge-asm/issues/134). **The first live register is #134's output**, and it transcribes into the watch document then. Nothing is duplicated before that: a second copy of a provisional register while [#135](https://github.com/winniel123/verge-asm/issues/135) is moving the filter is G4's own failure shape |
| Does the ruling depend on the queue's **membership** | **No, and this has now been tested rather than asserted.** The form is stated over *the register as of the release's tag*, whatever it contains. [#135](https://github.com/winniel123/verge-asm/issues/135) has ruled since this ticket opened — the sole-ground filter now requires a second ground to have carried **the cell's proposition** alone, and runs **per cell, never per row** — and it moved the register's membership without touching a word of the entry form. #134 will move it again. **The tightening is one-way**, so the register can only grow, and every argument here is stated over **members** rather than over a length |
| Is `CONTEXT.md` amended | **No, and that is a ruling rather than an omission** — ADR-0057's reason is unchanged and is if anything stronger here: the residue is a fact about a **spec-time reading act on documents**, and the product holds nothing about it |
| Does [ADR-0008](./0008-derivation-versions-move-on-content.md) fire | **No.** No row, class, tier or coverage figure moves; every rule is byte-identical, and a governance instrument is not reference data |

### The entry, as a form

Each entry is dated, carries the release's tag, and has five parts. **It quotes no length anywhere.**

| Part | What it carries | Falsified by |
| --- | --- | --- |
| **1 — the order** | The queue register the release read against, cited at its state, never re-enumerated | Naming a register state the entry could not have read |
| **2 — the head** | Every item **read**, in the order read, each by its `(cell, artefact, revision act)` triple, with what the reading found: *unchanged* · *a question raised, and the ticket it became* · *the cell moved, and the release act that moved it* | Naming an item claimed read whose artefact was not fetched |
| **3 — the intensive bound** | Per item in part 2: the artefacts opened, and the **class boundary** of the opening | Naming one artefact **inside** a stated class that was not opened |
| **4 — the extensive residue** | Every item **not read**, named, each carrying its rung and the G11 mark that would have promoted it | Naming one queue item that appears in neither part 2 nor part 4 |
| **5 — the gate's record** | The gate run this release's edits were completed against, cited | — |

**Part 5 is not decoration and it is not this ADR's to design.** A release's account of its curated
tables has exactly two halves — the **gate**'s result, which is closed, complete and terminating, and
the **residue**, which is open, sampled and bounded — and **neither is readable without the other**.
Publishing the gate result alone invites the reader to take a green gate as the whole assurance, which
is completeness arriving labelled as coverage: §7's failure in the governance register rather than the
interface one. The gate ledger's own shape belongs to
[#133](https://github.com/winniel123/verge-asm/issues/133), which is running G1–G11 to completion for
the first time. This ADR reserves its place in the same document and specifies nothing about it.

## Rationale

### 1. The three candidate sites are not three answers to one question — they answer three different ones

The ticket's three options look like alternatives and are not. *A standing section* is an answer about
**tense**. *A per-release statement* is an answer about **cadence**. *Something the gate emits* is an
answer about **authorship**. Separating them is what makes the ruling decidable, because each has its
own test and the three tests do not compete.

**Tense.** A disclosure is written in the tense of the thing it describes. The queue's *membership* is
a live absolute and belongs in a standing register. The queue's *spend* is an act at a moment and
belongs in a dated record. #125 wrote both into one section — §39.4 is titled *"the queue as of
evidence already held"* and is a dated record doing a register's job — and §39.9 flagged the result as
provisional. Splitting them is not a criticism of §39.4. It is the thing §39.4's own hedge is asking
for.

**Cadence.** ADR-0057 anchored the gate to the **edit** and the residue to the **release**, and those
are different clocks. The map has already measured what happens when a governance obligation is
anchored to the wrong clock: a standing duty on a person decayed three times, each time with no
artefact to show for it.

**Authorship.** *A machine may raise; only a release may rule.* An entry describing what a machine
checked may be machine-written. An entry describing what a **person read** may not.

### 2. The pathology this form is built against is the one the corpus has measured most

Three failures, all measured, all the same shape:

- **Five useless counts.** `3 → 2 → 3 → 2 → 3` with the membership changing at every step (§39.2). A
  count over the wrong unit moved without carrying information.
- **Nineteen regenerating identity sites.** ADR-0038 withdrew *watch list = weak tier* and named no
  successor, so nineteen later sentences wrote it back (§39.7) — one of them inside ADR-0032 §8,
  **below** the withdrawal box and demonstrably edited after it.
- **Two superseded sentences standing in the present tense in the document that specifies the check
  that finds them** — ADR-0032 §8's open questions 3 and 6, found by G4 on the way past (§39.8).

**A standing residue section would be a fourth instance and the most predictable one yet**, because it
is the only one of the three whose staleness is *guaranteed by the calendar rather than risked by
inattention*: the statement becomes false the moment the next release ships, whether or not anyone
edits it. An append-only ledger cannot fail that way. Each entry is true forever about the release it
names, which is what a dated record is for, and the map's own Notes already say so in general — *every
figure inside a Decisions entry is a dated record, not a current value*.

### 3. Enumerable and non-enumerable residues are different objects, and ADR-0040 only ever met one

ADR-0040's boundary is a **description** because it has to be: *63 LAMPS RFCs and 17 active drafts,
~180 IAB statements, the 247 RFCs carrying an "Also BCP" designation filtered to the security area*.
Nobody can list the documents that were **not** searched, because that set is the rest of the world.
So the disclosure names the smallest **extension** that could still move the answer, and falsification
runs through the boundary: name one document outside it.

The queue is not that shape. It is our own list, its members are `(cell, artefact, revision act)`
triples over cells we hold, and it is finite. Applying ADR-0040's boundary form to it would produce
*"we read the top of the queue and stopped"* — a sentence that is a length with the number filed off,
and one nobody can falsify without first guessing what we thought the queue contained.

> **Where the population is ours and enumerable, a residue is named rather than described.** Naming is
> strictly the stronger disclosure and it is available exactly when the population is ours, so the
> weaker form is never the honest one here. This is ADR-0040 sharpened, not contradicted: its rule was
> written for the case where naming is impossible, and it is silent about the case where it is not.

**And the two forms compose rather than compete**, which is why the entry has both. The list is ours,
so part 4 names. Reading **one item** means opening somebody else's corpus, which does not terminate,
so part 3 describes — with a class boundary, which is ADR-0040's own device, and with ADR-0040's
falsification test unchanged. **[measured]** ADR-0040's failure mode is exactly a class boundary drawn
too narrow: #68 searched the **specification** class, concluded the IETF sets no key-size floor, and
missed the deployment BCP that does. An intensive bound that names its class is the disclosure that
would have caught that.

### 4. The gate may not emit it, and the reason generalises

The temptation is strong and it should be recorded as strong. The gate is mechanical, it already runs
per edit, it has an artefact by construction, and G11 already computes `stale-against-tag` over every
footing cell — which is most of the raw material an entry needs. Emitting the residue from the gate
would make the disclosure free and enforced.

It fails on what the gate **is**. ADR-0057's cut is between what terminates and what does not, and
every property that makes the gate trustworthy comes from termination: it runs to completion, its
output is a fact rather than a sample, and it needs no judgement. The residue is the **name of the
remainder** — the part that could not be checked because it does not terminate. An instrument defined
by termination cannot certify a statement about non-termination without asserting something it did not
establish.

What it would emit if forced is the giveaway. A machine holding the register and the tickets can
compute *which items have a ticket* and *how many do not*, and it would emit a length — the one
quantity §39.2 bars. **The count returns through whichever door is left machine-operable**, which is
the same shape as §7's severity returning through the one door the model left open, in a different
register.

> **A terminating instrument may supply inputs to a disclosure about what does not terminate, and may
> never sign it.**

### 5. Siting is decided by the subject, not by the tidiest neighbour

`sensitive-ports.md` is the tidiest neighbour: the working is there, the register is there, and §42
of it is where this reasoning lands. It still loses, because the disclosure's **subject** is not that
table.

**[measured]** §39.4's register carries two cells that are **not port cells at all** —
`verge-core`'s frequency half at `nmap-services`, and `certificate-expiring`'s fraction — and they are
**interleaved by rung** among the port cells rather than appended after them: the frequency half sits
at **rung 1**, above every rung-2 and rung-3 port item on the register. A residue statement in
`sensitive-ports.md` would either have to describe an order over cells that document does not own, or
report a sub-order — which is the per-table sub-queue ADR-0057 refused when it ruled that all three
cause-piles land on **one** queue because *the revision act, not the cause, is what fixes priority*.

**This argument is stated over members and never over a length, and that is deliberate rather than
fastidious.** [#135](https://github.com/winniel123/verge-asm/issues/135) has since tightened the
sole-ground filter — a second ground counts only where it would have carried **the cell's
proposition** standing alone, and the filter runs **per cell, never per row** — which moved the
register's membership and will move it again at #134. **The tightening is one-way**: it narrows what
counts as a ground, so cells can only be **added**. An argument resting on *which* cells are on the
register therefore survives every growth this map has in flight, and an argument resting on *how many*
would already be stale. That is this ADR's own rule applied to this ADR.

The mirror case is `weak-key-and-signature.md`, which contributes **zero** items — **[measured]** §9.3,
*"no shipped default is involved, so silent de-attestation cannot reach them"*. Under a per-table rule
it would carry a standing residue statement about an empty contribution, restated release after
release. That is a **permanent caveat**, which ADR-0032 §7 forbids by name and for exactly this reason:
it names no act, resolves on nothing, and decays into decoration. Its zero is already correctly
recorded **once**, as a dated finding, in the box #125 wrote there.

### 6. The operator-facing release note loses for §7's own reason

A product changelog is the obvious home for a per-release statement and it is the wrong one. **Its
reader is the operator**, and ADR-0032 §7's fourth ground is that this disclosure *"has a real
consumer, and that consumer is not the operator"*. A residue statement in an operator-facing artefact
is the screen ruling defeated by moving the same content to a different operator surface: it would
still be a confidence signal the operator cannot act on, still constant across subjects, and still
severity wearing honesty's clothes. **The ruling is about who reads it, not about whether it renders**,
and that is why the watch document is a spec document for the curator rather than a release note for
the operator.

## Consequences

- **[`docs/spec/curated-table-watch.md`](../spec/curated-table-watch.md) is added**, specifying the
  release's account of its curated tables. It carries the entry form, the two ledgers and the register's
  siting. **Both ledgers are empty and each says why**: v1 has not shipped, so no release has spent a
  reading budget, and the gate's first whole run is #133's.
- **The queue register is not moved and not copied.** §39.4 remains the queue as of #125's pass and is
  provisional by its own §39.9. The first **live** register is #134's per-cell walk, and it transcribes
  into the watch document at that point. **[#135](https://github.com/winniel123/verge-asm/issues/135)
  has already changed what the register contains and this ruling did not move** — which is the
  membership-independence claim tested rather than asserted. The entry form never quotes a length and
  never enumerates the register inside itself, so a growing register costs it nothing.
- **This ADR quotes no queue length anywhere, by construction.** Where it needs the register it names
  **members**. A governance ADR that pinned its reasoning to today's count would be the failure §39.2
  measured five times, committed by the document forbidding it.
- **ADR-0057 gains a rider** recording that its *lives in each instrument's own document* clause is the
  siting of the **watch**, not of the **residue disclosure**, which this ADR sites. Written at
  ADR-0057's own site per ADR-0058, because a clause that names no successor is re-derived by the next
  session that needs one — #125's own lesson, applied to #125.
- **ADR-0032 §7 is marked and not amended.** A box distinguishes the weak-tier disclosure's siting from
  the residue's. **The screen ruling, its four grounds and its unmet condition are restated unchanged**,
  and the box says so in terms.
- **ADR-0040 gains a limb rather than a correction**: where the population is the project's own and
  enumerable, the residue is **named** member by member. Its existing form governs every case it was
  written for, and it is **confirmed by use** at the intensive bound.
- **ADR-0032 §7's disclosure obligation acquires its fourth application** and the first over a
  population we author. Limbs 1 and 2 — *name the retrieval that resolves it*, *say what a performed
  retrieval established* — are discharged per item by parts 2 and 4 of the entry.
- **No row, class, tier or coverage figure moves.** `sensitive-ports.md` stays at **38 pairs**, classes
  `12 / 7 / 19`, tiers `13 / 11 / 3`, coverage **27 of 38**, §4.6 at **24**. `verge-core` stays at
  **136**. ADR-0008 is not triggered.
- **Nothing reaches the interface**, and this ADR adds a reason to §7's four rather than testing them:
  the residue names artefacts, commits and reading acts, which the operator can act on even less than a
  tier.
- **`CONTEXT.md` is not amended**, for ADR-0057's reason.
- **One residue of this ruling's own is named** — the release's *reading budget* has no size and this
  ADR deliberately does not give it one, ADR-0057 having refused the quantum. So an entry with a
  one-item head and an entry with a seven-item head are equally compliant, and the form's only defence
  against a permanently empty head is that the empty head is **visible and dated** every time. That is a
  weaker guarantee than a quantum and a stronger one than the standing duty it replaces, and it is
  stated rather than smoothed.

## Alternatives rejected

| Option | Why it lost |
| --- | --- |
| **A standing section of `sensitive-ports.md`, rewritten each release** | **The nearest miss and the option that lost.** It is where the working already is and it needs no new document. It fails on **tense**: a residue describes a spend, so restated in the present tense one release later it is *false*, and it becomes false by the calendar rather than by inattention. It fails again on **subject** — **[measured]** the register carries non-port cells, and one of them sorts above the port cells by rung, so that document cannot state the order the bound is over. And it fails on **absence**: a rewritten section leaves the previous statement standing when a release skips its duty, which is the exact property ADR-0057 refused in the standing curator |
| **Something the gate emits** | Three independent failures. Wrong **clock** — the gate fires on the edit, the residue is owed by the release, and a release that edits nothing owes the most informative residue of all. Wrong **faculty** — a machine attests what it checked, never what a person read. Wrong **definition**, and this one is fatal: the gate *is* the instrument of what terminates, and the residue is the name of what does not; forced to emit one, a machine emits a **length**, which §39.2 bars |
| **A twelfth gate check — G12, "the residue statement exists"** | Attractive because it makes the obligation self-enforcing, and it is on the wrong clock in the other direction: the gate blocks the **edit**, so G12 would demand a release-scoped artefact at edit time and would be un-green for every edit made between releases. Enforcement comes instead from the ledger's shape — a tag-ordered append-only list makes an omission visible without an instrument having to look for it |
| **One residue statement per curated table, in that table's own document** | The reading of ADR-0032 §7 a session would reach for first, and it is a mis-read of §7's subject. The queue is **one order across four tables**; a per-table statement either re-creates the sub-queues ADR-0057 refused or describes an order its document does not own. And `weak-key-and-signature.md` would owe a standing statement of a **zero contribution**, which is a permanent caveat — the thing §7 forbids by name |
| **The product's release note / changelog** | Loses on §7's own fourth ground: its reader is the **operator**, and this disclosure's consumer is the **curator**. Moving curator-facing severity to a different operator-facing surface defeats §7 rather than honouring it |
| **A new ADR per release** | An ADR records a decision; a residue records a spend. It would also make the ADR sequence a calendar, and ADR numbers are a scarce coordinated resource across concurrent sessions — the map reserves them per agent for exactly that reason |
| **Describe the extensive residue by a boundary, as ADR-0040 does** | It is the incumbent form and it is available, which is why it is recorded rather than assumed away. It loses because the population is **ours and enumerable**: a described boundary over an enumerable set is a **count with the number filed off**, unfalsifiable without first guessing what we thought the list held, and §39.2 bars the count |
| **State the residue as a fraction — *read k of the queue*** | ADR-0034's manufactured figure and §39.2's barred indicator in one sentence, and it restores meaning to the queue's **length**, which is the failure #125 repaired |
| **Drop the intensive bound and disclose only which items were read** | The cheapest form and it is not a disclosure. *We read item 3* is unfalsifiable, because reading somebody else's corpus does not terminate and nothing says how far it went. **[measured]** ADR-0040's own founding failure was a class boundary drawn too narrow — #68 read the specification class and missed the deployment BCP — which is precisely the defect an intensive bound exposes |
| **Set a quantum after all, so the ledger cannot fill with empty heads** | ADR-0057 refused it and this ADR does not reopen it: a quantum manufactures a figure nobody can attest and restores meaning to the queue's length. The residue this leaves is named in the Consequences rather than cured |
