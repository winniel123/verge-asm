# ADR-0149: a consumer takes the data-layer interface it calls, and the seam, not the package, is the unit

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1272 ADR gaps: internal/release (sweep 4/14)](https://github.com/winniel123/verge-asm/issues/1272), gap 3
- **PR that deleted the comment:** [#1271](https://github.com/winniel123/verge-asm/pull/1271)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Sibling of, and not ruled by:** [ADR-0140](./0140-a-network-seam-is-a-runtime-parameter-the-caller-supplies-never-a-build-tag-and-never-a-hardcoded-client.md). That ADR rules the **network** seam and says in §5 that a network interface stays with its consumer. This ADR rules the **data** seam. The two share a shape and neither contains the other
- **Rests on:** [ADR-0001](./0001-stack-and-runtime.md), which chooses `sqlc` and `pgx` over an ORM. It picks the generator and says nothing about how wide a consumer's slice of the generated surface is

## Context

`internal/release/release.go:39` carried this, until #1271 deleted it:

```go
// Store is the narrow slice of the data layer the Checker needs: the one config
// read that carries the opt-out flag, and the release-cache write. *db.Queries
// satisfies it. It exposes nothing else, so a bug in the checker can touch only
// the release cache.
```

The comment states a rule that binds code outside its own file, and nothing on disk states it. That
is #1272's gap 3.

**The rule is what most of the repo already does.** Eighteen consumer-side data-layer
interfaces sit across `internal` and `cmd`. Seventeen are between one and nine methods wide.

| Consumer | Interface | Methods |
| --- | --- | --- |
| `internal/release/release.go:18` | `Store` | 2 |
| `internal/queue/hotlag.go:17` | `HotLagStore` | 1 |
| `internal/queue/addressexclusion.go:8` | `AddressExclusionStore` | 1 |
| `internal/queue/hot.go:45` | `EstateStore` | 3 |
| `internal/queue/edgefanout.go:148` | `EdgeFanoutStore` | 5 |
| `internal/retention/retention.go:35` | `Store` | 3 |
| `cmd/worker/remoterouter.go:23` | `remoteVantageStore` | 1 |
| `cmd/web/custodycensus.go:32` | `custodyCensusStore` | 9, over two embedded `queue` interfaces |
| `cmd/web/api_auth.go:16` | `apiAuthStore` | 4 |
| … eight more | | 1 to 5 |
| **`cmd/web/handlers.go:22`** | **`store`** | **178** |

The generated `internal/db.Querier` names **251**. `cmd/web`'s `store` names **178** of them, 71% of
the whole data layer, in one interface reached through one `server` field by 38 non-test files. It is
the only consumer that is not narrow, and it is why this rule needs a scope call rather than a
restatement.

**The console aggregate has already lost the guarantee it was meant to give.** The bearer path needs
`GetPersonalTokenByHash` and `UpdatePersonalTokenLastUsed`. Neither is among the 178. Rather than
take them as a parameter, `cmd/web/api_auth.go:16` declares a four-method `apiAuthStore` and reaches
it by asserting on the aggregate at `cmd/web/api_auth.go:24`:

```go
st, ok := s.store.(apiAuthStore)
```

`cmd/web/api_auth.go:31` then carries a fail-closed branch, a log line and a comment saying a wired
store always satisfies it. The concrete `*db.Queries` does. **The compiler cannot see that, because
the field is typed as the wide interface.** A parameter of type `apiAuthStore` would have been
checked at build time and the branch, the log line and the comment would not exist. The aggregate
grew past what any one handler needs and then could not answer the one question a narrow interface
answers for free.

**The cost is also carried in test.** `cmd/web/handlers_test.go` hand-writes **172** methods on
`fakeStore` inside 2,873 lines. Every query any handler in `cmd/web` adds costs a line on the
interface and a method on that fake, charged to every handler in the package.

## Decision

> **A consumer declares, in its own package, the data-layer interface naming the queries the code
> behind that seam actually calls, and takes that. It never takes `*db.Queries` and never takes the
> generated `Querier` as a supplied dependency. `*db.Queries` satisfies the declaration and is not
> named in it. The unit is the seam — the constructor parameter, function parameter or struct field
> at which the value is supplied — and not the package, so a package holding handler groups that call
> disjoint query sets declares one interface per group.**

### 1. The width is what the code behind the seam calls

Not what the package might one day call, and not what a related consumer already declares. A method
on the interface that no reachable call site invokes is removed. This is the whole content of the
rule: the interface is a statement of reach, so a bug behind that seam touches the rows it names and
no others.

`internal/release.Store` is the clean case. Two methods: the instance-config read that carries the
opt-out flag, and the release-cache write. The `Checker` cannot reach an account, a seed or a scan,
and no reviewer has to check that it does not.

### 2. The interface lives in the consumer, and there is no shared one

The declaration sits in the package that calls it. Seventeen consumer-side declarations therefore
repeat some method signatures, and the repetition is kept, for the same reason ADR-0140 §5 keeps
three `Doer` declarations: a shared interface puts one widening decision above every consumer, and a
consumer that never asked for a query gains reach the day someone else needs it.

Reuse across consumers happens by **embedding a narrow interface another consumer already owns**, not
by widening a common one. `cmd/web/custodycensus.go:32` embeds `queue.EdgeFanoutStore` and
`queue.AddressExclusionStore` and adds three methods of its own. That is nine, and the import edge
already existed.

### 3. The unit is the seam, not the package

A package-level reading would grade `cmd/web`'s `store` compliant, because the package as a whole
does call 178 queries. It would also grade every future aggregate compliant, which is the failure
mode this ADR exists to close.

"What the consumer needs" is measured at the value that is supplied. A handler that renders the
custody census needs nine queries whether or not it shares a package with the login form.
`cmd/web` proves this shape twice already, in `custodyCensusStore` and `apiAuthStore`.

### 4. The console `store` is a defect, and it is not exempt

**The scope call, stated plainly: a request-handler aggregate that serves a whole console earns no
exemption. `cmd/web/handlers.go`'s 178-method `store` is a defect under this rule.**

Three reasons, none of them about size.

**The exemption's premise is false in this code.** It would say the console's real dependency is the
whole console's data. But 178 is not the console's data. The bearer path's two queries are outside
it, `cmd/web` narrows twice inside the same package, and no handler calls more than a handful. The
aggregate matches no unit that exists.

**The aggregate has already been paid for and delivered nothing.** §Context's type assertion is the
measurement. The one property a wide interface could offer — every handler can reach every query
without a signature change — is exactly the property that pushed a capability check to run time and
put a fail-closed branch on the request path.

**Nothing about a console makes it a boundary.** `internal/queue` is a larger consumer of the data
layer than `cmd/web` and declares six interfaces of one to five methods. An HTTP mux is not a
different kind of thing from a job worker.

**Size is explicitly not the reason to exempt it, and it is not a reason to fix it here either.**
Splitting `store` into per-handler-group interfaces touches `cmd/web/handlers.go`, 38 handler files
and a 2,873-line test fake. That is a large production change with its own review, and it ships as
its own ticket.

### 5. What this rule does not reach

- **A package that owns a transaction boundary.** `internal/queue.Worker` (`worker.go:100`) and
  `internal/delivery.Runner` (`runner.go:67`) take a `*pgxpool.Pool` and build `*db.Queries`
  themselves, because they call `w.q.WithTx(tx)` (`worker.go:523`). `WithTx` returns `*db.Queries`
  and no consumer-side interface can express it. Owning the pool is a different act from being
  handed a store, and this rule binds the second.
- **An unexported helper threading an open transaction.** `foldOne(ctx, qtx *db.Queries, …)` and its
  many siblings in `internal/queue` are not supplied dependencies. They pass along a handle the
  package already owns, inside the transaction that owns it. Narrowing them would declare an
  interface per helper and buy nothing.
- **`internal/db`.** The generated package is `sqlc`'s output. `Querier` stays exactly as emitted.
  This rule is about who takes it, never about what it contains.
- **Whether the interface is exported.** `internal/release.Store` is exported and
  `cmd/web/api_auth.go`'s `apiAuthStore` is not. Both are correct. Go's visibility rules decide that,
  not this ADR.

## Consequences

- **`cmd/web/handlers.go`'s `store` is a known violation and is not fixed here.** It ships as its own
  ticket: split it at the handler group, drop the `apiAuthStore` type assertion for a typed
  parameter, and shrink `fakeStore` to match. **This ADR changes no Go code.**
- **Seventeen consumers are already compliant.** Closing this gap costs them nothing.
- **A new consumer has a document to be held to at review.** Before this, a reviewer asking for a
  narrow store was stating a preference and the console was the counter-example on hand.
- **[`comment-policy.md`](../spec/comment-policy.md) §4.5 gains one clause.** It cites the
  178-method `store` as a live measurement for a comment-placement ruling, and that measurement stays
  true. It gains a pointer so a reader does not take the aggregate's permanence for granted.
- **[ADR-0140](./0140-a-network-seam-is-a-runtime-parameter-the-caller-supplies-never-a-build-tag-and-never-a-hardcoded-client.md)
  gains one cross-reference line.** Its §5 states the consumer-side narrowness rule for the network
  seam. A reader landing there now learns the data seam is ruled separately.
- **`CONTEXT.md` gains nothing.** Interface width is a code-structure term, not a domain term.
- **Nothing enforces this.** No check fires on a wide interface, so review carries the rule. A count
  is not a gate: `custodyCensusStore` is nine methods and correct, and a nine-method interface under a
  consumer that calls three is wrong. The question is what the code behind the seam calls, and only a
  reader can answer it.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Take the generated `db.Querier` directly** | 251 methods at every seam. It hands `internal/release`'s two-query checker the whole estate, deletes the one fact the interface exists to state, and re-couples every consumer to `sqlc`'s output, so a query added for one package widens the reach of all of them. It also makes a test fake impossible to write by hand, so every test would need a real database or a generated mock |
| **One shared hand-written interface — a `store` package every consumer imports** | Puts a single widening decision above every consumer. The first package that needs a new query widens the seam under packages that never asked, and the reach a narrow interface states is lost on the first commit. It also creates an import edge between packages that share nothing else. ADR-0140 §5 refuses the same shape for `Doer`, and §2 here refuses it for the same reason |
| **Exempt a request-handler aggregate that serves a whole console** | Its premise is false in this code: two queries the bearer path needs are outside the 178, and `cmd/web` already declares two narrow stores of its own. The exemption also grades the one violating consumer as passing, so it fails on the only case where the rule has work to do |
| **State the rule at package scope — "a package declares the interface it needs"** | Grades `cmd/web`'s `store` compliant, because the package really does call 178 queries. It is the wording that lets any future aggregate through, and it is why §3 fixes the unit at the seam |
| **Fix `cmd/web`'s `store` on this ADR's own branch** | Mixes a production change across 38 handler files and a 2,873-line test fake into a docs change, and buries the code review under the ADR review |
| **A `commentlint` or `go vet` check that fails a data-layer interface above N methods** | Not decidable from the declaration. `custodyCensusStore`'s nine methods are all called and a three-method interface under a consumer that calls one is a violation. A threshold check passes the second and fires on the first, which trains reviewers to suppress it |
| **A section on [ADR-0140](./0140-a-network-seam-is-a-runtime-parameter-the-caller-supplies-never-a-build-tag-and-never-a-hardcoded-client.md)** | That ADR's subject is the network seam, and its Decision is about a caller supplying a runtime adapter. The data-layer question is not adapter substitution — `*db.Queries` is the only implementation in production — but reach. Filing it there would put a rule about blast radius inside a document about injection |
| **A clause on [ADR-0001](./0001-stack-and-runtime.md)** | ADR-0001 chooses `sqlc` and `pgx` over an ORM. It rules the generator, and interface width is a property of the consumers, none of which it names |
