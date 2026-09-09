---
title: Running verge-asm
section: Operating
order: 1
description: Deploy, configure, and operate the verge-asm stack with Docker Compose.
---

# Running verge-asm

How to deploy, configure and operate the stack. For what to *do* once it runs, see
[using.md](using.md). To build and test from source, see [verifying.md](verifying.md).

The authoritative reference for every decision below is
[`docs/spec/packaging-and-configuration.md`](../spec/packaging-and-configuration.md).
This guide is the operational digest.

---

## Prerequisites

- **Docker** with the Compose plugin. That is the whole list — the Go toolchain,
  `sqlc` and every dependency are baked into the images.
- A host that can reach the addresses and names you intend to measure.
- For **exposure** findings: a separate Linux host to run as a prober. The
  [`deploy/prober/`](../../deploy/prober/) recipe starts one with `docker compose`.
  The worked walkthrough is [prober.md](prober.md) (overview in
  [using.md → Provision a prober](using.md#3-add-an-internet-vantage-provision-a-prober)).

The images build and run on `linux/amd64` and `linux/arm64` only. Both are
first-class. The prober binary for *both* architectures ships in every image, so an
`arm64` instance can push to an `amd64` VPS and vice versa.

---

## First launch

```sh
cp .env.example .env
$EDITOR .env                 # set POSTGRES_PASSWORD
docker compose up -d --build
```

`web` runs the goose migrations against Postgres on startup — there is no separate
migrate step. Watch the stack start:

```sh
docker compose ps            # all three services -> running / healthy
docker compose logs -f web
```

Then follow [using.md](using.md) for the setup token and first-run checklist.

---

## Configuration

> **The environment configures the process. The database configures the product.**
> The environment holds only what must exist *before* the database does. Everything
> you declare — seeds, exclusions, scans and their cadences, source enablement,
> vantages, notification routing — is a row edited through the UI by an authenticated
> admin, because those acts need an author in the audit trail. There is no config
> file to mount.

### Environment variables

Set these in `.env`, or in your orchestrator when you run the binaries outside compose.

Compose reads `.env` for *interpolation* only. A variable reaches a container just where
`docker-compose.yml` names it. Every `VERGE_*` row below is named there, on the service
the **Service** column gives. The four rows marked **not read from `.env`** are the
exception. The image layout or the published port fixes each of those, so changing one
means an edit to `docker-compose.yml` itself.

| Variable | Service | Required | Default | Purpose |
| --- | --- | --- | --- | --- |
| `POSTGRES_PASSWORD` | all | **yes** | — | DB credential. Compose *fails* rather than defaulting it. |
| `POSTGRES_USER` | all | no | `verge` | DB user. |
| `POSTGRES_DB` | all | no | `verge` | DB name. |
| `VERGE_LISTEN_ADDR` | web | no | `:8080` | Listen address for the UI. **Not read from `.env`** — `docker-compose.yml` pins it to `:8080`, which is the container port the `ports:` mapping publishes. Move the *host* side with `VERGE_BIND` instead. |
| `VERGE_SETUP_TOKEN` | web | no | generated | Pin the first-run setup token instead of reading it from the logs. Single-use. |
| `VERGE_SECURE_COOKIES` | web | no | off | Set truthy (`1`/`true`/`yes`/`on`) when a TLS-terminating proxy fronts `web`, so the session cookie is marked `Secure` even though `web` sees plain HTTP. |
| `VERGE_TRUSTED_PROXIES` | web | no | empty | Comma-separated IPs and/or CIDRs of every hop between the client and `web` (e.g. `10.0.0.0/8, 192.0.2.7`). **Empty means no proxy is trusted.** The client IP is then the immediate peer, and `web` reads no `X-Forwarded-For` header at all. Set it whenever a reverse proxy fronts `web`. A fronted deployment that leaves it empty gives every client the same rate-limit key. Five failed logins from anyone then lock every account on the instance. The value keys the login rate limiter and **nothing else** — never identity, authorization or audit. Name the proxy's address as `web` sees it, which is the address the proxy connects *from*. A malformed entry fails the boot rather than trusting a smaller set in silence. See [ADR-0159](../adr/0159-an-unnamed-proxy-is-never-trusted-so-the-client-ip-is-the-immediate-peer-and-a-fronted-deployment-must-name-its-proxies.md). |
| `VERGE_PROBER_PATH` | worker | no | `/app/prober` | Path to the prober binary inside the image. **Not read from `.env`** — the path is where the `Dockerfile` puts the binary, so `docker-compose.yml` pins it. |
| `VERGE_PROBER_DIR` | worker | no | `/app/probers` | Directory of per-architecture prober binaries the off-host router pushes to remote SSH hosts, arch-matched by `uname` (an arm64 instance pushes an amd64 binary and vice versa). `VERGE_PROBER_PATH` is the own-arch single-binary fallback. **Not read from `.env`** — the directory is where the `Dockerfile` puts those binaries, so compose leaves the image default standing. |
| `VERGE_STATE_DIR` | web, worker | no | `/app/state` | On-disk home for generated secrets (session key, prober SSH private key). **Not read from `.env`** — `/app/state` is where the `web-state` and `worker-state` volumes mount, so compose leaves the image default standing. A value pointing anywhere else would write those secrets to the container's throwaway layer, and every recreate would invalidate all sessions and orphan every provisioned prober. Move the *storage* by repointing the volume. |
| `VERGE_PUBLIC_URL` | worker | no | empty | Absolute base URL used to build the link in each notification body. Empty leaves the link off rather than fabricating one. Set it in `.env` when you configure notification channels. |
| `VERGE_EXTERNAL_URL` | web | no | empty | The trusted origin the deployment is reached at (e.g. `https://verge.example.com`). It is the base for the SSO OIDC callback/redirect URL, taken from this value instead of the request `Host` header — **set it before configuring SSO**, or the callback URL registered with your IdP will not match and login fails. Empty falls back to the request host. Distinct from `VERGE_PUBLIC_URL`: `EXTERNAL_URL` is the **web** callback origin; `PUBLIC_URL` is the **worker** base for notification-body links. See [sso.md](sso.md). |
| `VERGE_LOG_RESET_LINKS` | web | no | off | When set to any non-empty value, logs the plaintext password-reset link. Off by default — the link is a bearer credential and must not land in logs (CWE-532). Enable only knowingly on a mail-less host that needs the link out of band from its own logs. |
| `VERGE_VERSION` | web, worker | no | `dev` | The build version shown in the UI footer and on **Settings → Instance**, and the version the update check compares against. **It applies to an unstamped build only.** A released image carries the version stamped in at build time and ignores this variable, logging one line at boot when you set it anyway. An unstamped build — the compose stack builds one — reads this value, and reads `dev` when it is unset. The number is bare (`0.1.0`), with no leading `v`. |
| `VERGE_RELEASE_FEED_URL` | worker | no | GitHub latest-release | The release feed the worker's daily update check reads **when checks are enabled**. Defaults to this repository's GitHub latest-release endpoint; point it at your own repo for a fork. Ignored while the update check is off — an air-gapped instance makes no call at all. See [Version & updates](#version--updates). |

`docker-compose.yml` assembles `DATABASE_URL` from the `POSTGRES_*` values. You
only set it directly if you run the binaries outside compose, or in external-database
mode below.

Three more apply **only** in external-database mode, and only when
`docker-compose.external-db.yml` is in the merge set:

| Variable | Service | Required | Default | Purpose |
| --- | --- | --- | --- | --- |
| `DATABASE_URL` | web, worker | **yes** | — | DSN of your external Postgres. Carries the credential. Compose *fails* rather than defaulting it. |
| `POSTGRES_SSLROOTCERT_SRC` | — | **yes** | — | Path **on the host** to your CA root. It is a bind source, not container env. Compose fails if it is unset, and the mount fails if it points at nothing. |
| `POSTGRES_SSLMODE` | web, worker | no | `verify-full` | Passed through as `PGSSLMODE`. Lower it only knowingly. |

### An external Postgres, with a verified TLS connection

The default stack runs its own `postgres` on the compose network with no published
port, and that hop is plaintext. To point `web` and `worker` at a database you run
elsewhere, add the override file:

```sh
docker compose -f docker-compose.yml -f docker-compose.external-db.yml up -d
```

It drops the bundled `postgres`, takes the DSN from `DATABASE_URL`, and mounts your CA
root read-only into both services. Four things to get right:

1. **Docker Compose v2.24.0 or newer.** The override uses the `!reset` merge tag. Check
   with `docker compose version`, and run `docker compose -f ... -f ... config` once
   before you rely on it.
2. **Set `POSTGRES_PASSWORD` anyway, to a random value nothing reads.** Compose
   interpolates the base file before it merges the override, so the base's required-variable
   guard still fires. This is expected, not a bug — see
   [packaging-and-configuration.md §9.4](../spec/packaging-and-configuration.md).
3. **Do not put `sslmode` in `DATABASE_URL`.** An explicit DSN parameter overrides
   `PGSSLMODE`, so a DSN carrying `?sslmode=disable` silently turns verification off and
   reports no error at all.
4. **The DSN host must be in the certificate's SAN.** `verify-full` checks the server
   certificate against the host you named. An IP in the DSN needs an IP SAN; a name needs
   a DNS SAN.

`POSTGRES_SSLROOTCERT_SRC` points at a **CA root** on the host — the anchor that signed
your database's certificate, not the certificate itself. It mounts read-only at
`/etc/verge/ca/root.crt`. If the path does not exist, `docker compose up` fails with
`bind source path does not exist` rather than creating an empty directory — see
[troubleshooting.md](troubleshooting.md#compose-up-fails-with-bind-source-path-does-not-exist).

### The `local` vantage resolver

One product-side default you may need to change before the first scan is the
recursive resolver the `dns` scan queries. The shipped `local` vantage carries it. It
ships as `127.0.0.11:53` — Docker's embedded DNS — which works on this `docker compose`
deployment by default. **Off compose** — bare-metal or a host-network install, where
`127.0.0.11` is not routed — set it to your own recursive resolver before the first `dns`
trigger. Otherwise the scan resolves nothing and commits a silent `Gap`. The `local`
vantage is resolver-only and has no prober page. Change it on the row directly — see
[using.md → Run the first batch](using.md#4-run-the-first-batch) for the exact command.
A provisioned prober vantage declares its resolver in the console. It changes on its
vantage card. This `psql` edit is only ever about the `local` row.

### Where secrets live

Ruled by [ADR-0053](../adr/0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md).
**The database holds no secret.**

| Secret | Held by | Origin |
| --- | --- | --- |
| Database credential | environment | you supply it (`POSTGRES_PASSWORD`; in external-database mode, inside `DATABASE_URL`) |
| Session signing key | `web-state` volume | generated by `web` on first boot |
| Prober SSH private key | `worker-state` volume | generated by `worker` at provisioning; only the public half leaves |
| Setup token | nowhere | generated, logged once, consumed on use |

`web` never renders a secret value — only **set / not set**, and the prober's
**public** key. A database dump therefore leaks neither the session key nor the SSH
private key: they live on the per-service state volumes, not in Postgres.

Three operator-typed credentials do sit in Postgres, and each is AEAD ciphertext under a
key Postgres never holds
([ADR-0172](../adr/0172-a-bearer-authenticator-seed-is-admitted-to-postgres-as-aead-ciphertext-and-the-sealing-key-stays-on-the-volume.md)):
the two-factor seed under a sub-key of the session signing key, and the webhook signing
secret and the OIDC client secret under sub-keys of the `transcript-key` volume
([#1679](https://github.com/winniel123/verge-asm/issues/1679)). A dump discloses
ciphertext and no key.

---

## Volumes

`docker-compose.yml` declares four named volumes. Back up these three:

| Volume | Holds | Losing it means |
| --- | --- | --- |
| `pgdata` | the entire estate — subjects, observations, spans, all declared data | total data loss |
| `web-state` | session signing key | all sessions invalidated; a new key is regenerated |
| `worker-state` | prober SSH private key | provisioned vantages must re-install the new public key |

This table excludes the fourth, `transcript-key`, on purpose.
[backup-and-restore.md → The transcript key](backup-and-restore.md#the-transcript-key)
states why, and what losing it costs.

For the restore side this section omits — taking and restoring a consistent `pgdata`
dump, what each state volume regenerates when lost, retention tuning, and a
back-up/test-restore checklist — see **[backup-and-restore.md](backup-and-restore.md)**.

---

## Networking and security posture

- `postgres` **publishes no port**. It is reached only over the compose network by
  `web` and `worker`.
- Only `web` publishes a port (`8080`). Put a TLS-terminating reverse proxy in front
  of it for any real deployment. Set `VERGE_SECURE_COOKIES=true` **and
  `VERGE_TRUSTED_PROXIES`**. `web` trusts no proxy the operator has not named. A
  fronted deployment that leaves `VERGE_TRUSTED_PROXIES` empty gives every client the
  same login rate-limit key, the proxy's address. Five failed logins from anyone then
  lock every account on the instance. Name every hop between the client and `web`.
  Ruled by [ADR-0159](../adr/0159-an-unnamed-proxy-is-never-trusted-so-the-client-ip-is-the-immediate-peer-and-a-fronted-deployment-must-name-its-proxies.md).
- Every service runs **non-root** (`65532:65532`), `cap_drop: [ALL]`,
  `no-new-privileges`. The prober inherits the same posture on the host it is pushed
  to. It runs as an ordinary unprivileged SSH user and needs no capability. Probing
  uses TCP connect rather than raw sockets.

The instance is a **high-value target**: its database is a complete, current map of
your attack surface. Treat access to `web` and to `pgdata` accordingly.

---

## Operating

### Health

Both `web` and `worker` ship a `-healthcheck` flag that compose runs on an interval.
`docker compose ps` shows the result. `web`'s check hits `/healthz`. `worker`'s check
verifies it can reach Postgres.

### The worker is single-instance

**Run exactly one `worker`. Do not `--scale worker=N`.** This guide once recommended
`--scale worker=3`. That recommendation was wrong. This guide withdraws it
([#1092](https://github.com/winniel123/verge-asm/issues/1092)).

The prober process holds the active-scan safety budget. That budget is 50 conn/s and
1 in-flight connection per host. It is also 200 conn/s across every target one vantage
probes. The worker
execs a fresh prober for every job, and its drain loop is single-threaded. So exactly one
prober runs at a time, on this host or on a remote vantage.

Those ceilings are enforced **per `Vantage`**. One vantage emits at most the declared
rate at one target. That is not the scope the numbers were chosen for. They are chosen
for what one **target** receives, because a destination's SYN backlog and connection
table do not care how many sources the traffic arrived from. The two scopes differ, and
the gap is disclosed rather than closed
([ADR-0137](../adr/0137-the-safety-budget-promises-a-targets-rate-and-enforces-a-vantages.md)).

**A target inside N declared vantages receives up to N times the declared rate**, and
nothing prevents that. Nothing gates the vantage count either — a cost you declared
deliberately is priced at policy time, not refused by a threshold
([ADR-0127](../adr/0127-the-address-scope-range-cap-has-no-ceiling-a-large-scope-is-priced-not-gated.md)).
Each `Batch` records the per-vantage figure. Multiply it by your vantage count to read
what one target receives. Dividing each vantage's rate by the vantage count would enforce
the promise exactly, and it is refused: it makes declaring a second vantage slow the
first, and comparing what different vantages see is the reason to declare more than one.

**Within one scan**, the per-host ceiling does not depend on the worker count. A `hot`
scan fans out **one job per `(Vantage, Address)` pair**, so two concurrent jobs of one
`Dispatch` never share a target from one position. Where a lagging `hot` tick would
re-enqueue a pair the previous dispatch has not drained, the tick is skipped and recorded
rather than run beside it. That gate does not arm while the stale-job reaper is off
(`VERGE_STALE_JOB_TIMEOUT` at or below zero), because one wedged job would then skip every
later `hot` tick.

**Across scans it does depend on the worker count.** `cold` probes with the same leaf and
the same per-host numbers, and it draws its addresses from the same estate. So an address
inside an opted-in `cold` scope sits in a `hot` job and a `cold` job at once. Those are two
`Scan`s and two `Dispatch`es, and the lag gate reads one scan's jobs. At one worker the
drain loop is serial, so the two never overlap. At N workers they can, and that host then
receives up to twice the per-host rate.

What N workers do break is the **per-vantage aggregate**. Nothing serialises probers per
vantage host. The worker pushes a fresh prober binary over SSH for each job, so N workers
run N concurrent probers on one vantage, and each paces to the full budget. That vantage
then emits up to N times the rate it declared.

Nothing warns you, and nothing refuses the work. Each `Batch` still records the
**declared** profile. A run that emitted twice the rate therefore records the single
rate. That record is wrong, and not merely incomplete.

Workers are byte-identical and carry no per-instance configuration, so a second worker
could not drift to a different aperture from its siblings. That argument covers
configuration, not rate. It does not make scaling safe.

The mechanism that would bound N probers on one vantage is a **grant**, not a Postgres
reservation. This guide once recommended the reservation, on the `ct` throttle's shape
([ADR-0106](../adr/0106-the-ct-poll-is-a-scan-that-schedules-and-a-ct-admission-is-a-name-citing-its-batch.md)).
It withdraws that recommendation. A prober on a remote vantage runs over SSH with no
route to the database, so the reservation cannot exist where the packets are emitted.
ADR-0137 rejects it on capability, not on cost.

The grant inverts it. The worker reaches Postgres, so the worker computes a budget from a
reservation at claim time and carries it to the prober in the `JobSpec` on stdin. That is
one round trip **per job**, never on the connect path, and it expires with the probe
timeout (`VERGE_PROBE_TIMEOUT`), so a departed worker never holds a share.

**This project does not build the grant.** It was deliberately falsifiable, and
[#1115](https://github.com/winniel123/verge-asm/issues/1115) falsified it. One worker
drains 1024 addresses against a refusing estate in 56 minutes. That is 3.9% of the daily
cadence. The same scan against a silently dropping estate needs 86% of the cadence. One
worker fits at the default address-scope cap in both cases. Nobody has to scale, so the
grant guards a case that does not occur.
[#1116](https://github.com/winniel123/verge-asm/issues/1116) closed it unbuilt on that
ground. The measurement is `docs/research/hot-scan-wall-clock-and-emitted-rate.md`, which
PR #1562 lands from branch `docs/1115-hot-scan-wallclock-measurement`. The ruling is the
#1116 amendment to
[ADR-0137](../adr/0137-the-safety-budget-promises-a-targets-rate-and-enforces-a-vantages.md).

**`--scale worker=N` stays forbidden, permanently.** Nothing bounds two probers on one
vantage, and this project plans nothing. #1115 measured what happens without a bound.
Four probers put 200 conn/s on a target whose `Batch` declared 50 conn/s for the whole
scan. Eight probers put 400 SYN/s on a vantage that declared a 200 conn/s ceiling. The
target refused every port in that run, so one SYN was one connect.

Three changes would reopen the grant, and the ADR-0137 amendment lists a fourth.

- **Declare a second vantage.** The job count is addresses times vantages. Two vantages
  at the default cap over a dropping estate need 172% of the cadence.
- **Raise the address cap over an estate that drops.** The measured cliff sits near 1,190
  addresses, just above the default 1024.
- **Raise the in-flight count and watch a real estate still miss its cadence.** Every
  `Batch` now records the count the exchange actually runs, and that count is 1
  ([#1572](https://github.com/winniel123/verge-asm/issues/1572)). **The serial exchange
  stays.** Raising it is a safety-budget change under the ADR-0137 amendment and not a
  performance one, so it waits on a decision about what the pacer then promises at a
  target. The serial exchange carries the whole 86% figure above, and the honest record
  is what makes the cost legible before anyone spends it.

Recorded `hot` skips are the signal for all three. Watch them before you reach for a
second worker.

### The rule also covers the worker-read kinds

Three job kinds never reach a prober. `Worker.process` handles `zone`, `ct` and
`ct-tail` itself and returns (`internal/queue/worker.go`). They emit no probe packets.
So the budget the rule protects does not apply to them. They share the cap because they
share one queue and one single-threaded drain loop.

**Relaxing the rule for them would buy no throughput.**
Every CT request reserves a slot on a per-source row in Postgres.
The reservation happens before the request goes on the wire.
Three paths reserve: `ct`, `ct-tail` and `ct-verify`.
One throttle value reaches all three from `cmd/worker/main.go`.
That reservation holds across processes by construction
([ADR-0106](../adr/0106-the-ct-poll-is-a-scan-that-schedules-and-a-ct-admission-is-a-name-citing-its-batch.md)).
A second reader worker would share the same slot cadence, and fetch no faster.
Only `zone` would gain from a second process, and the `zone` scan runs monthly.

**What a second reader would buy is decoupling, not scale.** A reservation sleeps the
drain loop until its slot arrives. One `ct-tail` poll issues up to 65 requests, and each
one reserves. So a worker-read job can hold the loop for minutes. The worker claims no
measurement job while it sleeps.

That hold is small against the cadences this install ships.
The `hot` and `ct` scans run daily, and the `zone` scan runs monthly.
The `ct-tail` source ships off, so a default install enqueues no tail job.
A hold of minutes against a daily cadence is under one percent of the period.

**This guide keeps one rule for one worker**
([#1108](https://github.com/winniel123/verge-asm/issues/1108)). It refuses a split into a
reader service and a measurement service. It also refuses concurrency inside the drain
loop. Neither pays for itself while the hold stays this small against these cadences.

Two limits on that decision, both real:

- **Nobody measured the hold.** The figure above is arithmetic over the shipped
  constants, not an observed time. Nothing records whether a measurement job has ever
  waited behind a worker-read job. The recorded `hot` skip names a lag, never its cause.
- **A CertSpotter token paces the tail 30 times slower.** The reservation interval then
  reads 360 seconds, and one tail poll can hold the loop for hours. That is a defect
  ([#1520](https://github.com/winniel123/verge-asm/issues/1520)). It is not a reason to
  restructure the worker.

Revisit this decision if a `hot` scan turns out to fill its daily cadence. #1115 owns
that number. A hold of minutes matters when the headroom is minutes.

### On-demand scan triggers

Scans normally fire on their own cadence. To dispatch one immediately — the operator/CI
path that produces observation rows on demand — trigger the worker by scan **kind**. The
triggered worker enqueues that scan, **drains the queue synchronously, then exits**:

```sh
docker compose run --rm worker -trigger dns
```

#### The scan kinds

Six kinds ship. Each is an accepted value for `-trigger`, and each has its own shipped
cadence:

| Kind | What it does | Cadence | Ships |
| --- | --- | --- | --- |
| `dns` | Resolves the name-scope seeds from every configured vantage (no port list). | daily | enabled |
| `hot` | **Active** TCP connect scan of the `verge-core` "hot" port set, per vantage. | daily | enabled |
| `cold` | **Active** TCP connect scan over the **full 1–65535** range, per opted-in scope. | monthly | **disabled** |
| `tls-acceptance` | TLS-handshake enumeration over the open `Service` population (no port list). | weekly | enabled |
| `zone` | Worker-read ingest of uploaded [zone files](zone-files.md) (no vantage, no prober). | monthly | enabled |
| `ct` | Worker-read crt.sh certificate-transparency poll (no vantage, no prober). | daily | enabled |

Three things to know before you trigger one:

- **`hot` and `cold` are active port scans.** They open real TCP connections across the
  target ports. So a `hot` scan can run for **minutes**, and a `cold` scan (all 65,535
  ports) considerably longer. `dns`, `zone` and `ct` are cheap by comparison.
- **`cold` ships disabled**, and a trigger **refuses a disabled scan** — it does not run
  it once as a one-off:

  ```
  worker: trigger cold: queue: cold Scan is disabled — a manual run
  dispatches an enabled Scan, never a one-off (ADR-0044)
  ```

  `cold` enables itself only once you opt a seed scope into it. Then `-trigger cold`
  dispatches normally. The same refusal applies to any kind an admin has disabled.
- **A trigger is an *extra* fan-out, not a reschedule.** It enqueues the scan keyed to
  "now" and does not reset the cadence schedule. The unique `(scan, scheduled_time)` key
  keeps a manual run from colliding with the automatic one.

#### `run` vs `exec`

Use **`docker compose run`**, not `exec`:

```sh
docker compose run --rm worker -trigger dns      # correct
```

`run` starts a **fresh, throwaway worker container** (same image, env and volumes as the
long-running `worker` service) that takes the trigger path, drains the queue, and exits —
`--rm` removes it afterward. It does **not** start the dispatcher loop, retention, or
delivery runners. Those belong to the daemon.

`docker compose exec worker …` runs a command **inside the already-running** worker
container instead of spawning a new one — useful for `-healthcheck`, but the wrong tool
for a trigger. If a long-running `worker` daemon is up, it shares the same Postgres queue.
So it may claim and drain the jobs your trigger enqueued (each job is claimed by exactly
one worker, never both). The trigger still works, but the fresh `run` container is the
clean, self-contained way to do it.

### Logs

```sh
docker compose logs -f web worker
```

The setup token, prober self-test result, and dispatcher/delivery/retention status
all surface here.

### Retention

Two retention sweeps run inside `worker`, both **off until you set the dial** in the
UI (v1 ships the corpus growing without bound):

- **Dispatch retention** — retires expired operational dispatch rows. It never
  touches observation or span data.
- **Observation retention** — retires only *evidential* observations past their own
  per-timeline bound *and* your dial. A derivation always reads live-tier data.

---

## Version & updates

**Settings → Instance** shows what this instance is running and whether a newer
release exists — and it does so as **check, surface and guide, never self-replace**.
A non-root, distroless container cannot and must not rewrite its own image, so the UI
*reports and guides* and the image swap stays a **host** action. This boundary is
[ADR-0124](../adr/0124-a-backup-carries-data-and-no-secret-and-updating-is-guided-not-self-applied.md).
The surface is admin-only.

- **Running version.** The card shows the running build — the same value in the footer
  — so you can see what is actually running. A released image shows the version stamped
  into it; an unstamped build shows `VERGE_VERSION`, or `dev` when that is unset.
- **Migrations-pending badge.** A best-effort count of embedded migrations newer than
  the highest one applied: **schema current**, or **N migrations pending** (a warning).
  It tells you whether a restart will migrate *before* you take one.
- **Daily release check — opt-out and air-gap-safe.** When enabled, the **worker**
  checks the release feed once a day, best-effort: a short timeout, no retry storm,
  and a failure reports nothing rather than alarming. It is **off by default** and
  fully declinable. With it disabled the instance makes **no network call ever**,
  not even on boot, and the card shows the air-gap copy. Toggle it on
  **Settings → Instance** (`POST /settings/updates/check`, admin). The feed URL is
  `VERGE_RELEASE_FEED_URL` (defaults to this repo's GitHub latest-release).
- **Guided host steps.** When a newer release is seen, the card shows the latest
  version and the host commands to run. The list is compiled into the build, and the
  feed never supplies it. There is no "update now" button, and the UI composes no
  shell of its own — it prints exactly these lines for you to run **on the host**:

  ```sh
  # on the host — verge cannot rewrite its own image
  docker compose pull
  docker compose up -d web worker
  docker compose ps web worker
  ```

  The block names no migration command. `web` applies every pending migration as it
  starts, and it refuses to serve if one fails. The last line reads the `HEALTHCHECK`
  both images declare, so it reports whether the new image is healthy.

Verge never re-images itself. A service that could replace the very binary that parses
your attack surface would be the maximal form of the thing this product hardens
against. Pulling the new image and recreating the containers is always your action on
the host.

---

## Upgrades

The [Version & updates](#version--updates) card surfaces *when* to upgrade. This is
*how*. If you deploy the published images, follow the guided host steps above. If you
build from source, pull and rebuild:

```sh
git pull
docker compose up -d --build
```

Either way, `web` applies any new migrations as it starts. The schema change
lands before the new `web`/`worker` code serves traffic. So **take a backup first** for
anything you cannot afford to roll forward through — see
[backup-and-restore.md → the pre-upgrade backup drill](backup-and-restore.md#the-pre-upgrade-backup-drill).

`web` applies migrations at startup and refuses to serve if one fails. If
`docker compose ps web worker` shows `web` restarting or unhealthy after an upgrade,
read `docker compose logs -n 50 web`. Each migration runs in its own transaction, so
the failing migration reverts and the schema stays at the last version that applied.

---

## Rolling back

Read the Release body first. Its **migration banner** says whether that release
carried a migration, and that answer picks your row below.

| Case | Route |
| --- | --- |
| No migration, no clone | Download the **previous** release's `docker-compose.yml` asset. Run `docker compose up -d`. One file swap, and nothing else. |
| No migration, cloned repository | Pin the previous image digest in your `docker-compose.yml`, the way that file already pins `postgres`. Then run `docker compose up -d`. **Never run `docker compose pull`** here, because it re-resolves `latest`. |
| A migration landed, either shape | Run `docker compose down`. Restore the [pre-upgrade dump](backup-and-restore.md#the-pre-upgrade-backup-drill) into a clean volume. Then start on the pinned old digest. **Restore before you start.** An old `web` against the new schema is the fault you are avoiding. |
| Any row above | The **in-app restore** is the wrong tool here. It runs inside the `web` you replace. See [Restoring — preflight, then a typed confirm](backup-and-restore.md#restoring--preflight-then-a-typed-confirm). |

**An image downgrade is not a schema downgrade.** `migrateUp` in `cmd/web/main.go`
calls `goose.Up` and nothing else. Every file in `db/migrations/` carries a
`-- +goose Down` block, and no code path ever runs one. Verge offers no migration
command either. So an older image meets the newer schema, and that schema stays.

A schema that a release only added to is safe under the old code. A schema that a
release removed from is not. **You cannot tell the two apart from the outside.** That
is why the migration row routes you to the dump and not to an image pin.

### The silence window

An instance that already runs the withdrawn release shows `current` and says nothing.
Once the feed serves the older version again, `isNewer(older, running)` is false.
`internal/release/release.go` then writes `current` and clears the latest fields.

Verge accepts this window and adds no signal for it. `isNewer` compares numeric cores
only, so the check has no words for a version someone withdrew. The successor release
closes the window.

**So a withdrawal contains new installs only.** Your running instance learns nothing
about the withdrawal from the feed.

---

## Stopping and resetting

```sh
docker compose down            # stop, keep data
docker compose down -v         # stop AND delete all volumes — destroys the estate
```
