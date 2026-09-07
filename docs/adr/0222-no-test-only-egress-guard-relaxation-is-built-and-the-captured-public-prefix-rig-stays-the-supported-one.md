# ADR-0222: no test-only `EgressGuard` relaxation is built, and the dial-control seam stays unexported. ~~The captured-public-prefix rig stays the supported one.~~ **ADR-0225 supersedes that last clause.**

- **Status:** Accepted (§3 superseded and §4 discharged by [ADR-0225](./0225-the-declared-address-scope-that-admitted-a-target-rides-the-job-and-the-egress-guard-reads-it.md))
- **Date:** 2026-09-07
- **Ticket:** [#1598 EgressGuard refuses every local prefix, so a leaf measurement needs a captured public prefix to test against](https://github.com/winniel123/verge-asm/issues/1598)
- **Upstream:** [#1115](https://github.com/winniel123/verge-asm/issues/1115) and PR [#1562](https://github.com/winniel123/verge-asm/pull/1562), which built the rig. [#1572](https://github.com/winniel123/verge-asm/issues/1572) carried finding 5 and left it unfiled
- **Superseded in part by:** [ADR-0225](./0225-the-declared-address-scope-that-admitted-a-target-rides-the-job-and-the-egress-guard-reads-it.md) — [#1610](https://github.com/winniel123/verge-asm/issues/1610) rules that the egress guard reads the declared address scope that admitted the target. ADR-0225 supersedes §3 below and discharges §4 below. This ADR's own reopening condition names that event. §1 and §2 stand unchanged, and ADR-0225 §4 obeys this ADR's §1 rather than amending it
- **Rests on:** [ADR-0121](./0121-the-operator-declared-recursive-resolver-is-trusted-and-exempt-from-the-discovered-authority-egress-guard.md), which rules that the egress guard depends on trust origin and not on the address. This ADR applies that same test to a test rig and refuses the exemption, because a test harness declares no realm
- **Read with:** [ADR-0079](./0079-authority-presupposes-denotation-a-non-globally-reachable-address-is-probed-only-inside-a-declared-realm.md), whose declared-address-scope route ~~the guard closes at the socket. §4 states that contradiction and refuses to settle it here~~ **the guard now reads at the socket ([ADR-0225](./0225-the-declared-address-scope-that-admitted-a-target-rides-the-job-and-the-egress-guard-reads-it.md)). §4 below stated that contradiction and refused to settle it here. ADR-0225 settles it, and §4 is discharged**
- **Read with:** [ADR-0166](./0166-a-verge-dev-build-is-a-capture-affordance-and-every-gate-it-opens-is-unreachable-in-a-released-build.md), whose residual-risk paragraph records what a review-held containment costs. §2 chooses a compiler-held one instead

## Context

`custody.EgressGuard` (`internal/custody/egressguard.go:10`) returns a `net.Dialer.Control`
function. It refuses any address `IsNonGloballyReachable` marks, before the socket opens
(`internal/custody/egressguard.go:22`).

### The four install sites, confirmed by symbol on `38eec12`

| Package | Symbol | Site |
| --- | --- | --- |
| `internal/measure/connectoutcome` | `NetConnector.Connect` | `leaf.go:85` |
| `internal/measure/connectoutcome` | `NetHandshaker.Handshake` | `tls.go:143` |
| `internal/measure/httpexchange` | `NetExchanger.Exchange` | `exchange.go:152` |
| `internal/measure/tlsacceptance` | `NetEnumerator.Handshake` | `enumerate.go:132` |

Three of the four construct the guard inline and hold no seam. `NetConnector` and `NetHandshaker`
carry one field each, `Timeout`. `NetEnumerator` carries one field, `Timeout`.

`internal/measure/resolutionwalk` is a fifth dial path and is not one of these four. ADR-0121 gives
it two dialers on a stated ground. This ADR does not reach it.

### One site already holds the seam this ticket asks for

`httpexchange.NetExchanger` carries an **unexported** `control` field
(`internal/measure/httpexchange/exchange.go:119`). `Exchange` reads it and installs
`custody.EgressGuard` when it is `nil` (`exchange.go:150-153`). One test sets it
(`exchange_net_test.go:33`), so an `httptest` server on loopback is reachable.

So the option the ticket calls *an injected policy the production path cannot supply* is not a
proposal. It is built, at one of the four sites. The seam and the guard arrived in the same commit,
`9a8c3de`.

### The two needs inside finding 5

Finding 5 of [`hot-scan-wall-clock-and-emitted-rate.md`](../research/hot-scan-wall-clock-and-emitted-rate.md)
says a supported test affordance would remove the need to capture a public prefix. Two different
needs sit inside that sentence, and only one of them is a test affordance.

**Need A is a Go test of one leaf's dial path.** A test binds a listener on loopback and calls
`Connect`, `Handshake` or `Exchange` against it. `httpexchange` already serves this need.

**Need B is an end-to-end run of the shipped binaries.** §2.2's rig runs `worker -trigger hot` and
the `prober` binary against a target, through the dispatcher, the queue and Postgres. That is what
#1115 measured. **No test-only seam can serve need B**, because a shipped binary must not carry a
seam that opens the egress boundary. A package-level hook is reachable from a Go test alone. It
never reaches `worker`.

Finding 5 asks for need A and offers it as a replacement for need B's rig. It is not one.

### The mismatch the investigation found

The custody gate admits a declared non-globally-reachable address, and the guard then refuses it.
Measured on `38eec12` with an `Estate` holding one address scope of `10.0.0.0/24`:

| Call | Result |
| --- | --- |
| `Estate.Derive(10.0.0.5)` | `operator` |
| `Estate.MayProbe(10.0.0.5, ClassUnverified)` | `true` |
| `Estate.MayProbe(10.0.0.5, ClassInternal)` | `true` |
| `Estate.MayProbe(10.0.0.5, ClassInternet)` | `false` |
| `EgressGuard("connectoutcome")("tcp", "10.0.0.5:80", nil)` | refusal |

The path is live. `fanOutHot` enumerates every declared address scope
(`internal/queue/hot.go:34`, `internal/queue/hot.go:135-150`). `BuildHotJobs` applies `MayProbe`
at dispatch (`internal/scan/hot.go:53`). The shipped `local` vantage presents no address, so
`exposure.VerifyClass` derives `unverified` (`internal/exposure/exposure.go:94`), and
`vc.IsInternet()` is false. The job reaches `NetConnector.Connect`, and the guard refuses it.

ADR-0079's worked table states the opposite outcome for that exact case: *"Operator declares
`10.0.0.0/24`; instance is the only prober → **Probed, unchanged.** The internal estate keeps its
measurements."*

This is a question about the shipped product, not about a test rig. §4 files it as
[#1610](https://github.com/winniel123/verge-asm/issues/1610) and refuses to settle it here.

> **#1610 is settled by [ADR-0225](./0225-the-declared-address-scope-that-admitted-a-target-rides-the-job-and-the-egress-guard-reads-it.md), and the mismatch
> above is repaired.** The table and the trace stay as the record of the tree on `38eec12`. Read
> them as a measurement of that tree, not as a live defect.

## Decision

> **No test-only relaxation of `EgressGuard` is built. A dial-control seam at a measurement leaf is
> an unexported struct field whose zero value installs the guard, and the Go compiler is what keeps
> it out of a shipped binary. The three sites without a seam gain none today, because no test needs
> one. ~~§2.2 of the hot-scan note is the supported rig for an end-to-end measurement, and it stays
> so.~~ §2.2 of the hot-scan note is one supported rig for an end-to-end measurement, and a declared
> address scope reaches a local target through the production path
> ([ADR-0225](./0225-the-declared-address-scope-that-admitted-a-target-rides-the-job-and-the-egress-guard-reads-it.md)).**

Four limbs.

### 1. The containment is the unexported field, and it holds before the seam exists

State the containment first, as the ticket requires.

A dial-control seam is an unexported field on the leaf's dialer type. Three properties follow, and
the compiler enforces all three.

- **A package outside the leaf cannot set it.** A composite literal naming an unexported field of
  another package fails to compile. `internal/queue`, `cmd/worker` and `cmd/prober` are all outside.
- **A production caller gets the guard by default.** The zero value is `nil`, and `nil` installs
  `custody.EgressGuard`. Forgetting the guard is not reachable. A caller must act to lose it, and
  the act does not compile.
- **The set of callers that could set it is one package, and it is enumerable by `grep`.**
  `httpexchange`'s only setter is a test file.

This is stronger than ADR-0166's `VERGE_DEV` containment. ADR-0166 records that its own claim is
*"held there by review alone"*. This claim is held by the compiler.

`internal/measure/egressseam_test.go` pins it. `TestDialControlSeamStaysUnexported` reads all four
dialer types by reflection and fails on any exported field carrying the
`net.Dialer.Control` signature. `TestOutsideCallerGetsTheGuard` builds a `NetExchanger` from
outside the package and asserts the refusal.

### 2. No seam is added to the three sites that lack one

`NetConnector`, `NetHandshaker` and `NetEnumerator` gain nothing here.

The ground is that no test needs one. A seam with no consumer is a widening of the egress boundary
that buys nothing today and must be defended forever. §1's rule is what governs the seam when a
test does need one, and adopting it then is a one-field change that the pinning test already
covers.

### 3. ~~§2.2 is the supported rig, and it is named as such~~ **§2.2 is one supported rig — SUPERSEDED by [ADR-0225](./0225-the-declared-address-scope-that-admitted-a-target-rides-the-job-and-the-egress-guard-reads-it.md)**

> **Superseded here, at the site that specifies it** ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)) by
> [#1610](https://github.com/winniel123/verge-asm/issues/1610) · [ADR-0225](./0225-the-declared-address-scope-that-admitted-a-target-rides-the-job-and-the-egress-guard-reads-it.md).
> This ADR's Reopening condition names that exact event and rules this section *"superseded rather
> than amended"*. The event happened on 2026-09-07.
>
> **A declared address scope now reaches a local target through the production path.** The egress
> guard reads the declared scope that admitted the target, so a declared `127.0.0.0/22` reaches a
> loopback listener, and `internal/measure/connectoutcome/realm_net_test.go` demonstrates it with no
> test-only seam. Read alone and in the present tense, the struck sentence sends a session to capture
> a public prefix before it can run a leaf measurement end to end. That capture is now optional.
>
> **What survives.** §2.2's rig works, and it stays the rig for an end-to-end measurement against a
> globally reachable target. Its capture procedure and its containment argument are untouched. What
> is withdrawn is the word *the*.

[`hot-scan-wall-clock-and-emitted-rate.md`](../research/hot-scan-wall-clock-and-emitted-rate.md)
§2.2 holds the captured-public-prefix rig. ~~It is the supported way to run a leaf measurement
end to end against a target the operator controls.~~ **It is one supported way to run a leaf
measurement end to end against a target the operator controls. A declared address scope is the
other way.** The note is amended to say so, and finding 5 is amended to record this ADR.

The cost is real and it is accepted. A reader needs a public allocation, or an understanding of why
holding `192.88.100.0/22` on a local bridge is contained. §2.2 states both. **A reader who declares
an address scope over the target pays neither.**

### 4. ~~The gate-versus-guard mismatch is a production question, and it is filed~~ **The mismatch was filed — DISCHARGED**

> **DISCHARGED by [#1610](https://github.com/winniel123/verge-asm/issues/1610) · [ADR-0225](./0225-the-declared-address-scope-that-admitted-a-target-rides-the-job-and-the-egress-guard-reads-it.md),
> and recorded here at the site that files it** ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).
> The mismatch is repaired. The declared address scope that admitted a target rides the job, the
> egress guard reads that scope at the socket, and an address inside no declared scope stays refused
> at every one of the four install sites. ADR-0225 decided the question on ADR-0079's and ADR-0121's
> grounds, which is what this section asked a later session to do.
>
> **Nothing below is withdrawn.** The refusal to settle an egress boundary inside a ticket about a
> test rig was correct, and the reasoning records why. Read this section as the history of a filing,
> not as an open question.

The mismatch in the Context is not repaired here, and it is not repaired by any test affordance.

This ADR refuses to settle it for one reason. A repair changes which destinations a shipped binary
sends packets to. ADR-0079 spent its whole length on that question and reached its answer through a
denotation argument. Reopening it inside a ticket about a test rig would decide an egress boundary
as a side effect of a testing convenience. That is the exact shape ADR-0079's *"Probe it more
gently"* row refuses.

It is filed as [#1610](https://github.com/winniel123/verge-asm/issues/1610). A later session decides
it on ADR-0079's and ADR-0121's grounds, not on this one's. ADR-0121 is the near precedent, and its
ruling is that the guard depends on **trust origin, not address**. A declared address scope and a
declared resolver are the same class of input.

## Consequences

- **Nothing new is reachable in a shipped binary.** This ADR builds no affordance. The one seam
  that exists is unchanged, and the four install sites are unchanged.
- **`internal/measure/egressseam_test.go` is new.** It holds two tests and no production code. It
  fails when a dial-control field becomes exported at any of the four sites.
- **A session that adopts the seam at one of the three remaining sites does not need a new ADR.**
  §1 permits it and §2 states the bar. The field must be unexported, and its zero value must
  install the guard.
- **The note gains two amendments** and no new section.
- **`CONTEXT.md` gains nothing.** No domain term moves.
- **The `resolutionwalk` dial path is untouched.** ADR-0121 governs it.
- ~~**The mismatch stays live until [#1610](https://github.com/winniel123/verge-asm/issues/1610)
  lands.** An operator who declares a private address scope today gets `resolution` and
  `dns-record` at full aperture and no connect measurement. Nothing in the console says why.~~
  **WITHDRAWN here, at the site that states it** ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).
  #1610 landed as [ADR-0225](./0225-the-declared-address-scope-that-admitted-a-target-rides-the-job-and-the-egress-guard-reads-it.md). An operator who
  declares a private address scope gets a connect measurement over the addresses that scope
  contains.

## Reopening condition

This ADR is reopened by exactly one event:
**[#1610](https://github.com/winniel123/verge-asm/issues/1610) rules that a declared,
non-internet-class target passes the guard.** A leaf measurement against a declared `10.0.0.0/24`
is then a supported production configuration. The rig follows it at no extra cost, and this ADR's
§3 is superseded rather than amended.

> **The event happened on 2026-09-07.** [#1610](https://github.com/winniel123/verge-asm/issues/1610) ·
> [ADR-0225](./0225-the-declared-address-scope-that-admitted-a-target-rides-the-job-and-the-egress-guard-reads-it.md) rules exactly this, and §3 above carries
> the supersession. The condition is spent, and it does not fire a second time.

It is **not** reopened by a session that finds §2.2's rig awkward, slow, or hard to reproduce. That
is the cost §3 accepts, and the ADR records it so that a later session does not re-argue it.

It is **not** reopened by a build-tagged proposal. The first row below disposes of that shape, and
the argument depends on no fact that a later tree can change.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A build-tagged test-only relaxation**, so `EgressGuard` returns `nil` under a tag | `go test ./...` builds the untagged tree, so the required `test` check would gate one of two behaviours and never run the other. The guard is the repo's SSRF backstop, and the variant CI does not exercise is the variant that permits egress. It also forks `internal/custody`, whose stated job is to be the one seam every site calls. ADR-0151's table already refuses a second dial path on that ground: *"a second dial path is a second place the guard can be forgotten"*. Finally, the containment is a build flag, which is the review-held shape ADR-0166 records as its residual risk. An unexported field is compiler-held and strictly stronger |
| **An exported dial-control field, or a `WithControl` constructor**, on each of the four types | The compiler then permits any caller to replace the guard, and the unreachability claim rests on review of four constructors plus every future one. This is the shape §1 exists to refuse. It also widens the guard from a fixed table read into an arbitrary predicate, which destroys the single-reading-rule property ADR-0079 built: *"there is no seam where two readings could drift apart"* |
| **An environment variable, in the shape of `VERGE_DEV`** | It is reachable in a shipped binary by construction, which the ticket's constraint forbids outright. ADR-0166 already prices this shape and its own Consequences call the result *"a full authentication bypass"* prevented by one unset variable. Siting a second such variable inside the egress boundary is worse than the first |
| **Relax the guard for loopback alone**, keeping RFC 1918 refused | Loopback is the address the guard most needs to refuse. ADR-0079's worked table names the prober measuring itself, and `169.254.169.254` retrieving cloud metadata into `http-identity`, as the two cases the refusal exists for. A carve-out for the prober's own host is a carve-out for the first of them |
| **Add the seam to all three remaining sites now**, so a later test needs no change | Three widenings with no consumer. Each must be defended for the life of the repo, and the first reader to find one will reasonably ask what test uses it. §1's rule makes the later adoption a one-field change, so nothing is saved by acting early |
| **Repair the gate-versus-guard mismatch in this ticket** | It changes which destinations a shipped binary sends packets to. That is ADR-0079's subject, decided through a denotation argument this ticket did not run. Deciding an egress boundary as a side effect of a testing convenience is the shape ADR-0079 refuses in its *"Probe it more gently"* row. §4 files it instead |
| **Close #1598 with no ADR**, since the rig works | The ground would then live nowhere, and finding 5's sentence would stand unanswered in the note. A later session reading *"a supported test affordance would remove the need to capture a public prefix"* would build one, because the sentence reads forward as an instruction. ADR-0058's test applies: a superseded sentence that would cause a competent session to build the thing is not withdrawn |
