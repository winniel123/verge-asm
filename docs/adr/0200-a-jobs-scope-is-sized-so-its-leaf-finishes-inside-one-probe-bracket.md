# ADR-0200: a job's scope is sized so its leaf finishes inside one probe bracket

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1319 ADR gaps: internal/scan (2/3)](https://github.com/winniel123/verge-asm/issues/1319), gap 1
- **PR that deleted the comment:** [#1318](https://github.com/winniel123/verge-asm/pull/1318)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Bounded by:** [ADR-0005](./0005-scan-execution-model.md), which rules **where a partition boundary may fall** — along any dimension the source keeps enumerability over, so its silence stays honest. That rule is about the honesty of a scope record. This rule is about the execution of a job whose boundary is already legal. Neither narrows the other, and this ADR does not amend ADR-0005
- **Sibling of, and not ruled by:** [ADR-0127](./0127-the-address-scope-range-cap-has-no-ceiling-a-large-scope-is-priced-not-gated.md), which removes every ceiling over a **declared** address scope and rules that a scope which cannot finish inside its cadence is reported and never refused. This ADR bounds **one job**, never a declared scope. A larger population makes more jobs and never a larger one, so ADR-0127's no-ceiling ruling is untouched
- **Rests on:** [ADR-0129](./0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md) §6, which makes the no-SNI handshake a new measurement with a `Scan` of its own. It created the streaming builder this rule first bound

## Context

[`internal/scan/edgefanout.go:18`](../../internal/scan/edgefanout.go) carried this text until #1318 deleted it:

```go
// EdgeFanoutAddressesPerJob bounds how many candidate edges one job measures. The leaf
// handshakes its scope serially under a per-handshake timeout, and the worker brackets
// one probe with DefaultProbeTimeout (5 minutes); a scope large enough to outlast that
// bound would be killed mid-measurement and retried whole, measuring the early edges
// again and the late ones never. Chunking keeps one job's worst case well inside the
// bound, and the chunks are independent — one slow edge delays its own job alone.
```

The sweep kept one compressed line at the same site, uncited under §4.7:

```go
// A scope outlasting the worker's 5-minute probe bracket is killed mid-run and retried whole.
```

**No ADR and no SPEC states the rule.** ADR-0005 rules the partition dimension. ADR-0127 rules the
declared scope against the cadence. Neither bounds the work inside one job.

### The bracket, measured

| Element | Site | Value |
| --- | --- | --- |
| The probe bracket | [`internal/queue/worker.go:208`](../../internal/queue/worker.go) | `DefaultProbeTimeout = 5 * time.Minute`, overridable by `VERGE_PROBE_TIMEOUT` at [`cmd/worker/main.go:90`](../../cmd/worker/main.go) |
| What the bracket wraps | `Worker.probe` | The probe alone. The terminal transaction runs under the parent context |
| How the bracket ends a probe | [`internal/queue/worker.go:41`](../../internal/queue/worker.go) | `exec.CommandContext(ctx, p.Path)`. The deadline kills the prober process |
| One candidate's own bound | [`internal/measure/edgefanout/run.go:35`](../../internal/measure/edgefanout/run.go) | `NetHandshaker{Timeout: 3 * time.Second}`, and [`internal/measure/connectoutcome/tls.go:133`](../../internal/measure/connectoutcome/tls.go) wraps the dial **and** the handshake in it |
| How the leaf walks its scope | `RunWithHandshaker` | Serially, one candidate at a time |
| The chunk | [`internal/scan/edgefanout.go`](../../internal/scan/edgefanout.go) | `EdgeFanoutAddressesPerJob = 50` |
| The worst case | arithmetic | 50 × 3 s = 150 s, which is half of the 5-minute bracket |
| The retry budget | [`internal/queue/edgefanout.go:60`](../../internal/queue/edgefanout.go) | `MaxAttempts: 5` |

### A job that outlasts the bracket records nothing at all, five times over

The deleted comment says the retry measures the early edges again and the late ones never. That is
true of the **network effort**. The **record** is worse, and it is the part a reader cannot recover
from the constant.

1. **The leaf buffers its whole output.** `RunWithHandshaker` fills `out []wire.Observation` and
   calls `writeNDJSON` only after the loop ends. A killed prober has therefore written **zero bytes**
   of NDJSON, however many handshakes finished.
2. **The worker discards a failed probe's observations.** `ExecProber.Probe` returns
   `wire.ProbeResult{Transcript: t}` on a non-nil run error. The `Observations` field stays empty.
   The partial work survives in the raw transcript alone, which no derivation reads.
3. **The retry is a whole new job.** `Worker.retry` re-enqueues the **same** spec with
   `Attempt + 1`. It resumes nothing.
4. **The fifth failure dead-letters with an empty scope.** `Worker.deadLetter` writes
   `RecordedScope` as `{"names":[]}`, which is ADR-0005's dead-letter rule honoured in code.

So a job whose scope cannot finish inside one bracket is not slow. It is **unmeasurable**. It burns
five attempts, records five failures, and lands one dead-lettered `Batch` that licenses no absence.

**The product consequence is silence, not a wrong answer.** `CONTEXT.md` rules that an `edge-fanout`
candidate nobody measured is **held** — neither reached nor vetoed. The 50 addresses in a
dead-lettered chunk stay held on every tick. The custody extension never decides them, and the
operator sees a dead-letter event rather than a stalled derivation.

### The bound is a property of the leaf, never of one constant

Three leaves walk a job's scope serially under a per-item timeout: `edge-fanout`
([`run.go`](../../internal/measure/edgefanout/run.go)), `connect-outcome` and `http-exchange`. Two
numbers set the worst case for each — the item count in the job, and the leaf's own per-item bound.
Both move. `EdgeFanoutAddressesPerJob` is 50 today. `VERGE_PROBE_TIMEOUT` is an operator dial. A rule
written as *"50 fits in 5 minutes"* is false on the day either number moves, so this ADR states the
behaviour and never the arithmetic.

## Decision

> **A builder sizes one job's scope so that the leaf which reads it finishes inside one probe
> bracket, at the leaf's own worst case. The bound is on the job and never on the declared scope: a
> larger population produces more jobs and never a larger one. A job that cannot finish inside the
> bracket is unmeasurable rather than slow, so its size is a correctness property of the builder and
> not a tuning choice.**

### 1. The unit bounded is one job, and the ground is the retry

A `Batch` records what completed. A probe killed at the bracket completes nothing, because the leaf
writes its output at the end and the worker drops a failed probe's observations. The retry starts the
whole scope again under the same bracket, so a job that fails once for size fails every time. The
budget then runs out and the chunk dead-letters. The bound exists to keep a job on the finishing side
of that line, and for no other reason.

### 2. The bound is stated in behaviour, never as a constant

The rule is *"the leaf finishes inside one bracket"*. It is not *"50 addresses"* and it is not
*"under 5 minutes"*. Both numbers move on their own schedule. One is a package constant. The other is
`VERGE_PROBE_TIMEOUT`. A builder honours this rule by carrying a worst case it can state — the item
count multiplied by the leaf's per-item bound — held clear of the bracket with margin. `edge-fanout`
holds 150 s against 300 s.

### 3. It binds every builder whose leaf walks its scope serially

Not `edge-fanout` alone. Any builder that puts N items in one job, where the leaf measures them one
after another under a per-item timeout, carries this bound. The port tiers satisfy it structurally
rather than by a constant. ADR-0005 already puts **one address per `Batch`**, so their job size is
fixed by the partition rule and by the `verge-core` port set. `http-identity` does not satisfy it,
and §5 records that.

### 4. The declared scope is untouched

A population too large for one job is split into more jobs. It is never refused, never capped, and
never priced differently. ADR-0127 removes every ceiling above the operator's own cap and rules that
an uncompletable scope is reported rather than gated. This rule cannot reintroduce a ceiling, because
it constrains a number the operator does not set and cannot see.

### 5. What this rule does not decide

- **The value of the constant.** 50 is a choice with margin. Any value whose worst case clears the
  bracket satisfies this ADR.
- **The per-item timeout.** `edge-fanout` uses 3 s because `connect-outcome` does. That is the
  leaf's parameter and ADR-0025 governs it.
- **`http-identity`'s job size.** `BuildHTTPIdentityJobs` puts **every reached `Endpoint` at one
  `Vantage`** in one job, with no chunk at all. That is a violation of this rule, and Consequences
  names it.
- **Whether the bracket is the right length.** This ADR takes `DefaultProbeTimeout` as given.

## Consequences

- **This ADR changes no Go code.** `edge-fanout` already complies. The rule it states was true and
  unwritten.
- **`BuildHTTPIdentityJobs` is a violation this ruling exposes, and it ships as its own ticket.**
  [`internal/scan/httpidentity.go`](../../internal/scan/httpidentity.go) groups every admitted
  `(address, port)` at one `Vantage` into a single `HTTPIdentityJob` with no cap.
  `httpexchange.RunWithExchanger` walks its targets serially and buffers its whole output, exactly as
  `edge-fanout` does, and `DefaultParams().TimeoutMillis` is **10 000**. So **31 unresponsive targets
  in one job exceed the 5-minute bracket**, before the per-host pacer's sleeps are counted. By §1 that
  job is unmeasurable. It dead-letters after five attempts, and the whole vantage's HTTP identity goes
  unrecorded. The fix is a chunk, exactly as `edge-fanout` carries one. **The change is not made
  here.**
- **No [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
  withdrawal is owed.** ADR-0058 binds a decision that **supersedes a mechanism**. This ADR supersedes
  nothing. ADR-0005's partitioning sentence — *"partitioning is governed by `completeness`, not by
  size"* — stays true and readable in the present tense. It says a boundary may not be cut where the
  cut destroys enumerability. It never said a job may be any size, and this rule does not make it say
  so.
- **ADR-0005 is not amended and gains no clause.** The two rules answer different questions, and the
  relation row above states which is which.
- **`CONTEXT.md` gains nothing.** Job size is an execution property. `Batch`, the domain term, does
  not move. A chunk is still one `Batch`.
- **Nothing enforces this.** No test asserts that a builder's worst case clears the bracket, and no
  check can compute one, because the per-item bound sits in the leaf and the item count sits in the
  builder. Review carries the rule. A reviewer's question is *what is this job's worst case, and
  against which bracket*.
- **The dial makes the rule an operator's to break.** A `VERGE_PROBE_TIMEOUT` below the leaf's worst
  case turns a compliant builder into a non-compliant one with no code change. That is a live hazard,
  and this ADR only names it.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Amend ADR-0005's partitioning row** | ADR-0005 rules which dimension a boundary may cut along, and its ground is the honesty of a scope record. This rule's ground is the retry: a killed probe records nothing, so an oversized job never completes. Filing an execution bound inside the enumerability rule would teach a later reader that a size argument can move a partition boundary, which is the one thing ADR-0005 refuses |
| **State the rule as a number — "a job carries at most 50 items"** | The bracket is an operator dial and the per-item timeout is a leaf parameter. A number stated here is false on the day either moves, and it is already wrong for `http-exchange` on the day it is written, because that leaf's per-item bound is not `edge-fanout`'s |
| **Cap the declared scope so that a job never needs a chunk** | Reintroduces the ceiling ADR-0127 removed, and reintroduces it in the safety path — [#27](https://github.com/winniel123/verge-asm/issues/27)'s refused threshold shape. It also solves nothing, because the item count in one job is not a function of the declared scope size once the fan-out streams |
| **Let an oversized job dead-letter, and read the dead-letter as the report** | The record is already honest — an empty scope licenses no absence — and it is still wrong for the product. `edge-fanout` candidates in a dead-lettered chunk stay **held** forever, so the custody extension stalls with no `Gap` and no message anywhere. The failure is invisible in every derived value |
| **Resume a partial batch instead of retrying it whole** | ADR-0005 rules that a retry is a new `Batch`, and its ground stands: a batch resumed later carries a timestamp that misstates when half its observations were made, and a retry on another worker could carry a different vantage. Resumption also needs the leaf to stream its output and the worker to keep a partial commit, which reopens ADR-0001's commit-together invariant |
| **Raise `DefaultProbeTimeout` until the largest job fits** | Trades a bounded failure for an unbounded one. The drain loop is single-threaded ([#853](https://github.com/winniel123/verge-asm/issues/853)), so a longer bracket lets one wedged job block every later job for longer, and the stale-job reaper's threshold must then move with it — [`internal/queue/reaper_test.go:32`](../../internal/queue/reaper_test.go) pins that ordering |
| **A `go vet` or `commentlint` check on the job size** | Not decidable at any single site. The item count is in the builder, the per-item bound is in the leaf, and the bracket is in the environment. No one file holds two of the three |
