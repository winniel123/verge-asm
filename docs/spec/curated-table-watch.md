# The release's account of its curated tables

- **Status:** Accepted — spec content for [#12](https://github.com/winniel123/verge-asm/issues/12)
- **Ruling:** [ADR-0078](../adr/0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md), on
  [ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)
- **Ticket:** [#136 Where does the curated-table queue's bounded-residue disclosure live?](https://github.com/winniel123/verge-asm/issues/136)

[ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) ruled that a curated
table is revised by **the release**, and that the watch over the four curated tables is **two
instruments**: a **gate** over what is *closed*, which terminates and so runs to completion over the
table as edited, and a **queue** over what is *open*, which cannot terminate and so can only be
**sampled**. It gave the release one obligation towards the queue — a **bounded residue** disclosure
— and sited nothing. [ADR-0078](../adr/0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md)
sites it here.

**This document is the release's account, and it has two halves that are read together.** The gate's
result is *closed, complete and terminating*. The residue is *open, sampled and bounded*. Publishing
either alone invites the reader to take one for the whole assurance, and a green gate read alone is
**completeness arriving labelled as coverage**.

**It is a curator's document and reaches no interface.**
[ADR-0032](../adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §7 keeps this
material off every v1 screen on four grounds, its condition is unmet
([ADR-0016](../adr/0016-an-annotation-moves-a-message-never-a-number.md)), and ADR-0078 does not
reopen it. Nor does this content belong in an operator-facing release note: §7's fourth ground is
that the consumer is **not the operator**, and moving it to a different operator surface would defeat
§7 rather than honour it.

**No length is quoted anywhere in this document** — not the register's, not a head's, not a residue's.
[`sensitive-ports.md`](../research/sensitive-ports.md) §39.2 bars the queue's count as an indicator,
its membership having changed five consecutive times without the count carrying the change.

---

## 1. The register — the queue, as it currently stands

**The register is a live absolute: the queue's current membership, in one place.** Its members are
`(cell, artefact, revision act)` triples over the cells of the four curated tables, ordered by the
five rungs of ADR-0057, tie-broken by how far the owner has moved past the tag the cell was read at,
in the owner's own release line.

> **The register is not yet transcribed here, and that is a state rather than an omission.**
> [`sensitive-ports.md`](../research/sensitive-ports.md) **§39.4** holds the queue, as built by
> [#125](https://github.com/winniel123/verge-asm/issues/125) and as tightened by
> [#135](https://github.com/winniel123/verge-asm/issues/135), and §39.9 marks it **provisional**: the
> per-cell independence test has been run only over cells already measured. The first **live**
> register is the output of [#134](https://github.com/winniel123/verge-asm/issues/134)'s per-cell
> walk over all 38 pairs plus the frequency half. **§39.4 is the register until then, and is cited
> rather than copied**: a second copy of a provisional list is exactly the shape gate check **G4**
> exists to catch, and a transcription made before #134 would be superseded on arrival.

**No figure for the register's size appears in this document, here or anywhere below.** #135 has
already moved it once and #134 is expected to move it again — and **the movement is one-way**, #135's
tightened filter narrowing what counts as a ground so that cells can only be **added**. Every
statement in this document is over the register's **members**, which is what §2.2 requires of an
entry and what §39.2 requires of anyone quoting the queue at all.

**The register spans four curated tables and is one order, not four.** **[measured]** §39.4 carries
two cells that are not port cells — `verge-core`'s frequency half at `nmap-services`, and
`certificate-expiring`'s fraction — and they are **interleaved by rung** among the port cells rather
than appended after them: the frequency half sits at **rung 1**, above every rung-2 and rung-3 port
item. That is why the residue disclosure below is one statement and not one per table: *how far down
it read* is a fact about the order, and no single table's document owns the order.

---

## 2. The residue ledger — one dated entry per release, appended

**Every release writes one entry. A release that read nothing writes one too**, with an empty head
and the whole register as its residue, saying why the head is empty
([#47](https://github.com/winniel123/verge-asm/issues/47): a group that renders when populated
renders when empty and says why). **Entries are appended and never rewritten**, and they are ordered
by release tag, so a release that skips its entry leaves a **visible hole** rather than a previous
entry standing and reading true.

### 2.1 The entry form — five parts

| Part | What it carries | Falsified by |
| --- | --- | --- |
| **1 — the order** | The register state this release read against, cited at that state. **Never re-enumerated inside the entry** | Naming a register state the entry could not have read |
| **2 — the head** | Every item **read**, in the order read, each by its `(cell, artefact, revision act)` triple, with what the reading found: **unchanged** · **a question raised**, and the ticket it became · **the cell moved**, and the release act that moved it | Naming an item claimed read whose artefact was not fetched |
| **3 — the intensive bound** | For each item in part 2: the artefacts **opened**, and the **class boundary** of that opening — *this owner's issued documentation set at tag X and its release notes, and no other class* | Naming one artefact **inside** a stated class boundary that the reading did not open |
| **4 — the extensive residue** | Every item **not read**, **named** by its triple, each carrying its rung and the **G11** mark that would have promoted it | Naming one register item appearing in neither part 2 nor part 4 |
| **5 — the gate's record** | The gate run this release's edits were completed against, cited to §3 | — |

### 2.2 The two bounds, and why they are stated differently

**The extensive residue is named, member by member.** The register is **our own list** and is
enumerable, so a described boundary would be a count with the number filed off — unfalsifiable
without first guessing what we thought the register held. **Where the population is the project's own
and enumerable, a residue is named rather than described**
([ADR-0078](../adr/0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md), sharpening
[ADR-0040](../adr/0040-a-specifications-silence-is-not-the-owners-silence.md)).

**The intensive bound is described, per item, with a class boundary.** Reading **one** item means
opening **somebody else's** corpus, which does not terminate. *We read item 3* is unfalsifiable;
*we opened this owner's issued documentation at tag `v1.37.0` and its release notes, and no other
class* is falsified by naming one artefact inside that class. This is ADR-0040's form unchanged,
applied at the item rather than at the list — and **[measured]** ADR-0040's founding failure was a
class boundary drawn too narrow, [#68](https://github.com/winniel123/verge-asm/issues/68) having read
the **specification** class and missed the **deployment BCP** that carried the number.

### 2.3 What the entry may not contain

| Barred | Why |
| --- | --- |
| **Any length** — of the register, of the head, of the residue | §39.2. A count over the queue moved five times without carrying the change, and ranking a release by how far it got restores exactly the meaning that bar removes |
| **A fraction** — *read k of the register* | A manufactured figure nobody can attest ([ADR-0034](../adr/0034-derive-the-claim-before-looking-for-the-owner.md)), and a length in another coat |
| **A quantum, or a target** | ADR-0057 refused it. An entry with a one-item head is as compliant as one with a seven-item head; the discipline is that the head is **visible and dated**, not that it is large |
| **A verdict on a row** | A machine may **prepare** an entry from G11's marks; only the release may **sign** it, and only a release may move a row. The watch's output is a **question**, and the answer is a retrieval, which is a ticket |
| **A permanent caveat** | ADR-0032 §7. Every part of the entry is falsifiable by the test in its own row above, which is what *bounded* means and *permanent* does not |

### 2.4 The ledger

**Empty, and the reason is stated rather than left blank.** v1 has not shipped, so no release has
spent a reading budget and no entry is owed yet. The first entry is owed by the first release that
ships a curated table.

| Release | Entry |
| --- | --- |
| — | **None yet.** No release has occurred |

---

## 3. The gate ledger — reserved

**The gate is eleven checks, G1–G11, run over the table as edited; an edit is complete only when the
gate is green over the post-edit state, and a red gate blocks the *edit*, never the release**
([ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md);
[`sensitive-ports.md`](../research/sensitive-ports.md) §39.6).

**This section's shape is [#133](https://github.com/winniel123/verge-asm/issues/133)'s and is
deliberately unspecified here.** The gate has never been run whole; #133 is running G1–G11 to
completion over the composed table for the first time and establishes the baseline every later edit
is measured against. ADR-0078 reserves the gate's record a place in this document — because the two
halves of a release's account are read together — and specifies nothing about its form.

**One thing the gate owes the residue ledger, whatever shape #133 gives it.** **G11** marks every
footing cell whose owner has moved past the tag the cell was read at. Those marks are the reason an
item is read or passed over, so **entry part 4 cites the G11 mark for every item it leaves unread**.
The gate **supplies inputs to the residue disclosure and never signs it**: an instrument defined by
termination cannot certify a statement about what does not terminate, and what it would emit if
forced is a **length**.

---

## 4. What this document is not

- **It is not the queue's reasoning.** That is
  [ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) and
  [`sensitive-ports.md`](../research/sensitive-ports.md) §39.
- **It is not a curated table**, and nothing in it is asserted about the world. It records what the
  project **did**, which is why it needs no evidence standard under
  [ADR-0032](../adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md).
- **It is not read by the product**, and the curator is not a subject in the model —
  [`CONTEXT.md`](../../CONTEXT.md) is not amended, per ADR-0057 and ADR-0078.
- **It is not a screen, and it is not a changelog.** ADR-0032 §7, on both counts.
