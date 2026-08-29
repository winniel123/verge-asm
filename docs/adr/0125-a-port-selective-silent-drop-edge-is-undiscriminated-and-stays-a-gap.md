# ADR-0125: A port-selective silent-drop edge is undiscriminated, so it stays a `Gap` — no positive signal is added

- **Status:** Accepted
- **Date:** 2026-08-29
- **Ticket:** [#833 Positive detection of port-selective provider edges (e.g. Cloudflare) — ADR needed](https://github.com/winniel123/verge-asm/issues/833)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Completes:** [ADR-0104](./0104-an-undiscriminated-reach-is-a-gap-and-a-blanket-responder-is-measured-not-listed.md) (its Thin-ground flag on the silent-drop case), and the [#778](https://github.com/winniel123/verge-asm/issues/778) line of work

## Context

`blanket-discrimination` ([ADR-0104](./0104-an-undiscriminated-reach-is-a-gap-and-a-blanket-responder-is-measured-not-listed.md),
`internal/measure/blanketdiscrim`) decides one fact about an `Address`: does it answer TCP on ports
that should be closed? It sends a control-port probe — eight random ports from the RFC 6335 dynamic
range 49152–65535 (`ports.go`), a range IANA assigns to no service — and calls the address a
**blanket responder** only where the **whole** control set completes the handshake (`Decide`
unanimity). A blanket responder's reaches fold to a `Gap` for the sixth cause: an undiscriminated
reach is never a value.

That predicate is built for the **answers-on-every-port** shape, and it cannot **positively**
identify a **port-selective** provider edge such as Cloudflare. This ticket is the design pass #833's
triage asked for. It rules the open question ADR-0104's Thin ground flagged: whether the model should
gain a distinct positive signal for a port-selective edge, or keep the incomplete-`Gap` treatment as
the whole story.

### Why a port-selective edge never reaches `VerdictBlanket`

Confirmed against #778 live evidence — 524 reaches on a Cloudflare-fronted address, all
`ReasonIncomplete` — and pinned by corpus row `B3/incomplete-probe-gap` plus the `Decide` unit cases:

- Cloudflare's edge proxies a small service set (80, 443, and a short published list) and **silently
  drops** connects to every other port. It never proxies the dynamic range the control set draws
  from.
- So every control connect **times out** rather than refusing. `controlResultOf`
  (`connectoutcome/certificate.go`) maps a timeout to `ControlIncomplete`. No control port refuses,
  no control port answers.
- `Decide` reads a set that is neither unanimously answered nor holds one refusal, and returns
  **`VerdictGap`** (the incomplete reason) — never `VerdictBlanket`. A **refusing** (RST) edge would
  instead clear to `VerdictNotBlanket`, a real reach value.

So a Cloudflare-fronted address always reaches inventory as an **incomplete** reach `Gap`, not a
measured-blanket `Gap`.

### What #778 already delivered

#778 loosened `inventoryProxyEdge` (`cmd/web/inventory.go`) to badge **both** the blanket reason and
the incomplete reason as a proxy edge — they carry the **same** sixth cause (`blanketdiscrim.GapCause`,
`"blanket-responder"`) — and propagates the flag to the bare `Address` row. So the "Hide proxy edge"
toggle already hides a Cloudflare-fronted address, **through the incomplete-`Gap` path**. The reach is
already `Gap`'d, so `sensitive-port-reached-from-internet` already returns `not-evaluable` and
inventory already drops the port from the open count. **The toggle is not broken. The operator value
is already delivered.** The gap #833 names is conceptual, not functional.

### The measurement that would be needed, and why it does not exist

A positive signal would have to read the **all-timeout-plus-a-reached-service** shape — answers on a
service port, drops (times out) the control ports — and call it a proxy edge. The problem is that this
shape does **not** discriminate a port-selective proxy edge from a **plain origin behind a
default-drop firewall**. Both:

- complete the handshake on their genuinely-open service ports, and
- **silently drop** (time out) a connect to the dynamic range, because a default-DROP policy is the
  common posture of a cloud security group, an iptables `-j DROP` chain, and most hardened origins.

The behaviour that made a blanket responder **measurable** — it answers on ports that should be
closed — is, by definition, **absent** in a port-selective edge: it drops those ports. There is
nothing left at the connect layer that a port-selective edge does and a default-drop origin does not.
The two are the same measurement from one vantage. Reading a positive verdict off that shape would
fabricate an attribution from an **absence** of evidence — the exact model-layer error ADR-0104
exists to refuse ("an undiscriminated answer is never a value", ADR-0066/ADR-0104 §2).

The only thing that would reliably single out **Cloudflare** — as opposed to any default-drop origin
— is out-of-band identity: a vendor CIDR/prefix list, or a heuristic that pattern-matches a
Cloudflare TLS certificate or a `Server: cloudflare` header. A prefix list is the project's standing
refusal ([ADR-0002](./0002-ownership-gates-probing.md),
[ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md),
[ADR-0089](./0089-an-instrument-supplies-the-test-never-the-premise.md)). A cert-issuer or
header heuristic is a vendor identity list wearing a measurement's clothes: it fails on an operator
running Cloudflare in "Full" mode with an origin certificate, false-negatives on the next CDN, and
asserts about the world what the connect instrument cannot measure (ADR-0089).

## Decision

**Keep the incomplete-`Gap` treatment as the whole story. Do not add a positive port-selective-edge
signal in v1.** A port-selective silent-drop edge is **undiscriminated** at the connect layer, and an
undiscriminated reach is a `Gap`, not a positive verdict. The existing `VerdictGap` /
`ControlIncomplete` path is the correct and complete v1 answer.

Five grounds, each a standing rule and not a new preference:

1. **No measurement discriminates a port-selective silent-drop edge from a default-drop origin.** The
   signature that makes a blanket responder detectable — it answers on ports that should be closed —
   is absent by construction in a port-selective edge, which drops them. From one vantage the two
   hosts produce the identical answers-on-service-ports, times-out-the-rest shape. There is no
   connect-layer measurement to build a positive signal on.

2. **A positive verdict from that shape fabricates surface from an absence.** It reads a proxy-edge
   attribution off timeouts — the incomplete case, which ADR-0104 §2 already rules is a `Gap` for the
   sixth cause. Calling it a positive detection is the model-layer damping ADR-0104 refuses, run in
   reverse: manufacturing a verdict where the measurement declined to decide.

3. **It would false-positive on the common cloud origin.** A normal origin behind a default-DROP
   security group shows exactly the all-timeout-plus-a-reached-service shape. A positive signal would
   badge that honest origin as a proxy edge, hide its genuinely-open ports from inventory, and send
   `sensitive-port-reached-from-internet` to `not-evaluable` on a real, attributable service. ADR-0104
   already flagged this exact risk in Thin ground. This ADR is its ruling.

4. **Anything that would positively single out Cloudflare is a list.** A CIDR/prefix list is the
   standing refusal (ADR-0002/0013/0089). A TLS-issuer or HTTP-header heuristic is the same refusal
   in a different artefact — a vendor identity on the critical path that ages, misses self-hosted
   proxies and Full-mode origin certs, and asserts what the instrument cannot measure (ADR-0089).

5. **#778 already delivers the operator value, and the honest wording is already shipped.** The
   incomplete-`Gap` already gaps the reach, damps the signal and inventory, badges the reach as a
   proxy edge, and feeds the "Hide proxy edge" toggle. `ReasonIncomplete` states the truth exactly —
   *"the control-port probe did not complete, so this reach could not be discriminated from a blanket
   responder"*. Upgrading that hedge to *"this is a proxy edge"* would overclaim on a fact the
   measurement did not establish.

### A corroborating signal is not foreclosed — only a deciding one

ADR-0104 leaves one door open: a list (or any out-of-band signal) *"may one day **corroborate** a
measurement, but it may not **decide** one."* This ADR keeps that door where it is. Nothing here bars
a **future** corroboration-only input — for example a cross-vantage or anycast-fleet observation that
many addresses in one scope share the identical timeout shape — from **raising confidence** in a
`Coverage` statement, provided it never **decides** a reach value and never fabricates a positive
verdict. That is a future ADR with its own measurement, not a v1 commitment. v1 refuses the
**deciding** signal, which is the one #833 asked about.

## Consequences

- **No behavioural code change.** `Decide`, `VerdictGap`, `ControlIncomplete`, and `controlResultOf`
  stand as they are. The incomplete-`Gap` path is affirmed as the intended v1 behaviour, not a
  placeholder.
- **One forward-pointer is redirected, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).**
  The comment at `connectoutcome/certificate.go` (`controlResultOf`) says positive provider-edge
  detection *"is a distinct signal for a later ADR."* Read alone and in the present tense, that
  sentence tells a future session to go build the deferred signal. It is redirected to cite this ADR
  and its refusal, so the pointer names a decision rather than an open invitation. This is a
  comment-only change with no behavioural effect.
