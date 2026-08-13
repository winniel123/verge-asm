# verge-asm

Self-hosted attack surface management for an operator's own estate. Its subject is not
inventory but **change**: what is exposed to the internet, and what moved since last time.

The language divides into three layers, and the division is load-bearing:

| Layer | What it holds | Drifts? |
| --- | --- | --- |
| **Declared** | What the operator tells us | No — it is input |
| **Observed** | What we measured | It is what drift is *made of* — but never compared directly |
| **Derived** | What we concluded | **Only across an identical derivation** |

A term's layer determines *on what condition* it may appear in a change report. Anything
Derived is a function of our own rules or thresholds, so comparing two Derived values
produced by *different* derivations reports a change in the observer as a change in the
world.

The rule is therefore not *never diff Derived* but **never compare across differing
derivations** — and it is enforced by a `Break` rather than left to discipline. Observations
are never compared to each other either; they are folded into `Span`s, and every `Span` is
Derived, because even one folded straight from observations composes a canonicaliser version
and a staleness bound. See [ADR-0007](./docs/adr/0007-drift-is-a-timeline-of-spans.md). What a
derivation *is*, and what makes its version move, is
[ADR-0008](./docs/adr/0008-derivation-versions-move-on-content.md).

A fourth group, **Operational**, sits outside the table on purpose: it records what the
system *did*, never what is true of the estate. Nothing in it may be read by the
comparison path at all.

## Language

### Declared

**Seed**:
An operator's assertion of where the estate ends — either a *name scope* (a registrable
domain) or an *address scope* (a CIDR). It declares a boundary, not a starting point. A
boundary can be drawn inwards too: a seed carries **exclusions**, exact names or subtrees
the operator declares are not theirs. Excluding a name that still resolves is legal —
*not mine* is a different claim from *not there* — and an excluded name is no longer
queried. See [ADR-0006](./docs/adr/0006-subjects-leave-by-measurement.md).
_Avoid_: target, root domain, scope target

**Source**:
Anything that can produce observations, carrying three properties: **authority**
(`declared` / `measured` / `inferred`), **completeness** (`enumerable` /
`corroborative`) and **consent** (`unencumbered` / `operator-accepted` /
`operator-credentialed`). The first two say how far to believe it; the third says whether
it may run without the operator having said so.
_Avoid_: provider, feed, integration

**Authority**:
How far a source's claim of existence is believed. The operator's zone file is
`declared`, our own prober is `measured`, a certificate SAN is `inferred`. It governs
**admission** — whose word is enough to put a subject in the estate — and nothing else. It is
not a precedence ordering: sources that disagree are both reported, never ranked, or a zone
file would keep a name alive that no longer resolves. See
[ADR-0007](./docs/adr/0007-drift-is-a-timeline-of-spans.md).

**Completeness**:
Whether a source's *silence* may mean absence. An `enumerable` source returns a complete
set over a declared scope, so silence within that scope is evidence. A `corroborative`
source can only ever assert presence — certificate transparency is append-only, so a
certificate's absence from a query means nothing, however the query went. The scope need
not be one the operator declared, and may be as small as a single subject: our own
resolver is `enumerable` over one `Name`, because a Name Error is a complete answer to
whether that name exists. What the rule requires is that the `Batch` record the scope,
not that anyone else have drawn it.

**Consent**:
Whose permission a source runs on. `unencumbered` sources run by default. An
`operator-accepted` source runs only once the operator has accepted that source's terms
themselves; an `operator-credentialed` one only once the operator supplies their own
credential, under their own account's terms. The distinction is not cosmetic — the same
service can sit in two different states depending on how it is reached — and it decides
what is in the aperture, which makes it a property of the observation pipeline rather than
of the deployment. See [ADR-0003](./docs/adr/0003-third-party-source-consent-bar.md).
_Avoid_: enabled, licensed, tier

**Vantage**:
A network position observations are made from, declared as intent and re-verified every
batch rather than trusted as configuration. Declared, but carries a Derived
**Availability** — the one property in the model whose layer differs from its term's.
_Avoid_: prober location, scanner, agent

