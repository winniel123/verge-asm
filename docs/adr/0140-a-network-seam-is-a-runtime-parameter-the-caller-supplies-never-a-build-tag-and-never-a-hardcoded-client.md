# ADR-0140: a network seam is a runtime parameter the caller supplies, never a build tag and never a hardcoded client

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1283 ADR gaps: internal/auth, internal/measure/blanketdiscrim, internal/proposer, internal/seed](https://github.com/winniel123/verge-asm/issues/1283)
- **Recorded independently by four more sweeps:** [#1272](https://github.com/winniel123/verge-asm/issues/1272) (`internal/release`), [#1279](https://github.com/winniel123/verge-asm/issues/1279) (`internal/measure/httpexchange`, `internal/measure/edgefanout`), [#1282](https://github.com/winniel123/verge-asm/issues/1282) (`internal/delivery`, `internal/report`), [#1296](https://github.com/winniel123/verge-asm/issues/1296) (`internal/measure/connectoutcome`)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8. The five records come from five sweep sessions that could not see each other. §8.10 directs dedup **by rule sentence**, and this ADR is what that dedup produced
- **Bounded by:** [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md), which fixes *where* the measurement corpus runs and does not say *how* the leaf is reached
- **Sibling, for the data seam:** [ADR-0149](./0149-a-consumer-takes-the-data-layer-interface-it-calls-and-the-seam-not-the-package-is-the-unit.md) rules how wide a consumer's data-layer interface is; §5 below rules only the network one
- **Reaches, and is not satisfied by:** [ADR-0124](./0124-a-backup-carries-data-and-no-secret-and-updating-is-guided-not-self-applied.md) §2, whose release-feed check is the one outbound call in the repo that breaks this rule

## Context

Five sweep sessions deleted five comment blocks in five packages, and each filed the same rule in its
own words. None could see the others. #1279 and #1296 both flagged the suspected duplicate, and
neither could confirm it.

| Record | Package | The sentence, as filed |
| --- | --- | --- |
| [#1272](https://github.com/winniel123/verge-asm/issues/1272) | `internal/release` | "A package that makes an outbound network call exposes it **behind an interface**, so a test is driven by a fake and no test ever touches the live network" |
| [#1279](https://github.com/winniel123/verge-asm/issues/1279) | `internal/measure/httpexchange`, `internal/measure/edgefanout` | "A measurement leaf puts its network side **behind an interface**, so one code path is driven by the production adapter in production and by a scripted in-process fake in test" |
| [#1282](https://github.com/winniel123/verge-asm/issues/1282) | `internal/delivery`, `internal/report` | "The delivery core carries no database and no live network, so … routing are exercised in full by a **fake HTTP doer**" |
| [#1283](https://github.com/winniel123/verge-asm/issues/1283) | `internal/proposer` | "All proposer network I/O goes through an **injected HTTP seam**, so no proposer path touches the real network under test" |
| [#1296](https://github.com/winniel123/verge-asm/issues/1296) | `internal/measure/connectoutcome` | "The `Connector` is **a parameter rather than a build-time seam**, so the corpus drives the leaf's verdict logic with no network and no container" |

They are one rule. But the five wordings do not agree on the mechanism, and the disagreement decides
code twice.

**First: four of the five wordings are satisfied by the one package that breaks the rule.**
`internal/release/fetcher.go:18` declares `Doer` with a single `Do` method. That is "behind an
interface" on its face. `NewHTTPFetcher` at `internal/release/fetcher.go:27` then takes a URL and no
client. It writes `&http.Client{Timeout: feedTimeout}` into the unexported field at
`internal/release/fetcher.go:30`. The interface is declared and never opened. Its only caller,
`cmd/worker/main.go:166`, has nothing to pass. So `HTTPFetcher.Latest` has no test, and the non-200
refusal, the `maxFeedBytes` cap over a hostile feed and the empty-`tag_name` refusal are all
unpinned. **The declaration that exists to make those testable is the thing that
makes their absence look answered.**

**Second: only #1296 says the seam is a runtime parameter, and it says so in a triage note.** A pair
of files under `//go:build` tags — the real adapter in one, a scripted one in the other — satisfies
"behind an interface". It yields a test that touches no network. It is also a different design, and
nothing on disk refuses it.
[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s Decision fixes *where* the corpus
runs: "Hermetically, against an **in-process scripted peer**: no network, no containers, no fixture
images". Its Consequences then call the corpus "a build-time artefact per leaf". A session reading
that for a mechanism can take it as a licence for a build-time seam.

The code has already answered both questions everywhere but one place, and has never written the
answer down.

## Decision

> **A package that reaches the network accepts its network adapter as a runtime parameter from its
> caller — a constructor argument or a call argument. Declaring the interface is half the rule and
> satisfies nothing on its own: the package must also let the caller supply the value. The seam is
> never a build tag, and never a client the package builds for itself and keeps.**

### 1. The seam is a parameter, and the composition root chooses

The value crosses the boundary at run time, in an ordinary argument position, and the caller picks
it. The composition roots are `cmd/web` and `cmd/worker`, and they pick the production adapter. A
test picks a fake. The golden corpus picks a scripted peer. One code path serves all three.

That is what the repo already does at every seam but one.

| Package | The interface | Where the caller supplies it |
| --- | --- | --- |
| `internal/delivery` | `Doer` (`runner.go:26`) | `NewRunner(pool, doer, …)` (`runner.go:76`) |
| `internal/report` | `delivery.Doer` (`notify.go:58`) | `NewNotifyRunner(pool, doer, …)` (`notify.go:65`), fed from `cmd/worker/main.go:134` |
| `internal/proposer` | `Doer` (`proposer.go:33`) | `DefaultRegistry(doer)` (`proposer.go:50`), then `NewARIN` (`arin.go:20`) and `NewCAIDA` (`caida.go:25`) |
| `internal/queue` | `CTFetcher` (`crtsh.go:34`) | `WithCT` (`crtsh.go:118`), `WithCTTail` and `WithCTVerify`, fed from `cmd/worker/main.go:94-95,210` |
| `internal/remoteexec` | `Conn` (`conn.go:20`) | `Probe(ctx, conn, …)` (`probe.go:61`) and `Inspect(ctx, conn)` (`probe.go:31`) |
| `internal/release` | `Fetcher` | the `Checker` takes it, and `release_test.go` drives `fakeFetcher` |
| `internal/measure`, six leaf packages | `Connector`, `Handshaker`, `Exchanger`, `Peer`, `Enumerator` | seven entrypoints: `RunWithConnector` (`connectoutcome/run.go:68`), `RunExchange` (`connectoutcome/certificate.go:142`), `RunWithHandshaker` (`edgefanout/run.go:39`), `RunWithExchanger` (`httpexchange/run.go:41`), `RunWithPeer` (`resolutionwalk/run.go:40`), `RunWithPeer` (`wildcarddiscrim/run.go:70`), `RunWithEnumerator` (`tlsacceptance/run.go:59`) |

Each measure package also exports a plain `Run(spec, w)` that builds the production adapter and calls
the `RunWith…` form. That is the composition root for a leaf, and it is the only place the real
adapter is named.

### 2. Declaring the interface satisfies nothing

`internal/release` declares `Doer` and closes it in the same file. Under this ADR that is a
violation, not a partial pass. The rule's test is one question: **can a caller supply a different
adapter?** For `HTTPFetcher` the answer is no, and the price is the untested `Latest` body above.

This clause is the whole reason the ADR exists. Four of the five records grade `internal/release` as
compliant.

### 3. The seam is never a build tag

Three reasons, in the order they bind.

**It changes what "one code path" means.** #1279 and #1296 both state the property as *one* code path
driven two ways. A build tag makes two code paths and tests one of them. The production path then
compiles only in a configuration no test runs, which is the failure the seam exists to prevent.

**It cannot express the leaf's real shape.** A build tag is chosen once per binary. The measure leaves
choose the adapter per call. `connectoutcome` alone takes a `Connector` in `RunWithConnector`, and a
`Connector` plus a `Handshaker` in `RunExchange`. A corpus row scripts a fresh peer for every row. No
tag can do that.

**It hides the choice from the reader.** A parameter names the adapter at the call site. A tag moves
that fact into the build invocation, where `go vet ./...` and `go test ./...` do not reach it and no
reviewer reads it.

`internal/measure` puts no build tag on any adapter today. This clause records the state the code is
already in.

### 4. The rule binds the consumer, and an adapter that builds its own client owes an adapter test

A production adapter is the value that gets injected, so something inside it builds a real client.
That is not a violation. `delivery.NewHTTPDoer` (`runner.go:30`), `queue.NewHTTPCTFetcher`
(`crtsh.go:46`) and `remoteexec.Dial` (`conn.go:49`) each construct their own transport, and each is
supplied by a caller one level up.

The bound on that permission: **an adapter carrying its own logic owes its own test against a
loopback server.** `TestHTTPCTFetcher` and `TestHTTPCTFetcherDoesNotFollowRedirect`
(`internal/queue/crtsh_test.go:36,75`) drive `HTTPCTFetcher` through `httptest`.
`TestHTTPDoerRefusesRedirects` (`internal/delivery/delivery_test.go:219`) pins the redirect refusal
directly. `internal/release/fetcher.go` fails both routes at once. It neither accepts a client nor
carries a test, and that is what makes it the one violation rather than one more adapter.

### 5. The interface stays with the consumer and is not consolidated

`Doer` is declared three times — `internal/delivery/runner.go:26`, `internal/proposer/proposer.go:33`,
`internal/release/fetcher.go:18` — with the same single method. The repetition is kept. Each consumer
declares the narrowest interface it needs. A shared package therefore cannot widen a seam under a
consumer that never asked for it. Three packages that share nothing else also gain no import edge. The
one exception holds the line: `internal/report` takes `delivery.Doer` because those two already share
the delivery path.

### 6. What this rule does not reach

- **The transport's configuration.** Timeouts, redirect policy and the `custody` dial guard are ruled
  elsewhere and are untouched here. Redirect policy is
  [ADR-0196](./0196-no-outbound-http-client-this-product-builds-follows-a-redirect-and-an-unfollowed-3xx-admits-nothing.md),
  which rules that no outbound client this product builds follows one.
- **`internal/measure`'s hermeticity requirement.** ADR-0021 owns it. This ADR names the mechanism
  that delivers it and does not restate the requirement.
- **Whether an adapter reaches the network at all.** ADR-0124 §2 keeps the release check opt-out and
  air-gap-safe. This ADR rules on how the client arrives, not on whether the call is made.

## Consequences

- **`internal/release/fetcher.go` is a known violation and is not fixed here.** `NewHTTPFetcher` must
  take a `Doer`, and `HTTPFetcher.Latest` must gain a test. That is a production signature change with
  its own review, and it ships as its own ticket. **This ADR changes no Go code.**
- **Every other package in §1 is already compliant.** The five records describe behaviour that exists,
  so closing this gap costs one constructor signature and one test file.
- **The five `adr-gap` records collapse to one rule.** §8.10's dedup has a document to point at, and a
  triage session no longer has to decide whether five wordings are one rule.
- **ADR-0021 gains a cross-reference row and one qualified clause.** A session reading it for the
  corpus mechanism now lands here, rather than inferring a build-time seam from "build-time
  artefact". ADR-0085 and ADR-0124 gain a cross-reference each on the same grounds.
- **`docs/spec/golden-corpus.md` gains nothing.** It already says "Run it hermetically against an
  in-process scripted peer", which names the mechanism correctly and misleads nobody.
- **`CONTEXT.md` gains nothing.** "Network seam" is a code-structure term, not a domain term, and the
  glossary rules the estate.
- **Nothing enforces this.** No check fires on a hardcoded client, so review carries the rule. The
  `internal/release` defect measures how long that survives unwritten. The seam was declared, the
  client was hardcoded, and five sweeps later nobody had noticed.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A build tag over two files — the real adapter in one, a scripted one in the other** | It makes two code paths and tests one, which is the exact failure the seam prevents. It cannot be chosen per call, so it cannot express `RunExchange`'s two adapters or a corpus that scripts a fresh peer per row. It moves the choice into the build invocation, where no reviewer and no `go test ./...` sees it |
| **Declare the interface and let the package build its own client — the declare-only reading** | This is what `internal/release` does, and it is why this ADR exists. It reads as compliant, admits no substitute, and leaves the HTTP body untested while looking tested. A seam a caller cannot reach is documentation, not a seam |
| **State the rule as "put the network behind an interface" and stop** | The wording four of the five records used. It grades the one violating package as passing, so it fails on the only case where a rule has work to do |
| **Consolidate the three `Doer` declarations into one shared package** | Creates an import edge between three packages with nothing else in common, and puts one widening decision above three consumers. The narrow consumer-side interface is what stops a seam growing methods its consumer never needed |
| **Ban a production adapter from constructing its own `http.Client`** | The adapter is the injected value, so something has to build the transport. Banning it pushes `NewHTTPDoer`'s dial guard and `NewHTTPCTFetcher`'s redirect refusal into the composition root, where they are duplicated per caller and lost on the first new one |
| **A `commentlint` rule or a `go vet` check that fails on a hardcoded client in a package that declares an interface** | Not decidable from the declaration. A legitimate adapter (`NewHTTPDoer`, `NewHTTPCTFetcher`) has the same shape as `NewHTTPFetcher`, and the difference is whether a caller supplies it. A check that fires on all three trains reviewers to suppress it |
| **Fix `internal/release` on this ADR's own branch** | Mixes a production signature change and a new test into a docs change, and buries the code review under the ADR review |
| **A section on [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)** | ADR-0021's subject is the version vector and the measurement corpus. This rule binds `internal/delivery`, `internal/report`, `internal/proposer`, `internal/queue`, `internal/remoteexec` and `internal/release`, none of which are measurement leaves. An amendment there would state a repo-wide rule inside a document scoped to one binary |
| **Six per-package ADRs, one per record** | Six documents stating one sentence, and the next package that dials out matches none of them. §8.10 asks for dedup by rule sentence, and this is one sentence |
