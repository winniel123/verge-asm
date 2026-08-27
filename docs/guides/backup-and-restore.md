---
title: Backup & restore
section: Operating
order: 2
description: Take a data-only backup from the UI or a full pgdata dump on the host and restore either, protect the two state volumes, and tune the retention dials that decide what the estate keeps.
---

# Backup & restore

What to back up, how to restore it, and how long the estate keeps its own data.
This guide expands the restore side that [running.md → Volumes](running.md#volumes)
and [running.md → Retention](running.md#retention) name but do not walk through.

There are **two** ways to take a backup, and they answer different needs:

- **In-app backup** (**Settings → Instance**) — a one-click, data-only download of
  the estate and its configuration, and a guided restore, with **no shell**. This is
  the first-class way to carry the estate to another host. It is documented first,
  below.
- **Host-level `pg_dump`** — a full logical dump of the whole database on the host,
  for disaster recovery into a clean volume and the pre-upgrade drill. Documented
  under [Host-level dumps with `pg_dump`](#host-level-dumps-with-pg_dump).

Three named volumes hold everything worth saving, and they are not equal. One is
the estate itself; losing it is total data loss. The other two hold a single
generated secret each, and losing either is a recoverable inconvenience — the
service mints a new one on next boot. Know which is which before you plan a
backup.

| Volume | Service | Mount | Holds | Cost of losing it |
| --- | --- | --- | --- | --- |
| `pgdata` | `postgres` | `/var/lib/postgresql/data` | the entire estate — subjects, observations, spans, every declared row | total data loss |
| `web-state` | `web` | `/app/state` | session signing key | all sessions invalidated; `web` regenerates a new key on boot |
| `worker-state` | `worker` | `/app/state` | prober SSH private key | provisioned vantages must re-install the new public key |

The service names and volumes above are the real ones from
[`docker-compose.yml`](../../docker-compose.yml); the commands below use them
verbatim.

---

## In-app backup & restore

**Settings → Instance** carries a **Backup** card and a **Restore** card — a
data-only backup you download in one click, and a guided restore, with no host
shell. Both are admin-only. The design decision behind them is
[ADR-0124](../adr/0124-a-backup-carries-data-and-no-secret-and-updating-is-guided-not-self-applied.md);
the export lives in [`cmd/web/backup.go`](../../cmd/web/backup.go).

### What the backup is — data-only, no session-minting key

The download is a **logical dump of the business tables** — the estate and its
configuration — written in Go straight over the pool `web` already holds (the
distroless image ships no `pg_dump`, so this is not a shell-out). It is **not** a
physical snapshot: it carries the subjects, spans, drift timeline, signals, scans,
channels, deliveries, retention and instance settings, accounts, SSO providers and
API tokens — everything `web` renders and `worker` writes — and nothing else.

What it deliberately **omits** is the machinery that would turn a leaked backup into
a live foothold:

- **The session signing key and the prober SSH private key are not in it.** Under
  [ADR-0053](../adr/0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md)
  those two keys live on the `web-state` and `worker-state` volumes, never in
  Postgres, so a dump that reads the database *cannot* contain them. On restore they
  **regenerate** — see below.
- **The live `session` table is excluded**, along with six other tables that are
  purely transient or short-lived (`password_reset`, `recovery_code`, `invite`,
  `heartbeat`, the CT-log throttle bucket, and the in-flight scan queue). Their rows
  would be meaningless — or actively harmful phantoms — after a restore, so the
  archive leaves them out. The exclusion is an **export invariant**, not an accident:
  the dump reads only an explicit table allowlist, so a future table is never swept
  in by a "dump everything" default.

One honest caveat: the archive is *data-only and carries no session-minting key*, but
it is **not** "zero secrets." The durable per-row credentials the database already
holds — password hashes, TOTP secrets, API-token hashes, SSO client secrets, channel
webhook secrets — ride with their rows, because a restore must reconstitute login,
tokens, SSO and delivery. The archive therefore carries the **same leak posture the
live database already has** under ADR-0053, no more and no less: **treat a backup
file with the same care you treat access to `pgdata`.**

### Taking a backup

On **Settings → Instance → Backup**, click **Download**. The server streams a
newline-delimited-JSON (`.ndjson`) archive named `verge-backup-<timestamp>.ndjson`
as a browser attachment:

- The **first line is a manifest** — the archive format and version, the **schema
  version** it was taken at (the highest applied migration), the timestamp, and the
  ordered table list. Restore reads this to check compatibility before it touches
  anything.
- Each following line is one table marker or one row (serialised by Postgres, so
  every column type round-trips faithfully). The stream is table-by-table over the
  pool, so a large estate backs up in constant memory.

On a clean download the card records the **last backup** time and size, which it then
shows. Store the file off the host — a backup that lives only on the machine it
protects is not a backup.

### Restoring — preflight, then a typed confirm

Restore is deliberately **guarded**, because it **overwrites** the estate. On
**Settings → Instance → Restore**:

1. **Upload the archive for a preflight.** The server validates it **without applying
   anything**: it parses the manifest, checks the schema is compatible with the
   running build, and counts the subjects it would restore. It shows you the file,
   when it was taken, the subject count, and the schema — or, if the archive is
   unparseable or its schema does not match, an error, with **nothing touched**.
   - A preflight is **refused while a scan is in flight** — stop the dispatch first,
     so a restore never races an in-progress write.
2. **Type `restore` to confirm.** The confirm dialog makes you type the exact word
   `restore` before the apply button unlocks — and the **server checks the word too**,
   never trusting the browser gate. This guards the most destructive act the instance
   offers.
3. **Apply.** The archive **overwrites** the estate and configuration, and then two
   keys **regenerate**:
   - The **session signing key** rotates, so **every existing session lapses** —
     every signed-in operator (including you) is signed out and signs in again, with a
     notice. This is loud and intended: a restored instance never carries a prior
     instance's session key.
   - The **prober SSH keypair** regenerates, so each provisioned vantage must
     re-install the new public key and **re-pin** (as with a lost `worker-state`
     volume — see [The two state volumes](#the-two-state-volumes)). Until it re-pins,
     that vantage reads `unavailable` and its exposure findings open a `Gap`.

   Token **Last used** never regresses across a restore — it rides in the backup data.

Because the manifest carries the schema version, a restore **across a migration bump**
is caught at preflight: an archive taken on an incompatible schema is refused before
it overwrites anything, rather than half-loaded.

> The in-app restore is the counterpart to the in-app backup — an `.ndjson` archive,
> data-only, guided. To restore a full host-level `pg_dump` instead, or to rebuild
> into a clean volume, use the `psql` / `pg_restore` path below.

---

## Host-level dumps with `pg_dump`

For a full logical dump of the **whole** database on the host — including the tables
the in-app archive excludes — or to rebuild into a clean volume, use `pg_dump` and
`psql`/`pg_restore` directly against the running `postgres` container. This is the
path for disaster recovery and the [pre-upgrade drill](#the-pre-upgrade-backup-drill).

### Backing up `pgdata`

The database holds no secret ([running.md → Where secrets live](running.md#where-secrets-live)),
but it holds everything else. Take a logical dump with `pg_dump` inside the
running `postgres` container — it is transactionally consistent without stopping
the stack, so `web` and `worker` keep serving while it runs.

```sh
docker compose exec -T postgres \
  pg_dump -U verge -d verge --clean --if-exists \
  > verge-$(date +%F).sql
```

- `-U verge -d verge` are the shipped defaults (`POSTGRES_USER` / `POSTGRES_DB`).
  If you overrode either in `.env`, substitute your values.
- `--clean --if-exists` makes the dump self-restoring: it drops each object before
  recreating it, so a restore does not fight leftover rows.
- `-T` disables TTY allocation so the redirect captures clean SQL.

For a smaller, faster-restoring artifact use the custom format instead, written to
a bind-mounted path so the file lands on the host:

```sh
docker compose exec -T postgres \
  pg_dump -U verge -d verge -Fc > verge-$(date +%F).dump
```

Store the dump off the host. A backup that lives only on the machine it protects
is not a backup.

---

### Restoring `pgdata`

#### Into the running stack

A plain-SQL dump taken with `--clean --if-exists` restores straight through
`psql`:

```sh
docker compose exec -T postgres \
  psql -U verge -d verge < verge-2026-08-23.sql
```

For a custom-format (`-Fc`) dump, pipe it through `pg_restore` instead:

```sh
docker compose exec -T postgres \
  pg_restore -U verge -d verge --clean --if-exists < verge-2026-08-23.dump
```

`web` applies goose migrations on boot, so restoring a dump from an **older**
schema and then starting the current image lets `web` migrate it forward. Restore
first, then `docker compose up -d`.

#### Into a clean volume

To rebuild from scratch — corruption, a moved host, `docker compose down -v` — let
compose recreate an empty `pgdata`, then load the dump before anything writes to
it:

```sh
docker compose down -v          # discards the old pgdata (and both state volumes)
docker compose up -d postgres   # fresh, empty database, health-gated
docker compose exec -T postgres psql -U verge -d verge < verge-2026-08-23.sql
docker compose up -d            # bring web and worker up onto the restored data
```

`docker compose down -v` deletes **all three** volumes, so this path also discards
`web-state` and `worker-state`. That is usually fine — see below — but if you
backed them up, restore them before the first `web`/`worker` start.

---

## The two state volumes

Neither state volume is in the database and neither is worth a scheduled backup on
its own — each holds one secret the owning service regenerates. Back them up only
to avoid the disruption that regeneration causes.

**`web-state` — session signing key.** Generated by `web` on first boot. Lose the
volume and `web` mints a new key: every existing session cookie stops verifying,
so **every signed-in operator is logged out** and signs in again. No data is lost
and no reconfiguration is needed.

**`worker-state` — prober SSH private key.** Generated by `worker` at prober
provisioning; only the **public** half ever leaves the instance
([prober.md](prober.md)). Lose the volume and `worker` generates a new keypair —
but every prober host still trusts the **old** public key in its
`authorized_keys`, so pushes fail until you re-provision each vantage and install
the new public key on it. This is the one state loss with real operational cost,
proportional to how many probers you run.

If you want to spare yourself either, snapshot the volumes while the stack is down.
Find their exact names first — Docker prefixes them with the compose project
(the directory name unless you set `COMPOSE_PROJECT_NAME`):

```sh
docker volume ls | grep -E 'web-state|worker-state'
```

Then tar each through a throwaway container (substitute the names you saw):

```sh
docker run --rm -v verge-asm_worker-state:/state -v "$PWD:/backup" \
  alpine tar czf /backup/worker-state.tgz -C /state .
```

Restore is the same command with `tar xzf` into an empty volume before the owning
service starts.

---

## The pre-upgrade backup drill

[running.md → Upgrades](running.md#upgrades) flags this, and it is the one time a
`pgdata` backup is non-negotiable: `web` applies new migrations **before** the new
code serves traffic, and a schema change is not always cleanly reversible. Take
the dump first, then upgrade:

```sh
docker compose exec -T postgres pg_dump -U verge -d verge -Fc \
  > verge-pre-upgrade-$(date +%F).dump
git pull
docker compose up -d --build
```

If the upgrade misbehaves, you can roll the database back by restoring that dump
into a clean volume (see above) and pinning the previous image. The state volumes
need no pre-upgrade snapshot — a session re-login and, at worst, prober
re-provisioning are recoverable without one.

---

## Retention — how long the estate keeps its own data

Backups protect against loss you did not choose. **Retention** is loss you *do*
choose: two sweeps inside `worker` that retire aged rows on a dial you set at
**Settings** (the delivery tab; the form posts to `POST /settings/retention`, and
the whole Settings page is admin-only). Both dials ship at **0 — unbounded** — v1
grows the corpus without limit until you turn a dial up. This matters to backups
because it decides how much there is to back up, and because it is the only
supported way to delete estate data.

The two dials are independent and floored differently:

- **Dispatch retention** (`dispatch_cadence_multiple`) — retires expired
  operational **dispatch** rows and nothing else; the sweep's data layer exposes
  no observation, span or batch method, so it structurally cannot touch measured
  data. Stated as a **multiple of the slowest enabled scan's cadence**, not a day
  count. `0` is unbounded; any positive value below **2 cadences** is rejected,
  because below that the coverage layer cannot answer whether the slowest scan
  ran.
- **Observation (evidential) retention** (`observation_currency_days`) — retires
  only **evidential** observations, in whole **days**. An observation is *live*
  while it is within the bound of the tightest scan covering its timeline —
  derivations read the live tier and it is **never** discarded — and *evidential*
  once past that bound, read by no derivation. Only evidential rows are eligible,
  and only past your dial. `0` is unbounded; a positive value below the tightest
  bound in force is rejected as a no-op (the whole corpus already outlives it).

A live observation is never retired no matter how low the dial goes: the delete
query evaluates each row's own per-timeline bound. The dial only governs how long
**evidential** rows — the record of what was once measured — are kept beyond that.

---

## Checklist — what to back up, how often, how to test a restore

| What | How | How often |
| --- | --- | --- |
| The estate + config (data-only) | [In-app backup](#in-app-backup--restore) (**Settings → Instance → Download**), stored off-host | routinely — the no-shell way to carry the estate to another host |
| `pgdata` (the whole database) | `pg_dump` as above, stored off-host | on a schedule matching your tolerance for lost days, and **always** before an upgrade |
| `web-state` | volume tar, optional | rarely — regeneration only forces a re-login |
| `worker-state` | volume tar, optional | before any host move if you run several probers, to avoid re-provisioning them |

**Test the restore, not just the backup.** A dump you have never restored is a
guess. Periodically:

1. Restore the latest dump into a **throwaway** stack — a separate directory, so
   `COMPOSE_PROJECT_NAME` differs and it gets its own volumes.
2. `docker compose up -d` and confirm `web` comes up healthy and the migrations
   apply cleanly (`docker compose logs web`).
3. Sign in and spot-check that subjects, observations and spans are present.
4. Tear it down with `docker compose down -v`.

If step 2 or 3 fails, the backup was not one. Find out on a rehearsal, not during
an incident.

---

## See also

- [running.md → Volumes](running.md#volumes),
  [→ Version & updates](running.md#version--updates) and
  [→ Retention](running.md#retention) — the operational digest this guide expands.
- [prober.md](prober.md) — how the prober SSH keypair is generated and where the
  private half stays.
- [ADR-0124](../adr/0124-a-backup-carries-data-and-no-secret-and-updating-is-guided-not-self-applied.md)
  — why the in-app backup is data-only, why the keys regenerate on restore, and why
  updating is guided, never self-applied.
- [troubleshooting.md](troubleshooting.md) — when a restore or a boot does not go
  to plan.
