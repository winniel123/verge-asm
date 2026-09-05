# ADR-0141: A periodic sweep loop logs and continues because the next tick retries, and the legibility rule does not reach it

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1291 ADR gaps: internal/measure/tlsacceptance, internal/retention](https://github.com/winniel123/verge-asm/issues/1291)
- **Also found by:** [#1321](https://github.com/winniel123/verge-asm/issues/1321) §5 (`internal/queue/reaper.go`) and [#1272](https://github.com/winniel123/verge-asm/issues/1272) §1 (`internal/release`). One rule, three findings, one record
- **Bounds:** [ADR-0108](./0108-a-batch-whose-instrument-could-not-reach-its-position-covers-nothing-and-the-failure-is-the-vantages.md) limb 6, at limb 6's own site, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
- **Rests on:** [ADR-0084](./0084-a-scan-is-a-cadence-over-an-exchange-and-an-uncovered-facet-has-no-currency-bound.md) (a missed cadence ripens into a `Gap`, so a skipped pass is already told)

## Context

Nine loops in the tree carry one shape. Each ticks on an interval, does one pass of work, and on a
failed pass logs a line and keeps ticking. None returns an error of its own beyond the context's.

| Loop | Site |
| --- | --- |
| `retention.Retirer.Run` | [`internal/retention/retention.go:70`](../../internal/retention/retention.go) |
| `retention.ObservationRetirer.Run` | [`internal/retention/observation.go:134`](../../internal/retention/observation.go) |
| `retention.TranscriptRetirer.Run` | [`internal/retention/transcript.go:67`](../../internal/retention/transcript.go) |
| `queue.Reaper.Run` | [`internal/queue/reaper.go:53`](../../internal/queue/reaper.go) |
| `queue.Dispatcher.Run` | [`internal/queue/queue.go:64`](../../internal/queue/queue.go) |
| `release.Checker.Run` | [`internal/release/release.go:76`](../../internal/release/release.go) |
| `delivery.Runner.Run` | [`internal/delivery/runner.go:105`](../../internal/delivery/runner.go) |
| `report.Dispatcher.Run` | [`internal/report/dispatcher.go:32`](../../internal/report/dispatcher.go) |
| `report.NotifyRunner.Run` | [`internal/report/notify.go:75`](../../internal/report/notify.go) |

Three separate ADR-gap sweeps found the same rule in three different packages and each recorded it
as uncited. #1291 found it in `internal/retention`, stated three times in three wordings. #1321 §5
found it at `queue.Reaper.Run`. #1272 §1 found it at `release.Checker.Run`, whose comment named
`retention.Retirer.Run` as the shape it was copying. So the rule already bound at least three
packages before anyone wrote it down.

**The code gave two different grounds for the same behaviour, and only one of them survives.**

- *The work is off the measurement path.* `internal/retention` stated this.
- *The next tick retries.* `internal/queue/reaper.go` stated this.

**The off-measurement-path ground is false at the reaper, and the reaper is one of the nine.**
`internal/queue/reaper.go:45` records that nothing else exits `running`. A dead worker therefore
strands its job and blocks `Dispatch` forever
([#853](https://github.com/winniel123/verge-asm/issues/853)). A failed reap sits squarely **on** the
measurement path: it is the reason a measurement does not run. A ground that is false at one of the
nine sites cannot be the ground for all nine.

**ADR-0108 is the only ADR that touches this.** Its rule is that a failure is legible as a failure
and never renders as a clean empty result. Limb 6 generalises that to *every backend failure path*
and carves out exactly one: scan dispatch. A dispatch failure produces no batch, so its effect is a
missed cadence the currency machinery already tells (ADR-0084). Read literally, limb 6 reaches all
nine loops and forbids the shape all nine have. That reading is wrong. The carve-out was written one
path wide when the ground behind it is one class wide.

[ADR-0044](./0044-a-one-off-measurement-has-no-currency.md) and ADR-0137 both mention a next tick,
but neither states an error rule. No ADR states the worker-loop rule.

## Decision

> **A periodic sweep loop logs a failed pass and keeps ticking, and the ground is that the next tick
> retries the same work. It is never that the work is off the measurement path. ADR-0108 limb 6 is
> bounded to paths that produce a measurement or an empty result, and does not reach a periodic
> sweep loop.**

Four limbs.

### 1. The loop returns only the context's error

Each of the nine loops handles a failed pass inside the pass. It logs, it returns from the pass, and
it waits for the next tick. `Run` returns only when `ctx` is done, and returns `ctx.Err()`. A
transient database error, a failed feed, a failed delete, a failed fan-out: none of them ends a
worker.

The cost of one failed pass is bounded by the interval, and that bound is the whole argument. A
retention sweep that fails once retires its rows one interval later. A reap that fails once
reclaims its stranded job one interval later.

### 2. The ground is the retry, not the position off the measurement path

The two grounds are not interchangeable, and only the retry generalises.

*Off the measurement path* claims that nothing downstream of the failed work reads a measured value.
That is true of retention and of the release check. **It is false of the reaper**, for the reason
`reaper.go:45` states: a stranded `running` job blocks dispatch, and blocked dispatch is a missing
measurement. It is also false of the queue dispatcher, whose failed pass is a missed cadence.

*The next tick retries* is true of all nine, because it is a property of the loop rather than of the
work. It is the ground this ADR adopts, and the only ground a comment at any of the nine sites may
state.

**The bound of the ground is the retry actually happening.** The rule licenses swallowing a failure
because the same work is attempted again on a fixed interval. A pass whose failure is not retried by
the next tick is outside this rule and gets no cover from it.

### 3. ADR-0108 limb 6 is bounded, at limb 6's own site

ADR-0108's rule is about a failure that is **indistinguishable from an empty result**. That hazard
needs a path that produces a result at all: a batch, a lookup, a delivery. A periodic sweep loop
produces none. Its failed pass produces no batch, no observation and no row a reader could mistake
for *we looked and there is nothing there*.

So limb 6 is bounded rather than carved out a second time. The bounding sentence goes **in ADR-0108,
at limb 6**, not only here. ADR-0058 requires it: a narrowed clause is withdrawn at the site that
specifies it. A reader who finds limb 6 and the nine loops must not have to find this ADR first to
know which one is wrong.

The existing scan-dispatch carve-out stays where it is. It is now an instance of the bound rather
than an exception to an unbounded rule.

### 4. The durable record of a failure, where the path has one, is the unit of work's job

This ADR rules on the loop, not on the work. Where a path already records its own failure durably,
that record is written inside the pass and is unaffected. A `Delivery` in state `undelivered`
carries its `last_error` and renders on the `Message` it failed to carry, exactly as ADR-0108 limb 6
requires. A failed release feed leaves the cache as it was rather than writing a guessed verdict
(ADR-0124). Those are decisions about the work.

The loop's own error is a narrower question: whether the worker keeps ticking. The answer is yes.

## Consequences

- **[ADR-0108](./0108-a-batch-whose-instrument-could-not-reach-its-position-covers-nothing-and-the-failure-is-the-vantages.md)
  limb 6 gains one bounding sentence** at its own site, naming this ADR.
- **`internal/retention/retention.go` states the retry ground** instead of the off-measurement-path
  ground, and cites this ADR. The old wording asserted the ground this ADR rejects.
- **`internal/queue/reaper.go` gains this ADR's citation** on the comment that already states the
  retry ground.
- **The seven other loops gain nothing.** They state no ground, so they assert nothing wrong, and a
  citation on each of nine sites is nine copies of one rule. The two sites that speak are the two
  sites that carry the citation.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** Retirer, Reaper and worker loop are not domain
  terms, and this is a decision rather than a term. A glossary entry would file a rule where readers
  look for a vocabulary.
- **No production behaviour changes.** All nine loops already have the shape this ADR states. The
  change is that the shape now has one ground and one record.
- **A tenth loop is admitted by stating the retry ground and citing this ADR.** A loop that cannot
  state it is a different case and needs its own decision.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Keep the off-measurement-path ground** *(`internal/retention`'s wording)* | **False at the reaper.** `internal/queue/reaper.go:45` states that nothing else exits `running`, so a dead worker strands its job and blocks `Dispatch` forever (#853). A failed reap is the reason a measurement does not run, which puts it on the measurement path. The ground also fails at the queue dispatcher, whose failed pass is a missed cadence. A ground that holds at some of the nine sites and not others is not the rule's ground |
| **Read ADR-0108 limb 6 literally and make every loop return its error** | Ends a worker on a transient database blip and trades a bounded one-interval delay for an unbounded outage. It also mistakes the hazard: limb 6 exists to stop a failure reading as a clean empty result, and a sweep that produced no result cannot read as one |
| **Add a second carve-out to limb 6, beside scan dispatch** | Treats one class as a second exception. The scan-dispatch bullet and the nine loops share one ground, so the honest edit states the bound and lets dispatch fall under it |
| **Surface a failed sweep on `Coverage`, or open a `Gap`** | A sweep failure is not a statement about the estate. `Coverage` would carry an operational fact keyed on nothing that moved, which ADR-0064's construction forbids. Where a sweep failure does delay a measurement, the missed cadence already ripens into a `Gap` under ADR-0084 |
| **Build a durable sweep-error store** | The same new subsystem ADR-0108 limb 6 already deferred for scan dispatch, carrying a failure the interval bound and the log line already tell. Deferred here for the same reason |
| **Write the rule in `CONTEXT.md`** | The glossary carries terms. Retirer, Reaper and worker loop have no entry, and adding one to hold a decision puts the decision where nobody reads for it |
| **One ADR per finding** (#1291, #1321 §5, #1272 §1) | Three records of one rule, in the shape the sweeps were already flagging: one rule stated three times in three wordings. #1321 §5 says outright that a triager should dedup its record against #1291's |
| **Amend ADR-0108 and file nothing** | Under ADR-0058's split an amendment carries a claim about the world. This is a rule about nine loops that ADR-0108 never ruled on, plus a rejection of the ground the code gave. ADR-0108 takes the bounding sentence its own limb needs and no more |
| **Leave the rule uncited and let each loop state its own ground** | The state three sweeps flagged. It produced two grounds, one of them false, across nine sites that behave identically |
