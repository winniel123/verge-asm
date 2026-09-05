# ADR-0219: a withdrawal is a fact about the estate, so the closure is unconditional and only the receipt is gated on the collector

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1323 ADR gaps: internal/queue (queue, cttail, withdrawal, hot, ctverify, scopegate)](https://github.com/winniel123/verge-asm/issues/1323), gap 10
- **PR that deleted the comment:** [#1322](https://github.com/winniel123/verge-asm/pull/1322)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0007](./0007-drift-is-a-timeline-of-spans.md), which makes the span timeline the estate's own record. A closure is a change to that record and not a notification about one
- **Rests on:** [ADR-0111](./0111-a-span-cites-the-batch-that-folded-it.md), which makes every closure cite the folding `Batch`. That citation is written by the closure, so it is written on the ungated side
- **Rests on:** [ADR-0074](./0074-an-aperture-narrowing-that-takes-its-carrier-with-it-fires-at-the-scope.md), which fixes the receipt as one count-carrying statement at the scope. The receipt is the gated half
- **Sibling of, and not ruled by:** [ADR-0218](./0218-the-address-exclusion-withdrawal-is-idempotent-by-construction-and-its-receipt-twins-the-previews-site-and-counts.md). That ADR rules how the receipt is built and how a second fold is safe. This one rules whether the receipt is built at all, and what happens when it is not
- **Rests on:** [ADR-0199](./0199-delivery-imports-queue-never-the-reverse-so-the-worker-takes-the-message-enqueuer-as-an-injected-function-that-joins-the-batch-transaction.md), whose Decision already states the whole-worker half — *"a worker nobody wires with that option writes no message, refuses nothing, and still measures and still commits."* It rules the **import edge** and the injected seam. This ADR rules where inside each fold the boundary falls, which is the half that decides whether an unwired worker's **estate** is the same
- **Sibling of, and not ruled by:** [ADR-0197](./0197-a-dev-mode-worker-produces-no-message-and-the-guard-runs-before-the-producer-reads-or-writes-anything.md). It rules a second suppression, `devMode`, and fixes its guard as the producer's first statement. That guard sits inside `produce`, downstream of every fold. This ADR rules the folds, so a dev-mode worker withdraws for the same reason an unwired one does
- **Sibling of, and not ruled by:** [ADR-0211](./0211-the-value-fold-decides-no-withdrawal-so-the-membership-path-closes-the-timelines-and-names-the-ground.md). It rules **which** fold may close a span with a reason. This rules that the closure, wherever it is ruled to belong, is written whether or not anything states it

## Context

[`internal/queue/withdrawal.go:82`](../../internal/queue/withdrawal.go) carried this text, until
#1322 deleted it:

```go
// The closure happens whether or not the producer is on. A withdrawal is a fact
// about the estate, not a message. `out` is the narrowing collector and is nil on
// the measurement-only path, exactly as departureCollector is.
```

The sweep left one compressed line at `withdrawal.go:37` and one at `worker.go:179`. Nothing on disk
states the rule. That is #1323's gap 10.

**Five folds carry the same split, and every one of them puts the estate write on the ungated
side.**

| Fold | Durable estate write, unconditional | Collector append, guarded |
| --- | --- | --- |
| `foldObservationsIntoSpans` | `InsertSpan` (`spanfold.go:88`) | `if changes != nil` (`spanfold.go:104`) |
| `foldEstateTransitions` | `closeSubjectTimelines` (`membership.go:53`) | `if deps != nil` (`membership.go:56`) |
| `foldNameSeedWithdrawals` | `closeSpansByID` (`nameseedwithdrawal.go:32`) | `if out != nil` (`nameseedwithdrawal.go:35`) |
| `foldAddressExclusionWithdrawals` | `closeSpansByID` (`withdrawal.go:34`) | `if out != nil` (`withdrawal.go:38`) |
| `foldSeedWithdrawals` | `closeSpansByID` (`seedwithdrawal.go:34`) | `if out != nil` (`seedwithdrawal.go:38`) |

**The collector is nil where the `Worker` was built without message production.**
`changeCollector`, `departureCollector` and `narrowingCollector` (`worker.go:163`, `:171`, `:178`)
each return nil when `w.produceMsgs` is false, and `produce` returns immediately in the same case
(`worker.go:188`). `produceMsgs` is set only by `WithMessages`.

**Today `cmd/worker/main.go:97` always calls `WithMessages`, so the nil path is the test path.** The
boundary is therefore a design boundary rather than a shipped configuration, and that is exactly why
it needs a record. A reviewer who reads the guard as *this fold is part of the message path* has no
counter-example in front of them.

## Decision

> **A withdrawal changes the estate, and the estate's record is written whether or not anything
> states it. The closure, its `descoped` reason and its batch citation run unconditionally. Only the
> receipt is gated on the collector. A `Worker` built with no message production still withdraws. It
> says nothing about it.**

Four limbs.

### 1. The closure is the act, and the receipt is a statement about the act

An address the operator excluded is no longer in the estate. That is true of the estate, not of the
operator's notification settings. The span timeline is where the product records what the estate
is (ADR-0007), so the closure belongs to the same layer as the measurement it ends.

The receipt is a sentence about the act, in the `Message` corpus, under ADR-0074's form. It is
downstream of the fact and it is optional in a way the fact is not.

### 2. The three writes a closure makes are all on the ungated side

`CloseSpan` writes `closed_at`, `closure_reason` and `closed_batch_id` together
(`membership.go:186`). All three are the estate's record.

`closed_batch_id` is the one worth naming, because it is the fact ADR-0111 requires: a span cites
the batch that folded it. If the closure moved under the collector guard, a build with no message
production would leave the span open, and the citation ADR-0111 rules would never be written at
all. The narrowing receipt is not what makes a closure traceable. The batch citation is.

### 3. A closure with no receipt is not a lost message

Turning message production on later produces no message for a closure already made, and that is
correct rather than a gap. A `Message` is written once and never recomputed (ADR-0074,
ADR-0134 §6), so a retrospective one would state an act at an instant it did not happen.

What survives is everything durable. The span carries its `descoped` reason and its batch citation,
the timeline shows the closure at the instant it happened, and the census counts the subject out.
An operator reading the estate reaches the same answer with or without the receipt. Only the
push notification is missing, and it is missing for acts that happened while nothing was listening.

### 4. The boundary forbids two edits that look like tidying

Both are the reason this rule needs a document.

- **Moving `closeSpansByID` under the `if out != nil` guard.** It reads as removing dead work when
  the collector is nil. It makes the estate's shape depend on whether messaging is wired, which is
  the one coupling this split exists to prevent.
- **Returning early when the collector is nil.** `if out == nil { return nil }` at the top of a fold
  is a shorter version of the same error, and it also skips the estate read, so nothing downstream
  can notice.

The safe shape is the one all five folds already have: compute the closure set, write it, then
append to the collector where there is one.

## Consequences

- **This ADR changes no Go code.** All five folds already split this way.
- **The nil-collector path is exercised by tests alone today.** `cmd/worker/main.go:97` wires
  `WithMessages` unconditionally, so no shipped configuration takes it. A build that omits it — a
  future measurement-only worker, or a migration tool that folds without notifying — is what the
  boundary is for, and the tests are what keep it working until then.
- **The rule binds four folds this issue does not name.** `foldObservationsIntoSpans`,
  `foldEstateTransitions`, `foldNameSeedWithdrawals` and `foldSeedWithdrawals` carry the same split
  for the same reason. This ADR states it once for all five, so a fifth fold is written to a rule
  rather than to the nearest example.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** No domain term moves. The estate's definition
  is unchanged, and this ADR states that the definition does not depend on the message corpus.
- **Nothing enforces the split.** No test asserts that a fold with a nil collector still closes its
  spans on every one of the five paths. **A table test that runs each fold with a nil collector and
  asserts the closure ships as its own ticket.**
- **Two comments carry the rule and no citation.** `withdrawal.go:37` and `worker.go:179` each state
  one half. Citations are recorded in this issue's manifest.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Skip the fold entirely when the collector is nil** | Makes the estate depend on the notification wiring. A worker with no message producer would keep withdrawn timelines open for ever, so the census, the coverage figures and the drift readings would all differ between two builds of the same product over the same data |
| **Close the spans, and buffer the receipts for a later producer** | Needs durable storage for an unsent receipt and a rule for how old a receipt may be before it is meaningless. It also contradicts ADR-0074's write-once form: the receipt would state an act at one instant and reach the operator at another, with the estate already moved on |
| **Gate the closure and let a later sweep close the orphans** | Adds a second closure path with no `Batch` to cite, which ADR-0111 forbids, and ADR-0133 §8.1 already rejected a declaration-time closure on the same ground. It also reintroduces the estate-wide sweep ADR-0134's Alternatives table refused |
| **Pass a no-op collector rather than nil, so the fold never branches** | The branch is not the cost. A non-nil collector means `produce` receives receipts it must then decide not to write, which moves the same decision one layer later and puts it beside the message-writing code, where a future edit would read it as a bug |
| **State the rule only at `narrowingCollector`** | It binds five folds in four files, and the collector constructor is the one place a reader is least likely to be standing when they make the §4 edit. The comment policy admits one clause beside one statement, and the rule needs the whole table in §Context to be checkable |
