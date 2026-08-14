# Drift is a timeline of spans, compared only within one derivation

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#8 Drift model: how change over time is represented and surfaced](https://github.com/winniel123/verge-asm/issues/8)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

This is the decision the product exists for. [#2](https://github.com/winniel123/verge-asm/issues/2)
and [#13](https://github.com/winniel123/verge-asm/issues/13) narrowed the differentiation to
one claim: change modelled as a **first-class, queryable object with its own lifecycle**,
across every facet rather than subdomains alone. reNgine already stores ports, addresses, DNS
and full certificate data, and still computes `new_subdomain_counts` with window functions over
scan pairs. If drift here is also a diff of two runs, there is nothing to build.

Six prior decisions arrive with obligations attached:

- **[#14](https://github.com/winniel123/verge-asm/issues/14)** made `Exposure` a conclusion
  constructed across vantages, so drift over it is drift over a computed value, and handed over
  the reachability lifecycle, hysteresis, and what a gap in observation means.
- **[#10](https://github.com/winniel123/verge-asm/issues/10)** required `Exposure` to be stored
  with the derivation version that produced it, and a *not-comparable* outcome that is neither
  changed nor unchanged.
- **[#15](https://github.com/winniel123/verge-asm/issues/15)** ruled that a subject first
  observed under a widened aperture is not "appeared", and left the mechanism here.
- **[ADR-0005](./0005-scan-execution-model.md)** handed over two obligations: read sibling
  batches from one dispatch *without reading the dispatch*, and refuse to treat a stale batch as
  current.
- **[ADR-0004](./0004-signals-are-release-coupled-rules.md)** established that a comparison is
  legal only where the effective rule version was unchanged, and that versions compose.
- **[ADR-0006](./0006-subjects-leave-by-measurement.md)** made membership a Derived view, split
  appearance into `appeared` and `returned`, and left a third member of that family unplaced.

## Decision

| Concern | Decision |
| --- | --- |
| Stored object | A **`Span`** — one row per period a value was held |
| Comparison unit | The **timeline**, never a batch pair |
| Ingest | A **fold over completed batches**, incremental |
| Timeline key | `(subject, facet, vantage, source)` — **one per source**; gains a facet-defined discriminator, see the [#36](https://github.com/winniel123/verge-asm/issues/36) amendment |
| Operator state on a span | **None** — spans are immutable |
| Comparison | Over a **canonical value**, structurally; a hash is only an index |
| Canonicalisers | **Versioned**, and they compose |
| Two values, no legal comparison | A **`Break`** |
| No value at all, for a period | A **`Gap`** |
| Materialised history | **Never re-derived** |
| Hysteresis / flap suppression | **Not in the model** — notification only |
| Currency of an observation | Within **`k` cadences** of the covering Declared `Scan` |
| `k` | **Fixed and release-coupled**, starting at 2 |
| Source conflict | **Reported, never resolved** |
| Aperture widening | A `Break`, plus a third appearance kind — **`revealed`** |
| Cascade closure | Writes a span, reason `cascaded` |
| Alerting | **On the cause, never per consequence** |

## Rationale

### The timeline is the comparison unit, not a batch pair

[#10](https://github.com/winniel123/verge-asm/issues/10) left this ticket the sentence "the
unit of comparison is a batch pair, and pairs are not always valid", which contradicts
[`CONTEXT.md`](../../CONTEXT.md)'s own rejection of `ScanRun` — *batches anchor scope, timelines
anchor comparison*. The glossary is right, and the batch-pair framing is `ScanRun` thinking one
level down: pair up two runs, diff them, discover the pairing is often invalid.

Under a timeline there is no pairing step to be invalid. Ingest is a fold: for each subject in a
completed batch's recorded scope, the observation either **extends** the open span, or **closes**
it and opens a new one. The batch never supplies the pairing — it supplies the **licence**, in
the form of its recorded scope, its vantage, its source's `completeness`, and whether it is
still current.

This is also how [ADR-0005](./0005-scan-execution-model.md)'s sibling-batch obligation is
discharged. Nothing ever groups by `Dispatch`, because nothing ever needs to: a derivation reads
*the latest current observation per vantage*, which is a fact about timelines and clocks, not
about which fan-out a batch belonged to.

### Spans, not events

The stored object is the **period**, not the change. A `Span` holds one value for one timeline
between an opening and a closing, and a **`Transition`** is the adjacency between two spans —
derived, never stored, because storing both is a second representation of one fact and the map's
standing seam rule argues against every one of those.

Three things decide it. Current state is a lookup rather than a fold, and current state is what
every signal evaluation, the `Subjects` screen and the ownership gate need constantly. Duration
becomes queryable — *this service has been exposed for eleven days*, *this one has flapped forty
times this week* — which is the sentence no count-shaped diff can produce, and therefore the
concrete form of the map's differentiation. And decisively: **a period of not-knowing is
naturally a span and unnaturally an event.** A vantage down for three days is a three-day span
of *we cannot say*; as an event log it needs an invented edge into ignorance and another back
out.

### One timeline per source, and what `authority` is actually for

The conflict that settles the key: the operator's zone file lists `old.example.com`; our own
resolver returns a Name Error from every vantage. Both sources are `enumerable` over scopes
covering that name. One timeline can hold one value, so a merged timeline must arbitrate — and
the arbitration rule would be a threshold we chose, sitting inside the comparison path, which is
the failure the three layers exist to prevent.

Keyed per source, both facts are recorded and reconciliation moves up into the Derived layer,
where `Exposure`, membership and `Ownership` already live. The rule that a `corroborative`
source may never close a span stops being a rule and becomes structural: such a source's
timeline simply *has* no closing events, because its silence covers nothing. Enabling a source
**adds a timeline** rather than perturbing existing ones. And comparability preconditions become
uniform within a timeline instead of varying row by row.

This is also where `authority` finally does work. It had been in the glossary since
[#7](https://github.com/winniel123/verge-asm/issues/7) without a single decision using it —
`completeness` and `consent` carried everything. It is **not** an ordering: if it were,
`declared` would outrank `measured` and a zone file would keep a dead name alive, which
[ADR-0006](./0006-subjects-leave-by-measurement.md) explicitly refuses. What it governs is
**admission** — whose word is enough to put a subject in the estate at all. ADR-0006 already
used it without naming it, in ruling that admission under a wildcard turns on `Citation`: a
certificate SAN survives, a guessed label does not.

So a conflict between two current values is **reported, not resolved**. Picking a winner is
smoothing, and [#21](https://github.com/winniel123/verge-asm/issues/21) established the opposite
habit — publish the weak tier rather than smooth it.

### `Break` and `Gap` are two different things, and `Shadowed` is a third

[`CONTEXT.md`](../../CONTEXT.md) and the map had accumulated eight separately-argued instances of
*we cannot say, and that is different from saying no*, with an open question about whether they
wanted one name. They do not. They sort by **what we have**:

| | We have | Shape | Instances |
| --- | --- | --- | --- |
| **`Gap`** | no value at all | a **span** | dead-lettered batch's empty scope, `Vantage` `unavailable`, `not-evaluable`, membership unsettled by a down vantage |
| **`Break`** | values on both sides, no legal comparison | an **edge** | derivation-version change, aperture widening |
| **`Shadowed`** | a value meaning *we cannot see* | an **observed value** | a name below a wildcard |

This is the line [#22](https://github.com/winniel123/verge-asm/issues/22) already drew —
*[#18](https://github.com/winniel123/verge-asm/issues/18) asks "we measured both, may we compare
them?", [#22](https://github.com/winniel123/verge-asm/issues/22) asks "did we measure it at
all?"* — which is exactly `Break` versus `Gap`. `Shadowed` is the odd one because it is the only
member that is a *value*, which is why ADR-0006 found it landing in the Observed layer while
every other instance is the absence of one.

Two instances leave the family on inspection. **`firewalled` versus `internal-only`**
([#14](https://github.com/winniel123/verge-asm/issues/14)) are both things we *can* say — a real
distinction between two derived states, swept in by resemblance. And **port-tier rotation** turns
out to manufacture nothing at all under the fold: a daily batch whose recorded scope excludes
port 5432 does not touch that timeline, so nothing closes and nothing breaks. That leaves `Break`
with exactly two causes.

[#22](https://github.com/winniel123/verge-asm/issues/22)'s *one treatment, four stated reasons*
is therefore confirmed as a presentation choice sitting over a model that genuinely has more
than one thing beneath it.

### History is written once and never re-derived

The Observed layer is append-only by nature. The Derived layer is append-only by **policy**: a
derivation version change does not rewrite the past, it **closes every open span of that
derivation and opens new ones under the new version, with a `Break` between them.** Nothing is
ever compared across a `Break`, so no transition is emitted and nothing is alerted.

This is what makes the exposure board legal, and it does so structurally rather than by
discipline. [#10](https://github.com/winniel123/verge-asm/issues/10) proposed reading the
never-diff-Derived rule as *same-derivation-or-no-comparison*; storing the version on the span
means a query cannot accidentally violate it, because the values on either side of a `Break` are
not the same kind of thing.

The cost is accepted and real: **a bug in a derivation leaves permanently wrong history that may
not be corrected in place.** A fix ships as a new version and a `Break`, which at least tells the
operator honestly why yesterday's board disagrees with today's. Recomputing instead would make
every past transition a function of today's release — the observer inside the comparison path at
the largest blast radius available.

### Comparison is over canonical values, and canonicalisers are versioned

[#4](https://github.com/winniel123/verge-asm/issues/4) handed over "the body-normalisation
function for drift hashing", and a hash is the wrong instrument for a product whose entire output
is *what* changed. A hash answers changed-or-not; the board's drill-down and the notification
payload need *the SAN list gained `dev.example.com`*. So the canonical value is stored and
compared structurally, with a hash kept only as an index and a short-circuit.

The more useful move is to recognise a canonicaliser as a **versioned derivation**, at which
point [ADR-0004](./0004-signals-are-release-coupled-rules.md)'s composition rule applies for
free. This neutralises the hazard that made canonical form a standing worry: serialising
certificate SAN entries in map order would otherwise re-diff every certificate in the estate
every run. As a versioned derivation it is a `Break` instead — the board says *not comparable*
and names the reason, which is true, rather than reporting forty thousand certificate changes,
which is a lie.

### No hysteresis in the model

[#14](https://github.com/winniel123/verge-asm/issues/14) offered two forms of damping: a
suspected-versus-confirmed rule borrowed from the dangling-DNS research, and hysteresis for
oscillating hosts. Both put a chosen threshold inside the comparison path, and
[ADR-0006](./0006-subjects-leave-by-measurement.md) has already refused exactly this shape once,
for membership — no counter, no invented intermediate state. A `suspected-exposed` span is
`unconfirmed` wearing a hat.

There is a positive argument too. Under spans, a flapping service produces many short spans, and
that is a **queryable fact the operator wants**. Damping at the model layer destroys it
permanently; damping at the notification layer does not. The drift history is what we measured;
the alert stream is what we judge worth waking someone for, and those are allowed to differ.

### Currency is `k` cadences of a Declared `Scan`

Three problems collapse into one rule. `Exposure` is computed across vantages that report minutes
apart, so recomputing naively on each arrival flaps `internal-only` → `exposed` every cycle.
ADR-0005 forbids the obvious fix, since sibling-batch membership is `Dispatch` knowledge.
Separately, ADR-0005 required that a stale batch not be treated as current, without saying when
staleness starts.

**An observation is current while it is within `k` cadences of the Declared `Scan` whose scope
covers that `(subject, facet, port, vantage)`** — the tightest such cadence where several apply.
Cadence lives on `Scan`, which is **Declared**, so reading it is legal in the way reading
`Dispatch` is not.

Cross-vantage timing then dissolves: at 02:00 the external vantage's last observation is
twenty-four hours old against a daily cadence, so it is current and is used, and no transition is
emitted. Tier rotation dissolves as described above. Past the bound the observation stops being
current, the derived value becomes non-constructible, and a `Gap` opens — the same mechanism as a
`Vantage` going `unavailable`, not a second one.

`k` is **fixed and release-coupled**, following ADR-0005's own blast-radius argument for the
availability window: this dial moves the whole board. It starts at 2 rather than 1 because
ADR-0005 treats skipped ticks as normal operation and
[#22](https://github.com/winniel123/verge-asm/issues/22) refuses to surface a single skipped
tick; at `k=1` every skip would open a `Gap`.

### Aperture has three inputs, and widening yields `revealed`

[#15](https://github.com/winniel123/verge-asm/issues/15) ruled that a subject first observed
under a widened aperture is not "appeared"; ADR-0006 split appearance into `appeared` and
`returned` and noted that a model with one appearance transition has nowhere to put the third
member. It lands in two places at once: the **`Break`** is the detection — a batch's recorded
source set differs from the prior one's for that scope, which is data
[#5](https://github.com/winniel123/verge-asm/issues/5) already requires — and **`revealed`** is
the record, a third opening kind on the membership timeline.

Both of [#15](https://github.com/winniel123/verge-asm/issues/15)'s riders then fall out of the
model instead of needing rules. *A release can trigger this, not just an operator*: a flipped
default appears in the batch's recorded source set exactly as an operator toggle does, so nothing
needs to know a release happened or consult an audit trail. *The risk is entirely on the enable
side*: every opt-in source is `corroborative`, and per-source keying means a corroborative
timeline has no closing events at all, so disabling one cannot manufacture a withdrawal. This is
the same trick [#14](https://github.com/winniel123/verge-asm/issues/14) used to make exposure
from an internal vantage impossible rather than forbidden.

Aperture is also broader than [#15](https://github.com/winniel123/verge-asm/issues/15) framed it.
It has **three inputs — enabled sources, port tiers, and the ownership gate** — and all three
produce `revealed` rather than `appeared`. [#36](https://github.com/winniel123/verge-asm/issues/36)
adds two more; see the amendment below.

### Alert on the cause, record the consequence

`Ownership` gets a timeline, which turns out to matter more than it looks. An `owned` →
`third-party` transition closes the probing gate ([ADR-0002](./0002-ownership-gates-probing.md)),
so every `Service` beneath that `Address` stops being measured. Those services must **not**
withdraw: ADR-0006 is unambiguous that subjects leave by measurement, and *ceasing to measure* is
not *measuring absence*. They enter a `Gap`. Getting this wrong retires a chunk of the estate
because a registry record changed.

The same shape appears twice more. ADR-0006's cascade rule takes every `Endpoint` beneath a
withdrawn `Name`; those spans must close, or a dead endpoint keeps an open span and every
current-state query returns it as live — but the closure is not independent evidence, so it
records reason `cascaded`, which is what lets the endpoints return coherently if the name does.
And a gate *opening* reveals a burst of services at once.

In all three cases one operator action or one registry change produces a burst of consequences,
so the rule is stated once rather than rediscovered three times: **alerting fires at the cause,
never per affected subject**, while every consequence is fully recorded.

## Amendment — [#36](https://github.com/winniel123/verge-asm/issues/36): the key carries a
discriminator, and aperture has five inputs

Two corrections, both of things this ADR asserted in substance and did not write down. Neither
disturbs the decision above.

**The timeline key is `(subject, facet, discriminator, vantage, source)`**, where the
discriminator is facet-defined and empty for every facet but `dns-record`, which carries the
qtype. Without it one `Name`'s `dns-record` timeline must hold every qtype in a single value, and
a batch that queried MX but not TXT writes a value asserting an empty TXT RRset it never measured
— [ADR-0009](./0009-verge-core-is-a-union.md)'s `{161}` defect arriving through the key instead of
the port list. The generalisation is evidence rather than convenience: `reachability` never needed
a discriminator **only because its subject `Service` already carries port and transport**, which
is the same fix applied one level up, hand-made, before anyone named it.

**Aperture has five inputs, not three.** Beside enabled sources, port tiers and the ownership
gate sit the **queried qtype set** and the **TLS candidate set** — the versions and ciphers a
handshake offers. Both start timelines that did not exist, so both yield `revealed` rather than
`appeared`, and pricing them here is what stops a release that adds CAA reporting a first-ever
record as `appeared`, or a TLS library upgrade that widens the client's offer reporting an
estate's worth of newly-accepted ciphers as change in the world. See
[ADR-0011](./0011-a-facet-is-six-parts.md).

*Corrected twice by [#54](https://github.com/winniel123/verge-asm/issues/54) /
[ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md).* The count is **six**,
`Vantage class` having joined at [ADR-0017](./0017-exposure-needs-both-legs.md). And *"both start
timelines that did not exist"* is true of the qtype set — a discriminator in the key — and **false
of the TLS candidate set**, which sits inside `tls-acceptance`'s value, so widening it moves a
running timeline and costs a `Break` rather than a `revealed`. The general rule: an aperture
widening **`Break`s the timelines it touches and `revealed`s the timelines it opens**, sorted by
whether the dimension sits in the key or in the value.

## Amendment — [#42](https://github.com/winniel123/verge-asm/issues/42): only `revealed`
generalises, and the gate does not open a `Gap` directly

Two corrections, recorded in full as [ADR-0014](./0014-only-revealed-generalises.md). Neither
disturbs the decision above.

**`revealed` is not *"a third opening kind on the membership timeline"*.** It belongs to any
timeline, because membership is a property of a **subject** while aperture is a property of
**looking**, and looking is per-timeline. `appeared` and `returned` stay membership-only. This ADR
had already committed to it in substance: the #36 amendment above rules that the qtype set and the
TLS candidate set *"start timelines that did not exist, so both yield `revealed`"*, and those are
facet timelines. An opening caused by neither the world nor our aperture — a `certificate` timeline
opening six days after its `Service` did, because TLS is on the weekly tier — is recorded, unnamed
and unalerted.

**A closing probing gate does not open a `Gap` directly.** *"They enter a `Gap`"* under **Alert on
the cause** above is amended: the gate stops **feeding** those timelines, the last value ages out
under the currency bound, and the `Gap` opens by the mechanism that already existed. Read the
original way, an operator toggling a `custody extension` off and on inside one cadence writes
`value → Gap → value` on every timeline beneath the address for a `Gap` during which no measurement
was ever missed. The attribution cost of the aged value is carried by the `Custody` timeline, which
is current from the instant of the toggle.

ADR-0014 also settles what this ADR left implicit: **`Gap` → value is an ordinary adjacency**, not
a fourth member of the opening family, and the `Gap` records its cause the way a cascaded closure
records `cascaded`.

## Amendment — [#48](https://github.com/winniel123/verge-asm/issues/48): the conflict lands on
`dns-record`, and only two enumerable sources can have one

Two clarifications, recorded in full as
[ADR-0020](./0020-a-conflict-needs-two-enumerable-sources.md). Neither disturbs the decision above.

**The worked example under *One timeline per source* names two sources and one name and not the
facet.** It is `dns-record`: the operator's zone file is a **reading** and cannot produce a
`resolution` value, which [ADR-0011](./0011-a-facet-is-six-parts.md) divided on walk-versus-reading
and which the two prober-decided values `Lame` and `Shadowed` settle independently.

**A conflict needs two `enumerable` sources.** *Reported, never resolved* governs the `source`
component of the key alone. A `corroborative` source holds no closing events, so a difference from
it is its own staleness; two `vantage`s measure different facts and **compose** — reading this
sentence as covering vantage would make [ADR-0010](./0010-exposure-composes-two-reaches.md) illegal,
and ADR-0006 already met the split-horizon case; and a `Seed` observes nothing at all. In v1
exactly one pair qualifies, which is what discharges the last consequence below.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md)'s three-layer table is amended.** *Derived — drifts? No,
  ever* is false as written, because **every `Span` is Derived** — even one folded straight from
  observations composes a canonicaliser version and `k`. The rule's intent survives exactly, in
  the form [#10](https://github.com/winniel123/verge-asm/issues/10) proposed: not *never diff
  Derived* but **never compare across differing derivations**, now enforced by a `Break` rather
  than by discipline.
- **Six terms enter the glossary** — `Span`, `Transition`, `Break`, `Gap`, and a defensive
  `Drift` entry whose main job is to say there is no `Drift` object. Without that entry someone
  builds a `Drift` table and [#7](https://github.com/winniel123/verge-asm/issues/7)'s `Finding`
  rejection is undone.
- **Spans carry no operator state.** An `Annotation` on `(subject, signal-name)` remains the only
  home for operator opinion; a drift record is a measurement, not a work item. If operators need
  to mark a transition seen, that is notification read-state, not a property of history.
- **The effective version of a Derived span composes at least four inputs** — the derivation
  rule, `Availability`'s version, the facet canonicaliser, and `k` — all release-coupled. Any
  release touching any of them breaks every span of that derivation estate-wide, and one `Break`
  inside the board's window makes the whole window not-comparable. This is the serious open risk
  and it is [#18](https://github.com/winniel123/verge-asm/issues/18)'s, re-scoped from *is this
  legal* to *how does the composite avoid moving every release*.
- **Per-facet canonical form graduates to its own ticket.** `resolution` is the hard one, carrying
  ADR-0006's four non-interchangeable outcomes, where three of the possible collapses are
  estate-scale.
- **Retention gains a hard floor.** The open span and the one preceding it can never be
  compacted: current state depends on the first and `returned` detection on the second.
- **A third precedent for drift that notifies as coverage.** After ADR-0006's `resolving →
  shadowed` come `revealed` and `owned` → `third-party`. At three instances the three
  notification classes are confirmed as a partition of **messages**, not of events.
- **Zone-declared names that do not resolve become visible** as a by-product of refusing to
  arbitrate between sources. That is a rule over observations, which is to say a `Signal`, and
  [#16](https://github.com/winniel123/verge-asm/issues/16) closed the v1 set — so it is recorded
  as an opportunity, not smuggled in here. **Discharged by
  [#48](https://github.com/winniel123/verge-asm/issues/48)** as two rules,
  `zone-declared-name-returns-name-error` and `resolved-name-absent-from-zone`; the set was never
  closed ([#35](https://github.com/winniel123/verge-asm/issues/35)).

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Drift computed on read from the observation timeline | Precisely reNgine's shape, which [#13](https://github.com/winniel123/verge-asm/issues/13) established we beat; and it makes history a function of today's release |
| Transition events as the stored object | Current state becomes a fold; duration becomes a query; and a *period* of not-knowing needs invented edges into and out of ignorance |
| Storing both spans and transitions | Two representations of one fact — a seam, and seams manufacture drift |
| Operator state on a drift record | `Finding` rebuilt with a better provenance story, re-admitting the ranked backlog [#16](https://github.com/winniel123/verge-asm/issues/16) refused severity to keep out |
| One timeline per `(subject, facet, vantage)`, sources merged | Forces arbitration between conflicting enumerable sources — a chosen threshold inside the comparison path |
| `authority` as a precedence ordering | Would let a zone file keep a dead name alive, contradicting [ADR-0006](./0006-subjects-leave-by-measurement.md) |
| One name for the whole not-comparable family | Three genuinely different things; a name spanning them stops carrying the specific rule at any site |
| Re-deriving history on a version change | Makes every past transition a function of today's release, estate-wide |
| Hash-based comparison | Answers changed-or-not; cannot render what changed, which is the product's entire output |
| Hysteresis or a suspected state in the model | An invented threshold in the comparison path, and it destroys the flap count permanently |
| Staleness as a fixed wall-clock duration | Wrong for a weekly tier and wrong for a daily one; cadence is already Declared and already correct |
| Operator-configurable `k` | A dial that silently makes the operator's whole board non-comparable — ADR-0005's own argument |
| Withdrawing subjects beneath a closed probing gate | Ceasing to measure is not measuring absence; retires the estate on a registry change |
| Alerting per cascaded subject | One decommissioned name becomes a burst; the root is the event, the cascade is its consequence |
