# Using verge-asm

A tour of the operator workflow, from first login to reading drift. Assumes the stack
is already up — see [running.md](running.md) if it is not.

The product's own vocabulary (`Seed`, `Custody`, `Vantage`, `Exposure`, `Span`,
`Coverage`) is defined in [`CONTEXT.md`](../../CONTEXT.md). This guide uses the terms;
that file is where they are pinned down.

---

## First-run setup

With no accounts in the database, `web` opens a one-time setup window and logs a
single-use token:

```sh
docker compose logs web | grep /setup
# web: no accounts yet — open /setup with this single-use token: <token>
```

1. Open <http://localhost:8080/setup>.
2. Paste the token and create your **admin** account.
3. The window closes as soon as the account exists — the token is spent and `/setup`
   stops accepting it. (Pin the token ahead of time with `VERGE_SETUP_TOKEN` if you
   would rather not read the logs.)

You can later invite more accounts and set roles under **Settings**, and enable TOTP
on your own account. Two roles exist: **admin** (can perform declared acts — seeds,
scans, channels) and a read-only viewer.

---

## The four-step checklist

The home page renders your `Coverage` as a checklist. Each step unlocks a capability;
until you complete them, the system is honest about what it cannot yet conclude rather
than guessing.

### 1. Declare your domain

Under **Seeds**, declare a `Seed` — your assertion of *where your estate ends*:

- a **name scope** — a registrable domain such as `example.com`; or
- an **address scope** — a CIDR you control, e.g. `203.0.113.0/24`.

A `Seed` is a boundary, not a starting gun: declaring one **queues** a scan rather
than firing one. An address scope enumerates — every address inside it becomes a
subject and is walked every cadence, which is what lets *no ports responded* be a
fact rather than a silence. A name scope enumerates nothing on its own; its addresses
are reached only by measured resolution, and only under a **custody extension**.

- Address scopes carry a **range-size cap** (1,024 addresses by default,
  configurable, checked at declaration). IPv6 space is not swept — reach an IPv6
  estate through a name scope with a custody extension.
- Draw the boundary inward with **exclusions** — exact names, subtrees, or address
  scopes you declare are *not yours*. *Not mine* is a different claim from *not
  there*; an excluded name is simply no longer queried. Preview an exclusion before
  committing it.
- Turn on a **custody extension** on a name scope to declare that the addresses its
  names resolve to are inside your boundary (the cloud-resident case, where you hold
  no address registrations). Transitivity stops where the resolution chain leaves the
  declared zone.

Registry lookups can **propose** seeds (**Proposals**), but a proposal you have not
confirmed asserts nothing and is probed by nothing. Confirm or decline each.

### 2. Upload a zone file

Under **Seeds → zone**, upload a zone file for a name scope to enable **removal
detection**. It is *uploaded, not mounted*: supply is a dated act, and the upload
instant is the observation instant, so *you stopped telling us* becomes detectable. A
file whose apex sits outside the scope it was uploaded against is refused, with the
reason.

The zone is covered by its own `zone` scan on the **re-supply interval** you set
(shipped monthly). Let a zone age past two intervals and it ages into a `Gap` — the
system tells you the source went stale instead of trusting old bytes.

### 3. Add an internet vantage (provision a prober)

**Exposure requires an outside observer, unconditionally.** A single all-in-one
install can build a complete, honest *internal* reachability inventory, but it cannot
see its own estate from the internet — probing your own public address from inside is
a hairpinning trap that never traverses the inbound policy. So `Exposure` needs a
**prober**: a second Linux host reached over SSH.

Under **Probers**, supply four non-secret values:

| Value | Note |
| --- | --- |
| host | a name or address the instance can reach on the SSH port |
| port | defaults to 22 |
| username | a **non-root** account |
| — | the instance generates the SSH keypair and renders the **public** half for you to install |

Then:

1. Install the rendered **public** key in that account's `authorized_keys` (harden it
   with `restrict` and, once your egress address is known, `from=`).
2. At provisioning the instance reads and **pins** the host key (a later change is a
   hard failure, never a prompt) and checks `uname -s` / `uname -m` — a host that is
   not Linux on `x86_64`/`aarch64` is refused there, with the reason.
