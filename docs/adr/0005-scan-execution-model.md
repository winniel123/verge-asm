# ADR-0005: Scan execution model

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#9 Scan scheduling and job execution model](https://github.com/winniel123/verge-asm/issues/9)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0001](./0001-stack-and-runtime.md) chose a Postgres-backed queue (`SELECT … FOR UPDATE
SKIP LOCKED` + `LISTEN/NOTIFY`) over a broker, on the grounds that **job outcome and
observation data must commit together** — a job marked complete over a rolled-back write
manufactures false removals across the estate. It accepted, as the cost of that choice,
that retry, backoff, dead-lettering and job visibility are built rather than inherited,
and handed that work here.

Three prior decisions constrain the shape:

- **[`CONTEXT.md`](../../CONTEXT.md)** already names the unit. A **`Batch`** is "one source,
  executed once, against one scope, from one vantage — recording the scope its silence
  covers." It is the unit of like-against-like comparison, and its scope record is what
  licenses an `enumerable` source's silence to count as evidence of absence.
- **`ScanRun` is explicitly rejected** — a scan fires many batches, and "grouping them is
  useful for progress display, not for comparison." The rejection concedes the display need
  without satisfying it.
- **`Scan` is Declared** — "the operator's configured recurring intent … the configured
  thing, never the executed one." So nothing about execution may live on it.

The map's standing rule governs every failure path: **a source that errors must produce no
observation, never an observation of absence.**

## Decision

| Concern | Decision |
| --- | --- |
| Unit of work | **One queue job = one `Batch`** |
| Batch partitioning | Along any dimension the source **retains enumerability** over |
| Port tiers | **Three `Scan`s**, one cadence each — not one `Scan` with tiered cadence *(amended to **four** by [ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md), on this row's own reasoning: `tls-acceptance`'s weekly enumeration is a fourth `Scan` whose scope is the open `Service` population and the TLS candidate set, not a port tier)*. *(Read **two** port tiers and **three** `Scan`s after [#78](https://github.com/winniel123/verge-asm/issues/78) retired the weekly tier — see the amendment under "Port tiers are three `Scan`s".)* |
| Tick firing | **Any worker**, under a Postgres advisory lock, idempotent on `(scan, scheduled_time)` |
| Fan-out | **Atomic** — the `Dispatch` row and all its job rows commit in one transaction |
| Overlap | **Skip**, recorded as a first-class operational event |
| Missed ticks | **No catch-up, no jitter** — skip to now, record the skips |
| Partial failure | Observations **kept**; batch scope records what **completed** |
| Dead letter | Batch records an **empty scope** — licenses no absence at all |
| Retry | A **new `Batch`**, never a resumption |
| Progress grouping | **`Dispatch`** — operational only, never a comparison key |
| Manual trigger | Dispatches an **existing `Scan`**; does not reset cadence |
| Backoff | **Per-job** exponential with jitter, plus a **per-source token bucket** in Postgres |
| Vantage availability | **Derived** from batch outcomes; window is **fixed and release-coupled** |

### Non-binding implementation defaults

- Per-worker max in-flight jobs: **10**.
- Job visibility timeout / heartbeat so a worker that dies releases its claim; a lost job
  counts as an attempt against its retry budget.
- Progress delivered over ADR-0001's SSE channel as a count query against the `Dispatch`.

## Rationale

### One job is one batch, sized by enumerability

The commit-together invariant ADR-0001 rejected Redis to preserve is free if — and only if —
the queue row *is* the batch: the single transaction that marks the job done writes the
observations and the scope record together. Any other arrangement reintroduces
reconciliation in exactly the code path where a mistake looks to the operator like an
attack-surface change.

The obvious objection is transaction length. ADR-0001 sizes a hot-set run at ~70k
reachability observations per day; a batch defined as "the prober against a whole address
scope" is a transaction open for hours, holding a `SKIP LOCKED` row throughout, discarding
everything on a crash at minute 200.

The resolution is that **partitioning is governed by `completeness`, not by size**: a batch
may be split along any dimension the source can still claim enumerability over. Our prober
against one address is enumerable over `(that address, that port set)`, so one address per
batch keeps transactions short *and* keeps the scope record honest. The operator-supplied
zone file is enumerable over an entire zone and is therefore always one batch — splitting it
per-name would destroy the only mechanism by which removal detection works at all. A
`corroborative` source may be split however is convenient, because its silence never
asserted anything.

This also disposes of a coordination problem rather than solving it.
[#4](https://github.com/winniel123/verge-asm/issues/4) requires rate limits **per target
host, not global**, and with `--scale worker=N` a cross-worker per-host limit would need
distributed coordination. Under this partitioning one address's ports are never split
across concurrent jobs, so the per-host limit is intra-job and needs no coordination.

That claim does **not** extend to third-party sources.
[#3](https://github.com/winniel123/verge-asm/issues/3) measured crt.sh needing a hard 5
req/min throttle, and that limit is per-source across the whole instance — so it lives in
Postgres alongside the queue, not in worker memory.

### Failure records what completed, not what was attempted

A batch that fails halfway has made real measurements, and discarding them throws away good
data to avoid a bookkeeping problem the scope record already solves. So partial results are
kept, and the batch's scope records what it **completed** — the extent over which its
silence is evidence. A port attempted and timed out is completed (a timeout *is* a
measurement); a port whose worker died before recording a result is not.

The dead-letter case is where this bites hardest, and where the naive answer is wrong. A
dead-lettered batch recording "attempted 140 ports, produced 0 observations" asserts 140
absences — the manufactured-drift failure ADR-0001 rejected Redis to avoid, reintroduced
through the failure path. **A dead-lettered batch records an empty scope and licenses no
absence whatsoever.** The attempt is still fully recorded, operationally, on the job and its
`Dispatch`, where it drives visibility and never touches the drift engine.

This is the general shape: **evidential coverage and operational attempt are two different
records**, and every failure path writes to the second without inflating the first.

A retry is a **new batch** for the same reason the glossary says "executed once" — a batch
resumed forty minutes later carries a timestamp that misstates when half its observations
were made, and a retry landing on a different worker could carry a different vantage.

### `Dispatch` is `ScanRun` with the dangerous half removed

The UI needs something to render progress against, and under this partitioning one tick fans
out into hundreds of batches. `CONTEXT.md` rejected `ScanRun` because making it a domain
term "invites change to be defined as a function of consecutive runs — which breaks as soon
as two scans run on different cadences over different port sets." That objection is about
**comparison**, not display.

So `Dispatch` exists, holds the fan-out and its progress, and is **structurally barred from
the comparison path**: it carries no observations, and the drift engine never reads it.
Batches anchor scope; timelines anchor comparison; `Dispatch` anchors neither. The risk is
plain — this is `ScanRun` with a haircut, and one `WHERE dispatch_id = previous_dispatch_id`
in a drift query undoes the rejection — which is why the prohibition is written into the
glossary entry rather than left to discipline.

It carries a **snapshot of the `Scan` config** taken at tick time. Correctness survives
either way, since every batch records the scope it actually covered, but legibility does
not: [#15](https://github.com/winniel123/verge-asm/issues/15) established that widening the
aperture is an estate-wide event and that a subject first observed under a widened aperture
is not "appeared". That is only attributable if the widening lands on a boundary the
operator can point at — dispatch 41 ran the old config, dispatch 42 the new one — rather
than being smeared through the middle of one fan-out.

Fan-out is **atomic** for the analogous reason. A partially-enqueued dispatch covers less of
the estate than its config says; under the completeness rule the missing batches are
correctly never spoken about, but nothing anywhere records that the dispatch under-covered.
The idempotency key on `(scan, scheduled_time)` makes the retry safe.

### Skipping, not queueing or overlapping

Overlapping runs put two batches with the same scope and vantage in flight simultaneously,
muddying which one a comparison should read. Queueing builds a backlog that never drains — a
scan slower than its cadence accumulates debt forever.

The load-bearing part is not the skip but the **record** of it: a silently-dropped tick is
indistinguishable from a tick that ran and found nothing, which is the map's false-absence
failure wearing a different hat.

The same reasoning settles downtime. **You cannot measure the past**, so firing a two-day-old
tick on restart measures the world now and delivers nothing except the record that time was
missed — which the skip record already delivers. Consequence handed to
[#8](https://github.com/winniel123/verge-asm/issues/8): a gap in observations is a *real*
gap, and the drift engine must not treat a three-day-old batch as current.

Tick jitter is a knob that does nothing here. Enqueueing is atomic and instant; the jobs
drain at whatever rate the concurrency cap allows, so **the cap is the throttle**.

### Port tiers are three `Scan`s

`Scan` is defined with a singular cadence, while [#4](https://github.com/winniel123/verge-asm/issues/4)
decided a ~140-port hot set daily, nmap top-1000 weekly and full range opt-in. Three `Scan`s
keeps the glossary definition intact and makes the aperture a configured object rather than a
hidden field — which matters because [#5](https://github.com/winniel123/verge-asm/issues/5)'s
scope records exist precisely so an aperture change cannot manufacture drift. "The operator
disabled the weekly deep scan" becomes a legible state.

*Amended by [ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md): **four `Scan`s,
not three.*** `tls-acceptance` is measured by enumerating N candidates per service, which
[#4](https://github.com/winniel123/verge-asm/issues/4) put on a weekly cadence on cost grounds
alone — a cadence, never a port set. Hanging it off the daily `Scan` as "every seventh run" is the
tiered-cadence shape this section rejects, so it takes a `Scan` of its own: weekly, scope the open
`Service` population plus the TLS candidate set, **no port tier**. Everything in this ADR applies
to it unchanged. The `certificate` handshake, by contrast, adds no `Scan` at all — it is a step in
the exchange that produces `reachability`, so it rides whichever of the three port tiers ran it.

**Manual runs dispatch an existing `Scan`**, never an ad-hoc one-off: an ad-hoc run with a
hand-picked port set produces a batch whose scope no configured object accounts for, which is
an aperture change with nothing to point at. A manual dispatch **does not reset the cadence**,
or an operator clicking "run now" each morning silently disables the cron they believe is
protecting them.

> **Amended 2026-08-14 by [#80](https://github.com/winniel123/verge-asm/issues/80) — two things,
> one a count and one a limb this section was missing.**
>
> **The count.** [#78](https://github.com/winniel123/verge-asm/issues/78) retired the weekly
> top-1000 tier, so this section's *"~140-port hot set daily, nmap top-1000 weekly and full range
> opt-in"* is two tiers, not three, and ADR-0028's amendment reads **three `Scan`s, not four**.
> **And the port count in the same phrase is wrong in its own right**: **[measured]** by
> [#97](https://github.com/winniel123/verge-asm/issues/97), `verge-core`'s frequency half is **123, all
> TCP** and the union is **136 pairs** — *"~140"* was never reproducible from
> [#4](https://github.com/winniel123/verge-asm/issues/4) §2.3's own two limbs
> ([`sensitive-ports.md`](../research/sensitive-ports.md) §29).
> Nothing in either argument depends on either number.
> [ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md)'s *"one declared set across
> all three reachability `Scan`s"* is stranded the same way and reads **two**.
>
> **The limb.** The ad-hoc prohibition above rests on the *configured object* alone, and there is a
> second and stronger reason it never stated: **a one-off has no cadence, so it has no currency
> bound.** ADR-0028 sets currency at `k` cadences of the covering Declared `Scan` and
> [ADR-0038](./0038-a-constant-is-a-product-only-where-the-quantity-is-readable.md) publishes the
> full-range row as 60 days on that arithmetic. A batch dispatched once leaves `k × cadence`
> undefined for every timeline it opens, so its observations either render as current forever or
> open a `Gap` at a time no rule can compute. That is why *"run the full range once at onboarding"*
> is not a cheaper version of the full-range tier but an inexpressible object, and why the widest
> tier ships **configured and disabled** with an empty scope list —
> [ADR-0044](./0044-a-one-off-measurement-has-no-currency.md).

### Vantage availability is Derived, and its window is fixed

[#14](https://github.com/winniel123/verge-asm/issues/14) made the external prober optional and
reached over SSH. If SSH is down for three days the internal batches keep succeeding and the
external ones keep failing, so `Exposure` — computed *across* vantages — silently degrades to
internal-only, where `firewalled` and `internal-only` become indistinguishable. That is
precisely the distinction #14 exists to draw.

So the scheduler detects it and publishes it: a vantage failing every attempt across a window
is `unavailable`, and the exposure derivation reads that state, making exposure
**non-constructible** rather than quietly computed from one vantage — the same construction
#14 used for day one.

Availability is **Derived**: not Observed (we never measured the vantage, we inferred it from
things that failed), and not Declared (the operator declared intent, per #14). It is the first
property in the model attached to a term in one layer while itself belonging to another, which
is worth stating explicitly rather than leaving to be rediscovered. The payoff is that
[ADR-0004](./0004-signals-are-release-coupled-rules.md)'s composition rule then applies for
free: `Exposure` reads availability, so `Exposure`'s version composes availability's version,
and changing the window makes evaluations non-comparable automatically.

The window is **fixed and release-coupled**. ADR-0004's test — would we ever want to push this
out of band? — says no, and the window feeds the derived value the landing view is built from,
so an operator turning that dial silently makes their entire board non-comparable.
[#16](https://github.com/winniel123/verge-asm/issues/16) did ship an operator-configurable N
for certificate expiry, and that precedent cuts against this; the distinction drawn is blast
radius — expiry-N invalidates comparability for one signal, this one for the whole board.

## Consequences

- **`Dispatch` enters the glossary** under a new **Operational** heading, outside the
  Declared/Observed/Derived table, because it belongs to none of the three and the boundary is
  the point. `ScanRun`'s rejection entry is amended to point at it.
- **`Vantage` gains a Derived `Availability` property**, and `Exposure` composes its version.
- **`Scan` is unchanged** — nothing about execution was added to it, which is what the
  Declared-layer definition required.
- **The drift engine inherits two obligations**, both handed to
  [#8](https://github.com/winniel123/verge-asm/issues/8): reading sibling batches from one
  dispatch without reading the dispatch itself, and refusing to treat a stale batch as
  current.
- **Third-party source throttling needs cross-worker state in Postgres** — a token bucket
  beside the queue. This is the one coordination problem the partitioning decision does not
  eliminate, and it should not be mistaken for solved.
- **Job visibility must include skips and dead letters**, since both are states the operator
  can only learn about from the operational record — by construction they leave no trace in
  the observation data.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Many jobs assembling one batch | Reintroduces partial-batch reconciliation, the exact thing ADR-0001 rejected Redis to avoid |
| One long-lived batch committed incrementally | Hours-long claim on a queue row; scope record becomes a protocol rather than a fact |
| Partition batches by size | Splits `enumerable` sources below their scope, destroying removal detection |
| Dedicated scheduler container | A third service to buy what one advisory lock and an idempotency key already provide |
| Ticking from `web` | ADR-0001 split the tiers by blast radius; scheduling is worker-side work |
| Queue overlapping ticks | A scan slower than its cadence accrues a backlog that never drains |
| Allow concurrent runs of one `Scan` | Two batches, same scope and vantage, in flight at once — ambiguous for comparison |
| Catch up missed ticks after downtime | You cannot measure the past; the skip record already carries the only information gained |
| Discard partial results on failure | Throws away real measurements to avoid a problem the scope record solves |
| Dead-lettered batch records its attempted scope | Asserts absences it never measured — manufactured drift through the failure path |
| Retry resumes the failed batch | "Executed once" — misstates measurement time and can silently change vantage |
| Ad-hoc manual scans | An aperture change with no configured object accounting for it |
| Operator-configurable availability window | A dial that silently makes the operator's whole board non-comparable |
