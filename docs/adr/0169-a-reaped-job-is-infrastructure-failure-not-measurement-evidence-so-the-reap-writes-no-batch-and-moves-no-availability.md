# ADR-0169: a reaped job is infrastructure failure, not measurement evidence, so the reap writes no `Batch` and moves no `Availability`

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1391 ADR gaps: db/queries (measurement, dispatch)](https://github.com/winniel123/verge-asm/issues/1391), gap 1
- **Sweep PR that compressed the comment:** [#1392](https://github.com/winniel123/verge-asm/pull/1392). The statement survives as one uncited line at [`db/queries/measurement.sql:84`](../../db/queries/measurement.sql) under [`comment-policy.md`](../spec/comment-policy.md) §4.7 route 3, rather than being deleted
- **Rests on:** [ADR-0108](./0108-a-batch-whose-instrument-could-not-reach-its-position-covers-nothing-and-the-failure-is-the-vantages.md), whose limb 4 makes a terminal `resolution-walk` batch outcome the sole producer of `Availability`, and whose limb 1 refuses a signal that is not a transport failure
- **Rests on:** [ADR-0005](./0005-scan-execution-model.md), which splits evidential coverage from operational attempt and rules that a port whose worker died before recording a result is not a measurement
- **Not bound by:** [ADR-0164](./0164-an-operator-ends-a-dispatch-by-recording-a-disposition-once-and-stop-keeps-the-running-jobs-while-terminate-rolls-their-staged-work-back.md) and [ADR-0165](./0165-a-recorded-dispatch-disposition-overrides-the-live-status-derivation-and-the-run-pages-status-word-is-one-token-that-styles-and-labels-the-badge.md), which rule how an **operator** ends a `Dispatch`. The reap is nobody's act
- **Not bound by:** [ADR-0141](./0141-a-periodic-sweep-loop-logs-and-continues-because-the-next-tick-retries-and-the-legibility-rule-does-not-reach-it.md), which rules what the reaper's **loop** does with a failed pass. This ADR rules what a **successful** pass is allowed to write

## Context

`ReapStaleRunningJobs` reclaims a `running` job whose worker died. It is the only exit from `running`
that no worker drives ([`internal/queue/reaper.go:45`](../../internal/queue/reaper.go)), and without
it a dead worker strands its job and blocks its `Dispatch` forever
([#853](https://github.com/winniel123/verge-asm/issues/853)).

The rule that a reap is not evidence was written down once, in the query's own comment. #1392
compressed it to one uncited line, and the gap is recorded because no document states it. **The
issue's location is stale:** the block sat at `db/queries/measurement.sql:132-134` on `860fa97` and
now reads as one line at `:84`, above a query occupying `:83-90`.

### What the reap actually writes

The whole write is four columns on one table.

```sql
UPDATE queue_job
SET state      = CASE WHEN attempt >= max_attempts THEN 'dead' ELSE 'ready' END,
    attempt    = attempt + 1,
    run_after  = now(),
    claimed_at = NULL
WHERE state = 'running' AND claimed_at < @cutoff::timestamptz;
```

That is `db/queries/measurement.sql:85-90`. **It can set `state = 'dead'`** — line `:86`, when the
job's `attempt` has already reached `max_attempts`. It touches no other table, and it never writes
`batch_id`, which `db/migrations/18804_measurement_queue.sql:24` declares nullable.

`internal/queue/reaper.go:24-26` narrows the store to that one method, so the reaper cannot reach
another query even by mistake, and `internal/queue/reaper.go:50` is its only call.

### Where `Availability` moves

`applyAvailability` (`internal/queue/availability.go:37-46`) is the only caller of
`MarkVantageAvailable` and `MarkVantageUnavailable` (`db/queries/vantages.sql:74`, `:79`), at
`internal/queue/availability.go:40` and `:42`. It has **exactly two call sites**:

| Site | Transaction | Batch written |
| --- | --- | --- |
| `internal/queue/worker.go:396` | `complete`'s job transaction, opened at `:382` | `InsertBatch` with `outcomeCompleted` at `:383`, `markDone` at `:454` |
| `internal/queue/worker.go:479` | `deadLetter`'s job transaction, opened at `:466` | `InsertBatch` with `outcomeDeadLettered` at `:467`, `markDead` at `:486` |

Both sit inside the job's terminal transaction, downstream of the `InsertBatch` that carries the
outcome, and both pass that same outcome as the argument.
`availabilityAfterOutcome` (`:22-35`) then answers `availabilityUnchanged` for anything that is not a
`resolution-walk` kind (`:24`) and for any outcome that is neither `completed` nor `dead-lettered`
(`:27-34`). **The reaper produces neither a kind nor an outcome, because it produces no `Batch`.**

### Why ADR-0108 does not already state this

ADR-0108 limb 4 rules that *"a terminal `dns` (resolution-walk) batch outcome derives the vantage's
`Availability`"*, and limb 1 rules that the signal is a transport failure and *"never … a null
count"*. `CONTEXT.md:1199` restates limb 4 and adds *"in the same transaction that writes the
batch"*.

Every one of those sentences is a rule about a path that **produces a batch**. The reaper produces
none, so limb 4 never reaches it, and neither does the `CONTEXT.md` sentence. The rule below is an
**entailment** of limb 4's exclusivity — if only a terminal batch outcome moves `Availability`, then
a path with no batch moves nothing — and not a statement in it.

[`comment-policy.md`](../spec/comment-policy.md) §8.3 is why an entailment is recorded rather than
dismissed: *"A source suppresses only where it states the rule."* A live, on-topic source can fail to
suppress a gap it never rules, and §8.3's shape 3 is that same failure read from inside the ADR
corpus. ADR-0108 is live and on topic and states the positive limb alone.

### The nearest neighbour rules coverage, not `Availability`

`docs/adr/0005-scan-execution-model.md:134-135` is the closest sentence on disk: *"A port attempted
and timed out is completed (a timeout* is *a measurement). A port whose worker died before recording
a result is not."* That rules what a `Batch`'s **recorded scope** may claim — the extent over which
its silence is evidence. It is about coverage. It says nothing about the `Vantage`'s `Availability`,
which is a different derived property with a different writer and a different consumer, and ADR-0005
predates `Availability` having a producer at all. Its next paragraph (`:144-145`) supplies the frame
this ADR uses: **evidential coverage and operational attempt are two different records.**

### The reaper's other two neighbours

ADR-0141 bounds ADR-0108 limb 6 at the reaper and rules that a **failed** sweep pass logs and
continues, because the next tick retries. `internal/queue/reaper.go:68` already carries that
citation. This ADR rules the complementary question — what a **succeeding** pass may write.

ADR-0164 and ADR-0165 rule an operator ending a `Dispatch`: a recorded human decision, written once
onto `dispatch.status` and rendered verbatim as the run's terminal word. A reap has no author. It is
the queue noticing that a process died.

## Decision

> **A job the stale-`running` reaper reclaims is infrastructure failure, not measurement evidence.
> The reap writes no `Batch` and moves no `Availability`, including when it reclaims the job to
> `dead`. `Availability` moves only on a terminal `resolution-walk` batch outcome, which the reaper
> never produces. A reaped `dns` job must never read as a resolver outage.**

Five limbs.

### 1. The reap's write is bounded to `queue_job`, and the bound is the rule

The reaper may reset a job's state, spend an attempt, clear its lease and re-time it. It may not
insert a `batch` row, insert an `observation` row, or move a `vantage`'s availability flag. Its store
interface (`internal/queue/reaper.go:24-26`) is one method wide, and that narrowness is a commitment,
not an accident.

### 2. A reaped `dead` job is terminal without a `Batch`, and that is not a contradiction

`state = 'dead'` on `queue_job` means *this job will not run again*. It does not mean *this job
dead-lettered*. The two are told apart in the row: `MarkJobDead` (`db/queries/measurement.sql:117`)
sets `batch_id` alongside the state, and the reap (`:85-90`) leaves it `NULL`. **A `dead` job with a
`NULL` `batch_id` is a job whose worker died, and it has produced no evidence of any kind.**

Any future code that folds `dead` into an availability decision must therefore read `batch_id`
first — and the honest form of that read is to route through `applyAvailability`'s outcome argument,
which no reaped job can supply.

### 3. What the reap does move is the operational record, and that is legible and correct

A reaped `dead` job counts in `db/queries/dispatch.sql:12`, `:32` and `:52`'s
`FILTER (WHERE j.state = 'dead')` rollups, colours a run-log line `error`
(`cmd/web/scans.go:727-728`), adds to the *"N dead-lettered"* stage detail (`:707-708`), and marks
its vantage `degraded` on that run page with *"missed N of M checks"* (`:756-757`, `:785-793`).

**All of that is permitted, and none of it is a claim about the estate.** *This run missed N checks
at this vantage* is true and scoped to the `Dispatch`. *This vantage is `unavailable`* is a claim
about the resolver that opens a `Gap` on the `Reach` of the vantage's class and makes `Exposure`
absent (ADR-0108 limb 5). The first is ADR-0005's operational attempt, the second its evidential
coverage, and the reap writes only the first. The run page's `dead-lettered` wording is a display
defect on the first record, not a move of the second; Consequences records it.

### 4. The reaper is not an operator act, so ADR-0164 does not extend to it

ADR-0164 rules that a disposition is *"a statement about the operator's first decision"*, written
once over `fanned-out`. The reaper writes no `dispatch.status`, so a reaped run stays `fanned-out`
and ADR-0165's live derivation still governs its word. The boundary holds in both directions: an
operator's `terminated` cancels jobs and rolls staged work back on purpose; the reaper reclaims a job
whose work was already lost when its process died. A shared *end this job* helper would erase exactly
the distinction each ADR exists to hold.

### 5. The attempt increment is the only coupling, and it moves nothing by itself

The reap spends an attempt (`db/queries/measurement.sql:87`), and `exhaustedRetries`
(`internal/queue/pure.go:36-38`) reads `attempt >= max_attempts` at `internal/queue/worker.go:303`.
So repeated worker crashes shorten the retry budget a later genuine probe failure spends before it
dead-letters.

**This is inside the rule, not an exception to it.** The availability move still requires a real
probe, a real transport failure and a real dead-lettered `Batch`. The reap changes only how many
attempts remain, which is a scheduling fact. A reap that did not spend an attempt would let a
crash-looping worker retry one job forever, so the bounded budget is the safer failure.

## Consequences

- **The concrete failure this forbids is named.** A worker crashing mid-`dns`-job, reaped to `dead`,
  marking its vantage `unavailable` — so an operator opens `Coverage`, reads a `Gap` on `Reach` and
  an absent `Exposure`, and investigates a resolver outage that never happened. The evidence is a
  process that died on our own host. ADR-0108 exists to stop *we could not look* reading as *there is
  nothing there*; this is its mirror image, *our worker died* reading as *the resolver is down*, and
  it is equally a false report.
- **No production behaviour changes.** The tree already obeys this. `applyAvailability`'s two call
  sites (`internal/queue/worker.go:396`, `:479`) both carry a batch outcome, and
  `internal/queue/reaper.go` reaches one query.
- **The surviving line at `db/queries/measurement.sql:84` gains this ADR's citation** and stops being
  the only statement of the rule.
- **A reaped `dead` job is still labelled `dead-lettered` on the run page**
  (`cmd/web/scans.go:707-708`). The count is right and the word is wrong. The fix is a
  `batch_id IS NULL` read in the dispatch job projection, carried as follow-up; this ADR records the
  defect rather than smuggling a display change into a document.
- **ADR-0108 is amended nowhere.** Limb 4 is written as a rule about batch outcomes and licenses
  nothing for a path without one, so no sentence there is superseded and
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s
  obligation does not fall due. **`CONTEXT.md` gains nothing** for the same reason, and *reaper* is
  not a domain term (ADR-0141 reached that conclusion already).
- **A second availability writer is now admitted only by argument.** ADR-0108 already records the
  host-key TOFU path as a future second writer of the same scalar. Anything else — including anything
  reading `queue_job.state` — has to overturn this ADR first.
- **Turning the reaper off creates no availability hazard.** `VERGE_STALE_JOB_TIMEOUT <= 0` disarms
  the sweep (`internal/queue/reaper.go:17-22`), which strands jobs and disarms the `hot` lag gate
  (ADR-0137). It moves no vantage either way, because there was never a move to lose.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Mark the vantage `unavailable` on a reap**, treating a dead worker as evidence the position cannot observe | Attributes our own process failure to the vantage. The worker runs on our host and reaches every vantage, so a crash is uncorrelated with any one position and this marks whichever vantage the crashed job happened to name. It is ADR-0108 limb 1's rejected shape in a new place: a signal inferred from an absence of results rather than proved at the socket. The cost is a `Gap` on `Reach` and an absent `Exposure` for a resolver that answered fine |
| **Write a `Batch` for the reaped job with outcome `dead-lettered`** and let the existing path move availability | Manufactures evidence. A `Batch` records what the batch **completed** (ADR-0005:129-135), and a job whose worker died completed nothing. It also makes the reap a writer of the drift-engine record from outside a job transaction, and a dead-lettered `dns` batch marks the vantage `unavailable` under ADR-0108 limb 4 — so it is the first row's cost by a longer route |
| **Add a third `Batch` outcome, `reaped` or `abandoned`**, with an availability rule of `unchanged` | Buys the correct behaviour by minting an object with no reader, which ADR-0108 rejected once already for `failed` / `vantage-unavailable`: *"a third terminal outcome would split the failure population without a reader for the split"*. The record for a died-mid-flight job already exists — `queue_job.state` with a `NULL` `batch_id` — and costs no row in the evidential store |
| **Forbid the reap from ever setting `state = 'dead'`**, so a reaped job is always `ready` | Removes the state whose ambiguity §2 resolves and buys an unbounded one: a job that deterministically crashes its worker requeues forever, spinning the queue against a poison payload with no terminal state and no `Dispatch` ever completing. The `attempt >= max_attempts` cap at `db/queries/measurement.sql:86` is what bounds that |
| **Rule this inside ADR-0108 as an amendment to limb 4** | Under ADR-0058's split an amendment carries a claim about the world that has changed. Nothing changed: limb 4 was always exclusive and the reaper was always outside it. This is a rule about a subject ADR-0108 never named, so it takes its own record and cites limb 4 as its ground |
| **Leave it as the uncited line in the SQL** | The statement then lives in the file least likely to be read by someone changing `internal/queue`, and dies at the next sweep that judges it recoverable. It also binds `worker.go`, `availability.go` and `vantages.sql`, which is `comment-policy.md` §8.2's gate B and is what makes it an ADR rather than a comment |
| **Have the reaper move availability only for `dns` jobs**, mirroring ADR-0108 limb 4's kind scoping | Copies limb 4's scoping without limb 4's premise. The kind gate exists because a **port probe** says nothing about resolver health; it does not license a **`dns` job that never ran** saying something about it. A job the resolver was never asked is not weaker evidence about the resolver — it is no evidence |
