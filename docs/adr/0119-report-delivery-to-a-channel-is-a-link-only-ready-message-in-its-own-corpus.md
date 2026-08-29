# ADR-0119: report delivery to a Channel is a link-only ready-message, in its own corpus

- **Status:** Accepted
- **Date:** 2026-08-24
- **Ticket:** [#508 notify-with-link report delivery](https://github.com/winniel123/verge-asm/issues/508)
- **Map:** [#499 report dispatch + delivery](https://github.com/winniel123/verge-asm/issues/499)

## Context

A `report_schedule` (migration 21700) is a declared, recurring report. The on-cadence
dispatcher ([ADR-0118](./0118-report-scheduling-dispatches-on-a-computable-window-and-the-receipt-is-its-own-dispatch-record.md))
cuts an artifact per window and stamps a `report_delivery` receipt in state `generated`,
`delivered_at` NULL. Until now a scheduled report never left the instance: there was no
destination and no send. T7/#508 gives a schedule a **delivery destination** and the transport
that reaches it.

The design ruling (collision #17, package v3.2.3,
[design-system/examples/console/Reports.jsx](../../design-system/examples/console/Reports.jsx))
settles the destination: a schedule **delivers to a Channel** — the same signed-HTTPS Channel the
Message delivery path uses (migration 20600) — chosen in a new wizard **Delivery** step, with
"Download only" as the default. This raised three questions the build had to answer honestly, all
in the shadow of [ADR-0039](./0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md)
("a Channel carries the Message, never the estate").

1. **What crosses the wire?** A report artifact is the estate rendered — signals, subjects,
   withdrawals, severities. ADR-0039 forbids any of that leaving in a Channel POST. So what does a
   Channel receive when a report is cut?
2. **What records the send?** A `report_delivery` is the receipt that an artifact was *generated*.
   Does the send outcome go on that row, or somewhere else?
3. **Is this a Message delivery?** The `delivery` table and its runner already POST to Channels.
   Does a report ride that machinery?

## Decision

> **A scheduled report delivers to a Channel as a LINK-ONLY ready-message — the report name, the
> run's period, and a session-authed link to the in-instance artifact, and nothing else. The send
> is recorded in its own `report_notification` corpus, separate from both `report_delivery` and
> `delivery`, and rides the delivery package's shared signed-HTTPS transport and `queue.Backoff`
> retry curve. ADR-0039 stands: the report body never leaves the instance.**

Four parts.

### 1. The body is link-only — ADR-0039-clean by construction

The ready-message is a new, minimal document
([internal/report/notify.go](../../internal/report/notify.go) `ReadyBody`):
`{"kind":"report-ready","report":<name>,"period_start":…,"period_end":…,"url":<baseURL + "/reports/delivery">}`.
That is the whole body. It deliberately does **not** reuse `delivery.BuildBody`, which carries a
Message's firing (class, cause, census count, headline) — a report run is none of those. The
report body — the signals, subjects and withdrawals the artifact renders — **never crosses the
wire**. The Channel receives a notice and a link. An operator follows the link into the instance,
behind session auth, to read the artifact. This is the ADR-0039 rule applied to reports: the
estate stays home. Only the notice and the link leave. The `ReadyBody` type has nowhere to put an
estate row, and a test asserts the marshalled body carries only the five permitted keys and not
one estate term.

### 2. A schedule binds a Channel; NULL is download-only

Migration 22700 adds `report_schedule.channel_id BIGINT REFERENCES channel(id)`, nullable. A
non-NULL binding is the destination. **NULL is download-only** — the artifact is generated and
stays viewable in-instance, and nothing is sent. The free-text `delivery_target` column (migration
21700) is **superseded** by the binding: it is left in place (migrations are append-only) but
written empty and no longer read as the destination. The wizard's Delivery step offers "Download
only" plus one option per declared Channel (by URL). The recurring-reports table gains a Delivery
column rendering the bound channel's URL or "download only".

### 3. The send is its own corpus — a notify failure never hides a generated artifact

Migration 22800 adds `report_notification`, mirroring the `delivery` table's
retry/backoff/dead-letter shape (`pending → sending → delivered / undelivered`, an attempt budget,
a `run_after` the shared `queue.Backoff` pushes out). It is **separate from `report_delivery`** for
one decisive reason: **a send failure must never mark the artifact failed.** The artifact was
generated and stays viewable regardless of whether its ready-message reached the Channel. So the
receipt records generation and the notification records the send, independently:

- On a **2xx**, the notify runner flips the receipt to `delivered` and stamps `delivered_at`, in the
  same transaction that marks the notification delivered — the two facts move together.
- On **dead-letter** (the attempt budget spent), the notification is `undelivered` and the receipt
  is **left `generated`** — the artifact is still there to open.

The dispatcher enqueues **exactly one** notification per won tick, in the same transaction that
stamps the receipt, and only when the schedule binds a Channel (`shouldNotify`). A download-only
schedule enqueues nothing.

### 4. It rides the shared transport, not the Message path

A report run is not a Message (ADR-0081) and does not route by class (ADR-0091), so it does **not**
ride the `delivery` runner. But the *transport* — the SSRF guard, the HMAC signing, the
redirect-refusing POST, the `queue.Backoff` curve — is genuinely shared. Rather than copy it, the
delivery package exposes `SendSigned(ctx, doer, resolver, url, body, secret, now)`: the one place
the guard and request-build live. Both `delivery.Runner.send` and the report `NotifyRunner` call
it, so both send by identical rules and a fix to the guard fixes both.

## Consequences

- **The estate never leaves for a report.** The body type cannot carry a row. The guard is a test,
  not a comment. ADR-0039 holds for reports as it does for Messages.
- **A failed send never destroys a report.** The generated artifact and the send outcome are
  independent rows. A dead-lettered notification leaves the artifact viewable. This is the whole
  reason `report_notification` is not a column on `report_delivery`.
- **One transport, two callers.** `SendSigned` is the shared signed-POST path. `delivery.Runner`'s
  behaviour is unchanged (it now calls `SendSigned` internally). The SSRF guard, no-bearer rule and
  redirect refusal are enforced once.
- **Download-only is the default and the safe floor.** A schedule with no channel bound sends
  nothing — the pre-T7 behaviour — so an install that declares no Channel, or a schedule left on
  "Download only", is a no-op end to end.
- **`delivery_target` is dead weight, retained.** It stays in the schema (append-only) but is
  written empty and read nowhere as a destination. A later cleanup could drop it, on the record.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| POST the rendered report (or its artifact) to the Channel | Straight violation of ADR-0039 — the estate would leave the instance in a Channel body; the whole point of a *notify-with-link* is that it does not |
| Reuse `delivery.BuildBody` for the report notification | It carries a Message firing (class/cause/census/headline); a report run is not a Message, and the fields would either lie or leak. A distinct minimal body has nowhere to put estate |
| Record the send outcome on the `report_delivery` receipt (a `failed` state on send failure) | A send failure would then hide a perfectly good, generated, viewable artifact. The artifact's existence and its notice's delivery are different facts and must not share a state column |
| Route the report through the `delivery` runner and table | `delivery` keys on `message_id` and routes by class (ADR-0081/0091); a report has neither. It would need a fake Message or a class axis that does not apply |
| Copy the SSRF guard + signing into the report package | Two copies of a security-critical guard drift; `SendSigned` keeps it in one place with `delivery.Runner`'s behaviour unchanged |
