---
title: Discovery sources
section: Scanning
order: 1
description: Discovery sources widen what verge-asm knows about your estate before it measures — the catalogue, consent tiers, and the crt.sh and RIR caveats.
---

# Discovery sources

Discovery sources widen what verge-asm knows about your estate *before* it measures.
This guide covers the **Sources** page (`/sources`): what the catalogue holds, the
consent tiers that decide what ships on, and that toggling is admin-only. It also covers
the two caveats that confuse first-run operators — crt.sh and the RIR proposers.

The catalogue is authored in [`cmd/web/sources.go`](../../cmd/web/sources.go). The
consent model is [ADR-0003](../adr/0003-third-party-source-consent-bar.md) and the
proposer-vs-source distinction is
[ADR-0012](../adr/0012-a-proposer-is-not-a-source.md).

---

## Sources vs proposers

Two different things sit on this page, and the difference matters:

- A **source** *observes* and **admits subjects** into your estate on its own authority
  — crt.sh reads certificate-transparency logs and admits the `Name`s it finds. It has an
  `authority` and a `completeness`, and enabling it widens your aperture.
- A **proposer** *observes nothing*. It answers an **org-name search** with candidate
  **address scopes** you might own, as `Proposal`s. A proposal admits nothing until you
  **confirm it into a `Seed`** — until then it is probed by nothing. Proposers carry
  `consent` alone. They have no authority and no completeness because they add no facet.

So enabling a source can change what is in your estate. Enabling a proposer only changes
what an org-name search is allowed to *suggest*.

---

## Consent tiers

Every catalogued entry ships under a **consent tier** — release-authored data, not a
per-install setting. It names *which door* the reading goes through, never who walked
through it. v1 uses two tiers:

| Tier | Meaning | Ships |
| --- | --- | --- |
| **`unencumbered`** | No terms bar the operator from this source, so the project runs it without you having to say so. | **on** by default |
| **`operator-accepted`** | The project could not clear the source's terms on your behalf and refuses to read them for a stranger. **You** accept the terms and bear the reading. | **off** — you enable it |

`operator-accepted` is your reading of the terms, **not** a certification that you
comply — the project simply declines to make that call for you. The model reserves a
third tier, `operator-credentialed` (a source needing your own API key), but **no v1
source uses it**.

A third disposition is not a tier at all: some entries are **barred**. They are excluded
on their terms and non-toggleable. They carry no consent tier, because the project does
not run them for anyone.

---

## The v1 catalogue

Ten entries ship. What each is, what it discovers, and its consent tier:

### Sources (observe and admit)

| Entry | Discovers | Tier | Ships |
| --- | --- | --- | --- |
| **crt.sh** | `Name`s from certificate-transparency SAN lists | `unencumbered` | **on** (see caveat below) |
| **HackerTarget** | — | — | **barred** — excluded on terms |
| **Cert Spotter** (unauthenticated) | — | — | **barred** — excluded on terms |

### Proposers (org-name search → address-scope proposals)

| Entry | Region / path | Tier | Ships |
| --- | --- | --- | --- |
| **ARIN** (`entities?fn=`) | North America, keyless org→prefix | `unencumbered` | **on** |
| **AFRINIC** (CAIDA ⋈ delegated-stats) | Africa, keyless org→prefix | `unencumbered` | **on** |
| **APNIC** (CAIDA ⋈ delegated-stats) | Asia-Pacific, keyless org→prefix | `unencumbered` | **on** |
| **RIPEstat** | RIPE region | `operator-accepted` | **catalogued — no runner** ([#241](https://github.com/winniel123/verge-asm/issues/241)) |
| **RIPE Database** | RIPE region | `operator-accepted` | **catalogued — no runner** (#241) |
| **APNIC registry** | APNIC region | `operator-accepted` | **catalogued — no runner** (#241) |
| **LACNIC registry** | Latin America | `operator-accepted` | **catalogued — no runner** (#241) |

The three keyless proposer paths ship on because they are `unencumbered`. The four registry
paths are `operator-accepted` **by tier**, but **no `proposer.Source` runner ships for them
yet**. They render consent+toggle but would emit nothing. So they are **catalogued — not
yet executing** (the #241 mechanism): non-toggleable, offering **no consent dialog**, and
off for everyone until a runner lands. At that point they return to *ship off — accept the
terms*.

The `/sources` modal buckets every entry by state. The buckets are *shipped on*, *ship off
— accept the terms* (empty in v1, since every `operator-accepted` entry is currently
runnerless), and *not run for anyone*. The third bucket holds both barred entries and
these catalogued-but-runnerless proposers.

---

## Enabling and disabling — admin only

Any logged-in account may **read** `/sources`. **Toggling a source on or off is
admin-only.** A viewer sees the catalogue with a notice — *"You have read access.
Enabling or disabling a source is admin-only"* — and no toggle control. The toggle route
itself is gated: a non-admin `POST` is refused with `403`.

Barred entries are non-toggleable for everyone, admins included.

**What a toggle records.** The current on/off value is stored per-install (one row per
source, the value overwritten in place). Like every declared term it keeps **no
per-toggle history and carries no actor or timestamp of its own**. It is **dated by the
`Batch` whose recorded source set it moved**, which is where the audit trail lives. So
"when did this source change" is answered by the batch record, not by a log line on the
toggle.

---

## Two caveats worth knowing

### crt.sh executes (as of the CT runner, ADR-0106)

Earlier builds catalogued crt.sh but shipped **no execution path**. Issue
[#241](https://github.com/winniel123/verge-asm/issues/241) held it *defined but inert*,
enabled in the catalogue yet running nothing. **That is no longer the case.** The crt.sh
CT runner ([#250](https://github.com/winniel123/verge-asm/issues/250), ADR-0106) landed a
real runner — the **`ct` scan**. So crt.sh is a **live source again**: it ships on,
polls crt.sh on a daily cadence (throttled), and admits the `Name`s it finds. See
[first-run.md → Confirm a scan ran](first-run.md#confirm-a-scan-actually-ran) and
[running.md → On-demand scan triggers](running.md#on-demand-scan-triggers) (`-trigger ct`).

The *catalogued-but-runnerless* mechanism #241 built now holds the four RIR **registry
proposers** — RIPEstat, RIPE Database, APNIC registry, LACNIC registry. Each is
`operator-accepted` by tier but ships with **no `proposer.Source` runner**. So it is
catalogued-yet-inert: rendered in the third *not run for anyone* bucket, **non-toggleable**,
with **no consent dialog offered**. It stays that way until a real runner lands and returns
it to *ship off — accept the terms* (the same reversal crt.sh made). The three keyless
proposer paths (ARIN, AFRINIC, APNIC via CAIDA) execute today. The four registry paths do
not.

### RIR proposers propose address scopes, not subdomains

The RIR entries (ARIN, AFRINIC, APNIC, RIPE, LACNIC) are **proposers**, and they behave
differently from what an operator expecting "subdomain discovery" assumes:

- They run **only on an org-name search** — go to **Proposals**, enter your organisation
  name, and the enabled proposers answer. They do not run on a cadence and they do not
  crawl.
- What they return is **address scopes** (CIDR prefixes the registries associate with
  that org name), never subdomains. If you want subdomain-shaped discovery, that is
  crt.sh's job, not a proposer's.
- A returned proposal **adds nothing to your estate** until you **confirm it into a
  `Seed`**. Decline it and the decline is recorded as an exclusion. Confirmation is the
  aperture act — the proposal alone asserts nothing and is probed by nothing.
