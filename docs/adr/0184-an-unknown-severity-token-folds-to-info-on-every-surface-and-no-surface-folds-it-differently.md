# ADR-0184: an unknown severity token folds to info on every surface, and no surface folds it differently

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1300 ADR gaps: internal/message](https://github.com/winniel123/verge-asm/issues/1300), gap 3
- **PR that deleted the comment:** [#1299](https://github.com/winniel123/verge-asm/pull/1299)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0116](./0116-the-design-package-is-normative-for-look-and-functionality.md). It rules
  that the design package is normative for look and functionality, and that where the domain lacks a
  datum the design renders, the fix is to build the datum. It built the five-level grade and withdrew
  `CONTEXT.md`'s older "a signal carries no severity" clause. A closed set is what makes an
  out-of-set token possible at all
- **Rests on:** [ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md). It rules
  that the design-system examples are the console's IA spec, ported verbatim, and states that severity
  is exactly `Critical / High / Medium / Low / Info` via `SeverityBadge`. That fixes the set this ADR
  folds onto
- **Bounded by:** [ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md). It rules
  that a message names what moved, and that the message vocabulary carries no valence word and no
  severity. Its subject is the `Message`, which has no grade to fold. This ADR binds the `Signal` grade
  and never reaches the message store
- **Bounded by:** [ADR-0114](./0114-the-report-pdf-is-rendered-in-process-from-the-artifact-not-from-html.md). It rules
  that the report PDF is a second **layout** of one `Artifact`, authored separately from the HTML. This
  ADR is why that separation is safe for the grade: both layouts call one normaliser, so a second
  layout cannot invent a second fold
- **Sibling of, and not ruled by:** [ADR-0183](./0183-the-severity-ramp-label-is-the-one-graded-word-the-product-draws-and-the-valence-refusal-does-not-reach-it.md). It rules
  what a grade may be called once it is normalised. This ADR rules what an out-of-set token normalises
  to. One is a vocabulary rule and one is a normalisation rule. Neither contains the other
- **Sibling of, and not ruled by:** [ADR-0185](./0185-a-severity-is-the-operator-facing-grade-so-it-composes-into-no-version-vector-and-is-not-a-fifth-part-of-a-rule.md). That ADR rules a re-rating out of the version vector, which is what lets a rule's grade move while its history stays comparable. A moved grade is one way a stored token goes stale, so that ADR supplies this one's population. It rules nothing about how a stale token renders

## Context

`internal/message/render.go:361` carried this, until #1299 deleted it:

```go
// normSev folds an unknown severity token to info rather than manufacturing
// urgency (mirrors signal.SeverityFor's calm fold and SeverityBadge's default),
// so a stale level never collides with critical.
```

The sweep kept a shortened form at `render.go:276` — *"An unknown token folds to info rather than
manufacturing urgency (mirrors signal.SeverityFor)"* — and dropped both of the clauses that carry the
rule: the third site it binds, and the collision the fold prevents. `pdf.go:190`'s *"An unknown token
folds to info"* went with the block around it. Nothing on disk states the rule.

### The token can be out of set, and the set is closed

[ADR-0116](./0116-the-design-package-is-normative-for-look-and-functionality.md) built the grade and
[ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md) closed the
set at five. `design-system/components/display/SeverityBadge.d.ts:3`–`:4` states the closure in the type,
with the comment *"Exactly these five levels — never synonymize"*. A closed set with a re-rating
history is exactly the shape that produces a stale token:

- A severity is **a property of the rule, identical for every instance it raises**
  (`internal/signal/severity.go:3`), and **a re-rating is deliberate and stays out of the version
  vector, so censuses stay comparable** (`severity.go:4`). So a re-rating changes a grade under rows
  already written, and nothing versions the change.
- `signal.SeverityFor` keys on the **rule name**, over three registries (`All`,
  `AllEndpointRules`, `AllServiceRules`). A renamed or retired rule resolves to nothing.
- The `Artifact` layer keys on the **token string** carried on `ArtifactSignal.Severity` and
  `ArtifactSeverityCount.Level`. Those arrive from a delivery record, so the token in hand can be older
  than the code reading it.

### Three folds exist, and they already agree

| Site | Input it folds | Result | Line |
| --- | --- | --- | --- |
| `internal/message.normSev` | a token string, against `artifactSevLevels` | `"info"` | `internal/message/render.go:275` |
| `internal/signal.SeverityFor` | a rule name, against three rule registries | `SevInfo`, and `false` | `internal/signal/severity.go:30` |
| `SeverityBadge` | a `level` prop, against `LEVELS` — and an absent prop, by default parameter | `"info"` | `design-system/components/display/SeverityBadge.jsx:5`, `:6` |

A fourth site folds the **order** rather than the value:

```go
func (s Severity) Rank() int {
	for i, sv := range SevOrder {
		if sv == s {
			return i
		}
	}
	// An unknown severity sorts last rather than colliding with critical at rank zero.
	return len(SevOrder)
}
```

`internal/signal/severity.go:18`. The explicit `return len(SevOrder)` is the whole content of that
site: the loop's natural fall-through in Go is the zero value, and rank zero is `SevCritical`.

**`normSev` reaches both render forms of the report.** `render.go:286`, `render.go:301` and
`render.go:345` on the screen form, `artifactdoc.go:141` and `artifactdoc.go:151` on the email/doc
form, and `pdf.go:68` and `pdf.go:147` on the print form. ADR-0114 authors the print layout separately
from the HTML, and this one function is what stops the two layouts from disagreeing about a grade.

### The collision is measurable, and it points at `critical` twice

**In the sort order.** `Severity.Rank()` is read at four sites that pick a **minimum** rank:

| Site | What it decides |
| --- | --- |
| `cmd/web/graph.go:504`, `worstSeverity` | the `data-sev` a graph node draws, and its halo |
| `cmd/web/subjects.go:1181`, `assetHeaderSeverity` | the grade in the asset-detail header |
| `cmd/web/seeds.go:527` | the grade rolled up onto a declared-name tree node |
| `cmd/web/reports.go:869` | the order of the delivered report's signal rows |

With rank zero on an unknown token, one stale token anywhere in an asset's signal list wins every one
of those minimums. The asset header, the graph node and the tree node all paint at the top of the ramp,
and the report's rows sort that signal first. `cmd/web/reports.go:146` reads
`sev.Rank() <= signal.SevHigh.Rank()` to set `Elevated`, so the stale row would also count as elevated.

**In the rendered grade.** `artifactSevLevels` (`render.go:273`) and `SevOrder` (`severity.go:16`) both
list `critical` first. A fold written as *take the first member* — the shortest correct-looking
normaliser over an ordered set — folds every unknown token to `critical`.
`artifactSeverityBadge` (`render.go:347`) special-cases `critical` as the ramp's only solid fill, so
that fold paints the loudest element in the system for a token nobody recognises.

**Both collisions are with the same member, and it is the member that costs the most.** The fold's
direction is not arbitrary and is not a tie-break. It is a refusal.

### An absent grade is not an unknown grade

`cmd/web.sevLabel` (`cmd/web/signals.go:671`) returns `""` for an empty token and title-cases anything
else. It is not a fourth fold, and it must not become one: `assetHeaderSeverity` returns `""` where a
subject has **no** open signal, and `search.go:199` reads a map that misses. Rendering `Info` there
would assert a grade nobody computed, which is the fabrication ADR-0116 §2 refuses at the empty state.

## Decision

> **An unknown or stale severity token folds to `info` — the calmest member of the ramp — on every
> surface that renders a grade or orders by one. The fold is identical on every surface, and no surface
> may fold differently. It prevents a collision with `critical` in the rendered grade and in the sort
> order alike. The reason is one sentence: a stale level must never manufacture urgency.**

### 1. The fold target is `info`, and the reason is the failure it refuses

A token the code does not recognise carries no information about the world. The product must draw
something, so it draws the level that claims the least.

The alternative directions all claim something the product did not measure. Folding to `critical` says
*look at this first* about a rule that may not exist. Folding to `medium` says *this is mid-scale*,
which is a claim about a scale position the token never had. Failing the render says *we cannot show
you this signal*, which reports less than we measured — the failure
[ADR-0010](./0010-exposure-composes-two-reaches.md) declined when it refused to make
internally-observed defects `not-evaluable` in order to express a grade the model then lacked.

`info` is the only member that is safe to be wrong about. A signal wrongly rendered `Info` is still in
the list, still sortable, still readable, and still one click from its rule. A signal wrongly rendered
`Critical` moves an operator, and it moves the asset header, the graph halo and the report's first row
with it.

### 2. The fold is over the token, never over the finding

The fold normalises **how a grade is named and ordered**. It does not change what fired, what is
stored, or what any rule concluded. `SeverityFor` returns `(SevInfo, false)`, and the second return is
the whole distinction: the caller can always tell a real `info` from a fold. `cmd/web/seeds.go:522`
reads it and skips the row rather than rolling a folded grade into a subject's tree.

Nothing damps a rule, nothing is suppressed, and no timeline moves.
[ADR-0007](./0007-drift-is-a-timeline-of-spans.md)'s refusal of model-layer damping is untouched,
because this rule sits at the render and never at the model.

### 3. One fold, and no surface may fold differently

Three sites fold today and they agree. That agreement is now a rule rather than a coincidence.

A second fold is worse than no fold. Two surfaces that disagree give an operator two grades for one
signal, and the operator has no way to tell which surface is lying. The report says `Info`, the graph
node draws `Critical`, and the record that would settle it is the stale token neither surface shows.

**A new surface reuses one of the three, and does not write a fourth.** In Go, that is
`internal/message.normSev` for an `Artifact` token and `internal/signal.SeverityFor` for a rule name.
In the design system, that is `SeverityBadge`. A helper downstream of a fold — `cmd/web.sevLabel` is
the example — may pass its input through, because its callers have already folded.

### 4. The order fold is part of the rule, not a separate one

`Severity.Rank()` folds an unknown token to `len(SevOrder)`, which is last. That is the same refusal
expressed over the sort key: the value fold keeps a stale token off the loud end of the ramp, and the
order fold keeps it off the top of every list. A ruling that covered only the value would leave
`worstSeverity` and `assetHeaderSeverity` free to hand a stale token the whole asset header.

### 5. What this rule does not reach

- **An absent grade.** A subject with no open signal has no grade. It renders as nothing, and it does
  not fold to `Info`. §Context states the three sites that rely on this.
- **The `Message`.** [ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md)
  gives a message no severity, so there is no token to fold. `CONTEXT.md`'s `Message` entry says so.
- **What a grade may be called once folded.** That is
  [ADR-0183](./0183-the-severity-ramp-label-is-the-one-graded-word-the-product-draws-and-the-valence-refusal-does-not-reach-it.md).
- **Whether a rule's grade is right.** A severity is a property of the rule
  (`internal/signal/severity.go:3`), and re-rating one is a rule change with its own review.

## Consequences

- **This ADR changes no Go code, no template and no test.** All four fold sites already behave this
  way. The ADR states the rule and names the sites it binds.
- **`internal/signal/severity_test.go:43` is the one test that asserts the fold**, and it asserts one
  half of it: `SeverityFor("no-such-rule")` must return `(info, false)`. No test asserts `normSev`'s
  fold, `SeverityBadge`'s fold, or `Rank()`'s last-place fold. That is a coverage gap this ADR exposes
  rather than creates, and **it ships as its own ticket.**
- **Nothing enforces the one-fold rule.** No check fires on a fourth normaliser, so review carries it.
  A count is not a gate: the three folds key on different inputs — a token, a rule name and a React
  prop — so no lint can recognise them as the same rule.
- **`cmd/web.sevLabel` stays as it is, and it stays downstream.** All eleven of its call sites take
  their token from `signal.SeverityFor`, from a folded `Artifact` token, or from a dev fixture. It must
  not gain a fold of its own, because the `""` it returns for an absent grade is correct and a fold
  would overwrite it with `Info`.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** Its `Signal` entry already states the five-level
  grade and cites ADR-0116. The fold is a render-layer normalisation over a token, not a domain term,
  and the glossary holds no term for it.
- **[ADR-0114](./0114-the-report-pdf-is-rendered-in-process-from-the-artifact-not-from-html.md) §1 gains
  its missing half on the record.** It states that the two render forms read one `Artifact`
  independently, and it names `artifactPDFItems` as what stops them drifting in *what they say*. This
  ADR names what stops them drifting in *how they grade*: both call `normSev`. No edit to ADR-0114 is
  required for that, and none is recorded.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Fold to `critical` — fail loud** | It is the collision the rule exists to refuse, and it is reachable by accident: `artifactSevLevels` and `SevOrder` both list `critical` first, so a normaliser written as *take the first member* lands there. It would also paint `artifactSeverityBadge`'s solid fill, the loudest element in the system, for a rule name nobody recognises, and it would win every minimum-rank rollup at `graph.go:504`, `subjects.go:1181` and `seeds.go:527` |
| **Fold to `medium` — the middle of the ramp** | It claims a scale position the token never carried. `medium` is a grade a rule was assigned, and asserting it for an unrecognised token puts a measured-looking value where nothing was measured. It also still sorts above `low` and `info`, so a stale token displaces real findings in the report's row order |
| **Refuse to render — drop the row, or error the page** | It reports less than we measured. The signal fired, the subject and the rule name are in hand, and dropping the row deletes a finding to avoid naming its grade. ADR-0010 declined the same trade when it refused to mark internally-observed defects `not-evaluable` in order to express a grade |
| **Render the raw token as-is** | `artifactSeverityBadge` and the `sevbadge` templates build CSS variable names from the token — `--sev-<l>-bg`, `--sev-<l>-fg`, `--sev-<l>-dot`. An unrecognised token produces custom properties that resolve to nothing, so the badge loses its background, its border and its dot. It also puts an arbitrary string into `data-sev`, which the graph's severity filter reads |
| **Let each surface fold as it sees fit** | Gives one signal two grades and gives the operator no way to tell which surface is stale. It is also the exact drift ADR-0114 §1 built `artifactPDFItems` to prevent, arriving through the grade instead of through the copy: two independently-authored layouts of one `Artifact` are safe only while the fold in front of them is one function |
| **Fold the value but leave `Rank()` alone** | `Rank()`'s fall-through is rank zero, which is `critical`'s rank. The value fold protects the badge and leaves `worstSeverity`, `assetHeaderSeverity`, the seed tree rollup and the report's row sort all handing a stale token the top of the list. §4 keeps the two limbs in one rule for that reason |
| **Merge this with [ADR-0183](./0183-the-severity-ramp-label-is-the-one-graded-word-the-product-draws-and-the-valence-refusal-does-not-reach-it.md)** | Two independent decisions. ADR-0183 could be reversed — the ramp drawn as colour and rank alone — and this fold would still be needed, because an out-of-set token still has to sort somewhere. This rule could be reversed and ADR-0183 would still stand. A merged file would make neither citable on its own |
| **A lint or a generated normaliser that guarantees the single fold** | The three folds key on different inputs, in two languages: a token string in `internal/message`, a rule name in `internal/signal`, and a React prop in the design system. No check can recognise them as one rule, and a check that fired on any `switch` over a severity token would fire on `pdfSevColor` and `artifactSeverityBadge`, which are correct |
