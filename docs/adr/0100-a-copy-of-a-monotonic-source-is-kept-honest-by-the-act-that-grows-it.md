# ADR-0100: A copy of a monotonic source is kept honest by the act that grows it, not by a clock

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#179 Does curated-table-watch.md's §1.1 need a re-sync step when the source register grows?](https://github.com/winniel123/verge-asm/issues/179)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[#155](https://github.com/winniel123/verge-asm/issues/155) transcribed the live queue register
(`sensitive-ports.md` §43.3, folded with [#151](https://github.com/winniel123/verge-asm/issues/151)'s
§48.3 ruling) into [`curated-table-watch.md`](../spec/curated-table-watch.md) §1.1, as
`(cell, artefact, revision act)` triples grouped by [ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)'s
five rungs. The box §1.1 supersedes had refused to copy, twice, on the grounds that *"a second copy of a
provisional list is exactly the shape gate check **G4** exists to catch"* — G4 being
[ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)'s check that *"no sentence the
edit supersedes stands unmarked anywhere the edit's terms occur."* #155 answered that risk for the
**provisional-register** case: the register `sensitive-ports.md` §39.4 cited was still moving under
[#134](https://github.com/winniel123/verge-asm/issues/134)'s pending per-cell walk, so a copy made then
would have forked on arrival. By the time #155 ran, #134 had discharged that provisionality, so §155
transcribed a since-stabilized register and made the copy an explicit, dated, attributed snapshot rather
than a second live authority.

**What #155 did not settle is what happens the next time the register grows.**
[`sensitive-ports.md`](../research/sensitive-ports.md) §39.2 fixes the one property that decides how
expensive that upkeep is: **the register only ever grows.**
[ADR-0077](./0077-a-second-ground-counts-only-where-it-would-have-carried-the-cells-proposition-alone.md)'s
filter (worked by [#135](https://github.com/winniel123/verge-asm/issues/135)) and every walk since narrow
what counts as a ground rather than widen it, so cells are **added** and never removed or reordered out.
A copy of a monotonic, append-only source has exactly one way to go stale: a growth lands in the source
and nothing lands in the copy. That is a different failure shape from the one #155 weighed — not a fork
on a still-moving filter, but an **omission** on a since-stabilized one — and #179 is this ADR's ticket
for it.

**Neither of the register's governing ADRs rules on a downstream copy's own upkeep.**
[ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) rules on the queue's own design
— what it keys on, its five rungs, its filter, its unit. [ADR-0078](./0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md)
rules on where the **residue disclosure** is sited, a different object from the register itself. Neither
addresses what keeps a **second, appended copy** of the register in sync as the register it copies grows.
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) comes closest —
it rules where a **withdrawal** is marked when a mechanism is superseded — but its Scope row is
explicit: *"Any mechanism … not figures … and not claims about the world."* §1.1's rows are not a
withdrawn mechanism when the register grows. Every existing row stays true, and the copy is merely
**incomplete** relative to the source. That is a sibling shape to ADR-0058's, not an instance of it, and
this ADR states the sibling rule, borrowing ADR-0058's own reasoning where it transfers directly.

The full working is [`curated-table-watch.md`](../spec/curated-table-watch.md) §5.

## Decision

| Concern | Decision |
| --- | --- |
| **What keeps §1.1 honest as the register grows** | **A step in each register-growth ticket**, not a clock and not a gate check. Every ticket that adds a cell to `sensitive-ports.md`'s register — in [#125](https://github.com/winniel123/verge-asm/issues/125)'s, [#135](https://github.com/winniel123/verge-asm/issues/135)'s, [#151](https://github.com/winniel123/verge-asm/issues/151)'s shape — appends the new member(s) to §1.1's matching rung table in the **same commit**, citing the ground exactly as `sensitive-ports.md` states it |
| **Is this a curation trigger** | **No.** Per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s own classification test — a trigger watches the **world** moving; this fires when **we** move, by editing `sensitive-ports.md` ourselves. No cadence is created and no obligation to re-read §1.1 exists between growth tickets |
| **Is this a gate check** | **No, and none of the existing thirteen reaches it.** G4 catches a superseded sentence standing unmarked; a growth never supersedes an existing §1.1 row, it only leaves the row set incomplete. Reaching this shape mechanically would mean widening [ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)'s own gate table, which this ADR declines to do — the register's one-way growth (§39.2) already makes the manual step as cheap as an automated one, since there is never a removal or reorder to reconcile, only an append |
| **Does this amend ADR-0057 or ADR-0078** | **No.** Neither rules on a downstream copy's upkeep; this ADR states a narrower, new rule that sits beside them without reopening the queue's design or the residue's siting |
| **Does this revert #155** | **No.** The failure #155 weighed (a copy forking on a still-provisional filter) and the failure this ADR fixes (a copy omitting a since-stabilized source's later growth) are different; #155's answer to the first stands, and this ADR answers the second without undoing it |
| **Where the obligation is stated** | [`curated-table-watch.md`](../spec/curated-table-watch.md) §5 (new), with a one-line forward pointer left in §1.1's own supersession chain per ADR-0058's *"a clause that names no successor is re-derived by the next session that needs one"* |
| **Standing-process consequence** | **A register-growth ticket is not complete until §1.1 carries its cells.** This is a rule about how growth tickets are resolved, not only about this one document, and belongs in the map's own Notes section as a standing convention |

## Rationale

### 1. The cost lands on the party who can pay it, same as ADR-0058

[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) ruled that a
withdrawal is marked by *"the pass that supersedes … it holds both states in hand — it has just read the
old mechanism in order to replace it. Every later reader holds only one. Deferring the edit does not
avoid it; it relocates it to a session that must first discover the discrepancy."* A register-growth
ticket is in the identical position for the growth shape: it has just read the new cell's ground, fixed
its rung, and placed it in tie-break order, to add it to `sensitive-ports.md`. Appending the same triple
to §1.1 is marginal work for that session and a rediscovery cost for any later one — the exact asymmetry
ADR-0058 priced for supersession, here for addition.

### 2. A periodic or gated check is the instrument the map already has a cost history for, and it loses on both counts it lost on before

[ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) refused a standing curator with
a duty to watch, because *"a duty whose discharge and whose absence look identical is unfalsifiable"* —
measured three times over, with no artefact to show for any of them. ADR-0058 refused making mechanism
withdrawal a **curation trigger** on the same cost ground: *"a trigger implies a watch, and the watch
here would be re-read every superseded document forever … against an expected yield that is zero except
when a pass has just superseded something."* A periodic staleness check over §1.1 has the same expected
value curve — zero between growth tickets, a spike exactly when one lands — which is precisely the
signal ADR-0058 reads as *this obligation belongs at the edit, not on a clock*. A gated check fares no
better on a different axis: it would require widening [ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)'s
own thirteen-check table with a fourteenth check for a shape none of the existing ones reaches (G4 is
supersession, not omission), which is a heavier intervention than a monotonic, append-only source
warrants.

### 3. Reverting the copy trades a cheap step for the cost #155 already measured

[#155](https://github.com/winniel123/verge-asm/issues/155)'s box records that citing without copying
left every reader who needed the register's cell kinds, rungs, and tie-break order to resolve the
pointer against `sensitive-ports.md` at its own, much larger scale. Reverting to that state to avoid a
bounded, append-only maintenance step is not a trade worth making: the step this ADR imposes costs less
than the pointer-resolution cost it would restore, for every reader after the first.

### 4. The register's monotonicity is what makes Option 1 cheap enough to rule for

Every argument in this ADR depends on §39.2's one-way growth: `sensitive-ports.md`'s register only
adds members, never removes or reorders them out. That is what makes the re-sync step an **append**
rather than a **reconciliation** — no growth ticket ever needs to delete or resequence a §1.1 row, only
add to it. Were the source able to shrink or reorder, the calculus here would not transfer, and a
gated or periodic check would be worth its cost. It is not. This ADR is stated over the register's
current, measured shape and would need re-arguing if that shape ever changed.

## Consequences

- **[`curated-table-watch.md`](../spec/curated-table-watch.md) gains §5**, stating the re-sync
  obligation as a standing procedure, and §1.1's own supersession chain gains a one-line forward pointer
  to it.
- **Every future register-growth ticket (in [#125](https://github.com/winniel123/verge-asm/issues/125)'s,
  [#135](https://github.com/winniel123/verge-asm/issues/135)'s, [#151](https://github.com/winniel123/verge-asm/issues/151)'s
  shape) is not complete until it has appended its cells to §1.1** in the same commit that adds them to
  `sensitive-ports.md`. This is a standing-process rule and belongs in the map's own Notes section as a
  convention, not only in this document.
- **No gate check is added.** [ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)'s
  thirteen-check table is unchanged and this ADR does not reopen it.
- **[ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) and
  [ADR-0078](./0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md) are unchanged.** Neither the
  queue's design nor the residue's siting moves.
- **No row, class, tier, rung or coverage figure moves.** This ADR adds a procedure and quotes no length
  anywhere, per [`sensitive-ports.md`](../research/sensitive-ports.md) §39.2.
- **`CONTEXT.md` is not amended.** The curator is not a subject in the model, unchanged from
  [ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) and
  [ADR-0078](./0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md)'s own reasoning.

## Alternatives rejected

| Option | Why it lost |
| --- | --- |
| **A periodic or gated staleness check** | Priced twice already in this corpus and it loses both times for the same reason: it is a **curation trigger** for a failure that fires when **we** move, not when the world does — [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s own classification test. Expected yield is zero between growth tickets and a spike exactly when one lands, which is the shape ADR-0058 already declined to put on a clock. A gated form would also require widening ADR-0057's own thirteen-check table for a shape (omission, not supersession) none of the existing checks reaches |
| **Revert [#155](https://github.com/winniel123/verge-asm/issues/155) to a cite-only pointer** | Answers the wrong risk. #155's G4 concern was a copy forking on a still-provisional filter, already resolved by [#134](https://github.com/winniel123/verge-asm/issues/134). This ADR's concern is a copy omitting a since-stabilized source's later growth — a cheaper problem with a cheaper fix, not a reason to undo the transcription and restore the pointer-resolution cost #155 already measured |
| **A fourteenth gate check** | Would reach only the shape this ADR names, at the cost of widening [ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)'s own table for a check whose population — cells added to the register since the copy was made — is already caught for free by the growth ticket that adds them, per this ADR's Option 1 |
| **A standing curator with a duty to re-sync §1.1** | The status quo this whole area of the map has already measured failing three times ([ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)'s own founding measurement) — a duty whose discharge and whose absence look identical |