- **The glossary records the ruling.** `CONTEXT.md`'s **Blanket responder** entry gains one sentence:
  the port-selective silent-drop variant resolves through the incomplete `Gap` and is deliberately
  **not** a separate positive signal, citing this ADR. This keeps a later session from re-opening the
  settled question (domain docs, "land glossary and ADR changes on `main`").
- **No leaf, no value, no version move.** The `reachability` value space stays the closed pair, the
  Derivation vector is unchanged, and no timeline `Break`s. This ruling **subtracts** a proposed
  feature. It adds nothing to measure.
- **The named-leaf count stays at six.** `blanket-discrimination` is unchanged. No new leaf joins it.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Add a positive `VerdictProxyEdge` value (or relax `Decide` to accept all-timeout as blanket)** | Reads a positive attribution off the incomplete case, which ADR-0104 §2 rules is a `Gap`. It conflates two **opposite** measurements — *answers on every port* and *drops every control port* — and false-positives on every default-drop origin, gapping its genuinely-open ports and sending the sharpest signal `not-evaluable` on a real service. It also widens the committed value space (ADR-0015) for a distinction a `Gap` already draws |
| **Detect Cloudflare by TLS certificate issuer or `Server`/`CF-RAY` header** | A vendor identity list wearing a measurement's clothes. It ages like a CIDR list, misses self-hosted reverse proxies and Full-mode origin certs, and asserts about the world what the connect instrument cannot measure — the exact thing ADR-0089 bars. A header a third party emits is not a measurement of the origin |
| **A published CDN/anycast prefix list as a positive detector** | The project's standing refusal (ADR-0002/0013/0089). A vendor's file on the critical path of the product's sharpest signal, which false-negatives the moment a new CDN or a customer's own nginx fronts an origin. Refused for the blanket case in ADR-0104; refused again here |
| **Upgrade `ReasonIncomplete` prose to assert "this is a proxy edge"** | Overclaims a fact the measurement did not establish. The incomplete probe could not discriminate a port-selective edge from a default-drop origin, and the honest reason already says so. Asserting the edge would be right on Cloudflare and wrong on the origin, from identical evidence |
| **Fire a distinct `Coverage` statement only for the all-timeout-plus-reached shape** | Same false-positive surface as the positive verdict — it would tell a default-drop origin's operator to "declare your origin IPs" when they already have the origin. The incomplete-`Gap` already badges the reach as a proxy edge for the toggle; a stronger standing statement needs a discriminating measurement this shape does not provide |

## Thin ground, flagged rather than smoothed

- **The default-drop-origin false positive is asserted, not measured across the population.** That a
  cloud security group and a hardened origin default to DROP is well established and is why the two
  shapes collapse, but verge-asm has not measured how often its served population sits behind a
  default-drop firewall versus a refusing one. The ruling errs toward `Gap` for that unmeasured
  fraction, which withholds a proxy-edge badge on some true edges rather than fabricating one on true
  origins — the same direction of caution as ADR-0104 and ADR-0068.
- **One vantage.** The evidence is #778's single-vantage reaches on Cloudflare-fronted addresses. A
  cross-vantage or anycast-fleet corroboration — the door §"A corroborating signal" leaves open — was
  not measured here and is not claimed. If such a signal is ever built, it must corroborate, never
  decide.
- **This ruling can be reopened by a new measurement, not by a new preference.** If a future
  connect-layer or protocol-layer measurement genuinely separates a port-selective edge from a
  default-drop origin without importing a vendor list, this ADR does not bar building on it. What it
  bars is a positive verdict read off the timeout shape alone, which discriminates nothing.
