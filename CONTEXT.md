# verge-asm

Self-hosted attack surface management for an operator's own estate. Its subject is not
inventory but **change**: what is exposed to the internet, and what moved since last time.

The language divides into three layers, and the division is load-bearing:

| Layer | What it holds | Drifts? |
| --- | --- | --- |
| **Declared** | What the operator tells us | No — it is input |
| **Observed** | What we measured | **Yes — the only layer that does** |
| **Derived** | What we concluded | No, ever |

A term's layer determines whether it may appear in a change report. Anything Derived is a
function of our own rules or thresholds, so diffing it would report a change in the
observer as a change in the world.

## Language

### Declared

**Seed**:
An operator's assertion of where the estate ends — either a *name scope* (a registrable
domain) or an *address scope* (a CIDR). It declares a boundary, not a starting point.
_Avoid_: target, root domain, scope target

**Source**:
Anything that can produce observations, carrying two properties: **authority**
(`declared` / `measured` / `inferred`) and **completeness** (`enumerable` /
`corroborative`).
_Avoid_: provider, feed, integration

**Authority**:
How far a source's claim of existence is believed. The operator's zone file is
`declared`, our own prober is `measured`, a certificate SAN is `inferred`.

**Completeness**:
Whether a source's *silence* may mean absence. An `enumerable` source returns a complete
set over a declared scope, so silence within that scope is evidence. A `corroborative`
source can only ever assert presence — certificate transparency is append-only, so a
certificate's absence from a query means nothing, however the query went.

**Vantage**:
A network position observations are made from, declared as intent and re-verified every
batch rather than trusted as configuration.
_Avoid_: prober location, scanner, agent

**Scan**:
The operator's configured recurring intent — which scopes, which ports, which vantages,
what cadence. The configured thing, never the executed one.
_Avoid_: job, scan job

**Annotation**:
Operator opinion attached to a subject — a suppression, an accepted risk. Kept separate
from observations so that opinion is never mistaken for measurement.
_Avoid_: status, triage state, finding state

### Observed

**Subject**:
Anything an observation can be about. Exactly four kinds — `Name`, `Address`, `Service`,
`Endpoint` — each with its own natural key and its own lifecycle.
_Avoid_: asset, entity, target

**Name**:
A fully-qualified domain name. Has DNS records; has no ports.
_Avoid_: domain, subdomain, hostname, host

**Address**:
An IP address. Has ports; has no DNS records. Reached from a `Name` only through an
observed resolution, never a fixed relationship.
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

**Certificate**:
An X.509 certificate, held as an immutable value and shared by fingerprint across every
endpoint presenting it. A certificate cannot change, so it cannot drift; what changes is
which certificate an `Endpoint` presents.
_Avoid_: cert record, TLS config

**Batch**:
One source, executed once, against one scope, from one vantage — recording the scope its
silence covers. The unit of like-against-like comparison.
_Avoid_: run, scan run, execution, sweep

**Citation**:
The single-hop link from a subject to the observation that introduced it. Following
citations backwards always terminates at a `Seed` or a `declared` source, which is what
makes "why is this here?" answerable for everything in the estate.
_Avoid_: provenance chain, lineage, discovery path

### Derived

**Ownership**:
Whether an `Address` is `owned`, `third-party`, or `unknown`, computed against seeds and
registry data. Governs what may be probed, not merely how it is displayed — see
[ADR-0002](./docs/adr/0002-ownership-gates-probing.md).
_Avoid_: in scope, authorized, mine

**Exposure**:
The reachability conclusion for a `Service`, computed across vantages rather than
observed by any one of them. `firewalled` → `exposed` is the transition the product
exists to catch.
_Avoid_: open, reachable, public

**Signal**:
A named, versioned rule evaluated over observations, citing the observations that
triggered it as evidence. It has no lifecycle of its own — its lifecycle is its
evidence's.
_Avoid_: finding, issue, alert, vulnerability, detection

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
different port sets.
