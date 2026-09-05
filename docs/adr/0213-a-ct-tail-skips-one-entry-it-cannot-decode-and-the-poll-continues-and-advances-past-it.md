# ADR-0213: a CT tail skips one entry it cannot decode, the poll continues, and the cursor advances past it

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1323 ADR gaps: internal/queue (queue, cttail, withdrawal, hot, ctverify, scopegate)](https://github.com/winniel123/verge-asm/issues/1323), gap 1
- **PR that deleted the comment:** [#1322](https://github.com/winniel123/verge-asm/pull/1322)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0106](./0106-the-ct-poll-is-a-scan-that-schedules-and-a-ct-admission-is-a-name-citing-its-batch.md), which rules that a non-200 admits nothing and never an absence. It rules the **whole response**. It says nothing about one entry inside a well-formed response
- **Rests on:** [ADR-0096](./0096-a-citation-never-ages-it-is-contradicted-and-only-an-enumerable-sources-silence-can-do-it.md), which rules that a citation never ages and that only an enumerable source's silence contradicts one. It is what makes a name the tail missed a delayed discovery rather than a wrong fact
- **Sibling of, and not ruled by:** [ADR-0141](./0141-a-periodic-sweep-loop-logs-and-continues-because-the-next-tick-retries-and-the-legibility-rule-does-not-reach-it.md). That ADR lets a periodic loop log a failed pass and keep ticking, and its §2 fixes the ground as *the next tick retries the same work*. A skipped CT entry is never read again. ADR-0141 §2 states its own bound — *"a pass whose failure is not retried by the next tick is outside this rule and gets no cover from it"* — so this skip needs a different ground, and §3 below supplies it
- **Bounded by, and it bounds:** [ADR-0191](./0191-an-unrecognised-ct-entry-type-costs-one-entry-on-the-framed-rfc-6962-path-and-fails-the-whole-unframed-tile.md). It rules the **parse** layer's answer to a value it does not know: `LeafSANs` returns `(nil, nil)` for an unrecognised `entry_type`, and `parseTileLeaf` fails the whole tile for the same value. This ADR rules the **queue** layer's answer to an entry that framed correctly and whose bytes it cannot read as a certificate. The two inputs are disjoint and §1 below sets the boundary. ADR-0191 §2's *"the tiled path refuses"* must not be read as *the tiled path never skips a leaf* — it does, on this ADR's input class, at `cttail.go:167`
- **Sibling of, and not ruled by:** [ADR-0207](./0207-an-enumeration-that-assembles-a-probing-target-set-drops-a-row-it-cannot-fully-name-and-never-fabricates-a-target.md). That rule runs on the **enumeration** side: a read that assembles a probing target set drops a row it cannot fully name, because a fabricated target would be probed. This rule runs on the **ingest** side: an entry the decoder cannot read is dropped, because a guessed name would be admitted. Both refuse to reconstruct a subject from a partial record. Neither contains the other, because an enumeration-side drop costs a probe the next cadence retries and this one is permanent (§2)

## Context

[`internal/queue/cttail.go:134`](../../internal/queue/cttail.go) and `cttail.go:227` carried this
text, until #1322 deleted it:

```go
// One malformed or unrecognised entry does not fail the poll: the tail
// reads a corroborative source, so it skips the entry and reads on.
```

The sweep left one compressed restatement on the RFC arm (`cttail.go:93`) and left the tiled arm's
identical skip with **no comment at all** (`cttail.go:168`). Nothing on disk states the rule. That
is #1323's gap 1.

**The rule is a boundary, and the boundary is the response, not the poll.** Both tail arms are dense
with failure handling, and every other failure ends the poll. The two arms hold **nine**
fail-the-poll sites and **two** skip sites.

| Failure | Site | Direction |
| --- | --- | --- |
| `get-sth` fetch or non-200 | `cttail.go:59` | fail the poll |
| `get-sth` body does not parse | `cttail.go:64` | fail the poll |
| `get-entries` fetch or non-200 | `cttail.go:80` | fail the poll |
| `get-entries` body does not parse | `cttail.go:84` | fail the poll |
| **one entry's SANs do not decode** | **`cttail.go:92`** | **skip and continue** |
| `checkpoint` fetch or non-200 | `cttail.go:121` | fail the poll |
| `checkpoint` body does not parse | `cttail.go:125` | fail the poll |
| tree size below the cursor | `cttail.go:131` | fail the poll |
| `tile/data` fetch or non-200 | `cttail.go:150` | fail the poll |
| `tile/data` body does not parse | `cttail.go:154` | fail the poll |
| **one tiled leaf's SANs do not decode** | **`cttail.go:167`** | **skip and continue** |

A fail-the-poll site calls `retryOrDeadLetterCT`. That retries on the queue backoff and then
dead-letters a `Batch` with an empty scope. That is ADR-0106's machinery. The two skip sites call
`w.log.Printf` and `continue`.

**The skip is permanent, and this is the fact the naive reading misses.** The cursor does not track
the entries that decoded. The RFC arm advances by the count the log returned
(`reached += int64(len(entries))`, `cttail.go:99`). The tiled arm advances to the end of the slice
it walked (`reached = tileBase + int64(limit)`, `cttail.go:173`). A skipped entry sits inside that
count. `admitCTTail` then writes `reached` through `AdvanceCTLogCursor`, and the next poll starts
there. `cttail.go:50` records the other half: *"the tail reads only forward deltas and never
backfills"*. So this product never reads a skipped entry again.

That measurement is what separates this rule from
[ADR-0141](./0141-a-periodic-sweep-loop-logs-and-continues-because-the-next-tick-retries-and-the-legibility-rule-does-not-reach-it.md).
Nine loops in the tree swallow a failure because the next tick does the same work again. This skip
swallows a failure that nothing repeats.

## Decision

> **A CT tail poll fails on a failure of the whole response, and it skips on a failure of one entry
> inside a well-formed response. The tail logs the entry, skips it, reads on, and advances the
> cursor past it. The skip is permanent. Its ground is that the tail reads a corroborative source
> and asserts no absence. Its ground is never that a later poll retries the entry.**

Four limbs.

### 1. The unit of the failure decides the direction, and the input class is content

Three failure classes arrive at a tail arm, and only the third is this ADR's.

| Class | What fails | Where it is decided | Direction |
| --- | --- | --- | --- |
| **Framing** | where one entry ends and the next begins | `ParseLogEntries`, `ParseDataTile`, `parseTileLeaf` | fail the poll (ADR-0191 §2, and §4 below) |
| **Vocabulary** | an `entry_type` the tail does not know | `LeafSANs`'s `default` arm | `(nil, nil)`. No names, no error, no skip (ADR-0191 §1) |
| **Content** | a leaf that framed, whose bytes are not a readable certificate | the `derr != nil` branch at `cttail.go:92` and `:167` | **skip the entry, continue the poll** |

The content class is five reachable errors: a leaf shorter than the header, an unsupported leaf
version, an unsupported leaf type, a truncated or zero `opaque<1..2^24-1>` length, and a DER that
`x509.ParseCertificate` refuses (`internal/scan/cttail.go:242`, `:245`, `:248`, `:290` and `:277`).
On the tiled arm only the last of those is reachable, through `CertSANs`.

A failure of the transport, the status code, or the body's own frame is a failure of the **whole
response**. The tail learned nothing from it. ADR-0106 therefore applies without change: the poll
retries, and past the attempt budget it dead-letters a `Batch` with an empty scope.

A failure to read **one entry** inside a body that framed correctly is a different fact. The tail
read the response. It read the entries before that one and it can read the entries after it. A poll
that fails here discards every name the response carried, so one unreadable leaf costs the whole
batch.

Both decode calls sit at this boundary. `scan.LeafSANs(e.LeafInput, e.ExtraData)` on the RFC arm and
`scan.CertSANs(ders[i])` on the tiled arm each take one entry and return one entry's names.

### 2. The cursor advances past the skipped entry, and the loss is accepted

The alternative is a cursor that holds at the first entry it cannot read. That turns one undecodable
leaf into a stalled log. The poll re-reads the same entry every cadence, it admits no name after it,
and the tail stops following that shard for good. A permanent stall is a worse failure than a
permanent gap of one entry, because the stall grows without bound and the gap does not.

The product therefore accepts the loss. The entry's names are not admitted, and no later poll
recovers them.

### 3. The ground is corroboration, and it is not retry

The tail is a corroborative source ([`ct-source-replacement.md`](../spec/ct-source-replacement.md)
§2.7). It admits `Name`s and observes nothing (ADR-0027, ADR-0106). A name it did not admit is a
name the estate has not yet heard of from this source, and no rule in the model reads that silence
as evidence.

- No absence is asserted. A CT source produces no observation, so it opens no `Gap` and closes no
  span.
- No citation is retired. ADR-0096 rules that only an **enumerable** source's silence contradicts a
  citation, and CT answers no query by name.
- Another source may still admit the name. The bulk-by-name producer, a resolution, and a later
  certificate all reach the same `Name`.

This is the only ground a comment at either skip site may state. *A later poll will pick it up* is
false, and §2 is why.

### 4. The rule does not reach a decode failure that frames the response

`scan.ParseLogEntries` and `scan.ParseDataTile` fail the poll, and they stay that way. Their failure
says the tail cannot tell where one entry ends and the next begins. It cannot skip an entry it
cannot delimit, and a cursor advanced over a body it did not parse asserts progress it did not make.

## Consequences

- **This ADR changes no Go code.** Both arms already behave this way.
- **A decoder defect across a whole shard is silent to the operator.** The tail skips every entry it
  cannot read, the cursor advances over all of them, and the only signal is one `w.log.Printf` line
  per entry. No job event, no `Coverage` line and no metric carries a skip count. The `names
  admitted` event the job emits (`cttail.go:233`) counts admissions and cannot fall below zero, so a
  shard that yields nothing reads the same as a quiet shard. **A skip count on the job's own event
  ships as its own ticket.**
- **The RFC arm's surviving line gains a citation**, recorded in this issue's manifest. **The tiled
  arm keeps no comment.** #1322 left `cttail.go:167` bare, and a second copy of the same reason
  seventy lines below its twin fails the comment policy's first gate.
- **[`ct-source-replacement.md`](../spec/ct-source-replacement.md) gains nothing.** §4 already fixes
  the tail as forward-only and corroborative. The per-entry rule is a code-level consequence of
  that, and the SPEC enumerates no failure sites.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** No domain term moves. A skipped entry mints no
  subject.
- **Nothing enforces the boundary.** A future arm that adds a per-entry decode step may fail the
  poll on it by copying the lines around it, because all of those fail the poll. Review carries the
  rule.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Fail the poll on any undecodable entry** | One unreadable leaf discards every name in the same response, and the retry re-reads the same leaf and fails the same way. Past the attempt budget the log dead-letters and stops being followed. A shard that carries one exotic certificate takes itself offline permanently |
| **Hold the cursor at the first undecodable entry and retry it next poll** | Turns a one-entry gap into a stalled shard. The tail never reads past the entry, so it loses every later certificate in that log rather than one. It also contradicts `cttail.go:50`: the tail reads forward deltas and never backfills, so no mechanism would ever clear the block |
| **Record the skipped index and backfill it on a later poll** | Needs durable per-entry state beside the cursor, and a backfill path the tail does not have. It buys back a leaf whose bytes the decoder already refused once. The decoder is deterministic, so the second read fails the same way unless the product ships a new decoder first |
| **Emit a `warn` job event for each skipped entry** | The drift event at `cttail.go:235` already fixes that a count is safe and a name is not (#780). A per-entry line carries a per-entry name in all but spelling, on a shard the operator does not own, and one poll may skip thousands. A count on the existing event is the shape that fits, and it is named above as its own ticket |
| **Rest the rule on [ADR-0141](./0141-a-periodic-sweep-loop-logs-and-continues-because-the-next-tick-retries-and-the-legibility-rule-does-not-reach-it.md)** | ADR-0141's ground is that the next tick retries the same work, and §2 bounds the rule to exactly that. The cursor advances past a skipped entry, so no tick retries it. Borrowing the ground licenses an unbounded loss under a rule written for a loss bounded by one interval |
| **State the rule in [ADR-0106](./0106-the-ct-poll-is-a-scan-that-schedules-and-a-ct-admission-is-a-name-citing-its-batch.md)** | ADR-0106 rules the crt.sh producer and the response boundary, and its measurement is a log that returns a spurious 404. This rule is about a well-formed 200 whose contents are partly unreadable, which is the opposite failure. It also binds two tail arms ADR-0106 predates |