3. On the first connection the prober observes your instance's egress address from
   `SSH_CLIENT`. The instance then renders *your egress is `203.0.113.5`* and offers
   it for declaration.

> **Don't hand-roll the host.** [`deploy/prober/`](../../deploy/prober/) is a
> `docker compose` recipe that stands up exactly this — a minimal, hardened, non-root
> SSH target — so steps 1–3 become "paste the rendered public key and `docker compose
> up`." The copy-paste walkthrough, including the host-key pin and the `restrict` /
> `from=` hardening, is in **[prober.md](prober.md)**.

Until an internet vantage exists, exposure claims are withheld — the system degrades
to internal-only and **never** reports `firewalled` or `exposed` for something it did
not look at.

### 4. Run the first batch

The `dns` scan resolves through the shipped **`local` vantage**, whose `resolver`
column names the recursive resolver to query. On the `docker compose` deployment this
ships as `127.0.0.11:53` — Docker's embedded DNS, reachable from the worker container —
so **on compose you do not need to change it**.

Off compose (bare-metal or a host-network install, where `127.0.0.11` is not routed)
point it at your own recursive resolver *before* the first scan. The `local` vantage is
resolver-only and has no prober page, so set its resolver directly on the row:

```sh
docker compose exec postgres \
  psql -U verge -d verge \
  -c "UPDATE vantage SET resolver = '203.0.113.53:53' WHERE name = 'local';"
```

Substitute your resolver's `host:port`. `-U verge -d verge` are the shipped defaults;
if you set `POSTGRES_USER` / `POSTGRES_DB` in `.env`, pass those values instead. A
resolver that nothing answers yields empty records and a `Gap` rather than real data —
and the scan still commits as `completed`, so a wrong resolver fails silently. This is
the one setting you may need to change before the first scan.

Scans dispatch on their own cadence, but you can kick the first one immediately from
the worker (see [running.md](running.md#running-a-scan-on-demand)):

```sh
docker compose run --rm worker -trigger dns
```

Once a batch commits, subjects, observations and spans appear across the pages below.

---

## Reading what it found

| Page | Shows |
| --- | --- |
| **Subjects** | The `Name`s, `Address`es, `Service`s and `Endpoint`s in your estate, each drilling into its facet timelines. |
| **Exposure** | What is reachable from the internet — constructible only once you have both an internal and an internet vantage. |
| **Signals** | The release-coupled rule firings (the v1 rule set), e.g. a sensitive port reached from the internet, a certificate expiring. |
| **Coverage** | What the system measured, what it did **not**, and why — `Gap`s, unread apertures, unevaluable rules. This is where *we cannot construct this claim* lives, and it is a feature, not an error. |
| **Messages** | The change surface: what moved since last time. Mark read individually or all at once. |
| **verge-core** | The default port aperture; admins can edit the *frequency* tier (the sensitive tier is not editable — a port you can hide is a signal you can silence). |
| **Sources** | Enable/disable discovery sources; a toggle is a dated, audit-trailed act. |

### Annotations

An **annotation** moves a message, never a number. Use it to acknowledge, mute or
reclassify what a signal is telling you without altering the underlying measurement.
It is the one declared act with no other dated residue, so it carries its own instant.
Withdraw it to restore the default message.

---

## Notification channels

By default **nothing is routed anywhere** — no channel ships configured, so the
delivery loop is a no-op until an admin declares one.

Under **Settings → Channels**, create a channel (an HTTP endpoint the worker `POST`s
to). Deliveries ride the same retry/back-off/dead-letter curve as the measurement
queue. A channel *carries the message, never the estate* — the body is a
notification, and a `Delivery` is an operational record of the attempt.

To include a working link back to the instance in each notification body, set
`VERGE_PUBLIC_URL` on the `worker` service to your public base URL (see
[running.md](running.md#environment-variables)). Left empty, the link is simply
omitted rather than fabricated.

---

## A mental model to keep

- **Declared** is your input and never drifts. **Observed** is what was measured.
  **Derived** is what was concluded — and two derived values are comparable *only*
  within one identical derivation, which the system enforces with a `Break` rather
  than trusting to discipline.
- When the answer is *we don't know yet*, the product says so in `Coverage` instead
  of inventing certainty. Reading Coverage is as much the job as reading Exposure.
