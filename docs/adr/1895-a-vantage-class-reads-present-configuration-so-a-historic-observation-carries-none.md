---
number: 1895
title: "A Vantage class reads present configuration, so a historic observation carries none"
slug: a-vantage-class-reads-present-configuration-so-a-historic-observation-carries-none
date: 2026-09-13
status: accepted
source: grilling
ticket: 1895
proof: {test: "internal/queue/vantageclass_history_test.go::TestReadBatchLegsClassifiesBothLegsUnderOneBinding"}
relations:
  - {kind: rests-on, adr: 105}
  - {kind: rests-on, adr: 92}
  - {kind: bounds, adr: 14}
---

# ADR-1895: A Vantage class reads present configuration, so a historic observation carries none

## Decision

**A `Vantage class` is a reading of present configuration, so a historic observation carries none.**
Every leg of one comparison classifies under one binding, so a scope edit moves the whole history at
once and the exposure delta reports no drift the measurements did not carry.

Rejected: pin the class at the instant it was recorded, on the `Batch` or on the span row.

A reader asks why, because a class that re-reads the past looks like a defect. The class verifies
declared intent against the scopes the operator holds **now**, and a live read under `now` beside a
historic read under `then` manufactures drift in the flagship value.

Reversal costs two corpora that do not exist — a per-instant scope set, and a per-observation
presented address — and a backfill with no source, so every row before the change reads under a class
nothing ever wrote.

## 1. Context

