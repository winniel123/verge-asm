# ADR-0165: A recorded Dispatch disposition overrides the live status derivation, and the run page's status word is one token that styles and labels the badge

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1353 ADR gaps: cmd/web/scans.go](https://github.com/winniel123/verge-asm/issues/1353)
- **PR that deleted the comments:** [#1352](https://github.com/winniel123/verge-asm/pull/1352)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0164](./0164-an-operator-ends-a-dispatch-by-recording-a-disposition-once-and-stop-keeps-the-running-jobs-while-terminate-rolls-their-staged-work-back.md), which rules the write side. This ADR rules what the run page does with what that write recorded

## Context

The run page at `/runs/{dispatch-id}` shows one batch-status word.
[`cmd/web/scans.go`](../../cmd/web/scans.go) computes it in `runStatusLabel`, and
[`design-system/templates/rundetail.tmpl`](../../design-system/templates/rundetail.tmpl) renders it.
The comment blocks that stated the rule cited `DF-F4b`, a token that resolves nowhere on disk. #1352
deleted them. [`comment-policy.md`](../spec/comment-policy.md) §8.3 shape 2 records the gap.

[`scans-monitor-bounding.md`](../spec/scans-monitor-bounding.md) is the live spec for the `/scans`
monitor and names the run page as the drill-down. It rules the card, the history window, the handler
split and the SQL. It never mentions a disposition, a stop, or a terminate, so it suppresses
nothing here.

**The template puts one value in two places.** `rundetail.tmpl` line 101 reads:

```html
<span class="rd-batch {{.Status}}"><span class="dot"></span>{{.Status}}…</span>
```

The same string is the CSS class and the visible label.

**The live derivation is wrong about a stopped run, and that is why the precedence exists.**
`toDispatchView` sets `Active: inFlight > 0`, where `inFlight` is `ready + running`. A stop cancels
the pending jobs and leaves the running ones alive (ADR-0164 limb 2). So a stopped `Dispatch` still
has running jobs, `Active` is still true, and a derivation from live counts alone would report
`running` about a run the operator already ended.

## Decision

> **The run page's batch status is one token that serves as both the `rd-batch` CSS class and the
> visible label. A `Dispatch` that carries a recorded operator disposition renders that disposition
> verbatim as its terminal status. Only where no disposition is recorded does the page derive the
> status live: in flight is `running`, any dead-lettered job is `failed`, and everything else is
> `complete`.**

Four limbs.

### 1. One token does two jobs

The status word is styled by matching it to a `.rd-batch.<word>` rule and is shown to the operator
unchanged. A new status word therefore needs a matching CSS rule in the same change, or it renders
unstyled.

`rundetail.tmpl` carries six such rules today: `scheduled`, `running`, `complete`, `failed`,
`stopped` and `terminated`. `runStatusLabel` emits five of them. `scheduled` has a treatment and no
producer.

### 2. A recorded disposition wins over the derivation

`dispatchOutcome` maps `stopped` and `terminated` to themselves. `runStatusLabel` returns that
outcome before it looks at anything else.

The ground is §Context's measurement. The derivation reads live job counts, and after a stop those
counts still describe a run with work in it. The recorded disposition is a fact about what an
operator did, and no count can carry it. Where the two disagree, the operator's act is the truth of
the run.

**A terminate agrees with the derivation and still takes the same path.** Its jobs are all
`cancelled`, so `Active` is false and `dead` is zero, and the derivation would answer `complete`.
That is worse than wrong, because a killed run would read as a clean finish. The precedence covers
both dispositions for one reason.

### 3. `fanned-out` is not an outcome, so a natural run is unchanged

`dispatchOutcome` returns the empty string for `fanned-out` and for an absent status.
`runStatusLabel` then falls through to the live derivation. Every `Dispatch` that no operator ended
behaves exactly as it did before the acts existed.

### 4. The status word is also the auto-refresh toggle

`runRefresh` returns `5` only when the status is the literal `running`, and `0` otherwise. The page
head reads that as an on switch for its `<meta http-equiv=refresh>`.

So a terminal word turns the auto-refresh off. **The stated cost is a stopped run.** Its running
jobs are still committing batches, and the page no longer refreshes itself, so the operator reloads
to watch them drain. The word is right and the page is static. This is accepted, because the
alternative is a page that keeps refreshing forever on a `Dispatch` that has reached its terminal
badge.

Any later status word inherits this. A word added to `runStatusLabel` is a refresh decision as well
as a badge decision.

## Consequences

- **`runStatusLabel` keeps its one-token comment and gains this ADR's citation.** That comment
  states the fact limb 1 rules and a reader cannot recover from the function body.
- **`runRefresh` keeps its comment and gains this ADR's citation.** Its `5` reads as a toggle rather
  than as a cadence, which is limb 4.
- **`buildRunView` and `dispatchOutcome` gain nothing.** The precedence is visible in
  `runStatusLabel`'s first `switch`, so a comment at either site restates the code.
- **No production behaviour changes.** The page already has this shape. The change is that the shape
  now has a record.
- **The progress meter and the badge disagree on an ended run, and the badge is right.** A cancelled
  job counts in `live` and never in `completed`, so a stopped or terminated `Dispatch` holds its
  percentage below 100 for good. The recorded disposition is the statement that the run is over. The
  meter is a statement about jobs, and it is accurate about jobs.
- **A new status word costs three edits.** The word in `runStatusLabel`, an `.rd-batch` rule in
  `rundetail.tmpl`, and a decision about `runRefresh`.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** Badge rendering is a console concern and not a
  domain term.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Derive the status live and ignore the recorded disposition** | It reports `running` for a stopped run whose jobs are still finishing, and `complete` for a terminated run whose jobs were killed. The second is the serious one: a killed run would read as a clean finish, which is the failure mode [ADR-0108](./0108-a-batch-whose-instrument-could-not-reach-its-position-covers-nothing-and-the-failure-is-the-vantages.md) exists to prevent |
| **Show both — the disposition and the derived state, side by side** | It asks the operator to reconcile *stopped* against *running* at the moment they want one answer. The disposition already implies the derived state, and the job rollup below the badge carries the detail for anyone who wants it |
| **Carry a separate `Label` field beside the `Status` class** | Two fields that must agree, with no mechanism that makes them agree. The template would gain a second hole and the handler a second assignment, to express one word |
| **Map `stopped` and `terminated` onto the existing `failed` treatment** | An operator ending a run is not a failure. `failed` means a dead-lettered job, and reusing it would make the two indistinguishable in the badge and in any later filter over the word |
| **Keep the auto-refresh alive on a stopped run until its jobs drain** | It makes `runRefresh` read the job counts as well as the word, so the refresh rule and the badge rule stop agreeing. Limb 4 keeps one input and states the reload cost |
| **Fold this rule into [ADR-0164](./0164-an-operator-ends-a-dispatch-by-recording-a-disposition-once-and-stop-keeps-the-running-jobs-while-terminate-rolls-their-staged-work-back.md)** | Different scope. ADR-0164 binds the worker, the queue schema and the migration. This binds one handler and one template. A session changing the badge should not have to read the write side to apply it |
| **State the rule in [`scans-monitor-bounding.md`](../spec/scans-monitor-bounding.md)** | That spec is a plan-only handoff for bounding the monitor card, and it is complete against its own map. Filing a rendering rule for a different page inside it would put a live rule in a delivered plan |
