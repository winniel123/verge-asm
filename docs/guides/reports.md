---
title: Reports
section: Signals & delivery
order: 3
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

> **Status, read this first.** In-instance report *scheduling* is **live**
> ([#499](https://github.com/winniel123/verge-asm/issues/499)): an admin can create,
> list, edit, delete and run-now a schedule, the `worker`'s on-cadence dispatcher runs
> due schedules and stamps a delivery receipt, and `/reports/delivery` renders the
> delivered artifact. What is **not** built is **off-instance delivery** — sending the
> produced digest to a recipient. There is no destination an operator can set: the
> "New schedule" wizard captures no recipient/channel field and `delivery_target` has no
> channel binding, so the notify path is unreachable. That binding is escalated as
> [`SPEC-CHANGE.md`](../../design-system/SPEC-CHANGE.md) **collision #17 (AWAITING
> DESIGN)** and tracked as T7 ([#508](https://github.com/winniel123/verge-asm/issues/508)).
> Until it is ruled, `delivery_target` is declared-but-unused and schedules are
> effectively **download-only / in-instance** — a run is *generated*, never *delivered*.

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
| `delivery_target` | Where the produced digest would be sent off-instance. **Not yet wired** — the wizard sets no recipient, so this is declared-but-unused free text; off-instance send is escalated as SPEC-CHANGE collision #17 ([#508](https://github.com/winniel123/verge-asm/issues/508)). |
| `created_by` | The admin who declared the schedule — the estate is single-tenant, so the list is unscoped and this is the only attribution the row carries. |

A `report_schedule` is **Declared**: it carries no timeline, no per-edit history. Edit is
a genuine in-place update of what was declared (`UpdateReportSchedule`), never a recompute,
and Delete is a hard delete (`DeleteReportSchedule`); the list is unbounded and
newest-first (`ListReportSchedules`).

### Creating, editing and running a schedule

Scheduling is live and every route below is admin-gated (`requireAdmin`) in
[`cmd/web/handlers.go`](../../cmd/web/handlers.go); a viewer is refused before the handler
runs. The handlers live in
[`cmd/web/reports_schedule.go`](../../cmd/web/reports_schedule.go):

- **Create** — `GET /reports/schedule/new` opens a three-step wizard (Scope / Cadence /
  Review), ported from `Reports.jsx`. With no client runtime the controlled state rides a
  post-back form: `POST /reports/schedule` re-renders each step and, on finish, files the
  schedule with `InsertReportSchedule`, then redirects to `/reports`. The wizard captures
  **no recipient field** — by design, so it does not over-promise a delivery the product
  cannot yet make (see the status note above).
- **Edit** — `GET /reports/schedule/{id}/edit` opens the same wizard prefilled from the
  row; `POST /reports/schedule/edit` updates it in place with `UpdateReportSchedule`.
- **Run now** — `POST /reports/schedule/run` cuts the artifact for the current period with
  the canonical renderer and stamps a `report_delivery` receipt (state `generated`, no
  `delivered_at` — nothing leaves the instance).
- **Delete** — `POST /reports/schedule/delete` hard-deletes the row (idempotent: a stale
  id is a no-op, not an error).

The **on-cadence dispatcher** ([`internal/report/dispatcher.go`](../../internal/report/dispatcher.go),
wired into the `worker` in [`cmd/worker/main.go`](../../cmd/worker/main.go); ADR-0118) polls
each minute, floors "now" to a per-cadence tick (`CadenceWindow`), and — under a
per-schedule advisory lock — stamps exactly one receipt per `(schedule, tick)`. Idempotency
is durable: `TryInsertScheduledDelivery` inserts `ON CONFLICT (schedule_id, scheduled_tick)
DO NOTHING` against the partial-unique index (migration
[`22600`](../../db/migrations/22600_report_delivery_scheduled_tick.sql)), so a second poll
inside a window is a recorded skip, never a second run. Every run is *generated* in-instance,
never *delivered*: the off-instance send remains escalated (collision #17 / T7).

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

The delivery backing store is live: `report_delivery` receipts are stamped by Run-now and by
the on-cadence dispatcher, and `/reports/delivery` opens the **single most-recent non-failed
delivery** across every schedule (`reportDeliveryArtifact` in
[`cmd/web/reports.go`](../../cmd/web/reports.go)). A receipt snapshots no content — the artifact
**recomputes** its figures from the receipt's period bounds at render time, so nothing is carried
off-instance (ADR-0039). Where no schedule has ever run, both handlers render a **zero
`Artifact`** — the design-system empty-state inside the delivered-document frame, and a valid but
empty-state PDF — rather than fabricate a document. Once a delivery names a window, the PDF gains
a period-dated filename (`report-<start>-to-<end>.pdf`).

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
