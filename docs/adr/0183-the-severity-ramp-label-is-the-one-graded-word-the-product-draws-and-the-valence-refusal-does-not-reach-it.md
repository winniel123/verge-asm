# ADR-0183: the severity ramp label is the one graded word the product draws, and the valence refusal does not reach it

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1300 ADR gaps: internal/message](https://github.com/winniel123/verge-asm/issues/1300), gap 5
- **PR that deleted the comment:** [#1299](https://github.com/winniel123/verge-asm/pull/1299)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Bounded by:** [ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md). It rules that a
  message names what moved, and that the vocabulary carries no valence word and no severity. This ADR
  does not reopen that refusal. It fixes the refusal's reach at authored prose and exempts one closed
  enum's member name at the ramp element
- **Rests on:** [ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md). It rules
  that the design-system examples are the console's IA spec, ported verbatim, and states that severity
  is exactly `Critical / High / Medium / Low / Info` via `SeverityBadge`. The product must draw the grade
- **Rests on:** [ADR-0116](./0116-the-design-package-is-normative-for-look-and-functionality.md). It rules
  that the design package is normative for look and functionality, and that where the domain lacks a
  datum the design renders, the fix is to build the datum. It built the five-level grade
- **Withdraws a clause of:** [ADR-0114](./0114-the-report-pdf-is-rendered-in-process-from-the-artifact-not-from-html.md). It rules
  that the report PDF is rendered in-process in pure Go from the `Artifact`, and its §2 states that no
  severity ramp appears in the print form and that tone selects a colour only, never text. That sentence
  is withdrawn here, at its own site, per
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
- **Sibling of, and not ruled by:** [ADR-0184](./0184-an-unknown-severity-token-folds-to-info-on-every-surface-and-no-surface-folds-it-differently.md). It rules
  how an unknown grade token normalises. This ADR rules what the normalised token may be called. One is
  a normalisation rule and one is a vocabulary rule. Neither contains the other

## Context

`internal/message/render.go:443` carried this, until #1299 deleted it:

```go
// Marked data-sev so the
// valence guard exempts the ramp label (the one loud voice) as it does the token
// stylesheet.
```

The same rule was stated three more times in the print render. `internal/message/pdf.go`'s file
header carried it:

```go
// The severity ramp (P2.10) is the one
// loud voice: its label is drawn in the severity colour, like the on-screen
// SeverityBadge, and is exempt from the graded-prose view the valence guard reads,
// so the print form keeps the same domain guarantees.
```

`pdf.go:187` carried it at `pdfSevColor`:

```go
// The print document draws the severity word in this colour, the severity ramp's
// one loud voice; it never grades prose. An unknown token folds to info.
```

And `artifactPDFStrings` carried it twice in body, at `pdf.go:149` and `pdf.go:154`:

```go
// The count only. The level word is the severity ramp — the one loud
// voice — drawn as colour + label like the badge, and exempt from the
// valence prose view exactly as a delta's tone is (colour, never text).
```

```go
// The signal, its asset and the raised date. The severity is the ramp,
// drawn as colour + label; it is not part of the graded prose view.
```

**Every one of those citations resolves to a document that is not in the tree.** The only source any
of them named was `P2.10`, which is `design-system/PARITY-CHART.md`. That file was retired with the
design-system handoff workflow ([ADR-0145](./0145-design-system-is-the-shared-home-and-source-of-truth-for-ui-assets-and-a-session-edits-it-in-the-repo.md),
superseded [ADR-0109](./0109-design-system-components-are-authored-in-claude-design-and-imported.md) and
[ADR-0116](./0116-the-design-package-is-normative-for-look-and-functionality.md)). So the rule now has
no written site at all, and the ADR corpus states the opposite of it in one place.

### The two accepted rules that collide

[ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md) §3 is the refusal:

> **No message may contain a word that says whether the news is good or bad.** Not *resolved*,
> *fixed*, *cleared*, *improved*, *critical*, *warning*, *high*, *low* or *OK*. Not a colour standing
> for one, and not a severity field standing for all of them.

[ADR-0116](./0116-the-design-package-is-normative-for-look-and-functionality.md)'s Consequences is the
requirement, and [`CONTEXT.md`](../../CONTEXT.md)'s `Signal` entry now carries it: a signal carries a
five-level grade — critical / high / medium / low / info — assigned per rule. ADR-0110 fixes the
rendered form as `SeverityBadge`, with the words `Critical / High / Medium / Low / Info`.

**The collision is measurable at the word list.** `internal/message/render.go:15` declares
`ValenceWords`, 31 members. Two of the 31 touch the ramp:

| Ramp member | In `ValenceWords` | Named in ADR-0064 §3's prose |
| --- | --- | --- |
| `critical` | **yes** | **yes** |
| `high` | no | **yes** |
| `medium` | no | no |
| `low` | no | **yes** |
| `info` | no | no |
| `severity` (the scale's name, not a member) | **yes** | as *a severity field* |

So the guard refuses one of five members and ADR-0064's prose refuses three of five. Neither refuses
all five, and no document says why the split falls there. A reader who applies the prose literally
must delete `Critical`, `High` and `Low` from the product and keep `Medium` and `Info`, which is not a
grade.

### The ramp label is drawn as text at ~~17~~ **31** source lines

~~Six in Go, eleven in the console templates.~~

> **WITHDRAWN in part, by [#1567](https://github.com/winniel123/verge-asm/issues/1567).** PR
> [#1548](https://github.com/winniel123/verge-asm/pull/1548) deleted three of the draw sites this
> section counts, and the three struck rows below name them. All three sat inside `artifactSeverityBars` and
> `artifactSeverityBadge`, which `SPEC-CHANGE #23g` orphaned when `RenderArtifact` moved to
> `renderArtifactDoc`. Every other row stands. This withdrawal states no new total: the split above
> counts six Go sites against the table's seven Go rows, so a replacement number needs a fresh
> measurement of the whole surface, which that ticket does not open.

> **RE-MEASURED, by [#1599](https://github.com/winniel123/verge-asm/issues/1599), on `b3bd49f`.**
> **The unit is one source line that draws a ramp member's name as user-visible text.** A line counts
> once, whatever number of members it names, and whatever number of times a template calls it. So the
> `sevbadge` define counts once, and its eleven call sites add nothing to the total. On that unit the
> tree holds **31** lines: **17 in Go**, **12 in `design-system/templates/`**, and **2 in
> `design-system/components/`**. The table below carries all 31. The three rows PR
> [#1548](https://github.com/winniel123/verge-asm/pull/1548) struck sit outside the total.

> **Three populations sit outside the 31, and this paragraph names them.** `cmd/web/devfixtures.go`
> draws the ramp word at 27 further lines, on dev-only routes no operator reaches, so they stay
> outside the count. The design-system gallery cards demonstrate a control rather than draw a grade —
> `design-system/components/forms/forms.card.html:34` and
> `design-system/components/display/display.card.html:35`. Three sites name the scale or a slice of it
> in copy, rather than draw a signal's grade — `design-system/templates/signals.tmpl:155`,
> `design-system/templates/reports.tmpl:176` and `cmd/web/reports.go:539`. Each population still
> owes §3 its vocabulary.

| Surface | Site | The word it draws | Its colour |
| --- | --- | --- | --- |
| ~~Screen, ramp bars~~ | ~~`internal/message/render.go:307`~~ | ~~`sevTitle(l)`~~ | ~~`--secondary`, beside a bar filled `--sev-<l>-dot`~~ — **WITHDRAWN**, PR [#1548](https://github.com/winniel123/verge-asm/pull/1548) deleted `artifactSeverityBars` |
| ~~Screen, badge, critical~~ | ~~`internal/message/render.go:348`~~ | ~~`sevTitle(l)`~~ | ~~`--sev-critical-text` on `--sev-critical-fill`~~ — **WITHDRAWN**, PR [#1548](https://github.com/winniel123/verge-asm/pull/1548) deleted `artifactSeverityBadge` |
| ~~Screen, badge, high → info~~ | ~~`internal/message/render.go:351`~~ | ~~`sevTitle(l)`~~ | ~~`--sev-<l>-fg`~~ — **WITHDRAWN**, PR [#1548](https://github.com/winniel123/verge-asm/pull/1548) deleted `artifactSeverityBadge` |
| Email / doc form, ramp bars | `internal/message/artifactdoc.go:147` | the member name, lower case | the template's `--sev-<l>-dot` bar |
| Email / doc form, signal rows | `internal/message/artifactdoc.go:154` | `sevTitle(level)` | the `sevbadge` template's ramp tokens |
| Print, ramp bars | `internal/message/pdf.go:266` | `strings.ToUpper(sevTitle(it.level))` | `pdfSevColor(it.level)` |
| Print, signal rows | `internal/message/pdf.go:276` | `strings.ToUpper(sevTitle(it.signal.Severity))` | `pdfSevColor(...)` |
| Console, dashboard signal rows | `cmd/web/auth.go:703` | `sevLabel(in.Severity)` | the `sevbadge` template's ramp tokens |
| Console, dashboard ramp bars | `cmd/web/auth.go:731` | the member name, lower case | `--text-secondary`, beside a `--sev-<l>-dot` bar |
| Console, dashboard critical stat | `cmd/web/auth.go:777` | `Critical` | the stat tile's own tokens |
| Console, signals severity filter | `cmd/web/signals.go:368` | `Critical`, `High`, `Medium`, `Low`, `Info` | the control's own tokens |
| Console, signals rows and detail | `cmd/web/signals.go:522` and `:585` | `sevLabel(...)` | the `sevbadge` template's ramp tokens |
| Console, search results | `cmd/web/search.go:188` and `:207` | `sevLabel(...)` | the `sevbadge` template's ramp tokens |
| Console, graph node signals | `cmd/web/graph.go:472` | `sevLabel(sev.String())` | the `sevbadge` template's ramp tokens |
| Console, subject and asset views | `cmd/web/subjects.go:549`, `:980` and `:1237` | `sevLabel(...)` | the `sevbadge` template's ramp tokens |
| Console, reports ramp bars | `cmd/web/reports.go:303` | `strings.ToUpper(string(sev))` | `--text-secondary`, beside a `--sev-<l>-dot` bar |
| Console, every screen with a signal | `design-system/templates/signals.tmpl:2` and `:4` — **10** `sevbadge` calls and **1** `sevbadge-md` call, across **7** templates | `{{.SevLabel}}` | the `--sev-<l>-*` tokens |
| Console, signals severity filter | `design-system/templates/signals.tmpl:180` and `:182` | `{{.Sev}}`, then each of `.SevOptions` | the control's own tokens |
| Console, dashboard ramp bars | `design-system/templates/dashboard.tmpl:141` | `{{.Sev}}` | `--text-secondary`, beside a `--sev-<l>-dot` bar |
| Console, reports ramp bars | `design-system/templates/reports.tmpl:196` | `{{.Label}}` | `--text-secondary`, beside a `--sev-<l>-dot` bar |
| Report artifact, ramp bars | `design-system/templates/reportartifact.tmpl:28` | `{{.Label}}` | `--text-secondary`, beside a `--sev-<l>-dot` bar |
| Console, graph severity filter | `design-system/templates/graph.tmpl:93`–`97` | `Critical`, `High`, `Medium`, `Low`, `Info` | the control's own tokens |
| Design system, the component itself | `design-system/components/display/SeverityBadge.jsx:10` and `:15` | `Critical`, then the title-cased member | the `--sev-<l>-*` tokens |

**The 31 lines sum from the live rows above.** Go carries 17 — four in `internal/message` and thirteen in
`cmd/web`. `design-system/templates/` carries 12. `SeverityBadge.jsx` carries 2. The three struck rows
sit outside all three figures.

**One measured correction to the record.** The deleted comments say the label is drawn in the severity
colour on every surface. That holds at 16 of the 31 sites. It fails at six ramp-bar label sites, which
draw in `--text-secondary` beside a bar the severity colour fills. It fails again at nine
severity-filter and stat-tile sites, which draw in the control's own colour. The struck
~~`render.go:307`~~ was the first of those six ramp-bar sites. This ADR's rule turns on the **word**,
not on the colour, so the correction does not move the decision.

> **The site moved and the correction stands.** PR
> [#1548](https://github.com/winniel123/verge-asm/pull/1548) deleted `render.go:307` with
> `artifactSeverityBars`. The delivered form draws the same bar label in `--text-secondary`, at
> `design-system/templates/reportartifact.tmpl:28`, which is the row for `artifactdoc.go:147` above.
> The exception outlives the deletion.

### The guard cannot see the tokens it would have to exempt

`RenderArtifactPDF` builds the drawn page and the guarded text from one ordered sequence
(`artifactPDFItems`), which is ADR-0114's own anti-drift mechanism. The guarded projection drops the
ramp label at both roles:

```go
case roleSeverityBar:
    out = append(out, strconv.Itoa(it.count))
```

`pdf.go:113` emits the count and never `it.level`. `pdf.go:115` emits `Signal`, `Asset` and `Raised`
and never `it.signal.Severity`. So `ContainsValence` has never read a ramp label, on any surface, in
either form.

**The screen form is not guarded at all.** `internal/message/render_test.go` never calls
`ContainsValence`. Four call sites exist in the package's tests — `message_test.go:79`, `:86`, `:89`,
`narrowing_test.go:69`, `pdf_test.go:88`, `pdf_test.go:121` — and every one of them reads message copy
or the print projection. The HTML render has no valence assertion.

`pdf_test.go:82` then carries this:

```go
// The ramp title names the scale, so the prose view drops it as the HTML test does.
if s == artifactSeverityTitle {
    continue
}
```

**There is no HTML test that does this.** The comment describes a test that does not exist, and the
skip it justifies is a string-equality skip on `artifactSeverityTitle` — `"Open signals by severity"`
(`render.go:212`) — which the guard would otherwise fail on the word `severity`.

### What ADR-0114 says instead

[ADR-0114](./0114-the-report-pdf-is-rendered-in-process-from-the-artifact-not-from-html.md) states:

> **The print form keeps the domain guarantees.** No valence word grades the copy, and no severity
> ramp appears — tone selects a colour only, never text (the drift palette and the delta tone), the
> same rule `RenderArtifact` obeys and its test asserts.

The print render draws a ramp, and it draws the level as text. `pdf.go:266` and `pdf.go:276` are the
two sites. The clause is false as written, and it carries no withdrawal marker.

## Decision

> **A signal's severity grade is drawn as its own name — `Critical`, `High`, `Medium`, `Low`, `Info` —
> on every surface that renders the ramp, screen and print. The severity ramp is the one graded voice
> the product allows. ADR-0064's valence refusal binds authored prose, the words this product writes
> about a finding. It does not reach the name of a closed, defined enum whose members the domain fixes.
> The exemption is the enum member's own name, at the ramp element, and nothing else.**

### 1. A grade that cannot say its own name is not a grade

ADR-0116 built the datum and ADR-0110 fixed its rendered form. Both require a five-level grade to reach
the operator. A grade reaches the operator only where the operator can read it, sort by it and speak it
to a colleague. A colour alone cannot do that: it does not survive a screen reader, a grayscale print,
or a sentence.

The alternative is not "a quieter grade". It is no grade, and a reversal of ADR-0116 and ADR-0110 taken
by implication rather than on the record. This ADR refuses to take it that way.

### 2. What ADR-0064 refuses is authored prose

ADR-0064 §3's four reasons are all reasons about **sentences the product writes**. A clear is not
always good news, so the sentence must not say *resolved*. A widening is neither good nor bad, so the
sentence must not put a valence on it. A severity in the notification layer is an unmeasured threshold
rendered instead of applied. A severity column on the message store performs the collapse #22 refused.

Every one of those is about the product editorialising over a finding it made. None of them is about
the product naming a member of a closed set the domain defines.

**The two objects are different, and the corpus already separates them.** ADR-0064's subject is the
`Message`. `CONTEXT.md`'s `Message` entry carries the refusal and says so. `CONTEXT.md`'s `Signal`
entry carries the grade and cites ADR-0116 for it, and it says in terms that a `Message` still carries
no severity. This ADR changes neither entry. It says what a rendered `Signal` grade is called.

### 3. The exemption is the member name, at the ramp element, and nothing else

The exemption is narrow and mechanical, so that it cannot be widened by argument at a later site.

**The exempt token is one of exactly five strings** — `critical`, `high`, `medium`, `low`, `info`, in
any case — and it is exempt only where it is the value of an `ArtifactSignal.Severity`,
`ArtifactSeverityCount.Level` or `signal.Severity`, drawn at the element that renders that value.

Everything else stays bound:

- A sentence that says a finding is *critical* is prose. It is refused.
- A sentence that says a finding is *high* is prose. It is refused, and none of `high`, `medium` or
  `low` is in `ValenceWords` today, so the guard will not catch it. Review carries that.
- A colour standing for a valence stays refused. ADR-0064 §3 refuses it in terms, and the delta tone
  and the drift palette both obey it — `pdfDeltaColor` (`pdf.go:161`) selects a colour and never a word.
- The change vocabulary keeps its own drift palette and never the severity ramp
  ([ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md)). This
  exemption does not reach it.

> **AMENDED, by [#1607](https://github.com/winniel123/verge-asm/issues/1607).** "Everything else"
> above means every other **rendered grade**. Two populations render no grade, and both stand
> outside §3 and outside ADR-0064. §4 names the scale's own name. §6 names a member name drawn as a
> set name.

### 4. The name of the scale is outside both rules

Two strings name the **scale** rather than a member: `artifactSeverityTitle`, `"Open signals by
severity"` (`render.go:212`), and the `Severity` column header (`render.go:325`).

Neither is an enum member name, so §3's exemption does not reach them. Neither says whether any
finding is good or bad, so ADR-0064's refusal does not reach them either. A column header naming an
axis makes no claim about any row under it. They are outside both rules.

`ValenceWords` does not encode that distinction, because it carries `severity` as a member. That is why
~~the code marks both elements `data-sev="title"` and `data-sev="header"` (`render.go:299`, `render.go:325`)
and why~~ `pdf_test.go:83` skips ~~one of them~~ **the ramp title** by string equality. The word list is
what is wrong, not the copy.

> **WITHDRAWN, by [#1567](https://github.com/winniel123/verge-asm/issues/1567).** PR
> [#1548](https://github.com/winniel123/verge-asm/pull/1548) deleted both marked elements with
> `artifactSeverityBars` and `artifactSignalsTable`, and no `data-sev="title"` and no
> `data-sev="header"` survives anywhere in the tree. The delivered form draws the same two strings
> unmarked, at `design-system/templates/reportartifact.tmpl:25` and `:39`. `artifactSeverityTitle`
> (`internal/message/render.go:158`) and the `pdf_test.go:83` skip both stand, so §4's rule holds on
> its own terms.

### 5. ADR-0114's ramp sentence is withdrawn at its own site

Per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md), the
withdrawal is written at the sentence that specifies the mechanism, in the same change, with a
replacement supplied rather than a strike alone. ADR-0058's reader test applies directly: §2's clause,
read alone and in the present tense, tells a session that the print form draws no ramp, and a session
holding it would delete `pdf.go:266` and `pdf.go:276`.

The replacement wording states that the print render draws the ramp label as text in the severity
colour, and that tone still selects a colour only for every other element. The rest of §2 is unchanged:
no valence word grades the copy, and an empty `Artifact` renders the design-system empty state.

### 6. A member name drawn as a set name is outside both rules, and the nine sites stand

> **RULED, by [#1607](https://github.com/winniel123/verge-asm/issues/1607), on `38eec12`.** §Context
> counts nine draw sites that §3 and §4 both miss. This section rules them, so a later session does
> not reopen the question.

Nine of the 31 draw sites name a ramp member and render no subject's grade. §3 misses them, because
no one of them draws the value of an `ArtifactSignal.Severity`, an `ArtifactSeverityCount.Level` or a
`signal.Severity`. §4 misses them too, because each draws a member's name and not the scale's name.

**The test is what the drawn token refers to.** A token that grades one named subject in view is a
`Severity` value, and §3 governs it. A token that names the grade itself grades no subject at all.
It is outside §3, and it is outside ADR-0064.

**A set name is the second form.** An operator reads a filter option as the set the control selects.
An operator reads a stat-tile heading as the population its number counts. Neither reading puts a
grade on any signal in view.

**ADR-0064's refusal does not reach a set name.** ADR-0064 §3 refuses a word that says whether the
news is good or bad. A filter option says nothing about any signal in the list below it. A stat-tile
heading says nothing about the count beside it beyond which signals the count includes. §4's reason
carries here unchanged. A label naming an axis makes no claim about any row under it. A label naming
one point on that axis makes no claim either.

**The nine sites, re-measured on `38eec12`.**

| Site | The word it draws | Its role |
| --- | --- | --- |
| `cmd/web/signals.go:368` | the five member names in `SevOptions` | the signals severity filter's option set |
| `design-system/templates/signals.tmpl:180` | `{{.Sev}}`, the current selection | the signals severity filter's trigger |
| `design-system/templates/signals.tmpl:182` | each of `.SevOptions` | the signals severity filter's option list |
| `design-system/templates/graph.tmpl:93`–`97` | `Critical`, `High`, `Medium`, `Low`, `Info` | the graph severity filter's five options |
| `cmd/web/auth.go:777` | `Critical` | the dashboard stat tile's heading |

**The code already reads all nine as the closed set's own domain.**

- `filterSignalRows` lower-cases the drawn option, then compares it to a row's `Severity`
  (`cmd/web/signals.go:604`).
- The graph filter reads the option button's `data-sev` attribute, which carries the same lower-case
  member name (`design-system/templates/graph.tmpl:327`).
- The dashboard stat counts a signal when its severity equals `signal.SevCritical`
  (`cmd/web/auth.go:690`).

No site authors a word about a finding. Each site names a member of the set §3 names.

**No site changes.** The nine ship as they are. A later session must not replace a drawn member name
with a colour, a rank or a paraphrase. A later session must not add `high`, `medium`, `low` or `info`
to `ValenceWords` on account of these sites.

**The colour split is evidence about styling and not about vocabulary.** §Context records that these
nine draw in the control's own colour, and that 16 sites draw in the severity colour. This ADR's rule
turns on the word and never on the colour, so the split does not move this ruling.

**One caption on `cmd/web/auth.go:777` names the scale rather than a member.** The caption reads
*"highest severity"*. It names the ramp's top rank and grades no finding, so §4 covers it. It draws
no member name, so it changes no figure in §Context.

## Consequences

- **This ADR changes no Go code, no template and no test.** §Context measures ~~17~~ **31** draw
  sites. The 22 that render a signal's own grade are already correct under §3. The ADR states what
  they already do and closes the record. **PR
  [#1548](https://github.com/winniel123/verge-asm/pull/1548) deleted three of those sites.** The other
  nine draw a member name in a severity filter or a stat heading. §3 exempts a rendered `Severity`
  value, so it reaches none of the nine. ~~**That gap ships as its own ticket.**~~ **RULED in §6, by
  [#1607](https://github.com/winniel123/verge-asm/issues/1607).** §6 places the nine outside §3,
  outside §4 and outside ADR-0064's refusal. All nine stand unchanged, so this bullet's first
  sentence still holds for the whole surface.
- **[ADR-0114](./0114-the-report-pdf-is-rendered-in-process-from-the-artifact-not-from-html.md) loses
  one clause and gains a replacement.** The edit is recorded in this issue's manifest and is applied
  by the batch parent, not by this ADR's author, at ADR-0114's own site. ADR-0114's other three
  limbs are untouched.
- **The print form's valence guard is blind to the ramp, and that is a defect this ruling exposes.**
  `artifactPDFStrings` drops the ramp label at `pdf.go:113` and the per-signal token at `pdf.go:115`.
  The guard therefore cannot enforce §3's boundary: it cannot tell a ramp label it must permit from a
  prose *critical* it must refuse, because it sees neither. The projection should carry both tokens and
  the guard should exempt them by their `data-sev` role rather than by removing them. **It ships as its
  own ticket.**
- **No test applies the valence guard to the screen form, and that is a second defect.**
  `render_test.go` never calls `ContainsValence`. The screen render is the form ADR-0114 calls *"the
  same rule `RenderArtifact` obeys and its test asserts"*, and no such assertion exists. **It ships as
  its own ticket.**
- **`ValenceWords` carries `severity`, which fires on the scale's own name.** §4 places that name
  outside both rules, so the word list is wrong rather than the copy, and `pdf_test.go:83`'s
  string-equality skip is a workaround for it. **It ships as its own ticket, with the two above.**
- **`pdf_test.go:82`'s comment describes a test that does not exist.** It claims the HTML test drops the
  ramp title. **It ships as its own issue**, filed separately from this batch.
- **Review carries `high`, `medium` and `low` in prose.** The guard's word list holds `critical` and
  not the other four members, so a sentence calling a finding *high* passes the guard today. This ADR
  does not add them to the list, because adding them would fire on every ramp label the moment the
  guard can see one. The correct change is the role-aware guard named above, and it is that ticket's.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing and loses nothing.** Its `Signal` entry already
  carries the five-level grade and cites ADR-0116. Its `Message` entry already carries the valence
  refusal and cites ADR-0064, and it already states that a `Message` carries no severity. The exemption
  is a property of a rendered `Signal` grade, which is neither entry's subject.
- **A future surface that renders the ramp has a document to be held to.** Before this, the rule lived
  in five comment blocks in two files, all citing a retired parity chart, and the ADR corpus contradicted
  them in ADR-0114.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Refuse the label — draw the ramp as colour and rank alone** | It reverses ADR-0116 and ADR-0110 by implication. It also deletes the datum at the point of use: the operator sorts and triages by the word, and a colour survives neither a screen reader nor a grayscale print nor a sentence spoken to a colleague. It would rewrite ~~17~~ **31** draw sites, including `SeverityBadge.jsx`, which ADR-0110 names as the rendered form — **three of them are gone, PR [#1548](https://github.com/winniel123/verge-asm/pull/1548) deleted them** |
| **Refuse `critical` alone and keep the other four** | This is what the code enforces today by accident, because `ValenceWords` holds `critical` and not `high`, `medium`, `low` or `info`. It breaks the ramp at its top: a four-level grade whose worst level has no name is not the five-level grade ADR-0116 built, and the operator cannot see the one row that matters most |
| **Delete `critical` and `severity` from `ValenceWords`** | Cheap and wrong in the other direction. The word list is the only mechanical guard over authored prose, and dropping `critical` licenses a prose sentence calling a finding critical — exactly ADR-0064 §3's first refusal, which #35 grounds on four rules where a clear can be an attack having succeeded |
| **Exempt anything marked `data-sev`** | ~~The mark is already on two elements that are not member names — `data-sev="title"` on the ramp heading and `data-sev="header"` on the column header (`render.go:299`, `render.go:325`).~~ **WITHDRAWN, PR [#1548](https://github.com/winniel123/verge-asm/pull/1548) deleted both marks — see §4.** An attribute-scoped exemption would be widened by whoever adds the next `data-sev` mark, and the boundary this ADR draws would move without a decision. §3 fixes the exemption on the token, not on the mark |
| **Rule it in [ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md) as an amendment** | ADR-0064's subject is the `Message` vocabulary, and its §3 argues from four properties of message copy. A `Signal` grade is a different object, carried by a different `CONTEXT.md` entry and built by a different ADR. Filing the exemption inside the refusal would invite a reader to take it as a hole in the refusal rather than a boundary on its reach |
| **Rule it in [ADR-0114](./0114-the-report-pdf-is-rendered-in-process-from-the-artifact-not-from-html.md), where the contradiction sits** | ADR-0114 rules the print render only, and this rule binds the screen form, the email/doc form, eleven console template invocations and the design-system component. Filing it there would state a product-wide vocabulary rule inside a document about a PDF library choice, and it would leave the screen form's exemption unwritten |
| **Merge this with [ADR-0184](./0184-an-unknown-severity-token-folds-to-info-on-every-surface-and-no-surface-folds-it-differently.md)** | Two independent decisions. One says what an out-of-set token normalises to. The other says what an in-set token may be called. A reader could accept either and refuse the other, and a merged file would make one of them unciteable on its own |
| **Fix the guard, the projection and the tests on this ADR's branch** | The projection change reaches `artifactPDFStrings`, the guard's contract, `ValenceWords` and two test files, and it changes what CI asserts about shipped copy. That is a production change with its own review, and the batch this ADR lands in is documentation only |
| **Widen §3 so the exemption covers any element that draws a member name** (#1607) | It dissolves the boundary §3 exists to hold. A prose sentence sits inside an element too, so the widened rule would exempt *"this finding is critical"* wherever a reader calls that element a label. §6 rules the nine sites on what the drawn token refers to, and it leaves §3's mechanical test on a rendered value untouched |
| **Refuse the nine — draw the two severity filters and the stat tile without a member name** (#1607) | Each filter would offer five unnamed choices, and the stat tile would head a number with nothing. The operator could not say which set a control selects. The grade ADR-0116 built would stop reaching the operator at the one control built to use it, so §1's argument applies unchanged |
| **Rule the nine in a new ADR of their own** (#1607) | The question is the reach of this ADR's own §3 and §4. A reader of ADR-0183 alone must be able to settle it. A separate file would leave this ADR's Consequences saying the gap is open, and #1607 exists because that sentence read as an invitation to reopen it |
