# verge-asm

> ⚠️ **Alpha software.** verge-asm is in active early development. Expect breaking
> changes, incomplete features, and rough edges. Interfaces, the database schema, and
> behaviour may change without migration paths between releases. Do not depend on it
> for production security monitoring yet. Review it yourself before you run it
> against anything you care about.

**Self-hosted attack surface management for your own estate.** Its subject is not
inventory but **change**: what of yours is exposed to the internet, and what moved
since last time.

You declare seed domains and IP ranges. verge-asm discovers the internet-exposed
assets that grow from them. It actively probes what it is allowed to probe. It makes
**exposure drift a first-class, queryable object** — a timeline across ports,
certificates, DNS records, and HTTP identity, not subdomains alone. It is a
single-tenant, self-hosted web application you run with
`docker compose`. AGPL-3.0.

> **New to the domain?** Read [`docs/spec/v1-spec.md`](docs/spec/v1-spec.md) for the
> shape of the whole system. Then read [`CONTEXT.md`](CONTEXT.md) for the full domain
> model. This README is the operator's entry point. Those two are the source of truth
> for *why* it behaves as it does.

---

## What it does

- **Declare a boundary, not a starting point.** You give it `Seed`s — a name scope (a
  registrable domain) or an address scope (a CIDR). Nothing is probed until a `Seed`
  gives it custody of the listener.
- **Discovers outward.** Records followed from your seeds (a CNAME to a CDN, a
  shared-hosting IP) surface assets you did not enter. It distinguishes ownership of
  what is *found* from ownership of what was *declared*.
- **Probes safely.** TCP connect (never SYN), an identifiable User-Agent, one
  `GET /` per endpoint, rate-limited and unprivileged. Active probing is gated by
  `Custody`, which is derived from your seeds alone.
- **Models change as a durable object.** Every `(subject, facet, discriminator,
  vantage, source)` gets a `Span` timeline. The `Span` has its own transition grammar
  (`appeared` / `returned` / `revealed`). A strict rule governs when two observations
  may legally be compared at all.
- **Tells you when it *cannot* tell you.** Where coverage is incomplete or a
  conclusion is unconstructible, it says so (`Coverage`, `Gap`) rather than
  fabricating a clean bill of health.

For the full feature surface — the six facets, the seventeen signal rules, exposure,
vantages and notification channels — see [`docs/spec/`](docs/spec/).

