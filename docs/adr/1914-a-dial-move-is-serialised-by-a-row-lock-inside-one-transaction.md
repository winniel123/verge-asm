---
number: 1914
title: "A dial move is serialised by a row lock inside one transaction"
slug: a-dial-move-is-serialised-by-a-row-lock-inside-one-transaction
date: 2026-09-13
status: accepted
source: fix
ticket: 1914
proof: {test: "cmd/web/dialtx_test.go::TestTwoConcurrentSubmitsOfOneDialMoveRecordOneRow"}
relations:
  - {kind: amends, adr: 1909, clause: "6"}
---

# ADR-1914: A dial move is serialised by a row lock inside one transaction

## Decision

**A dial's read, compare, write and record run in one transaction that opens with
`SELECT … FOR UPDATE` on the row the dial lives on.** All eight `act.DialMove` sites take one
wrapper.

Outside the lock, two submits of one move both read the old value and both record, so the
append-only corpus takes a second `Dial moved` row it can never retract. A stale read loses a row
instead: it compares equal after another admin moved the dial.

Rejected: `UPDATE … WHERE col IS DISTINCT FROM $1` recording on rows affected. It reaches five of
the eight sites, splits the guard in two, and stops a no-op stamping the attribution columns, which
reverses [ADR-1909](./1909-a-dial-submitted-at-its-current-value-writes-no-act-row.md) §5.1.
Rejected: an advisory lock, which buys a key convention for rows that already exist.

Reversal returns the race to eight sites.

**So `cmd/web` holds a transaction outside the restore path.**

## 1. Context

[ADR-1909](./1909-a-dial-submitted-at-its-current-value-writes-no-act-row.md) §6 named this race and
repaired none of it. It stated both interleavings, recorded that
[`docs/spec/audit-act.md`](../spec/audit-act.md) §2.2 had blessed the shape since the corpus shipped,
and left the ruling to [#1914](https://github.com/winniel123/verge-asm/issues/1914). The shape is
older than the guard: `updateCoverageRetention` carried it from the start. ADR-1909 changed the
count, from two sites to eight.

Each of the eight reads the stored value, writes the new one, compares the two Go values in process,
and records on a difference. No handler opened a transaction and none took a lock, so nothing held
the read and the write together.

The dials sit on three tables. `retention_settings` and `instance_config` are single-row, keyed
`WHERE id = true`. The zone and DNS cadences are one row each in `scan`, keyed by a `UNIQUE` kind.
Every dial therefore has a real row to lock.

## 2. Why the conditional `UPDATE` loses

It is the cheaper repair. It is atomic in one statement, it drops the extra read, and it needs no
transaction. It fails on reach.

**It reaches five sites.** Three retention dials share one `retention_settings` row, and two
different handlers write that row with different subsets of its columns. A per-column predicate
cannot say *this dial moved* for a submit that rewrites its two siblings from the values it just
read. `updateRetention` and `updateCoverageRetention` would each need a predicate over columns the
other one owns.

**A partial repair is two mechanisms.** Five sites guarding on rows affected and three guarding on a
locked read is the outcome §7.6 ruling 1 argues against, and the split holds for as long as nobody
closes it.

**It reverses ADR-1909 §5.1.** `WHERE col IS DISTINCT FROM $1` matches no row on a no-op, so
`api_updated_by`, `api_updated_at` and their five siblings stop taking the submitting account. §5.1
commits to the opposite, and the settings panel renders that pair. Narrowing the attribution columns
is §9's territory and a separate decision with its own reversal cost. This ADR does not make it.

## 3. Why a row lock and not an advisory lock

`pg_advisory_xact_lock` is the tool for a critical section with no row to hold. Here every dial has
one. An advisory lock would add a global key convention — a numbering scheme, a place to record it,
and a rule for the next session that needs one — to guard rows that `FOR UPDATE` already guards.

`FOR UPDATE` is also the narrower lock. It blocks a second writer of the same settings row and
nothing else. The three retention dials share a row, so locking the row serialises them together,
which is the exact case the conditional `UPDATE` could not express.

## 4. What the transaction now holds

> **Amended** by [ADR-1944: The observation floor is derived in the statement that writes the dial](./1944-the-observation-floor-is-derived-in-the-statement-that-writes-the-dial.md), 2026-09-14. <!-- adr-marker amends 1944 -->

The wrapper takes the lock, reads, mutates, compares and records, then commits. `txRecorder` already
existed for the restore path, so the `Act` insert binds to the transaction without a second
mechanism.

**The record commits with the move it records.** A failed insert rolls the mutation back, where
before it left the move standing and logged the loss. This follows §7.6's reading for the restore: a
failed statement poisons the transaction, so the whole act must go.

**A cancelled request now drops both.** The pool recorder detaches its context to outlive an operator
navigating away, per §7.6 ruling 5. A transaction cannot: a detached insert would outlive the
rollback tearing its own transaction down. So a cancelled dial submit leaves neither a move nor a
row. Ruling 5 protects the corpus from a move with no row, and that outcome is now unreachable.

**The request body is read before the lock, not under it.** The first `r.FormValue` pulls the body off
the network. Taken inside the section it would let a slow client hold the settings row for the read
timeout, and block every other submit of that row. Each handler reads its form values first and closes
over them.

**The coverage floor is read inside the transaction.** `updateCoverageRetention` clamps to a floor
derived from the enabled scans, so `ListEnabledScans` and `ListCoveringScanKinds` moved into the
section. This does not close the floor's own staleness: the zone and DNS dials write `scan`, and a
cadence committed after these two reads still leaves the clamp reading a floor that has moved. Closing
that needs a lock on the scan rows, which is ADR-0081's question and not this one.

## 5. The price, named rather than hidden

**`cmd/web` holds a transaction outside `restore.go`.** ADR-1909 §6 read §7.6 ruling 4's shape as an
argument against the transactional repair. This decision accepts the cost instead, because the
alternative reaches five sites of eight. [`docs/spec/audit-act.md`](../spec/audit-act.md) §7.6 now
names two exceptions rather than one, and carries this repair as ruling 10.

The cost is one connection held across four statements on an admin route, and a second place where a
`defer tx.Rollback` must be right. It is identical under any mechanism that serialises a compare
against a write, so it is not a cost the conditional `UPDATE` would have avoided at the three shared
dials.

**The seam is a store method, not a pool field.** `pgStore` wraps the generated queries and the pool
together and is what `newServer` receives, so a deployment cannot reach a dial handler with the guard
unwired. A pool assigned after construction, as `restore.go` takes it, can be forgotten.

**The `Record` conformance gate reads one more shape.** The critical section takes its recorder as a
parameter, so the receiver is a name rather than `s.recorder()` or `txRecorder(tx)`. The gate now
binds names off the `recorder` type in the declaration, which is narrower than matching a name.

## 6. What reversal costs

Removing the wrapper is a small diff at eight sites. It puts both interleavings back, and every row
written in between stays in the corpus with nothing on it to say which reading produced it.

That is the same permanence ADR-1909 §7 named. A repeated row and a true row are identical once
written, so a reader cannot separate them later. The distinguishing fact is the value the dial held
when the losing submit read it, and no column holds it.
