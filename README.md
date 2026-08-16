# verge-asm

**Self-hosted attack surface management for your own estate.** Its subject is not
inventory but **change**: what of yours is exposed to the internet, and what moved
since last time.

verge-asm takes the seed domains and IP ranges you declare, discovers the
internet-exposed assets that grow from them, actively probes what it is allowed to
probe, and makes **exposure drift a first-class, queryable object** — a timeline
tracked across ports, certificates, DNS records and HTTP identity, not subdomains
alone. It is a single-tenant, self-hosted web application you run with
`docker compose`. AGPL-3.0.

> **New to the domain?** Read [`docs/spec/v1-spec.md`](docs/spec/v1-spec.md) for the
> shape of the whole system, then [`CONTEXT.md`](CONTEXT.md) for the full domain
> model. This README is the operator's entry point; those two are the source of truth
> for *why* it behaves as it does.

---

## What it does

- **Declare a boundary, not a starting point.** You give it `Seed`s — a name scope (a
  registrable domain) or an address scope (a CIDR). Nothing is probed until a `Seed`
  gives it custody of the listener.
- **Discovers outward.** Records followed from your seeds (a CNAME to a CDN, a
  shared-hosting IP) surface assets you did not type in — and ownership of what is
  *found* is distinguished from ownership of what was *declared*.
- **Probes safely.** TCP connect (never SYN), an identifiable User-Agent, one
  `GET /` per endpoint, rate-limited and unprivileged. Active probing is gated by
  `Custody`, which is derived from your seeds alone.
- **Models change as a durable object.** Every `(subject, facet, discriminator,
  vantage, source)` gets a `Span` timeline with its own transition grammar
  (`appeared` / `returned` / `revealed`) and a strict rule about when two
  observations may legally be compared at all.
- **Tells you when it *cannot* tell you.** Where coverage is incomplete or a
  conclusion is unconstructible, it says so (`Coverage`, `Gap`) rather than
  fabricating a clean bill of health.

For the full feature surface — the six facets, the seventeen signal rules, exposure,
vantages and notification channels — see [`docs/spec/`](docs/spec/).

---

## Architecture at a glance

Three services, one image, built from one Go module.

| Service | Role | Listener |
| --- | --- | --- |
| `web` | The only listener. Serves the operator UI, applies DB migrations on startup, mints the first-run setup token. | `:8080` |
| `worker` | No listener. Dispatches each `Scan` on its cadence, claims jobs, commits each `Batch` of observations, runs delivery and retention. | none |
| `postgres` | The estate. A complete, current map of your attack surface. Publishes no port. | none (internal) |

A fourth binary, **`prober`**, is the measurement tool: it reads one job spec on
stdin and writes NDJSON observations to stdout. It is exec'd locally for internal
measurement and **pushed over SSH** to external hosts to give you an internet
`Vantage` — the outside observer that makes `Exposure` constructible.

```
cmd/            web, worker, prober entry points
internal/       the domain, one package per concept (scan, signal, exposure, drift, …)
db/migrations/  goose SQL migrations (web applies them on boot)
db/queries/     sqlc query definitions  →  internal/db (generated)
docs/           spec, ADRs (docs/adr/), research notes, agent guides
design-system/  the canonical visual layer
```

---

## Quick start

You need [Docker](https://docs.docker.com/get-docker/) with Compose. Nothing else —
Go, sqlc and the toolchain live inside the images.

```sh
cp .env.example .env
# Edit .env and set POSTGRES_PASSWORD — compose fails rather than defaulting it.

docker compose up -d --build
```

`web` applies the database migrations itself on startup, so the stack is ready once
all three services report healthy:

```sh
docker compose ps
```

With no accounts yet, `web` prints a **single-use setup token** to its logs:

```sh
docker compose logs web | grep /setup
```

Open <http://localhost:8080/setup>, paste the token, and create your admin account.
From there the four-step checklist walks you through declaring your first `Seed`.

Full walkthrough: **[docs/guides/using.md](docs/guides/using.md)**.

---

## Guides

| Guide | Covers |
| --- | --- |
| **[Running](docs/guides/running.md)** | Configuration, secrets, volumes, healthchecks, scaling workers, manual scans, upgrades. |
| **[Using](docs/guides/using.md)** | First-run setup, the four-step checklist, provisioning a prober, reading coverage/exposure/drift, notification channels. |
| **[Prober](docs/guides/prober.md)** | A worked `docker compose` example for a dedicated internet vantage on a second host — the [`deploy/prober/`](deploy/prober/) recipe, host-key pin, and key hardening. |
| **[Verifying](docs/guides/verifying.md)** | Building, testing, `sqlc` codegen, the golden corpus, and reproducing CI locally through Docker. |

---

## License

AGPL-3.0. See [`LICENSE`](LICENSE). Because it is AGPL, running a modified copy as a
network service obliges you to offer that modified source to its users.
