---
title: Integrations
section: Operating
order: 10
description: Integrations are third-party install tiles whose install is a Declared act — the tile states, install and disconnect, and why an integration is never a delivery channel.
---

# Integrations

An **integration** is a third-party install tile on **Settings → Integrations**: Slack,
PagerDuty, Jira, Splunk, an S3 bucket, and the rest of the catalogued library. Installing
one is a **Declared** act — the operator says "this integration is installed against this
deployment" — and that single choice is the only per-install fact verge-asm stores. This
guide covers what a tile holds, its three states, installing and disconnecting, and the
one distinction that trips operators up: an integration is **not** a delivery channel.

The tile grid, its catalogue, and the install/disconnect handlers live in
[`cmd/web/integrations.go`](../../cmd/web/integrations.go); the install-state table is
[`db/migrations/21200_integration_state.sql`](../../db/migrations/21200_integration_state.sql).
The screen is the design-system console screen
`design-system/examples/console/Integrations.jsx` ported verbatim
([ADR-0110](../adr/0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md)).

---

## What a tile is — catalogue plus one install fact

Like the [source catalogue](sources.md), the integration library is **release data**: a
tile's identity, letter mark, category, description, and the **consent grants** it would
receive are authored by the project and held in the binary — the same for every install.
The catalogue is the `integrationCatalog` slice in
[`cmd/web/integrations.go`](../../cmd/web/integrations.go); eight tiles ship, in four
categories (Notify, Ticketing, SIEM, Storage).

The **only per-install fact** is the operator's install state, and that is all the
`integration_state` table holds — one row of `(slug, state)`, keyed on the tile's stable
**slug** (never the display name, so the catalogue can relabel a tile without stranding a
row):

```sql
CREATE TABLE integration_state (
    slug  TEXT PRIMARY KEY,
    state TEXT NOT NULL
);
```

Like every Declared term, the install carries **no timeline, no actor, and no instant of
its own** ([ADR-0073](../adr/0073-an-operator-dial-carries-no-author-however-specific-its-target.md),
[ADR-0093](../adr/0093-an-instant-on-a-declared-term-is-earned-by-an-act-nothing-else-dates.md)):
re-installing is an upsert of the one current state, disconnecting deletes the row, and
neither keeps a history to read.

---

## The three tile states

A tile is always in exactly one of three states. Two of them are **stored** in
`integration_state`; the third is the **absence** of a row.

| State | Stored value | What it means | Badge |
| --- | --- | --- | --- |
| **`available`** | *(no row)* | Nothing installed. The absence of a row **is** the available state — it is never a stored sentinel. | `available` |
| **`installed`** | `installed` | Installed and delivering. | `installed` |
| **`needs-config`** | `needs-config` | Installed but not yet configured to deliver — a pending/attention state. The drawer shows a "Configuration needed" callout; finish setup and deliveries resume. | `needs config` |

The effective state is computed in `fillIntegrationsSection`: the stored value where a row
exists, and `available` **otherwise**. No install state is ever fabricated — a fresh
deployment shows every tile `available`, and only an operator's act moves a tile off it.
`installed` and `needs-config` together are the **connected** states — the two that carry
a Disconnect.

> **Divergence to note.** The design-system tile
> [`IntegrationTile.jsx`](../../design-system/components/display/IntegrationTile.jsx)
> names its third state `attention` and labels it *"needs attention"*. The shipped Go
> render uses the slug `needs-config` and the label *"needs config"* for the same
> pending/error state. The state model is identical (available · installed ·
> pending); only the wording was tightened on the port.

---

## Installing an integration

Installing is **admin-only** and consent-gated:

1. Open a tile to read its **drawer**. The drawer lists the **consent grants** the
   integration would receive — what it can *read* (signals, observations, drift
   summaries) and, louder, any **write-back** grant. Grants are **all-or-nothing**: a
   display of what install means, not checkboxes. Reaching the Install button means the
   grants have been shown, so the click **is** the consent.
2. Press **Install**. This `POST`s to `/settings/integrations/install`, which upserts the
   tile's slug to state `installed`.

A **write-back** grant (e.g. Jira issue transitions, PagerDuty acknowledgements) is a
**proposal, never an act** — the estate is never mutated by an integration. An operator
confirms any proposed annotation; the integration only suggests it.

An unknown slug is refused with `400` rather than written.

---

## Disconnecting an integration

Disconnect is **admin-only** and never fires on the tile click. The connected tile's
drawer offers a **Disconnect link** to a **confirm dialog**; only the dialog's confirm
button `POST`s to `/settings/integrations/disconnect`, which **deletes the row**, returning
the tile to `available`.

Disconnecting **forgets the local install only**. Nothing is deleted on the integration's
own side — as the confirm dialog says, *"Nothing was deleted on the {category} side — this
only forgets the local install."* A `confirm` parameter aimed at an `available` tile offers
no destructive act, because there is nothing to disconnect.

Both routes are wrapped in `requireAdmin` in
[`cmd/web/handlers.go`](../../cmd/web/handlers.go):

```go
mux.HandleFunc("POST /settings/integrations/install", s.requireAdmin(s.installIntegration))
mux.HandleFunc("POST /settings/integrations/disconnect", s.requireAdmin(s.disconnectIntegration))
```

A non-admin `POST` is refused with `403`; an anonymous one is bounced to `/login`.

---

## An integration is not a delivery channel

This is the distinction to hold onto, and the code is **explicit** about it. From
[`cmd/web/integrations.go`](../../cmd/web/integrations.go):

> An integration is a third-party install tile, **NEVER a delivery channel** (which
> carries messages) and **NEVER a discovery source** (which observes): the word stays
> distinct.

The `integration_state` migration says the same, and adds *why* it keeps its own table:

> An integration is NOT a channel and NOT a source: it is a third-party install tile,
> distinct from a delivery channel (which carries messages) and from a discovery source
> (which observes) — so it keeps its own table, never folded into channels or
> `source_state`.

In practice:

- A **[notification channel](notification-channels.md)** is the delivery primitive. It
  needs **no integration** — Settings → Channels delivers **raw JSON to any URL** on its
  own. The Integrations screen states this in a banner: *"Channels need no integration…
  Integrations add formatting, acks, and state mapping on top."*
- An **integration** is a formatting-and-mapping layer on top of that push: Slack cards,
  PagerDuty incidents, Jira issues, Splunk HEC events. It **receives** what verge-asm
  pushes; the flow is one-way where possible (*"Verge pushes, integrations receive"*).
- A **[discovery source](sources.md)** is neither — it *observes* and admits subjects into
  the estate. Sources and Integrations are separate Settings tabs and never bleed into
  one another.

So: reach for a **channel** to route a signal somewhere; reach for an **integration** to
make that delivery land as a native object in a third-party tool. See
[notification-channels.md](notification-channels.md) for the delivery side, and
[using.md](using.md) for where notifications fit in the operating loop.