**Vantage class**:
Which side of the operator's boundary a `Vantage` sits on — `internet` or `internal` —
declared as intent and re-verified every batch against the seeds and registry ranges the
system already holds. It is what `Reach` is measured per, and therefore what makes
`Exposure` a conclusion across two classes rather than a reading from one prober.
_Avoid_: external, network position, inside/outside

**Scan**:
The operator's configured recurring intent — which scopes, which ports, which vantages,
what cadence. The configured thing, never the executed one.
_Avoid_: job, scan job

**Annotation**:
Operator opinion attached to a subject — a suppression, an accepted risk. Kept separate
from observations so that opinion is never mistaken for measurement. It never removes a
subject: that is a claim about where the estate ends, so it belongs to `Seed`.
_Avoid_: status, triage state, finding state

### Observed

**Subject**:
Anything an observation can be about. Exactly four kinds — `Name`, `Address`, `Service`,
`Endpoint` — each with its own natural key, and each with its own lifecycle except
`Address`.
_Avoid_: asset, entity, target

**Name**:
A fully-qualified domain name. Has DNS records; has no ports. The only subject whose
departure needed deciding: it leaves when our own resolver measures a Name Error from
every available vantage, never because time passed. Under a `Shadowed` answer it cannot
leave at all, and stays visibly unconfirmed until the operator supplies coverage or
excludes it — as it cannot beneath a `Lame` delegation, where there is nobody left to
return a Name Error and the names beneath hold a `Gap` rather than a value. See
[ADR-0006](./docs/adr/0006-subjects-leave-by-measurement.md).
_Avoid_: domain, subdomain, hostname, host

**Address**:
An IP address. Has ports; has no DNS records. Reached from a `Name` only through an
observed resolution, never a fixed relationship. Alone among the subjects it has no
lifecycle of its own — nothing ever observes an address's *existence* — so it is in the
estate exactly while a current resolution cites it or a `Seed` covers it.
_Avoid_: IP, host, node

**Service**:
An `(Address, port, transport)` triple — the subject reachability is measured against.
_Avoid_: port, open port, socket

**Endpoint**:
A `(Name, Service)` pair — the only key under which HTTP identity is single-valued,
because two names on one address and port legitimately serve different content.
_Avoid_: URL, site, web asset, vhost

**Observation**:
A single measured fact: at a time, from a vantage, in a batch, a source reported that a
subject had a given value for a given facet. One concept across every facet, so that
change detection is written once rather than per facet.
_Avoid_: result, record, datapoint, scan result

**Facet**:
Which aspect of a subject an observation measured — `resolution`, `dns-record`,
`reachability`, `certificate`, `http-identity`. Adding one means adding a way to compare
its values, not a new way to detect change.
_Avoid_: attribute, field, property

**Shadowed**:
The value a `resolution` observation takes when the answer matches a wildcard's measured
poison signature — neither the synthesised answer nor a failure. Recorded as a measured
value rather than discarded, because *we cannot see here* is a fact the operator needs and
the alternative manufactures drift: repoint one wildcard and every fictional name beneath
it reports a resolution change the same night. Whether a name is admitted under a wildcard
turns on its `Citation`, not on this answer — a certificate SAN survives, a guessed label
does not.
_Avoid_: unverifiable, synthetic, wildcard hit

**Lame**:
The value a `resolution` observation takes when every nameserver the parent zone delegates
a `Name` to was reached and none of them serves it — RFC 8499 §7's *lame delegation*, and
the project uses the protocol's own word rather than a security taxonomy's. It is a
**measurement of the operator's infrastructure, not a failure of ours**, which is what
separates it from a source error, and it is only available because the measurement binary
queries the delegated authorities directly: a recursive resolver's SERVFAIL cannot tell a
dead delegation from a bad upstream, and attribution by inference is the *whose fault was
it* judgement that would make this a coverage gap wearing a value's clothes. A delegation
only *partly* lame is not this value — the name still resolves, so `resolution` has not
moved — and is recorded per nameserver on `dns-record`.
_Avoid_: servfail, dangling, broken delegation, dead NS

