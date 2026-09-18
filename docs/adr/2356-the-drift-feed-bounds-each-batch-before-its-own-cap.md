---
number: 2356
title: "The Drift feed bounds each batch before its own cap"
slug: the-drift-feed-bounds-each-batch-before-its-own-cap
date: 2026-09-18
status: accepted
source: fix
ticket: 2356
proof: {test: "internal/dbtest/drift_feed_batch_bound_test.go::TestOneLargeFoldLeavesRoomForEveryOtherBatch2325"}
relations:
  - {kind: amends, adr: 178}
---

# ADR-2356: The Drift feed bounds each batch before its own cap

## Decision

> **The Drift feed bounds each batch at 50 edges before its own 500-event cap, and states the two
> truncations apart.** A fold above 50 shows as its own 50 rather than taking the window.
>
> A reader asks why ADR-0178 §1's *one bound, one site* gives way. Because §1 and §2 answered
> volume, and one fold larger than the cap is a different adversary: a dead prober Gaps thousands
> at once, and `/drift` then shows one batch.
>
> §2 refuses a bound that **depends on** batch size. A constant does not. The absence of one let
> batch size decide the window's contents.
>
> Rejected: state the eviction without bounding it. That leaves the feed showing one batch.
>
> Reversal returns the window to the first fold that fills it.

## 1. Context

`db/queries/span.sql#ListRecentDriftEvents` sorted newest-batch-first and cut with a bare
`LIMIT @max_events`, called with `cmd/web/drift.go#driftFeedLimit` = 500. The read carried no
per-batch bound, so one fold that moved more than 500 spans took every slot, and `/drift`, its CSV
export and `/api/v1` then showed **one batch**. Every other batch inside the window disappeared.

The truncation marker said the list was capped. It did not say one batch consumed it, so a reader
could not tell a busy window from a swallowed one.

