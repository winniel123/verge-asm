# ADR-0001: Stack and runtime for web, worker, and persistence

- **Status:** Accepted
- **Date:** 2026-08-01
- **Ticket:** [#6 Stack and runtime for web, worker, and persistence](https://github.com/winniel123/verge-asm/issues/6)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

verge-asm is an AGPL-3.0, self-hosted, single-tenant web application, deployed via
`docker compose`, whose differentiation is **exposure drift as a first-class,
queryable object with its own lifecycle** across ports, certificates, DNS records
and HTTP identity.

Four prior decisions constrain this one:

- **[#5](https://github.com/winniel123/verge-asm/issues/5)** put all vantage-dependent
  measurement — DNS resolution, TCP connect, TLS, HTTP GET — into a single **Go binary**
  with a job spec on stdin and NDJSON on stdout, exec'd locally for internal measurement
  and pushed over SSH for external. It explicitly left this ticket open: the worker need
  only spawn a subprocess and read NDJSON. #5 also requires that **every batch records its
  scope** (ports attempted, sources queried, enumerated not named) so that widening the
  aperture cannot manufacture drift.
- **[#11](https://github.com/winniel123/verge-asm/issues/11)** decided built-in local
  accounts, admin/viewer roles, a single-use bootstrap token, optional TOTP, and a
  permission check on every mutating endpoint from the first commit.
- **[#14](https://github.com/winniel123/verge-asm/issues/14)** made **exposure derived,
  never observed** — probers emit vantage-stamped reachability observations and the
  per-(asset, port) state is computed over them — and established that the optional
  external prober is reached by the instance **pushing a binary over SSH**.
- The map's standing rule: **a source that errors must produce no observation, never an
  observation of absence.** In a drift product, a manufactured absence is a false
  "asset removed" alert.

The map further notes that the instance is a **high-value target** — its database is a
complete, current map of the operator's attack surface — and that `docker compose up`
must produce a working instance with no external services to provision.

## Decision

| Concern | Decision |
| --- | --- |
| Persistence | **PostgreSQL**, core only — no extensions beyond `contrib` |
| Queue | **Postgres-backed**: `SELECT … FOR UPDATE SKIP LOCKED` + `LISTEN/NOTIFY` |
| Backend | **Go** across web, worker and measurement binary |
| Frontend | **Server-rendered** `html/template` + **htmx**, SSE for the live drift feed |
| Topology | **One image, two compose services** (`web`, `worker`), plus `postgres` |
| API | **No JSON API in v1**; session-authed CSV/JSON export from the UI |

### Non-binding implementation defaults

Recorded for continuity, not decided at ADR altitude — a later session may change these
without reopening this ADR:

- `pgx` as the driver and **`sqlc`** for queries, **not an ORM**. The drift engine is
  window-function heavy; an ORM abstracts precisely the layer that carries the
  product's differentiation. sqlc gives compile-checked raw SQL.
- `goose` for migrations.
- Stdlib `net/http.ServeMux` for routing — since Go 1.22 it handles method and wildcard
  patterns natively, which suits the map's small-dependency-footprint constraint.
  Reach for `chi` only if middleware ergonomics prove painful.

## Rationale

### Postgres over SQLite

"No external services to provision" reads as an argument for SQLite, but a Postgres
container in the compose file the operator already runs is not a provisioning step — it
is one container and one volume.

The arithmetic decides it. A small org with ~500 live assets against the ~140-port hot
set from [#4](https://github.com/winniel123/verge-asm/issues/4) produces ~70k reachability
observations per day — ~25M rows/year — before certificates, DNS records and HTTP
identity. Per #14 exposure is *derived*, so the read pattern is temporal window queries
over the largest table in the system, running concurrently with a worker streaming NDJSON
inserts. SQLite offers exactly one writer, and the drift queries want `DISTINCT ON`,
partial indexes and native interval arithmetic.

"SQLite now, Postgres later" was rejected explicitly: it taxes every query in the drift
engine to the intersection of both dialects, for the lifetime of v1 — and the drift engine
is the differentiation.

### Postgres-backed queue over a broker

This is not generic broker-aversion. #5 requires every batch to record its scope, and the
map forbids observations of absence. Together they mean **job outcome and observation data
must commit together**. If a job records "completed, 140 ports attempted" while the
observation write rolled back, the system has manufactured 140 false removals across the
estate — the exact failure class the map keeps ruling against.

A Redis broker places job state in a different durability domain from the results,
yielding at-least-once delivery that must be reconciled by hand in the one code path where
a mistake looks like an attack-surface change to the operator. `SKIP LOCKED` collapses that
to a single transaction: either the scan happened and is recorded, or it did not.

The accepted cost is that retries, backoff, dead-lettering and job visibility are built
rather than inherited. That work now belongs to
[#9](https://github.com/winniel123/verge-asm/issues/9).

### Go across the whole system

#5 already commits the project to Go for the measurement binary, and that fact carries
further than it first appears. The instance↔binary contract is a job spec in, NDJSON out.
With both sides in Go, the contract is **one shared struct definition** (`internal/wire`)
rather than two parallel definitions maintained by hand across a language boundary. A
schema mismatch there does not raise — it silently misparses a field and surfaces as
**false exposure drift**.

Three supporting arguments:

- **Packaging.** One `GOARCH` matrix builds web, worker and prober for `linux/amd64` and
  `linux/arm64`, rather than an interpreted image plus a cross-compiled sidecar.
- **Attack surface.** A static binary on distroless has no interpreter, no package
  manager and no transitive C extensions. Weighed against the map's note that the instance
  is a high-value target, this is a stated-priority argument rather than aesthetics.
- **#11's auth is hand-rolled regardless.** Local accounts, roles, TOTP and a single-use
  bootstrap token yield no batteries-included win from Django or FastAPI, because the
  batteries were already declined.

Costs accepted: no Alembic-grade migration ergonomics, no auto-generated OpenAPI, and
plainer templating than Jinja. `dnspython` is genuinely stronger than `miekg/dns` for
AXFR and zone-file work, but not by enough to split the toolchain.

Elixir/Phoenix was the closest rival — Oban would have supplied the queue decision outright
and LiveView the frontend one — but it splits the toolchain from the Go binary, and the
wire contract returns to being hand-maintained.

### Server-rendered over SPA

An SPA ships third-party npm JavaScript into the exact page that renders the operator's
complete attack-surface inventory; a supply-chain or XSS problem there is close to
worst-case for this product. It also reintroduces the second toolchain the Go decision
removed, along with a lockfile to audit.

#11 leans the same way: cookie sessions with a per-handler permission check are markedly
simpler than SPA token custody, and TOTP enrolment plus the single-use bootstrap token are
natural form-post flows.

The accepted cost is real: the core screen is a filterable, sortable inventory with a live
change feed, and heavy client-side filter state is where htmx starts to hurt.

Scope note: **what** the screens are remains [#10](https://github.com/winniel123/verge-asm/issues/10)'s
question. This ADR decides only how they are rendered.

### Two services, one image

The security argument is specific to #14. The optional external prober is pushed over SSH,
so some component holds an SSH key granting access to a host outside the network — and
that component is the worker.

The web tier is what an attacker reaches first: it is the only listener, it parses session
cookies and TOTP codes, and it renders operator-supplied seed data. Were web and worker one
process, a foothold in an HTTP handler would also be a foothold on the SSH key to the
external vantage and on the operator-supplied zone file the map calls "the product's spine."

Splitting them costs one `services:` block from the **same image** — no second build, no
second toolchain — and yields `--scale worker=N` for #9 plus isolation from an OOM during a
large NDJSON ingest.

Secrets split by blast radius:

- `web` — session key, database credentials
- `worker` — database credentials, SSH key (#14), operator-supplied zone file

### No JSON API in v1

With htmx the HTTP surface is HTML fragments and SSE, so no API falls out for free;
shipping one is a deliberate act. Two arguments against doing so in v1:

1. An API token is a bearer credential that **bypasses the TOTP #11 just decided on** — a
   weaker second authenticated surface guarding the full inventory.
2. **The integration need here is push, not pull.** Nobody polls an ASM tool; they want to
   be told when `firewalled → exposed` fires. That is served by the map's separate
   notification-channels question, so deferring the API costs no integration story.

A session-authed CSV/JSON export from the inventory covers "get the data out" without a new
credential class.

## Consequences

- **Unblocks [#9](https://github.com/winniel123/verge-asm/issues/9)**, which inherits
  retry, backoff, dead-lettering and job visibility.
- **API tokens for programmatic access are out of scope for v1** (redirected to drift
  notification channels).
- **Packaging narrows** to a single `GOARCH` matrix, but first-run configuration must now
  cover which container receives which secret — the split that gives the topology decision
  its teeth.
- The `internal/wire` package becomes a load-bearing artifact: it is the single definition
  of the job spec and NDJSON result types, and the reason a language boundary cannot
  manufacture drift.
- Compose shape: `web`, `worker`, `postgres`. Two volumes' worth of operator concern
  (`pgdata`, plus supplied secrets).

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| SQLite | One writer against a streaming ingest; loses the temporal SQL the drift engine needs |
| SQLite now, Postgres later | Taxes the differentiating queries to a dialect intersection for all of v1 |
| Redis + Celery/Asynq/BullMQ | Second durability domain; at-least-once reconciliation in the false-drift code path |
| In-process, no persistent queue | A restart mid-scan leaves no record of the attempt; forecloses #9 |
| Python/FastAPI + Go binary | Duplicates the wire contract across a language boundary, where mismatch means false drift |
| TypeScript full-stack | Wins the frontend contract, loses the binary contract — the seam that carries the drift data |
| Elixir/Phoenix + Oban | Strong fit for queue and UI; splits the toolchain from the Go binary |
| SPA + JSON API | npm in the page rendering the attack surface; reintroduces the second toolchain |
| Unified process, separable by flag | Ships the weak topology as the default, and maintains both paths forever |
| Read-only or full JSON API | Bearer credential bypassing TOTP; integration need is already served by push |
