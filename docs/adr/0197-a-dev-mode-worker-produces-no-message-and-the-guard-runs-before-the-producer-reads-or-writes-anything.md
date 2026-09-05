# ADR-0197: A dev-mode worker produces no message, and the guard runs before the producer reads or writes anything

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1315 ADR gaps: internal/queue (2/7)](https://github.com/winniel123/verge-asm/issues/1315), gap 2
- **PR that deleted the comment:** [#1314](https://github.com/winniel123/verge-asm/pull/1314)
- **Dedup:** this ADR is also the surviving record for [#1316](https://github.com/winniel123/verge-asm/issues/1316) gap 4's **message** half. [`comment-policy.md`](../spec/comment-policy.md) §8.10 dedups by rule sentence and #1315 §2 won the sentence
- **Sibling of, and not ruled by:** [`raw-job-output.md`](../spec/raw-job-output.md) §2.5, which states the **transcript** half — capture is a `WithTranscripts` seam, off when unwired and under `devMode`. It rules nothing about a message. The two halves read one field and are two rules
- **Rests on:** [ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md), which rules that a message names what moved and reads it from the fold. It fixes what a message is and when one is owed. It does not rule any build in which none is owed
- **Bounded by:** [`comment-policy.md`](../spec/comment-policy.md) §4.7, which rules `AL-25` and `G2` unrepairable tokens. Both are cited by the surviving lines this ADR now supplies a citation for

## Context

`internal/queue/produce.go:60` carried this, until [#1314](https://github.com/winniel123/verge-asm/pull/1314) deleted it:

```go
// cmd/worker opts in with the live delivery enqueuer. **VERGE_DEV guard (AL-25):**
// even opted-in, a devMode worker produces NOTHING — a real deployment never serves
// fixtures, and the golden fixtures must stay message-free so G2 does not move.
```

The compressed survivor sits at `internal/queue/produce.go:55`. A second survivor states the same
rule from the wiring side, at `cmd/worker/main.go:87`:

```go
// A fixture-only install serves fixtures, never live estate, so it writes no message (AL-25).
```

Both citations are dead. [`comment-policy.md`](../spec/comment-policy.md) line 872 rules `AL-25`,
`AUDIT-LEDGER`, `AL-2`, `P0.7`, `DF-F4` and `G2` unrepairable **not by token**, and line 881 records
that the first five appear in zero files under `docs/` and in `CONTEXT.md`. Nothing on disk states
the rule, so §8.3 suppresses nothing and the rule is unwritten.

### The wiring, and where the guard sits

`cmd/worker/main.go:88` reads the environment once and passes the same flag to both builders:

```go
devMode := isTruthy(env.OrDefault("VERGE_DEV", ""))
worker := queue.NewWorker(...).
    ...
    WithMessages(delivery.EnqueueForMessage, devMode).
    WithTranscripts(transcriptKey, devMode).
```

So the shipped worker is opted in to message production and to transcript capture at the same
moment it declares itself a dev-mode worker. The two are not alternatives.

`produceMessages` (`internal/queue/produce.go:53`) then opens:

```go
_ = batchID
// A devMode worker produces nothing, so the golden fixtures stay message-free.
if devMode || (len(changes) == 0 && len(departures) == 0 && len(narrowings) == 0) {
    return nil
}
```

The guard is the **first executable statement** of the producer and it is unconditional. Everything
the producer does afterwards is behind it:

| Step | Site | What it does |
| --- | --- | --- |
| `coveredAddressScope` | `produce.go:154` | store read |
| `ListServiceReachabilitySpansByClass` | `produce.go:159` | store read |
| `PreviousBatchTime` | `produce.go:166` | store read |
| `InsertMessage` | `produce.go:65` | store write |
| `enqueue` | `produce.go:72` | store write, via `delivery.EnqueueForMessage` |

**A dev-mode worker therefore performs no store read and no store write on the message path.** The
guard is not a filter over produced messages. Nothing is computed and nothing is inserted.

### What a produced row would actually do in a dev-mode install

The deleted comment gives two grounds. The first is exactly right. The second names the wrong corpus,
and the measurement is worth having, because the true consequence is heavier than the one the comment
claims.

**Ground one — a real deployment never serves fixtures — is correct, and it is the operative one.**
A `VERGE_DEV` install's message surfaces are served from authored fixtures and never from the
`message` table. `cmd/web/messages.go:145` short-circuits the inbox to `s.inboxFixtureData` before it
touches the store. `cmd/web/devfixtures.go:492` holds a hardcoded `devCoverageMessages` list, and
`cmd/web/devfixtures.go:2167` reads a `messages` block out of `design-system/fixtures/fixtures.json`.
A row a dev-mode worker wrote would be invisible on every console screen that renders messages, so
nobody could see it, read it, or reconcile it against the fixture set that is shown instead.

**Ground two — "the golden fixtures must stay message-free so `G2` does not move" — names a corpus a
message cannot reach.** The `golden-corpus` CI job runs `go test ./internal/measure/...
./internal/custody/...` and nothing else. The seven corpora on disk are the six measure leaves and
`internal/custody`. A `message` row is written by `internal/queue` after the fold and is read by
neither package, so no message can move a golden corpus digest or a leaf version.
[`golden-corpus.md`](../spec/golden-corpus.md) contains zero occurrences of `message`. `G2` is
separately dead: [`comment-policy.md`](../spec/comment-policy.md) line 934 records that it resolves
falsely in two unrelated senses, five uses being ADR-0057's gate-check label and one the retired
design-system parity gate.

**The ground the comment missed is the one with teeth.** `produceMessages` does not only insert a
row. It calls `enqueue`, which `cmd/worker/main.go:97` binds to `delivery.EnqueueForMessage`. That
function lists every channel, tests each against the message's class, and inserts a `delivery` row
per match (`internal/delivery/runner.go:88-101`). The delivery `Runner` then polls and performs a
real signed HTTP POST to the operator's declared channel. **A dev-mode worker without this guard
would page a real operator about a fixture estate.** That is a live outbound act, and it is why the
guard is the producer's first statement rather than a filter near the insert.

## Decision

> **A `devMode` worker produces no message at all. The rule holds even where message production is
> opted in, so `WithMessages` and `devMode` together mean production is wired and suppressed. The
> guard is the first statement of the producer and runs before any store read and before any store
> write, so a dev-mode worker reads nothing, inserts no `message` row and enqueues no delivery.**

### 1. The rule is *no message*, not *no delivery* and not *fewer messages*

A dev-mode install serves fixtures. The estate a dev-mode worker measures is a fixture estate, so
every message it could compute would be a true statement about a thing that is not real. There is no
smaller correct output than none.

This is why the guard is not `enqueue == nil` and is not a class filter. Suppressing only the
delivery would leave `message` rows in a database whose console renders fixtures instead of them —
invisible, undeletable through the UI, and waiting for the install to be switched out of dev mode.

### 2. The guard runs before the producer touches the store

Stated as a position and not only as an outcome: the check sits at
`internal/queue/produce.go:56`, ahead of `buildMessages`, ahead of the three store reads
`flagshipMessages` performs, and ahead of `InsertMessage`.

The position is load-bearing. A guard placed after `buildMessages` would produce the same visible
result and would still read the store inside the batch transaction on every fold of a dev-mode
worker. A guard placed at the `enqueue` call would insert the rows. The current position is the only
one at which the rule "produces nothing" is literally true.

It shares its statement with the empty-work check, `len(changes) == 0 && len(departures) == 0 &&
len(narrowings) == 0`. The two conditions have different reasons — one is *this build never speaks*,
the other is *nothing moved* — and they are combined because both return the same nil at the same
place. The compound is not a claim that they are the same rule.

### 3. It is the message half of one flag, and the transcript half is ruled elsewhere

`Worker.devMode` (`internal/queue/worker.go:124`) governs two suppressions.
`captureOn` (`internal/queue/worker.go:142`) is `captureTranscripts && !w.devMode`, and
[`raw-job-output.md`](../spec/raw-job-output.md) §2.5 rules that half. This ADR rules the message
half and does not restate the transcript half.

One flag, two rules, because the two have different consequences. A suppressed transcript loses a
diagnostic. A suppressed message stops an outbound POST.

### 4. What this rule does not reach

- **An unwired producer.** `(*Worker).produce` (`internal/queue/worker.go:186`) returns nil when
  `produceMsgs` is false, so a measurement-only worker writes no message either. That is a different
  rule with a different reason — the seam is optional so a build can omit it — and it is not
  `devMode`.
- **The console's own dev surfaces.** `cmd/web` reads `VERGE_DEV` independently at
  `cmd/web/main.go:107`, and roughly thirty handlers branch on `s.devMode` to render a fixture.
  Whether a console screen serves a fixture is not ruled here.
- **What a message says.** [ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md)
  rules that. This ADR rules a build in which none is written.
- **The `-seed-fixtures` loader.** `cmd/web/main.go:45` bars it outside a `VERGE_DEV` build and it
  reseeds the open-span corpus. It writes no `message` row and is not reached by this rule.

## Consequences

- **This ADR changes no Go code.** The guard exists, is first, and is unconditional.
- **The record's second ground is corrected on the record.** The `golden-corpus` CI job cannot see a
  `message` row, and `G2` names nothing. A later reader must not go looking for a golden corpus that
  a message could move. The operative grounds are the fixture-served console and the live outbound
  delivery, both measured in §Context.
- **`Worker.devMode` is written by two builders and the last one wins.** `WithMessages`
  (`internal/queue/worker.go:156`) and `WithTranscripts` (`internal/queue/worker.go:133`) each assign
  the field. `cmd/worker/main.go` passes the same value to both, so the defect is latent today. A
  wiring that passed `true` to one and `false` to the other would silently clear the guard, and no
  test would fail. **This is a defect and it ships as its own ticket:** the flag should be set once,
  by its own builder or by `NewWorker`, so that the two seams cannot disagree.
- **No test pins the rule.** `internal/queue` has no case asserting that a `devMode` worker inserts
  no `message` row and enqueues no delivery. **This ships as its own ticket.** ADR-0140 §4 sets the
  precedent that logic an adapter carries owes its own test, and the shape here is the same: a rule
  that is a single boolean is a rule a refactor can drop in one keystroke.
- **Two survivors gain a citation.** `internal/queue/produce.go`'s `produceMessages` guard and
  `cmd/worker/main.go`'s `devMode` read were uncited and cited a dead token respectively. The
  `produce.go` line also stated the corrected-away golden ground and is rewritten to the true one.
  Both edits are applied at their own sites and cite this ADR's §1.
- **`internal/queue/worker.go:140`'s survivor is not touched here.** It states the joint transcript
  and message rule and belongs to [#1316](https://github.com/winniel123/verge-asm/issues/1316)'s
  record.
- **No [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
  withdrawal is owed.** [`raw-job-output.md`](../spec/raw-job-output.md) §2.5 keeps stating the
  transcript half, correctly and in the present tense. No mechanism it specifies stops existing.
- **`CONTEXT.md` gains nothing.** `devMode` is a build flag, not a domain term. `VERGE_DEV` appears in
  `CONTEXT.md` zero times today and should stay that way.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Suppress the delivery enqueue alone, and keep writing the `message` row** | The rows land in a database whose console renders fixtures instead of them (`cmd/web/messages.go:145`). They are invisible, unreadable and unclearable through the UI, and they become live the day the install drops `VERGE_DEV`. The write also costs the batch transaction three store reads and an insert per fold, for a row nothing may read |
| **Do not opt a dev-mode worker into `WithMessages` at all, in `cmd/worker`** | It moves the rule into the composition root, where it is one `if` a future wiring change can drop with no test failing, and it splits the flag: `WithTranscripts` would still take `devMode` while `WithMessages` did not. The guard inside the producer holds for every caller, including a test that wires the seam directly |
| **Place the guard just before `InsertMessage`** | The visible result is identical and three store reads still run inside the batch transaction on every fold. It also stops the rule being statable as "produces nothing", which is the sentence a reader needs |
| **Rely on `golden-corpus.md` to carry the rule** | It cannot. The `golden-corpus` CI job runs `./internal/measure/... ./internal/custody/...`, and a `message` row is written by `internal/queue` and read by neither. `golden-corpus.md` contains zero occurrences of `message`. This is the ground the deleted comment gave, and it was the wrong corpus |
| **Repair the `AL-25` and `G2` citations rather than write an ADR** | [`comment-policy.md`](../spec/comment-policy.md) §4.7 rules both tokens unrepairable and line 934 records that `G2` resolves falsely in two senses, so a grep-repair would produce a citation that is wrong and looks right. The surface it points to has to be written first, and that is this ADR |
| **State the rule in [`raw-job-output.md`](../spec/raw-job-output.md) §2.5, beside the transcript half** | That section rules a transcript seam under a ticket about raw job output. A suppressed transcript loses a diagnostic; a suppressed message stops an outbound signed POST to an operator's channel. Filing the second under the first hides the heavier consequence inside a document about capture |
| **Fix the two-builder `devMode` assignment on this ADR's own branch** | It changes `NewWorker`'s or the builders' signatures and every construction site in `cmd/worker` and in `internal/queue`'s tests. That is a production change with its own review, and it is named in Consequences instead |
