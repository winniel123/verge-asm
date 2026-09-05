# ADR-0204: a Scan kind and a leaf kind are separate namespaces, and a shared string is a coincidence

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1319 ADR gaps: internal/scan (2/3)](https://github.com/winniel123/verge-asm/issues/1319), gap 5
- **PR that deleted the comment:** [#1318](https://github.com/winniel123/verge-asm/pull/1318)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0084](./0084-a-scan-is-a-cadence-over-an-exchange-and-an-uncovered-facet-has-no-currency-bound.md), which makes a `Scan` a **cadence over an exchange**. A `Scan` is therefore named for the coverage it provides, and a leaf for the exchange it runs. This ADR states the consequence that the two names are separate
- **Rests on:** [ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md), which makes a facet's cadence the cadence of its exchange. It is why `tls-acceptance` needed a `Scan` of its own, and so why one string ended up on both sides
- **Bounded by:** [ADR-0027](./0027-a-source-may-admit-without-observing.md), which admits a source that observes nothing. The worker-read `Scan`s it licenses run **no leaf at all**, and §3 records that their two namespaces coincide because one of them is empty

## Context

[`internal/scan/httpidentity.go:17`](../../internal/scan/httpidentity.go) carried this text until
#1318 deleted it:

```go
// HTTPIdentityKind is the DB Scan kind this file dispatches. Unlike `tls-acceptance`
// — whose Scan and leaf share one name — the `http-identity` Scan decides the
// `http-identity` facet by dispatching the `http-exchange` leaf, so the Scan kind and
// the leaf kind differ, exactly as the port tiers' `hot`/`cold` dispatch
// `connect-outcome`.
```

The sweep kept one compressed line at the same site, uncited under §4.7:

```go
// A Scan names the facet it decides; the leaf it dispatches names the exchange it runs.
```

**No ADR, no SPEC and no `CONTEXT.md` entry states that the two are separate namespaces.** The
`Derivation` entry in `CONTEXT.md` lists the leaves. The `Scan` entry lists the Scans. Neither says
what the relation between the two lists is.

### The two lists, and where they meet

`scan.kind` holds a Scan kind. `queue_job.kind` and `wire.JobSpec.Kind` hold a **leaf** kind, which
`cmd/prober/main.go` switches on. Nine Scan rows ship.

| `scan.kind` | `JobSpec.Kind` | The leaf that runs | Same string? |
| --- | --- | --- | --- |
| `dns` | `resolution-walk` | `internal/measure/resolutionwalk` | no |
| `hot` | `connect-outcome` | `internal/measure/connectoutcome` | no |
| `cold` | `connect-outcome` | `internal/measure/connectoutcome` | no |
| `http-identity` | `http-exchange` | `internal/measure/httpexchange` | no |
| `tls-acceptance` | `tls-acceptance` | `internal/measure/tlsacceptance` | **yes** |
| `edge-fanout` | `edge-fanout` | `internal/measure/edgefanout` | **yes** |
| `zone` | `zone` | **none** — the worker reads it | not applicable |
| `ct` | `ct` | **none** — the worker reads it | not applicable |
| `ct-tail` | `ct-tail` | **none** — the worker reads it | not applicable |

Two leaves are in neither column. `wildcard-discrimination` has a prober arm and no `Scan` dispatches
it — membership composes it (ADR-0086). `blanket-discrimination` is composed by `connect-outcome`
and is not dispatched at all (ADR-0104 §2).

So: six Scans dispatch four leaves, two of the six share a string with the leaf they dispatch, three
Scans dispatch nothing, and two leaves belong to no Scan. **No total function exists in either
direction.**

### The three constants say it three ways

`internal/scan` declares the Scan kind three different ways for the three non-worker-read Scans, and
each way encodes a different relation:

```go
const HTTPIdentityKind  = "http-identity"          // its own literal: the strings differ
const TLSAcceptanceKind = tlsacceptance.Kind       // an alias: the strings coincide
const EdgeFanoutKind    = edgefanout.Kind          // an alias: the strings coincide
```

A reader meeting `HTTPIdentityKind = "http-identity"` beside `EdgeFanoutKind = edgefanout.Kind`
cannot tell from the declarations whether the alias is meaningful or whether the literal is an
oversight. The deleted comment was the only place that answered.

### The cost of the confusion is silent, twice

**Compare a job kind to a Scan constant and the branch never fires.**
[`internal/queue/availability.go:24`](../../internal/queue/availability.go) is the live case:

```go
if !vantageValid || kind != resolutionwalk.Kind {
	return availabilityUnchanged
}
```

The `kind` there is the batch's kind, which is the leaf's. Written as `scan.DNSKind` the condition
would be true for every job, the function would always return `availabilityUnchanged`, and a `Vantage`
whose resolver went down would never be marked unavailable. ADR-0108's whole availability derivation
would stop, and nothing would error.

**Stamp a Scan kind onto a job spec and the job completes having measured nothing.**
`cmd/prober/main.go`'s `default` arm answers an unknown kind with one bare observation:

```go
default:
	// A kind with no leaf yet still answers job-spec-in / NDJSON-out (ADR-0001).
	return wire.EncodeObservation(stdout, wire.Observation{Batch: spec.Batch, Kind: spec.Kind})
```

That is not an error. The prober exits 0. `toObservationParams` then skips the line, because it
carries no facet. `Worker.complete` inserts a `Batch` with `Outcome: completed` and
`RecordedScope: job.AttemptedScope` — **the full attempted scope** — and zero observations. A
completed batch asserting coverage over a scope nothing measured is precisely the manufactured-absence
failure ADR-0005 and ADR-0001 are built to prevent, and it arrives through a one-word mistake in a
builder.

## Decision

> **A `Scan` kind and a leaf kind are separate namespaces. A `Scan` kind names the coverage the
> operator configures — the facet a cadence covers. A leaf kind names the exchange the prober runs.
> Where two names are equal they are equal by coincidence, and neither namespace may be derived from
> the other. A `Scan` kind belongs in `scan.kind` and in the dispatcher's own switch. A leaf kind
> belongs in `wire.JobSpec.Kind`, in `queue_job.kind`, in `batch.kind`, and in every comparison the
> worker or the recording side makes.**

### 1. Which namespace a value belongs to is fixed by where it is read

- **The Scan namespace** is read by `Dispatcher.fanOut` and `fanOutAtomic`
  ([`internal/queue/queue.go`](../../internal/queue/queue.go)), by `hotLagGateApplies`, and by
  `GetScanByKind`. Every one of those reads `db.Scan.Kind`.
- **The leaf namespace** is read by `cmd/prober/main.go`, by `availabilityAfterOutcome`, by
  `toEdgeFanoutRows`, and by anything reading `job.Kind` or `batch.kind`.

A comparison is correct when the constant's namespace matches the field's. Nothing else decides it,
and the compiler checks none of it, because both namespaces are `string`.

### 2. A shared string is a coincidence and never an identity

`tls-acceptance` and `edge-fanout` name a `Scan` and a leaf with one text. That happened because each
of those measurements needed a cadence of its own (ADR-0028, ADR-0129 §6) and there was one obvious
name for the thing. It is not a rule, and nothing may be built on it. In particular no code may
convert one namespace to the other, in either direction, by string equality.

### 3. A worker-read Scan has no leaf, so its namespaces coincide trivially

`zone`, `ct` and `ct-tail` run no prober. ADR-0027 licenses a source that admits without observing,
and ADR-0106 puts the CT poll on a `Scan`. Their job specs carry the **Scan** kind, and
`Worker.process` branches on `spec.Kind == scan.ZoneKind` before it reaches the prober. That is legal
because the leaf namespace is empty for them, and it is the one case where a Scan kind legitimately
appears in `JobSpec.Kind`.

### 4. The alias constant is the correct expression where the strings coincide

`const EdgeFanoutKind = edgefanout.Kind` states in code that these two are held equal on purpose. It
is better than a second literal, because a second literal would let the two drift apart with no
signal. §Consequences names the cost it carries.

### 5. What this rule does not decide

- **How a Scan is named.** ADR-0084 and ADR-0028 rule that a `Scan` is a cadence over an exchange.
  The name follows from the coverage.
- **How a leaf is named.** `CONTEXT.md`'s `Derivation` entry rules it — *"a leaf is named for what
  it decides, never for the artefact that ships it"*.
- **Which leaf a Scan should dispatch.** That is the measurement design, held per Scan.
- **The leaf version.** ADR-0021 and the golden corpus rule that, and a Scan has no version.

## Consequences

- **This ADR changes no Go code.** Every comparison in the tree is on the correct side today.
- **The alias couples a shipped `scan.kind` row to a leaf constant, and that is a live hazard this
  ruling names.** `EdgeFanoutKind = edgefanout.Kind` feeds `GetScanByKind`, and
  [`db/migrations/24500_edge_fanout_scan.sql`](../../db/migrations/24500_edge_fanout_scan.sql) inserts
  the literal `'edge-fanout'`. So a rename of the leaf's `Kind` silently changes the string the
  dispatcher looks the `Scan` up by, `GetScanByKind` returns no row, and the `Scan` stops firing with
  no compile error and no failing test. The same holds for `tls-acceptance`. **§4 keeps the alias, and
  a rename of either leaf constant must ship a migration.**
- **`CONTEXT.md`'s `Scan` entry is out of date, and it is not fixed here.** It reads *"there are
  seven and only two are port tiers"* and it names `hot`, `cold`, `tls-acceptance`, `zone`, `dns`,
  `ct` and `edge-fanout`. Nine `Scan` rows ship. `http-identity`
  ([`db/migrations/23400_http_identity_scan.sql`](../../db/migrations/23400_http_identity_scan.sql))
  and `ct-tail`
  ([`db/migrations/23900_ct_log_cursor.sql`](../../db/migrations/23900_ct_log_cursor.sql)) are
  missing from the count and from the list. That is a documentation defect this ADR found while
  building its table, and it **ships as its own ticket**. The table above is the current record until
  it is fixed.
- **A new Scan gains a checklist question.** *Which leaf does this Scan dispatch, and is that leaf's
  kind the string the job spec carries?* The answer for a worker-read Scan is *none*, and §3 covers
  it.
- **Nothing enforces this.** Both namespaces are `string` and no type separates them. A named type
  per namespace would let the compiler check every comparison. It is a real option and it is a
  refactor across `internal/scan`, `internal/queue`, `internal/wire` and `cmd/prober`, so it is not
  taken here.
- **No [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
  withdrawal is owed.** This ADR supersedes no mechanism and contradicts no ADR.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Make the two namespaces one — rename every Scan to its leaf** | `hot` and `cold` both dispatch `connect-outcome`, so the merge is not injective and one of the two port tiers would lose its name. The names also carry different information to the operator: `hot` and `cold` state a cadence and a port set, which is what the operator configures, while `connect-outcome` states an exchange, which the operator never chooses |
| **Derive the leaf kind from the Scan kind with a lookup table** | The table is the dispatcher's `switch` and it already exists. A second copy as data would have to stay in step with the builders, and it would have to hold `zone`, `ct` and `ct-tail` mapping to nothing, which is exactly the case a lookup gets wrong |
| **Give both namespaces one named Go type — `type Kind string`** | Makes the two look interchangeable at every call site while changing nothing about the fact that they are not. A shared type would make `availabilityAfterOutcome(kind, ...)` compile against `scan.DNSKind` just as it does today, so it buys no check and adds a false signal |
| **Two named types, one per namespace** | The right shape and a large refactor. It touches every constant in `internal/scan`, `wire.JobSpec.Kind`, `db.ClaimJobRow.Kind`, the prober's switch and every test that builds a spec from a literal. It is worth doing and it is not a documentation change |
| **Forbid a Scan and a leaf from sharing a string, and rename `edge-fanout`'s Scan** | Renames a shipped `scan.kind` row to remove a coincidence that costs nothing on its own. The hazard is the **alias constant's** coupling to a migration literal, and a rename does not remove that. It moves it |
| **Write the rule as a comment on each of the three constants** | It was, and #1318 removed it under the comment policy, correctly: a rule that binds `internal/queue`, `cmd/prober` and two migrations is not recoverable from the three declarations it was written on, and three copies drifted into three different wordings |
