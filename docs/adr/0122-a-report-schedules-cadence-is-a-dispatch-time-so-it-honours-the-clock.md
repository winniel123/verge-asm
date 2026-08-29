# ADR-0122: a report schedule's cadence is an operator-authored dispatch time, so it honours the clock — presets to the minute, Custom as real cron

- **Status:** Accepted
- **Date:** 2026-08-26
- **Ticket:** [#639 Report-schedule cron/cadence engine](https://github.com/winniel123/verge-asm/issues/639)
- **Map:** [#630 v3.16.2 consumption map](https://github.com/winniel123/verge-asm/issues/630)
- **Supersedes in part:** [ADR-0118](./0118-report-scheduling-dispatches-on-a-computable-window-and-the-receipt-is-its-own-dispatch-record.md) — only its **cron-refusal clauses**: §1's *"the dispatcher dispatches on that window and never interprets a cron predicate … a custom / unrecognised cadence falls to the weekly window"*, the Consequences bullet *"Custom cadences are coarse — a cron string dispatches weekly"*, and the two Alternatives-rejected rows that reject evaluating cron. Everything else in ADR-0118 stands: `CadenceWindow` as the artifact **period** (§1's window vocabulary), the receipt-is-the-dispatch-record idempotency design (§2), and in-instance-only dispatch (§3).
- **Preserves:** [ADR-0091](./0091-the-routing-unit-is-the-class-and-the-cause-is-refused-as-a-routing-key.md) in full, and [ADR-0039](./0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md) §4/§6.

## Context

This ADR transcribes a **standing binding ruling** (design side, 2026-08-24, drift-audit
ruling **#31**, `design-system/SPEC-CHANGE.md`) and the operator's explicit direction to build
the full engine. It does **not re-decide** whether report schedules honour the clock — that is
ruled. It scopes the supersession and records the reconciliation with the ADRs the prior design
cited.

`report_schedule` (migration 21700) stores a **cadence** as a free-text label: a preset from the
wizard's CadenceSelect (`every 6h`, `daily · 08:00`, `weekly · mon 09:00`, `monthly · 1st`) or,
under "Custom…", an operator-authored 5-field **cron** string (`cmd/web/reports_schedule.go`).
ADR-0118 (T3/#502) built the first dispatcher and deliberately **did not honour either**:

- `report.scheduledTick` floored `now` to the cadence **duration** from the Unix epoch, so
  *"daily · 08:00"* fired at the epoch 24-hour boundary (≈00:00 UTC) — **never at 08:00**.
- `report.CadenceWindow` mapped every Custom/cron label to the **weekly** window, and the code
  comment stated the doctrine: *"a custom / cron cadence is NOT evaluated as an operator-authored
  predicate (ADR-0091) … no versioned-rule-set predicate is ever evaluated."*

ADR-0118 named this a deliberate interim and wrote its own reopening condition: *"Minute-accurate
custom cadences are a later refinement (a real cron evaluator would reopen the ADR-0091 question
deliberately)"* and *"If minute-accurate custom cadence is ever required, it reopens this ADR and
the ADR-0091 question … a real predicate evaluator … would be adopted, if ever, **on the record**."*
This ADR is that reopening, on the record.

### A citation correction

The `cadence.go` doc comment, its test, ticket #639, and map #630 cite the no-cron doctrine as
**"ADR-0091 and ADR-0117."** ADR-0117 is *"a session is a server-side record, so it can be
revoked"* — it says nothing about cron, cadence, or rule-set predicates. The substantive
report-scheduling decision the citation means is **ADR-0118**. The "0117" is a stale
mis-citation that propagated from the code comment into the ticket and the map. This ADR
supersedes the real clauses in **ADR-0118** and leaves ADR-0117 (sessions) untouched. The stale
`cadence.go`/test citations are corrected at their sites (ADR-0058: withdraw at the site that
specifies it).

## Decision

> **A report schedule's cadence is an operator-authored *dispatch time*, not an evaluated *rule
> predicate over the estate*. The dispatcher therefore fires each schedule at the clock time the
> operator declared — presets honoured to the minute, and a Custom cadence interpreted as a real
> 5-field cron expression — computed in UTC. `CadenceWindow` continues to name only the artifact
> *period*; the *fire instant* is computed separately. The idempotency shape is unchanged.**

### 1. Why this does not reopen what ADR-0091/ADR-0039 guard

ADR-0091 (inheriting ADR-0039 §4) refuses *"an operator-authored predicate over a **versioned
rule set**"* as a **routing** key, on the stated hazard that it *"fails silently the first time a
rule is renamed."* That hazard is about a predicate that reads the **estate's rules/subjects** to
decide a **routing** outcome, and breaks when the corpus it references moves.

A schedule's cron expression is none of that. It is a **pure predicate over the wall clock** —
*is this minute a firing minute?* — evaluated against `time.Now()`, and it is:

- **Not a routing key.** It decides *when a run happens*, never *who receives it*. Routing stays
  by class alone (ADR-0091 untouched). A schedule with no bound Channel is download-only and
  routes to no one at all.
- **Not over the estate.** It references no rule, no subject, no span, no signal — nothing that
  can be renamed. It cannot "fail silently when a rule is renamed" because it names no rule. It
  reads a clock, and clocks are not versioned.
- **Not a rule-set evaluation.** It produces no admission, no class, no derivation. It fires a
  document cut. The corpus separation ADR-0039/ADR-0041 protect is not entered.

So honouring cron reintroduces **none** of the versioned-rule-set risk ADR-0091 and ADR-0039
guard. It is a firing schedule, exactly the thing ADR-0118 said would be "a firing schedule"
rather than "a real predicate evaluator." (Had the cadence instead selected *which subjects* a
report covers by an operator expression over the estate, that WOULD be the guarded hazard and
this build would have stopped and escalated. It does not: the cadence times the cut. The
sections/period decide the content, unchanged.) ADR-0091's decision — the routing unit is the
class — is not touched, and ADR-0039 §6's coalescing/suppression refusals are not touched (a fire
schedule delays, holds, and suppresses nothing).

### 2. `CadenceWindow` (period) and the fire tick (instant) are separate

ADR-0118 §1's `CadenceWindow` stands **unchanged** as the single source of truth for the artifact
**period** — how much of the estate a run summarises (6h / daily / weekly / monthly), shared by
Run-now and the dispatcher so a manual and a scheduled run cover the same span. What this ADR
replaces is only the **fire instant**: `scheduledTick`'s epoch floor becomes `DispatchTick`, which
returns the most recent instant at or before `now` on which the cadence declares a firing —
the operator's declared clock time. The two concerns are orthogonal: a schedule fires at 08:00 and
still summarises the daily window.

### 3. Grammar and storage — parse-per-tick, no migration

The stored `cadence TEXT` already carries everything needed. **No new columns and no migration**
are added (storage decision: **parse-per-tick**, deterministic and stateless).

- **Presets** are the closed, model-owned CadenceSelect set. Each is translated to an equivalent
  cron, honouring the clock time and day the label literally carries: `daily · 08:00` → `0 8 * * *`,
  `weekly · mon 09:00` → `0 9 * * 1`, `every 6h` → `0 */6 * * *`, `monthly · 1st` → `0 0 1 * *`
  (the label carries no clock time, so it fires at **day-start, 00:00**).
- **Custom** is parsed as a 5-field cron directly.

A small **in-repo 5-field cron evaluator** does the work (`internal/report/cadence.go`) — no new
dependency, matching the dependency-free stance of the queue dispatcher. It implements standard
Vixie semantics:

- `*`, lists, ranges, and `*/n` steps.
- Day-of-week `0–7` with both `0` and `7` Sunday.
- The dom/dow **union** rule (when both day fields are restricted a day matches if either does).

### 4. Timezone — UTC, because no instance timezone is modelled

Nothing in the schema or config carries a timezone. Every stored instant is UTC. A cadence's
declared clock time is therefore interpreted in **UTC**, and `DispatchTick` computes in UTC. This
is stated, not assumed. When an instance timezone is ever modelled, `DispatchTick` is the one
place that resolves the location.

### 5. Idempotency and missed-tick policy are unchanged

The receipt-is-the-dispatch-record design (ADR-0118 §2) is untouched. `DispatchTick`'s fire tick
is the value the partial-unique `(schedule_id, scheduled_tick)` key admits, so two poll ticks
between one firing and the next resolve to the **same** tick (the second conflicts — a recorded
skip, never a second run), and the tick advances only when the clock crosses the next declared
firing. **Missed firings are not caught up:** `DispatchTick` returns the single most-recent
firing, so a worker down over one does not backfill it — currency, not history, exactly as ADR-0118
and the queue dispatcher already behave. The ruling implies no change here, so none is made.

### 6. An invalid Custom cron is refused at authoring, never coerced

An uninterpretable cadence is **refused at schedule create/edit**, not silently defaulted. The
refusal surface is the wizard's **Cadence step** (`cmd/web/reports_schedule.go`): the per-step
validity gate now requires a Custom cron that both is non-empty **and** parses
(`report.ValidateCron`), so the wizard neither advances past nor finishes from the Cadence step
while the cron is malformed, and a finish POST bounces back to that step. The client JS already
blocked an empty cron. The server now additionally blocks a malformed one, so a hand-crafted POST
cannot file an uninterpretable schedule. A cadence that somehow reaches the dispatcher
uninterpretable (a legacy or hand-edited row) is **skipped with a log line**, never fired on a
wrong default.

## Consequences

- **Schedules fire when the operator asked.** `daily · 08:00` fires at 08:00 UTC. `0 9 * * 1`
  fires Mondays at 09:00. `*/15 * * * *` every quarter hour — verified by clock-injected table
  tests, no Postgres required.
- **ADR-0118 is amended at its cron clauses only.** Its window vocabulary, its no-second-table
  idempotency, and its in-instance stance are unchanged and still cited by the dispatcher.
- **ADR-0091 and ADR-0039 are untouched.** Routing stays by class. No estate predicate is
  evaluated. §1 records why a fire schedule is not the guarded hazard.
- **No schema change.** Parse-per-tick over the existing `cadence` column. The receipt store and
  its partial index are unchanged.
- **The mis-citation is corrected.** `cadence.go` and its test cited ADR-0117 (sessions) for the
  no-cron doctrine. Those comments now cite this ADR and ADR-0118.
- **UI is unchanged.** No template moves (SPEC-CHANGE #31 is backend + docs). The wizard renders
  exactly as before — the refusal reuses the existing per-step gate.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Keep ADR-0118's epoch floor / weekly-bucket | The bug this ticket fixes: `daily · 08:00` fires at ≈00:00 and every Custom cron fires weekly, contradicting the wizard the operator filled in. Ruling #31 and ADR-0118's own reopening condition retire it |
| Normalise the cadence into columns (kind/min/hour/dow/cron_expr/tz) via a migration | Available and reasonable, but the `cadence TEXT` label already encodes everything deterministically; a migration + backfill buys no correctness here and adds schema surface. Parse-per-tick is stateless and cheaper. (If a per-schedule timezone is ever modelled, that is the migration to revisit) |
| Adopt a third-party cron library (e.g. robfig/cron) | A dependency for a 5-field parser the tree can carry itself; the queue dispatcher is dependency-free and this matches it. A vetted library would be a fine later swap, noted rather than taken |
| Silently default an invalid Custom cron to weekly | The thing the ruling forbids ("not silently weekly-defaulted"): it files a schedule that fires at a time the operator never chose. Refused at authoring instead |
| Escalate as a doctrine conflict (AWAITING DESIGN) | Not available: honouring cron introduces no evaluated rule-predicate over the estate (§1), the ruling is binding and explicit, and the operator directed the full build. Size is not grounds to escalate |
