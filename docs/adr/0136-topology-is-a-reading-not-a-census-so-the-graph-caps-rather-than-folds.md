# ADR-0136: `Topology` is a reading, not a census, so the graph caps rather than folds

- **Status:** Accepted
- **Date:** 2026-09-02
- **Ticket:** [#1090 The graph is unreadable at scale: three unbounded columns, no aggregation, no level of detail](https://github.com/winniel123/verge-asm/issues/1090)
- **Follows:** [ADR-0105](./0105-inventory-is-a-read-over-the-open-span-corpus-not-a-second-thesis.md), which established that one corpus carries more than one projection and that a projection adds no store, observation, `Derivation` leaf or value
- **Constrained by:** [ADR-0051](./0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md) (a subject key is the thing denoted), [ADR-0072](./0072-absence-is-a-property-of-a-cell-and-withdrawn-is-the-only-population.md) (a listing states no denominator), [ADR-0131](./0131-the-console-is-vanilla-server-rendered-prg-and-the-htmx-stack-is-withdrawn.md) (the console is server-rendered PRG)
- **Relates to:** [#1089](https://github.com/winniel123/verge-asm/issues/1089), which framed the export and the minimap on the content bounds and gave the template `ContentW`/`ContentH`

## Context

`/graph` draws the estate in three fixed columns. `buildGraph` places every node at
`y = graphRowTop + idx*graphRowStep` — 44 plus 46 per node — with no cap, no wrap and no
aggregation. The columns sit at `graphColName = 130`, `graphColAddr = 560` and
`graphColSvc = 1000`, so the drawing is 1200px wide and as tall as the longest column demands.

A column of N subdomains measures `(N-1)*46 + 98` px. Fourteen nodes fill the 640px viewport.
A thousand fill 47 thousand pixels.

**This is a reachable state, not a hypothetical.** An address scope enumerates in full: every
address inside a declared CIDR is a subject from the declaration, and the hot tier walks every
one of them each cadence ([ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md)). The
default range size cap is 1,024 addresses per scope. So one declared `/22` produces a
1,024-node address column, and #1090 reports exactly that.

Nodes never collide — the 46px pitch prevents it. The failure is aspect ratio.

**The operator cannot zoom out of it.** The view JS clamps the scale to `[0.5, 2.5]` and the
reset control returns to `k = 1`. There is no fit-to-content. So on a 47,102px column the
operator sees 1,280px at the widest available zoom, which is 2.7% of the content, and reaches
the rest only by panning about thirty-seven screenfuls. #1090's own cost table describes zoom
levels between 1.4% and 28%. The UI can reach none of them. The screen fails harder than the
report states, and by a different route.

**Why this needed a ruling.** Each remedy changes what the screen means, so none of them is a
maintainer-free choice:

- A **cap** is cheap and honest if it states what it dropped, but it stops the graph being a
  census.
- **Aggregation** keeps completeness and adds a node no measurement produced.
- **Level of detail** changes only rendering, but does not fix the aspect ratio alone.
- **A different layout** is the largest change and discards the column semantics.

The blocking question was the second one: is a folded prefix node a rendering, or a `Subject`?

`Subject` has exactly four kinds — `Name`, `Address`, `Service`, `Endpoint` — and they split on
who supplies the key. A prefix rollup is none of them. No source delivers it, no measurement
produces it, and it has no lifecycle. Admitting one would either mint a fifth kind or place an
unkeyed thing in a drawing whose every other node is a subject. That is the cost the fold was
always going to carry, and it is why the fold loses below.

## Decision

### 1. `Topology` is a **reading**, and `Inventory` keeps the census promise

The graph answers *what does the estate point at* — the `Name` to `Address` resolution edges
and the `Address` to `Service` edges the open-span corpus already states. It answers that
question **for shape**, not for completeness.

This is ADR-0105's move made a second time. Inventory reads the open span across subjects and
renders the value it holds. Topology reads the same rows for the relationships between them.
Same corpus, a third projection. No new table, no new observation, no new `Derivation` leaf and
no new value in any facet's space.

The two projections make **different promises**, and that difference is the whole of this ADR.
`Inventory` shows every subject and states no denominator (ADR-0072). `Topology` may show
fewer subjects than the corpus holds. Where it does, **it says so**.

An operator who needs the census has one, on the screen that promises it. Nothing is hidden by
this decision — it is relocated to the surface that can carry it.

### 2. The graph **caps** and never **folds**

No address folds into its prefix. No subdomain folds into its parent. **The drawing holds no
node that is not one of the four `Subject` kinds.**

Where the population exceeds the cap, the graph draws fewer nodes and states the shortfall. It
does not invent a node to stand for the ones it dropped. A rollup would be a thing the model has
no kind for, produced by no measurement, and the operator would have no way to ask what it is.

`classifyNameTypes` already derives the domain apex tier off the observed name set, and that
stays. It is not a counter-example: an apex node **is** an observed `Name`, and the derivation
picks which of the four kinds a real subject is. It mints nothing.

### 3. The **bound is the operator's own `Seed`**

`/graph` takes a `?scope` token that draws one declared scope's sub-estate. This follows the
`?period` token on the Drift feed and stays inside ADR-0131's server-rendered PRG. It is the
**primary** instrument: it bounds the population by an operator act rather than by a product
rule, and it drops nothing.

The range size cap makes the two fit. One address scope is at most 1,024 addresses by default,
so one scope is one bounded drawing.

With no `?scope` given, the graph draws the whole estate under the cap in §4. A bookmarked
`/graph` keeps working and a small estate is unchanged.

### 4. The cap is **per column, deterministic, and derived — 20**

Twenty nodes per column, in the existing sorted-key order, first N.

**Per column**, because a cap over the connected structure cannot promise a bound: one `Name`
that resolves across a full scope re-opens the problem it was meant to close.

**Deterministic sorted order**, because it is stable across reloads and makes no product
judgment about which subject matters more. Ordering by severity would turn a shape reading into
a second triage screen — that is the Signals screen's job — and would make the drawing change
under the operator as rules fire.

**Twenty, derived.** A column of N subdomains measures `(N-1)*46 + 98` px. Fit to the 640px
viewport, an 11px label renders at `11 * 640 / height`. Twenty nodes render it at 7.2px and
twenty-one at 6.9px. So 20 is the largest column whose labels still clear the legibility
threshold in §5 when the drawing is fit to the viewport. The cap is not a round number someone
liked. It is §5's threshold solved for N, and the constant carries that derivation.

### 5. Label suppression shares that one threshold

Below the zoom at which an 11px label renders under 7px, the subdomain, ip and service labels
are suppressed. The domain apex label renders at 13px/600 and **always shows**, so the operator
never loses the answer to *which apex am I looking at*.

§4 and §5 are one constraint written twice, deliberately. A cap and a suppression threshold
chosen independently would drift apart and neither would be defensible on its own.

### 6. An edge the cap cuts is **counted and stated**

`buildGraph` today drops any edge whose endpoint has no position, silently. Under a cap that
guard stops being a defensive nicety and starts deleting real relationships. So the drop is
counted, and the count joins the shortfall statement.

The statement names the remedy. It tells the operator that a scope selection shows the rest.
Stating a truncation and naming its remedy is already the house habit: the Drift feed caps at
500 events and "states plainly when the cap truncated the view rather than dropping rows
silently".

### 7. The three columns stay

No wrap, no radial layout, no force layout. Bounded at 20 per column the drawing is 1200x972,
which is close to the viewport the screen was designed for. A new layout would solve an aspect
ratio problem that §3 and §4 have already removed, and would discard the left-to-right
`Name` → `Address` → `Service` reading that is the thing the screen teaches.

This is revisited only if the cap proves too tight in use.

## Consequences

- **The graph stops being a census, and says so.** An operator reading it for completeness is
  told, on the screen, that they are on the wrong surface, and where to go.
- **`Topology` enters the glossary.** The difference between its promise and `Inventory`'s is
  load-bearing after this ADR and had no home before it.
- **The cap and the label threshold move together.** Changing one without the other breaks the
  derivation in §4, so both live behind one stated constant.
- **A large estate now needs an operator act to be seen in full** — a scope selection. That is
  a real cost. It buys a first load that is legible instead of one that is a hairball.
- **The zoom repair is independent and lands first.** Fit-to-content and a content-derived clamp
  floor fix a defect that exists under every option in #1090, and commit to none of them.
- `graphColumnCap` is a **rendering** limit. It bounds no measurement, no scan and no
  enumeration. The hot tier still walks every declared address.

## Alternatives rejected

**Aggregation — fold addresses into their prefix.** Rejected. It keeps a completeness the
reading no longer promises, and it pays for that with a node that is none of the four `Subject`
kinds, that no measurement produced, and that the drawer could not describe. §1 makes the
completeness promise `Inventory`'s, which removes the only reason to want the fold.

**A cap chosen rather than derived.** Rejected. A round number invites the question *why 50 and
not 100* and has no answer. §4's twenty answers it: any larger column cannot render its own
labels at the zoom that shows it whole.

**Severity-first selection.** Rejected. It makes the graph a triage surface that Signals already
is, and the drawing would change under the operator as rules fire.

**A different layout — wrap, radial, or force.** Rejected for now, not on merit. It is the
largest change available and it addresses an aspect ratio that the cap has already bounded. It
also costs the column semantics. §7 leaves it open.

**Level of detail alone.** Rejected as a complete answer, adopted as a part of one. Suppressing
labels does not shorten a 47,102px column. It is §5 because it makes §4's bound legible, not
because it is a remedy on its own.

**Leaving `/graph` uncapped and relying on pan.** Rejected. Thirty-seven screenfuls of panning
is not a relationship view. The screen's whole claim is that it shows relationships together.
