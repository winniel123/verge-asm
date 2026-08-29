---
title: JSON API
section: Signals & delivery
order: 5
description: The read-only /api/v1 JSON surface — enabling it, bearer auth with personal tokens, the five read endpoints and their shapes, and the 404-when-disabled / 405 / 401 semantics.
---

# JSON API

verge-asm exposes a small, **read-only** JSON surface at `/api/v1`. Your own tooling
can **pull** the current estate on its own cadence, rather than a human clicking export
in a browser. That tooling might be a dashboard, a CMDB sync, or a ticketing hook. It is
the machine-readable counterpart to [reading-the-estate.md](reading-the-estate.md),
and the *pull* sibling to the outbound [notification channels](notification-channels.md)
that *push* when a signal fires.

Two things bound it, and both are load-bearing:

- **It can only read.** No endpoint under `/api/v1` mutates anything — there is no
  write surface to enable. A leaked token reads the inventory. It cannot change the
  estate or its configuration.
- **It is off until an admin turns it on.** A fresh instance answers `404` on every
  `/api/v1` path, byte-for-byte indistinguishable from a build that has no API at
  all. Minted tokens stay inert until the surface is enabled.

The rationale for both — why a read-only bearer bypasses nothing the login TOTP was
guarding, and why "off" is made indistinguishable from "absent" — is
[ADR-0123](../adr/0123-a-token-api-is-read-only-opt-in-and-a-bearer-path-separate-from-sessions.md).
The handlers live in [`cmd/web/api_v1.go`](../../cmd/web/api_v1.go) and the bearer
spine in [`cmd/web/api_auth.go`](../../cmd/web/api_auth.go).

---

## Enabling the API

The surface ships **disabled**. An admin enables it under **Settings → API access**
(`POST /settings/api`) — a single instance-wide switch that records **who** enabled
it and **when**. Until it is enabled, every `/api/v1` request `404`s no matter how
valid the token. The Profile token card then shows the inert-tokens note.

Enabling the surface is a deliberate, audit-recorded choice. It opens a second
authenticated way to read the full inventory. So the default posture of every
instance is *no API reachable at all* until an admin decides otherwise.

Disabling it again is immediate — the very next request `404`s — and no token is
destroyed. The tokens simply go inert until an admin enables the surface again.

---

## Authenticating — a personal token as a bearer

Each request carries a personal API token as a bearer credential:

```
Authorization: Bearer vg_pat_…
```

