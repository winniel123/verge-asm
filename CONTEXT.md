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
domain) or an *address scope* (a CIDR). It declares a boundary, not a starting point: no name is
ever generated from a name scope, and no seed is a starting **gun**, since declaring one queues a
scan rather than firing one. **An address scope is nevertheless its own complete extension, and it
enumerates.** Every address inside it is a subject from the declaration — all 2^(width−n) of them,
since exempting the network and broadcast addresses would infer a subnetting we never measure —
walked every cadence whether or not anything has ever answered there. That is what gives *no ports
responded* a subject to be a fact about, and what lets a listener appearing in declared space fire
the flagship `Reach` move instead of arriving as a line in a membership census. A **name** scope
enumerates nothing: its addresses are reached only by measured resolution, and only under a
`custody extension`. So an address scope has a `Coverage` denominator and a name scope has none,
which is the same fact twice. An address scope carries a **range size cap** — **1,024 addresses** by
default, operator-configurable, checked when the scope is declared and applied per scope rather than
to a sum — because a boundary asserting a measurement the shipped configuration cannot complete
inside its own cadence is not a boundary; custody at a larger scale belongs to a `custody extension`.
The cap is a statement about what we can measure and never about whether the claim is true, so a
false declaration is as unprevented as it ever was. **A CIDR is family-agnostic and the cap counts
addresses**: `/22` is that count's IPv4 spelling and `/118` its IPv6 one, one knob at one setting
rather than a rule about families. An address scope is held in the same form its addresses are —
family, prefix octets, length — and **containment is family-matched prefix comparison**, so the
`Custody` lookup and `Vantage class`'s re-verification are tests over addresses and never over
their spellings, which is what keeps the probing gate from turning on a rendering. So an IPv6 address scope is declarable at `/118` and longer — in
practice the `/128`, which is how a v6-only prober's own address is covered — and every prefix an
operator is actually assigned is refused by the cap, since a `/64` would take on the order of 10¹¹
years to walk. **IPv6 space is not swept and no configuration makes it sweepable**; an IPv6 estate is
reached by a name scope with a `custody extension`, which is family-agnostic and already works,
AAAA being in the shipped resolution offer. Declaring or widening an address scope is an
**aperture widening** — `revealed`, one coverage-class message at the scope carrying a count of
timelines opened, never one message per address. A
boundary can be drawn inwards too: a seed carries **exclusions** — exact names, subtrees, or
address scopes the operator declares are not theirs. Excluding a name that still resolves is legal —
*not mine* is a different claim from *not there* — and an excluded name is no longer
queried. A name scope and its name exclusions are held in the same form their names are, and
**subtree containment is label-wise suffix comparison** over the key: the candidate's labels end
with the scope's labels, compared label by label. That is the name-side twin of the address rule
above, and it is what stops `evilexample.com` reading as inside `example.com`, which a suffix test
over text would admit and the probing gate would then open on. A name's own labels end with its own
labels, so **a subtree exclusion covers the subtree's own name** as well as everything beneath it —
which is exactly where it parts from the wildcard it is mistaken for, `*.example.com` matching
`foo.example.com` and never `example.com`. The consequence is stated rather than hidden: *the names
beneath a name but not the name itself* is a set v1 has **no object for**, since the three exclusion
kinds are all it has and the remaining set is infinite. Because those three kinds are the whole of
what may be typed, **a refused declaration names a route and never takes it** — a route may move the
shipped configuration, never the operator's claim, so a form reaching a *different* set is named and
never pre-filled, converted or auto-corrected. See
[ADR-0052](./docs/adr/0052-a-declaration-refusal-names-a-route-and-never-takes-it.md).
Registry lookups **propose** seeds and never author them: a `Proposal` the operator
has not confirmed is not a `Seed`, asserts nothing, and is read by nothing, which is what
keeps a third party's file out of the probing gate. Declining one is an exclusion of this kind
rather than a suppression, since it is a claim about where the estate ends. A name scope may
additionally carry a **custody extension** — see below. See
[ADR-0002](./docs/adr/0002-ownership-gates-probing.md),
[ADR-0006](./docs/adr/0006-subjects-leave-by-measurement.md),
[ADR-0012](./docs/adr/0012-a-proposer-is-not-a-source.md),
[ADR-0047](./docs/adr/0047-an-address-scope-is-its-own-enumeration.md) and
[ADR-0049](./docs/adr/0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md).
_Avoid_: target, root domain, scope target

**Custody extension**:
A property of a **name scope** alone: the operator's declaration that the addresses its names
resolve to are inside the boundary, and therefore under their `Custody`. Off by default, declared
once on a scope the operator already authors — one act, never a queue of discovered addresses to
approve. It is what makes the model describe a **cloud-resident estate**, where the operator holds
no registry resources and every address they use is titled to somebody else. Transitivity stops
where the resolution chain **leaves the declared zone**: a direct A record extends, a CNAME to a
foreign name does not, and that boundary is measured rather than read off a list of providers. That
test is the same label-wise suffix comparison a `Seed`'s subtrees run, over the `Name` key and never
over a spelling — so this gate, like the address one, cannot turn on a rendering. Its
extension is recomputed rather than typed, which is a safety property and not a convenience — a
literal address scope over a released elastic address holds the gate open on whoever holds it
next, while an extension simply stops covering it. What it cannot see is an `ALIAS` flattened into
an A record at a zone apex, so the declaration states that case in the operator's own words and the
current extension is rendered for them to read — display, never per-address approval, and a **census
with no denominator**, since how many addresses it *ought* to cover is completeness of the estate,
which the operator is the only source for. Because both of those carriers are point-in-time and the
extension is recomputed, a live extension **gaining** an address also notifies, in the coverage
class, once per scope per cadence: it is the only place the probing gate opens with no Declared act
behind it. That message states the difference and carries no verdict — it is not a `Transition`, and
its count is read from the same computation as the census. See
[ADR-0013](./docs/adr/0013-custody-is-control-and-extends-by-declaration.md).
_Avoid_: transitive scope, auto-discovery, implicit seed, follow-the-DNS

**Source**:
Anything whose word can put a subject in the estate, carrying three properties: **authority**
(`declared` / `measured` / `inferred`), **completeness** (`enumerable` /
`corroborative`) and **consent** (`unencumbered` / `operator-accepted` /
`operator-credentialed`). The first two say how far to believe it; the third says whether
it may run without the operator having said so. Observing a facet is the usual way a source
admits a subject and **not the only one**: certificate transparency observes no facet at all —
a log entry witnesses that a certificate was issued, never that anything presented it — and it
is still a source, because `authority: inferred` is exactly the property it exercises over the
`Name`s a SAN carries. That property has a **boundary**, and it is where a SAN stops naming
anything: a **wildcard** SAN carries no `Name` at all — it denotes a set of names rather than one,
matches `foo.example.com` and never `example.com`, and is a matching construct in a *presented*
identifier rather than a domain name. So it admits none of the names beneath it, no name of its
own, and not the parent either, and a certificate whose SANs are all wildcards admits nothing —
which is a documented limit of certificate transparency rather than a hole, since a wildcard
certificate conceals an estate's names rather than disclosing them. Admitting on some of a
batch's rows and nothing on others is what `corroborative` `completeness` already absorbs. A thing
that admits **nothing at all** is not a source, however
registry-shaped it looks: it yields `Proposal`s, and only `consent` applies to it. See
[ADR-0012](./docs/adr/0012-a-proposer-is-not-a-source.md),
[ADR-0027](./docs/adr/0027-a-source-may-admit-without-observing.md) and
[ADR-0060](./docs/adr/0060-a-wildcard-san-is-a-pattern-over-names-and-admits-none-of-them.md).
_Avoid_: provider, feed, integration

**Proposal**:
A candidate address scope offered to the operator by a source that *proposes* rather than
observes — a registry path returning ranges it believes the operator holds. It is real only
once confirmed into a `Seed`, and until then it is read by nothing: it never gates probing and
never enters the estate. Declared, though the direction of travel is the reverse of every
other Declared term — this is what we tell the *operator*, and it earns its layer because its
only consumer is the operator's declaration act, and because like `Seed` it is input and does
not drift. It carries **`consent` alone**: `authority` governs admission and a proposal admits
nothing, `completeness` governs whether silence is evidence and a proposal's silence licenses
nothing. It records **which kind of record produced it** — an RIR delegation, or a compelled
reassignment written by an upstream provider — because those carry different caveats and the
operator is the one judging them. Confirming one retains it as provenance on the resulting
`Seed`; declining one is a `Seed` exclusion. Proposals are produced **only in answer to an
operator act** — expanding an address scope they declared, or searching the org-name box — never
on a cadence, so they never accumulate into a queue to be worked through. Confirming is therefore
**one scope at a time** while declining may be done over a whole lookup at once: the two acts fail
in opposite directions, so they are deliberately not symmetric. A record re-offered with different
contents is a **new** `Proposal`, never an existing one changed — it is Declared and has no
timeline. See [ADR-0012](./docs/adr/0012-a-proposer-is-not-a-source.md) and
[ADR-0022](./docs/adr/0022-confirmation-is-singular.md).
_Avoid_: suggestion, candidate range, discovered scope, pending seed

**Authority**:
How far a source's claim of existence is believed. The operator's zone file is
`declared`, our own prober is `measured`, a certificate SAN is `inferred`. It governs
**admission** — whose word is enough to put a subject in the estate — and nothing else. It is
not a precedence ordering: sources that disagree are both reported, never ranked, or a zone
file would keep a name alive that no longer resolves. Disagreeing at all takes **two
`enumerable` sources** holding current values on one timeline key: a `corroborative` source's
difference is its own staleness, two `vantage`s measure different facts and compose, and a
`Seed` observes nothing. In v1 exactly one such pair exists — the operator's zone file against
our own resolver, on `dns-record`. See
[ADR-0007](./docs/adr/0007-drift-is-a-timeline-of-spans.md) and
[ADR-0020](./docs/adr/0020-a-conflict-needs-two-enumerable-sources.md).

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
`operator-accepted` source runs only once the operator has taken on a reading the project
declined to make: we could not clear the source's terms for the modal operator and will not
read them on a stranger's behalf, so the operator — the party actually bound by them —
makes the call and bears it. It is not a certification to us that they are inside the
terms, and it covers terms that cannot be retrieved at all, which is why *we could not read
them* is a fact about our own assessment rather than a fourth value here. An
`operator-credentialed` source runs only once the source has **granted** that operator
permission, so it runs under their own terms rather than the public ones — usually an API key
they supply, but a countersigned agreement is the same thing without a token. A grant is not a
reading: where the operator adjudicates a tension they bear, the value is `operator-accepted`;
where the source actually said yes to them, it is this one. The distinction is not cosmetic —
the same service can sit in two different states depending on how it is reached, so `consent`
keys on the **instrument**, never on the registry or vendor behind it. It **names the door,
never who walked through it**: the value is authored by the project, ships in the release and
is the same for every install, so an operator's own act *satisfies* it and never moves it —
what varies per install is the consent record, or the credential in use. Of the three values
only `operator-credentialed` asserts anything about a **third party's** own conduct, so it is
honest only where the instrument **enforces** the grant and the request fails without it; an
unenforced grant carried as this value is the declared-status bar ADR-0003 rejected. An
operator may be taken at their word about themselves, and never about somebody else. Where the
instrument observes, `consent` also decides what is in the aperture; where it only *proposes*
it decides whether the request may be made and nothing more. Either way it is a property of
the observation pipeline rather than of the deployment. See
[ADR-0003](./docs/adr/0003-third-party-source-consent-bar.md),
[ADR-0018](./docs/adr/0018-a-clear-conditional-is-not-an-ambiguity.md) and
[ADR-0023](./docs/adr/0023-consent-names-the-door.md).
_Avoid_: enabled, licensed, tier

