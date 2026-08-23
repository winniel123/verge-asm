# ADR-0114: the report PDF is rendered in-process from the artifact, not from HTML by an external engine

- **Status:** Accepted
- **Date:** 2026-08-23
- **Ticket:** [#345 Report artifact: Download PDF button disabled — PDF export not implemented](https://github.com/winniel123/verge-asm/issues/345)

## Context

The report-artifact page (`/reports/delivery`, [ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md))
shipped a **disabled** "Download PDF" control titled *"Export is not wired yet."* It was the only
export format the console advertised with no backend: CSV and JSON export from `/reports/export`
work; PDF did not. `internal/message.RenderArtifact` — the on-screen render of a delivered report —
was written all along as *"the one canonical rendered form that also serves as the PDF / email
spec,"* carrying its own self-contained token styles so it could stand alone as a document body. So
the intent was always that a PDF exists; only the machinery did not.

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

1. **`Artifact` is the single source of report *content*; the PDF is a second *layout* of it.** The
   HTML render (`RenderArtifact`) and the PDF render (`RenderArtifactPDF`) each read the same
   `Artifact` struct independently. This is a deliberate, named departure from the "one canonical
   rendered form" phrasing in `RenderArtifact`'s comment: there is one canonical *content* model
   (the struct), and two render forms author its layout separately, because no pure-Go engine can
   turn the HTML/CSS into a PDF under `CGO_ENABLED=0`. To keep the two forms from drifting in *what
   they say*, `RenderArtifactPDF` builds both the drawn document and the text the valence guard
   reads from one ordered content sequence (`artifactPDFItems`).

2. **The print form keeps the domain guarantees.** No valence word grades the copy, and no severity
   ramp appears — tone selects a colour only, never text (the drift palette and the delta tone), the
   same rule `RenderArtifact` obeys and its test asserts. An `Artifact` with no delivered content
   renders the design-system empty-state, never a fabricated document ([ADR-0110]).

3. **The PDF is the delivered report, not the operational export.** `/reports/delivery/pdf` renders
   the delivered-report document; `/reports/export?format=csv|json` renders the operational
   activity series. They are different surfaces backed by different reads and must not be conflated —
   a PDF of the scans-per-day series is out of scope here.

## Consequences

- **One new dependency, chosen to fit the image rather than break it:** `github.com/go-pdf/fpdf`
  is pure Go with no CGO and no external binary, so the distroless-static `web` image is unchanged —
  no Dockerfile edit, no browser, no new base image. It is the first third-party *rendering*
  dependency in the tree; ADR-0001's small-footprint constraint is honoured because the alternative
  that would have broken it (a headless browser) was rejected here, on the record.
- **Fidelity is deliberately not pixel-exact to the HTML.** `fpdf`'s standard fonts are the
  built-in PDF cores (Helvetica / Courier), not the design system's Instrument Sans / Geist Mono,
  and the layout is authored for a single print column rather than reflowed from the CSS. The PDF
  reads as the same document — same content, same palette, same drift vocabulary — but is not a
  screenshot of the page. Embedding the brand fonts is a later refinement, not a blocker.
- **The PDF follows the delivery backend for free.** There is no report-delivery store yet
  ([#285](https://github.com/winniel123/verge-asm/issues/285);
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