[#1890](https://github.com/winniel123/verge-asm/issues/1890) measured where the class is derived, and
[#1896](https://github.com/winniel123/verge-asm/issues/1896) carried the finding into
[CONTEXT.md](../../CONTEXT.md): the class is derived wherever it is used, from the addresses the
vantage presents and the declared address `Seed`s, and no row stores it. The `vantage.class` column is
vestigial and read by nothing ([#709](https://github.com/winniel123/verge-asm/issues/709)).

That entry states **where** the derivation happens. It states nothing about whether the answer is
stable, and #1896 left the question here on purpose. This ADR answers it.

Ten call sites reach `vantageclass.Derive`. Eight read present state. **Two compare across time**,
and those two are the whole of the exposure at issue:

| Site | What it reads | Historic? |
| --- | --- | --- |
| The dispatch realm gate (`internal/scan/hot.go`, `cold.go`, `tlsacceptance.go`, `httpidentity.go`) | the vantages it is about to dispatch | no |
| `vantageFactsClass` and its fan-outs (`cmd/web/auth.go`, `cmd/web/exposure.go`, `cmd/web/settings.go`, `cmd/web/signals.go`) | every `Vantage` row | no |
| `foldExposure` (`cmd/web/exposure.go`), `currentExposedCount` (`cmd/web/deltas.go`) | the **open** reachability spans | no |
| `collapseNameResolutions` (`cmd/web/signals.go`) | the live resolution tier | no |
| The widening census (`internal/queue/vantageclass.go`) | every `Vantage` row, under the fold's own binding | no |
| **`exposureCountDeltas`** (`cmd/web/deltas.go`) | the open spans **and** a snapshot at the previous batch | **yes** |
| **`readBatchLegs`** (`internal/queue/produce.go`) | the same pair, bounded to the batch's candidate services | **yes** |

**The second one alerts.** `readBatchLegs` derives one `covered`, applies it to `legsFromCurrent` and
to `legsFromAt` at `PreviousBatchTime`, and hands the pair to `flagshipMessages`, which asks
`exposure.Flagship(before, after)` and writes the `Message`. So a split binding there would not move
a dashboard figure. It would fire, or withhold, a real alert.

Both sites already bind once, and both say so in a comment beside the binding. That is what makes
this a decision about the term rather than a detail of one handler: the rule holds at two
independent readers, and the one that matters most is the one an operator is paged by.

## 2. An as-of scope predicate cannot be built, and would be silently wrong

Pinning needs a predicate that answers *was this address covered then*. The database cannot answer it.

**A `Seed` withdrawal is a hard `DELETE`.** `WithdrawSeed` deletes the `seed` row and writes a
`seed_withdrawal` tombstone in the same statement, and migration `24700`'s own note gives the ground:
a soft delete would put a filter on every existing reader, and a missed filter re-admits a withdrawn
scope silently. So the surviving record of a scope is one of two shapes — a live `seed` row carrying
the **latest** declaration's `created_at`, or a tombstone carrying a withdrawal's.

Declare a scope, withdraw it, declare it again, and the first declaration's instant survives nowhere.
An as-of read at an instant inside that first window returns *not covered*, which is a wrong answer
rather than a missing one. [ADR-0134](./0134-a-seed-withdrawal-is-recorded-by-a-tombstone-because-the-mover-does-not-survive-the-act.md)
built the tombstone to name a mover, not to date a scope's life, and it says so: *"Withdraw a scope,
declare it again and withdraw it again, and each act is its own dated fact."* Each **act**, not each
interval.

**The predicate is not the scope list alone.** `addressScopeCovered` subtracts the live `exclusion`
corpus, because an excluded range is not the operator's
([ADR-0133](./0133-an-address-exclusion-is-a-limb-of-the-custody-derivation.md) §4). An `exclusion`
is also hard-deleted, and it carries no tombstone at all. So the second limb of the predicate has
strictly less history than the first.

A pinned class built on either would read a boundary the operator never held. That is worse than
reading today's, which is at least a boundary they do hold.

## 3. The other input is current too, so pinning the scope side pins nothing

The ticket proposes the span row as a pinning site, on the ground that its `dialled_addr` and `egress`
already sit there. **They do not.** The `span` table carries no address column
(`db/migrations/19000_span.sql`). `ListServiceReachabilitySpansByClass` and its `…At` twin reach the
two facts through `JOIN vantage v ON v.id = sp.vantage_id`, and select `v.egress` and
`v.dialled_addr`.

Those two columns are overwritten in place. `SetVantageProbeFacts` sets `platform`, `egress` and
`dialled_addr` on each probe, and keeps no prior value. A prober that changes egress address
therefore re-reads every past leg's class already, by the same mechanism and with no scope edit
involved. The ticket does not name this input, and it is the one a pinned scope set would leave
moving.

So a pinned class is not one corpus. It is two: a per-instant scope set, and a per-observation record
of what the vantage presented. Neither exists, and neither can be backfilled from what does.

## 4. Why a pinned class manufactures drift in the flagship

[CONTEXT.md](../../CONTEXT.md) already holds the hazard, on the `Custody` separation: *"a `Vantage`
whose class moved because a DNS answer changed would shuttle observations between the two legs of
`Exposure` and manufacture drift in the flagship value."*

Under the rejected alternative, an address-scope edit does exactly that, and it needs no DNS answer to
do it. The `cur` leg would read the boundary as declared now. The `prev` leg would read a boundary
from before the edit. The two legs then disagree about which class a leg's value lands on.

At `exposureCountDeltas` the cost is a widget: it reports a swing in `exposed`, `firewalled` or
not-reached that no measurement carried, and the operator reads a change in their own configuration
as a finding about their estate.

**At `readBatchLegs` the cost is an alert.** `composeInternetLeg` reads the internet leg out of each
side, and `exposure.Flagship` fires on `not-reached` → `reached` across the pair. A vantage that
crosses the boundary between the two legs moves its outcome onto, or off, the internet leg — so the
pair can transition with nothing measured. The operator is paged for their own scope edit, or, in the
other direction, a real transition beneath the edit goes unpaged. This is the site
[ADR-0029](./0029-an-alert-fires-on-a-leg.md) governs, and a manufactured firing there is worth more
than a wrong tile.

This is the closed direction the class test was narrowed for, read at the other end.
[CONTEXT.md](../../CONTEXT.md) narrows the live test to `internet` *"because a vantage wrongly read as
`internal` moves observations onto the leg that never alerts."* That clause protects a **live** read,
and the ticket is right that it says nothing about a historic one. The protection a historic read
needs is different, and it is this: the two legs of a comparison must not be read under two boundaries.

A test holds each of the two sites, and §10 quotes both. Each fixture measures one `Service` at two
vantages with the same outcome at both instants, so nothing in the measurement can move. Split either
binding and the same fixture reports a change.

## 5. The operator is told nothing, and that is the existing rule

The ticket asks whether an address-scope edit that moves a class is an event the operator is told
about. **No, and no new `Message` is minted for it.**

[ADR-0092](./0092-an-operator-dials-movement-is-not-a-cause-and-an-annotation-never-lapses.md) rules
that an operator dial's movement is not one of the causes, so it fires no `Message`. Declaring or
withdrawing an address `Seed` is an operator act, so the same rule reaches it: the system announces
what it found, never what the operator just did. The act is recorded where an act belongs —
`seed.declared` in the `Act` corpus
([ADR-1891](./1891-an-operator-act-is-recorded-as-a-fifth-operational-corpus-under-a-four-limb-predicate.md),
one row per scope under
[ADR-1902](./1902-one-act-row-per-subject-never-one-per-request.md)) — and the operator is the
principal that submitted it.

There is also nothing here for a message to be *about*. A message announcing "your history was
reclassified" presupposes a recorded class that moved. Under this decision there is none. What moved
is the reading, and it moved everywhere at once.

## 6. What is not stable, stated rather than smoothed

`Exposure`'s flagship value is **not** stable under an address-scope edit, and this decision does not
make it so. A scope edit changes what the operator's boundary is, so it changes what
`exposed`/`firewalled` mean, so it changes the count. The dashboard's absolute figure may move between
two page loads with no batch between them.

That is the intended reading, because the class *is* the boundary the operator declares, and a figure
computed against a boundary they no longer hold is the wrong figure. What this decision buys is
narrower and provable: **a comparison never straddles two boundaries.** §4's test is the assertion.

## 7. Three near neighbours, and why none of them rules this

| ADR | Why it differs |
| --- | --- |
| [ADR-0105](./0105-inventory-is-a-read-over-the-open-span-corpus-not-a-second-thesis.md) | The nearest *derive rather than store* ruling: inventory is a read over the open-span corpus, not a second corpus. It rules that a **view** needs no corpus of its own. It says nothing about a classifier inside a read, nor about a read whose rows are historic |
| [ADR-0014](./0014-only-revealed-generalises.md) | Detects a widening by diffing a `Batch`'s recorded scope against the prior one's. #1890 confirmed the class satisfies that criterion through a normalised join rather than a scope field. The diff needs a recorded value to disagree with, and this dimension records none, so ADR-0014 catches the widening and cannot reach the re-reading |
| [ADR-0008](./0008-derivation-versions-move-on-content.md) | The nearest thing to a pinned classifier that this repo **does** keep: a span carries the `Derivation` vector it was produced under, and two spans compare only where the vectors are equal. That vector pins the rules that produced the **value**. The class produces no value — it selects which leg a value lands on — and it is a declaration of intent rather than a component version, so it is not a leaf of that vector |

## 8. Consequences

- **No migration and no schema change.** The decision is that nothing new is recorded.
- **Both historic readers keep their single binding, and each now has a test that fails if it is
  split.** `exposureCountDeltas` and `readBatchLegs` keep the comments beside their bindings, and §4
  is the citation for both.
- **The vestigial `vantage.class` column stays vestigial.** Nothing here gives it a reader. A pinned
  class would not have lived there either: it is per-vantage, and the pin would need to be
  per-observation.
- **[CONTEXT.md](../../CONTEXT.md)'s `Vantage class` entry is unchanged.** #1896 wrote it to state
  where the derivation happens and to claim nothing about stability. That wording is correct under
  this decision, so this ADR adds no edit to it.

## 9. Where this is thin, stated rather than smoothed

> **Amended** by [ADR-1946: An address-scope edit reaches the operator through an Act reader, never a Message](./1946-an-address-scope-edit-reaches-the-operator-through-an-act-reader-never-a-message.md), 2026-09-14. <!-- adr-marker amends 1946 -->

**The frequency is unmeasured.** The ticket says so, and nothing here changes it. Nobody has counted
how often an operator edits an address scope that covers a prober's presented address. If it turns
out to be common, the reading of the past moves often, and the argument of §6 gets harder to hold —
not because it becomes false, but because an operator meets a moving absolute figure more often than
the design assumed.

**The widening message does not cover this.** §5 rules out a new message on the ground that an
operator act is not a cause. It is worth being plain that the existing message does not step in
either. `vantageClassMessages` suppresses a widening when
`ReachFoldedBeforeAtVantages` finds an earlier completed batch that folded `Reach` at any vantage of
the candidate class — and after a scope edit the vantage newly read as `internal` is usually the same
vantage that folded before. So the first fold after the edit can open a leg in a class the operator
has never seen a message for. #1890 verified that mechanism against the widening it was built for.
This is a different case, and it is unaddressed rather than handled.

**The eight present-state sites are not covered by §4's tests, and cannot be.** The four dispatch
sites derive the class to pick a probing realm, and a gate must read present configuration. The other
four read `Vantage` rows or open spans. None of the eight compares across time, so the invariant §4
proves has nothing to say about them, and none is claimed for them.

**The census in §1 is a reading of the tree on this date, not a fence.** Nothing stops an eleventh
call site, and no check counts them. A new two-legged reader that derives its own second binding would
break this rule with both §10 tests green. The repair, if that becomes a real risk, is a seam that
hands out one binding per fold rather than a rule written down here.

## 10. Proof

`{test: "internal/queue/vantageclass_history_test.go::TestReadBatchLegsClassifiesBothLegsUnderOneBinding"}`.

One test per historic reader, and both fixtures measure `203.0.113.10:443/tcp` at two vantages —
`not-reached` at the one presenting `198.51.100.200`, `reached` at the one presenting `192.0.2.7` —
with the same rows on both legs. Nothing in the measurement can move, so anything that moves is the
classifier.

**The alerting site.** `TestReadBatchLegsClassifiesBothLegsUnderOneBinding` runs `readBatchLegs` twice,
once with `192.0.2.0/24` declared and once without, and asserts the class of each `prev` leg against
its `cur` twin:

> `t.Errorf("prev leg %d = %q but cur reads %q; one binding serves both", i, prev[i], cur[i])`

Give `legsFromAt` its own predicate and the subtest fails, naming the disagreement:
`prev … "=internet" but cur reads "=internal"`. Its sibling
`TestAScopeEditAloneFiresNoFlagship` runs the whole of `produceMessages` under both scope settings and
asserts no `service` message is written either way.

**The dashboard site.** `cmd/web/vantageclass_history_test.go::TestAScopeEditMovesBothExposureDeltaLegsTogether`
holds the same rule at `exposureCountDeltas`. Before the scope is declared, both vantages read
`internet`, one class holds both readings, and the `Service` has no `Exposure` value:

> `{"not-reached", notReached, drift.Delta{Current: 1, Previous: 1}}`

Declaring `192.0.2.0/24` moves the second vantage to `internal` and the `Service` to `firewalled`, in
both legs at once:

> `{"firewalled", firewalled, drift.Delta{Current: 1, Previous: 1}}`

Classify the `prev` leg under the pre-edit boundary instead and the same fixture reports
`firewalled {Current: 1, Previous: 0}` — a swing of one, from an edit.
