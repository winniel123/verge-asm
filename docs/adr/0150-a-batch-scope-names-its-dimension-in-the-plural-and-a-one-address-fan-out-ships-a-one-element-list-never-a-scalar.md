# ADR-0150: a Batch scope names its dimension in the plural, and a one-address fan-out ships a one-element list, never a scalar

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1312 ADR gaps: internal/scan 3/3](https://github.com/winniel123/verge-asm/issues/1312), gap 1
- **PR that deleted the comment:** [#1313](https://github.com/winniel123/verge-asm/pull/1313)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0005](./0005-scan-execution-model.md), which sets **one address per `Batch`**. It rules the execution unit. It says nothing about the field shape that unit leaves on the wire
- **Rests on:** [ADR-0127](./0127-the-address-scope-range-cap-has-no-ceiling-a-large-scope-is-priced-not-gated.md), which names the streamed per-address fan-out and states that the `Batch` unit is unchanged. It also rules no field shape

## Context

[`internal/scan/hot.go:21`](../../internal/scan/hot.go) carried this text until #1313 deleted it:

```go
// HotJob is one queue job the hot Scan produces: one Vantage, ONE Custody-
// admitted address (one address per Batch, ADR-0005/ADR-0127), and the
// `verge-core` TCP/UDP sets. Only the TCP pairs are probed; the UDP pairs travel
// so the Batch records them in scope (open or closed for TCP; recorded-not-probed
// for UDP). Addresses holds the single address as a one-element slice so the wire
// `connect-outcome` scope stays a list, unchanged.
```

The sweep kept one compressed line at the same site, uncited under §4.7:

```go
// The single admitted address rides as a one-element slice so the wire scope stays a list.
```

**The rule binds code outside `hot.go`, and no ADR states it.** ADR-0005 rules the partition. One
`Batch` carries one address, so the scope record stays honest. ADR-0127 confirms the unit is
unchanged under a chunked fan-out. Neither rules what the scope field looks like once the fan-out
has left one address in a job.

**Two producers build the same scope type.** `internal/scan/hot.go:63` and
[`internal/scan/cold.go:89`](../../internal/scan/cold.go) both write
`Addresses: []string{a.Unmap().String()}`. Both then marshal
[`connectoutcome.Scope`](../../internal/measure/connectoutcome/run.go), whose `Addresses` field
carries the tag `json:"addresses"` with no `omitempty`.

**Four readers outside `internal/scan` decode that field as a list.**

| Reader | Site | What it does with the list |
| --- | --- | --- |
| The `connect-outcome` leaf | [`internal/measure/connectoutcome/run.go:43`](../../internal/measure/connectoutcome/run.go) | Ranges the list and crosses it with the TCP port set |
| The certificate step | [`internal/measure/connectoutcome/certificate.go:170`](../../internal/measure/connectoutcome/certificate.go) | Ranges the list to size and fill a per-address verdict map |
| The #773 re-gate | [`internal/queue/scopegate.go:28`](../../internal/queue/scopegate.go) | Unmarshals one union shape across every leaf kind and admits only the addresses the scope names |
| The drift feed | [`cmd/web/driftfeed.go:240`](../../cmd/web/driftfeed.go) | Unmarshals `addresses` as `[]json.RawMessage` and counts it for the batch label |

The re-gate and the drift feed never see the producer. They read the recorded scope out of the
`recorded_scope` column. A scalar there is not a compile error at either site. The re-gate would
admit nothing and drop every legitimate observation. The drift feed would silently render no label.

**The union shape is what makes the field name load-bearing.** `scopegate.go` unmarshals one
`scopeShape` for every leaf kind. `internal/measure/edgefanout/run.go:16` also names its dimension
`addresses`, and it carries up to `EdgeFanoutAddressesPerJob = 50` members. One decoder serves both
because the name has one shape everywhere it appears.

## Decision

> **A `Batch` scope field names a measurable dimension, and the name and the shape follow the
> dimension rather than the cardinality of one `Batch`. A tier that fans out to one member per
> `Batch` ships a one-element list. It never collapses the field to a scalar, and it never renames
> it. ADR-0005's partition is an execution decision, and it must not reach the wire contract.**

Five limbs.

### 1. The dimension names the field, and one `Batch`'s cardinality does not

`addresses`, `names`, `services` and `targets` are plural because the dimension admits many members.
A tier chooses how many members one `Batch` carries. That choice is ADR-0005's, and it is free to
change. The field name and the JSON shape are the contract, and they do not move with it.

### 2. One admitted address ships as `["203.0.113.1"]`

The hot tier and the cold tier both admit exactly one address per `Batch`, and both wrap it. This is
the rule the deleted comment stated, and it now binds both producers rather than one.

A later tier that admits one address inherits the same rule. A later tier that admits many needs no
change, because the shape already carries many.

### 3. The empty scope is written as an explicit empty slice

[`EmptyHotScope`](../../internal/scan/hot.go) and
[`EmptyColdScope`](../../internal/scan/cold.go) pass `[]string{}` rather than a nil slice. The tag
carries no `omitempty`, so a nil slice would render `null` and an explicit empty slice renders `[]`.
A dead-lettered `Batch` records that it covered nothing, and `[]` states that. A caller that builds
an empty scope by hand uses an explicit empty slice for the same reason.

### 4. The rule binds both scope surfaces, not only the dispatched one

Each job carries two scope documents, and they share the field.

- The dispatched scope is `JobSpec.Scope`, which `JobSpec` marshals from `connectoutcome.Scope`.
- The recorded scope is `AttemptedScope`, which lands in the `recorded_scope` column.

The re-gate compares an observation against the recorded scope, so the two documents must agree on
the shape. A change to one alone breaks that comparison.

### 5. A reader may not infer cardinality from the shape

A one-element list is not a promise that the list holds one element. Every reader listed in the
Context ranges the list or counts it. None of them indexes element zero. A reader that assumed one
member would break when the fan-out changed, which is the coupling this ADR removes.

## Consequences

- **[`internal/scan/hot.go:21`](../../internal/scan/hot.go) gains this ADR's citation.** It is the
  one site that states the ground, so it is the one site that carries the citation.
- **[`internal/scan/cold.go`](../../internal/scan/cold.go) gains nothing.** Its comment at line 82
  states the one-address partition and cites ADR-0005, which is correct and is a different rule. A
  citation at every site that wraps an address would be one rule copied across sites that state no
  ground for it.
- **[`connectoutcome.Scope`](../../internal/measure/connectoutcome/run.go) gains nothing.** It
  declares the shape and asserts no ground.
- **No production behaviour changes.** Every site already has the shape this ADR states. The change
  is that the shape now has a record.
- **[ADR-0005](./0005-scan-execution-model.md) and
  [ADR-0127](./0127-the-address-scope-range-cap-has-no-ceiling-a-large-scope-is-priced-not-gated.md)
  gain nothing, and [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
  does not fire.** Neither ADR states a clause about the wire shape, so this ADR narrows, bounds and
  withdraws nothing in either.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** `Batch` already has an entry there. This is a
  decision about a wire field, not a term.
- **This ADR does not rule whether a shape change needs a leaf `Version` bump.**
  `internal/measure/wildcarddiscrim/run.go:24` records that a scope change may justify a bump under
  ADR-0021. Whether that reaches `connect-outcome` was not verified here and stays open.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Collapse the field to a scalar `address` for the hot and cold tiers** | Four readers outside `internal/scan` decode `addresses` as a list, and two of them read it out of the database rather than from the producer. Neither fails to compile. The #773 re-gate would admit nothing and drop every legitimate observation, and the drift feed would render no label |
| **Keep the list but drop it to a scalar in the recorded scope alone** | Limb 4's failure exactly. The re-gate compares an observation against the recorded scope, so the two documents must carry one shape |
| **Add a scalar `address` beside the list** | Two spellings of one fact, and nothing decides which one a reader trusts. It also breaks the single `scopeShape` union, which serves every leaf kind from one set of field names |
| **Rely on ADR-0005 to cover the shape** | ADR-0005 rules the partition and the scope record's honesty. It never mentions the field's JSON shape, and ADR-0127 adds only that the unit is unchanged. A reader holding ADR-0005 alone can conclude that one address per `Batch` licenses a scalar |
| **Write `null` for an empty scope and let the reader treat it as empty** | The tag carries no `omitempty`, so a nil slice already renders `null`. `EmptyHotScope` and `EmptyColdScope` pass an explicit empty slice instead, so a dead-lettered `Batch` states that it covered nothing rather than saying nothing |
| **Leave the rule as one uncited line in `hot.go`** | The state #1312 recorded. The rule binds four readers in three other packages, and the only statement of it sat in the file whose behaviour is the least likely to break from it |