Mint the token on **Profile → Personal API tokens** (see
[authentication.md → API tokens](authentication.md#api-tokens)). The plaintext
`vg_pat_…` string is shown **once**, at creation. Verge keeps only its SHA-256 hash.

The token authenticates *which account* the request acts as. The account's role is
read **live** from its row on every request, never frozen into the token. So
demoting or disabling that account takes effect on its tokens' very next request,
with no reissue. Because the whole surface is read-only, an admin's token and a
viewer's token can both **only read**. The role distinction changes nothing about
what `/api/v1` will do.

### The bearer path is separate from your browser session

The API path and the HTML/session path share **no** authentication machinery:

- A token **never** mints a cookie and is **never** accepted on the HTML surface.
- The session cookie is **never** accepted on `/api/v1` — a browser session cannot
  drive the API by riding its cookie.

So the API cannot be reached by CSRF or by a stolen cookie (it reads no cookie). And
the HTML app cannot be reached by a stolen token (it mints nothing the app trusts). A
compromise of one credential class is not a foothold on the other.

### Last used

Each successful token request records a coarsened **Last used** time — at most once
per hour per token. So it is a "is this token still live?" signal rather than a
fine-grained access log of your own integration traffic. A token that has never
authenticated a request reads as **never**. Last used is visible on
**Profile → Personal API tokens**.

---

## Response semantics

Every response is `application/json; charset=utf-8`. There is no HTML, no cookie, and
no redirect-to-sign-in on this surface. A failure is a status code and a JSON body,
never a login page.

| Situation | Status | Meaning |
| --- | --- | --- |
| API disabled, or path unknown under `/api/v1` | `404` | Surface-off is made indistinguishable from surface-absent. A disabled instance and an unknown path both return the bare `404 page not found`, so a probe cannot tell "API exists but is off" from "this build has no API". Checked **before** the credential — a disabled instance never emits `401`/`403`. |
| Any non-`GET` verb on a resource | `405` | The surface is read-only. `POST`/`PUT`/`PATCH`/`DELETE` are refused with `Allow: GET` and no body — no mutating verb is even routed. |
| Missing / malformed / unknown token, or a token whose account is gone | `401` | Uniform for every credential failure (with a `WWW-Authenticate: Bearer` challenge); the body distinguishes none of "no token", "unknown token", "removed account". |
| Valid token on an enabled surface | `200` | The JSON projection for that resource. |
| Store read failure | `500` | `{"error":"internal error"}` — a fixed label carrying no estate shape or stack; the detail is only logged. |

Both "disabled" and "unknown path" answer `404`. So treat any `404` from `/api/v1`
as "the surface is not serving this". Check that an admin has enabled it. Then check
that the path is one of the five below.

---

## Endpoints

Five `GET` resources project the estate. Each is a thin, stable JSON projection of a
read the HTML console already renders. Each uses the same store reads and in-process
builders the pages call, with no new derivation. The reads that project
current-subject data pass the same live-tier gate the console uses. So an evidential
row past its retention bound is structurally unreadable here, exactly as it is on the
pages.

| Resource | Projects | Console equivalent |
| --- | --- | --- |
| `GET /api/v1/inventory` | every open span, grouped by subject | [Inventory](reading-the-estate.md) |
| `GET /api/v1/subjects` | the current Name / Service / Endpoint census | [Subjects](reading-the-estate.md) |
| `GET /api/v1/drift` | the batch-grouped transition feed (default 7-day window) | [Drift](reading-the-estate.md) |
| `GET /api/v1/signals` | the fired-signal census — open, annotated, withdrawn | [Signals](signals.md) |
| `GET /api/v1/coverage` | the aperture census meters, per declared scope | [Coverage](reading-the-estate.md) |

Timestamps are RFC-3339 UTC strings. A value the system currently cannot state is
returned honestly (an empty string, a `gap` flag, or a `null` total) rather than a
fabricated zero.

### `GET /api/v1/inventory`

Every open span grouped by subject — what you have right now.

```json
{
  "groups": [
    {
      "kind": "service",
      "label": "Services",
      "subjects": [
        {
          "kind": "service",
          "key": "…",
          "type": "…",
          "facets": [
            { "label": "Port", "value": "443/tcp", "gap": false, "since": "2026-08-20T14:05:00Z" }
          ]
        }
      ]
    }
  ]
}
```

A facet the system currently cannot state carries `"gap": true` with an empty
`"value"` — never a zero in place of a real read.

### `GET /api/v1/subjects`

The current subject census, split into the three families.

```json
{
  "names":     [ { "key": "…", "value": "example.com",     "observed_at": "2026-08-25T09:00:00Z" } ],
  "services":  [ { "key": "…", "value": "203.0.113.4:443", "observed_at": "2026-08-25T09:00:00Z" } ],
  "endpoints": [ { "key": "…", "value": "…" } ]
}
```

`observed_at` is omitted when the row carries no instant.

### `GET /api/v1/drift`

The batch-grouped transition feed over the default 7-day window — what moved.

```json
{
  "period": "7d",
  "transition_count": 3,
  "movement": { "opened": 2, "closed": 1 },
  "batches": [
    {
      "label": "…",
      "meta": "…",
      "events": [
        {
          "change": "opened",
          "family": "service",
          "subject": "…",
          "detail": "…",
          "time": "2026-08-25T09:00:00Z",
          "reason": "…"
        }
      ]
    }
  ]
}
```

`transition_count` is the total across all batches. `movement` tallies events per
change kind. `reason` is omitted when a transition carries none.

### `GET /api/v1/signals`

The fired-signal census, one array per tab.

```json
{
  "open": [
    {
      "id": "…",
      "signal": "…",
      "severity": "…",
      "asset": "…",
      "ip": "203.0.113.4",
      "port": "443",
      "first_seen": "2026-08-20T14:05:00Z",
      "last_seen": "2026-08-25T09:00:00Z"
    }
  ],
  "annotated": [],
  "withdrawn": []
}
```

`ip`, `port`, `first_seen` and `last_seen` are omitted when absent.

### `GET /api/v1/coverage`

The aperture census — one meter per declared scope.

```json
{
  "meters": [
    { "label": "…", "counted": "128", "total": null,   "unit": "names",     "pct": 0,  "detail": "…" },
    { "label": "…", "counted": "40",  "total": "1024",  "unit": "addresses", "pct": 4,  "detail": "…" }
  ]
}
```

`total` is `null` for a name scope — a census bar that enumerates nothing on its own.
It is a pre-formatted string for an address scope, exactly as the Coverage screen
shows. Neither claims a proportion of the estate.

---

## A worked call

```sh
# Enable the surface first: Settings → API access (admin).
# Mint a token: Profile → Personal API tokens (copy the vg_pat_… value once).

curl -s https://verge.example.com/api/v1/inventory \
  -H "Authorization: Bearer vg_pat_…"
```

- A `404` means the surface is disabled (or the path is wrong) — enable it under
  **Settings → API access**.
- A `401` means the token is missing, malformed, unknown, or its account is gone.
- A `405` means you used a non-`GET` verb. The API only reads.

> **Versioning is in the path.** Today's surface is `/api/v1`, and its shapes are the
> stable projections above. A future revision would live under a new version prefix
> rather than change these in place.

---

## See also

- [authentication.md → API tokens](authentication.md#api-tokens) — minting,
  revoking, and how a token differs from a session.
- [reading-the-estate.md](reading-the-estate.md) — the human console the API mirrors,
  and which page answers which question.
- [ADR-0123](../adr/0123-a-token-api-is-read-only-opt-in-and-a-bearer-path-separate-from-sessions.md)
  — why the surface is read-only, opt-in, and separated from sessions.
</content>
