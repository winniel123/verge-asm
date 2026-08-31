# ADR-0129: A shared foreign proxy edge is measured by fan-out, never read from a provider list

- **Status:** Accepted
- **Date:** 2026-08-31
- **Ticket:** [#943 List vs measurement: how verge-asm knows an address is a foreign shared edge](https://github.com/winniel123/verge-asm/issues/943)
- **Map:** [#936 Map: suppress scanning of foreign shared proxy edges](https://github.com/winniel123/verge-asm/issues/936)
- **Amends:** [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md)

## Context

A `custody extension` on a name scope opens the probing gate over the addresses that
scope's names resolve to ([ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md)).
Transitivity stops where the resolution chain leaves the declared zone: a direct A record
extends, a CNAME to a foreign name does not.

The narrow gap: a `Name` in the operator's own zone with a **direct A record** (or an
`ALIAS`/`ANAME` flattened to A at the apex) pointing at a **shared** third-party edge —
Cloudflare, Fastly, CloudFront, Akamai. The chain never leaves the zone, so the label-suffix
test passes and the extension covers the edge `Address`. That address then enters the estate
as a `Subject` and queues probes against infrastructure titled to a provider and shared across
thousands of unrelated tenants. The scan wastes resources and measures the provider, not the
operator.

Two prior rulings constrain any fix:

- **The boundary is measured, never read from a list of providers.** `CONTEXT.md` states this
  twice — for `Custody` (this ADR's parent) and for `Vantage class` ("no registry file may
  decide which side of the boundary a prober is on, and no `custody extension` either"). A
  published provider IP-range list is exactly that ruled-out list.
- **ADR-0013 §3 already rejected a shared/dedicated test.** Its rejected-alternatives table
  refuses "a test for whether an address is *dedicated* rather than *shared*", because every
  version was "an invented threshold inside the safety path" or "a list of hyperscaler ranges".

Two research tickets fed this decision.
[#940](https://github.com/winniel123/verge-asm/issues/940) found that published edge ranges
exist for some providers but not cleanly for all: Akamai is authenticated-only
(`operator-credentialed`), Fastly and Google expose no edge-only subset, and no list grants a
redistribution licence. [#941](https://github.com/winniel123/verge-asm/issues/941) found that
**shared-ness is measurable without a provider list**, anchored on one signal.

## Decision

### 1. The discriminator is fan-out, a measurement

An address is a **shared foreign edge** when one edge IP presents identities for **many
unrelated registrable domains**. The measurement, per
[#941](https://github.com/winniel123/verge-asm/issues/941):

1. Collect the hostnames the IP presents — from Certificate Transparency SANs and from an
   active no-SNI TLS handshake that returns the edge's default certificate.
2. Reduce each hostname to its registrable domain (eTLD+1) with the **Public Suffix List**.
   The PSL is a registry-suffix dataset the system consumes to compute a boundary, not a list
   of providers.
3. Count the distinct, unrelated registrable domains. A high count is a direct observation of
   shared-ness.

No provider list gates the decision. Ownership signals (RDAP/ASN), certificate brand strings
(`sni.cloudflaressl.com` and the like), and PTR patterns are **enrichment or human-readable
labelling only**, with zero weight on the boundary decision. Each of them, used as the
discriminator, degenerates into "read the boundary from a provider list" — the exact stance
this project refuses. Ownership in particular is orthogonal to shared-ness: a cloud-resident
estate is titled to third parties by design, so "this IP is titled to Cloudflare" does not
mean "not the operator's".

### 2. It escapes ADR-0013 §3 because the failure direction is reversed

ADR-0013 §3 rejected a shared/dedicated test that would sit in the **gate-opening** safety
path. There, a false "dedicated" reading opens the probing gate onto a stranger's machine —
the silent, dangerous direction.

Shared-edge suppression runs the **other way**. It **refuses to extend custody**, so it
**withholds** a probe. Its false positive — the system wrongly reads the operator's own edge
as shared and declines to probe it — is a **coverage question that surfaces loudly** as a
`Gap`, which the model already makes visible. This is the project's own rule applied to itself:
"a false `exposed` is investigated, a false quiet reading is not" — and here the quiet reading
is the one that surfaces, never one that hides. The threshold is therefore not in the safety
path ADR-0013 forbade.

The residual risk is stated rather than smoothed: an operator's own dedicated edge misread as
shared is a real miss. Two things bound it. The fan-out count is low for a genuine single
estate, which the graded threshold ([#955](https://github.com/winniel123/verge-asm/issues/955)
calibration ticket) is set to respect, and the miss surfaces as coverage, never as silence.

### 3. `shared-edge` is a Derived property, a second Observed input to `Custody`

The determination is Derived. The fan-out **count** is Observed. The **threshold** that turns
a count into "shared" is a **declared parameter of the `Custody` derivation**, versioned per
[ADR-0008](./0008-derivation-versions-move-on-content.md). Moving the threshold is therefore a
`Break` — a change in the observer — never drift.

This satisfies the [#55](https://github.com/winniel123/verge-asm/issues/55) amendment to
ADR-0013, which was careful that the custody coverage message "contains no number the product
chose". The product's number lives in the versioned derivation — outside the operator's dials,
outside the census payload. The message still reports only the boundary: which addresses the
extension stopped covering, each with the name that cites it.

`Custody` today has two Declared inputs (address scopes, the extension declaration) and one
Observed input (the resolutions of names in an extending scope). This adds a **second Observed
input** to the same gate. `shared-edge` is a property of an `Address`. It is **not** a new
`Seed` kind (§3 of ADR-0013 already refused a third kind) and **not** a new `Subject`.

### 4. Suppression refuses to extend custody — the exact placement is [#944](https://github.com/winniel123/verge-asm/issues/944)

The expected shape is: the extension refuses to extend onto a measured shared foreign edge, so
that address never becomes a `Subject` and never queues a probe — ADR-0013 §5's gate-narrowing
shape. **Where** the suppression acts, and **whether the operator is still told** their name
fronts a shared edge, is [#944](https://github.com/winniel123/verge-asm/issues/944), still
open. This ADR does not settle it.

### 5. v1 ships fan-out alone

Fan-out is keyless and first-run capable on the single-vantage modal install. Anycast
(Signal 2 in [#941](https://github.com/winniel123/verge-asm/issues/941)) needs geographically
dispersed vantages the modal self-hosted install does not have, and it is a corroborator, never
the discriminator. It is **out of scope** for this effort. The enrichment signals in §1 stay
labelling-only.

### 6. Collection is CT plus an active no-SNI handshake

Certificate Transparency records which certificate carries names X, Y, Z — **not which IP
serves it**. The whole determination is about an `Address`, so a SAN bundle must be bound to
the edge IP by an active probe. One no-SNI TLS handshake per candidate returns the default
certificate and supplies that binding, and its SAN set is itself a fan-out sample that CT
misses on SNI-required edges. HTTP `Host` probing is a lower-priority confirmation for names
already in hand.

This active no-SNI handshake is a **new measurement**. It carries
[ADR-0011](./0011-a-facet-is-six-parts.md)'s six obligations and needs a home `Scan`, a
cadence, and a currency bound ([ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md),
[ADR-0044](./0044-a-one-off-measurement-has-no-currency.md)). That pipeline is a scope decision
of its own, carried by [#954](https://github.com/winniel123/verge-asm/issues/954).

## Consequences

- **The modal cloud-resident install stops probing shared provider edges** its own names front,
  without the operator maintaining any list.
- **`Vantage class` is untouched.** ADR-0013 §6 reads literal address-scope `Seed`s only, never
  the extension. This refinement modifies the extension, so it cannot move which side of the
  boundary a prober sits on.
- **`Custody` gains a second Observed input,** so a `Custody` value can now move because a
  fan-out measurement moved. Every such move is co-caused by an `Address` membership change or
  a resolution change the model already surfaces, as ADR-0013 §5 established for the first
  Observed input.
- **The measurement inherits [#941](https://github.com/winniel123/verge-asm/issues/941)'s stated
  limits.** Encrypted ClientHello (ECH) removes the plaintext SNI and blunts active enumeration;
  passive CT is unaffected. Dedicated per-tenant certificates emit no shared SAN bundle. These
  are false negatives — the system fails to suppress and probes an edge it could have skipped —
  which is the loud, wasteful direction, not the silent one.
- **A number the product chose now affects a gate.** It is confined to the versioned
  `Custody` derivation, so it moves as a `Break`, never as drift, and never reaches the message
  or the census.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Consume a published provider-range list** as the discriminator | Reads the boundary from a list of providers — refused twice in `CONTEXT.md`. It also fails on coverage: Akamai is authenticated-only, Fastly and Google expose no clean edge-only subset ([#940](https://github.com/winniel123/verge-asm/issues/940)), and no list grants a redistribution licence |
| **Hybrid — measure, with a list as a corroborating hint** | The moment a list nudges the boundary decision it is the ruled-out list, at reduced weight. A list may only label a finding for a human, never move the gate |
| **Ship anycast corroboration in v1** | Needs distributed vantages the modal single-vantage install lacks. It is a corroborator, never the discriminator, so dropping it costs no correctness |
| **A new `Seed` kind or a new `Subject`** | Nothing here needs a new key or lifecycle. `shared-edge` is a Derived property of an existing `Address`, and the mechanism reuses ADR-0013 §5's gate-narrowing shape |
