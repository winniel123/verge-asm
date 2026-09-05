# ADR-0177: a rule two surfaces must agree on lives in one shared function, and a second surface may never re-derive it

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1349 ADR gaps: cmd/web/reports.go](https://github.com/winniel123/verge-asm/issues/1349), gap 2, and
  [#1339 ADR gaps: cmd/web/seeds.go](https://github.com/winniel123/verge-asm/issues/1339), gap 4
- **Sweep PRs that deleted the comments:** [#1350](https://github.com/winniel123/verge-asm/pull/1350)
  (`cmd/web/reports.go`) and [#1340](https://github.com/winniel123/verge-asm/pull/1340)
  (`cmd/web/seeds.go`)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0145](./0145-design-system-is-the-shared-home-and-source-of-truth-for-ui-assets-and-a-session-edits-it-in-the-repo.md),
  which makes `design-system/` the source of truth and names what the web app actually embeds, and so
  fixes where the Go/JSX boundary in §3 falls
- **Not bound by:** [ADR-0149](./0149-a-consumer-takes-the-data-layer-interface-it-calls-and-the-seam-not-the-package-is-the-unit.md),
  which rules how *wide* a consumer's slice of the data layer is. Reach is not agreement, and a
  correctly narrow seam on each of two surfaces still lets them disagree
- **Narrows:** [ADR-0147](./0147-assets-watched-is-the-distinct-subject-count-over-the-open-span-corpus.md)'s
  Context correction that "the heatmap ramp has **one** consuming surface, not two". That records the
  count. This ADR rules what a second consumer would have to do

## Context

Two comments, deleted uncited by two sweeps, each asserted that a rule is shared rather than
per-surface. Neither rule is stated under `docs/spec/`, `docs/adr/`, `docs/guides/`, `docs/research/`
or `CONTEXT.md`, and both cited dead tokens — `P0.3`, which resolves to #444 ("Trend series for
Reports"), and `DF-F1`, which resolves nowhere.

**Instance one: the scan-activity fold.** `bucketScanActivity` (`cmd/web/reports.go:678`) buckets
`Dispatch` rows into per-day counts and derives the window and in-flight totals. It has exactly two
production callers: `foldScanActivity` at `cmd/web/reports.go:703`, behind the `/reports` page at
`cmd/web/reports.go:570`, and `/reports/export` at `cmd/web/reports_export.go:53`. The sharing is
real today.

**The issue's account of the ramp is wrong, and ADR-0147 already corrected it.** #1349 gap 2 says the
page and the export both "intensify through `drift.HeatLevels`". They do not.
`drift.HeatLevels` (`internal/drift/trend.go:67`) has one production caller,
`cmd/web/reports.go:710`. The export emits raw counts — `scans_per_day` CSV rows at
`cmd/web/reports_export.go:137`, `scans_per_day[].scans` at `:185` — and never reaches a level.
ADR-0147 records the same count at its `:47`–`:49`. The *fold* is shared across two surfaces; the
*ramp* is one surface's presentation of it.

**A live comment still asserts the wrong half.** `internal/drift/trend.go:69` reads *"One rule, so
the page fold, the export and any later surface ramp identically."* The export does not ramp at all.

**The thresholds are real, and they are split across two files.** The 0–4 level is
`internal/drift/trend.go:81`, a ceiling of `count × 4 / max` with `max` floored at 1
(`:70`), zero preserved at level 0 (`:78`) and the top clamped at `heatSteps = 4` (`:65`, `:85`).
The level-to-fill table is `pct := []int{0, 28, 48, 72, 100}` at `cmd/web/reports.go:709`, mixed into
`--surface` at `:720`, with a level-0 cell drawn as `--surface-sunken` over `--row-sep` at `:717`.

**The deleted comment's claim that these mirror `HeatmapCalendar.jsx` is TRUE, and the file is in the
tree.** `design-system/components/display/HeatmapCalendar.jsx:7` computes
`v <= 0 ? 0 : Math.max(1, Math.ceil((v / max) * 4))` over a `max` floored at 1 (`:4`), and `:8` carries
the identical table and `color-mix` expression. The Go and the JSX agree value for value.

**Instance two: the declared-token boundary.** `parseSeedTokens` lives at
`cmd/web/onboarding.go:38-42` — **not** in `cmd/web/seeds.go`, where both the issue and this ADR's
brief place it. Its rule is one line: `strings.FieldsFunc` splitting on `,` or `unicode.IsSpace`, so
commas, spaces, tabs and newlines all split and `FieldsFunc` drops empty runs — no trim, no case
fold, no dedupe. Its production callers are the `/scope` declare form
(`cmd/web/seeds.go:110`) and onboarding's two fields, `seeds` and `seedsadd`, both at
`cmd/web/onboarding.go:58`. `cmd/web/scope_bulk_test.go:15` pins the two against one input.

**Neither served surface forks it.** `design-system/templates/scope.tmpl:171` and
`design-system/templates/onboarding.tmpl:91` are plain text inputs that post raw; onboarding's script
(`:140-174`) only enables the Next button.

The prohibition is the half no function embodies. A shared function survives a comment sweep because
it is code. Nothing in the code stops a third surface from writing its own ramp or its own splitter.

## Decision

> **Where two or more surfaces must agree about the same fact, that fact is computed in exactly one
> shared function and every surface calls it. A second surface may never re-derive the rule, however
> small the re-derivation looks. A rule is shared when two surfaces present the SAME fact to a user
> and a disagreement between them would be a defect the user can see; where two surfaces present
> different facts that merely happen to compute alike, sharing them is a coupling, not a rule.**

### 1. The test, and both sides of it in this codebase

The test is a question about the user, not about the code's shape.

**Same fact — must share.** A day's scan count. The `/reports` heat cell carries it as a tooltip
(`pluralScans`, `cmd/web/reports.go:714`); the CSV carries it as a `scans_per_day` row
(`cmd/web/reports_export.go:137`). An operator can open the page and the export of the same
`?period` and read the two side by side. If a boundary condition — the UTC whole-day truncation at
`cmd/web/reports.go:680`, the newest-first row cap at `:31`, the out-of-span drop at `:693` — moved
in one and not the other, the two would print different numbers for one date. That is a visible
defect, so `bucketScanActivity` is one function.

**Different facts that compute alike — must not share.** `drift.HeatLevels` normalises day counts
against the busiest day; `reportsSignalCensus` normalises a severity bar against the busiest severity
at `cmd/web/reports.go:303`, `Pct: n * 100 / max`, over a `max` floored at 1 at `:290`. Same shape —
take the maximum of a slice, floor it at one, scale each element against it. Folding them into one
helper would make widening the heat ramp from four steps to five move a severity bar's width. The two
answer different questions on different axes and no operator compares them, so the duplication is
correct and the shared helper would be the defect.

**The rule is therefore not "share everything".** Two functions may be identical and stay separate.
The question is whether a user could hold their two outputs up against each other.

### 2. The tokenizer under the test

The `/scope` declare form and onboarding's tag field present the same fact: *what counts as one
declared scope*. An operator who pastes `a.com, b.com` into onboarding and gets two chips, then
pastes the same string on `/scope` and gets one scope, is looking at a defect. So the split rule is
one function, and the tokenizer's shape never becomes a per-form question.

The rule reaches the split boundary and stops there. What each surface does with the tokens after it
is its own: onboarding dedupes with `dedupeStrings` (`cmd/web/onboarding.go:58`), the declare loop
dedupes and refuses per token against the address cap (`cmd/web/seeds.go:119`–`:126`). Different acts
on the same tokens, not two answers to one question.

The shared function is named for the fact and not for its first caller, and it sits where every
surface can reach it: `drift.HeatLevels` is one package below the console. A rule extracted into a
helper hanging off one handler invites the next surface to skip it.

### 3. A surface is something the console serves, so the Go/JSX duplication is outside this rule

`design-system/designfs.go:11` embeds `templates/*.tmpl`, `tokens/*.css` and `fixtures/*.json`.
`components/` is not embedded and is never served, so `HeatmapCalendar.jsx` and `TagInput.jsx` are the
design's statement of a component (ADR-0145) and no operator sees a number they produce beside one the
console produced. Two consequences follow, both deliberate.

- **The ramp's Go/JSX duplication stands.** Go cannot call JSX, and the JSX is the *source* the Go
  port is checked against — a design input, not a second surface.
- **`TagInput.jsx:9`–`:21` states a different commit rule** — `trim()`, a trailing comma stripped,
  commit on Enter or `,`, no whitespace split — and that is not a defect. It is off the request path,
  and the served onboarding page tokenizes in Go.

### 4. Why this is not ADR-0149's seam

ADR-0149's subject is **reach**: how much of the estate a bug behind one boundary can touch. This
ADR's subject is **agreement**: whether two surfaces that already have their reach can print
contradictory answers. `/reports` and `/reports/export` both read through `cmd/web`'s `store` and call
one query, `ListDispatchProgress` — narrow by ADR-0149, and still free until this ADR to fold that
query's rows two ways, because the divergence happens above the seam, in the fold. The independence
runs both ways: a fact shared through one function says nothing about how much data it may reach.

### 5. `openSignalsCount` is a live violation

`/reports` renders "Open signals" from `reportsSignalCensus`, which counts at
`cmd/web/reports.go:282-289` and reaches the template at `:640` and
`design-system/templates/reports.tmpl:137`. `/reports/export` writes the same number from
`openSignalsCount` (`cmd/web/reports.go:116-126`) into `summary,open_signals`
(`cmd/web/reports_export.go:131`) and `kpis.open_signals` (`:157`).

Two functions, one fact, two surfaces, and a user can compare them. They agree today only because both
sum `len(c.Fired)` over `signal.EvaluateCorpus` of the same corpus, and nothing holds that in place: a
change to what counts as open — a severity floor, a dedupe by subject — lands in one and not the other,
and the page and the CSV then disagree about a number in the same band.

**The fix is one function.** `openSignalsCount` delegates:
`open, _, _, ok := s.reportsSignalCensus(r); return open, ok`. `buildSignalCorpus` still runs once,
and the export discards a severity slice it does not read. **This ADR changes no Go code**; the fix
ships as its own ticket.

## Consequences

- **`internal/drift/trend.go:69` is wrong on `main` and must be corrected**, not merely cited. Its
  claim that the export ramps is false at `cmd/web/reports_export.go:53`.
- **`cmd/web/seeds.go:109` and `cmd/web/reports.go:678` gain citations to this ADR**, so the next
  reader of either shared function finds the prohibition and not just the sharing.
- **Nothing enforces this.** No check fires when a new handler writes its own ramp.
  `cmd/web/scope_bulk_test.go:15` is the only test of this shape in the tree. Review carries the rule,
  and §1's test is the question review asks.
- **A future export that ships an intensity calls `drift.HeatLevels`**, rather than re-deriving four
  quartiles or copying the `pct` table out of `cmd/web/reports.go:709`.
- **One known violation is named and left standing** (§5), on the footing ADR-0149 gives
  `cmd/web`'s 178-method `store`.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A documented convention with no shared function** — write the rule and let each surface implement it | It is the state instance one is already in for the ramp, and instance two would be in without `parseSeedTokens`. A convention is checked only by a reader who knows to look, and the divergence it prevents is invisible until an operator compares two screens. This rule's whole content is that agreement must be structural; a convention makes it a review habit |
| **A golden test comparing the two surfaces' output** | It pins the outputs' equality, not the rule, so both derivations survive and every future boundary condition needs a row on each side. [ADR-0152](./0152-a-golden-corpus-locks-the-hermetic-fold-and-never-the-live-adapter-so-an-adapter-change-is-an-uncovered-move.md) measures the failure mode: a golden locks the fold it covers and leaves the move it did not cover uncovered. It also fires late — after both surfaces are written and disagreeing — where a shared function makes the disagreement unrepresentable |
| **Duplicate the constant with a comment telling the reader to keep them in step** | A `// keep in sync` restatement fails the comment policy's unrecoverable gate, and the sweep that deleted both source comments would delete it. It also depends on the editor of one copy reading a comment on the other, which is the one thing a copy guarantees they will not do. `pct` at `cmd/web/reports.go:709` and its JSX twin are the cross-language case where this is unavoidable; inside Go it is a choice |
| **Share on structural similarity — one helper wherever two functions compute alike** | Couples `reportsSignalCensus`'s severity bars (`cmd/web/reports.go:290`, `:303`) to `drift.HeatLevels`'s day ramp, so a change to the heat ramp's step count silently moves a bar on a different chart. §1's test exists to refuse exactly this: the two present different facts, and no user compares them |
| **Rule it on [ADR-0149](./0149-a-consumer-takes-the-data-layer-interface-it-calls-and-the-seam-not-the-package-is-the-unit.md)** as another clause about seams | §4's measurement: `/reports` and `/reports/export` are compliant under ADR-0149 — one narrow query each, through one seam — and still able to fold that query's rows two ways. Filing an agreement rule inside a reach rule would put it where the one case it exists for is already graded passing |
| **Let the second surface re-derive where the derivation is one line** | `openSignalsCount` is the measurement: five lines, correct today, already a second answer to a number the page prints. Size does not bound the divergence — a rule change lands in the function the author is editing, not in the copy elsewhere |