**Vantage**:
A network position observations are made from, declared as intent and re-verified every
batch rather than trusted as configuration. Declared, but carries a Derived
**Availability** — the one property in the model whose layer differs from its term's.
The **recursive resolver it resolves through is part of that position**, and therefore part of this
term's identity rather than of any leaf's parameters: a DNS answer is a function of where the query
appeared to come from, so two resolvers are two positions in the only sense an answer can tell.
**[measured]** one wildcarded name drew its addresses from two **disjoint** pools at two vantages in
one week, and one authority's per-query rotation read as **one** answer through one resolver and
**seven** through another at the same TTL. This is
[ADR-0025](./docs/adr/0025-an-offer-is-scope-only-where-the-value-enumerates-it.md)'s ruling on EDNS
Client Subnet — *a `Vantage` in an option's clothes*, belonging **in the key, never in the scope
record** — generalised at the clause to the resolver itself. Two consequences. Declaring a different
resolver is a different `Vantage`, so its timelines **open** (`revealed`) rather than `Break`ing the
estate, which is what keeps an operator's change of upstream DNS from clamping everything. And it is
why the resolver may **not** be a declared parameter: a parameter is never operator-configurable
(see `Derivation`), while choosing a resolver plainly is the operator's. What *is* authored is the
**kind** of query path — through this resolver, or direct to the delegated authorities — which is a
declared parameter of `resolution-walk` and `wildcard-discrimination`, one value per `Batch`. See
[ADR-0070](./docs/adr/0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md).
_Avoid_: prober location, scanner, agent

