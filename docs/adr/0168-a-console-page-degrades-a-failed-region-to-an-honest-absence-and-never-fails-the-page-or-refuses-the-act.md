# ADR-0168: a console page degrades a failed region to an honest absence, and never fails the page or refuses the act

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1360 ADR gaps: cmd/web sources.go and cold.go](https://github.com/winniel123/verge-asm/issues/1360), gap 1
- **Ticket:** [#1339 ADR gaps: cmd/web/seeds.go](https://github.com/winniel123/verge-asm/issues/1339), gaps 2 and 3
- **Sweep PR that deleted the comment:** [#1361](https://github.com/winniel123/verge-asm/pull/1361) (cold.go), [#1340](https://github.com/winniel123/verge-asm/pull/1340) (seeds.go)
- **Rests on:** [ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md), which supplies *never fabricated data* — the rule about a datum that does not exist; and [ADR-0072](./0072-absence-is-a-property-of-a-cell-and-withdrawn-is-the-only-population.md), which supplies the shape an absence takes on a screen
- **Rests on:** [ADR-0134](./0134-a-seed-withdrawal-is-recorded-by-a-tombstone-because-the-mover-does-not-survive-the-act.md) §5, which makes the withdrawal land on the next membership fold and so leaves the preview binding nothing
- **Not bound by:** [ADR-0124](./0124-a-backup-carries-data-and-no-secret-and-updating-is-guided-not-self-applied.md) §2, whose best-effort is one declinable outbound call to a third party
- **Narrows:** [`docs/guides/api.md`](../guides/api.md) line 103, whose *"Store read failure → `500`"* row is true of a subject read and false of a region read

## Context

`coveragePage` stated one rule five times, in five uncited blocks, and PR #1361 deleted all five: *a
failed read empties its own region rather than 500ing the page*. **Every line number in #1360 has
moved.** The pre-sweep sites `:233`, `:245`, `:273`, `:289` and `:297` are now the reads at
`cmd/web/cold.go:142` (zone declarations), `:147` (current Service subjects), `:162` and `:165`
(blanketed reach, unavailable vantages), `:171` (the signal corpus) and `:176`–`:177` (zone cadence
and zone-file status). Nothing states the rule at any of them. **#1360's three "unswept residue"
sites are gone too** — `exposure.go`, `drift.go` and `custodycensus.go` carry no statement of it
today, and `custodycensus.go` is 130 lines with no line 237.

**#1339's two are compressed rather than gone**, and both survive uncited: `cmd/web/seeds.go:251` —
*"A failed count degrades the block; refusing the act would leave no route to the withdrawal"* — and
`cmd/web/seeds.go:448` — *"An additive card's failed read degrades that card, never the whole Scope
screen."* Those two lines are the last in-tree statement of the rule.

The posture is not incidental. `cmd/web` holds 86 conditional-adoption sites of the
`if v, err := …; err == nil` shape outside tests, and **41 adopt a store or corpus read**; the rest
parse a form value, a URL or an address. `dashboardData` (`cmd/web/auth.go:517`) contains no
`serverError` at all, which is what `cold.go`'s deleted text meant by *"exactly as the dashboard's
signal regions do."*

**This is not ADR-0110's rule.** Its Consequences bullet at line 103 (#1360 cites line 74; the line
moved) governs *a datum that does not exist*: a screen with no backing data ships a design-system
empty-state, never fabricated data. **This ADR's subject is the failure of a read** — the value may
well exist, and we could not obtain it. The two are complements, which is why four of the five
deleted blocks stated them side by side, and why never-fabricate is what stops this rule degrading
into a lie.

**ADR-0124 §2 does not generalise, and its own argument says why.** Its *"Why best-effort and
opt-out"* rests the release-feed check's silence on three facts: a loud failure would *"make a
self-hosted tool's availability depend on a third-party feed"*, the check is the feature's *"only
outbound reach"*, and it is *"fully declinable"*. A console page reads our own Postgres, which is
none of those three. Every premise fails, so the conclusion does not travel.

**A failed read is not a `Gap`.** `CONTEXT.md` spends `Gap` on coverage the estate could not obtain,
on one facet timeline of a living subject. A console read that errors is a fault in our own process
and writes nothing. And `docs/spec/comment-policy.md` §8.2 names this gap in its own body — *"a
degraded console read empties its own region and never fails the page"* — as its worked example of a
rule nothing cites. That records the gap as open; it does not close it.

## Decision

> **A console page prefers a partial honest page to a failed one. Where a read that feeds ONE region
> fails, the handler renders that region as an honest absence, serves the rest of the page, and
> never returns a 500. It never substitutes a value for the read it did not get. Where the failed
> read is an advisory figure on a confirm step, the step still offers the act. The page's own
> SUBJECT read is not a region read: it fails loudly.**

### 1. A region read is best-effort, and its failure empties the region

A **region** is a card, a list, a meter row or a callout the page composes beside others and that no
other region derives from. Its read is adopted only on a nil error; a failure leaves the region's
zero value and the page serves.

The reason is availability arithmetic. `coveragePage` makes eight such reads, so if each were loud
the screen's availability would be the product of eight, and a fault in the corpus builder at
`cold.go:171` would take away the aperture meters, the coverage messages and the stale-zone callout,
none of which it feeds. A 500 costs the operator every region to protect one.

**Logging is decided per site.** `cold.go:153` degrades but logs, under a cited reason at `:152`:
that read *hides evidence* rather than emptying a region, because a missing edge count silently
drops the `Covers` and `SharedEdges` fields from a meter that otherwise renders normally. A
degradation the operator can see needs no log line; one they cannot see does.

### 2. Emptying means an honest absence, never a substituted value

This is the limb that keeps §1 from fabricating. The region renders *no rows*, *no card*, or a note
saying the read did not resolve. It does not render a number. Two shipped examples:
`design-system/templates/scope.tmpl:235` renders *"The fan-out measurement did not resolve on this
load. Nothing is listed rather than a guessed edge."* against the `CustodyCensusFailed` flag that
`cmd/web/seeds.go:449` sets, and `scope.tmpl:182` renders *"The count did not resolve"* for the
confirm block.

**A count is not an honest absence.** `cold.go:147`'s failure leaves `walked` nil, and `addressMeter`
then renders `counted 0`, `Pct 0`. ADR-0120 defines that numerator as *"the subjects the batch
walked"*, so a zero asserts the batch walked none of them — a measurement claim from a read that
never resolved, and an alarm about the operator's estate raised by a database hiccup. The same defect
stands at `cold.go:142`, where a failed zone read renders a name scope's meter as `0 declared names`.
**Both sites contradict this ADR and are named as defects, not ratified.** The fix is small: carry
the read's failure into `apertureMeters` and withhold the numerator, exactly as
`oldestCurrentInRange` already withholds the as-of label when it has nothing.

### 3. An advisory figure degrades the confirm block and never refuses the act

`previewSeedWithdrawal` renders the narrowing receipt for a `Seed` withdrawal. On a failed receipt
it logs, sets `confirm.Failed`, and serves the confirm block with the act still offered
(`cmd/web/seeds.go:264`–`:273`).

Two facts make that correct, and both are checkable. **The step is the only route:**
`/seeds/delete` appears once in the whole template corpus, at
`design-system/templates/scope.tmpl:184`, inside `{{with .SeedConfirm}}`, and the chip's remove
control at `:170` posts to `/seeds/preview`. **The count binds nothing:** `deleteSeed`
(`cmd/web/seeds.go:276`) reads no receipt — it calls `WithdrawSeed` and returns — and under ADR-0134
§5 the withdrawal is performed by the next membership fold from a tombstone. So refusing would cost
the operator their only route to a withdrawal, over a figure the act never consults.

**`(ADR-0134 §5)` was the wrong citation, and #1339 is right about that.** §5 says the fold *"runs
on the next membership fold, last, and its tombstone is then consumed"*, and never mentions the
preview. The nearest sentence in that ADR runs the other way: line 28 says the *exclusion* half is
driven from the declaration side *"so the mover is a row the fold can still read, and the preview and
the act read the same shape."* That is a claim about ADR-0133's exclusion design, not about a `Seed`,
and read carelessly it argues against this limb. §5 supports the ruling only by the route above, and
never as a fact about the preview. The citation is now this ADR.

**The act's own write is not advisory.** `seeds.go:291` 500s when `WithdrawSeed` fails. A silent
failure there would tell the operator a scope was withdrawn when it was not.

### 4. The boundary: what is not best-effort

A read is **not** a region read when the page's claim about the world depends on it. Three classes,
each with its shipped example.

| Class | Example | Why loud |
| --- | --- | --- |
| **The page's own subject** | `cold.go:136`–`:140`, `ListSeeds` on Coverage; `seeds.go:408`–`:411`, `ListSeeds` on Scope | Every meter and card on both screens is one `Seed`. Degrading it renders *no scopes declared* — a lie about the operator's own declaration, and one they may act on |
| **A subject fetched by key** | `cmd/web/subjects.go:221`–`:231` | `pgx.ErrNoRows` renders a distinct missing-subject page; any other error 500s. **No rows and the read failed are different facts**, and the handler keeps them apart |
| **The act** | `seeds.go:291`, `cold.go:71`, `:76`, `:81` | A write reported as done when it was not is unrecoverable by reload |

The test is not the surface. `apiCoverage` (`cmd/web/api_v1.go:258`) makes the identical split on the
identical data: `ListSeeds` reaches `apiReadError` at `:262`, while the zone and subject reads at
`:266` and `:272` degrade with a log line. The split follows the **subject**, so it holds wherever
that page's data is projected. `dashboardData` is the limiting case the other way: it has no subject,
being a summary of several, so no read on it is loud — consistent with this rule, not an exception.

## Consequences

- **`docs/guides/api.md:103` is narrowed.** Its *"Store read failure → 500"* row describes only the
  subject read; a region read on the same endpoint logs and serves.
- **`cold.go:142` and `:147` are defects under §2.** Both render a fabricated zero. They are named
  here so a later session does not read the code as the rule.
- **`driftPage` (`cmd/web/drift.go:171`–`:181`) degrades its own subject.** A failed
  `ListRecentDriftEvents` yields `HasEvents: false`, rendering *no drift this period*. Drift is the
  thesis screen (ADR-0110), and this is the exact class §4 forbids.
- **`foldExposure` (`cmd/web/exposure.go:97`–`:104`) returns an empty board and zero counts** when
  its two subject reads fail, so Exposure reports *0 exposed* on a fault. The page already has an
  honest shape for *we cannot say* — the `Withheld` branch at `exposure.go:56`.
- **Three Scope reads are loud where §1 makes them region reads:** `ListExclusions` (`seeds.go:413`
  → `Exclusions`), `ListVantages` (`:418` → `CoverageMsgs`, through `coverageMessages`) and
  `proposalLookups` (`:433` → `Proposals`). Each takes the whole screen down for one card.
- **`seeds.go:251` and `:448` get their citation rather than a rewrite.** A comment beside a
  best-effort read is otherwise redundant under the comment policy's gate 1, unless it carries a fact
  the shape does not, as `cold.go:152` does.
- **This ADR licenses nothing about writes, exports or the act.** Every 500 named in §4 stays.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Every read is loud — one failure, one 500** | A page's availability becomes the product of its reads'. `coveragePage` makes nine, so the screen fails nine times more often than any one query, and eight of those failures remove regions the failed query does not feed. It also hands the operator the one response carrying no information: `serverError` (`cmd/web/auth.go:1887`) writes a bare *internal error* |
| **Every read is best-effort, subject included** | What `driftPage` and `foldExposure` do today, and it produces ADR-0110's forbidden fabrication by another route: an empty feed and an empty board are indistinguishable from *nothing happened*, so the page states a falsehood about the estate with full confidence. Degrading a subject does not make a page partial; it makes it wrong |
| **Substitute a zero or a last-known value for a failed region read** | A zero is a measurement claim: under ADR-0120 the address meter's numerator means *the subjects the batch walked*, so a substituted zero reports an uncovered scope on a database hiccup. A cached value is worse — stale by an unstated interval, with nothing on the page saying so |
| **Refuse the confirm step when its count fails** | `/seeds/delete` is reachable only through that block (`scope.tmpl:184`), so the refusal removes the operator's only route to a withdrawal, over a figure `deleteSeed` never reads. It converts a read fault into the loss of an operator capability |
| **A page-wide "some data is unavailable" banner instead of per-region notes** | It says something is missing without saying what, so every region becomes suspect. The per-region note is what keeps the rest of the page trustworthy, which is the whole benefit being bought |
| **Extend ADR-0124 §2's best-effort rule to cover this** | Its three premises — a third party, an outbound reach, a declinable feature — are all false of a read against our own store. A rule kept alive by an argument that does not reach it will be misapplied next |
