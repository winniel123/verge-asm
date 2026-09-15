---
number: 1985
title: "A timeline row on a per-vantage facet is refused by the database without a vantage"
slug: a-timeline-row-on-a-per-vantage-facet-is-refused-by-the-database-without-a-vantage
date: 2026-09-14
status: accepted
source: grilling
ticket: 1985
proof: {ticket: 1985}
relations:
  - {kind: rests-on, adr: 17}
  - {kind: rests-on, adr: 14}
  - {kind: rests-on, adr: 129}
---

# ADR-1985: A timeline row on a per-vantage facet is refused by the database without a vantage

## Decision

**A `span` or `observation` row on a per-vantage facet cannot be stored without a
`vantage_id`. A `CHECK` on both tables refuses it.**

The predicate is a deny-list. A row needs a vantage unless its `source` is `zone`, or its
facet is `certificate`. A facet added later is covered by default, and it must argue its
way out.

A reader asks why, because no production writer emits the row. The answer is that a render
already lied about one. The asset page read `never looked` over a measured service, because
the class read inner-joins `vantage` and the census read carries no vantage predicate.

Rejected: a fifth `LegStatus` word for an unattributable reading, which is vocabulary ahead
of reality. Rejected: a left outer join, which under this `CHECK` admits nothing.

Reversal drops two constraints, and restores a representable state that no fold writes.

## 1. Context

