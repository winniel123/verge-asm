---
title: Signals reference
section: Signals & delivery
order: 1
description: The per-signal reference for the v1 rule set — what each signal means, its subject, and when it fires.
---

# Signals reference

A **signal** is a named fact with evidence — a release-coupled rule that reads what
the system measured and reports whether a specific condition holds for a subject. This
page is the per-signal reference: what each v1 signal is, which subject it is about,
and when it fires. The **Signals** page in the UI shows the current firings; this is
where you look up what one *means*.

The rules themselves live in [`internal/signal/`](../../internal/signal/) and their
design is fixed by
[ADR-0004](../adr/0004-signals-are-release-coupled-rules.md). This reference is
**release-coupled**: it describes the v1 rule set (seventeen rules) and is meant to be
verified against the code, never to drift from it.

---

## Rule status — what fires on a default install

Not every rule is wired yet. **On a default single-host install, 9 of the 17 rules can
fire** — the four Name-only rules, the four HTTP-identity endpoint rules
(**P0.11**, landed), and `tls-1.0-accepted` (**P0.9**, landed). The other **8 are
dormant**:

- the **six certificate rules** render `not-evaluable` for every presented chain,
  because the parsed-leaf attributes they read are not stored yet; they wake when the
  certificate-parsing leaf lands (**P0.10**, blocked on design collision #37);
- the **two internet-gated flagship rules** need an internet-class `Vantage` (a
  provisioned prober), so a single-host install never enters their domain — the
  flagship wiring is deferred to **#700** (the off-host SSH transport that carries a
  prober landed with P0.8).

