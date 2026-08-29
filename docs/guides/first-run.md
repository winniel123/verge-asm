---
title: First-run mental model
section: Getting started
order: 2
description: The mental model behind a first run — the three layers, why Exposure needs a second host, and how to tell a real scan from a silent failure.
---

# First-run mental model

The questions a first-time operator actually asks, answered in one place. Read this
guide alongside the checklist in [using.md](using.md). This guide is the *why* behind the
steps. It also covers the two things that most often confuse a first run. The first is why
`Exposure` needs a second host. The second is how to tell a scan that ran from one that
failed in silence.

---

## The three layers: declared, observed, derived

Everything in verge-asm is one of three things. Keeping them apart explains most of the
product:

- **Declared** is your input — seeds, exclusions, scan cadences, vantages, source
  toggles. It never drifts. It always has an author in the audit trail.
- **Observed** is what a scan measured — a resolution answer, a TCP connect outcome, a
  certificate.
- **Derived** is what the system concluded from observations — `Reach`, `Exposure`, a
  signal. Two derived values are comparable **only** within one identical derivation. The
  system enforces that with a `Break`. It does not trust to discipline.

When the honest answer is *we don't know yet*, the product says so in `Coverage` instead
of inventing certainty. Reading Coverage is as much the job as reading Exposure.

---

## Vantage class, and why `Exposure` needs two legs

A **vantage** is a network position you observe from. Its **class** is which side of
*your* boundary it is on — `internet` or `internal`. This one axis makes `Exposure` a
conclusion rather than a single reading.

### Exposure needs two legs

The system composes `Exposure` from two `Reach` legs. One leg is what an
**internet**-class vantage found. The other is what an **internal**-class vantage found.
`Exposure` **exists only where both legs hold a value**
([ADR-0017](../adr/0017-exposure-needs-both-legs.md)). The four states are:

| | internet `reached` | internet `not-reached` |
| --- | --- | --- |
| **internal `reached`** | `exposed` | `firewalled` |
| **internal `not-reached`** | `edge-only` | `unreachable` |

A **one-legged reading is not an `Exposure` at all.** With only an internal leg, the
system renders that leg's `Reach` on its own. It states plainly *we never looked from
outside*. It will **never** report `firewalled` or `exposed` for something it did not
observe from the internet.

### Why that needs a second host

An internet-class vantage exists **exactly where a second host observed this instance's
presented address**. So `Exposure` requires a **prober**, unconditionally. A single
all-in-one install can build a complete, honest *internal* reachability inventory. But it
**cannot see its own estate from the internet**. Probing your own public address from
inside hairpins. It never traverses the inbound policy. So the reading would be a trap,
not a measurement.

This is the trap behind a common first-run setup: **deploying the instance and the
prober both outside your network gives you only the internet leg.** Two outside
observers are still one side of the boundary. To get the *internal* leg, you need a
vantage **inside** your network. To have the instance's own vantage verify as `internal`,
declare an address scope. That scope must cover its presented address (in practice a `/32`
or `/128`).
Provisioning a prober declares *this vantage is on the internet*. Declaring that address
scope declares *this one is inside my boundary*. There is no enum to set. The acts are the
declaration. See
[using.md → Add an internet vantage](using.md#3-add-an-internet-vantage-provision-a-prober)
and [prober.md](prober.md).

> Adding the first internet vantage does not make your estate appear to escalate
> overnight. It **opens** the `Exposure` timelines (recorded as `revealed`, one
> coverage-class message). It does not transition every service to `exposed`.

---

## Confirm a scan actually ran

A scan can commit as `completed` and still measure nothing. The most common cause is a
`dns` scan pointed at a resolver that answers nothing. That scan yields empty records and
a `Gap`. It **still commits successfully**. So "the command exited 0" is not the same as
"it produced data." Check, in order:

1. **The worker logs.** A triggered scan drains synchronously. It logs how many jobs it
   enqueued and drained:

   ```sh
   docker compose logs worker | tail -n 40
   ```

   The dispatcher, delivery, and retention status all surface here too.

2. **Coverage.** The **Coverage** page is where *we could not construct this claim*
   lives — `Gap`s, unread apertures, unevaluable rules. A scan that ran but resolved
   nothing appears here as a `Gap`. It is not an error and not absent data. If you
   expected subjects and Coverage shows a `Gap`, suspect the resolver or an empty scope.
   Suspect a crash last.

3. **Subjects.** Once a batch genuinely commits data, the `Name`s, `Address`es,
   `Service`s and `Endpoint`s appear under **Subjects**, each drilling into its facet
   timelines.

The single setting most likely to cause a silent empty `dns` scan is the `local`
vantage's resolver on an off-compose install. See
[using.md → Run the first batch](using.md#4-run-the-first-batch).

---

## Where results appear

The full page tour is in
[using.md → Reading what it found](using.md#reading-what-it-found). Two pages matter
most on a first run:

- **Coverage** is where *we could not construct this claim* lives — `Gap`s, unread
  apertures, unevaluable rules. On a first run it is the page that tells you a scan
  ran-but-found-nothing rather than crashed. Read it before you conclude the estate is
  empty.
- **Exposure** only populates once you have **both** an internal and an internet
  vantage. Until then the surviving leg's `Reach` renders on its own (see above).

Three areas hold the rest:

- The estate itself — Names, Addresses, Services, Endpoints — appears under **Subjects**.
- Rule firings appear under **[Signals](signals.md)**.
- Discovery toggles appear under **[Sources](sources.md)**.

---

## Caveat: scanning a CDN-fronted domain measures the edge, not your origin

If a name you scan resolves to a **CDN, anycast, or reverse-proxy edge** (Cloudflare,
Fastly, and the like), then probing its **resolved IPs measures that edge — not your
origin.** These edges complete the TCP handshake on effectively every port. So a `hot`
(or `cold`) port scan reports the whole range as `reached`. The "surface" you see is
the CDN's front door, not the machine that actually runs your service. The numbers are
real. They are just about the wrong host.

To measure your real surface, **declare your origin IPs as an address scope.** An address
scope enumerates. The system walks every address in it directly, bypassing the resolved
edge. So the `Reach` you get is your origin's.

Mind the interaction with the **custody extension**. Enabling it over a *name* scope
pulls the addresses those names resolve to in-boundary. It then probes them fully. Over a
CDN-fronted name, that means you probe the third-party or CDN edge IPs as if they were
yours. This is exactly the confusion above, now inside your declared boundary. Prefer an
**address scope** naming your origin over a custody extension on a CDN-fronted name.

> This is the cheapest interim mitigation for CDN-fronted domains.
> [#247](https://github.com/winniel123/verge-asm/issues/247) tracks the deeper fix.

---

A first run can leave you with an empty `Exposure`, nothing delivered, a scan you cannot
confirm ran, or a `Gap` you did not expect. For the symptom-to-fix walkthrough, see
**[troubleshooting.md](troubleshooting.md)**.
