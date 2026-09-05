---
title: Notification channels
section: Signals & delivery
order: 2
description: Channels are the outbound https endpoints the worker POSTs each message to — declaring them, what fires, and why a delivery is a record and never a message.
---

# Notification channels

A **channel** is an HTTP endpoint the `worker` POSTs to when a message fires. It is
the outbound half of the notification model: the in-app store holds every message,
and a channel *carries the message, never the estate*. This guide covers the default
posture and how to declare and edit channels under **Settings → Channels**. It also
covers what fires, what a body contains, and why a `Delivery` is an operational record
of an attempt rather than a message.

The ruling is [ADR-0039](../adr/0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md).
The full enumeration is [`docs/spec/notification-channels.md`](../spec/notification-channels.md).
The outbound code lives in [`internal/delivery/`](../../internal/delivery/) and the
`Settings → Channels` handlers in [`cmd/web/settings.go`](../../cmd/web/settings.go).

---

## Default posture: nothing is routed anywhere

**No channel ships configured.** A default install writes and renders every message
in the store and carries it nowhere else. The delivery loop enqueues nothing until an
admin declares a channel. Configuring one is purely additive: it never changes what a
message says or when it fires, only whether a copy is POSTed to a channel.

The store is complete by construction. Every message is written and rendered whatever
happens to any channel. So a misconfigured, disabled, or dead channel loses no fact.
That is why the one surface that reports a delivery failure (the store) is the one
surface that cannot have one.

---

## The in-app store — the complement to channels

The store is the always-present surface a channel copies *from*. A global bell carrying
an unread count sits on every screen. In the V3 shell it opens the **Inbox** at
`/inbox`. The older messages fold at `/messages` stays as the viewer-readable
mirror. Both read the same message store, and read/unread state routes through the
shared `POST /messages/read` and `/messages/read-all` acts. Any logged-in account may
read them. Declaring a channel is admin-only.

The store is never a channel: it is not configured, cannot fail, and no message may skip
it. A channel is the opposite on every count — configured, able to fail, and skippable
by class routing or by there being no channel at all.

---

## Declaring a channel — Settings → Channels

Creating, editing, or deleting a channel is an **admin** act. The routes
(`POST /settings/channels`, `/settings/channels/update`, `/settings/channels/delete`)
sit behind an admin check. A viewer sees the channel table read-only, with no edit
controls.

A channel holds four things:

| Field | Value |
| --- | --- |
| **URL** | An absolute `https://` URL. `http://` is refused except to a **loopback address literal** (§ below). |
| **Secret** | Optional. Signs the body with `HMAC-SHA256`; **write-only** — set, replaced, or cleared, never rendered back. |
| **Classes** | A subset of `drift` · `coverage` · `clock`. At least one must be chosen — this is the only routing axis. |
| **Enabled** | Boolean. A disabled channel is skipped at routing time; disabling is not a delete. |

**Declaring one.**

1. Enter the URL.
2. Tick at least one class.
3. Optionally set a secret.
4. Declare it — it lands enabled.

There is no cap on the number of channels. Each is one POST per message it subscribes
to.

**Editing one.** The edit form updates the URL, classes, and enabled state together.
The secret has its own write path. An edit that leaves the secret field blank keeps
the stored one untouched. A typed value replaces it. The **clear the secret** box
removes it, and wins over any typed value.

**Deleting one.** Delete removes the channel row and is idempotent — deleting a row
that is already gone satisfies the intent either way.

### The URL rules

The URL is validated at configuration time, not at delivery time:

- An absolute `https://` URL is accepted. The exception is a host that is an IP literal
  in a non-globally-reachable range (loopback, link-local including the
  `169.254.169.254` cloud-metadata address, private ranges). That is refused as an
  internal address.
- `http://` is accepted **only** where the host is a loopback address *literal*. A
  loopback *hostname* is not accepted here: resolving a name to confirm it is loopback
  is exactly the rebinding surface the literal check avoids.

At delivery time the runner adds a second guard. It resolves the target host and
refuses to POST if any resolved address is non-globally-reachable. The HTTP dialer
also refuses the socket if the kernel is about to connect to such an address. This
closes the DNS-rebinding gap a literal-only config check leaves open. A body that is
the operator's attack surface never leaves for an internal host.

---

## What fires, and what a body contains

**This guide does not decide what fires.** The message set is closed elsewhere —
[ADR-0026](../adr/0026-the-facet-layer-is-evidence-not-a-channel.md),
[ADR-0029](../adr/0029-an-alert-fires-on-a-leg.md),
[ADR-0031](../adr/0031-membership-alerts-at-the-root-of-the-entering-subtree.md), and
[ADR-0033](../adr/0033-a-move-carries-the-rule-that-opens-at-fired.md) — and a channel
carries whatever that set produces. Every message names one of four **causes**, which
merge into the three routing **classes**:

| Cause | Meaning | Routing class |
| --- | --- | --- |
| `drift` | The estate's own object moved. | `drift` |
| `aperture` | Us — our own aperture widened. | `coverage` |
| `declared-input` | The operator's own declared input moved — e.g. an operator exclusion narrowing a subject out of the estate. | `coverage` |
| `threshold` | Only a clock or threshold was crossed; no measurement moved. | `clock` — *planned; see below* |