| Rule | Subject | Status on a default install |
| --- | --- | --- |
| `lame-delegation` | Name | **Live** |
| `cname-target-name-error` | Name | **Live** |
| `zone-declared-name-returns-name-error` | Name | **Live** (needs an uploaded zone file) |
| `resolved-name-absent-from-zone` | Name | **Live** (needs an uploaded zone file) |
| `non-globally-reachable-address-resolved-from-internet` | Name | **Dormant** — needs an internet vantage (#700) |
| `certificate-expired` | Endpoint | **Dormant** — certificate-parsing leaf (P0.10, collision #37) |
| `certificate-not-yet-valid` | Endpoint | **Dormant** — certificate-parsing leaf (P0.10) |
| `certificate-expiring` | Endpoint | **Dormant** — certificate-parsing leaf (P0.10) |
| `certificate-self-signed` | Endpoint | **Dormant** — certificate-parsing leaf (P0.10) |
| `certificate-weak-key-or-signature` | Endpoint | **Dormant** — certificate-parsing leaf (P0.10) |
| `certificate-hostname-san-mismatch` | Endpoint | **Dormant** — certificate-parsing leaf (P0.10) |
| `plaintext-http-no-https` | Endpoint | **Live** (P0.11) |
| `redirect-does-not-upgrade-to-tls` | Endpoint | **Live** (P0.11) |
| `redirect-to-host-outside-estate` | Endpoint | **Live** (P0.11) |
| `unauthenticated-request-answered` | Endpoint | **Live** (P0.11) |
| `tls-1.0-accepted` | Service | **Live** (P0.9) |
| `sensitive-port-reached-from-internet` | Service | **Dormant** — needs an internet vantage (#700) |

A **dormant** rule is never faked: it renders `not-evaluable` (certificate rules) or
sits `outside-domain` (the internet-gated rules) on a default install, never a
manufactured verdict. Provisioning a [prober](prober.md) wakes the two internet-gated
rules; the certificate rules wait on the leaf.

---

## What a signal is — and is not

- **A named fact, not a per-finding score.** The fact a signal names carries **no score you
  tune per finding**: verge-asm's subject is *change*, so urgency comes from the transition
  that surfaced a signal (see **Messages**), never from a number stapled to a static backlog.
  What a signal *does* carry is its **rule's severity** — every rule ships at one of **five**
  release-authored levels, the P0.1 ramp **Critical / High / Medium / Low / Info**
  ([#1](https://github.com/winniel123/verge-asm/issues/1)), resolved by the web layer to rank
  and badge the current-state census (the on-screen `sevbadge` / `SeverityBadge`). Severity
  ranks *rules*, not findings, and is never the source of urgency — there is no per-finding
  dial to add, and a port you can hide is a signal you can silence.
- **A rule that ships at release cadence.** A condition qualifies as a signal only if
  its reference data changes at release cadence. Anything you would want to push
  updates to *out of band* — a growing corpus of indicators — is a signature database,
  and it is deliberately out of scope. Four v1 signals read a small piece of
  release-coupled reference data — a port list, a key-and-algorithm table, an expiry
  horizon, and a set of address ranges (each noted below); the other thirteen read none.
- **Pure and attributable.** A signal is a pure function of its inputs and its rule.
  Hold the rule version constant and any change in the signal set is attributable to
  the world. Across a rule-version change the two sets are **not compared at all**, so
  a rule edit never masquerades as drift in your estate.

### The four verdicts

Every rule returns one of four outcomes for each subject. These are the **engine's**
per-subject verdicts, evaluated data-side. The **Signals** screen itself does not paint a
per-rule census: it renders a **flat per-instance table** — one severity-badged row per
currently-**fired** `(rule, subject)` pair, with a stable `SIG-####` id, filters, sort and
a row drawer (P2.2; the design package is normative for functionality). So `not-fired`,
`not-evaluable` and `outside-domain` shape the evaluation but are **not rows you see** on
the screen; they are how a rule decides which instances fire.

| Verdict | Meaning |
| --- | --- |
| **fired** | The condition holds for this subject. |
| **not-fired** | The condition was evaluated and does not hold. |
| **not-evaluable** | The rule could not decide — the evidence is about *our own sight* (a `Shadowed` value) or there is no value at all (a `Gap`). Never counted as "did not fire." |
| **outside-domain** | The rule was never about this subject (e.g. a CNAME rule over a name that has no CNAME). Not rendered at all, so the "did not fire" column never swells with subjects the rule does not concern. |

The flat table is a **current-state** picture — the `(rule, subject)` pairs firing
*now*, one row each — never a delta. (The rule censuses are still evaluated data-side to
mint those rows, but they no longer paint a three-column grouping.) The change surface
lives in **Messages**.

---

## Name signals

These read the resolution facets (`resolution-walk`, `wildcard-discrimination`) and are
about a `Name`.

### `lame-delegation`
Fires when a Name's delegated nameservers were all reached and **none serves the
zone** — a lame delegation. `not-evaluable` while resolution is `Shadowed` or a `Gap`.
This is one of two signals that also alert on *clearing*: a delegation that stops being
lame may be an attacker claiming the orphaned name, so the change is reported as *this
changed*, never *resolved*.

### `cname-target-name-error`
Fires when a Name holds a `CNAME` whose **target does not exist** (`NameError`) — the
dangling-CNAME setup behind classic subdomain takeover. `outside-domain` for names with
no CNAME; `not-evaluable` where the name or its target is `Shadowed` or unreadable. Also
alerts on clearing, for the same takeover reason as `lame-delegation`.

### `zone-declared-name-returns-name-error`
Fires when your **zone file declares a name** that your resolver says **does not exist**
(`NameError`) — the zone promises a record the world cannot see. `outside-domain` for
names your zone does not declare; `not-evaluable` on `Lame`/`Shadowed`/`Gap`. Requires an
uploaded [zone file](zone-files.md).

### `resolved-name-absent-from-zone`
Fires when a name **resolves inside your declared zone** but your **zone file does not
contain it** — a live name the authoritative export omits. `outside-domain` outside a
declared zone; `not-evaluable` when `Shadowed`. Requires an uploaded zone file.

### `non-globally-reachable-address-resolved-from-internet`
Fires when an **internet-class** resolution returns an address that is **not globally
reachable** (loopback, link-local, RFC 1918 / ULA private space, or an IANA
special-purpose range) — an internal address leaking into a public DNS answer. This is a
**vantage-scoped** signal ([ADR-0071](../adr/0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md)):
it is read only at the internet vantage and has no internal twin. `outside-domain`
without an internet-class answer; `not-evaluable` when that answer is `Shadowed`. Its
notion of *globally reachable* is **release-coupled reference data** — the IANA
special-purpose address ranges (loopback, link-local, RFC 1918 / ULA, and the
special-purpose registries), applied as fixed classification.

---

## Endpoint signals

These are about an `Endpoint` and read its `certificate` and `http-identity` facets.
Their evidence already presupposes something reached the endpoint, so — unlike the
Service exposure signal — they carry no vantage gate.

The five certificate-detail rules all fire over an endpoint whose certificate was
**presented**, and return `not-evaluable` when the certificate was presented but its
attributes could not be read:

### `certificate-expired`
Fires when the presented certificate is **past `not_after`**.

### `certificate-not-yet-valid`
Fires when the presented certificate's **`not_before` is in the future**.

### `certificate-expiring`
Fires when the presented certificate is **within the expiry horizon `N`**. `N` is a
release-fixed parameter, not an operator dial: one third of the certificate's validity
period, and one half where that period is 10 days or less. It is a **curated**,
release-coupled value.

### `certificate-self-signed`
Fires when the presented certificate is **self-signed**.

### `certificate-weak-key-or-signature`
Fires when the presented certificate uses a **weak key size or a deprecated signature
algorithm**. The key-size floor and deprecated-algorithm set are a **curated table**
(see [`docs/research/weak-key-and-signature.md`](../research/weak-key-and-signature.md)).

### `certificate-hostname-san-mismatch`
Fires when the presented chain's **SANs do not cover the endpoint's Name**.
`outside-domain` for a nameless endpoint; `not-evaluable` when the certificate details
are unreadable.

### `plaintext-http-no-https`
Fires when an endpoint that **responded to HTTP presents no TLS anywhere** — plaintext
with no HTTPS counterpart. `outside-domain` where HTTP did not respond; `not-evaluable`
where TLS was not measured. (Not gated on internet reach — gating it would smuggle
severity back in as evaluability.)

### `redirect-does-not-upgrade-to-tls`
Fires when an HTTP **3xx redirect's `Location` is not `https`** (a relative Location
keeps the plaintext scheme and fires). `outside-domain` for non-redirect responses.

### `redirect-to-host-outside-estate`
Fires when an HTTP **3xx redirect points to a host outside your estate**. A relative
Location (no host) does not fire. `outside-domain` for non-redirect responses.

### `unauthenticated-request-answered`
Fires when an **unauthenticated `GET /` is answered with a 2xx** rather than challenged.
A `401`/`403` is not-fired (correctly challenged); any other status is `outside-domain`.

---

## Service signals

These are about a `Service` (a `(port, transport)` on an address).

### `tls-1.0-accepted`
Fires when a Service that **completed a TLS handshake accepts TLS 1.0**. `outside-domain`
for services that completed no handshake; `not-evaluable` when the handshake completed
but the accepted versions could not be read. Reads the `tls-acceptance` facet.

### `sensitive-port-reached-from-internet`
Fires when a Service on **verge-core's sensitive-port list is `reached` from the
internet vantage**. This is the product's flagship signal and the **only** v1 signal
that reads `Exposure` — specifically the internet `Reach` leg. `outside-domain` for ports
not on the sensitive list; `not-evaluable` where there is **no internet-class value**
(no prober, or a `Gap`), because a claim about internet reach with nobody looking from
outside is not a not-fired — it is unanswerable. The sensitive-port list is a **curated
table**. See [first-run.md → Exposure needs two legs](first-run.md#exposure-needs-two-legs).

---

## Why a signal can read `not-evaluable`

`not-evaluable` is not an error — it is the system being honest that it *cannot
construct the claim* rather than guessing. A certificate rule over an endpoint whose
certificate was `Shadowed`, or the internet-exposure signal on an install with no
prober, both read `not-evaluable`. This is the same discipline as `Coverage`: reading
where the system stops is as much the job as reading where it fires. See
[using.md → Reading what it found](using.md#reading-what-it-found).
