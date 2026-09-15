---
number: 2027
title: "An inventory row counts a reading, and a per-vantage facet names its derived class"
slug: an-inventory-row-counts-a-reading-and-a-per-vantage-facet-names-its-derived-class
date: 2026-09-15
status: accepted
source: grilling
ticket: 2027
proof: {ticket: 2027}
relations:
  - {kind: rests-on, adr: 11}
  - {kind: rests-on, adr: 103}
  - {kind: rests-on, adr: 167}
  - {kind: rests-on, adr: 1985}
---

# ADR-2027: An inventory row counts a reading, and a per-vantage facet names its derived class

## Decision

**An inventory facet row counts a reading, never a subject. The listing renders one row
per open span, and it collapses nothing.**

Where the discriminator is empty and the row carries a vantage, the label names the
derived vantage class. Where two rows of one subject and facet share a class, both also
name their vantage. Where the row carries no vantage, the label stays bare. The rule holds
for every facet with an empty discriminator, not for `reachability` alone.

A reader asks why the label names a class where the span key names a vantage. A person
reads the label, and ADR-0103 derives a provisioned prober's name from its endpoint.

Rejected: collapsing the per-vantage rows, an arbitration the span key refuses. Rejected:
the vantage name as the label, which leaks that endpoint.

Reversal restores a listing that renders two identical labels over contradictory
summaries.

## 1. Context

The `Span` key and the inventory label disagree about what names a row, and
[#2027](https://github.com/winniel123/verge-asm/issues/2027) found the first subject where a reader
sees it.

`CONTEXT.md` keys a span on `(subject, facet, discriminator, vantage, source)`, and states that the
discriminator is empty for every facet but `dns-record`. So the vantage is a key part, and for five
facets it is the **only** part that separates two rows of one subject.

`cmd/web/inventory.go#inventoryFacetLabel` takes the facet and the discriminator alone. It never
reads a vantage. `cmd/web/inventory.go#buildInventory` already stores one on every row, as the `van`
field of `cmd/web/inventory.go#inventoryFacet`, and nothing reads that field.

The result is two rows with one label. `internal/measure/connectoutcome/emit.go#EmitService` sets no
discriminator, so a service measured from two vantages renders `reachability` twice. The condition
reaches `reachability`, `resolution`, `tls-acceptance`, `certificate` and `http-identity`.

[ADR-1985](./1985-a-timeline-row-on-a-per-vantage-facet-is-refused-by-the-database-without-a-vantage.md)
is what exposed it. Before that effort the seeder wrote one reachability span with a NULL vantage,
so the design corpus could not reach the case. The corpus now models a producible estate, and the
service card reads `reached` above `not-reached` under one label.

## 2. Why a reading, and not a subject

The alternative is a collapse: fold the per-vantage rows of one subject and facet into a single row,
the way `cmd/web/subjects.go#buildAssetPorts` folds the spans of one port. It loses on three counts.

**The span key already refuses the arbitration.** `CONTEXT.md` states the grain at the `Span` entry:
one timeline per source, *so two sources that disagree hold two true facts rather than forcing an
arbitration*. A collapse is an arbitration performed in a render, over a key the estate deliberately
did not collapse.

**It has no winner rule.** `reached` and `not-reached` from two positions are both true. Nothing in
this decision says which one the single row would carry, and inventing a precedence here would state
a rule about exposure inside a listing that renders no exposure.

**The class-keyed collapse we already own hides rows.** `cmd/web/vantageclass.go#collapseReachLegs`
keeps only the most recent row per derived class. That is correct for the asset page, whose unit is
a leg pair, and it is wrong for a census: a third vantage would vanish from the listing with nothing
to say it had.

The asset page is untouched by this ruling. Its collapse answers its own question.

## 3. Why the derived class, and not the vantage name

The vantage name is the obvious label, and it is the one thing the row may not say.

[ADR-0103](./0103-a-vantage-is-one-position-and-the-prober-is-optional-provisioning-detail.md) holds
one `vantage` row carrying a mandatory measurement identity and an optional prober connection, and
it derives a provisioned prober's name from that connection as `username@host:port`. That separation
is the point of the ADR. A listing row that prints the name prints the endpoint, so a page whose job
is to census an estate would publish the provisioning detail of the machine that measured it.

The derived class is the axis the estate already reasons on. `internal/custody/gate.go#VantageClass`
closes it at `internet`, `internal` and `unverified`, and the exposure surfaces, the asset page and
the signals all key on it. `internal/vantageclass/vantageclass.go#Derive` is the one seam that
computes it, from the presented-address facts and never from the vestigial `vantage.class` column.

The class is coarser than the vantage, so two vantages of one class still collide. That is why both
rows of a within-class tie also name their vantage. The common case stays clean, and the tie stays
honest rather than silent. The name is a last resort here, not the default, which is what keeps the
ADR-0103 separation intact for every row that does not need it.

## 4. Why a bare label survives

ADR-1985 made the vantage structural on a per-vantage facet, and its predicate is a deny-list with
two exemptions: a row needs a vantage unless its `source` is `zone`, or its facet is `certificate`.
So a row with an empty discriminator and no vantage is still representable, and it has no class to
name.

The label stays bare there. That is not a gap in the rule, it is the rule applied: an unattributed
reading names no position, and the estate holds at most one such row per subject and facet, so it
collides with nothing. `certificate` is the live case.

The rule is stated over the discriminator and the vantage, not over a list of facets. A facet added
later inherits it on the day it is added, and a facet that is genuinely not per-vantage keeps a bare
label for the same reason `certificate` does.

## 5. What this pins in the design corpus

[ADR-0167](./0167-a-design-corpus-a-live-read-cannot-produce-is-served-as-a-pinned-fixture-and-the-live-path-renders-the-honest-projection.md)
§1 licenses a pinned fixture only where a screen's figure has **no first-class read behind it**, and
it states that an empty database is not a qualification.

The inventory's read exists. `design-system/fixtures/fixtures.json` nevertheless pins rows the read
cannot return. Its Addresses group carries `reachability` rows whose discriminator holds a vantage
name, which no producer writes, and whose value holds an `answers` outcome outside the closed
reachability pair. `internal/queue/pure.go#subjectKindFor` maps `reachability` to the `service` kind,
so the fold writes no address-kind reachability span at all. Its `dns-records` rows carry an empty
discriminator, where `dns-record` is the one facet that carries a qtype.

Those rows hold no licence under ADR-0167 today, and this ruling is what makes them load-bearing: a
corpus that is the source of truth for UI work would otherwise pin a label rule against readings no
fold can produce. They are repaired in the same pull request that implements this decision.

## 6. What reversal costs

Reverting is a narrower `inventoryFacetLabel` and a `van` field nobody reads. No stored row changes,
and no other surface moves.

What comes back is the render this decision removes: two rows, one label, contradictory summaries,
and nothing on either row saying which position produced which reading. A reader cannot tell the two
apart, and the pinned corpus asserts that shape is intended output.

That is cheap to reverse mechanically and expensive to reverse in meaning, which is why the rule is
recorded here rather than left beside the label function.