A `Reach` leg renders four states after
[#1961](https://github.com/winniel123/verge-asm/issues/1961): `reached`, `not reached`,
`never looked` and `stopped looking`.
[ADR-0014](./0014-only-revealed-generalises.md) separated *no timeline* from `Gap`, and
[ADR-0017](./0017-exposure-needs-both-legs.md) built `Exposure` on that separation. A leg that was
never configured is our aperture. A `Gap` is our failure. `internal/exposure/exposure.go#LegStatus`
keeps the two apart on purpose.

[#1985](https://github.com/winniel123/verge-asm/issues/1985) found a fifth situation with no word.
**We looked, and we cannot say from where.**

Two reads disagree, and the asset page is the first surface that renders both.
`db/queries/signals.sql#ListServiceReachabilitySpansByClass` inner-joins `vantage`, so a `service`
/ `reachability` span whose `vantage_id` is NULL never reaches the leg map. The asset census comes
from `db/queries/span.sql#ListAllOpenSpans`, which filters on `closed_at IS NULL` alone. The row
survives the census, its legs do not survive the class read, and `cmd/web/exposure.go#legFrom`
returns the not-present branch for both. The page then reads `never looked / never looked` over a
service the estate measured.

`/exposure` does not show this. It builds its rows from the class read alone, so a row there cannot
outlive its own legs.

## 2. No writer can produce the row

This is the fact that decided the shape of the answer. The state is representable in the schema, and
it is reachable from no production writer.

- `internal/queue/pure.go#subjectKindFor` maps the `reachability` facet to the `service`
  kind, so the only producer is a `connect-outcome` job.
- `internal/scan/cold.go#BuildColdJobs` and `internal/scan/hot.go#BuildHotJobs` both return early on
  an empty vantage list, and both stamp the loop's vantage id. Neither emits the zero value.
- `span.vantage_id` and `queue_job.vantage_id` carry a plain `REFERENCES vantage (id)` with no
  `ON DELETE SET NULL`, and `db/queries` holds no vantage delete. A vantage cannot be removed out
  from under a span.
- The producers that do enqueue a NULL vantage — `crtsh`, `cttail`, `zone` and the edge fan-out —
  emit only facets that are deliberately not per-vantage.

The one violating row in the estate is the seeder's. `cmd/web/seedfixtures.go#seedInventoryFixtures`
writes `vantage_id` NULL and `source` `'resolver'` for every fixture span, including the rows whose
facet the fold writes as `'prober'` and per vantage. A fixture that models an estate no producer can
create manufactures a phantom bug, and this issue is the proof of that.

So the decision is not a repair of a live corruption. It is a decision about which of two
representations of one invariant the estate keeps.

## 3. Why the schema, and not a render

The invariant is *a per-vantage reading names the vantage it came from*. Three places could hold it,
and only one holds it once.

**A fifth `LegStatus` value** teaches the vocabulary a state no fold can write. `Reach`
would carry a word whose only instance is a seeder artefact, and every reader of the enum
would then have to learn why. §5 takes this apart.

**A census filter**, in SQL or in Go, is a second mechanism for one invariant. A second mechanism is
how the census read and the class read drifted apart in the first place. It is also untestable here:
a filter that cannot fire has no test that distinguishes it from its absence.

**A `CHECK`** puts the invariant where the partial unique index of the span migration already puts
"the open span is the current state". One expression, enforced on write, and no read has to agree
with any other read to keep it true. Every reader inherits it.

## 4. The predicate is a deny-list, and it fails closed

> **Amended** by [ADR-2028: An exception on the per-vantage CHECK names a writer that a test proves exists](./2028-an-exception-on-the-per-vantage-check-names-a-writer-that-a-test-proves-exists.md), 2026-09-15. <!-- adr-marker amends 2028 -->

The constraint could be written as an allow-list — name the per-vantage facets, and require a
vantage for those. It is written the other way. A row requires a vantage **unless** it is
zone-sourced or a `certificate`.

The producer table is the reason. `certificate` is mixed inside one source, because
`connect-outcome` sets the vantage and the edge fan-out nulls it under
[ADR-0129](./0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md).
`dns-record` is mixed across two sources, because the resolution walk sets a vantage and the zone
reader does not. So neither a facet-only predicate nor a source-only predicate states the rule, and
both exceptions have to be named at the exception with their citation.

The deny-list decides what happens to the facet nobody has written yet. Under an allow-list a new
per-vantage facet passes in silence, and the estate acquires the same hole under a new name. Under a
deny-list it is constrained on the day it is added, and a genuinely vantage-less facet must
argue its way onto the exception line. Silence fails closed.

## 5. Why `Reach` gains no fifth word

ADR-0017 rules that `Exposure` exists only where both legs hold a value, and that the two
absences are different objects. The proposal here was a third absence: *measured,
unattributable*. It loses on two counts.

It is vocabulary ahead of reality. §2 shows that no fold writes the row, so the word would name an
artefact of a seeder rather than a fact about an estate. A reader of `LegStatus` would then carry a
case that the corpus cannot produce, and the next session would have to rediscover why.

It also does not survive the alternative it competes with. Under the `CHECK` the row does not exist,
so there is nothing for the word to name. The two decisions are not independent: taking the schema
answer removes the referent of the vocabulary answer.

The left outer join loses for the same reason, and for one more. Under the `CHECK` it admits
nothing. It would also change no render if it did admit a row:
`cmd/web/vantageclass.go#collapseReachLegs` keys legs by the derived class, and
`internal/vantageclass/vantageclass.go#Derive` derives a NULL-vantage row to `unverified`.
The asset row reads only the `internal` and `internet` keys.

## 6. The constraint is plain, not `NOT VALID`

A `NOT VALID` constraint would let a developer's existing seeded database survive the migration.
That is declined.

Production holds no violating row, because §2 shows no writer that can make one. The only violating
rows are in a developer's own database, and they came from a seeder that this decision rewrites in
the same effort. A plain constraint makes the migration fail loudly on that database, and the web
binary panics at boot the way a duplicate goose version does. The developer re-seeds.

The price of `NOT VALID` is a constraint that is true of new rows and silent about old
ones, which is the weaker of the two guarantees and the harder one to reason about later.
The price of a plain constraint is one re-seed, once, by the people who can read the error
message.

## 7. The stale comment this corrects

The span migration's `NULLS NOT DISTINCT` index comment says the shipped resolver position "carries
no vantage row, hence a NULL vantage_id". That is no longer true.
[ADR-0202](./0202-a-vantages-resolver-comes-from-its-row-alone-and-the-code-side-fallback-is-removed.md)
made the vantage row the sole source of a resolver, and the resolution walk now fails with
`ErrNoResolver` rather than falling back. The shipped resolver position **is** a vantage row, and it
is the only vantage insert in the migration set.

The `NULLS NOT DISTINCT` clause keeps its purpose under a different facet. The real NULL-vantage
timeline is `dns-record` written by the zone reader under the `zone` source. The clause holds that
timeline to one open span, exactly as before.

## 8. What reversal costs

Reverting is two `DROP CONSTRAINT` statements. It restores a state that is representable
and that no fold writes, so nothing starts working that did not work before, and no stored
row changes.

What comes back is the disagreement between the two reads. A row could again survive the census and
lose its legs, and the asset page could again assert *we never looked* about a measured
service. That assertion is the harm, and it is an aperture claim, which
[ADR-0095](./0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md)
rules must count what the instrument cannot report. A false one is worse than a missing reading.

This decision is therefore cheap to reverse mechanically and expensive to reverse in
meaning, which is why it is recorded here rather than left in a migration comment.