> **`threshold` is not yet emitted.** The producer folds a message at the *cause*,
> and it runs inside a completed batch's transaction over that batch's observations
> (`internal/queue/produce.go`). A `threshold` firing is by definition the one cause
> with **no** observation behind it — "only time passed". So a pure horizon crossing
> produces no observation, no batch, and no change for the producer to fold. The
> motivating case is a certificate crossing its *expiring* window. The certificate's
> `not_after` does not move when the clock passes it. So re-measuring the unchanged
> certificate yields no transition either. Emitting it faithfully needs a dedicated
> **clock-driven sweep**. That sweep is a periodic evaluator. It reads open certificate
> spans, fires `message.Threshold` on a horizon crossing, and persists what it fired so
> it fires once. It is a separate mechanism from the observation-driven batch producer,
> and is not yet built. Until it lands, `certificate-expiring` surfaces only as a
> read-time `Signal` on the Signals screen, never as a routed `threshold` message. The
> other three causes fire today.

A channel subscribes on a subset of the three classes and **nothing finer**. The cause
travels in the body as a field the operator reads. But the router never uses it as a
routing key. Per-cause, per-rule, and per-subject routing are all refused
([ADR-0091](../adr/0091-the-routing-unit-is-the-class-and-the-cause-is-refused-as-a-routing-key.md)).

**When a message is sent:** when the fold that caused it completes, with the census that
fold could see. It is not held for a set of tiers and not emitted incrementally per
batch. A message is computed **once at the cause** and never recomputed. The store
rendering and the channel body are two renderings of that one computation.

### The body

One JSON document per message, identical across every channel and every retry, carrying
exactly what the in-app message carries and **no rows**:

| Field | Carries |
| --- | --- |
| `message` | Stable, unique identifier — unchanged across retries, the receiver's de-duplication key. |
| `class` | `drift` · `coverage` · `clock`. |
| `cause` | Which of the four causes fired (read, never routed on). |
| `subject` | The bare `(kind, key)` of the thing the message fired at — the key, never a rendered label. |
| `instant` | The instant of the **cause**, not of the delivery attempt. |
| `headline` | The rendered sentence, byte-identical to the in-app message's. |
| `census` | A **count** where the firing has one, omitted otherwise — never the rows behind it. |
| `link` | An absolute URL into this instance at the fired-at object or scope (see below). |

No field enumerates the services behind a census count, the addresses behind a
`resolution` move, or the evidence behind a `Signal`. The receiver's disk and log
pipeline accumulate *what happened*. An operator who wants *what they have* follows the
link and authenticates.

### Authentication, and that it is one-way

When a secret is set, the POST carries an `HMAC-SHA256` signature over
`<unix-seconds>.<body>` in the `X-Verge-Signature` header. The `X-Verge-Timestamp`
header is always present because it is part of the signed input. With no secret, the URL
is the only credential, as it is for any incoming-webhook receiver. There is **no bearer
header, ever**. A bearer sits in the receiver's access log and a signature does not. And
the signature authenticates *us to them*, not the reverse. The channel is strictly
one-way: no callback, no ack, no fetch, no inbound surface.

---

## Links back to the instance — `VERGE_PUBLIC_URL`

Each body's `link` is an absolute URL into this instance at the object or scope the
message fired at. It is built from **`VERGE_PUBLIC_URL`**, set on the `worker` service:

```sh
# in .env / the worker service env
VERGE_PUBLIC_URL=https://verge.example.com
```

Left empty (the default), the link is simply **omitted rather than fabricated**. Add it
to the `worker` env whenever you configure channels. See
[running.md → Environment variables](running.md#environment-variables). The target
follows the mover:

- `drift` and `threshold` link to the fired-at subject's own page.
- `declared-input` links to its `Source`.
- An `aperture` widening links to the `Seed` whose scope moved, never to Coverage's
  standing aperture statement.

---

## A `Delivery` is a record of an attempt, never a message

A **`Delivery`** is the operational record of one POST attempt to one channel. It is not
a message: it is not about a subject, carries no evidence, has no cause, and **never
touches `Coverage`**. A delivery failure is not the world moving, our looking changing,
or a clock crossing — so it earns no place in the message set.

### What counts as a failure

| Outcome | Verdict |
| --- | --- |
| `2xx` | Delivered. |
| Any `3xx` | **Failed** — the redirect is not followed (it would move delivery to a host the operator never declared). |
| `4xx`, `5xx` | Failed. |
| Timeout, connection refused, DNS failure, TLS failure | Failed. |

**Retry budget: five attempts over roughly one hour, exponential, then dead-lettered.**
Deliveries ride the measurement queue's own retry / backoff / dead-letter curve rather
than a second mechanism — the same one [running.md](running.md#retention) governs for
scans. The budget is fixed and project-authored, not an operator dial: it governs
request rate against someone else's server. A dead-lettered delivery marks the delivery
undelivered and **leaves the message untouched**. *A dead-lettered `Delivery` licenses
no silence.* It means *we could not reach you*, never *nothing fired*.

The semantics are:

- At-least-once at the receiver (the message identifier de-duplicates).
- No ordering (retries and concurrent batches reorder).
- No back-pressure — a dead channel never blocks measurement or the writing of a `Span`.

### Where a delivery is seen

| Where | What it shows |
| --- | --- |
| On the **message**, in the store | Whether it was delivered, to which channels, whether any is dead-lettered, and, where one failed, its reason as drill-down. |
| On the **channel**, on its own surface | Current state, consecutive failures, and the last error string as **drill-down**. |
| Nowhere else | It is never a message and never a log line to go read. |

The raw error appears as drill-down in two places: on the message it failed to carry
(ADR-0108, ADR-0180 §3) and on the channel surface. On the channel it sits as a
configuration statement next to the thing the operator would change. It is never a
top-level log.

---

## Related

- [running.md](running.md) — deploying the stack and setting `VERGE_PUBLIC_URL` on the
  `worker` service.
- [using.md](using.md) — the first-run checklist. Its **Notification channels** note
  points here.
- [reports.md](reports.md) — the other store-side read surface.
- [troubleshooting.md](troubleshooting.md) — diagnosing a channel that fails to deliver.
