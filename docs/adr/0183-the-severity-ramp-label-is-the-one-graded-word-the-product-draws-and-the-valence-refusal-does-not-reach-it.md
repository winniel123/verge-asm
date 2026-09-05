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

### The ramp label is drawn as text at 17 sites

Six in Go, eleven in the console templates.

| Surface | Site | The word it draws | Its colour |
| --- | --- | --- | --- |
| Screen, ramp bars | `internal/message/render.go:307` | `sevTitle(l)` | `--secondary`, beside a bar filled `--sev-<l>-dot` |
| Screen, badge, critical | `internal/message/render.go:348` | `sevTitle(l)` | `--sev-critical-text` on `--sev-critical-fill` |
| Screen, badge, high → info | `internal/message/render.go:351` | `sevTitle(l)` | `--sev-<l>-fg` |
| Email / doc form, ramp bars | `internal/message/artifactdoc.go:147` | the member name, lower case | the template's `--sev-<l>-dot` bar |
| Email / doc form, signal rows | `internal/message/artifactdoc.go:154` | `sevTitle(level)` | the `sevbadge` template's ramp tokens |
| Print, ramp bars | `internal/message/pdf.go:266` | `strings.ToUpper(sevTitle(it.level))` | `pdfSevColor(it.level)` |
| Print, signal rows | `internal/message/pdf.go:276` | `strings.ToUpper(sevTitle(it.signal.Severity))` | `pdfSevColor(...)` |
| Console, every screen with a signal | `design-system/templates/signals.tmpl:2` and `:4`, invoked **11** times across **7** templates | `{{.SevLabel}}` | the `--sev-<l>-*` tokens |
| Console, graph severity filter | `design-system/templates/graph.tmpl:93`–`97` | `Critical`, `High`, `Medium`, `Low`, `Info` | the control's own tokens |
| Design system, the component itself | `design-system/components/display/SeverityBadge.jsx:10` and `:15` | `Critical`, then the title-cased member | the `--sev-<l>-*` tokens |

**One measured correction to the record.** The deleted comments say the label is drawn in the severity
colour on every surface. That is true at six of the seven Go and template classes above. It is not true
of the screen ramp-bar label at `render.go:307`, which draws in `--secondary` beside a bar filled with
the severity colour. This ADR's rule turns on the **word**, not on the colour, so the correction does
not move the decision.

### The guard cannot see the tokens it would have to exempt

`RenderArtifactPDF` builds the drawn page and the guarded text from one ordered sequence
(`artifactPDFItems`), which is ADR-0114 §1's own anti-drift mechanism. The guarded projection drops the
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

[ADR-0114](./0114-the-report-pdf-is-rendered-in-process-from-the-artifact-not-from-html.md) §2 states:

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

### 4. The name of the scale is outside both rules

Two strings name the **scale** rather than a member: `artifactSeverityTitle`, `"Open signals by
severity"` (`render.go:212`), and the `Severity` column header (`render.go:325`).

Neither is an enum member name, so §3's exemption does not reach them. Neither says whether any
finding is good or bad, so ADR-0064's refusal does not reach them either. A column header naming an
axis makes no claim about any row under it. They are outside both rules.

`ValenceWords` does not encode that distinction, because it carries `severity` as a member. That is why
the code marks both elements `data-sev="title"` and `data-sev="header"` (`render.go:299`, `render.go:325`)
and why `pdf_test.go:83` skips one of them by string equality. The word list is what is wrong, not the
copy.

### 5. ADR-0114 §2's ramp sentence is withdrawn at its own site

Per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md), the
withdrawal is written at the sentence that specifies the mechanism, in the same change, with a
replacement supplied rather than a strike alone. ADR-0058's reader test applies directly: §2's clause,
read alone and in the present tense, tells a session that the print form draws no ramp, and a session
holding it would delete `pdf.go:266` and `pdf.go:276`.

The replacement wording states that the print render draws the ramp label as text in the severity
colour, and that tone still selects a colour only for every other element. The rest of §2 is unchanged:
no valence word grades the copy, and an empty `Artifact` renders the design-system empty state.