**Certificate**:
An X.509 certificate, held as an immutable value and shared by fingerprint across every
endpoint presenting it. A certificate cannot change, so it cannot drift; what changes is
which certificate an `Endpoint` presents.
_Avoid_: cert record, TLS config

**Batch**:
One source, executed once, against one scope, from one vantage — recording the scope its
silence covers. The unit of like-against-like comparison. The recorded scope is what the
batch **completed**, never what it attempted, so a batch that failed outright covers
nothing and licenses no absence. It may be partitioned along any dimension its source
still retains completeness over, and no further.
_Avoid_: run, scan run, execution, sweep

**Citation**:
The single-hop link from a subject to the observation that introduced it. Following
citations backwards always terminates at a `Seed` or a `declared` source, which is what
makes "why is this here?" answerable for everything in the estate. It is load-bearing in
both directions: a subject whose last citation goes stale has no chain back to a `Seed`,
which withdraws it *and* closes the probing gate on it.
_Avoid_: provenance chain, lineage, discovery path

### Derived

**Ownership**:
Whether an `Address` is `owned`, `third-party`, or `unknown`, computed against seeds and
registry data. Governs what may be probed, not merely how it is displayed — see
[ADR-0002](./docs/adr/0002-ownership-gates-probing.md). It is the one Derived value whose
change carries a safety consequence, so it holds a `Span` timeline: closing the gate opens a
`Gap` beneath the address rather than withdrawing anything, and opening it reveals services
that are `revealed`, never `appeared`.
_Avoid_: in scope, authorized, mine

**Availability**:
Whether a `Vantage` is currently able to observe, concluded from its recent batch
outcomes over a fixed window rather than measured directly. A vantage that has failed
every attempt across the window is `unavailable`, which opens a `Gap` on the `Reach` of
its class — so `Exposure` that would need it is absent rather than quietly computed from
the class that still answers. Derived, though the `Vantage` it belongs to is Declared — we
never measured the vantage, we inferred it from what failed.
_Avoid_: health, status, up, reachable

**Reach**:
What vantages of one `Vantage class` found for one `Service` — `reached` or `not-reached`,
and nothing else. There is no value for *we did not look*: a `Batch` whose recorded scope
excludes the port never feeds the timeline, so the absence is a `Gap`. A named `Derivation`
leaf, so a rule may read one leg and compose that leaf alone.
_Avoid_: reachable, probe result, port state, not-checked

**Exposure**:
The reachability conclusion for a `Service`, composed from the internet `Reach` and the
internal `Reach` rather than observed by any one vantage. Its five values — `exposed`
(both `reached`), `firewalled`, `internal-only`, `edge-only`, `unreachable` — are a
**projection** of those two legs, so a rule reads a leg and never a value. The internet
leg going `not-reached` → `reached` is the move the product exists to catch, and it spans
two of the five. Reads `Availability`, and therefore composes its version. Held as a `Span`
written once under the version that produced it, never recomputed on read — a correction ships
as a new version and a `Break`, not as rewritten history. See
[ADR-0010](./docs/adr/0010-exposure-composes-two-reaches.md).
_Avoid_: open, reachable, public

**Signal**:
A named, versioned rule evaluated over observations — and over other Derived values, in
which case its version composes theirs — citing the observations that triggered it as
evidence. It has no lifecycle of its own; its lifecycle is its evidence's. Versions are
per rule, never one set-wide version, so an edit to one rule leaves the rest comparable.
A signal carries no severity: it is a named fact, and urgency belongs to the transition
that surfaced it. Evaluated where its evidence is absent it returns `not-evaluable`,
which is not the same as not firing. See
[ADR-0004](./docs/adr/0004-signals-are-release-coupled-rules.md).
_Avoid_: finding, issue, alert, vulnerability, detection, severity

**Derivation**:
The named, versioned procedure that produced a Derived value. Its version moves on an
**output-affecting change** and never because a release shipped — enforced by a golden corpus
in CI rather than by discipline, since neither a hash of the code (bumps on a refactor) nor a
hash of the parameters (silent on a behavioural fix) tracks what we care about. A `Span`
carries the **vector** of derivation versions it was produced under: one leaf per named
derivation, flattened across everything that derivation reads, with parameters held inside
their own leaf rather than beside it. Comparison is legal exactly where two vectors are equal.
See [ADR-0008](./docs/adr/0008-derivation-versions-move-on-content.md).
_Avoid_: rule version, schema version, algorithm

