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
| API | ~~**No JSON API in v1**~~; session-authed CSV/JSON export from the UI — **a read-only, opt-in JSON API is now admitted, [ADR-0123](./0123-a-token-api-is-read-only-opt-in-and-a-bearer-path-separate-from-sessions.md) (2026-08-26)** |

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

The arithmetic decides it. A small org with ~500 live assets against the ~~~140-port~~ **131-port**
hot set from [#4](https://github.com/winniel123/verge-asm/issues/4) produces ~~~70k~~ **~66k**
reachability observations per day — ~~~25M~~ **~24M** rows/year — before certificates, DNS records and
HTTP identity. Per #14 exposure is *derived*, so the read pattern is temporal window queries
over the largest table in the system, running concurrently with a worker streaming NDJSON
inserts. SQLite offers exactly one writer, and the drift queries want `DISTINCT ON`,
partial indexes and native interval arithmetic.

> **`~140` was never `verge-core`'s size.** **[measured]** by
> [#97](https://github.com/winniel123/verge-asm/issues/97): the frequency half is **123, all TCP**, the
> union is **136 pairs**, and **131** are probed on default settings with UDP off
> ([`sensitive-ports.md`](../research/sensitive-ports.md) §29, composed with
> [#95](https://github.com/winniel123/verge-asm/issues/95)'s two admissions). **The original figures are
> left standing and the decision does not move** — this ADR turns on the order of magnitude and on the
> concurrent-writer pattern, not on the port count, and it is
> [ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md) that supersedes the estate sizing anyway:
> a single declared `/22` is 1,024 addresses, twice this whole estate.

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

  > **Amended 2026-08-15 by [#124](https://github.com/winniel123/verge-asm/issues/124) — the two
  > architectures stand and three things this sentence leaves implicit are now rules, because read
  > alone it would have a session build a single-architecture image and add a third architecture on
  > request.** Full statement in
  > [`packaging-and-configuration.md`](../spec/packaging-and-configuration.md) §1.
  >
  > **An architecture is in the matrix exactly where
  > [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s golden corpus is run on it in
  > CI.** ADR-0021's gate is what stops a value moving while the world stands still, and an
  > architecture the corpus never runs on ships five leaves unverified there — invisibly, because
  > two installs on one release hold **equal derivation vectors** and the model therefore licenses
  > the comparison. This is not hypothetical: the Go specification permits fused multiply-add
  > (*"an implementation may combine multiple floating-point operations into a single fused
  > operation … and produce a result that differs"*), Go's `arm64` backend emits it and baseline
  > `amd64` has none — so **`GOAMD64` is pinned at `v1`**, and **a declared parameter expressed as a
  > fraction is evaluated in exact integer arithmetic**, or
  > [#67](https://github.com/winniel123/verge-asm/issues/67)'s cure for `certificate-expiring`'s `N`
  > reintroduces the disease one layer down.
  >
  > **`CGO_ENABLED=0` is a measurement decision.** The pushed prober must be statically linked
  > because the VPS's libc is unknown, and cgo would let Go's `net` package use the **system**
  > resolver — a second answer path, chosen at build time, for a question
  > [ADR-0070](./0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md)
  > has fixed to one declared path.
  >
  > **Each image variant embeds the prober binary for *every* architecture in the matrix**, which is
  > not the same statement as *the image is multi-arch*. [#14](https://github.com/winniel123/verge-asm/issues/14)
  > has the instance push the exact binary it wants, and the prober host's architecture is the
  > operator's rather than ours. **The matrix's ceiling is the prober's, not the instance's.**
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

- ~~`web` — session key, database credentials~~
- ~~`worker` — database credentials, SSH key (#14), operator-supplied zone file~~

> **This split is WITHDRAWN at the site that specifies it, 2026-08-15 by
> [#124](https://github.com/winniel123/verge-asm/issues/124) /
> [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md)
> ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).** The
> topology decision above, its blast-radius rationale and the `web`/`worker` split are **untouched
> and confirmed**; what is withdrawn is the two rows, because **they do not deliver the argument
> that produced them.** Both rows begin with the database credential, so anything routed through the
> database is reachable from `web` whatever the table says — and the **zone file** is routed through
> it by every other requirement in the project (it is evidence, and `web` renders its census), while
> the **SSH key** would be by the obvious implementation of *the operator pastes a key*. A delivery
> table is not a boundary.
>
> **ADR-0053's rule:** a secret is held only where the act it authorises is performed, and the store
> two containers share holds none. **Postgres holds no secret.** The session key is generated by
> `web` into a `web`-only volume; the prober's SSH private key is generated by `worker` into a
> `worker`-only volume and only its public half ever leaves; the database credential is the one
> deliberate exception and is **required rather than defaulted** (`${POSTGRES_PASSWORD:?}`); and the
> zone file is **not a secret** at all. Full table in
> [`packaging-and-configuration.md`](../spec/packaging-and-configuration.md) §3.

### No JSON API in v1

> **WITHDRAWN in part, 2026-08-26 by [#660](https://github.com/winniel123/verge-asm/issues/660) /
> [ADR-0123](./0123-a-token-api-is-read-only-opt-in-and-a-bearer-path-separate-from-sessions.md)
> ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).** A
> **read-only, opt-in, off-by-default** JSON API is now in scope. Argument 1 below is answered, not
> discarded: it refuses a bearer that *bypasses TOTP*, and TOTP guards **mutation** — a read-only
> bearer can perform no mutation and no Declared act, so it bypasses nothing (a session already
> reads/exports the inventory without a second challenge). Argument 2 (push, not pull) is narrowed:
> the push story still holds and is unchanged, but the pull need is now real and served by the
> read-only API alongside it. A **mutating** API stays refused. The two arguments are left standing
> below for their reasoning; read them as scoped to a *write-capable* API.

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
- ~~**API tokens for programmatic access are out of scope for v1** (redirected to drift
  notification channels).~~
  > **WITHDRAWN in part, 2026-08-26 by [#660](https://github.com/winniel123/verge-asm/issues/660) /
  > [ADR-0123](./0123-a-token-api-is-read-only-opt-in-and-a-bearer-path-separate-from-sessions.md)
  > ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
  > Personal tokens for **read-only** programmatic access are now in scope (opt-in, off by default,
  > separate from sessions). Notification channels are not a *redirect* of this need any more — they
  > serve push; the read-only API serves pull. Tokens for **mutating** access remain out of scope.
- **Packaging narrows** to a single `GOARCH` matrix, but first-run configuration must now
  cover which container receives which secret — the split that gives the topology decision
  its teeth.
  > **Discharged 2026-08-15 by [#124](https://github.com/winniel123/verge-asm/issues/124) /
  > [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md).**
  > The answer is that **first-run configuration receives almost none of them**: two of the three are
  > generated by their holder and never supplied, and the third is required rather than defaulted.
- The `internal/wire` package becomes a load-bearing artifact: it is the single definition
  of the job spec and NDJSON result types, and the reason a language boundary cannot
  manufacture drift.
- Compose shape: `web`, `worker`, `postgres`. ~~Two volumes' worth of operator concern
  (`pgdata`, plus supplied secrets).~~ **Three named volumes —
  `pgdata`, `web-state`, `worker-state` — and *supplied secrets* is withdrawn with the split above:
  the two per-service volumes exist because their contents are **generated**, not supplied
  ([ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md)).**

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
| ~~Read-only or full JSON API~~ **Full (write-capable) JSON API** | Bearer credential bypassing TOTP; integration need is already served by push. **The read-only half is now admitted — [ADR-0123](./0123-a-token-api-is-read-only-opt-in-and-a-bearer-path-separate-from-sessions.md) (2026-08-26): a read-only bearer bypasses no factor TOTP guards. Only a *write-capable* API stays rejected here.** |

## Amendment — [#119](https://github.com/winniel123/verge-asm/issues/119): the redirect is discharged, and the secret split gains a fourth entry

This ADR deferred the JSON API into the map's notification-channels question in terms — *"the
integration need here is push, not pull ... so deferring the API costs no integration story"* —
and that sentence was promissory until now.
[ADR-0039](./0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md)
discharges it. **Nothing in this ADR's Decision moves; two things around it do.**

**The push story exists and costs no inbound credential.** A `Channel` is an outbound, one-way,
signed `https` POST — it authenticates us to the receiver, grants no read of the instance, and
opens no second authenticated surface, so the reason this ADR gave for deferring the API is
preserved rather than traded away. ADR-0039 also refuses the cheap alternative deliberately: a
**pull** feed is the one option #6's constraint genuinely kills, because a feed reader holds no
session and does no TOTP.

~~**What is still not served is pull.** An operator who wanted to poll gets this ADR's session-authed
CSV/JSON export and nothing else. That is unchanged and is now the settled position rather than a
deferral.~~

> **WITHDRAWN 2026-08-26 by [#660](https://github.com/winniel123/verge-asm/issues/660) /
> [ADR-0123](./0123-a-token-api-is-read-only-opt-in-and-a-bearer-path-separate-from-sessions.md)
> ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).** Pull
> **is** now served, by a **read-only, opt-in** JSON API (`/api/v1`) — an operator can poll the
> current inventory with a personal bearer token, off by default, on a path separate from sessions.
> The session-authed export still exists; it is no longer *the only* pull surface. Push (ADR-0039)
> is unaffected.

**The secret split gains a fourth entry, and it is the first of its kind.** Delivery is outbound
network work and therefore **worker**-side, so a channel secret is a `worker` secret. It is the
first secret on this instance whose compromise is useful **outside** it — the session key, database
credentials, SSH key and zone file are all levers on this deployment; a channel secret is a lever on
somebody else's receiver. Recorded for
[#124](https://github.com/winniel123/verge-asm/issues/124), which owns the split.
