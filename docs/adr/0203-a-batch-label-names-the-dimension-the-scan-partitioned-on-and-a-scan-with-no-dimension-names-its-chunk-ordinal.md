# ADR-0203: a Batch label names the dimension the Scan partitioned on, and a Scan with no dimension names its chunk ordinal

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1319 ADR gaps: internal/scan (2/3)](https://github.com/winniel123/verge-asm/issues/1319), gap 4
- **PR that deleted the comment:** [#1318](https://github.com/winniel123/verge-asm/pull/1318)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0005](./0005-scan-execution-model.md), which makes **one queue job one `Batch`** and partitions along a dimension the source keeps enumerability over. It fixes the unit and the boundary. It never says what the resulting `Batch` is called
- **Rests on:** [ADR-0129](./0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md) §5, which rules that `edge-fanout` has **no vantage dimension**, because the default certificate is not a function of vantage. That is the ADR which removed the dimension every other fan-out names
- **Sibling of, and not ruled by:** [ADR-0127](./0127-the-address-scope-range-cap-has-no-ceiling-a-large-scope-is-priced-not-gated.md), which makes the address fan-out stream rather than materialise. It removed the slice a builder could otherwise index for a label. It rules memory, never naming

## Context

[`internal/scan/edgefanout.go:24`](../../internal/scan/edgefanout.go) carried this text until #1318
deleted it:

```go
	// Chunk is the job's ordinal within its tick, and it names the Batch. The Scan has
	// no vantage and no seed to key a Batch on, and the fan-out is streamed, so there is
	// no slice index to read one from.
```

The sweep kept one compressed line at the same site, uncited under §4.7:

```go
	Chunk     int // names the Batch: this Scan has no vantage or seed to key one on
```

**No ADR and no SPEC states what a `Batch` label must be.** ADR-0005 rules the unit and the
partition. Nothing rules the name.

### Every label in the tree names a domain dimension, except one

`internal/queue` builds the label at the enqueue site, one `fmt.Sprintf` per Scan:

| Scan | Label | Site | The dimension it names |
| --- | --- | --- | --- |
| `dns` | `scan:%d:vantage:%d` | [`queue.go:321`](../../internal/queue/queue.go) | The `Vantage` |
| `http-identity` | `scan:%d:vantage:%d` | [`httpidentity.go:38`](../../internal/queue/httpidentity.go) | The `Vantage` |
| `tls-acceptance` | `scan:%d:vantage:%d` | [`tlsacceptance.go:79`](../../internal/queue/tlsacceptance.go) | The `Vantage` |
| `hot` | `scan:%d:vantage:%d:addr:%s` | [`hot.go:169`](../../internal/queue/hot.go) | The `Vantage` and the one address |
| `cold` | `scan:%d:vantage:%d:addr:%s` | [`cold.go:68`](../../internal/queue/cold.go) | The `Vantage` and the one address |
| `zone` | `scan:%d:seed:%d` | [`zone.go:98`](../../internal/queue/zone.go) | The `Seed` |
| `ct` | `scan:%d:seed:%d` | [`crtsh.go:388`](../../internal/queue/crtsh.go) | The `Seed` |
| `ct-tail` | `scan:%d:log:%s` | [`cttail.go:307`](../../internal/queue/cttail.go) | The CT log |
| **`edge-fanout`** | **`scan:%d:edges:%d`** | [`edgefanout.go:39`](../../internal/queue/edgefanout.go) | **Nothing in the domain. The chunk's ordinal** |

`edge-fanout` has no dimension left to name, and the reason is a decision rather than an oversight:

- **No vantage.** ADR-0129 §5 removed it. `enqueueEdgeFanoutJob` passes `VantageID: pgtype.Int8{}`.
- **No seed.** Its population is `Estate.EdgeFanoutPopulation()` — the custody-extension candidates
  **and** the addresses of declared address scopes, merged and de-duplicated. It is a derived set
  across every `Seed`, so no one `Seed` names a chunk.
- **No address.** The port tiers name one address because ADR-0005 puts one address per `Batch`.
  `edge-fanout` measures up to `EdgeFanoutAddressesPerJob` of them.
- **No slice index.** ADR-0127 makes the fan-out stream. `BuildEdgeFanoutJobs` takes an
  `iter.Seq[netip.Addr]` and nothing holds the population, so there is no materialised position to
  read.

### What a label actually is, and what it is not

The label is easy to over-read, so the measurement matters.

**It is not a database key.** [`db/migrations/18803_measurement_batch.sql`](../../db/migrations/18803_measurement_batch.sql)
declares `batch` with an identity `id` and the columns `scan_id`, `dispatch_id`, `vantage_id`,
`kind`, `outcome`, `offers`, `recorded_scope` and `created_at`. **There is no label column.** The
`Batch` row is keyed on `id`.

**It is not persisted at all.** The label rides `wire.JobSpec.Batch` into the prober, and every leaf
stamps it onto `wire.Observation.Batch` on the way out. `toObservationParams` in
[`internal/queue/pure.go`](../../internal/queue/pure.go) copies `Facet`, `Subject`, `Discriminator`,
`Value` and the batch **id**. It never copies `o.Batch`. No column in the schema holds it, and no
code outside `internal/wire` reads it back.

**What it does is correlate raw job output.** The label appears in the job spec that
`ProberTranscript.SentScope` holds verbatim, and again on every NDJSON line in `Stdout`. A person
reading the raw record — the corpus [`docs/spec/raw-job-output.md`](../spec/raw-job-output.md)
governs — uses it to tie the lines to the unit of work that produced them.

**No label in the tree is unique over time.** `scan:7:vantage:2` is the same string on every `dns`
tick, and `scan:7:edges:3` is the same string on every `edge-fanout` tick. Uniqueness comes from the
`Dispatch` the job belongs to, which is on the job row and on the `Batch` row. The ordinal restarting
at 0 each tick is therefore not a defect of the ordinal. It is the property every label already has.

### The ordinal is unstable in a way a dimension is not

`EdgeFanoutPopulation` yields the extension candidates first, then walks each declared address scope,
skipping duplicates, non-globally-reachable addresses and excluded ones. So one new resolution that
adds one candidate shifts **every** later chunk boundary by one address. `edges:3` in this tick and
`edges:3` in the next tick hold different addresses whenever the candidate set moved.

`vantage:2` does not behave that way. It denotes the same `Vantage` in every tick, so it survives
comparison across ticks. That difference is the whole reason a rule is needed rather than a
convention.

## Decision

> **A `Batch` label names the dimension its `Scan` partitioned on, so that a reader of the raw job
> output can say what the unit was. Where a `Scan` has no domain dimension to name, the label carries
> the partition's own ordinal within the tick. An ordinal names a position in one enumeration and
> denotes nothing outside it, so nothing may join on it, and nothing may compare the same ordinal
> across two ticks.**

### 1. The label is a name for the raw record, never a key

It is not stored, not indexed, and not unique over time. Its one job is to let a person holding a
transcript say which job spec produced which NDJSON lines. Every rule below follows from that and
from nothing else.

### 2. A Scan with a dimension names it

`vantage:%d`, `seed:%d`, `log:%s`, and `addr:%s` beside a vantage for the one-address tiers. The
dimension named is the one ADR-0005's partition cut along, so the label states the partition rather
than restating the job id.

### 3. A Scan with no dimension names its chunk ordinal, and the ordinal is per-tick

`edge-fanout` is the first such `Scan` and its label is `edges:%d`. The counter starts at 0 in each
call to the builder, which is once per dispatch. The `Dispatch` supplies the uniqueness, exactly as
it does for every other label. A synthetic global counter or a random token would buy uniqueness the
label does not need and lose the one thing an ordinal gives — a reader can see that `edges:3` is the
fourth chunk of this tick.

### 4. An ordinal denotes nothing, and that is a prohibition

Because the population is streamed and derived, a chunk boundary is a function of enumeration order.
So:

- Nothing joins on a label. There is no column to join on, and §1 says why that is correct.
- Nothing compares the same ordinal across two ticks. Two chunks with one ordinal are two different
  address sets whenever the candidate population moved, which is on any tick a name resolved to a new
  address.
- No derivation reads a label. The drift engine reads `Batch` rows and their `recorded_scope`. The
  scope names the addresses. The label names none of them.

### 5. What this rule does not decide

- **The chunk size.** [ADR-0200](./0200-a-jobs-scope-is-sized-so-its-leaf-finishes-inside-one-probe-bracket.md)
  rules that, on a different ground: the leaf must finish inside one probe bracket. This rule takes
  the boundary as given and names the result.
- **The `Batch` row's identity.** The `batch` table is keyed on `id` and this ADR does not touch it.
- **Whether `edge-fanout` should have a dimension.** ADR-0129 §5 already ruled that it has none.

## Consequences

- **This ADR changes no Go code.** All nine label forms comply. The rule it states was true and
  unwritten.
- **The `edge-fanout` label was reachable only through the deleted comment.** Without it a reader
  seeing `edges:3` had no way to learn the number is an enumeration position and not a stable
  denotation. That is the reading this ADR exists to prevent.
- **A new `Scan` gains a question with an answer.** *What does this Scan partition on* — and if
  nothing, the label carries the ordinal, which is legitimate and bounded by §4.
- **`CONTEXT.md` gains nothing.** The `Batch` entry already rules the unit — *"one source, executed
  once, against one scope, from one vantage"* — and a label is not part of that key. The `Dispatch`
  entry already carries the prohibition against a comparison keyed on a run, which §4 restates one
  level down.
- **No [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
  withdrawal is owed.** ADR-0005 is unchanged and states nothing about naming.
- **Nothing enforces §4.** No test asserts that no derivation reads a label. The schema enforces it
  by accident, because there is no column to read, and that accident is worth stating: if a label
  column were ever added, §4 would become a rule discipline has to carry.
- **A defect is exposed and it is small. It ships as its own ticket if it is worth doing at all.**
  `jobView.Batch` in [`cmd/web/scans.go`](../../cmd/web/scans.go) is named for a label and holds
  `BatchOutcome`, the outcome word. The run log line therefore renders `kind · state · vantage ·
  completed`. Nothing is wrong with the output. The field name invites a later reader to think the
  console shows a label, which it does not, because nothing stores one.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Give the chunk a globally unique token — a UUID or a monotonic counter** | Buys uniqueness the label does not need, because the `Dispatch` already supplies it and no label in the tree is unique over time. It costs the one property the ordinal has: a reader of a transcript can see which chunk of the tick they are holding, and can tell that `edges:0` came before `edges:7` |
| **Key the label on the chunk's first address — `edges:addr:198.51.100.1`** | Reads as a denotation and is not one. The first address of a chunk is an artefact of enumeration order, so the same address heads a different chunk on the next tick, and two ticks' `edges:addr:198.51.100.1` cover different sets. It is §4's prohibited reading built into the label |
| **Hash the chunk's address set into the label** | Makes the label a content key, which is a real thing but not this thing. `recorded_scope` already records the scope by content (ADR-0025), so the hash would be a second, weaker copy of a record that already exists — and it would break the one correlation the label is for, because a person cannot type or recognise it |
| **Give `edge-fanout` a synthetic vantage so it can use `vantage:%d`** | ADR-0129 §5 removed the vantage dimension deliberately: the default certificate is not a function of vantage, and vantage-varying fan-out is anycast, out of v1. A synthetic vantage would put a dimension in the label that the measurement does not have, and `enqueueEdgeFanoutJob` would have to write a `vantage_id` that means nothing |
| **Leave the label empty for a Scan with no dimension** | `cmd/web/scans.go` already branches on `j.Batch != ""`, and an empty label makes the raw transcript unreadable at exactly the Scan whose jobs are hardest to tell apart — many chunks per tick, all the same kind, none carrying a vantage |
| **Record the label in a `batch.label` column so the console can render it** | Adds a stored copy of a name that denotes nothing, and immediately invites the join §4 forbids. Everything the console needs is already on the row: the `Scan`, the `Dispatch`, the kind and the recorded scope |
