---
number: 1909
title: "A dial submitted at its current value writes no Act row"
slug: a-dial-submitted-at-its-current-value-writes-no-act-row
date: 2026-09-13
status: accepted
source: fix
ticket: 1909
map: 1826
proof: {test: "cmd/web/act_unmoved_dial_test.go::TestASubmitThatLeavesADialWhereItStoodRecordsNothing"}
relations:
  - {kind: rests-on, adr: 1891}
---

# ADR-1909: A dial submitted at its current value writes no Act row

## Decision

**A dial submitted at its current value writes no `Act`.** The handler reads the stored value before
the mutation, compares, and records only on a difference. The rule reaches the eight classes that
embed `act.DialMove` and no others.

An `Act` asserts **a change of state**, not an intent to set. A toggle re-submitted at the position
it already held leaves nothing behind, so a row reading `Dial moved · API access · on` asserts a
movement that did not happen. The corpus is append-only and never deleted, so no later write corrects
it.

Rejected: unconditional recording, on [`docs/spec/audit-act.md`](../spec/audit-act.md) §7.6 ruling
7's reached-the-setter boundary. That boundary was drawn for a zero-jobs dispatch, which left a
`Dispatch` row behind.

Reversal unguards six handlers and leaves every false row already written standing.

**So five handlers now read a value their mutation does not need.**

## 1. Context

[ADR-1891](./1891-an-operator-act-is-recorded-as-a-fifth-operational-corpus-under-a-four-limb-predicate.md)
records an operator act as an `Act` under a four-limb predicate. Limb 1 is *changed the estate's
declaration*. It rules **what** is recorded.
[ADR-1902](./1902-one-act-row-per-subject-never-one-per-request.md) rules **how many** rows one
submit writes, once the handler directs more than one subject.

Neither rules the case where the submit directed a subject and **changed nothing about it**. The two
questions come apart at a dial. A dial's form re-posts its current value on every submit that touches
any other control on the same panel.

The corpus shipped with the question open, and it shipped inconsistent. Eight handlers write a
`DialMove` class. Two compared and six did not:

| Site | Class | Before |
| --- | --- | --- |
| `cmd/web/retentionpanel.go` | `observation.currency.set` | guarded |
| `cmd/web/retentionpanel.go` | `dispatch.cadence.set` | guarded |
| `cmd/web/seeds.go` | `zone.cadence.set` | unconditional |
| `cmd/web/seeds.go` | `dns.cadence.set` | unconditional |
| `cmd/web/settings.go` | `transcript.currency.set` | unconditional |
| `cmd/web/settings.go` | `address.cap.set` | unconditional |
| `cmd/web/settings.go` | `update.check.moved` | unconditional |
| `cmd/web/settings.go` | `api.access.moved` | unconditional |

[`docs/spec/audit-act.md`](../spec/audit-act.md) §2.2 ruled the two-dial route and nothing else, and
`retentionpanel.go` carried the rule as a comment rather than as a decision.

