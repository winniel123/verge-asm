# ADR-0178: the Drift feed caps at 500 events, states its truncation, and ships the whole period within that bound

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1363 ADR gaps: cmd/web messages.go, restore.go and drift.go](https://github.com/winniel123/verge-asm/issues/1363), gaps 4 and 5
- **Sweep PR that deleted the comment:** [#1365](https://github.com/winniel123/verge-asm/pull/1365)
- **Rests on:** [ADR-0158](./0158-a-read-only-console-screen-may-scope-its-rendered-rows-in-the-client-and-a-screen-that-submits-a-form-carries-its-scope-in-the-query-string.md), whose limb 1 admits a client-side view scope on a console screen that submits no mutating form and whose limb 4 rules that such a scope narrows the rendered window, never the estate. Drift's group collapsing and kind chips **are** that admitted shape and are not re-ruled here; [ADR-0131](./0131-the-console-is-vanilla-server-rendered-prg-and-the-htmx-stack-is-withdrawn.md), which settles **where the JS runs** — inline vanilla JavaScript over server-rendered markup, no fetch/swap layer — and leaves open **what the server ships**; [ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md), which makes the design package's *"composition … and copy"* the console's IA spec, so the callout's wording is its site and not this one's
- **Not bound by:** [ADR-0105](./0105-inventory-is-a-read-over-the-open-span-corpus-not-a-second-thesis.md), the citation the deleted comment carried. It rules that Inventory is a read over the open-span corpus, *"not a second corpus and not a second thesis"*, and states nothing about view JS and nothing about what a handler ships
- **Supplies the ground for:** [ADR-0136](./0136-topology-is-a-reading-not-a-census-so-the-graph-caps-rather-than-folds.md) §6, which borrows this cap as precedent while ruling the graph's own edge cap. That clause takes a pointer here under [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)

## Context

`cmd/web/drift.go` held the cap's reason and the whole-feed contract in two comment blocks. #1365 deleted both and left two uncited residues — `:148` (*"A 90d window on a mature estate has no natural bound, so the feed reads under a cap"*) and `:183` (*"The tmpl's JS toggles a group open, so the full period feed always ships, never a filtered one"*). Both are true. Neither says why the number is 500, and neither says why a server-side filter is forbidden.

**What the tree holds, read on 2026-09-05.**

| Fact | Site |
| --- | --- |
| `const driftFeedLimit int32 = 500` | `cmd/web/drift.go:150` |
| The widest **preset** is 90d | `cmd/web/drift.go:79` |
| A **custom** range parses any two `YYYY-MM-DD` dates and has no floor, so the widest reachable window is unbounded outright | `cmd/web/drift.go:114-135`, parse at `:123-124` |
| The read passes the cap to the query as `MaxEvents` | `cmd/web/drift.go:170-172` |
| The query cuts newest-first — `ORDER BY batch_at DESC, batch_id DESC, subject_kind, subject_key, facet, discriminator, opened_at` then `LIMIT @max_events` | `db/queries/span.sql:133-134` |
| The truncation flag | `cmd/web/drift.go:178` |
| The page states it: *"Showing the most recent {{.FeedLimit}} transitions for this period. Narrow the period to see older change."* | `design-system/templates/drift.tmpl:127-128` |
| The CSV export states it in a trailing row, and the handler logs it | `cmd/web/driftfeed.go:177-178`, `cmd/web/drift.go:283-284` |
| `driftPage`'s only post-read pass counts events and sets `Collapsed` after the second group. It filters no group and no event by kind | `cmd/web/drift.go:182-189` |
| The template's own `<script>` collapses a group, and filters by kind over `data-kind` on each rendered event | `drift.tmpl:208-215`, `:216-240`; attribute at `:142`, chips at `:125` |
| Both Drift routes are `GET`, and the template's only `<form>` is a `GET` navigation to `/drift?start=&end=` | `cmd/web/handlers.go:387-388`, `drift.tmpl:106` |

**ADR-0136 §6 does not suppress this record.** It is `Accepted`, and `docs/adr/0136-topology-is-a-reading-not-a-census-so-the-graph-caps-rather-than-folds.md:132-134` reads:

> Stating a truncation and naming its remedy is already the house habit: the Drift feed caps at
> 500 events and "states plainly when the cap truncated the view rather than dropping rows
> silently".

That is [`comment-policy.md`](../spec/comment-policy.md) §8.3's third measured shape — *"An ADR that states the rule only in its Context, quoting the code it is about to lose"* — one section over. The quoted clause is **verbatim from the comment #1365 deleted**, it is used as precedent while §6 rules the graph's edge cap, and it carries **the number without the reason**. After #1365 the quotation marks point at nothing on disk. A source suppresses only where it states the rule, and ADR-0136 states the graph's.

**The `ADR-0105` citation the deleted comment carried was wrong, and it is already gone from the template.** The same wrong citation stood at `drift.tmpl:12` in the design-owned header (*"view JS in tmpl, ADR-0105 precedent"*). PR **#1396** (`e1c8809`, the D3 asset sweep) deleted that whole header — not #1420. Nothing in the template cites ADR-0105 today, so no repair is owed there.

**One thing the code does not do.** For a **custom** range the query takes no upper bound (`db/queries/span.sql:109`, `:129`), so the 500 most recent events **since the start date** are read and `filterDriftRowsUntil` trims to the end date at `cmd/web/drift.go:176` — **before** `truncated` is computed at `:178`. A historical custom range therefore reads 500 rows that all post-date it, drops them all, and renders an empty screen with `Truncated` false. It tells the operator there is no change where there is change it never fetched.

## Decision

> **The Drift transition feed reads and renders at most `driftFeedLimit` = 500 events. The most recent events in the window win, and the page STATES that the cap truncated the view. Within that bound the server carries the WHOLE period feed to the template: the feed is bounded once, by recency, and never by a view predicate.**

### 1. The cap is 500 events, applied once, at the read

The number is a render and transport bound on the thesis screen, not a statement about the estate. The widest preset is 90 days and a custom range is unbounded, so the window an operator can ask for has no natural ceiling on a mature estate, and every span open and close inside it joins the feed. The screen must neither fail nor balloon.

500 is the largest feed that still renders as one scrollable document and returns as one bounded result set rather than a stream. It is **not** derived from a threshold the way [ADR-0136](./0136-topology-is-a-reading-not-a-census-so-the-graph-caps-rather-than-folds.md) §4 derives 20 per graph column, and this ADR does not pretend otherwise. It is a stated bound, generous against the observed feed and cheap to move (§5).

The cap is applied **at the query**, never by discarding rows in Go. One bound, one site.

### 2. Recency is the only ordering the cut may use

`db/queries/span.sql:133` sorts `batch_at DESC, batch_id DESC` before `LIMIT`, so the events that survive are the newest. The oldest tail is what is lost, which is the right tail to lose on a screen whose question is *what moved since last time*. Two properties follow, and are ruled rather than merely observed:

- **The cut is not batch-aligned.** The 500th row can fall inside a batch, so the oldest visible group may be a partial batch whose count pill states what was rendered, not what the batch holds. Rounding the cut to a batch boundary would make the bound depend on batch size.
- **The cut may never be re-ordered by severity, subject, or change kind.** That would make the omitted tail a product judgement about which change matters, which is the Signals screen's job.

### 3. The page states the truncation, on every carrier of the feed

Silence would let an operator read a capped feed as a complete period and conclude that nothing else moved. This ADR rules the **requirement**; the copy is [ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md)'s. The console states it in the `.dr-callout` and the CSV export states it in a trailing row. A new carrier inherits the obligation.

The statement must be **true**, and today it is not. It is suppressed on a historical custom range (Context, last paragraph), and it overstates on every window: the callout names `FeedLimit` while the classifier drops rows it cannot narrate (`cmd/web/driftfeed.go:27-30`, `:78-79`), so fewer than 500 transitions render. The honest number is `TransitionCount`, which the handler already computes at `cmd/web/drift.go:182-188`.

### 4. Within the bound, the server ships the whole period feed

`driftPage` bounds the window and does nothing else to it. `Collapsed` is a **rendered initial state**, drawn closed at `drift.tmpl:139` and toggled open by the template's own script; the kind chips hide and show already-rendered `.dr-ev` elements. ADR-0158 admits that shape and states what it costs the client. What it does not state is the converse obligation on the server, and this limb is it:

> **A server-side view predicate may not be added to `driftPage` without moving this contract.** No filter by change kind, family, subject, or facet may narrow `Groups` on the server while the client holds the same filter.

The ground is that the client cannot re-fetch what the server withheld. Every chip is a DOM toggle with no request behind it, so a server that shipped only `withdrawn` events would leave the other five chips inert with no route back to the hidden rows. The Movement tally, computed over the same rows at `cmd/web/driftfeed.go:31` and rendered under a *"This period"* heading, would silently become the filter's total rather than the period's.

A screen that genuinely needs a server-side predicate falls under ADR-0158 limb 1's second half: the predicate moves into the query string, the chips are withdrawn, and this contract is superseded at this site. It is not added alongside.

### 5. The condition under which 500 moves

**500 moves when the callout fires on a routine window.** If an operator on an ordinary estate hits the cap on the default 7d preset, the number is too small for the feed it bounds and it raises a truncation the stated remedy cannot answer — there is no narrower period worth offering. Raise it, and re-check that the page still renders as one document.

**500 does not move because a 90d window truncates.** That is the case it was chosen for, and the CSV export carries the same window for an operator who wants the rows outside the browser. Any move re-reads the four other call sites that share the constant, because they do not all render a page.

## Consequences

- **An operator with more than 500 events in a window loses the oldest tail of it, and is told so.** On a 90d window during an estate expansion that can be most of the period. Narrowing the period recovers the tail, because a narrower window re-reads under the same cap and reaches proportionally further back. `TransitionCount` and the Movement tally are computed over the capped rows, so they state **the window as shown**, never the period.
- **The truncation statement is wrong in two ways today.** Both are defects against §3, and neither is fixed here.
- **`driftFeedLimit` is shared by four other reads, and only one renders this page**: the previous-window compare (`cmd/web/drift.go:238`), the CSV export (`:274`), the read-only API (`cmd/web/api_v1.go:176`), and the run-outcome join (`cmd/web/scans.go:820-822`, which passes the zero instant and so reads the 500 most recent events estate-wide). This ADR rules the **console feed**. The run-outcome join is the site whose correctness, not its legibility, rests on the cap, and it is a separate defect on a separate ticket.
- **`cmd/web/drift.go`'s two residual comments gain citations**, and [ADR-0136](./0136-topology-is-a-reading-not-a-census-so-the-graph-caps-rather-than-folds.md) §6 gains a pointer here at its borrowed sentence, per ADR-0058.
- **No production behaviour changes by this ADR.** §1, §2 and §4 state shapes the code already has.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **No cap — read the whole period** | The widest reachable window is a custom range with no floor, and every span open and close in it joins the feed. The query result, the Go slice and the rendered DOM then all grow with the estate's whole history. The thesis screen is the one an operator opens first; it cannot be the one that times out |
| **Paginate the feed** | Buys the older tail at the cost of page state on a screen that already carries period state, and ADR-0158 limb 1 would force the cursor into the query string beside it. It also breaks the kind chips, which scope the rendered page and would silently come to mean *this page only*. The CSV export already serves the operator who wants every row |
| **Cap by batch — the N most recent batches** | The bound then depends on batch size, which is the estate's property and not ours. One large batch would either blow the page or, capped at one batch, return a feed shorter than the screen. Capping by event bounds exactly the thing that is rendered |
| **Filter by change kind on the server, and keep the chips** | The chips issue no request (`drift.tmpl:216-240`), so a filtered ship leaves five of six inert with no way back to the hidden rows, and the Movement tally would state a filter's total under a *"This period"* heading. The client cannot re-fetch what the server withheld |
| **Fold the older tail into a "+N earlier changes" row** | A count of events the screen cannot name is a number with no reading behind it, and it invites an operator to read a residue count as a severity signal. Drift's unit is a named change on a named subject |
| **Leave the rule in a code comment** | It was in one, in two places, and the sweep deleted both — which is how ADR-0136 §6 came to quote a sentence that no longer exists on disk. The rule binds `db/queries/span.sql`, `drift.tmpl` and five Go call sites, so it is `comment-policy.md` §8.2 gate B by construction |
