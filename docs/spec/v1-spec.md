# verge-asm v1 spec

- **Status:** Accepted — the terminal artefact of [Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Ticket:** [#12 Assemble the v1 spec](https://github.com/winniel123/verge-asm/issues/12)

This document is the map's destination: a spec complete enough to hand to an implementation
session without that session needing to re-litigate any decision the map already made. It does
not restate reasoning — a decision lives in exactly one place, its ADR or ticket — it assembles
and narrates. Where a figure is quoted (a port count, a rule count, a cell count), it is a **dated
record** as of this document's own composition (2026-08-15); the live absolutes are the map's
Notes section and the source documents this spec points into.

**How to read this**: start here for the shape of the whole system, then zoom into
[`CONTEXT.md`](../../CONTEXT.md) for the domain model's full text, the ADRs under `docs/adr/` for
the reasoning behind any one rule, and the spec files under `docs/spec/` (`measurement-offers.md`,
`packaging-and-configuration.md`, `golden-corpus.md`, `curated-table-watch.md`,
`notification-channels.md`) and the research notes under `docs/research/` for enumerations that
are lists rather than decisions and will be revised without a new ADR. The `prototypes/` directory
holds dated readings of individual screens; this document's UI section assembles them into one IA.

---

## 1. Destination & scope

**verge-asm** is an AGPL-3.0, self-hosted, single-tenant web application. An operator supplies
seed domains and IP ranges; the product discovers internet-exposed assets from those seeds via
passive sources and then active probing, and makes **exposure drift a first-class, queryable
object with its own lifecycle** — tracked across ports, certificates, DNS records and HTTP
identity, not subdomains alone.

That last clause is the entire differentiation and it is deliberately narrow.
[Prior art](https://github.com/winniel123/verge-asm/issues/2) established that discovery, active
probing, scan-to-scan diffing and cert/protocol risk signals all exist in open source today —
reNgine ships subdomain drift with scheduling and alerting, and its unreleased branch models
ports, IPs, DNS and full certificate data. What does not exist anywhere surveyed is **change
modelled as a durable object across every asset facet** — a `Span` timeline per `(subject, facet,
discriminator, vantage, source)`, comparable only within one derivation, with its own transition
grammar (`appeared` / `returned` / `revealed`) and its own notion of when two observations may
legally be compared at all. The spec has to earn that claim or the project has no reason to exist.

**Posture.** The operator owns the assets scanned, so no scope-consent machinery is needed for the
operator's own estate — but discovery follows records outward by construction (a CNAME to a CDN,
a shared-hosting IP), and ownership of what's *found* is not the same as ownership of what was
*declared*. [ADR-0002](../adr/0002-ownership-gates-probing.md) and
[ADR-0013](../adr/0013-custody-is-control-and-extends-by-declaration.md) derive **`Custody`** —
control of the listener, never registry title — from `Seed`s alone, and it gates every active
probe (§3). **Single-tenant, self-hosted** via `docker compose`; no multi-tenancy, billing, or
hosted infrastructure. **The user** is a small-org security owner asking *what of ours is exposed
to the internet, and what changed?* **The instance is a high-value target** — its database is a
complete, current map of the operator's attack surface — and every architecture and secrets
decision in §4 is weighed against that.

The design system at [`design-system/`](../../design-system/) is canonical for the visual layer;
its IA and vocabulary are not — it predates [core domain model](https://github.com/winniel123/verge-asm/issues/7)
and ships a `Findings` section this spec's UI (§6) replaces. Where the kit and `CONTEXT.md`
collide, `CONTEXT.md` wins. Invoke the `verge-asm-design` skill before writing any markup.

---

## 2. Domain model

The domain model is [`CONTEXT.md`](../../CONTEXT.md) in full — it is the canonical glossary and is
not restated here. Its organising move: four layers, and the division is load-bearing.

| Layer | What it holds | Drifts? |
| --- | --- | --- |
| **Declared** | What the operator tells us (`Seed`, `Source` consent, `Scan` config, `Annotation`, `Channel`) | No — it is input |
| **Observed** | What we measured (`Subject`, `Observation`, `Facet`, `Batch`, `Citation`) | What drift is *made of* — but observations are never compared directly |
| **Derived** | What we concluded (`Custody`, `Reach`, `Exposure`, `Signal`, `Span`, `Transition`, `Break`, `Gap`) | Only across an identical `Derivation` |
| **Operational** | What the system *did* (`Dispatch`, `Message`, `Delivery`) | Never read by the comparison path |

A term's layer determines *on what condition* it may appear in a change report: comparing two
Derived values produced by *different* derivations reports a change in the observer as a change in
the world, so the rule is never *never diff Derived* but **never compare across differing
derivations** — enforced by a `Break` rather than by discipline
([ADR-0007](../adr/0007-drift-is-a-timeline-of-spans.md),
[ADR-0008](../adr/0008-derivation-versions-move-on-content.md)).

Four subjects, each with its own key and (save `Address`) its own lifecycle: `Name` (FQDN, key is
the label sequence), `Address` (IP, key is family + octets, no lifecycle of its own — in the
estate exactly while a current resolution cites it or a `Seed` covers it), `Service`
(`(Address, port, transport)`, exists open or closed for every pair in the recorded scope), and
`Endpoint` (`(Name, Service)`, the only key under which HTTP identity and the presented
certificate chain are single-valued). Six facets — `resolution`, `dns-record`, `reachability`,
`certificate`, `http-identity`, `tls-acceptance` — each a closed value space, a decoder, a
canonicaliser, a differ, a discriminator and a batch-scope obligation
([ADR-0011](../adr/0011-a-facet-is-six-parts.md)).

Terms **deliberately refused**: `Asset` (flattens `Name`/`Address`, which have different keys and
lifecycles), `Host` (ambiguous between the two), `Finding` (invites a stored, diffable object —
use `Signal`, which is derived and never diffed), `Discovery`/`Probe` as nouns (both are `Source`s
differing only in `authority` and `completeness`), `ScanRun` (comparison must never read a grouping
of batches — use `Dispatch`, which is Operational precisely so the comparison path cannot reach
it).

---

## 3. Discovery & probing pipeline

### 3.1 Sources

A `Source` is anything whose word can put a subject in the estate, carrying **authority**
(`declared` / `measured` / `inferred`), **completeness** (`enumerable` / `corroborative`) and
**consent** (`unencumbered` / `operator-accepted` / `operator-credentialed`). Observing a facet is
the usual way a source admits a subject and not the only one — certificate transparency observes
no facet at all and admits `Name`s purely on `authority: inferred`
([ADR-0027](../adr/0027-a-source-may-admit-without-observing.md)). A thing that admits **nothing**
— a registry lookup returning candidate ranges — is not a `Source` at all; it yields a `Proposal`,
governed by `consent` alone and read by nothing until the operator confirms it into a `Seed`
([ADR-0012](../adr/0012-a-proposer-is-not-a-source.md)).

**Which sources ship on.** A source ships enabled by default only if the modal operator — a small
commercial organisation inventorying its own estate — is inside its terms on two limbs: the
software's inherent behaviour (automated querying, storage, retention) must be permitted, and the
operator's identity/purpose must be inside the terms (`operator-accepted` fails on "personal or
non-commercial use only"; it does not fail on a prohibition on *reselling the source's data*)
([ADR-0003](../adr/0003-third-party-source-consent-bar.md)). Applying that bar: **HackerTarget**
and unauthenticated **Cert Spotter** are excluded on terms; **CIRCL**, **Rapid7 Sonar**, **ICANN
CZDS**, **Wayback CDX** and **bgp.tools** are excluded on availability, needing no policy at all;
**crt.sh** ships on, throttled to 5 req/min, with a non-200 producing no observation rather than an
observation of absence (the map's standing rule); **RIPEstat**, the **RIPE Database**, **APNIC**'s
registry path and **LACNIC**'s registry path ship off under `operator-accepted` (asked, no reply —
[ADR-0003](../adr/0003-third-party-source-consent-bar.md)'s amendments); **AFRINIC** and **APNIC**
clear a keyless org→prefix path via CAIDA ⋈ delegated-stats and ship on; **ARIN**'s `entities?fn=`
org-name path ships on keyless. A **BGP** leg (route-collector-derived address scopes) is out of
v1 entirely — a routing announcement names the path, not the estate
([ADR-0063](../adr/0063-a-routing-announcement-names-the-path-not-the-estate.md)) — even where the
source (RouteViews) clears the consent bar cleanly; clearing the bar is not an argument for
shipping. The **operator's own zone file** is Tier-0 ground truth, uploaded (never mounted — a
supply act needs an instant) through `web`, and CZDS output is reclassified into this path rather
than treated as a queried source.

### 3.2 Seeds & aperture

A `Seed` is the operator's assertion of where the estate ends: a **name scope** (registrable
domain) or an **address scope** (CIDR). An address scope **enumerates** — every address inside it
is a subject from the declaration, walked every cadence whether or not anything has ever answered,
capped at 1,024 addresses by default (operator-configurable, family-agnostic — the cap counts
addresses, so `/22` and `/118` are the same knob) — while a name scope enumerates nothing; its
addresses arrive only by measured resolution. A name scope may additionally carry a **custody
extension**: a declaration that the addresses its names resolve to are under the operator's
`Custody`, off by default, transitivity stopping where the resolution chain leaves the declared
zone or where it resolves to a **non-globally-reachable** address
([ADR-0079](../adr/0079-authority-presupposes-denotation-a-non-globally-reachable-address-is-probed-only-inside-a-declared-realm.md)).

`Custody` (`operator` / `third-party`) is derived from `Seed`s alone — never from registry
expansion — and **gates every active probe totally**: a `third-party` address is connected to on
no port, by no tier, at any rate; there is no narrower-probe carve-out
([ADR-0019](../adr/0019-the-probing-gate-is-total-over-an-address.md)). A non-globally-reachable
address (RFC 1918-and-siblings, read off the IANA special-purpose registries' `Globally Reachable`
column) is a machine **per network realm**, so it is connected to only where a declared **address
scope** covers it and only from a non-`internet`-class `Vantage` — a `custody extension` alone does
not open this gate, because it never declares which realm the prober is in
([ADR-0079](../adr/0079-authority-presupposes-denotation-a-non-globally-reachable-address-is-probed-only-inside-a-declared-realm.md)).
An install holding custody of nothing still measures `resolution` and `dns-record` at full
aperture — **a query is not a connect** — and gets a DNS-only product until it makes a custody
claim; that is the honest shape of the modal cloud-resident operator on day one, not a degraded
state.

**Seven aperture inputs**, each a dimension of what a `Batch` records as its completed scope:
enabled sources, port sets, vantages, the TLS candidate set, the qtype set, the control-probe
population (the parents of resolved names, for wildcard discrimination), and the queried address
scope. A cadence is **not** an aperture input — a `Batch` records what it asked about, never how
often.

### 3.3 The measurement binary

One statically-linked Go binary (`CGO_ENABLED=0` — see §4.2) performs every vantage-dependent
measurement: DNS resolution, TCP connect, TLS handshake, HTTP GET. It decides values through five
named **`Derivation` leaves** — `connect-outcome`, `tls-handshake`, `http-exchange`,
`resolution-walk`, `wildcard-discrimination` — each versioned separately and gated bidirectionally
in CI by a golden corpus (§4.4). `resolution-walk` and `wildcard-discrimination` are the two leaves
membership itself composes: `Shadowed` (wildcard discrimination's suppression) cites no `Address`,
so it decides membership as affirmatively as `resolution-walk`'s own outcomes
([ADR-0086](../adr/0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md)).

**Safety profile**, from [`safe-active-probing.md`](../research/safe-active-probing.md):

| Knob | Default |
| --- | --- |
| Technique | TCP connect (never SYN) — non-root, `cap_drop: [ALL]`, no added capabilities |
| Host discovery | Skipped (`-Pn`) — targets are seeded, not swept for liveness |
| Port-scan rate | ≤ 50 conn/s per host, ≤ 20 concurrent, 3 s connect timeout, 2 retries |
| HTTP | `GET /` only, 64 KB capped body read, 10 s timeout, ≤ 10 req/s per host |
| Redirects | Not followed by default; a declared parameter of `http-exchange`, not an operator dial |
| Admin-panel / credential probing | Response-matching only; default-credential login attempts never, not even opt-in |
| TLS | Certificate fetched every run (rides the `reachability` exchange); version/cipher enumeration weekly, own `Scan` |
| Global ceiling | 200 pkt/s across all targets; round-robin by host, never by port |
| Adaptive back-off | Halves the rate on timeout/RST-spike/429/503; never touches the deadline (ADR-0021 keeps it outside `connect-outcome`) |

TCP connect is chosen over SYN specifically to avoid `CAP_NET_RAW`/root in the container — SYN
scanning gates on effective UID, not just the capability bit, and the trade-off (more packets,
target logs the connection) is inverted in verge-asm's favour: logging *is* the auditability
property, and completed-and-closed connections clear firewall state tables faster than dangling
half-open SYNs. Every offer the binary makes — the TLS candidate set, the queried qtype set, the
ALPN list, the EDNS options, the DNS transport policy — is enumerated in the job spec and recorded
on the `Batch` **by content**, never taken from a library default: a default is not a declaration,
and a narrow offer manufactures a false negative the operator can never see
([ADR-0025](../adr/0025-an-offer-is-scope-only-where-the-value-enumerates-it.md); full enumeration
in [`measurement-offers.md`](./measurement-offers.md)). None of these offers is
operator-configurable — an offer the operator can narrow is a finding the operator can silence.

### 3.4 Scan tiers & cadence

Five `Scan`s, two of them port tiers:

| `Scan` | Scope | Cadence | Notes |
| --- | --- | --- | --- |
| **hot** | `verge-core` (§3.5) | daily | the only tier that ships enabled |
| **cold** | full 1–65535 | monthly, opt-in **per `Seed` scope** | ships configured and disabled with an empty scope list; never runs unasked, including at onboarding — a one-off has no cadence, so it has no currency bound ([ADR-0044](../adr/0044-a-one-off-measurement-has-no-currency.md)) |
| **`tls-acceptance`** | every open `Service`, the TLS candidate set | weekly | no port list — an enumeration, not a port tier ([ADR-0028](../adr/0028-a-facets-cadence-is-the-cadence-of-its-exchange.md)) |
| **`zone`** | name scopes holding a supplied zone file | the operator's declared re-supply interval, shipped monthly | no port list, no vantage choice, worker-read; batches restate the file's observations **at the operator's supply instant**, never at our read |
| **`dns`** | name scopes, unconditionally | daily, independent of `Custody` | no port list; every configured `Vantage`; covers `resolution` and our own resolver's `dns-record` — the two facets that had no covering `Scan` and therefore no currency bound until [ADR-0084](../adr/0084-a-scan-is-a-cadence-over-an-exchange-and-an-uncovered-facet-has-no-currency-bound.md) |

`certificate` rides whichever port tier makes the TCP connect and is not a `Scan` of its own — its
handshake is a step inside the `reachability` exchange, not a separate measurement
([ADR-0028](../adr/0028-a-facets-cadence-is-the-cadence-of-its-exchange.md)). A manual run
dispatches an existing `Scan`; there is no ad-hoc one-off port set. UDP is off by default and is
not a knob that becomes "on" at a bigger budget — `connect-outcome` cannot produce an honest UDP
value at all; an honest instrument needs a sixth leaf, `datagram-outcome`, deciding
`answered │ refused │ unanswered` against a per-pair elicitation payload
([ADR-0083](../adr/0083-silence-decides-only-on-a-connection-oriented-transport.md)); the payload
table is researched ([`udp-elicitation-payloads.md`](../research/udp-elicitation-payloads.md)) but
the leaf is deferred past v1 (§7).

### 3.5 `verge-core` and the sensitive-port curated table

`verge-core` — the daily hot-tier port set — is a **union**:
`frequency-set ∪ sensitive-list`
([ADR-0009](../adr/0009-verge-core-is-a-union.md)). The frequency half is the project's own
selection, informed by (never derived wholesale from) `nmap-services`' 2008-vintage open-frequency
data, plus a modern-services supplement chosen because each port maps to a named v1 risk signal —
123 TCP ports as of this writing. The sensitive half is [`sensitive-ports.md`](../research/sensitive-ports.md)'s
curated list of ports that are **never legitimately internet-facing** — 38 `(port, transport)`
pairs in three claim classes as of this writing, admitted only on a three-limb attestation
standard: a **named claim** (from a closed set — never internet-facing / never correct outside a
narrow window / requires authentication), a source's own **attestation** (the owner of the
protocol or product, retrieved over the artefact rather than over the row — a bare IANA registry
description does not clear this gate), and a **determinacy** gate (the port must stand as a
reliable surrogate for the service — a squat is contested only where the competing convention is
live). Composed, `verge-core` is **136 pairs (131 TCP, 5 UDP)**; only the TCP pairs are probed on
default settings, since UDP is off — so `Coverage`'s aperture statement reads a small, non-zero
count of sensitive pairs unread on a default install, never zero, and the corresponding
`sensitive-port-reached-from-internet` rule's evaluability count is untouched by that gap (the
rule reads a leg on a `Service`; the UDP pairs simply never produce one). `verge-core` is shipped
as an editable list file, and the frequency half alone is operator-editable — the sensitive half is
not, per `CONTEXT.md`'s `Derivation` entry: a declared parameter is authored by the project and ships
in the release, and none is ever operator-configurable, because moving one would move a version and
`Break` the estate without a release and without a golden-corpus row moving.

**Governance.** A curated table is revised by **the release**, never by a standing operator or
curator duty. Two instruments watch it: a **gate** of closed, terminating checks (currently
thirteen, `G1`–`G13`) run to completion over the table as edited every release, and a **queue**
over what is structurally open and therefore only ever sampled — keyed on the *revision act*
(the smallest act by the artefact's owner that would falsify the cell) rather than on the footing
tier, because tier grades evidential distance and empirically disagrees with volatility. A release
discloses the queue's **bounded residue** — never a raw count, which the project measured being
read as a false indicator five times running
([`curated-table-watch.md`](./curated-table-watch.md), [ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md),
[ADR-0078](../adr/0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md)).

### 3.6 What v1 does not probe

Full detail and the argument for each is in §7 (Explicitly out of scope). In brief: no UDP beyond
the deferred elicitation-payload leaf; no sweeping of IPv6 space (a `/64` is ~4×10¹¹ years at the
shipped rate ceiling — IPv6 addresses enter by resolution like any other subject and are probed
like one, they are simply never swept); no wire-protocol listener negotiation (a whole new facet,
canonicaliser and safety surface for zero net new firings past what HTTP-layer signals already
catch); no wildcard-content discrimination beyond the pass/fail `Shadowed` value (DNSSEC's proof is
the only sound second discriminator and it is unavailable on 14 of 15 measured zones); no
technology fingerprinting (a fingerprint verdict is a function of the signature database, so
diffing it reports a change in the observer as a change in the world — the generalisation is
*only observed values enter drift*).

---

## 4. Architecture

### 4.1 Stack & runtime

Go throughout — web, worker and the measurement binary share one wire contract
(`internal/wire`), so a schema mismatch cannot silently misparse a field into false exposure
drift. **PostgreSQL**, core only, no extensions beyond `contrib`; **no ORM** (`pgx` + `sqlc`,
compile-checked raw SQL — the drift engine is window-function heavy, and an ORM abstracts exactly
the layer that carries the differentiation). **Server-rendered** `html/template` + **htmx**, with
**SSE** for the live drift feed — an SPA would ship third-party npm into the exact page rendering
the operator's complete attack-surface inventory, which is close to worst-case for this product's
threat model. **No JSON API in v1**: an API token is a bearer credential that bypasses the TOTP
auth flow (§4.3), and the integration need is push (notifications, §5.3) rather than pull; a
session-authed CSV/JSON export from the UI covers "get the data out."

**Queue: Postgres-backed**, `SELECT … FOR UPDATE SKIP LOCKED` + `LISTEN/NOTIFY`, not a broker.
Job outcome and observation data must commit **together** — a job recorded "completed, N ports
attempted" over a rolled-back observation write manufactures N false removals across the estate,
which is exactly the false-drift failure the whole project exists to prevent. One queue job is one
`Batch`; batch partitioning follows whatever dimension the source retains `completeness` over (one
address per batch for the prober, a whole zone for the operator's zone file — splitting an
enumerable source below its scope destroys removal detection). A dead-lettered batch records an
**empty** scope, never its attempted one — asserting the attempted scope would manufacture
absences it never measured. Retry is always a **new** batch, never a resumption. Progress groups
into a `Dispatch` (Operational, structurally barred from the comparison path — it carries no
observations and the drift engine never reads it), fired under a Postgres advisory lock, idempotent
on `(scan, scheduled_time)`, fanned out atomically. Overlapping ticks are **skipped and recorded**,
never queued or run concurrently; missed ticks are **not** caught up — you cannot measure the past.
A `Vantage`'s `Availability` is Derived from a fixed, release-coupled window of recent batch
outcomes, never an operator dial (widening it would silently make the whole board non-comparable).

### 4.2 Deployment topology

**One image, two compose services** (`web`, the only listener, and `worker`, no listener) **plus
upstream `postgres`** (no published port) — `postgres` runs the standard upstream image, never the
project's build. `linux/{amd64,arm64}` only, `CGO_ENABLED=0` always,
`GOAMD64` pinned at `v1` (Go's floating-point contraction is architecture-dependent — an unpinned
level makes a declared fraction like `certificate-expiring`'s horizon evaluate differently on two
architectures of the same release). An architecture is in the matrix exactly where the golden
corpus (§4.4) runs on it in CI — an architecture the corpus never touches ships derivation output
that is unverified there while the model licenses comparing it to output from a verified one. Each
image variant embeds the pushed prober binary for **every** matrix architecture, because the
prober's architecture is the operator's own VPS, chosen independently of the instance's.

Every service runs `user:` non-root, `cap_drop: [ALL]`, no `cap_add`, no `privileged: true`, no
`network_mode: host`. The pushed prober binary inherits the same posture on its host — an ordinary
unprivileged SSH user, needing no capability at all, a direct consequence of choosing TCP connect
over SYN.

**Secrets: held only where the act they authorise is performed** — the database is the shared
store and holds none (`${POSTGRES_PASSWORD:?}` is the one deliberate, required-not-defaulted
exception) ([ADR-0053](../adr/0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md)).
The session signing key is generated by `web` into a `web`-only volume, never persisted to
Postgres — a read-only database leak (backup, replica, export) would otherwise convert into live
admin sessions with no password and no TOTP. The prober's SSH keypair is generated by `worker`
into a `worker`-only volume; only the **public** half ever leaves the instance, so no private key
is ever an input to the deployment. `web` never renders a secret value back, only **set**/**not
set** and the prober's public key. The operator's zone file is **not a secret** — it is evidence,
uploaded through `web`, stored and read by both services (§3.4). Three named volumes: `pgdata`,
`web-state`, `worker-state`. Full table: [`packaging-and-configuration.md`](./packaging-and-configuration.md) §2–3.

**Declaring vantage intent.** There is no `network_position` field and no setup wizard step — both
were specified and both are withdrawn as relative (`external` to what?). A `Vantage class`
(`internet`/`internal`) is re-verified every batch against the address the vantage is observed to
**present** — the address it dialled, for a prober; `SSH_CLIENT`'s reported address, for the
instance itself, as observed by a prober from outside. **`Exposure` therefore requires a prober,
unconditionally** — with none, the instance's own presented address is unobserved, its class is
`unverified`, and the product degrades to a complete, honest internal-only inventory rather than
fabricating exposure claims. Provisioning a prober (host, port, non-root username; the instance
generates the keypair) declares "this vantage is on the internet"; declaring an address scope
covering the instance's own presented address declares "this one is inside my boundary." The
prober's SSH host key is pinned at provisioning; a change is a hard failure (the vantage goes
`unavailable`), never a silent re-trust prompt.

### 4.3 Auth & access

Local accounts only — no SSO/OIDC/reverse-proxy forward-auth (a misconfigured trusting proxy is a
whole bypass class, and it is refused rather than risked; see §7). Two roles, admin/viewer. The
first admin is bootstrapped by a single-use setup token written to container logs. TOTP is
optional per account. A permission check runs on every mutating endpoint from the first commit.
Every Declared act (a `Seed`, an exclusion, a `Scan` config change, a `Channel`) is an
authenticated admin act, audit-relevant by construction — which is also why configuration lives
in the database and not in a mounted file: *if a change to it should appear in the audit trail, it
may not live in the environment* ([`packaging-and-configuration.md`](./packaging-and-configuration.md) §5.1).
The environment holds only what must exist before the database does: the database URL/credential,
the listen address, and the setup-token escape hatch.

### 4.4 Correctness machinery: the golden corpus

Every `Derivation` leaf's version must move on an **output-affecting change and only on one** —
enforced bidirectionally in CI by a checked-in corpus of `(job-spec fragment, authored peer
script, expected NDJSON)` rows run hermetically (no network, no containers) against every matrix
architecture. The gate fails if a corpus row's expected output moves without the leaf's version
bumping, *and* if the version bumps without any row moving, a declared parameter changing, or an
"uncovered move" being explicitly registered. This is what makes two installs on the same release
holding equal derivation vectors a licence to compare their values — a leaf whose output silently
drifted on a dependency upgrade would otherwise manufacture false drift across every install on
the next release. `resolution-walk` and `wildcard-discrimination` — the two leaves membership
itself composes — carry the densest obligation: 46 cells between them (27 + 19), including paired
boundary pins (a boundary is pinned by two rows, one on each side, since a row cannot detect a
collapse toward itself). Full enumeration: [`golden-corpus.md`](./golden-corpus.md).

### 4.5 Notification transport

A `Channel` is an outbound-only, admin-configured **signed `https` POST** — an absolute URL, an
optional secret (HMAC-SHA256 over body + timestamp when set; the URL alone is the credential when
not), and a subset of three routing classes (`drift` / `coverage` / `clock`, defaulting to all
three; routing is by class alone, never by rule or subject). **No bearer header, ever** — the
signature authenticates *us to the receiver*; nothing authenticates the receiver back, because
there is nothing for it to ask: the channel is one-way, no callback, no inbound surface, no pull
feed (the one option the no-JSON-API constraint genuinely kills, since a feed reader holds no
session and does no TOTP). The body carries exactly what the in-app message carries and **no
rows** — no service list behind a census count, no address set behind a resolution move — computed
once at the cause and never recomputed, read from the same computation that renders the in-app
message. Retries run on the queue's own retry/backoff/dead-letter machinery (§4.1) rather than a
second mechanism: five attempts over roughly an hour, exponential, then dead-lettered. A
dead-lettered delivery **licenses no silence** — the message stays in the store, marked
undelivered, and a failed delivery is never itself a message (it has no cause among the model's
closed four). No channel ships configured by default. Full enumeration:
[`notification-channels.md`](./notification-channels.md).

### 4.6 Data retention

Retention is a property of what may still be **read**, never of age
([ADR-0041](../adr/0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md)). Three
corpora, three different rules:

- **Observations** hold two tiers. **Live** — within `k` cadences of the tightest `Scan` covering
  that timeline — is what every derivation reads and may never be discarded. Past that, an
  observation is **evidential**: a derivation may not read it and may never re-derive history from
  it, so discarding it moves no value on any timeline; it is kept only for a person asking *what did
  we actually measure*. The bound is keyed on the **timeline it bounds** —
  `(subject, facet, discriminator, vantage, source)` — and never collapsed: a row is retained while
  its age is inside **either** its own bound **or** the operator's retention dial, whichever is
  longer. **The dial's floor is the tightest bound in force** — below that it changes no row at all —
  so **the control collapses to one number and the query never does**; it still reads each row's own
  bound
  ([ADR-0094](../adr/0094-a-retention-control-collapses-and-a-retention-query-never-does.md)). Two
  populations sit outside the ordinary rule: a timeline with **no covering `Scan`** has an **undefined**
  bound, so it is never retired (reachable in v1 only where an operator disables a `Scan`); a
  **withdrawn** subject's timelines carry **no floor at all** — the dial alone governs them.
- **`Span`s are never compacted**, on two independent grounds: deleting the span before an open one
  converts `returned` into `appeared`, a clock silently moving a fact about the world, and the corpus
  is proportional to **drift** rather than to **time** — at the shipped `/22` ceiling it is
  ~672,000 rows and flat, against ~98M observation rows a year, so it is the small corpus and not the
  one that needs a dial.
- **`Dispatch`** is the one corpus a wall clock **may** retire — it carries no observations and the
  comparison path is structurally barred from reading it. Its retention window is an **operator
  dial**, floored at `k` cadences of the slowest **enabled** `Scan` (below that, `Coverage` cannot
  answer whether the slowest scan ran), stated as a multiple rather than a day count. **v1 ships it
  unbounded** — one row per firing is nothing to retire yet.
- **`Message` and `Delivery` ship with no retention dial in v1 at all** — the store is defined as
  unable to fail, and a dial is a supported way for it to fail anyway; `Delivery` travels with its
  `Message` rather than being a corpus in its own right
  ([ADR-0081](../adr/0081-a-floor-is-territory-and-an-unbounded-default-is-a-position.md)). See
  [Out of scope](#7-explicitly-out-of-scope) for the reopening condition.

---

## 5. Drift model

### 5.1 The unit of comparison

Change is never a diff between two point-in-time snapshots. It is a **`Span`** — one period during
which a `(subject, facet, discriminator, vantage, source)` timeline held a single value — opened,
current, closed. One timeline per source (two sources that disagree hold two true facts rather
than an arbitration), keyed so a `Batch` covering MX and not TXT never asserts an empty TXT RRset
it never measured. A `Span` carries the vector of `Derivation` versions it was produced under; two
spans compare **only** where their vectors are equal. Comparison across differing vectors is not
merely inadvisable, it is **structurally unavailable** — the boundary between them is a `Break`,
derived on read from the two vectors, never stored, naming the leaf that moved
([ADR-0007](../adr/0007-drift-is-a-timeline-of-spans.md),
[ADR-0008](../adr/0008-derivation-versions-move-on-content.md)).

A **`Transition`** is the adjacency between two consecutive spans, derived on read and never
stored. Three named kinds, and they live in different places: `appeared` (discovery) and
`returned` (a decommission undone) are **membership-only**, because they describe a subject;
`revealed` (a widened aperture — *we* started looking, the world did not move) belongs to **any**
timeline, because aperture is a property of looking and looking is per-timeline. `returned`
composes **every witness a presence read currently relies on** — one per `Vantage class`,
existential within a class, agreed across classes — so a `Break` on any relied-upon witness voids
`returned` for the whole subject and it re-enters reading `appeared`, the correct carrier and an
honest word for what happened after a version bump destroyed the history
([ADR-0097](../adr/0097-returned-composes-every-witness-a-presence-read-rests-on.md)). A `Gap` is a
span holding **no value** — the period over which the system could not say, opened by a
dead-lettered batch, an unavailable vantage, evidence aged past its currency bound, or an answer
the system could not read (a truncated RRset, an undiscriminated wildcard). A gap never withdraws
a subject; ceasing to measure is not measuring absence.

**Subjects leave only by measurement**, never by a clock. A `Name` leaves on a Name Error from
every available `Vantage class` composed correctly (cross-class agreement, never a survivor-only
reading over an empty set). An `Address` is in the estate exactly while a current resolution cites
it or a `Seed` covers it. A `Closure`'s reason is a closed union of three, sorted on what it rests
on: `measured-absent` (independent evidence about *this* subject), `uncited` (evidence about
*another* subject — a child beneath a withdrawn root, or a resolution that stopped citing an
address), `descoped` (our own aperture narrowed) — and only a `descoped` closure blocks the
subsequent `returned` reading, because a narrowing is not a decommission
([ADR-0006](../adr/0006-subjects-leave-by-measurement.md),
[ADR-0087](../adr/0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md)).

### 5.2 Signals

A `Signal` is a named, versioned rule evaluated over observations (and possibly other Derived
values), carrying no severity — urgency belongs to the transition that surfaced it, not to the
rule. Four parts: its `Predicate domain` (the extension of its own name — the population of which
the fact it names *could* be true), its predicate, its `not-evaluable` case, and its version
vector. Its census is always three members over one population — fired / did not fire /
`not-evaluable` — never a delta, trend or series, and every member is enumerable in full: none is
sampled, ranked, grouped or truncated, including a member that happens to be most of the estate. A
rule is excluded by its **fact** (no evidence, or the evidence never determines it) or its
**aperture** (the measurement costs more than it buys) — never by the shape of the resulting set:
being true of nearly the whole estate is not disqualifying on its own.

**v1 ships seventeen rules** (a dated count): **five** reading facts about `Name`s alone
(`lame-delegation`, `cname-target-name-error`, `zone-declared-name-returns-name-error`,
`resolved-name-absent-from-zone`, `non-globally-reachable-address-resolved-from-internet`), unbounded
by any port tier, and **twelve** reading a facet on a `Service` or `Endpoint` and therefore bounded by
which port tiers are enabled — the port tier bounds **which subjects exist**, never **which rules can
speak**. `sensitive-port-reached-from-internet`, the one rule naming a port, is one of those twelve:
it reads `Reach` on a `Service` like the rest, additionally restricted to `verge-core`'s probed
pairs.
([ADR-0024](../adr/0024-a-rules-domain-is-the-extension-of-its-name.md) carries the full table.)
Where a rule reads a curated table asserting about the world (the sensitive-port list, §3.5), that
table — never the rule — is what the attestation standard governs.

### 5.3 Messages and notification

A `Message` is one firing of one **cause**, computed once at the cause and never recomputed
(recomputing would reach back across a `Break`). Four causes, and every message names what
**moved**, read from the fold rather than the rule: the estate's own object (drift), us — our own
aperture or a rule of ours (an aperture widening), the operator's own declared input, or nothing at
all where only a threshold was crossed (clock class). No valence word anywhere in the vocabulary —
nothing is *resolved*, *fixed*, *critical* or *OK*, because a clear is not always good news and a
widening is neither. Class is a property of the **firing**, not the rule: the three clock-reading
rules (certificate-expiry class) fire clock class where the span they read is unchanged and drift
class where it moved. Every message is written and rendered **unconditionally** — the store is not
a `Channel`, cannot be disabled, cannot fail.

**The flagship**: the internet `Reach` leg going `not-reached` → `reached` is the move the product
exists to catch, fired whether or not an internal leg even exists, carrying the census of every
facet that opens beneath the newly-reached `Service`. The internal leg is recorded and **never**
alerted, in either direction — an internal port opening or closing is the commonest intentional
change on that leg. A `Service` or `Endpoint` **entering** the estate is never itself a message
— it rides the census of the membership message fired at the root of its entering sub-tree, once,
because every timeline beneath a new subject opens and no alerting predicate in the product is
opening-shaped.

**Where a message links.** Per mover: an object in the estate (drift) links to that object's own
page; a threshold-crossed message (clock) links to the object whose span the rule read; the
operator's own declared input links to the `Source` the rule reads; an aperture-widening message
(us) links to the `Seed` whose scope moved — **never** to `Coverage`'s standing aperture statement,
which is deliberately constant and would lose which act the message was about
([#159](https://github.com/winniel123/verge-asm/issues/159)).

---

## 6. UI / Information Architecture

### 6.1 Overview

Six nav destinations plus a global, non-nav message element carrying an unread count on every
screen (opening a list whose rows each link to the object or scope the message fired at,
§5.3) — the nav was deliberately held to five destinations and the sixth was spent on `Coverage`
rather than a seventh
([#10](https://github.com/winniel123/verge-asm/issues/10),
[#22](https://github.com/winniel123/verge-asm/issues/22),
[`notification-channels.md`](./notification-channels.md) §1.1):

| Destination | Renders |
| --- | --- |
| **Exposure** (the landing view) | The exposure board — a 2×2 grid over the internet/internal `Reach` legs, never the raw inventory |
| **Subjects** | The estate listing (search, no denominator, `Citation` per row) and each subject's own drill-down page |
| **Signals** | Every v1 rule's fired/not-fired/`not-evaluable` census, and `Annotation` management |
| **Seeds** | Declare/confirm/decline scopes, custody extensions, source enablement's entry point |
| **Coverage** | The aperture statement, per-`Scan` rows, retention, and the day-one checklist |
| **Settings** | Accounts, `Channel`s, retention dials — the operator's dials, gated per §4.3's auth model and §4.6's floor rules |

### 6.2 Exposure (landing view)

The landing view is the **exposure board**, deliberately *not* the estate inventory — that
question was settled explicitly against a filterable-table default
([#10](https://github.com/winniel123/verge-asm/issues/10)). It renders `Exposure`'s 2×2 projection
over the internet and internal `Reach` legs (`exposed` / `edge-only` / `firewalled` /
`unreachable`), a "what moved" panel for the flagship internet-leg transition, and per-vantage-class
availability. Four preconditions, each with its own non-alarming rendering rather than a blank
grid: **no exposure can be constructed** (fewer than two `Vantage class`es hold a current value —
the custody-of-nothing and no-prober cases both land here, honestly, rather than defaulting to a
false `internal-only` reading); **no `Service` in the estate at all**; **rules changed, nothing to
compare yet** (a `Break` on the composing derivation); and the populated board. A precondition panel
and a populated board can co-exist as distinct renders of the same screen — an earlier assumption
that they could not was corrected by the exposure-cells prototype. One-legged installs (no internal
vantage configured, or a connectionless leg that never decides) render the surviving leg's raw
`Reach` under "we never looked," never a fifth `Exposure` value (`internal-only` was withdrawn —
[ADR-0017](../adr/0017-exposure-needs-both-legs.md)).

### 6.3 Coverage

The aperture statement — one line per aperture input (§3.2), stating what the tier is, its cadence,
and its on/off state, never a proportion of the *operator's* estate (that is
[#28](https://github.com/winniel123/verge-asm/issues/28)'s refused estate-completeness score). The
sensitive-pairs line carries two figures that never fuse: pairs **outside the recorded scope**
(an invitation the operator can act on — enabling the cold tier or a UDP leg moves this) and pairs
**inside** the recorded scope that the instrument is structurally incapable of reporting as reached
(a UDP pair probed with no elicitation payload — no available action changes this one). The zero-
coverage state is a **rendering**, not a wizard: a four-step day-one checklist — *Declare your
domain* → *Upload a zone file* → *Add an internet vantage* → *Run the first batch* — each step
naming a capability and, where an act genuinely exists, pointing at the surface that performs it,
never adding a prompt of its own
([#28](https://github.com/winniel123/verge-asm/issues/28),
[#51](https://github.com/winniel123/verge-asm/issues/51)). `Coverage` also carries retention (the
two dials — the operational `Dispatch` floor and the observation-currency floor, §4.6) and is the
entry point for the source-enablement modal (§6.4).

### 6.4 Seeds

Six jobs on one screen, discovered by watching an early variant silently acquire a wrong one
(a shared row component leaked a per-address `Approve` affordance the model explicitly
refuses — [#123](https://github.com/winniel123/verge-asm/issues/123)): declaring a name or address
scope; managing exclusions; declaring/withdrawing a custody extension; confirming or declining
registry `Proposal`s (confirmation is **singular**, one scope at a time; decline may be bulk over
a whole lookup — the two acts fail in opposite directions and are deliberately asymmetric,
[ADR-0022](../adr/0022-confirmation-is-singular.md)); the source-enablement prompt (two marked
groups — *what you may be able to resolve* vs *what nobody has been able to resolve*, rendering
even when a group is empty, [#47](https://github.com/winniel123/verge-asm/issues/47)); and
provisioning a prober (§4.2). A **narrowing receipt** shows the count of what a narrowing act would
withdraw *before* the operator commits it, but only where the withdrawal message would actually
fire — most narrowings are silent because the ground is still cited elsewhere
([#166](https://github.com/winniel123/verge-asm/issues/166),
[#167](https://github.com/winniel123/verge-asm/issues/167)). An address-scope declaration states
its realm claim explicitly wherever the CIDR is non-globally-reachable, moot elsewhere
([#168](https://github.com/winniel123/verge-asm/issues/168)). Custody-extension census is
**display, never per-address approval** — a rendering of what the extension currently covers, with
no denominator (estate completeness is unmeasurable) and no state to approve.

### 6.5 Signals

Every rule's census renders as fired / not-fired / `not-evaluable`, current state only — never a
delta or trend. A member row is never the `Subjects` row component: it carries no `Citation`, no
search, and its header count is exactly `list.length`, locked — the base/special-case split runs
the other way from what intuition suggests, because a `Subjects` row silently leaking a false
denominator onto an estate listing is a materially worse failure than a stray search box on a
census card
([ADR-0102](../adr/0102-a-subjects-row-is-the-base-a-census-member-row-is-its-explicit-modifier.md)).
Every census member is drillable unconditionally, and the drill-down for a member that happens to
be most of the estate carries **no rows at all** — its length was never what would have made it a
findings list ([#74](https://github.com/winniel123/verge-asm/issues/74)). A fully-annotated fired
census (every member accepted) renders as **prose**, categorical, never a mute count
([#164](https://github.com/winniel123/verge-asm/issues/164)) — `Annotation` management lives here:
declaring one, on one `(subject, signal-name)` pair, with no status, no expiry and no author (every
operator dial in the model is unattributed); withdrawing one is not itself a message, since
neither declaring nor withdrawing mints a cause.

### 6.6 Subjects

The estate listing and each subject's own drill-down page. The listing carries search (its
absence would manufacture a false absence at the search box) and **no denominator** (estate
completeness is unmeasurable and refusing to say so is the closest analogue the model has to a
lie) ([ADR-0072](../adr/0072-absence-is-a-property-of-a-cell-and-withdrawn-is-the-only-population.md)).
A subject's own page is the **only surface carrying both its rules and its closed timelines**,
reachable by key even when withdrawn — the "why is this here" card (its `Citation` chain), its
current or closed timelines, and every rule whose `Predicate domain` includes it
([#161](https://github.com/winniel123/verge-asm/issues/161)). A withdrawn subject is marked as
naming a population of no current member — derived on read, no status field, no age, reached only
by its own key, never listed.

### 6.7 Message panel & first run

The message store's global element opens a list of every message, unbounded (no cap or load-more
shipped — no install has yet accumulated enough live volume to say whether one is needed,
[#160](https://github.com/winniel123/verge-asm/issues/160)), each row linking per §5.3's rule.
Click-through targets exist for every fired-at object: an `Endpoint`'s own page, a name-scope
`Seed`'s declared-`Source` view, and a `Service`'s facets opened beneath it when its internet leg
opened ([#180](https://github.com/winniel123/verge-asm/issues/180)). At estate sizes past a few
hundred rows, plain prev/next pagination is unusable and needs jump/first/last controls — an
ordinary widget fix, not a magnitude-conditional treatment.

First run follows the day-one checklist directly (§6.3): declare a domain, optionally upload a
zone file, optionally provision a prober (rendering the instance's own egress address for
declaration once the first connection lands), run the first batch. Nothing runs unattended before
the operator has acted — no config-save scan, no unconditional onboarding sweep.

---

## 7. Explicitly out of scope

Every entry below is a closed decision with its own ADR or ticket; this is the index, not the
argument. Entries marked *reopens on X* are deferrable with no rework; entries marked otherwise are
refused on a ground independent of effort or a future release.

**Product surface.** Full vulnerability scanning (CVE/nuclei-template detection) — v1 reports only
risk signals that fall out of probing for free. Technology fingerprinting — a fingerprint verdict
is a function of the signature database, so diffing it reports observer change as world change
([#5](https://github.com/winniel123/verge-asm/issues/5)). Operator-authored signal rules — an
un-versioned derivation inside the comparison path; *reopens for v1.1* once `Annotation` and
versioning have run in production ([#16](https://github.com/winniel123/verge-asm/issues/16)). A
per-row evidence tier anywhere in the interface — publishing the weak footing tier is confirmed,
putting it on a screen is severity arriving labelled as honesty
([#33](https://github.com/winniel123/verge-asm/issues/33)). Cloud-provider connectors (AWS/Azure/
GCP CAASM) — deferred past v1. Hosted multi-tenant SaaS, and offensive/bug-bounty workflows against
assets the operator does not own — both outside the map's destination outright.

**Access & integration.** A JSON API and API tokens — a bearer token bypasses the TOTP auth flow
over the operator's complete attack surface; the integration need is push, not pull
([#6](https://github.com/winniel123/verge-asm/issues/6)). SSO/OIDC/reverse-proxy forward-auth —
header-trust auth is a bypass class of bug the moment the proxy is misconfigured
([#11](https://github.com/winniel123/verge-asm/issues/11)). A pull notification surface (RSS/Atom/
polling) — the one channel option the no-JSON-API constraint genuinely kills. Vendor notification
integrations (Slack/Discord/Telegram/Teams/Matrix/PagerDuty) and per-vendor body shapes — a body
format with no owner and no watch; *reopens if the curated-table watch (§3.5) acquires an owner for
it*. An operator-editable notification payload template — hands away the no-rows rule in a text
area. Email/SMTP as a channel — refused on the credential (an SMTP secret is usually a
send-as-the-organisation credential and yields only *the relay accepted it*), not on cost; *reopens
for v1.1* with an honest delivery vocabulary. Coalescing/digest windows/flap suppression — every
candidate rests on an unmeasured base rate; class routing is the answer, and the residue (one noisy
rule silences its class) is accepted.

**Discovery aperture.** The BGP leg as a source of address scopes — a routing announcement names
who carries packets toward a prefix, never who controls what listens in it
([ADR-0063](../adr/0063-a-routing-announcement-names-the-path-not-the-estate.md)). Third-party bulk
registry dumps as an instrument — the credential is not machine-checkable and the download is not
gated, so the toggle would assert a signed agreement the software can never see; *reopens for v1.1*.
Sweeping IPv6 address space — one `/64` is ≈4.1×10¹¹ years at the shipped rate ceiling; IPv6
addresses still enter by resolution and are probed normally, they are simply never swept
([ADR-0049](../adr/0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md)). The
CAA and PTR qtypes — CAA's rationale was withdrawn elsewhere, PTR serves under 1% of operators and
needs `dns-record` to key on `Address`; *deferrable with no rework*. DNSSEC records and the DO bit —
collected by no v1 rule; a validation failure is invisible, decided on thin ground and marked as
such. HTTP/3/QUIC — invisible one layer earlier than an ALPN gap; v1's aperture is honestly TCP.
TLS 1.3 cipher-suite enumeration — Go ignores `Config.CipherSuites` for TLS 1.3 entirely; not
expressible without a library change. The wire-protocol prober and a `listener-negotiation` facet —
a whole new facet, canonicaliser, prober leaf and safety surface for zero net new firings past what
already ships. The `wildcard-synthesis` facet (a wildcard's actual content) — v1 holds a wildcard's
content nowhere and a repoint is invisible by design; both halves of a repair (the control-probe
population and the value space) are now specified but the facet itself is priced out. UDP beyond
`connect-outcome`'s existing (uninformative) union — an honest instrument needs the deferred
`datagram-outcome` leaf and its elicitation-payload table.

**Retention & operational record.** A retention dial on the `Message` store or `Delivery` corpus —
`Delivery` travels with its `Message` and is not a corpus in its own right; *reopens on* a measured
message volume large enough to need one, and then as a stated cap with a labelled floor, never a
silent horizon. Compaction of the `Span` corpus — at the shipped ceiling it is the small, flat one
(~672,000 rows) against ~98M observation rows a year; compacting it saves a rounding error and buys
a second invisible horizon. A byte/row budget as the retention instrument — makes the horizon
shortest exactly when the estate is most active. A rebuild command (keep observations longer,
re-derive spans later) — refused at the drift model's root; keeping observations longer than spans
buys nothing. An operator-act record with an actor on every mutation, and any surface over it —
every Declared term is a current value with no timeline, so no consumer needs one on the modal
install (one mutating role); *reopens when the spec admits a second party who can mutate*.

**Signals refused on evidence, not cost.** A `sensitive-port-reached-from-internal` signal — the
sensitive-port list is attested for what is never *internet*-facing; internally a Redis on 6379 is
correct configuration ([#58](https://github.com/winniel123/verge-asm/issues/58)). SNMP on the
sensitive list — no party was found entitled to say exposing it is never correct; *reopens on* a
first-party admission that the protocol assumes callers are inside a boundary. Certificate-
transparency mis-issuance detection — CT holds no timeline after admission, is append-only (so
*mis-issued* and *outlived its name* are opposite facts under one predicate), and the join key does
not exist on the shipped instrument. A `name-served-from-third-party-infrastructure` signal — reads
a Declared act (the operator's own choice of hosting), fires on ~100% of a SaaS-fronted estate
permanently, cites no observation; the fact is rendered as a `Coverage` census line instead. A
`certificate-not-conforming-to-issuance-policy` signal — a different fact needing its own table
against a document, not the world; *reopens for v1.1*. Reading a CA's ARI renewal window per
certificate — a CA load-balancing instrument, not a compliance fact, and a per-certificate outbound
request to every CA in the WebPKI. A coherence gate on shipped defaults ("does anybody actually run
this default?") — a judgement about deployment reality with no owner. A determinacy survey proving
no other service listens on a number — unbounded and unfalsifiable; one document defeats a
determinacy claim and no number of documents proves its negation.

---

*This spec is the map's terminal artefact. Resolving [#12](https://github.com/winniel123/verge-asm/issues/12)
closes the map — implementation sessions start here.*
