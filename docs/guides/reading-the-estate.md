---
title: Reading your attack surface
section: Getting started
order: 4
description: Where to look for each answer — coverage, exposure, drift, inventory, the graph, search, and a single scan run — and which route each lives on.
---

# Reading your attack surface

Once a batch has committed, the estate is readable from a handful of pages, and each
answers a *different* question. This guide is the **where do I look** companion to
[first-run.md](first-run.md), which is the **what does it mean**. first-run.md explains
the three layers — `Declared`, `Observed`, `Derived`. It also explains why `Exposure`
needs two legs, and how to tell a scan that ran from one that failed in silence. This
page does **not restate** them — it maps each question to the page and route that answers
it. The terms themselves (`Coverage`, `Exposure`, `Span`, `Gap`, `Inventory`, `Drift`)
are defined in [`CONTEXT.md`](../../CONTEXT.md). The phrasing below follows those
definitions.

Every read surface is **viewer-readable** — none requires the admin role. A read never
mutates. The only mutations that live on these pages (triggering a scan, marking a
message read) are flagged where they appear.

---

## The read surfaces at a glance

| The question you are asking | Page | Route |
| --- | --- | --- |
| *Is what I am looking at complete?* | Coverage | `/coverage` |
| *What do I have right now?* | Inventory | `/inventory` |
| *What is reachable from the internet?* | Exposure | `/exposure` |
| *What moved since last time?* | Drift | `/drift` |
| *How does it all connect?* | Graph | `/graph` |
| *Where is this one thing?* | Search | `/search` |
| *Did a scan run, and what did it touch?* | Scans / a run | `/scans`, `/run/<id>` |

The same measured corpus answers *what moved* and *what do I have*. One read diffs the
timeline over time. The other reads each open `Span` forward by subject
([ADR-0105](../adr/0105-inventory-is-a-read-over-the-open-span-corpus-not-a-second-thesis.md)).
They are two reads, not two stores.

---

## Coverage — what was and was not looked at