## Consequences

- **This ADR changes no Go code, no template and no test.** Every one of the 17 draw sites in §Context
  is already correct under this rule. The ADR states what they already do and closes the record.
- **[ADR-0114](./0114-the-report-pdf-is-rendered-in-process-from-the-artifact-not-from-html.md) §2 loses
  one clause and gains a replacement.** The edit is recorded in this issue's manifest and is applied
  by the batch parent, not by this ADR's author, at ADR-0114 §2's own site. ADR-0114's other three
  limbs are untouched.
- **The print form's valence guard is blind to the ramp, and that is a defect this ruling exposes.**
  `artifactPDFStrings` drops the ramp label at `pdf.go:113` and the per-signal token at `pdf.go:115`.
  The guard therefore cannot enforce §3's boundary: it cannot tell a ramp label it must permit from a
  prose *critical* it must refuse, because it sees neither. The projection should carry both tokens and
  the guard should exempt them by their `data-sev` role rather than by removing them. **It ships as its
  own ticket.**
- **No test applies the valence guard to the screen form, and that is a second defect.**
  `render_test.go` never calls `ContainsValence`. The screen render is the form ADR-0114 §2 calls *"the
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
| **Refuse the label — draw the ramp as colour and rank alone** | It reverses ADR-0116 and ADR-0110 by implication. It also deletes the datum at the point of use: the operator sorts and triages by the word, and a colour survives neither a screen reader nor a grayscale print nor a sentence spoken to a colleague. It would rewrite 17 draw sites, including `SeverityBadge.jsx`, which ADR-0110 names as the rendered form |
| **Refuse `critical` alone and keep the other four** | This is what the code enforces today by accident, because `ValenceWords` holds `critical` and not `high`, `medium`, `low` or `info`. It breaks the ramp at its top: a four-level grade whose worst level has no name is not the five-level grade ADR-0116 built, and the operator cannot see the one row that matters most |
| **Delete `critical` and `severity` from `ValenceWords`** | Cheap and wrong in the other direction. The word list is the only mechanical guard over authored prose, and dropping `critical` licenses a prose sentence calling a finding critical — exactly ADR-0064 §3's first refusal, which #35 grounds on four rules where a clear can be an attack having succeeded |
| **Exempt anything marked `data-sev`** | The mark is already on two elements that are not member names — `data-sev="title"` on the ramp heading and `data-sev="header"` on the column header (`render.go:299`, `render.go:325`). An attribute-scoped exemption would be widened by whoever adds the next `data-sev` mark, and the boundary this ADR draws would move without a decision. §3 fixes the exemption on the token, not on the mark |
| **Rule it in [ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md) as an amendment** | ADR-0064's subject is the `Message` vocabulary, and its §3 argues from four properties of message copy. A `Signal` grade is a different object, carried by a different `CONTEXT.md` entry and built by a different ADR. Filing the exemption inside the refusal would invite a reader to take it as a hole in the refusal rather than a boundary on its reach |
| **Rule it in [ADR-0114](./0114-the-report-pdf-is-rendered-in-process-from-the-artifact-not-from-html.md), where the contradiction sits** | ADR-0114 rules the print render only, and this rule binds the screen form, the email/doc form, eleven console template invocations and the design-system component. Filing it there would state a product-wide vocabulary rule inside a document about a PDF library choice, and it would leave the screen form's exemption unwritten |
| **Merge this with [ADR-0184](./0184-an-unknown-severity-token-folds-to-info-on-every-surface-and-no-surface-folds-it-differently.md)** | Two independent decisions. One says what an out-of-set token normalises to. The other says what an in-set token may be called. A reader could accept either and refuse the other, and a merged file would make one of them unciteable on its own |
| **Fix the guard, the projection and the tests on this ADR's branch** | The projection change reaches `artifactPDFStrings`, the guard's contract, `ValenceWords` and two test files, and it changes what CI asserts about shipped copy. That is a production change with its own review, and the batch this ADR lands in is documentation only |
