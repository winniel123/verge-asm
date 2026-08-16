# ADR-0104: An undiscriminated reach is a `Gap`, and a blanket responder is measured, never read off a list

- **Status:** Accepted
- **Date:** 2026-08-16
- **Ticket:** [#247 CDN/anycast/proxy edges answer on all ports — overstated surface and false-positive sensitive-port signals](https://github.com/winniel123/verge-asm/issues/247)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

A CDN, anycast front, or reverse-proxy edge — Cloudflare, Fastly, and the rest — completes the TCP
handshake on **every** port before deciding what to do with the connection. A connect-scan
(`connect-outcome`, [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)) is a
`net.Dialer.DialContext("tcp", …)` that closes immediately, so against such an address **every**
`(port, transport)` in scope folds to `reached`. The measurement is correct. The **reach value is a
lie about the origin** — the address answers whether or not anything listens, so `reached` no longer
witnesses a service the way it does on an ordinary host.

**[measured]** — the ticket's repro, a real Cloudflare-fronted domain, 2026-08-16. Seed
`example.com` (name scope, custody extension on) resolves to `104.21.61.6` and `172.67.204.130`. A
`hot` scan produced **262 `Service`s — 131 ports × 2 addresses — 100% `reached`**, kubelet `10250`,
memcached `11211`, and NFS `111` among them; the `certificate` step on those same services returned
`no-tls` / `tls-refused`, i.e. **open TCP with no service behind it**. Independently confirmed it is
the edge and not a probe bug: a plain `nc` from a neutral container connected to `104.21.61.6` on
`13`, `111`, `11211`, `10250`, `9999`, and the arbitrary ephemeral port `62345`, while a TEST-NET
control host correctly refused. **Cloudflare answers TCP on all ports; a TEST-NET host answers on
none.**

Three consequences, and none is a probe defect:

- **Inventory ([#243](https://github.com/winniel123/verge-asm/issues/243)) is wildly overstated.** A
  CDN-fronted domain reads as *hundreds of open ports* that exist on no origin. The operator's real
  surface is behind the edge and was never scanned.
- **The sharpest signal in the product fires on noise.**
  `sensitive-port-reached-from-internet` ([ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md),
  `internal/signal/service.go`) reads the internet `Reach` leg. On a blanket responder **every**
  sensitive port is `reached`, so the moment an internet-vantage prober exists the signal fires on
  kubelet / memcached / NFS for every CDN-fronted address.
- **The custody extension amplifies it.** Turning the custody extension on over a name scope declares
  the resolved (Cloudflare, shared, third-party) addresses in-boundary
  ([ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md)), so they take full port + TLS
  + cert probing — measuring the CDN, not the estate.

**The model already owns the shape of this problem, one facet over.** `resolution-walk` produces a
DNS answer; `wildcard-discrimination` ([ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md),
[ADR-0068](./0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md),
[ADR-0070](./0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md))
decides whether that answer can be **trusted**, by sending a **control probe** — random labels that
should not exist — and reading `Shadowed` where the authority synthesises an answer for them. A name
whose answer is indistinguishable from that synthesis is not `Resolved`; *"an undiscriminated answer
is never a value"* ([CONTEXT.md](../../CONTEXT.md) `Gap`; ADR-0066). The blanket responder is the
**reachability twin**: a control probe of ports that should be **closed**, read for whether the
address answers regardless.

This ADR is the design pass #247's triage asked for. It rules the concept, its detection, its
consequence, and its surfacing, and splits the mechanics into `ready-for-agent` tickets.

## Decision

### 1. A **blanket responder** is measured by a control probe, never read off a prefix list

A new named `Derivation` leaf, **`blanket-discrimination`**, decides one fact about an `Address`:
does it answer TCP on ports that should be closed? The leaf sends a **control-port probe** — a
batch-generated set of ports a well-behaved origin refuses — and the address is a **blanket
responder** exactly where the control set answers.

- **The control set is generated, not curated.** It is a set of **random high ports** plus at most a
  small structured decoy or two, generated per batch, exactly as
  [ADR-0069](./0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md)
  generates control labels for DNS. A real host does not *listen* on a random ephemeral port; a
  blanket responder answers it. The set exists to **falsify port-independence**: an origin's open
  ports are port-specific, a blanket responder's are not, and the control set is what tells them
  apart. `62345` in the repro is one draw from exactly this set.
- **A published CDN/anycast prefix list is refused as the detector**, and the refusal is the
  project's standing one, not a new preference. [ADR-0002](./0002-ownership-gates-probing.md) and
  [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) bar a vendor list from
  deciding what gates or shapes probing; custody transitivity is *"measured rather than read off a
  list of providers."* [ADR-0089](./0089-an-instrument-supplies-the-test-never-the-premise.md) is
  the general form — an instrument supplies the **test**, and a claim about the world (this address
  is an edge) must be measured, not asserted by us off somebody's file. A prefix list ages,
  false-negatives on a self-hosted nginx in front of an origin, and puts a third party's artefact on
  the critical path of the product's sharpest signal. The signature measures the exact behaviour the
  false positive rests on.

`blanket-discrimination` is its own leaf and not a parameter of `connect-outcome`, for the reason
ADR-0021 makes `wildcard-discrimination` its own leaf beside `resolution-walk`: *a leaf is named for
what it decides.* `connect-outcome` decides *did the handshake complete*; `blanket-discrimination`
decides *is that completion trustworthy or a blanket artefact*. Two decisions, two leaves, two
golden corpora. **The named-leaf count moves from five to six.**

### 2. On a blanket responder, a reach is **undiscriminated, and an undiscriminated reach is a `Gap`**

Where `blanket-discrimination` finds an address is a blanket responder, that address's `Service`s
hold **no `reachability` value** — a `Gap` recording a **sixth cause**: *we could not discriminate
this connect from an address that answers on every port.* Not `reached`, not `not-reached`. This is
the reachability sentence of ADR-0066's rule, verbatim in shape: **an undiscriminated answer is
never a value.**

It follows for **every** port on the address, including `443`, and that is the point: from this
vantage we cannot tell a real origin service behind the edge from the edge answering for it. A
`reached` we cannot attribute to a listener is a presence we never discriminated, and recording it
as `reached` is the reachability form of the fictional inventory ADR-0066 exists to refuse.

Three riders fix what this is and is not.

- **`reachability`'s value space is untouched.** It stays the closed pair `reached` / `not-reached`
  ([CONTEXT.md](../../CONTEXT.md) `Reach`). We add **no** third value. The undiscriminated case is
  the **absence** of a value — a `Gap` — not a new member of the union, so
  [ADR-0015](./0015-the-value-space-is-the-commitment.md)'s value-space commitment does not widen and
  `Exposure`'s 2×2 projection is not disturbed.
- **It is a `Gap` and not a `Shadowed`-style value, and the reason is a real disanalogy.** `Shadowed`
  is a *value* rather than a `Gap` because it must **withdraw**: a name whose answer is synthesised
  cites no `Address`, so `Shadowed` carries none and every `Address` held only by that citation
  leaves the estate — an **affirmative membership decision**
  ([ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md)), and a `Gap`
  *never withdraws a subject*. A blanketed reach withdraws **nothing**: the `Address` is Cloudflare's
  and is cited by a current resolution (or covered by the operator's own scope), so it is in the
  estate on its own ground, and its `Service`s exist open-or-closed regardless
  ([CONTEXT.md](../../CONTEXT.md) `Service`). The reach case needs a `Gap`'s *absence of a value*,
  and needs none of `Shadowed`'s withdrawing power — so it takes the `Gap`, and
  `blanket-discrimination` is **not** in the membership vector.
- **This is a measurement, decided by the binary inside one batch** — the control-port probe and the
  service's own connect are both in it, the population is computable at batch start, and the leaf
  decides ([ADR-0011](./0011-a-facet-is-six-parts.md)). It is not assembled afterward from two
  observations, and it is not a read-layer re-frame. Where the control probe **did not complete**,
  the reach is a `Gap` for the same sixth cause — we could not run the discrimination — exactly as an
  incomplete wildcard control probe yields a `Gap` rather than a value.

### 3. The signal and the inventory are damped **at the measurement**, never in the rule

Because a blanketed `Service`'s internet `Reach` leg is a `Gap`, `sensitive-port-reached-from-internet`
reads no internet-class value and returns **`not-evaluable`** — its existing behaviour when the leg
is absent (`internal/signal/service.go`, `HasInternetReach == false`). **No rule is narrowed.** The
signal's `Predicate domain` is untouched, its version does not move, and the census still counts the
port. This matters: narrowing a rule to make it fire less often is the model-layer damping `Drift`
refuses ([ADR-0016](./0016-an-annotation-moves-a-message-never-a-number.md),
[CONTEXT.md](../../CONTEXT.md) `Signal`). We do not tell the rule to ignore edges; we **measure that
the fact it reads is undiscriminated**, and the rule reports `not-evaluable` on its own terms.
Inventory ([#243](https://github.com/winniel123/verge-asm/issues/243)) counts an origin-reached port
from a `reachability` value; a `Gap` is not one, so a blanket responder's ports drop out of the open
count without a special case.

### 4. The blanket responder is **surfaced on `Coverage`, never silently absorbed**

Turning a blanket responder's reaches into `Gap`s must not be quiet, or the operator reads *nothing
open* where the honest statement is *we cannot see your origin from here.* Two surfacings, both on
existing carriers:

- **On the subject**, the `Service`'s reach `Gap` renders its cause in the operator's words: *this
  address answers on all ports — it is a proxy edge, not your origin.*
- **On `Coverage`**, a standing statement in the aperture register
  ([ADR-0095](./0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md)):
  *these addresses answer on all ports — a proxy edge, not your origin; declare your origin IPs as an
  address scope to measure the real surface.* This is a read surface, **not a `Transition` and not a
  new message cause**: a blanket responder is a standing property of where the estate resolves, not a
  move, and where a custody extension *gains* such an address the coverage-class message the
  extension already fires ([CONTEXT.md](../../CONTEXT.md) `Custody extension`) is the carrier. It
  ties to the #242 docs caveat and to #243's inventory framing.

### 5. The custody extension is measured on, not fenced off

A custody extension over a name scope will pull resolved Cloudflare addresses in-boundary and probe
them, and `blanket-discrimination` will find them. We **do not refuse to probe** custody-extended
addresses — that would be a vendor-list gate by another name, and custody is measured, not read off a
list. We probe them, measure that they are blanket responders, `Gap` their reaches, and tell the
operator on `Coverage` that the extension is measuring the edge and how to reach the origin. That is
the honest resolution and it is consistent with ADR-0013.

## Consequences

- **The named-leaf count moves from five to six**, and `blanket-discrimination` joins
  `connect-outcome`, `tls-handshake`, `http-exchange`, `resolution-walk`, `wildcard-discrimination`.
  [CONTEXT.md](../../CONTEXT.md) `Derivation` and [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s
  enumeration are amended; each *"still five leaves total"* row in
  [ADR-0062](./0062-a-wildcards-synthesis-is-a-fact-about-the-name-it-was-probed-under.md),
  [ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md),
  [ADR-0068](./0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md),
  [ADR-0069](./0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md) and
  [ADR-0070](./0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md)
  is struck at the clause under [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md),
  each read *"and that leaf adds none"* rather than a live count.
- **`reachability` gains a sixth `Gap` cause and no value.** [CONTEXT.md](../../CONTEXT.md) `Gap`'s
  *"every gap in v1 is one of the five above"* becomes six; `Reach` names `blanket-discrimination` in
  its Derivation vector and the undiscriminated-reach `Gap`; `Exposure` is unchanged, since a `Gap`
  leg is a case it already holds. The value space stays the closed pair, so no `reachability`
  timeline `Break`s on a new value.
- **`reachability`'s Derivation vector gains a leaf, and that `Break`s every `reachability` timeline
  once — non-vacuously.** Unlike ADR-0066, where the seventh aperture input was free because
  `resolution` had not shipped, `reachability` has shipped and its timelines exist. Composing
  `blanket-discrimination` into the vector is an output-affecting change
  ([ADR-0008](./0008-derivation-versions-move-on-content.md)), so it moves the vector and `Break`s
  the reach half of the estate. This is the **correct** price and the model's own: a `Break` clamps a
  view's horizon rather than blanking it, the current-state census still renders, and *a correction
  ships as a new version and a `Break`, not as rewritten history*
  ([CONTEXT.md](../../CONTEXT.md) `Exposure`). Stated, not smoothed.
- **The sharpest signal stops firing on noise, with no rule edit.**
  `sensitive-port-reached-from-internet` returns `not-evaluable` on a blanketed `Service` because its
  evidence is a `Gap`, and its version, domain, and census machinery are untouched.
- **`Exposure` reads a `Gap` leg on a blanket responder** and therefore yields no `Exposure` there —
  no false `exposed` — under the one-legged / silent-leg handling
  [ADR-0017](./0017-exposure-needs-both-legs.md) already specifies.
- **The `Address` gains a derived property and no table.** There is no `address` row; blanket-ness is
  derived by the binary and carried as the reach `Gap` and the `Coverage` statement, not stored as a
  column. No migration is required for the concept itself.
- **The control-port probe spends packets and rides the existing safety budget.** It is TCP connects
  against a target host, so `connect-outcome`'s per-host and global pacing
  (`SafetyProfile`) bind it exactly as they bind the port tiers; the split ticket prices the control
  set against that ceiling, as ADR-0066 priced the control labels.
- **The word *edge* is deliberately not used for this concept.** `Exposure`'s `edge-only`
  (internet-reached, internal-not; [#32](https://github.com/winniel123/verge-asm/issues/32),
  `internal/exposure`) already owns *edge*. A blanket responder is a CDN/anycast/proxy front, and the
  two meanings must not collide — the surfacing copy says *proxy edge* in prose but the modelled term
  is **blanket responder**.
- **Two `ready-for-agent` tickets split from here**, each blocked by this ADR:
  - **α — the `blanket-discrimination` leaf**: control-port set generation, the probe, the blanket
    verdict, its golden corpus, folding a blanketed address's `Service`s to a `Gap` with the sixth
    cause, and the safety accounting. It also owns the mechanical seam that `buildServiceFacts`
    (`cmd/web/signals.go`) reads the latest `reachability` **observation** today and must read the
    **span** (which carries `is_gap`) under this ruling, so a blanketed leg reads as absent.
  - **β — inventory damping and `Coverage` surfacing**: drop `Gap`'d reaches from the #243 open
    count, render the reach `Gap`'s cause on the subject page (`cmd/web/subjects.go`), and add the
    aperture-register `Coverage` statement (`cmd/web/sources.go`). The signal side is free — a `Gap`
    leg already yields `not-evaluable` — so β is inventory plus surfacing.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A published CDN/anycast prefix list as the detector** | A vendor's file on the critical path of the product's sharpest signal — ADR-0002/0013 bar a list from shaping probing, ADR-0089 requires the claim be measured not asserted by us. It ages, misses self-hosted reverse proxies, and false-negatives the moment a new CDN or a customer's own nginx fronts an origin. The measured signature reads the exact behaviour the false positive rests on and needs no list to maintain. A list may one day *corroborate* a measurement, but it may not *decide* one |
| **A new `Blanketed` value in `reachability`'s space** — the tightest `Shadowed` analog | `Shadowed` is a value because it **withdraws** fictional `Address`es (ADR-0086); a blanketed reach withdraws nothing — the address is cited by resolution and stays. So the reach case needs `Gap`'s absence-of-value and none of `Shadowed`'s membership power. And a third reach value widens the committed value space (ADR-0015) and breaks `Exposure`'s 2×2 projection, for a distinction a `Gap` already draws. More cost, less faithful |
| **Fold blanket detection into `connect-outcome` as a parameter** — keeps the leaf count at five | ADR-0021: *a leaf is named for what it decides.* `connect-outcome` decides whether the handshake completed; whether that completion is trustworthy is a **different decision** with its own control-probe aperture and its own golden corpus, exactly as `wildcard-discrimination` is separate from `resolution-walk`. Hiding it inside `connect-outcome` would make one leaf's version move on two unrelated behaviours and put the control-port population where no aperture input can be diffed |
| **Damp the signal in the rule** — exclude blanketed ports from `sensitive-port-reached-from-internet`'s domain, or make it read `not-reached` | Model-layer damping, which `Drift` refuses (ADR-0016, `Signal`). It also spreads the discrimination across every consumer of `Reach` — inventory, `Exposure`, and every future rule reading the leg — so a reader that forgets it re-opens the false positive. Discriminating at the **measurement** (ADR-0066's lesson) gives every downstream reader the honest value for free, exactly once |
| **Refuse to probe custody-extended addresses that look like a CDN** | A vendor-list gate wearing a custody hat — the thing ADR-0013 forbids. And it would silently hide the operator's real problem (they are measuring the edge) instead of surfacing it. We probe, measure the blanket, and tell them |
| **Silently absorb blanket responders into `Gap`s with no `Coverage` statement** | Turns *hundreds of open ports* into *nothing open* with no explanation, which reads as *your origin is closed* when the truth is *we cannot see your origin from here.* Route 3 of the ticket is explicit that the finding must be surfaced. The `Gap` records its cause and `Coverage` states it |
| **Treat blanket-ness as `Address` membership / withdraw the address** | The address is real and in the estate on its own ground — cited by a current resolution, or covered by the operator's scope (ADR-0047). Withdrawing it would lose a true fact (the operator *does* resolve to Cloudflare) and manufacture drift when resolution repoints. Blanket-ness is about the reach of its `Service`s, not the existence of the `Address` |

## Thin ground, flagged rather than smoothed

- **Nothing has run a control-port probe inside a batch.** Every claim here about *what the leaf
  does* is specification; what is **[measured]** is the edge's TCP behaviour against one Cloudflare
  estate and one TEST-NET control, from one vantage, on one day (the ticket repro). The signature —
  *all control ports answer ⇒ blanket* — is RFC-free and rests on the observation that a real host
  does not listen on random ephemeral ports, which is robust but untested at scale in the population
  verge-asm serves.
- **The control-set size and the blanket threshold are unset here and are the split ticket's.** How
  many random ports, whether *all* must answer or a super-majority, and how to price them against the
  200 pkt/s ceiling are ADR-0069-shaped parameter decisions the detection ticket owns. A too-small
  set risks a false blanket verdict on a host that happens to answer one draw (a honeypot, an
  intercepting middlebox); the set must falsify port-independence with margin, and that margin is
  unmeasured.
- **A partial blanket is possible and unmodelled.** An address that answers on a wide band but
  genuinely refuses some ports is neither a clean origin nor a clean blanket responder. v1 rules the
  binary — blanket or not — and a partial responder that clears the threshold reads as blanket,
  withholding true reach values on its genuinely-open ports. That is the same direction of caution as
  `Shadowed` erring toward itself (ADR-0068): a false `Gap` withholds one value, a false `reached`
  fabricates surface. Flagged, not closed.
- **The `Break` cost is real and paid once across the whole reach estate.** It is the honest price of
  a measurement correction and the model handles it, but it is not free the way ADR-0066's was, and
  anyone reading this expecting a zero-cost ruling should read that sentence again.