**Span**:
One period during which a timeline held a single value, keyed by `(subject, facet, vantage,
source)` — one timeline per source, so two sources that disagree hold two true facts rather
than forcing an arbitration. It opens, it is current, it closes; the open span is the current
state. Carries the versions it was derived under, which is what makes comparison legal. See
[ADR-0007](./docs/adr/0007-drift-is-a-timeline-of-spans.md).
_Avoid_: interval, state, record, period, snapshot

**Transition**:
The adjacency between two consecutive `Span`s on one timeline. Derived on read, never stored
— storing it alongside the spans would be a second representation of one fact. Membership
transitions distinguish three ways in: `appeared` (discovery), `returned` (a decommission
undone), and `revealed` (a widened aperture — *we* started looking, the world did not move).
_Avoid_: change, event, diff, delta

**Break**:
A boundary between two `Span`s that may not be compared, though both hold values. Two causes
only: a `Derivation` vector changed, or the aperture widened. Nothing is compared across a
break, so it emits no `Transition`, alerts nothing, and **no duration or count crosses it** —
a truncated one renders as a labelled floor, never as a bare number. What it withdraws is the
licence to reach *back across* it and not the data, so a view **clamps its horizon** to the
most recent break rather than going blank, and the current-state census still renders. Derived
on read from the two spans' vectors and never stored, which is what lets it name the leaf that
moved. Distinct from `Gap`, which is the absence of a value rather than of a licence to
compare.
_Avoid_: seam (reserved for architectural boundaries), fault, discontinuity, version bump

**Gap**:
A `Span` holding no value — the period over which we could not say. Opened by a dead-lettered
`Batch`'s empty scope, by a `Vantage` becoming `unavailable`, by evidence absent where a
`Signal` would be `not-evaluable`, and by an observation ageing past its currency bound. A gap
never withdraws a subject: ceasing to measure is not measuring absence.
_Avoid_: outage, downtime, missing, unknown, stale

**Drift**:
What a `Transition` between two `Span`s *is*. It is the product's subject and deliberately
**not a modelled thing** — there is no `Drift` object, no drift table and nothing that
accumulates state, or `Finding` returns under a new name.
_Avoid_: change set, diff, delta, drift record

### Operational

**Dispatch**:
One firing of one `Scan` at one scheduled time, holding the resulting fan-out of batches,
its progress, and the `Scan` config it was fired under. It exists for display and
operational visibility; it carries no observations, and **the comparison path must never
read it**. Batches anchor scope, timelines anchor comparison, a dispatch anchors neither.
See [ADR-0005](./docs/adr/0005-scan-execution-model.md).
_Avoid_: scan run, run, execution, job group

## Terms deliberately not used

**Asset** —
It flattens `Name` and `Address` into one thing, and they have different keys and
different lifecycles. A name repointing to a new address is one subject changing plus one
subject appearing; under a single `Asset` it becomes either a silent mutation or a false
removal. Acceptable as a collective noun in the interface ("847 assets"), never as a
modelled thing.

**Host** —
Reads as `Name` to one person and `Address` to the next, which is precisely the
distinction the model rests on.

**Finding** —
Invites a stored object that accumulates state and gets diffed. A rule outcome diffed
across a rule-set change reports every host in the estate as newly exposed the morning
after an upgrade. Use `Signal`, which is derived and never diffed.

**Discovery** / **Probe** (as nouns) —
Both are sources differing only in `authority` and `completeness`. Naming them as
separate kinds of thing re-splits the pipeline that one `Observation` exists to unify.
They remain fine as verbs.

**ScanRun** —
A scan fires many batches, and grouping them is useful for progress display, not for
comparison. Making it a domain term invites change to be defined as a function of
consecutive runs — which breaks as soon as two scans run on different cadences over
different port sets. The display need is met by `Dispatch`, which is Operational precisely
so that the grouping cannot be reached from the comparison path.
