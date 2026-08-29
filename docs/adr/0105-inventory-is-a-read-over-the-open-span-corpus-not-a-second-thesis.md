# ADR-0105: Inventory is a read over the open-span corpus, not a second corpus and not a second thesis

- **Status:** Accepted
- **Date:** 2026-08-16
- **Ticket:** [#243 UI surfaces change but not inventory — operators can't see what they actually have](https://github.com/winniel123/verge-asm/issues/243)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

The product's subject is **change**, and [CONTEXT.md](../../CONTEXT.md)'s first sentence says so: *"its
subject is not inventory but change."* The UI is built to that thesis — the Subjects listing renders a
resolution outcome and a reachability verdict, the drill-down renders facet **timelines** and the
`Break`s between them. But an operator legitimately needs to answer a second question the same corpus
already holds — *what do I actually have right now?* — and today the UI never answers it. A
`dns-record` span renders as *"5 records"*, a `resolution` as *"Resolved"*, a `certificate` as a bare
verdict. The addresses, the records, the presented chain, the HTTP identity are all in
`observation.value` and folded into `span.value`, readable only from a Postgres session. #240 already
opened one seam — the drill-down expands a `dns-record` and a `resolution` span to its actual items —
but the gap is systemic: it spans every facet and there is no estate-wide *"here is what you hold"*
surface at all.

#243 was triaged `ready-for-human` because *committing the product to a second axis* is a maintainer
call, not a mechanical one, and it asked three questions with no defensible default: **does inventory
belong as a first-class axis in a change-first product. What is the IA. And how is "current state"
defined against the change/span model.** The maintainer's answer is **yes, as a first-class axis**.
This ADR is the design pass that ruling requires: it commits the axis, defines it entirely against
the existing model so it dilutes nothing, and pins what *current* means.

**The model already holds the answer, and holds it in exactly one place.** [ADR-0007](./0007-drift-is-a-timeline-of-spans.md)
folds observations into `Span`s. The `span` table's partial unique index
(`span_open_timeline_idx`, `db/migrations/19000_span.sql`) makes **at most one open span per
`(subject, facet, discriminator, vantage, source)` timeline** — and the migration says why in as many
words: *"current state is that one row, read as a lookup … what makes 'the open span is the current
state' a structural guarantee rather than a query convention."* Inventory does not need a new corpus,
a new observation, a new leaf, or a new value. **It is the open-span corpus, read forward by subject
instead of diffed over time.** That is the whole idea, and everything below follows from it.

## Decision

### 1. Inventory is a **read**, not a second corpus — the same `span` rows, projected differently

Change reads a subject's spans **down a timeline** and derives the `Break`s between them. Inventory
reads the **open** span **across subjects** and renders the value it holds. Same rows, two
projections. There is **no** new table, no new observation, no new `Derivation` leaf, and no new
value in any facet's space ([ADR-0015](./0015-the-value-space-is-the-commitment.md) is untouched).
This is what makes the axis *complementary, not diluting*: it commits the product to no new thing to
measure or version — only to **showing what is already measured**. The facet layer is evidence, not a
channel ([ADR-0026](./0026-the-facet-layer-is-evidence-not-a-channel.md)). Inventory is that evidence
read for its own sake rather than for the message a move would carry.

The read is `ListAllOpenSpans` (`db/queries/span.sql`): every span with `closed_at IS NULL`, ordered
by subject. Like every span read it is **not** routed through the live-tier observation gate
([ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md), #237): it
reads the already-derived, never-compacted `span` corpus, not the observation tier, so no `@as_of`
bound applies and settled current state is never hidden by the gate.

### 2. A subject's inventory is **the value of each facet's open span**, and *current* means exactly that

*Current state* is defined, with no residue of ambiguity, as **the value the open span on a timeline
holds**. The `span_open_timeline_idx` guarantees there is at most one, so "the current
`(facet, discriminator, vantage, source)` value" is a lookup, not an aggregation the read layer
invents. Inventory renders that value using the facets' **admitted closed sets**
([ADR-0011](./0011-a-facet-is-six-parts.md)) — `resolution`'s addresses, `dns-record`'s RRs,
`certificate`'s presented chain, `http-identity`'s status/Server/title/challenge/redirect,
`tls-acceptance`'s accepted versions and their selected suites — the same bytes the change views
summarise, shown rather than counted. `reachability` is the one facet with no per-item breakdown: its
whole value **is** its outcome (`reached`/`not-reached`/`Gap`), so it renders as a summary with nothing
to expand.

**Inventory is dated by `opened_at` and claims no freshness beyond it.** Every open span carries
`opened_at` — *the instant this value has been held since* — and inventory renders precisely that
(*"since 2026-08-14 09:00 UTC"*), never *"as of now"*. This is the honest reading of a currency-bound
model: a value's currency is a property of its facet's **cadence** ([ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md),
[ADR-0084](./0084-a-scan-is-a-cadence-over-an-exchange-and-an-uncovered-facet-has-no-currency-bound.md)),
and inventory states *what is held and since when*, not *what is true this second*. A facet with **no
currency bound** — a one-off measurement ([ADR-0044](./0044-a-one-off-measurement-has-no-currency.md))
or an uncovered facet whose Scan is disabled (ADR-0084) — is not special-cased and not hidden: it
holds an open span with an `opened_at`, and inventory shows the value dated by it, exactly as it shows
a cadenced one. The date is the currency statement. Inventory never manufactures a stronger one.

### 3. A **withdrawn** subject holds no inventory — the axis and membership agree by construction

A withdrawal **closes** a subject's timelines ([ADR-0082](./0082-a-withdrawn-subjects-timelines-close-and-the-withdrawn-period-is-on-no-timeline.md)):
after it, the subject has **no open span**. So a withdrawn subject is simply **absent** from the
open-span read — it needs no membership filter, no `NameError`/`Shadowed` suppression logic
([ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md)) re-implemented
at the inventory layer, and no denominator. Inventory and estate membership rest on the **same fact**
(an open span exists), so they cannot disagree. A withdrawn subject remains reachable by its own key
on the **change** view (its drill-down), which still renders its closed history — inventory is *what
you have*, the drill-down is *what happened*, and a withdrawn subject belongs only to the second.

This is also why inventory needs no membership re-derivation for the `NameError`/`Shadowed` case: a
Name whose latest resolution suppresses its membership has that outcome as the value of its **open**
`resolution` span, so it appears in inventory carrying that outcome as its stated current value — an
honest *"what this holds now"* — while the addresses it no longer cites have themselves left the
estate (their timelines closed) and so are absent. The open-span corpus already encodes the whole of
it.

### 4. A **`Gap` is inventory** — a value the system currently cannot state, shown as such

An open span may be a `Gap` — the period over which the system **could not say**
([CONTEXT.md](../../CONTEXT.md) `Gap`). The freshest instance is [ADR-0104](./0104-an-undiscriminated-reach-is-a-gap-and-a-blanket-responder-is-measured-not-listed.md):
a blanket responder's reach is undiscriminated, and *an undiscriminated reach is a `Gap`*. Inventory
renders a `Gap` **as a `Gap`** — neither hidden (which would read as *nothing here*) nor coerced into
a fabricated value (which would read as *reached* on a blanket responder). This is the same discipline
ADR-0104 rules at the measurement, read forward: a `Gap` is not a `reachability` value, so a blanketed
`Service`'s ports do **not** read as open in inventory — the axis inherits ADR-0104's damping for
free, with no special case, exactly as that ADR's §3 anticipated the *"#243 open count"* would.

### 5. The IA is **one estate-wide surface plus inline expansion**, not a parallel per-subject page

The surface is `GET /inventory`: the estate's open spans grouped by subject, each subject's facets
listed with the **actual value** expandable inline to its individual records — the #240 expand-seam
(`spanDetails`) **generalised to every facet**. The per-subject *"what this holds now"* view is **not**
a new page duplicating the drill-down. It is the drill-down's existing **Current** rendering, which now
expands *every* facet's open span (not only `dns-record`/`resolution`) through the same shared seam.
Estate-wide inventory and per-subject inventory are therefore **one mechanism** — `valueLabel` +
`spanDetails` over an open span — surfaced in two places, so the axis adds a read and a template, not a
parallel model.

Like the Subjects listing, `/inventory` **states no denominator**
([ADR-0072](./0072-absence-is-a-property-of-a-cell-and-withdrawn-is-the-only-population.md)): how much
an estate *ought* to hold is its completeness, which only the operator knows, so the screen states no
total and renders no count-of-subjects.

### 6. Change stays the thesis; inventory is a projection of it

[CONTEXT.md](../../CONTEXT.md)'s opening — *"its subject is not inventory but change"* — is **kept, and
qualified at its own site** ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)):
change remains what the product is *about* and the only thing the comparison path reads, and inventory
is a **read forward over the same derived corpus**, never a diff and never a second thing to model. The
sentence would be *wrong*, read alone, only if it were taken to mean the corpus cannot answer *"what do
I have"* — and it can, because the open span is the current value. So the amendment adds the
complementary clause and leaves the thesis standing.

