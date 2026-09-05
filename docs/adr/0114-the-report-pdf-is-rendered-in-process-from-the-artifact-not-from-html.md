# ADR-0114: the report PDF is rendered in-process from the artifact, not from HTML by an external engine

- **Status:** Accepted
- **Date:** 2026-08-23
- **Ticket:** [#345 Report artifact: Download PDF button disabled — PDF export not implemented](https://github.com/winniel123/verge-asm/issues/345)

## Context

The report-artifact page (`/reports/delivery`, [ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md))
shipped a **disabled** "Download PDF" control titled *"Export is not wired yet."* It was the only
export format the console advertised with no backend: CSV and JSON export from `/reports/export`
work. PDF did not. `internal/message.RenderArtifact` — the on-screen render of a delivered report —
was written all along as *"the one canonical rendered form that also serves as the PDF / email
spec,"* carrying its own self-contained token styles so it could stand alone as a document body. So
the intent was always that a PDF exists. Only the machinery did not.

Wiring PDF was left for a human ([#345] triage) because it turns on a decision the export writer's
own doc comment had already flagged and deliberately deferred: *"there is no HTML-to-PDF machinery
in the tree … either fabricating a delivery or standing up a renderer."* Two things had to be
decided — **whether to take on an HTML-to-PDF dependency, and what the PDF should contain.**

The runtime settles the first. The `web` binary ships in a **distroless
`static-debian12:nonroot`** image, built **`CGO_ENABLED=0`** — statically linked, no shell, no
package manager, no system libraries ([ADR-0001](./0001-stack-and-runtime.md): the static binary on
distroless *"has no interpreter, no package manager and no transitive C extensions,"* weighed as a
stated-priority attack-surface argument, since the instance's database is a complete map of the
operator's attack surface). Every real HTML-to-PDF route contradicts that image:

- **Headless Chrome (`chromedp`)** needs a Chromium binary plus hundreds of MB of shared libraries
  in the runtime image.
- **`wkhtmltopdf`** needs an external binary and its own shared-library closure.

Either means abandoning distroless-static for a large base image with a browser or binary and its
libc — trading away the exact attack-surface posture ADR-0001 chose, to render a report.

The second question — what the PDF contains — is already answered by the codebase: the delivered
**report document** `RenderArtifact` produces (identity, KPI band, the drift-vocabulary change
sections, the delivery receipt), **not** the operational activity series that `/reports/export`'s
CSV/JSON carry. Those are a different thing (the scans-per-day series and KPIs), and their writer
says so.

## Decision

> **The report PDF is rendered in-process, in pure Go, from the `message.Artifact` data — never by
> feeding `RenderArtifact`'s HTML to an external engine.** `internal/message.RenderArtifactPDF`
> (built on `github.com/go-pdf/fpdf`, a pure-Go, no-CGO, no-external-binary library) draws the same
> delivered-report content the HTML render draws, laid out for print. `GET /reports/delivery/pdf`
> serves its bytes as an attachment; the "Download PDF" control links there.

Three clarifications, so the boundary is not re-drawn each time:

1. **`Artifact` is the single source of report *content*. The PDF is a second *layout* of it.** The
   HTML render (`RenderArtifact`) and the PDF render (`RenderArtifactPDF`) each read the same
   `Artifact` struct independently. This is a deliberate, named departure from the "one canonical
   rendered form" phrasing in `RenderArtifact`'s comment: there is one canonical *content* model
   (the struct), and two render forms author its layout separately, because no pure-Go engine can
   turn the HTML/CSS into a PDF under `CGO_ENABLED=0`. To keep the two forms from drifting in *what
   they say*, `RenderArtifactPDF` builds both the drawn document and the text the valence guard
   reads from one ordered content sequence (`artifactPDFItems`).

2. **The print form keeps the domain guarantees.** No valence word grades the copy. ~~and no severity
   ramp appears — tone selects a colour only, never text (the drift palette and the delta tone), the
   same rule `RenderArtifact` obeys and its test asserts.~~ An `Artifact` with no delivered content
   renders the design-system empty-state, never a fabricated document ([ADR-0110]).

   > **The struck clause is WITHDRAWN at the site that specifies it, 2026-09-05 by
   > [ADR-0183](./0183-the-severity-ramp-label-is-the-one-graded-word-the-product-draws-and-the-valence-refusal-does-not-reach-it.md)
   > / [#1300](https://github.com/winniel123/verge-asm/issues/1300)
   > ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
   > **The replacement:** the print render draws the severity ramp, and it draws each level as its own
   > name — `CRITICAL`, `HIGH`, `MEDIUM`, `LOW`, `INFO` — as text, in that level's colour
   > (`RenderArtifactPDF`'s `roleSeverityBar` and `roleSignal`, `internal/message/pdf.go`). That label is
   > the one graded word the product draws, and ADR-0183 exempts it from the no-valence vocabulary of
   > [ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md). For every other
   > element — the delta tone and the drift palette — tone still selects a colour only and never text,
   > and no valence word grades the copy. The rest of this limb stands: the screen form obeys the same
   > rule, and an empty `Artifact` renders the design-system empty state.
   > **The clause was false when written.** The PDF render has drawn the level as text since
   > [#345](https://github.com/winniel123/verge-asm/issues/345) wired it. The sentence *"its test
   > asserts"* is also unsupported: `internal/message/render_test.go` never calls `ContainsValence`, so
   > no test applies the valence guard to the HTML render. ADR-0183's Consequences names both as
   > defects that ship as their own tickets.

3. **The PDF is the delivered report, not the operational export.** `/reports/delivery/pdf` renders
   the delivered-report document. `/reports/export?format=csv|json` renders the operational
   activity series. They are different surfaces backed by different reads and must not be conflated —
   a PDF of the scans-per-day series is out of scope here.

## Consequences

- **One new dependency, chosen to fit the image rather than break it:** `github.com/go-pdf/fpdf`
  is pure Go with no CGO and no external binary, so the distroless-static `web` image is unchanged —
  no Dockerfile edit, no browser, no new base image. It is the first third-party *rendering*
  dependency in the tree. ADR-0001's small-footprint constraint is honoured because the alternative
  that would have broken it (a headless browser) was rejected here, on the record.
- **Fidelity is deliberately not pixel-exact to the HTML.** `fpdf`'s standard fonts are the
  built-in PDF cores (Helvetica / Courier), not the design system's Instrument Sans / Geist Mono,
  and the layout is authored for a single print column rather than reflowed from the CSS. The PDF
  reads as the same document — same content, same palette, same drift vocabulary — but is not a
  screenshot of the page. Embedding the brand fonts is a later refinement, not a blocker.
- **The PDF follows the delivery backend for free.** There is no report-delivery store yet
  ([#285](https://github.com/winniel123/verge-asm/issues/285),
  [#290](https://github.com/winniel123/verge-asm/issues/290)/[#291](https://github.com/winniel123/verge-asm/issues/291)),
  so both `/reports/delivery` and its PDF render the empty-state `Artifact` today. When a delivery
  store lands, both handlers fill the same struct with real data and the download follows with no
  change to the render path.
- **If a future requirement needs pixel-exact HTML-to-PDF** (complex reflow, embedded charts, exact
  CSS fidelity), that reopens this ADR — it would mean revisiting the distroless-static image and
  ADR-0001's attack-surface posture, which is the cost this decision declined to pay.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Headless Chrome (`chromedp`) rendering `RenderArtifact`'s HTML | Needs a Chromium binary + libs in the image; abandons distroless-static and ADR-0001's attack-surface posture |
| `wkhtmltopdf` external binary | Same image cost; an unmaintained external binary and its libc closure inside a hardened image |
| Drop the disabled "Download PDF" button | The console loses a format `RenderArtifact` was explicitly built to be the spec for; declined once the dependency was shown to fit the image |
| PDF of the operational `/reports/export` series | Conflates the delivered-report document with the activity export — different surface, different read |

## Amendment — [#1456](https://github.com/winniel123/verge-asm/issues/1456): a delivered receipt names the delivery target's host, never the target, and both render forms obey it from the one `Artifact`

**The Decision is unchanged, and this ADR gains no new reach.** The amendment states a rule the
Decision's clarification 1 already carries. That clarification makes the `Artifact` the single
source of report *content* and each render form a second *layout* of it. So a rule about what the
receipt line says is a rule about the content model, and both forms obey it without a second
decision. The rule had no document before
[#1447](https://github.com/winniel123/verge-asm/issues/1447), which is how two comments came to
cite an ADR that never ruled it.

> **A delivered report's receipt names the host of the delivery target, never the target itself.**
> A `delivery_target` is free text an operator typed, and an operator may embed a token in a webhook
> URL. Where no host parses, the receipt names no host. The raw string is never the fallback.

### The rule as the shipped code behaves

One derivation, one producer, two renderings.

**The derivation.** `deliveryTargetHost` (`cmd/web/reports.go`) parses the schedule's
`delivery_target` with `url.Parse`. It answers `u.Host` where the parse succeeds and the host is
non-empty, and the empty string in every other case. It has no third answer, and the raw target
reaches no caller. A mistyped target therefore costs the operator the host line and leaks no token.

**The producer.** `buildReportDeliveryArtifact` (`cmd/web/reports.go`) is the only site in the tree
that sets `Artifact.ChannelHost`, and it sets the field inside the `DeliveredAt.Valid` branch. An
undelivered receipt therefore names no host either. Every other `Artifact` leaves the field zero:
the operational export (`writeReportsExportPDF`, `cmd/web/reports_export.go`) and the two discarded
confirmation renders (`cmd/web/reports_schedule.go`, `internal/report/dispatcher.go`).

**The print rendering.** `artifactReceipt` (`internal/message/render.go`) answers `not delivered` on
a zero `Delivered`. Otherwise it writes `delivered <instant>`, then appends the host only where
`ChannelHost` is non-empty. `RenderArtifactPDF` reads it through the ordered content sequence in
`internal/message/pdf.go`.

**The console rendering.** `BuildArtifactDoc` (`internal/message/artifactdoc.go`) copies
`ChannelHost` onto the document's `DeliveredTo`, and
[`reportartifact.tmpl`](../../design-system/templates/reportartifact.tmpl) prints it under the same
non-empty guard. Both forms read the one field, so neither can name the target while the other names
the host.

**A test holds the rule.** `cmd/web/reportdelivery_test.go` seeds a `delivery_target` of
`https://ops.example.test/hook/s3cr3t-token`. It then asserts that the delivery page names
`ops.example.test` and never names `s3cr3t-token`.

### Why the rule lives here and not in ADR-0180

[ADR-0180](./0180-a-message-detail-is-a-census-plus-its-delivery-receipts-and-carries-no-prose-body.md)
§3 states the same sentence about a **message** detail's delivery receipts. That is the closest
statement on disk, and it is the wrong citation for the report path.

**ADR-0180 §5 excludes this path by name.** It rules that nothing in `internal/message` renders a
`Message` at all. It names `RenderArtifact` and `RenderArtifactPDF` and lists their four report
callers. It then concludes that the PDF *"cannot disagree with this rule because it is not about
the same object"*. ADR-0180's own header lists this ADR under *"Not bound by"*. Its
[#1447](https://github.com/winniel123/verge-asm/issues/1447) amendment restates the fence and
closes with *"Do not cite this ADR from the report path."*

**So the fence is correct, and widening it is not the repair.** A `Message` and an `Artifact` are
different objects, with different tables and different producers. The shared Go package name is the
whole of the resemblance. An ADR-0180 amendment that reached an `Artifact` would falsify that ADR's
own title. The two rules agree because the hazard is one hazard, and neither is authority for the
other.

### What #1456 moved elsewhere, and what it leaves standing

Three sites recorded this rule before the amendment. #1456 repaired two of them in the same change,
and left the third in place.

- **The two comment sites needed no new reason clause, only a new citation.**
  `deliveryTargetHost` and `artifactReceipt` each carried the reason already, and each cited
  [`docs/guides/reports.md`](../guides/reports.md). **#1456 moved both to `ADR-0114 #1456`**, at the
  site that carries the citation
  ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)). Neither
  reason clause changed.
- **ADR-0180's #1447 amendment named the guide as the rule's home.** **#1456 struck that clause and
  named this amendment in its place**, in ADR-0180's own body, under the same ADR-0058 mark. §5 and
  ADR-0180's Decision stay untouched.
- **[`docs/guides/reports.md`](../guides/reports.md) keeps its operator-facing statement**, under
  *The receipt names the host, never the whole delivery URL*. The guide says what an operator reads
  on a receipt. This ADR rules why the product refuses the other rendering. The guide now links this
  amendment, and its paragraph about ADR-0180 restates the fence above and stays true.

**The past tense is exact.** #1456 landed this amendment and those two repairs in one change. No
reader ever saw a tree where this document ruled the report path and the two comments cited the
guide.
