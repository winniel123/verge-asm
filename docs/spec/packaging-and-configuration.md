# Packaging, default configuration, and the operator's dials

- **Status:** Accepted — spec content for [#12](https://github.com/winniel123/verge-asm/issues/12)
- **Ticket:** [#124 Packaging, default configuration, and which container receives which secret](https://github.com/winniel123/verge-asm/issues/124)
- **Rulings:** [ADR-0053](../adr/0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md) (secrets), [ADR-0001](../adr/0001-stack-and-runtime.md) (topology and the `GOARCH` matrix, amended here), [ADR-0005](../adr/0005-scan-execution-model.md) (the `Scan` set, amended here)

This is deliberately a separate file from the ADRs that rule it, on
[`measurement-offers.md`](./measurement-offers.md)'s precedent: **the tables below will be revised
and an ADR is a decision that will not.** The knob table in §5 in particular is a list, and its
membership moves whenever a rule, a leaf or a `Scan` moves.

Nothing here is a Dockerfile. The map's *plan, don't do* rule holds: this specifies the artefacts,
the defaults and the surfaces, and an implementation session writes the files.

---

## 1. The `GOARCH` matrix

### 1.1 The matrix

| Axis | Value |
| --- | --- |
| `GOOS` | **`linux`**, and only `linux` |
| `GOARCH` | **`amd64`** and **`arm64`**, and only those two |
| `CGO_ENABLED` | **`0`**, always, for every artefact |
| `GOAMD64` | **`v1`** — pinned, not left to the builder |
| `GOARM64` | the toolchain default (`v8.0`) — pinned, not left to the builder |
| Base image | distroless static, per [ADR-0001](../adr/0001-stack-and-runtime.md) |
| Image | a **manifest list** over both architectures |

### 1.2 The rule that decides membership

> **An architecture is in the matrix exactly where [ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md)'s
> golden corpus is run on it in CI.**

This is not a build preference. ADR-0021's gate is bidirectional — *output moved and the version did
not → fail* — and it is the whole mechanism by which a measurement value cannot move while the world
stands still. An architecture the corpus is not run on ships five leaves whose output is
**unverified on that architecture**, and the failure is invisible in band: two installs on one
release hold **equal derivation vectors**, so the model licenses the comparison, and the values were
produced by two different implementations.

`amd64` and `arm64` are in because CI runs the corpus on both natively. Anything else — `386`,
`arm/v7`, `riscv64`, `ppc64le`, `s390x` — is out until somebody pays for a runner, and the price is
the corpus run rather than a build flag.

### 1.3 This is not hypothetical: Go's floating-point contraction is architecture-dependent

The Go language specification permits fused multiply-add:

> "An implementation may combine multiple floating-point operations into a single fused operation,
> possibly across statements, and produce a result that differs from the value obtained by executing
> and rounding the instructions individually."
> — [Go spec, Floating-point operators](https://go.dev/ref/spec#Floating_point_operators) `[spec]`

Go's `arm64` backend emits `FMADD`; the baseline `amd64` target (`GOAMD64=v1`) has no FMA
instruction to emit. So **one expression, one release, two architectures, two results** — and the
model has no object for it, because a `Derivation` version is a claim about *content*, and the
content is identical.

Two consequences, and both are rules rather than cautions:

1. **`GOAMD64` is pinned at `v1`.** At `v3` the same binary gains FMA on the same architecture, so
   an unpinned level makes the divergence a property of *who ran the build*.
2. **A declared parameter expressed as a fraction is evaluated in exact integer arithmetic.**
   [#67](https://github.com/winniel123/verge-asm/issues/67) cured `certificate-expiring`'s `N` by
   shipping the fraction (⅓ of the certificate's validity period, ½ below a ten-day validity) rather
   than the product — the right cure, and it introduces a division. Evaluated in `float64` that
   division is a candidate for contraction; evaluated on whole seconds in integer arithmetic it is
   not. **The cure must not reintroduce the disease one layer down**: without this, one certificate
   at one instant can be `certificate-expiring` on an `amd64` instance and not on an `arm64` one.
   This binds every fraction in the declared-parameter set, present and future
   ([`project-authored-constants.md`](../research/project-authored-constants.md) §8.1's *ship the
   rule* cure).

### 1.4 `CGO_ENABLED=0` is a measurement decision, not a size one

Two things follow from it that a session reading it as *smaller image* would lose:

- **The pushed prober must be statically linked**, because the host it is pushed to is the
  operator's and its libc is unknown — a glibc-linked binary does not run on Alpine. #14's *the
  instance ships the exact binary it wants at invocation* is only true of a binary with no runtime
  dependencies. This is a **hard requirement of the push model**, not an aesthetic.
- **It fixes which resolver implementation the binary uses.** With cgo available, Go's `net` package
  may use the **system** resolver, which honours `/etc/hosts`, nsswitch and mDNS — a second answer
  path, chosen at build time, for a question
  [ADR-0070](../adr/0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md)
  has already fixed to one declared path. `CGO_ENABLED=0` removes the alternative rather than
  documenting against it.

### 1.5 The image carries a prober binary for **every** architecture in the matrix

This is the constraint nobody had written, and it is not the same statement as *the image is
multi-arch*.

[#14](https://github.com/winniel123/verge-asm/issues/14) makes the instance **push the exact binary
it wants at invocation**, which is what makes version skew structurally impossible. The prober host's
architecture is the **operator's**, chosen when they provisioned a VPS, and it need not match the
instance's: an `arm64` instance on a Pi or on Graviton pushing to an `amd64` VPS is an ordinary
deployment, and the reverse is commoner still.

So: **each image variant embeds the prober binary for both matrix architectures**, and the
instance selects by the prober's reported architecture rather than by its own. The cost is one extra
static binary in the image and it is the price of #14's no-skew property surviving contact with a
heterogeneous estate.

> **The matrix's ceiling is the prober's, not the instance's.** An architecture the image cannot push
> to is not in the matrix, however happily the image itself runs there.

### 1.6 Architecture and OS are checked when the vantage is declared, never at first push

The prober's `uname -s` and `uname -m` are read over the SSH connection **at provisioning**, and a
host that is not `Linux` on `x86_64` or `aarch64` is **refused there** — with the reason, and with
#14's *SSH access is a hard prerequisite* extended to name the platform.

This is [ADR-0047](../adr/0047-an-address-scope-is-its-own-enumeration.md)'s address-scope cap
pattern: *checked when the scope is declared*, not applied as a filter while a batch runs. Deferring
it to the first push would produce a vantage that exists, never verifies, and degrades `Exposure` to
non-constructible — a legible failure, but one that arrives hours later at 03:00 and reads as an
outage rather than as a typo.

---

## 2. Compose shape

Three services, one image, three volumes.

| Service | Image | Listener | Volumes |
| --- | --- | --- | --- |
| `web` | the one image | the only one | `web-state` |
| `worker` | the one image | none | `worker-state` |
| `postgres` | upstream | none published | `pgdata` |

Carried forward from [#4](https://github.com/winniel123/verge-asm/issues/4) §3.3 and §10 and not
reopened: **`user:` non-root, `cap_drop: [ALL]`, no `cap_add`, no `privileged: true`, no
`network_mode: host`, no `sysctls:`.** The prober binary inherits the same posture on the host it is
pushed to: it is invoked as an ordinary unprivileged SSH user and needs no capability, because
[#4](https://github.com/winniel123/verge-asm/issues/4) §3.3 chose TCP connect over SYN precisely so
that it would not.

`postgres` publishes no port. The database is reached over the compose network by `web` and
`worker` alone.

---

## 3. Which container receives which secret

Ruled by [ADR-0053](../adr/0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md).
**The database holds no secret.**

| Secret | Held by | How it arrives | Where it lives |
| --- | --- | --- | --- |
| Database credential | `web`, `worker`, `postgres` | environment, **required** — `${POSTGRES_PASSWORD:?}`, so compose fails rather than defaulting | the environment |
| Session signing key | `web` | **generated by `web`** on first boot | `web-state` |
| Prober SSH private key | `worker` | **generated by `worker`** at provisioning; only the public half leaves | `worker-state` |
| Setup token ([#11](https://github.com/winniel123/verge-asm/issues/11)) | `web` | generated, written to stdout, single-use | nowhere — it is consumed |
| Prober host key pin | — | observed at provisioning | the database — **it is public** |
| Operator-supplied zone file | — | uploaded through `web` | the database — **it is evidence, not a secret** |

Two riders the table cannot carry:

- **`web` renders no secret value, ever** — not a value, not a masked value. It renders **set** /
  **not set**, and the prober's **public** key.
- **The prober's SSH host key is pinned at provisioning and a change is a hard failure**, never a
  prompt. An unpinned push lets whoever answers on that address run our binary and return NDJSON,
  which is inventory fabrication by a party the operator never provisioned. The failure path already
  exists and is the right one: the vantage stops verifying, becomes `unavailable`
  ([ADR-0005](../adr/0005-scan-execution-model.md)), opens a `Gap`, and makes `Exposure`
  non-constructible.

Operator-side guidance, documented rather than enforced (it is on their `authorized_keys`, not in our
compose file): install the public key under `restrict`, and — once the instance has rendered its own
egress address (§4.2) — under `from=` that address.

---

## 4. Declaring vantage intent, and provisioning the prober

### 4.1 There is no `network_position` field and no setup prompt

Both were specified and both are dead:

- `network_position: internal | external | unknown` was **rejected by
  [#14](https://github.com/winniel123/verge-asm/issues/14)** as relative — *external to what* is the
  problem — and replaced by a verified claim. `safe-active-probing.md` §8.2 rec 1 still specified it
  and is struck there.
- *A first-run setup step should ask where the deployment sits* (§8.2 rec 3, and §9's
  `Vantage / network position | prompted at setup`) is a **wizard**, and
  [#22](https://github.com/winniel123/verge-asm/issues/22) into
  [#28](https://github.com/winniel123/verge-asm/issues/28) made the day-one checklist a **rendering**.
  Both are struck.

What replaces them is a declaration the model already has.

### 4.2 A `Vantage`'s class is verified over the address it **presents**

`CONTEXT.md`'s `Vantage class` fixes the test — *every batch, against the address-scope `Seed`s the
system already holds, and against nothing else; one uncovered address and it verifies `internet`* —
and says nothing about **which addresses are in the set**. They are the addresses the vantage is
observed to **present**, and an interface address is not one.

| Vantage | Presented address | Observed by |
| --- | --- | --- |
| a **prober** | the address the instance dialled | **construction** — the instance reached it |
| the **instance** | `SSH_CLIENT`'s first field | the **prober**, from outside |

Both are [#14](https://github.com/winniel123/verge-asm/issues/14) decision 7's own self-contained
checks, unchanged. What is new is that they are the *whole* of the set.

**The forcing argument is a cap the project already ships.** Under *every address it holds*, a NATed
instance can only verify `internal` by declaring its own LAN as an address scope — and
[ADR-0049](../adr/0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md)'s cap
refuses any scope above **1,024** addresses, so an operator on `10.0.0.0/8` **cannot declare it at
all**. On that reading `Exposure` is unreachable by construction on a large class of ordinary
estates. Narrowing the set to the presented address is therefore forced, not chosen.

It also lands where #4 §8.1 already pointed: an interface address behind NAT is not what the world
sees, and probing your own public address from inside is that section's hairpinning trap
(RFC 4787 REQ-9) — a false *closed* or a false *open* that never traversed the inbound policy.

[ADR-0049](../adr/0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md)'s
**every-not-any** quantifier is untouched and runs over this set. One residue is narrowed rather than
closed and is stated rather than assumed away: **a dual-stacked prober dialled over IPv4 has an IPv6
egress nobody observed**, so `every` runs over a set that does not contain it. Closing it needs a
second observation from a second family, which the instrument does not make.

### 4.3 Consequently, `Exposure` requires a prober — unconditionally

An internet `Vantage` exists exactly where a second host observed this one's presented address, and
v1 has exactly one such observer: a prober reached over SSH. With no prober, the instance's presented
address is **unobserved**, its class is **unverified**, and #14 decision 6 disposes of that already —
*no exposure claims, degrade to internal-only, never `firewalled`, because we did not look.*

This makes the four-step checklist's step 3 — *Add an internet vantage — enables exposure. With one
vantage the conclusion has no definition* — **structurally true rather than conventional.**

> **#14's VPS sentence is amended at the clause.** *"An operator who deploys the whole stack on a VPS
> is classified as an internet vantage automatically and receives real exposure findings"* rests on
> the check being uniform across vantages. It is not, and #14's own decision 7 says why: the
> instance's egress is *"read from `SSH_CLIENT` as the prober observes it"*, which requires a prober.
> The two sentences are inconsistent, and the inconsistency is resolved in the direction the rest of
> the model already runs — #58's *on a one-vantage-class install there is no `Exposure`*,
> [ADR-0017](../adr/0017-exposure-needs-both-legs.md)'s both legs, and #14's own *the default
> no-prober deployment is a complete, honest internal reachability inventory.* A VPS-hosted instance
> gets exposure the same way every other install does: by provisioning a prober.

### 4.4 The intent, and where it is declared

#14 decision 6's requirement — *declaration alone is a lie waiting to happen; detection alone fails
silently; intent gives the check something to contradict* — is kept in substance and loses its field.

| Intent | Declared by | Contradicted by |
| --- | --- | --- |
| *this vantage is on the internet* | **provisioning a prober** — supplying an SSH host is the act | its presented address falling inside a declared address scope |
| *this vantage is inside my boundary* | **declaring an address scope that covers the instance's presented address** — in practice a `/32` or `/128` | the presented address not being covered |

Both are acts on Declared objects the model already has, both are audit-trailed like every other
Declared act ([#11](https://github.com/winniel123/verge-asm/issues/11)), and neither is a settings
enum whose value can drift away from the deployment it describes.

**The instance's own egress is knowable only after a prober exists**, so the sequence is: provision
the prober → the first connection observes `SSH_CLIENT` → the instance renders *your egress is
`203.0.113.5`* and offers it for declaration. Until it is declared, **both** vantages verify
`internet`, there is no internal leg, and `Exposure` is non-constructible — which is loud, honest,
and states an act the operator can perform. It never fabricates `exposed`, which is
[#4](https://github.com/winniel123/verge-asm/issues/4) §8's named worst outcome.

That rendering is a **`Coverage`** statement, not an alert: it is *we cannot construct the claim*,
which is the band #22 built. Whether the offer is literally an
[ADR-0012](../adr/0012-a-proposer-is-not-a-source.md) `Proposal` — a new producer, in answer to an
operator act — is left open; see the ticket's *handed onward*.

### 4.5 What the operator supplies to provision a prober

Four values, none of them a secret:

| Value | Note |
| --- | --- |
| host | a name or an address the instance can reach on the SSH port |
| port | defaults to 22 |
| username | a **non-root** account |
| — | the instance generates the keypair and renders the **public** half to install |

And two things read, not typed: the host key (pinned) and `uname -s` / `uname -m` (§1.6).

---

## 5. The operator's dials

### 5.1 Where configuration lives

> **The environment configures the process; the database configures the product.**
>
> **The test:** if a change to it should appear in the audit trail, it may not live in the
> environment.

The environment holds only what must exist before the database does — the database URL and
credential, the listen address, and #11's `VERGE_SETUP_TOKEN` escape hatch. Everything Declared —
`Seed`s, exclusions, `custody extension`s, `Scan`s and their scopes and cadences, source enablement,
vantages, notification routing — is a row, edited through the UI by an authenticated admin, because
[#11](https://github.com/winniel123/verge-asm/issues/11) requires an author for *who changed the seed
list, who launched a scan against production* and a file has no author.

Two consequences: **there is no configuration file to mount**, and `--scale worker=N` workers are
byte-identical, so no worker can be running a different aperture from its siblings.

### 5.2 The gate every dial must pass

A knob ships only if it clears all three:

1. **It sits outside every `Derivation`.** `CONTEXT.md`'s `Derivation` and
   [ADR-0004](../adr/0004-signals-are-release-coupled-rules.md) / [#60](https://github.com/winniel123/verge-asm/issues/60):
   *a declared parameter is authored by the project and ships in the release; an operator's dial may
   sit anywhere outside every derivation and nowhere inside one.* A settings field inside a leaf is
   the one actor that can `Break` the estate without a release.
2. **It cannot silence a finding by narrowing.** [ADR-0009](../adr/0009-verge-core-is-a-union.md)'s
   *a port the operator can hide is a signal the operator can silence*, generalised by
   [`measurement-offers.md`](./measurement-offers.md) §1.7 to *an offer the operator can narrow is a
   finding the operator can silence*.
3. **If it moves the aperture, it moves a named dimension.** A `Scan`'s scope, a `Seed`, a `Seed`
   exclusion — something a `Batch`'s recorded scope diffs, so the widening or narrowing carries its
   `revealed`, its `Gap` and its coverage-class message
   ([ADR-0005](../adr/0005-scan-execution-model.md), [ADR-0014](../adr/0014-only-revealed-generalises.md)).

### 5.3 The knob table, swept

[`safe-active-probing.md`](../research/safe-active-probing.md) §9 proposed eighteen. Every row is
walked against §5.2's gate; the note's table is struck row by row at its own site.

| Knob | Verdict |
| --- | --- |
| **Rate limit per host** (50 pkt/s) | **Ships.** [ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md) puts rate-limiting outside every leaf and [`project-authored-constants.md`](../research/project-authored-constants.md) §6.6 confirms it is not a declared parameter at all. It moves wall-clock and, at the extreme, a batch's completed scope — which is coverage, recorded and rendered |
| **Concurrency** (20 per host / 200 global) | **Ships**, same ground |
| **Port set per tier** | **Ships for the frequency half only** — ADR-0009, unchanged. The sensitive half is not editable: gate 2 |
| **Full-range sweep** | **Ships**, opt-in per `Seed` scope — [ADR-0044](../adr/0044-a-one-off-measurement-has-no-currency.md), unchanged |
| **Scan cadence per tier** | **Ships.** Declared on the `Scan`, read at evaluation time by the currency bound, exactly as [`project-authored-constants.md`](../research/project-authored-constants.md) §6.2 prescribes |
| **Quiet hours / maintenance windows** | **Ships.** It moves *when*, never *what* |
| **Per-target enable/disable, and pause-all** | **Ships in the form the model already has.** Pause-all is operational. A per-target **disable** is an aperture narrowing and is therefore a `Seed` **exclusion** — a Declared claim about where the estate ends — never a hidden mute, or it is gate 2 arriving per target |
| **Adaptive back-off aggressiveness** | **Ships**, and the reason is narrow: ADR-0021 places back-off outside `connect-outcome` **because it halves the rate and never the deadline**. A knob that could reach the deadline would fail gate 1 |
| **Target range size cap** (1,024) | **Ships.** `CONTEXT.md` already makes it operator-configurable; it is checked at declaration and read by no rule |
| **TLS enumeration cadence** (weekly) | **Ships** — it is the `tls-acceptance` `Scan`'s cadence ([#61](https://github.com/winniel123/verge-asm/issues/61)). The candidate **set** does not: `measurement-offers.md` §1.7 |
| **Source address / interface** | **Amended, not a plain knob.** Pinning egress changes the address the vantage presents, so it is a **different `Vantage`** under [ADR-0070](../adr/0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md)'s generalisation — its timelines **open** (`revealed`) rather than a setting quietly moving where answers were drawn from |
| **Follow redirects** | **REFUSED.** `http-exchange` decides `Responded(status, Location, WWW-Authenticate, Server, title)`; following a redirect changes the status and the title, so this is a **declared parameter of `http-exchange`**, valued at **not followed** — which is §4.3's own recommendation and the map's own *redirects unfollowed*. Gate 1 |
| **HTTP probe paths** | **REFUSED, and the row is stale as well.** v1 makes **one** exchange per `Endpoint`, `GET /` (§4.1); there is no path list to shrink or extend. §4.4's path-probe-plus-matcher is exposed-panel detection, which is [#5](https://github.com/winniel123/verge-asm/issues/5)'s refused fingerprinting and never entered the value space |
| **User-Agent** | **Split, and both halves are refused as knobs.** Being **identifiable** is a project commitment (§10's attributability posture) and stays. Being **changeable** fails gate 1: a WAF that blocks unknown agents returns a different response, so the string moves `http-identity`. It is a **declared parameter of `http-exchange`**, valued at the identifying string. Cost stated: an estate whose WAF blocks us records the WAF's identity — which is a true answer to the question the facet asks, *what does a client that names nothing meet* — and the alternative, sending a browser's string, is the impersonation §10 refuses |
| **Cert expiry thresholds** (30 / 14 / 7) | **REFUSED, and it was refused twice already.** [#60](https://github.com/winniel123/verge-asm/issues/60) made `N` a declared parameter and *"that is precisely why it may not be a dial"*; [#67](https://github.com/winniel123/verge-asm/issues/67) replaced 30 days with ⅓ of the certificate's validity. This row is the ADR-0058 site nobody had struck |
| **`suspect-firewall` port threshold** (100) | **REFUSED — there is no such rule.** The v1 set is **seventeen** ([ADR-0024](../adr/0024-a-rules-domain-is-the-extension-of-its-name.md)) and `suspect-firewall` is not among them; §6.3's suppression heuristic never entered the model. A threshold for a rule that does not exist is not a knob |
| **Vantage / network position** | **REFUSED as a field** — §4.1 |
| **UDP scanning** | **Not a knob in v1 — see §6.** It is not *off by default*; it is outside the instrument |

### 5.4 What is not a dial and never becomes one

Restated in one place because it is where a session reaches by reflex: the TLS candidate set, the
qtype set, the ALPN list, the EDNS options, the DNS transport policy, every timeout and retry budget,
the control-label count and construction, the match predicate, the query path, the capped body read,
`k`, and `certificate-expiring`'s fractions. All are declared parameters
([ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md)'s table,
[`project-authored-constants.md`](../research/project-authored-constants.md) §6.8), and none is ever
operator-configurable.

---

## 6. What a default install measures

> **v1 measures `verge-core`'s TCP pairs daily on every address in custody, ~~and nothing else,~~
> until the operator says otherwise.**
>
> ***And nothing else* is withdrawn at this site**
> ([ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)). The
> sentence is about the **port-tier aperture**, and read alone in the present tense it denies two
> measurements a default install makes: `tls-acceptance`'s weekly enumeration over open `Service`s,
> and — since [#142](https://github.com/winniel123/verge-asm/issues/142) /
> [ADR-0084](../adr/0084-a-scan-is-a-cadence-over-an-exchange-and-an-uncovered-facet-has-no-currency-bound.md)
> — the **`dns` `Scan`, daily over the name scopes and ungated by `Custody`**, which is the only
> thing an install holding custody of nothing measures at all
> ([ADR-0019](../adr/0019-the-probing-gate-is-total-over-an-address.md)). The figures below are
> untouched: they are about ports, and `dns` has none.

That is [ADR-0044](../adr/0044-a-one-off-measurement-has-no-currency.md)'s sentence with **three
words added**, and the three words move a published figure.

`verge-core` is **136 pairs — 131 TCP and 5 UDP** ([ADR-0009](../adr/0009-verge-core-is-a-union.md)),
and ADR-0009 states in terms that **131 are probed on default settings, UDP being off**. The five
UDP pairs — `69`, `137`, `138`, `623`, `11211` — reach `verge-core` from the **sensitive list alone**.
So on a default install **five of the thirty-eight sensitive pairs are not read**, and #80's
aperture-statement figure `0 of 38 sensitive pairs unread` is false there.

The inference that produced it is stated in #80 and is the thing that fails: *ADR-0009's union puts
every sensitive pair inside the hot set, so `sensitive-port-reached-from-internet` reads all of them
daily.* Membership of `verge-core` is not measurement. **The aperture is `verge-core` ∩ the transports
the shipped configuration probes**, and the honest count is over the intersection.

The port-tier line on `Coverage` therefore reads **`5 of 38 sensitive pairs unread`**, and — because
[#44](https://github.com/winniel123/verge-asm/issues/44) decision 7 permits a count of our own lists
and #80 permits a pointer where an action exists — it names the transport and points at the tier
config. What it must **not** do is round it to zero, which is the unearned clean bill of health #80
went out of its way to say this figure was not.

The ~~`0 of 16 rules unevaluable`~~ half is **untouched**: `sensitive-port-reached-from-internet` reads a
leg on a `Service`, and its domain is populated by the 131 TCP pairs, so the rule speaks. What the
five UDP pairs cost is **subjects**, not evaluability — there is no `Service` for a pair outside the
recorded scope.

> **The denominator is superseded here, at the site that states it — the rule set is seventeen.**
> [#128](https://github.com/winniel123/verge-asm/issues/128) ·
> [ADR-0071](../adr/0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md) moved
> it from sixteen and discharged its other consequences by name — ADR-0024's table, ADR-0004, the v1
> walk's row, *thirteen of sixteen* → *thirteen of seventeen* — but **not this figure**, which was
> #124's and sat in a different document.
> **The numerator is deliberately not re-filled**, because it is not obviously still `0`: the
> seventeenth rule reads `resolution` rather than a port, so the *aperture* argument above does not
> reach it, but [#137](https://github.com/winniel123/verge-asm/issues/137) ·
> [ADR-0079](../adr/0079-authority-presupposes-denotation-a-non-globally-reachable-address-is-probed-only-inside-a-declared-realm.md)
> and [#138](https://github.com/winniel123/verge-asm/issues/138) ·
> [ADR-0080](../adr/0080-a-vantage-composition-is-cross-class-or-class-scoped-and-only-one-takes-a-quantifier.md)
> together establish that it is **permanently `not-evaluable` on an install with no internet-class
> vantage** — which is a second, install-shaped route to unevaluability this figure has never had to
> count. Writing `0 of 17` would assert an answer nobody has walked.
> Found by [#141](https://github.com/winniel123/verge-asm/issues/141) and recorded by the merging
> session; the walk is ticketed.

~~**Whether v1's instrument could measure a UDP `Service` honestly at all is not settled here.**~~
**SETTLED by [#141](https://github.com/winniel123/verge-asm/issues/141) /
[ADR-0083](../adr/0083-silence-decides-only-on-a-connection-oriented-transport.md), and struck here
per [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).**
`connect-outcome`'s union is `connected │ refused │ no-response`, and a connected UDP socket sends no
packet, so `connected` would be a fact about our own kernel. ~~It is ticketed rather than guessed.~~
**It cannot, and neither could a widened `connect-outcome` — but an honest instrument is
constructible: `answered │ refused │ unanswered`, decided by a **sixth leaf** `datagram-outcome` and
specified rather than shipped. `refused` is reused unchanged; `no-response` is **not**, because it
projects to `not-reached` and a connectionless exchange did not decide that; `unanswered` projects
onto no `Reach` and therefore returns `not-evaluable`. **Nothing in this section's figures moves** —
`verge-core` is still 136 pairs / 131 probed, and the aperture statement still reads
`5 of 38 sensitive pairs unread`, because UDP does not turn on here. What ADR-0083 adds to this
section is the reason it should not: a payload-free UDP leg buys **zero** net new firings, and the
payload table that would change that is the wire prober
[ADR-0015](../adr/0015-the-value-space-is-the-commitment.md) deferred.**

---

## 7. The zone file: supply, cadence, and the zone it records

[#48](https://github.com/winniel123/verge-asm/issues/48) shipped two rules that read the operator's
zone file and flagged its own thin ground: *"if it is a file mounted once and never re-read, route 4
never fires and a stale file is invisible instead of `not-evaluable`. That belongs to the packaging
patch."* Here it is, and the answer is not a mount.

### 7.1 It is uploaded, not mounted, because supply is an act

A bind-mounted file has no supply act and therefore no instant. Worse, **re-reading unchanged bytes
produces a current observation of a stale fact** — so a cadence over a mount does not fix #48's
problem, it hides it more thoroughly than a single read does. The zone file is uploaded by an
authenticated admin, which is the **second** checklist step's own word (*Upload a zone file — enables
removal detection*), and the upload instant is the observation instant.

**Upload is the only supply mechanism in v1**, which strikes two of the three
[`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §3.3 offers at their site.
An **on-disk path** is a mount with no supply act and therefore no instant. **AXFR with a TSIG key**
is not another way to obtain the same source at all — it is a **different source**, with `measured`
authority against an authority we walked rather than `declared` authority from the operator, and it
appears in no enumerated source set. Out of v1; if it is ever added, its TSIG key is `worker`'s under
[ADR-0053](../adr/0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md)
and it needs its own row in the source table, not a checkbox on this one.

> **A `declared` source's observation takes its instant from the operator's supply act, never from
> our read of it.** That is what makes #48's fourth absence register — *you stopped telling us* —
> reachable at all.

### 7.2 It gets a `Scan`, because a cadence needs one

`CONTEXT.md`'s `Scan` — *recurring is load-bearing; currency is `k` cadences of the covering `Scan`,
so a measurement with no cadence has no currency bound* — and
[ADR-0044](../adr/0044-a-one-off-measurement-has-no-currency.md) together mean the zone file's
staleness is unbounded unless something covers it. Nothing did.

**A fourth `Scan`, `zone`.** Scope: the name scopes holding a supplied zone. **No port list, no
vantage choice** — the worker reads it.

> **There are now five, and this section's *fourth* is an ordinal rather than a count** —
> [#142](https://github.com/winniel123/verge-asm/issues/142) /
> [ADR-0084](../adr/0084-a-scan-is-a-cadence-over-an-exchange-and-an-uncovered-facet-has-no-currency-bound.md)
> added **`dns`** on the same reasoning, covering `resolution` and our own resolver's `dns-record`.
> Read **two port tiers and five `Scan`s** wherever this document counts them. *(ADR-0005's #124
> amendment points at this section for the "full statement" of that hole; the statement was never
> written here, and it is now discharged rather than transcribed.)* Cadence: the **operator's re-supply interval**, shipped at
**monthly**, which is their declaration of how often they will re-export. Its batches restate the
stored file's observations at the **supply instant**, so `k × cadence` bounds the operator's promise
rather than our read, and a zone last supplied more than two cadences ago ages into a `Gap` — #48's
route 4, firing exactly as designed.

A name scope with no zone supplied leaves the scope list empty, which is ADR-0005's own legible
state.

**Why not the daily tier.** Hanging the read off a port-tier `Scan` would give a hand-supplied export
a **two-day** currency bound, so an operator who did everything right would see the source go
`not-evaluable` on the third day; and disabling the port tier would silently stop the zone being
read. Both are the hidden-field failure ADR-0005's *port tiers are three `Scan`s* section refuses.

### 7.3 The zone is read from the file, which discharges #48's second obligation by construction

#48 requires the batch to record **the zone, not the registrable domain** — *"a scope recorded one
level too wide fires this rule on every name in every delegated subzone."*

**The zone is read from the file's own `$ORIGIN` and SOA owner name**, never from the filename, never
from the `Seed`. A zone export states its own apex; a `Seed` states a registrable domain; and where
those differ the file is right about what it enumerates. A file whose apex sits outside the name
scope it was uploaded against is **refused at upload**, with the reason — the same
declaration-time check §1.6 and ADR-0047's cap use.

An operator with delegated subzones uploads one file per zone, and each carries its own scope.

---

## 8. The four-step checklist, and what this document supplies

[#28](https://github.com/winniel123/verge-asm/issues/28) and
[#51](https://github.com/winniel123/verge-asm/issues/51) fixed the checklist as the zero-coverage
**rendering** of `Coverage` at two densities. Its four steps are *Declare your domain* · *Upload a
zone file* · *Add an internet vantage* · *Run the first batch*.

Steps 2 and 3 had capability annotations and no mechanism. §7 supplies step 2's and §4 supplies
step 3's. Neither adds a step, neither adds a prompt, and neither makes the checklist a wizard: each
step names a capability and, per [ADR-0044](../adr/0044-a-one-off-measurement-has-no-currency.md)'s
allowance, may point at the surface where the act is performed, **because here an action genuinely
exists**.