**Vantage class**:
Which side of the operator's boundary a `Vantage` sits on — `internet` or `internal` —
declared as intent and re-verified every batch against the **address-scope** `Seed`s the system
already holds, and against nothing else: no registry file may decide which side of the boundary a
prober is on, for the same reason none may open the probing gate, and **no `custody extension`
either**. The test is over ~~**every address the vantage holds**~~ **every address the vantage is
observed to *present*** — a narrowing, and a forced one — of either family: one uncovered
address and it verifies `internet`, which is the closed direction, because a vantage wrongly read as
`internal` moves observations onto the leg that never alerts.
**A presented address is one an outside observer saw**, and v1 has exactly two: a prober's is the
address the instance dialled, known by construction, and the instance's own is `SSH_CLIENT` as the
prober reports it — [#14](https://github.com/winniel123/verge-asm/issues/14)'s two self-contained
checks, which are now the *whole* of the set. **An interface address is not a presented address.**
Read the older wording literally and a NATed instance could verify `internal` only by declaring its
own LAN as an address scope, which [ADR-0049](./docs/adr/0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md)'s
1,024-address cap **refuses outright** for anything above a `/22` — so an operator on `10.0.0.0/8`
could never verify one, and `Exposure` would be unreachable by construction on their install.
[ADR-0049](./docs/adr/0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md)'s
**every-not-any** quantifier is untouched and runs over this set, so a **dual-stacked** prober dialled
over one family has the other family's egress unobserved — a residue narrowed rather than closed, and
disclosed rather than assumed away.
The narrowing has a consequence worth stating plainly: **an internet `Vantage` exists exactly where a
second host observed this one's presented address, so `Exposure` requires a prober, unconditionally.**
With none, the instance's address is unobserved, its class is `unverified`, and #14's disposal
applies — no exposure claims, internal-only, never `firewalled`, because we did not look. **Intent is
declared by an act rather than by a field**: provisioning a prober declares *this vantage is on the
internet*, and declaring an address scope covering the instance's presented address (in practice a
`/32` or `/128`) declares *this one is inside my boundary*. There is no `network_position` enum and
no setup prompt; both were specified in `safe-active-probing.md` §8.2 and both are struck there
([#124](https://github.com/winniel123/verge-asm/issues/124)). A **target's** family moves nothing —
the class says where the prober sits, not what it is looking at. This is the one place where two things fairly called *the operator's addresses* mean
different sets, deliberately: `Custody` may move on a resolution, and a `Vantage` whose class
moved because a DNS answer changed would shuttle observations between the two legs of `Exposure`
and manufacture drift in the flagship value. It is what `Reach` is measured per,
and therefore what makes `Exposure` a conclusion across two classes rather than a reading from
one prober. It is **not `Reach`'s alone**: a `Signal` whose fact is scoped to a vantage restricts its
`Predicate domain` to that class and carries the class in its name, which is what admits
`non-globally-reachable-address-resolved-from-internet` on `resolution` and refuses its internal twin
([ADR-0071](./docs/adr/0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md)).
It is also the axis every `Vantage composition` is cut on, and the **only** one: a reader of a
per-vantage facet either requires every class to agree or names one class and quantifies inside it,
and there is no third shape
([ADR-0080](./docs/adr/0080-a-vantage-composition-is-cross-class-or-class-scoped-and-only-one-takes-a-quantifier.md)).
It is also an **aperture input**: the first vantage of a class widens what
`Exposure`'s composition covers, and because `Exposure` needs both legs there is nothing there to break — it
**opens** the `Reach` and `Exposure` timelines, yielding `revealed` and one coverage-class message
([ADR-0029](./docs/adr/0029-an-alert-fires-on-a-leg.md)). The cost is stated rather than hidden: a vantage inside operator address space the
operator never declared verifies as `internet`, which over-reports `exposed`. That is the loud
failure and the intended one — a false `exposed` is investigated, a false quiet reading is
not — and the undeclared space surfaces as a coverage question. A prober that holds only an IPv6
address inside the operator's own network has one route and only one: a `/128` address scope naming
that address, since the extension is barred here by design. See
[ADR-0012](./docs/adr/0012-a-proposer-is-not-a-source.md),
[ADR-0013](./docs/adr/0013-custody-is-control-and-extends-by-declaration.md) §6 and
[ADR-0049](./docs/adr/0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md).
_Avoid_: external, network position, inside/outside

**Scan**:
The operator's configured recurring intent — which scopes, which ports, which vantages,
what cadence. The configured thing, never the executed one. **Not every `Scan` is a port
tier**: ~~two are, and the third is `tls-acceptance`'s weekly enumeration, whose scope is the
open `Service` population and the TLS candidate set.~~ **there are four and only two are port
tiers.** The third is `tls-acceptance`'s weekly enumeration, whose scope is the open `Service`
population and the TLS candidate set; the fourth is **`zone`**, whose scope is the name scopes
holding a supplied zone file and which has **no port list and no vantage choice at all**, the
worker reading it. A measurement that needs a cadence of its
own takes a `Scan` of its own, because a slower cadence hidden inside another `Scan` makes the
aperture a hidden field rather than a configured object. **Recurring is load-bearing and not
incidental**: currency is `k` cadences of the covering `Scan`, so a measurement with no cadence
has no currency bound, which is why there is no one-off measurement in the model and why the
widest port tier ships **configured and disabled** rather than running once unasked. That is also
why the zone file takes a `Scan`: it is a `Source` whose staleness the model must be able to
report, and its cadence is the **operator's declared re-supply interval** — their promise about how
often they will re-export, shipped at monthly. Its batches restate the file's observations **at the
operator's supply instant** rather than at our read, because re-reading unchanged bytes on a cadence
produces a current observation of a stale fact, and it is that instant the bound runs from
([#48](https://github.com/winniel123/verge-asm/issues/48)'s fourth absence register, *you stopped
telling us*). A `Scan`
whose scope list is empty is a legible state; an aperture with nothing configured behind it is
not. See [ADR-0005](./docs/adr/0005-scan-execution-model.md),
[ADR-0028](./docs/adr/0028-a-facets-cadence-is-the-cadence-of-its-exchange.md) and
[ADR-0044](./docs/adr/0044-a-one-off-measurement-has-no-currency.md).
_Avoid_: job, scan job

**Annotation**:
An operator's declaration, about one `(subject, signal-name)` pair, that a fired rule is an
**accepted risk on a thing we are still measuring**. Its whole effect is on the **message**: a
`not-fired` → `fired` `Transition` on an annotated pair is **recorded and is not a message**. It
moves no number — not a count, a timeline, a domain, a census or a subject. ~~a suppression~~ and
~~attached to a subject~~ are **superseded here, at the site that specifies them**
([ADR-0058](./docs/adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)):
both read forward as mechanisms three decisions had already refused. Kept separate from
observations so that opinion is never mistaken for measurement, and the separation is now
structural rather than a caution — **operator opinion may move a message and may never move a
number**. So an annotated pair is still measured on its cadence, still holds every `Span` it held,
still sits **inside** the rule's `Predicate domain` at the same version, and is still counted under
`fired`. Three routes are barred by name and none reopens here: it **never removes a subject** —
that is a claim about where the estate ends, so it belongs to `Seed`
([ADR-0006](./docs/adr/0006-subjects-leave-by-measurement.md)); it **never narrows a `Predicate
domain`**, since an accepted risk is a subject of which the fact is still true
([ADR-0024](./docs/adr/0024-a-rules-domain-is-the-extension-of-its-name.md)); and it **never marks
a `Span`**, which is immutable, or `Finding` returns with better provenance
([ADR-0007](./docs/adr/0007-drift-is-a-timeline-of-spans.md)). It is legal where it is because the
model already puts every operator dial **outside** every derivation, beside notification routing and
flap suppression — this is that object keyed on a subject rather than on a channel, which is the
whole reason it needs a name. Six riders. It reaches the **firing edge alone**: never `fired` →
`not-fired`, which on four rules means a name somebody else could claim, and never the coverage
class, which is *we stopped looking* and is unreachable in any case since no coverage message is
keyed on a signal name. It **never removes a row from a census it appears in**, a census being
payload that is enumerable in full — so an annotated rule opening at `fired` still rides the census
of the move above it. It **may not partition a census**: `fired` is not cut into accepted and
outstanding, or the rule acquires a second population it does not version. It carries **no timeline,
no status and no expiry** — editing one is a **new** `Annotation`, as with `Proposal`, and an expiry
would be a state moving because time passed. It is keyed on the **exact subject and never travels**,
the stated cost being that a redeploy onto a new address lapses the acceptance rather than silencing
a subject nobody chose. And it **states its reason** and does **not** appear on `Coverage`: a `Seed`
exclusion is there because it shrinks the estate, and this shrinks nothing. Its home is `Signals`.
It carries the **instant it was declared and no author** — the one Declared term holding operator
prose is still an operator dial, and every dial in the model is unattributed, so
[#127](https://github.com/winniel123/verge-asm/issues/127)'s ruling that no operator act is written
down with an actor on it holds here without exception. A date names nobody; an undated standing mute
on an object with no expiry cannot be reviewed at all.
See [ADR-0016](./docs/adr/0016-an-annotation-moves-a-message-never-a-number.md) and
[ADR-0073](./docs/adr/0073-an-operator-dial-carries-no-author-however-specific-its-target.md).
_Avoid_: status, triage state, finding state, suppression, exception, risk acceptance workflow,
author, declared by

**Channel**:
The operator's declaration of where `Message`s go — an absolute `https` URL, an optional
secret, and the subset of the three classes it receives. Declared on `Proposal`'s own layer
test: it is input, it does not drift, and nothing in the comparison path reads it. **Zero or
more, and none ships configured**; creating one is an admin act and its secret is write-only,
never rendered back. A channel is **one-way** — it carries a message out and grants no read of
anything, so it opens no second authenticated surface and leaves
[#6](https://github.com/winniel123/verge-asm/issues/6)'s bearer-token bypass shut; a **pull**
feed is the shape that would re-open it, and there is none. It carries the **message and never
the estate**: the body holds the message's own content and no rows, so what accumulates
wherever it lands is a list of what happened rather than a reconstructable copy of what the
operator has. Routing is by **class** and nothing finer, a per-rule or per-subject filter being
an operator-authored predicate over a versioned rule set. See
[ADR-0039](./docs/adr/0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md).
_Avoid_: webhook, integration, endpoint (reserved), destination, sink, subscriber

### Observed

**Subject**:
Anything an observation can be about. Exactly four kinds — `Name`, `Address`, `Service`,
`Endpoint` — each with its own natural key, and each with its own lifecycle except
`Address`. The four split on **who supplies the key**: the world supplies a `Name`'s FQDN and
an `Address`'s IP, while the model composes a `Service`'s from an `Address` already in the
estate and `verge-core`, which is ours, and an `Endpoint`'s from two subjects already in the
estate. A natural key is the **thing denoted, never the text that named it** — a rendering is
whatever printed it, and keying on one would make subject identity a function of a step that is
not part of the measurement. So the normalisation from what a source delivered to the thing it
denotes is **fixed at v1 and carries no version**: it is not a canonicaliser, it composes no
`Derivation` leaf, and it may consult only the single value being keyed. That is not uniformity
foregone but the only honest shape available, since moving one re-keys subjects rather than moving
values, and a re-key is a sentence the model has no object for — a `Break` says two values may not
be compared, not that this timeline belongs to a different row. A **composed** key holds the
subject, never its rendering. See
[ADR-0051](./docs/adr/0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md).
So a **membership message fires on a `Name` or an `Address` alone** — the other two
can bring no ground the model was not already accounting for, and their membership is another
subject's membership restated, recorded and never notified in either direction. It fires
**once, at the root of the entering sub-tree**: the entering subject whose own `Citation`
points at something already in the estate, a `Seed` or a `Batch`. That message carries the
**census** of what entered beneath it, and it is the only carrier a new subject has, since
every timeline beneath it *opens* and no alerting predicate in the product is opening-shaped.
A subject first observed under a widened aperture is not `appeared` at all — it is `revealed`,
which is what makes a first run one coverage-class message with no special case. See
[ADR-0031](./docs/adr/0031-membership-alerts-at-the-root-of-the-entering-subtree.md).
_Avoid_: asset, entity, target

**Name**:
A fully-qualified domain name. Has DNS records; has no ports. Its key is the **label sequence**,
never a spelling of it: the ordered labels, each an octet string, terminated by the root label, so
the key is absolute by construction and *fully-qualified* is a statement about it rather than about a
convention. A name on the wire is octets and not text — a sequence of length-prefixed labels the
protocol never reads as characters — so the case, the separating dots and any Unicode form are added
by whatever printed it, exactly as an address's spelling is. What folds is what the protocol itself
folds and nothing else: **ASCII case**, the 26 letters and no other octet, because DNS defines that
equivalence and preserves case on the wire while four ordinary things move it. The **trailing dot** is
neither stored nor stripped — it is the presentation format's marker for *absolute*, consumed by the
parse, and the root label it marks is in the key; inside a master file it is **load-bearing**, since a
relative owner name is completed by that file's own origin and `www.example.com` without the dot is a
different name from `www.example.com.`. An **A-label** is a label like any other and is never decoded:
`xn--` is what every measured path carries, while a **U-label** presented as text has two denotations
— a raw-octet label and an A-label — and separating them needs Unicode tables a key may not consult,
so it is **refused rather than interpreted**. A name that cannot be keyed is not a subject: it is
absent from the `Batch`'s recorded scope and writes no value and no `Gap`. It is **rendered** one way
only — labels joined by dots, no trailing dot, computed on read, the same string SNI carries — and v1
renders no U-label at all. Comparison is label-wise octet equality, and containment is label-wise
**suffix** equality, so no test in the model ever compares a name as a string. See
[ADR-0055](./docs/adr/0055-a-names-key-is-the-label-sequence-and-we-fold-only-what-the-protocol-folds.md).
A **wildcard** is not one of these, anywhere: `*.example.com` keys as a label sequence like any
other — the key function reads no octet's meaning — but it **denotes a set of names rather than a
name**, so there is no thing for a subject to be, and it is a subject from **no source**. In a
certificate it is a matching construct in a *presented* identifier and admits nothing, which is
what `authority` grades; in a zone file it is the rule an authority applies, whose effects the
model already measures as the wildcard poison signature — a measurement whose subject is the `Name`
those effects were probed **under**, never the wildcard, and one v1 holds nowhere
([ADR-0062](./docs/adr/0062-a-wildcards-synthesis-is-a-fact-about-the-name-it-was-probed-under.md)).
That name is a discriminated name's **parent**, and a parent is a label sequence the probe
constructs rather than a subject it admits, so it need not be held or exist
([ADR-0066](./docs/adr/0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md)).
The refusal takes the protocol's own test
where the protocol supplies one and the blunt one where it does not: only a leftmost label of
exactly `*` is a wildcard in DNS — `the*` and `**` are ordinary labels and ordinary names, and a
literal asterisk label is written `\042` — while in a certificate a partial wildcard is a pattern
to one client and a label to the next, so **no certificate identity containing an asterisk in any
label admits a `Name`**. See
[ADR-0060](./docs/adr/0060-a-wildcard-san-is-a-pattern-over-names-and-admits-none-of-them.md).
The only subject whose
departure needed deciding: it leaves when our own resolver measures a Name Error ~~from
every available vantage~~ **on a cross-class `Vantage composition`** — every `Vantage class` holding
a current value and agreeing — never because time passed. The struck phrase is **superseded here, at
the site that specifies it**
([ADR-0058](./docs/adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)):
read alone it excludes an unavailable vantage and concludes from the survivor, which is the opposite
of what ADR-0006 says one sentence later, and over an empty set it is vacuously **true** and
withdraws the whole estate
([ADR-0080](./docs/adr/0080-a-vantage-composition-is-cross-class-or-class-scoped-and-only-one-takes-a-quantifier.md)).
Under a `Shadowed` answer it cannot
leave at all, and stays visibly unconfirmed until the operator supplies coverage or
excludes it — as it cannot beneath a `Lame` delegation, where there is nobody left to
return a Name Error and the names beneath hold a `Gap` rather than a value. See
[ADR-0006](./docs/adr/0006-subjects-leave-by-measurement.md).
_Avoid_: domain, subdomain, hostname, host

**Address**:
An IP address. Has ports; has no DNS records. Its key is the **address**, never a spelling of it:
the family and the octets, four for IPv4 and sixteen for IPv6, compared as octets and never as a
string. The world hands us those octets — an A record is 32 bits of RDATA, an AAAA record 128, a
certificate `iPAddress` SAN an OCTET STRING — so the text is *ours*, and one address having many
legal spellings in IPv6 costs nothing. RFC 5952 §4 for IPv6 and the dotted quad for IPv4 are how it
is **rendered**, computed on read and never compared. An **IPv4-mapped** address (`::ffff:0:0/96`)
keys as the IPv4 address it represents — one subject, since that block is defined as a way of
writing an IPv4 address rather than as an address, and a listener answering there answered on IPv4.
Nothing else folds: the IPv4-compatible block contains `::` and `::1`, and NAT64, 6to4 and Teredo
addresses are real IPv6 addresses reachable only by their own paths. A textual form with two
denotations — a leading zero read as octal, a packed integer — is **refused rather than
interpreted**, and an address that cannot be keyed is not a subject: it is absent from the `Batch`'s
recorded scope and writes no value and no `Gap`. See
[ADR-0051](./docs/adr/0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md).
Reached from a `Name` only through an
observed resolution, never a fixed relationship. Alone among the subjects it has no
lifecycle of its own — nothing ever observes an address's *existence* — so it is in the
estate exactly while a current resolution cites it or a `Seed` covers it. The two limbs are
**disjunctive**, and the second is not redundant with the probing gate: an address inside a
declared address scope is a subject **from the declaration**, before anything has been observed
about it, with its `Citation` hopping straight to the `Seed`. Its `Service`s then hold `Reach` =
`not-reached` — a **measured value**, not a `Gap` and not *nothing at all* — until something
answers there. Leaving a declared scope, by exclusion or by a narrower declaration, withdraws it
and takes its timelines with it, unless a current resolution still cites it. See
[ADR-0047](./docs/adr/0047-an-address-scope-is-its-own-enumeration.md).
_Avoid_: IP, host, node

**Service**:
An `(Address, port, transport)` triple — the subject reachability is measured against. The
`Address` in it is the **subject**, never a rendering of one, so this key and `Endpoint`'s above it
inherit the address form rather than restating it; a triple built from a string would put a second
normalisation site in the model and two spellings back on one host. It exists
for every `(port, transport)` in the recorded scope, **open or closed**, which is what gives
`unreachable` a subject to be a verdict about; so its membership is its `Address`'s membership
restated, and a port opening is a `Reach` move and **never** a membership event. See
[ADR-0031](./docs/adr/0031-membership-alerts-at-the-root-of-the-entering-subtree.md).
_Avoid_: port, open port, socket

**Endpoint**:
A `(Name, Service)` pair — the only key under which HTTP identity **and the presented
certificate chain** are single-valued, because two names on one address and port legitimately
serve different content and, under SNI, different certificates. Keyed on `Service` instead,
`certificate` would either force the arbitration `Span` refuses or record whichever name was
probed last, manufacturing drift on every virtual host every run. The `Name`
may be **absent**, meaning *the default response to a client that names nothing*: a real,
distinguishable measurement mode rather than a null in a key, and the only one available on an
address-scope `Seed` where no name is known yet. Absence is a **distinguished variant** of the key
and never an empty name — an empty text is refused and so is the root alone, and neither may collide
with the nameless endpoint. Both legs are **subjects** and never renderings of them, so this key
inherits the `Name` and `Address` forms rather than restating them; the SNI the `certificate`
handshake sends is a rendering of the `Name` key, computed on read like every other. It closes when
**either** leg withdraws — its
`Name` or its `Service` — so a nameless endpoint simply has one leg. See
[ADR-0011](./docs/adr/0011-a-facet-is-six-parts.md).
_Avoid_: URL, site, web asset, vhost

**Observation**:
A single measured fact: at a time, from a vantage, in a batch, a source reported that a
subject had a given value for a given facet. One concept across every facet, so that
change detection is written once rather than per facet. *At a time* is **when the source spoke, not
when we read it**, and the two come apart for exactly one class: a `declared` source's observation
takes its instant from the **operator's supply act**. Re-parsing a stored zone file on a cadence
would otherwise produce a *current* observation of a *stale* fact, which is worse than never
re-reading it — it makes staleness invisible instead of `not-evaluable`, and
[#48](https://github.com/winniel123/verge-asm/issues/48)'s fourth absence register, *you stopped
telling us*, is the thing that would go unreachable. It is held in **two tiers**, and the
boundary is what may still read it rather than how old it is. **Live** — within `k` cadences of
the tightest `Scan` covering it — is what every derivation reads, and it may never be discarded.
Past that it is **evidential**: a derivation may not read a stale observation and may never
re-derive history from one, so discarding it moves no value on any timeline and it is read only by
a person asking *what did we actually measure*. That is why the raw corpus is the one that grows
without bound — linear in time — while the corpus that may never be compacted is the small flat
one. Widening `k` changes what **future** folds read and never what past folds did, so an
observation discarded under the old bound was never going to be read again. See
[ADR-0041](./docs/adr/0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md).
_Avoid_: result, record, datapoint, scan result

**Facet**:
Which aspect of a subject an observation measured — `resolution`, `dns-record`,
`reachability`, `certificate`, `http-identity`, `tls-acceptance`. Adding one never means
writing a new way to detect change: the fold, `Span`, `Break`, `Gap` and `Transition` are
facet-agnostic. It does mean writing **six things** — a value space, a **decoder** per source, a
**canonicaliser**, a **differ**, a **discriminator** (empty for all but `dns-record`, which
carries the qtype), and a **batch-scope obligation** naming what its silence covers. Every value
space is a **closed union**, never a record with optional fields, because each facet's measured
negatives — `NoTLS`, `TLSRefused`, `NoHTTPResponse`, a Name Error — are values and must not collapse
into *we did not look*. A negative is named for **the exchange we made**, never for a property of
the listener our own offer decided: `http-identity`'s was `NotHTTP` until
[ADR-0025](./docs/adr/0025-an-offer-is-scope-only-where-the-value-enumerates-it.md), which is a
claim about the world that an `http/1.1`-only offer cannot carry against an h2-only listener. The two costs are **asymmetric**, and it is the asymmetry that decides what must be settled
early: adding a *facet* is strictly additive and costs `revealed` plus one message, while widening
an existing facet's *value space* moves the output of rows that already produced observations and
therefore `Break`s every timeline it has. A facet's value space is decided once; the rules read
over it are free forever. A facet is also **evidence and not a channel**: a `Transition` on a facet
timeline is not a message on its own account, and in v1 exactly one is — a `resolution` move that
opens an `Endpoint` no membership message covers. `dns-record` has no channel at all, so an MX or
TXT change is recorded and reaches nobody until a rule reads it. See
[ADR-0011](./docs/adr/0011-a-facet-is-six-parts.md),
[ADR-0015](./docs/adr/0015-the-value-space-is-the-commitment.md) and
[ADR-0026](./docs/adr/0026-the-facet-layer-is-evidence-not-a-channel.md).
_Avoid_: attribute, field, property

**Shadowed**:
The value a `resolution` observation takes when the answer was **not discriminated from its
parent's synthesis** — neither the synthesised answer nor a failure. ~~when the answer matches a
wildcard's measured poison signature~~ is **superseded here, at the site that specifies it**
([ADR-0058](./docs/adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)):
*matches* named our instrument standing in for the fact, and the fact is epistemic, which the
justification below always said. A wildcard is discriminated **only where its synthesis is
determinate**, measured **per component** — one `(qtype asked, RR type in the answer)` pair — whose
signature is `NoSynthesis` │ `Determinate(RRset)` │ `Indeterminate`. A name is `Shadowed` unless it
**differs at some determinate component**; an `Indeterminate` component is never consulted, and
where no component is determinate every name beneath that parent is `Shadowed`. It therefore errs
**toward this value by construction**: an unmeasurable component is refused the power to exempt,
because a false `Shadowed` withholds one value inside one facet while a false `Resolved` fabricates
an address set that cites `Address`es, opens `Endpoint`s and feeds `Exposure`. See
[ADR-0068](./docs/adr/0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md).
Recorded as a measured
value rather than discarded, because *we cannot see here* is a fact the operator needs and
the alternative manufactures drift: repoint one wildcard and every fictional name beneath
it reports a resolution change the same night. Whether a name is admitted under a wildcard
turns on its `Citation`, not on this answer — a certificate SAN survives, a guessed label
does not. That rule governs names **beneath** a wildcard and is the whole of the question, because
the wildcard name itself is a subject nowhere: it denotes a set rather than a thing, so nothing is
left for this value to be undecided about. It is also why no query is ever made for a name whose
leftmost label is `*` — such a query is answered by exact match and would read as its own poison
signature, and the ruling that it is not a subject closes that by construction rather than by a
carve-out in a leaf
([ADR-0060](./docs/adr/0060-a-wildcard-san-is-a-pattern-over-names-and-admits-none-of-them.md)).
It is a value on `dns-record` as well as `resolution`, since a wildcard synthesises
answers for *any* qtype — and it is **all-or-nothing across a name's qtypes**, holding on
`resolution` and on every `dns-record` discriminator or on none, because RFC 4592 blocks synthesis
for every type once the name exists, so a name discriminated at **any** component has **no**
synthesised RRset (ADR-0068). Deciding it takes two measurements — the name's answer and the zone's
poison signature — so like `Lame`, `NoTLS` and `NoHTTPResponse` it is decided by the **measurement
binary inside one batch**, never assembled afterwards from two observations. **Both are drawn on one
query path**, the batch's declared one, because a control probe asked somewhere the candidate was
not is not a measurement of the candidate's synthesis — **[measured]** direct to its own authority
`s3.amazonaws.com` carries **no A record**, a *determinate* `NoSynthesis`, while a resolver answers
every name beneath it with eight addresses, so a skewed pair discriminates **every** fictional label
and records it `Resolved` with a fabricated set. **A control probe is asked from where the answer it
discriminates was asked from**, and the path is one declared parameter with one value per `Batch`,
shared with `resolution-walk`
([ADR-0070](./docs/adr/0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md)).
A determinacy verdict read over that path is a claim about the **vantage** rather than about the
authority, which is exactly what the predicate needs, since the vantage is the one the candidate's
own answer came from too. That signature is a fact
about **the `Name` the control labels were generated under** — the name whose immediate child space
the wildcard synthesises into, named by the measurement rather than inferred from it, which is why it
is not *the apex*; and it admits nothing, since an answer served for a name that does not exist may
cite no `Address` and open no `Endpoint`. **A name is discriminated at its parent**: the control-probe
population of a `Batch` is the set of **immediate parents of the `Name`s in that batch's resolution
scope**, deduplicated, intersected with the `Seed` name scopes they sit inside, and recorded on the
`Batch` **by content** as the **seventh aperture input** — never a declared parameter of
`wildcard-discrimination`, whose control-label count and match predicate ~~stay where they are~~
**have since both been valued — the match predicate by ADR-0068, and the count and construction by
ADR-0069: ~~five~~ nine random labels plus one structured label, each exactly one label — the count
raised from five by [#115](https://github.com/winniel123/verge-asm/issues/115) on a measured
mechanism, per-label sharding, with the construction untouched.** That
needs no depth rule, because a control label constructed under a parent falls off the tree at the
same closest encloser its children do, whatever depth the wildcard sits at and **whether or not the
parent exists or is a `Name` we hold** — a probe site is a label sequence we construct rather than a
subject we cite, so nothing is admitted. **That equivalence is exactly why a control label is
*one* label**: a multi-label label has ancestors between it and the parent, and where one exists it
falls off at a deeper encloser and measures a different wildcard. The probe runs the batch's **declared qtype set**, all
seven, since this value is committed on `dns-record` for *any* qtype. A wildcard at or above the
operator's own apex is out of reach, the probing gate stopping the population at the `Seed`. Where
the probe under a name's parent **did not complete**, that name records a **`Gap`** and never a
value — *an undiscriminated answer is never a value* — while a probe that completed and found no
wildcard licenses everything beneath it. See
[ADR-0066](./docs/adr/0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md). **v1 holds it nowhere and ships no carrier for it.** It is
an input to this decision and a value on no timeline, so a wildcard being **repointed** is invisible:
every name beneath it stays `Shadowed` across the move, which is the suppression above working as
intended on the fictional names and swallowing the real one alongside them. The carrier is named and
**deferred rather than refused** — a facet of its own, never a value on `dns-record`, whose key is
already occupied by what the authority served for that subject itself. ~~Where no signature was
measured beneath a wildcard this value is unavailable and the synthesised answer reads as `Resolved`,
so **which names are control-probed** is a live question rather than a detail.~~ *(Settled by
ADR-0066, above: no held `Name` is left undiscriminated, and an incomplete probe yields a `Gap`
rather than a `Resolved` answer nobody could read.)* See
[ADR-0062](./docs/adr/0062-a-wildcards-synthesis-is-a-fact-about-the-name-it-was-probed-under.md).
_Avoid_: unverifiable, synthetic, wildcard hit, poison signature (our instrument's name, never the fact)

**Lame**:
The value a `resolution` observation takes when every nameserver the parent zone delegates
a `Name` to was reached and none of them serves it — RFC 8499 §7's *lame delegation*, and
the project uses the protocol's own word rather than a security taxonomy's. It is a
**measurement of the operator's infrastructure, not a failure of ours**, which is what
separates it from a source error, and it is only available because the measurement binary
queries the delegated authorities directly: a recursive resolver's SERVFAIL cannot tell a
dead delegation from a bad upstream, and attribution by inference is the *whose fault was
it* judgement that would make this a coverage gap wearing a value's clothes. That walk is **not
governed by the query path**
([ADR-0070](./docs/adr/0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md)):
a parameter able to route it through a resolver would delete this value, and a setting that silences
a finding is refused here as it is on an `Offer`. The walk's converse also holds and had never been
stated — **it supplies no address set**. `Resolved(unordered address set)`, `NoData` and `NameError`
are read on the declared path, because the leaf holds two answers to one name and **[measured]** they
differ: at `s3.amazonaws.com`, at one instant, the delegated authority answers a synthesised name
with a CNAME and no address while a resolver answers it with eight. What the walk decides is this
value and the per-nameserver `serves │ does-not-serve` RRset below, and nothing else. A delegation
only *partly* lame is not this value — the name still resolves, so `resolution` has not
moved — and is recorded per nameserver on `dns-record`, whose NS qtype therefore holds an RRset
of `(nameserver, serves | does-not-serve)` pairs rather than a name list. Like `Shadowed` it is
also a value on `dns-record`: the authorities were reached and refused to serve, so every qtype
is equally unanswerable.
_Avoid_: servfail, dangling, broken delegation, dead NS

**Certificate**:
An X.509 certificate, held as an immutable value and shared by fingerprint across every
endpoint presenting it. A certificate cannot change, so it cannot drift; what changes is
which certificate an `Endpoint` presents — held as the **ordered chain of fingerprints, leaf
first**, since order is on the wire. Its two negatives are distinct and both are values:
**`TLSRefused`** where the peer spoke TLS and accepted no candidate we offered, and **`NoTLS`**
where nothing on the port spoke TLS at all. Collapsing them files an SSLv3-only or SNI-required
listener under *not a TLS server*. The handshake feeding this facet sends **SNI equal to the
`Endpoint`'s name** — no SNI for the nameless one — and **no ALPN extension at all**, so that a
listener refusing our application protocols cannot cost us a chain we could otherwise read: ALPN
belongs to `http-exchange`, and a value a *second* leaf's parameter can decide is the `NotHTTP`
defect one facet across. What the handshake *negotiated* is not here and is not a
property of a certificate: a negotiated version is a function of our own ClientHello, so it
would move estate-wide on a library upgrade with nothing in the world having changed. See
`tls-acceptance`. **Certificate transparency is not a source of this facet**, and cannot be:
the value space's two variants are both outcomes of a wire exchange, and a log entry witnesses
issuance rather than presentation — so CT admits `Name`s and holds no timeline here or anywhere.
Attributing a logged certificate to an `Endpoint` nobody watched serve it would assert a
presence no scope record can catch, which is the no-false-absence rule read in the direction
that has no guard. The handshake that feeds it is **a step in the exchange that produces
`reachability`**, not a scan of its own and not a tier: neither negative can be read without
knowing the port was open, and the value space has no variant meaning *the port was shut*. So the
facet **has no single cadence** — it rides whichever port tier ran the exchange, and its currency
is the `Service`'s own, the tightest cadence covering that port. All tiers fold into one timeline
and therefore declare **one** candidate set between them. Currency alone does not license the
**clock class**, which is the one place a rule reads an always-current wall clock against a
possibly stale observed value: those three rules read a value only while its **age is within the
certificate's own horizon** — `⅓` of `not_after − not_before`, `½` below a ten-day validity, the
same horizon `certificate-expiring` computes — so the bound is `min(k × cadence, N)` and no new
number is declared. `k` is untouched and **failing that gate is not a `Gap`**: the value is present
and current, and only those three rules decline to read it, `not-evaluable` on the cause *we
measured; this rule cannot read the answer*. The other certificate rules are unaffected, comparing
observed values that age together. See
[ADR-0027](./docs/adr/0027-a-source-may-admit-without-observing.md),
[ADR-0028](./docs/adr/0028-a-facets-cadence-is-the-cadence-of-its-exchange.md) and
[ADR-0043](./docs/adr/0043-a-clock-reading-rule-bounds-its-evidence-in-the-subjects-own-units.md).
_Avoid_: cert record, TLS config, negotiated version, cipher

**tls-acceptance**:
The facet holding which protocol versions and cipher suites a `Service` **accepted** — measured
by enumeration on **its own weekly `Scan`**, one handshake per candidate, and attempted against
every open `Service` rather than a curated implicit-TLS port list. *Weekly* is a cadence bought
against N handshakes per service, never a port tier: scoping it to the top-1000 tier would leave
`verge-core`'s sensitive-only members unenumerated, since they rank nowhere by frequency. So its
`Scan`'s scope is the open `Service` population and the candidate set, and it is the fourth `Scan`
rather than a fifth port set. *Accepted* is the measured verb; *supported*
is a capability claim the measurement cannot carry. The **candidate set is the `Batch`'s recorded
scope, never part of the value**, so an offer of nine ciphers can never assert the tenth was
refused, and see [ADR-0028](./docs/adr/0028-a-facets-cadence-is-the-cadence-of-its-exchange.md) for
why this `Scan`'s candidate set and the `certificate` handshake's need not match. The candidate set
is **declared by us and recorded as what went on the wire, never taken
from the TLS library's defaults**: a default is not a declaration, and one left in place hides a
TLS-1.0-only listener as `NoTLS` on exactly the estate `tls-1.0-accepted` exists for. Widening the
offer is an aperture change, and because the candidate set sits inside the **value** rather than in
the key it costs a **`Break`** on every timeline of this facet **and on every `certificate`
timeline** — the same one declared set is carried by both TLS exchanges — not a `revealed`, which is
an opening kind and opens nothing here. Versions are enumerated TLS 1.0–1.3; **cipher suites only
for TLS 1.0–1.2**, because Go's `Config.CipherSuites` is ignored for TLS 1.3 and a per-candidate
negative over candidates we did not choose would move estate-wide on a library upgrade. See
[ADR-0025](./docs/adr/0025-an-offer-is-scope-only-where-the-value-enumerates-it.md),
[ADR-0030](./docs/adr/0030-an-offer-is-admitted-on-a-finding-or-on-a-falsity-it-prevents.md) and
the list itself in [`docs/spec/measurement-offers.md`](./docs/spec/measurement-offers.md).
_Avoid_: tls config, tls support, cipher scan

**Offer**:
What the measurement binary puts on the wire to decide what a value is *able* to say — the TLS
candidate set, the queried qtype set, the ALPN list, the EDNS options, the DNS transport policy.
Never a library default: a default is not a declaration, and one left in place is recorded either
as a version string that cannot tell a widening from a narrowing or as a parameter that is silent
on the change that matters. An offer is the `Batch`'s recorded **scope** where any value it feeds
carries a **per-candidate negative** — *we asked about X and X was not there* — and then it is scope
for every batch that makes it; otherwise it is a **declared parameter** of the leaf that made it.
A candidate is admitted on one of two limbs, never on an attestation, because an offer asserts
nothing about the world: its **acceptance is a finding**, or its **absence would make the
measurement false**. The recorded scope is what went on the wire and never what we intended, so
the build fails where a declared candidate is not offerable by the linked library. None of the five
is operator-configurable: an offer the operator can narrow is a finding the operator can silence.
See [ADR-0025](./docs/adr/0025-an-offer-is-scope-only-where-the-value-enumerates-it.md),
[ADR-0030](./docs/adr/0030-an-offer-is-admitted-on-a-finding-or-on-a-falsity-it-prevents.md) and
[`docs/spec/measurement-offers.md`](./docs/spec/measurement-offers.md).
_Avoid_: client config, scan settings, probe options, default

**Batch**:
One source, executed once, against one scope, from one vantage — recording the scope its
silence covers. The unit of like-against-like comparison. The recorded scope is what the
batch **completed**, never what it attempted, so a batch that failed outright covers
nothing and licenses no absence. It may be partitioned along any dimension its source
still retains completeness over, and no further. Every dimension of the recorded scope is recorded
**by content** — what we asked for, as it went on the wire — and never by the identity of the
library that chose it for us, since a widening is detected by diffing named dimensions and two
version strings cannot tell a widening from a narrowing. It also records the **`Derivation` leaf
versions of the measurement procedures it ran**, since a prober leaf's content is fixed when the
measurement happens and not when the fold reads it. Among its recorded dimensions is the
**control-probe population** — the `Name`s wildcard control labels were generated under, which are
the **parents** of the names in its resolution scope — because a name whose parent was not probed
can never be `Shadowed`, which is a silence rather than a value
([ADR-0066](./docs/adr/0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md)).
It is **read by the comparison path**, which is what separates it from `Dispatch` and puts it in the
observation corpus rather than the operational record however much *what we ran* it looks like: it is
**retained while any observation it produced is retained, or while it is the current `Citation` of a
subject in the estate** — the hop a source that admits without observing leaves behind. Deleting one
on the operational record's schedule strands an observation's scope and, where it is a `Citation`,
withdraws a subject. See
[ADR-0041](./docs/adr/0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md).
_Avoid_: run, scan run, execution, sweep

**Citation**:
The single-hop link from a subject to the observation that introduced it. Following
citations backwards always terminates at a `Seed` or a `declared` source, which is what
makes "why is this here?" answerable for everything in the estate. It is load-bearing in
both directions: a subject whose last citation goes stale has no chain back to a `Seed`,
which withdraws it *and* closes the probing gate on it. Where a source **admits without
observing** there is no observation to point at, and the hop is that source's `Batch` —
which already records the source, the vantage, the time and the scope, so the chain still
terminates at the `Seed` that scope was drawn from and a bad source is still identified by
traversing everything it introduced. See
[ADR-0027](./docs/adr/0027-a-source-may-admit-without-observing.md).
_Avoid_: provenance chain, lineage, discovery path

### Derived

**Custody**:
Whether an `Address` is under the `operator`'s control or is `third-party`, computed against
`Seed`s **alone**. It means **control of what listens**, never registry title: the party who
can consent to a scan is whoever controls the listener, and a cloud provider holds title to
every address it rents while being unable to say whether a tenant's port should be open. **Nor
routing reachability, which is the same failure with a second party in it**: the AS announcing a
route toward a prefix carries packets toward it and controls nothing inside it, so a transit
provider's announced set holds its customers' networks alongside its own, and a route collector is
evidence about the **path** rather than about the estate. That is why no BGP leg proposes here and
none ships in v1
([ADR-0063](./docs/adr/0063-a-routing-announcement-names-the-path-not-the-estate.md)). Named
`Ownership` until [ADR-0013](./docs/adr/0013-custody-is-control-and-extends-by-declaration.md),
which renamed it because the old word reads as title and had already produced a ticket arguing
the model could not describe its own modal operator. Registry data proposes seeds to the
operator; it is never read by the derivation, so no third party's file can open the probing
gate. Governs what may be probed, not merely how it is displayed — see
[ADR-0002](./docs/adr/0002-ownership-gates-probing.md),
[ADR-0012](./docs/adr/0012-a-proposer-is-not-a-source.md) and
[ADR-0013](./docs/adr/0013-custody-is-control-and-extends-by-declaration.md). The gate it governs
is **total over the `Address`**: a `third-party` address is connected to on **no port**, by no
tier and at no rate, because the objection is **authority** and port count is a quantifier over
one act — ADR-0002's ~~*"only the ports the `Name` implies (443, 80)"*~~ limb is **withdrawn at
the site that specifies it**
([ADR-0019](./docs/adr/0019-the-probing-gate-is-total-over-an-address.md)), leaving the opt-in
limb, which is the `custody extension`. A public name resolving to an address is the *name
holder's* invitation to their own service and never the *listener holder's* consent to be
measured, which is the whole distinction the rename below draws. `operator` is **necessary and not
sufficient**: the gate asks *may we connect to this address*, which presupposes the address denotes
one machine, and a **non-globally-reachable** address denotes one **per network realm**. So an
address whose most specific block in the IANA special-purpose registries reads `Globally Reachable`
= `False` is connected to only where a declared **address scope** covers it — a `custody extension`
may not open the gate over one, which generalises ADR-0013 §6's refusal to let an extension decide a
realm — and only from a `Vantage` that is not `internet`-class, which by its own definition sits
outside every realm the operator declared. The internet `Reach` leg over such an address is not
merely unmeasured but **unmeasurable**, so a connect from there measures a different machine and
files it on the flagship timeline
([ADR-0079](./docs/adr/0079-authority-presupposes-denotation-a-non-globally-reachable-address-is-probed-only-inside-a-declared-realm.md)).
So an install holding custody of
nothing measures `resolution` and `dns-record` at full aperture — **a query is not a connect** —
and measures none of the four facets that ride a TCP connect; its `Service` population is empty,
and every rule below that connect has an empty `Predicate domain`. There is **no
third value**: *covered by a `Seed`?* is a total question with no lookup left to fail, so
everything not covered is `third-party`, which is the closed direction. It is the one Derived
value whose change carries a safety consequence, so it holds a `Span` timeline. Alone among
Derived values its inputs are **not** all Declared — a `custody extension` makes it a function
of measured resolutions, so it moves when the world moves and not only when the operator does.
Its two causes part cleanly: the operator withdrawing an extension closes the gate beneath
addresses that are still cited, while a resolution ceasing to cite an address withdraws the
`Address` itself, which takes its timelines with it and leaves no `Gap` behind. **Narrowing an
address scope** — an exclusion, or a smaller CIDR — behaves like the second and not the first: the
addresses leave with their timelines and open no `Gap`, since a `Seed` no longer covers them and
there is no subject left to hold one, unless a current resolution still cites them
([ADR-0047](./docs/adr/0047-an-address-scope-is-its-own-enumeration.md)). Closing the
gate does **not** open a `Gap` directly — it stops feeding those timelines, and the last value
ages out under the currency bound, so a toggle inside one cadence is a non-event rather than a
`value → Gap → value` burst. The aged value is not a stale attribution, because this timeline
is current from the instant of the toggle and reads `third-party` beside it. **Rules keep reading
that value while it is current**, so a withdrawal cannot silence a rule that was firing — a
setting that switched off a measurement is refused here as it is on an `Offer` — and what follows
the bound is a `Gap` **naming the operator's own act**, never a blank and never a clear. The
population it leaves behind **can only shrink**: a shut gate observes no new `Service`, so no
subject can join it. Opening the gate
produces both a `revealed` opening on addresses never probed before and a closing `Gap` on
addresses that were. See
[ADR-0014](./docs/adr/0014-only-revealed-generalises.md).
_Avoid_: ownership, owned, in scope, authorized, mine

**Availability**:
Whether a `Vantage` is currently able to observe, concluded from its recent batch
outcomes over a fixed window rather than measured directly. A vantage that has failed
every attempt across the window is `unavailable`, which opens a `Gap` on the `Reach` of
its class — so `Exposure` that would need it is absent rather than quietly computed from
the class that still answers. Derived, though the `Vantage` it belongs to is Declared — we
never measured the vantage, we inferred it from what failed.
_Avoid_: health, status, up, reachable

**Vantage composition**:
How a reader of a **per-vantage** facet turns the set of per-vantage values into the one value it
reads. Every Derived value over such a facet performs one, and there are exactly **two kinds**.
**Cross-class**: every `Vantage class` the install runs must hold a current value and they must
**agree** — disagreement is **incommensurability rather than evidence**, since the classes are
looking at different worlds, which is what split horizon *is*, so the composed value is
`not-evaluable`, as it is where a class has no available vantage. **No quantifier is expressible on
this kind**: agreement is the whole of it. **Class-scoped**: one named `Vantage class`, read over the
available vantages of that class alone, with a **quantifier** the reader states from a closed union
of two — `existential` or `unanimous`. Disagreement here is **variance rather than
incommensurability** — geo-DNS, per-query rotation, anycast — and the quantifier is what says which
way variance falls. **Across classes a difference is a fact about our aperture; within a class it is
a fact about the authority**, and that asymmetry is the whole of the cut. A reader takes the
class-scoped kind exactly where the fact it is **named for** is scoped to a vantage
([ADR-0071](./docs/adr/0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md))
and the cross-class kind everywhere else; and within the class-scoped kind a **presence** claim
composes `existential` while an **absence** claim composes `unanimous`, because one vantage receiving
an answer establishes that the answer was served, while no number of vantages failing to receive one
establishes that none exists — a vantage that did not ask is not a vantage that got nothing. **An
empty in-scope set is `not-evaluable` under both kinds and never vacuous**: `unanimous` over an empty
set is vacuously *true*, and read that way *"every available vantage agrees on `NameError`"* withdraws
every `Name` in the estate the night every vantage goes unavailable. It is **not** a `Derivation` leaf
and never becomes one — it has no subject, no value space, no `Span` and no place in a version
vector, and sits inside its reader's own leaf, so changing one moves that reader and nothing else.
`Reach` is the one composition that *is* a leaf, and it can be because `reachability` has a single
consumer shape; `resolution` has two, so one stored composed value would be wrong for one consumer on
every `Name`. `Exposure` is **not** one of these — it composes two already-composed `Reach` legs and
is a projection rather than a quantified read. See
[ADR-0080](./docs/adr/0080-a-vantage-composition-is-cross-class-or-class-scoped-and-only-one-takes-a-quantifier.md),
[ADR-0006](./docs/adr/0006-subjects-leave-by-measurement.md) and
[ADR-0020](./docs/adr/0020-a-conflict-needs-two-enumerable-sources.md).
_Avoid_: aggregation, consensus, quorum, merge, the composition (unqualified)

**Reach**:
What vantages of one `Vantage class` found for one `Service` — `reached` or `not-reached`,
and nothing else. It is a **class-scoped `Vantage composition`** and its quantifier is
**`existential`**: one vantage of the class reaching the `Service` is `reached`, since that is a
**presence** claim and a service reachable from one internet position and geo-blocked at another is
reachable from the internet. Unanimity would under-report in the closed direction, against
`Vantage class`'s own stated failure direction. The quantifier had **never been written down** — this
term named its class and stopped — and is filled by
[ADR-0080](./docs/adr/0080-a-vantage-composition-is-cross-class-or-class-scoped-and-only-one-takes-a-quantifier.md)
with no version moving, nothing having been specified for a corpus row to move against.
There is no value for *we did not look*: a `Batch` whose recorded scope
excludes the port never feeds the timeline, so the absence is a `Gap` where the timeline was
already running and **nothing at all** where it never began — a `Gap` is a span, and an
absent timeline has none, per
[ADR-0014](./docs/adr/0014-only-revealed-generalises.md). A named `Derivation`
leaf, so a rule may read one leg and compose that leaf alone. It is also the object **alerting**
reads: the internet leg going `not-reached` → `reached` is the product's flagship message, fired
whether or not the other leg exists — and it **carries the census** of what opened beneath the
newly-reached `Service`, since `certificate`, `http-identity`, `tls-acceptance` and every rule over
them open there and an opening reaches nobody
([ADR-0026](./docs/adr/0026-the-facet-layer-is-evidence-not-a-channel.md)) — while the internal
leg is recorded and **never** alerted in
either direction — an internal port opening or closing is the commonest intentional change on that
leg, which is the ground on which a withdrawal is not alerted either. A leg that **opens** at
`reached` — a `Service` that was internet-reachable the first time we ever looked at it — emits
no `Transition`, so the flagship predicate does not match it; that news is carried by the census
on the entering subject's membership message and by nothing else. See
[ADR-0029](./docs/adr/0029-an-alert-fires-on-a-leg.md) and
[ADR-0031](./docs/adr/0031-membership-alerts-at-the-root-of-the-entering-subtree.md).
_Avoid_: reachable, probe result, port state, not-checked

**Exposure**:
The reachability conclusion for a `Service`, composed from the internet `Reach` and the
internal `Reach` rather than observed by any one vantage. It exists only where **both** legs
hold a value, and its four values — `exposed` (both `reached`), `edge-only`, `firewalled`,
`unreachable` (both `not-reached`) — are a **projection** of that 2×2, so a rule reads a leg
and never a value. A **one-legged reading is not an `Exposure` and gets no name**: where a
vantage class was never configured the surviving leg's `Reach` is rendered on its own under
*we never looked*, and where a configured leg went silent the `Exposure` timeline holds a
`Gap` under *we stopped looking*. `internal-only` was a fifth value until
[ADR-0017](./docs/adr/0017-exposure-needs-both-legs.md) and is withdrawn — it named a
one-legged reading, which is a fact about which vantages the operator runs rather than about
the `Service`. The internet leg going `not-reached` → `reached` is the move the product exists
to catch, and it spans **half the grid** rather than one cell — so the alert is fired by that
**leg** and never by this value, which makes `Exposure` a board axis and a census and **never an
alert source**, and makes the flagship fire identically on a one-legged install where no `Exposure`
exists at all ([ADR-0029](./docs/adr/0029-an-alert-fires-on-a-leg.md)). Reads `Availability`, and
therefore composes its version. Held as a `Span`
written once under the version that produced it, never recomputed on read — a correction ships
as a new version and a `Break`, not as rewritten history. See
[ADR-0010](./docs/adr/0010-exposure-composes-two-reaches.md) and
[ADR-0017](./docs/adr/0017-exposure-needs-both-legs.md).
_Avoid_: open, reachable, public, internal-only

**Signal**:
A named, versioned rule evaluated over observations — and over other Derived values, in
which case its version composes theirs — citing the observations that triggered it as
evidence. It has no lifecycle of its own; its lifecycle is its evidence's — **not its
subject's membership**, so a rule whose evidence is still current fires on a `Name` that has
withdrawn. Versions are
per rule, never one set-wide version, so an edit to one rule leaves the rest comparable.
A signal carries no severity: it is a named fact, and urgency belongs to the transition
that surfaced it. Evaluated where its evidence is absent — **or held and unreadable by this
rule**, as when a clock-reading rule's observation has aged past the subject's own horizon
([ADR-0043](./docs/adr/0043-a-clock-reading-rule-bounds-its-evidence-in-the-subjects-own-units.md))
— it returns `not-evaluable`,
which is not the same as not firing — but that word needs a **subject**, and where the
aperture never produced one there is no outcome to return and no row to render, so the
honesty lands on the aperture statement rather than on the rule. Its census is therefore
three members over one population — fired, did not fire, `not-evaluable` — counted over the
rule's `Predicate domain` and never over the timelines it happens to hold, or the
never-evaluable population is invisible by construction and the census is the clean bill of
health this term exists to refuse. That census is current state and never a comparison, so it
may not be rendered as a delta, a trend or a series — subtracting two of them conflates a moved
domain with a moved predicate, which is the comparison the drift model already makes correctly
one level down. Each member's count **is** the length of its own list, so every member is
enumerable in full and none is ever sampled, ranked, grouped or truncated: a member that cannot
be opened is a count whose basis cannot be shown, and a member is in any case the only cut of the
population the rule itself versions. What differs between those rows belongs to the row and what
is the same on every one of them belongs above them, so a member that is most of the estate takes
no treatment of its own — its length was never what would have made it a findings list.
Where the predicate reads a **declared parameter**, the census states that
parameter **as the rule expresses it** — the fraction, never its product on any one subject and
never the spread of thresholds the estate happens to produce, since a declared parameter is ours
and constant in every install while a statistic over the estate is neither, and is not a member of
the partition. A rule is **four parts** — its domain, its predicate, its `not-evaluable` case
and its version vector — plus one cost, the new measurement it requires, which is weighed and is
never a correctness objection. A cost weighed and **declined** is a complete exclusion needing no
principle beside it, and a fact fitting none of the rules we already have is a candidate for a new
rule rather than a reason to drop the fact — the shape of the set is a fact about the set, and rules
are versioned per rule precisely so that no rule's admission is a fact about any other
([ADR-0065](./docs/adr/0065-a-rule-is-excluded-by-its-fact-or-by-its-aperture-never-by-the-shape-of-the-set.md)).
It is **named for the fact it reads** — never for a
conclusion its evidence cannot carry, never for a protocol, and never for the **content** of a
table or parameter that decides the fact, which is why no rule reading a curated table encodes
that table in its name — and its scope is however many
protocols happen to express that fact, so covering exactly one is not disqualifying while three
protocols expressing one fact must be one signal rather than three. Being true of *most* of the
estate is likewise not a defect: a signal is a census, urgency is the transition's, and narrowing
a rule to make it fire less often is the model-layer damping `Drift` refuses. **Its edges are where
the drift class actually lives**, and they are cut per rule rather than per facet: `not-fired` →
`fired` is a message for every rule, since on a subject already in the estate that edge is by
construction *something got worse*; `fired` → `not-fired` is **silent except where a third party
could have caused the clearing**, which is exactly the four rules whose clearing condition is a
name somebody else can claim; `fired` → `not-evaluable` and `not-evaluable` → any value are both
coverage class, the second fired at the cause and stating the value it closed to. Entering or
leaving a `Predicate domain` is neither — it is not a `Transition` — but a rule that **opens at
`fired`** is nonetheless carried, and by exactly one of two things: the census of a message above
it where one exists, or **the facet `Transition` beneath it** where the subject entered the domain
because a value the rule reads moved. Where there is neither — the timeline merely opened, on a
slower `Scan` we authored — it reaches nobody, because a schedule arriving is not the world moving.
Where the fact a rule reads is **scoped to a vantage**, that scope is carried in the rule's name and
in its `Predicate domain`, and the rule is evaluated nowhere else: a claim read outside the context
that gives its operative term meaning ranks nothing, which is why
`sensitive-port-reached-from-internal` is refused and
`non-globally-reachable-address-resolved-from-internet` ships
([ADR-0071](./docs/adr/0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md)).
Where a rule reads a **curated table asserting about the world**, that table — not the rule — is
what an evidence standard governs, and the naming discipline above is usually why there is no such
table to govern: a rule named for the fact it reads needs no list to say what the fact means. Where
such a table exists, **transcribing an owner's artefact is not authoring a table and selecting from
one is** — the rows may be somebody else's, the *selection* is always ours, so a table whose selection
predicate is a column of the artefact itself is attested by that artefact whole, while a table whose
selection predicate is a judgement of ours owes an owner for the judgement (ADR-0071).
See
[ADR-0004](./docs/adr/0004-signals-are-release-coupled-rules.md),
[ADR-0015](./docs/adr/0015-the-value-space-is-the-commitment.md),
[ADR-0024](./docs/adr/0024-a-rules-domain-is-the-extension-of-its-name.md) — whose v1 table
enumerates ~~**sixteen**~~ **seventeen** rules since
[#128](https://github.com/winniel123/verge-asm/issues/128), against the stale *ten* in three ADRs —
[ADR-0026](./docs/adr/0026-the-facet-layer-is-evidence-not-a-channel.md),
[ADR-0032](./docs/adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md),
[ADR-0033](./docs/adr/0033-a-move-carries-the-rule-that-opens-at-fired.md),
[ADR-0065](./docs/adr/0065-a-rule-is-excluded-by-its-fact-or-by-its-aperture-never-by-the-shape-of-the-set.md)
and
[ADR-0071](./docs/adr/0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md).
_Avoid_: finding, issue, alert, vulnerability, detection, severity

**Predicate domain**:
The population one `Signal` is asked about — the denominator of its census, and one of the four
parts of a rule. It is **the extension of the rule's name**: the subjects of which the fact the
rule is named for could be asserted. So it may exclude a subject only where that fact could not
be true of it, and excluding one the rule would have fired on is the model-layer damping `Drift`
refuses, whatever it is called. It may be **measured** rather than structural — `NoTLS` puts an
endpoint outside every certificate rule's domain — and it may cite only evidence the rule itself
declares, so it composes no leaf the rule does not already compose and sits **inside** the rule's
own leaf rather than beside it. Editing one is an output-affecting change: the leaf moves and the
rule `Break`s uniformly. A subject outside the domain is **not rendered at all** — not a census
member, not a row, not a state, and not a transition — because a census is current state and may
never be rendered as a delta, so the operator is never shown a domain difference and there is
nothing for a message to explain. *(The earlier reason — that the subject left the domain by a
facet `Transition` "already stored and already alerted" — is **withdrawn** by
[ADR-0026](./docs/adr/0026-the-facet-layer-is-evidence-not-a-channel.md): most facet transitions
are stored and silent. The stated cost is that a rule leaving its domain while `fired` goes quiet
with no message — the shrinking direction, and overwhelmingly the operator's own remediation.)*
A rule **entering** its domain at `fired` is carried by the move that admitted it, where the
subject entered because a value the rule reads moved
([ADR-0033](./docs/adr/0033-a-move-carries-the-rule-that-opens-at-fired.md)); what stays uncarried
is only the opening with no move beneath it, where a slower `Scan` reached the facet for the first
time.
Distinct from `not-evaluable`, which is
a value about **our own sight** (`Shadowed`), and from a `Gap`, which is no value at all: outside
the domain means *the question does not arise*. A **total** domain is legal. An **empty** one is
legal too and renders as a no-population panel, never as a census of zeroes. See
[ADR-0024](./docs/adr/0024-a-rules-domain-is-the-extension-of-its-name.md) and
[ADR-0026](./docs/adr/0026-the-facet-layer-is-evidence-not-a-channel.md).
_Avoid_: scope, filter, applicability, eligible set, in-scope subjects

**Derivation**:
The named, versioned procedure that produced a Derived value — or, inside the measurement
binary, that **decided** an observed value rather than reporting it. Something is a derivation
exactly where **its output can move while the world does not**, which is why five decision
procedures inside the one binary are named leaves (`connect-outcome`, `tls-handshake`,
`http-exchange`, `resolution-walk`, `wildcard-discrimination`) while the binary itself is not:
a leaf is named for what it decides, never for the artefact that ships it. Its version moves on an
**output-affecting change** and never because a release shipped — enforced by a golden corpus
in CI rather than by discipline, since neither a hash of the code (bumps on a refactor) nor a
hash of the parameters (silent on a behavioural fix) tracks what we care about. The gate runs
**both ways**: a version may move only where a corpus row's output moved, a declared parameter
changed, or an uncovered move was recorded — so a version that moves for nothing fails the build
as loudly as an output that moves for free. A `Span`
carries the **vector** of derivation versions it was produced under: one leaf per named
derivation, flattened across everything that derivation reads, with parameters held inside
their own leaf rather than beside it. Comparison is legal exactly where two vectors are equal.
A **declared parameter** is therefore authored by the project and ships in the release, and
**none is ever operator-configurable**: it sits inside a leaf, so moving one moves a version and a
moved version is a `Break`, which makes a settings field the one actor that could break the estate
without a release and without a corpus row moving — and it would leave two installs on one release
comparing as comparable while holding different content behind one leaf. An operator's dial may sit
anywhere **outside** every derivation and nowhere inside one, which is where the coverage alert
threshold, notification routing and any flap suppression may legally sit. *(~~all flap suppression
already are~~ — **withdrawn** by [#119](https://github.com/winniel123/verge-asm/issues/119) /
[ADR-0039](./docs/adr/0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md):
the sentence asserted a mechanism that has never existed. The licence is unchanged and v1
exercises none of it — v1 ships routing by class and #22's coverage threshold, and no suppression,
no coalescing and no digest window.)* A rule's declared parameters
**express fractions of quantities the rule reads**, wherever the quantity is one the world moves —
a parameter shipped as the *product* of a fraction and a moving quantity is a measurement of the
world on the day it was written, and it goes stale with nothing in the repository changing and no
document anywhere being retracted. See
[ADR-0008](./docs/adr/0008-derivation-versions-move-on-content.md),
[ADR-0021](./docs/adr/0021-a-version-leaf-is-a-decision-not-a-binary.md) and
[ADR-0034](./docs/adr/0034-derive-the-claim-before-looking-for-the-owner.md).
_Avoid_: rule version, schema version, algorithm

**Span**:
One period during which a timeline held a single value, keyed by `(subject, facet, discriminator,
vantage, source)` — the discriminator being facet-defined and empty for all but `dns-record`,
which carries the qtype, or a batch covering MX and not TXT would assert an empty TXT RRset it
never measured. One timeline per source, so two sources that disagree hold two true facts rather
than forcing an arbitration. It opens, it is current, it closes; the open span is the current
state — so a **withdrawn** subject's timelines close rather than holding an open span, there being no
current state for one to hold. That clause was **tested on its own and confirmed**
([#140](https://github.com/winniel123/verge-asm/issues/140) ·
[ADR-0082](./docs/adr/0082-a-withdrawn-subjects-timelines-close-and-the-withdrawn-period-is-on-no-timeline.md)),
on grounds independent of this wording: a closed span is free to keep while an **open span must be
fed**, so holding one open means either rotting to a `Gap` within `k` cadences or querying every
withdrawn subject forever; withdrawal needs **every available vantage** to agree, so it is a fact
about the subject and a span is keyed per `(vantage, source)`; and ADR-0007 already closes a
cascaded subject's spans, *or a dead one keeps an open span and every current-state query returns it
as live*. The withdrawn period is therefore **on no timeline at all** — not a value and not a `Gap` —
which is what leaves the span before it and the span after a return **adjacent**, and is why
`returned` is derivable by ordinary machinery wherever no `Break` sits between them. Carries the
versions it was derived under, which is what makes
comparison legal. See [ADR-0007](./docs/adr/0007-drift-is-a-timeline-of-spans.md).
**This corpus is never compacted**, and the two reasons are independent. Deleting the span before an
open one converts `returned` into `appeared`, which is a clock moving a value about the world —
[ADR-0006](./docs/adr/0006-subjects-leave-by-measurement.md)'s refusal read at the storage layer. And
one is written when a value **moves**, so the corpus is proportional to **drift** rather than to time:
at the shipped ceiling of one declared `/22` it is ~672,000 rows and **flat**, against ~98M
observation rows a **year**, so the corpus that may never be deleted is the small one and the corpus
that grows without bound is the one nothing may read. ~~The open span and the one preceding it can
never be compacted~~ is **superseded here, at the site that specifies it**
([ADR-0058](./docs/adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)):
read alone it says the rest may be, and none of it may. That floor survives as the **precondition on
any compaction a later version ships**, read *within one derivation*, and it gains a third limb it
never had — a truncated timeline renders as a **labelled floor** and never as an opening, since a
truncation that reads as an opening is `appeared` manufactured by storage. **Retention may never be
the tighter clamp**: the `Break` clamp is visible and names the leaf that moved, and a retention
horizon biting before it is a second horizon nobody can see. See
[ADR-0041](./docs/adr/0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md).
_Avoid_: interval, state, record, period, snapshot

**Transition**:
The adjacency between two consecutive `Span`s on one timeline. Derived on read, never stored
— storing it alongside the spans would be a second representation of one fact. Three ways in
are named, and they do not all live in the same place: `appeared` (discovery) and `returned`
(a decommission undone) are **membership only**, because they describe a subject, while
`revealed` (a widened aperture — *we* started looking, the world did not move) belongs to
**any** timeline, because membership is a property of a subject and aperture is a property of
looking, which is per-timeline. An opening caused by neither is recorded, unnamed and
unalerted — a `tls-acceptance` timeline opens days after its `Service` did because the
enumeration runs on its own weekly `Scan`, and nothing about the world or our aperture moved.
A `Gap` closing is none of these: it is an ordinary adjacency, and always an observer event,
since a `Gap` exists only where we could not say. **A `Transition` is a message only where it is
the sole carrier of a fact the operator asked for** — the layer it sits in never decides, and the
**facet layer is evidence rather than a channel**. So the internet `Reach` leg going
`not-reached` → `reached` is a message and `refused` ↔ `no-response` beneath it is not, though
both are facet moves and only one moves the projection a predicate reads. Where two layers would
report one fact the message fires at the cause and the other rides its census, which is why
`sensitive-port-reached-from-internet`'s firing edge never fires a message of its own. **A move
carries the rule that opens at `fired` beneath it**, once per `Transition` and with the census of
every rule that opened, since entering a `Predicate domain` is not itself a `Transition` and has no
edge of its own — and because no `Transition` crosses a `Gap` or a `Break` and an opening emits
none, that one test excludes a membership entry, an aperture widening, a closing `Gap` and a slower
tier without a case for any of them. See
[ADR-0026](./docs/adr/0026-the-facet-layer-is-evidence-not-a-channel.md) for the enumeration over
all six facets,
[ADR-0033](./docs/adr/0033-a-move-carries-the-rule-that-opens-at-fired.md) for the growing
direction it under-enumerated, and
[ADR-0014](./docs/adr/0014-only-revealed-generalises.md) — whose worked example this was, and
which named `certificate` until
[ADR-0028](./docs/adr/0028-a-facets-cadence-is-the-cadence-of-its-exchange.md) found that
facet's handshake rides the `reachability` exchange and opens with its `Service`.
**`returned` is the one member a `Break` destroys rather than clamps**, and the cause is a release
rather than an operator setting. A withdrawn subject's timeline is closed, so a `Break` between the
withdrawal and the return leaves the reopening with nothing legally before it: no `Transition` is
derived, the subject re-enters the estate, and the membership message fires reading **`appeared`** —
the carrier correct and the word a lie. It cannot be repaired afterwards, since history is never
re-derived, so the release **states** the loss on the re-baseline message that already names the leaf
that moved. Membership therefore composes the **narrowest** vector that decides presence and a
release may not widen it: a `Name`'s and a cited `Address`'s membership both compose
`resolution-walk`, which [ADR-0021](./docs/adr/0021-a-version-leaf-is-a-decision-not-a-binary.md)
puts on a **dependency** cadence, while a `Seed`-covered `Address` composes **nothing at all** — a
`Seed` is Declared and carries no vector — so that one population's membership timeline cannot break.
See
[ADR-0041](./docs/adr/0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md).
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
`Signal` would be `not-evaluable`, by an observation ageing past its currency bound, and by an
answer we cannot read — a truncated RRset no fallback transport recovered, or a `resolution` we
could not discriminate because the control probe under the name's parent did not complete. **A
truncated answer is never a value and an undiscriminated answer is never a value**
([ADR-0025](./docs/adr/0025-an-offer-is-scope-only-where-the-value-enumerates-it.md),
[ADR-0066](./docs/adr/0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md)). A gap
never withdraws a subject: ceasing to measure is not measuring absence. It **records its
cause**, which is what makes a fourth opening kind unnecessary: a value arriving after a gap
is accounted for by the gap sitting before it. Distinct from a timeline that never existed —
a batch whose recorded scope excludes a thing does not touch its timeline at all, so there is
no span to hold. Nothing is compared across a gap and no `Transition` crosses one, but the
values on either side **may be shown together, labelled as undatable**: unlike a `Break`, both
are the same kind of thing under one derivation, and what is missing is *when* it moved, not
the licence to compare. See
[ADR-0014](./docs/adr/0014-only-revealed-generalises.md).
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
It is **the whole of the operational record and the only corpus a wall clock may retire** — and the
fence above is the reason rather than a coincidence, since the property that makes a record safe to
delete on a schedule is the property that makes it safe to keep out of the comparison path. Its
retention window is an **operator dial**, the strongest instance of *outside every derivation* in the
model, floored at `k` cadences of the slowest **enabled** `Scan` — stated as a multiple and never as a
day count, the cadence being a quantity the operator moves — below which `Coverage` cannot answer
whether the slowest scan ran. The cost is stated: an aged-out dispatch may be the only evidence a
believed-in measurement never happened, which is a forensic loss and the operator's to price. **v1
ships it unbounded**, one row per firing being nothing to retire. See
[ADR-0041](./docs/adr/0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md).
_Avoid_: scan run, run, execution, job group

**Message**:
One firing of one cause, computed **once at the cause** and never recomputed — recomputing one
would reach back across a `Break`. It carries its class, the key of the subject or scope it
fired at, the instant of the cause, and its census where it has one; it holds read-state and
its delivery outcomes and no other operator state. Operational, and the layer is the point: a
message records that the operator was **told**, never what is true of the estate. The fact
itself is in the timelines, so if the two ever disagree the timeline wins and the message is
still a true record of what we said — which is what stops a stored message being a second
representation of one fact, and what keeps `Finding` from returning as a diffed message log.
The comparison path may never read one, nothing is ever concluded by comparing two, and
alerting fires **at the cause**, so the unit is the message and never the affected subject.
Every message is written and rendered **unconditionally**: the store is not a `Channel`, has no
configuration, cannot be disabled and cannot fail. Which transitions are messages is settled
elsewhere and not by this entry — see
[ADR-0026](./docs/adr/0026-the-facet-layer-is-evidence-not-a-channel.md),
[ADR-0029](./docs/adr/0029-an-alert-fires-on-a-leg.md) and
[ADR-0039](./docs/adr/0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md).
_Avoid_: alert, notification, event, finding, incident, ticket

**Delivery**:
One `Channel`'s attempt to carry one `Message`, holding the attempt count and the outcome.
Operational, on `Dispatch`'s terms. Retries on a bounded, project-authored budget and is then
**dead-lettered** — on the queue's existing machinery rather than a second one beside it — and
a dead-lettered delivery loses the **escalation** and never the message, which is still in the
store, unread and marked undelivered. At-least-once at the receiver, unordered, and never
back-pressuring measurement: no exactly-once claim is made because no exactly-once mechanism
exists. **A dead-lettered `Delivery` licenses no silence**, as a dead-lettered `Batch` licenses
no absence. It is **never itself a `Message`** — a delivery failure is not the world moving,
our looking changing, or a clock crossing, so it has no cause and gets no fifth one — and it
never touches `Coverage`, which answers *is what I am looking at complete?* rather than *were
you told?*. It is rendered on the message it failed to carry and on the channel it belongs to,
and nowhere else. See
[ADR-0039](./docs/adr/0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md).
_Avoid_: send, push, notification attempt, dispatch (reserved), retry

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