**The fold that does it already exists.** `db/queries/signals.sql#ListOutageReachGapVantages` carries
the note *"an outage Gaps thousands at once (#2180)"*. A vantage going unavailable closes every open
span it fed and opens a Gap behind each, so one dead prober on a large estate produces thousands of
span edges in one write. That is not a hypothetical estate size; it is the shape
[#2180](https://github.com/winniel123/verge-asm/issues/2180) already measured.

[#2325](https://github.com/winniel123/verge-asm/issues/2325) is the repository owner's ruling of
2026-09-17 that the feed's bound is global where the thing it protects against is per batch.

## 2. The two limbs this ADR moves

ADR-0178 §1 ends:

> The cap is applied **at the query**, never by discarding rows in Go. One bound, one site.

ADR-0178 §2 rules two properties, and the first is:

> **The cut is not batch-aligned.** The 500th row can fall inside a batch, so the oldest visible
> group may be a partial batch whose count pill states what was rendered, not what the batch holds.
> Rounding the cut to a batch boundary would make the bound depend on batch size.

**Both move.** There are now two bounds — a per-batch rank filter and the global `LIMIT` — and the
cut is batch-aware.

**What stands, untouched.** §2's second property is not weakened: the cut is still never re-ordered
by severity, subject or change kind, so the omitted tail is still not a product judgement about which
change matters. §1's reason for 500 stands, and 500 is unchanged. §3's requirement that the page
state its truncation stands, and this ADR extends it to a second truncation rather than replacing it.
§4's whole-period contract is untouched.

## 3. A constant is not batch-size dependence

§2's stated harm is a bound that **depends on** batch size. The per-batch bound is
`cmd/web/drift.go#driftBatchLimit` = 50, a constant. No batch's size changes what any other batch
receives.

The pre-change behaviour is what actually made the window depend on batch size: with one bound, the
number of batches `/drift` could show was `500 / (size of the largest batch in the window)`. A
500-span fold reduced it to one. The bound this ADR adds removes that dependence rather than
introducing it.

**The repo has already ruled this exact shape once.** `docs/spec/scans-monitor-bounding.md` §3's
first rule reads:

> Today Active and History are both carved from one `ListDispatchProgress(scansHistoryLimit)` read,
> split in the handler. A burst of in-flight jobs eats the shared 50 and silently shrinks completed
> history. Give History its **own dedicated window**.

One shared cap, one greedy participant, another participant silently starved. The remedy there was a
dedicated window per participant. Here the participants are batches and they are not enumerable in
advance, so the remedy is a per-participant bound instead of a per-participant window. The argument
is the same argument, and #933 made it before #2325 did.

50 is not derived from a threshold, and this ADR does not pretend otherwise. It is stated: the dev
fixtures model consecutive batches at 3, 2 and 2 transitions and the `internal/dbtest` fixtures seed
ordinary batches at 1–3 spans, so an ordinary fold is a single-digit number of edges and 50 is well
clear of it; `500 / 50` guarantees 10 batches a place, past the groups `/drift` renders expanded. It
matches `cmd/web/scans.go#scansHistoryLimit` and the per-class cap
[#2221](https://github.com/winniel123/verge-asm/issues/2221) records. Like 500, it is cheap to move.

## 4. The probe row, and §1's "never in Go"

§1's second sentence is the harder one, and this ADR does not soften it by re-reading it. The feed
asks for one row past the bound — `cmd/web/drift.go#driftBatchRead` is `driftBatchLimit + 1` — and Go
drops the probe. That is a row discarded in Go.

It is the repo's own ruled pattern for truncation detection. `docs/spec/scans-monitor-bounding.md`
§3's fourth rule is *"fetch with `LIMIT N+1` (fetch 51, show 50, `Truncated = len > 50`)"*, and
`cmd/web/scans.go` implements it. It also avoids the defect
[#2222](https://github.com/winniel123/verge-asm/issues/2222) reports, where a capped flag set by
equality claims a truncation that did not happen.

**Rejected: a `COUNT(*) OVER (PARTITION BY batch_id)` column.** It keeps every row the query returns,
which is what §1 literally asks for. It widens `ListRecentDriftEventsRow`, and two struct conversions
in `cmd/web/scans.go` and `internal/dbtest/run_outcome_scope_test.go` convert that row to and from
`ListDriftEventsForBatchesRow`. Widening one side breaks both. The cost is paid at every call site to
honour a clause about where a bound is applied, and the bound *is* applied at the query — only its
detection is not.

So §1's first sentence holds in substance and its second does not hold literally. This ADR moves the
limb rather than claiming the practice complies with it.

## 5. What this ADR does not rule

**The value of 50.** #2325 left N unruled and the pull request picked it. §3 records the reasoning,
not a derivation. Moving it needs no ADR, exactly as ADR-0178 §5 holds for 500.

**The query plan.** The `ranked` CTE's window sorts on a different key from the final `ORDER BY`, so
Postgres may pay two sorts where the bare `ORDER BY … LIMIT` could take a top-N heapsort — on the
unbounded custom window §1 names. Unmeasured.
[#2351](https://github.com/winniel123/verge-asm/issues/2351) holds it.

**A batch whose every row the classifier drops.** `cmd/web/driftfeed.go#buildDriftFeed` renders no
group for it, so its per-batch truncation is unreported. That is pre-existing and unchanged. Fixing
it needs a truncation carrier that does not hang off a rendered group.

**Whether a batch cut by the global `LIMIT` reports a per-batch truncation.** It does not. §2's
surviving property admits the batch-unaligned cut, and the window callout already tells the operator
the oldest tail is missing.

**Any bound on another feed.** This ADR reaches `ListRecentDriftEvents` alone.

## 6. Consequences

`/drift` shows at least 10 batches in any window that holds them, whatever the largest fold did. A
batch above 50 states its own truncation, in the console, in the CSV export and in `/api/v1`, apart
from the window's. Those two facts are different and an operator needs the second to know a fold was
large rather than the window busy.

A reader of `ListRecentDriftEvents` now holds two bounds and a probe row where ADR-0178 promised one
bound and one site. That is the cost, and §4 is where it is paid.

This ADR is expressible only because
[#2233](https://github.com/winniel123/verge-asm/issues/2233) made a clause-less `amends` valid.
ADR-0178's `## Decision` is unnumbered, so before that ruling no relation could name it and this
amendment had no route.

Reversal returns the window to the first fold that fills it, silently, which is the state
[#2325](https://github.com/winniel123/verge-asm/issues/2325) describes.

## 7. Proof

The proof names `TestOneLargeFoldLeavesRoomForEveryOtherBatch2325`, in
`internal/dbtest/drift_feed_batch_bound_test.go`. It seeds one fold above the bound plus several
small batches into a real Postgres, reads the feed, and asserts every batch appears.

| What the case seeds | What it locks |
| --- | --- |
| one fold above `driftBatchLimit`, then several small batches | the large fold does not evict the others |
| the large fold's own rows | it contributes at most the bound, plus the probe |

Removing the rank filter turns it red: the large fold takes the window and the small batches
disappear, which is the defect exactly.

That suite runs against a `postgres` service in the **required** `query-harness` job, which calls
`scripts/assert-dbtest-ran.sh` and fails on any `--- SKIP` and on a run with no `--- PASS`
([#2255](https://github.com/winniel123/verge-asm/issues/2255)). So this proof gates a merge, and it
cannot pass by skipping — the gap
[#2305](https://github.com/winniel123/verge-asm/issues/2305) reports for a proof whose suite no
required check runs does not apply here.

`cmd/web/drift_batch_bound_test.go` carries the rendering half — that the two truncations are stated
apart on the page, in the CSV export and in `/api/v1` — in the required `test` job.
