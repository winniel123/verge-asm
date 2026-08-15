# ADR-0077: A second ground counts only where it would have carried the cell's proposition standing alone — and the filter is run per cell, never per row

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#135 Does the queue's sole-ground filter require a ground strong enough to have carried the cell alone?](https://github.com/winniel123/verge-asm/issues/135)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)
([#125](https://github.com/winniel123/verge-asm/issues/125)) built the de-attestation queue and gave it
one filter: **sole-ground only**. Its words are *"a cell is an item exactly where **one** artefact
carries it and no second **independent** artefact carries the same cell"*. #125 flagged the filter as
the rule's soft edge in the same pass, at
[`sensitive-ports.md`](../research/sensitive-ports.md) §39.9:

> *"`10250/tcp` … is off the queue on the ground that `ports-and-protocols.md`'s `Used By: Self,
> Control plane` cell is a second, independent artefact. §16.9 calls that cell **the thinnest placement
> in the table**. A reader who holds that a cell too thin to carry a tier is also too thin to count as a
> **ground** puts `10250` back on the queue at rung 2. That reader is not obviously wrong, and the
> question is a real one: **the filter counts grounds, and nothing yet says a ground must be strong
> enough to have carried the cell alone.**"*

Two things are underdetermined, and only one of them is the one the flag names.

**What a ground has to be strong enough to do.** *Carries the same cell* admits at least three readings:
that the artefact touches the row at all; that it would have sustained what the cell asserts; that it
would have sustained the cell **at the cell's own tier**. The three give different queues, and the
difference is large — the third empties §39.4's largest *not an item* row (*"every graded row with an
owner sentence and a configuration artefact"*), which is most of the table.

**Which object the filter is applied to.** ADR-0057's own unit is a **`(cell, artefact, revision act)`
triple** and its own diagnosis of §8's five useless counts is that *"the list enumerates **rows**, and
what changes is a **cell's** supporting artefact"*. §39.4 then disposes of `10250/tcp` and `10255/tcp`
**as rows**, on the reasoning that the act *"demotes rather than de-attests"*. That is the row-level
reading of a rule whose declared unit is the cell, one section after the unit was fixed — and it is what
conceals the case the flag is groping for. **[measured]** §19.12 states of the same row: *"Claim 3 has
**no second support** on this row"*.

This ADR is about the **filter**. It moves no row, no class, no footing tier and no coverage figure, and
[ADR-0008](./0008-derivation-versions-move-on-content.md) is not triggered. The full working is
`sensitive-ports.md` **§41**.

## Decision

| Concern | Decision |
| --- | --- |
| **Is *exists* the bar?** | **No.** A second artefact removes a cell from the queue only where it would, **standing alone and with the first artefact struck out**, have carried **that cell's proposition**. Mere presence in the row's evidence is not a ground |
| **What is *the cell's proposition*** | **What the cell asserts about the world**, stripped of our grading of it: for a footing cell, *the owner places this `(port, transport)` pair inside a boundary it names*; for a claim cell, *the named claim in §2.1's closed set holds for this pair*; for a *why* cell, *the stated ground obtains*. It is read off the cell, never judged |
| **Does a tier count as part of the proposition?** | **No.** A footing **tier** grades **evidential distance** ([ADR-0059](./0059-a-footing-tier-grades-evidential-distance-never-the-owners-conviction.md)) — the count of premises **the reader supplies**. It is a disclosure about *us*, not an assertion about the owner. A second artefact that would have carried the proposition at a **lower** tier **is** a second ground; the act **demotes** and the cell survives, which is ADR-0057 §3 correctly stated |
| **The unit the filter is applied to** | **The cell, never the row.** ADR-0057's unit was already the cell; §39.4's disposal of `10250`/`10255` was made at the row and is superseded. **A row with two cells is tested twice**, and may be an item on one and not the other |
| **What does *not* count as a second ground** | An artefact carrying a **different cell of the same row** (membership where the cell asserts position; a claim where the cell asserts a footing); an artefact carrying **part** of the proposition; a **corroborator** (§2.3, unchanged — a corroborator was never a ground and so can never remove an item) |
| **What the test is not** | It is **not** a re-grading and **not** a retrieval. It is run over artefacts the note **already cites**, by striking out the carrying artefact and asking what the cell would then have said. If the answer needs bytes we do not hold, the cell is **undetermined** and is named, never assumed either way |
| **Direction of failure** | **Toward more watching.** Every reading this ADR refuses would put **fewer** cells on the queue. Where the test is undetermined the cell goes **on** |
| **Does anything come off the queue** | **No.** The bar can only narrow what counts as a second ground, so it can only add items. All eight of §39.4's items stand unchanged |

### The test, stated so it can be run

For a cell **C** carried by artefact **A**:

1. **Strike A out.** Ask what C would say on the remaining artefacts this note already cites for this row.
2. If some remaining artefact **B**, alone, still yields C's **proposition** — at any tier — then B is a
   second ground and **C is not a queue item**. Two acts are needed to falsify C.
3. If every remaining artefact yields a **different proposition** — a different cell, a fragment of the
   cell, or nothing — then **C is a queue item at A's rung**, whatever else the row holds.
4. If step 2 cannot be decided from bytes the project holds, **C is a queue item** and the undetermined
   step is named. A ground that must be retrieved before it can be counted is not yet a ground.

**A worked pair, both from measurements already in the note.**

| Cell | Carried by | Strike it out | Verdict |
| --- | --- | --- | --- |
| `10250/tcp`'s **footing** — *the owner places this pair inside a boundary it names* | `security-checklist.md`'s *"The Kubernetes API, kubelet API and etcd are not exposed publicly on Internet"* | §18.7: *"`10250` falls back to the **scoping tier** on the ports table"* — the proposition survives, the **tier** does not | **Not an item.** A demotion, exactly as ADR-0057 §3 says |
| `10250/tcp`'s **claim** — *Claim 3 holds for this pair* | `ports-and-protocols.md`'s `Used By: Self, Control plane` | **[measured]** §19.12: *"Claim 3 has **no second support** on this row"* | **An item**, rung 2. §39.4 never tested it, having tested the row |

**That is the whole answer to the flag.** §39.9's reader is right that `10250` belongs on the queue and
wrong about which cell puts it there. It is not that the `Used By` cell is too thin to be a ground — it
is a perfectly good ground for the proposition it carries. It is that the note leans on it **twice**, for
two different cells, and counting it once as a *second* support for the footing hid that it is the
**only** support for the claim.

## Rationale

### 1. *Carries the same cell* was always the tighter rule; the row-level application is what loosened it

ADR-0057 did not write *another artefact touching the row*. It wrote *carries the same cell* — and a
cell is a cell, not a row. Read as written and applied to the object it names, the rule already refuses
the `10250` disposal. This ADR is therefore a **reading with a bar attached** rather than a reversal,
and that matters for the direction of travel: nothing #125 decided is overturned, and its worked
example survives at the cell it was actually about.

### 2. A tier is our disclosure, so a demotion is not a de-attestation — and this is the limb worth defending

The strongest form of the ticket's candidate is *a second ground must carry the cell **at its own
tier***, and it is the option that lost. It is refused on a measurement and on a warrant.

**The measurement.** §39.4's largest *not an item* row is *"every graded row with an owner sentence
**and** a configuration artefact — two independent acts are needed. **This is most of the list**, which
is why the queue is short."* Every such row has the `10255` shape: strike the sentence and the row falls
to the weak tier on the default. Under the tier-strict reading every one of them becomes an item, the
queue becomes approximately the graded table, and the **filter removes nothing** — while ADR-0057
assigns support count exactly one job, *"it removes items rather than ordering them"*. A filter that
removes nothing is not a filter, and a reading order that is the whole table is not an order.

**The warrant.** [ADR-0046](./0046-a-negatives-corpus-is-its-owners-class-list-and-only-a-sole-ground-negative-is-exposed.md)
limb 1 — the arithmetic ADR-0057 read on the positive side — is about a claim being **falsified**, not
about it being **re-graded**. And ADR-0059 fixes a footing tier as *"the count of premises **the reader**
must supply"*: it is a statement about our evidential distance, authored by us. An owner's act cannot
falsify a sentence the owner did not write. What the act does is make our disclosure **over-stated**,
and over-statement of our own distance is what the **gate** is for — G2 walks every graded footing cell
against the tier criterion as it currently stands, and G11 marks a cell **stale-against-tag** whenever
the owner's release line has moved past the tag the cell was read at.

**The residue is named rather than smoothed.** G11 is vacuous for an artefact with no tag to diff — a
continuously-published page with no version pin, which is rung 2's definition. So a tier demotion on an
**untagged** rung-1 or rung-2 artefact is caught by neither instrument, and this ADR declines to fix it
by widening the queue. It is a **gate** shortfall and it is ticketed as one.

### 3. The four-pairs-one-sentence concentration is an argument for the cell, not the row

ADR-0057's own headline measurement is that *"four pairs are one item where one Microsoft sentence
carries `445`, `139`, `137` and `138`"* — the row is the wrong container in the **many-pairs-one-cell**
direction. §39.4's `10250` disposal is the same error in the **one-row-many-cells** direction, made two
subsections later. Fixing one and not the other leaves the unit half-applied, and a half-applied unit is
how the identity ADR-0057 spent a whole section sweeping regenerated in the first place
(rationale 5: *a sentence that names no successor is re-derived by the next session that needs one*).

### 4. Undetermined counts as on, and that is the cheap direction

§39.9 leaves `2379`/`2380` undetermined: §13.6 records `etcd.conf.yml.sample` as *"neither"* operative
nor example, so whether a restricting default is available as a fallback ground is open, and settling it
is a retrieval. Both pairs are **on** the queue today. The rule keeps them there: an artefact whose
status as a ground cannot be read off bytes we hold is not a ground yet. The cost of a wrongly-included
item is one reading; the cost of a wrongly-excluded one is a silent de-attestation, which is the failure
ADR-0032 §8 named and has already been paid once on `623/udp`.

### 5. Why the bar is not *the strongest artefact wins*

A tempting near-miss is *the second ground must be at least as strong as the first*. It is refused for
the same reason the tier reading is: **strength** is our grading, and grading is not what the queue
measures. The queue measures **how easily the ground moves** — the revision act — and a weaker artefact
at a higher rung is a **better** fallback than a stronger one at rung 1. Ranking grounds by strength
would re-import the tier as a key through the filter's back door, one ADR after it was refused as a key
through the front.

## Consequences

- **The queue moves from eight items to nine**, over **eleven** `(port, transport)` pairs and **two**
  non-port cells. The one item added is `10250/tcp`'s **claim** cell at `ports-and-protocols.md`'s
  `Used By: Self, Control plane`, rung 2. `sensitive-ports.md` §41.4 states it with its ground; §39.4 is
  marked at its clause.
- **Nothing leaves the queue.** The bar narrows what counts as a second ground and can only add. All
  eight of §39.4's items stand with their grounds unchanged.
- **`10250/tcp`'s and `10255/tcp`'s footing cells stay off**, now for a stated reason rather than a
  row-level one: §18.7 measures both fallbacks, to the scoping tier and to the weak tier respectively.
  ADR-0057 §3's *exposure to a demotion is not exposure to de-attestation* is **confirmed** and is
  re-sited at the cell.
- **No row, class, tier or coverage figure moves.** The list stays at **38 pairs**, classes
  `12 / 7 / 19`, tiers `13 / 11 / 3`, coverage **27 of 38**, §4.6 **24 entries**, `verge-core` **136**.
  ADR-0008 is **not triggered** — `sensitive-port-reached-from-internet` is byte-identical and a
  governance instrument is not reference data.
- **ADR-0057's filter row and its §3 worked example are marked at their own clauses**, per
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md), and so is its
  eight-item Consequence. The mark supplies a **replacement**, not a strike-through — ADR-0057's own
  rationale 5.
- **[#134](https://github.com/winniel123/verge-asm/issues/134)'s walk is unblocked and its unit is
  fixed**: it walks **cells**, not rows — each pair's claim cell, its footing cell where it has one, and
  any *why* cell carrying a proposition — and applies the four-step test to each. Rows whose claim and
  footing rest on **different** artefacts were counted as two-ground rows and are now two separately
  testable cells, so the walk should expect the queue to grow.
- **A gate shortfall is opened rather than closed**: a tier demotion on an artefact with no retrievable
  tag is caught by neither G2 nor G11. Ticketed; not repaired here, because repairing it in the queue is
  the option this ADR refuses.
- **`CONTEXT.md` is not amended**, on ADR-0057's own last Decision row: the curator is not a subject in
  the model and the product holds nothing about it.
- **ADR-0046, ADR-0059 and §2.3 are confirmed by use and none is amended.** §2.3's corroborator bar is
  strictly stronger than this rule and is untouched.

## Alternatives rejected

| Option | Why it lost |
| --- | --- |
| **The incumbent — *a second independent artefact exists*, applied at the row** | The option that lost, and it is ADR-0057 as §39.4 ran it. It tests an object the ADR had already refused as a unit two subsections earlier, and **[measured]** it conceals a live case: §19.12 records `10250`'s Claim 3 as having *"no second support on this row"* while §39.4 lists `10250/tcp` under *not an item*. A rule whose declared unit is the cell cannot be evaluated on the row |
| **The ticket's candidate at full strength — the second ground must carry the cell *at its own tier*** | The best argument on the other side and it is refused on a measurement: it empties §39.4's largest *not an item* row, which is *"most of the list"*, so the filter removes nothing and the queue becomes the graded table. And it mislocates the failure — a tier is **our** disclosure of evidential distance (ADR-0059), so an owner's act cannot falsify it; over-statement of our own distance is the gate's business (G2, G11), not the queue's |
| **The second ground must be at least as *strong* as the first** | Re-imports the footing tier as a key through the filter, one ADR after ADR-0057 refused it as a key outright. Strength is our grading; the queue measures how easily a ground **moves**, and a weaker artefact at rung 5 is a better fallback than a stronger one at rung 1 |
| **Leave *carries the same cell* undefined and let each pass read it** | This is what produced the defect. ADR-0057's own rationale 5 — *a withdrawal that supplies no replacement does not hold* — applies to an **undefined** clause as much as to a superseded one: a term with no fixed denotation is re-derived by whoever meets it, and #125 and #135 read it two different ways within one section |
| **Put tier demotions on the queue as a second kind of item** | It builds the fourth pile ADR-0038 refused and ADR-0057 declined to reopen, and it restores meaning to the queue's **length** — the failure being repaired. The demotion hazard is real and is recorded as a **gate** shortfall with its own ticket |
| **Treat an undetermined second ground as absent, and leave the cell off pending a retrieval** | Backwards on cost. A wrongly-included item costs one reading; a wrongly-excluded one costs a silent de-attestation, which has already happened once (`623/udp`, §36.7) on a row that was never watched |
| **Rule out of scope — the filter is #134's to settle while it walks** | Refused: #134 is **blocked** on this precisely because the answer changes what it counts, and a walk that decides its own criterion mid-pass produces a result nobody can re-derive. The criterion is ruled first and the walk is executed against it |
