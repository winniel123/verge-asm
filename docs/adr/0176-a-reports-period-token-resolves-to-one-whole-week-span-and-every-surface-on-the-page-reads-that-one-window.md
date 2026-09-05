# ADR-0176: a `/reports` period token resolves to one whole-week span, and every surface on the page reads that one window

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1349 ADR gaps: cmd/web/reports.go](https://github.com/winniel123/verge-asm/issues/1349), gap 1
- **Sweep PR that deleted the comment:** [#1350](https://github.com/winniel123/verge-asm/pull/1350)
- **Rests on:** [ADR-0177](./0177-a-rule-two-surfaces-must-agree-on-lives-in-one-shared-function-and-a-second-surface-may-never-re-derive-it.md),
  which rules the general case — a rule two surfaces must agree on lives in one shared function.
  This ADR does not restate that prohibition. It supplies the particulars ADR-0177 leaves open for
  `/reports`: the vocabulary, each token's span, and what an unrecognised token does
- **Rests on:** [ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md),
  line 103, which is why a window with no backing rows draws the design's empty pattern
  (`reports.tmpl:233`) rather than a shorter fabricated series
- **Not bound by:** [ADR-0158](./0158-a-read-only-console-screen-may-scope-its-rendered-rows-in-the-client-and-a-screen-that-submits-a-form-carries-its-scope-in-the-query-string.md).
  A period is not a view scope over rows the server already rendered. It moves the row cap, the fold
  width and the query bounds, so it must reach the server, and it does
- **Sibling of, and not ruled by:** the `/drift` feed's `?period` selector
  (`cmd/web/drift.go:68-97`), which shares this token vocabulary and gives it **literal** windows.
  §6 rules the divergence for `/reports` alone and does not move `/drift`

## Context

`cmd/web/reports.go` carried a nine-line declaration comment on `reportsPeriod` stating the whole
token-to-span table, the one-window rule and the retirement of an older `?weeks=` parameter. Its
only citation was `SPEC-CHANGE #23b`. `design-system/SPEC-CHANGE.md` is not on `main`, and `#23b` is
a design-collision id, not an issue — §4.7's third dangling family. The sweep deleted the block, and
the rule now has no statement anywhere: `docs/spec/`, `docs/adr/`, `docs/guides/`,
`docs/research/` and `CONTEXT.md` state no part of it.

**The code today.** `reportsPeriods` (`cmd/web/reports.go:46-53`) returns four presets. The spans
are:

| Token | Label | Weeks | Days folded |
| --- | --- | --- | --- |
| `24h` | `Last 24h` | 4 | 28 |
| `7d` | `Last 7d` | 12 (`reportsHeatWeeks`, `:36`) | 84 |
| `30d` | `Last 30d` | 26 | 182 |
| `90d` | `Last 90d` | 52 | 364 |

Every row is at `cmd/web/reports.go:49-52`. The mapping the issue gives is correct, and
`TestResolveReportsWindow` (`cmd/web/reports_test.go:730-738`) pins all four.

**The issue's vocabulary is incomplete.** It calls the vocabulary "24h/7d/30d/90d". There is a
fifth form. `resolveReportsWindow` (`:92-113`) accepts an ISO-8601 pair — either as `?start=` and
`?end=` (`:93`), or folded into one `?period=custom_<start>_<end>` token (`:94-98`,
`parseReportsCustomToken` at `:73-82`) — and converts it to whole weeks by rounding **up**:
`days := int(ed.Sub(sd).Hours()/24) + 1`, then `weeks := (days + 6) / 7` (`:104-108`). The template
carries the form that submits it (`design-system/templates/reports.tmpl:114-121`). So the resolver
has two entry forms and one output shape, and the output is always a whole number of weeks.

**The default's span is the design's, not the code's.** `design-system/fixtures/fixtures.json:2466-2467`
declares `"range_label": "Last 7d"` beside `"range_weeks": 12`, and its `periods` array
(`:2448-2465`) carries token and label with **no span at all**. The design labels its own
twelve-week activity view "Last 7d" and leaves the span to the server. `reportsHeatWeeks = 12`
(`cmd/web/reports.go:36`) is that number.

**One surface already shows the mismatch on screen.** The heatmap's accessible name is
"Scans per day, {{.RangeLabel}}" (`reports.tmpl:217`) and the axis label immediately under it reads
"{{.RangeWeeks}} weeks ago" (`:221`). Under the default that renders as *Scans per day, Last 7d*
over a grid captioned *12 weeks ago*.

**One KPI card is not windowed at all.** `reportsSignalCensus` (`cmd/web/reports.go:275-289`) takes
`*http.Request` only for `r.Context()` — `buildSignalCorpus` (`cmd/web/signals.go:878-892`) reads no
query parameter — so the open-signals count is a standing census as of now, and its delta compares
against `previousBatchInstant`, not against the previous period (`:339-347`). The template still
labels that card `{{.RangeLabel}}` and captions it "vs previous period"
(`reports.tmpl:135-138`). The other two KPI cards are genuinely windowed, off `windowStart` and
`doubleStart` at `:562-563`.

## Decision

> **`/reports` offers a fixed period vocabulary — four presets plus one custom ISO-8601 range —
> and every token resolves to one whole-week span. The one resolved window drives the KPI band,
> the trend chart, the new-assets bars, the scans-per-day heatmap and `/reports/export` alike, so
> no surface on the page can read a different period from the page.**

### 1. The vocabulary is closed at four presets plus one range form

`24h`, `7d`, `30d`, `90d`, and `custom_<start>_<end>`. Nothing else is a period. A new preset is
added to `reportsPeriods` and nowhere else, because that slice is what the picker renders
(`reports.go:670`, `reports.tmpl:111`) and what the resolver matches against (`:59-62`).

### 2. The span is whole weeks, and the four preset spans are the table above

The unit is a week because the folds are weekly: `reportsTrendBucket` is seven days
(`reports.go:130`), the heat grid is seven rows deep (`reports.tmpl:217`), and `days = weeks * 7`
at both call sites (`reports.go:559`, `reports_export.go:36`). A custom range rounds up to the
enclosing week, so a two-day range folds as one week and never as a partial column.

### 3. An unknown, absent or malformed value folds to `7d` and is never an error

`resolveReportsPeriod` falls through to `reportsDefaultPeriod` (`:56`, `:64-68`). A reversed or
unparseable date pair fails `:100-102` and falls through to the same place. A half-supplied range —
one of `start`/`end` — falls through too. `/reports` is a bookmarkable read, so a stale or hand-typed
link renders the default page rather than a 400. The trailing `return reportsPeriods()[0]` at `:69`
is unreachable while `reportsDefaultPeriod` names a member; it is a totality guard, not a second
default.

### 4. The window resolves once, in one function, and every consumer takes it as a parameter

`resolveReportsWindow` has exactly two callers in the tree: `reportsPage` at `cmd/web/reports.go:557`
and `reportsExport` at `cmd/web/reports_export.go:34`. There is no second copy and no second parse of
`?period`. Nine consumers read the resolved value and none re-derives it: the Dispatch row cap
(`reports.go:567`, `reports_export.go:50`), the heatmap fold (`reports.go:570`), the new-assets bars
(`:588`), the new-assets KPI and its delta (`:584-585`), the MTTW KPI, delta and sparkline
(`:615-631`), the signal trend chart (`:598`), the picker's own badge and preset marks (`:668-672`),
and the export's range header, `scans_per_day` rows and PDF period bounds
(`reports_export.go:36-40`, `:76-77`).

The export reads the token from the link the page renders — `/reports/export?format=…&period={{.Period}}`
at `reports.tmpl:124`, `:127` and `:128` — and `{{.Period}}` is `window.Token` (`reports.go:671`), not
the raw query value. A custom range therefore round-trips as one `custom_…` token even though the
operator submitted two parameters. Export format is orthogonal: `?format=` is `csv`, `json` or `pdf`,
defaulting to `csv`, and an unrecognised format is a 400 (`reports_export.go:24-31`).

**One consumer is missing from that list, and the ruling names it rather than covering it.** The
open-signals count and its delta take no window (`reports.go:275-289`, `:339-347`). The rule above
binds it prospectively — when that card is windowed, it takes this window as a parameter and does
not resolve its own — and it does not retroactively make the card's current `RangeLabel` caption
true.

### 5. `?weeks=` is retired, and `docs/guides/reports.md` still documents it

`?weeks=` appears in no `.go`, `.tmpl`, `.js` or `.json` file outside tests. The resolver reads
`period`, `start` and `end` and nothing else, so a `?weeks=` value is silently ignored rather than
rejected.

`docs/guides/reports.md:79-80` still shows it:

```
GET /reports/export?format=csv&weeks=12
GET /reports/export?format=json&weeks=26
```

Under [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) that
site is withdrawn at its own site. The replacement is `?period=`:

```
GET /reports/export?format=csv&period=7d
GET /reports/export?format=json&period=30d
GET /reports/export?format=pdf&period=custom_2026-08-01_2026-08-22
```

The drift is not cosmetic. `weeks=26` reads today as the **default** twelve-week window, so a guide
reader following line 80 takes a file half the width the guide promises, with no warning.

### 6. The token names the window the operator asked for; the span is the design's activity view

A reader who takes `7d` to mean "seven days of heatmap" is reading it wrong. The token is the
operator's *choice of range*, drawn from the design's own picker. The weeks are the *fold width*
the design pairs with that choice, and for the default the design states the pairing itself
(`fixtures.json:2466-2467`). The two are not the same quantity and the code does not pretend they
are: the label and the week count are rendered side by side at `reports.tmpl:217` and `:221`.

`/drift` gives the same four tokens literal windows — `7d` is `7 * 24 * time.Hour`
(`cmd/web/drift.go:78`). That divergence is real and it stays: a transition feed lists events inside
a window, while an activity heatmap needs enough columns to read a trend off. One token vocabulary,
two screens, two span rules, and each screen resolves its own.

**The label is nonetheless misleading to an operator**, and this ADR says so rather than defending
it. "Last 7d" over eighty-four days of cells is a claim a careful reader will disbelieve before they
disbelieve the picker.

## Consequences

- **Adding a preset or moving a span touches `reportsPeriods` alone**, and both surfaces move
  together. The export cannot drift from the page: they share a resolver, and the page hands the
  export its already-resolved token rather than the raw query.
- **An operator comparing a `Last 7d` heatmap against a `Last 7d` drift feed sees different date
  ranges.** This is the cost of §6 and it is paid on screen, not hidden.
- **A follow-up is owed on the label.** Two candidates: caption the heatmap with its own span
  ("84 days") instead of `RangeLabel`, or rename the preset. Both are design changes; neither is
  taken here.
- **A second follow-up is owed on the open-signals card**, which is labelled with the range and
  computed without it (`reports.go:275-289`, `reports.tmpl:135-138`). Either the card stops claiming
  the range or the census starts reading it.
- **A custom range is bounded only by the date format.** `weeks` is unbounded above and
  `reportsDispatchLimit` is `weeks * 250` (`reports.go:31-33`), so a hand-typed hundred-year range
  asks the database for a very large `LIMIT`. The `#nosec` note at `:32` argues only that the
  product fits `int32`, which is a different claim from "this query is safe to run".
- **The deleted comment does not return.** The rule has a document, so declaration position stays
  empty under the comment policy.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A free-form day count**, e.g. `?days=45` | The folds are weekly: `reportsTrendBucket` is seven days and the heat grid is seven rows. A 45-day request renders a ragged final column, and the picker cannot mark any preset as current because the value space is no longer enumerable. The design ships a preset list plus a range form, and a free-form count matches neither control |
| **Per-surface periods**, e.g. `?trend=30d&heat=90d` | Two figures on one page would answer two questions while sharing one badge, and every cross-reading an operator makes ("47 signals over the period the heat shows") becomes false. It also multiplies the export's problem: `/reports/export` would need one parameter per surface, and a link copied from the page would carry whichever subset the template happened to render |
| **Keep `?weeks=` alongside `?period=`** | Two parameters for one quantity need a precedence rule every consumer must know, and they can disagree: `?period=30d&weeks=4` has no honest reading, and whichever wins, the picker's checkmark (`reports.tmpl:111`) and the export link (`:124`) are computed from the token and would contradict the rendered figures. The retirement also cost nothing — `?weeks=` is a raw fold width with no label, so nothing could render a badge from it |
| **Give `7d` a literal seven-day span**, matching `/drift` | It matches the label and destroys the surface. Seven cells is not a heatmap, the trend chart collapses to one weekly bucket, and the design's own fixture pins twelve weeks against that label. It would also break the delta on every windowed KPI, which compares the window against the equally long window before it (`:562-563`) |
| **Reject an unknown token with a 400** | `/reports` is a bookmarkable read reached from links an operator keeps, so a renamed preset would turn every saved link into an error page. The fallback is also what makes the custom form safe: a half-filled range degrades to the default page instead of a validation error |
