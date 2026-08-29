# ADR-0111: a Span cites the Batch that folded it, opening the estate-wide drift feed

- **Status:** Accepted
- **Date:** 2026-08-23
- **Ticket:** [#288 Drift: estate-wide classified transition feed + batch grouping](https://github.com/winniel123/verge-asm/issues/288)
- **Origin:** [#283 T7 Drift](https://github.com/winniel123/verge-asm/issues/283) (V2 console migration, map #275)

## Context

The `/drift` screen is the product's thesis — *what moved since last time, grouped by
batch*. It was ported verbatim from the design-system example (ADR-0110), but its
transition timeline rendered the empty-state because the store exposed no estate-wide,
batch-grouped change feed. The two span reads it had — `ListAllOpenSpans` (the current
open set) and `ListSpansForSubject` (one subject's history) — answer *current state* and
*one subject's timeline*, never *what changed across the estate, grouped by the batch
that changed it*.

A transition is not a stored object (ADR-0007): it is the adjacency between two
consecutive spans, derived on read. So the feed is a **read** — the missing piece was
never a transition table, it was the ability to say *which Batch caused a given span
open or close*, so the derived transitions can be grouped by batch the way the screen's
whole composition is organised.

The `span` table carried no such link. A span records the period a value was held. The
fold that produced it ran inside a completed Batch's transaction (ADR-0007,
`internal/queue/spanfold.go`), but the batch id was discarded once the span was written.

ADR-0041 draws the line that governs whether we may add the link at all:

> Where `Batch` sits — **Corpus 1, not corpus 2.** A `Batch` is read by the comparison
> path; a `Dispatch` may not be.

`Batch` already travels with its observations because the fold reads it for scope and a
`Break` reads its recorded source set — it is *in* the comparison path. `Dispatch` was
made Operational **precisely so the grouping cannot be reached from the comparison
path**. A span citing a batch is therefore legal. A span citing a dispatch would not be.

## Decision

A `Span` cites the `Batch` that folded it, on each side it has:

- `opened_batch_id BIGINT REFERENCES batch (id)` — the Batch whose fold **opened** this
  span.
- `closed_batch_id BIGINT REFERENCES batch (id)` — the Batch whose fold **closed** this
  span (an ordinary value move closes the old span and opens the new one in the same
  batch transaction, so a `changed` transition's two spans share a batch on this pair of
  columns).

Both are **nullable** and both reference `batch`, never `dispatch` — the corpus-1
citation ADR-0041 permits, not the corpus-2 grouping it forbids. Nullable because:

1. Spans written before this migration carry no batch id (the corpus is never rewritten,
   ADR-0041), so they are honestly un-attributable — the feed groups what it can and
   states nothing about the rest.
2. An open span has no closing batch yet.
3. A withdrawal closure is *not* a batch fold — whether a subject left the estate is a
   cross-class composition (`internal/estate.WithdrawnCrossClass`, ADR-0087), not one
   batch's outcome — so a withdrawal-closed span may legitimately carry no
   `closed_batch_id`. The feed attributes withdrawals to a batch only where one is
   genuinely responsible.

The fold write path (`spanfold.go`) threads the batch id from `worker.complete` — where
`InsertBatch` already returns it — into `OpenSpan` (as `opened_batch_id`) and, on a value
move, into `CloseSpan` (as `closed_batch_id`). No new instant, no new decision: the id is
the one the batch already has.

The estate-wide feed (`ListRecentDriftEvents`) then reads spans opened or closed by a
batch within a period (capped at a bounded number of most-recent events so an
`all`-time read on a mature estate cannot load an unbounded corpus), joins each to its
`batch` row for the group header — the batch kind and how long ago it folded
(`<kind> scan · <relative> ago`), with a scope sub-label read from `recorded_scope`
(`N names` / `N addresses` / `N services`) — and classifies each event into the six
change kinds on read using the existing `internal/drift` grammar — `appeared`/`returned`
via `MembershipReturn`, `changed` as an ordinary value move with a before/after diff,
`withdrawn`/`descoped` from a close's `closure_reason`. Nothing is stored. The batch
citation only groups what the fold already derived.

## Consequences

- The drift screen's transition timeline, movement summary, period selector and CSV
  export all read one estate-wide feed. `/drift` stops rendering a permanent
  empty-state once a second batch has folded a value that moved.
- The comparison-path separation (ADR-0041) is preserved: the feed reads `span` and
  `batch` only — never `dispatch`. The existing operational reads (`ListDispatchProgress`
  for the "Batch detail" nav link) are untouched and remain corpus-2 reads used purely
  for navigation.
- `withdrawn`/`descoped` transitions are wired end-to-end but stay dormant until the
  withdrawal-persistence path (`drift.CloseWithdrawal` → a reasoned `CloseSpan`) is
  itself wired into a worker — today no production path closes a span with a reason, so
  the feed honestly shows none. When that path lands it needs no change here: a reasoned
  close already classifies.
- The `span` corpus gains two columns but no new write semantics beyond carrying an id
  the fold already held. The corpus is still never compacted or deleted (ADR-0041).
