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

## Amendment — [#944](https://github.com/winniel123/verge-asm/issues/944): §4's deferred placement is a veto at the extension's reach, and the operator is told by display, never by a message

§4 named the expected shape and deferred two things to
[#944](https://github.com/winniel123/verge-asm/issues/944): **where** suppression acts, and
**whether the operator is still told**. This amendment settles both. It also reconciles §3 and §4,
which described two different mechanisms.

### The suppression is a veto at the custody extension's reach

The `custody extension` computes which in-zone-cited addresses it pulls into the estate. The
fan-out measurement is a **second input to that reach computation**. An in-zone direct-A address
measured as a shared foreign edge is **excluded from the reach**, exactly as a CNAME-to-foreign
target is excluded by ADR-0013's label-suffix test.

So the edge `Address` **never becomes a `Subject`**, holds **no `Custody` value**, opens **no
`Gap`**, and queues **no probe**. It is the existing foreign-boundary behaviour, applied to one
more case. No new subject population is created, and no new `Gap` kind is created. A measured shared
edge and a CNAME-to-foreign edge reach the same resting state: the boundary stops before the edge,
and the edge is outside the estate.

### This reconciles §3 and §4

§4's *"never becomes a `Subject`"* is correct. §3's *"second Observed input to the `Custody`
derivation"* is also correct, and it acts at the **extension-reach step**: a vetoed address is a
**non-member**, never a `third-party` subject. The two sections are consistent under this reading.

The rejected reading (**Model 2**) let the edge enter as a `third-party` subject with the gate
closed. It is refused. It invents a population the model does not have — a `third-party` `Address`
that is a `Subject`. Today a resolved foreign address is a value on a `Name`'s `resolution`
timeline, never a subject of its own, and an address becomes a subject only where an address-scope
`Seed` covers it or the extension pulls it in. Model 2 also contradicts §4 and would raise a fresh
`Gap` question the veto reading never raises.

### The operator is told by display, never by a message

The veto is visible on the **§7 custody-extension panel** — the *census with no denominator*. A
companion section lists the in-zone `Name`s whose direct-A target we measured as a shared foreign
edge and therefore declined to cover, each with the citing name and the **remedy**: declare the
origin IPs as an address scope to monitor the true origin. The panel row carries **no number the
product chose** — the threshold stays in the versioned `Custody` derivation (§3).

Three reasons fix the register at **display, not notification, and not silence**:

- **Not silence.** A silent decline, resting on a measurement plus a product-chosen threshold, is
  the *fails-silently* hazard ADR-0013 refuses. An operator who asks *"why is my API's address not
  covered?"* must find the answer and the remedy.
- **Not a message.** A veto is the **safe** direction — a withheld probe. §2's own rule is *"a
  false quiet reading is not investigated"*. The ADR-0013 §7 / [#55](https://github.com/winniel123/verge-asm/issues/55)
  coverage-class message exists for the **dangerous** direction, where the gate opens onto a
  stranger with no Declared act. That safety justification does not transfer to a veto.
- **The surface is §7, not `Coverage`'s aperture statement.** A veto is an extension-boundary
  decision — which addresses the extension pulls in — which is §7's subject. A blanket responder is
  a *measured* reachability we could not attribute; a shared edge is *not probed at all*, so the
  instrument-limit framing of the aperture statement does not apply. The shared edge and the
  blanket responder share a mover (a third-party edge in front) and a remedy (declare origin IPs),
  but not a cause. They stay on separate surfaces.

### A newly-declined edge does not fire a message

Re-pointing a `Name` off a dedicated origin onto a shared edge is a real coverage loss, but the
ordinary accounting already carries it: the old dedicated `Address` **leaves by measurement**
([ADR-0006](./0006-subjects-leave-by-measurement.md)) with no `Gap`, because the name stopped citing
it. The aperture does not widen, so there is no `revealed` to carry a message. The nag test seals
it — the modal cloud install fronts everything behind a CDN, so a message on decline would fire on
exactly the population that chose this deliberately, and would train that population to ignore the
safety channel. Change stays visible on the panel, pulled and never pushed, as a CNAME-to-foreign
edge is silent today.

## Amendment — [#954](https://github.com/winniel123/verge-asm/issues/954): the fan-out measurement is a membership-deciding probe on its own `Scan`, not a six-part facet, and the SAN set is what is Observed

§6 called the no-SNI handshake *"a new measurement [that] carries [ADR-0011](./0011-a-facet-is-six-parts.md)'s
six obligations and needs a home `Scan`, a cadence, and a currency bound,"* and deferred that pipeline to
[#954](https://github.com/winniel123/verge-asm/issues/954). #954 specs it and finds two clauses of §6
wrong: it is **not a facet**, and it has **no currency bound**. §3's *"the fan-out count is Observed"* is
sharpened in the same pass. Nothing above is struck; the Decision's shape — a measured discriminator that
vetoes at the extension's reach — is untouched.

### The measurement is membership-deciding, not a facet

A facet holds a `Span` timeline on a subject ([ADR-0011](./0011-a-facet-is-six-parts.md)). The vetoed edge
is a **non-member with no subject and no timeline** (§4, the #944 amendment). So the measurement cannot be
a facet: it has nothing to hang a timeline on in the exact case it exists to decide. It is instead the
**`wildcard-discrimination` shape** — a measurement the binary makes to *decide membership*, recorded on
its `Batch` **by content**, composed into the `Custody` derivation. It carries none of ADR-0011's six
parts, because it is not a facet, and it opens no timeline, so it has no differ, no discriminator and no
facet's batch-scope obligation. §6's *"carries the six obligations"* is withdrawn; the pipeline it deferred
to #954 is the one specced here.

### It rides a `Scan` of its own — the seventh

The probe is an active TLS **connect** to a candidate edge IP, run **before** that address is a member. No
existing `Scan` can carry it: `dns` and `ct` are connect-free, `hot` and `tls-acceptance` run over members
only. So it takes a `Scan` of its own — the **seventh**, `edge-fanout` — on
[ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md)'s rule that a measurement needing a
cadence of its own takes a `Scan` of its own. That `Scan` schedules and gives `Coverage` a row. Because its
result feeds a derivation and holds no facet timeline, it carries **no currency bound and no withdrawal
power** — exactly as `ct` does, for the parallel reason. Its cadence ships at **daily**: the
membership-granting input (`resolution`, via `dns`) is daily, and a slower fan-out probe would leave an edge
probed as a member before the veto. It has **no vantage dimension** — fan-out is the keyless single-vantage
signal, and vantage-varying fan-out is anycast, out of scope (§5).

### Its population and gate

The `Scan`'s scope is the **custody-extension candidates alone** — the direct-A targets, and the apex
`ALIAS`/`ANAME` flattened to A, of in-zone names that pass the label-suffix test. It ships **enabled** and
**`unencumbered`**, gated by the custody-extension declaration itself: no extension means an empty scope and
no probe, a legible empty-scope state. The one handshake is a strict **subset** of the probing the extension
already authorizes, run one step earlier, and it *reduces* total probing. A pure narrowing needs no
widening-style consent dial, so this adds no `Source` toggle of its own.

### The absence rule stands in for the currency bound

The `Custody` derivation decides each candidate on the fan-out result: **shared** declines the reach
(census: *declined*, per #944); **not-shared** reaches it; **enabled but not yet measured** *holds* the
reach (census: *measurement pending*), bounded by the daily cadence; a **disabled or errored** `Scan`
reaches it, the pre-ADR-0129 behaviour. This is **hold-then-open**. It admits no direct-A edge until fan-out
clears it, so the modal all-CDN install never shows appear-then-withdraw churn, and it falls back to
reach-everything only where the feature is off. The census gains one **pending** reason beside #944's
**declined**; neither is a new membership state — the address is *not-reached* with a reason.

### §3 sharpened: the SAN set is Observed, the count is a reduction

§3 said *"the fan-out count is Observed."* More precisely, the **hostname set the edge presents** is
Observed — the no-SNI default certificate's SANs, read through the `Certificate` entity **by fingerprint**,
recorded on the fan-out `Batch` by content. The **count** is a *reduction* over that set: eTLD+1 via the
Public Suffix List, dedup, count the unrelated registrable domains. The PSL reduction, the count and the
threshold are all **versioned parameters of the `Custody` derivation**. So a PSL update or a threshold move
is a `Break`, never drift, and no product-chosen number reaches the census or the operator's dials (§3,
[#55](https://github.com/winniel123/verge-asm/issues/55)). The threshold's **value** stays
[#955](https://github.com/winniel123/verge-asm/issues/955)'s to calibrate; this amendment fixes only that
the SAN set is the Observed wire fact and everything interpretive is versioned derivation.

### CT never binds to the IP

The ticket asked how CT SANs bind to an `Address`. The standing `Certificate` rule already answers it: CT
**is not a source of that facet and cannot be**, because a log entry witnesses issuance, never presentation,
and attributing a logged certificate to an endpoint nobody watched serve it asserts a presence no scope
record can catch. So CT never binds a certificate to a serving IP. CT admits `Name`s only; the `dns` `Scan`
resolves them; a name that actively resolves to the edge is bound by measured `resolution`, not by CT. The
**no-SNI handshake is the sole channel that binds a certificate to the edge IP.** CT contributes to fan-out
only indirectly, as resolved names that corroborate — which is why the fan-out sample is deliberately small
on SNI-required and ECH edges, the loud/wasteful direction §2 accepts.

## Amendment — [#955](https://github.com/winniel123/verge-asm/issues/955): the threshold is a boolean count fixed at 100, and the single-estate false positive is bounded, never measured away

§3 made the fan-out threshold a versioned parameter of the `Custody` derivation and deferred its
value to [#955](https://github.com/winniel123/verge-asm/issues/955). This amendment sets it, and
sharpens what the count counts.

### The determination is boolean; the calibration is where the threshold sits

The reach decision has two outcomes: the extension reaches the edge, or it vetoes it (the #944
amendment). It has no third resting state — the *pending* state the
[#954](https://github.com/winniel123/verge-asm/issues/954) amendment added is a **currency** state,
not a count band. So `shared-edge` is a **boolean** determination: one threshold turns the fan-out
count into `shared` or `not-shared`, and the veto reads that boolean.

[#941](https://github.com/winniel123/verge-asm/issues/941) recommended treating the count "as a
graded signal, not a boolean". This amendment reads that advice as calibration guidance, not as a
value shape. The grading it asked for lives in **where the threshold sits**, never in a multi-valued
output the binary veto would collapse anyway. The raw fan-out count may still render on the §7 census
for a human; the value that gates the veto is the boolean.

### The count is the distinct registrable domains, with no relatedness filter

The Observed input is the **SAN set** the edge presents (the #954 amendment). The derivation reduces
it to registrable domains with the Public Suffix List and counts the **distinct eTLD+1s** — both the
reduction and the count are versioned parameters of the `Custody` derivation, per #954. "Unrelated"
means "distinct registrable domain" and nothing more. A relatedness filter — clustering eTLD+1s that
"look like one brand" — is an ownership heuristic in disguise, and §1 already refused ownership as the
discriminator. So the count carries no such filter.

### The single-estate-many-brands false positive is bounded, never measured away

One owner may legitimately front many of its own registrable domains on one **dedicated** origin IP.
Fan-out counts those domains and cannot tell them from a shared edge's tenants — a single multi-SAN
certificate produces the same count, and the only signal that would separate them is ownership, which
§1 rules out. So this false positive is **not measurable away**. Three levers bound it, none of them a
cleverer measurement:

1. **A high threshold.** A genuine single estate rarely fronts a hundred unrelated registrable domains
   on one IP ([#941](https://github.com/winniel123/verge-asm/issues/941)). The threshold sits above
   where a single estate lands.
2. **A loud surface.** A wrongly-vetoed edge surfaces as coverage on the §7 census (the #944
   amendment), never as silence. §2's safe direction holds: the miss is visible, and its remedy is
   stated.
3. **A remedy.** The operator declares the origin IPs as an address scope. The overlap between that
   declaration and the measurement is [#956](https://github.com/winniel123/verge-asm/issues/956).

The residual miss §2 stated stands, now with its bound named: the estate must reach a hundred
unrelated registrable domains on one IP before the misread occurs, and it surfaces loudly when it
does.

### The initial value is 100, an absolute integer

`shared-edge` is `true` when the count of distinct unrelated eTLD+1s is **at least 100**. The value is
an **absolute integer**, not a fraction: fan-out has no natural denominator, unlike
`certificate-expiring`'s horizon.

100 sits between the two bands [#941](https://github.com/winniel123/verge-asm/issues/941) measured. A
real shared edge presents "dozens to thousands" of unrelated registrable domains, so 100 catches the
large majority of them. A single estate "rarely fronts hundreds" on one IP, so 100 clears the
single-estate band. The choice favours the **safe direction**: a high value makes a false veto of the
operator's own edge rare, at the cost of probing small shared edges that present fewer than a hundred
identities. That cost is the loud, wasteful direction §2 already accepts, never the silent one.

The number is a first value on qualitative evidence, not a measured optimum. It moves on evidence as
the next section governs.

### What moves the version, and what pins it

The threshold is a **declared parameter of the `Custody` derivation**, governed by
[ADR-0008](./0008-derivation-versions-move-on-content.md): project-authored, fixed at the release, and
**never operator-configurable**. Moving the value — or the count function that feeds it — changes the
derivation's output, so it moves the `Custody` version as a **`Break`**, never as drift. This
satisfies §3 and the [#55](https://github.com/winniel123/verge-asm/issues/55) constraint: the number
lives inside the versioned derivation, outside the operator's dials and outside the census payload.

The move is pinned the way ADR-0008 pins every declared parameter: by a golden-corpus row whose output
the value decides. The `Custody` derivation's corpus gains **boundary rows** — an Observed SAN set that
reduces to 99 distinct unrelated eTLD+1s derives `not-shared` (the edge is reached); one that reduces
to 100 derives `shared` (the edge is vetoed). A value move with no version bump then fails the corpus's
A6 gate, exactly as the two membership leaves are pinned today. This ticket is planning only, so it
specifies the row shape and the boundary; it does not author the corpus.

The [#956](https://github.com/winniel123/verge-asm/issues/956) address-scope remedy is an operator
**act that satisfies the reach**, in the shape [ADR-0023](./0023-consent-names-the-door.md) gave
`consent`: it never moves the threshold's version. An operator cannot turn this dial, so no install can
move a `Custody` version without a release.

## Amendment — [#956](https://github.com/winniel123/verge-asm/issues/956): a literal address-scope `Seed` is disjoint from the veto, not ranked above it, and the contradiction is displayed on the scope's own census

§2 named the address-scope declaration as the remedy for an edge wrongly measured as shared, and the
[#955](https://github.com/winniel123/verge-asm/issues/955) amendment deferred *"the overlap between that
declaration and the measurement"* to [#956](https://github.com/winniel123/verge-asm/issues/956). This
amendment settles it, and the finding is that **there is no overlap to adjudicate**. The veto and the
`Seed` act on different limbs of subject membership, so no precedence rule is needed. What #956 adds is
the law that makes the disjointness legible, and a display for the one case the map's own mechanism
creates: the system now holds evidence against a declaration.

### The two mechanisms are disjoint, not ranked

[`CONTEXT.md`](../../CONTEXT.md) holds that an `Address` is a subject exactly while a current resolution
cites it **or** a `Seed` covers it. Two limbs, a disjunction. The
[#944](https://github.com/winniel123/verge-asm/issues/944) amendment put the veto at the **custody
extension's reach**, which decides which in-zone-cited addresses the extension pulls in — the resolution
limb, and nothing else. An address-scope `Seed` satisfies the **other** limb:
[ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md) makes every address inside a declared CIDR a
subject **from the declaration**, its `Citation` hopping straight to the `Seed`, before anything has been
observed about it.

So an address the operator declared is a probed subject at any fan-out count. The veto does not reach it —
not because a precedence rule prefers the declaration, but because the veto is scoped to a limb the
declaration never uses.

### The law, stated once because it is stated nowhere findable

A measurement may **narrow a Derived reach**. It may never **overrule a Declared act**.

ADR-0013 §6 is this law one level up: `Vantage class` reads literal address-scope `Seed`s only, never the
extension. [ADR-0079](./0079-authority-presupposes-denotation-a-non-globally-reachable-address-is-probed-only-inside-a-declared-realm.md)
is the same law pointing the other way: a non-globally-reachable address is probed only where a declared
address scope covers it, and a `custody extension` **may not open the gate over one**. In all three the
literal declaration is the stronger instrument, and in all three the extension is the weaker. #956 is that
law applied to the fan-out veto. It is written out here because §6's version reads as a rule about probers,
and a session reasoning about `Custody` will not find it there.

A **specificity test** was considered and rejected: the declaration wins where the scope is a `/32` naming
the address, and loses where a `/13` swallows it. Every version of it is an invented threshold inside the
safety path — the shape [#27](https://github.com/winniel123/verge-asm/issues/27) and ADR-0013 §3 already
refused — and it would make the boundary depend on a second number the product chose.

### §3 sharpened: `shared-edge` carries no weight on the `Seed` limb

§3 called `shared-edge` *"a second Observed input to `Custody`"*, and the #944 amendment narrowed that to
*"it acts at the extension-reach step."* The corollary is now explicit: on the address-scope limb
`shared-edge` carries **no weight at all**, and an address a `Seed` covers derives `operator` at any
fan-out count. Left as written, §3 reads as a global input to the gate, and that reading contradicts this
amendment directly.

### The system now holds evidence against a declaration, and it displays it

An operator may declare a CIDR containing measured shared edges — `104.16.0.0/13`, say. Enumeration walks
it, and the system probes a provider's edge **while holding a measurement that says so**. Before this map
the model tolerated a false declaration because it held nothing to the contrary: ADR-0013's *"a false
declaration is as unprevented as it ever was"* was a statement about **ignorance**, never a licence for
**silence**. The evidence now exists, and evidence held and not shown is the *fails-silently* shape
ADR-0013 §7 refuses.

The response is **display, and never a gate**. A gate here would be the veto reading the `Seed` limb, which
the two sections above refuse.

**The surface is the address scope's own membership census** — which, unlike the §7 custody-extension
census, has a `Coverage` denominator. The row states the boundary fact and an instrument that already
exists: *this scope covers N addresses; M of them present a fan-out above the threshold; exclude them from
this scope if they are not yours.* The remedy is a `Seed` **exclusion**, which
[ADR-0012](./0012-a-proposer-is-not-a-source.md) already extended to address scopes. The row carries **no
number the product chose** — the threshold stays inside the versioned `Custody` derivation (§3, the #955
amendment) — and **no verdict**. *You may have over-asserted* is the sentence ADR-0013's nag test forbids,
and it is forbidden here.

Three surfaces were refused:

| Refused surface | Why |
| --- | --- |
| `Coverage`'s **aperture statement** | [ADR-0095](./0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md) counts what the instrument **cannot report**, not what it did not look at. A declared shared edge is looked at, and the instrument reports it fine |
| The **§7 custody-extension census** | §7's subject is which addresses the **extension** pulls in, and an address-scope address is not an extension member. The #944 amendment kept the shared edge and the blanket responder on separate surfaces for this same reason |
| A **coverage-class message** | The [#55](https://github.com/winniel123/verge-asm/issues/55) message exists because *"the probing gate opens over an address with **no Declared act at all**"*. Here a Declared act stands behind it, so that justification does not transfer — and the nag test bites hardest on the operator who declared a CDN range deliberately |

### `edge-fanout` widens its population, and one of #954's two arguments does not travel

The display needs the measurement, and the [#954](https://github.com/winniel123/verge-asm/issues/954)
amendment scoped `edge-fanout` to *"the custody-extension candidates alone"*, empty until an extension is
declared. That scope now also carries **the addresses of declared address scopes**. On that population the
result is **labelling only, never membership-deciding**.

#954 shipped the `Scan` with no consent dial of its own on two clauses: the handshake is *"a strict subset
of the probing the extension already authorizes"*, and *"it reduces total probing."* The **second clause
does not hold here** — on a declared address the handshake narrows nothing and adds a connect — so it is
**withdrawn for this population**. The first clause carries the authority alone:
[ADR-0019](./0019-the-probing-gate-is-total-over-an-address.md) makes the probing gate **total over the
`Address`**, so an address a `Seed` covers is already connected to on every port, and one further handshake
asks for no authority the declaration did not already give. So there is still **no consent dial**.

Two costs, stated rather than smoothed. #954's *"empty until a custody extension is declared"* legibility is
**gone**: an install holding address scopes and no extension now runs `edge-fanout`. And the `Scan` now
serves two purposes — deciding membership on one population, labelling on the other.

### Absence on the labelling population is open-then-label

#954's absence rule is **hold-then-open**: an unmeasured candidate is held out of the reach, and the census
carries a *pending* row. It **cannot reach this population**. ADR-0047 makes a declared address a subject
from the declaration, walked every cadence *"whether or not anything has ever answered there"* — there is no
reach to withhold, so there is nothing to hold. An unmeasured declared address is **probed normally and
carries no row**, and the row appears once fan-out has measured it. That is **open-then-label**. It is
written down because a session will otherwise carry hold-then-open across by analogy, and hold a declared
subject out of its own scope's census — a *pending* row on every address of every scope on the first day,
which is noise rather than a census.

### The dual-limb address keeps its §7 row, qualified

One address may be **vetoed from the extension and covered by an address scope at once**: an in-zone name's
direct-A target that the operator has also declared. It is a probed subject by the `Seed`. §7 keeps its row
and states both limbs — *declined by the extension; covered by address scope X*. A bare *declined* is true
about the extension and reads as a contradiction to the person the census exists for. Dropping the row
breaks the #944 amendment's fixed register of **not silence**: the extension did decline, and an operator
who later withdraws that `Seed` must be able to see why the coverage vanished. The nearest rule in the model
points the same way — an `Annotation` *"never removes a row from a census it appears in"*.

### The corpus row that pins the disjointness

The #955 amendment gave the `Custody` derivation boundary rows at 99 and 100. This amendment adds one more,
and it is the strongest guard #956 leaves behind: **a `Seed`-covered address whose Observed SAN set reduces
to at least 100 distinct unrelated eTLD+1s derives `operator`, and is reached.** A future session that
"repairs" the apparent inconsistency by making the veto global fails the corpus's A6 gate at once. This
ticket is planning only, so it specifies the row and its expectation and does not author the corpus.