## Consequences

- **No migration, no new corpus, no leaf-count change, no version move.** Inventory is a `SELECT … WHERE
  closed_at IS NULL` and a renderer. The `span` schema, every facet's value space, the `Derivation`
  leaf count, and every timeline's vector are all untouched, so **no timeline `Break`s** on account of
  this axis. This is the cheapest possible way to ship the second axis, and it is cheap precisely
  because the axis is a read.
- **`ListAllOpenSpans` is added to `db/queries/span.sql`** and threaded through the `store` interface.
  It is a plain, ungated span read alongside `ListOpenSpansForSubject` and `ListSpansForSubject`.
- **`valueLabel`/`spanDetails` (`cmd/web/subjects.go`) become the shared inventory renderer** and gain
  `certificate` (its presented chain), `http-identity` (its admitted closed set), and `tls-acceptance`
  (its accepted versions and selected suites) expansions. Because these are the same functions the
  drill-down timelines use, the drill-down's Current rows now expand those facets too — the systemic
  completion of #240. `reachability` stays summary-only: its whole value **is** its outcome, so it has
  no per-item breakdown.
- **A `Gap` reads as a `Gap` on the new surface**, so ADR-0104's blanket-responder damping needs no
  inventory-specific code: a `Gap` is not a `reachability` value and never renders as an open port.
