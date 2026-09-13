---
number: 1895
title: "A Vantage class reads present configuration, so a historic observation carries none"
slug: a-vantage-class-reads-present-configuration-so-a-historic-observation-carries-none
date: 2026-09-13
status: accepted
source: grilling
ticket: 1895
proof: {test: "cmd/web/vantageclass_history_test.go::TestAScopeEditMovesBothExposureDeltaLegsTogether"}
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

Five sites derive the class. Four of them read present state, and one reads history:

| Site | What it reads | Historic? |
| --- | --- | --- |
| Dispatch (`internal/scan/hot.go`, `internal/scan/cold.go`) | the vantages it is about to dispatch | no |
| The dashboard chip (`cmd/web/auth.go`) | every `Vantage` row | no |
| `foldExposure` (`cmd/web/exposure.go`) | the **open** reachability spans | no |
| `collapseNameResolutions` (`cmd/web/signals.go`) | the live resolution tier | no |
| `exposureCountDeltas` (`cmd/web/deltas.go`) | the open spans **and** a snapshot at the previous batch | **yes** |

So the whole of the exposure at issue is one function: the `prev` leg of the dashboard's exposure
delta. It already classifies both legs under one binding, and the comment beside it says why. This
ADR is the record of that choice, because it is a decision about the term and not a detail of one
handler.

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
do it. The `cur` leg of `exposureCountDeltas` would read the boundary as declared now. The `prev` leg
would read a boundary from before the edit. The two legs then disagree about which class a leg's value
lands on, and the widget reports a swing in `exposed`, `firewalled` or not-reached that no measurement
carried. The operator reads that as a change in their estate. It is a change in their own
configuration, already known to them, reported as a finding.

This is the closed direction the class test was narrowed for, read at the other end.
[CONTEXT.md](../../CONTEXT.md) narrows the live test to `internet` *"because a vantage wrongly read as
`internal` moves observations onto the leg that never alerts."* That clause protects a **live** read,
and the ticket is right that it says nothing about a historic one. The protection a historic read
needs is different, and it is this: the two legs of a comparison must not be read under two boundaries.

`TestAScopeEditMovesBothExposureDeltaLegsTogether` holds it. The fixture measures one `Service` at two
vantages with the same outcome at both instants, so every delta leg must read zero. Declaring a scope
that covers one vantage's presented address moves the reading from not-reached to `firewalled`, and
moves `Current` and `Previous` together. Classify the `prev` leg under the pre-edit boundary instead
and the same fixture reports `firewalled` `{Current: 1, Previous: 0}` — a swing of one, from an edit.

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
- **`exposureCountDeltas` keeps its single binding, and now has a test that fails if it is split.**
  The comment beside the binding stays; §4 is its citation.
- **The vestigial `vantage.class` column stays vestigial.** Nothing here gives it a reader. A pinned
  class would not have lived there either: it is per-vantage, and the pin would need to be
  per-observation.
- **[CONTEXT.md](../../CONTEXT.md)'s `Vantage class` entry is unchanged.** #1896 wrote it to state
  where the derivation happens and to claim nothing about stability. That wording is correct under
  this decision, so this ADR adds no edit to it.

## 9. Where this is thin, stated rather than smoothed

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

**The dispatch sites are not covered by §4's test.** `internal/scan/hot.go` and
`internal/scan/cold.go` derive the class to pick a probing realm, and they read present configuration
because a gate must. No comparison happens there, so the invariant §4 proves has nothing to say about
them, and none is claimed.

## 10. Proof

`{test: "cmd/web/vantageclass_history_test.go::TestAScopeEditMovesBothExposureDeltaLegsTogether"}`.

The fixture measures `203.0.113.10:443/tcp` at two vantages, `not-reached` at the one presenting
`198.51.100.200` and `reached` at the one presenting `192.0.2.7`, at both instants. Before the scope
is declared both vantages read `internet`, one class holds both readings, and the `Service` has no
`Exposure` value:

> `{"not-reached", notReached, drift.Delta{Current: 1, Previous: 1}}`

Declaring `192.0.2.0/24` moves the second vantage to `internal` and the `Service` to `firewalled`, in
both legs at once:

> `{"firewalled", firewalled, drift.Delta{Current: 1, Previous: 1}}`

The final loop is the invariant itself — `Current` equals `Previous` for every value — and it fails
with a swing of one when the `prev` leg is classified under a separate binding.
