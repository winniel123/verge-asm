---
title: Reports
section: Operating
order: 6
description: Reports are scheduled digests of the whole estate — how a report differs from a notification, the recurring-schedule shape, and the delivered-report artifact and its PDF.
---

# Reports

A **report** is a scheduled digest of your *estate*: a periodic document that takes
stock of where your attack surface stands and delivers that snapshot on a cadence. It is
a different thing from a **notification**, and the difference is the whole reason this
screen exists.

The Reports screen is canonical `/reports` ([`cmd/web/reports.go`](../../cmd/web/reports.go)).
The distinction below is ruled by
[ADR-0039](../adr/0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md)
and enumerated in
[`docs/spec/notification-channels.md`](../spec/notification-channels.md).

> **Status, read this first.** Report *scheduling* has no dispatch or delivery backend
> in v1 ([#344](https://github.com/winniel123/verge-asm/issues/344),
> [#285](https://github.com/winniel123/verge-asm/issues/285)). The schedule model and the
> delivery-artifact surfaces are wired and honest, but nothing yet runs a schedule or
> produces a delivered document. Where a control is disabled or a route returns the
> empty-state, this guide says so rather than describing behaviour that does not run.

---

## A report is not a notification

They travel in opposite directions and carry opposite payloads.

- A **notification** is a single **message** — one fact that moved (an internet leg went
  `not-reached` → `reached`, a certificate expired). ADR-0039 rules that a `Channel`
  **carries the message and never the estate**: the outbound POST carries that one
  sentence to your escalation target, and the complete map stays on the instance. See
  [notification-channels.md](notification-channels.md) for how messages, channels and
  deliveries fit together.
- A **report** is the opposite: it carries **the estate**, not a single event — a rolled-up
  digest of the current census, delivered on a schedule rather than fired by a cause. It
  is the periodic "where do we stand" document, not the "this just changed" alert.

Because a report carries the whole picture off the instance, it inherits the same
high-value-target caution the notification layer does: the database is a complete, current
map of your attack surface (ADR-0039 §Context), and a delivered report is a copy of a
slice of it. Deliver reports only to targets you would trust with that map.

---

## The Reports screen

`/reports` (login-gated; a viewer reads it, mutates nothing) folds the estate's period
analytics into one view — a KPI band, a scans-per-day activity heatmap wired from real
`Dispatch` history, and, lower on the page, a **recurring-reports table** and a **"New
schedule" wizard**.

Two of the analytics regions are honest empty-states rather than fabricated series: a
signal census carries no severity and is never a trend, so the example mock's "by
severity" and "open signals over time" charts have no real data behind them and are
empty-stated on purpose (see the handler comment in
[`cmd/web/reports.go`](../../cmd/web/reports.go); ADR-0024). The one legitimate series
is operational — scans-per-day is activity volume, and that heatmap is real.

You can export the operational figures the page shows (the KPI band and the
scans-per-day series for the selected `?weeks=` range) as a file:

```
GET /reports/export?format=csv&weeks=12
GET /reports/export?format=json&weeks=26
```

An export is a read of the same figures the screen renders, so a viewer may take it. Note
this is the **operational activity export**, not a delivered report — the delivered-report
artifact is a separate surface, covered below.

---

## Recurring report schedules

A recurring report is one row of the `report_schedule` model
([`internal/db/report_schedule.sql.go`](../../internal/db/report_schedule.sql.go)). Its
shape is:

| Field | Meaning |
| --- | --- |
| `name` | What the schedule is called in the recurring-reports table. |
| `sections` | The chosen report sections, stored as a JSON array. Defaults to an empty array at the column, so a schedule with no sections still inserts. |
| `cadence` | How often the digest is produced (e.g. weekly). |
| `format` | The delivered document's form (e.g. `pdf`). |
| `delivery_target` | Where the produced digest is sent. |
| `created_by` | The admin who declared the schedule — the estate is single-tenant, so the list is unscoped and this is the only attribution the row carries. |

A `report_schedule` is **Declared**: it carries no timeline, no per-edit history. The
model exposes exactly a plain insert (`InsertReportSchedule`) and an unbounded,
newest-first list (`ListReportSchedules`) — there is **no content update and no delete**
query.

### Creating, editing and running a schedule — not yet wired

The ticket for this guide anticipated create/edit/delete and an on-demand
`/reports/schedule/run` route. **Those routes do not exist**, and the guide follows the
code:

- The only registered schedule route is `POST /reports/schedule`
  ([`cmd/web/handlers.go`](../../cmd/web/handlers.go)), and its handler **refuses with
  `501 Not Implemented`** ([`cmd/web/reports_schedule.go`](../../cmd/web/reports_schedule.go)).
  A schedule row that nothing reads on a cadence and nothing delivers would be a
  declaration the product cannot honour, so — rather than persist a row that silently
  never runs — the handler declines. It stays registered behind `requireAdmin` so a
  hand-crafted POST meets a clear, deterministic refusal instead of a bare `405`, and **no
  `report_schedule` row is ever filed** from normal use or a crafted request.
- In the UI, the "New schedule" wizard is rendered **disabled**, alongside its already-disabled
  sibling controls — Run now, Edit, Delete, View last delivery ([#344](https://github.com/winniel123/verge-asm/issues/344)).
- There is therefore **no `/reports/schedule/run`, `/edit` or `/delete` route** to call. The
  on-cadence dispatcher and delivery backend are the real "wire report scheduling" feature,
  explicitly out of scope until [#290/#291](https://github.com/winniel123/verge-asm/issues/290)
  populate report content and a scheduling dispatcher lands. When they do, the create
  handler regains its wizard and its `InsertReportSchedule` call; the model above is the
  shape it will file.

---

## The delivered-report artifact

A produced report is durable — a **delivered report is a record, not a mutation** (ADR-0039
rules a `Delivery` is an *operational* record). The console reads that record at a stable
route:

```
GET /reports/delivery        # the delivered artifact, in the console
GET /reports/delivery/pdf    # the same artifact as a downloadable PDF
```

Both are login-gated; a viewer reads either, because reading a delivered record is not a
mutation. The route is deliberately fixed so the recurring-reports table's "view last
delivery" link stays valid.

The on-screen document and the PDF are **two render forms of one `Artifact`**. The HTML is
drawn by `internal/message.RenderArtifact`; the PDF by
`internal/message.RenderArtifactPDF` ([`internal/message/pdf.go`](../../internal/message/pdf.go)) —
a **pure-Go render** (go-pdf/fpdf, no CGO, no external binary) chosen so it runs inside the
distroless-static `web` image with no separate rendering engine. Both read the same content
model in the same order, so the download can never disagree with what the page shows
(ADR-0114). "Download PDF" on the artifact page serves the `/reports/delivery/pdf` bytes as
an `application/pdf` attachment named `report-delivery.pdf`.

### What you see today

There is **no delivery backing store yet** ([#285](https://github.com/winniel123/verge-asm/issues/285);
[#290/#291](https://github.com/winniel123/verge-asm/issues/290) populate report content), so no
delivered artifact exists to read. Rather than fabricate a document, both handlers render a
**zero `Artifact`** — the design-system empty-state inside the delivered-document frame, and a
valid but empty-state PDF. When a delivery store lands, both handlers fill the same struct with
real data of the same shape and the render path does not change: the download follows for free,
and the PDF gains a period-dated filename once the artifact has a delivery window to name.

---

## How report delivery relates to notification channels

Reports and channels sit either side of the same ADR-0039 line, and it is worth holding
them apart:

- A **notification `Channel`** carries a **single message** outbound and never the estate.
  It fires on a cause, produces one `Delivery` per attempt, and a failed delivery loses the
  escalation but never the fact — the fact is already unconditionally in the store. See
  [notification-channels.md](notification-channels.md).
- A **report** carries the **estate digest** on a **schedule**, and its delivered artifact
  is the durable record described above.

Both are *outbound* and both produce operational delivery records; neither turns the
instance into a pull feed a reader could poll, which ADR-0039 refuses outright. The
practical operator takeaway: route a **channel** when you need to know *the moment a leg
moves*; use a **report** when you need a *periodic standing snapshot* of the whole surface.

For getting the stack up and configuring the `worker` that will run scheduled work, see
[running.md](running.md); for the first-run walkthrough, [using.md](using.md) and
[first-run.md](first-run.md).
