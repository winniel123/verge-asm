---
number: 1944
title: "The observation floor is derived in the statement that writes the dial"
slug: the-observation-floor-is-derived-in-the-statement-that-writes-the-dial
date: 2026-09-14
status: accepted
source: fix
ticket: 1944
proof: {test: "cmd/web/retentionpanel_test.go::TestAScanWriteUnderTheSectionCannotPersistABelowFloorDial"}
relations:
  - {kind: amends, adr: 1914, clause: "4"}
---

# ADR-1944: The observation floor is derived in the statement that writes the dial

## Decision

**The observation currency dial is written by an `UPDATE` that derives its floor in the same
statement, and the clamped value comes back through `RETURNING`.**

The dial lock is on `retention_settings` and holds no `scan` row. Under READ COMMITTED a write to
`scan` that commits between a floor read and the `UPDATE` leaves a value below the floor then in
force.

Rejected: `FOR SHARE` over the enabled scans. It cannot lock a row that is still `FALSE`, and it
touches no `batch` or `observation` row, so the worker moves the floor straight through it.
Rejected: re-read and re-clamp before the write, which makes the window smaller and leaves it open.

The price is one derivation expressed twice, in Go and in SQL.

Reversal puts the window back and returns the floor to one home.

## 1. Context

[ADR-1914](./1914-a-dial-move-is-serialised-by-a-row-lock-inside-one-transaction.md) §4 moved
`ListEnabledScans` and `ListCoveringScanKinds` inside the dial transaction, and named what that did
not close. Its own words: a cadence committed after these two reads still leaves the clamp reading a
floor that has moved. It left the ruling to [#1944](https://github.com/winniel123/verge-asm/issues/1944).

The floor is derived, never stored.
[ADR-0081](./0081-a-floor-is-territory-and-an-unbounded-default-is-a-position.md) and
[ADR-0094](./0094-a-retention-control-collapses-and-a-retention-query-never-does.md) both rule
that. It is `retention.FloorCadences` multiplied by the tightest cadence among the covering scans,
divided by the seconds in a day, rounded up. A scan covers when it is enabled and some observation
of its own reaches a batch.

Four writers move that floor, and the dial lock serialises against none of them. The zone and the
DNS cadence dials write `scan`. The cold-custody handler flips `scan.enabled` from a subquery over
the declared scopes. The worker writes `batch` and `observation`, and the observation sweep deletes
them, so a cover appears or drops with no operator present at all.

## 2. Why a lock on the scan rows loses

It is the obvious repair, and it was the one ADR-1914 §4 pointed at. It fails on reach, twice.

**It cannot lock the row that matters most.** `SyncColdScanEnabled` sets `enabled` from an `EXISTS`
over the cold scopes, so it can move the `cold` row from `FALSE` to `TRUE`. A `FOR SHARE` read of
`WHERE enabled = TRUE` never locks a row that is currently `FALSE`. That is a phantom, and no lock
taken over a predicate excludes one.

**It does not reach the worker.** The cover relation is a semi-join onto `batch` and `observation`.
A lock the web command takes on rows in `scan` serialises against no writer of those two tables.
The worker needs no operator, so this is the row a discipline between two admins cannot cover.

Re-reading the floor and comparing before the `UPDATE` loses for a plainer reason. It moves the
window, from between the reads and the write to between the second read and the write. A smaller
window is not a closed one, and the next session reading the code cannot tell that it was meant to
be narrow rather than absent.

## 3. Why the derivation may live in two places

The floor stays derived, so this ADR reopens neither ADR-0081 nor ADR-0094. What it admits is a
second expression of the same derivation: `retention.ObservationFloorDays` in Go, and the `CASE` in
`UpdateCoverageRetentionSettings` in `db/queries/retention.sql`.

Three things hold the two in step.

**The numbers cross the boundary as parameters.** The statement takes `floor_cadences` and
`seconds_per_day` as arguments. `retention.FloorCadences` and `retention.SecondsPerDay` remain the
only place either constant is written down. A change to `k` moves both sides at once.

**The arithmetic is written the same way on both sides.** The SQL rounds up as
`(bound + spd - 1) / spd` over integers, which is the Go expression character for character. A
reader comparing the two compares one line.

**The cover relation is one query's shape, copied.** The `EXISTS` in the statement is
`ListCoveringScanKinds`' own semi-join. The renderer still reaches the floor through the Go path,
so the two are exercised side by side on every load of the panel.

The alternative was to keep one home for the derivation and accept the window. The window is the
thing the ticket exists to close, so the duplication is the cost this decision pays rather than the
one it avoids.

## 4. What the statement reaches

The clamp now reads the floor committed as of its own snapshot, in the statement that writes the
row. No `scan`, `batch`, or `observation` write can interleave between the two, because there is no
longer a between. That answer covers all four writers, the worker included.

It does not reach forward. A floor that rises after this transaction commits leaves the dial below
its own stated ground until the next submit re-clamps it, exactly as a floor that rises with no
submit in flight does. That is the standing behaviour of a derived floor and not a race, and
closing it is a question about the dial's meaning rather than its serialisation.

The `retention_settings` row lock stays. It guards the compare against the stored value that decides
whether an `Act` row is written, which is ADR-1914's subject and is untouched here. The guard now
compares the value the statement returned, so a clamp the statement applied is a move the corpus
records.

## 5. What the proof can and cannot show

The repo holds no live-Postgres test. Every test in `cmd/web` runs against an in-memory fake, whose
dial transaction is a mutex with no row locks and no snapshot. The proof is shaped by that.

The fake models the statement: it reads the scan set at write time and derives the floor from the
constants the handler passed it. The test commits a competing write at the last instant before the
statement runs, in three interleavings — cold custody withdrawing the tightest cover, the sweep
retiring the last observation under it, and cold custody enabling a tighter one. Each asserts what
landed against the floor then in force. All three fail against the shape this ADR replaces.

What this does not prove is that the SQL says what the Go says. No offline check reads a query's
meaning, and `sqlc` gates the generated types alone. Two of the three ties in §3 are readable in one
screen for that reason.

## 6. What reversal costs

Reverting is a small diff: the handler reads the two scan queries again and clamps in Go. It puts
the window back at every one of the four writers, and it restores a single home for the derivation.

The value already persisted below a floor is not a permanent record. It is a dial position, and the
next submit re-clamps it, which is why this decision was reached on reach rather than on damage.
