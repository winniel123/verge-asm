---
title: Troubleshooting
section: Operating
order: 3
description: Confirming silent failures on a first run — a scan that ran vs. one that failed quietly, empty exposure, undelivered notifications, boot migrations, and the Gap cases.
---

# Troubleshooting

verge-asm's whole posture is to **tell you when it cannot tell you**. A claim it
cannot construct becomes a `Coverage` statement, not a fabricated answer and not a crash.
That makes most first-run trouble *quiet* rather than loud. The command exits `0`, the
page renders, and nothing measured. This guide is the checklist for confirming that a
thing actually happened, oriented around the failures a first run hits most.

Read it after [running.md](running.md) (the operational digest) and
[first-run.md](first-run.md) (the mental model). For exposure specifically, keep
[prober.md](prober.md) open. For delivery, keep [notification-channels.md](notification-channels.md).

---

## A scan that ran vs. one that failed in silence

A scan can commit as `completed` and still measure nothing. The classic case is a
`dns` scan pointed at a resolver that answers nothing. That scan yields empty records
and a `Gap` while committing successfully. "The trigger exited `0`" is not "it produced
data." Confirm dispatch in this order:

1. **The trigger's own output.** A manual `docker compose run --rm worker -trigger dns`
   drains synchronously and prints how many jobs it enqueued and that it finished:

   ```
   worker: triggered dns, 3 job(s) enqueued
   worker: trigger drained
   ```

   `0 job(s) enqueued` means nothing was queued at all. The usual cause is no seed scope
   for that kind, or a **disabled** scan. A trigger *refuses* a disabled kind rather than
   running it once. `cold` ships disabled — see
   [running.md → On-demand scan triggers](running.md#on-demand-scan-triggers).

2. **The worker logs.** The long-running daemon logs the dispatcher, delivery and
   retention runners here too:

   ```sh
   docker compose logs worker | tail -n 40
   ```

3. **`/scans`.** The scans page lists dispatches and their runs. Each run drills into
   **`/run/<id>`**, where an individual dispatch's job outcomes live — the surface for "did
   *this* run touch anything."

4. **Coverage.** `/coverage` is where *we could not construct this claim* lives — `Gap`s,
   unread apertures, unevaluable rules. A scan that ran-but-resolved-nothing appears here
   as a `Gap`, **not** as an error and **not** as absent data. If you expected subjects and
   Coverage shows a `Gap`, suspect the resolver or an empty scope first. Suspect a crash last.

5. **Subjects.** Once a batch genuinely commits data, the `Name`s, `Address`es, `Service`s
   and `Endpoint`s appear under **Subjects** (served from `/inventory`), each drilling into
   its facet timelines.

> The single setting most likely to cause a silent empty `dns` scan is the `local`
> vantage's resolver on an off-compose install. It ships `127.0.0.11:53` (Docker's embedded
> DNS), which is not routed on a bare-metal or host-network install. Set it to your own
> recursive resolver before the first `dns` trigger. See
> [running.md → The `local` vantage resolver](running.md#the-local-vantage-resolver).

---

## Exposure is empty or withheld

`Exposure` is composed from **two `Reach` legs** — an `internet`-class vantage's reading
and an `internal`-class one's — and **exists only where both legs hold a value**. If
Exposure is blank, one leg is missing, and the system is degrading to internal-only on
purpose. It will **never** print `firewalled` or `exposed` for something it did not observe
from the internet.

- **No prober provisioned.** An internet-class vantage exists exactly where a second host
  observed this instance's presented address, so exposure requires a **prober**,
  unconditionally. Until one exists, exposure claims are withheld and only the surviving
  (internal) leg's `Reach` renders on its own. Provision one:
  [prober.md](prober.md).

- **The hairpinning trap.** Deploying the instance and the prober **both outside** your
  network gives you only the internet leg. Two outside observers are still one side of the
  boundary. Probing your own public address from inside hairpins and never traverses the
  inbound policy, so that reading would be a trap, not a measurement. You need a vantage
  **inside** your network. Declare an address scope covering the instance's presented
  address (a `/32` or `/128`) so its own vantage verifies `internal`. See
  [first-run.md → why `Exposure` needs two legs](first-run.md#vantage-class-and-why-exposure-needs-two-legs).

- **Both vantages verify `internet`.** After provisioning a prober but before declaring
  your egress, both legs are internet-class and there is still no internal leg — Exposure
  stays non-constructible. Declare the egress verge rendered (prober step 3) to unlock it:
  [prober.md → Confirm exposure is now constructible](prober.md#step-5--confirm-exposure-is-now-constructible).

Adding the first internet vantage does **not** escalate your estate overnight. It *opens*
the Exposure timelines (recorded as `revealed`, one coverage-class message) rather than
transitioning every service to `exposed`.

---

## Nothing is being delivered

Notification delivery runs inside `worker` and is a **no-op on a default install**. No
channel ships configured, so nothing is ever routed until an admin declares one. If
signals fire but no notification arrives, check in this order:

- **A channel is configured.** With no channel declared there is nothing to route, and
  the delivery runner sits idle by design. Configure one first —
  [notification-channels.md](notification-channels.md).

- **`Delivery` failures in the logs.** A routed delivery that the endpoint rejects rides
  the queue's shared retry/backoff curve — **five attempts over roughly an hour, then
  dead-lettered**. It is never a second retry mechanism. The shapes:

  ```
  delivery: 42 attempt 2 failed, retrying: <reason>
  delivery: 42 dead-lettered after 5 attempts: <reason>
  ```

  A dead-lettered delivery is a delivery problem only. The underlying `Message` is never
  touched, so nothing about the finding is lost.

- **Missing `VERGE_PUBLIC_URL`.** The absolute base for each notification body's link. It
  is **not required** for delivery — an empty value leaves the link *off* rather than
  fabricating one, so bodies still send, just without a click-through. If notifications arrive
  but have no link back, set `VERGE_PUBLIC_URL` on the **`worker`** service env (not `web`).
  See [running.md → Environment variables](running.md#environment-variables).

---

## Migrations on boot

`web` runs the goose migrations against Postgres **on startup** — there is no separate
migrate step, and `worker` applies none (web owns that). The schema change lands *before*
the new `web`/`worker` code serves traffic, which is why an upgrade wants a `pgdata`
backup first.

A failed migration is **fatal for `web`** — the process exits rather than serving against a
half-migrated schema. So the symptom is `web` restart-looping and never reaching
`running / healthy` in `docker compose ps`. Read it from the `web` logs:

```sh
docker compose logs web | tail -n 30
```

The failure is logged with a `web: migrate:` prefix and the underlying `apply migrations`
cause, e.g.:

```
web: migrate: web: apply migrations: <goose error>
```

A healthy boot instead reaches `web: listening on :8080`. If `web` never passes the
migrate line, the cause is in Postgres — an unreachable DB, a bad `POSTGRES_PASSWORD`, or a
migration that could not apply. It is not in the UI.

---

## `compose up` fails with "bind source path does not exist"

Only in external-database mode. The full error names a path:

```
Error response from daemon: invalid mount config for type "bind":
bind source path does not exist: /etc/ssl/certs/my-db-ca.crt
```

That path is your **CA root**, and `POSTGRES_SSLROOTCERT_SRC` points at it. The error means
the variable is set to somewhere the file is not. Put the CA root there, or correct the
variable:

```sh
ls -l "$POSTGRES_SSLROOTCERT_SRC"
```

This failure is deliberate. The override sets `create_host_path: false` so Docker refuses
the mount instead of quietly creating an **empty directory** at that path — which would boot
the stack and then fail `verify-full` with a TLS error pointing nowhere near the real cause.

A *different* error comes from compose before any container exists, when the variable
is not set at all: `required variable POSTGRES_SSLROOTCERT_SRC is missing a value`.
Both guards exist because neither covers the other: `docker compose config` reports
the unset variable, and exits 0 on a source path that does not exist. See
[running.md → An external Postgres](running.md#an-external-postgres-with-a-verified-tls-connection).

---

## "Why does it say Gap / Coverage incomplete?"

A `Gap` is the honest *we-could-not-construct-this*, not an error. Common causes:

- **A scan resolved nothing** — the empty-`dns` case above. The `Gap` is the product
  telling you the batch committed but found no records. Chase the resolver or the scope.
- **Only one exposure leg** — Exposure is unconstructible until both legs hold a value
  (above). Coverage says so plainly instead of guessing.
- **A CDN-fronted domain.** If a name resolves to a **CDN, anycast, or reverse-proxy edge**
  (Cloudflare, Fastly, and the like), probing its resolved IPs measures **that edge, not
  your origin**. These edges complete the TCP handshake on nearly every port, so a `hot`
  scan reports the whole range as `reached`. The numbers are real. They are about the wrong
  host. Declare your **origin IPs as an address scope** — each address is then walked
  directly. Prefer that over a custody extension on a CDN-fronted name. Full treatment:
  [first-run.md → CDN-fronted domain caveat](first-run.md#caveat-scanning-a-cdn-fronted-domain-measures-the-edge-not-your-origin).

Coverage is as much the job as reading Exposure — read `/coverage` before you conclude the
estate is empty. Deeper reading of these surfaces is in
[reading-the-estate.md](reading-the-estate.md).

---

## Setup token and first-run access

On first boot, with **no accounts yet**, `web` opens a single-use setup window and logs the
token:

```
web: no accounts yet — open /setup with this single-use token: <token>
```

- **Pin it instead of reading the logs.** Set `VERGE_SETUP_TOKEN` on the `web` service and
  that value is used verbatim. It is the one config that may live in the environment, because
  it must exist before the database has an admin.

- **Token spent / `/setup` closed.** The window shuts the instant the **first account
  exists** — that is what makes the token single-use. Once an account exists, `web` carries
  no setup token at all, so `/setup` **redirects to `/login`** rather than offering a form.
  If you reach `/login` when you expected `/setup`, an admin was already created. Sign in,
  or recover the password through `/forgot`. On a host with no mail the reset link is
  written to the `web` logs, same as the setup token.

- **"Invalid setup token."** The value submitted did not match. Re-copy it from the `web`
  logs (or from your `VERGE_SETUP_TOKEN`) as one unbroken string — no trailing whitespace or
  line-wrap.

---

## Healthz and the self-test

Two surfaces confirm a service is live without opening the UI, both reported by
`docker compose ps`:

- **`web`'s `/healthz`.** The `-healthcheck` flag hits `/healthz`, which records a heartbeat
  and returns `{"status":"ok","checked_at":…}` with `200`. A `503` means the heartbeat write
  to Postgres failed — a database problem, logged as `web: healthz: record heartbeat:`.
  `worker`'s check verifies only that it can reach Postgres.

- **The prober self-test.** On startup `worker` execs its prober once to prove the
  job-spec-in / NDJSON-out contract works. It is logged, **not fatal**:

  ```
  worker: prober self-test ok: {…}
  worker: prober self-test failed: <reason>
  ```

  A failed self-test does not stop the worker. But it is your earliest signal that the
  pushed-binary path is broken before you try to provision a prober.

---

## Where to look next

| Symptom | Start here |
| --- | --- |
| Scan ran but no subjects | [Confirm a scan ran](#a-scan-that-ran-vs-one-that-failed-in-silence), then `/coverage` |
| Exposure blank or withheld | [prober.md](prober.md) |
| Signals fire, nothing arrives | [notification-channels.md](notification-channels.md) |
| `web` restart-looping on boot | [Migrations on boot](#migrations-on-boot) |
| `compose up` rejects a bind mount | [bind source path does not exist](#compose-up-fails-with-bind-source-path-does-not-exist) |
| Reading Coverage / Exposure in depth | [reading-the-estate.md](reading-the-estate.md) |