> **Implementation status (alpha).** The seventeen rules describe the v1 spec target.
> **15 of them fire on a default install today** — the four Name-only rules,
> `tls-1.0-accepted` (P0.9), the **six certificate rules** (P0.10), and the four
> HTTP-identity endpoint rules (P0.11). The remaining **two internet-gated flagship
> rules** need a provisioned internet vantage. A fully-provisioned install then fires
> **17 of 17**.
> See [docs/guides/signals.md](docs/guides/signals.md#rule-status--what-fires-on-a-default-install).
> Expect the same partial-coverage caveat elsewhere while the tool is in alpha.

---

## Architecture at a glance

Three services, one image, built from one Go module.

| Service | Role | Listener |
| --- | --- | --- |
| `web` | The only listener. Serves the operator UI, applies DB migrations on startup, mints the first-run setup token. | `:8080` |
| `worker` | No listener. Dispatches each `Scan` on its cadence, claims jobs, commits each `Batch` of observations, runs delivery and retention. | none |
| `postgres` | The estate. A complete, current map of your attack surface. Publishes no port. | none (internal) |

A fourth binary, **`prober`**, is the measurement tool. It reads one job spec on
stdin and writes NDJSON observations to stdout. It is exec'd locally for internal
measurement. It is **pushed over SSH** to external hosts to give you an internet
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

1. Copy the example environment file to `.env`. Then set `POSTGRES_PASSWORD` in
   `.env`. Compose fails rather than defaulting it.

   ```sh
   cp .env.example .env
   ```
2. Build and start the stack.

   ```sh
   docker compose up -d --build
   ```
3. Confirm all three services report healthy. `web` applies the database migrations
   itself on startup.

   ```sh
   docker compose ps
   ```
4. Read the **single-use setup token** from the `web` logs. With no accounts yet,
   `web` prints it there.

   ```sh
   docker compose logs web | grep /setup
   ```
5. Open <http://localhost:8080/setup>. Paste the token, and create your admin account.

From there, a four-step checklist covers declaring your first `Seed`. Full
walkthrough: **[docs/guides/using.md](docs/guides/using.md)**.

---

## Guides

Grouped as they appear in the docs-site nav.

### Getting started

| Guide | Covers |
| --- | --- |
| **[Using](docs/guides/using.md)** | First-run setup, the four-step checklist, provisioning a prober, reading coverage/exposure/drift, notification channels. |
| **[First run](docs/guides/first-run.md)** | The mental model behind the checklist: declared/observed/derived, why `Exposure` needs two legs, confirming a scan ran, and the CDN-fronted-domain caveat. |
| **[Zone files](docs/guides/zone-files.md)** | What a zone file is, why removal detection needs one, and how to export one from common DNS providers or your own name servers. |
| **[Reading your attack surface](docs/guides/reading-the-estate.md)** | Where to look for each answer — coverage, exposure, drift, inventory, graph, search — and how to read a single scan run. |

### Operating

| Guide | Covers |
| --- | --- |
| **[Running](docs/guides/running.md)** | Configuration, secrets, volumes, healthchecks, scaling workers, on-demand scan triggers, upgrades. |
| **[Sources](docs/guides/sources.md)** | Discovery sources and proposers: consent tiers, admin-only toggling, and the crt.sh and RIR-proposer caveats. |
| **[Prober](docs/guides/prober.md)** | A worked `docker compose` example for a dedicated internet vantage on a second host — the [`deploy/prober/`](deploy/prober/) recipe, host-key pin, and key hardening. |
| **[Signals](docs/guides/signals.md)** | The v1 signal reference — every rule, its subject, and when it fires. The release-coupled philosophy and the five-level severity ramp (Critical / High / Medium / Low / Info). |
| **[Notification channels](docs/guides/notification-channels.md)** | The outbound HTTPS endpoints the worker POSTs each message to — declaring them, what fires, and why a `Delivery` is a record and never a message. |
| **[Reports](docs/guides/reports.md)** | Scheduled digests of the estate — how a report differs from a notification, the schedule shape, and the delivered-report artifact and its PDF. |
| **[Accounts, invites & roles](docs/guides/accounts.md)** | The admin/viewer model, inviting operators and the invite lifecycle, changing a role, re-enrolling a lost second factor, and removing an account. |
| **[Authentication](docs/guides/authentication.md)** | Your own account security — TOTP enrollment and the login challenge, API tokens, active sessions, and the password change and reset flows. |
| **[Single sign-on (SSO)](docs/guides/sso.md)** | Configuring OpenID Connect single sign-on — declaring a provider, registering the callback URL, and linking an identity to a local account. |
| **[Integrations](docs/guides/integrations.md)** | Third-party install tiles whose install is a Declared act — the tile states, install and disconnect, and why an integration is never a delivery channel. |
| **[Backup & restore](docs/guides/backup-and-restore.md)** | Taking and restoring a consistent `pgdata` dump, protecting the two state volumes, and tuning the retention dials. |
| **[Troubleshooting](docs/guides/troubleshooting.md)** | Confirming silent failures — a scan that ran vs. one that failed quietly, empty exposure, undelivered notifications, boot migrations, and the `Gap` cases. |

### Contributing

| Guide | Covers |
| --- | --- |
| **[Verifying](docs/guides/verifying.md)** | Building, testing, `sqlc` codegen, the golden corpus, and reproducing CI locally through Docker. |

---

## Built with AI

In the interest of full transparency: verge-asm was built with substantial help from
AI. Anthropic's Claude wrote or assisted with the large majority of the code,
documentation, and design in this repository. That work ran from the domain model and
migrations to this README. Every change still passes human review, testing, and the CI
gates described in [`docs/guides/verifying.md`](docs/guides/verifying.md). You should
still know that AI was a primary author of this tool. Weigh that as you see fit before
you run it against your own estate.

---

## License

AGPL-3.0. See [`LICENSE`](LICENSE). Because it is AGPL, running a modified copy as a
network service obliges you to offer that modified source to its users.
