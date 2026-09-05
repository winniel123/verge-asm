# ADR-0199: delivery imports queue, never the reverse, so the worker takes the message enqueuer as an injected function that joins the batch transaction

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1316 ADR gaps: internal/queue (edgefanout.go, worker.go)](https://github.com/winniel123/verge-asm/issues/1316), gap 3
- **PR that deleted the comment:** [#1317](https://github.com/winniel123/verge-asm/pull/1317)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md), which rules what a `Message` is and that it is computed once, at the cause. It never says which package writes the row, and it never says how the row reaches a `Channel`
- **Rests on:** [ADR-0039](./0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md), which rules that a `Delivery` is an Operational record and carries no estate. It rules what the row records. It does not rule when the row is written
- **Bounded by:** [ADR-0164](./0164-an-operator-ends-a-dispatch-by-recording-a-disposition-once-and-stop-keeps-the-running-jobs-while-terminate-rolls-their-staged-work-back.md), which rules that a terminate rolls a running job's staged work back. §3 below is what puts a message and its `Delivery` rows inside that staged work
- **Sibling of, and not ruled by:** [ADR-0140](./0140-a-network-seam-is-a-runtime-parameter-the-caller-supplies-never-a-build-tag-and-never-a-hardcoded-client.md), which rules the **network** seam. Both rules inject a value at run time. ADR-0140 injects so that a test reaches no network. This ADR injects so that a layer edge stays one-way, and the two reasons decide different code
- **Sibling of, and not ruled by:** [ADR-0149](./0149-a-consumer-takes-the-data-layer-interface-it-calls-and-the-seam-not-the-package-is-the-unit.md), which rules how **wide** a consumer's data-layer interface is. Its §5 names `internal/queue.Worker` and `internal/delivery.Runner` as pool owners and exempts them. This ADR rules the **direction** of the edge between those two packages
- **Sibling of, and not ruled by:** [ADR-0197](./0197-a-dev-mode-worker-produces-no-message-and-the-guard-runs-before-the-producer-reads-or-writes-anything.md), which rules the **second** condition on the same seam: a `devMode` worker produces no message even when the option is wired. That is a fixture-install rule. This ADR rules the wiring itself, and neither contains the other

## Context

`internal/queue/worker.go:167-173` carried this, until #1317 deleted it:

```go
// The message producer's seam (P0.7), wired via WithMessages. When enabled the
// batch tx folds each signal/drift transition into a message and routes it to its
// bound channels via enqueue (delivery.EnqueueForMessage, injected to avoid the
// delivery→queue import cycle). Off on a worker built without it — the
// measurement-only construction and its tests write no message. devMode suppresses
// production entirely even when enabled: a fixture-only install never writes a
// message (AL-25), so the golden fixtures stay message-free and G2 does not move.
```

The same block restated the rule at `worker.go:234-239` and `worker.go:280-284`, pre-sweep. `P0.7`
names nothing in `docs/` or in `CONTEXT.md`, so [`comment-policy.md`](../spec/comment-policy.md) §4.7
rules the token unrepairable and directs the sweep to record the gap. That is #1316's gap 3.

**The quote carries two rules and this ADR rules one of them.** Its last sentence states the
`devMode` suppression.
[ADR-0197](./0197-a-dev-mode-worker-produces-no-message-and-the-guard-runs-before-the-producer-reads-or-writes-anything.md)
rules that half, on #1315. This ADR rules the dependency direction, the injection and the unwired
case, and it does not re-rule `devMode`.

### The edge, measured

| Fact | Site | Value |
| --- | --- | --- |
| `internal/delivery` imports `internal/queue` | `internal/delivery/runner.go:23` | one import, for one symbol |
| The one symbol it uses | `internal/delivery/runner.go:184` | `queue.Backoff(claim.Attempt + 1)` |
| `internal/queue` imports `internal/delivery` | — | nowhere |
| First-party packages in `internal/queue`'s transitive closure | `go list -deps` | 22, and `internal/delivery` is not one |
| Packages in `internal/queue`'s whole transitive closure | `go list -deps` | 276 |
| Packages in `internal/delivery`'s whole transitive closure | `go list -deps` | 277 |
| The one package in the second set and not the first | — | `internal/delivery` |
| Packages that import `internal/delivery` | `go list` | 3: `cmd/web`, `cmd/worker`, `internal/report` |
| Files that import `internal/queue` | `grep` | 9, across `cmd/web`, `cmd/worker`, `internal/delivery`, `internal/report` |

**`internal/delivery`'s dependency closure is `internal/queue`'s closure plus `internal/delivery`
itself, and nothing else.** 277 against 276, and the single difference is the package under test.
Delivery is a thin layer that sits directly on the measurement stack. The measurement stack carries
276 packages beneath it and needs none of delivery's.

The type the injected function speaks is what makes the one-way edge writable. `internal/message`
imports no other first-party package. `internal/queue` and `internal/delivery` both import it. So
`internal/queue` can name `message.Class` in a function signature without naming `internal/delivery`.

### What the injected value is, and where it is bound

| Step | Site |
| --- | --- |
| The field, typed as a function | `internal/queue/worker.go:125` |
| The option that supplies it | `internal/queue/worker.go:156` (`WithMessages`) |
| The one production wiring site | `cmd/worker/main.go:97`, passing `delivery.EnqueueForMessage` |
| The exported value it passes | `internal/delivery/runner.go:86` |
| The batch transaction | `internal/queue/worker.go:517` (`inTx`), which calls `w.q.WithTx(tx)` at `:523` |
| The transaction the message rides | `internal/queue/worker.go:382`, `complete`'s `runJobTx` call |
| The call that binds the open handle into the seam | `internal/queue/worker.go:193-195` |
| The narrowed in-package seam type | `internal/queue/produce.go:25` (`enqueueFunc`) |
| The enqueue call | `internal/queue/produce.go:72` |

`produce` closes over `qtx`, the handle `inTx` opened, and hands the closure to `produceMessages`:

```go
enqueue = func(c context.Context, messageID int64, class message.Class) (int, error) {
	return w.enqueue(c, qtx, messageID, class)
}
```

So `delivery.EnqueueForMessage` runs on the worker's open transaction. It calls `q.ListChannels` and
`q.InsertDelivery` on that same handle (`internal/delivery/runner.go:88`, `:97`). It opens nothing of
its own, and it takes no pool.

### Three comments state the rule, and none of them cites a document

The ticket records one survivor. There are three.

| Site | Text |
| --- | --- |
| `internal/queue/worker.go:121` | `// The enqueuer is injected because internal/delivery imports this package.` |
| `internal/queue/produce.go:23` | `// delivery imports queue, so the reverse edge would cycle: this seam is injected, never imported.` |
| `cmd/worker/main.go:91` | `// The message hook is injected so internal/queue never imports internal/delivery.` |

Three sites in two packages assert the same rule, and no ADR, no `docs/spec/` file and no
`CONTEXT.md` entry states it.

### The cycle is not the ground, and this is why the rule needs writing down

All three comments give the same reason. The reverse edge would cycle. **That reason is true today
and it is contingent.** The edge from `internal/delivery` to `internal/queue` exists for one symbol,
`queue.Backoff`. `internal/report/notify.go:146` calls the same function. A session that moved
`Backoff` into a package below both would open the cycle. `internal/queue` could then import
`internal/delivery`, the compiler would accept it, and every comment on the subject would have been
satisfied and made irrelevant in the same commit.

The transitive cost of that move is one package (277 against 276), so a size argument does not carry
the rule either.

**What carries the rule is layering.** Measurement is the lower layer. A worker measures, folds and
commits without any notion of a channel, a webhook or a signature. The comments name the compiler
error and not the reason the compiler error is welcome, so the reason has never been written down.

### The transaction fact the schema already carries

`db/migrations/20700_delivery.sql:24` declares `message_id BIGINT NOT NULL REFERENCES message (id)`.
A `Delivery` row cannot exist before its `Message` row. `internal/queue/produce.go:65` inserts the
`Message` and `:72` enqueues against it, in the order the constraint requires, inside one transaction.

`db/migrations/20700_delivery.sql:42` then declares `UNIQUE (message_id, channel_id)`. One routed
pair yields one row, forever. ADR-0064 rules that a message is computed once at the cause. So a
`Message` that commits with no `Delivery` row is a message no later pass re-routes.

The rollback path is live. `internal/queue/worker.go:333` declares `errJobCanceled`, and the three
guarded terminal writes return it at `:341`, `:352` and `:363` when they match no row. `runJobTx` at
`:368` swallows it after `inTx` rolled the transaction back. ADR-0164 rules that a terminate rolls a running job's staged work back. Because the
enqueue joins the batch transaction, that rollback discards the batch, the observations, the spans,
the message and its `Delivery` rows together.

### Four other seams on the same struct, and they do not agree with each other

`Worker` takes six injected seams. Their unwired behaviour is already three different things.

| Seam | Option | Unwired behaviour | Site |
| --- | --- | --- | --- |
| `ctFetcher` | `WithCT` | **Refuses the job** | `internal/queue/crtsh.go:126` |
| `ctTailFetcher` | `WithCTTail` | **Refuses the job** | `internal/queue/cttail.go:34` |
| `ctVerifyFetcher` | `WithCTVerify` | **Refuses the call** | `internal/queue/ctverify.go:50` |
| `ctSource` | `WithCT` | Falls back to the keyless crt.sh path | `internal/queue/crtsh.go:130` |
| `router` | `WithRouter` | Probes locally | `internal/queue/worker.go:318` |
| `enqueue` | `WithMessages` | **Writes no message and refuses nothing** | `internal/queue/worker.go:188` |

`internal/queue/worker.go:106` states the refusing rule for the CT seams: *an unwired seam refuses
rather than silently admitting nothing*. The message seam does the opposite, four lines further down
the same struct, and nothing states why the two differ.

### The measurement-only construction exists, and it is in test

`cmd/worker/main.go:92` is the only caller of `NewWorker`, and `cmd/worker/main.go:97` is the only
caller of `WithMessages`. So production has one worker and it is wired.

The unwired worker is built eight times, in `internal/queue`'s own tests, as a composite literal that
sets no `produceMsgs` and no `enqueue`:

| Site | Purpose |
| --- | --- |
| `internal/queue/ctverify_test.go:46` | CT verification against a scripted fetcher |
| `internal/queue/ctverify_test.go:277` | the unwired-verifier refusal |
| `internal/queue/probetimeout_test.go:28` | the probe deadline |
| `internal/queue/probetimeout_test.go:54` | the disabled probe deadline |
| `internal/queue/transcript_test.go:414`, `:417`, `:420`, `:425` | the capture guard and the sealed transcript |

No file in `internal/queue`, test or production, imports `internal/delivery`. `go test
./internal/queue` therefore links no HTTP client, no HMAC signer and no outbound dial guard.

## Decision

> **`internal/delivery` may import `internal/queue`. `internal/queue` must never import
> `internal/delivery`, or any package that reaches it. Measurement is the lower layer and it stands
> alone. The worker reaches delivery only through a function value its caller supplies, the injected
> call runs on the batch transaction the worker already holds, and a worker nobody wires with that
> option writes no message, refuses nothing, and still measures and still commits.**

### 1. The direction is fixed, and the cycle must not close

`internal/delivery` imports `internal/queue`. `internal/queue` imports no delivery code, and imports
no package that reaches delivery code.

**The reason is layering, not the compiler.** Measurement is below notification. A worker claims a
job, probes, gates the observations, folds them into spans and commits a `Batch`. Not one of those
acts needs a `Channel`, a webhook URL, an HMAC signature or a `Delivery` row. A package that cannot
measure without a message producer cannot be built for measurement-only use.

The compiler error is the enforcement, and the enforcement is welcome. It is not the ground. Section
Context measures why: the delivery-to-queue edge carries one symbol, `queue.Backoff`, and moving that
symbol would open the cycle without changing anything about the layering. **This limb binds after
such a move.** A session that opens the cycle does not thereby earn the reverse edge.

`internal/queue` may keep exporting to `internal/delivery`. `Backoff` is a queue concept and delivery
reuses the queue's retry curve on purpose. That edge is the layering, working.

### 2. The enqueuer is a supplied function value, never an import

The worker declares the seam as a field typed as a function (`internal/queue/worker.go:125`) and
takes the value through the `WithMessages` option (`:156`). It never names `internal/delivery` and
never names `delivery.EnqueueForMessage`.

The signature is writable in `internal/queue` because every type in it lives below both packages:
`context.Context`, `*db.Queries`, `int64` and `message.Class`. `internal/message` imports no other
first-party package, so naming `message.Class` costs no edge.

**A function value, not an interface.** The seam has one method and one implementation. ADR-0149 §2
keeps a narrow declaration with its consumer, and a one-method interface here would add a named type
in `internal/queue` and a receiver in `internal/delivery` and state nothing the function type does not.
`internal/queue/produce.go:25` already narrows the seam a second time inside the package, to
`enqueueFunc`, which drops the `*db.Queries` parameter because the closure carries it.

### 3. The injected call joins the batch transaction and never opens its own

`produce` binds the open `*db.Queries` handle into the closure it passes down
(`internal/queue/worker.go:193-195`). Every write the injected function makes lands in the
transaction `inTx` opened at `worker.go:518` and commits at `:526`. The injected function takes no
pool and begins no transaction.

**A message must be atomic with the batch that justifies it.** Three facts force this and each is
already on disk.

- **The schema forbids the other order.** `delivery.message_id` references `message (id)`
  (`db/migrations/20700_delivery.sql:24`). An enqueue in its own transaction would read a `Message`
  row that has not committed.
- **A message is computed once at the cause.** ADR-0064 rules it, and
  `UNIQUE (message_id, channel_id)` (`:42`) makes one routed pair one row. A commit that lands the
  message and then fails to enqueue leaves a message no later pass re-routes.
- **A terminate must discard the whole staged batch.** ADR-0164 rules it, and
  `internal/queue/worker.go:368` implements it. A message written outside the batch transaction
  survives that rollback, and an operator then reads a message for work that never happened.

**This is also why a callback fired after commit is refused.** Such a callback moves the message and
its routing outside the atom, and re-opens all three failures above.

### 4. An unwired enqueuer writes no message, and refuses nothing

`internal/queue/worker.go:188` returns `nil` when `produceMsgs` is false. The batch, the
observations, the certificate material, the spans, the estate transitions, the narrowings, the job
event and the terminal write all still happen, and the transaction still commits.

**This is the direct consequence of limb 1, and it is not an exception.** The seam is a function
value. An unwired worker has a zero value in a field. It has no disabled feature, no flag to read
back and no degraded mode to describe. The measurement-only build is the ordinary build of the lower
layer.

**The CT seams refuse and this one does not, and the difference is evidential.** An unwired CT
fetcher that returned a clean, empty CT batch would assert that certificate transparency named
nothing. That is a false absence, and `internal/queue/crtsh.go:126` refuses rather than state it.
A message asserts nothing about the estate. ADR-0064 rules that a message names what moved, and the
timelines hold the fact whether or not a message reports it. An absent message loses a notification.
It never creates a false reading.

**The unwired case is not a supported production mode, and this limb does not make it one.**
`cmd/worker/main.go:97` wires the option, and a shipped worker that stopped messaging would be a
defect in `cmd/worker`, not a licensed configuration. This limb rules that `internal/queue` compiles,
tests and commits without the option, and nothing more.

### 5. The worker binary is the single wiring site

`cmd/worker/main.go:97` is the only place that names `internal/queue` and `internal/delivery`
together for this purpose, and it is the only caller of `WithMessages`.

**The composition root joins the layers, and no other file may.** This is ADR-0140 §1's shape applied
to a layering seam rather than a network one. A second wiring site would be a second place that
decides whether a batch routes its messages, and the two could disagree per binary.

`cmd/web` and `internal/report` import both packages already. Neither builds a `Worker`, and neither
may. A package that wants a batch to route a message asks `cmd/worker` to wire it.

## Consequences

- **This ADR changes no Go code.** All five limbs describe what the tree does today, verified at
  `internal/queue/worker.go:121-197`, `internal/queue/produce.go:23-75`,
  `internal/delivery/runner.go:23-103` and `cmd/worker/main.go:88-99`.
- **Three comments gain a citation.** `internal/queue/worker.go:121`, `internal/queue/produce.go:23`
  and `cmd/worker/main.go:91` each assert this rule and each is uncited under §4.7. All three gain
  `(ADR-0199 §1, #1316)`. #1316 recorded one survivor. There are three, and the record is corrected
  here.
- **`WithMessages(nil, …)` is a defect this ruling exposes, and it ships as its own ticket.**
  `internal/queue/worker.go:157` sets `produceMsgs = true` whatever the enqueuer is. A caller that
  passes `nil` turns message production on with no route. `internal/queue/produce.go:69-71` then
  skips the enqueue for every message, so the batch commits messages that no `Delivery` row carries
  and no later pass re-routes, because ADR-0064 computes a message once. Limb 2 makes the function
  value the whole payload of the option, so a `nil` payload must either leave the seam unwired or be
  refused. Production never reaches this: `cmd/worker/main.go:97` passes a real function.
- **The `enqueue == nil` branch has no test.** All eleven `produceMessages` calls in
  `internal/queue/produce_test.go` pass `fakeEnqueuer`. The branch ships with the defect above and is
  fixed with it.
- **Nothing enforces limb 1 in CI.** `go build` refuses the reverse edge only while
  `internal/delivery` imports `internal/queue`. A session that moved `queue.Backoff` below both
  packages would remove the enforcement and keep the rule. An import-direction check is not proposed
  here, because one edge does not justify a lint tool, and this ADR is the document a reviewer holds
  the change to.
- **`CONTEXT.md` gains nothing.** Package layering is a code-structure fact, not a domain term.
  `Message` and `Delivery` already carry their domain rules from ADR-0064 and ADR-0039.
- **No `docs/spec/` file is amended.** [`notification-channels.md`](../spec/notification-channels.md)
  rules what a `Channel` and a `Delivery` are and never reaches the write path.
  [`raw-job-output.md`](../spec/raw-job-output.md) §2.4 rules the rollback and names no message.
- **A reader can test every limb without this ticket.** Limb 1 is `go list -deps ./internal/queue`.
  Limb 2 is the field type at `worker.go:125`. Limb 3 is the closure at `worker.go:193`. Limb 4 is
  the early return at `worker.go:188`. Limb 5 is `grep -rn WithMessages`.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Import `internal/delivery` from `internal/queue` and call `delivery.EnqueueForMessage` directly** | It does not compile. `internal/delivery/runner.go:23` imports `internal/queue`, so the reverse edge closes a cycle. Suppose the cycle were opened first, by moving `queue.Backoff` below both packages. The call would then compile and the layering would be gone: `internal/queue`'s dependency closure would name the notification layer, `go test ./internal/queue` would link the outbound HTTP client, the HMAC signer and `custody.IsNonGloballyReachable`'s dial guard, and the eight unwired workers in `internal/queue`'s tests would carry a webhook path they never exercise. Delivery's closure is queue's plus one package (277 against 276), so the direction is free to state and free to keep, and only one of the two directions lets the lower layer stand alone |
| **Move `queue.Backoff` into a shared package and then reverse the edge** | The move on its own is defensible — `internal/delivery/runner.go:184` and `internal/report/notify.go:146` both call it. Reversing the edge afterwards is not. It would put a webhook client, a shared-secret signer and a retry runner underneath every measurement test in the repo, to save one function parameter at one call site |
| **Declare a one-method `MessageEnqueuer` interface in `internal/queue` instead of a function field** | The seam has one implementation, `delivery.EnqueueForMessage`, and it is a package-level function with no receiver and no state. An interface would force a wrapper type in `internal/delivery` and a named type in `internal/queue`, and state nothing the function type does not. `internal/queue/produce.go:25` already narrows the seam once inside the package, to a closure that carries `qtx`, which an interface method cannot express without a second constructor per transaction |
| **Fire the enqueue from a callback after the batch commits** | It breaks all three atomicity facts at once. `delivery.message_id` references `message (id)`, so the callback must run after commit and can then fail on its own. ADR-0064 computes a message once at the cause and `UNIQUE (message_id, channel_id)` makes one routed pair one row, so a failed callback loses the routing permanently. ADR-0164's terminate rolls a job's staged work back, and a post-commit callback would route a message for a batch that rolled back |
| **Have the delivery runner poll `message` for unrouted rows instead of taking an enqueue at all** | It replaces an exact seam with a scan of the message corpus, and it needs a second piece of state — routed or not — that the schema does not have. The `Delivery` row is that state today, and it is written by the act that creates the message. The poll would also route a message whose batch is still in flight, because the runner cannot see the worker's open transaction |
| **Make an unwired enqueuer refuse the job, the way `WithCT` does** | The CT seams refuse because a clean, empty CT batch is a false absence: it says certificate transparency named nothing. A message asserts nothing about the estate. ADR-0064 puts the fact in the timelines, and the message reports it. An unwired producer loses a notification and creates no false reading, so a refusal here would stop the lower layer measuring for no evidential gain |
| **Put a `produceMessages` flag on `NewWorker` so the unwired case is explicit** | The unwired case is already a zero value in a function field. A flag adds a second thing to keep in step with the field and creates the exact state limb 2's Consequences records as a defect — production on, route absent. The measurement-only build needs no declaration, and asking it to make one is the whole thing limb 4 refuses |
| **Wire the enqueuer inside `internal/queue`'s own package initialisation** | It would need the import, so it is the first alternative under another name. It would also move the wiring out of the composition root, where `cmd/worker/main.go:88` reads `VERGE_DEV` and `:97` passes it down, and no other binary could then build a worker that does not message |
| **State the rule as a clause on [ADR-0140](./0140-a-network-seam-is-a-runtime-parameter-the-caller-supplies-never-a-build-tag-and-never-a-hardcoded-client.md)** | ADR-0140 rules the network seam, and its subject is adapter substitution so that a test reaches no network. `delivery.EnqueueForMessage` touches no network. It writes two queries on a transaction. Filing this there would put a layering rule inside a document about test isolation, and it would say nothing about limb 3, which is the limb this seam turns on |
| **State the rule as a clause on [ADR-0149](./0149-a-consumer-takes-the-data-layer-interface-it-calls-and-the-seam-not-the-package-is-the-unit.md)** | ADR-0149 rules how wide a consumer's data-layer slice is, and its §5 exempts a package that owns a transaction boundary, naming `queue.Worker` and `delivery.Runner`. Width and direction are different questions. This seam is already inside that exemption, so the clause would land on the one paragraph of ADR-0149 that says the rule does not reach here |
| **Repair the `P0.7` token the deleted comment cited** | [`comment-policy.md`](../spec/comment-policy.md) §4.7 measures `P0.7` in the third dangling family and rules it unrepairable by token. Nothing in `docs/` or `CONTEXT.md` carries the identifier, so there is no live source to re-cite. §4.7 directs the sweep to keep the reason uncited and record the gap, which is what #1316 did |