Ticket [#1832](https://github.com/winniel123/verge-asm/issues/1832) raised the gap on 2026-09-11.
Ticket [#1833](https://github.com/winniel123/verge-asm/issues/1833) then wired one more unconditional
site on that precedent. The question left map
[#1826](https://github.com/winniel123/verge-asm/issues/1826) as residual 3 of five.

## 2. Why ruling 7 does not carry

§7.6 ruling 7 puts the boundary at **whether `Trigger` was reached**, not how many jobs came out. A
zero-jobs dispatch writes an `Act`. Read as a general principle, that boundary says a handler records
whenever its setter ran, and six sites shipped on exactly that reading.

**The analogy fails on what each act left behind.** Ruling 7's own sentence gives the ground: *"`n ==
0` did call it, and a `Dispatch` exists, so the instance acted."* The `Dispatch` row is the residue.
A dial re-submitted at its stored value calls `UPDATE`, the `UPDATE` matches a row, and the row's
values are what they already were. There is no residue, because nothing about the estate is different
after the submit than before it.

So ruling 7 is not narrowed here. It is read at the scope it was written for — an act that produced a
record — and that scope does not contain the unmoved dial.

## 3. Why the corpus cannot absorb the false row

§7.6 bars recorder-before, and states why in terms this decision inherits. It produces **an `Act` with
no act**, and *"append-only generates no `DELETE`, so a phantom row can never be retracted."*

An unmoved dial produces the identical row by a different route. Take `Dial moved · transcript
currency · 14 days`, written after a submit that left the dial at 14 days. Its Action and Subject
together assert a movement. The §6 reader renders the Action label verbatim, so the false claim is
what an operator reads.

Two facts make this worse than a stray row rather than equal to one.

- **Toggles are the cheap case.** `update.check.moved` and `api.access.moved` render as a switch.
  Re-submitting a switch at its current position takes one click and is the ordinary result of a
  double submit or a browser back-and-resubmit.
- **The rows outnumber the true ones over time.** The corpus is unbounded and never reclaimed (§5.1),
  and no dial trims it. A panel that re-posts every dial on every save accumulates false rows at the
  rate of saving, not at the rate the dials move.

## 4. Why the boundary is `act.DialMove` and not "every idempotent submit"

The rule needs an edge a later session can apply without re-deciding it. **A class that embeds
`act.DialMove` is that edge.** It is eight classes, and it is checkable in code. The embedded struct
is exactly the payload whose Subject renders as `<dial> · <value>`, which is the cell that carries the
false assertion.

Two neighbours sit outside it, each on its own ground.

- **`cold.moved` embeds `ColdMove`, not `DialMove`.** It is already guarded, and it guards on a
  different mechanism. Its mutation returns `pgx.ErrNoRows` when the scope already stood where the
  submit put it. The store reports the no-op, so no read-compare is needed. Nothing here unguards it.
- **`integration.installed` is not a dial.** An install has no stored value to compare against a
  submitted one. Whether a re-install is an act is a different question with a different answer.

The wider rule — *any submit that changes nothing writes nothing* — was weighed and not taken. It
needs a per-class account of what "unchanged" means for a non-dial, and it would be decided here for
classes nobody has examined.

## 5. The price, named rather than hidden

**Five handlers now read a value their mutation does not need.** `updateRetention` already read its
settings row and pays nothing. The other five gain one `SELECT` on a panel submit, against
`instance_config` or the scan cadence — reads the same panel's renderer already makes.

**A failed read refuses the act.** `setZoneInterval`, `setDnsInterval`, `updateAddressCap`,
`updateCheckToggle` and `apiToggle` return `serverError` when the comparison read fails, before the
mutation runs. This is a real cost. A database fault on a read the mutation does not need now blocks a
dial the operator could otherwise set.

It is the honest branch. A handler that cannot see the stored value cannot tell a move from a repeat.
Recording anyway writes the row this ADR bars, and that row can never be retracted. Refusing leaves
the operator an error and a retry. The refusal is also the shape `updateCoverageRetention` already
had, so the corpus gains no second failure mode.

**The refusal does not branch on direction, and that costs something.** During a database fault the
handler refuses an admin turning API access **off**. The read that refuses them exists only to decide
whether to write an audit row. A rule that let the safe direction through would draw a boundary on the
submitted value. Every later dial class would then have to interpret that boundary. One rule beat a
narrower one that reads better in one incident.

### 5.1 The attribution columns still record the no-op, and this ADR does not move them

The mutation is unguarded. A no-op submit still runs the `UPDATE`, so `api_updated_by`,
`api_updated_at` and their five siblings take the submitting account and the current instant. The
settings panel renders that pair, so an operator who saves an unmoved toggle sees a fresh **By** and
**At** while the corpus writes nothing.

**This divergence is deliberate and narrow.** The two mechanisms answer different questions. An
attribution column answers *who last wrote this row*, which a no-op write genuinely did. An `Act`
answers *what a principal changed*, which a no-op did not. §9 governs the columns, and narrowing them
is a separate decision about a shipped rendering with its own reversal cost.

## 6. The guard is not atomic, stated rather than smoothed

> **Amended** by [ADR-1914: A dial move is serialised by a row lock inside one transaction](./1914-a-dial-move-is-serialised-by-a-row-lock-inside-one-transaction.md), 2026-09-13. <!-- adr-marker amends 1914 -->

**The comparison read and the mutation are two statements, and nothing holds them together.** Every
handler issues a standalone `SELECT`, then an unrelated `UPDATE`, on no transaction. Two
interleavings misbehave, and this ADR ships with both.

- **A repeated row.** API access stands on. Two admins submit *off*. Both read `true`, both write
  `false`, both compare `false != true`, and the corpus takes **two** `Dial moved · API access · off`
  rows for one move. The second row is the phantom §7.6 bars.
- **A lost row.** API access stands off. One admin reads `false` from a stale page. A second admin
  writes `true` and records it. The first then writes `false`, compares `false == false`, and records
  nothing. The estate is off and the corpus's last row says on.

**This is not new, and this ADR did not introduce it.** `updateCoverageRetention` has carried the
identical shape since the corpus shipped, and [`docs/spec/audit-act.md`](../spec/audit-act.md) §2.2
blessed it. What changed is the count: two sites carried the race before, and eight carry it now.

**The repair is a separate decision, on three grounds.** A conditional mutation —
`UPDATE … WHERE col IS DISTINCT FROM $1`, recording on rows affected — is atomic and drops the extra
read. It reaches **five** of the eight. The three retention dials share one row, so a per-dial
predicate cannot express them. A partial repair therefore puts two guard mechanisms in one corpus,
against §7.6 ruling 1's preference for one shape. It would also stop the no-op reaching the
attribution columns, which is §9's territory and §5.1's stated behaviour. And `cmd/web` holds no
transaction outside `restore.go` by §7.6 ruling 4, so the transactional repair fights that shape too.

**So the race is named here and repaired nowhere.**
[#1914](https://github.com/winniel123/verge-asm/issues/1914) carries it. A session that takes it up
rules on the mechanism first, and on §5.1 second.

## 7. What reversal costs

Unguarding the six handlers is a small diff. It does not undo the decision.

Every false row written between the reversal and the next ruling stays in the corpus. It sits beside
the true ones, with nothing on the row to tell them apart. A reader cannot recover which `Dial moved`
rows recorded a movement, because the distinguishing fact was never stored. That fact is the value
the dial held before the submit.

That is why this is a decision and not a detail of six handlers. Every later read of the corpus pays
the cost of getting it wrong, and the handlers pay none of it.