- **CONTEXT.md gains an `Inventory` term and its opening sentence gains a complementary clause.** The
  glossary term pins *current = the open span's value, dated by `opened_at`*, and cross-references
  `Span`, `Drift`, and `Gap`.
- **The axis is honest about currency without modelling it further.** Inventory shows `opened_at` and
  never a synthetic *"as of now"*, so it inherits each facet's real currency (ADR-0028/0044/0084)
  rather than asserting one. No staleness badge is introduced in v1. The date is the statement.
- **Address subjects have no drill-down yet**, so an `address`-kind open span (none is produced today)
  would render in inventory as plain text with no link — additive and forward-compatible with a future
  Address surface, never a blocker.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Model inventory as its own corpus / materialised table** | The open span already **is** current state, by a structural unique index (`db/migrations/19000_span.sql`). A second store would duplicate it, drift from it, and need its own retention story — for a fact ADR-0007 already keeps in one place. Inventory is a read, and paying for a corpus buys nothing |
| **Add a `current-value` facet or a third value to a space** | There is nothing new to measure — the value is the open span's. A new facet or value would move `Derivation` vectors and `Break` timelines (ADR-0008) to render data that already exists, widening the committed value space (ADR-0015) for a pure rendering need |
| **Define *current* as "latest observation"** | The observation tier is live-gated and never compared directly (CONTEXT.md `Observed`); "latest observation" is a re-derivation the gate exists to forbid (#237, ADR-0041). The **span** is the derived, gate-free current value — the only definition of *current* the model actually licenses |
| **Show inventory for withdrawn subjects too, filtered by a membership re-derivation** | A withdrawal closes the timelines (ADR-0082), so there is no open span and nothing to show — and re-deriving membership at the read layer would duplicate ADR-0086's composition and risk disagreeing with it. Absence falls out of the open-span read for free; the withdrawn subject's history stays on its change drill-down where it belongs |
| **Hide `Gap`s from inventory, or coerce them to a value** | Hiding a `Gap` reads as *nothing here*; coercing it reads as a fabricated value (a blanket responder's ports as *reached*). ADR-0104 rules the `Gap` is the honest statement — *we cannot say* — and inventory must carry it as such, or it re-opens the exact false surface ADR-0104 closed |
| **A per-subject inventory page separate from the drill-down** | Two renderings of one subject's current facets would fork and diverge. The drill-down's Current section already renders the open span; generalising its expansion to every facet makes it the per-subject inventory, and `/inventory` is the estate-wide read over the same seam. One mechanism, two surfaces |
| **Reverse the thesis — make inventory co-equal in CONTEXT.md's opening** | Change is still the only thing the comparison path reads and the only thing the product *versions*. Inventory is a projection with no model of its own. Overstating it as co-thesis would invite future work to *diff inventory*, which is just change wearing the wrong name. The clause is complementary, deliberately subordinate |

## Thin ground, flagged rather than smoothed

- **No staleness surfacing in v1.** Inventory dates each value by `opened_at` and stops there. An open
  span whose facet is uncovered (ADR-0084) or one-off (ADR-0044) can be arbitrarily old and inventory
  says only *since when*, not *and possibly stale*. Reading the `opened_at` against the facet's cadence
  to render a staleness signal is a real, defensible next step and is deliberately **not** taken here —
  the honest date is the v1 commitment, a staleness verdict is a later one.
- **Estate-wide grouping is by subject kind, flat within a kind.** A large estate renders a long list
  with no server-side search or pagination on `/inventory` in v1 (the Subjects listing has search. This
  read does not yet). The read itself is a single ordered `SELECT`, so adding a search/scope filter is
  additive. It is unbuilt, not designed against.
- **`address`-kind inventory is forward-declared, not exercised.** No `address` facet produces a span
  today, so the unlinked-plain-text rendering for that kind is specification, not `[measured]`. It
  costs nothing and blocks nothing, but it has not run.
- **The currency claim rests on `opened_at` being the value's true age, which is exactly what a `Span`
  means** — but a facet whose cadence changed mid-life carries an `opened_at` from before the change,
  and inventory does not annotate that. It is the same honesty the drill-down already ships (it shows
  `since` too), inherited rather than newly incurred, and flagged so no one reads the date as a cadence
  guarantee.