`/coverage` answers *is what I am looking at complete?* rather than *what is exposed?*.
It is where **we could not construct this claim** lives: `Gap`s, unread apertures, and
unevaluable rules. A scan that ran but resolved nothing appears here as a `Gap` — **not
as an error and not as absent data**. So on a first run, read Coverage *before* you
conclude the estate is empty. The full first-run drill is in
[first-run.md → Confirm a scan actually ran](first-run.md#confirm-a-scan-actually-ran).

Two things sit on the page:

- The **aperture statement** — one line per aperture input, its cadence, and whether it
  is on. An **address scope** has a `Coverage` denominator (every address in it is
  walked, so *no ports responded* is a fact). A **name scope** has none, because it
  enumerates nothing on its own. That asymmetry is the same fact stated twice.
- **`Gap`s and coverage messages** — a `Gap` is a `Span` holding no value, *the period
  over which we could not say*. It records its cause:
  - a dead-lettered batch,
  - a vantage gone `unavailable`,
  - an observation aged past its currency bound,
  - or a **blanket responder** whose reach cannot be attributed to a listener
    ([ADR-0104](../adr/0104-an-undiscriminated-reach-is-a-gap-and-a-blanket-responder-is-measured-not-listed.md)).

  A `Gap` never withdraws a subject — ceasing to measure is not measuring absence.

---

## Inventory and subjects — the current map

`/inventory` answers *what do I actually have right now?*. It reads the **open-span
corpus**: at most one open `Span` per `(subject, facet, discriminator, vantage, source)`
timeline. So each open span **is** the value that timeline currently holds — resolved
addresses, DNS records, the certificate chain, HTTP identity, TLS acceptance, rendered
inline. It is dated by the span's `opened_at` (*held since*), never *as of now*, and it
states **no denominator**. An estate's completeness is yours to know, not the product's
to assert.

A facet the system currently cannot value renders **as a `Gap`**, neither hidden nor
coerced — a blanket responder's ports read as *undiscriminated*, not open. A **withdrawn**
subject has no open span and so appears on no inventory listing. It is reached only by
its own key, which still shows its closed history.

The `Name`, `Address`, `Service` and `Endpoint` drill-downs — each opening into its facet
timelines — live under the subjects routes. Those routes are `/subjects/service`,
`/subjects/endpoint`, `/subjects/{key}`, and an individual asset at `/asset/{key}`.

> **Divergence to know.** The top-level `/subjects` listing that older docs (and
> [using.md](using.md#reading-what-it-found)) call the *Subjects* page now **301-redirects
> to `/inventory`**. The current-map listing was folded into Inventory under
> [ADR-0105](../adr/0105-inventory-is-a-read-over-the-open-span-corpus-not-a-second-thesis.md).
> The per-subject drill-downs above are unchanged. Read *Subjects* as *Inventory plus its
> drill-downs*.

---

## Exposure — the two-leg construction

`/exposure` is the reachability **conclusion** for a `Service`, composed from the
internet `Reach` and the internal `Reach` rather than observed by any single vantage. It
**exists only where both legs hold a value**, and its four states are a projection of
that 2×2:

| | internet `reached` | internet `not-reached` |
| --- | --- | --- |
| **internal `reached`** | `exposed` | `firewalled` |
| **internal `not-reached`** | `edge-only` | `unreachable` |

A **one-legged reading is not an `Exposure`** and gets no name. Where you never
provisioned an internet vantage, the surviving internal leg's `Reach` renders on its own
under *we never looked*. The page will **never** report `firewalled` or `exposed` for
something it did not observe from the internet. So a fresh single-host install shows an
empty or internal-only Exposure page until you add a prober. *Why* that second host is
mandatory is in
[first-run.md → Why `Exposure` needs two legs](first-run.md#vantage-class-and-why-exposure-needs-two-legs)
and [prober.md](prober.md). Exposure is a **current-state census, never a series** — it
carries no trend delta.

---

## Drift — the timeline of spans

`/drift` is the product's own subject: **what moved since last time**. A `Span` is one
period during which a timeline held a single value. `Drift` is what a `Transition`
between two adjacent `Span`s *is* — deliberately **not a modelled thing**. So there is no
drift table and nothing that accumulates state, only the fold read over the timelines.

The page groups transitions by `Batch` for a selected window. The period selector rides
`?period=` (e.g. `?period=all`), and the movement summary counts change by kind. Nothing
is compared across a `Gap` and no `Transition` crosses one. The values on either side
may be shown together but are labelled **undatable**. What is missing is *when* it moved,
not the licence to compare. A **"Batch detail"** entry opens the run behind the most
recent batch (see below). The change vocabulary and the reading of drift are toured in
[using.md](using.md#a-mental-model-to-keep).

---

## Graph — how it connects

`/graph` folds the estate's **open spans** into the real `Name` → `Address` → `Service`
topology and joins the live fired-signal census onto its nodes. It invents nothing. With
no subject measured, it shows the empty-state rather than a fabricated canvas. Use it to
see which service sits on which address behind which name, and which nodes carry an open
signal.

Two listboxes sit in the header, and only one of them changes what is drawn. The **scope**
listbox appears once a declared seed gives the page more than one scope. It rides
`?scope=`, and the handler reads it. A subject outside the chosen seed is dropped before
placement, so a scope narrows the fold. The **severity** listbox appears whenever the
canvas is not empty, and it narrows nothing. It runs in the page's own JavaScript, and it
offers *All severities* plus the five levels the model grades a signal by — **Critical**,
**High**, **Medium**, **Low**, **Info**. Picking a level leaves every node drawn. It hides
the open-signal halo on each node whose severity is not the picked one, and it greys that
node's minimap dot. So the control emphasises a level. It does not filter the drawing to
one. The choice lives in the browser and resets on any reload.

Three edges are worth knowing. A node carries the **worst** severity among its open
signals, so a node holding a `Critical` and a `Low` lights under *Critical* only. A service
node keeps its severity-tinted dot under every choice, because only the halo answers to the
listbox. And a node its column's cap left out is not drawn at all, so no severity choice
reveals it — the callout above the canvas counts what the cap held back
([ADR-0136](../adr/0136-topology-is-a-reading-not-a-census-so-the-graph-caps-rather-than-folds.md)).

A node opens a drawer beside the canvas. The drawer names the node, its type, its open
ports, its first-seen date, and its open signals. It links to no other page.

---

## Search — find one thing

`/search` is the full-page search the shell's command palette opens. The query rides
`?q=`. Results are bucketed by kind — **Assets** (current `Name` subjects),
**Signals** (fired members of the live corpus), **Batches** (recent runs), and
**Documentation**. Every row links to the route that already exists
(`/asset/{key}`, `/signals`, `/run/{id}`). It reads only what a viewer already reads
elsewhere. It mutates nothing.

---

## Reading a single scan run and the scans list

`/scans` is the read-only monitor over the queue: in-flight `Dispatch`es and their
per-job progress. It is where you confirm a scan is actually moving. The **on-demand
trigger** on this page (`POST /scans/trigger`) is the one mutation here and is
**admin-only**. A viewer sees the monitor without the trigger control.

`/run/<id>` is the per-run drill-in — one `Dispatch` (a fan-out of one `Scan`), reached
from a Drift *"Batch detail"* entry or a Search *Batches* row. The `<id>` is a plain
integer `Dispatch` id. A `Dispatch` is **Operational** — it records what the system
*did*, never what is true of the estate. So the comparison path never reads it. It is
here for operational visibility, and it is the corpus a wall clock may retire. An id that
does not resolve renders a *No such run* page, never a fabricated run.

---

## Exporting

Three read surfaces export to CSV — a read of the same facts the page already shows,
never a mutation, and viewer-accessible:

| Export | Route | What it contains |
| --- | --- | --- |
| Inventory | `/inventory/export` | one row per facet the open-span corpus holds |
| Signals | `/signals/export` | the current signal census the [Signals](signals.md) page evaluates |
| Drift | `/drift/export` | the transition feed for the active `?period=` |

Each fabricates nothing: an empty corpus yields a header-only file, never invented rows.
Each reads under the same cap its page uses, so an unbounded corpus cannot stream an
unbounded file. A truncated export carries a trailing marker row. Per-signal meaning is
in [signals.md](signals.md). Which discovery sources widened the aperture in the first
place is in [sources.md](sources.md).
