# ADR-0206: A fan-out over an empty population is a legible zero-job dispatch, and it fabricates no `Batch`

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1320 ADR gaps: internal/queue (#1200, sweep 6/7)](https://github.com/winniel123/verge-asm/issues/1320), gap 1
- **PR that deleted the comment:** [#1324](https://github.com/winniel123/verge-asm/pull/1324)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0005](./0005-scan-execution-model.md), which fixes one `Dispatch` per `(Scan, scheduled tick)` and rules that a `Batch` is one queue job's outcome. This ADR takes both and rules what the tick records when the fan-out builds no job
- **Generalises:** [ADR-0129](./0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md)'s #954 amendment, which states the shape for the `edge-fanout` `Scan` alone — *"no extension means an empty scope and no probe, a legible empty-scope state"*. It rules one population and says nothing about the other eight
- **Generalises:** [ADR-0106](./0106-the-ct-poll-is-a-scan-that-schedules-and-a-ct-admission-is-a-name-citing-its-batch.md), which states it for the `ct` `Scan` and names `zone`'s parallel — *"the same legible zero-job state `zone` has when no file is supplied"*. It rules a source toggle, not a fan-out in general
- **Sibling of, and not ruled by:** [ADR-0108](./0108-a-batch-whose-instrument-could-not-reach-its-position-covers-nothing-and-the-failure-is-the-vantages.md). That ADR rules the dead-lettered `Batch` whose recorded scope is empty. That is the failure path, where a `Batch` exists. This ADR rules the success path, where no job and therefore no `Batch` exists. §4 states the difference
- **Sibling of, and not ruled by:** [ADR-0163](./0163-an-absent-certificate-material-row-is-a-fan-out-of-zero-and-is-reached-and-only-an-absent-measurement-row-is-pending.md). That ADR rules a **fan-out of zero** over one edge's SAN set, inside the `Custody` derivation. This ADR rules a **fan-out of zero jobs** by the dispatcher. The two share a phrase and no subject
- **Depends on:** [ADR-0208](./0208-the-queue-reads-a-services-subject-and-never-re-parses-its-rendering-so-a-rendered-key-is-an-identity-token-alone.md). An empty population is a legible success only where the population is honestly empty. While the queue re-parses a rendered subject key, a key-form change empties every population at once and this ADR would rule that outage a healthy tick. That ADR removes the parse and so removes the composition

## Context

`internal/queue/tlsacceptance.go:24` carried this in Go declaration position, until #1324 deleted it:

```go
// fanOutTLSAcceptance enqueues one tls-acceptance job per Vantage over the Services
// reached from that Vantage. With no reached Service — the shipped state before any
// hot Scan has run — it produces no jobs, a legible empty scope rather than an error.
```

`internal/queue/httpidentity.go:21` carried the same three sentences over the `http-identity`
population, and #1324 deleted them too. Neither block carried a citation. Nothing on disk states
the rule for either `Scan`. That is #1320's gap 1.

### The record's third site does not exist

#1320's body says the cold fan-out *"states it a third time in its own words"*. It does not.
`internal/queue/cold.go` carries three surviving comment lines — an ADR-0044 citation about the
opt-in, an ADR-0055 citation about zone folding, and a cost note on the excluded-range walk. None
of them mentions an empty population, a zero-job dispatch or an error. The rule survived in exactly
the two deleted blocks quoted above.

### Nine fan-outs, one shape, and no ruling above any of them

The dispatcher holds nine per-`Scan` fan-out entry points. Every one of them ends the empty case
with `return 0, nil`.

| `Scan` | Entry point | Population that can be empty | What empties it on a shipped install |
| --- | --- | --- | --- |
| `dns` | `fanOutDNS` (`queue.go:279`) | name `Seed`s plus admitted names | no name `Seed` declared |
| `zone` | `fanOutZone` (`queue.go:305`) | uploaded zone files | no file supplied |
| `hot` | `fanOutHot` (`hot.go:21`) | resolved addresses plus declared scopes | no `Seed` has resolved yet |
| `cold` | `fanOutCold` (`cold.go:15`) | the opted-in cold scope | ships disabled (ADR-0044) |
| `tls-acceptance` | `fanOutTLSAcceptance` (`tlsacceptance.go:15`) | reached `Service`s | no hot `Scan` has reported `reached` |
| `http-identity` | `fanOutHTTPIdentity` (`httpidentity.go:13`) | reached `Service`s | as above |
| `ct` | `fanOutCT` (`crtsh.go:337`) | name `Seed`s, gated on the source toggle | source off, or no name `Seed` |
| `ct-tail` | `fanOutCTTail` (`cttail.go:283`) | the selected tail logs | the source ships off |
| `edge-fanout` | `fanOutEdgeFanout` (`edgefanout.go:23`) | custody-extension candidates | no extension declared (ADR-0129 #954) |

**The shipped state of a fresh install is eight zero-job dispatches per cycle.** Eight of the nine
`Scan`s ship `enabled = TRUE` in their own migrations, and `cold` ships disabled. On an install with
no `Seed` and no zone file, every one of the eight enabled `Scan`s claims its tick and enqueues
nothing. Reading zero as a dispatch failure would put eight error lines on the log of a correctly
installed instance, every tick, until an operator declares a `Seed`. That is the measurement that
makes the rule non-obvious: the empty case is not a rare edge, it is the state the product ships in.

### The tick is recorded before the population is read

Both dispatch paths claim the tick first and build jobs second.

- The atomic path (`queue.go:118`) takes the advisory lock, calls `TryFanOut`, then switches to the
  per-`Scan` fan-out, then commits. A zero-job fan-out reaches `tx.Commit` on the same line a
  thousand-job fan-out does.
- The streamed path (`queue.go:170`) calls `claimDispatch`, which commits the `dispatch` row in its
  own transaction before `fanOutHot`, `fanOutCold` or `fanOutEdgeFanout` runs at all.

`TryFanOut` (`db/queries/measurement.sql:45`) inserts one `dispatch` row with status `fanned-out`,
`ON CONFLICT ON CONSTRAINT dispatch_tick_key DO NOTHING`. The row is therefore present for a
zero-job tick and for a full one, and it is present in both paths.

### The `Batch` cannot carry the empty state, because zero jobs make zero `Batch`es

`db/migrations/18803_measurement_batch.sql` states the relation at its own site: *"A Batch is one
queue job's outcome (v1 spec §4.1): one queue job is one Batch."* Every `InsertBatch` call in
`internal/queue` sits on a job's completion or dead-letter path in `worker.go`, `crtsh.go`,
`cttail.go`, `zone.go` or `edgefanout.go`. No dispatcher code writes a `Batch`. So a fan-out that
enqueues nothing produces nothing for the worker to complete, and there is no `Batch` to look at.

## Decision

> **A `Scan`'s fan-out over an empty population enqueues zero jobs, returns no error, and writes no
> `Batch`. It is a success path and a legible state, not a dispatch failure. The record of the empty
> tick is the `dispatch` row the tick already wrote, and the dispatcher fabricates no second artefact
> to prove that nothing happened.**

### 1. The rule binds every `Scan` that fans out

All nine entry points in the §Context table, and every fan-out added later. The rule is not a
property of the `edge-fanout` population, of the `ct` source toggle or of the reached-`Service`
read. It is a property of the dispatch model: a `Scan` is a Declared recurring intent, and a tick of
that intent over an empty aperture covers nothing. Covering nothing is a measurement outcome the
model already has words for. It is not an instrument fault.

A fan-out still returns an error for the things that really are faults: a failed read of the
population, a failed enqueue, a failed commit. Every entry point does exactly this today —
`return 0, err` on a read failure, and `return 0, nil` only after a clean read that yielded no
candidate. The distinction the rule draws is between **we could not look** and **we looked and there
was nothing**.

### 2. The recorded artefact is the `dispatch` row, and it is named

**The zero-job tick's artefact is its `dispatch` row: `(scan_id, scheduled_time, status =
'fanned-out')`, written by `TryFanOut` at `db/queries/measurement.sql:45` into the table
`db/migrations/18802_measurement_dispatch.sql` defines.**

This is not a new obligation. The row is written before the population is read, in both dispatch
paths, so no fan-out can produce a tick with no artefact. What this ADR rules is that the row **is
the record** and that nothing further is owed.

The row is legible to an operator rather than only to the table. `ListDispatchProgress`,
`ListActiveDispatchProgress` and `ListConcludedDispatchProgress` (`db/queries/dispatch.sql`) all
reach `queue_job` through a `LEFT JOIN`, so a dispatch with no job renders with `total = 0` instead
of vanishing. `ListConcludedDispatchProgress`'s `HAVING count(*) FILTER (WHERE j.state IN ('ready',
'running')) = 0` admits it on the first read. The console's run and scan-history surfaces read
`ListDispatchProgress` from `cmd/web/restore.go:404`, `cmd/web/drift.go:290`,
`cmd/web/settings.go:1034`, `cmd/web/search.go:210` and `cmd/web/reports.go:567`.

The dispatcher also logs the tick with its count — `"%s fanned out %d job(s) at %s"`, on both paths.
The log line is a convenience and not the artefact. ADR-0108 already ruled that a state legible only
in a server log is not legible, and this ADR does not re-open that.

### 3. No `Batch` is manufactured, and this is the load-bearing half

**A zero-job fan-out writes no `Batch`, and a future reader must not add one.**

A `Batch` is a unit of probing work with an outcome, a set of offers and a recorded scope. Writing
an empty one to prove that nothing was probed would put a row in the corpus for work that was never
done. That is the same fabrication this ticket's gap 2 refuses one layer down, where an enumeration
drops a target it cannot name rather than reconstructing one, and it is the fabrication ADR-0025
refuses when it makes an offer a declaration rather than a library default.

The cost is concrete rather than aesthetic. `batch` rows are the corpus every currency, coverage and
retention read runs over. A fabricated `completed` `Batch` with an empty scope on every empty tick
would enter that corpus at the cadence of the `Scan` — the `ct-tail` `Scan` alone ships at a
300-second cadence, so a source left off would manufacture 288 rows a day that assert a completed
measurement nothing performed.

### 4. The dead-lettered `Batch`'s empty scope is a different case, and the two are not merged

ADR-0005 rules that *"a dead-lettered batch records an empty scope and licenses no absence"*, and
ADR-0108 applies it to an unreachable vantage. Read quickly, that looks like this rule. It is not.

| | Dead-lettered `Batch` with an empty scope | Zero-job fan-out |
| --- | --- | --- |
| Path | Failure. The job ran and could not complete | Success. There was no work |
| Does a job exist? | Yes, and it exhausted its attempts | No |
| Does a `Batch` exist? | **Yes.** That is the point — it records *we tried and covered nothing* | **No.** There is nothing for one to be the outcome of |
| What the empty scope asserts | Nothing. Recording the attempted scope would manufacture absences never measured | Not applicable. There is no scope record at all |
| Where it is written | `worker.go:467`, on the dead-letter path | Nowhere |
| Who is at fault | The instrument, or the target | Nobody. The aperture is empty |

The two share only the word *empty*. Merging them would mean either giving the failure path no
`Batch` — which would delete the record ADR-0005 built to hold *we tried* — or giving the success
path one, which is §3's fabrication. The union of the two is therefore refused in both directions.

### 5. What this rule does not reach

- **A `Scan` that is disabled.** `dispatchDue` never lists it, so it claims no tick and writes no
  `dispatch` row. `cold` is in this state on every install that has not opted in (ADR-0044). Absence
  of a dispatch row is the record there, and it is a different sentence from a dispatch row with
  zero jobs.
- **A tick already dispatched.** `TryFanOut` returns no row, the dispatcher logs *already
  dispatched, skipped* and commits. The `dispatch` table's own migration comment rules that the
  first row records the overlap and that no second row is written. This ADR adds nothing there.
- **A hot tick skipped by the cadence-lag gate** (`queue.go:213`, ADR-0137 §4). That tick claims its
  dispatch row deliberately and enqueues nothing, but the ground is an undrained predecessor rather
  than an empty population. It is a third case, already ruled, and this ADR does not restate it.
- **A partial fan-out that fails mid-stream.** `streamEnqueue` returns the count committed so far
  with the error. #847's currency surfaces report the under-covered tick. That is an error path and
  §1 keeps it one.
- **Whether a population *should* be empty.** The rule says an empty population is legible. It does
  not say it is healthy. An operator who declared a `Seed` and still sees zero jobs has a different
  problem, and the coverage surfaces are where it is read.

## Consequences

- **This ADR changes no Go code and no SQL.** All nine fan-outs already implement the rule, the
  `dispatch` row is already written before the population is read, and the console already renders a
  zero-job dispatch through the `LEFT JOIN`. The gap was that nothing said so.
- **ADR-0129's #954 amendment and ADR-0106 are confirmed, not narrowed.** Each stated the shape for
  one `Scan`. Each remains correct at its own site, and each gains a pointer to the general rule so
  a reader does not conclude the shape is that `Scan`'s alone.
- **A future `Scan` inherits the rule without arguing it again.** The tenth fan-out returns
  `0, nil` over an empty population and writes no `Batch`, and a reviewer has a document rather than
  a preference.
- **No test covers the zero-population path in `internal/queue`.** No file under
  `internal/queue/*_test.go` names `fanOutTLSAcceptance` or `fanOutHTTPIdentity`. The rule is
  therefore held by review and not by a gate. Adding a per-fan-out zero-population test is a change
  worth its own ticket, and this ADR does not open it.
- **`CONTEXT.md` gains nothing.** `Scan`, `Dispatch` and `Batch` already carry the terms this rule
  uses and none of them changes meaning.
- **The dead-letter rule is untouched.** ADR-0005's and ADR-0108's empty recorded scope stays exactly
  as ruled. §4 exists so that a reader who arrives from either does not carry the failure-path
  reasoning into the success path.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Return an error from a fan-out over an empty population** | Eight of the nine `Scan`s ship enabled, and on an install with no `Seed` all eight are empty. The dispatcher would log eight errors per tick on a correctly installed instance, which trains an operator to ignore the one line that later matters. It also inverts the meaning of the return: the reader could no longer tell *we could not read the population* from *the population is empty* |
| **Write an empty `completed` `Batch` to prove the tick did nothing** | Manufactures a record of work never done, in the one corpus every currency, coverage and retention read runs over. At `ct-tail`'s 300-second cadence a source left off would produce 288 such rows a day. It is gap 2's fabrication moved up a layer: gap 2 refuses to invent a target, and this would invent a whole measurement |
| **Write a `dead-lettered` `Batch` with an empty scope, reusing ADR-0005's machinery** | `dead-lettered` means *failed after every retry* and it names an instrument or target fault. An empty aperture is neither. It would also make the two cases indistinguishable on every operator surface, which is exactly the split §4 keeps |
| **Add a second `dispatch` status — `empty`, beside `fanned-out`** | The `dispatch` table's `CHECK (status IN ('fanned-out'))` and its own migration comment fix the column at one value on purpose, because the unique tick key makes a second row structurally impossible. A second status would put the job count in two places, and the count is already derivable by the `LEFT JOIN` the three progress queries use |
| **Leave the record on the dispatcher's log line alone** | ADR-0108 already refused this shape: a state legible only to someone reading the server log is indistinguishable, on every operator surface, from a clean empty result. The `dispatch` row is what makes it readable, and the log line rides beside it |
| **State the rule once more on each `Scan`'s own ADR** | Nine sites for one sentence, and the ninth arrives with the tenth `Scan`. ADR-0129 and ADR-0106 each did this honestly for their own population and neither could bind the others, which is the gap this ADR closes |
| **Merge this with the dead-lettered `Batch`'s empty-scope rule into one ADR about emptiness** | The two rules move in opposite directions — one requires a `Batch`, the other forbids one — and they are triggered by opposite conditions. A single document would have to state both and then spend its length keeping them apart, and a reader who took the wrong half would either delete the failure record or fabricate a success one |
